package scanners

import (
	"context"
	"errors"
)

// Common errors
var (
	ErrProjectNotFound = errors.New("project not found")
	ErrInvalidProject  = errors.New("invalid project")
	ErrScanFailed      = errors.New("scan failed")
)

// DependencyPath represents a path from root to the dependency
type DependencyPath struct {
	Path  []string // Ordered list of dependencies from root to target
	Depth int      // Depth in the dependency tree
}

// Dependency represents a single project dependency
type Dependency struct {
	Name        string            // Name of the dependency
	Version     string            // Version of the dependency
	Type        string            // Type of dependency (npm, go, etc.)
	IsDirectDep bool              // Whether this is a direct dependency
	Parent      string            // Immediate parent dependency
	Parents     []string          // All direct parent dependencies
	Paths       []DependencyPath  // All possible paths to this dependency
	Properties  map[string]string // Additional properties specific to the dependency type
	Depth       int               // Minimum depth in the dependency tree
}

// ScanResult contains the results of a dependency scan
type ScanResult struct {
	Dependencies []Dependency
	Graph        *DependencyGraph
}

// DependencyGraph represents the complete dependency structure
type DependencyGraph struct {
	Nodes map[string]*Dependency
	Edges map[string][]string
}

// Scanner interface defines the methods required for a dependency scanner
type Scanner interface {
	DetectProject(ctx context.Context, dir string) bool
	ScanDependencies(ctx context.Context, dir string) (*ScanResult, error)
	GetType() string
}

// BaseScanner provides common functionality for scanners
type BaseScanner struct {
	scannerType string
}

// NewBaseScanner creates a new base scanner with the specified type
func NewBaseScanner(scannerType string) BaseScanner {
	return BaseScanner{
		scannerType: scannerType,
	}
}

// GetType returns the scanner type
func (s BaseScanner) GetType() string {
	return s.scannerType
}

// Helper functions for graph operations
func (g *DependencyGraph) FindAllPaths(from, to string) []DependencyPath {
	visited := make(map[string]bool)
	var paths []DependencyPath
	g.findPaths(from, to, []string{}, visited, &paths)
	return paths
}

func (g *DependencyGraph) findPaths(current, target string, path []string, visited map[string]bool, results *[]DependencyPath) {
	if visited[current] {
		return
	}

	newPath := append(path, current)

	if current == target {
		*results = append(*results, DependencyPath{
			Path:  append([]string{}, newPath...),
			Depth: len(newPath) - 1,
		})
		return
	}

	visited[current] = true
	for _, next := range g.Edges[current] {
		g.findPaths(next, target, newPath, visited, results)
	}
	visited[current] = false
}

// CalculateDepth returns the minimum depth of a dependency
func (g *DependencyGraph) CalculateDepth(name string) int {
	visited := make(map[string]bool)
	return g.calculateMinDepth(name, visited)
}

func (g *DependencyGraph) calculateMinDepth(name string, visited map[string]bool) int {
	if visited[name] {
		return -1
	}

	visited[name] = true
	minDepth := -1

	for parent, children := range g.Edges {
		for _, child := range children {
			if child == name {
				parentDepth := g.calculateMinDepth(parent, visited)
				if parentDepth != -1 {
					depth := parentDepth + 1
					if minDepth == -1 || depth < minDepth {
						minDepth = depth
					}
				}
			}
		}
	}

	visited[name] = false
	return minDepth
}
