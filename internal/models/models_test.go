package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAnalysisMarshaling(t *testing.T) {
	analysis := &Analysis{
		ProjectPath:       "/test/path",
		AnalysisTimestamp: time.Now(),
		PackageManagers: map[string]*ScannerResult{
			"test": {
				Dependencies: map[string]string{
					"dep1": "1.0.0",
				},
			},
		},
	}

	data, err := json.Marshal(analysis)
	assert.NoError(t, err)

	var decoded Analysis
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, analysis.ProjectPath, decoded.ProjectPath)
}
