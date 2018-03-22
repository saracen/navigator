package repository

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type IndexManagerTestSuite struct {
	suite.Suite
	indexManager *IndexManager
}

func (suite *IndexManagerTestSuite) SetupSuite() {
	suite.indexManager = NewIndexManager()
	suite.indexManager.Create("default")
}

func (suite *IndexManagerTestSuite) TestGetExisting() {
	_, err := suite.indexManager.Get("default")
	suite.NoError(err, "error getting existing index")
}

func (suite *IndexManagerTestSuite) TestGetMissing() {
	_, err := suite.indexManager.Get("stable")
	suite.Error(err, "stable index should not exist")
}

func (suite *IndexManagerTestSuite) TestNames() {
	suite.indexManager.Create("new-index")
	suite.Len(suite.indexManager.Names(), 2)
}

func TestIndexManagerTestSuite(t *testing.T) {
	suite.Run(t, new(IndexManagerTestSuite))
}
