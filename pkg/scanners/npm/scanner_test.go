package npm

import (
	"context"
	"deplister/pkg/scanners"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNPMScanner_DetectProject(t *testing.T) {
	dir := t.TempDir()

	scanner := NewScanner()

	// Should return false when no package.json exists
	assert.False(t, scanner.DetectProject(context.Background(), dir))

	// Create empty package.json
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)
	assert.NoError(t, err)

	// Should return true when package.json exists
	assert.True(t, scanner.DetectProject(context.Background(), dir))
}

func TestNPMScanner_ScanDependencies(t *testing.T) {
	dir := t.TempDir()

	// Write test files with nested dependencies
	packageJSON := `{
		"name": "test-project",
		"dependencies": {
			"react": "^18.2.0",
			"react-dom": "^18.2.0"
		},
		"devDependencies": {
			"prettier": "^1.19.1"
		}
	}`

	packageLockJSON := `{
		"name": "test-project",
		"packages": {
			"": {
				"name": "test-project"
			},
			"node_modules/react": {
				"version": "18.2.0",
				"resolved": "https://registry.npmjs.org/react/-/react-18.2.0.tgz",
				"integrity": "sha512-abcd1234",
				"dependencies": {
					"loose-envify": "^1.1.0"
				}
			},
			"node_modules/react-dom": {
				"version": "18.2.0",
				"resolved": "https://registry.npmjs.org/react-dom/-/react-dom-18.2.0.tgz",
				"integrity": "sha512-efgh5678",
				"dependencies": {
					"loose-envify": "^1.1.0",
					"scheduler": "^0.23.0"
				}
			},
			"node_modules/loose-envify": {
				"version": "1.4.0",
				"resolved": "https://registry.npmjs.org/loose-envify/-/loose-envify-1.4.0.tgz",
				"integrity": "sha512-xyz123",
				"dependencies": {
					"js-tokens": "^3.0.0"
				}
			},
			"node_modules/js-tokens": {
				"version": "4.0.0",
				"resolved": "https://registry.npmjs.org/js-tokens/-/js-tokens-4.0.0.tgz",
				"integrity": "sha512-abc890"
			},
			"node_modules/scheduler": {
				"version": "0.23.0",
				"resolved": "https://registry.npmjs.org/scheduler/-/scheduler-0.23.0.tgz",
				"integrity": "sha512-def567",
				"dependencies": {
					"loose-envify": "^1.1.0"
				}
			},
			"node_modules/prettier": {
				"version": "1.19.1",
				"resolved": "https://registry.npmjs.org/prettier/-/prettier-1.19.1.tgz",
				"integrity": "sha512-xyz789",
				"dev": true
			}
		}
	}`

	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0644)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(packageLockJSON), 0644)
	assert.NoError(t, err)

	scanner := NewScanner()
	result, err := scanner.ScanDependencies(context.Background(), dir)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify we got all dependencies (including nested ones)
	assert.Len(t, result.Dependencies, 6) // react, react-dom, prettier, loose-envify, js-tokens, scheduler

	// Helper to find dependency by name
	findDep := func(name string) *scanners.Dependency {
		for _, dep := range result.Dependencies {
			if dep.Name == name {
				return &dep
			}
		}
		return nil
	}

	// Check direct dependencies
	reactDep := findDep("react")
	assert.NotNil(t, reactDep)
	assert.Equal(t, "18.2.0", reactDep.Version)
	assert.Equal(t, "npm", reactDep.Type)
	assert.True(t, reactDep.IsDirectDep)
	assert.Equal(t, "production", reactDep.Properties["dependencyType"])
	assert.Equal(t, "https://registry.npmjs.org/react/-/react-18.2.0.tgz", reactDep.Properties["resolved"])
	assert.Equal(t, "sha512-abcd1234", reactDep.Properties["integrity"])
	assert.Equal(t, 1, reactDep.Depth) // Direct dependency has depth 1

	prettierDep := findDep("prettier")
	assert.NotNil(t, prettierDep)
	assert.Equal(t, "1.19.1", prettierDep.Version)
	assert.Equal(t, "npm", prettierDep.Type)
	assert.True(t, prettierDep.IsDirectDep)
	assert.Equal(t, "development", prettierDep.Properties["dependencyType"])
	assert.Equal(t, 1, prettierDep.Depth)

	// Check nested dependencies
	looseEnvifyDep := findDep("loose-envify")
	assert.NotNil(t, looseEnvifyDep)
	assert.False(t, looseEnvifyDep.IsDirectDep)
	assert.Equal(t, 2, looseEnvifyDep.Depth)
	assert.Contains(t, looseEnvifyDep.Parents, "react")
	assert.Contains(t, looseEnvifyDep.Parents, "react-dom")

	jsTokensDep := findDep("js-tokens")
	assert.NotNil(t, jsTokensDep)
	assert.False(t, jsTokensDep.IsDirectDep)
	assert.Equal(t, 3, jsTokensDep.Depth)
	assert.Contains(t, jsTokensDep.Parents, "loose-envify")

	// Check dependency paths
	reactPaths := reactDep.Paths
	assert.Len(t, reactPaths, 1)
	assert.Equal(t, []string{"", "react"}, reactPaths[0].Path)

	jsTokensPaths := jsTokensDep.Paths
	assert.Greater(t, len(jsTokensPaths), 0)
	// Should have multiple paths through react and react-dom
	foundReactPath := false
	foundReactDomPath := false
	for _, path := range jsTokensPaths {
		if len(path.Path) == 4 && path.Path[1] == "react" {
			foundReactPath = true
		}
		if len(path.Path) == 4 && path.Path[1] == "react-dom" {
			foundReactDomPath = true
		}
	}
	assert.True(t, foundReactPath, "Should have path through react")
	assert.True(t, foundReactDomPath, "Should have path through react-dom")

	// Verify graph structure
	assert.NotNil(t, result.Graph)
	assert.Contains(t, result.Graph.Edges["react"], "loose-envify")
	assert.Contains(t, result.Graph.Edges["loose-envify"], "js-tokens")
}
