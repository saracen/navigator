package repository

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/suite"
	"k8s.io/helm/pkg/chartutil"
)

const dependencyIndexYaml = `apiVersion: v1
entries:
  mychart:
  - created: "2018-03-16T00:30:49Z"
    name: mychart
    urls:
    - foobar/mychart-0.1.0.tgz
    version: 0.1.0
generated: "2018-03-16T01:38:43.0089988Z"`

type DependencyManagerTestSuite struct {
	suite.Suite
	dm *DependencyManager
}

func (suite *DependencyManagerTestSuite) SetupSuite() {
	suite.dm = NewDependencyManager(log.NewNopLogger())
}

func (suite *DependencyManagerTestSuite) TestDownload() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(dependencyIndexYaml))
	}))
	defer ts.Close()

	dependencies := []*chartutil.Dependency{
		&chartutil.Dependency{Name: "mychart", Version: "0.1.0", Repository: ts.URL},
	}

	downloaded, err := suite.dm.Download(dependencies)
	if suite.NoError(err, "error downloading dependency") {
		suite.NotZero(len(downloaded), "downloaded zero dependencies")
	}
}

func TestDependencyManagerTestSuite(t *testing.T) {
	suite.Run(t, new(DependencyManagerTestSuite))
}
