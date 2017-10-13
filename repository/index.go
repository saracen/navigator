package repository

import (
	"io"
	"sync"
	"time"

	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/repo"

	"github.com/ghodss/yaml"
)

// Index handles the indexing of charts.
type Index struct {
	mutex sync.RWMutex
	file  *repo.IndexFile
	cache []byte

	cached bool
}

// NewIndex returns a new Index.
func NewIndex() *Index {
	return &Index{
		file: repo.NewIndexFile(),
	}
}

// Add adds a new package to the index.
func (i *Index) Add(md *chart.Metadata, urls []string, createdAt time.Time) bool {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	cr := &repo.ChartVersion{
		URLs:     urls,
		Metadata: md,
		Created:  createdAt,
	}

	if ee, ok := i.file.Entries[md.Name]; !ok {
		i.file.Entries[md.Name] = repo.ChartVersions{cr}
	} else {
		cv, err := i.file.Get(md.Name, md.Version)

		if err != nil {
			// If this is the first of this package+version, add it to the index
			i.file.Entries[md.Name] = append(ee, cr)
		} else if cv.Created.After(cr.Created) {
			// If this package+version already exists, always index the latest
			*cv = *cr
		} else {
			return false
		}
	}

	i.cache = nil

	return true
}

// Get returns the metadata of a specific chart version.
func (i *Index) Get(name, version string) (*repo.ChartVersion, error) {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	return i.file.Get(name, version)
}

// WriteTo writes out a YAML serialized representation of the Index. This data
// is cached so that subsequent calls won't re-serialize an index that has not
// changed.
func (i *Index) WriteTo(w io.Writer) (n int64, err error) {
	i.mutex.RLock()
	if i.cache != nil {
		written, err := w.Write(i.cache)
		return int64(written), err
	}
	i.mutex.RUnlock()

	i.mutex.Lock()
	defer i.mutex.Unlock()

	i.file.SortEntries()
	i.file.Generated = time.Now()
	i.cache, err = yaml.Marshal(i.file)
	if err != nil {
		return 0, err
	}

	written, err := w.Write(i.cache)
	return int64(written), err
}

// Unmarshal decodes a YAML serialized repository index.
func (i *Index) Unmarshal(data []byte) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	return yaml.Unmarshal(data, &i.file)
}
