package repository

import (
	"io"
	"testing"

	"github.com/stretchr/testify/suite"
)

type FakeRepository struct {
}

func (r *FakeRepository) URL() string {
	return ""
}

func (r *FakeRepository) Name() string {
	return "fake"
}

func (r *FakeRepository) ChartPackage(name string) (Archiver, error) {
	if name == "error" {
		return &FakeArchiver{}, ErrInvalidPackageName
	}
	return &FakeArchiver{}, nil
}

func (r *FakeRepository) Update() error {
	return nil
}

type FakeArchiver struct {
}

func (a *FakeArchiver) Archive(io.Writer) error {
	return nil
}

type RepositoryTestSuite struct {
	suite.Suite
	indexManager *IndexManager
}

func (suite *RepositoryTestSuite) TestIndexDirectoryMatch() {
	directories := IndexDirectories{
		{IndexName: "default", Name: "stable"},
		{IndexName: "default", Name: "stable/charts"},
	}

	suite.False(directories.Match("incubator"))
	suite.True(directories.Match("stable/charts/mycharts"))
	suite.True(directories.Match("stable"))
}

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}
