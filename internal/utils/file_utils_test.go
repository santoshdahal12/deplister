package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileExists(t *testing.T) {
	// Create temp file
	tmpfile, err := os.CreateTemp("", "test")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	// Test existing file
	assert.True(t, FileExists(tmpfile.Name()))

	// Test non-existing file
	assert.False(t, FileExists("nonexistent.txt"))
}

func TestFindFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	assert.NoError(t, err)

	matches := FindFiles(tmpDir, []string{"test.txt"})
	assert.Len(t, matches, 1)
	assert.Equal(t, testFile, matches[0])
}
