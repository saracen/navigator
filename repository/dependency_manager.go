package repository

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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

	repositories      map[string]*singleflightIndex
	repositoriesMutex sync.Mutex
}

type singleflightIndex struct {
	*Index
	sync.Mutex
}

// NewDependencyManager returns a new dependency manager.
func NewDependencyManager(logger log.Logger) *DependencyManager {
	return &DependencyManager{
		client:       &http.Client{Timeout: time.Second * 10},
		logger:       logger,
		repositories: make(map[string]*singleflightIndex),
	}
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
		repository, err := url.Parse(dep.Repository)
		if err != nil {
			return nil, fmt.Errorf("Chart dependency %v:%v has invalid repository: %v", dep.Name, dep.Version, dep.Repository)
		}

		if repository.Scheme != "http" && repository.Scheme != "https" {
			return nil, fmt.Errorf("Chart dependency %v:%v has unsupported repository scheme: %v://", dep.Name, dep.Version, repository.Scheme)
		}

		wg.Add(1)
		go func(idx int, dep *chartutil.Dependency) {
			defer wg.Done()

			url, err := dm.getPackageURL(dep)
			if err != nil {
				states[idx].err = err
				return
			}

			archive, err := dm.download(ctx, url)
			if err != nil {
				states[idx].err = err
				cancel()
				return
			}

			states[idx].data = archive
		}(idx, dep)
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

func (dm *DependencyManager) download(ctx context.Context, url string) (body []byte, err error) {
	defer func(begin time.Time) {
		if err == nil {
			level.Info(dm.logger).Log("event", "download", "url", url, "took", time.Since(begin))
		} else {
			level.Error(dm.logger).Log("event", "download", "url", url, "took", time.Since(begin), "err", err)
		}
	}(time.Now())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := dm.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func (dm *DependencyManager) repository(repository string) *singleflightIndex {
	dm.repositoriesMutex.Lock()
	defer dm.repositoriesMutex.Unlock()

	// Initialise the repository with an empty index. The repository is fetched
	// and updated when a package required does not exist in it.
	if _, ok := dm.repositories[repository]; !ok {
		dm.repositories[repository] = &singleflightIndex{Index: NewIndex()}
	}

	return dm.repositories[repository]
}

func (dm *DependencyManager) getPackageURL(dep *chartutil.Dependency) (string, error) {
	index := dm.repository(dep.Repository)

	index.Lock()
	defer index.Unlock()

	// attempt to get chart dependency
	chart, err := index.Get(dep.Name, dep.Version)

	// update repository if dependency doesn't exist
	if err != nil {
		body, err := dm.download(context.TODO(), fmt.Sprintf("%s/index.yaml", strings.TrimSuffix(dep.Repository, "/")))
		if err != nil {
			return "", err
		}

		if err := index.Unmarshal(body); err != nil {
			return "", err
		}
	}

	// make another attempt to fetch chart dependency
	chart, err = index.Get(dep.Name, dep.Version)
	if err != nil {
		return "", err
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
		return "", fmt.Errorf("Chart dependency %v:%v has invalid package url: %v", dep.Name, dep.Version, rawChartURL)
	}

	return chartURL.String(), nil
}
