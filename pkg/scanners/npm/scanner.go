package npm

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"deplister/pkg/scanners"
)

type NPMScanner struct {
	scanners.BaseScanner
}

type PackageJSON struct {
	Name                 string            `json:"name"`
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	PeerDependencies     map[string]string `json:"peerDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
	Workspaces           []string          `json:"workspaces"`
}

type PackageLock struct {
	Name         string                `json:"name"`
	Dependencies map[string]LockDep    `json:"dependencies"`
	Packages     map[string]PackageDep `json:"packages"`
}

type LockDep struct {
	Version   string            `json:"version"`
	Resolved  string            `json:"resolved"`
	Integrity string            `json:"integrity"`
	Requires  map[string]string `json:"requires"`
	Dev       bool              `json:"dev"`
	Optional  bool              `json:"optional"`
	Peer      bool              `json:"peer"`
}

type PackageDep struct {
	Version      string            `json:"version"`
	Resolved     string            `json:"resolved"`
	Integrity    string            `json:"integrity"`
	Dependencies map[string]string `json:"dependencies"`
	Dev          bool              `json:"dev"`
	Optional     bool              `json:"optional"`
	Peer         bool              `json:"peer"`
}

type dependencyGraph struct {
	nodes    map[string]*PackageDep
	edges    map[string][]string
	versions map[string]string
	metadata map[string]map[string]string
}

func newDependencyGraph() *dependencyGraph {
	return &dependencyGraph{
		nodes:    make(map[string]*PackageDep),
		edges:    make(map[string][]string),
		versions: make(map[string]string),
		metadata: make(map[string]map[string]string),
	}
}

func NewScanner() *NPMScanner {
	return &NPMScanner{
		BaseScanner: scanners.NewBaseScanner("npm"),
	}
}

func (s *NPMScanner) DetectProject(ctx context.Context, dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "package.json"))
	return err == nil
}

func (s *NPMScanner) ScanDependencies(ctx context.Context, dir string) (*scanners.ScanResult, error) {
	if !s.DetectProject(ctx, dir) {
		return nil, scanners.ErrProjectNotFound
	}

	pkg, err := s.readPackageJSON(dir)
	if err != nil {
		return nil, err
	}

	lockFile, err := s.readPackageLock(dir)
	if err != nil {
		return nil, err
	}

	graph := s.buildDependencyGraph(pkg, lockFile)
	if graph == nil {
		return nil, scanners.ErrInvalidProject
	}

	result := &scanners.ScanResult{
		Dependencies: make([]scanners.Dependency, 0),
		Graph: &scanners.DependencyGraph{
			Nodes: make(map[string]*scanners.Dependency),
			Edges: graph.edges,
		},
	}

	directDeps := s.getDirectDependencies(pkg)

	// Convert graph to result
	for name := range graph.nodes {
		if name == "" {
			continue
		}

		// Calculate all possible paths to this dependency
		paths := result.Graph.FindAllPaths("", name)
		minDepth := -1
		for _, path := range paths {
			if minDepth == -1 || path.Depth < minDepth {
				minDepth = path.Depth
			}
		}

		// Get all immediate parents
		var parents []string
		for parent, children := range graph.edges {
			for _, child := range children {
				if child == name && parent != "" {
					parents = append(parents, parent)
				}
			}
		}

		props := graph.metadata[name]
		if props == nil {
			props = make(map[string]string)
		}
		props["manager"] = "npm"

		// Determine if it's a direct dependency
		_, isDirect := directDeps[name]

		dependency := scanners.Dependency{
			Name:        name,
			Version:     graph.versions[name],
			Type:        "npm",
			IsDirectDep: isDirect,
			Parent:      "",
			Parents:     parents,
			Paths:       paths,
			Properties:  props,
			Depth:       minDepth,
		}

		if len(parents) > 0 {
			dependency.Parent = parents[0]
		}

		result.Dependencies = append(result.Dependencies, dependency)
		result.Graph.Nodes[name] = &dependency
	}

	if len(result.Dependencies) == 0 {
		return nil, scanners.ErrInvalidProject
	}

	return result, nil
}

func (s *NPMScanner) buildDependencyGraph(pkg *PackageJSON, lockFile *PackageLock) *dependencyGraph {
	graph := newDependencyGraph()
	directDeps := s.getDirectDependencies(pkg)

	// Handle new package-lock format (v3)
	if len(lockFile.Packages) > 0 {
		for pkgPath, dep := range lockFile.Packages {
			// Skip the root package
			if pkgPath == "" {
				continue
			}

			name := pkgPath
			if filepath.Base(pkgPath) == "node_modules" {
				continue
			}
			name = strings.TrimPrefix(name, "node_modules/")

			graph.nodes[name] = &dep
			graph.versions[name] = dep.Version

			// Store metadata
			metadata := make(map[string]string)
			if depType, ok := directDeps[name]; ok {
				metadata["dependencyType"] = depType
			} else if dep.Dev {
				metadata["dependencyType"] = "development"
			} else {
				metadata["dependencyType"] = "production"
			}

			if dep.Optional {
				metadata["optional"] = "true"
			}
			if dep.Peer {
				metadata["peer"] = "true"
			}
			if dep.Resolved != "" {
				metadata["resolved"] = dep.Resolved
			}
			if dep.Integrity != "" {
				metadata["integrity"] = dep.Integrity
			}
			graph.metadata[name] = metadata

			// Add edges from dependencies
			for depName := range dep.Dependencies {
				graph.edges[name] = append(graph.edges[name], depName)
			}

			// Add edges for direct dependencies from root
			if _, isDirect := directDeps[name]; isDirect {
				graph.edges[""] = append(graph.edges[""], name)
			}
		}
	} else {
		// Handle legacy package-lock format
		for name, lockDep := range lockFile.Dependencies {
			graph.versions[name] = lockDep.Version

			// Store metadata
			metadata := make(map[string]string)
			if depType, ok := directDeps[name]; ok {
				metadata["dependencyType"] = depType
			} else if lockDep.Dev {
				metadata["dependencyType"] = "development"
			} else {
				metadata["dependencyType"] = "production"
			}

			if lockDep.Optional {
				metadata["optional"] = "true"
			}
			if lockDep.Peer {
				metadata["peer"] = "true"
			}
			if lockDep.Resolved != "" {
				metadata["resolved"] = lockDep.Resolved
			}
			if lockDep.Integrity != "" {
				metadata["integrity"] = lockDep.Integrity
			}
			graph.metadata[name] = metadata

			// Add edges from requires
			for reqName := range lockDep.Requires {
				graph.edges[name] = append(graph.edges[name], reqName)
			}

			// Add edges for direct dependencies from root
			if _, isDirect := directDeps[name]; isDirect {
				graph.edges[""] = append(graph.edges[""], name)
			}
		}
	}

	return graph
}

func (s *NPMScanner) readPackageJSON(dir string) (*PackageJSON, error) {
	content, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil, err
	}

	var pkg PackageJSON
	if err := json.Unmarshal(content, &pkg); err != nil {
		return nil, err
	}

	return &pkg, nil
}

func (s *NPMScanner) readPackageLock(dir string) (*PackageLock, error) {
	content, err := os.ReadFile(filepath.Join(dir, "package-lock.json"))
	if err != nil {
		return nil, err
	}

	var lock PackageLock
	if err := json.Unmarshal(content, &lock); err != nil {
		return nil, err
	}

	return &lock, nil
}

func (s *NPMScanner) getDirectDependencies(pkg *PackageJSON) map[string]string {
	directDeps := make(map[string]string)
	for name := range pkg.Dependencies {
		directDeps[name] = "production"
	}
	for name := range pkg.DevDependencies {
		directDeps[name] = "development"
	}
	for name := range pkg.PeerDependencies {
		directDeps[name] = "peer"
	}
	for name := range pkg.OptionalDependencies {
		directDeps[name] = "optional"
	}
	return directDeps
}
