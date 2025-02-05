package models

import (
	"time"
)

type Analysis struct {
	ProjectPath       string                    `json:"project_path"`
	AnalysisTimestamp time.Time                 `json:"analysis_timestamp"`
	PackageManagers   map[string]*ScannerResult `json:"package_managers"`
}

type ScannerResult struct {
	Dependencies    map[string]string `json:"dependencies,omitempty"`
	DevDependencies map[string]string `json:"dev_dependencies,omitempty"`
	Errors          []string          `json:"errors,omitempty"`
}
