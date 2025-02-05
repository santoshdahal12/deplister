package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"deplister/pkg/scanners"
	"deplister/pkg/scanners/golang"
	"deplister/pkg/scanners/npm"
)

type OutputFormat struct {
	ProjectType  string             `json:"projectType"`
	Dependencies []DependencyOutput `json:"dependencies"`
}

type DependencyOutput struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Type        string            `json:"type"`
	IsDirectDep bool              `json:"isDirectDependency"`
	Parent      string            `json:"parent,omitempty"`
	Properties  map[string]string `json:"properties,omitempty"`
}

// Scanner registry
var availableScanners = []scanners.Scanner{
	npm.NewScanner(),
	golang.NewScanner(),
}

func main() {
	var (
		projectPath  string
		textOutput   bool
		outputFile   string
		prettyOutput bool
	)

	flag.StringVar(&projectPath, "path", ".", "Path to the project directory")
	flag.StringVar(&outputFile, "out", "", "Output file path (default: stdout)")
	flag.BoolVar(&textOutput, "text", false, "Output in human-readable text format")
	flag.BoolVar(&prettyOutput, "pretty", false, "Pretty print JSON output (ignored with -text)")
	flag.Parse()

	// Convert to absolute path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		os.Exit(1)
	}

	// Detect project type and scan dependencies
	ctx := context.Background()
	var projectScanner scanners.Scanner
	var projectType string

	for _, scanner := range availableScanners {
		if scanner.DetectProject(ctx, absPath) {
			projectScanner = scanner
			projectType = scanner.GetType()
			break
		}
	}

	if projectScanner == nil {
		fmt.Fprintf(os.Stderr, "No supported project found at %s\n", absPath)
		fmt.Fprintf(os.Stderr, "Supported project types: npm, go\n")
		os.Exit(1)
	}

	result, err := projectScanner.ScanDependencies(ctx, absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning dependencies: %v\n", err)
		os.Exit(1)
	}

	if textOutput {
		outputText(result, projectType, outputFile)
	} else {
		outputJSON(result, projectType, outputFile, prettyOutput)
	}
}

func outputJSON(result *scanners.ScanResult, projectType, outputFile string, pretty bool) {
	output := OutputFormat{
		ProjectType:  projectType,
		Dependencies: make([]DependencyOutput, len(result.Dependencies)),
	}

	for i, dep := range result.Dependencies {
		output.Dependencies[i] = DependencyOutput{
			Name:        dep.Name,
			Version:     dep.Version,
			Type:        dep.Type,
			IsDirectDep: dep.IsDirectDep,
			Parent:      dep.Parent,
			Properties:  dep.Properties,
		}
	}

	var writer io.Writer = os.Stdout
	if outputFile != "" {
		file, err := os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		writer = file
	}

	encoder := json.NewEncoder(writer)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func outputText(result *scanners.ScanResult, projectType, outputFile string) {
	var writer io.Writer = os.Stdout
	if outputFile != "" {
		file, err := os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		writer = file
	}

	fmt.Fprintf(writer, "Project Type: %s\n", projectType)
	fmt.Fprintln(writer, "Dependencies:")
	fmt.Fprintln(writer, "-------------")

	for _, dep := range result.Dependencies {
		depType := "Production"
		if t, ok := dep.Properties["dependencyType"]; ok {
			depType = t
		}

		directness := "Indirect"
		if dep.IsDirectDep {
			directness = "Direct"
		}

		fmt.Fprintf(writer, "%s@%s (%s, %s)\n", dep.Name, dep.Version, depType, directness)

		if resolved, ok := dep.Properties["resolved"]; ok {
			fmt.Fprintf(writer, "  Source: %s\n", resolved)
		}

		if !dep.IsDirectDep && dep.Parent != "" {
			fmt.Fprintf(writer, "  Required by: %s\n", dep.Parent)
		}

		if replacedBy, ok := dep.Properties["replaced_by"]; ok {
			fmt.Fprintf(writer, "  Replaced by: %s@%s\n", replacedBy, dep.Properties["replaced_version"])
		}

		fmt.Fprintln(writer)
	}
}
