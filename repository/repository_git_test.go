package repository

import (
	"testing"

	"bytes"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/suite"
)

type RepositoryGitTestSuite struct {
	suite.Suite
	index *Index
	repo  Repository
}

func (suite *RepositoryGitTestSuite) SetupSuite() {
	suite.index = NewIndex()
	suite.repo = NewGitBackedRepository(log.NewNopLogger(), suite.index, "", "../.git", []string{"repository/testdata/charts"})
	suite.Nil(suite.repo.Update(), "error updating git repository")
}

func (suite *RepositoryGitTestSuite) TestURL() {
	suite.Equal("../.git", suite.repo.URL(), "git url not as expected")
}

func (suite *RepositoryGitTestSuite) TestChartPackage() {
	chart, err := suite.index.Get("mychart", "0.1.0")
	if suite.NoError(err, "error checking index for package") {
		// name is the URL of the chart, minus mychart-0.1.0.tgz
		name := strings.TrimSuffix(chart.URLs[0], "/mychart-0.1.0.tgz")

		archiver, err := suite.repo.ChartPackage(name)
		if suite.NoError(err, "error finding chart package in repository") {
			suite.Nil(archiver.Archive(new(bytes.Buffer)), "no error archiving chart package")
		}
	}
}
func TestRepositoryGitTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryGitTestSuite))
}
