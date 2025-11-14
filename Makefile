.PHONY: help build build-patch build-minor build-major test test-coverage test-verbose test-race fmt lint clean run validate learning-stats learning-export learning-clear

# Variables
BINARY_NAME = conductor
VERSION_FILE = VERSION
CURRENT_VERSION := $(shell cat $(VERSION_FILE))
GO = go
GOFLAGS = -v

help:
	@echo "Conductor Build System"
	@echo ""
	@echo "Build targets:"
	@echo "  make build              - Build binary with current version"
	@echo "  make build-patch        - Increment patch version (1.0.0 → 1.0.1) and build"
	@echo "  make build-minor        - Increment minor version (1.0.0 → 1.1.0) and build"
	@echo "  make build-major        - Increment major version (1.0.0 → 2.0.0) and build"
	@echo ""
	@echo "Build targets:"
	@echo "  make test               - Run all tests"
	@echo "  make test-verbose       - Run tests with verbose output"
	@echo "  make test-race          - Run tests with race detection"
	@echo "  make test-coverage      - Run tests with coverage report"
	@echo ""
	@echo "Quality targets:"
	@echo "  make fmt                - Format all Go code"
	@echo "  make lint               - Run linter (requires golangci-lint)"
	@echo ""
	@echo "Execution targets:"
	@echo "  make run                - Run conductor from source"
	@echo "  make validate           - Show validate command help"
	@echo ""
	@echo "Learning system targets:"
	@echo "  make learning-stats     - View learning statistics"
	@echo "  make learning-export    - Export learning data to JSON"
	@echo "  make learning-clear     - Clear learning history"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean              - Remove binary and coverage files"
	@echo ""
	@echo "Current version: $(CURRENT_VERSION)"

# Build with current version
build:
	@echo "Building conductor v$(CURRENT_VERSION)..."
	@mkdir -p ~/bin
	@REPO_ROOT=$$(pwd) && \
	$(GO) build $(GOFLAGS) -ldflags "-X github.com/harrison/conductor/internal/cmd.Version=$(CURRENT_VERSION) -X github.com/harrison/conductor/internal/cmd.ConductorRepoRoot=$${REPO_ROOT}" -o ~/bin/$(BINARY_NAME) ./cmd/conductor
	@if [ -f ./$(BINARY_NAME) ]; then rm ./$(BINARY_NAME); fi
	@if ! echo $$PATH | grep -q $$HOME/bin; then \
		echo "export PATH=\"\$$HOME/bin:\$$PATH\"" >> ~/.zshrc; \
		echo "export PATH=\"\$$HOME/bin:\$$PATH\"" >> ~/.bashrc; \
		echo "⚠️  Added ~/bin to PATH in ~/.zshrc and ~/.bashrc"; \
		if [ -n "$$ZSH_VERSION" ]; then \
			source ~/.zshrc; \
			echo "✅ PATH updated and sourced in current shell"; \
		elif [ -n "$$BASH_VERSION" ]; then \
			source ~/.bashrc; \
			echo "✅ PATH updated and sourced in current shell"; \
		fi; \
	fi
	@echo "✅ Binary built: ~/bin/conductor"
	@echo "✅ Use: conductor --version"

# Helper function to increment version
define increment_version
	@echo "Current version: $(CURRENT_VERSION)"
	@python3 -c "import sys; parts = '$(CURRENT_VERSION)'.split('.'); \
		$(1); \
		print('.'.join(map(str, parts)))" > $(VERSION_FILE)_tmp && \
	mv $(VERSION_FILE)_tmp $(VERSION_FILE)
	@echo "New version: $$(cat $(VERSION_FILE))"
endef

# Increment patch version and build
build-patch:
	$(call increment_version,parts[2] = str(int(parts[2]) + 1))
	@$(MAKE) build

# Increment minor version and build
build-minor:
	$(call increment_version,parts[1] = str(int(parts[1]) + 1); parts[2] = '0')
	@$(MAKE) build

# Increment major version and build
build-major:
	$(call increment_version,parts[0] = str(int(parts[0]) + 1); parts[1] = '0'; parts[2] = '0')
	@$(MAKE) build

# Run all tests
test:
	@echo "Running all tests..."
	$(GO) test ./...

# Run tests with verbose output
test-verbose:
	@echo "Running all tests (verbose)..."
	$(GO) test -v ./...

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	$(GO) test -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -coverprofile=coverage.out ./...
	@echo "Coverage report generated: coverage.out"
	@echo "View in browser: go tool cover -html=coverage.out"
	@$(GO) tool cover -func=coverage.out | grep total

# Format all Go code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Code formatted"

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Install with: brew install golangci-lint"; exit 1; }
	golangci-lint run

# Run conductor from source
run:
	$(GO) run ./cmd/conductor

# Show validate command help
validate:
	$(GO) run ./cmd/conductor validate --help

# Learning system commands
learning-stats:
	$(GO) run ./cmd/conductor learning stats

learning-export:
	$(GO) run ./cmd/conductor learning export

learning-clear:
	$(GO) run ./cmd/conductor learning clear

# Clean build artifacts
clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out
	@echo "Cleaned"
