# Conductor

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
- **Skip Completed Tasks**: Resume interrupted plans by skipping already-completed tasks
- **File Locking**: Safe concurrent updates to plan files
- **Agent Discovery**: Automatic detection of available Claude agents
- **Dual Format Support**: Both Markdown and YAML plan formats
- **Dry Run Mode**: Test execution without actually running tasks
- **Progress Logging**: Real-time console output and file-based logs
- **Adaptive Learning** (v2.0+): AI-powered learning system that improves over time
  - Tracks execution history and failure patterns
  - Automatically adapts agent selection after failures
  - Learns from past successes to optimize future runs
  - CLI commands for statistics and insights

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
```

#### Using Make (Recommended)

```bash
# Install to $GOPATH/bin
make install

# Verify installation
conductor --version
```

#### Manual Build

```bash
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
| `--dry-run` | bool | false | Simulate execution without running tasks |
| `--max-concurrency` | int | 3 | Maximum parallel tasks per wave |
| `--timeout` | duration | 30m | Timeout for entire execution |
| `--verbose` | bool | false | Enable detailed logging |
| `--log-dir` | string | .conductor/logs | Directory for execution logs |
| `--skip-completed` | bool | false | Skip tasks marked as completed in plan |
| `--retry-failed` | bool | false | Retry tasks marked as failed in plan |

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
timeout: 30m              # Total execution timeout
dry_run: false            # Simulate without executing

# Resume & retry settings
skip_completed: false     # Skip tasks marked as completed
retry_failed: false       # Retry tasks marked as failed

# Logging settings
log_dir: .conductor/logs  # Log directory
log_level: info           # Log level (debug/info/warn/error)
```

For detailed configuration options, see [Usage Guide](docs/usage.md).

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
# First run - task fails with backend-developer agent
$ conductor run plan.md
# [ERROR] Task 3 failed: compilation error

# Second run - task fails again
$ conductor run plan.md --retry-failed
# [ERROR] Task 3 failed: compilation error

# Third run - learning system adapts
$ conductor run plan.md --retry-failed
# [INFO] Adapting agent: backend-developer → golang-pro
# [SUCCESS] Task 3 completed successfully
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
2. **QC-Review Hook**: Extracts failure patterns from quality control output
3. **Post-Task Hook**: Records execution results to SQLite database

Learning data is stored locally in `.conductor/learning/` (excluded from git).

**See [Learning System Guide](docs/learning.md) for complete documentation.**

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

- **[Learning System Guide](docs/learning.md)** - Adaptive learning documentation (v2.0+)
- **[Usage Guide](docs/usage.md)** - Detailed CLI reference and examples
- **[Plan Format Guide](docs/plan-format.md)** - Plan format specifications
- **[Phase 2A Guide](docs/phase-2a-guide.md)** - Multi-file plans and plan splitting
- **[Troubleshooting Guide](docs/troubleshooting.md)** - Common issues and solutions
- **[Split Plan Examples](docs/examples/split-plan-README.md)** - Example split plans
- **[Plugin Docs](plugin/docs)** - Plan generation & design tools

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

Conductor uses semantic versioning with automatic version bumping:

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

The VERSION file serves as the single source of truth and is automatically injected into the binary at build time.

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

See [Phase 2A Guide](docs/phase-2a-guide.md) for detailed documentation and examples.

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
