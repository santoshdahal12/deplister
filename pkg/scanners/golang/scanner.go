// pkg/scanners/golang/scanner.go

package golang

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"deplister/pkg/scanners"
)

type GoScanner struct {
	scanners.BaseScanner
}

type ModuleInfo struct {
	Path     string `json:"Path"`
	Version  string `json:"Version"`
	Main     bool   `json:"Main"`
	Indirect bool   `json:"Indirect"`
}

func NewScanner() *GoScanner {
	return &GoScanner{
		BaseScanner: scanners.NewBaseScanner("go"),
	}
}

func (s *GoScanner) DetectProject(ctx context.Context, dir string) bool {
	goModPath := filepath.Join(dir, "go.mod")
	_, err := os.Stat(goModPath)
	return err == nil
}

func (s *GoScanner) ScanDependencies(ctx context.Context, dir string) (*scanners.ScanResult, error) {
	if !s.DetectProject(ctx, dir) {
		return nil, scanners.ErrProjectNotFound
	}

	// Run `go list -m -json all` to get detailed module information
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-json", "all")
	cmd.Dir = dir

	output, err := cmd.Output()
	if err != nil {
		return nil, scanners.ErrScanFailed
	}

	moduleInfos := make([]ModuleInfo, 0)
	for _, line := range strings.Split(string(output), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var info ModuleInfo
		if err := json.Unmarshal([]byte(line), &info); err != nil {
			return nil, scanners.ErrInvalidProject
		}
		moduleInfos = append(moduleInfos, info)
	}

	result := &scanners.ScanResult{
		Dependencies: make([]scanners.Dependency, 0),
	}

	// Skip the first module (it's the main module)
	for _, info := range moduleInfos[1:] {
		dep := scanners.Dependency{
			Name:        info.Path,
			Version:     info.Version,
			Type:        "go",
			IsDirectDep: !info.Indirect,
			Properties: map[string]string{
				"indirect": stringify(info.Indirect),
			},
		}
		result.Dependencies = append(result.Dependencies, dep)
	}

	return result, nil
}

func stringify(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
