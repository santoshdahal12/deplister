# Deplister

A powerful dependency analysis tool designed for modern software projects. Deplister provides comprehensive insights into your project dependencies through advanced graph analysis and relationship tracking, supporting both Go and NPM ecosystems.

[![Tag](https://img.shields.io/github/v/tag/santoshdahal12/deplister?include_prereleases)](https://github.com/santoshdahal12/deplister/releases)

## Overview

Deplister is built to provide deep visibility into your project's dependency structure, making it an invaluable tool for:
- DevOps teams implementing dependency scanning in CI/CD pipelines
- Development teams seeking to understand and optimize their dependency graph
- Security teams tracking dependency relationships and potential vulnerabilities

## Key Features

### Multi-Package Manager Support
- **Go Modules**
  - Complete dependency graph analysis
  - Module replacement tracking
  - Version constraint analysis
  - Direct and indirect dependency resolution

- **NPM Packages**
  - Deep dependency resolution
  - Package-lock.json analysis
  - Development vs production dependency classification
  - Peer dependency tracking

### Advanced Analysis Capabilities
- Comprehensive dependency graph generation
- Path tracking between dependencies
- Parent-child relationship mapping
- Circular dependency detection
- Version conflict identification

### Rich Metadata Collection
- Detailed version tracking and constraints
- Dependency type classification
- Package manager specific properties
- Module replacement tracking (Go-specific)
- Package scope analysis (NPM-specific)

### Flexible Output Formats
- Standard output (default)
- JSON format (compact or pretty-printed)
- Human-readable text format
- Easy integration with other tools and pipelines

## Installation

### Using Go Install
```bash
go install github.com/santoshdahal12/deplister@latest
```

### Building from Source
```bash
git clone https://github.com/santoshdahal12/deplister.git
cd deplister
go build
```

## Usage

### Basic Command
```bash
deplister [options] 
```

### Command Options
```
-path string
      Path to the project directory (default ".")
-out string
      Output file path (default: stdout)
-pretty
      Pretty print JSON output (ignored with -text)
-text
      Output in human-readable text format
-help
      Help text
```

### Example Commands
```bash
# Analyze current directory with default JSON output
deplister

# Analyze specific directory with pretty-printed JSON
deplister -path /path/to/project -pretty

# Generate human-readable text output
deplister -text

# Save analysis to file
deplister -out dependencies.json -pretty
```

## Integration Examples

### GitHub Actions
```yaml
steps:
  - uses: actions/checkout@v3
  - uses: actions/setup-go@v4
    with:
      go-version: '1.21'
  - name: Install Deplister
    run: go install github.com/santoshdahal12/deplister@latest
  - name: Analyze Dependencies
    run: deplister -pretty > dependency-report.json
```

### Jenkins Pipeline
```groovy
pipeline {
    stages {
        stage('Dependency Analysis') {
            steps {
                sh 'go install github.com/santoshdahal12/deplister@latest'
                sh 'deplister -text > dependency-report.txt'
            }
        }
    }
}
```

## Contributing

We welcome contributions! Here's how you can help:

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

### Contribution Guidelines
- Include tests for new features
- Update documentation as needed
- Follow existing code style
- Ensure all tests pass
- Add meaningful commit messages

## Support

If you need help or want to report an issue:

1. Search [existing issues](https://github.com/santoshdahal12/deplister/issues)
2. Open a new issue with:
   - Your environment details (OS, Go version, etc.)
   - Steps to reproduce
   - Expected vs actual behavior
   - Relevant logs or output

## Roadmap

### Upcoming Features
- [ ] Support for additional package managers (Yarn, Maven , Gradle etc)
- [ ] Interactive visualization interface


## License

This project is licensed under the MIT License.
