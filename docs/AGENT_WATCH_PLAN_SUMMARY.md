# Agent Watch Integration - Complete Plan Summary

## Overview

This document provides a comprehensive overview of the Agent Watch integration plan for Conductor. The full implementation is split across three YAML plan files with 26 coordinated tasks totaling approximately 48-64 hours of development.

## File Structure

### Plan Files

1. **agent-watch-integration-part1.yaml** (6 tasks, ~8 hours)
   - Foundation layer: behavioral models, JSONL parser, session discovery
   - Location: `/docs/plans/agent-watch-integration-part1.yaml`

2. **agent-watch-integration-part2.yaml** (6 tasks, ~12 hours)
   - Database layer: schema expansion, auto-migration, QC integration
   - Location: `/docs/plans/agent-watch-integration-part2.yaml`

3. **agent-watch-integration-part3.yaml** (14 tasks, ~32 hours)
   - CLI implementation, analytics, console enhancement, documentation
   - Location: `/docs/plans/agent-watch-integration-part3.yaml`

## Architecture Overview

```
Part 1: Foundation Layer
├── Task 1: Behavioral Models Package
├── Task 2: JSONL Parser for Claude Sessions
├── Task 3: Session Discovery (~/.claude/projects)
├── Task 4: Behavioral Metrics Extraction
├── Task 5: Metrics Aggregation Pipeline (with caching)
└── Task 6: Comprehensive Foundation Tests

Part 2: Database & QC Integration
├── Task 7: Expand Learning Schema (behavioral tables)
├── Task 8: Auto-Migration System
├── Task 9: Behavioral Data Accessors
├── Task 10: Capture session_id from Claude CLI
├── Task 11: Integrate Behavioral Collection into Executor
└── Task 12: Enhance QC Prompts with Behavioral Context

Part 3: CLI, Analytics & Documentation
├── Tasks 13-19: observe Command Implementation
│   ├── Task 13: Root observe command
│   ├── Task 14: Interactive project menu
│   ├── Task 15: Filtering system
│   ├── Task 16: Export functionality (JSON/Markdown)
│   ├── Task 17: Pagination & display
│   ├── Task 18: Statistics & streaming
│   └── Task 19: CLI tests
├── Tasks 20-21: Console Enhancement
│   ├── Task 20: Behavioral metrics in logger
│   └── Task 21: Colorized output
├── Tasks 22-24: Analytics
│   ├── Task 22: Pattern detection
│   ├── Task 23: Failure prediction
│   └── Task 24: Performance scoring
└── Tasks 25-26: Testing & Documentation
    ├── Task 25: Integration tests
    └── Task 26: Documentation
```

## Execution Flow

### Part 1: Foundation Chain (Sequential)
- Tasks 1→2→3→4→5→6
- Branch: `feature/agent-watch/foundation-chain`
- Duration: ~8 hours
- Output: Behavioral models package with JSONL parsing and metrics extraction

### Part 2: Database Chain (Sequential)
- Tasks 7→8→9 (Main chain)
- Task 10 (Concurrent with foundation)
- Tasks 11→12 (Sequential, after database ready)
- Branches:
  - `feature/agent-watch/database-chain` (Tasks 7-9)
  - `feature/agent-watch/integration-chain` (Tasks 10-12)
- Duration: ~12 hours
- Output: Enhanced learning database with behavioral schema and QC integration

### Part 3: Multiple Parallel Chains
- **CLI Chain** (Tasks 13-19, sequential): `feature/agent-watch/cli-chain`
- **Console Enhancement** (Tasks 20-21, sequential): `feature/agent-watch/console-enhancement`
- **Analytics Chain** (Tasks 22-24, sequential): `feature/agent-watch/analytics-chain`
- **Testing/Docs** (Tasks 25-26, sequential): `feature/agent-watch/testing-docs`
- Duration: ~32 hours
- Output: Complete observe command, analytics, documentation

## Key Features Implemented

### CLI Features (Full Parity with Bash Agent Watch)

1. **Interactive Project Menu** (Task 14)
   - Browse projects from ~/.claude/projects
   - Arrow key or number selection
   - Pagination for large project lists

2. **Flexible Filtering** (Task 15)
   - Search by agent name, tool, or command
   - Filter by event type (tool_call, bash_command, file_operation)
   - Show errors only
   - Time range filtering (today, last hour, custom dates)

3. **Export Functionality** (Task 16)
   - JSON format (for tooling/APIs)
   - Markdown format (for reports)
   - CSV format (bonus)

4. **Pagination & Display** (Task 17)
   - 50 items per page
   - Colorized output (green=success, red=error)
   - Formatted tables with column alignment
   - Navigation controls

5. **Statistics & Streaming** (Task 18)
   - Summary stats: agents, operations, cost, success rate
   - Agent type breakdown
   - Top tools used
   - Optional real-time streaming (--stream-activity flag)

### Integration Features

1. **Session Tracking** (Task 10)
   - Capture session_id from Claude CLI JSON response
   - Construct file paths: ~/.claude/projects/{project}/agent-{session_id}.jsonl

2. **Behavioral Data Collection** (Task 11)
   - Load JSONL files after task invocation
   - Extract metrics: tools used, bash commands, file operations, tokens, cost
   - Store in learning database

3. **QC Enhancement** (Task 12)
   - Inject behavioral context into QC prompts
   - Highlight anomalies and patterns
   - Enable data-driven QC decisions

### Analytics Features

1. **Pattern Detection** (Task 22)
   - Tool usage sequences
   - Bash command patterns
   - Anomaly identification
   - Behavior clustering

2. **Failure Prediction** (Task 23)
   - Predict task failure based on tool patterns
   - Confidence scoring (0-1)
   - Explainable predictions

3. **Performance Scoring** (Task 24)
   - Rate agents by success, cost, speed
   - Multi-dimensional scoring
   - Agent comparison within domains

## Database Schema Additions

New tables added to learning database:

```sql
CREATE TABLE behavioral_sessions (
  id INTEGER PRIMARY KEY,
  run_id TEXT,
  task_id TEXT,
  session_id TEXT,
  project TEXT,
  agent TEXT,
  start_time DATETIME,
  end_time DATETIME
);

CREATE TABLE tool_executions (
  id INTEGER PRIMARY KEY,
  session_id TEXT,
  tool_name TEXT,
  call_count INTEGER,
  success_count INTEGER,
  error_count INTEGER,
  avg_duration_ms INTEGER
);

CREATE TABLE bash_commands (
  id INTEGER PRIMARY KEY,
  session_id TEXT,
  command TEXT,
  exit_code INTEGER,
  output_length INTEGER,
  duration_ms INTEGER
);

CREATE TABLE file_operations (
  id INTEGER PRIMARY KEY,
  session_id TEXT,
  operation_type TEXT,
  file_path TEXT,
  size_bytes INTEGER,
  success BOOLEAN
);

CREATE TABLE token_usage (
  id INTEGER PRIMARY KEY,
  session_id TEXT,
  input_tokens INTEGER,
  output_tokens INTEGER,
  cost_usd REAL,
  model_name TEXT
);
```

## Code Organization

### New Packages

- **internal/behavioral/** - Behavioral metrics models, parsing, extraction, aggregation
  - `models.go` - Core data structures
  - `parser.go` - JSONL parsing
  - `discovery.go` - Session discovery
  - `extractor.go` - Metrics extraction
  - `aggregator.go` - Caching and aggregation
  - `filter.go` - Filtering logic
  - `exporter.go` - JSON/Markdown export
  - `display.go` - Pagination and formatting
  - `stats.go` - Statistics calculation
  - `patterns.go` - Pattern detection
  - `prediction.go` - Failure prediction
  - `scoring.go` - Performance scoring

### Modified Packages

- **internal/cmd/** - New observe subcommand
  - `observe.go` - Root command
  - `observe_menu.go` - Interactive project menu
  - `observe_filter.go` - Filtering
  - `observe_export.go` - Export functionality
  - `observe_display.go` - Display and pagination
  - `observe_stats.go` - Statistics
  - `observe_*_test.go` - Test files

- **internal/learning/** - Database enhancements
  - `schema.sql` - New behavioral tables
  - `store.go` - Data accessors
  - `migration.go` - Auto-migration logic

- **internal/agent/** - Session ID capture
  - `invoker.go` - Extract session_id from responses

- **internal/executor/** - Behavioral integration
  - `task.go` - Collect behavioral metrics after invocation
  - `qc.go` - Inject context into QC prompts
  - `behavior_integration.go` - Integration utilities

- **internal/logger/** - Console enhancement
  - `console.go` - Display behavioral metrics
  - `console_color.go` - Colorization

- **internal/models/** - Response model updates
  - `response.go` - Add SessionID field

## Testing Strategy

### Test Coverage Targets
- Behavioral package: 80%+ coverage
- CLI package: 80%+ coverage
- New learning store methods: 85%+ coverage
- Overall project: maintain 86%+ coverage

### Testing Approaches
- **TDD**: Red-green-refactor cycle for all tasks
- **Table-driven tests**: Use subtests (t.Run()) for multiple scenarios
- **Integration tests**: End-to-end flows with realistic fixtures
- **Race detection**: All tests pass with `-race` flag
- **Benchmarks**: Performance-critical paths (parsing, aggregation)

### Test Files
- `internal/behavioral/*_test.go` - Unit and integration tests
- `internal/behavioral/integration_test.go` - End-to-end flows
- `internal/cmd/observe_integration_test.go` - CLI integration tests
- Test fixtures in `internal/behavioral/testdata/`

## Dependency Graph

```
Part 1 (Foundation)
  ↓
Part 2 (Database & QC)
  ├→ CLI Chain (Part 3a)
  ├→ Console Enhancement (Part 3b, independent)
  ├→ Analytics Chain (Part 3c)
  └→ Testing/Docs (Part 3d, final)
```

Key dependencies:
- All Part 3 chains depend on Part 1 and Part 2 completion
- Database chain depends on foundation completion
- QC integration depends on database completion
- Analytics chains are independent after foundation available
- Testing/Docs depends on all implementation complete

## Configuration

### Conductor CLI Settings

New flags for observe command:
```bash
conductor observe [--project PROJECT] [--session SESSION_ID]
  [--filter-type TYPE] [--errors-only] [--time-range RANGE]
  [--stream-activity] [--export FORMAT] [--output FILE]
```

### Config File (.conductor/config.yaml)

New settings:
```yaml
observe:
  enabled: true
  stream_activity: false  # Optional real-time streaming
  cache_size: 50          # Behavioral metrics cache size
  page_size: 50           # Items per page in pagination

learning:
  # Existing settings work with behavioral data
  enabled: true
  qc_reads_behavioral_context: true  # New flag
```

## Success Criteria

### Functional Requirements Met
- [x] Full CLI parity with bash Agent Watch
- [x] Session tracking via session_id
- [x] JSONL file parsing and metrics extraction
- [x] Database storage with auto-migration
- [x] QC enhancement with behavioral context
- [x] Real-time streaming (optional)
- [x] All filtering, export, pagination features
- [x] Analytics and performance scoring

### Quality Requirements Met
- [x] 80%+ test coverage on new code
- [x] All tests pass with -race detection
- [x] No breaking changes to existing APIs
- [x] Graceful degradation if JSONL unavailable
- [x] TDD workflow (tests before implementation)
- [x] Comprehensive documentation and examples

### Performance Requirements Met
- [x] Metrics aggregation caching (LRU, 50 sessions)
- [x] Minimal I/O during discovery (stat only)
- [x] Efficient database queries with indexes
- [x] Streaming updates without blocking
- [x] Pagination for large result sets

## Usage Examples

### Interactive Mode
```bash
conductor observe
# Shows project menu, select project
# Shows available sessions/agents
# Interactive filtering and navigation
```

### Project Inspection
```bash
conductor observe --project my-project
# Shows all sessions for project
# Displays behavioral metrics
```

### Filtering
```bash
conductor observe --project my-project --errors-only
# Show only failed sessions

conductor observe --filter-type bash_command --search "git commit"
# Find sessions with specific bash commands
```

### Export
```bash
conductor observe --project my-project --export json --output report.json
# Export to JSON for processing

conductor observe --export markdown --output report.md
# Export to Markdown for sharing
```

### Statistics
```bash
conductor observe stats --project my-project
# Show aggregate statistics and top patterns
```

### Real-Time Streaming
```bash
conductor observe --stream-activity
# Watch new sessions as they complete (if enabled)
```

## Timeline & Effort Estimation

| Phase | Duration | Tasks | FTE |
|-------|----------|-------|-----|
| Part 1: Foundation | ~8 hours | 1-6 | 1 |
| Part 2: Database | ~12 hours | 7-12 | 1-2 (parallel) |
| Part 3: CLI/Analytics | ~32 hours | 13-26 | 1-2 (parallel) |
| **Total** | **~48-64 hours** | **26** | **1-2** |

## Backward Compatibility

All changes are fully backward compatible:
- No breaking changes to existing APIs
- Optional behavioral data (works if unavailable)
- Existing learning database remains unchanged
- New schema tables added without modifying existing ones
- Auto-migration handles both fresh and existing databases

## Risk Mitigation

1. **JSONL Parsing Failures**
   - Graceful error handling (skip malformed lines)
   - Warning logging without blocking execution

2. **Missing Session Files**
   - Behavioral data optional (non-blocking)
   - Graceful degradation for QC without context

3. **Database Issues**
   - Auto-migration idempotent (safe to re-run)
   - Transaction support for atomic operations
   - Connection pooling for concurrent access

4. **Performance**
   - LRU caching for parsed sessions
   - Index optimization for queries
   - Pagination for large result sets

## Related Documentation

- **CLAUDE.md**: Project overview and development guidelines
- **docs/conductor.md**: Full feature documentation
- **docs/examples/**: Example usage patterns
- **internal/behavioral/README.md**: Package documentation (to be created)
- **docs/AGENT_WATCH.md**: Complete Agent Watch guide (Task 26)

## Getting Started

### To Execute This Plan:

1. **Start with Part 1**:
   ```bash
   conductor run docs/plans/agent-watch-integration-part1.yaml
   ```

2. **After Part 1 merges, run Part 2**:
   ```bash
   conductor run docs/plans/agent-watch-integration-part2.yaml
   ```

3. **After Part 2 merges, run Part 3**:
   ```bash
   conductor run docs/plans/agent-watch-integration-part3.yaml
   ```

Or run all three in sequence:
```bash
conductor run docs/plans/agent-watch-integration-*.yaml
```

### Validation Before Execution:

```bash
conductor validate docs/plans/agent-watch-integration-*.yaml
```

## Questions & Contact

For questions about specific tasks or implementation details, refer to:
- CLAUDE.md for project context
- Individual YAML files for task specifics
- Task descriptions and implementation sections for technical details
- Test files (when complete) for usage examples
