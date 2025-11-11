# Conductor Usage Guide

Complete reference for using the Conductor CLI tool.

## Table of Contents

- [Installation](#installation)
- [Command Reference](#command-reference)
- [Configuration](#configuration)
- [Plan Execution](#plan-execution)
- [Output Interpretation](#output-interpretation)
- [Real-World Examples](#real-world-examples)

## Installation

### Prerequisites

1. **Go 1.21 or higher**
   ```bash
   go version  # Should be 1.21+
   ```

2. **Claude Code CLI**
   ```bash
   claude --version  # Must be in PATH
   claude auth status  # Must be authenticated
   ```

3. **Git** (optional, for cloning repository)

### Build from Source

```bash
# Clone repository
git clone https://github.com/harrison/conductor.git
cd conductor

# Build binary
go build ./cmd/conductor

# Verify installation
./conductor --version

# Optional: Install to PATH
go install ./cmd/conductor
```

## Command Reference

### `conductor validate`

Validates plan file syntax, dependencies, and detects cycles.

**Usage:**
```bash
conductor validate <plan-file>
```

**Arguments:**
- `<plan-file>` - Path to plan file (.md or .yaml)

**Examples:**
```bash
# Validate Markdown plan
conductor validate implementation-plan.md

# Validate YAML plan
conductor validate plan.yaml

# Validate plan in subdirectory
conductor validate docs/plans/feature-plan.md
```

**Output:**
```
✓ Plan file loaded successfully
✓ Found 10 tasks
✓ No circular dependencies detected
✓ All dependencies valid
✓ Plan is valid
```

### `conductor run`

Executes implementation plan with parallel task orchestration.

**Usage:**
```bash
conductor run [flags] <plan-file>
```

**Arguments:**
- `<plan-file>` - Path to plan file (.md or .yaml)

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | false | Simulate execution without running tasks |
| `--max-concurrency` | int | 3 | Maximum parallel tasks per wave |
| `--timeout` | duration | 30m | Timeout for entire execution |
| `--verbose` | bool | false | Enable detailed logging |
| `--log-dir` | string | .conductor/logs | Directory for execution logs |

**Examples:**

```bash
# Basic execution
conductor run plan.md

# Dry run (no actual execution)
conductor run plan.md --dry-run

# Custom concurrency limit
conductor run plan.md --max-concurrency 5

# With timeout
conductor run plan.md --timeout 1h

# Verbose output
conductor run plan.md --verbose

# Custom log directory
conductor run plan.md --log-dir ./execution-logs

# Combined flags
conductor run plan.md \
  --max-concurrency 5 \
  --timeout 1h \
  --verbose \
  --log-dir ./logs
```

### `conductor --version`

Displays version information.

**Usage:**
```bash
conductor --version
```

### `conductor --help`

Shows help information for all commands.

**Usage:**
```bash
conductor --help
conductor validate --help
conductor run --help
```

## Plan Execution Modes

### Skip Completed Tasks

Conductor can resume interrupted plans by skipping already-completed tasks. This is useful when:
- A long-running plan is interrupted mid-execution
- You want to re-run a plan but skip previously successful tasks
- You're iterating on specific failing tasks

**Mark tasks as completed in your plan file:**

```markdown
## Task 1: Already Completed
**Status**: completed

This task has already been executed successfully.
```

```yaml
tasks:
  - id: 1
    name: Already Completed
    status: completed
    description: This task has already been executed successfully.
```

**Skip completed tasks during execution:**

```bash
# Skip all completed tasks
conductor run plan.md --skip-completed

# Skip completed tasks and retry failed ones
conductor run plan.md --skip-completed --retry-failed

# Or set as default in config file
skip_completed: true
```

**Behavior:**
- Tasks marked with `Status: completed` are skipped
- Skipped tasks create synthetic GREEN results
- Dependencies from skipped tasks are still satisfied
- Pending and failed tasks execute normally

### Retry Failed Tasks

Retry previously failed tasks on plan resume:

```bash
# Retry tasks marked as failed
conductor run plan.md --retry-failed

# Combined: skip completed and retry failed
conductor run plan.md --skip-completed --retry-failed
```

**Behavior:**
- Without `--retry-failed`: failed tasks are skipped
- With `--retry-failed`: failed tasks are re-executed
- Use to fix issues and continue a partially-failed plan

## Configuration

### Configuration File

Conductor supports optional configuration via `.conductor/config.yaml`.

**Create configuration:**
```bash
# Copy example config
mkdir -p .conductor
cp .conductor/config.yaml.example .conductor/config.yaml

# Edit configuration
vim .conductor/config.yaml
```

**Configuration Schema:**

```yaml
# Execution settings
max_concurrency: 3        # Maximum parallel tasks per wave (default: 3)
timeout: 30m              # Total execution timeout (default: 30m)
dry_run: false            # Simulate without executing (default: false)

# Resume & retry settings
skip_completed: false     # Skip tasks marked as completed (default: false)
retry_failed: false       # Retry tasks marked as failed (default: false)

# Quality control settings
qc_enabled: true          # Enable QC reviews (default: true)
max_retries: 2            # Retry attempts on RED verdict (default: 2)
qc_agent: quality-control # Agent for QC reviews (default: quality-control)

# Logging settings
log_dir: .conductor/logs  # Log directory (default: .conductor/logs)
log_level: info           # Log level: debug, info, warn, error (default: info)
log_to_file: true         # Enable file logging (default: true)
log_to_console: true      # Enable console logging (default: true)

# Agent settings
agent_dir: ~/.claude/agents  # Directory for agent discovery (default: ~/.claude/agents)

# File locking settings
lock_timeout: 30s         # File lock acquisition timeout (default: 30s)
lock_retry_interval: 100ms # Lock retry interval (default: 100ms)
```

### Configuration Priority

Configuration is loaded in this order (later overrides earlier):

1. Default values (hardcoded in binary)
2. `.conductor/config.yaml` (if exists)
3. Command-line flags (highest priority)

## Plan Execution

### Execution Flow

1. **Load Plan**: Parse plan file (auto-detect format)
2. **Validate**: Check dependencies, detect cycles
3. **Calculate Waves**: Group parallel-executable tasks
4. **Execute Waves**: Run tasks in waves (parallel within, sequential between)
5. **Quality Control**: Review each task output
6. **Retry Logic**: Retry RED verdicts up to max_retries
7. **Update Plan**: Mark completed tasks
8. **Log Results**: Write execution logs

### Wave-Based Execution

Tasks execute in waves based on dependencies:

```
Wave 1: Tasks with no dependencies (parallel)
  ↓
Wave 2: Tasks depending only on Wave 1 (parallel)
  ↓
Wave 3: Tasks depending on Wave 1 or 2 (parallel)
  ↓
...
```

### Concurrency Control

Maximum parallel tasks controlled by `--max-concurrency` flag:

```bash
# Run at most 5 tasks in parallel per wave
conductor run plan.md --max-concurrency 5
```

### Quality Control

Each task output is reviewed by a quality control agent:

**Verdicts:**
- **GREEN**: Task completed successfully, proceed
- **YELLOW**: Task completed with warnings, proceed
- **RED**: Task failed, retry up to max_retries

## Output Interpretation

### Console Output

**Normal Mode:**
```
Loading plan from implementation-plan.md
Validating plan...
Found 10 tasks in 5 waves

Wave 1 (2 tasks):
  [1/2] Task 1: Setup... ✓ GREEN (2m15s)
  [2/2] Task 2: Config... ✓ GREEN (1m30s)

Execution completed successfully
Total time: 29m15s
```

### Log Files

Execution logs written to `.conductor/logs/` (or custom `--log-dir`):

```bash
.conductor/logs/
├── conductor-2025-11-10-143000.log
├── task-1-setup-143015.log
├── task-2-config-143030.log
└── ...
```

## Real-World Examples

### Example 1: Feature Implementation

**Plan file (`feature-plan.md`):**
```markdown
# User Authentication Feature

## Task 1: Database Schema
**File(s)**: migrations/001_users.sql
**Estimated time**: 5 minutes

Create users table with authentication fields.

## Task 2: User Model
**File(s)**: models/user.go
**Depends on**: Task 1
**Estimated time**: 10 minutes

Implement User model with password hashing.

## Task 3: Auth Service
**File(s)**: services/auth.go
**Depends on**: Task 2
**Estimated time**: 15 minutes

Implement authentication service with JWT tokens.

## Task 4: API Endpoints
**File(s)**: handlers/auth.go
**Depends on**: Task 3
**Estimated time**: 15 minutes

Implement /login and /register endpoints.

## Task 5: Documentation
**File(s)**: docs/api/authentication.md
**Depends on**: Task 4
**Estimated time**: 10 minutes

Document authentication API endpoints.
```

**Execution:**
```bash
# Validate plan
conductor validate feature-plan.md

# Run with verbose output
conductor run feature-plan.md --verbose

# Run with increased concurrency
conductor run feature-plan.md --max-concurrency 5 --verbose
```

### Example 2: Parallel Development

**Plan file (`parallel-plan.md`):**
```markdown
# Multi-Component Feature

## Task 1: Foundation
**File(s)**: base.go
**Estimated time**: 10 minutes

Create foundation.

## Task 2: Component A
**File(s)**: component_a.go
**Depends on**: Task 1
**Estimated time**: 15 minutes

Implement component A.

## Task 3: Component B
**File(s)**: component_b.go
**Depends on**: Task 1
**Estimated time**: 15 minutes

Implement component B.

## Task 4: Integration
**File(s)**: integration.go
**Depends on**: Task 2, Task 3
**Estimated time**: 10 minutes

Integrate components.
```

**Execution:**
```bash
# Execute with high parallelism
conductor run parallel-plan.md --max-concurrency 5 --verbose
```

**Wave execution:**
- Wave 1: Task 1 (Foundation)
- Wave 2: Task 2, Task 3 (parallel)
- Wave 3: Task 4 (Integration)

### Example 3: Microservices

**Execution:**
```bash
# High concurrency for parallel service development
conductor run microservices-plan.md \
  --max-concurrency 6 \
  --timeout 2h \
  --verbose
```

## Best Practices

### Plan Design
1. **Start with independent tasks**: Maximize parallelism in Wave 1
2. **Group related tasks**: Keep dependencies logical and minimal
3. **Avoid over-dependencies**: Only depend on what you actually need
4. **Balance task size**: 5-15 minute tasks work best
5. **Use meaningful names**: Clear task names help debugging

### Execution
1. **Validate first**: Always validate before running
2. **Use dry run**: Test execution plan with `--dry-run`
3. **Start conservative**: Use low `--max-concurrency` initially
4. **Monitor logs**: Check log files for issues
5. **Keep plans updated**: Mark completed tasks to avoid re-execution

### Performance
1. **Tune concurrency**: Adjust based on task complexity and resource usage
2. **Set appropriate timeouts**: Allow enough time for complex tasks
3. **Use agent specialization**: Assign appropriate agents for tasks
4. **Split large tasks**: Break into smaller, parallelizable subtasks
5. **Review QC feedback**: Use to improve future plans

## See Also

- [Plan Format Guide](plan-format.md) - Plan format specifications
- [Troubleshooting Guide](troubleshooting.md) - Common issues and solutions
- [README](../README.md) - Project overview
