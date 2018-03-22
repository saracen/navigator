package repository

import (
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

var (
	// ErrInvalidPackageName is raised when a chart package cannot be found
	ErrInvalidPackageName = errors.New("invalid package name")
	// ErrRepositoryNotFound is raised when the helm repository referenced in a package link cannot be found
	ErrRepositoryNotFound = errors.New("repository not found")
)

// Archiver produces a compressed tar archive.
type Archiver interface {
	Archive(io.Writer) error
}

// Repository represents a generic repository.
type Repository interface {
	URL() string
	Name() string
	ChartPackage(string) (Archiver, error)
	Update() error
}

// IndexDirectory maps a directory to a named index
type IndexDirectory struct {
	IndexName string
	Name      string
}

// IndexDirectories is a slice of IndexDirectory
type IndexDirectories []IndexDirectory

// Match checks whether a path matches any of the index directories
func (id IndexDirectories) Match(path string) bool {
	for _, directory := range id {
		if strings.HasPrefix(path, directory.Name) {
			return true
		}
	}
	return false
}

// pathHeadTail is similar to path.Split, but returns the first component of the path (head) and then everything else as the tail
func pathHeadTail(p string) (string, string) {
	i := strings.Index(p, "/")
	if i < 0 {
		return p, ""
	}
	return p[:i], p[i+1:]
}

// repoCommitChartFromPath returns the repository name and commit chart path from a resource path.
func repoCommitChartFromPath(p string) (string, string) {
	head, tail := pathHeadTail(p)
	return head, path.Dir(tail)
}

// repoCommitChartToPath returns a path containing the repository and commit chart directory
func repoCommitChartToPath(repo, commit, directory, name, version string) string {
	return path.Join(repo, commit, directory, fmt.Sprintf("%s-%s.tgz", name, version))
}
