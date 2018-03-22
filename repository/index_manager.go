package repository

import (
	"errors"
	"sort"
)

var (
	// ErrIndexNotFound is raised when the named index does not exist
	ErrIndexNotFound = errors.New("index not found")
)

// IndexManager manages multiple indexes
type IndexManager struct {
	indexes map[string]*Index
}

// NewIndexManager returns a new IndexManager
func NewIndexManager() *IndexManager {
	return &IndexManager{
		indexes: make(map[string]*Index),
	}
}

// Get returns an instance by name
func (m *IndexManager) Get(name string) (*Index, error) {
	if index, ok := m.indexes[name]; ok {
		return index, nil
	}

	return nil, ErrIndexNotFound
}

// Names returns all index names assigned to the manager
func (m *IndexManager) Names() []string {
	names := make([]string, 0, len(m.indexes))
	for name := range m.indexes {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// Create creates a new named index
func (m *IndexManager) Create(name string) *Index {
	if _, ok := m.indexes[name]; !ok {
		m.indexes[name] = NewIndex()
	}

	return m.indexes[name]
}
