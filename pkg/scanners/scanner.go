package scanners

import (
	"context"
	"errors"
)

// Dependency represents a single package dependency
type Dependency struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Type        string            `json:"type"`
	Properties  map[string]string `json:"properties,omitempty"`
	IsDirectDep bool              `json:"is_direct"`
}

type ScanResult struct {
	Dependencies []Dependency `json:"dependencies"`
	Errors       []error      `json:"errors,omitempty"`
}

// Scanner defines the interface that all package manager scanners must implement
type Scanner interface {
	// Name returns the name of the package manager (e.g., "npm", "go", "pip")
	Name() string

	// DetectProject checks if the given directory contains a project of this type
	DetectProject(ctx context.Context, dir string) bool

	// ScanDependencies scans the project directory and returns all dependencies
	ScanDependencies(ctx context.Context, dir string) (*ScanResult, error)
}

var (
	ErrProjectNotFound = errors.New("project configuration not found")
	ErrInvalidProject  = errors.New("invalid project configuration")
	ErrScanFailed      = errors.New("dependency scan failed")
)

type BaseScanner struct {
	name string
}

func NewBaseScanner(name string) BaseScanner {
	return BaseScanner{name: name}
}

func (b BaseScanner) Name() string {
	return b.name
}
