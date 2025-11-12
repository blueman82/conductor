# Makefile for Conductor - Autonomous Multi-Agent Orchestration CLI

# Variables
BINARY_NAME=conductor
CMD_PATH=./cmd/conductor
GO_VERSION=1.25.4
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Build information
VERSION=$(shell cat VERSION 2>/dev/null || echo "0.0.0")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X github.com/harrison/conductor/internal/cmd.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

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
.PHONY: all build build-patch build-minor build-major test test-verbose coverage clean install help fmt vet lint tidy check deps

# Default target
all: clean fmt vet build test

## help: Show this help message
help:
	@echo "Conductor - Makefile targets:"
	@echo ""
	@echo "  make build          - Build the conductor binary to ./$(BINARY_NAME)"
	@echo "  make build-patch    - Bump patch version and build (1.1.0 → 1.1.1)"
	@echo "  make build-minor    - Bump minor version and build (1.1.0 → 1.2.0)"
	@echo "  make build-major    - Bump major version and build (1.1.0 → 2.0.0)"
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

## build-patch: Bump patch version (x.y.Z) and build
build-patch:
	@echo "Bumping patch version..."
	@current=$$(cat VERSION); \
	major=$$(echo $$current | cut -d. -f1); \
	minor=$$(echo $$current | cut -d. -f2); \
	patch=$$(echo $$current | cut -d. -f3); \
	patch=$$(($$patch + 1)); \
	echo "$$major.$$minor.$$patch" > VERSION; \
	echo "Version bumped: $$current → $$major.$$minor.$$patch"
	@$(MAKE) build

## build-minor: Bump minor version (x.Y.0) and build
build-minor:
	@echo "Bumping minor version..."
	@current=$$(cat VERSION); \
	major=$$(echo $$current | cut -d. -f1); \
	minor=$$(echo $$current | cut -d. -f2); \
	minor=$$(($$minor + 1)); \
	echo "$$major.$$minor.0" > VERSION; \
	echo "Version bumped: $$current → $$major.$$minor.0"
	@$(MAKE) build

## build-major: Bump major version (X.0.0) and build
build-major:
	@echo "Bumping major version..."
	@current=$$(cat VERSION); \
	major=$$(echo $$current | cut -d. -f1); \
	major=$$(($$major + 1)); \
	echo "$$major.0.0" > VERSION; \
	echo "Version bumped: $$current → $$major.0.0"
	@$(MAKE) build

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

## install: Install to $GOPATH/bin with optional PATH configuration
install:
	@echo "Installing $(BINARY_NAME) to \$$GOPATH/bin..."
	$(GOINSTALL) $(LDFLAGS) $(CMD_PATH)
	@echo "Install complete: $$(go env GOPATH)/bin/$(BINARY_NAME)"
	@echo ""
	@bash -c '\
		GOPATH=$$(go env GOPATH); \
		GOBIN=$$GOPATH/bin; \
		if echo $$PATH | grep -q "$$GOBIN"; then \
			echo "✓ $$GOBIN is already in your PATH"; \
		else \
			echo "✗ $$GOBIN is NOT in your PATH"; \
			echo ""; \
			read -p "Would you like to add it? (y/n) " -n 1 -r; \
			echo ""; \
			if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
				SHELL_RC=""; \
				if [ -f ~/.zshrc ]; then \
					SHELL_RC=~/.zshrc; \
				elif [ -f ~/.bash_profile ]; then \
					SHELL_RC=~/.bash_profile; \
				elif [ -f ~/.bashrc ]; then \
					SHELL_RC=~/.bashrc; \
				fi; \
				if [ -z "$$SHELL_RC" ]; then \
					echo "Could not find shell config file. Please add manually:"; \
					echo "  export PATH=\$$PATH:$$GOBIN"; \
				else \
					if ! grep -q "$$GOBIN" "$$SHELL_RC"; then \
						echo "export PATH=\$$PATH:$$GOBIN" >> "$$SHELL_RC"; \
						echo "✓ Added to $$SHELL_RC"; \
						echo ""; \
						echo "Reload your shell:"; \
						echo "  source $$SHELL_RC"; \
					else \
						echo "✓ Already configured in $$SHELL_RC"; \
					fi; \
				fi; \
			else \
				echo ""; \
				echo "To use conductor globally, add this to your shell config:"; \
				echo "  export PATH=\$$PATH:$$GOBIN"; \
				echo ""; \
				echo "Or use the full path: $$GOBIN/$(BINARY_NAME)"; \
			fi; \
		fi; \
	'

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
