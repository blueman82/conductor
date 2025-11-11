# Conductor

[![CI](https://github.com/blueman82/conductor/actions/workflows/ci.yml/badge.svg)](https://github.com/blueman82/conductor/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/blueman82/conductor/branch/main/graph/badge.svg)](https://codecov.io/gh/blueman82/conductor)
[![Go Report Card](https://goreportcard.com/badge/github.com/blueman82/conductor)](https://goreportcard.com/report/github.com/blueman82/conductor)
[![Go Version](https://img.shields.io/github/go-mod/go-version/blueman82/conductor)](https://github.com/blueman82/conductor)

**Autonomous Multi-Agent Orchestration for Claude Code**

Conductor is a production-ready Go CLI tool that executes implementation plans by orchestrating multiple Claude Code agents in coordinated parallel waves. It parses plan files, calculates task dependencies using graph algorithms, and manages parallel execution with quality control reviews.

## What It Does

Conductor automates complex multi-step implementations by:

1. **Parsing** implementation plans in Markdown or YAML format
2. **Calculating** task dependencies and execution waves using topological sorting
3. **Orchestrating** parallel execution of tasks within each wave
4. **Spawning** Claude Code CLI agents to execute individual tasks
5. **Reviewing** task outputs with quality control agents
6. **Managing** retries on failures and marking completed tasks
7. **Logging** detailed execution progress with file-based output

## Key Features

- **Wave-Based Execution**: Tasks execute in parallel within waves, sequentially between waves
- **Dependency Management**: Automatic dependency graph calculation with cycle detection
- **Quality Control**: Automated review of task outputs (GREEN/RED/YELLOW verdicts)
- **Retry Logic**: Automatic retry on failures (up to 2 attempts per task)
- **File Locking**: Safe concurrent updates to plan files
- **Agent Discovery**: Automatic detection of available Claude agents
- **Dual Format Support**: Both Markdown and YAML plan formats
- **Dry Run Mode**: Test execution without actually running tasks
- **Progress Logging**: Real-time console output and file-based logs

## Quick Start

### Prerequisites

- **Go 1.21+** installed
- **Claude Code CLI** in your PATH (`claude` command available)
- Claude Code authenticated and configured

### Installation

```bash
# Clone the repository
git clone https://github.com/blueman82/conductor.git
cd conductor

# Build the binary
go build ./cmd/conductor

# Verify installation
./conductor --version

# Optional: Install the plugin for plan generation
# In a Claude Code session, run: /plugin
# Then select "Browse Plugins" to install conductor-tools
# Or for development/testing: claude --plugin-dir ./plugin
```

### Your First Execution

1. **Create a plan file** (see [Plan Format Guide](docs/plan-format.md))

```markdown
# My First Plan

## Task 1: Setup
**File(s)**: setup.md
**Estimated time**: 2 minutes

Create initial project structure.

## Task 2: Implementation
**File(s)**: main.go
**Depends on**: Task 1
**Estimated time**: 5 minutes

Implement main logic.
```

2. **Validate the plan**

```bash
./conductor validate my-plan.md
```

3. **Run the plan**

```bash
./conductor run my-plan.md --verbose
```

## Basic Usage

### Validate a Plan

Validates plan syntax, dependencies, and detects cycles:

```bash
conductor validate plan.md
```

### Run a Plan

Executes the plan with parallel task orchestration:

```bash
# Normal execution
conductor run plan.md

# Dry run (no actual execution)
conductor run plan.md --dry-run

# With custom concurrency limit
conductor run plan.md --max-concurrency 5

# With timeout
conductor run plan.md --timeout 30m

# Verbose output
conductor run plan.md --verbose

# Custom log directory
conductor run plan.md --log-dir ./execution-logs
```

### Command-Line Flags

**`conductor run` flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | false | Simulate execution without running tasks |
| `--max-concurrency` | int | 3 | Maximum parallel tasks per wave |
| `--timeout` | duration | 30m | Timeout for entire execution |
| `--verbose` | bool | false | Enable detailed logging |
| `--log-dir` | string | .conductor/logs | Directory for execution logs |

**`conductor validate` flags:**

No flags required. Simply pass the plan file path.

## Configuration

Conductor supports optional configuration via `.conductor/config.yaml`:

```bash
# Copy example config
cp .conductor/config.yaml.example .conductor/config.yaml

# Edit configuration
vim .conductor/config.yaml
```

**Example configuration:**

```yaml
# Execution settings
max_concurrency: 3        # Parallel tasks per wave
task_timeout: 5m          # Individual task timeout
execution_timeout: 30m    # Total execution timeout

# Quality control settings
qc_enabled: true          # Enable QC reviews
max_retries: 2            # Retry attempts on RED

# Logging settings
log_dir: .conductor/logs  # Log directory
log_level: info           # Log level (debug/info/warn/error)
```

For detailed configuration options, see [Usage Guide](docs/usage.md).

## Plan Format

Conductor supports two plan formats:

### Markdown Format

```markdown
# Plan Title

## Task 1: Task Name
**File(s)**: file1.go, file2.go
**Depends on**: None
**Estimated time**: 5 minutes
**Agent**: code-implementation

Task description and requirements.
```

### YAML Format

```yaml
plan:
  name: Plan Title
  tasks:
    - id: 1
      name: Task Name
      files: [file1.go, file2.go]
      depends_on: []
      estimated_time: 5 minutes
      agent: code-implementation
      description: Task description and requirements.
```

For complete format specifications, see [Plan Format Guide](docs/plan-format.md).

## Conductor Plugin (Included)

Generate implementation plans automatically using Claude Code slash commands. The plugin is included in this monorepo under the `plugin/` directory.

### Installation

```bash
# From the conductor repository root
cp -r plugin ~/.claude/plugins/conductor-tools

# Verify installation
ls ~/.claude/plugins/conductor-tools
```

### Usage

```bash
# Use commands in Claude Code
/doc design.md                  # Generate Markdown plan
/doc-yaml design.md            # Generate YAML plan
/cook-man design.md            # Interactive plan generation
/cook-auto design.md           # Automated plan generation (optional MCP)
```

See [plugin/docs](plugin/docs) for complete plugin documentation.

**What it does:**
- Design → Plan conversion (Markdown & YAML)
- Interactive or automated plan generation
- Integrates seamlessly with conductor binary

## Documentation

- **[Usage Guide](docs/usage.md)** - Detailed CLI reference and examples
- **[Plan Format Guide](docs/plan-format.md)** - Plan format specifications
- **[Troubleshooting Guide](docs/troubleshooting.md)** - Common issues and solutions
- **[Plugin Docs](plugin/docs)** - Plan generation & design tools

## Architecture Overview

```
Plan File (.md/.yaml)
  → Parser (auto-detects format)
  → Dependency Graph (Kahn's algorithm for topological sort)
  → Wave Calculator (groups parallel-executable tasks)
  → Orchestrator (coordinates execution)
  → Wave Executor (manages parallel task execution)
  → Task Executor (spawns claude CLI, handles retries)
  → Quality Control (reviews output, GREEN/RED/YELLOW)
  → Plan Updater (marks tasks complete)
```

**Key Components:**

- **Parser**: Auto-detects and parses Markdown/YAML plan files
- **Graph Builder**: Calculates dependencies using Kahn's algorithm
- **Orchestrator**: Coordinates wave-based execution with bounded concurrency
- **Task Executor**: Spawns Claude CLI agents with timeout and retry logic
- **Quality Control**: Reviews task outputs using dedicated QC agent
- **Plan Updater**: Thread-safe updates to plan files with file locking

## Development

### Build from Source

```bash
# Build binary
go build ./cmd/conductor

# Run tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run specific package tests
go test ./internal/executor/ -v
```

### Testing

- Extensively tested for reliability and concurrency
- Test-driven development workflow
- Validated across task orchestration, dependency resolution, and multi-agent coordination

### Code Quality

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Vet code
go vet ./...
```

## Project Status

**Current Status**: Production-ready v1.0

### Conductor Core
- ✅ Complete implementation with full test coverage
- ✅ `conductor validate` and `conductor run` commands
- ✅ Wave-based parallel execution
- ✅ Quality control reviews
- ✅ File locking for concurrent updates
- ✅ Agent discovery system
- ✅ Comprehensive documentation

### Conductor Plugin
- ✅ 4 slash commands (`/doc`, `/doc-yaml`, `/cook-auto`, `/cook-man`)
- ✅ Complete documentation and guides
- ✅ Three-tier accessibility (executor, generator, automation)
- ✅ AI Counsel MCP integration (optional)

## Dependencies

Minimal external dependencies:

- `github.com/spf13/cobra` - CLI framework
- `github.com/yuin/goldmark` - Markdown parsing
- `gopkg.in/yaml.v3` - YAML parsing

No networking, no ML frameworks, no web frameworks. Just local CLI orchestration.

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass (`go test ./...`)
5. Submit a pull request

## License

[MIT License](LICENSE)

## Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/blueman82/conductor/issues)
- **Troubleshooting**: [Troubleshooting Guide](docs/troubleshooting.md)

## Acknowledgments

Built with Go and powered by Claude Code CLI agents.
