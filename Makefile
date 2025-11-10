# Makefile for Conductor - Autonomous Multi-Agent Orchestration CLI

# Variables
BINARY_NAME=conductor
CMD_PATH=./cmd/conductor
GO_VERSION=1.25.4
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Build information
VERSION?=1.0.0
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOINSTALL=$(GOCMD) install
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet
GOMOD=$(GOCMD) mod

# Detect OS for proper clean commands
ifeq ($(OS),Windows_NT)
    RM=del /Q
    RMDIR=rmdir /S /Q
    BINARY_EXT=.exe
else
    RM=rm -f
    RMDIR=rm -rf
    BINARY_EXT=
endif

# Phony targets (not actual files)
.PHONY: all build test test-verbose coverage clean install help fmt vet lint tidy check deps

# Default target
all: clean fmt vet build test

## help: Show this help message
help:
	@echo "Conductor - Makefile targets:"
	@echo ""
	@echo "  make build          - Build the conductor binary to ./$(BINARY_NAME)"
	@echo "  make test           - Run all tests with coverage"
	@echo "  make test-verbose   - Run tests with verbose output"
	@echo "  make coverage       - Generate coverage report and open HTML"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make install        - Install to \$$GOPATH/bin"
	@echo "  make fmt            - Format code with gofmt"
	@echo "  make vet            - Run go vet"
	@echo "  make lint           - Run golangci-lint (if installed)"
	@echo "  make tidy           - Tidy go modules"
	@echo "  make check          - Run fmt, vet, and test"
	@echo "  make deps           - Download dependencies"
	@echo "  make all            - Run clean, fmt, vet, build, and test"
	@echo "  make help           - Show this help message"
	@echo ""

## build: Build the conductor binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)$(BINARY_EXT) $(CMD_PATH)
	@echo "Build complete: ./$(BINARY_NAME)$(BINARY_EXT)"

## test: Run all tests with coverage
test:
	@echo "Running tests with coverage..."
	$(GOTEST) -v ./... -cover -coverprofile=$(COVERAGE_FILE)
	@echo "Test coverage:"
	@$(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep total

## test-verbose: Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	$(GOTEST) -v ./... -cover

## coverage: Generate coverage report and open HTML
coverage:
	@echo "Generating coverage report..."
	$(GOTEST) ./... -coverprofile=$(COVERAGE_FILE)
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"
	@echo "Opening coverage report in browser..."
ifeq ($(OS),Windows_NT)
	start $(COVERAGE_HTML)
else ifeq ($(shell uname -s),Darwin)
	open $(COVERAGE_HTML)
else
	xdg-open $(COVERAGE_HTML) 2>/dev/null || echo "Please open $(COVERAGE_HTML) in your browser"
endif

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	$(RM) $(BINARY_NAME)$(BINARY_EXT)
	$(RM) $(COVERAGE_FILE)
	$(RM) $(COVERAGE_HTML)
	$(RM) main$(BINARY_EXT)
	$(RMDIR) dist 2>/dev/null || true
	@echo "Clean complete"

## install: Install to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME) to \$$GOPATH/bin..."
	$(GOINSTALL) $(LDFLAGS) $(CMD_PATH)
	@echo "Install complete"
	@echo ""
	@echo "To use conductor globally, add \$$GOPATH/bin to your PATH:"
	@echo ""
	@echo "  Add to ~/.zshrc or ~/.bash_profile:"
	@echo "  export PATH=\$$PATH:\$$GOPATH/bin"
	@echo ""
	@echo "Then reload: source ~/.zshrc"
	@echo ""
	@echo "Or run directly: $$(go env GOPATH)/bin/$(BINARY_NAME)"

## fmt: Format code with gofmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...
	@echo "Format complete"

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...
	@echo "Vet complete"

## lint: Run golangci-lint (if installed)
lint:
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run
	@echo "Lint complete"

## tidy: Tidy go modules
tidy:
	@echo "Tidying go modules..."
	$(GOMOD) tidy
	@echo "Tidy complete"

## check: Run fmt, vet, and test
check: fmt vet test
	@echo "All checks passed!"

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	@echo "Dependencies downloaded"
