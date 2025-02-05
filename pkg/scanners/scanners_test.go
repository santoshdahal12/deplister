package scanners

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockScanner struct {
	BaseScanner
	detectResult bool
	scanResult   *ScanResult
	scanError    error
}

func NewMockScanner(scannerType string) *MockScanner {
	return &MockScanner{
		BaseScanner:  NewBaseScanner(scannerType),
		detectResult: true,
		scanResult: &ScanResult{
			Dependencies: []Dependency{},
		},
	}
}

func (m *MockScanner) DetectProject(ctx context.Context, dir string) bool {
	return m.detectResult
}

func (m *MockScanner) ScanDependencies(ctx context.Context, dir string) (*ScanResult, error) {
	if m.scanError != nil {
		return nil, m.scanError
	}
	return m.scanResult, nil
}

func TestBaseScanner_Type(t *testing.T) {
	tests := []struct {
		name     string
		scanner  string
		expected string
	}{
		{"npm", "npm", "npm"},
		{"empty", "", ""},
		{"complex", "python-pip", "python-pip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewBaseScanner(tt.scanner)
			assert.Equal(t, tt.expected, scanner.GetType())
		})
	}
}

func TestDependency_Validation(t *testing.T) {
	tests := []struct {
		name string
		dep  Dependency
		want bool
	}{
		{
			name: "complete",
			dep: Dependency{
				Name:        "express",
				Version:     "4.17.1",
				Type:        "npm",
				IsDirectDep: true,
				Properties:  map[string]string{"manager": "npm"},
			},
			want: true,
		},
		{
			name: "with_parent",
			dep: Dependency{
				Name:        "accepts",
				Version:     "1.3.7",
				Type:        "npm",
				IsDirectDep: false,
				Parent:      "express",
			},
			want: true,
		},
		{
			name: "minimal",
			dep: Dependency{
				Name:    "lodash",
				Version: "4.17.21",
				Type:    "npm",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validateDependency(t, tt.dep)
		})
	}
}

func TestScanResult_Validation(t *testing.T) {
	tests := []struct {
		name   string
		result ScanResult
	}{
		{
			name:   "empty",
			result: ScanResult{Dependencies: []Dependency{}},
		},
		{
			name: "with_deps",
			result: ScanResult{
				Dependencies: []Dependency{{
					Name:        "express",
					Version:     "4.17.1",
					Type:        "npm",
					IsDirectDep: true,
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validateScanResult(t, tt.result)
		})
	}
}

func TestErrorDefinitions(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"project_not_found", ErrProjectNotFound, "project not found"},
		{"invalid_project", ErrInvalidProject, "invalid project"},
		{"scan_failed", ErrScanFailed, "scan failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func validateDependency(t *testing.T, dep Dependency) {
	t.Helper()

	assert.NotEmpty(t, dep.Name, "name required")
	assert.NotEmpty(t, dep.Version, "version required")
	assert.NotEmpty(t, dep.Type, "type required")

	if dep.Parent != "" {
		assert.False(t, dep.IsDirectDep, "dependency with parent cannot be direct")
	}
}

func validateScanResult(t *testing.T, result ScanResult) {
	t.Helper()

	assert.NotNil(t, result.Dependencies, "dependencies slice required")

	for _, dep := range result.Dependencies {
		validateDependency(t, dep)
	}
}
