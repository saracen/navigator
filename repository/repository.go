package repository

import (
	"errors"
	"io"
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
