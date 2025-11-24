# Agent Watch Usage Examples

Common use cases and workflows for Agent Watch.

## Getting Started

### First-Time Usage

```bash
# Check what projects are available
conductor observe

# Output:
# Agent Watch Interactive Mode
# =============================
#
# Available Projects:
# ----------------------------------------------------------------------
#   [1] myapp                          (5 sessions, 1.24 MB)
#   [2] api-service                    (12 sessions, 3.45 MB)
#   [3] frontend                       (8 sessions, 2.10 MB)
# ----------------------------------------------------------------------
#
# Select project (1-3) or 'q' to quit:
```

### Quick Stats Check

```bash
# View overall statistics
conductor observe stats

# Output:
# === SUMMARY STATISTICS ===
#
# Total Sessions:      25
# Total Agents:        8
# Total Operations:    1,234
# Success Rate:        87.5%
# Error Rate:          12.5%
# Average Duration:    4m 32s
# Total Cost:          $0.1234
#
# Total Input Tokens:  125,000
# Total Output Tokens: 45,000
# Total Tokens:        170,000
#
# --- Top Tools (by usage) ---
#  1. Read                 Count:  450  Success: 99.1%  Errors:  0.9%
#  2. Write                Count:  120  Success: 95.8%  Errors:  4.2%
#  3. Bash                 Count:   98  Success: 88.7%  Errors: 11.3%
```

## Daily Workflows

### Morning Status Check

```bash
# What happened in the last 24 hours?
conductor observe stats --time-range 24h

# Any errors overnight?
conductor observe errors --time-range 24h
```

### Project Deep Dive

```bash
# Focus on a specific project
conductor observe stats --project myapp

# Check tool usage for that project
conductor observe tools --project myapp

# Analyze file operations
conductor observe files --project myapp
```

### Debugging a Failed Session

```bash
# Find sessions with errors
conductor observe errors --project myapp --time-range 7d

# Examine specific session
conductor observe session abc123-session-id

# Check what bash commands failed
conductor observe bash --errors-only --session abc123
```

## Monitoring Workflows

### Real-Time Monitoring

```bash
# Watch all activity in real-time
conductor observe stream

# Output (updates every 2 seconds):
# Real-time streaming mode
# Watching for new sessions... (Press Ctrl+C to stop)
#
# [14:32:45] SUCCESS | Task: Build API endpoint | Agent: backend-developer | Duration: 2m 15s
# [14:35:12] SUCCESS | Task: Write tests | Agent: test-automator | Duration: 1m 45s
# [14:38:01] FAILED  | Task: Deploy | Agent: devops-engineer | Duration: 3m 22s

# Watch specific project
conductor observe stream --project myapp
```

### Daily Reports

```bash
# Generate daily report
conductor observe export --format markdown --output report-$(date +%Y%m%d).md --time-range 24h

# Generate JSON for further analysis
conductor observe export --format json --output metrics-$(date +%Y%m%d).json --time-range 24h
```

## Analysis Workflows

### Tool Performance Analysis

```bash
# Which tools are failing most?
conductor observe tools --errors-only

# Tool usage over the past week
conductor observe tools --time-range 7d

# Compare tool usage across projects
conductor observe tools --project backend
conductor observe tools --project frontend
```

### Agent Performance Comparison

```bash
# View agent rankings
conductor observe stats

# Look for patterns in agent failures
conductor observe errors --time-range 30d
```

### Cost Analysis

```bash
# Check token usage and costs
conductor observe stats

# Export for spreadsheet analysis
conductor observe export --format csv --output costs.csv --time-range 30d
```

## Export Workflows

### Generate Documentation

```bash
# Create a markdown report for team sharing
conductor observe export --format markdown --output weekly-report.md --time-range 7d
```

### Data Analysis Pipeline

```bash
# Export to JSON for Python/R analysis
conductor observe export --format json --output data.json

# Process with jq
conductor observe export --format json | jq '.tool_executions[] | select(.error_rate > 0.1)'
```

### Backup Metrics

```bash
# Archive monthly data
conductor observe export --format json --output "archive/metrics-$(date +%Y%m).json" --time-range 30d
```

## Troubleshooting Workflows

### High Error Rate Investigation

```bash
# Step 1: Identify scope of problem
conductor observe stats --time-range 7d

# Step 2: Find error patterns
conductor observe errors --time-range 7d

# Step 3: Check specific tool issues
conductor observe tools --errors-only --time-range 7d

# Step 4: Analyze bash failures
conductor observe bash --errors-only --time-range 7d

# Step 5: Export data for deeper analysis
conductor observe export --format json --errors-only --time-range 7d --output errors.json
```

### Slow Task Investigation

```bash
# Check average durations
conductor observe stats --project myapp

# Look at specific session timelines
conductor observe session slow-session-id

# Analyze bash command durations
conductor observe bash --project myapp
```

### Resource Usage Analysis

```bash
# Check file operations
conductor observe files --project myapp

# See which files are accessed most
conductor observe files --time-range 30d

# Export for analysis
conductor observe export --format json --output file-analysis.json
```

## Integration Examples

### CI/CD Integration

```bash
#!/bin/bash
# Post-build metrics collection

# Export metrics after each build
conductor observe export --format json \
  --project myapp \
  --time-range 1h \
  --output "artifacts/metrics-${BUILD_NUMBER}.json"

# Check error rate threshold
error_rate=$(conductor observe stats --project myapp --time-range 1h 2>/dev/null | grep "Error Rate" | awk '{print $3}')
if (( $(echo "$error_rate > 20" | bc -l) )); then
  echo "Warning: High error rate detected: ${error_rate}%"
  exit 1
fi
```

### Slack Notification Script

```bash
#!/bin/bash
# Send daily summary to Slack

summary=$(conductor observe stats --time-range 24h 2>/dev/null | head -20)
curl -X POST -H 'Content-type: application/json' \
  --data "{\"text\":\"Daily Agent Watch Summary:\n\`\`\`${summary}\`\`\`\"}" \
  "$SLACK_WEBHOOK_URL"
```

### Grafana/Prometheus Integration

```bash
#!/bin/bash
# Export metrics for Prometheus pushgateway

conductor observe export --format json --time-range 5m | jq -r '
  "agent_watch_sessions_total \(.total_sessions)",
  "agent_watch_success_rate \(.success_rate)",
  "agent_watch_error_rate \(.error_rate)"
' | curl --data-binary @- http://pushgateway:9091/metrics/job/agent_watch
```

## Filtering Examples

### Combined Filters

```bash
# High-precision filtering
conductor observe tools \
  --project myapp \
  --errors-only \
  --time-range 7d \
  --filter-type tool
```

### Time-Based Analysis

```bash
# Compare this week vs last week
conductor observe stats --time-range 7d
# (manually run for comparison)

# Check specific date range
conductor observe stats --time-range 2025-01-15
```

### Project Comparison

```bash
# Compare multiple projects
for project in backend frontend api; do
  echo "=== $project ==="
  conductor observe stats --project $project --time-range 7d
done
```

## Best Practices

1. **Regular Monitoring**: Run `conductor observe stats` daily to catch issues early
2. **Export Archives**: Weekly export of JSON data for historical analysis
3. **Filter Effectively**: Use time ranges to focus on relevant data
4. **Stream During Development**: Use `conductor observe stream` during active development
5. **Error Thresholds**: Set up alerts when error rate exceeds threshold
6. **Cost Tracking**: Export monthly reports for cost analysis
