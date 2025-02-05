# Basic Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Binary names
BINARY_NAME=deplister
BINARY_UNIX=$(BINARY_NAME)_unix

# Build directory
BUILD_DIR=build

.PHONY: all test build clean deps fmt vet lint run help

all: test build

## clean: Remove build artifacts and cleanup dependencies
clean:
	@echo "Cleaning build cache..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@rm -f $(BINARY_UNIX)
	@$(GOCMD) clean
	@rm -f coverage.out

## deps: Download and install dependencies
deps:
	@echo "Downloading dependencies..."
	@$(GOMOD) download
	@$(GOMOD) verify
	@$(GOMOD) tidy

## fmt: Run go fmt on all source files
fmt:
	@echo "Running go fmt..."
	@$(GOFMT) ./...

## vet: Run go vet on all source files
vet:
	@echo "Running go vet..."
	@$(GOCMD) vet ./...

## lint: Run linter
lint:
	@if command -v golangci-lint >/dev/null; then \
		echo "Running golangci-lint..."; \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin; \
		golangci-lint run; \
	fi

## test: Run tests with coverage
test: deps fmt vet
	@echo "Running tests..."
	@$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@$(GOCMD) tool cover -func=coverage.out

## test-html: Generate HTML coverage report
test-html: test
	@$(GOCMD) tool cover -html=coverage.out

## build: Build the binary
build: deps
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -v cmd/deplister/main.go

## run: Build and run the binary
run: build
	@echo "Running..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

## init: Initialize a new project (clean start)
init: clean
	@echo "Initializing new project..."
	@rm -f go.mod go.sum
	@$(GOMOD) init deplister
	@$(GOGET) github.com/stretchr/testify@v1.8.4
	@$(GOMOD) tidy

## update: Update dependencies to latest versions
update: 
	@echo "Updating dependencies..."
	@$(GOGET) -u ./...
	@$(GOMOD) tidy

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^##' Makefile | sed -e 's/## //g' | column -t -s ':'

# Default target
.DEFAULT_GOAL := help