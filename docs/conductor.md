# Conductor Complete Reference

Complete documentation for Conductor - a multi-agent orchestration CLI for autonomous task execution.

---

## Table of Contents

- [Installation & Setup](#installation--setup)
- [Plan Format](#plan-format)
  - [Markdown Format](#markdown-format)
  - [YAML Format](#yaml-format)
  - [Task Metadata](#task-metadata)
- [Usage & Commands](#usage--commands)
  - [conductor validate](#conductor-validate)
  - [conductor run](#conductor-run)
  - [Learning Commands](#learning-commands)
  - [Observe Commands](#observe-commands-agent-watch)
  - [Budget Commands](#budget-commands)
- [Configuration](#configuration)
  - [Quality Control Settings](#quality-control)
  - [Feedback Storage Settings](#dual-feedback-storage-v210)
  - [Learning Settings](#configuration-learning)
  - [Integration Tasks](#integration-tasks-v250)

  - [Pattern Intelligence](#pattern-intelligence-v224)
  - [Budget & Rate Limits](#budget--rate-limits-v220)
- [Multi-File Plans & Objective Splitting](#multi-file-plans--objective-splitting)
  - [Core Concepts](#core-concepts)
  - [Usage Examples](#multi-file-usage-examples)
  - [Best Practices for Split Plans](#best-practices-for-split-plans)
  - [Advanced Patterns](#advanced-patterns)
- [Best Practices](#best-practices)
  - [Worktree Organization](#worktree-organization)
  - [Dependency Design](#dependency-design)
  - [Plan Design Guidelines](#plan-design-guidelines)
- [Adaptive Learning System](#adaptive-learning-system)
  - [Overview](#learning-overview)
  - [How It Works](#how-it-works)
  - [Inter-Retry Agent Swapping](#inter-retry-agent-swapping-v210)
  - [Structured QC Responses](#structured-qc-response-format-v210)
  - [Dual Feedback Storage](#dual-feedback-storage-v210)
  - [Execution History Format](#execution-history-format-v210)
  - [Configuration](#configuration-learning)
  - [CLI Commands](#cli-commands-learning)
- [Text-to-Speech Voice Feedback](#text-to-speech-voice-feedback-v214)
  - [Prerequisites](#prerequisites-1)
  - [Server Setup](#server-setup)
  - [Configuration](#configuration-1)
  - [Available Voices](#available-voices)
- [Troubleshooting & FAQ](#troubleshooting--faq)
- [Development Workflow](#development-workflow)

---

## Installation & Setup

### Prerequisites

1. **Claude Code CLI** (Required)
   ```bash
   claude --version  # Must be in PATH
   claude auth status  # Must be authenticated
   ```

2. **Go 1.21 or higher** (Optional, only needed for building from source)
   ```bash
   go version  # Should be 1.21+ (for source builds only)
   ```

3. **Git** (Optional, for cloning repository)

### Install Pre-Built Binary (Recommended)

Download a pre-built binary from [GitHub Releases](https://github.com/blueman82/conductor/releases):

```bash
# macOS (Apple Silicon)
curl -L https://github.com/blueman82/conductor/releases/download/v2.21.0/conductor-darwin-arm64 -o conductor
chmod +x conductor
sudo mv conductor /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/blueman82/conductor/releases/download/v2.21.0/conductor-darwin-amd64 -o conductor
chmod +x conductor
sudo mv conductor /usr/local/bin/

# Linux
curl -L https://github.com/blueman82/conductor/releases/download/v2.21.0/conductor-linux-amd64 -o conductor
chmod +x conductor
sudo mv conductor /usr/local/bin/

# Verify installation
conductor --version
# v2.21.0
```

**Benefits of using pre-built binaries:**
- ✅ No build step required
- ✅ Version always matches the release
- ✅ Fast installation (30 seconds vs 1-2 minutes to build)
- ✅ Works immediately, no PATH updates needed

### Plan Generation Plugin (Recommended)

Install the **Conductor Tools Plugin** to generate plans from Claude Code:

**In Claude Code:**

```bash
/plugin
```

Search for and install **conductor-tools**

Then use these commands:
```bash
/doc "feature description"           # Generate Markdown plan
/doc-yaml "feature description"      # Generate YAML plan
/cook-man "feature description"      # Interactive design session
```

**Key Benefits:**
- ✅ Analyzes your project structure automatically
- ✅ Recommends appropriate agents based on task type
- ✅ Detects dependencies between tasks
- ✅ Multiple output formats (Markdown or YAML)
- ✅ Plans are immediately executable with `conductor run`

See [Conductor Tools Plugin](../../plugin/conductor-tools/) for complete documentation and [Installation Guide](../../plugin/conductor-tools/INSTALLATION.md) for setup help.

### Build from Source (for Developers)

Clone the repository and build locally:

```bash
# Clone repository
git clone https://github.com/blueman82/conductor.git
cd conductor

# Build binary using Make (Recommended)
# This creates ~/bin/conductor for system-wide access
make build

# Verify the build (use from anywhere if ~/bin in PATH)
conductor --version
```

The `make build` command:
- Compiles the binary to `~/bin/conductor`, making it available system-wide
- **Automatically adds ~/bin to ~/.zshrc and ~/.bashrc if not already in PATH**
- Reminds you to run `source ~/.zshrc` if PATH was added

**PATH Setup:**

The `make build` command automatically adds `~/bin` to your shell configuration files (~/.zshrc and ~/.bashrc) if it's not already in your PATH. After building:

- **If ~/bin was already in PATH**: `conductor` command is ready to use immediately
- **If ~/bin was added by make build**: Run `source ~/.zshrc` (or `source ~/.bashrc`) to activate it in current shell

To verify ~/bin is in PATH:
```bash
echo $PATH | grep ~/bin
```

To manually add ~/bin to PATH (if needed):
```bash
export PATH="$HOME/bin:$PATH"
# Add to ~/.zshrc or ~/.bashrc to make permanent
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc    # For zsh
# OR
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc   # For bash
```

### VERSION File Management

Conductor uses semantic versioning stored in a `VERSION` file at the project root.

**Key Points:**
- The `VERSION` file stores the semantic version (e.g., `2.0.1`)
- Version is injected at build time via `-ldflags` in the Makefile
- The file **is tracked in git** (required for version management)
- Each developer has a local copy that may differ from others (for local development)
- The file **must exist locally** before running make build-patch/minor/major

**Build Commands and VERSION:**

- **`make build`** - Uses current VERSION from file, injects into binary. Works even if VERSION is missing (but shows blank version). No changes to VERSION file.

- **`make build-patch`** - Increments patch version (1.0.0 → 1.0.1), then builds. **Requires VERSION file to exist.**

- **`make build-minor`** - Increments minor version (1.0.0 → 1.1.0), resets patch to 0, then builds. **Requires VERSION file to exist.**

- **`make build-major`** - Increments major version (1.0.0 → 2.0.0), resets minor and patch to 0, then builds. **Requires VERSION file to exist.**

**Example Usage:**
```bash
# Build with current version (2.0.1) to ~/bin/conductor
make build

# Use from anywhere (if ~/bin in PATH)
conductor --version

# Bump patch version to 2.0.2 and build
make build-patch

# Bump minor version to 2.1.0 and build
make build-minor

# Bump major version to 3.0.0 and build
make build-major
```

---

## Plan Format

Conductor supports two plan formats that are automatically detected by file extension.

### Markdown Format

#### Basic Structure

```markdown
# Plan Title

Optional plan description.

## Task N: Task Name
**File(s)**: file1.go, file2.go
**Depends on**: Task 1, Task 2
**Estimated time**: 10 minutes
**Agent**: agent-name

Task description and detailed requirements.

## Task N+1: Next Task
...
```

#### Format Specification

**Plan Header:**
```markdown
# Plan Title

Optional description paragraph.
Can span multiple lines.
```

- **Title**: First H1 heading (`#`) in file
- **Description**: Optional text between title and first task
- **Frontmatter**: Optional YAML configuration

**Task Definition:**
```markdown
## Task N: Task Name
**File(s)**: file1.go, file2.go
**Depends on**: Task 1, Task 2
**Estimated time**: 10 minutes
**Agent**: agent-name
**WorktreeGroup**: group-name
**Status**: completed

Task description and requirements.
Multiple paragraphs allowed.

Code examples also allowed:
\`\`\`go
func example() {}
\`\`\`
```

- **Task Header**: H2 heading (`##`) with `Task N:` prefix
- **Metadata**: Bold key-value pairs
- **Description**: Markdown content after metadata

### YAML Format

#### Basic Structure

```yaml
plan:
  name: Plan Title
  description: Optional plan description
  tasks:
    - id: 1
      name: Task Name
      files: [file1.go, file2.go]
      depends_on: []
      estimated_time: 10 minutes
      agent: agent-name
      worktree_group: group-name
      status: completed
      description: Task description.
```

#### Format Specification

**Plan Object:**
```yaml
plan:
  name: string           # Required: Plan title
  description: string    # Optional: Plan description
  tasks: []             # Required: List of tasks
```

**Task Object:**
```yaml
- id: int                    # Required: Task number
  name: string               # Required: Task name
  files: [string]            # Optional: List of files
  depends_on: [string]       # Optional: Task dependencies
  estimated_time: string     # Optional: Estimated duration
  agent: string              # Optional: Agent name
  worktree_group: string     # Optional: Worktree group name
  status: string             # Optional: completed|failed|in-progress
  description: string        # Optional: Task description
```

### Task Metadata

#### File(s) / files

**Purpose**: Specify files modified by task

**Format:**
- Markdown: `**File(s)**: file1.go, file2.go`
- YAML: `files: [file1.go, file2.go]`

**Rules:**
- Multiple files separated by commas (Markdown) or array (YAML)
- Relative or absolute paths allowed
- Optional but recommended
- Used for file overlap detection

#### Depends on / depends_on

**Purpose**: Specify task dependencies

**Format:**
- Markdown: `**Depends on**: Task 1, Task 2`
- YAML: `depends_on: [Task 1, Task 2]`

**Rules:**
- Reference tasks by "Task N" format
- Multiple dependencies separated by commas or array
- Dependencies must exist in plan
- Cannot create circular dependencies
- "None" or empty list for no dependencies

#### Estimated time / estimated_time

**Purpose**: Indicate expected task duration

**Format:**
- Markdown: `**Estimated time**: 10 minutes`
- YAML: `estimated_time: 10 minutes`

**Rules:**
- Freeform text field
- No validation or parsing
- Used for planning only
- Optional but recommended

#### Agent / agent

**Purpose**: Specify Claude agent for task execution

**Format:**
- Markdown: `**Agent**: agent-name`
- YAML: `agent: agent-name`

**Rules:**
- Agent must exist in `~/.claude/agents/`
- Optional (uses default if not specified)
- Agent discovery scans numbered directories and root

#### Status / status

**Purpose**: Track task completion status for resumable execution

**Format:**
- Markdown: `**Status**: completed` or `[x]` checkbox
- YAML: `status: completed`

**Valid Values:**
- `completed` - Task successfully completed
- `failed` - Task failed in previous execution
- `in-progress` - Task is currently running
- (empty) - Task not yet executed

**Rules:**
- Optional field (defaults to empty/pending)
- Set manually or automatically by conductor after execution
- Used with `--skip-completed` flag to resume plans
- Skipped tasks create synthetic GREEN results

**Examples:**

Markdown with explicit status:
```markdown
## Task 1: Already Done
**Status**: completed

This task was already completed and will be skipped.
```

Markdown with checkbox (shorthand):
```markdown
## Task 1: Already Done
- [x] This task is marked as completed
```

YAML format:
```yaml
- id: 1
  name: Already Done
  status: completed
  description: This task was already completed.
```

**Resume Examples:**
```bash
# First run: marks completed tasks in plan file
conductor run plan.md

# Resume later: skip completed tasks
conductor run plan.md --skip-completed

# Retry failed tasks on resume
conductor run plan.md --skip-completed --retry-failed
```

#### WorktreeGroup / worktree_group

**Purpose**: Assign task to worktree group for organizational purposes

**Format:**
- Markdown: `**WorktreeGroup**: backend-core`
- YAML: `worktree_group: backend-core`

**Rules:**
- Optional field (defaults to empty if not specified)
- Used for task organization and grouping in multi-file plans
- Group names should use hyphens for multi-word names (no spaces)
- Informational metadata - not enforced by conductor execution engine
- Groups can be defined in plan configuration for validation

**Examples:**

Markdown:
```markdown
## Task 2: API Implementation
**File(s)**: api/routes.go
**Depends on**: Task 1
**WorktreeGroup**: backend-core

Implement REST API endpoints.
```

YAML:
```yaml
- id: 2
  name: API Implementation
  files: [api/routes.go]
  depends_on: [1]
  worktree_group: backend-core
  description: Implement REST API endpoints.
```

### Dependencies

#### Dependency Syntax

Dependencies reference other tasks by their task number:

**Correct:**
```markdown
**Depends on**: Task 1
**Depends on**: Task 1, Task 2
```

```yaml
depends_on: [Task 1]
depends_on: [Task 1, Task 2]
```

#### Dependency Rules

1. **Must Exist**: All referenced tasks must exist in plan
2. **No Cycles**: Cannot create circular dependencies
3. **Forward References**: Can depend on tasks defined later in plan
4. **Wave Grouping**: Dependencies determine execution waves

#### Wave Examples

**Simple Dependency:**
```markdown
## Task 1: Setup
**Depends on**: None

## Task 2: Implementation
**Depends on**: Task 1
```
Execution: Task 1 (Wave 1) → Task 2 (Wave 2)

**Parallel Dependencies:**
```markdown
## Task 1: Setup
**Depends on**: None

## Task 2: Database
**Depends on**: Task 1

## Task 3: API
**Depends on**: Task 1

## Task 4: Tests
**Depends on**: Task 2, Task 3
```
Execution:
- Wave 1: Task 1
- Wave 2: Task 2, Task 3 (parallel)
- Wave 3: Task 4

### Validation Rules

Conductor validates plans before execution:

#### 1. Format Validation

**Markdown:**
- At least one H1 heading (plan title)
- Tasks must use H2 headings with "Task N:" prefix
- Task numbers must be sequential integers
- Metadata must follow task heading

**YAML:**
- Valid YAML syntax
- Required fields: `plan.name`, `plan.tasks`
- Each task must have `id` and `name`
- Task IDs must be unique integers

#### 2. Dependency Validation

- All dependencies must reference existing tasks
- No circular dependencies (checked via DFS)
- Dependencies use "Task N" format

**Valid:**
```markdown
## Task 1: Setup
**Depends on**: None

## Task 2: Implementation
**Depends on**: Task 1
```

**Invalid - Circular:**
```markdown
## Task 1: A
**Depends on**: Task 2

## Task 2: B
**Depends on**: Task 1
```
Error: Circular dependency detected

#### 3. File Validation

- File paths should not overlap across tasks
- Same file modified by multiple tasks may cause conflicts

**Warning Example:**
```markdown
## Task 1: Setup
**File(s)**: main.go

## Task 2: Implementation
**File(s)**: main.go
```
Warning: File main.go modified by multiple tasks

#### 4. Agent Validation

- Agent must exist in `~/.claude/agents/`
- Agent discovery checks numbered directories (01-10) and root

**Valid:**
```markdown
**Agent**: code-implementation
```
(Assumes `~/.claude/agents/code-implementation.md` exists)

---

## Usage & Commands

### `conductor validate`

Validates plan file syntax, dependencies, and detects cycles.

**Usage:**
```bash
conductor validate <plan-file> [<plan-file>...]
```

**Arguments:**
- `<plan-file>` - Path to plan file (.md or .yaml)
- Multiple files can be validated together

**Examples:**
```bash
# Validate Markdown plan
conductor validate implementation-plan.md

# Validate YAML plan
conductor validate plan.yaml

# Validate multiple files together
conductor validate setup.md features.md deployment.md

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
conductor run [flags] <plan-file> [<plan-file>...]
```

**Arguments:**
- `<plan-file>` - Path to plan file (.md or .yaml)
- Multiple files can be executed together

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | false | Simulate execution without running tasks |
| `--max-concurrency` | int | 3 | Maximum parallel tasks per wave |
| `--timeout` | duration | 30m | Timeout for entire execution |
| `--verbose` | bool | false | Enable detailed logging |
| `--skip-completed` | bool | false | Skip tasks marked as completed |
| `--retry-failed` | bool | false | Retry tasks marked as failed |
| `--log-dir` | string | .conductor/logs | Directory for execution logs |

**Examples:**

```bash
# Basic execution
conductor run plan.md

# Execute multiple files together
conductor run setup.md features.md deployment.md

# Dry run (no actual execution)
conductor run plan.md --dry-run

# Custom concurrency limit
conductor run plan.md --max-concurrency 5

# With timeout
conductor run plan.md --timeout 1h

# Verbose output
conductor run plan.md --verbose

# Run only a specific task
conductor run plan.md --task 3

# Resume execution (skip completed tasks)
conductor run plan.md --skip-completed

# Resume and retry failed tasks
conductor run plan.md --skip-completed --retry-failed

# Custom log directory
conductor run plan.md --log-dir ./execution-logs

# Combined flags
conductor run plan.md \
  --max-concurrency 5 \
  --timeout 1h \
  --verbose \
  --log-dir ./logs
```

### Plan Execution Modes

#### Skip Completed Tasks

Conductor can resume interrupted plans by skipping already-completed tasks:

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

#### Run Single Task

Execute only a specific task from the plan:

```bash
# Run only task 3
conductor run plan.md --task 3

# Combine with dry-run to preview
conductor run plan.md --task 3 --dry-run
```

**Behavior:**
- Only the specified task executes, all others are skipped
- Dependencies are NOT auto-included (task runs in isolation)
- Useful for debugging or re-running a specific task
- Works with both execution and dry-run modes

#### Retry Failed Tasks

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

### Learning Commands

Conductor provides commands for observing and managing learning data.

#### `conductor learning stats`

Display learning statistics for a plan file.

**Usage:**
```bash
conductor learning stats <plan-file>
```

**Example:**
```bash
$ conductor learning stats my-plan.md

Learning Statistics for: my-plan.md
=====================================
Total executions: 15
Success rate: 73.3% (11/15)
Failed attempts: 4

Agent Performance:
  golang-pro:         8 executions, 87.5% success
  backend-developer:  4 executions, 50.0% success
  fullstack-dev:      3 executions, 66.7% success

Common Failure Patterns:
  - compilation_error: 3 occurrences
  - test_failure: 2 occurrences

Recent Executions (last 5):
  1. Task 3 - golang-pro - GREEN - 2025-01-12 14:30
  2. Task 2 - backend-developer - RED - 2025-01-12 14:25
  3. Task 1 - golang-pro - GREEN - 2025-01-12 14:20
```

#### `conductor learning show`

Show detailed execution history for a specific task.

**Usage:**
```bash
conductor learning show <plan-file> <task-number>
```

**Example:**
```bash
$ conductor learning show my-plan.md "Task 3"

Execution History: Task 3
==========================

Execution #1 - 2025-01-12 14:30:45
  Session: session-20250112-143045
  Run: 2
  Agent: golang-pro
  Verdict: GREEN
  Duration: 45s
  Patterns: none

Execution #2 - 2025-01-12 13:15:22
  Session: session-20250112-131522
  Run: 1
  Agent: backend-developer
  Verdict: RED
  Duration: 62s
  Patterns: compilation_error, test_failure
  Output: "compilation failed: syntax error at line 42"
```

#### `conductor learning clear`

Clear learning data for a plan file or entire database.

**Usage:**
```bash
# Clear specific plan
conductor learning clear <plan-file>

# Clear entire database
conductor learning clear --all
```

**Example:**
```bash
$ conductor learning clear my-plan.md
Warning: This will delete all execution history for: my-plan.md
Are you sure? (y/n): y
✓ Deleted 15 execution records for my-plan.md
```

#### `conductor learning export`

Export learning data to JSON or CSV format.

**Usage:**
```bash
conductor learning export <plan-file> [--format json|csv] [--output file]
```

**Example:**
```bash
# Export to JSON (stdout)
$ conductor learning export my-plan.md --format json
[
  {
    "task_number": "Task 1",
    "agent": "golang-pro",
    "success": true,
    "timestamp": "2025-01-12T14:30:45Z",
    "duration_seconds": 45
  }
]

# Export to CSV file
$ conductor learning export my-plan.md --format csv --output data.csv
✓ Exported 15 records to data.csv
```

### Observe Commands (Agent Watch)

Conductor provides behavioral observability for Claude Code agents via the `observe` command family. These commands analyze session JSONL files in `~/.claude/projects/`.

#### `conductor observe`

Interactive mode with project selection.

```bash
conductor observe
```

#### `conductor observe stats`

Display summary statistics for agent sessions.

```bash
conductor observe stats [--project <name>] [--time-range 24h]
```

#### `conductor observe session`

Session-specific analysis.

```bash
conductor observe session <session-id>
```

#### `conductor observe live`

Real-time watching of new JSONL events. Shows only new content after startup.

```bash
conductor observe live [--project <name>]
```

**Features:**
- Skips historical data on startup
- Shows "New Agent Started" banner when subagents spawn
- Displays Edit tool diffs with full old/new strings
- Color-coded output (assistant, user, tools, errors)

#### `conductor observe transcript`

Formatted transcript of session events.

```bash
conductor observe transcript <session-id> [--no-color] [--compact]
```

#### `conductor observe tools`

Analyzes tool invocation patterns from Claude Code agent sessions.

```bash
conductor observe tools [flags]
```

**Features:**
- Shows tool usage frequency
- Calculates success/failure rates per tool
- Displays average execution duration
- Ranks tools by usage count

**Flags:**
```
--limit, -n         Number of tools to display (default: 20, 0 for all)
--project, -p       Filter by project name pattern
```

**Examples:**
```bash
# Show top 20 most-used tools
conductor observe tools

# Show all tools
conductor observe tools --limit 0

# Filter by project
conductor observe tools --project myapp

# Show 50 tools for a specific project
conductor observe tools --project backend --limit 50
```

**Output Example:**
```
=== Tool Usage Analysis ===

Tool                               Calls      Success      Failures    Avg Duration   Success%
------------------------------- --------- ------------ ---------- --------------- ----------
Read                              15234      15207             27             45ms      99.8%
Edit                               8921       8666            255            120ms      97.2%
Bash                               7456       6678            778            850ms      89.5%
Write                              5432       5401             31            200ms      99.4%

(Showing 4 of 250 tools. Use --limit flag to see more)
```

#### `conductor observe bash`

Analyzes bash command execution patterns.

```bash
conductor observe bash [flags]
```

**Features:**
- Shows command execution frequency
- Tracks exit codes and success rates
- Lists most common and failed commands
- Displays average command duration

**Flags:**
```
--limit, -n         Number of commands to display (default: 20, 0 for all)
--project, -p       Filter by project name pattern
```

**Examples:**
```bash
# Show top 20 bash commands
conductor observe bash

# Show commands for specific project
conductor observe bash --project ci-pipeline

# Show all commands with statistics
conductor observe bash --limit 0
```

**Output Example:**
```
=== Bash Command Analysis ===

Command                                      Calls      Success      Failures    Avg Duration   Success%
-------------------------------------- --------- ------------ ---------- --------------- ----------
git status                              450        440             10           500ms        97.8%
go test ./...                           320        290             30          2500ms        90.6%
make build                              280        270             10          1500ms        96.4%
npm run test                            200        195              5          1200ms        97.5%

(Showing 4 of 180 commands. Use --limit flag to see more)
```

#### `conductor observe files`

Analyzes file operation patterns (read, write, edit).

```bash
conductor observe files [flags]
```

**Features:**
- Shows file access patterns
- Tracks operation types (read/write/edit)
- Displays success/failure rates
- Shows total bytes affected per file

**Flags:**
```
--limit, -n         Number of files to display (default: 20, 0 for all)
--project, -p       Filter by project name pattern
```

**Examples:**
```bash
# Show top 20 accessed files
conductor observe files

# Show all file operations
conductor observe files --limit 0

# Filter by project
conductor observe files --project myapp
```

**Output Example:**
```
=== File Operation Analysis ===

File                              Op Type       Calls      Success      Failures    Avg Duration    Total Bytes
-------------------------------------- --------- ------------ ---------- --------------- --------
/src/main.go                      read          200        200              0             30ms           100000
/src/main.go                      write          50         48              2             80ms            25000
/config/config.yaml               read           30         30              0             15ms             5000
/src/handler.go                   read          180        180              0             35ms            75000
/src/handler.go                   edit           40         40              0             90ms            12000

(Showing 5 of 320 file operations. Use --limit flag to see more)
```

#### `conductor observe errors`

Analyzes error patterns from all operation types.

```bash
conductor observe errors [flags]
```

**Features:**
- Aggregates errors from tools, bash commands, and file operations
- Shows error frequency and patterns
- Tracks error type breakdown
- Displays last occurrence timestamp

**Flags:**
```
--limit, -n         Number of errors to display (default: 20, 0 for all)
--project, -p       Filter by project name pattern
```

**Examples:**
```bash
# Show top 20 error patterns
conductor observe errors

# Show all errors
conductor observe errors --limit 0

# Show errors for specific project
conductor observe errors --project infrastructure
```

**Output Example:**
```
=== Error Pattern Analysis ===

Type            Component                                    Error Message                               Count       Last Occurred
------- ---------------------------------- -------------------------------------------------- ------- --------------------
tool            Read                         permission denied                                    5       2025-11-25 15:30:22
bash            go test ./...                exit code 1                                          8       2025-11-25 15:22:10
file            read: /missing/file.txt      file not found                                       3       2025-11-25 15:15:45
tool            Bash                         connection timeout                                   2       2025-11-25 15:10:33

--- Error Summary ---
Total Errors: 18
Tool Errors: 7 (38.9%)
Bash Errors: 8 (44.4%)
File Errors: 3 (16.7%)
```

#### `conductor observe project`

Project-level metrics.

```bash
conductor observe project <project-name>
```

#### `conductor observe export`

Export data to JSON, Markdown, or CSV.

```bash
conductor observe export --format json --output metrics.json
conductor observe export --format markdown --output report.md
conductor observe export --format csv --output data.csv
```

#### `conductor observe ingest`

Real-time JSONL ingestion daemon.

```bash
conductor observe ingest                      # One-time import
conductor observe ingest --watch              # Run as daemon
conductor observe ingest --watch --verbose    # Verbose daemon mode
```

### Budget Commands

Commands for managing rate limit state and resuming paused executions.

#### `conductor budget list-paused`

Lists all paused execution states waiting for rate limit reset.

**Usage:**
```bash
conductor budget list-paused
```

**Example Output:**
```
Paused Executions:
  Session: abc123def456
  Plan: implementation-plan.md
  Status: paused
  Paused At: 2025-01-15 14:30:00
  Resume At: 2025-01-15 19:30:00 (in 5h 0m)
  Completed: 8/15 tasks
  Current Wave: 3
```

#### `conductor budget resume`

Resumes paused executions that are ready (rate limit window expired).

**Usage:**
```bash
# Resume all ready executions
conductor budget resume

# Resume specific session by ID
conductor budget resume abc123def456
```

**Behavior:**
- Checks if `ResumeAt` time has passed
- Loads saved execution state
- Continues from saved wave with completed tasks skipped
- Cleans up state file after successful completion

### Other Commands

#### `conductor --version`

Displays version information.

**Usage:**
```bash
conductor --version
```

#### `conductor --help`

Shows help information for all commands.

**Usage:**
```bash
conductor --help
conductor validate --help
conductor run --help
conductor learning --help
```

---

## Configuration

### Configuration File

Conductor supports optional configuration via `.conductor/config.yaml`.

**Create configuration:**
```bash
mkdir -p .conductor
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

# Console output configuration
console:
  # Color output control
  # - auto: Auto-detect based on TTY (default)
  # - true: Force colors on
  # - false: Force colors off
  colors_enabled: auto

# Quality control settings
quality_control:
  enabled: true              # Enable QC reviews (default: true)
  # review_agent: conductor-qc # DEPRECATED: use agents section below
  retry_on_red: 2            # Retry attempts on RED verdict (default: 2)

  # Multi-agent QC configuration (v2.2+)
  agents:
    # Selection mode: "auto", "explicit", "mixed", or "intelligent"
    # - auto: Auto-select based on file extensions (.go→golang-pro)
    # - explicit: Use only agents from explicit_list
    # - mixed: Auto-select + additional agents
    # - intelligent: Claude-based selection analyzing task context (v2.4+)
    mode: auto

    # explicit_list: [security-auditor, code-reviewer]  # For mode=explicit
    # additional: [security-auditor]                     # For mode=mixed
    # blocked: [deprecated-agent]                        # Agents to never use

    # Intelligent selection settings (v2.4+)
    max_agents: 4                    # Max QC agents to select (default: 4)
    cache_ttl_seconds: 3600          # Cache TTL in seconds (default: 3600)
    selection_timeout_seconds: 90    # Claude selection timeout (default: 90)
    require_code_review: true        # Always include code-reviewer baseline (default: true)

# Intelligent QC Agent Selection Example (v2.4+)
# For domain-aware, context-sensitive agent selection:
#
# quality_control:
#   enabled: true
#   retry_on_red: 2
#   agents:
#     mode: intelligent
#     max_agents: 3
#     cache_ttl_seconds: 1800
#     require_code_review: true
#     blocked: []

# Feedback storage settings
feedback:
  store_in_plan_file: true   # Store execution history in plan file (default: true)
  store_in_database: true    # Store execution history in database (default: true)
  format: json               # Response format: json or text (default: json)
  store_on_green: true       # Store successful executions (default: true)
  store_on_red: true         # Store failed executions (default: true)
  store_on_yellow: true      # Store warning executions (default: true)

# Executor settings (v2.11+)
executor:
  # Enable Claude-based error classification (default: false)
  enable_claude_classification: false

  # Enable intelligent task agent selection (v2.15+, default: false)
  # When task.Agent is empty, uses Claude to select the best agent
  intelligent_agent_selection: true

# Logging settings
log_dir: .conductor/logs  # Log directory (default: .conductor/logs)
log_level: info           # Log level: debug, info, warn, error (default: info)

# Learning settings
learning:
  # Enable or disable the learning system (default: true)
  enabled: true

  # Database path (default: .conductor/learning/executions.db)
  db_path: .conductor/learning/executions.db

  # Enable agent swapping during retry loops (default: true)
  # When a task fails, uses IntelligentAgentSwapper to select a better agent
  swap_during_retries: true

  # Enhance prompts with learned context (default: true)
  enhance_prompts: true

  # Enable QC to read plan file context (default: true)
  # QC agent loads execution history from current run before reviewing
  qc_reads_plan_context: true

  # Enable QC to read database context (default: true)
  # QC agent loads historical execution data before reviewing
  qc_reads_db_context: true

  # Maximum context entries to load (default: 10)
  # Limits the number of historical attempts included in QC context
  max_context_entries: 10

  # Days to keep execution history (default: 90, 0 = forever)
  keep_executions_days: 90

  # Maximum execution records per task (default: 100)
  max_executions_per_task: 100

# Text-to-Speech settings (v2.14+)
# Optional voice feedback via local Orpheus TTS server
tts:
  # Enable/disable TTS (default: false)
  # When disabled, no TTS API calls are made
  enabled: false

  # Orpheus TTS server URL
  base_url: "http://localhost:5005"

  # TTS model name
  model: "orpheus"

  # Voice selection: tara, leah, jess, leo, dan, mia, zac, zoe
  voice: "tara"

  # Request timeout (default: 30s)
  # Orpheus typically takes 2-3s per phrase
  timeout: 30s
```

### Configuration Priority

Configuration is loaded in this order (later overrides earlier):

1. Default values (hardcoded in binary)
2. `.conductor/config.yaml` (if exists)
3. Command-line flags (highest priority)

### Execution Flow

1. **Load Plan**: Parse plan file (auto-detect format)
2. **Validate**: Check dependencies, detect cycles
3. **Calculate Waves**: Group parallel-executable tasks
4. **Execute Waves**: Run tasks in waves (parallel within, sequential between)
5. **Quality Control**: Review each task output
6. **Retry Logic**: Retry RED verdicts up to max_retries
7. **Update Plan**: Mark completed tasks
8. **Log Results**: Write execution logs

### Runtime enforcement pipeline

Conductor v2.9+ enforces runtime checks during task execution. All enforcement modes default to **enabled**. The pipeline runs in this order:

```
┌─────────────────────────────────────────────────────────────────┐
│                    TASK EXECUTION PIPELINE                      │
├─────────────────────────────────────────────────────────────────┤
│  1. PREFLIGHT PHASE                                             │
│     └─ Dependency Checks (runtime_metadata.dependency_checks)   │
│        └─ On failure: BLOCK task, return error immediately      │
│                                                                 │
│  2. AGENT INVOCATION                                            │
│     └─ Claude agent executes the task                           │
│                                                                 │
│  3. POST-AGENT PHASE                                            │
│     ├─ Test Commands (test_commands array)                      │
│     │  └─ On failure: BLOCK task, skip QC                       │
│     ├─ Criterion Verification (success_criteria[].verification) │
│     │  └─ On failure: Pass results to QC for judgment           │
│     └─ Documentation Targets (runtime_metadata.doc_targets)     │
│        └─ On failure: Pass results to QC for judgment           │
│                                                                 │
│  4. QUALITY CONTROL                                             │
│     └─ QC reviews output + verification results                 │
│                                                                 │
│  5. PACKAGE GUARD (concurrent tasks only)                       │
│     └─ Prevents multiple tasks from modifying same Go package   │
└─────────────────────────────────────────────────────────────────┘
```

#### Enforcement Flags

| Flag | Config Key | Default | Description |
|------|------------|---------|-------------|
| `--no-enforce-dependency-checks` | `enforce_dependency_checks` | `true` | Run preflight commands before agent |
| `--no-enforce-test-commands` | `enforce_test_commands` | `true` | Run test commands after agent (blocks on failure) |
| `--no-enforce-package-guard` | `enforce_package_guard` | `true` | Prevent concurrent Go package modifications |
| `--no-enforce-doc-targets` | `enforce_doc_targets` | `true` | Verify documentation targets before QC |
| `--no-verify-criteria` | `verify_criteria` | `true` | Run criterion verification commands |

#### Test Failures vs Verification Failures

**Critical distinction:**

- **Test command failure** = Immediate task block, QC skipped entirely
- **Verification failure** = Results fed into QC prompt for human/agent judgment

Test commands are hard gates. Verification commands are soft signals.

#### Example Configuration

```yaml
# .conductor/config.yaml
enforce_dependency_checks: true   # Preflight checks
enforce_test_commands: true       # Post-agent tests (hard gate)
enforce_package_guard: true       # Go package isolation
enforce_doc_targets: true         # Doc target verification
verify_criteria: true             # Criterion verification (soft signal)
```

#### Plan Metadata for Enforcement

```yaml
tasks:
  - task_number: 1
    name: "Implement feature"
    files:
      - "internal/feature/impl.go"

    # Preflight checks (run before agent)
    runtime_metadata:
      dependency_checks:
        - command: "go build ./..."
          description: "Verify build succeeds"
        - command: "test -f go.mod"
          description: "Verify Go module exists"

      documentation_targets:
        - location: "docs/feature.md"
          section: "# Configuration"

    # Post-agent tests (hard gate)
    test_commands:
      - "go test ./internal/feature/..."

    # Criterion verification (soft signal)
    success_criteria:
      - criterion: "Function handles edge cases"
        verification:
          command: "go test -run TestEdgeCases"
          expected: "PASS"
```

See [Runtime Enforcement Examples](examples/runtime-enforcement.md) for detailed walkthroughs.

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

#### Intelligent QC Agent Selection (v2.4+)

Beyond file-extension based selection, Conductor v2.4 introduces intelligent agent selection using Claude to analyze task context and recommend the most appropriate QC agents.

**Selection Modes:**

| Mode | Description | Use Case |
|------|-------------|----------|
| `auto` (default) | File-extension mapping (.go→golang-pro, .py→python-pro) | Standard projects with clear file types |
| `explicit` | Use only agents from `explicit_list` | When you know exactly which agents to use |
| `mixed` | Auto-select + additional agents | Combine auto-detection with specific agents |
| `intelligent` | Claude-based selection with task + executing agent context | Domain-aware, context-sensitive selection |

**Intelligent Selection Configuration:**

```yaml
quality_control:
  enabled: true
  retry_on_red: 2
  agents:
    mode: intelligent
    max_agents: 3
    cache_ttl_seconds: 1800
    require_code_review: true
    blocked: []
```

**Benefits of Intelligent Selection:**

1. **Domain-Aware**: Automatically recommends `security-auditor` for authentication tasks, `database-optimizer` for database operations, `performance-analyst` for optimization work
2. **Stack-Aware**: Considers the executing agent's strengths and potential blind spots when selecting QC agents
3. **Cost-Controlled**: Caches selection results to minimize redundant API calls during retries
4. **Fallback Safety**: Automatically reverts to `auto` mode if intelligent selection fails

**Guardrails and Safeguards:**

- **MaxAgents Cap**: Limits selection to configured maximum (default: 4 agents)
- **Code-Reviewer Baseline**: Always includes `code-reviewer` agent when `require_code_review: true`
- **Registry Validation**: All recommended agents are validated against the registry allowlist
- **Blocked Agents Filtered**: Agents in `blocked` list are never selected
- **Graceful Fallback**: Returns to `auto` mode on API errors or parsing failures

**Intelligent Selection Flow:**

```
1. Build Prompt
   ├─ Include task context (name, files, description)
   ├─ Include executing agent information
   └─ Request agent recommendations

2. Query Claude
   └─ Single API call for agent recommendations

3. Apply Guardrails
   ├─ Cap at MaxAgents
   ├─ Validate against registry
   ├─ Filter blocked agents
   └─ Ensure code-reviewer baseline (if required)

4. Cache Results
   └─ Store for cache_ttl_seconds to avoid redundant calls
```

**Configuration Options:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `mode` | string | `auto` | Selection mode: auto, explicit, mixed, or intelligent |
| `max_agents` | int | `4` | Maximum number of QC agents to select |
| `cache_ttl_seconds` | int | `3600` | Cache TTL in seconds (1 hour default) |
| `selection_timeout_seconds` | int | `90` | Timeout for Claude intelligent selection calls |
| `require_code_review` | bool | `true` | Always include code-reviewer as baseline |
| `blocked` | []string | `[]` | Agents to never use for QC |
| `explicit_list` | []string | `[]` | Agents for explicit mode |
| `additional` | []string | `[]` | Extra agents for mixed mode |

**Example Scenarios:**

*Authentication Task:*
```yaml
# Task: Implement JWT authentication
# Intelligent selection recommends:
# - code-reviewer (baseline)
# - security-auditor (auth-specific)
# - golang-pro (if .go files)
```

*Database Migration:*
```yaml
# Task: Add database indexes for performance
# Intelligent selection recommends:
# - code-reviewer (baseline)
# - database-optimizer (DB-specific)
# - performance-analyst (optimization focus)
```

*API Endpoint:*
```yaml
# Task: Create REST API endpoint
# Intelligent selection recommends:
# - code-reviewer (baseline)
# - api-designer (REST patterns)
# - golang-pro (if .go files)
```

#### Structured QC Response Format (v2.1.0)

Quality control agents return structured JSON responses for richer analysis:

**QC Response Schema:**
```json
{
  "verdict": "RED",
  "feedback": "Implementation is incomplete. Error handling not implemented.",
  "issues": [
    {
      "severity": "critical",
      "description": "Missing error handling for database connection failures",
      "location": "internal/db/client.go:45"
    },
    {
      "severity": "warning",
      "description": "Unused import detected",
      "location": "internal/db/client.go:3"
    }
  ],
  "recommendations": [
    "Add try-catch for database connection",
    "Implement retry logic with exponential backoff",
    "Remove unused imports"
  ],
  "should_retry": true,
  "suggested_agent": "golang-pro"
}
```

**Schema Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `verdict` | string | QC decision: GREEN, RED, or YELLOW |
| `feedback` | string | Human-readable summary of the review |
| `issues` | array | List of specific issues found |
| `issues[].severity` | string | Issue severity: critical, warning, or info |
| `issues[].description` | string | Detailed issue description |
| `issues[].location` | string | File:line or component where issue found |
| `recommendations` | array | Suggested improvements or fixes |
| `should_retry` | boolean | Whether the task should be retried |
| `suggested_agent` | string | Alternative agent for retry (optional) |

**Benefits of Structured Responses:**
- Actionable insights with specific locations
- Automatic agent adaptation based on suggestions
- Detailed issue tracking for pattern analysis
- Clear retry decisions with reasoning
- Enhanced debugging with structured feedback

### Integration Tasks (v2.5.0)

Integration tasks enable cross-component validation when implementations depend on multiple other tasks. Use this feature for tasks that integrate functionality from different modules or services.

#### What are Integration Tasks?

Integration tasks are regular tasks that explicitly define cross-component validation criteria alongside component-level success criteria.

**Purpose:**
- Validate that an implementation correctly integrates with other components
- Ensure cross-module interfaces work properly
- Verify data flow between services
- Check that dependent modules are utilized correctly

**When to Use Integration Tasks:**
- Task depends on 2+ other tasks
- Task requires validation of interactions between components
- Task tests interfaces or integration points
- Need both component-level AND cross-component validation

**Example Use Cases:**
- Task 4: Create API endpoints that use auth service from Task 2 and database from Task 1
- Task 6: Implement webhook that integrates with message queue and external service
- Task 8: Build admin dashboard that pulls data from multiple APIs

#### Task Type Field

The `type` field categorizes tasks for execution behavior:

**Valid Values:**
- `"component"` - Standalone implementation (no cross-component dependencies)
- `"integration"` - Requires cross-component validation
- `"regular"` - Default type (backward compatible)

**Setting Task Type:**

In YAML format:
```yaml
- id: 4
  name: API Endpoints
  type: integration              # Marks as integration task
  depends_on: [Task 1, Task 2]
  success_criteria:
    - "All endpoints implemented"
  integration_criteria:
    - "Uses auth service correctly"
    - "Queries database properly"
```

In Markdown format:
```markdown
## Task 4: API Endpoints
**Type**: integration
**Depends on**: Task 1, Task 2
**File(s)**: internal/api/endpoints.go

Implement REST endpoints.
```

**Type Field Rules:**
- Optional field (defaults to "regular")
- Must be: "component", "integration", or "regular"
- Affects QC behavior and agent selection
- Validated at parse time

#### Dual Criteria System

Integration tasks support two independent criteria types:

**1. success_criteria** - Component-level validation
- Validates the task implementation itself
- No dependencies on other components
- Example: "All error cases handled", "Code follows best practices"

**2. integration_criteria** - Cross-component validation
- Validates interaction with dependent components
- Checks interfaces, data flow, contracts
- Example: "Correctly uses auth service methods", "Data matches expected format"

**How QC Processes Both:**

```
QC Agent Receives:
  ├─ Component-Level Criteria (success_criteria)
  │  └─ Verify implementation quality
  ├─ Integration-Level Criteria (integration_criteria)
  │  └─ Verify cross-component interaction
  └─ Dependency Context
     └─ Code from dependent tasks

QC Validation:
  ├─ Step 1: Verify each success criterion
  ├─ Step 2: Verify each integration criterion
  └─ Step 3: Return unified GREEN/RED/YELLOW verdict
```

**Criterion Numbering:**

Criteria are numbered continuously across both types:

```yaml
success_criteria:
  - "Criterion 0: Auth endpoints implemented"    # 0
  - "Criterion 1: Error handling complete"        # 1

integration_criteria:
  - "Criterion 2: Uses JWT module correctly"      # 2
  - "Criterion 3: Stores tokens in database"      # 3
```

QC prompt shows: "0. Auth endpoints implemented ... 3. Stores tokens in database"

#### Integration Prompt Enhancement

When a task is marked as integration or has dependencies, the agent prompt is automatically enhanced with dependency context.

**Automatic Context Injection:**

Before agent execution, conductor builds integration context including:
- List of all dependent tasks
- Key files modified by each dependency
- Explanation of integration points

**Example Original Prompt:**
```
Implement REST API endpoints for user management.
```

**Enhanced with Integration Context:**
```
# INTEGRATION TASK CONTEXT

Before implementing, you MUST read and understand these dependent tasks:

## Task 1: Database Schema
- Key files: internal/db/schema.sql, internal/db/models.go
This task created the database schema and models.

## Task 2: Authentication Service
- Key files: internal/auth/jwt.go, internal/auth/tokens.go
This task implemented JWT authentication.

---

Implement REST API endpoints for user management.
```

**When Enhancement Applies:**
- Task type is "integration" AND/OR
- Task has entries in `depends_on` field
- Task is part of multi-task plan

**What Gets Included:**
1. Each dependency task's name and number
2. Files modified by dependent task
3. Brief context about that task
4. Original task prompt unchanged

#### YAML Examples

**Component Task Example:**

```yaml
- id: 1
  name: Database Schema
  type: component
  files: [internal/db/schema.sql]
  depends_on: []
  description: Create database schema and models.
  success_criteria:
    - "All tables defined with proper constraints"
    - "Foreign keys reference correct tables"
```

**Integration Task Example:**

```yaml
- id: 4
  name: REST API Endpoints
  type: integration
  files: [internal/api/endpoints.go]
  depends_on: [Task 1, Task 2, Task 3]
  description: |
    Implement REST endpoints that integrate with:
    - Database (Task 1)
    - Authentication (Task 2)
    - Validation (Task 3)
  success_criteria:
    - "All endpoints implemented with proper HTTP methods"
    - "Request/response validation complete"
    - "Error handling for all failure cases"
  integration_criteria:
    - "Uses JWT validation from auth service"
    - "Queries correctly use database models"
    - "Input validation uses validator service"
    - "Endpoint returns expected data format"
```

**Multi-File Plan with Integration:**

File 1: `database.md`
```markdown
## Task 1: Database Schema
**Type**: component
**Files**: schema.sql

Create PostgreSQL schema.
```

File 2: `backend.md`
```markdown
## Task 2: API Implementation
**Type**: integration
**Depends on**: Task 1
**Files**: routes.go

Implement API endpoints.

**Success Criteria**:
- All endpoints implemented

**Integration Criteria**:
- Correctly queries database
```

Execute together:
```bash
conductor run database.md backend.md
```

#### Best Practices

**Task Type Selection:**

| Scenario | Type | Reason |
|----------|------|--------|
| Initial setup (DB, config) | component | No dependencies on other tasks |
| Single-module feature | component | Only affects one module |
| Endpoints using multiple services | integration | Depends on auth + database |
| Service that integrates external API | integration | Needs cross-system validation |
| Tests for integration | integration | Validates component interactions |

**Writing Good Criteria:**

**Component Criteria (success_criteria):**
- ✅ "All error cases handled with proper error messages"
- ✅ "Unit tests cover 90%+ of code"
- ✅ "Code follows Go naming conventions"
- ❌ "Works with authentication" (this is integration)

**Integration Criteria (integration_criteria):**
- ✅ "Correctly calls authentication service methods"
- ✅ "Returns data in format expected by frontend"
- ✅ "Respects rate limits from queue service"
- ❌ "Code is well-documented" (this is component-level)

**Dependency Structure:**

**Good Pattern - Layered Dependencies:**
```
Layer 1 (No deps):   [Task 1: Database]
                         ↓
Layer 2 (Depends 1): [Task 2: Auth] [Task 3: Cache]
                         ↓
Layer 3 (Depends 1-2): [Task 4: API] (type: integration)
```

**Avoid - Complex Cross-Layer Dependencies:**
```
Task 1 → Task 3
↓        ↑
Task 2 ← (tangled, hard to validate)
```

**Markdown Format with Criteria:**

```markdown
## Task 4: REST API
**Type**: integration
**Depends on**: Task 1, Task 2
**Files**: api.go

Implement REST endpoints.

### Success Criteria
- All CRUD endpoints implemented
- Comprehensive error handling
- Input validation on all endpoints

### Integration Criteria
- Uses database models from Task 1
- Calls auth verification from Task 2
- Returns proper HTTP status codes
```

#### QC Behavior with Integration Tasks

**Multi-Agent Consensus:**

When task has integration criteria:
1. Multiple QC agents review the task (if intelligent selection enabled)
2. Each agent verifies all criteria (both success and integration)
3. Agent must agree on verdict for each criterion
4. Unanimous consensus required: all agents must pass criterion for GREEN

**Per-Criterion Verification:**

QC response includes results for each criterion:
```json
{
  "verdict": "RED",
  "criteria_results": [
    {
      "index": 0,
      "passed": true,
      "evidence": "All endpoints implemented with correct methods"
    },
    {
      "index": 1,
      "passed": true,
      "evidence": "Comprehensive error handling in place"
    },
    {
      "index": 2,
      "passed": false,
      "evidence": "Uses deprecated database methods instead of models from Task 1"
    }
  ]
}
```

**Verdict Aggregation Rules:**

- **GREEN**: All criteria (success AND integration) verified by all agents
- **RED**: Any criterion failed OR agents disagree on any criterion
- **YELLOW**: Minor issues detected, but fundamental requirements met

**Agent Selection for Integration Tasks:**

With intelligent agent selection enabled, conductor:
1. Detects task is integration type
2. Requests agents specialized in cross-component review
3. May recommend: code-reviewer, architecture-validator, integration-tester
4. These agents understand dependency relationships

#### Configuration

Enable intelligent agent selection for better integration task validation:

```yaml
# .conductor/config.yaml

quality_control:
  enabled: true
  agents:
    mode: intelligent      # Enable smart agent selection
    max_agents: 4
    require_code_review: true

  # Integration-aware configuration
  retry_on_red: 2         # Allow retries for integration issues
```

#### When Integration Tasks Make Sense

**Use integration tasks when:**
- Feature requires combining multiple components
- Cross-module interfaces need validation
- Data flows between systems must be verified
- Multiple agents should review for different aspects

**Use regular tasks when:**
- Task is self-contained within single module
- No cross-module dependencies
- Single agent can verify quality
- Simpler component-level validation sufficient

#### Troubleshooting Integration Tasks

**Integration Criteria Not Validated:**

```bash
# Check that criteria are in plan
$ grep -A5 "integration_criteria" plan.yaml

# Check QC response includes integration criteria verification
$ cat .conductor/logs/qc-*.log | grep "integration"
```

**Task Type Not Recognized:**

```bash
# Validate plan
$ conductor validate plan.md

# Should show task type in validation output
```

**Integration Context Not Appearing:**

```bash
# Check task dependencies
$ grep "depends_on" plan.yaml

# Context only injected if task has dependencies or type=integration
```

### Pattern Intelligence (v2.24+)

Prior art detection and duplicate prevention using the STOP protocol (Search/Think/Outline/Prove).

**What It Does:**

1. **Search**: Searches git history, GitHub issues, docs, and learning database for similar work
2. **Think**: Analyzes task complexity and identifies potential risks
3. **Outline**: Generates recommended approach based on prior art
4. **Prove**: Suggests verification steps based on similar successful tasks

**Configuration:**

```yaml
pattern:
  # Enable/disable Pattern Intelligence (default: false)
  enabled: true

  # Operating mode:
  # - block: Fail tasks with high similarity to existing work
  # - warn: Log warning but allow execution (default)
  # - suggest: Include analysis in prompt without blocking
  mode: warn

  # Similarity threshold to trigger action (0.0-1.0, default: 0.8)
  similarity_threshold: 0.8

  # Duplicate threshold (0.0-1.0, default: 0.9)
  duplicate_threshold: 0.9

  # Minimum confidence before taking action (0.0-1.0, default: 0.7)
  min_confidence: 0.7

  # Enable STOP protocol analysis (default: true when enabled)
  enable_stop: true

  # Enable duplicate task detection (default: true when enabled)
  enable_duplicate_detection: true

  # Include pattern analysis in agent prompts (default: true)
  inject_into_prompt: true

  # Maximum patterns to include in prompt injection (default: 5)
  max_patterns_per_task: 5

  # Maximum related files to include in analysis (default: 10)
  max_related_files: 10

  # Cache TTL in seconds (default: 3600)
  cache_ttl_seconds: 3600

  # Require implementing agent to justify custom implementations (v2.24+)
  # When STOP finds prior art, QC will request justification
  require_justification: true
```

**Prior Art Justification:**

When `require_justification: true` and STOP finds existing solutions:

1. QC agents receive prior art context in their review prompt
2. QC requests justification for why custom implementation was needed
3. Weak/missing justification (< 50 chars, "n/a", "not applicable") → YELLOW verdict
4. Strong technical justification → proceeds normally

**Example justification evaluation:**
- ❌ "n/a" → YELLOW (too weak)
- ❌ "Different use case" → YELLOW (too short)
- ✅ "The existing implementation uses regex which is too slow for our use case. We need O(1) lookup via hash table." → Passes

**Graceful Degradation:**

Pattern Intelligence never blocks execution due to its own errors:
- If search fails → logs warning, continues without pattern context
- If store unavailable → continues with in-memory only
- If disabled → zero behavior change from previous versions

### Budget & Rate Limits (v2.20+, enhanced v2.21+)

Intelligent rate limit auto-resume with state persistence and mid-task recovery.

**Key Features:**
- Parses actual reset time from Claude CLI output (not hardcoded delays)
- **Session resume (v2.21+)**: Uses `--resume <session_id>` to preserve agent context on retry
- **Git diff fallback (v2.21+)**: Injects partial progress into prompt when no session ID available
- Waits with countdown announcements when within max wait duration
- Saves execution state and exits cleanly for long waits (>6h by default)
- Resume paused executions via CLI commands

**Configuration:**

```yaml
budget:
  # Enable budget tracking (default: true)
  enabled: true

  # Enable intelligent wait/exit on rate limits (default: true)
  auto_resume: true

  # Maximum time to wait for rate limit reset (default: 6h)
  # If wait exceeds this, saves state and exits cleanly
  max_wait_duration: 6h

  # How often to announce countdown during wait (default: 15m)
  announce_interval: 15m

  # Extra buffer after reset before resuming (default: 60s)
  safety_buffer: 60s
```

**How It Works:**

1. **Rate Limit Detection**: When Claude CLI returns a rate limit error, Conductor parses the actual reset time from the output (unix timestamp, human-readable time, or retry headers)

2. **Wait Decision**:
   - If wait ≤ `max_wait_duration`: Wait with periodic countdown announcements
   - If wait > `max_wait_duration`: Save state and exit (typically weekly limits)

3. **State Persistence**: Saves to `.conductor/state/<session-id>.json`:
   - Current wave and completed tasks
   - Rate limit info with reset time
   - Plan file path for resume

4. **Resume**: Use `conductor budget resume` to continue paused executions

**Rate Limit Patterns Detected:**

| Pattern | Example | Type |
|---------|---------|------|
| Unix timestamp | `Claude AI usage limit reached\|1705420800` | Session |
| Human-readable | `Your limit will reset at 2pm (America/New_York)` | Session |
| Retry header | `retry in 300 seconds` | Session |
| JSON error | `{"error":"429 rate_limit_error", "retry_after": 300}` | API |
| Weekly limit | Wait > 24 hours | Weekly |

**Limit Types:**

| Type | Typical Wait | Behavior |
|------|--------------|----------|
| `session` | ~5 hours | Wait with countdown if within max_wait_duration |
| `weekly` | Days | Save state, exit, resume later |
| `unknown` | Varies | Falls back to 5-hour default |

**CLI Commands:**

```bash
# List paused executions
conductor budget list-paused

# Resume all ready executions
conductor budget resume

# Resume specific session
conductor budget resume <session-id>
```

**Example Workflow:**

```bash
# Start a long plan
conductor run large-plan.md --verbose

# ... 3 hours in, rate limit hit ...
# Conductor detects: reset in 5 hours
# Since 5h < 6h (max_wait_duration), it waits with countdown:
#   "Rate limit hit. Waiting 5h 0m until reset..."
#   "Rate limit countdown: 4h 45m remaining..."
#   "Rate limit countdown: 4h 30m remaining..."

# ... or if reset would be in 24+ hours (weekly limit) ...
# Conductor saves state and exits:
#   "Wait time (24h) exceeds max (6h). Saving state..."
#   "Execution paused. Resume with: conductor budget resume abc123"

# Later, resume when ready
conductor budget resume
```

**TTS Announcements (if enabled):**

When TTS is enabled, Conductor announces rate limit countdowns:
- "Rate limit hit. Waiting 5 hours until reset."
- "Rate limit countdown: 4 hours 30 minutes remaining."
- "Rate limit expired. Resuming execution."

**Mid-Task Recovery (v2.21+):**

When a rate limit occurs mid-task (agent has made partial progress), Conductor uses two strategies:

1. **Session Resume (Primary)**: Captures the Claude CLI session ID and uses `--resume <session_id>` on retry. This preserves the agent's full conversation context, allowing it to continue exactly where it left off.

2. **Git Diff Fallback**: If no session ID is available (e.g., rate limit occurred before JSON parsing completed), Conductor injects a git diff summary into the retry prompt:
   ```
   ## RATE LIMIT RECOVERY - PARTIAL PROGRESS DETECTED
   The following files were modified before the rate limit.
   DO NOT re-apply changes that already exist.
   
    internal/executor/task.go | 45 +++++++++++++++++
    internal/models/task.go   |  3 ++
    2 files changed, 48 insertions(+)
   ```

This prevents duplicate work and helps the agent understand what was already accomplished.

**Graceful Degradation:**

- If parsing fails → Falls back to 5-hour default wait
- If state save fails → Logs error, continues waiting
- If resume fails → Reports error with troubleshooting info

### Output & Logs

**Console Output:**
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

**Log Files:**

Execution logs written to `.conductor/logs/`:

```bash
.conductor/logs/
├── conductor-2025-11-10-143000.log
├── task-1-setup-143015.log
├── task-2-config-143030.log
└── ...
```

---

## Multi-File Plans & Objective Splitting

**Status**: ✅ Fully Implemented and Tested (Phase 2A)
**Test Coverage**: 37 integration tests, 100% backward compatible

### Overview

Phase 2A extends Conductor with the ability to split large implementation plans across multiple files while maintaining full dependency tracking and execution orchestration.

**Key Benefits:**
- Break monolithic plans into focused, manageable files
- Maintain full dependency graphs across files
- Enable team collaboration on different plan segments
- Better plan organization and version control
- Support partial plan re-execution on split plans

### Core Concepts

#### 1. Multi-File Plan Loading

Conductor automatically loads, validates, and merges multiple plan files:

```bash
# Load and execute multiple plan files
conductor run phase1-setup.md phase2-implementation.md phase3-testing.md

# Validate split plans before execution
conductor validate setup.md features.md deployment.md
```

**Format Auto-Detection**: Each file's format is detected independently:
- File `setup.md` → Markdown parser
- File `features.yaml` → YAML parser
- File `deploy.yaml` → YAML parser
- Mixed formats work seamlessly

#### 2. Objective Plan Splitting

Split a plan objectively by logical boundaries rather than arbitrary task counts:

**Good Split**:
```
frontend.md       → All UI/frontend tasks
backend.md        → Server/backend tasks
deployment.md     → Infrastructure/ops tasks
```

**Avoid**:
```
plan-part1.md     → Tasks 1-7
plan-part2.md     → Tasks 8-14  (arbitrary split)
```

#### 3. Worktree Groups

Group related tasks and define execution rules:

```markdown
## Task 1: Database Setup
**WorktreeGroup**: infrastructure

Initialize and seed database.

## Task 2: API Server
**WorktreeGroup**: backend-core

Start API server.

## Task 3: User Service
**WorktreeGroup**: backend-features

Build user management API.
```

**Group Definition** (in YAML plan):
```yaml
worktree_groups:
  - group_id: infrastructure
    description: Infrastructure and database setup
    execution_model: sequential
    isolation: strong
    rationale: State-dependent, must execute in order

  - group_id: backend-features
    description: API feature implementations
    execution_model: parallel
    isolation: weak
    rationale: Independent features can execute in parallel
```

#### 4. FileToTaskMap

Conductor automatically tracks which file each task originated from:

```
FileToTaskMap:
  "setup.md" → [1, 2, 3, 4]
  "features.md" → [5, 6, 7, 8]
  "deploy.md" → [9, 10]
```

**Benefits:**
- Resume on specific file segments
- Partial re-execution support
- Better logging and tracking
- File-aware task reporting

### Multi-File Usage Examples

#### Example 1: Basic Split Plan

**setup.md**:
```markdown
# Backend Setup Plan

## Task 1: Initialize Database
**Files**: infrastructure/database.sql
**Depends on**: None

Create PostgreSQL database and schema.

## Task 2: Setup Redis Cache
**Files**: infrastructure/cache-config.yaml
**Depends on**: Task 1

Configure Redis for caching.
```

**features.md**:
```markdown
# Backend Features Plan

## Task 3: Implement Auth Service
**Files**: internal/auth/auth.go
**Depends on**: Task 1, Task 2

Add JWT authentication.

## Task 4: Create User API
**Files**: internal/api/users.go
**Depends on**: Task 3

Implement user CRUD endpoints.
```

**Execution**:
```bash
# Validate all files together
conductor validate setup.md features.md

# Dry-run to test without execution
conductor run setup.md features.md --dry-run

# Execute with verbose output
conductor run setup.md features.md --verbose

# Resume with skip-completed
conductor run setup.md features.md --skip-completed
```

#### Example 2: Microservices Architecture

**auth-service.md** (6 tasks):
```markdown
# Authentication Service

## Task 1: Database Setup
**WorktreeGroup**: auth-infra

Initialize auth database.

## Task 2: JWT Implementation
**WorktreeGroup**: auth-core
**Depends on**: Task 1

Implement JWT token handling.

## Task 3: OAuth Integration
**WorktreeGroup**: auth-features
**Depends on**: Task 2

Add OAuth2 provider support.
```

**api-service.md** (8 tasks):
```markdown
# Main API Service

## Task 4: API Framework
**WorktreeGroup**: api-core

Set up HTTP server.

## Task 5: User Endpoints
**WorktreeGroup**: api-features
**Depends on**: Task 4

Create user management endpoints.

## Task 6: Product Endpoints
**WorktreeGroup**: api-features
**Depends on**: Task 4

Create product catalog endpoints.
```

**deployment.md** (5 tasks):
```markdown
# Deployment & Infrastructure

## Task 7: Terraform Setup
**WorktreeGroup**: deployment

Configure infrastructure.

## Task 8: Docker Build
**WorktreeGroup**: deployment
**Depends on**: Task 7

Build container images.

## Task 9: Deploy to K8s
**WorktreeGroup**: deployment
**Depends on**: Task 8

Deploy to Kubernetes cluster.
```

**Execution**:
```bash
# Execute all services together
conductor run auth-service.md api-service.md deployment.md

# With max concurrency per group
conductor run *.md --max-concurrency 4

# Monitor with file-aware logging
conductor run *.md --verbose --log-dir ./logs
```

#### Example 3: Phased Delivery

**phase1-foundation.md**:
```markdown
# Phase 1: Foundation

## Task 1: Project Structure
**Depends on**: None

Initialize project layout.

## Task 2: Build Pipeline
**Depends on**: Task 1

Set up CI/CD.
```

**phase2-features.md**:
```markdown
# Phase 2: Features

## Task 3: Feature A
**Depends on**: Task 2

Implement first feature.

## Task 4: Feature B
**Depends on**: Task 2

Implement second feature.
```

**phase3-polish.md**:
```markdown
# Phase 3: Polish

## Task 5: Performance Optimization
**Depends on**: Task 3, Task 4

Optimize critical paths.

## Task 6: Release Preparation
**Depends on**: Task 5

Prepare for release.
```

**Execution Strategy**:
```bash
# Execute all phases at once
conductor run phase1-*.md phase2-*.md phase3-*.md

# Or incrementally
conductor run phase1-foundation.md
conductor run phase1-foundation.md phase2-features.md --skip-completed
conductor run phase1-foundation.md phase2-features.md phase3-polish.md --skip-completed
```

### Best Practices for Split Plans

#### File Organization

1. **One Unit Per File**
   - Each file represents a cohesive feature/module/service
   - Aim for 5-20 tasks per file
   - Related tasks stay together

2. **Clear Naming**
   ```
   ✅ backend-setup.md          # Clear, specific
   ✅ frontend-features.md       # Clear, specific
   ✅ deployment-k8s.md          # Clear, specific

   ❌ part1.md, part2.md         # Vague
   ❌ tasks-1-5.md               # Arbitrary split
   ```

3. **File Order Matters**
   - List files in execution order when possible
   - Dependencies override file order, but order aids understanding
   - Reduce cross-file dependencies

#### Dependency Management

1. **Minimize Cross-File Dependencies**
   ```markdown
   ✅ Good: Task 3 (api.md) depends on Task 1,2 (db.md)
      One clear dependency boundary

   ❌ Avoid: Task 3 (a.md) → Task 5 (b.md) → Task 7 (c.md) → Task 4 (a.md)
      Tangled cross-file dependencies
   ```

2. **Clear Dependency Declarations**
   ```markdown
   ## Task 3: User API
   **Depends on**: Task 1, Task 2

   Create user endpoints (depends on DB setup and Cache setup).
   ```

3. **Validate Cross-File Links**
   ```bash
   # Always validate split plans together
   conductor validate setup.md features.md deploy.md
   ```

#### Worktree Groups

1. **Group by Execution Needs**
   ```yaml
   worktree_groups:
     # Sequential: State-dependent tasks
     - group_id: database
       execution_model: sequential
       isolation: strong
       rationale: Database setup and migrations must execute in order

     # Parallel: Independent tasks
     - group_id: services
       execution_model: parallel
       isolation: weak
       rationale: Microservices can be built independently
   ```

2. **Isolation Levels**
   - `none` - No isolation (default, allows any overlap)
   - `weak` - Weak isolation (can share resources)
   - `strong` - Strong isolation (no resource sharing, may require separate worktrees)

3. **Execution Models**
   - `parallel` - Run all tasks in group in parallel (when dependencies allow)
   - `sequential` - Run tasks in group one at a time

#### Testing Split Plans

1. **Validation First**
   ```bash
   # Validate all files together to catch cross-file issues
   conductor validate setup.md api.md frontend.md deploy.md
   ```

2. **Dry-Run Before Execution**
   ```bash
   # Test without actually executing tasks
   conductor run setup.md api.md --dry-run --verbose
   ```

3. **Check Dependency Graph**
   ```bash
   # Verbose output shows task execution order and dependencies
   conductor run *.md --verbose --dry-run
   ```

### Merging Strategy

Conductor uses these rules when merging multiple plan files:

1. **Task Number Preservation**
   - Task 1 from setup.md stays as Task 1
   - Task 1 from features.md becomes Task N (renumbered)
   - Automatic renumbering to maintain uniqueness

2. **Dependency Resolution**
   - Cross-file dependencies maintained
   - Cycle detection works across files
   - Dependency graph validated before execution

3. **Deduplication**
   - Duplicate task names not allowed (error if same task in multiple files)
   - File origins preserved for logging and resume

4. **WorktreeGroup Merging**
   - All groups from all files merged
   - Group IDs must be unique across files
   - Execution respects all group constraints

### Advanced Patterns

#### Pattern 1: Optional Features

**core-plan.md** (required):
```markdown
## Task 1: Database
## Task 2: API Server
## Task 3: Auth
```

**optional-features.md** (optional):
```markdown
## Task 4: Analytics
**Depends on**: Task 2

Add analytics tracking.

## Task 5: Admin Panel
**Depends on**: Task 2, Task 3

Create admin interface.
```

**Execution**:
```bash
# Run core only
conductor run core-plan.md

# Run with optional features
conductor run core-plan.md optional-features.md
```

#### Pattern 2: Conditional Deployment

**app-plan.md** (always):
```markdown
## Task 1-5: Build application
```

**deploy-staging.md** (for staging):
```markdown
## Task 6: Deploy to Staging
**Depends on**: Task 5
```

**deploy-prod.md** (for production):
```markdown
## Task 7: Deploy to Production
**Depends on**: Task 5
```

**Execution**:
```bash
# Build only
conductor run app-plan.md

# Build and deploy to staging
conductor run app-plan.md deploy-staging.md

# Build and deploy to production
conductor run app-plan.md deploy-prod.md
```

#### Pattern 3: Multi-Service Deployment

**shared-infra.md** (prerequisite):
```markdown
## Task 1-3: Shared infrastructure
```

**service-auth.md** (first service):
```markdown
## Task 4-6: Auth service
**Depends on**: Task 3
```

**service-api.md** (second service):
```markdown
## Task 7-10: API service
**Depends on**: Task 3
```

**service-frontend.md** (third service):
```markdown
## Task 11-13: Frontend
**Depends on**: Task 10 (API ready)
```

**Execution**:
```bash
# Build everything together
conductor run shared-infra.md service-auth.md service-api.md service-frontend.md

# Or in phases
conductor run shared-infra.md
conductor run shared-infra.md service-auth.md service-api.md --skip-completed
```

---

## Best Practices

### Worktree Organization

#### ExecutionModel

Defines how tasks in a group should be executed:

**Sequential**: Tasks execute one after another
```yaml
execution_model: sequential
```
- Task 1 completes → Task 2 starts → Task 3 starts
- Guarantees ordering within group
- Best for state-dependent operations

**Parallel**: Tasks execute concurrently (when dependencies allow)
```yaml
execution_model: parallel
```
- Task 1, 2, 3 can start together
- Limited by task dependencies
- Best for independent operations

#### Isolation

Defines whether tasks can share resources:

**Strong**: No shared resources
```yaml
isolation: strong
```
- Each task has independent environment
- May require separate worktrees/directories
- Best for infrastructure and ops tasks

**Weak**: Can share some resources
```yaml
isolation: weak
```
- Tasks can use shared database, cache
- Tasks can modify shared state
- Best for application features

**None**: Maximum resource sharing
```yaml
isolation: none
```
- Default, no isolation enforced
- Tasks can freely interact
- Best for utility and helper tasks

#### Group Definition Patterns

**Infrastructure Groups**:
```yaml
worktree_groups:
  - group_id: database
    description: Database schema and migrations
    execution_model: sequential
    isolation: strong
    rationale: |
      Database migrations must execute in order. Each migration
      builds on the previous state. Strong isolation prevents
      concurrent schema modifications.
```

**Feature Groups**:
```yaml
worktree_groups:
  - group_id: user-service
    description: User management API and logic
    execution_model: parallel
    isolation: weak
    depends_on: database
    rationale: |
      User endpoints (get, create, update, delete) are independent
      and can be implemented in parallel. Weak isolation allows
      shared database access.
```

**Testing Groups**:
```yaml
worktree_groups:
  - group_id: unit-tests
    description: Unit test suite
    execution_model: parallel
    isolation: none
    rationale: |
      Unit tests are independent and can run in parallel.
      No isolation needed.

  - group_id: integration-tests
    description: Integration test suite
    execution_model: sequential
    isolation: strong
    depends_on: unit-tests
    rationale: |
      Integration tests may conflict if run simultaneously.
      Strong isolation and sequential execution ensure reliability.
```

#### Task Assignment Patterns

**Single Task per Group**:
```markdown
## Task 1: Initialize Database Schema
**WorktreeGroup**: database-schema

Create base database schema.
```

**When to Use:**
- Task is critical and shouldn't be parallelized
- Task has strict ordering requirements
- Task conflicts with other operations

**Related Tasks in Group**:
```markdown
## Task 4: Implement GET /users
**WorktreeGroup**: user-api

## Task 5: Implement POST /users
**WorktreeGroup**: user-api

## Task 6: Implement PUT /users/:id
**WorktreeGroup**: user-api
```

**Advantages:**
- Logically grouped related functionality
- Can be developed by single team member
- Clear scope and responsibility

#### Decision Matrix

Use this matrix to decide group configuration:

```
QUESTION 1: Can tasks run simultaneously?
  ├─ NO (state-dependent)
  │  └─ execution_model: sequential
  └─ YES (independent)
     └─ execution_model: parallel

QUESTION 2: Will tasks share resources?
  ├─ YES, safely (shared DB, cache)
  │  └─ isolation: weak
  ├─ NO (separate environments)
  │  └─ isolation: strong
  └─ NONE (utility tasks)
     └─ isolation: none

QUESTION 3: Does group depend on others?
  ├─ YES → List as depends_on
  └─ NO → depends_on: []
```

### Dependency Design

1. **Start with Foundation**: Place setup/infrastructure tasks first
2. **Maximize Parallelism**: Minimize dependencies where possible
3. **Logical Dependencies**: Only depend on what you actually need
4. **Avoid Chains**: Long dependency chains reduce parallelism
5. **Test Independence**: Make test tasks independent when possible

### Plan Design Guidelines

1. **Task Size**: 5-15 minute tasks work best
2. **Clear Task Names**: Use descriptive, actionable names
3. **Specify Files**: Always include `File(s)` metadata
4. **Estimate Time**: Provide realistic time estimates
5. **Use Agents**: Specify appropriate agent for task type
6. **Detailed Descriptions**: Include enough detail for autonomous execution
7. **Code Examples**: Add code snippets for complex requirements

### Execution Best Practices

1. **Validate First**: Always validate before running
2. **Use Dry Run**: Test execution plan with `--dry-run`
3. **Start Conservative**: Use low `--max-concurrency` initially
4. **Monitor Logs**: Check log files for issues
5. **Keep Plans Updated**: Mark completed tasks to avoid re-execution

### Performance Optimization

1. **Tune Concurrency**: Adjust based on task complexity and resource usage
2. **Set Appropriate Timeouts**: Allow enough time for complex tasks
3. **Use Agent Specialization**: Assign appropriate agents for tasks
4. **Split Large Tasks**: Break into smaller, parallelizable subtasks
5. **Review QC Feedback**: Use to improve future plans

---

## Adaptive Learning System

**Version**: Conductor v2.17.0
**Status**: Production-ready

### Learning Overview

The adaptive learning system is a feedback mechanism that improves task execution over time.

**What It Does:**
- Records every task execution attempt with success/failure data
- Analyzes failure patterns using keyword detection
- Adapts agent selection after repeated failures
- Enhances prompts with learned context from past executions
- Improves success rates over time through cross-run intelligence
- Enables inter-retry agent swapping based on QC recommendations

**Benefits:**
- Automatic recovery from repeated failures
- Reduced manual intervention
- Pattern recognition for common failure types
- Historical intelligence across multiple runs
- Graceful degradation (learning failures never break task execution)
- Intelligent agent selection during retry loops

**Key Features:**
- SQLite-based execution history database
- Three integration hooks: pre-task, QC-review, post-task
- Structured QC responses with agent suggestions
- Inter-retry agent swapping based on QC feedback
- Dual feedback storage (plan files and database)
- Pattern detection for 6+ common failure types
- Session and run number tracking
- Four CLI observability commands
- Configurable adaptation thresholds
- Project-local learning data (`.conductor/learning/`)

### How It Works

#### Execution Pipeline with Learning

```
1. Pre-Task Hook
   ├─ Query execution history for this task
   ├─ Load context from database and/or plan file (configurable)
   ├─ Analyze failure patterns and agent performance
   ├─ Adapt agent if 2+ failures detected (configurable)
   └─ Enhance prompt with learning context

2. Task Execution
   └─ Execute task with (potentially adapted) agent

3. QC-Review Hook
   ├─ Load historical context (if qc_reads_plan_context/qc_reads_db_context enabled)
   ├─ Capture structured QC response (verdict, issues, recommendations)
   ├─ Parse suggested_agent for inter-retry swapping
   └─ Extract failure patterns from output

4. Post-Task Hook
   ├─ Record execution to dual storage (plan file + database)
   ├─ Store: agent used, QC verdict, patterns, timing, feedback
   └─ Update run statistics

5. Inter-Retry Agent Swap (if RED verdict)
   ├─ Check if suggested_agent provided by QC
   ├─ Verify swap_during_retries enabled in config
   └─ Swap to suggested agent for next retry attempt
```

#### Inter-Retry Agent Swapping (v2.1.0)

When a task fails with a RED verdict, the QC agent can suggest a better agent for the retry:

```
Initial Execution:
  Task 5 → backend-developer agent → RED verdict
  QC suggests: "golang-pro" (better at Go-specific tasks)

Retry 1:
  Task 5 → golang-pro agent (auto-swapped) → GREEN verdict
```

**Configuration:**
```yaml
learning:
  swap_during_retries: true        # Enable inter-retry swapping
  min_failures_before_adapt: 2     # Require 2+ failures before adapting
```

**Workflow:**

1. Task executes with assigned agent
2. QC reviews output and returns RED verdict
3. QC includes `suggested_agent` field in response
4. Conductor checks if `swap_during_retries` is enabled
5. If enabled, conductor swaps to suggested agent for next retry
6. Retry executes with new agent
7. If successful, learning system records which agent succeeded

**Benefits:**
- Faster recovery from failures without manual intervention
- Leverages QC agent's understanding of task requirements
- Automatic correction of agent mismatches
- Learning system tracks successful agent swaps for future runs

#### Failure Pattern Detection

The system detects these common patterns:

| Pattern | Keywords Detected | Example Scenario |
|---------|-------------------|------------------|
| `compilation_error` | "compilation failed", "syntax error" | Code fails to compile |
| `test_failure` | "test failed", "assertion failed" | Tests don't pass |
| `dependency_missing` | "module not found", "import error" | Missing dependencies |
| `permission_error` | "permission denied", "access denied" | File/directory access issues |
| `timeout` | "timeout", "exceeded deadline" | Long-running operations |
| `runtime_error` | "runtime error", "panic", "exception" | Runtime failures |

#### Agent Adaptation Logic

```go
if failures >= config.MinFailuresBeforeAdapt (default: 2) {
    // Query database for agents that succeeded on similar tasks
    suggestedAgent = findBestAlternativeAgent(taskPattern, triedAgents)

    if suggestedAgent != "" && suggestedAgent != currentAgent {
        // Switch to better-performing agent
        task.Agent = suggestedAgent
        log.Info("Adapting agent: %s → %s", currentAgent, suggestedAgent)
    }
}
```

#### Session Management

Each conductor run gets a unique session ID:
- Format: `session-YYYYMMDD-HHMMSS`
- Example: `session-20250112-143045`
- Used for grouping executions and tracking run numbers

Run numbers increment per plan file:
- First execution of `plan.md`: run 1
- Second execution of `plan.md`: run 2
- Independent tracking per plan file

### Configuration (Learning)

#### Default Configuration

```yaml
learning:
  # Enable or disable the learning system (default: true)
  enabled: true

  # Database path (default: .conductor/learning/executions.db)
  db_path: .conductor/learning/executions.db

  # Auto-adapt agent selection based on learned patterns (default: false)
  auto_adapt_agent: false

  # Enable agent swapping during retry loops (default: true)
  swap_during_retries: true

  # Enhance prompts with learned context (default: true)
  enhance_prompts: true

  # Enable QC to read plan file context (default: true)
  qc_reads_plan_context: true

  # Enable QC to read database context (default: true)
  qc_reads_db_context: true

  # Maximum context entries to load (default: 10)
  max_context_entries: 10

  # Minimum failures before adapting strategy (default: 2)
  min_failures_before_adapt: 2

  # Days to keep execution history (default: 90, 0 = forever)
  keep_executions_days: 90

  # Maximum execution records per task (default: 100)
  max_executions_per_task: 100
```

#### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Master switch for learning system |
| `db_path` | string | `.conductor/learning/executions.db` | Path to SQLite database file |
| `auto_adapt_agent` | bool | `false` | Automatically switch agents on failures |
| `swap_during_retries` | bool | `true` | Enable inter-retry agent swapping |
| `enhance_prompts` | bool | `true` | Add learned context to prompts |
| `qc_reads_plan_context` | bool | `true` | QC loads execution history from plan |
| `qc_reads_db_context` | bool | `true` | QC loads execution history from database |
| `max_context_entries` | int | `10` | Maximum historical entries for QC context |
| `min_failures_before_adapt` | int | `2` | Failure threshold before adapting |
| `keep_executions_days` | int | `90` | Days to retain history (0 = forever) |
| `max_executions_per_task` | int | `100` | Max records per task |

#### Configuration File Location

Place configuration at `.conductor/config.yaml`:

```yaml
# .conductor/config.yaml

# Execution settings
max_concurrency: 3
timeout: 30m

# Learning settings
learning:
  enabled: true
  auto_adapt_agent: true
  min_failures_before_adapt: 3
```

#### Disabling Learning

To completely disable the learning system:

```yaml
learning:
  enabled: false
```

#### Dual Feedback Storage (v2.1.0)

Conductor stores execution feedback in two complementary locations:

**1. Plan File Storage (Human-Readable)**

Execution history is embedded directly in plan files:

**Markdown Format:**
```markdown
## Task 5: Implement error handling

### Execution History

#### Attempt 1 - 2025-11-15T16:41:59Z
**Agent**: backend-developer
**Verdict**: RED
**Output**:
```
Implementation incomplete. Missing error handling...
```
**QC Feedback**:
```
Task failed validation. Error handling not implemented for database connections.
```

#### Attempt 2 - 2025-11-15T16:45:30Z
**Agent**: golang-pro
**Verdict**: GREEN
**Output**:
```
Successfully implemented error handling with retry logic...
```
**QC Feedback**:
```
Implementation complete with proper error handling.
```
```

**YAML Format:**
```yaml
tasks:
  - id: 5
    name: Implement error handling
    agent: golang-pro
    execution_history:
      - attempt_number: "1"
        agent: "backend-developer"
        verdict: "RED"
        agent_output: "Implementation incomplete..."
        qc_feedback: "Task failed validation..."
        timestamp: "2025-11-15T16:41:59Z"
      - attempt_number: "2"
        agent: "golang-pro"
        verdict: "GREEN"
        agent_output: "Successfully implemented..."
        qc_feedback: "Implementation complete..."
        timestamp: "2025-11-15T16:45:30Z"
```

**2. Database Storage (Long-Term Analysis)**

Structured data stored in SQLite for pattern analysis:
- Location: `.conductor/learning/executions.db`
- Queryable execution records
- Cross-run pattern detection
- Agent performance metrics
- Failure categorization

**Configuration:**
```yaml
feedback:
  store_in_plan_file: true   # Enable plan file storage
  store_in_database: true    # Enable database storage
  format: json               # Response format (json or text)
  store_on_green: true       # Store successful executions
  store_on_red: true         # Store failed executions
  store_on_yellow: true      # Store warning executions
```

**Benefits of Dual Storage:**
- Plan files provide immediate visibility into task history
- Database enables long-term pattern analysis
- No duplicates: single authoritative record per attempt
- Human-readable format for debugging
- Machine-readable format for automation
- Complete execution audit trail

### CLI Commands (Learning)

See [Learning Commands](#learning-commands) section above for detailed documentation of:
- `conductor learning stats`
- `conductor learning show`
- `conductor learning clear`
- `conductor learning export`

### Usage Examples

#### Basic Usage (Learning Enabled by Default)

```bash
# First run - learning records execution
$ conductor run plan.md

# Second run - learning adapts if there were failures
$ conductor run plan.md --skip-completed

# View statistics
$ conductor learning stats plan.md
```

#### Scenario 1: Inter-Retry Agent Swapping (v2.1.0)

```yaml
# .conductor/config.yaml
learning:
  enabled: true
  swap_during_retries: true
  qc_reads_plan_context: true
  qc_reads_db_context: true
quality_control:
  retry_on_red: 2
```

**What happens:**

1. **Attempt 1**: Task 5 executes with `backend-developer` agent
2. **QC Review**: Returns RED verdict with `"suggested_agent": "golang-pro"`
3. **Retry 1**: Conductor swaps to `golang-pro` agent (inter-retry swap)
4. **Result**: Task 5 succeeds with suggested agent

**Console output:**
```
[INFO] Executing Task 5 with agent: backend-developer
[WARN] Task 5 received RED verdict
[INFO] QC suggested agent: golang-pro
[INFO] Swapping agent for retry: backend-developer → golang-pro
[INFO] Retry 1/2: Executing Task 5 with agent: golang-pro
[SUCCESS] Task 5 completed: GREEN
```

**QC Response:**
```json
{
  "verdict": "RED",
  "feedback": "Implementation uses non-idiomatic Go patterns",
  "issues": [
    {
      "severity": "critical",
      "description": "Using panic for error handling instead of returning errors",
      "location": "internal/db/client.go:42"
    }
  ],
  "recommendations": [
    "Use Go error handling patterns (return error, not panic)",
    "Switch to an agent specialized in Go development"
  ],
  "should_retry": true,
  "suggested_agent": "golang-pro"
}
```

#### Scenario 2: Automatic Agent Adaptation (Cross-Run)

```yaml
# .conductor/config.yaml
learning:
  enabled: true
  auto_adapt_agent: true
  min_failures_before_adapt: 2
```

**What happens:**

1. **First run**: Task 3 fails with `backend-developer` agent
2. **Second run**: Task 3 fails again with `backend-developer` agent
3. **Third run**: System automatically switches to `golang-pro` agent
4. **Result**: Task 3 succeeds with adapted agent

**Console output:**
```
[INFO] Pre-task hook: Analyzing Task 3 history...
[INFO] Detected 2 failures with agent: backend-developer
[INFO] Adapting agent: backend-developer → golang-pro
[INFO] Executing Task 3 with agent: golang-pro
[SUCCESS] Task 3 completed: GREEN
```

#### Scenario 3: Pattern-Based Learning

```bash
# Run plan that encounters compilation errors
$ conductor run backend-plan.md

# Learning records pattern: compilation_error
# View detected patterns
$ conductor learning show backend-plan.md "Task 5"

# Next run - prompt enhanced with pattern context
$ conductor run backend-plan.md --skip-completed
```

**Enhanced prompt includes:**
```
Previous attempts failed with: compilation_error
Common issues: syntax errors, missing imports
Suggestion: Verify syntax before execution
```

#### Scenario 4: Cross-Run Intelligence

```bash
# Day 1: First execution
$ conductor run feature-plan.md
# 3 tasks succeed, 2 fail with test_failure

# Day 2: Retry with learning
$ conductor run feature-plan.md --retry-failed
# Learning suggests agents that succeeded on similar tasks
# Failed tasks now use better-performing agents

# Check improvement
$ conductor learning stats feature-plan.md
# Shows: "Success rate improved from 60% to 100%"
```

#### Scenario 5: Exporting for Analysis

```bash
# Export learning data
$ conductor learning export plan.md --format json > data.json

# Analyze with jq
$ cat data.json | jq '.[] | select(.success == false)'

# Find most reliable agent
$ cat data.json | jq -r '.[] | select(.success == true) | .agent' | sort | uniq -c | sort -rn
   15 golang-pro
    8 backend-developer
    5 fullstack-dev
```

#### Scenario 6: QC Context Loading

```yaml
# .conductor/config.yaml
learning:
  qc_reads_plan_context: true
  qc_reads_db_context: true
  max_context_entries: 10
```

**What happens:**

1. Task executes and produces output
2. QC agent receives context including:
   - Last 10 execution attempts from plan file
   - Historical patterns from database
   - Previous QC feedback
3. QC makes informed decision based on historical context
4. QC can suggest alternative agents based on past successes

**Benefits:**
- QC understands task history before reviewing
- Avoids repeating same suggestions
- Learns from what worked and what did not
- Provides contextually relevant recommendations

### Database Schema

```sql
-- Task execution history
CREATE TABLE task_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_number TEXT NOT NULL,
    task_name TEXT NOT NULL,
    agent TEXT,
    prompt TEXT NOT NULL,
    success BOOLEAN NOT NULL,
    output TEXT,
    error_message TEXT,
    duration_seconds INTEGER,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    context TEXT  -- JSON blob for patterns, session, run number
);

-- Approach history (tracks tried approaches)
CREATE TABLE approach_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_pattern TEXT NOT NULL,
    approach_description TEXT NOT NULL,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    last_used TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT
);
```

### Execution History Format (v2.1.0)

Each task execution attempt is recorded with the following schema:

**Plan File Format (YAML):**
```yaml
execution_history:
  - attempt_number: "1"
    agent: "backend-developer"
    verdict: "RED"
    agent_output: |
      Started implementing error handling...
      Encountered issue with database connection...
      Implementation incomplete.
    qc_feedback: |
      Task failed validation. Error handling not implemented properly.
      Missing retry logic for transient failures.
      Suggested agent: golang-pro
    timestamp: "2025-11-15T16:41:59Z"

  - attempt_number: "2"
    agent: "golang-pro"
    verdict: "GREEN"
    agent_output: |
      Implemented error handling with proper Go patterns.
      Added retry logic with exponential backoff.
      All tests passing.
    qc_feedback: |
      Implementation complete and follows Go best practices.
      Error handling is comprehensive.
    timestamp: "2025-11-15T16:45:30Z"
```

**Schema Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `attempt_number` | string | Sequential attempt number (1, 2, 3...) |
| `agent` | string | Agent used for this attempt |
| `verdict` | string | QC verdict: GREEN, RED, or YELLOW |
| `agent_output` | string | Raw output from the task agent |
| `qc_feedback` | string | Feedback from quality control review |
| `timestamp` | string | ISO 8601 timestamp of execution |

**Plan File Format (Markdown):**
```markdown
### Execution History

#### Attempt 1 - 2025-11-15T16:41:59Z
**Agent**: backend-developer
**Verdict**: RED
**Output**:
```
Started implementing error handling...
Implementation incomplete.
```
**QC Feedback**:
```
Task failed validation. Missing retry logic.
```

#### Attempt 2 - 2025-11-15T16:45:30Z
**Agent**: golang-pro
**Verdict**: GREEN
**Output**:
```
Implemented error handling with proper patterns.
```
**QC Feedback**:
```
Implementation complete.
```
```

**No Duplicate Storage:**
Each execution attempt is stored exactly once with full context. The plan file contains human-readable history while the database stores the same information in queryable format. This ensures a single source of truth without redundancy.

### Learning FAQ

**Q: Does learning work with all plan formats?**
A: Yes, learning works with both Markdown and YAML plan formats.

**Q: Is learning data version controlled?**
A: No, `.conductor/learning/` is in `.gitignore`. Learning data is project-local and should not be committed.

**Q: Does learning work with multi-file plans?**
A: Yes, learning tracks execution history per plan file, so multi-file plans maintain separate learning contexts.

**Q: Can I share learning data across team?**
A: Not recommended. Learning data is optimized for individual developer environments.

**Q: Does learning slow down execution?**
A: Minimal impact. Hook operations are async where possible, and database queries are optimized with indexes.

**Q: Should I enable `auto_adapt_agent` by default?**
A: Recommended to start with `false` (manual control) and enable after understanding behavior.

**Q: Does learning work in CI/CD environments?**
A: Yes, but learning database won't persist between runs unless you configure persistent storage. For CI/CD, you may want to disable learning.

**Q: What's the difference between inter-retry swapping and auto_adapt_agent?**
A: Inter-retry swapping (`swap_during_retries`) occurs during the retry loop based on QC suggestions. Auto-adapt agent (`auto_adapt_agent`) occurs at pre-task hook based on historical failures across runs.

**Q: How does QC know which agent to suggest?**
A: QC analyzes the task output and failure patterns. It can suggest agents based on:
- Task type (Go code suggests golang-pro)
- Failure patterns (syntax errors suggest different approach)
- Historical success data from database context

**Q: Can I disable dual feedback storage?**
A: Yes, configure in `.conductor/config.yaml`:
```yaml
feedback:
  store_in_plan_file: false  # Disable plan file storage
  store_in_database: true    # Keep database storage only
```

---

## Text-to-Speech Voice Feedback (v2.14+)

### Overview

Conductor supports optional voice announcements via a local Orpheus TTS server. When enabled, execution events are announced audibly, enabling hands-free monitoring of long-running plans.

### What Gets Announced

| Event | Example |
|-------|---------|
| Wave start | "Starting Wave 1 with 3 tasks" |
| Agent deployment | "Deploying agent golang-pro" |
| QC agent selection | "Deploying QC agents code-reviewer and qa-expert" |
| QC selection rationale | Full intelligent selection reasoning |
| Individual QC verdicts | "code-reviewer says GREEN" |
| Aggregated QC result | "QC passed" / "QC failed" / "QC passed with warnings" |
| Wave complete | "Wave 1 completed, all tasks passed" |
| Run complete | "Run completed. All 5 tasks passed" |

### Prerequisites

| Component | Purpose | Installation |
|-----------|---------|--------------|
| **Ollama** | Local LLM runtime | [ollama.ai](https://ollama.ai) |
| **Orpheus Model** | TTS model (~2.4GB) | `ollama pull legraphista/Orpheus` |
| **Orpheus-FastAPI** | OpenAI-compatible API | [GitHub](https://github.com/Lex-au/Orpheus-FastAPI) |
| **afplay** (macOS) | Audio playback | Built-in |
| **aplay** (Linux) | Audio playback | `apt install alsa-utils` |

### Server Setup

```bash
# 1. Install Ollama from https://ollama.ai

# 2. Pull Orpheus model
ollama pull legraphista/Orpheus

# 3. Clone and run Orpheus-FastAPI
git clone https://github.com/Lex-au/Orpheus-FastAPI.git
cd Orpheus-FastAPI
pip install -r requirements.txt
python main.py  # Runs on http://localhost:5005

# 4. Verify server
curl http://localhost:5005/
```

### Configuration

Add to `.conductor/config.yaml`:

```yaml
tts:
  enabled: true                        # Enable TTS (default: false)
  base_url: "http://localhost:5005"    # Orpheus server URL
  model: "orpheus"                     # TTS model
  voice: "tara"                        # Voice: tara, leah, jess, leo, dan, mia, zac, zoe
  timeout: 30s                         # Request timeout
```

### Available Voices

| Voice | Description |
|-------|-------------|
| tara | Female, clear and professional |
| leah | Female, warm tone |
| jess | Female, energetic |
| leo | Male, neutral |
| dan | Male, deep |
| mia | Female, soft |
| zac | Male, casual |
| zoe | Female, bright |

### Graceful Degradation

TTS is designed to never block execution:

| Condition | Behavior |
|-----------|----------|
| `tts.enabled: false` | No TTS API calls |
| Server unavailable | Silently ignored |
| Queue full (100 msgs) | New announcements dropped |
| Unsupported OS | Silently ignored |
| HTTP timeout | Silent failure, continues |

### Architecture

```
Execution Event → Logger Interface → TTSLogger → Announcer → Client → Orpheus
                                                                    → WAV → afplay/aplay
```

**Design decisions:**
- **Fire-and-forget**: TTS errors never block task execution
- **Lazy health check**: Server availability checked once on first announcement
- **Serialized queue**: 100-message buffer prevents overlapping audio
- **Graceful degradation**: Disabled TTS = zero behavior change from pre-TTS versions

---

## Troubleshooting & FAQ

### Common Errors

#### "Circular dependency detected"

**Symptoms:**
```
Error: Circular dependency detected in task graph
Cycle: Task 1 → Task 2 → Task 3 → Task 1
```

**Cause:** Tasks have circular dependencies.

**Solution:**
1. Review dependency chain in error message
2. Identify which dependency to remove
3. Restructure tasks to break the cycle

**Example Fix:**

Before (circular):
```markdown
## Task 1: A
**Depends on**: Task 3

## Task 2: B
**Depends on**: Task 1

## Task 3: C
**Depends on**: Task 2
```

After (fixed):
```markdown
## Task 1: A
**Depends on**: None

## Task 2: B
**Depends on**: Task 1

## Task 3: C
**Depends on**: Task 2
```

#### "Task depends on non-existent task"

**Symptoms:**
```
Error: Task 2 depends on non-existent Task 99
```

**Cause:** Task references dependency that doesn't exist in plan.

**Solution:**
1. Check task numbers in plan
2. Verify dependency format: "Task N" (capital T, space, number)
3. Ensure referenced task is defined

#### "File modified by multiple tasks"

**Symptoms:**
```
Warning: File main.go modified by Task 1 and Task 3
```

**Cause:** Multiple tasks specify the same file.

**Solutions:**

**Option 1: Add Dependencies**
```markdown
## Task 1: Initial Implementation
**File(s)**: main.go

## Task 3: Enhancement
**File(s)**: main.go
**Depends on**: Task 1
```

**Option 2: Split Files**
```markdown
## Task 1: Core Logic
**File(s)**: core.go

## Task 3: Helper Functions
**File(s)**: helpers.go
```

#### "Unknown agent"

**Symptoms:**
```
Error: Agent 'custom-agent' not found in ~/.claude/agents/
```

**Solution:**

1. Check agent directory:
```bash
ls -la ~/.claude/agents/
```

2. Verify agent file exists:
```bash
ls ~/.claude/agents/custom-agent.md
```

3. Use existing agent or create missing agent

#### "Agent Not Found Errors"

**Symptoms:**

When executing a plan with agent validation enabled, you may see errors like:
```
Agent 'code-reviewer' not found in registry (referenced in quality_control.agents config)

Available agents: python-pro, golang-pro, javascript-pro
```

Or for task-specific agents:
```
Agent 'ml-specialist' not found in registry (referenced in task 5)

Available agents: python-pro, golang-pro, javascript-pro
```

**Common Causes:**

1. **Misspelled agent name**: Agent name in plan or config doesn't match available agents
2. **Missing agent directory**: Agent file not present in `~/.claude/agents/`
3. **Invalid agent format**: Agent file exists but has incorrect format or structure

**Solutions:**

**Solution 1: Check agent name spelling**
```bash
# Validate plan and show agent names used
conductor validate plan.yaml

# List available agents
ls ~/.claude/agents/
```

**Solution 2: List all available agents**
```bash
# View agent directory
ls -la ~/.claude/agents/

# Search for specific agent by name pattern
ls ~/.claude/agents/ | grep code
```

**Solution 3: Verify agent file format**

Valid agent file at `~/.claude/agents/code-reviewer.md`:
```markdown
# Code Reviewer Agent

Description of the agent's role and capabilities.
```

Check the agent file exists and is readable:
```bash
cat ~/.claude/agents/code-reviewer.md
```

**Next Steps:**

- Fix agent name spelling in plan or config
- Create missing agent if needed
- Re-run with corrected agent references
```bash
conductor run plan.yaml
```

#### "Context deadline exceeded" / "Timeout"

**Symptoms:**
```
Error: Task 1 execution timeout: context deadline exceeded
```

**Solutions:**

**Option 1: Increase Timeout**
```bash
conductor run plan.md --timeout 1h

# Or via config
cat > .conductor/config.yaml << EOF
execution_timeout: 1h
task_timeout: 15m
EOF
```

**Option 2: Split Large Tasks**
Break complex tasks into smaller subtasks.

### Validation Issues

#### "Invalid plan format"

**Symptoms:**
```
Error: Failed to parse plan file: invalid format
```

**Solution:**

1. **Check file extension**:
   - Markdown: `.md` or `.markdown`
   - YAML: `.yaml` or `.yml`

2. **Validate Markdown format**:
```markdown
# Plan Title (required H1)

## Task 1: Task Name (required H2 with "Task N:" prefix)
**File(s)**: files.go (metadata with bold keys)

Task description.
```

3. **Validate YAML format**:
```yaml
plan:                    # Required root key
  name: Plan Title       # Required
  tasks:                 # Required
    - id: 1              # Required
      name: Task Name    # Required
```

4. **Use validate command**:
```bash
conductor validate plan.md
```

#### "No tasks found"

**Symptoms:**
```
Error: No tasks found in plan file
```

**Solution:**

1. **Markdown**: Ensure H2 headings with "Task N:" prefix:
```markdown
## Task 1: First Task
```
NOT:
```markdown
## First Task (missing "Task 1:" prefix)
```

2. **YAML**: Ensure tasks array not empty:
```yaml
plan:
  name: Plan
  tasks:
    - id: 1
      name: First Task
```

### Execution Problems

#### "Quality control failed" / "RED verdict"

**Symptoms:**
```
Task 1 received RED verdict from quality control
Retry 1/2...
Error: Task 1 failed after 2 retries
```

**Solution:**

1. **Check task logs**:
```bash
cat .conductor/logs/task-1-*.log
```

2. **Review QC feedback**:
```
Quality Control: RED

Feedback: Implementation incomplete. Missing error handling
for database connection failures.
```

3. **Fix underlying issue**:
   - Clarify task requirements
   - Improve task description
   - Add code examples
   - Simplify complex tasks

4. **Adjust retry limit**:
```yaml
# .conductor/config.yaml
max_retries: 3
```

#### "Claude CLI not found"

**Symptoms:**
```
Error: Failed to execute claude command: executable file not found in $PATH
```

**Solution:**

1. **Verify installation**:
```bash
which claude
claude --version
```

2. **Install Claude Code CLI**:
Follow instructions at https://claude.ai/code

3. **Add to PATH**:
```bash
export PATH="$PATH:/path/to/claude"
echo 'export PATH="$PATH:/path/to/claude"' >> ~/.bashrc
```

4. **Verify authentication**:
```bash
claude auth status
```

#### "Permission denied"

**Symptoms:**
```
Error: Failed to write to plan file: permission denied
Error: Failed to create log directory: permission denied
```

**Solution:**

1. **Check file permissions**:
```bash
ls -la plan.md
ls -la .conductor/
```

2. **Fix permissions**:
```bash
chmod 644 plan.md
chmod 755 .conductor/
chmod 755 .conductor/logs/
```

3. **Use custom log directory**:
```bash
conductor run plan.md --log-dir ~/conductor-logs
```

#### "File lock timeout"

**Symptoms:**
```
Error: Failed to acquire file lock after 30s
```

**Solution:**

1. **Check for running instances**:
```bash
ps aux | grep conductor
```

2. **Remove stale lock**:
```bash
rm .conductor/.lock
```

3. **Increase lock timeout**:
```yaml
# .conductor/config.yaml
lock_timeout: 1m
```

### Learning System Troubleshooting

#### Learning Database Not Created

**Symptom**: No `.conductor/learning/` directory created

**Solution:**
```bash
# Check config
$ cat .conductor/config.yaml | grep -A5 learning

# Verify learning enabled
learning:
  enabled: true  # Must be true

# Check permissions
$ ls -ld .conductor/

# Manually create directory
$ mkdir -p .conductor/learning
```

#### "CGO_ENABLED Required" Build Error

**Symptom**: Build fails with CGO-related errors

**Cause**: `mattn/go-sqlite3` requires C compiler (CGO)

**Solution:**
```bash
# Install C compiler (macOS)
$ xcode-select --install

# Install C compiler (Ubuntu/Debian)
$ sudo apt-get install build-essential

# Ensure CGO enabled
$ export CGO_ENABLED=1

# Rebuild
$ go build ./cmd/conductor
```

#### Agent Not Adapting Despite Failures

**Symptom**: Same agent used after multiple failures

**Solution:**
```bash
# Check configuration
$ cat .conductor/config.yaml | grep auto_adapt

# Enable auto-adaptation
learning:
  auto_adapt_agent: true  # Enable this
  min_failures_before_adapt: 2

# Check failure count
$ conductor learning show plan.md "Task 3"

# Check available agent history
$ conductor learning stats plan.md
```

#### Inter-Retry Swapping Not Working

**Symptom**: Agent not swapping during retries despite QC suggestions

**Solution:**
```bash
# Check swap configuration
$ cat .conductor/config.yaml | grep swap_during_retries

# Enable inter-retry swapping
learning:
  swap_during_retries: true  # Must be true

# Verify QC response includes suggested_agent
# Check task logs for QC response:
$ grep "suggested_agent" .conductor/logs/task-*.log

# Check QC agent returns proper JSON format
# QC must return structured response with suggested_agent field
```

#### QC JSON Parsing Errors

**Note (v2.8+)**: JSON schema enforcement via `--json-schema` flag guarantees valid JSON responses. Parsing errors should not occur in normal operation.

**Symptom**: "Schema enforcement failed" errors (rare)

**Solution:**
```bash
# Check QC logs for raw response
$ cat .conductor/logs/qc-*.log

# Verify Claude CLI version supports --json-schema
$ claude --help | grep json-schema

# Check if QC response structure matches schema
# Required fields: verdict (GREEN/RED/YELLOW), feedback
```

**Valid QC JSON Response:**
```json
{
  "verdict": "GREEN",
  "feedback": "Implementation complete",
  "issues": [],
  "recommendations": [],
  "should_retry": false
}
```

**Schema Enforcement (v2.8+)**: The `--json-schema` flag ensures Claude's response matches the expected structure at generation time, eliminating issues with:
- Non-JSON text before/after JSON
- Missing required fields
- Invalid field types
- Markdown code fences wrapping JSON

#### Context Not Loading for QC

**Symptom**: QC not seeing execution history

**Solution:**
```bash
# Check context loading configuration
learning:
  qc_reads_plan_context: true   # Load from plan file
  qc_reads_db_context: true     # Load from database
  max_context_entries: 10       # Increase if needed

# Verify plan file has execution_history section
$ grep "execution_history" plan.yaml

# Check database has execution records
$ sqlite3 .conductor/learning/executions.db \
  "SELECT COUNT(*) FROM task_executions;"

# Increase context limit if needed
learning:
  max_context_entries: 20  # Default is 10
```

#### Learning Commands Show No Data

**Symptom**: `learning stats` shows "No execution data found"

**Solution:**
```bash
# Verify database exists
$ ls -la .conductor/learning/conductor-learning.db

# Check from same directory as execution
$ cd /path/to/project
$ conductor learning stats plan.md

# Verify database not empty
$ sqlite3 .conductor/learning/conductor-learning.db "SELECT COUNT(*) FROM task_executions;"
```

### Multi-File Plan Issues

#### Cross-File Dependencies Not Found

**Symptoms:**
```
Error: Task 5 depends on Task 8, not found in setup.md
```

**Solution:**
```bash
# Check task numbers
grep "^## Task" setup.md features.md

# Update references if needed
```

#### WorktreeGroup Not Found

**Symptoms:**
```
Error: Task 3 references unknown group "backend-infra"
```

**Solution**: Define group in plan's YAML metadata:
```yaml
worktree_groups:
  - group_id: backend-infra
    description: Backend infrastructure
    execution_model: sequential
    isolation: strong
```

#### Dependency Cycle Across Files

**Symptoms:**
```
Error: Circular dependency detected: Task 1 → Task 4 → Task 2 → Task 1
```

**Solution**: Refactor cross-file dependencies:
```markdown
# Move Task 2 to features.md to avoid cycle
# Ensure: setup.md → features.md (one-way dependency)
```

### Debug Mode

#### Enabling Verbose Logging

```bash
# Console output
conductor run plan.md --verbose

# File logging (always enabled)
cat .conductor/logs/conductor-*.log
```

#### Examining Task Output

```bash
# View specific task log
cat .conductor/logs/task-1-setup-*.log

# View all task logs
ls -l .conductor/logs/task-*.log

# Search for errors
grep -i error .conductor/logs/*.log

# Search for RED verdicts
grep "RED" .conductor/logs/*.log
```

#### Testing Individual Tasks

Run Claude CLI directly to test task execution:

```bash
# Test task execution
claude -p "Your task prompt here" \
  --settings '{"disableAllHooks": true}' \
  --output-format json

# With specific agent
claude -p "use the code-implementation subagent to: implement feature" \
  --settings '{"disableAllHooks": true}' \
  --output-format json
```

#### Dry Run Testing

```bash
# See wave structure
conductor run plan.md --dry-run

# Check concurrency limits
conductor run plan.md --dry-run --max-concurrency 5

# Check dependency resolution
conductor validate plan.md
```

### Performance Tuning

#### Slow Execution

**Symptoms:**
- Execution takes much longer than expected
- Tasks running sequentially when they should be parallel

**Solutions:**

1. **Check wave grouping**:
```bash
conductor validate plan.md
```

2. **Increase concurrency**:
```bash
conductor run plan.md --max-concurrency 5
```

3. **Reduce dependencies**: Remove unnecessary dependencies to enable more parallelism

4. **Optimize task sizes**:
- Split large tasks (>15 min) into smaller tasks
- Combine tiny tasks (<2 min) into larger tasks

#### High Resource Usage

**Solutions:**

1. **Reduce concurrency**:
```bash
conductor run plan.md --max-concurrency 2
```

2. **Run fewer intensive tasks in parallel**: Restructure plan to avoid parallel resource-intensive tasks

### Frequently Asked Questions

**Q: Can I run multiple plans simultaneously?**
A: Not recommended. File locking prevents concurrent updates to same plan.

**Q: Can I pause/resume execution?**
A: Not currently supported. Use `--skip-completed` to resume after interruption.

**Q: How do I skip completed tasks?**
A: Mark tasks with `Status: completed` in plan file, then use `--skip-completed` flag.

**Q: Can I run specific tasks only?**
A: Not directly. Create new plan with only desired tasks.

**Q: How do I handle secrets in plans?**
A: Do not include secrets in plan files. Use environment variables or configuration files.

### Getting Help

#### Documentation

- [README](../README.md) - Project overview
- [CLAUDE.md](../CLAUDE.md) - Development guide and architecture

#### GitHub

- **Issues**: https://github.com/harrison/conductor/issues
- **Discussions**: https://github.com/harrison/conductor/discussions
- **Source Code**: https://github.com/harrison/conductor

#### Reporting Bugs

When reporting issues, include:

1. **Conductor version**:
```bash
conductor --version
```

2. **Error message** (full output):
```bash
conductor run plan.md 2>&1 | tee error.log
```

3. **Plan file** (sanitized if needed)

4. **Environment**:
   - OS: macOS, Linux, etc.
   - Go version: `go version`
   - Claude CLI version: `claude --version`

---

## Development Workflow

This section covers development practices, project structure, and testing guidelines for contributors.

### Building

**Using Make (Recommended):**
```bash
# Build binary to ~/bin/conductor (system-wide installation)
make build

# Use from anywhere (if ~/bin in PATH)
conductor --version
conductor run plan.md
```

**Auto-increment and build:**
These commands automatically increment the VERSION file and build the binary:
```bash
make build-patch   # 1.0.0 → 1.0.1 and build
make build-minor   # 1.0.0 → 1.1.0 and build
make build-major   # 1.0.0 → 2.0.0 and build
```

**Manual build (without Makefile):**
```bash
# Build to local directory
go build -o conductor ./cmd/conductor

# Or build and install to ~/bin
mkdir -p ~/bin
go build -o ~/bin/conductor ./cmd/conductor
```

**Verify the build:**
```bash
conductor --version
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
conductor run path/to/plan.md --verbose
```

Validate a plan:
```bash
conductor validate path/to/plan.md
```

### Project Structure

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

### Understanding the Codebase

#### Key Components

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

### Configuration

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

### Learning System Development

The learning system stores execution history in `.conductor/learning/`:

#### View Learning Data
```bash
conductor learning stats      # Statistics
conductor learning export     # Export to JSON
conductor learning clear      # Clear history (with confirmation)
```

#### Understanding Learning Flow
1. **Pre-Task Hook**: Loads historical failures and injects context
2. **Post-Task Hook**: Stores execution outcome
3. **Analyzer**: Identifies patterns from stored data

### Troubleshooting Development

#### Import issues
Module path is `github.com/harrison/conductor`:
```go
import "github.com/harrison/conductor/internal/executor"
```

#### Test failures
Run with verbose output:
```bash
go test -v ./internal/executor/
```

Check specific test:
```bash
go test ./internal/executor/ -run TestWaveCalculation
```

#### Build issues
Clean and rebuild:
```bash
make clean
make build
```

Verify Go version:
```bash
go version  # Should be 1.21+
```

### Submitting Changes

1. Create a feature branch
2. Make changes with tests
3. Run all tests: `make test`
4. Run linter: `make lint`
5. Format code: `make fmt`
6. Commit with clear message
7. Push and create pull request

See CONTRIBUTING.md for detailed guidelines.

---

## See Also

- [README](../README.md) - Project overview and quick start
- [CLAUDE.md](../CLAUDE.md) - Development guide, architecture, and TDD practices
- [GitHub Repository](https://github.com/harrison/conductor) - Source code and issue tracker
