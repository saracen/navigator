package repository

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"k8s.io/helm/pkg/chartutil"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// DependencyManager handles the downloading of chart dependencies. A chart
// dependency is a reference to a versioned chart within a remote repository.
//
// To download a chart, first its repository index is downloaded and then
// searched to find the chart dependency URLs.
type DependencyManager struct {
	client *http.Client
	logger log.Logger

	indexManager *IndexManager

	// local repositories
	local map[string]Repository

	// remote repositories
	remote      map[string]*singleflightIndex
	remoteMutex sync.Mutex
}

type singleflightIndex struct {
	*Index
	sync.Mutex
}

type repositoryLink struct {
	Alias string
	URL   *url.URL
}

// NewDependencyManager returns a new dependency manager.
func NewDependencyManager(logger log.Logger, indexManager *IndexManager) *DependencyManager {
	return &DependencyManager{
		client:       &http.Client{Timeout: time.Second * 10},
		logger:       logger,
		indexManager: indexManager,
		local:        make(map[string]Repository),
		remote:       make(map[string]*singleflightIndex),
	}
}

// AddRepository adds a local repository for resolving local dependencies.
func (dm *DependencyManager) AddRepository(repo Repository) {
	dm.local[repo.Name()] = repo
}

// IndexManager returns the index manager.
func (dm *DependencyManager) IndexManager() *IndexManager {
	return dm.indexManager
}

// Download fetches multiple dependencies concurrently and returns a map of
// the (chart name, archive data).
func (dm *DependencyManager) Download(dependencies []*chartutil.Dependency) (map[string][]byte, error) {
	var wg sync.WaitGroup

	type state struct {
		data []byte
		err  error
	}

	states := make([]state, len(dependencies))
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	for idx, dep := range dependencies {
		var err error

		link := &repositoryLink{}
		alias := strings.HasPrefix(dep.Repository, "@") || strings.HasPrefix(dep.Repository, "alias:")
		if alias {
			link.Alias = strings.TrimPrefix(strings.TrimPrefix(dep.Repository, "alias:"), "@")
		} else {
			link.URL, err = url.Parse(dep.Repository)
			if err != nil {
				return nil, fmt.Errorf("Chart dependency %v:%v has invalid repository: %v", dep.Name, dep.Version, dep.Repository)
			}

			if link.URL.Scheme != "http" && link.URL.Scheme != "https" {
				return nil, fmt.Errorf("Chart dependency %v:%v has unsupported repository scheme: %v://", dep.Name, dep.Version, link.URL.Scheme)
			}

			link.URL.Path = path.Join(link.URL.Path, "index.yaml")
		}

		wg.Add(1)
		go func(idx int, dep *chartutil.Dependency, link *repositoryLink) {
			defer wg.Done()

			if link.URL == nil {
				states[idx].data, states[idx].err = dm.fetchLocalPackage(dep, link)
				return
			}

			packageURL, err := dm.getPackageURL(dep, link)
			if err != nil {
				states[idx].err = err
				return
			}

			archive, err := dm.download(ctx, packageURL)
			if err != nil {
				states[idx].err = err
				cancel()
				return
			}

			states[idx].data = archive
		}(idx, dep, link)
	}

	wg.Wait()

	archives := make(map[string][]byte)
	for idx, dep := range dependencies {
		err := states[idx].err
		if err != nil {
			return nil, err
		}
		archives[dep.Name+".tgz"] = states[idx].data
	}

	return archives, nil
}

func (dm *DependencyManager) fetchLocalPackage(dep *chartutil.Dependency, link *repositoryLink) (body []byte, err error) {
	index, err := dm.indexManager.Get(link.Alias)
	if err != nil {
		return nil, err
	}

	chart, err := index.Get(dep.Name, dep.Version)
	if err != nil {
		return nil, err
	}

	repo, directory := repoCommitChartFromPath(chart.URLs[0])
	archiver, err := dm.local[repo].ChartPackage(directory)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	err = archiver.Archive(buf)

	return buf.Bytes(), err
}

func (dm *DependencyManager) download(ctx context.Context, downloadURL *url.URL) (body []byte, err error) {
	defer func(begin time.Time) {
		if err == nil {
			level.Info(dm.logger).Log("event", "download", "url", downloadURL, "took", time.Since(begin))
		} else {
			level.Error(dm.logger).Log("event", "download", "url", downloadURL, "took", time.Since(begin), "err", err)
		}
	}(time.Now())

	req, _ := http.NewRequest("GET", downloadURL.String(), nil)
	req = req.WithContext(ctx)

	resp, err := dm.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func (dm *DependencyManager) repository(repository string) *singleflightIndex {
	dm.remoteMutex.Lock()
	defer dm.remoteMutex.Unlock()

	// Initialise the repository with an empty index. The repository is fetched
	// and updated when a package required does not exist in it.
	if _, ok := dm.remote[repository]; !ok {
		dm.remote[repository] = &singleflightIndex{Index: NewIndex()}
	}

	return dm.remote[repository]
}

func (dm *DependencyManager) getPackageURL(dep *chartutil.Dependency, link *repositoryLink) (*url.URL, error) {
	index := dm.repository(dep.Repository)

	index.Lock()
	defer index.Unlock()

	// update repository if dependency doesn't exist
	// todo: when else do we update a remote repository?
	if _, err := index.Get(dep.Name, dep.Version); err != nil {
		body, err := dm.download(context.TODO(), link.URL)
		if err != nil {
			return nil, err
		}

		if err := index.Unmarshal(body); err != nil {
			return nil, err
		}
	}

	chart, err := index.Get(dep.Name, dep.Version)
	if err != nil {
		return nil, err
	}

	var rawChartURL string
	if len(chart.URLs) > 0 {
		rawChartURL = chart.URLs[0]
	}

	chartURL, err := url.Parse(rawChartURL)
	if err == nil && !chartURL.IsAbs() {
		chartURL, err = url.Parse(dep.Repository + "/" + chartURL.Path)
	}
	if err != nil {
		return nil, fmt.Errorf("Chart dependency %v:%v has invalid package url: %v", dep.Name, dep.Version, rawChartURL)
	}

	return chartURL, nil
}
