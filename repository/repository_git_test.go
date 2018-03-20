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
	suite.Nil(suite.repo.Update(), "error updating git repository")
	suite.Nil(suite.repo.Update(), "error fetching new git updates")
}

func (suite *RepositoryGitTestSuite) TestURL() {
	suite.Equal("../.git", suite.repo.URL(), "git url not as expected")
}

func (suite *RepositoryGitTestSuite) TestChartPackage() {
	index, err := suite.indexManager.Get("default")
	if !suite.NoError(err, "error getting default index") {
		return
	}

	for _, testChart := range testCharts {
		chart, err := index.Get(testChart.Name, testChart.Version)
		if suite.NoError(err, "error checking index for package") {
			name := path.Dir(chart.URLs[0])

			archiver, err := suite.repo.ChartPackage(name)
			if suite.NoError(err, "error finding chart package in repository") {
				suite.Nil(archiver.Archive(new(bytes.Buffer)), "no error archiving chart package")
			}
		}
	}
}

func TestRepositoryGitTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryGitTestSuite))
}
