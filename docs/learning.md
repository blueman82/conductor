# Adaptive Learning System

Conductor v2.0.0 introduces an intelligent adaptive learning system that learns from task execution history to improve future performance. The system automatically tracks failures, identifies patterns, and adapts agent selection to increase success rates over time.

## Table of Contents

- [Overview](#overview)
- [How It Works](#how-it-works)
- [Architecture](#architecture)
- [Configuration](#configuration)
- [CLI Commands](#cli-commands)
- [Usage Examples](#usage-examples)
- [Troubleshooting](#troubleshooting)
- [FAQ](#faq)

## Overview

### What Is It?

The adaptive learning system is a feedback mechanism that:

- **Records** every task execution attempt with success/failure data
- **Analyzes** failure patterns using keyword detection
- **Adapts** agent selection after repeated failures
- **Enhances** prompts with learned context from past executions
- **Improves** success rates over time through cross-run intelligence

### Benefits

- **Automatic Recovery**: System learns which agents work best for specific tasks
- **Reduced Manual Intervention**: No need to manually adjust plans after failures
- **Pattern Recognition**: Identifies common failure types (compilation errors, test failures, etc.)
- **Historical Intelligence**: Leverages execution history across multiple runs
- **Graceful Degradation**: Learning failures never break task execution

### Key Features

- SQLite-based execution history database
- Three integration hooks: pre-task, QC-review, post-task
- Pattern detection for 6+ common failure types
- Session and run number tracking
- Four CLI observability commands
- Configurable adaptation thresholds
- Project-local learning data (`.conductor/learning/`)

## How It Works

### Execution Pipeline with Learning

```
1. Pre-Task Hook
   ├─ Query execution history for this task
   ├─ Analyze failure patterns and agent performance
   ├─ Adapt agent if 2+ failures detected (configurable)
   └─ Enhance prompt with learning context

2. Task Execution
   └─ Execute task with (potentially adapted) agent

3. QC-Review Hook
   ├─ Capture QC verdict (GREEN/RED/YELLOW)
   └─ Extract failure patterns from output

4. Post-Task Hook
   ├─ Record execution to database
   ├─ Store: agent used, QC verdict, patterns, timing
   └─ Update run statistics
```

### Failure Pattern Detection

The system detects these common patterns:

| Pattern | Keywords Detected | Example Scenario |
|---------|-------------------|------------------|
| `compilation_error` | "compilation failed", "syntax error" | Code fails to compile |
| `test_failure` | "test failed", "assertion failed" | Tests don't pass |
| `dependency_missing` | "module not found", "import error" | Missing dependencies |
| `permission_error` | "permission denied", "access denied" | File/directory access issues |
| `timeout` | "timeout", "exceeded deadline" | Long-running operations |
| `runtime_error` | "runtime error", "panic", "exception" | Runtime failures |

### Agent Adaptation Logic

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

### Session Management

Each conductor run gets a unique session ID:
- Format: `session-YYYYMMDD-HHMMSS`
- Example: `session-20250112-143045`
- Used for grouping executions and tracking run numbers

Run numbers increment per plan file:
- First execution of `plan.md`: run 1
- Second execution of `plan.md`: run 2
- Independent tracking per plan file

## Architecture

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

-- Schema version tracking
CREATE TABLE schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Component Architecture

```
┌─────────────────────────────────────────────────┐
│                 Orchestrator                     │
│  - Initializes learning store                   │
│  - Generates session ID                         │
│  - Tracks run number                            │
└─────────────────┬───────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────┐
│              Task Executor                       │
│  ┌───────────────────────────────────────────┐ │
│  │ Pre-Task Hook                             │ │
│  │ - Analyze failures                        │ │
│  │ - Adapt agent if needed                   │ │
│  │ - Enhance prompt                          │ │
│  └───────────────────────────────────────────┘ │
│                                                  │
│  ┌───────────────────────────────────────────┐ │
│  │ Execute Task                              │ │
│  │ - Invoke claude CLI with agent           │ │
│  └───────────────────────────────────────────┘ │
│                                                  │
│  ┌───────────────────────────────────────────┐ │
│  │ QC-Review Hook                            │ │
│  │ - Capture verdict                         │ │
│  │ - Extract patterns                        │ │
│  └───────────────────────────────────────────┘ │
│                                                  │
│  ┌───────────────────────────────────────────┐ │
│  │ Post-Task Hook                            │ │
│  │ - Record execution                        │ │
│  │ - Store patterns and metrics             │ │
│  └───────────────────────────────────────────┘ │
└─────────────────┬───────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────┐
│           Learning Store (SQLite)               │
│  - RecordExecution()                            │
│  - GetExecutionHistory()                        │
│  - AnalyzeFailures()                            │
│  - GetRunCount()                                │
└─────────────────────────────────────────────────┘
```

### Hook Integration Points

**1. Pre-Task Hook** (`internal/executor/task.go`)
- **When**: Before task execution
- **Purpose**: Analyze history and adapt strategy
- **Actions**:
  - Query past executions for this task
  - Count failures with current agent
  - Suggest alternative agent if threshold reached
  - Enhance prompt with learned context

**2. QC-Review Hook** (`internal/executor/task.go`)
- **When**: After quality control review
- **Purpose**: Extract patterns from output
- **Actions**:
  - Parse QC verdict (GREEN/RED/YELLOW)
  - Scan output for failure patterns
  - Store patterns in task metadata

**3. Post-Task Hook** (`internal/executor/task.go`)
- **When**: After task completion (success or failure)
- **Purpose**: Record execution history
- **Actions**:
  - Build execution record
  - Include patterns from QC-review hook
  - Write to learning database
  - Log success/failure

## Configuration

### Default Configuration

```yaml
learning:
  # Enable or disable the learning system (default: true)
  enabled: true

  # Database path (default: .conductor/learning)
  db_path: .conductor/learning

  # Auto-adapt agent selection based on learned patterns (default: false)
  auto_adapt_agent: false

  # Enhance prompts with learned context (default: true)
  enhance_prompts: true

  # Minimum failures before adapting strategy (default: 2)
  min_failures_before_adapt: 2

  # Days to keep execution history (default: 90)
  # Set to 0 to keep forever
  keep_executions_days: 90

  # Maximum execution records per task (default: 100)
  max_executions_per_task: 100
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Master switch for learning system |
| `db_path` | string | `.conductor/learning` | Path to SQLite database |
| `auto_adapt_agent` | bool | `false` | Automatically switch agents on failures |
| `enhance_prompts` | bool | `true` | Add learned context to prompts |
| `min_failures_before_adapt` | int | `2` | Failure threshold before adapting |
| `keep_executions_days` | int | `90` | Days to retain history (0 = forever) |
| `max_executions_per_task` | int | `100` | Max records per task |

### Configuration File Location

Place configuration at `.conductor/config.yaml` in your project root:

```yaml
# .conductor/config.yaml

# Execution settings
max_concurrency: 3
timeout: 30m

# Learning settings
learning:
  enabled: true
  auto_adapt_agent: true
  min_failures_before_adapt: 3  # More conservative
```

### Disabling Learning

To completely disable the learning system:

```yaml
learning:
  enabled: false
```

Or via environment (for testing):

```bash
# Learning system respects enabled: false in config
conductor run plan.md
```

## CLI Commands

Conductor provides four commands for observing and managing learning data:

### `conductor learning stats`

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
  4. Task 3 - backend-developer - RED - 2025-01-12 14:15
  5. Task 2 - golang-pro - GREEN - 2025-01-12 14:10
```

### `conductor learning show`

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

Statistics:
  Total attempts: 2
  Success rate: 50%
  Agents tried: backend-developer, golang-pro
```

### `conductor learning clear`

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

$ conductor learning clear --all
Warning: This will delete ALL learning data
Are you sure? (y/n): y
✓ Deleted 47 execution records across all plans
```

### `conductor learning export`

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
  },
  ...
]

# Export to CSV file
$ conductor learning export my-plan.md --format csv --output data.csv
✓ Exported 15 records to data.csv

# View CSV
$ cat data.csv
task_number,agent,success,timestamp,duration_seconds,patterns
Task 1,golang-pro,true,2025-01-12T14:30:45Z,45,""
Task 2,backend-developer,false,2025-01-12T14:25:12Z,62,"compilation_error,test_failure"
```

## Usage Examples

### Basic Usage (Learning Enabled by Default)

```bash
# First run - learning records execution
$ conductor run plan.md

# Second run - learning adapts if there were failures
$ conductor run plan.md --skip-completed

# View statistics
$ conductor learning stats plan.md
```

### Scenario 1: Automatic Agent Adaptation

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

### Scenario 2: Pattern-Based Learning

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

### Scenario 3: Cross-Run Intelligence

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

### Scenario 4: Exporting for Analysis

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

### Scenario 5: Disabling Learning for Specific Run

```yaml
# Create temporary config
# .conductor/config-no-learning.yaml
learning:
  enabled: false
```

```bash
# Run without learning (for testing)
$ conductor run plan.md --config .conductor/config-no-learning.yaml
```

## Troubleshooting

### Learning Database Not Created

**Symptom**: No `.conductor/learning/` directory created

**Possible causes:**
1. Learning disabled in config
2. Permission issues creating directory
3. Invalid `db_path` configuration

**Solution:**
```bash
# Check config
$ cat .conductor/config.yaml | grep -A5 learning

# Verify learning enabled
learning:
  enabled: true  # Must be true

# Check permissions
$ ls -ld .conductor/
# Should show write permissions

# Manually create directory
$ mkdir -p .conductor/learning
```

### "CGO_ENABLED Required" Build Error

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

### Agent Not Adapting Despite Failures

**Symptom**: Same agent used after multiple failures

**Possible causes:**
1. `auto_adapt_agent` is `false`
2. Failure threshold not reached
3. No alternative agents with success history

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
# Verify >= 2 failures with same agent

# Check available agent history
$ conductor learning stats plan.md
# Verify other agents have success records
```

### Database Locked Errors

**Symptom**: "database is locked" errors during execution

**Cause**: Multiple conductor instances accessing same database

**Solution:**
```bash
# Wait for other instances to complete
$ ps aux | grep conductor

# Or use different database per instance (worktrees)
$ conductor run plan.md --config worktree1/config.yaml
```

### Old Data Accumulating

**Symptom**: Database growing too large

**Solution:**
```yaml
# Configure data retention
learning:
  keep_executions_days: 30  # Shorter retention
  max_executions_per_task: 50  # Lower limit

# Or manually clear old data
$ conductor learning clear --all
```

### Learning Commands Show No Data

**Symptom**: `learning stats` shows "No execution data found"

**Possible causes:**
1. Learning disabled during execution
2. Database path mismatch
3. Different working directory

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

## FAQ

### General Questions

**Q: Does learning work with all plan formats?**
A: Yes, learning works with both Markdown and YAML plan formats.

**Q: Is learning data version controlled?**
A: No, `.conductor/learning/` is in `.gitignore`. Learning data is project-local and should not be committed.

**Q: Does learning work with multi-file plans?**
A: Yes, learning tracks execution history per plan file, so multi-file plans maintain separate learning contexts.

**Q: Can I share learning data across team?**
A: Not recommended. Learning data is optimized for individual developer environments. Each developer builds their own learning context.

### Performance Questions

**Q: Does learning slow down execution?**
A: Minimal impact. Hook operations are async where possible, and database queries are optimized with indexes.

**Q: How large does the database get?**
A: Depends on configuration. With defaults (90 days, 100 executions/task), expect ~10-50MB for active projects.

**Q: Can I use a different database backend?**
A: Currently SQLite only. The design could support other backends, but SQLite is ideal for local CLI tools.

### Configuration Questions

**Q: What's the difference between `auto_adapt_agent` and `enhance_prompts`?**
A:
- `auto_adapt_agent`: Automatically changes the agent after failures
- `enhance_prompts`: Adds learned context to prompts but keeps same agent

Both can be enabled simultaneously.

**Q: Should I enable `auto_adapt_agent` by default?**
A: Recommended to start with `false` (manual control) and enable after understanding behavior.

**Q: How do I reset learning for a specific task?**
A: Currently requires clearing all data for the plan file:
```bash
conductor learning clear plan.md
```

### Troubleshooting Questions

**Q: Learning failures are breaking my execution**
A: This shouldn't happen. Learning uses graceful degradation. Check logs for the specific error and file a bug report.

**Q: Can I view the raw SQLite database?**
A: Yes:
```bash
sqlite3 .conductor/learning/conductor-learning.db
.schema  # View schema
SELECT * FROM task_executions LIMIT 10;  # Query data
```

**Q: How do I completely disable learning?**
A: Set `enabled: false` in config:
```yaml
learning:
  enabled: false
```

**Q: Does learning work in CI/CD environments?**
A: Yes, but learning database won't persist between runs unless you configure persistent storage. For CI/CD, you may want to disable learning:
```bash
# Set in CI config
learning:
  enabled: false
```

### Advanced Questions

**Q: Can I customize pattern detection keywords?**
A: Not currently configurable. Pattern keywords are hard-coded in `internal/learning/analysis.go`. Future versions may support custom patterns.

**Q: How does conductor choose alternative agents?**
A: Query finds agents that succeeded on similar tasks (same plan file) and haven't been tried yet for this task.

**Q: Can I export learning data for ML analysis?**
A: Yes! Use `conductor learning export --format json` to get structured data for external analysis.

**Q: Does learning work with custom agents?**
A: Yes, learning tracks any agent used, including custom agents from `~/.claude/agents/`.

**Q: Can I manually record executions?**
A: Not via CLI currently. The learning system only records during conductor-managed execution.

## See Also

- [Configuration Guide](../CLAUDE.md#configuration) - Full configuration reference
- [Usage Guide](usage.md) - CLI command reference
- [Architecture Overview](../CLAUDE.md#architecture-overview) - System architecture
- [Troubleshooting](troubleshooting.md) - General troubleshooting guide
