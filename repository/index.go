package repository

import (
	"bytes"
	"compress/gzip"
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

	cache           []byte
	cacheCompressed []byte
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
		} else if cr.Created.After(cv.Created) {
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

// Count returns the number of charts and versions indexed.
func (i *Index) Count() (int, int) {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	var versions int
	for _, v := range i.file.Entries {
		versions += len(v)
	}

	return len(i.file.Entries), versions
}

// WriteTo writes out a YAML serialized representation of the Index. This data
// is cached so that subsequent calls won't re-serialize an index that has not
// changed.
func (i *Index) WriteTo(w io.Writer) (n int64, err error) {
	return i.writeTo(w, false)
}

// CompressedWriteTo is the same as WriteTo but with gzip compressed data.
func (i *Index) CompressedWriteTo(w io.Writer) (n int64, err error) {
	return i.writeTo(w, true)
}

func (i *Index) writeTo(w io.Writer, compressed bool) (n int64, err error) {
	written, err := i.writeCache(w, compressed)
	if err != nil || written > 0 {
		return int64(written), err
	}

	i.mutex.Lock()
	defer i.mutex.Unlock()

	i.file.SortEntries()
	i.file.Generated = time.Now()
	i.cache, err = yaml.Marshal(i.file)
	if err != nil {
		return 0, err
	}

	buf := new(bytes.Buffer)
	compressor, _ := gzip.NewWriterLevel(buf, gzip.BestCompression)
	if _, err = compressor.Write(i.cache); err != nil {
		return 0, err
	}

	compressor.Close()
	i.cacheCompressed = buf.Bytes()

	if compressed {
		written, err = w.Write(i.cacheCompressed)
	} else {
		written, err = w.Write(i.cache)
	}

	return int64(written), err
}

func (i *Index) writeCache(w io.Writer, compressed bool) (int, error) {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	if i.cache == nil {
		return 0, nil
	}
	if compressed {
		return w.Write(i.cacheCompressed)
	}
	return w.Write(i.cache)
}

// Unmarshal decodes a YAML serialized repository index.
func (i *Index) Unmarshal(data []byte) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	i.cache = nil

	return yaml.Unmarshal(data, i.file)
}
