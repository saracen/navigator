package repository

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/suite"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

const dependencyIndexYaml = `apiVersion: v1
entries:
  mychart:
  - created: "2018-03-16T00:30:49Z"
    name: mychart
    urls:
    - foobar/mychart-0.1.0.tgz
    version: 0.1.0
  foochart:
    - created: "2018-03-16T00:30:49Z"
      name: foochart
      urls:
      - scheme://bad
      version: 0.1.0
  barchart:
    - created: "2018-03-16T00:30:49Z"
      name: barchart
      urls:
      - https://invalid url here
      version: 0.1.0
generated: "2018-03-16T01:38:43.0089988Z"`

type DependencyManagerTestSuite struct {
	suite.Suite
	dm *DependencyManager
}

func (suite *DependencyManagerTestSuite) SetupSuite() {
	suite.dm = NewDependencyManager(log.NewNopLogger(), NewIndexManager())
}

func (suite *DependencyManagerTestSuite) TestRemoteDownload() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.yaml":
			w.Write([]byte(dependencyIndexYaml))
		case "/foobar/mychart-0.1.0.tgz":
			w.WriteHeader(http.StatusOK)
		case "/bad-index/index.yaml":
			w.Write([]byte("apiVersion:::@"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	tests := []struct {
		url     string
		chart   string
		success bool
	}{
		{ts.URL, "mychart", true},
		{ts.URL, "unknown", false},
		{ts.URL, "foochart", false},
		{ts.URL, "barchart", false},
		{ts.URL + "/bad", "mychart", false},
		{ts.URL + "/bad-index", "mychart", false},
	}

	for idx, test := range tests {
		dependencies := []*chartutil.Dependency{
			{Name: test.chart, Version: "0.1.0", Repository: test.url},
		}

		_, err := suite.dm.Download(dependencies)
		if test.success {
			suite.NoError(err, "test index: %v", idx)
		} else {
			suite.Error(err, "test index: %v", idx)
		}
	}
}

func (suite *DependencyManagerTestSuite) TestRepositoryURL() {
	suite.dm.IndexManager().Create("empty")
	index := suite.dm.IndexManager().Create("fake")
	md := &chart.Metadata{
		Name:    "mychart",
		Version: "0.1.0",
		Annotations: map[string]string{
			RepositoryAnnotation: "fake",
			PathAnnotation:       "error",
		},
	}
	index.Add(md, []string{"invalid://test"}, time.Now())
	suite.dm.AddRepository(&FakeRepository{})

	invalids := []string{
		"https://spaced domain.com",
		"mysql://localhost",
		"https://#",
		"alias:unknown",
		"alias:empty",
		"alias:fake",
	}

	for _, invalid := range invalids {
		dependencies := []*chartutil.Dependency{
			{Name: "mychart", Version: "0.1.0", Repository: invalid},
		}

		_, err := suite.dm.Download(dependencies)
		suite.Error(err)
	}
}

func TestDependencyManagerTestSuite(t *testing.T) {
	suite.Run(t, new(DependencyManagerTestSuite))
}
