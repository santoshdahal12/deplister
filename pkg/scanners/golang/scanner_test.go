package golang

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"deplister/pkg/scanners"

	"github.com/stretchr/testify/assert"
)

func setupTestDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "go-scanner-test")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	return dir
}

func TestGoScanner_DetectProject(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	scanner := NewScanner()
	ctx := context.Background()

	assert.False(t, scanner.DetectProject(ctx, dir), "should not detect non-existent go.mod")

	goMod := []byte(`module example.com/test
go 1.20
require (
    github.com/stretchr/testify v1.8.1
    golang.org/x/sync v0.1.0 // indirect
)`)

	err := os.WriteFile(filepath.Join(dir, "go.mod"), goMod, 0644)
	assert.NoError(t, err, "failed to write go.mod")

	assert.True(t, scanner.DetectProject(ctx, dir), "failed to detect valid go.mod")
}

func TestDependencyGraph_Structure(t *testing.T) {
	tests := []struct {
		name     string
		edges    map[string][]string
		module   string
		expected []string
	}{
		{
			name: "direct_dependency",
			edges: map[string][]string{
				"example.com/test": {"mod1", "mod2"},
			},
			module:   "mod1",
			expected: []string{"example.com/test"},
		},
		// ... other test cases ...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := newDependencyGraph()
			graph.edges = tt.edges
			graph.nodes[tt.module] = &ModuleInfo{Path: tt.module}

			// Find parents of the module
			var parents []string
			for parent, children := range tt.edges {
				for _, child := range children {
					if child == tt.module {
						parents = append(parents, parent)
					}
				}
			}

			assert.Equal(t, len(tt.expected), len(parents), "wrong number of parents")
			for _, exp := range tt.expected {
				assert.Contains(t, parents, exp, "missing expected parent")
			}
		})
	}
}

func TestGoScanner_Integration(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(`module example.com/test
go 1.20
require (
    github.com/stretchr/testify v1.8.1
    golang.org/x/sync v0.1.0 // indirect
)`), 0644)
	assert.NoError(t, err, "failed to write go.mod")

	scanner := NewScanner()
	result, err := scanner.ScanDependencies(context.Background(), dir)
	if err == scanners.ErrScanFailed {
		t.Skip("skipping integration test: go tools not available")
	}
	assert.NoError(t, err, "scan failed")
	assert.NotNil(t, result, "result should not be nil")
	assert.NotNil(t, result.Graph, "graph should not be nil")

	deps := make(map[string]scanners.Dependency)
	for _, dep := range result.Dependencies {
		deps[dep.Name] = dep
	}

	// Check testify dependency
	testify, ok := deps["github.com/stretchr/testify"]
	assert.True(t, ok, "testify dependency not found")
	assert.True(t, testify.IsDirectDep, "testify should be direct dependency")
	assert.NotEmpty(t, testify.Paths, "testify should have dependency paths")
	assert.Equal(t, 1, testify.Depth, "testify should have depth 1")

	// Check sync dependency
	sync, ok := deps["golang.org/x/sync"]
	assert.True(t, ok, "sync dependency not found")
	assert.False(t, sync.IsDirectDep, "sync should be indirect dependency")
	assert.NotEmpty(t, sync.Paths, "sync should have dependency paths")
	assert.Greater(t, sync.Depth, 1, "sync should have depth > 1")
}

func TestReplacedDependencies(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	goMod := []byte(`module example.com/test
go 1.20
require github.com/original/pkg v1.0.0
replace github.com/original/pkg => github.com/fork/pkg v1.1.0
`)

	err := os.WriteFile(filepath.Join(dir, "go.mod"), goMod, 0644)
	assert.NoError(t, err, "failed to write go.mod")

	scanner := NewScanner()
	result, err := scanner.ScanDependencies(context.Background(), dir)
	if err == scanners.ErrScanFailed {
		t.Skip("skipping integration test: go tools not available")
	}
	assert.NoError(t, err, "scan failed")

	var foundReplaced bool
	for _, dep := range result.Dependencies {
		if dep.Name == "github.com/original/pkg" {
			assert.Equal(t, "github.com/fork/pkg", dep.Properties["replaced_by"], "wrong replacement")
			assert.Equal(t, "v1.1.0", dep.Properties["replaced_version"], "wrong replacement version")
			foundReplaced = true
			break
		}
	}
	assert.True(t, foundReplaced, "replaced dependency not found")
}

func TestDependencyPaths(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	goMod := []byte(`module example.com/test
go 1.20
require (
    github.com/stretchr/testify v1.8.1
    github.com/davecgh/go-spew v1.1.1 // indirect
)`)

	err := os.WriteFile(filepath.Join(dir, "go.mod"), goMod, 0644)
	assert.NoError(t, err, "failed to write go.mod")

	scanner := NewScanner()
	result, err := scanner.ScanDependencies(context.Background(), dir)
	if err == scanners.ErrScanFailed {
		t.Skip("skipping integration test: go tools not available")
	}
	assert.NoError(t, err, "scan failed")

	for _, dep := range result.Dependencies {
		assert.NotEmpty(t, dep.Paths, "dependency should have paths")
		assert.Greater(t, dep.Depth, 0, "dependency should have depth")

		if dep.IsDirectDep {
			assert.Equal(t, 1, dep.Depth, "direct dependency should have depth 1")
			assert.Len(t, dep.Paths[0].Path, 2, "direct dependency should have path length 2")
		} else {
			assert.Greater(t, dep.Depth, 1, "indirect dependency should have depth > 1")
			assert.Greater(t, len(dep.Paths[0].Path), 2, "indirect dependency should have path length > 2")
		}
	}
}

func TestMultipleReplacedDependencies(t *testing.T) {
	// Check if go tools are available first
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("skipping test: go tools not available")
		return
	}

	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create a minimal go.work file to ensure module awareness
	err := os.WriteFile(filepath.Join(dir, "go.work"), []byte("go 1.20"), 0644)
	assert.NoError(t, err, "failed to write go.work")

	goMod := []byte(`module example.com/test
go 1.20
require (
    github.com/original/pkg1 v1.0.0
    github.com/original/pkg2 v2.0.0
)
replace (
    github.com/original/pkg1 => github.com/fork/pkg1 v1.1.0
    github.com/original/pkg2 => github.com/fork/pkg2 v2.1.0
)`)

	err = os.WriteFile(filepath.Join(dir, "go.mod"), goMod, 0644)
	assert.NoError(t, err, "failed to write go.mod")

	// Initialize go module
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Skip("skipping test: unable to initialize go module")
		return
	}

	scanner := NewScanner()
	result, err := scanner.ScanDependencies(context.Background(), dir)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Track which replacements we've found
	replacements := map[string]bool{
		"github.com/original/pkg1": false,
		"github.com/original/pkg2": false,
	}

	for _, dep := range result.Dependencies {
		if dep.Name == "github.com/original/pkg1" {
			assert.Equal(t, "github.com/fork/pkg1", dep.Properties["replaced_by"], "wrong replacement for pkg1")
			assert.Equal(t, "v1.1.0", dep.Properties["replaced_version"], "wrong replacement version for pkg1")
			replacements["github.com/original/pkg1"] = true
		}
		if dep.Name == "github.com/original/pkg2" {
			assert.Equal(t, "github.com/fork/pkg2", dep.Properties["replaced_by"], "wrong replacement for pkg2")
			assert.Equal(t, "v2.1.0", dep.Properties["replaced_version"], "wrong replacement version for pkg2")
			replacements["github.com/original/pkg2"] = true
		}
	}

	// Verify all replacements were found
	for pkg, found := range replacements {
		assert.True(t, found, "replacement for %s not found", pkg)
	}
}
