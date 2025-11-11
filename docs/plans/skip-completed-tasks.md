# Implementation Plan: Skip Completed Tasks

**Created**: 2025-11-11
**Target**: Enable conductor to read task completion status from plan files, verify actual completion via re-execution, and skip GREEN/YELLOW tasks. Support `--skip-completed` CLI flag, `skip_completed` config option, and `--retry-failed` flag.
**Estimated Tasks**: 9

## Context for the Engineer

You are implementing this feature in a codebase that:
- Uses Go 1.25.4 with minimal external dependencies
- Follows TDD principles with table-driven tests
- Uses spf13/cobra for CLI, goldmark for Markdown parsing, yaml.v3 for YAML
- Already has comprehensive test infrastructure (86%+ coverage)
- Implements interfaces for dependency injection (InvokerInterface, Reviewer, PlanUpdater)
- Uses stubs/mocks in tests with mutex-protected recordings

**You are expected to**:
- Write tests BEFORE implementation (TDD - red/green/refactor)
- Commit after each completed task (9 commits minimum)
- Follow existing code patterns (Table-driven tests, stubs with mutex recording)
- Use interface injection for testability (mock PlanUpdater, Reviewer, etc.)
- Maintain 85%+ test coverage for all packages
- Table-driven tests with comprehensive edge case coverage

## Worktree Groups

**Group chain-1**: Complete Skip Feature Implementation
- **Tasks**: Task 1 → Task 2 → Task 3 → Task 4 → Task 5 → Task 6 → Task 7 → Task 8 → Task 9
- **Branch**: `feature/skip-completed-tasks/chain-1`
- **Execution**: Sequential (each task depends on previous data model changes)
- **Rationale**: Models → Parsers → Executor → Config → CLI → Integration → Tests → Docs

## Prerequisites Checklist

- [ ] Go 1.25.4 installed (`go version`)
- [ ] Git configured for commits (`git config user.name` / `git config user.email`)
- [ ] All tests pass locally (`go test ./...`)
- [ ] Feature branch created from `main` or appropriate base branch
- [ ] IDE/editor configured for Go formatting (gofmt automatic on save)

---

## Task 1: Extend Task Model with Status Fields

**Agent**: golang-pro
**File(s)**: `internal/models/task.go`, `internal/models/task_test.go`
**Worktree Group**: chain-1
**Estimated time**: 15m

### What you're building
Add `Status` and `CompletedAt` fields to the Task struct so that task status can be persisted in memory after parsing. Add helper methods `IsCompleted()` and `CanSkip()` for checking task status in executor logic.

### Test First (TDD)

**Test file**: `internal/models/task_test.go` (extend existing)

**Test structure**:
- TestTask_StatusFields - verify new fields exist and are accessible
- TestTask_IsCompleted - verify status checking logic
- TestTask_CanSkip - verify skip eligibility with retry config

### Implementation

Add two fields to Task struct and implement two helper methods. Status field tracks task state: "completed", "in-progress", "failed", or empty. CompletedAt stores timestamp. Helper methods enable executor logic to check if tasks should be skipped without direct field access.

**Code structure**:
```go
type Task struct {
    // ... existing fields ...
    Status      string        // NEW: "completed", "in-progress", "failed", or ""
    CompletedAt *time.Time    // NEW: timestamp when completed
}

func (t *Task) IsCompleted() bool {
    return t.Status == "completed"
}

func (t *Task) CanSkip(retryFailed bool) bool {
    if t.Status == "" {
        return false // No status = execute
    }
    if t.Status == "in-progress" {
        return false // In-progress = re-execute to verify
    }
    if t.Status == "failed" {
        return !retryFailed // Skip if not retrying failed
    }
    return t.Status == "completed" || t.Status == "yellow"
}
```

### Verification

**Manual testing**:
1. Create task with no status, verify `IsCompleted()` returns false
2. Set status to "completed", verify `IsCompleted()` returns true
3. Call `CanSkip(true)` with failed status, verify returns true

**Automated tests**:
```bash
go test ./internal/models -v
```

### Commit

**Commit message**:
```
feat: add status tracking to Task model

Add Status and CompletedAt fields to enable resumable execution.
Add IsCompleted() and CanSkip() helper methods for executor logic.
```

**Files to commit**:
- `internal/models/task.go`
- `internal/models/task_test.go`

---

## Task 2: Update Markdown Parser to Extract Status

**Agent**: golang-pro
**File(s)**: `internal/parser/markdown.go`, `internal/parser/markdown_test.go`
**Depends on**: Task 1
**Worktree Group**: chain-1
**Estimated time**: 20m

### What you're building
Enhance the Markdown parser to extract task completion status from plan files. Parse checkbox state `[x]` vs `[ ]` and inline status annotations `(status: completed)` from task headings.

### Implementation

In parseTaskMetadata(), extract checkbox state, parse status annotations, and override with explicit annotation if present.

### Commit

**Commit message**:
```
feat: extract task status from markdown plans

Parse checkbox state [x] and (status: xxx) annotations.
Status extracted during plan parsing and assigned to Task struct.
```

**Files to commit**:
- `internal/parser/markdown.go`
- `internal/parser/markdown_test.go`

---

## Task 3: Update YAML Parser to Extract Status

**Agent**: golang-pro
**File(s)**: `internal/parser/yaml.go`, `internal/parser/yaml_test.go`
**Depends on**: Task 1
**Worktree Group**: chain-1
**Estimated time**: 15m

### What you're building
Enhance the YAML parser to extract the `status` field and completed_date/completed_at timestamps. Transfer these to the Task model so status persists after loading plans.

### Commit

**Commit message**:
```
feat: extract task status and timestamp from YAML plans

Transfer status field from YAML to Task struct.
Parse completed_date and completed_at to Task.CompletedAt.
```

**Files to commit**:
- `internal/parser/yaml.go`
- `internal/parser/yaml_test.go`

---

## Task 4: Add SkipCompleted to Config Model

**Agent**: golang-pro
**File(s)**: `internal/config/config.go`, `internal/config/config_test.go`, `.conductor/config.yaml.example`
**Depends on**: Task 1
**Worktree Group**: chain-1
**Estimated time**: 15m

### What you're building
Add `SkipCompleted bool` and `RetryFailed bool` fields to the Config struct. These will be read from the YAML config file and overridable via CLI flags.

### Commit

**Commit message**:
```
feat: add skip_completed and retry_failed config options

Add SkipCompleted (default: true) and RetryFailed (default: false) to Config.
Support loading from YAML config files.
```

**Files to commit**:
- `internal/config/config.go`
- `internal/config/config_test.go`
- `.conductor/config.yaml.example`

---

## Task 5: Add CLI Flags for Skip and Retry

**Agent**: golang-pro
**File(s)**: `internal/cmd/run.go`, `internal/cmd/run_test.go`
**Depends on**: Task 4
**Worktree Group**: chain-1
**Estimated time**: 15m

### What you're building
Add `--skip-completed` / `--no-skip-completed` and `--retry-failed` CLI flags to the `conductor run` command. These flags should override config file values.

### Commit

**Commit message**:
```
feat: add --skip-completed and --retry-failed CLI flags

Add CLI flags to control skip and retry behavior.
Flags override config file values.
```

**Files to commit**:
- `internal/cmd/run.go`
- `internal/cmd/run_test.go`

---

## Task 6: Implement Skip Logic in WaveExecutor

**Agent**: golang-pro
**File(s)**: `internal/executor/wave.go`, `internal/executor/wave_test.go`
**Depends on**: Task 1, Task 5
**Worktree Group**: chain-1
**Estimated time**: 25m

### What you're building
Add skip logic to WaveExecutor to filter out completed tasks before execution. For skipped tasks, create synthetic TaskResult with GREEN status and "Skipped" output. Only skip if skip_completed is enabled and task's CanSkip() returns true.

### Commit

**Commit message**:
```
feat: implement task skip logic in wave executor

Filter completed tasks before execution.
Create synthetic GREEN results for skipped tasks.
Respect skip_completed config and retry_failed flags.
```

**Files to commit**:
- `internal/executor/wave.go`
- `internal/executor/wave_test.go`

---

## Task 7: Wire Skip Config Through Orchestrator

**Agent**: golang-pro
**File(s)**: `internal/executor/orchestrator.go`, `internal/executor/orchestrator_test.go`, `internal/cmd/run.go`
**Depends on**: Task 6
**Worktree Group**: chain-1
**Estimated time**: 20m

### What you're building
Thread the skip_completed and retry_failed configuration through the orchestrator to the wave executor. Update the run command to pass merged config (CLI + file) through the execution chain.

### Commit

**Commit message**:
```
feat: wire skip config through orchestrator to executor

Pass skip_completed and retry_failed config to WaveExecutor.
Config flows from CLI/file through execution chain.
```

**Files to commit**:
- `internal/executor/orchestrator.go`
- `internal/executor/orchestrator_test.go`
- `internal/cmd/run.go`

---

## Task 8: Integration Tests for Skip Behavior

**Agent**: golang-pro
**File(s)**: `test/integration/conductor_test.go`, `test/integration/fixtures/`
**Depends on**: Task 7
**Worktree Group**: chain-1
**Estimated time**: 30m

### What you're building
Create comprehensive integration tests that exercise the complete skip-completed feature end-to-end. Test with both Markdown and YAML plans, various status combinations, and config/CLI interactions.

### Commit

**Commit message**:
```
test: add comprehensive integration tests for skip feature

E2E tests for skip behavior with Markdown and YAML plans.
Tests for config/CLI overrides and resume scenarios.
```

**Files to commit**:
- `test/integration/conductor_test.go`
- `test/integration/fixtures/resume-completed-markdown.md`
- `test/integration/fixtures/resume-completed-yaml.yaml`

---

## Task 9: Documentation and Example Plans

**Agent**: golang-pro
**File(s)**: `README.md`, `docs/usage.md`, `docs/plan-format.md`, `CLAUDE.md`
**Depends on**: Task 8
**Worktree Group**: chain-1
**Estimated time**: 20m

### What you're building
Document the new skip-completed feature in README, usage guide, and plan format guide. Create example plans showing how to mark tasks as completed and test resume functionality. Update CLAUDE.md with architecture changes.

### Commit

**Commit message**:
```
docs: document skip-completed feature

Update README with skip capabilities.
Add usage examples and configuration docs.
Document status fields in plan format guide.
```

**Files to commit**:
- `README.md`
- `docs/usage.md`
- `docs/plan-format.md`
- `CLAUDE.md`

---

## Testing Strategy

### Unit Tests
- **Location**: `internal/**/*_test.go`
- **Naming**: `Test{ComponentName}_{Feature}`
- **Run command**: `go test ./internal/... -v`
- **Coverage target**: 85%+

### Integration Tests
- **Location**: `test/integration/conductor_test.go`
- **What to test**: End-to-end skip behavior with real file parsing
- **Run**: `go test ./test/integration -v`

---

## Commit Strategy

Each task produces one commit. Sequence:

1. Task 1: `feat: add status tracking to Task model`
2. Task 2: `feat: extract task status from markdown plans`
3. Task 3: `feat: extract task status and timestamp from YAML plans`
4. Task 4: `feat: add skip_completed and retry_failed config options`
5. Task 5: `feat: add --skip-completed and --retry-failed CLI flags`
6. Task 6: `feat: implement task skip logic in wave executor`
7. Task 7: `feat: wire skip config through orchestrator to executor`
8. Task 8: `test: add comprehensive integration tests for skip feature`
9. Task 9: `docs: document skip-completed feature`

---

## Summary

This plan adds resumable execution to conductor through 9 focused tasks:

1. Data model: Status tracking in Task struct
2. Markdown parsing: Extract completion status from checkboxes and annotations
3. YAML parsing: Extract status fields and timestamps
4. Config model: Add skip_completed and retry_failed options
5. CLI integration: Add flags to control skip behavior
6. Executor logic: Filter completed tasks, create synthetic results
7. Configuration wiring: Thread config through orchestrator chain
8. Integration testing: Comprehensive E2E tests with resume scenarios
9. Documentation: Update guides with examples and configuration

**Start with**: Task 1 - Add Status fields to Task model
**First commit**: After Task 1 is complete and tested
**Estimated time**: 3.5-4 hours for all 9 tasks including testing
