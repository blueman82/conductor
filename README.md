# Conductor

**Autonomous Multi-Agent Orchestration for Claude Code**

Conductor is a production-ready Go CLI tool that executes implementation plans by orchestrating multiple Claude Code agents in coordinated parallel waves. It parses plan files, calculates task dependencies using graph algorithms, and manages parallel execution with quality control reviews.

## Table of Contents

- [What It Does](#what-it-does)
- [Key Features](#key-features)
- [Quick Start](#quick-start)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
  - [Your First Execution](#your-first-execution)
- [Basic Usage](#basic-usage)
  - [Validate a Plan](#validate-a-plan)
  - [Run a Plan](#run-a-plan)
  - [Resume Interrupted Plans](#resume-interrupted-plans)
  - [Command-Line Flags](#command-line-flags)
- [Configuration](#configuration)
  - [Config File Location](#config-file-location)
  - [Setup](#setup)
  - [Configuration Priority](#configuration-priority)
  - [Example Configuration](#example-configuration)
  - [Build-Time Configuration](#build-time-configuration)
- [Adaptive Learning System](#adaptive-learning-system-v20)
- [Plan Format](#plan-format)
  - [Markdown Format](#markdown-format)
  - [YAML Format](#yaml-format)
- [Conductor Plugin](#conductor-plugin-included)
- [Documentation](#documentation)
- [Architecture Overview](#architecture-overview)
- [Development](#development)
- [Multi-File Plans](#multi-file-plans)
- [Project Status](#project-status)
- [Dependencies](#dependencies)
- [Contributing](#contributing)
- [License](#license)
- [Support](#support)
- [Acknowledgments](#acknowledgments)

---

## What It Does

Conductor automates complex multi-step implementations by:

1. **Parsing** implementation plans in Markdown or YAML format
2. **Calculating** task dependencies and execution waves using topological sorting
3. **Orchestrating** parallel execution of tasks within each wave
4. **Spawning** Claude Code CLI agents to execute individual tasks
5. **Reviewing** task outputs with quality control agents
6. **Managing** retries on failures and marking completed tasks
7. **Logging** detailed execution progress with file-based output

[⬆ back to top](#table-of-contents)

## Key Features

- **Wave-Based Execution**: Tasks execute in parallel within waves, sequentially between waves
- **Dependency Management**: Automatic dependency graph calculation with cycle detection
- **Quality Control**: Automated review of task outputs (GREEN/RED/YELLOW verdicts)
- **Retry Logic**: Automatic retry on failures (up to 2 attempts per task)
- **Skip Completed Tasks**: Resume interrupted plans by skipping already-completed tasks
- **File Locking**: Safe concurrent updates to plan files
- **Agent Discovery**: Automatic detection of available Claude agents
- **Dual Format Support**: Both Markdown and YAML plan formats
- **Dry Run Mode**: Test execution without actually running tasks
- **Progress Logging**: Real-time console output and file-based logs
- **Adaptive Learning** (v2.0+): AI-powered learning system that improves over time
  - Tracks execution history and failure patterns
  - Automatically adapts agent selection after failures
  - Inter-retry agent swapping based on QC suggestions
  - Dual feedback storage (plain text + structured JSON)
  - Learns from past successes to optimize future runs
  - CLI commands for statistics and insights

[⬆ back to top](#table-of-contents)

## Quick Start

### Prerequisites

- **Claude Code CLI** in your PATH (`claude` command available)
- Claude Code authenticated and configured
- **Go 1.21+** (optional, only needed for building from source)

### Installation

#### Option 1: Quick Install (Recommended)

Download pre-built binary from the latest release:

```bash
# macOS (Apple Silicon)
curl -L https://github.com/blueman82/conductor/releases/download/v2.0.5/conductor-darwin-arm64 -o conductor
chmod +x conductor
sudo mv conductor /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/blueman82/conductor/releases/download/v2.0.5/conductor-darwin-amd64 -o conductor
chmod +x conductor
sudo mv conductor /usr/local/bin/

# Linux (x86_64)
curl -L https://github.com/blueman82/conductor/releases/download/v2.0.5/conductor-linux-amd64 -o conductor
chmod +x conductor
sudo mv conductor /usr/local/bin/

# Verify installation
conductor --version
# v2.0.5
```

#### Option 2: Build from Source

Clone and build locally (requires Go 1.21+):

```bash
# Clone the repository
git clone https://github.com/blueman82/conductor.git
cd conductor

# Install to $GOPATH/bin using Make (Recommended)
make install

# Or manual build
go build ./cmd/conductor

# Verify installation
conductor --version
# v2.0.5
```

#### Option 3: Install Plugin for Plan Generation

In Claude Code, run:

```bash
/plugin
```

Then select "Browse Plugins" to install **conductor-tools** for:
- `/doc` - Generate Markdown plans
- `/doc-yaml` - Generate YAML plans
- `/cook-man` - Interactive plan design
- `/cook-auto` - Autonomous plan generation (optional MCP)

### Your First Execution

1. **Create a plan file** (see [Plan Format Guide](docs/conductor.md#plan-format))

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

[⬆ back to top](#table-of-contents)

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

# Skip completed tasks and retry failed ones
conductor run plan.md --skip-completed --retry-failed
```

### Resume Interrupted Plans

Conductor supports resumable execution by skipping tasks marked as completed:

```bash
# Run plan, marking completed tasks in the file
conductor run plan.md

# Resume the plan later, skipping completed tasks
conductor run plan.md --skip-completed

# Retry failed tasks on resume
conductor run plan.md --skip-completed --retry-failed
```

### Command-Line Flags

**`conductor run` flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `.conductor/config.yaml` | Path to config file (loaded from conductor repo root) |
| `--dry-run` | bool | false | Validate plan without executing tasks |
| `--max-concurrency` | int | -1 | Max concurrent tasks per wave (-1 = use config, 0 = unlimited) |
| `--timeout` | duration | 10h | Maximum execution time (e.g., 30m, 2h, 1h30m) |
| `--verbose` | bool | false | Show detailed execution information |
| `--log-dir` | string | `.conductor/logs` | Directory for execution logs |
| `--skip-completed` | bool | false | Skip tasks marked as completed in plan |
| `--no-skip-completed` | bool | false | Do not skip completed tasks (overrides config) |
| `--retry-failed` | bool | false | Retry tasks marked as failed in plan |
| `--no-retry-failed` | bool | false | Do not retry failed tasks (overrides config) |

**`conductor validate` flags:**

No flags required. Simply pass the plan file path.

**`conductor learning` commands:**

```bash
# View statistics and insights for a plan
conductor learning stats <plan-file>

# Show detailed history for a specific task
conductor learning show <plan-file> <task-name>

# Export learning data to JSON for analysis
conductor learning export <output-file>

# Clear learning history (with confirmation)
conductor learning clear
```

[⬆ back to top](#table-of-contents)

## Configuration

### Config File Location

Configuration file is automatically loaded from the **conductor repository root**:
```
{conductor-repo-root}/.conductor/config.yaml
```

This ensures consistent configuration regardless of your current working directory.

### Setup

```bash
# Copy example config to your conductor repository
cp .conductor/config.yaml.example .conductor/config.yaml

# Edit configuration
vim .conductor/config.yaml
```

### Configuration Priority

Settings are applied in this order (highest to lowest priority):
1. **CLI Flags** (e.g., `--max-concurrency 5`)
2. **Config File** (`.conductor/config.yaml`)
3. **Built-in Defaults**

CLI flags always override config file settings.

### Example Configuration

```yaml
# Execution settings
max_concurrency: 3        # Parallel tasks per wave (0 = unlimited)
timeout: 10h              # Total execution timeout
dry_run: false            # Simulate without executing

# Resume & retry settings
skip_completed: false     # Skip tasks marked as completed
retry_failed: false       # Retry tasks marked as failed

# Logging settings
log_dir: .conductor/logs  # Log directory
log_level: info           # Log level (debug/info/warn/error)

# Adaptive Learning (v2.0+)
learning:
  enabled: true           # Enable learning system (default: true)
  auto_adapt_agent: false # Auto-switch agents on failures (default: false)
  enhance_prompts: true   # Add learned context to prompts (default: true)
  min_failures_before_adapt: 2  # Failure threshold before adapting
  keep_executions_days: 90      # Retention period for learning data
```

For detailed configuration options, see [Usage Guide](docs/conductor.md#usage--commands).

### Build-Time Configuration

When you build conductor with `make build`, two values are automatically injected into the binary:

1. **Version** - From the VERSION file (e.g., 2.0.1)
2. **Repository Root** - Path to conductor repository

This ensures:
- ✅ Database always created in conductor repo (`.conductor/learning/executions.db`)
- ✅ Config always loaded from conductor repo (`.conductor/config.yaml`)
- ✅ Works correctly from any directory

[⬆ back to top](#table-of-contents)

## Adaptive Learning System (v2.0+)

Conductor v2.0 introduces an intelligent learning system that automatically improves task execution over time by learning from past successes and failures.

### What It Does

- **Records Execution History**: Tracks every task execution with agent, verdict, patterns, and timing
- **Detects Failure Patterns**: Identifies common issues (compilation errors, test failures, missing dependencies)
- **Adapts Agent Selection**: Automatically switches to better-performing agents after repeated failures
- **Enhances Prompts**: Enriches task prompts with learned context from execution history
- **Provides Insights**: CLI commands for viewing statistics, history, and trends

### Quick Example

```bash
# Run with inter-retry agent swapping enabled
$ conductor run plan.md --verbose

# Task fails with backend-developer agent
# [ERROR] Task 3 failed: compilation error

# QC suggests golang-pro agent
# [INFO] QC suggested agent: golang-pro

# Conductor automatically swaps to golang-pro for retry
# [INFO] Adapting agent: backend-developer → golang-pro
# [SUCCESS] Task 3 completed successfully with golang-pro
```

### CLI Commands

```bash
# View learning statistics for a plan
$ conductor learning stats plan.md

# Show execution history for a specific task
$ conductor learning show plan.md "Task 3"

# Export learning data for analysis
$ conductor learning export plan.md --format json

# Clear learning data
$ conductor learning clear plan.md
```

### Configuration

Enable learning in `.conductor/config.yaml`:

```yaml
learning:
  enabled: true                    # Master switch (default: true)
  auto_adapt_agent: true          # Auto-switch agents on failures (default: false)
  enhance_prompts: true           # Add learned context to prompts (default: true)
  min_failures_before_adapt: 2    # Failure threshold (default: 2)
  keep_executions_days: 90        # Data retention (default: 90)
```

### How It Works

1. **Pre-Task Hook**: Analyzes execution history and adapts strategy before task runs
2. **Task Execution**: Runs task with selected agent, captures structured JSON output
3. **QC Review**: Quality control agent evaluates output, suggests alternative agents if needed
4. **Inter-Retry Swapping**: On RED verdict with suggested agent, automatically swaps for retry
5. **Dual Storage**: Persists both plain text feedback and structured JSON to database
6. **Post-Task Hook**: Records execution results to SQLite database for pattern analysis

Learning data is stored locally in `.conductor/learning/` (excluded from git).

**See [Learning System Guide](docs/conductor.md#adaptive-learning-system) for complete documentation.**

[⬆ back to top](#table-of-contents)

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

For complete format specifications, see [Plan Format Guide](docs/conductor.md#plan-format).

[⬆ back to top](#table-of-contents)

## Conductor Tools Plugin

Generate Conductor-ready implementation plans directly from Claude Code using the **Conductor Tools Plugin**.

### Quick Installation

In Claude Code:

```bash
/plugin
```

Search for **conductor-tools** and click install. Done! Your commands are ready:

```bash
/doc "feature description"           # Generate Markdown plan
/doc-yaml "feature description"      # Generate YAML plan
/cook-man "feature description"      # Interactive design session
```

### Why Use the Plugin?

- ✅ **Auto-discovers** your project structure (reads codebase before planning)
- ✅ **Smart agent selection** - recommends appropriate agents
- ✅ **Dependency detection** - identifies task ordering
- ✅ **Multiple formats** - Markdown (human) or YAML (automation)
- ✅ **Interactive design** - `/cook-man` refines requirements before planning
- ✅ **Immediately executable** - generated plans work directly with `conductor run`

### Quick Example

```bash
# Generate a plan
/doc "Add OAuth2 authentication with JWT"

# Execute immediately
conductor run generated-plan.md --verbose
```

### Available Commands

| Command | Output | Best For |
|---------|--------|----------|
| `/doc` | Markdown `.md` | Human-readable, team discussion |
| `/doc-yaml` | YAML `.yaml` | Automation, tooling, config management |
| `/cook-man` | Interactive | Complex features, design validation |

### Documentation

- **[Plugin README](plugin/conductor-tools/README.md)** - Features and examples
- **[Installation Guide](plugin/conductor-tools/INSTALLATION.md)** - Detailed setup (with troubleshooting)
- **[Plugin Source](plugin/conductor-tools/)** - Implementation details

### Manual Installation (if needed)

```bash
# From conductor repository
cd plugin/conductor-tools
cp -r . ~/.claude/plugins/conductor-tools
# Restart Claude Code
```

See [Installation Guide](plugin/conductor-tools/INSTALLATION.md) for other methods and troubleshooting.

[⬆ back to top](#table-of-contents)

## Documentation

- **[Complete Reference Guide](docs/conductor.md)** - Comprehensive guide covering usage, plan formats, multi-file plans, learning system, and troubleshooting
- **[Development Setup](docs/setup.md)** - Local development environment setup
- **[Split Plan Examples](docs/examples/split-plan-README.md)** - Example split plans
- **[Plugin Docs](plugin/docs)** - Plan generation & design tools

[⬆ back to top](#table-of-contents)

## Architecture Overview

### Execution Pipeline

**Single-File Plans:**
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

**Multi-File Plans:**
```
Multiple Plan Files (.md/.yaml)
  → Multi-File Loader (auto-detects format per file)
  → Plan Merger (validates, deduplicates, merges)
  → Dependency Graph (cross-file dependencies)
  → Wave Calculator (respects worktree groups)
  → [Rest of pipeline as above]
  → Plan Updater (file-aware task tracking)
```

**Key Components:**

- **Parser**: Auto-detects and parses Markdown/YAML plan files
- **Multi-File Loader**: Loads and merges multiple plans with cross-file dependency validation
- **Graph Builder**: Calculates dependencies using Kahn's algorithm
- **Orchestrator**: Coordinates wave-based execution with bounded concurrency
- **Task Executor**: Spawns Claude CLI agents with timeout and retry logic
- **Quality Control**: Reviews task outputs using dedicated QC agent
- **Plan Updater**: Thread-safe updates to plan files with file locking
- **Worktree Groups**: Organize tasks into execution groups with isolation levels

[⬆ back to top](#table-of-contents)

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

### Version Management

Conductor uses semantic versioning with automatic version bumping.

**Recommended**: Download pre-built binaries from [GitHub Releases](https://github.com/blueman82/conductor/releases) to ensure version consistency without needing to build locally.

**For developers** building from source:

```bash
# View current version
cat VERSION

# Build with current version
make build

# Bump patch version (1.1.0 → 1.1.1) and build
make build-patch

# Bump minor version (1.1.0 → 1.2.0) and build
make build-minor

# Bump major version (1.1.0 → 2.0.0) and build
make build-major
```

The VERSION file serves as the single source of truth and is automatically injected into the binary at build time. When using pre-built releases, versions always match - no build step needed.

[⬆ back to top](#table-of-contents)

## Multi-File Plans

Conductor supports splitting large implementation plans across multiple files with automatic merging and dependency management:

```bash
# Load and execute multiple plan files
conductor run setup.md features.md deployment.md

# Validate split plans before execution
conductor validate *.md
```

**Features:**
- ✅ Multi-file plan loading with auto-format detection
- ✅ Objective plan splitting by feature/component/service
- ✅ Cross-file dependency management
- ✅ Worktree groups for execution control
- ✅ File-to-task mapping for resume operations
- ✅ 100% backward compatible with single-file plans

See [Multi-File Plans Guide](docs/conductor.md#multi-file-plans--objective-splitting) for detailed documentation and examples.

[⬆ back to top](#table-of-contents)

## Project Status

**Current Status**: Production-ready v2.0.0

Conductor is feature-complete with:
- ✅ Complete implementation with 86%+ test coverage
- ✅ `conductor validate` and `conductor run` commands
- ✅ Wave-based parallel execution with dependency management
- ✅ Quality control reviews with automated retries
- ✅ Multi-file plan loading and merging
- ✅ Worktree group organization with isolation levels
- ✅ Auto-incrementing version management (VERSION file)
- ✅ File locking for concurrent updates
- ✅ Agent discovery system
- ✅ **Adaptive learning system** (v2.0)
  - SQLite-based execution history
  - Automatic agent adaptation
  - Pattern detection and analysis
  - Four CLI learning commands
- ✅ Comprehensive documentation

### Conductor Plugin
- ✅ 4 slash commands (`/doc`, `/doc-yaml`, `/cook-auto`, `/cook-man`)
- ✅ Complete documentation and guides
- ✅ Three-tier accessibility (executor, generator, automation)
- ✅ AI Counsel MCP integration (optional)

[⬆ back to top](#table-of-contents)

## Dependencies

Minimal external dependencies:

- `github.com/spf13/cobra` - CLI framework
- `github.com/yuin/goldmark` - Markdown parsing
- `gopkg.in/yaml.v3` - YAML parsing

No networking, no ML frameworks, no web frameworks. Just local CLI orchestration.

[⬆ back to top](#table-of-contents)

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass (`go test ./...`)
5. Submit a pull request

[⬆ back to top](#table-of-contents)

## License

[MIT License](LICENSE)

[⬆ back to top](#table-of-contents)

## Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/blueman82/conductor/issues)
- **Troubleshooting**: [Troubleshooting Guide](docs/conductor.md#troubleshooting--faq)

[⬆ back to top](#table-of-contents)

## Acknowledgments

Built with Go and powered by Claude Code CLI agents.
