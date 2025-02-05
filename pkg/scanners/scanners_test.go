package scanners

import (
	"context"
	"testing"
)

type MockScanner struct {
	BaseScanner
	detectFunc func(ctx context.Context, dir string) bool
	scanFunc   func(ctx context.Context, dir string) (*ScanResult, error)
}

func NewMockScanner(name string) *MockScanner {
	return &MockScanner{
		BaseScanner: NewBaseScanner(name),
		detectFunc:  func(ctx context.Context, dir string) bool { return true },
		scanFunc: func(ctx context.Context, dir string) (*ScanResult, error) {
			return &ScanResult{Dependencies: []Dependency{}}, nil
		},
	}
}

func (m *MockScanner) DetectProject(ctx context.Context, dir string) bool {
	return m.detectFunc(ctx, dir)
}

func (m *MockScanner) ScanDependencies(ctx context.Context, dir string) (*ScanResult, error) {
	return m.scanFunc(ctx, dir)
}

func TestBaseScanner(t *testing.T) {
	t.Run("Name returns correct scanner name", func(t *testing.T) {
		expected := "test-scanner"
		scanner := NewBaseScanner(expected)

		if got := scanner.Name(); got != expected {
			t.Errorf("BaseScanner.Name() = %v, want %v", got, expected)
		}
	})
}

func TestDependency(t *testing.T) {
	t.Run("Dependency structure initialization", func(t *testing.T) {
		dep := Dependency{
			Name:        "test-package",
			Version:     "1.0.0",
			Type:        "npm",
			IsDirectDep: true,
			Properties: map[string]string{
				"key": "value",
			},
		}

		if dep.Name != "test-package" {
			t.Errorf("Expected Name to be 'test-package', got %v", dep.Name)
		}
		if dep.Version != "1.0.0" {
			t.Errorf("Expected Version to be '1.0.0', got %v", dep.Version)
		}
		if dep.Type != "npm" {
			t.Errorf("Expected Type to be 'npm', got %v", dep.Type)
		}
		if !dep.IsDirectDep {
			t.Error("Expected IsDirectDep to be true")
		}
		if val, exists := dep.Properties["key"]; !exists || val != "value" {
			t.Errorf("Expected Properties[\"key\"] to be 'value', got %v", val)
		}
	})
}

func TestScanResult(t *testing.T) {
	t.Run("ScanResult initialization and modification", func(t *testing.T) {
		result := &ScanResult{
			Dependencies: []Dependency{
				{
					Name:    "package1",
					Version: "1.0.0",
				},
				{
					Name:    "package2",
					Version: "2.0.0",
				},
			},
			Errors: []error{
				ErrProjectNotFound,
			},
		}

		if len(result.Dependencies) != 2 {
			t.Errorf("Expected 2 dependencies, got %d", len(result.Dependencies))
		}

		if len(result.Errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(result.Errors))
		}

		if result.Errors[0] != ErrProjectNotFound {
			t.Errorf("Expected error to be ErrProjectNotFound")
		}
	})
}

func TestMockScanner(t *testing.T) {
	ctx := context.Background()

	t.Run("MockScanner with custom detect function", func(t *testing.T) {
		scanner := NewMockScanner("test")
		scanner.detectFunc = func(ctx context.Context, dir string) bool {
			return dir == "valid-dir"
		}

		if !scanner.DetectProject(ctx, "valid-dir") {
			t.Error("Expected detection to return true for 'valid-dir'")
		}

		if scanner.DetectProject(ctx, "invalid-dir") {
			t.Error("Expected detection to return false for 'invalid-dir'")
		}
	})

	t.Run("MockScanner with custom scan function", func(t *testing.T) {
		scanner := NewMockScanner("test")
		expectedDep := Dependency{Name: "test-dep", Version: "1.0.0"}
		scanner.scanFunc = func(ctx context.Context, dir string) (*ScanResult, error) {
			return &ScanResult{
				Dependencies: []Dependency{expectedDep},
			}, nil
		}

		result, err := scanner.ScanDependencies(ctx, "test-dir")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result.Dependencies) != 1 {
			t.Fatalf("Expected 1 dependency, got %d", len(result.Dependencies))
		}

		if result.Dependencies[0].Name != expectedDep.Name {
			t.Errorf("Expected dependency name %s, got %s", expectedDep.Name, result.Dependencies[0].Name)
		}
	})

	t.Run("MockScanner error handling", func(t *testing.T) {
		scanner := NewMockScanner("test")
		scanner.scanFunc = func(ctx context.Context, dir string) (*ScanResult, error) {
			return nil, ErrProjectNotFound
		}

		result, err := scanner.ScanDependencies(ctx, "test-dir")
		if err != ErrProjectNotFound {
			t.Errorf("Expected ErrProjectNotFound, got %v", err)
		}
		if result != nil {
			t.Error("Expected nil result when error occurs")
		}
	})
}

func TestErrors(t *testing.T) {
	t.Run("Error variables are defined", func(t *testing.T) {
		errors := []error{
			ErrProjectNotFound,
			ErrInvalidProject,
			ErrScanFailed,
		}

		for _, err := range errors {
			if err == nil {
				t.Error("Expected error to be defined")
			}
			if err.Error() == "" {
				t.Error("Expected error to have a non-empty message")
			}
		}
	})
}
