# Development Setup

Complete guide for setting up a Conductor development environment.

## Prerequisites

- Go 1.21 or higher
- Git
- Claude Code CLI installed and authenticated
- Python 3 (for version management scripts)

## Local Development Setup

### 1. Clone the Repository

```bash
git clone https://github.com/harrison/conductor.git
cd conductor
```

### 2. Install Dependencies

```bash
go mod download
go mod verify
```

### 3. Build the Binary

Using Make:
```bash
make build
```

Or manually:
```bash
go build -o conductor ./cmd/conductor
```

Verify the build:
```bash
./conductor --version
```

## Development Workflow

### Building

Build with the current version:
```bash
make build
```

Auto-increment and build:
```bash
make build-patch   # 1.0.0 → 1.0.1
make build-minor   # 1.0.0 → 1.1.0
make build-major   # 1.0.0 → 2.0.0
```

### Running Tests

Run all tests:
```bash
make test
```

Run with verbose output:
```bash
make test-verbose
```

Run specific package:
```bash
go test ./internal/executor/ -v
```

Run with race detection:
```bash
make test-race
```

Generate coverage report:
```bash
make test-coverage
# View in browser
go tool cover -html=coverage.out
```

### Code Quality

Format code:
```bash
make fmt
```

Lint code (requires golangci-lint):
```bash
# Install linter
brew install golangci-lint

# Run linter
make lint
```

Vet code:
```bash
go vet ./...
```

### Running Conductor

From source:
```bash
make run
```

With a plan file:
```bash
./conductor run path/to/plan.md --verbose
```

Validate a plan:
```bash
./conductor validate path/to/plan.md
```

## Project Structure

```
conductor/
├── cmd/
│   └── conductor/           # CLI entrypoint
├── internal/
│   ├── parser/              # Plan file parsing (Markdown & YAML)
│   ├── executor/            # Task execution & orchestration
│   ├── agent/               # Claude agent discovery & invocation
│   ├── models/              # Core data structures
│   ├── learning/            # Adaptive learning system
│   ├── config/              # Configuration management
│   └── cmd/                 # CLI commands (validate, run, learning)
├── plugin/                  # Claude Code plugin for plan generation
├── docs/                    # Documentation
├── Makefile                 # Build automation
├── VERSION                  # Semantic version file
└── go.mod                   # Go module definition
```

## Understanding the Codebase

### Key Components

**Parser** (`internal/parser/`)
- Markdown parser: Extracts tasks from `## Task N:` headings
- YAML parser: Parses structured plan files
- Both support auto-detection based on file extension

**Executor** (`internal/executor/`)
- `graph.go`: Dependency graph and wave calculation (Kahn's algorithm)
- `task.go`: Individual task execution with retry logic
- `wave.go`: Wave executor with bounded concurrency
- `qc.go`: Quality control review integration

**Agent** (`internal/agent/`)
- `discovery.go`: Scans `~/.claude/agents/` for available agents
- `invoker.go`: Executes `claude` CLI commands with proper flags

**Learning** (`internal/learning/`)
- `store.go`: Persistent execution history storage
- `analyzer.go`: Pattern recognition from historical data
- `hooks.go`: Pre/post-task hooks for learning integration

### Testing

The project uses test-driven development:
1. Write failing test first
2. Implement minimal code to pass
3. Refactor for clarity
4. Commit

Key test files:
```
internal/parser/parser_test.go
internal/executor/graph_test.go
internal/executor/task_test.go
internal/learning/store_test.go
```

## Configuration

Default configuration location: `.conductor/config.yaml`

Example configuration:
```yaml
max_concurrency: 3
timeout: 30m
dry_run: false
skip_completed: false
retry_failed: false
log_dir: .conductor/logs
log_level: info

learning:
  enabled: true
  dir: .conductor/learning
  max_history: 100
```

## Learning System Development

The learning system stores execution history in `.conductor/learning/`:

### View Learning Data
```bash
./conductor learning stats      # Statistics
./conductor learning export     # Export to JSON
./conductor learning clear      # Clear history (with confirmation)
```

### Understanding Learning Flow
1. **Pre-Task Hook**: Loads historical failures and injects context
2. **Post-Task Hook**: Stores execution outcome
3. **Analyzer**: Identifies patterns from stored data

## Troubleshooting Development

### Import issues
Module path is `github.com/harrison/conductor`:
```go
import "github.com/harrison/conductor/internal/executor"
```

### Test failures
Run with verbose output:
```bash
go test -v ./internal/executor/
```

Check specific test:
```bash
go test ./internal/executor/ -run TestWaveCalculation
```

### Build issues
Clean and rebuild:
```bash
make clean
make build
```

Verify Go version:
```bash
go version  # Should be 1.21+
```

## Submitting Changes

1. Create a feature branch
2. Make changes with tests
3. Run all tests: `make test`
4. Run linter: `make lint`
5. Format code: `make fmt`
6. Commit with clear message
7. Push and create pull request

See CONTRIBUTING.md for detailed guidelines.

## Additional Resources

- [README.md](README.md) - Project overview and quick start
- [CLAUDE.md](CLAUDE.md) - Architecture and design details
- [docs/](docs/) - Complete documentation
- [Plugin Documentation](plugin/docs) - Plan generation tools
