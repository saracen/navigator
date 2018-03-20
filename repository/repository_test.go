package repository

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type RepositoryTestSuite struct {
	suite.Suite
	indexManager *IndexManager
}

func (suite *RepositoryTestSuite) TestIndexDirectoryMatch() {
	directories := IndexDirectories{
		{IndexName: "default", Name: "stable"},
		{IndexName: "default", Name: "stable/charts"},
	}

	suite.False(directories.Match("incubator"), "should not match")
	suite.True(directories.Match("stable/charts/mycharts"), "should match")
	suite.True(directories.Match("stable"), "should match")
}

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}
