package repository

import (
	"errors"
	"io"
	"strings"
)

const (
	// RepositoryAnnotation is an annotation to tell navigator what repository a chart is from
	RepositoryAnnotation = "navigator/repository"

	// PathAnnotation is an annotation to tell navigator what path within a repository a chart is from
	PathAnnotation = "navigator/path"
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
