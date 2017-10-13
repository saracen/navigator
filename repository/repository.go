package repository

import (
	"io"
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