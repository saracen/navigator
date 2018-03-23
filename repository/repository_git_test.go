package repository

import (
	"testing"

	"bytes"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/suite"
	"k8s.io/helm/pkg/chartutil"
)

type RepositoryGitTestSuite struct {
	suite.Suite
	indexManager *IndexManager
	repo         Repository
}

var testCharts = []struct {
	Name    string
	Version string
}{
	{"mychart", "0.1.0"},
	{"mydependencychart", "0.1.0"},
}

func (suite *RepositoryGitTestSuite) SetupSuite() {
	suite.indexManager = NewIndexManager()
	suite.indexManager.Create("default")

	logger := log.NewNopLogger()
	dependencyManager := NewDependencyManager(logger, suite.indexManager)

	suite.repo = NewGitBackedRepository(logger, dependencyManager, "repo", "../.git", []IndexDirectory{{Name: "repository/testdata/charts", IndexName: "default"}})

	// clone
	suite.Nil(suite.repo.Update())

	// fetch
	suite.Nil(suite.repo.Update())
}

func (suite *RepositoryGitTestSuite) TestURL() {
	suite.Equal("../.git", suite.repo.URL())
}

func (suite *RepositoryGitTestSuite) TestChartPackage() {
	index, err := suite.indexManager.Get("default")
	if !suite.NoError(err) {
		return
	}

	for _, testChart := range testCharts {
		chart, err := index.Get(testChart.Name, testChart.Version)
		if suite.NoError(err) {
			_, name := repoCommitChartFromPath(chart.URLs[0])

			archiver, err := suite.repo.ChartPackage(name)
			if suite.NoError(err) {
				buf := new(bytes.Buffer)

				if suite.NoError(archiver.Archive(buf)) {
					_, err := chartutil.LoadArchive(buf)
					suite.NoError(err)
				}
			}
		}
	}
}

func TestRepositoryGitTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryGitTestSuite))
}
