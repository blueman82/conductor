# Agent Watch Integration - Quick Reference

## Plan Files Location

- **Part 1**: `/docs/plans/agent-watch-integration-part1.yaml` (Tasks 1-6)
- **Part 2**: `/docs/plans/agent-watch-integration-part2.yaml` (Tasks 7-12)
- **Part 3**: `/docs/plans/agent-watch-integration-part3.yaml` (Tasks 13-26)
- **Summary**: `/docs/AGENT_WATCH_PLAN_SUMMARY.md`

## Task Quick Reference

### Part 1: Foundation (8 hours)

| Task | Name | Key Outputs | Dependencies |
|------|------|------------|--------------|
| 1 | Behavioral models package | models.go, doc.go | None |
| 2 | JSONL parser | parser.go, parser_test.go | Task 1 |
| 3 | Session discovery | discovery.go, discovery_test.go | Tasks 1, 2 |
| 4 | Behavioral metrics extraction | extractor.go, extractor_test.go | Tasks 2, 3 |
| 5 | Metrics aggregation pipeline | aggregator.go, aggregator_test.go | Tasks 3, 4 |
| 6 | Foundation tests | integration_test.go, testdata/ | Tasks 1-5 |

**Branch**: `feature/agent-watch/foundation-chain`

### Part 2: Database & QC (12 hours)

| Task | Name | Key Outputs | Dependencies |
|------|------|------------|--------------|
| 7 | Expand learning schema | schema.sql updates | None |
| 8 | Auto-migration | migration.go, store.go updates | Task 7 |
| 9 | Behavioral data accessors | store.go methods | Task 8 |
| 10 | Capture session_id | invoker.go, response.go | Task 7 |
| 11 | Behavioral collection (executor) | task.go, executor updates | Tasks 9, 10 |
| 12 | QC prompt enhancement | qc.go, behavior_context.go | Task 11 |

**Branches**:
- `feature/agent-watch/database-chain` (Tasks 7-9)
- `feature/agent-watch/integration-chain` (Tasks 10-12)

### Part 3: CLI, Analytics, Docs (32 hours)

#### CLI Chain (Tasks 13-19, 7 hours)

| Task | Name | Key Outputs | Dependencies |
|------|------|------------|--------------|
| 13 | Observe root command | observe.go, observe_test.go | None |
| 14 | Interactive project menu | observe_menu.go, project.go | Task 13 |
| 15 | Filtering system | observe_filter.go, filter.go | Task 13 |
| 16 | Export (JSON/Markdown) | observe_export.go, exporter.go | Tasks 13, 15 |
| 17 | Pagination & display | observe_display.go, display.go | Task 13 |
| 18 | Statistics & streaming | observe_stats.go, stats.go | Task 13 |
| 19 | CLI integration tests | observe_integration_test.go | Tasks 13-18 |

**Branch**: `feature/agent-watch/cli-chain`

#### Console Enhancement (Tasks 20-21, 1.5 hours)

| Task | Name | Key Outputs | Dependencies |
|------|------|------------|--------------|
| 20 | Console behavioral metrics | console.go updates | None |
| 21 | Colorized output | console_color.go | Task 20 |

**Branch**: `feature/agent-watch/console-enhancement`

#### Analytics Chain (Tasks 22-24, 4.5 hours)

| Task | Name | Key Outputs | Dependencies |
|------|------|------------|--------------|
| 22 | Pattern detection | patterns.go, patterns_test.go | None |
| 23 | Failure prediction | prediction.go, prediction_test.go | Task 22 |
| 24 | Performance scoring | scoring.go, scoring_test.go | Task 22 |

**Branch**: `feature/agent-watch/analytics-chain`

#### Testing & Docs (Tasks 25-26, 3.5 hours)

| Task | Name | Key Outputs | Dependencies |
|------|------|------------|--------------|
| 25 | Integration tests | integration_test.go (comprehensive) | All tasks |
| 26 | Documentation | AGENT_WATCH.md, examples | All tasks |

**Branch**: `feature/agent-watch/testing-docs`

## Code Locations

### New Packages
```
internal/behavioral/
├── models.go              # Core data structures
├── parser.go              # JSONL parsing
├── discovery.go           # Session discovery
├── extractor.go           # Metrics extraction
├── aggregator.go          # Caching & aggregation
├── filter.go              # Filtering logic
├── exporter.go            # JSON/Markdown export
├── display.go             # Pagination & formatting
├── stats.go               # Statistics
├── patterns.go            # Pattern detection
├── prediction.go          # Failure prediction
├── scoring.go             # Performance scoring
├── project.go             # Project info helpers
└── testdata/              # Test fixtures
    ├── sample-session.jsonl
    ├── full-session-example.jsonl
    └── malformed-events.jsonl
```

### Modified Packages
```
internal/cmd/
├── observe.go             # Root command
├── observe_menu.go        # Project menu
├── observe_filter.go      # Filtering
├── observe_export.go      # Export functionality
├── observe_display.go     # Display & pagination
├── observe_stats.go       # Statistics & streaming
└── observe_*_test.go      # Test files

internal/learning/
├── schema.sql             # Expanded schema
├── migration.go           # Auto-migration
└── store.go               # Behavioral accessors

internal/agent/
└── invoker.go             # Session ID capture

internal/executor/
├── task.go                # Behavioral collection
├── qc.go                  # QC enhancement
└── behavior_integration.go # Integration utilities

internal/logger/
├── console.go             # Metrics display
└── console_color.go       # Colorization

internal/models/
└── response.go            # SessionID field
```

## Key Features by Task

### Session Tracking (Task 10)
- Extract `session_id` from Claude CLI JSON response
- Store in `AgentResponse.SessionID`
- Enables JSONL file discovery

### Behavioral Metrics (Tasks 4, 11)
Extracted from JSONL events:
- **Tool Executions**: call count, success rate, errors, duration
- **Bash Commands**: command text, exit code, output length, duration
- **File Operations**: operation type, path, size, success
- **Token Usage**: input/output tokens, cost in USD

### Filtering (Task 15)
```bash
# Search by name/command
conductor observe --filter-type bash_command --search "git"

# Errors only
conductor observe --errors-only

# Time range
conductor observe --time-range "last_hour"
conductor observe --time-range "2025-11-24"
```

### Export (Task 16)
```bash
# JSON for APIs
conductor observe --export json --output report.json

# Markdown for sharing
conductor observe --export markdown --output report.md
```

### QC Enhancement (Task 12)
Behavioral context added to QC prompts:
- Tool usage patterns
- Error/failure rates
- Cost summary
- Anomaly detection

## Database Schema (Task 7)

New tables:
- `behavioral_sessions` - Session metadata (run, task, project, agent, timestamps)
- `tool_executions` - Tools used (name, counts, success rate, duration)
- `bash_commands` - Bash operations (command, exit code, duration)
- `file_operations` - File changes (type, path, success)
- `token_usage` - Token and cost tracking

All tables linked to task_executions via foreign keys and indexed.

## Test Coverage Targets

| Component | Coverage Target |
|-----------|-----------------|
| behavioral package | 80%+ |
| cmd (observe) | 80%+ |
| learning (behavioral accessors) | 85%+ |
| executor (behavioral integration) | 80%+ |
| Overall project | 86%+ (maintain existing) |

## Configuration

### New Conductor Flags
```bash
conductor observe [--project PROJECT]
  [--session SESSION_ID]
  [--filter-type TYPE]
  [--errors-only]
  [--time-range RANGE]
  [--stream-activity]
  [--export FORMAT]
  [--output FILE]
```

### New Config Options (.conductor/config.yaml)
```yaml
observe:
  enabled: true
  stream_activity: false
  cache_size: 50
  page_size: 50
```

## Execution Order

```
1. Part 1 (Foundation) → Merge to main
2. Part 2 (Database) → Merge to main
3. Part 3 (CLI/Analytics/Docs) - 4 parallel chains:
   ├── CLI Chain (tasks 13-19)
   ├── Console Enhancement (tasks 20-21)
   ├── Analytics Chain (tasks 22-24)
   └── Testing/Docs (tasks 25-26)
   → All merge to main
```

## Common Commands During Development

```bash
# Validate plan syntax
conductor validate docs/plans/agent-watch-integration-*.yaml

# Run Part 1
conductor run docs/plans/agent-watch-integration-part1.yaml

# Test during development
go test ./internal/behavioral/... -v
go test ./internal/behavioral/... -race
go test ./... -cover

# View behavioral package coverage
go test ./internal/behavioral/... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Manual testing
go run ./cmd/conductor observe
go run ./cmd/conductor observe --project test-project
go run ./cmd/conductor observe --help
```

## Documentation Files

- **CLAUDE.md** - Project overview (update with Agent Watch section)
- **AGENT_WATCH_PLAN_SUMMARY.md** - This comprehensive guide
- **AGENT_WATCH_QUICK_REFERENCE.md** - Quick lookup (this file)
- **docs/AGENT_WATCH.md** - User guide (Task 26)
- **docs/examples/agent-watch-usage.md** - Usage examples (Task 26)
- **docs/examples/agent-watch-filtering.md** - Filtering guide (Task 26)

## Success Indicators

After each part:

**Part 1 Complete**: `internal/behavioral/` package functional with JSONL parsing
```bash
go test ./internal/behavioral/... -v
# All tests pass, 80%+ coverage
```

**Part 2 Complete**: Database schema updated, QC context working
```bash
conductor run test-plan.yaml
# Sessions tracked, behavioral data stored, QC sees context
```

**Part 3 Complete**: Full `conductor observe` command operational
```bash
conductor observe
# Interactive menu works, filtering works, export works
```

## Troubleshooting Reference

| Issue | Solution | Related Tasks |
|-------|----------|--------------|
| JSONL files not found | Check ~/.claude/projects structure | 3 |
| Session ID missing | Verify Claude CLI version | 10 |
| Behavioral data empty | Check JSONL format, parser logs | 2, 4 |
| QC not receiving context | Verify database storage | 11, 12 |
| observe command fails | Check tool registration | 13 |
| Pagination issues | Verify page size config | 17 |
| Export format errors | Check exporter implementation | 16 |
| Performance slow | Check cache/aggregator | 5, 8 |

## Version Information

**Target Conductor Version**: 2.5.2+
**Go Version**: 1.25.4+
**Dependencies**: All existing (no new external deps)

## File Statistics

**Lines of Code by Part**:
- Part 1: ~2,000 LOC (models, parser, discovery, extraction)
- Part 2: ~1,500 LOC (schema, migration, QC integration)
- Part 3: ~3,500 LOC (CLI, analytics, console)
- Tests: ~4,000 LOC
- **Total**: ~11,000 LOC

**Files Created/Modified**:
- New files: ~25
- Modified files: ~10
- Test files: ~20
- Total files affected: ~55

## Related Issues/PRs

(Links to be updated once tasks are in flight)

- Agent Watch feature request
- Behavioral metrics enhancement
- QC context injection
- observe command implementation

## Feedback & Discussion

For feedback on the plan:
1. Review AGENT_WATCH_PLAN_SUMMARY.md for full context
2. Check individual YAML files for task details
3. Ask clarifying questions before starting
4. Discuss any dependency concerns

---

**Last Updated**: 2025-11-24
**Estimated Total Effort**: 48-64 hours (1-2 engineers)
**Status**: Ready for execution (plan phase complete)
