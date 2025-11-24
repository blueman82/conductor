# Agent Watch (observe) CLI Usage Examples

This document provides practical examples of using the `conductor observe` command for analyzing Claude Code agent behavior.

## Quick Start

```bash
# Interactive mode - shows project selection menu
conductor observe

# List available projects
conductor observe --project myapp stats

# View real-time activity
conductor observe stream

# Export metrics to JSON
conductor observe export --format json --output metrics.json
```

## Common Use Cases

### 1. Project Overview

View aggregate statistics for a specific project:

```bash
conductor observe --project conductor stats
```

Output:
```
=== SUMMARY STATISTICS ===

Total Sessions:      42
Total Agents:        8
Total Operations:    1,247
Success Rate:        88.1%
Error Rate:          11.9%
Average Duration:    4m 32s
Total Cost:          $12.45

Total Input Tokens:  45,678
Total Output Tokens: 23,456
Total Tokens:        69,134

--- Top Tools (by usage) ---
 1. Read                  Count:  345  Success:  98.8%  Errors:   0.4%
 2. Write                 Count:  187  Success:  96.3%  Errors:   3.7%
 3. Edit                  Count:  234  Success:  99.1%  Errors:   0.9%
 4. Bash                  Count:  156  Success:  92.3%  Errors:   7.7%
 5. Grep                  Count:   89  Success: 100.0%  Errors:   0.0%

--- Agent Performance ---
 1. backend-developer              Success Count: 15
 2. test-automator                 Success Count: 12
 3. golang-pro                     Success Count: 8
```

### 2. Filter by Event Type

Analyze specific types of operations:

```bash
# View only tool executions
conductor observe --filter-type tool stats

# View only bash commands
conductor observe --filter-type bash stats

# View only file operations
conductor observe --filter-type file stats
```

### 3. Error Analysis

Focus on errors to identify issues:

```bash
# Show only errors
conductor observe --errors-only stats

# Show errors in last 24 hours
conductor observe --errors-only --time-range 24h stats

# Export errors to JSON for analysis
conductor observe --errors-only export --format json --output errors.json
```

### 4. Time Range Filtering

Analyze behavior over specific time periods:

```bash
# Last hour
conductor observe --time-range 1h stats

# Last 24 hours
conductor observe --time-range 24h stats

# Last 7 days
conductor observe --time-range 7d stats

# Last 30 days
conductor observe --time-range 30d stats

# Today only
conductor observe --time-range today stats

# Yesterday only
conductor observe --time-range yesterday stats
```

### 5. Real-Time Streaming

Watch agent activity as it happens:

```bash
# Stream all activity
conductor observe stream

# Stream for specific project
conductor observe --project myapp stream
```

Output:
```
Real-time streaming mode
Watching for new sessions... (Press Ctrl+C to stop)

[14:32:15] SUCCESS | Task: Implement user auth | Agent: backend-developer | Duration: 3m 45s
[14:35:22] FAILED  | Task: Add tests | Agent: test-automator | Duration: 1m 12s
[14:37:08] SUCCESS | Task: Fix linting | Agent: golang-pro | Duration: 45s
```

### 6. Export for External Analysis

Export data in various formats:

```bash
# Export to JSON (pretty-printed)
conductor observe export --format json --output report.json

# Export to Markdown for documentation
conductor observe export --format markdown --output report.md

# Export to CSV for spreadsheet analysis
conductor observe export --format csv --output report.csv

# Export filtered data
conductor observe --project myapp --errors-only export --format json --output errors.json

# Export to stdout (pipe to other tools)
conductor observe export --format json | jq '.tool_executions[] | select(.error_rate > 0.1)'
```

### 7. Search Sessions

Find specific sessions or operations:

```bash
# Search by session ID
conductor observe --session abc123 stats

# Search by project name
conductor observe --project conductor stats

# Combined filters
conductor observe --project myapp --session xyz789 --errors-only stats
```

### 8. Subcommand Examples

#### Project Analysis

```bash
# View project-level metrics
conductor observe project conductor

# Compare multiple projects
conductor observe project myapp
conductor observe project conductor
conductor observe project api-service
```

#### Session Analysis

```bash
# Analyze specific session
conductor observe session abc123-def456

# View session with filters
conductor observe --session abc123 --errors-only stats
```

#### Tool Usage Analysis

```bash
# View tool usage patterns
conductor observe tools

# Filter by project
conductor observe --project myapp tools

# Export tool metrics
conductor observe tools export --format json --output tools.json
```

#### Bash Command Analysis

```bash
# View bash command statistics
conductor observe bash

# Filter for failures
conductor observe --errors-only bash

# Analyze specific time range
conductor observe --time-range 24h bash
```

#### File Operation Analysis

```bash
# View file operations
conductor observe files

# Filter by operation type
conductor observe --filter-type file files

# Export file metrics
conductor observe files export --format csv --output file-ops.csv
```

#### Error Pattern Analysis

```bash
# View all errors
conductor observe errors

# Recent errors
conductor observe --time-range 24h errors

# Project-specific errors
conductor observe --project myapp errors
```

## Advanced Workflows

### Performance Debugging

Identify slow operations:

```bash
# View all stats
conductor observe stats

# Focus on operations with high duration
conductor observe export --format json | jq '.tool_executions[] | select(.avg_duration_ms > 1000)'
```

### Cost Tracking

Monitor API usage costs:

```bash
# View cost breakdown
conductor observe stats | grep "Total Cost"

# Export for detailed analysis
conductor observe export --format json --output cost-analysis.json
```

### Agent Effectiveness

Compare agent performance:

```bash
# View agent breakdown
conductor observe stats | grep -A 20 "Agent Performance"

# Export for comparison
conductor observe export --format markdown --output agent-report.md
```

### Continuous Monitoring

Set up periodic exports for monitoring:

```bash
# Daily export
conductor observe --time-range 24h export --format json --output "daily-$(date +%Y%m%d).json"

# Weekly summary
conductor observe --time-range 7d export --format markdown --output weekly-report.md
```

## Integration with Other Tools

### jq (JSON Processing)

```bash
# Extract specific metrics
conductor observe export --format json | jq '.success_rate'

# Find high-error tools
conductor observe export --format json | jq '.tool_executions[] | select(.error_rate > 0.1) | .name'

# Calculate total token cost
conductor observe export --format json | jq '.token_usage.cost_usd'
```

### grep (Pattern Matching)

```bash
# Find specific patterns
conductor observe export --format markdown | grep "ERROR"

# Count occurrences
conductor observe export --format markdown | grep -c "FAILED"
```

### awk (Data Processing)

```bash
# Extract columns from CSV
conductor observe export --format csv | awk -F',' '{print $2, $3}'

# Calculate averages
conductor observe export --format csv | awk -F',' 'NR>1 && $1=="Summary" {sum+=$3; count++} END {print sum/count}'
```

## Tips and Best Practices

1. **Use filters to reduce noise**: Start broad, then narrow with filters
2. **Export regularly**: Keep historical data for trend analysis
3. **Monitor costs**: Track token usage to optimize agent usage
4. **Identify patterns**: Look for recurring errors or slow operations
5. **Compare agents**: Use performance data to select optimal agents
6. **Stream for debugging**: Watch real-time activity when troubleshooting
7. **Combine filters**: Use multiple filters for precise analysis
8. **Automate exports**: Set up periodic exports for continuous monitoring

## Troubleshooting

### No projects found

```bash
# Check Claude projects directory
ls -la ~/.claude/projects/

# Ensure sessions exist
ls -la ~/.claude/projects/*/
```

### Empty results

```bash
# Remove filters to see all data
conductor observe stats

# Check time range
conductor observe --time-range 30d stats
```

### Export errors

```bash
# Ensure output directory exists
mkdir -p ./reports

# Use absolute paths
conductor observe export --format json --output /full/path/to/report.json
```

## Next Steps

- View detailed architecture in `docs/AGENT_WATCH_ARCHITECTURE.md`
- Understand subcommand mapping in `docs/AGENT_WATCH_SUBCOMMAND_MAPPING.md`
- Explore behavioral package API in `internal/behavioral/`
- Extend with custom exporters in `internal/behavioral/exporter.go`
