# Conductor Analyze Commands Guide

## Quick Start

The `conductor observe` subsystem includes four analysis commands for understanding agent behavior patterns:

```bash
# Tool usage analysis - See which tools are used most and their success rates
conductor observe tools

# Bash command analysis - Analyze command patterns and failures
conductor observe bash

# File operation analysis - Track file access patterns
conductor observe files

# Error analysis - Identify and track error patterns
conductor observe errors
```

## Command Details

### 1. conductor observe tools

Analyzes tool invocation patterns from Claude Code agent sessions.

**Usage:**
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

### 2. conductor observe bash

Analyzes bash command execution patterns.

**Usage:**
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

### 3. conductor observe files

Analyzes file operation patterns (read, write, edit).

**Usage:**
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

### 4. conductor observe errors

Analyzes error patterns from all operation types.

**Usage:**
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

## Common Use Cases

### Identify Performance Bottlenecks

```bash
# Find slowest tools
conductor observe tools --limit 0 | sort -k5 -rn

# Find slowest bash commands
conductor observe bash --limit 0 | sort -k5 -rn

# Find slowest file operations
conductor observe files --limit 0 | sort -k7 -rn
```

### Troubleshoot Failures

```bash
# Find most problematic tools
conductor observe tools | grep -E "^\s+[0-9]" | sort -k4 -rn | head -10

# Identify bash command failures
conductor observe bash | grep -E "^\s+[0-9]" | sort -k4 -rn | head -10

# Analyze error patterns
conductor observe errors
```

### Project-Specific Analysis

```bash
# Analyze specific project
conductor observe tools --project backend
conductor observe bash --project backend
conductor observe files --project backend
conductor observe errors --project backend
```

### Pagination

```bash
# View in chunks for large datasets
conductor observe tools --limit 50
conductor observe tools --limit 50 | tail -20  # Skip header, show next batch

# Get full list for processing
conductor observe tools --limit 0 > tools-report.txt
```

## Database Requirements

The analyze commands query the behavioral database which is populated by:

1. **Agent Watch imports** - Import JSONL files:
   ```bash
   conductor observe import ~/.claude/projects/myapp/session.jsonl
   ```

2. **Automatic tracking** - If Agent Watch integration is enabled in config

## Output Format

All commands display results in a tabular format:

- **Header row** - Column names (Tool, Command, File, etc.)
- **Data rows** - One entry per unique item
- **Summary line** - Shows pagination info if limited

Columns vary by command:
- **tools**: Tool name, Calls, Success, Failures, Avg Duration, Success %
- **bash**: Command, Calls, Success, Failures, Avg Duration, Success %
- **files**: File path, Operation type, Calls, Success, Failures, Avg Duration, Total bytes
- **errors**: Type, Component, Error message, Count, Last occurred

## Performance Notes

- **Query time**: < 1 second for typical datasets (100K+ records)
- **Memory usage**: Minimal, limited by `--limit` parameter
- **Database indexing**: Optimized with indexes on common filter fields
- **Scalability**: Supports databases with millions of records

## Limitations

- **Time filtering**: Current version doesn't support `--time-range` (planned for v2.9)
- **Export formats**: Display only (JSON/CSV export planned for v2.9)
- **Comparison**: Single-point analysis (trending comparison planned for v2.9)
- **Streaming**: Not available for analysis commands (available for `stream` command)

## Integration with Other Commands

Use analyze commands with other observe commands:

```bash
# Get overall stats first
conductor observe stats

# Then dive deeper with analysis commands
conductor observe tools
conductor observe bash
conductor observe files
conductor observe errors
```

## Troubleshooting

### Command not found
```
Error: unknown command "tools"
```
**Solution**: Ensure conductor is rebuilt: `go build ./cmd/conductor`

### No data displayed
```
No tool data available
```
**Solution**: Ensure behavioral data is imported:
```bash
conductor observe import ~/.claude/projects/*/session.jsonl
```

### Configuration error
```
load config: conductor repo root not configured
```
**Solution**: Ensure `.conductor/config.yaml` exists in project root

## Next Steps

1. **Import data** - Use `conductor observe import` to load behavioral data
2. **Explore stats** - Run `conductor observe stats` for overview
3. **Deep dive** - Use analysis commands (tools, bash, files, errors)
4. **Export results** - Copy output or use `conductor observe export` (v2.9+)

## Additional Resources

- [Agent Watch Documentation](docs/AGENT_WATCH.md)
- [Usage Examples](docs/examples/agent-watch-usage.md)
- [Filtering Guide](docs/examples/agent-watch-filtering.md)
- [Configuration Guide](docs/examples/AGENT_WATCH_CONFIG.md)
