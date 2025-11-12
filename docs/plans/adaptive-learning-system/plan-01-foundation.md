# Implementation Plan: Adaptive Learning System for Conductor v2.0.0 (Part 1)

**Created**: 2025-01-12
**Target**: Implement SQLite-based learning system that records task execution history, analyzes failure patterns, automatically adapts agent selection after repeated failures, and provides CLI observability commands
**Estimated Tasks**: 12 (Tasks 1-12 of 25 total)

## Context for the Engineer

You are implementing this feature in a codebase that:
- Uses Go 1.25.4 with Clean Architecture pattern
- Follows strict TDD with 86.4% test coverage target
- Tests with Go testing package (testing, testify/assert, testify/require)
- Uses SQLite via github.com/mattn/go-sqlite3 (CGO-based)
- Is a CLI orchestration tool that executes implementation plans by spawning Claude Code agents

**Current Status**: Conductor v1.1.0 is production-ready. This learning system will be v2.0.0 (major version bump).

**You are expected to**:
- Write tests BEFORE implementation (TDD red-green-refactor)
- Commit after each completed task with conventional commit format
- Follow existing code patterns in internal/ packages
- Keep changes minimal and focused (YAGNI)
- Avoid code duplication (DRY - reference existing utilities)
- Use worktrees for parallel development when tasks are independent
- Add comprehensive error handling with graceful degradation
- Maintain 86%+ test coverage for all new code

## Worktree Groups

This plan identifies task groupings for parallel execution using git worktrees. Each group operates in isolation with its own branch.

**Group chain-1-database-foundation**: Tasks 1→2→3
- **Tasks**: Task 1, Task 2, Task 3 (executed sequentially in dependency order)
- **Branch**: `feature/adaptive-learning-v2/chain-1-database`
- **Execution**: Sequential (dependencies enforced)
- **Isolation**: Separate worktree from other groups
- **Rationale**: Core database and learning store functionality must be built sequentially as each task depends on the previous

**Group chain-2-executor-integration**: Tasks 4→5→6→7
- **Tasks**: Task 4, Task 5, Task 6, Task 7 (executed sequentially)
- **Branch**: `feature/adaptive-learning-v2/chain-2-executor`
- **Execution**: Sequential
- **Isolation**: Separate worktree from other groups
- **Rationale**: Executor integration tasks form a dependency chain - hooks must be implemented before orchestrator integration

**Group chain-3-cli-commands**: Tasks 8→9→10→11→12
- **Tasks**: Task 8, Task 9, Task 10, Task 11, Task 12 (executed sequentially)
- **Branch**: `feature/adaptive-learning-v2/chain-3-cli`
- **Execution**: Sequential
- **Isolation**: Separate worktree from other groups
- **Rationale**: CLI commands build on each other - stats command provides foundation for show, clear, and export

### Worktree Management

**Creating worktrees**:
```bash
# For chain-1 (database foundation)
git worktree add ../conductor-chain-1-database -b feature/adaptive-learning-v2/chain-1-database

# For chain-2 (executor integration) - wait for chain-1 to merge
git worktree add ../conductor-chain-2-executor -b feature/adaptive-learning-v2/chain-2-executor

# For chain-3 (CLI commands) - can start after chain-1 completes
git worktree add ../conductor-chain-3-cli -b feature/adaptive-learning-v2/chain-3-cli
```

**Switching between worktrees**:
```bash
cd ../conductor-chain-1-database
cd ../conductor-chain-2-executor
cd ../conductor-chain-3-cli
```

**Cleaning up after merge**:
```bash
git worktree remove ../conductor-chain-1-database
git branch -d feature/adaptive-learning-v2/chain-1-database
```

## Prerequisites Checklist

- [ ] **Required tools installed**: Go 1.25.4+, git, CGO compiler (for mattn/go-sqlite3), sqlite3 CLI (optional, for manual DB inspection)
- [ ] **Development environment setup**: GOPATH configured, conductor builds successfully with `go build ./cmd/conductor`
- [ ] **Understanding of existing codebase**: Reviewed internal/ package structure, read CLAUDE.md, understand current execution flow
- [ ] **Familiarity with SQLite and CGO**: Understand SQLite schema design, CGO build requirements, database/sql package in Go
- [ ] **Git worktrees understood**: Comfortable with `git worktree add/list/remove` commands for parallel development
- [ ] **Test environment ready**: Can run `go test ./...` successfully, understand table-driven test patterns
- [ ] **Branch created from main**: Working from main branch with latest changes pulled

## Task 1: Create learning package with database schema and initialization

**Agent**: golang-pro
**File(s)**: internal/learning/schema.sql, internal/learning/store.go, internal/learning/store_test.go
**Depends on**: None
**WorktreeGroup**: chain-1-database-foundation
**Estimated time**: 2h

Create the foundational learning package with SQLite database schema embedded using go:embed. This task establishes the database structure for storing task executions, failure patterns, and approach history. The schema includes proper indexes for fast lookups and supports concurrent access patterns needed by the orchestrator.

### Test-First Approach

**Test file**: internal/learning/store_test.go

**Test structure**:
- TestNewStore - database initialization
- TestInitSchema - schema creation and versioning
- TestStoreConnection - connection management
- TestSchemaVersion - version tracking
- TestDatabasePath - path resolution (relative, absolute)

**Mocks needed**:
- No mocks needed - use in-memory SQLite (:memory:)

**Fixtures**:
- Test database path helpers
- Temporary directory creation

**Assertions**:
- Database file created at correct path
- Schema tables exist (task_executions, approach_history, schema_version)
- Indexes created successfully
- Schema version set to 1
- Connection can be opened and closed

**Edge cases**:
- Database already exists (should not recreate)
- Invalid path (should return error)
- Permission denied (should return error)
- Concurrent initialization attempts

**Example test skeleton**:
```go
package learning

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
    tests := []struct {
        name    string
        dbPath  string
        wantErr bool
    }{
        {
            name:    "creates database successfully",
            dbPath:  filepath.Join(t.TempDir(), "test.db"),
            wantErr: false,
        },
        {
            name:    "handles in-memory database",
            dbPath:  ":memory:",
            wantErr: false,
        },
        {
            name:    "returns error for invalid path",
            dbPath:  "/invalid/path/db.db",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            store, err := NewStore(tt.dbPath)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            require.NotNil(t, store)
            defer store.Close()

            // Verify schema initialized
            version, err := store.getSchemaVersion()
            require.NoError(t, err)
            assert.Equal(t, 1, version)
        })
    }
}
```

### Implementation

**Approach**:
1. Create internal/learning/ package directory
2. Design SQL schema with two main tables: task_executions and approach_history
3. Embed schema.sql using //go:embed directive
4. Implement Store struct with *sql.DB connection
5. Add NewStore() constructor that initializes database and runs schema
6. Include schema versioning for future migrations
7. Add proper error handling with wrapped errors
8. Implement Close() method for cleanup

**Code structure**:
```go
// internal/learning/store.go
package learning

import (
    "database/sql"
    "embed"
    "fmt"
    _ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

type Store struct {
    db *sql.DB
    dbPath string
}

func NewStore(dbPath string) (*Store, error) {
    // Ensure directory exists
    // Open SQLite connection
    // Initialize schema
    // Return store
}

func (s *Store) Close() error {
    if s.db != nil {
        return s.db.Close()
    }
    return nil
}

func (s *Store) initSchema() error {
    _, err := s.db.Exec(schemaSQL)
    return err
}
```

**Key points**:
- Use mattn/go-sqlite3 driver: Import with underscore: `_ "github.com/mattn/go-sqlite3"`
- Embed SQL schema: Use //go:embed schema.sql directive to embed file in binary
- Schema versioning: Include schema_version table with INSERT OR IGNORE for idempotency
- Error wrapping: Use fmt.Errorf with %w to wrap errors for better stack traces
- Directory creation: Use os.MkdirAll to create parent directories if needed
- Connection pooling: database/sql handles pooling automatically, single *sql.DB instance is safe

**Integration points**:
- Imports: `database/sql`, `embed`, `os`, `path/filepath`, `_ "github.com/mattn/go-sqlite3"`
- Error handling: Wrap all errors with context: `fmt.Errorf("init schema: %w", err)`

### Verification

**Manual testing**:
1. Create test directory and database: `mkdir -p /tmp/conductor-test && touch /tmp/conductor-test/test.db`
2. Run package tests: `go test ./internal/learning/ -v` (expected: All tests pass, database created with correct schema)
3. Inspect database schema: `sqlite3 /tmp/conductor-test/test.db ".schema"` (expected: Shows task_executions, approach_history, schema_version tables with indexes)

**Automated tests**:
```bash
go test ./internal/learning/store_test.go -v -run TestNewStore
```

**Success criteria**:
- All store initialization tests pass
- Schema creates all required tables and indexes
- Schema version table tracks version 1
- No compiler errors
- go vet passes

### Commit

```
feat: add learning database schema with SQLite

Create foundational learning package with:
- SQLite schema embedded via go:embed
- task_executions table for execution history
- approach_history table for tracking tried approaches
- schema_version table for migration support
- Store struct with initialization and cleanup
- Comprehensive tests for database creation

Part of adaptive learning system v2.0.0
```

**Files to commit**:
- internal/learning/schema.sql
- internal/learning/store.go
- internal/learning/store_test.go

## Task 2: Implement learning store CRUD operations for execution recording

**Agent**: golang-pro
**File(s)**: internal/learning/store.go, internal/learning/store_test.go
**Depends on**: Task 1
**WorktreeGroup**: chain-1-database-foundation
**Estimated time**: 2h

Implement core CRUD operations in the learning store for recording task executions, querying execution history, and managing approach history. This provides the data layer for all learning functionality with proper error handling and concurrent access support.

### Test-First Approach

**Test file**: internal/learning/store_test.go

**Test structure**:
- TestRecordExecution - basic execution recording
- TestRecordExecution_Concurrent - parallel writes
- TestGetExecutionHistory - query by plan/task
- TestGetRunCount - count runs per plan
- TestRecordApproach - track tried approaches

**Fixtures**:
- Sample TaskExecution structs
- Test database with seed data

**Assertions**:
- Execution recorded with all fields
- Timestamps set automatically
- Concurrent writes don't corrupt data
- Queries return correct filtered results

**Edge cases**:
- Duplicate execution IDs
- Null/empty fields
- Very long output strings
- Special characters in fields

**Example test skeleton**:
```go
func TestRecordExecution(t *testing.T) {
    store := setupTestStore(t)
    defer store.Close()

    exec := &TaskExecution{
        PlanFile: "test.md",
        TaskNumber: "Task 1",
        AgentUsed: "golang-pro",
        QCVerdict: "GREEN",
    }

    err := store.RecordExecution(context.Background(), exec)
    require.NoError(t, err)

    // Verify record exists
    history, err := store.GetExecutionHistory(context.Background(), "test.md", "Task 1")
    require.NoError(t, err)
    assert.Len(t, history, 1)
}
```

### Implementation

**Approach**:
1. Define TaskExecution and ApproachHistory structs
2. Implement RecordExecution with prepared statements
3. Implement GetExecutionHistory with filtering
4. Add GetRunCount for session tracking
5. Implement RecordApproach for approach history
6. Use context for all operations (timeout support)
7. Add proper error wrapping and logging

**Code structure**:
```go
type TaskExecution struct {
    ID              int64
    PlanFile        string
    TaskNumber      string
    SessionID       string
    RunNumber       int
    AgentUsed       string
    QCVerdict       string
    ExecutionTimeMs int64
    FailurePatterns []string
    ExecutedAt      time.Time
}

func (s *Store) RecordExecution(ctx context.Context, exec *TaskExecution) error {
    // Marshal JSON fields
    // Execute INSERT with context
    // Return wrapped error
}
```

**Key points**:
- Use prepared statements: Prevent SQL injection and improve performance
- Context timeout support: All DB operations accept context.Context
- JSON marshaling for arrays: Store FailurePatterns as JSON TEXT field
- Transaction support: Future-proof for multi-record operations

**Integration points**:
- Imports: `context`, `encoding/json`, `time`
- Error handling: Wrap all DB errors with context

### Verification

**Manual testing**:
1. Run store tests: `go test ./internal/learning/ -v -run TestRecordExecution` (expected: All CRUD operations pass)

**Automated tests**:
```bash
go test ./internal/learning/ -v
```

**Success criteria**:
- All CRUD tests pass
- Concurrent write tests pass
- No data races detected

### Commit

```
feat: implement learning store with execution recording

Add CRUD operations for learning database:
- RecordExecution for storing task execution history
- GetExecutionHistory for querying past executions
- GetRunCount for session tracking
- RecordApproach for tracking tried approaches
- Full test coverage with concurrent access tests
```

**Files to commit**:
- internal/learning/store.go
- internal/learning/store_test.go

## Task 3: Implement failure analysis engine with agent suggestion

**Agent**: golang-pro
**File(s)**: internal/learning/analysis.go, internal/learning/analysis_test.go
**Depends on**: Task 2
**WorktreeGroup**: chain-1-database-foundation
**Estimated time**: 2h

Build the failure analysis engine that examines execution history to identify patterns, suggest alternative agents, and provide actionable recommendations. This is the "intelligence" layer that makes learning decisions.

### Test-First Approach

**Test file**: internal/learning/analysis_test.go

**Test structure**:
- TestAnalyzeFailures_NoHistory
- TestAnalyzeFailures_TwoFailures
- TestAnalyzeFailures_CommonPatterns
- TestFindBestAlternativeAgent
- TestExtractFailurePatterns
- TestGenerateApproachSuggestion

**Fixtures**:
- Mock execution history with various patterns

**Assertions**:
- Correctly identifies when to adapt (2+ failures)
- Suggests best alternative agent based on history
- Extracts patterns from output text
- Generates actionable suggestions

**Edge cases**:
- No history available
- All agents tried
- Conflicting patterns
- Insufficient data for statistical significance

**Example test skeleton**:
```go
func TestAnalyzeFailures_TwoFailures(t *testing.T) {
    store := setupTestStoreWithHistory(t, 2, "RED")

    analysis, err := store.AnalyzeFailures(ctx, "plan.md", "Task 1")
    require.NoError(t, err)

    assert.Equal(t, 2, analysis.FailedAttempts)
    assert.True(t, analysis.ShouldTryDifferentAgent)
    assert.NotEmpty(t, analysis.SuggestedAgent)
}
```

### Implementation

**Approach**:
1. Create FailureAnalysis struct with metrics
2. Implement AnalyzeFailures to query and analyze history
3. Add pattern extraction with keyword matching
4. Implement agent suggestion algorithm (SQL queries)
5. Generate human-readable recommendations
6. Include statistical thresholds (min 5 executions for significance)

**Code structure**:
```go
type FailureAnalysis struct {
    TotalAttempts      int
    FailedAttempts     int
    TriedAgents        []string
    CommonPatterns     []string
    SuggestedAgent     string
    SuggestedApproach  string
    ShouldTryDifferentAgent bool
}

func (s *Store) AnalyzeFailures(ctx context.Context, planFile, taskNumber string) (*FailureAnalysis, error) {
    // Query execution history
    // Count failures and identify patterns
    // Suggest alternative agent
    // Generate recommendations
}
```

**Key points**:
- Pattern detection via keyword matching: Check for: compilation_error, test_failure, dependency_missing, etc.
- Statistical significance: Require min 5 executions before trusting agent suggestions
- Graceful degradation: Return general-purpose agent if no good alternatives found

**Integration points**:
- Imports: `strings`, `context`
- Error handling: Handle empty history gracefully, return defaults when insufficient data

### Verification

**Manual testing**:
1. Run analysis tests: `go test ./internal/learning/analysis_test.go -v` (expected: All analysis logic passes)

**Automated tests**:
```bash
go test ./internal/learning/ -run Analysis -v
```

**Success criteria**:
- Correctly identifies 2+ failure threshold
- Suggests valid alternative agents
- Pattern extraction works reliably

### Commit

```
feat: add failure analysis and agent suggestion logic

Implement intelligent failure analysis:
- AnalyzeFailures examines execution history
- Pattern detection via keyword matching
- Agent suggestion algorithm with SQL queries
- Statistical significance thresholds
- Human-readable recommendations
- Comprehensive test coverage
```

**Files to commit**:
- internal/learning/analysis.go
- internal/learning/analysis_test.go

## Task 4: Add pre-task hook for failure analysis and agent adaptation

**Agent**: golang-pro
**File(s)**: internal/executor/task.go, internal/executor/task_test.go
**Depends on**: Task 3
**WorktreeGroup**: chain-2-executor-integration
**Estimated time**: 1h 30m

Integrate the pre-task hook into TaskExecutor that queries the learning database, analyzes past failures, and adapts the agent/prompt before execution. This is the first of three hook integration points.

### Test-First Approach

**Test file**: internal/executor/task_test.go

**Test structure**:
- TestPreTaskHook_NoHistory
- TestPreTaskHook_AdaptsAgent
- TestPreTaskHook_EnhancesPrompt
- TestPreTaskHook_LearningDisabled

**Mocks needed**:
- Mock LearningStore interface

**Fixtures**:
- Task with various failure histories

**Assertions**:
- Hook called before task execution
- Agent changed when 2+ failures detected
- Prompt enhanced with learning context
- No-op when learning disabled

**Edge cases**:
- Learning store returns error
- No suggested agent available
- Agent already optimal

**Example test skeleton**:
```go
func TestPreTaskHook_AdaptsAgent(t *testing.T) {
    mockStore := &MockLearningStore{
        AnalysisResult: &FailureAnalysis{
            FailedAttempts: 2,
            TriedAgents: []string{"backend-developer"},
            SuggestedAgent: "golang-pro",
            ShouldTryDifferentAgent: true,
        },
    }

    executor := NewTaskExecutor(Config{
        learningStore: mockStore,
    })

    task := &Task{Agent: "backend-developer"}
    executor.preTaskHook(ctx, task)

    assert.Equal(t, "golang-pro", task.Agent)
}
```

### Implementation

**Approach**:
1. Add LearningStore field to TaskExecutor struct
2. Implement preTaskHook method
3. Call hook at start of Execute method
4. Query learning store for failure analysis
5. Adapt agent if ShouldTryDifferentAgent is true
6. Enhance prompt with learning context
7. Log decisions for observability

**Code structure**:
```go
type TaskExecutor struct {
    // existing fields
    learningStore *learning.Store
    sessionID     string
    runNumber     int
    planFile      string
}

func (e *TaskExecutor) preTaskHook(ctx context.Context, task *models.Task) error {
    if e.learningStore == nil {
        return nil // Learning disabled
    }

    analysis, err := e.learningStore.AnalyzeFailures(ctx, e.planFile, task.Number)
    if err != nil {
        log.Warn("pre-task hook failed: %v", err)
        return nil // Don't break execution
    }

    if analysis.ShouldTryDifferentAgent {
        oldAgent := task.Agent
        task.Agent = analysis.SuggestedAgent
        log.Info("Switching agent: %s → %s", oldAgent, task.Agent)
    }

    if analysis.FailedAttempts > 0 {
        task.Prompt = enhancePromptWithLearning(task.Prompt, analysis)
    }

    return nil
}
```

**Key points**:
- Graceful degradation: Learning failures never break task execution
- Logging for observability: Log all agent switches and learning decisions
- Respect user intent: Only switch if configured to do so

**Integration points**:
- Imports: `github.com/harrison/conductor/internal/learning`
- Services to inject: LearningStore from internal/learning
- Error handling: Catch and log learning errors, never return error from hook

### Verification

**Manual testing**:
1. Run executor tests: `go test ./internal/executor/ -run PreTaskHook -v` (expected: All hook tests pass)

**Automated tests**:
```bash
go test ./internal/executor/ -v
```

**Success criteria**:
- Hook integrates without breaking existing tests
- Agent adaptation works correctly
- Learning failures don't break execution

### Commit

```
feat: add pre-task hook with failure analysis

Integrate pre-task hook into TaskExecutor:
- Query learning database before execution
- Adapt agent based on failure analysis
- Enhance prompts with learning context
- Graceful degradation on errors
- Comprehensive test coverage
```

**Files to commit**:
- internal/executor/task.go
- internal/executor/task_test.go

## Task 5: Add QC-review hook for pattern extraction

**Agent**: golang-pro
**File(s)**: internal/executor/task.go, internal/executor/task_test.go
**Depends on**: Task 4
**WorktreeGroup**: chain-2-executor-integration
**Estimated time**: 1h

Implement QC-review hook that captures quality control verdicts and extracts failure patterns from output. This hook runs after QC review and before retries.

### Test-First Approach

**Test file**: internal/executor/task_test.go

**Test structure**:
- TestQCReviewHook_ExtractsPatterns
- TestQCReviewHook_GreenVerdict
- TestQCReviewHook_RedVerdict

**Fixtures**:
- QC outputs with various patterns

**Assertions**:
- Patterns extracted from RED verdicts
- No patterns for GREEN verdicts
- Metadata stored in task

**Edge cases**:
- Empty QC feedback
- Multiple patterns in single output

**Example test skeleton**:
```go
func TestQCReviewHook_ExtractsPatterns(t *testing.T) {
    output := "compilation failed: syntax error"
    verdict := "RED"

    patterns := extractFailurePatterns(verdict, "", output)

    assert.Contains(t, patterns, "compilation_error")
}
```

### Implementation

**Approach**:
1. Add qcReviewHook method to TaskExecutor
2. Call after QC review in Execute method
3. Extract patterns using keyword matching
4. Store patterns in task metadata for post-task hook
5. Log patterns at debug level

**Code structure**:
```go
func (e *TaskExecutor) qcReviewHook(ctx context.Context, task *models.Task, verdict, feedback, output string) {
    patterns := extractFailurePatterns(verdict, feedback, output)

    if task.Metadata == nil {
        task.Metadata = make(map[string]interface{})
    }
    task.Metadata["qc_verdict"] = verdict
    task.Metadata["failure_patterns"] = patterns
}

func extractFailurePatterns(verdict, feedback, output string) []string {
    // Keyword matching logic
}
```

**Key points**:
- Pattern keywords: compilation_error, test_failure, dependency_missing, permission_error, timeout, runtime_error
- Case insensitive matching: strings.ToLower for all comparisons

**Integration points**:
- Error handling: Never fail task execution on pattern extraction errors

### Verification

**Manual testing**:
1. Run QC hook tests: `go test ./internal/executor/ -run QCReviewHook -v` (expected: Pattern extraction works)

**Automated tests**:
```bash
go test ./internal/executor/ -v
```

**Success criteria**:
- Patterns correctly extracted
- Metadata stored properly

### Commit

```
feat: add QC-review hook for pattern extraction

Implement QC-review hook:
- Extract failure patterns from QC output
- Store patterns in task metadata
- Support for 6 pattern types
- Comprehensive test coverage
```

**Files to commit**:
- internal/executor/task.go
- internal/executor/task_test.go

## Task 6: Add post-task hook for execution recording

**Agent**: golang-pro
**File(s)**: internal/executor/task.go, internal/executor/task_test.go
**Depends on**: Task 5
**WorktreeGroup**: chain-2-executor-integration
**Estimated time**: 1h

Implement post-task hook that records complete execution history to the learning database after task completion (success or failure).

### Test-First Approach

**Test file**: internal/executor/task_test.go

**Test structure**:
- TestPostTaskHook_RecordsExecution
- TestPostTaskHook_IncludesPatterns
- TestPostTaskHook_HandlesStoreError

**Mocks needed**:
- Mock LearningStore

**Assertions**:
- Execution recorded with all fields
- Patterns from metadata included
- Store errors logged but don't fail task

**Edge cases**:
- Store unavailable
- Missing metadata

**Example test skeleton**:
```go
func TestPostTaskHook_RecordsExecution(t *testing.T) {
    mockStore := &MockLearningStore{}
    executor := NewTaskExecutor(Config{learningStore: mockStore})

    task := &Task{Number: "Task 1"}
    result := &TaskResult{Status: "GREEN"}

    executor.postTaskHook(ctx, task, result, "GREEN")

    assert.True(t, mockStore.RecordCalled)
}
```

### Implementation

**Approach**:
1. Add postTaskHook method
2. Call at end of Execute method (after QC/retries)
3. Build TaskExecution struct from task+result
4. Extract patterns from task metadata
5. Call store.RecordExecution
6. Log success/failure

**Code structure**:
```go
func (e *TaskExecutor) postTaskHook(ctx context.Context, task *models.Task, result *models.TaskResult, verdict string) {
    if e.learningStore == nil {
        return
    }

    exec := &learning.TaskExecution{
        PlanFile:    e.planFile,
        TaskNumber:  task.Number,
        SessionID:   e.sessionID,
        RunNumber:   e.runNumber,
        AgentUsed:   task.Agent,
        QCVerdict:   verdict,
        // ... more fields
    }

    if err := e.learningStore.RecordExecution(ctx, exec); err != nil {
        log.Warn("failed to record execution: %v", err)
    }
}
```

**Key points**:
- Extract metadata: Get failure_patterns from task.Metadata if present
- Graceful degradation: Log errors but don't fail task

**Integration points**:
- Error handling: Catch and log all recording errors, never fail task on recording error

### Verification

**Manual testing**:
1. Run post-task hook tests: `go test ./internal/executor/ -run PostTaskHook -v` (expected: Recording works correctly)

**Automated tests**:
```bash
go test ./internal/executor/ -v
```

**Success criteria**:
- Execution recorded successfully
- All fields populated correctly

### Commit

```
feat: add post-task hook with learning storage

Implement post-task hook:
- Record complete execution history
- Extract patterns from metadata
- Graceful error handling
- Full test coverage
```

**Files to commit**:
- internal/executor/task.go
- internal/executor/task_test.go

## Task 7: Integrate learning system into orchestrator with session management

**Agent**: golang-pro
**File(s)**: internal/executor/orchestrator.go, internal/executor/orchestrator_test.go
**Depends on**: Task 6
**WorktreeGroup**: chain-2-executor-integration
**Estimated time**: 1h 30m

Update orchestrator to initialize learning store, generate session IDs, track run numbers, and pass learning context to executors.

### Test-First Approach

**Test file**: internal/executor/orchestrator_test.go

**Test structure**:
- TestOrchestrator_WithLearning
- TestOrchestrator_WithoutLearning
- TestOrchestrator_SessionIDGeneration
- TestOrchestrator_RunNumberTracking

**Assertions**:
- Learning store passed to executors
- Session ID generated correctly
- Run number calculated from history

**Edge cases**:
- Learning disabled (nil store)
- First run vs subsequent runs

**Example test skeleton**:
```go
func TestOrchestrator_WithLearning(t *testing.T) {
    config := OrchestratorConfig{
        LearningStore: mockStore,
        SessionID: "test-session",
    }

    orch := NewOrchestrator(plan, config)

    assert.NotNil(t, orch.learningStore)
}
```

### Implementation

**Approach**:
1. Add learning fields to OrchestratorConfig
2. Pass learning store to wave/task executors
3. Generate session ID if not provided
4. Query run number from learning store
5. Include session context in all executors

**Code structure**:
```go
type OrchestratorConfig struct {
    // existing fields
    LearningStore *learning.Store
    SessionID     string
    RunNumber     int
    PlanFile      string
}

func NewOrchestrator(plan *models.Plan, config OrchestratorConfig) *Orchestrator {
    // Generate session ID if empty
    if config.SessionID == "" {
        config.SessionID = generateSessionID()
    }

    // Calculate run number
    if config.LearningStore != nil && config.RunNumber == 0 {
        config.RunNumber = config.LearningStore.GetRunCount(config.PlanFile) + 1
    }

    return &Orchestrator{...}
}
```

**Key points**:
- Session ID format: session-YYYYMMDD-HHMM
- Run number starts at 1: First run of a plan is run 1

**Integration points**:
- Error handling: Handle nil learning store gracefully

### Verification

**Manual testing**:
1. Run orchestrator tests: `go test ./internal/executor/ -run Orchestrator -v` (expected: Learning integration works)

**Automated tests**:
```bash
go test ./internal/executor/ -v
```

**Success criteria**:
- Orchestrator passes learning to executors
- Session management works correctly

### Commit

```
feat: integrate learning system into orchestrator

Add learning support to orchestrator:
- Pass learning store to executors
- Generate session IDs
- Track run numbers
- Full test coverage
```

**Files to commit**:
- internal/executor/orchestrator.go
- internal/executor/orchestrator_test.go

## Task 8: Implement 'conductor learning stats' command

**Agent**: golang-pro
**File(s)**: internal/cmd/learning_stats.go, internal/cmd/learning_stats_test.go
**Depends on**: Task 2
**WorktreeGroup**: chain-3-cli-commands
**Estimated time**: 1h 30m

Create CLI command to display learning statistics for a plan file including success rates, agent performance, common failures, and task-level metrics.

### Test-First Approach

**Test file**: internal/cmd/learning_stats_test.go

**Test structure**:
- TestStatsCommand_ValidPlan
- TestStatsCommand_NoData
- TestStatsCommand_InvalidPlan
- TestStatsCommand_FormattedOutput

**Mocks needed**:
- Mock LearningStore

**Fixtures**:
- Sample execution data

**Assertions**:
- Correct statistics calculated
- Output formatted properly
- Handles missing data gracefully

**Edge cases**:
- No executions found
- Plan file doesn't exist

**Example test skeleton**:
```go
func TestStatsCommand_ValidPlan(t *testing.T) {
    cmd := newStatsCommand()
    output := captureOutput(func() {
        cmd.Run(cmd, []string{"plan.md"})
    })

    assert.Contains(t, output, "Total executions:")
    assert.Contains(t, output, "Success rate:")
}
```

### Implementation

**Approach**:
1. Create cobra command for 'learning stats'
2. Accept plan file as argument
3. Query learning store for statistics
4. Calculate success rates, agent performance
5. Format output with tables
6. Handle errors gracefully

**Code structure**:
```go
func newStatsCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "stats [plan-file]",
        Short: "Show learning statistics for a plan",
        Args:  cobra.ExactArgs(1),
        RunE:  runStats,
    }
}

func runStats(cmd *cobra.Command, args []string) error {
    planFile := args[0]
    store := openLearningStore()
    stats := store.GetStatistics(planFile)
    printStatistics(stats)
    return nil
}
```

**Key points**:
- Table formatting: Use fatih/color for colored output
- Database path resolution: Check .conductor/learning/ relative to plan file

**Integration points**:
- Imports: `github.com/spf13/cobra`, `github.com/fatih/color`
- Error handling: Handle missing database gracefully, show helpful message if no data

### Verification

**Manual testing**:
1. Run stats command: `./conductor learning stats plan.md` (expected: Statistics displayed)

**Automated tests**:
```bash
go test ./internal/cmd/ -run Stats -v
```

**Success criteria**:
- Command works end-to-end
- Output is readable and helpful

### Commit

```
feat: add learning stats command

Implement 'conductor learning stats':
- Display success rates
- Show agent performance
- List common failures
- Task-level metrics
- Formatted table output
```

**Files to commit**:
- internal/cmd/learning_stats.go
- internal/cmd/learning_stats_test.go

## Task 9: Implement 'conductor learning show' command

**Agent**: golang-pro
**File(s)**: internal/cmd/learning_show.go, internal/cmd/learning_show_test.go
**Depends on**: Task 8
**WorktreeGroup**: chain-3-cli-commands
**Estimated time**: 1h

Create CLI command to show detailed execution history for a specific task including all attempts, agents used, verdicts, and timestamps.

### Test-First Approach

**Test file**: internal/cmd/learning_show_test.go

**Test structure**:
- TestShowCommand_ValidTask
- TestShowCommand_NoHistory
- TestShowCommand_FormattedOutput

**Assertions**:
- History displayed chronologically
- All execution details shown

**Edge cases**:
- Task never executed
- Task name with spaces

**Example test skeleton**:
```go
func TestShowCommand_ValidTask(t *testing.T) {
    cmd := newShowCommand()
    output := captureOutput(func() {
        cmd.Run(cmd, []string{"plan.md", "Task 1"})
    })

    assert.Contains(t, output, "Execution History")
}
```

### Implementation

**Approach**:
1. Create cobra command for 'learning show'
2. Accept plan file and task number
3. Query execution history
4. Display chronologically with details
5. Highlight successes vs failures

**Code structure**:
```go
func newShowCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "show [plan-file] [task-number]",
        Short: "Show detailed history for a task",
        Args:  cobra.ExactArgs(2),
        RunE:  runShow,
    }
}
```

**Key points**:
- Chronological order: Most recent first
- Color coding: Green for GREEN, red for RED

**Integration points**:
- Imports: `github.com/spf13/cobra`
- Error handling: Handle missing history gracefully

### Verification

**Manual testing**:
1. Run show command: `./conductor learning show plan.md "Task 1"` (expected: History displayed)

**Automated tests**:
```bash
go test ./internal/cmd/ -run Show -v
```

**Success criteria**:
- Detailed history shown
- Output is clear

### Commit

```
feat: add learning show command for task details

Implement 'conductor learning show':
- Display execution history
- Show all attempts
- Agent and verdict details
- Chronological ordering
```

**Files to commit**:
- internal/cmd/learning_show.go
- internal/cmd/learning_show_test.go

## Task 10: Implement 'conductor learning clear' command

**Agent**: golang-pro
**File(s)**: internal/cmd/learning_clear.go, internal/cmd/learning_clear_test.go
**Depends on**: Task 8
**WorktreeGroup**: chain-3-cli-commands
**Estimated time**: 45m

Create CLI command to clear learning data, either for a specific plan file or the entire database, with confirmation prompt for safety.

### Test-First Approach

**Test file**: internal/cmd/learning_clear_test.go

**Test structure**:
- TestClearCommand_SinglePlan
- TestClearCommand_AllData
- TestClearCommand_Confirmation

**Assertions**:
- Data cleared correctly
- Confirmation required for --all
- Reports count of deleted records

**Edge cases**:
- No data to clear
- User cancels confirmation

**Example test skeleton**:
```go
func TestClearCommand_SinglePlan(t *testing.T) {
    // Test clearing specific plan
}
```

### Implementation

**Approach**:
1. Create cobra command for 'learning clear'
2. Support --all flag for entire database
3. Require confirmation prompt
4. Delete records from database
5. Report count of deleted records

**Code structure**:
```go
func newClearCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "clear [plan-file]",
        Short: "Clear learning data",
        RunE:  runClear,
    }
    cmd.Flags().Bool("all", false, "Clear entire database")
    return cmd
}
```

**Key points**:
- Confirmation prompt: Require y/n confirmation for safety
- Report deleted count: Show how many records were deleted

**Integration points**:
- Imports: `bufio`, `os`
- Error handling: Handle missing database gracefully

### Verification

**Manual testing**:
1. Run clear command: `./conductor learning clear plan.md` (expected: Data cleared after confirmation)

**Automated tests**:
```bash
go test ./internal/cmd/ -run Clear -v
```

**Success criteria**:
- Clears data correctly
- Confirmation works

### Commit

```
feat: add learning clear command

Implement 'conductor learning clear':
- Clear specific plan data
- Clear entire database with --all
- Confirmation prompt for safety
- Report deleted record count
```

**Files to commit**:
- internal/cmd/learning_clear.go
- internal/cmd/learning_clear_test.go

## Task 11: Implement 'conductor learning export' command

**Agent**: golang-pro
**File(s)**: internal/cmd/learning_export.go, internal/cmd/learning_export_test.go
**Depends on**: Task 8
**WorktreeGroup**: chain-3-cli-commands
**Estimated time**: 1h

Create CLI command to export learning data to JSON or CSV format for external analysis or backup purposes.

### Test-First Approach

**Test file**: internal/cmd/learning_export_test.go

**Test structure**:
- TestExportCommand_JSON
- TestExportCommand_CSV
- TestExportCommand_InvalidFormat

**Assertions**:
- JSON export valid
- CSV export valid
- Error on invalid format

**Edge cases**:
- Empty database
- Output to file vs stdout

**Example test skeleton**:
```go
func TestExportCommand_JSON(t *testing.T) {
    // Test JSON export
}
```

### Implementation

**Approach**:
1. Create cobra command for 'learning export'
2. Support --format flag (json/csv)
3. Query all executions
4. Marshal to requested format
5. Write to stdout or file

**Code structure**:
```go
func newExportCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "export [plan-file]",
        Short: "Export learning data",
        RunE:  runExport,
    }
    cmd.Flags().String("format", "json", "Export format (json|csv)")
    return cmd
}
```

**Key points**:
- JSON marshaling: Use encoding/json with proper indentation
- CSV format: Use encoding/csv with headers

**Integration points**:
- Imports: `encoding/json`, `encoding/csv`
- Error handling: Validate format flag

### Verification

**Manual testing**:
1. Export to JSON: `./conductor learning export plan.md --format json` (expected: Valid JSON output)

**Automated tests**:
```bash
go test ./internal/cmd/ -run Export -v
```

**Success criteria**:
- Both formats work
- Output is valid

### Commit

```
feat: add learning export command

Implement 'conductor learning export':
- Export to JSON format
- Export to CSV format
- Output to stdout or file
- Full test coverage
```

**Files to commit**:
- internal/cmd/learning_export.go
- internal/cmd/learning_export_test.go

## Task 12: Wire up learning subcommands to root command

**Agent**: golang-pro
**File(s)**: internal/cmd/root.go, internal/cmd/learning.go, internal/cmd/root_test.go
**Depends on**: Task 11
**WorktreeGroup**: chain-3-cli-commands
**Estimated time**: 30m

Create 'conductor learning' parent command and wire up all subcommands (stats, show, clear, export) to make them accessible from CLI.

### Test-First Approach

**Test file**: internal/cmd/root_test.go

**Test structure**:
- TestLearningCommand_SubcommandsRegistered
- TestLearningCommand_HelpText

**Assertions**:
- All subcommands registered
- Help text correct

**Example test skeleton**:
```go
func TestLearningCommand_SubcommandsRegistered(t *testing.T) {
    rootCmd := NewRootCommand()
    learningCmd := findCommand(rootCmd, "learning")

    assert.NotNil(t, learningCmd)
    assert.Equal(t, 4, len(learningCmd.Commands()))
}
```

### Implementation

**Approach**:
1. Create parent 'learning' command
2. Add all subcommands (stats, show, clear, export)
3. Register with root command
4. Add helpful description and examples

**Code structure**:
```go
func newLearningCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "learning",
        Short: "Learning system commands",
        Long:  "Commands for viewing and managing conductor's learning system",
    }

    cmd.AddCommand(newStatsCommand())
    cmd.AddCommand(newShowCommand())
    cmd.AddCommand(newClearCommand())
    cmd.AddCommand(newExportCommand())

    return cmd
}

// In root.go
rootCmd.AddCommand(newLearningCommand())
```

**Key points**:
- Command organization: Parent command with 4 subcommands
- Help text: Clear descriptions and examples

### Verification

**Manual testing**:
1. Run learning help: `./conductor learning --help` (expected: Shows all subcommands)
2. Run each subcommand: `./conductor learning stats --help` (expected: Command-specific help shown)

**Automated tests**:
```bash
go test ./internal/cmd/ -v
```

**Success criteria**:
- All commands accessible
- Help text correct

### Commit

```
feat: integrate learning commands into CLI

Wire up learning subcommands:
- Create parent 'learning' command
- Register all 4 subcommands
- Add to root command
- Complete CLI integration
```

**Files to commit**:
- internal/cmd/root.go
- internal/cmd/learning.go
- internal/cmd/root_test.go

---

**Note**: This is Part 1 of 2. Continue with plan-02-configuration-testing.md for Tasks 13-25.
