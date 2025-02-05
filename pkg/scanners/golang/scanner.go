package golang

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/santoshdahal12/deplister/pkg/scanners"
)

type GoScanner struct {
	scanners.BaseScanner
}

type ModuleInfo struct {
	Path     string       `json:"Path"`
	Version  string       `json:"Version"`
	Main     bool         `json:"Main"`
	Indirect bool         `json:"Indirect"`
	Replace  *ModuleInfo  `json:"Replace,omitempty"`
	Requires []ModuleInfo `json:"Require,omitempty"`
}

type dependencyGraph struct {
	nodes    map[string]*ModuleInfo
	edges    map[string][]string
	versions map[string]string
	metadata map[string]map[string]string
}

func newDependencyGraph() *dependencyGraph {
	return &dependencyGraph{
		nodes:    make(map[string]*ModuleInfo),
		edges:    make(map[string][]string),
		versions: make(map[string]string),
		metadata: make(map[string]map[string]string),
	}
}

func NewScanner() *GoScanner {
	return &GoScanner{
		BaseScanner: scanners.NewBaseScanner("go"),
	}
}

func (s *GoScanner) DetectProject(ctx context.Context, dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil
}

func (s *GoScanner) ScanDependencies(ctx context.Context, dir string) (*scanners.ScanResult, error) {
	if !s.DetectProject(ctx, dir) {
		return nil, scanners.ErrProjectNotFound
	}

	graph, err := s.buildDependencyGraph(ctx, dir)
	if err != nil {
		return nil, err
	}

	mainModule := s.findMainModule(graph)
	if mainModule == "" {
		return nil, scanners.ErrInvalidProject
	}

	result := &scanners.ScanResult{
		Dependencies: make([]scanners.Dependency, 0),
		Graph: &scanners.DependencyGraph{
			Nodes: make(map[string]*scanners.Dependency),
			Edges: graph.edges,
		},
	}

	// Get direct dependencies from go.mod
	directDeps, err := s.getDirectDependencies(dir)
	if err != nil {
		return nil, err
	}

	for modPath, info := range graph.nodes {
		if modPath == mainModule {
			continue
		}

		// Calculate all possible paths to this dependency
		paths := result.Graph.FindAllPaths(mainModule, modPath)
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
				if child == modPath && parent != mainModule {
					parents = append(parents, parent)
				}
			}
		}

		props := graph.metadata[modPath]
		if props == nil {
			props = make(map[string]string)
		}
		props["manager"] = "go"

		// Set dependency type
		if !info.Indirect {
			props["dependencyType"] = "direct"
		} else {
			props["dependencyType"] = "indirect"
		}

		if info.Replace != nil {
			props["replaced_by"] = info.Replace.Path
			props["replaced_version"] = info.Replace.Version
		}

		dependency := scanners.Dependency{
			Name:        info.Path,
			Version:     info.Version,
			Type:        "go",
			IsDirectDep: !info.Indirect && directDeps[modPath], // Use both Indirect flag and direct deps check
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
		result.Graph.Nodes[modPath] = &dependency
	}

	if len(result.Dependencies) == 0 {
		return nil, scanners.ErrInvalidProject
	}

	return result, nil
}

// getDirectDependencies reads go.mod file and returns a map of direct dependencies
func (s *GoScanner) getDirectDependencies(dir string) (map[string]bool, error) {
	goModPath := filepath.Join(dir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, err
	}

	directDeps := make(map[string]bool)
	lines := strings.Split(string(content), "\n")
	inRequireBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Handle single-line requires
		if strings.HasPrefix(line, "require ") && !strings.HasSuffix(line, "(") {
			fields := strings.Fields(line)[1:]
			if len(fields) > 0 {
				dep := fields[0]
				if !strings.Contains(line, "// indirect") {
					directDeps[dep] = true
				}
			}
			continue
		}

		// Handle require blocks
		if line == "require (" {
			inRequireBlock = true
			continue
		}
		if line == ")" {
			inRequireBlock = false
			continue
		}

		if inRequireBlock {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				dep := fields[0]
				if !strings.Contains(line, "// indirect") {
					directDeps[dep] = true
				}
			}
		}
	}

	return directDeps, nil
}

func (s *GoScanner) buildDependencyGraph(ctx context.Context, dir string) (*dependencyGraph, error) {
	graph := newDependencyGraph()

	listCmd := exec.CommandContext(ctx, "go", "list", "-m", "-json", "all")
	listCmd.Dir = dir
	listOutput, err := listCmd.Output()
	if err != nil {
		return nil, scanners.ErrScanFailed
	}

	decoder := json.NewDecoder(strings.NewReader(string(listOutput)))
	for decoder.More() {
		var info ModuleInfo
		if err := decoder.Decode(&info); err != nil {
			return nil, scanners.ErrInvalidProject
		}
		graph.nodes[info.Path] = &info
		graph.versions[info.Path] = info.Version

		// Store metadata
		metadata := make(map[string]string)
		if info.Indirect {
			metadata["dependencyType"] = "indirect"
		} else {
			metadata["dependencyType"] = "direct"
		}
		if info.Replace != nil {
			metadata["replaced"] = "true"
		}
		graph.metadata[info.Path] = metadata
	}

	graphCmd := exec.CommandContext(ctx, "go", "mod", "graph")
	graphCmd.Dir = dir
	graphOutput, err := graphCmd.Output()
	if err != nil {
		return nil, scanners.ErrScanFailed
	}

	scanner := bufio.NewScanner(strings.NewReader(string(graphOutput)))
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) != 2 {
			continue
		}

		fromMod := parts[0]
		toMod := parts[1]

		fromPath := strings.Split(fromMod, "@")[0]
		toPath := strings.Split(toMod, "@")[0]

		graph.edges[fromPath] = append(graph.edges[fromPath], toPath)
	}

	return graph, nil
}

func (s *GoScanner) findMainModule(graph *dependencyGraph) string {
	for path, info := range graph.nodes {
		if info.Main {
			return path
		}
	}
	return ""
}
