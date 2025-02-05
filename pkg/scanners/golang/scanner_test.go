// pkg/scanners/golang/scanner_test.go

package golang

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"deplister/pkg/scanners"
)

func TestGoScanner_DetectProject(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "go-scanner-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	goMod := []byte(`module example.com/test

go 1.20

require (
    github.com/stretchr/testify v1.8.1
    golang.org/x/sync v0.1.0 // indirect
)
`)

	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), goMod, 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	scanner := NewScanner()
	ctx := context.Background()

	if !scanner.DetectProject(ctx, tempDir) {
		t.Error("Expected to detect Go project, but didn't")
	}

	if scanner.DetectProject(ctx, filepath.Join(tempDir, "nonexistent")) {
		t.Error("Expected not to detect Go project in non-existent directory")
	}
}

func TestGoScanner_Name(t *testing.T) {
	scanner := NewScanner()
	if scanner.Name() != "go" {
		t.Errorf("Expected scanner name to be 'go', got '%s'", scanner.Name())
	}
}

func TestGoScanner_ScanDependencies(t *testing.T) {
	// This test requires a real Go environment and valid go.mod
	// We'll test the error cases and basic functionality
	scanner := NewScanner()
	ctx := context.Background()

	t.Run("non-existent directory", func(t *testing.T) {
		result, err := scanner.ScanDependencies(ctx, "/nonexistent/dir")
		if err != scanners.ErrProjectNotFound {
			t.Errorf("Expected ErrProjectNotFound, got %v", err)
		}
		if result != nil {
			t.Error("Expected nil result")
		}
	})

	t.Run("invalid go.mod", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "go-scanner-invalid")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create an invalid go.mod file
		if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("invalid"), 0644); err != nil {
			t.Fatalf("Failed to write go.mod: %v", err)
		}

		result, err := scanner.ScanDependencies(ctx, tempDir)
		if err == nil {
			t.Error("Expected error for invalid go.mod")
		}
		if result != nil {
			t.Error("Expected nil result")
		}
	})
}

func TestModuleInfo_Parsing(t *testing.T) {
	jsonData := `{
        "Path": "github.com/example/module",
        "Version": "v1.2.3",
        "Main": false,
        "Indirect": true
    }`

	var info ModuleInfo
	if err := json.Unmarshal([]byte(jsonData), &info); err != nil {
		t.Fatalf("Failed to parse ModuleInfo: %v", err)
	}

	if info.Path != "github.com/example/module" {
		t.Errorf("Expected Path to be 'github.com/example/module', got '%s'", info.Path)
	}
	if info.Version != "v1.2.3" {
		t.Errorf("Expected Version to be 'v1.2.3', got '%s'", info.Version)
	}
	if info.Main {
		t.Error("Expected Main to be false")
	}
	if !info.Indirect {
		t.Error("Expected Indirect to be true")
	}
}
