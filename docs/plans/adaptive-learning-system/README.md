# Implementation Plan: Adaptive Learning System for Conductor v2.0.0

**Created**: 2025-01-12
**Target Release**: v2.0.0 (Major)
**Total Tasks**: 25
**Estimated Time**: 25-30 hours
**Status**: Multi-file plan with Phase 2A format

## Overview

This multi-file implementation plan adds adaptive learning capabilities to Conductor, enabling it to learn from task execution failures across runs and automatically adapt agent selection and task prompts based on historical patterns.

**Key Features**:
- SQLite-backed execution history tracking
- Automatic failure pattern detection
- Agent switching after 2+ failures
- Prompt enhancement with learning context
- CLI commands for observability (stats, show, clear, export)
- Session management with auto-generated IDs
- Graceful degradation (learning failures don't break conductor)

## Plan Files (Conductor Auto-Discovery)

Conductor automatically discovers and orchestrates these files in sequence:

### 1. [plan-01-database-learning-foundation.md](./plan-01-database-learning-foundation.md)

**Tasks**: 1-8 (Database Foundation & Failure Analysis)
**Line Count**: ~2,157 lines
**Worktree Groups**: chain-1, chain-2
**Dependencies**: None (can start immediately)

**Phase**: Foundation
- Task 1: Schema definition with embedded SQL
- Task 2: Store implementation with database initialization
- Task 3: TaskExecution model and recording
- Task 4: Session and run number tracking
- Task 5: Error handling and nil safety
- Task 6: FailureAnalysis model and query logic
- Task 7: Agent suggestion algorithm
- Task 8: Integrate suggestions into AnalyzeFailures

**Status**: ✅ Complete and ready for execution

### 2. [plan-02-hooks-cli-integration.md](./plan-02-hooks-cli-integration.md)

**Tasks**: 9-25 (Hook Integration, CLI, Orchestrator, Testing)
**Worktree Groups**: chain-3, chain-4, chain-5, independent-1, independent-2, chain-6
**Dependencies**: Requires plan-01 Tasks 1-8 completion

**Phase**: Integration & Features
- Tasks 9-12: Hook integration (pre-task, QC-review, post-task)
- Tasks 13-17: CLI commands (stats, show, clear, export)
- Tasks 18-20: Orchestrator integration
- Task 21: Configuration extension
- Task 22: Documentation
- Tasks 23-25: Integration testing

**Status**: 🚧 In progress (Tasks 9-10 detailed, remaining tasks follow established pattern)

## Getting Started with Conductor

### Validate the Multi-File Plan
```bash
conductor validate docs/plans/adaptive-learning-system/
```

### Run with Orchestration
```bash
conductor run docs/plans/adaptive-learning-system/
```

Conductor will automatically:
- Discover both plan-*.md files
- Respect cross-file dependencies
- Execute in correct order (plan-01 → plan-02)
- Track task completion across files

### Manual Execution (without Conductor)

If executing tasks manually:

1. **Start with plan-01** - Complete all 8 tasks in sequence
2. **Move to plan-02** - Begin with Task 9 once Tasks 1-8 are done
3. **Follow worktree groups** - See each plan file for detailed worktree management

## Worktree Group Summary

This plan uses 8 worktree groups for maximum parallelism:

| Group | Tasks | Execution | Dependencies |
|-------|-------|-----------|--------------|
| **chain-1** | 1-5 | Sequential | None |
| **chain-2** | 6-8 | Sequential | After chain-1 |
| **chain-3** | 9-12 | Sequential | After chain-2 |
| **chain-4** | 13-17 | Sequential | After chain-1 (parallel with chain-2/3) |
| **chain-5** | 18-20 | Sequential | After chain-1, chain-3 |
| **independent-1** | 21 | Parallel-safe | After chain-1 |
| **independent-2** | 22 | Parallel-safe | After chain-5 |
| **chain-6** | 23-25 | Sequential | After all previous tasks |

**Parallelism Opportunities**:
- chain-2 and chain-4 can run in parallel after chain-1
- chain-3 depends on chain-2, so runs after
- independent-1 can start early alongside chain-2/3/4
- independent-2 waits for core features but runs independently

## Architecture Summary

**New Package**: `internal/learning/`
- `schema.sql` - Embedded database schema
- `store.go` - Database operations and connection management
- `models.go` - TaskExecution and FailureAnalysis structs
- `analysis.go` - Failure analysis and agent suggestion algorithms
- `patterns.go` - Pattern extraction from QC feedback

**Modified Packages**:
- `internal/executor/` - Add learning hooks to task executor
- `internal/cmd/` - Add learning subcommands
- `internal/config/` - Extend config with learning settings
- `internal/orchestrator/` - Pass learning store to executors

**Database**: SQLite at `.conductor/learning/conductor-learning.db`

## Testing Strategy

**Approach**: Strict TDD (Test-Driven Development)
- Write tests BEFORE implementation
- Red → Green → Refactor cycle for every function
- Target: 86%+ coverage (maintain project standard)

**Test Organization**:
- Unit tests: `internal/learning/*_test.go`
- Integration tests: `internal/executor/*_test.go` (with learning)
- End-to-end: `test/integration/conductor_learning_test.go`

**Mocking Strategy**:
- LearningStore interface for executor tests
- In-memory SQLite (`:memory:`) for learning tests
- Table-driven tests with subtests

## Key Implementation Principles

1. **TDD**: All tasks follow red-green-refactor cycle
2. **Graceful Degradation**: Learning failures never break conductor
3. **Nil Safety**: All learning methods handle nil Store gracefully
4. **Context Support**: Respect context cancellation throughout
5. **Error Wrapping**: Descriptive errors with fmt.Errorf
6. **DRY**: Reuse existing patterns from conductor codebase
7. **YAGNI**: Implement only what's specified, no extras

## Configuration Example

```yaml
# .conductor/config.yaml
learning:
  enabled: true                      # Enabled by default
  db_path: .conductor/learning       # Project-local by default

  # Behavior settings
  auto_adapt_agent: true             # Auto-switch agents after failures
  enhance_prompts: true              # Add learning context to prompts
  min_failures_before_adapt: 2       # Threshold for agent switching

  # Database maintenance
  keep_executions_days: 90          # Auto-purge old data
  max_executions_per_task: 100      # Keep only recent N per task
```

## Dependencies Added

```go
// go.mod
require (
    github.com/mattn/go-sqlite3 v1.14.18  // SQLite driver (CGO)
    // ... existing dependencies ...
)
```

## Version Management

This feature will be released as **v2.0.0** (major version bump):
- Significant new capability
- New CLI commands
- Database introduction
- Behavioral changes (adaptive agent selection)

## Success Criteria

- ✅ All 25 tasks completed with tests passing
- ✅ 86%+ test coverage maintained
- ✅ Learning database created automatically
- ✅ Agent adaptation works after 2 failures
- ✅ CLI commands provide observability
- ✅ Documentation complete
- ✅ Integration tests validate end-to-end flow

## Resources

**Existing Code References**:
- SQLite patterns: `internal/filelock/filelock.go` (file operations)
- Config patterns: `internal/config/config.go` (YAML, path resolution)
- Executor patterns: `internal/executor/task.go` (hooks, error handling)
- Test patterns: `internal/executor/task_test.go` (table-driven, mocks)

**External Documentation**:
- mattn/go-sqlite3: https://github.com/mattn/go-sqlite3
- SQLite docs: https://sqlite.org/docs.html
- Go embed: https://pkg.go.dev/embed

## Next Steps

1. **For Conductor execution**:
   ```bash
   conductor run docs/plans/adaptive-learning-system/
   ```

2. **For manual execution**:
   - Start with plan-01-database-learning-foundation.md, Task 1
   - Follow TDD: Write test → Implement → Verify → Commit
   - Use worktrees for parallelism where indicated

3. **For review**:
   - Each task has complete test skeleton
   - Implementation guidance with code snippets
   - Verification steps
   - Commit messages

## Contact & Support

- **Issues**: Use conductor GitHub issues
- **Questions**: Refer to CLAUDE.md for architecture details
- **Pattern Reference**: See existing conductor packages for coding standards

---

**Plan Generation**: Phase 2A format with auto-discovery support
**Format**: Markdown with Phase 2A naming convention
**Conductor Version**: Compatible with v1.1.0+
