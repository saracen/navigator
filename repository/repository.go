package repository

import (
	"errors"
	"io"
)

var (
	ErrInvalidPackageName = errors.New("invalid package name")
	ErrRepositoryNotFound = errors.New("repository not found")
)

// Archiver produces a compressed tar archive.
type Archiver interface {
	Archive(io.Writer) error
}

// Repository represents a generic repository.
type Repository interface {
	URL() string
	ChartPackage(string) (Archiver, error)
	Update() error
}
