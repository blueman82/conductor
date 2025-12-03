<p align="center">
  <img src="assets/logo.png" alt="Conductor Logo" width="400">
</p>

<h1 align="center">Conductor</h1>

<p align="center">
  <strong>Autonomous Multi-Agent Orchestration for Claude Code</strong>
</p>

<p align="center">
  <img src="assets/conductor-demo.gif" alt="Conductor Demo" width="700">
</p>

Conductor is a production-ready Go CLI tool that executes implementation plans by orchestrating multiple Claude Code agents in coordinated parallel waves. It parses plan files, calculates task dependencies using graph algorithms, and manages parallel execution with quality control reviews.

## Table of Contents

- [What It Does](#what-it-does)
- [Key Features](#key-features)
- [Quick Start](#quick-start)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
  - [Your First Execution](#your-first-execution)
- [Basic Usage](#basic-usage)
- [Configuration](#configuration)
- [Plan Format](#plan-format)
- [Documentation](#documentation)
- [Project Status](#project-status)
- [Contributing](#contributing)
- [License](#license)
- [Support](#support)

---

## What It Does

Conductor automates complex multi-step implementations by:

1. **Parsing** implementation plans in Markdown or YAML format
2. **Calculating** task dependencies and execution waves using topological sorting
3. **Orchestrating** parallel execution of tasks within each wave
4. **Spawning** Claude Code CLI agents to execute individual tasks
5. **Reviewing** task outputs with quality control agents
6. **Managing** retries on failures and tracking completed tasks
7. **Learning** from execution history to improve future runs

[⬆ back to top](#table-of-contents)

## Key Features

- **Wave-Based Execution**: Tasks execute in parallel within waves, sequentially between waves
- **Dependency Management**: Automatic dependency graph calculation with cycle detection, cross-file dependencies (v2.6+), and validation
- **Quality Control**: Automated review of task outputs with GREEN/RED/YELLOW verdicts
- **Retry Logic**: Automatic retry on failures (up to 2 attempts per task)
- **Skip Completed Tasks**: Resume interrupted plans by skipping already-completed tasks
- **Dry Run Mode**: Test execution without actually running tasks
- **Multi-File Plans** (v2.5+): Split large plans across multiple files with cross-file dependencies
- **Adaptive Learning** (v2.0+): AI-powered learning system that improves over time
- **Structured Success Criteria** (v2.3+): Per-criterion verification with multi-agent consensus
- **Intelligent QC Agent Selection** (v2.4+): Claude-based recommendations for domain specialists
- **Integration Tasks** (v2.5+): Dual-level validation for component and cross-component interactions
- **Agent Watch** (v2.7+): Behavioral analytics for Claude Code agents with real-time streaming
- **JSON Schema Enforcement** (v2.8+): Guaranteed response structure from agents
- **Runtime Enforcement** (v2.9+): Hard gates (test commands) and soft signals (criterion verification)
- **Agent Validation** (v2.13+): Pre-execution validation of agent availability and configuration

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
curl -L https://github.com/blueman82/conductor/releases/download/v2.13.0/conductor-darwin-arm64 -o conductor
chmod +x conductor
sudo mv conductor /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/blueman82/conductor/releases/download/v2.13.0/conductor-darwin-amd64 -o conductor
chmod +x conductor
sudo mv conductor /usr/local/bin/

# Linux (x86_64)
curl -L https://github.com/blueman82/conductor/releases/download/v2.13.0/conductor-linux-amd64 -o conductor
chmod +x conductor
sudo mv conductor /usr/local/bin/

# Verify installation
conductor --version
```

#### Option 2: Build from Source

Clone and build locally (requires Go 1.21+):

```bash
git clone https://github.com/blueman82/conductor.git
cd conductor
make install
conductor --version
```

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
conductor validate my-plan.md
```

3. **Run the plan**

```bash
conductor run my-plan.md --verbose
```

[⬆ back to top](#table-of-contents)

## Basic Usage

### Validate a Plan

Validates plan syntax, dependencies, and detects cycles:

```bash
conductor validate plan.md
conductor validate *.md                    # Multi-file validation
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

# Skip completed tasks and retry failed ones
conductor run plan.md --skip-completed --retry-failed

# Multi-file execution
conductor run setup.md features.md deployment.md
```

### Command-Line Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `.conductor/config.yaml` | Path to config file |
| `--dry-run` | bool | false | Validate without executing |
| `--max-concurrency` | int | -1 | Max parallel tasks per wave |
| `--timeout` | duration | 10h | Maximum execution time |
| `--verbose` | bool | false | Show detailed output |
| `--log-dir` | string | `.conductor/logs` | Log directory |
| `--skip-completed` | bool | false | Skip completed tasks on resume |
| `--retry-failed` | bool | false | Retry failed tasks on resume |

### Learning Commands

```bash
# View statistics for a plan
conductor learning stats plan.md

# Show task execution history
conductor learning show plan.md "Task Name"

# Export learning data
conductor learning export output.json

# Clear learning history
conductor learning clear
```

[⬆ back to top](#table-of-contents)

## Configuration

### Config File

Configuration is loaded from `.conductor/config.yaml` in the conductor repository root:

```yaml
# Execution settings
max_concurrency: 3           # Parallel tasks per wave (0 = unlimited)
timeout: 10h                 # Total execution timeout

# Resume & retry settings
skip_completed: false        # Skip completed tasks
retry_failed: false          # Retry failed tasks

# Logging settings
log_dir: .conductor/logs     # Log directory
log_level: info              # Log level (debug/info/warn/error)

# Adaptive Learning
learning:
  enabled: true
  enhance_prompts: true
  swap_during_retries: true

# Quality Control
quality_control:
  enabled: true
  retry_on_red: 2
  agents:
    mode: intelligent        # auto|explicit|mixed|intelligent
    max_agents: 3
```

**Priority** (highest to lowest):
1. CLI Flags
2. Config File (`.conductor/config.yaml`)
3. Built-in Defaults

For complete configuration options, see [Configuration Guide](docs/conductor.md#configuration).

[⬆ back to top](#table-of-contents)

## Plan Format

Both Markdown and YAML formats support all conductor features (v2.14+) — feature parity between formats means you can choose based on your preference.

### Markdown Example

```markdown
# Plan Title

## Task 1: Task Name
**File(s)**: file1.go, file2.go
**Depends on**: None
**Estimated time**: 5 minutes
**Agent**: code-implementation

**Success Criteria**:
- Criterion 1
- Criterion 2

Task description and requirements.
```

### YAML Example

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
      success_criteria:
        - "Criterion 1"
        - "Criterion 2"
      description: Task description.
```

### Key Features

- **Single & Multi-File Plans**: Support for splitting large plans across multiple files
- **Success Criteria** (v2.3+): Define per-task success criteria for QC verification
- **Integration Tasks** (v2.5+): Distinguish component tasks from integration tasks
- **Cross-File Dependencies** (v2.6+): Reference tasks in other files with explicit notation
- **Test Commands**: Automated testing and validation
- **Runtime Metadata**: Dependency checks and documentation targets
- **Structured Criteria** (v2.9+): Advanced verification with commands and expected results
- **Key Points** (v2.9+): Implementation notes and references

For complete format specifications and examples:
- **Markdown Format**: [Markdown Format Guide](docs/MARKDOWN_FORMAT.md)
- **YAML Format**: [Plan Format Guide](docs/conductor.md#plan-format)

[⬆ back to top](#table-of-contents)

## Documentation

- **[Complete Reference Guide](docs/conductor.md)** - Comprehensive usage, plan formats, learning system, and troubleshooting
- **[Development Setup](docs/setup.md)** - Local development environment
- **[Example Plans](docs/examples/)** - Sample split and multi-file plans
- **[CLAUDE.md](CLAUDE.md)** - Developer guide for Claude Code integration

[⬆ back to top](#table-of-contents)

## Project Status

**Current Status**: Production-ready v2.13.0

Conductor is feature-complete with:
- Wave-based parallel execution with dependency management
- Quality control reviews with automated retries (up to 2 attempts)
- Multi-file plan support with cross-file dependencies
- Adaptive learning system with execution history and pattern detection
- Structured success criteria with per-criterion verification
- Intelligent QC agent selection with domain-specific expertise
- Integration tasks with dual-level validation
- Agent Watch behavioral analytics with real-time streaming
- JSON schema enforcement for guaranteed response structure
- Runtime enforcement with test commands and criterion verification
- 86%+ test coverage with 465+ tests
- Comprehensive documentation and examples

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
