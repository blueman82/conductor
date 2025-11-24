# Agent Watch - Behavioral Observability for Claude Code Agents

Agent Watch provides comprehensive observability into Claude Code agent behavior, enabling analysis of tool usage patterns, bash commands, file operations, errors, and performance metrics.

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Command Reference](#command-reference)
- [Filtering](#filtering)
- [Export Formats](#export-formats)
- [Analytics Features](#analytics-features)
- [Configuration](#configuration)
- [Performance Tuning](#performance-tuning)
- [Troubleshooting](#troubleshooting)

## Overview

Agent Watch extracts and analyzes behavioral data from Claude Code session JSONL files stored in `~/.claude/projects/`. It provides:

- **Session Analysis**: Track individual agent sessions with duration, success rates, and error counts
- **Tool Usage Metrics**: Analyze which tools are used most frequently and their success rates
- **Bash Command Patterns**: Identify common bash commands and failure patterns
- **File Operation Tracking**: Monitor read/write/edit operations across sessions
- **Error Analysis**: Detect and categorize errors for debugging
- **Performance Scoring**: Score and rank agents based on multiple dimensions
- **Failure Prediction**: Predict task failures based on historical patterns
- **Pattern Detection**: Identify anomalies and behavioral clusters

## Installation

Agent Watch is included in Conductor v2.7+. No additional installation required.

```bash
# Verify installation
conductor observe --help
```

## Quick Start

```bash
# Interactive mode - select project from menu
conductor observe

# View statistics for all projects
conductor observe stats

# Stream real-time activity
conductor observe stream

# Export metrics to JSON
conductor observe export --format json --output metrics.json

# Filter by project and show only errors
conductor observe stats --project myapp --errors-only
```

## Command Reference

### `conductor observe`

Root command with interactive project selection.

```bash
conductor observe [flags]
```

**Global Flags (available to all subcommands):**

| Flag | Short | Description |
|------|-------|-------------|
| `--project` | `-p` | Filter by project name |
| `--session` | `-s` | Filter by session ID |
| `--filter-type` | | Filter by type: tool, bash, file |
| `--errors-only` | | Show only errors |
| `--time-range` | | Time range: 1h, 24h, 7d, 30d, today, yesterday |

### `conductor observe project [name]`

View project-level behavioral metrics.

```bash
conductor observe project myapp
conductor observe project --time-range 7d
```

**Output includes:**
- Session count and success rate
- Tool usage patterns
- File operation statistics
- Error rates and common issues
- Agent performance comparison

### `conductor observe session [id]`

Analyze a specific agent session.

```bash
conductor observe session abc123
```

**Output includes:**
- Session metadata (duration, status, agent used)
- Tool execution timeline
- Bash command history
- File operations
- Token usage and cost
- Errors and issues

### `conductor observe tools`

Analyze tool usage patterns.

```bash
conductor observe tools --project myapp
conductor observe tools --errors-only
```

**Output includes:**
- Tool execution counts
- Success and error rates per tool
- Average execution duration
- Most/least used tools
- Tool usage trends over time

### `conductor observe bash`

Analyze bash command patterns.

```bash
conductor observe bash --time-range 24h
conductor observe bash --errors-only
```

**Output includes:**
- Command frequency
- Exit codes and success rates
- Most common commands
- Failed commands
- Output patterns

### `conductor observe files`

Analyze file operation patterns.

```bash
conductor observe files --project myapp
conductor observe files --filter-type file
```

**Output includes:**
- Read/Write/Edit/Delete counts
- Most accessed files
- File size distributions
- Operation success rates
- File modification patterns

### `conductor observe errors`

Analyze error patterns and issues.

```bash
conductor observe errors --time-range 7d
conductor observe errors --project myapp
```

**Output includes:**
- Error frequency by type
- Common error patterns
- Error rates over time
- Tool/bash errors breakdown
- Failed operations summary

### `conductor observe stats`

Display summary statistics.

```bash
conductor observe stats
conductor observe stats --project myapp
```

**Output includes:**
- Total sessions and agents
- Success and error rates
- Average duration
- Token usage and cost
- Top tools by usage
- Agent performance breakdown

### `conductor observe stream`

Stream real-time activity.

```bash
conductor observe stream
conductor observe stream --project myapp
```

Polls the database every 2 seconds for new sessions and displays them as they occur.

### `conductor observe export`

Export behavioral data to file.

```bash
conductor observe export --format json --output metrics.json
conductor observe export --format markdown --output report.md
conductor observe export --format csv --output data.csv
conductor observe export --format json  # Output to stdout
```

**Flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--format` | `-f` | json | Export format: json, markdown (or md), csv |
| `--output` | `-o` | (stdout) | Output file path |

## Filtering

### Time Range Formats

```bash
# Relative durations
--time-range 1h      # Last 1 hour
--time-range 24h     # Last 24 hours
--time-range 7d      # Last 7 days
--time-range 30d     # Last 30 days
--time-range 15m     # Last 15 minutes

# Keywords
--time-range today      # Since midnight today
--time-range yesterday  # Since midnight yesterday

# ISO dates
--time-range 2025-01-15
--time-range 2025-01-15T14:30:00Z
```

### Filter Types

```bash
# Filter by event type
--filter-type tool   # Show only tool executions
--filter-type bash   # Show only bash commands
--filter-type file   # Show only file operations

# Show only errors
--errors-only

# Combine filters (AND logic)
conductor observe tools --project myapp --errors-only --time-range 24h
```

### Search

The `--session` flag also acts as a general search filter:

```bash
conductor observe stats --session "migration"  # Search in session IDs, agents, status
```

## Export Formats

### JSON

Structured data with full metrics:

```json
{
  "total_sessions": 42,
  "success_rate": 0.85,
  "error_rate": 0.15,
  "average_duration": "5m30s",
  "total_cost": 0.0234,
  "tool_executions": [...],
  "bash_commands": [...],
  "file_operations": [...],
  "token_usage": {...},
  "agent_performance": {...}
}
```

### Markdown

Human-readable report with tables:

```markdown
# Behavioral Metrics Report

## Summary
- **Total Sessions**: 42
- **Success Rate**: 85.00%
- **Error Rate**: 15.00%

## Tool Executions
| Tool Name | Count | Success Rate | Error Rate |
|-----------|-------|--------------|------------|
| Read      | 150   | 98.67%       | 1.33%      |
| Write     | 45    | 95.56%       | 4.44%      |
```

### CSV

Machine-readable format for analysis:

```csv
Type,Name,Value
Summary,TotalSessions,42
Summary,SuccessRate,0.8500
ToolExecution,Read,Count:150|SuccessRate:0.9867
```

## Analytics Features

### Pattern Detection

Agent Watch automatically detects:

- **Tool Sequences**: Common sequences of tool executions (n-grams)
- **Bash Patterns**: Frequently used command patterns
- **Anomalies**: Statistical outliers in duration, error rates, and tool usage
- **Behavior Clusters**: Groups of similar sessions using k-means clustering

### Failure Prediction

Predicts task failure probability based on:

- Historical tool failure rates
- Similar session outcomes (Jaccard similarity)
- High-risk tool combinations
- Session complexity metrics

**Output includes:**
- Probability (0.0-1.0)
- Confidence level
- Risk factors identified
- Mitigation recommendations

### Performance Scoring

Multi-dimensional agent scoring:

| Dimension | Weight | Description |
|-----------|--------|-------------|
| Success | 40% | Task completion rate |
| Cost Efficiency | 25% | Token usage relative to baseline |
| Speed | 20% | Duration relative to baseline |
| Error Recovery | 15% | Ability to recover from errors |

Scores are normalized to 0-1 scale and adjusted for sample size confidence.

### Agent Ranking

Rank agents globally or within domains:

- **Global**: Overall composite score ranking
- **Domain-specific**: backend, frontend, devops, testing, security, general

## Configuration

### Learning Database

Agent Watch reads from the same database used by Conductor's adaptive learning system:

```yaml
# .conductor/config.yaml
learning:
  enabled: true
  db_path: ".conductor/learning/executions.db"
```

### Project Discovery

Projects are discovered from:

```
~/.claude/projects/
├── myapp/
│   └── sessions/
│       ├── session1.jsonl
│       └── session2.jsonl
└── another-project/
    └── sessions/
```

## Performance Tuning

### Large Datasets

For projects with many sessions:

```bash
# Filter to reduce data volume
conductor observe stats --time-range 7d

# Export for offline analysis
conductor observe export --format json -o data.json --time-range 30d
```

### Database Optimization

The SQLite database uses indexes on:
- `task_executions.plan_file`
- `behavioral_sessions.session_start`
- `behavioral_sessions.task_execution_id`

### Memory Considerations

- Pagination is automatic for large result sets (50 items per page)
- Streaming mode polls every 2 seconds to minimize memory usage
- Export writes to temporary file first (atomic write pattern)

## Troubleshooting

### No Projects Found

```
Error: no projects found in ~/.claude/projects
```

**Solution**: Ensure Claude Code has been used and has generated session files.

### Database Connection Errors

```
Error: open learning store: ...
```

**Solution**: Check that `.conductor/learning/executions.db` exists and is not corrupted.

### Invalid Time Range

```
Error: invalid time range format: xyz
```

**Solution**: Use supported formats:
- Relative: `1h`, `24h`, `7d`, `30d`, `15m`
- Keywords: `today`, `yesterday`
- ISO: `2025-01-15`, `2025-01-15T14:30:00Z`

### No Data for Filters

If filtering returns no results:
- Remove filters one at a time to identify which is too restrictive
- Check time range covers relevant period
- Verify project name matches exactly

### Export Permission Errors

```
Error: failed to write temporary file: permission denied
```

**Solution**: Ensure write permissions to output directory or use stdout.

## Integration with QC

Agent Watch enhances Quality Control reviews by:

1. **Context Injection**: QC agents receive behavioral metrics when reviewing tasks
2. **Pattern Warnings**: QC is informed of high-risk tool combinations
3. **Historical Context**: Past failures for similar tasks inform review criteria

Enable in config:

```yaml
learning:
  qc_reads_plan_context: true
  qc_reads_db_context: true
```

## API Reference

### Data Structures

#### Session

```go
type Session struct {
    ID          string    `json:"id"`
    Project     string    `json:"project"`
    Timestamp   time.Time `json:"timestamp"`
    Status      string    `json:"status"`
    AgentName   string    `json:"agent_name"`
    Duration    int64     `json:"duration"`
    Success     bool      `json:"success"`
    ErrorCount  int       `json:"error_count"`
}
```

#### BehavioralMetrics

```go
type BehavioralMetrics struct {
    TotalSessions    int
    SuccessRate      float64
    AverageDuration  time.Duration
    TotalCost        float64
    ToolExecutions   []ToolExecution
    BashCommands     []BashCommand
    FileOperations   []FileOperation
    TokenUsage       TokenUsage
    ErrorRate        float64
    TotalErrors      int
    AgentPerformance map[string]int
}
```

#### FilterCriteria

```go
type FilterCriteria struct {
    Search     string
    EventType  string    // tool, bash, file
    ErrorsOnly bool
    Since      time.Time
    Until      time.Time
}
```

### Internal Packages

- `internal/behavioral/` - Core data models and logic
- `internal/cmd/observe*.go` - CLI command implementations
- `internal/learning/` - Database storage integration

## See Also

- [Usage Examples](examples/agent-watch-usage.md)
- [Filtering Guide](examples/agent-watch-filtering.md)
- [Analytics Features](examples/agent-watch-analytics.md)
- [CLAUDE.md](../CLAUDE.md) - Project documentation
