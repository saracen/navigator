package repository

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage/memory"
	"gopkg.in/src-d/go-git.v4/utils/ioutil"

	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/ignore"
	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/ghodss/yaml"
)

const (
	requirementsName = "requirements.yaml"
	lockfileName     = "requirements.lock"
)

type repository struct {
	logger log.Logger

	name        string
	url         string
	backend     *git.Repository
	directories []IndexDirectory

	dm *DependencyManager

	indexManager *IndexManager

	// A map of chart filename + file hash, so we know which charts have already
	// been processed
	visited map[string]struct{}

	head plumbing.Hash
}

// NewGitBackedRepository returns a new git-backed based repository.
func NewGitBackedRepository(logger log.Logger, dependencyManager *DependencyManager, name, url string, directories []IndexDirectory) Repository {
	repo := &repository{
		logger:       logger,
		name:         name,
		url:          url,
		directories:  directories,
		visited:      make(map[string]struct{}),
		indexManager: dependencyManager.IndexManager(),
		dm:           dependencyManager,
	}

	dependencyManager.AddRepository(repo)
	return repo
}

func (r *repository) URL() string {
	return r.url
}

func (r *repository) Name() string {
	return r.name
}

func (r *repository) Update() (err error) {
	begin := time.Now()

	if r.backend == nil {
		r.backend, err = git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
			URL:           r.url,
			ReferenceName: plumbing.Master,
			SingleBranch:  true,
		})
		if err != nil {
			return err
		}
	} else {
		err := r.backend.Fetch(&git.FetchOptions{RefSpecs: []config.RefSpec{"refs/heads/*:refs/heads/*"}})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return err
		}
	}

	level.Info(r.logger).Log("event", "fetching", "repository", r.url, "took", time.Since(begin))

	var ref *plumbing.Reference
	defer func(begin time.Time) {
		if err == nil {
			level.Info(r.logger).Log("event", "indexing", "repository", r.url, "head", ref.Hash(), "took", time.Since(begin))
		} else {
			level.Error(r.logger).Log("event", "indexing", "repository", r.url, "head", ref.Hash(), "took", time.Since(begin), "err", err)
		}
	}(time.Now())

	ref, err = r.backend.Head()
	if err != nil {
		return err
	}

	if r.head == ref.Hash() {
		return nil
	}

	iter, err := r.backend.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return err
	}

	err = iter.ForEach(r.parseCommit)
	if err != nil {
		return err
	}
	r.head = ref.Hash()

	return nil
}

func (r *repository) parseCommit(c *object.Commit) error {
	level.Debug(r.logger).Log("event", "parsing", "commit", c.Hash.String())

	tree, err := c.Tree()
	if err != nil {
		return err
	}

	for _, directory := range r.directories {
		var subtree *object.Tree

		if directory.Name == "" {
			subtree = tree
		} else {
			subtree, err = tree.Tree(directory.Name)
			if err == object.ErrDirectoryNotFound {
				level.Debug(r.logger).Log("event", "parsing", "commit", c.Hash.String(), "directory", directory, "err", err)
				return nil
			}
			if err != nil {
				return err
			}
		}

		if err = subtree.Files().ForEach(r.processFile(c, directory)); err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) processFile(c *object.Commit, directory IndexDirectory) func(f *object.File) error {
	return func(f *object.File) error {
		// ignore if not Chart.yaml file
		if path.Base(f.Name) != chartutil.ChartfileName {
			return nil
		}

		// ignore if already processed chart
		key := f.Name + f.Hash.String()
		if _, ok := r.visited[key]; ok {
			level.Debug(r.logger).Log("event", "already-indexed", "commit", c.Hash.String(), "directory", directory.Name, "file", f.Name)
			return nil
		}
		r.visited[key] = struct{}{}

		// load chart metadata
		md, err := r.loadMetadataFile(f)
		if err != nil {
			level.Error(r.logger).Log("event", "parsing", "commit", c.Hash.String(), "directory", directory.Name, "file", f.Name, "err", err)
			return nil
		}

		// get index for this repository
		index, err := r.indexManager.Get(directory.IndexName)
		if err != nil {
			return err
		}

		// index chart
		url := repoCommitChartToPath(r.name, c.Hash.String(), path.Join(directory.Name, path.Dir(f.Name)), md.Name, md.Version)
		if added := index.Add(md, []string{url}, c.Committer.When); added {
			level.Debug(r.logger).Log("event", "indexed", "commit", c.Hash.String(), "directory", directory.Name, "file", f.Name, "chart", md.Name, "version", md.Version)
		}

		return nil
	}
}

// VersionedChartPackage returns a versioned chart package that exists in the
// git repository at the commit and chart name provided.
func (r *repository) ChartPackage(name string) (Archiver, error) {
	commit, name := pathHeadTail(name)
	if name == "" {
		return nil, ErrInvalidPackageName
	}

	hash := plumbing.NewHash(commit)
	c, err := r.backend.CommitObject(hash)
	if err != nil {
		return nil, err
	}

	tree, err := c.Tree()
	if err != nil {
		return nil, err
	}

	// check that the package is in one of the specified directories
	if !IndexDirectories(r.directories).Match(name) {
		return nil, object.ErrDirectoryNotFound
	}

	tree, err = tree.Tree(name)
	if err != nil {
		return nil, err
	}

	// load helm ignore file
	rules, err := r.loadIgnoreFile(tree)
	if err != nil {
		return nil, err
	}
	rules.AddDefaults()

	// load helm dependencies
	dependencies, err := r.loadDependencies(tree)
	if err != nil {
		return nil, err
	}

	deps, err := r.dm.Download(dependencies)
	if err != nil {
		return nil, err
	}

	return &versionedChartPackage{path.Base(name), rules, tree.Files(), deps}, nil
}

func (r *repository) loadMetadataFile(f *object.File) (*chart.Metadata, error) {
	contents, err := f.Contents()
	if err != nil {
		return nil, err
	}

	return chartutil.UnmarshalChartfile([]byte(contents))
}

func (r *repository) loadIgnoreFile(t *object.Tree) (*ignore.Rules, error) {
	f, err := t.File(chartutil.IgnorefileName)
	if err != nil && err != object.ErrFileNotFound {
		return nil, err
	}
	if f == nil {
		return ignore.Empty(), nil
	}

	reader, err := f.Reader()
	if err != nil {
		return nil, err
	}
	defer ioutil.CheckClose(reader, nil)

	return ignore.Parse(reader)
}

func (r *repository) loadDependencies(t *object.Tree) ([]*chartutil.Dependency, error) {
	var requirementsLock *chartutil.RequirementsLock
	err := r.loadSerializedFile(t, lockfileName, &requirementsLock)
	if err != nil && err != object.ErrFileNotFound {
		return nil, err
	}

	if requirementsLock != nil {
		return requirementsLock.Dependencies, nil
	}

	var requirements *chartutil.Requirements
	err = r.loadSerializedFile(t, requirementsName, &requirements)
	if err != nil && err != object.ErrFileNotFound {
		return nil, err
	}

	if requirements != nil {
		return requirements.Dependencies, nil
	}

	return nil, nil
}

func (r *repository) loadSerializedFile(t *object.Tree, name string, obj interface{}) error {
	f, err := t.File(name)
	if err != nil {
		return err
	}

	contents, err := f.Contents()
	if err != nil {
		return err
	}

	return yaml.Unmarshal([]byte(contents), obj)
}

type fileInfo struct {
	os.FileInfo
	name string
	dir  bool
}

func newFileInfo(name string, dir bool) *fileInfo {
	return &fileInfo{name: name, dir: dir}
}

func (f *fileInfo) Name() string {
	return f.name
}

func (f *fileInfo) IsDir() bool {
	return f.dir
}

type versionedChartPackage struct {
	name  string
	rules *ignore.Rules
	files *object.FileIter
	deps  map[string][]byte
}

func (a *versionedChartPackage) Archive(w io.Writer) (err error) {
	zipper := gzip.NewWriter(w)
	zipper.Header.Extra = []byte("+aHR0cHM6Ly95b3V0dS5iZS96OVV6MWljandyTQo=")
	zipper.Header.Comment = "Helm"
	defer zipper.Close()

	twriter := tar.NewWriter(zipper)
	defer twriter.Close()

	for name, data := range a.deps {
		h := &tar.Header{
			Name: path.Join(a.name, "charts", name),
			Mode: 0755,
			Size: int64(len(data)),
		}

		if err := twriter.WriteHeader(h); err != nil {
			return err
		}

		if _, err := twriter.Write(data); err != nil {
			return err
		}
	}

	return a.files.ForEach(func(f *object.File) error {
		// ignore file
		if a.rules.Ignore(f.Name, newFileInfo(path.Base(f.Name), false)) {
			return nil
		}

		// ignore directories
		dir := strings.Split(path.Dir(f.Name), "/")
		for i := 0; i < len(dir); i++ {
			if a.rules.Ignore(path.Join(dir[:i+1]...), newFileInfo(dir[i], true)) {
				return nil
			}
		}

		h := &tar.Header{
			Name: path.Join(a.name, f.Name),
			Mode: 0755,
			Size: f.Size,
		}

		if err := twriter.WriteHeader(h); err != nil {
			return err
		}

		r, err := f.Reader()
		if err != nil {
			return err
		}
		defer ioutil.CheckClose(r, nil)

		_, err = io.Copy(twriter, r)

		return err
	})
}
