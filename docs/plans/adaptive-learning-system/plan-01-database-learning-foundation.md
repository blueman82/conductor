# Implementation Plan: Adaptive Learning System for Conductor v2.0.0

**Created**: 2025-01-12
**Target**: Implement adaptive learning system that records task execution history, analyzes failure patterns across runs, automatically switches agents after 2 failures, enhances prompts with learning context, and provides CLI observability commands
**Estimated Tasks**: 25
**Estimated Total Time**: 25-30 hours

## Context for the Engineer

You are implementing this feature in Conductor, a production-ready Go CLI tool (v1.1.0) that orchestrates multiple Claude Code agents to execute implementation plans. The codebase:

- **Language**: Go 1.25.4
- **Architecture**: Clean separation of concerns with internal packages (agent, cmd, config, executor, filelock, logger, models, parser, updater)
- **Testing**: Strict TDD approach with 86.4% coverage (465+ tests passing)
- **Commit Style**: Conventional commits with types (feat:, docs:, chore:, etc.)
- **Dependencies**: Minimal (cobra, goldmark, yaml.v3, flock, fatih/color)
- **Current State**: Full wave-based execution with QC reviews, agent discovery, multi-file plan support

**You are expected to**:
- Write tests BEFORE implementation (strict TDD - Red, Green, Refactor)
- Commit frequently after each completed task
- Follow existing code patterns (see internal/executor/task.go, internal/config/config.go)
- Keep changes minimal (YAGNI)
- Avoid duplication (DRY)
- Maintain 86%+ test coverage

**Key existing patterns to follow**:
- Interface-based design for testability (see InvokerInterface, Reviewer, PlanUpdater)
- Table-driven tests with t.Run() subtests
- Error wrapping with fmt.Errorf
- Context-based timeouts and cancellation
- File locking for concurrent safety (internal/filelock)
- Configuration merging (config → CLI flags override)

## Worktree Groups

This plan uses worktree groups for parallel execution with git worktrees.

**Group chain-1**: Database Foundation Chain
- **Tasks**: Task 1, Task 2, Task 3, Task 4, Task 5
- **Branch**: `feature/adaptive-learning-system/chain-1`
- **Execution**: Sequential (dependencies enforced)
- **Isolation**: Separate worktree
- **Rationale**: Database schema, store implementation, and schema tests must be completed sequentially to build the learning foundation

**Group chain-2**: Analysis & Pattern Detection Chain
- **Tasks**: Task 6, Task 7, Task 8
- **Branch**: `feature/adaptive-learning-system/chain-2`
- **Execution**: Sequential (depends on chain-1 completion)
- **Isolation**: Separate worktree
- **Rationale**: Failure analysis depends on database schema being in place

**Group chain-3**: Hook Integration Chain
- **Tasks**: Task 9, Task 10, Task 11, Task 12
- **Branch**: `feature/adaptive-learning-system/chain-3`
- **Execution**: Sequential (depends on chain-2 completion)
- **Isolation**: Separate worktree
- **Rationale**: Hook integration requires learning store and analysis logic to be functional

**Group chain-4**: CLI Commands Chain
- **Tasks**: Task 13, Task 14, Task 15, Task 16, Task 17
- **Branch**: `feature/adaptive-learning-system/chain-4`
- **Execution**: Sequential (depends on chain-1 completion, can run parallel with chain-2/chain-3)
- **Isolation**: Separate worktree
- **Rationale**: CLI commands only need database layer, independent of hook logic

**Group chain-5**: Orchestrator Integration Chain
- **Tasks**: Task 18, Task 19, Task 20
- **Branch**: `feature/adaptive-learning-system/chain-5`
- **Execution**: Sequential (depends on chain-1, chain-3 completion)
- **Isolation**: Separate worktree
- **Rationale**: Orchestrator integration ties together all components

**Group independent-1**: Configuration Extension
- **Tasks**: Task 21
- **Branch**: `feature/adaptive-learning-system/independent-1`
- **Execution**: Parallel-safe (depends only on chain-1)
- **Isolation**: Separate worktree
- **Rationale**: Config changes are isolated and can run independently

**Group independent-2**: Documentation
- **Tasks**: Task 22
- **Branch**: `feature/adaptive-learning-system/independent-2`
- **Execution**: Parallel-safe (depends on chain-5 completion)
- **Isolation**: Separate worktree
- **Rationale**: Documentation can be written after all features are implemented

**Group chain-6**: Integration Testing Chain
- **Tasks**: Task 23, Task 24, Task 25
- **Branch**: `feature/adaptive-learning-system/chain-6`
- **Execution**: Sequential (depends on all previous tasks)
- **Isolation**: Separate worktree
- **Rationale**: Integration tests validate entire feature working together

### Worktree Management

**Creating worktrees**:
```bash
# For each group, create a worktree
git worktree add ../conductor-chain-1 -b feature/adaptive-learning-system/chain-1
git worktree add ../conductor-chain-2 -b feature/adaptive-learning-system/chain-2
git worktree add ../conductor-chain-3 -b feature/adaptive-learning-system/chain-3
git worktree add ../conductor-chain-4 -b feature/adaptive-learning-system/chain-4
git worktree add ../conductor-chain-5 -b feature/adaptive-learning-system/chain-5
git worktree add ../conductor-independent-1 -b feature/adaptive-learning-system/independent-1
git worktree add ../conductor-independent-2 -b feature/adaptive-learning-system/independent-2
git worktree add ../conductor-chain-6 -b feature/adaptive-learning-system/chain-6
```

**Switching between worktrees**:
```bash
cd ../conductor-chain-1  # Work on database foundation
cd ../conductor-chain-4  # Work on CLI commands (can happen in parallel)
```

**Cleaning up after merge**:
```bash
git worktree remove ../conductor-chain-1
git branch -d feature/adaptive-learning-system/chain-1
# Repeat for all worktrees
```

## Prerequisites Checklist

Before starting, verify:

- [ ] Go 1.21+ installed (`go version`)
- [ ] SQLite development headers installed (for mattn/go-sqlite3 CGO compilation)
  - macOS: `brew install sqlite3`
  - Ubuntu: `sudo apt-get install libsqlite3-dev`
  - Already included on most systems
- [ ] Development environment working (`go test ./... ` passes)
- [ ] Git repository on `main` branch
- [ ] All existing tests passing (86.4% coverage baseline)
- [ ] Access to write to `.conductor/learning/` directory
- [ ] Understanding of conductor architecture (read CLAUDE.md, README.md)

---

## Task 1: Create Learning Package Structure with Schema Definition

**Agent**: golang-pro
**File(s)**: `internal/learning/schema.sql`, `internal/learning/doc.go`
**Depends on**: None
**Worktree Group**: chain-1
**Estimated time**: 30m

### What you're building

The foundation of the learning system: SQL schema definition for tracking task executions and approach history. This schema will be embedded in the binary using `//go:embed` and used to initialize SQLite databases. The schema includes tables for storing execution history, failure patterns, and learning metadata with proper indexes for fast queries.

### Test First (TDD)

**Test file**: `internal/learning/schema_test.go`

**Test structure**:
```go
// Test schema SQL syntax is valid
func TestSchema_ValidSQL(t *testing.T)
  - Parse schema.sql content
  - Verify no syntax errors
  - Check all tables are created

// Test schema includes required tables
func TestSchema_RequiredTables(t *testing.T)
  - Verify task_executions table exists
  - Verify approach_history table exists
  - Verify schema_version table exists

// Test schema includes required indexes
func TestSchema_RequiredIndexes(t *testing.T)
  - Verify indexes on (plan_file, task_number)
  - Verify indexes on qc_verdict
  - Verify index on session_id
```

**Test specifics**:
- Mock: None (test actual SQL parsing)
- Use: `database/sql` with `:memory:` SQLite database
- Assert: Schema creates tables successfully, indexes exist
- Edge cases: Empty schema, malformed SQL, missing tables

**Example test skeleton**:
```go
package learning

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchema_ValidSQL(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Execute schema
	_, err = db.Exec(schemaSQL)
	require.NoError(t, err, "schema SQL should be valid")
}

func TestSchema_RequiredTables(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(schemaSQL)
	require.NoError(t, err)

	// Query for tables
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	require.NoError(t, err)
	defer rows.Close()

	tables := make([]string, 0)
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		tables = append(tables, name)
	}

	assert.Contains(t, tables, "task_executions")
	assert.Contains(t, tables, "approach_history")
	assert.Contains(t, tables, "schema_version")
}

func TestSchema_RequiredIndexes(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(schemaSQL)
	require.NoError(t, err)

	// Query for indexes
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='index'")
	require.NoError(t, err)
	defer rows.Close()

	indexes := make([]string, 0)
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		indexes = append(indexes, name)
	}

	assert.Contains(t, indexes, "idx_task_lookup")
	assert.Contains(t, indexes, "idx_session")
	assert.Contains(t, indexes, "idx_failures")
}
```

### Implementation

**Approach**:
Create the SQL schema file and package documentation. The schema includes three tables: `task_executions` (complete execution history), `approach_history` (track tried approaches), and `schema_version` (for future migrations).

**File 1: internal/learning/schema.sql**:
```sql
-- Learning System Database Schema v1
-- Stores task execution history, failure patterns, and learning metadata

CREATE TABLE IF NOT EXISTS task_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- Task identification
    plan_file TEXT NOT NULL,
    task_number TEXT NOT NULL,
    task_name TEXT NOT NULL,
    task_prompt TEXT NOT NULL,

    -- Execution context
    session_id TEXT NOT NULL,
    run_number INTEGER DEFAULT 1,
    attempt_number INTEGER DEFAULT 1,

    -- Agent details
    agent_used TEXT,
    agent_source TEXT,

    -- Results
    qc_verdict TEXT,
    execution_time_ms INTEGER,
    error_message TEXT,
    output_summary TEXT,

    -- Learning data
    failure_patterns TEXT,
    success_indicators TEXT,

    -- Timestamps
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Unique constraint per execution
    UNIQUE(plan_file, task_number, session_id, run_number, attempt_number)
);

CREATE INDEX IF NOT EXISTS idx_task_lookup
    ON task_executions(plan_file, task_number);

CREATE INDEX IF NOT EXISTS idx_session
    ON task_executions(session_id);

CREATE INDEX IF NOT EXISTS idx_failures
    ON task_executions(qc_verdict)
    WHERE qc_verdict IN ('RED', 'YELLOW');

CREATE TABLE IF NOT EXISTS approach_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_file TEXT NOT NULL,
    task_number TEXT NOT NULL,

    -- What was tried
    agent_used TEXT,
    prompt_variation TEXT,
    approach_description TEXT,

    -- Outcome
    succeeded BOOLEAN,
    qc_verdict TEXT,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(plan_file, task_number, agent_used, prompt_variation)
);

CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO schema_version (version) VALUES (1);
```

**File 2: internal/learning/doc.go**:
```go
// Package learning implements adaptive learning capabilities for Conductor.
//
// The learning system tracks task execution history across conductor runs,
// analyzes failure patterns, and automatically adapts agent selection and
// task prompts based on what has worked (or failed) previously.
//
// Key components:
//   - Store: SQLite-backed storage for execution history
//   - Analysis: Failure pattern detection and agent suggestion
//   - Hooks: Integration points in task executor for learning
//
// Database schema is embedded in the binary and initialized automatically.
package learning

import _ "embed"

//go:embed schema.sql
var schemaSQL string
```

**Key points**:
- Use `//go:embed` to embed SQL in binary (follows Go 1.16+ pattern)
- Schema includes composite unique constraint to prevent duplicate records
- Indexes optimize common queries (plan + task lookup, session tracking, failure filtering)
- TEXT fields for JSON arrays (failure_patterns, success_indicators)
- schema_version table enables future migrations

**Integration points**:
- Imports needed: `embed` package for schema embedding
- No external services yet (just schema definition)
- Config values: None (schema is static)

### Verification

**Manual testing**:
1. Run `go test ./internal/learning/` to verify schema tests pass
2. Check that `schemaSQL` variable is accessible: `go build ./internal/learning/`
3. Verify schema creates in-memory database without errors

**Automated tests**:
```bash
go test ./internal/learning/ -v -run TestSchema
```

**Expected output**:
```
=== RUN   TestSchema_ValidSQL
--- PASS: TestSchema_ValidSQL (0.00s)
=== RUN   TestSchema_RequiredTables
--- PASS: TestSchema_RequiredTables (0.00s)
=== RUN   TestSchema_RequiredIndexes
--- PASS: TestSchema_RequiredIndexes (0.00s)
PASS
```

### Commit

**Commit message**:
```
feat(learning): add database schema for adaptive learning

Create SQLite schema for tracking task execution history:
- task_executions: Complete execution logs with patterns
- approach_history: Track what approaches have been tried
- schema_version: Support future migrations

Schema embedded in binary using go:embed for portability.

Related to #adaptive-learning-system
```

**Files to commit**:
- `internal/learning/schema.sql`
- `internal/learning/doc.go`
- `internal/learning/schema_test.go`

---

## Task 2: Implement Learning Store with Database Initialization

**Agent**: golang-pro
**File(s)**: `internal/learning/store.go`
**Depends on**: Task 1
**Worktree Group**: chain-1
**Estimated time**: 1h 30m

### What you're building

The Store struct and methods for initializing SQLite databases and managing connections. This includes database path resolution (project-local at `.conductor/learning/`), schema initialization, connection management, and graceful error handling. The Store will be the foundation for all learning operations.

### Test First (TDD)

**Test file**: `internal/learning/store_test.go`

**Test structure**:
```go
// Test store creation and initialization
func TestNewStore_Success(t *testing.T)
  - Create store with temp database path
  - Verify database file created
  - Verify schema initialized
  - Verify schema_version table has version 1

// Test store handles missing directory
func TestNewStore_CreatesDirectory(t *testing.T)
  - Use non-existent directory path
  - Verify directory is created
  - Verify database is created

// Test store handles invalid path
func TestNewStore_InvalidPath(t *testing.T)
  - Use invalid path (e.g., /root/nope)
  - Expect error returned
  - Verify no database created

// Test close functionality
func TestStore_Close(t *testing.T)
  - Create store
  - Call Close()
  - Verify subsequent operations fail gracefully

// Test path resolution
func TestResolvePath_ProjectLocal(t *testing.T)
  - Test relative path resolution
  - Test absolute path handling
  - Test home directory expansion (~/)
```

**Test specifics**:
- Mock: None (use real temp directories)
- Use: `t.TempDir()` for isolated test databases
- Assert: Database exists, schema applied, version correct
- Edge cases: Missing dirs, invalid permissions, already initialized DB

**Example test skeleton**:
```go
package learning

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore_Success(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Verify database file exists
	assert.FileExists(t, dbPath)

	// Verify schema initialized
	var version int
	err = store.db.QueryRow("SELECT version FROM schema_version").Scan(&version)
	require.NoError(t, err)
	assert.Equal(t, 1, version)
}

func TestNewStore_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "subdir", "nested", "test.db")

	store, err := NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	assert.FileExists(t, dbPath)
}

func TestNewStore_InvalidPath(t *testing.T) {
	// Attempt to create in unwritable location
	_, err := NewStore("/root/nopermission/test.db")
	assert.Error(t, err)
}

func TestStore_Close(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	require.NoError(t, err)

	err = store.Close()
	assert.NoError(t, err)

	// Subsequent operations should fail gracefully
	err = store.Close()
	// Double close should be handled
}
```

### Implementation

**Approach**:
Create the Store struct with database connection, path resolution logic, and initialization methods. Follow patterns from `internal/filelock/filelock.go` for file handling and `internal/config/config.go` for path resolution.

**Code structure**:
```go
package learning

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// Store manages the learning database for execution history and analysis.
type Store struct {
	db     *sql.DB
	dbPath string
}

// NewStore creates and initializes a new learning database.
// Creates the directory structure if it doesn't exist.
// Applies the embedded schema if database is new.
func NewStore(dbPath string) (*Store, error) {
	// Resolve path (handle ~/ expansion, relative paths)
	resolvedPath, err := resolvePath(dbPath)
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}

	// Create directory if needed
	dir := filepath.Dir(resolvedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create learning directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite3", resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Initialize schema
	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize schema: %w", err)
	}

	return &Store{
		db:     db,
		dbPath: resolvedPath,
	}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// initSchema applies the embedded SQL schema to the database.
func initSchema(db *sql.DB) error {
	_, err := db.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("execute schema: %w", err)
	}
	return nil
}

// resolvePath resolves a database path, handling ~/, relative, and absolute paths.
func resolvePath(dbPath string) (string, error) {
	// Handle empty path
	if dbPath == "" {
		return "", fmt.Errorf("database path cannot be empty")
	}

	// Handle home directory expansion
	if strings.HasPrefix(dbPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home directory: %w", err)
		}
		dbPath = filepath.Join(home, dbPath[2:])
	}

	// Handle relative paths - convert to absolute
	if !filepath.IsAbs(dbPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}
		dbPath = filepath.Join(cwd, dbPath)
	}

	return dbPath, nil
}
```

**Key points**:
- Follow error wrapping pattern: `fmt.Errorf("context: %w", err)`
- Use `os.MkdirAll` with 0755 permissions (consistent with .conductor/ directory)
- SQLite connection string is just the file path
- Schema initialization uses embedded `schemaSQL` from Task 1
- Close() is idempotent (safe to call multiple times)

**Integration points**:
- Imports needed: `database/sql`, `github.com/mattn/go-sqlite3`, `os`, `path/filepath`, `strings`
- Services to inject: None (Store is self-contained)
- Config values: dbPath (passed to NewStore)

### Verification

**Manual testing**:
1. Run tests: `go test ./internal/learning/ -v -run TestNewStore`
2. Create a store manually and inspect database:
```go
store, _ := NewStore(".conductor/learning/test.db")
defer store.Close()
// Use sqlite3 CLI to inspect: sqlite3 .conductor/learning/test.db ".tables"
```
3. Verify directory creation works
4. Verify ~/ expansion works

**Automated tests**:
```bash
go test ./internal/learning/ -v -run TestNewStore
go test ./internal/learning/ -v -run TestStore_Close
```

**Expected output**:
```
=== RUN   TestNewStore_Success
--- PASS: TestNewStore_Success (0.01s)
=== RUN   TestNewStore_CreatesDirectory
--- PASS: TestNewStore_CreatesDirectory (0.01s)
=== RUN   TestNewStore_InvalidPath
--- PASS: TestNewStore_InvalidPath (0.00s)
=== RUN   TestStore_Close
--- PASS: TestStore_Close (0.00s)
PASS
coverage: 89.3% of statements
```

### Commit

**Commit message**:
```
feat(learning): implement Store with database initialization

Add Store struct for managing learning SQLite databases:
- Database path resolution (relative, absolute, ~/ expansion)
- Directory creation for .conductor/learning/
- Schema initialization using embedded SQL
- Connection management with graceful Close()

Follows existing patterns from filelock and config packages.

Related to #adaptive-learning-system
```

**Files to commit**:
- `internal/learning/store.go`
- `internal/learning/store_test.go`
- `go.mod` (updated with mattn/go-sqlite3 dependency)
- `go.sum` (updated checksums)

---

## Task 3: Implement TaskExecution Model and Recording

**Agent**: golang-pro
**File(s)**: `internal/learning/store.go` (extend), `internal/learning/models.go`
**Depends on**: Task 2
**Worktree Group**: chain-1
**Estimated time**: 1h

### What you're building

Data models for TaskExecution and the RecordExecution method for storing execution history. This includes JSON marshaling for failure patterns, proper timestamp handling, and transactional inserts. The TaskExecution struct captures all relevant execution metadata including agent used, QC verdict, timing, and learned patterns.

### Test First (TDD)

**Test file**: `internal/learning/store_test.go` (extend)

**Test structure**:
```go
// Test recording successful execution
func TestStore_RecordExecution_Success(t *testing.T)
  - Create TaskExecution with GREEN verdict
  - Call RecordExecution
  - Verify row inserted with correct data
  - Verify ID returned

// Test recording failed execution with patterns
func TestStore_RecordExecution_WithPatterns(t *testing.T)
  - Create TaskExecution with RED verdict
  - Include failure_patterns and error_message
  - Call RecordExecution
  - Verify patterns stored as JSON

// Test duplicate execution constraint
func TestStore_RecordExecution_DuplicateFails(t *testing.T)
  - Insert same execution twice (same plan, task, session, run, attempt)
  - Expect unique constraint error on second insert

// Test JSON marshaling for patterns
func TestTaskExecution_JSONMarshaling(t *testing.T)
  - Create TaskExecution with string array patterns
  - Verify marshals to JSON correctly
  - Verify unmarshals back to same data
```

**Test specifics**:
- Mock: None (use real temp database)
- Use: setupTestDB() helper (create temp store)
- Assert: Data inserted, can be queried back, JSON fields parse correctly
- Edge cases: Empty patterns, nil patterns, very long output, special characters

**Example test skeleton**:
```go
func setupTestDB(t *testing.T) *Store {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := NewStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return store
}

func TestStore_RecordExecution_Success(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	exec := &TaskExecution{
		PlanFile:        "test-plan.md",
		TaskNumber:      "Task 1",
		TaskName:        "Test Task",
		TaskPrompt:      "Do something",
		SessionID:       "session-test",
		RunNumber:       1,
		AttemptNumber:   1,
		AgentUsed:       "golang-pro",
		AgentSource:     "specified",
		QCVerdict:       "GREEN",
		ExecutionTimeMs: 1500,
		OutputSummary:   "Task completed successfully",
	}

	err := store.RecordExecution(ctx, exec)
	require.NoError(t, err)

	// Verify data inserted
	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM task_executions WHERE plan_file = ?", "test-plan.md").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestStore_RecordExecution_WithPatterns(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	exec := &TaskExecution{
		PlanFile:        "test-plan.md",
		TaskNumber:      "Task 2",
		TaskName:        "Failing Task",
		TaskPrompt:      "Do something that fails",
		SessionID:       "session-test",
		RunNumber:       1,
		AttemptNumber:   1,
		AgentUsed:       "golang-pro",
		AgentSource:     "specified",
		QCVerdict:       "RED",
		ExecutionTimeMs: 2500,
		ErrorMessage:    "compilation failed",
		FailurePatterns: []string{"compilation_error", "syntax_error"},
	}

	err := store.RecordExecution(ctx, exec)
	require.NoError(t, err)

	// Verify patterns stored as JSON
	var patternsJSON string
	err = store.db.QueryRow("SELECT failure_patterns FROM task_executions WHERE task_number = ?", "Task 2").Scan(&patternsJSON)
	require.NoError(t, err)

	var patterns []string
	err = json.Unmarshal([]byte(patternsJSON), &patterns)
	require.NoError(t, err)
	assert.Equal(t, []string{"compilation_error", "syntax_error"}, patterns)
}
```

### Implementation

**Approach**:
Create TaskExecution model with proper JSON handling and implement RecordExecution method. Follow patterns from `internal/models/task.go` for struct design.

**File 1: internal/learning/models.go**:
```go
package learning

import (
	"time"
)

// TaskExecution represents a single task execution record in the learning database.
type TaskExecution struct {
	ID int64

	// Task identification
	PlanFile   string
	TaskNumber string
	TaskName   string
	TaskPrompt string

	// Execution context
	SessionID     string
	RunNumber     int
	AttemptNumber int

	// Agent details
	AgentUsed   string // Which agent executed this task
	AgentSource string // "specified" | "registry" | "learned"

	// Results
	QCVerdict       string // "GREEN" | "RED" | "YELLOW"
	ExecutionTimeMs int64
	ErrorMessage    string
	OutputSummary   string

	// Learning data
	FailurePatterns    []string // JSON: patterns detected in failures
	SuccessIndicators  []string // JSON: patterns detected in successes

	ExecutedAt time.Time
}
```

**File 2: internal/learning/store.go (extend)**:
```go
// RecordExecution stores a task execution record in the learning database.
func (s *Store) RecordExecution(ctx context.Context, exec *TaskExecution) error {
	if exec == nil {
		return fmt.Errorf("execution cannot be nil")
	}

	// Marshal pattern slices to JSON
	patternsJSON, err := json.Marshal(exec.FailurePatterns)
	if err != nil {
		return fmt.Errorf("marshal failure patterns: %w", err)
	}

	indicatorsJSON, err := json.Marshal(exec.SuccessIndicators)
	if err != nil {
		return fmt.Errorf("marshal success indicators: %w", err)
	}

	query := `
		INSERT INTO task_executions (
			plan_file, task_number, task_name, task_prompt,
			session_id, run_number, attempt_number,
			agent_used, agent_source, qc_verdict,
			execution_time_ms, error_message, output_summary,
			failure_patterns, success_indicators
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.ExecContext(ctx, query,
		exec.PlanFile, exec.TaskNumber, exec.TaskName, exec.TaskPrompt,
		exec.SessionID, exec.RunNumber, exec.AttemptNumber,
		exec.AgentUsed, exec.AgentSource, exec.QCVerdict,
		exec.ExecutionTimeMs, exec.ErrorMessage, exec.OutputSummary,
		string(patternsJSON), string(indicatorsJSON),
	)
	if err != nil {
		return fmt.Errorf("insert execution: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get insert id: %w", err)
	}

	exec.ID = id
	return nil
}
```

**Key points**:
- Use context.Context for cancellation support
- Marshal string slices to JSON for storage
- Handle nil slices (empty JSON array)
- Return inserted ID by mutating exec.ID
- Error wrapping with descriptive context

**Integration points**:
- Imports needed: `context`, `encoding/json`, `time`
- Services to inject: None
- Config values: None

### Verification

**Manual testing**:
1. Run tests: `go test ./internal/learning/ -v -run TestStore_RecordExecution`
2. Manually insert and query:
```bash
# After running tests, inspect database
cd /tmp && sqlite3 <test-db-path>
SELECT * FROM task_executions;
SELECT json_extract(failure_patterns, '$[0]') FROM task_executions;
```

**Automated tests**:
```bash
go test ./internal/learning/ -v -run TestStore_RecordExecution
go test ./internal/learning/ -v -run TestTaskExecution
```

**Expected output**:
```
=== RUN   TestStore_RecordExecution_Success
--- PASS: TestStore_RecordExecution_Success (0.01s)
=== RUN   TestStore_RecordExecution_WithPatterns
--- PASS: TestStore_RecordExecution_WithPatterns (0.01s)
=== RUN   TestStore_RecordExecution_DuplicateFails
--- PASS: TestStore_RecordExecution_DuplicateFails (0.00s)
PASS
coverage: 91.2% of statements
```

### Commit

**Commit message**:
```
feat(learning): add TaskExecution model and recording

Implement data models and database operations:
- TaskExecution struct with JSON pattern arrays
- RecordExecution() method for storing history
- Context support for cancellation
- Unique constraint enforcement

JSON marshaling handles failure patterns and success indicators.

Related to #adaptive-learning-system
```

**Files to commit**:
- `internal/learning/models.go`
- `internal/learning/store.go`
- `internal/learning/store_test.go`

---

## Task 4: Implement Session and Run Number Tracking

**Agent**: golang-pro
**File(s)**: `internal/learning/store.go` (extend)
**Depends on**: Task 3
**Worktree Group**: chain-1
**Estimated time**: 45m

### What you're building

Helper methods for tracking sessions and run numbers. Includes GetRunCount (how many times a plan has been run) and session ID generation logic. This enables conductor to automatically increment run numbers and generate unique session IDs like "session-20250112-1430".

### Test First (TDD)

**Test file**: `internal/learning/store_test.go` (extend)

**Test structure**:
```go
// Test get run count for new plan
func TestStore_GetRunCount_NewPlan(t *testing.T)
  - Query run count for plan that hasn't been executed
  - Expect count of 0

// Test get run count with executions
func TestStore_GetRunCount_WithExecutions(t *testing.T)
  - Insert 3 executions with different run_number (1, 2, 3)
  - Call GetRunCount
  - Expect count of 3

// Test get run count with multiple tasks
func TestStore_GetRunCount_MultipleTasks(t *testing.T)
  - Insert executions for multiple tasks in same run
  - Verify only counts distinct runs (not tasks)

// Test session ID generation format
func TestGenerateSessionID_Format(t *testing.T)
  - Generate session ID
  - Verify format: session-YYYYMMDD-HHMM
  - Verify uniqueness (generate multiple IDs)
```

**Test specifics**:
- Mock: None (use real database)
- Use: setupTestDB() helper
- Assert: Run counts accurate, session IDs formatted correctly
- Edge cases: No executions, many runs, concurrent session generation

**Example test skeleton**:
```go
func TestStore_GetRunCount_NewPlan(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	count := store.GetRunCount(ctx, "new-plan.md")
	assert.Equal(t, 0, count)
}

func TestStore_GetRunCount_WithExecutions(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	// Insert executions across 3 runs
	for run := 1; run <= 3; run++ {
		exec := &TaskExecution{
			PlanFile:      "test-plan.md",
			TaskNumber:    "Task 1",
			TaskName:      "Test",
			TaskPrompt:    "Test prompt",
			SessionID:     fmt.Sprintf("session-%d", run),
			RunNumber:     run,
			AttemptNumber: 1,
			AgentUsed:     "golang-pro",
			QCVerdict:     "GREEN",
		}
		err := store.RecordExecution(ctx, exec)
		require.NoError(t, err)
	}

	count := store.GetRunCount(ctx, "test-plan.md")
	assert.Equal(t, 3, count)
}

func TestGenerateSessionID_Format(t *testing.T) {
	id1 := GenerateSessionID()
	id2 := GenerateSessionID()

	// Verify format
	assert.Regexp(t, `^session-\d{8}-\d{4}$`, id1)
	assert.Regexp(t, `^session-\d{8}-\d{4}$`, id2)

	// IDs should be unique (or at least different if generated rapidly)
	// Note: may be same if generated in same minute
	t.Logf("ID1: %s, ID2: %s", id1, id2)
}
```

### Implementation

**Approach**:
Add query methods for run tracking and session ID generation. Follow simple, efficient SQL queries.

**Code structure** (add to internal/learning/store.go):
```go
// GetRunCount returns the number of distinct runs for a given plan file.
// Used to determine the next run number.
func (s *Store) GetRunCount(ctx context.Context, planFile string) int {
	query := `
		SELECT COALESCE(MAX(run_number), 0)
		FROM task_executions
		WHERE plan_file = ?
	`

	var count int
	err := s.db.QueryRowContext(ctx, query, planFile).Scan(&count)
	if err != nil {
		// If error, return 0 (assume no runs)
		return 0
	}

	return count
}

// GenerateSessionID creates a unique session identifier.
// Format: session-YYYYMMDD-HHMM
func GenerateSessionID() string {
	now := time.Now()
	return fmt.Sprintf("session-%s", now.Format("20060102-1504"))
}
```

**Key points**:
- Use COALESCE to handle NULL when no executions exist
- GetRunCount returns max run_number (not count of rows)
- Session ID format matches timestamp for easy sorting/identification
- Session ID generation is stateless (no database query needed)
- Error in GetRunCount returns 0 (graceful degradation)

**Integration points**:
- Imports needed: `context`, `time`, `fmt`
- Called by: Orchestrator during initialization
- Returns: Simple int for run count, string for session ID

### Verification

**Manual testing**:
1. Run tests: `go test ./internal/learning/ -v -run TestStore_GetRunCount`
2. Test session ID format:
```go
id := GenerateSessionID()
fmt.Println(id) // Should print: session-20250112-1430
```
3. Verify run count increments correctly

**Automated tests**:
```bash
go test ./internal/learning/ -v -run TestStore_GetRunCount
go test ./internal/learning/ -v -run TestGenerateSessionID
```

**Expected output**:
```
=== RUN   TestStore_GetRunCount_NewPlan
--- PASS: TestStore_GetRunCount_NewPlan (0.00s)
=== RUN   TestStore_GetRunCount_WithExecutions
--- PASS: TestStore_GetRunCount_WithExecutions (0.01s)
=== RUN   TestGenerateSessionID_Format
--- PASS: TestGenerateSessionID_Format (0.00s)
PASS
```

### Commit

**Commit message**:
```
feat(learning): add session and run number tracking

Implement methods for tracking execution sessions:
- GetRunCount: Query max run number for plan file
- GenerateSessionID: Create unique session identifiers

Session ID format: session-YYYYMMDD-HHMM for easy identification.
Run numbers increment automatically across conductor runs.

Related to #adaptive-learning-system
```

**Files to commit**:
- `internal/learning/store.go`
- `internal/learning/store_test.go`

---

## Task 5: Add Database Initialization Guards and Error Handling

**Agent**: golang-pro
**File(s)**: `internal/learning/store.go` (extend)
**Depends on**: Task 4
**Worktree Group**: chain-1
**Estimated time**: 30m

### What you're building

Error handling improvements and initialization guards to make the learning system robust. Includes checking if database is already initialized, handling nil Store gracefully throughout, and ensuring all Store methods have proper error handling that doesn't break conductor execution.

### Test First (TDD)

**Test file**: `internal/learning/store_test.go` (extend)

**Test structure**:
```go
// Test repeated initialization is idempotent
func TestStore_Init_Idempotent(t *testing.T)
  - Initialize store twice
  - Verify no errors
  - Verify schema_version still has single row

// Test nil store operations don't panic
func TestStore_NilOperations(t *testing.T)
  - Test RecordExecution with nil store
  - Test GetRunCount with nil store
  - Verify graceful handling (no panics)

// Test database connection errors
func TestStore_ConnectionError(t *testing.T)
  - Simulate connection failure
  - Verify error is wrapped and descriptive

// Test context cancellation handling
func TestStore_ContextCancellation(t *testing.T)
  - Create context with timeout
  - Cancel context during operation
  - Verify error returned, no hanging
```

**Test specifics**:
- Mock: Context cancellation for timeout tests
- Use: Invalid paths, closed connections
- Assert: No panics, errors are descriptive, graceful degradation
- Edge cases: Nil store, cancelled context, closed database

**Example test skeleton**:
```go
func TestStore_Init_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Initialize first time
	store1, err := NewStore(dbPath)
	require.NoError(t, err)
	store1.Close()

	// Initialize again (same path)
	store2, err := NewStore(dbPath)
	require.NoError(t, err)
	defer store2.Close()

	// Verify schema_version has single row
	var count int
	err = store2.db.QueryRow("SELECT COUNT(*) FROM schema_version").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestStore_ContextCancellation(t *testing.T) {
	store := setupTestDB(t)

	// Create context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	exec := &TaskExecution{
		PlanFile:   "test.md",
		TaskNumber: "Task 1",
		SessionID:  "test",
		RunNumber:  1,
	}

	err := store.RecordExecution(ctx, exec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}
```

### Implementation

**Approach**:
Add defensive checks and improve error handling. Make all operations safe with nil Store (return benign errors instead of panicking).

**Code additions** (to internal/learning/store.go):
```go
// Safe wrapper methods that handle nil Store

// RecordExecution (update to handle nil store)
func (s *Store) RecordExecution(ctx context.Context, exec *TaskExecution) error {
	if s == nil || s.db == nil {
		// Graceful degradation: if store is nil, log and return nil
		// This allows conductor to continue even if learning fails
		return nil
	}

	if exec == nil {
		return fmt.Errorf("execution cannot be nil")
	}

	// Check context before expensive operations
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before recording: %w", err)
	}

	// ... rest of existing implementation ...
}

// GetRunCount (update to handle nil store)
func (s *Store) GetRunCount(ctx context.Context, planFile string) int {
	if s == nil || s.db == nil {
		return 0 // Safe default
	}

	// ... rest of existing implementation ...
}

// initSchema (update to be idempotent)
func initSchema(db *sql.DB) error {
	// Schema uses IF NOT EXISTS, so it's already idempotent
	// But let's verify schema_version
	_, err := db.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("execute schema: %w", err)
	}

	// Verify initialization
	var version int
	err = db.QueryRow("SELECT version FROM schema_version").Scan(&version)
	if err != nil {
		return fmt.Errorf("verify schema version: %w", err)
	}

	if version != 1 {
		return fmt.Errorf("unexpected schema version: got %d, want 1", version)
	}

	return nil
}
```

**Key points**:
- All Store methods check for nil receiver (s == nil)
- RecordExecution checks context before expensive operations
- initSchema verifies initialization succeeded
- Errors are descriptive and wrapped
- Graceful degradation: nil store doesn't break conductor

**Integration points**:
- Error handling follows conductor patterns (return error, log warning)
- No panics allowed (learning failures shouldn't crash conductor)
- Context cancellation respected

### Verification

**Manual testing**:
1. Run all learning tests: `go test ./internal/learning/ -v`
2. Test nil store manually:
```go
var store *Store
err := store.RecordExecution(ctx, exec) // Should not panic
```
3. Verify coverage: `go test ./internal/learning/ -cover`

**Automated tests**:
```bash
go test ./internal/learning/ -v -run TestStore_Init
go test ./internal/learning/ -v -run TestStore_Nil
go test ./internal/learning/ -v -run TestStore_Context
go test ./internal/learning/ -cover
```

**Expected output**:
```
=== RUN   TestStore_Init_Idempotent
--- PASS: TestStore_Init_Idempotent (0.01s)
=== RUN   TestStore_ContextCancellation
--- PASS: TestStore_ContextCancellation (0.00s)
PASS
coverage: 93.5% of statements in ./internal/learning
```

### Commit

**Commit message**:
```
feat(learning): add robust error handling and nil safety

Improve Store robustness:
- Nil store checks in all methods (no panics)
- Context cancellation handling
- Idempotent schema initialization
- Graceful degradation (learning failures don't break conductor)

All Store methods are now safe to call even if initialization failed.

Related to #adaptive-learning-system
```

**Files to commit**:
- `internal/learning/store.go`
- `internal/learning/store_test.go`

---

## Task 6: Implement FailureAnalysis Model and Query Logic

**Agent**: golang-pro
**File(s)**: `internal/learning/analysis.go`, `internal/learning/models.go` (extend)
**Depends on**: Task 5
**Worktree Group**: chain-2
**Estimated time**: 1h

### What you're building

The FailureAnalysis struct and AnalyzeFailures method that queries execution history to identify patterns. This method examines past attempts for a specific task, counts failures by agent, detects common patterns, and determines if a different agent should be tried. This is the core intelligence of the learning system.

### Test First (TDD)

**Test file**: `internal/learning/analysis_test.go`

**Test structure**:
```go
// Test analysis with no history
func TestStore_AnalyzeFailures_NoHistory(t *testing.T)
  - Query task with no executions
  - Expect FailureAnalysis with zero counts
  - ShouldTryDifferentAgent should be false

// Test analysis with single failure
func TestStore_AnalyzeFailures_SingleFailure(t *testing.T)
  - Insert 1 RED execution
  - Expect FailedAttempts = 1
  - ShouldTryDifferentAgent should be false (threshold is 2)

// Test analysis triggers agent switch at 2 failures
func TestStore_AnalyzeFailures_TwoFailures(t *testing.T)
  - Insert 2 RED executions with same agent
  - Expect ShouldTryDifferentAgent = true
  - Verify TriedAgents includes the failing agent

// Test pattern detection
func TestStore_AnalyzeFailures_CommonPatterns(t *testing.T)
  - Insert 3 failures with patterns: ["compilation_error", "syntax_error"]
  - Insert 1 failure with pattern: ["test_failure"]
  - Expect CommonPatterns includes patterns from 50%+ of failures
```

**Test specifics**:
- Mock: None (use test database)
- Use: setupTestDB(), helper to insert test executions
- Assert: Counts accurate, patterns detected, thresholds correct
- Edge cases: Mixed verdicts, no patterns, all patterns different

**Example test skeleton**:
```go
package learning

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_AnalyzeFailures_NoHistory(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	analysis, err := store.AnalyzeFailures(ctx, "new-plan.md", "Task 1")
	require.NoError(t, err)

	assert.Equal(t, 0, analysis.TotalAttempts)
	assert.Equal(t, 0, analysis.FailedAttempts)
	assert.False(t, analysis.ShouldTryDifferentAgent)
}

func TestStore_AnalyzeFailures_TwoFailures(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	// Insert 2 RED executions
	for i := 1; i <= 2; i++ {
		exec := &TaskExecution{
			PlanFile:        "test-plan.md",
			TaskNumber:      "Task 1",
			TaskName:        "Test Task",
			TaskPrompt:      "Test",
			SessionID:       fmt.Sprintf("session-%d", i),
			RunNumber:       i,
			AttemptNumber:   1,
			AgentUsed:       "backend-developer",
			AgentSource:     "specified",
			QCVerdict:       "RED",
			ErrorMessage:    "compilation failed",
			FailurePatterns: []string{"compilation_error"},
		}
		err := store.RecordExecution(ctx, exec)
		require.NoError(t, err)
	}

	analysis, err := store.AnalyzeFailures(ctx, "test-plan.md", "Task 1")
	require.NoError(t, err)

	assert.Equal(t, 2, analysis.TotalAttempts)
	assert.Equal(t, 2, analysis.FailedAttempts)
	assert.True(t, analysis.ShouldTryDifferentAgent)
	assert.Contains(t, analysis.TriedAgents, "backend-developer")
	assert.Contains(t, analysis.CommonPatterns, "compilation_error")
}
```

### Implementation

**Approach**:
Create FailureAnalysis model and implement the analysis query. Follow SQL patterns from existing conductor code (see `internal/executor/graph.go` for complex queries).

**File 1: internal/learning/models.go (extend)**:
```go
// FailureAnalysis contains analysis of past task execution failures.
type FailureAnalysis struct {
	TotalAttempts           int
	FailedAttempts          int
	TriedAgents             []string // Agents that have been used
	CommonPatterns          []string // Patterns appearing in 50%+ of failures
	LastError               string
	SuggestedAgent          string // Best alternative based on history
	SuggestedApproach       string // Human-readable recommendation
	ShouldTryDifferentAgent bool   // True if FailedAttempts >= 2
}
```

**File 2: internal/learning/analysis.go**:
```go
package learning

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// AnalyzeFailures examines past execution attempts for a task and identifies patterns.
// Returns analysis suggesting whether to try a different agent.
func (s *Store) AnalyzeFailures(ctx context.Context, planFile, taskNumber string) (*FailureAnalysis, error) {
	if s == nil || s.db == nil {
		return &FailureAnalysis{}, nil
	}

	analysis := &FailureAnalysis{
		TriedAgents:    make([]string, 0),
		CommonPatterns: make([]string, 0),
	}

	// Query all past executions for this task
	query := `
		SELECT
			agent_used,
			qc_verdict,
			failure_patterns,
			error_message,
			run_number,
			attempt_number
		FROM task_executions
		WHERE plan_file = ? AND task_number = ?
		ORDER BY executed_at DESC
		LIMIT 10
	`

	rows, err := s.db.QueryContext(ctx, query, planFile, taskNumber)
	if err != nil {
		return nil, fmt.Errorf("query executions: %w", err)
	}
	defer rows.Close()

	agentsSeen := make(map[string]bool)
	patternCounts := make(map[string]int)

	for rows.Next() {
		var agent, verdict, patternsJSON, errorMsg string
		var runNum, attemptNum int

		if err := rows.Scan(&agent, &verdict, &patternsJSON, &errorMsg, &runNum, &attemptNum); err != nil {
			continue
		}

		analysis.TotalAttempts++

		if verdict == "RED" || verdict == "YELLOW" {
			analysis.FailedAttempts++
			analysis.LastError = errorMsg

			// Track agents tried
			if !agentsSeen[agent] {
				analysis.TriedAgents = append(analysis.TriedAgents, agent)
				agentsSeen[agent] = true
			}

			// Count failure patterns
			var patterns []string
			if err := json.Unmarshal([]byte(patternsJSON), &patterns); err == nil {
				for _, pattern := range patterns {
					patternCounts[pattern]++
				}
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan rows: %w", err)
	}

	// Identify common patterns (appeared in 50%+ of failures)
	if analysis.FailedAttempts > 0 {
		threshold := analysis.FailedAttempts / 2
		for pattern, count := range patternCounts {
			if count >= threshold && count > 0 {
				analysis.CommonPatterns = append(analysis.CommonPatterns, pattern)
			}
		}
	}

	// Determine if we should try different agent
	analysis.ShouldTryDifferentAgent = analysis.FailedAttempts >= 2

	return analysis, nil
}
```

**Key points**:
- LIMIT 10 keeps query fast (only recent attempts matter)
- JSON unmarshal with error handling (graceful if patterns malformed)
- Pattern threshold: 50% of failures (simple heuristic)
- ShouldTryDifferentAgent threshold: 2 failures (configurable via const later)
- Distinct agent tracking with map

**Integration points**:
- Imports: `context`, `database/sql`, `encoding/json`, `fmt`
- Called by: Pre-task hook in executor
- Returns: FailureAnalysis struct

### Verification

**Manual testing**:
1. Run tests: `go test ./internal/learning/ -v -run TestStore_AnalyzeFailures`
2. Manually verify pattern detection:
```go
// Insert test data with patterns
// Query and inspect analysis.CommonPatterns
```

**Automated tests**:
```bash
go test ./internal/learning/ -v -run TestStore_AnalyzeFailures
go test ./internal/learning/ -cover
```

**Expected output**:
```
=== RUN   TestStore_AnalyzeFailures_NoHistory
--- PASS: TestStore_AnalyzeFailures_NoHistory (0.00s)
=== RUN   TestStore_AnalyzeFailures_TwoFailures
--- PASS: TestStore_AnalyzeFailures_TwoFailures (0.01s)
=== RUN   TestStore_AnalyzeFailures_CommonPatterns
--- PASS: TestStore_AnalyzeFailures_CommonPatterns (0.01s)
PASS
coverage: 94.1% of statements
```

### Commit

**Commit message**:
```
feat(learning): implement failure analysis and pattern detection

Add AnalyzeFailures method for intelligent task analysis:
- Query past executions for specific task
- Count failures by agent
- Detect common patterns (50%+ threshold)
- Determine if agent switch recommended (2+ failures)

FailureAnalysis struct provides actionable insights for adaptation.

Related to #adaptive-learning-system
```

**Files to commit**:
- `internal/learning/analysis.go`
- `internal/learning/models.go`
- `internal/learning/analysis_test.go`

---

## Task 7: Implement Agent Suggestion Algorithm

**Agent**: golang-pro
**File(s)**: `internal/learning/analysis.go` (extend)
**Depends on**: Task 6
**Worktree Group**: chain-2
**Estimated time**: 1h

### What you're building

The agent suggestion algorithm that queries execution history to find the best alternative agent when the current agent has failed multiple times. Uses success rates and historical data to make intelligent suggestions, with graceful fallbacks when no data exists.

### Test First (TDD)

**Test file**: `internal/learning/analysis_test.go` (extend)

**Test structure**:
```go
// Test agent suggestion with similar successful tasks
func TestStore_FindBestAgent_SimilarTasks(t *testing.T)
  - Insert GREEN executions for other tasks with golang-pro
  - Query best agent excluding backend-developer
  - Expect golang-pro suggested

// Test agent suggestion with overall success rates
func TestStore_FindBestAgent_OverallRates(t *testing.T)
  - Insert executions across multiple plans
  - Calculate success rates for each agent
  - Verify highest success rate agent suggested

// Test fallback to general-purpose
func TestStore_FindBestAgent_NoData(t *testing.T)
  - Query with no execution history
  - Expect "general-purpose" returned

// Test excludes already-tried agents
func TestStore_FindBestAgent_ExcludesTriedAgents(t *testing.T)
  - Have golang-pro with 100% success
  - Exclude golang-pro in query
  - Verify different agent suggested
```

**Test specifics**:
- Mock: None
- Use: setupTestDB(), insert varied test data
- Assert: Suggestions match expected logic, exclusions work
- Edge cases: All agents tried, no agents available, tie in success rates

**Example test skeleton**:
```go
func TestStore_FindBestAgent_SimilarTasks(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	// Insert successful executions with golang-pro for similar tasks
	for i := 2; i <= 5; i++ {
		exec := &TaskExecution{
			PlanFile:    "test-plan.md",
			TaskNumber:  fmt.Sprintf("Task %d", i),
			TaskName:    "Go Task",
			TaskPrompt:  "Implement Go feature",
			SessionID:   "session-1",
			RunNumber:   1,
			AgentUsed:   "golang-pro",
			QCVerdict:   "GREEN",
		}
		require.NoError(t, store.RecordExecution(ctx, exec))
	}

	// Find best agent excluding backend-developer
	agent := store.findBestAlternativeAgent(ctx, "test-plan.md", "Task 1", []string{"backend-developer"})
	assert.Equal(t, "golang-pro", agent)
}

func TestStore_FindBestAgent_NoData(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	// No execution history
	agent := store.findBestAlternativeAgent(ctx, "new-plan.md", "Task 1", []string{})
	assert.Equal(t, "general-purpose", agent)
}
```

### Implementation

**Approach**:
Implement two-tier suggestion algorithm: first try finding agents successful on similar tasks in same plan, then fall back to overall success rates across all plans.

**Code structure** (add to internal/learning/analysis.go):
```go
// findBestAlternativeAgent suggests a different agent based on historical success.
// First tries agents successful on similar tasks, then falls back to overall rates.
func (s *Store) findBestAlternativeAgent(ctx context.Context, planFile, taskNumber string, excludeAgents []string) string {
	if s == nil || s.db == nil {
		return "general-purpose"
	}

	// Build exclusion placeholders
	exclusions := make([]interface{}, 0)
	for _, agent := range excludeAgents {
		exclusions = append(exclusions, agent)
	}

	// Strategy 1: Find agents successful on similar tasks in this plan
	if agent := s.findAgentFromSimilarTasks(ctx, planFile, taskNumber, exclusions); agent != "" {
		return agent
	}

	// Strategy 2: Find agent with best overall success rate
	if agent := s.findAgentByOverallRate(ctx, exclusions); agent != "" {
		return agent
	}

	// Fallback: general-purpose agent
	return "general-purpose"
}

// findAgentFromSimilarTasks queries successful agents in the same plan.
func (s *Store) findAgentFromSimilarTasks(ctx context.Context, planFile, taskNumber string, exclusions []interface{}) string {
	// Build query with dynamic exclusion list
	query := `
		SELECT agent_used, COUNT(*) as success_count
		FROM task_executions
		WHERE plan_file = ?
		  AND qc_verdict = 'GREEN'
		  AND task_number != ?
	`

	args := []interface{}{planFile, taskNumber}

	if len(exclusions) > 0 {
		placeholders := make([]string, len(exclusions))
		for i := range exclusions {
			placeholders[i] = "?"
		}
		query += fmt.Sprintf(" AND agent_used NOT IN (%s)", strings.Join(placeholders, ","))
		args = append(args, exclusions...)
	}

	query += `
		GROUP BY agent_used
		ORDER BY success_count DESC
		LIMIT 1
	`

	var agent string
	var count int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&agent, &count)
	if err != nil {
		return ""
	}

	return agent
}

// findAgentByOverallRate finds agent with best success rate across all plans.
func (s *Store) findAgentByOverallRate(ctx context.Context, exclusions []interface{}) string {
	query := `
		SELECT agent_used,
		       SUM(CASE WHEN qc_verdict = 'GREEN' THEN 1 ELSE 0 END) * 100.0 / COUNT(*) as success_rate
		FROM task_executions
	`

	args := make([]interface{}, 0)

	if len(exclusions) > 0 {
		placeholders := make([]string, len(exclusions))
		for i := range exclusions {
			placeholders[i] = "?"
		}
		query += fmt.Sprintf(" WHERE agent_used NOT IN (%s)", strings.Join(placeholders, ","))
		args = append(args, exclusions...)
	}

	query += `
		GROUP BY agent_used
		HAVING COUNT(*) >= 5
		ORDER BY success_rate DESC
		LIMIT 1
	`

	var agent string
	var rate float64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&agent, &rate)
	if err != nil {
		return ""
	}

	return agent
}
```

**Key points**:
- Two-tier strategy: similar tasks → overall rates → fallback
- Dynamic exclusion list using SQL IN clause
- Minimum 5 executions for statistical significance (overall rate query)
- Success rate calculation: (GREEN count / total) * 100
- Empty string return signals "no suggestion" (caller handles fallback)

**Integration points**:
- Imports: `strings` for Join
- Called by: AnalyzeFailures (can integrate in Task 8)
- Returns: Agent name or "general-purpose"

### Verification

**Manual testing**:
1. Run tests: `go test ./internal/learning/ -v -run TestStore_FindBestAgent`
2. Verify SQL queries manually:
```sql
-- Test in sqlite3
SELECT agent_used, COUNT(*) FROM task_executions WHERE qc_verdict='GREEN' GROUP BY agent_used;
```

**Automated tests**:
```bash
go test ./internal/learning/ -v -run TestStore_FindBestAgent
go test ./internal/learning/ -cover
```

**Expected output**:
```
=== RUN   TestStore_FindBestAgent_SimilarTasks
--- PASS: TestStore_FindBestAgent_SimilarTasks (0.01s)
=== RUN   TestStore_FindBestAgent_OverallRates
--- PASS: TestStore_FindBestAgent_OverallRates (0.01s)
=== RUN   TestStore_FindBestAgent_NoData
--- PASS: TestStore_FindBestAgent_NoData (0.00s)
PASS
```

### Commit

**Commit message**:
```
feat(learning): add agent suggestion algorithm

Implement intelligent agent selection based on historical data:
- Two-tier strategy: similar tasks → overall success rates
- Dynamic agent exclusion to avoid re-trying failed agents
- Minimum sample size requirement (5 executions)
- Graceful fallback to general-purpose agent

Uses SQL success rate calculations for data-driven suggestions.

Related to #adaptive-learning-system
```

**Files to commit**:
- `internal/learning/analysis.go`
- `internal/learning/analysis_test.go`

---

## Task 8: Integrate Agent Suggestion into AnalyzeFailures

**Agent**: golang-pro
**File(s)**: `internal/learning/analysis.go` (extend)
**Depends on**: Task 7
**Worktree Group**: chain-2
**Estimated time**: 30m

### What you're building

Integration of the agent suggestion algorithm into the AnalyzeFailures method, plus generation of human-readable approach suggestions based on detected patterns. This completes the analysis layer by providing actionable recommendations that the pre-task hook can use.

### Test First (TDD)

**Test file**: `internal/learning/analysis_test.go` (extend)

**Test structure**:
```go
// Test full analysis with agent suggestion
func TestStore_AnalyzeFailures_WithSuggestion(t *testing.T)
  - Insert 2 RED executions with backend-developer
  - Insert GREEN executions with golang-pro on other tasks
  - Call AnalyzeFailures
  - Verify SuggestedAgent is golang-pro
  - Verify SuggestedApproach is generated

// Test approach suggestion for compilation errors
func TestGenerateApproachSuggestion_CompilationError(t *testing.T)
  - Create analysis with "compilation_error" pattern
  - Generate suggestion
  - Verify contains "syntax validation" or similar

// Test approach suggestion with no patterns
func TestGenerateApproachSuggestion_NoPatterns(t *testing.T)
  - Create analysis with empty patterns
  - Expect generic "different approach" suggestion
```

**Test specifics**:
- Mock: None
- Use: Existing test helpers
- Assert: Suggestions populated, approaches make sense
- Edge cases: No patterns, multiple patterns, unknown patterns

**Example test skeleton**:
```go
func TestStore_AnalyzeFailures_WithSuggestion(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	// Insert failures with backend-developer
	for i := 1; i <= 2; i++ {
		exec := &TaskExecution{
			PlanFile:        "test-plan.md",
			TaskNumber:      "Task 1",
			SessionID:       fmt.Sprintf("session-%d", i),
			RunNumber:       i,
			AgentUsed:       "backend-developer",
			QCVerdict:       "RED",
			FailurePatterns: []string{"compilation_error"},
		}
		require.NoError(t, store.RecordExecution(ctx, exec))
	}

	// Insert success with golang-pro on other task
	exec := &TaskExecution{
		PlanFile:   "test-plan.md",
		TaskNumber: "Task 2",
		SessionID:  "session-1",
		RunNumber:  1,
		AgentUsed:  "golang-pro",
		QCVerdict:  "GREEN",
	}
	require.NoError(t, store.RecordExecution(ctx, exec))

	// Analyze
	analysis, err := store.AnalyzeFailures(ctx, "test-plan.md", "Task 1")
	require.NoError(t, err)

	assert.True(t, analysis.ShouldTryDifferentAgent)
	assert.Equal(t, "golang-pro", analysis.SuggestedAgent)
	assert.NotEmpty(t, analysis.SuggestedApproach)
	assert.Contains(t, analysis.SuggestedApproach, "syntax")
}
```

### Implementation

**Approach**:
Extend AnalyzeFailures to call agent suggestion and generate approach text. Add helper for generating human-readable suggestions.

**Code additions** (to internal/learning/analysis.go):
```go
// Update AnalyzeFailures to include suggestions
func (s *Store) AnalyzeFailures(ctx context.Context, planFile, taskNumber string) (*FailureAnalysis, error) {
	// ... existing code up to pattern detection ...

	// Determine if we should try different agent
	analysis.ShouldTryDifferentAgent = analysis.FailedAttempts >= 2

	// If should adapt, get agent suggestion and approach
	if analysis.ShouldTryDifferentAgent {
		analysis.SuggestedAgent = s.findBestAlternativeAgent(ctx, planFile, taskNumber, analysis.TriedAgents)
		analysis.SuggestedApproach = generateApproachSuggestion(analysis)
	}

	return analysis, nil
}

// generateApproachSuggestion creates human-readable advice based on patterns.
func generateApproachSuggestion(analysis *FailureAnalysis) string {
	if len(analysis.CommonPatterns) == 0 {
		return "Try a fundamentally different implementation approach"
	}

	// Map patterns to suggestions
	suggestions := make([]string, 0)
	for _, pattern := range analysis.CommonPatterns {
		switch pattern {
		case "compilation_error":
			suggestions = append(suggestions, "Focus on syntax validation before implementation")
		case "test_failure":
			suggestions = append(suggestions, "Implement tests incrementally with the code")
		case "dependency_missing":
			suggestions = append(suggestions, "Verify all dependencies are installed first")
		case "permission_error":
			suggestions = append(suggestions, "Check file permissions and ownership")
		case "timeout":
			suggestions = append(suggestions, "Break task into smaller subtasks")
		case "runtime_error":
			suggestions = append(suggestions, "Add defensive null/nil checks and error handling")
		default:
			suggestions = append(suggestions, fmt.Sprintf("Address recurring pattern: %s", pattern))
		}
	}

	if len(suggestions) > 0 {
		// Return most relevant suggestion
		return suggestions[0]
	}

	return "Try a different implementation approach"
}
```

**Key points**:
- Agent suggestion only generated when ShouldTryDifferentAgent is true
- Approach suggestions map common patterns to actionable advice
- Returns first (most relevant) suggestion
- Generic fallback for unknown patterns
- Suggestions are brief, actionable, developer-friendly

**Integration points**:
- Called by: Pre-task hook
- Uses: Agent suggestion from Task 7
- Returns: Complete FailureAnalysis with all fields populated

### Verification

**Manual testing**:
1. Run full analysis tests: `go test ./internal/learning/ -v`
2. Test suggestion generation with various patterns
3. Verify coverage: `go test ./internal/learning/ -cover`

**Automated tests**:
```bash
go test ./internal/learning/ -v -run TestStore_AnalyzeFailures_WithSuggestion
go test ./internal/learning/ -v -run TestGenerateApproachSuggestion
go test ./internal/learning/ -cover
```

**Expected output**:
```
=== RUN   TestStore_AnalyzeFailures_WithSuggestion
--- PASS: TestStore_AnalyzeFailures_WithSuggestion (0.02s)
=== RUN   TestGenerateApproachSuggestion_CompilationError
--- PASS: TestGenerateApproachSuggestion_CompilationError (0.00s)
PASS
coverage: 95.3% of statements in ./internal/learning
```

### Commit

**Commit message**:
```
feat(learning): integrate agent and approach suggestions

Complete failure analysis with actionable recommendations:
- Populate SuggestedAgent using historical success data
- Generate human-readable SuggestedApproach from patterns
- Map common patterns to specific advice
- Generic fallback for unknown situations

AnalyzeFailures now provides complete intelligence for adaptation.

Related to #adaptive-learning-system
```

**Files to commit**:
- `internal/learning/analysis.go`
- `internal/learning/analysis_test.go`

---

