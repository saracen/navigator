package repository

import (
	"path"
	"testing"

	"bytes"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/suite"
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

	suite.repo = NewGitBackedRepository(logger, dependencyManager, "", "../.git", []IndexDirectory{{Name: "repository/testdata/charts", IndexName: "default"}})

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
			name := path.Dir(chart.URLs[0])

			archiver, err := suite.repo.ChartPackage(name)
			if suite.NoError(err) {
				suite.Nil(archiver.Archive(new(bytes.Buffer)))
			}
		}
	}
}

func TestRepositoryGitTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryGitTestSuite))
}
