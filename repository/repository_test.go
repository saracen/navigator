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

func (suite *RepositoryTestSuite) TestPathUtility() {
	headtails := [][3]string{
		{"a/b", "a", "b"},
		{"a/b/c", "a", "b/c"},
		{"a", "a", ""},
		{"", "", ""},
		{"/", "", ""},
	}

	for _, headtail := range headtails {
		head, tail := pathHeadTail(headtail[0])
		suite.Equal(headtail[1], head, headtail[0])
		suite.Equal(headtail[2], tail, headtail[0])
	}

	path := "/repo/commit/long/path/to/chart-0.1.0.tgz"
	repo, chartpath := repoCommitChartFromPath(path)
	suite.Equal("repo", repo)
	suite.Equal("commit/long/path/to", chartpath)
	suite.Equal(path, repoCommitChartToPath("repo", "commit", "long/path/to", "chart", "0.1.0"))
}

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}
