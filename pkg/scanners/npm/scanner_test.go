// pkg/scanners/npm/scanner_test.go

package npm

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNPMScanner_DetectProject(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "npm-scanner-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test package.json
	packageJSON := []byte(`{
        "name": "test-project",
        "version": "1.0.0",
        "dependencies": {
            "express": "^4.17.1"
        },
        "devDependencies": {
            "jest": "^27.0.0"
        }
    }`)

	if err := os.WriteFile(filepath.Join(tempDir, "package.json"), packageJSON, 0644); err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	scanner := NewScanner()
	ctx := context.Background()

	if !scanner.DetectProject(ctx, tempDir) {
		t.Error("Expected to detect NPM project, but didn't")
	}

	result, err := scanner.ScanDependencies(ctx, tempDir)
	if err != nil {
		t.Fatalf("Failed to scan dependencies: %v", err)
	}

	if len(result.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(result.Dependencies))
	}

	var foundExpress, foundJest bool
	for _, dep := range result.Dependencies {
		switch dep.Name {
		case "express":
			foundExpress = true
			if dep.Properties["dependencyType"] != "production" {
				t.Error("Express should be a production dependency")
			}
		case "jest":
			foundJest = true
			if dep.Properties["dependencyType"] != "development" {
				t.Error("Jest should be a development dependency")
			}
		}
	}

	if !foundExpress {
		t.Error("Express dependency not found")
	}
	if !foundJest {
		t.Error("Jest dependency not found")
	}
}
