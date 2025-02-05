package utils

import (
	"os"
	"path/filepath"
)

func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func FindFiles(root string, patterns []string) []string {
	var matches []string
	for _, pattern := range patterns {
		matches = append(matches, findPattern(root, pattern)...)
	}
	return matches
}

func findPattern(root, pattern string) []string {
	var matches []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && info.Name() == pattern {
			matches = append(matches, path)
		}
		return nil
	})
	return matches
}
