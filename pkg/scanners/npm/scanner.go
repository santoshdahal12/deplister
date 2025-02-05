// pkg/scanners/npm/scanner.go

package npm

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"deplister/pkg/scanners"
)

type PackageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

type NPMScanner struct {
	scanners.BaseScanner
}

func NewScanner() *NPMScanner {
	return &NPMScanner{
		BaseScanner: scanners.NewBaseScanner("npm"),
	}
}

func (s *NPMScanner) DetectProject(ctx context.Context, dir string) bool {
	packageJSONPath := filepath.Join(dir, "package.json")
	_, err := os.Stat(packageJSONPath)
	return err == nil
}

func (s *NPMScanner) ScanDependencies(ctx context.Context, dir string) (*scanners.ScanResult, error) {
	if !s.DetectProject(ctx, dir) {
		return nil, scanners.ErrProjectNotFound
	}

	packageJSONPath := filepath.Join(dir, "package.json")
	content, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return nil, err
	}

	var pkg PackageJSON
	if err := json.Unmarshal(content, &pkg); err != nil {
		return nil, scanners.ErrInvalidProject
	}

	result := &scanners.ScanResult{
		Dependencies: make([]scanners.Dependency, 0),
	}

	// Process regular dependencies
	for name, version := range pkg.Dependencies {
		result.Dependencies = append(result.Dependencies, scanners.Dependency{
			Name:        name,
			Version:     version,
			Type:        "npm",
			IsDirectDep: true,
			Properties: map[string]string{
				"dependencyType": "production",
			},
		})
	}

	// Process dev dependencies
	for name, version := range pkg.DevDependencies {
		result.Dependencies = append(result.Dependencies, scanners.Dependency{
			Name:        name,
			Version:     version,
			Type:        "npm",
			IsDirectDep: true,
			Properties: map[string]string{
				"dependencyType": "development",
			},
		})
	}

	return result, nil
}
