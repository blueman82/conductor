# Agent Watch Filtering Guide

Comprehensive guide to filtering behavioral data in Agent Watch.

## Filter Types Overview

| Filter | Flag | Description |
|--------|------|-------------|
| Project | `--project`, `-p` | Filter by project name |
| Session | `--session`, `-s` | Filter by session ID or search term |
| Event Type | `--filter-type` | Filter by event: tool, bash, file |
| Errors Only | `--errors-only` | Show only items with errors |
| Time Range | `--time-range` | Filter by time period |

## Filter Logic

All filters use **AND logic**. When multiple filters are applied, results must match ALL criteria.

```bash
# Example: Results must match ALL of:
# - Project is "myapp"
# - Has errors
# - Within last 24 hours
conductor observe tools --project myapp --errors-only --time-range 24h
```

## Time Range Filtering

### Relative Durations

```bash
# Minutes
--time-range 15m   # Last 15 minutes
--time-range 30m   # Last 30 minutes
--time-range 60m   # Last 60 minutes

# Hours
--time-range 1h    # Last 1 hour
--time-range 6h    # Last 6 hours
--time-range 12h   # Last 12 hours
--time-range 24h   # Last 24 hours

# Days
--time-range 1d    # Last 1 day
--time-range 7d    # Last 7 days
--time-range 14d   # Last 14 days
--time-range 30d   # Last 30 days
```

### Keyword Shortcuts

```bash
# Today (since midnight)
--time-range today

# Yesterday (full day)
--time-range yesterday
```

### ISO Date Formats

```bash
# Date only (starts at 00:00:00)
--time-range 2025-01-15

# Date with time
--time-range 2025-01-15T14:30:00

# Full ISO 8601 with timezone
--time-range 2025-01-15T14:30:00Z
```

### Time Range Examples

```bash
# All activity from last week
conductor observe stats --time-range 7d

# Activity since specific date
conductor observe errors --time-range 2025-01-01

# Today's sessions only
conductor observe stats --time-range today

# Compare yesterday vs today
conductor observe stats --time-range yesterday  # Run first
conductor observe stats --time-range today       # Compare
```

## Project Filtering

### Basic Usage

```bash
# Filter by exact project name
conductor observe stats --project myapp
conductor observe tools -p backend-api
```

### Finding Projects

```bash
# Interactive mode shows available projects
conductor observe

# This lists projects with session counts and sizes
# Use the project name from the list for filtering
```

### Project Name Matching

Project names are matched against the `plan_file` path in the database using LIKE patterns:

```bash
# Matches any plan_file containing "myapp"
--project myapp
```

## Event Type Filtering

Filter results by event category:

```bash
# Show only tool executions
conductor observe stats --filter-type tool

# Show only bash commands
conductor observe bash --filter-type bash

# Show only file operations
conductor observe files --filter-type file
```

### Event Type Behavior

| Flag Value | Included Data |
|------------|---------------|
| `tool` | Tool executions (Read, Write, Edit, Bash, etc.) |
| `bash` | Bash command executions |
| `file` | File operations (read, write, edit, delete) |
| (empty) | All event types |

### Combining with Subcommands

```bash
# These are equivalent
conductor observe tools --filter-type tool
conductor observe tools

# Filter file subcommand to only errors
conductor observe files --errors-only
```

## Error Filtering

### Basic Error Filter

```bash
# Show only sessions/items with errors
conductor observe stats --errors-only

# Show only failed bash commands
conductor observe bash --errors-only

# Show only failed file operations
conductor observe files --errors-only
```

### Error Detection Logic

| Data Type | Error Condition |
|-----------|-----------------|
| Sessions | `error_count > 0` |
| Tool Executions | `total_errors > 0` |
| Bash Commands | `success == false` |
| File Operations | `success == false` |

### Combining Error Filters

```bash
# Failed bash commands in the last 24 hours
conductor observe bash --errors-only --time-range 24h

# Project-specific errors
conductor observe errors --project myapp --time-range 7d

# Export errors for analysis
conductor observe export --errors-only --format json --output errors.json
```

## Search Filtering

The `--session` flag doubles as a search filter:

```bash
# Search by session ID
conductor observe stats --session abc123

# Search in project names, agent names, status
conductor observe stats --session "migration"
```

### Search Scope

Search is case-insensitive and matches:
- Session ID
- Project name
- Agent name
- Status

```bash
# Find all sessions with "test" in any field
conductor observe stats --session test

# Find sessions by specific agent
conductor observe stats --session "backend-developer"
```

## Combined Filter Examples

### Development Debugging

```bash
# Recent errors in a specific project
conductor observe errors --project myapp --time-range 1h

# Failed bash commands with output
conductor observe bash --errors-only --project myapp
```

### Performance Analysis

```bash
# Tool performance over time
conductor observe tools --time-range 7d --project myapp

# File operation patterns
conductor observe files --time-range 30d
```

### Weekly Reports

```bash
# Export weekly summary
conductor observe export \
  --format markdown \
  --time-range 7d \
  --output weekly-report.md

# Export errors only
conductor observe export \
  --format json \
  --errors-only \
  --time-range 7d \
  --output weekly-errors.json
```

### Cross-Project Comparison

```bash
# Compare projects side by side
echo "=== Backend ===" && conductor observe stats --project backend --time-range 7d
echo "=== Frontend ===" && conductor observe stats --project frontend --time-range 7d
echo "=== API ===" && conductor observe stats --project api --time-range 7d
```

## Filter Validation

### Time Range Validation

Invalid time ranges return errors:

```bash
conductor observe stats --time-range "invalid"
# Error: invalid time range format: invalid
```

Valid formats are automatically detected:
- Relative: `Nh` (hours), `Nd` (days), `Nm` (minutes)
- Keywords: `today`, `yesterday`
- ISO: `YYYY-MM-DD`, `YYYY-MM-DDTHH:MM:SS`

### Event Type Validation

Invalid event types return errors:

```bash
conductor observe stats --filter-type "invalid"
# Error: invalid event type 'invalid': must be one of: tool, bash, file
```

### Time Range Ordering

Since time cannot be after Until time:

```go
// Validation automatically checks:
if since.After(until) {
    return error
}
```

## Performance Considerations

### Large Result Sets

Filters reduce data volume and improve performance:

```bash
# Without filters (slow on large datasets)
conductor observe stats

# With filters (faster)
conductor observe stats --time-range 7d --project myapp
```

### Pagination

Results are automatically paginated (50 items per page) regardless of filters.

### Export Considerations

```bash
# Export full dataset (may be large)
conductor observe export --format json

# Export filtered subset (smaller, faster)
conductor observe export --format json --time-range 7d --project myapp
```

## Troubleshooting Filters

### No Results

If filters return no data:

1. Remove filters one at a time
2. Check time range covers relevant period
3. Verify project name is correct
4. Confirm data exists in database

```bash
# Broaden search progressively
conductor observe stats --project myapp --time-range 30d
conductor observe stats --project myapp
conductor observe stats
```

### Unexpected Results

If results don't match expectations:

1. Check filter logic (AND, not OR)
2. Verify time range format
3. Check case sensitivity of search

```bash
# Debug by viewing unfiltered data first
conductor observe stats

# Then add filters incrementally
conductor observe stats --project myapp
conductor observe stats --project myapp --time-range 7d
conductor observe stats --project myapp --time-range 7d --errors-only
```

## Filter API Reference

### FilterCriteria Structure

```go
type FilterCriteria struct {
    Search     string    // Case-insensitive search
    EventType  string    // "tool", "bash", "file"
    ErrorsOnly bool      // true = errors only
    Since      time.Time // Start of range
    Until      time.Time // End of range
}
```

### Validation Function

```go
func (fc *FilterCriteria) Validate() error {
    // Checks:
    // - Since is not after Until
    // - EventType is valid (tool, bash, file)
    // - Returns nil if valid
}
```

### Time Parsing

```go
func ParseTimeRange(timeRange string) (time.Time, error) {
    // Handles:
    // - "1h", "24h", "7d", "30d", "15m"
    // - "today", "yesterday"
    // - "2025-01-15", "2025-01-15T14:30:00Z"
}
```
