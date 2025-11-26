package learning

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Migration represents a database schema migration
type Migration struct {
	Version     int
	Description string
	SQL         string
}

// migrations is the ordered list of all database migrations
var migrations = []Migration{
	{
		Version:     1,
		Description: "Initial schema with task_executions and approach_history",
		SQL: `
-- Task execution history table
CREATE TABLE IF NOT EXISTS task_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_file TEXT,
    run_number INTEGER DEFAULT 1,
    task_number TEXT NOT NULL,
    task_name TEXT NOT NULL,
    agent TEXT,
    prompt TEXT NOT NULL,
    success BOOLEAN NOT NULL,
    output TEXT,
    error_message TEXT,
    duration_seconds INTEGER,
    qc_verdict TEXT,
    qc_feedback TEXT,
    failure_patterns TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    context TEXT
);

CREATE INDEX IF NOT EXISTS idx_task_executions_task ON task_executions(task_number);
CREATE INDEX IF NOT EXISTS idx_task_executions_timestamp ON task_executions(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_task_executions_success ON task_executions(success);
CREATE INDEX IF NOT EXISTS idx_task_executions_plan_file ON task_executions(plan_file);
CREATE INDEX IF NOT EXISTS idx_task_executions_run_number ON task_executions(run_number);

-- Approach history table
CREATE TABLE IF NOT EXISTS approach_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_pattern TEXT NOT NULL,
    approach_description TEXT NOT NULL,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    last_used TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT
);

CREATE INDEX IF NOT EXISTS idx_approach_history_task ON approach_history(task_pattern);
CREATE INDEX IF NOT EXISTS idx_approach_history_success ON approach_history(success_count DESC);
`,
	},
	{
		Version:     2,
		Description: "Add behavioral tables for detailed session tracking",
		SQL: `
-- Behavioral sessions table
CREATE TABLE IF NOT EXISTS behavioral_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_execution_id INTEGER NOT NULL,
    session_start TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    session_end TIMESTAMP,
    total_duration_seconds INTEGER,
    total_tool_calls INTEGER DEFAULT 0,
    total_bash_commands INTEGER DEFAULT 0,
    total_file_operations INTEGER DEFAULT 0,
    total_tokens_used INTEGER DEFAULT 0,
    context_window_used INTEGER DEFAULT 0,
    FOREIGN KEY (task_execution_id) REFERENCES task_executions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_behavioral_sessions_task_id ON behavioral_sessions(task_execution_id);
CREATE INDEX IF NOT EXISTS idx_behavioral_sessions_start ON behavioral_sessions(session_start DESC);

-- Tool executions table
CREATE TABLE IF NOT EXISTS tool_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL,
    tool_name TEXT NOT NULL,
    parameters TEXT,
    execution_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    duration_ms INTEGER,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    FOREIGN KEY (session_id) REFERENCES behavioral_sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tool_executions_session_id ON tool_executions(session_id);
CREATE INDEX IF NOT EXISTS idx_tool_executions_tool_name ON tool_executions(tool_name);
CREATE INDEX IF NOT EXISTS idx_tool_executions_success ON tool_executions(success);

-- Bash commands table
CREATE TABLE IF NOT EXISTS bash_commands (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL,
    command TEXT NOT NULL,
    execution_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    duration_ms INTEGER,
    exit_code INTEGER,
    stdout_length INTEGER,
    stderr_length INTEGER,
    success BOOLEAN NOT NULL,
    FOREIGN KEY (session_id) REFERENCES behavioral_sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_bash_commands_session_id ON bash_commands(session_id);
CREATE INDEX IF NOT EXISTS idx_bash_commands_success ON bash_commands(success);
CREATE INDEX IF NOT EXISTS idx_bash_commands_exit_code ON bash_commands(exit_code);

-- File operations table
CREATE TABLE IF NOT EXISTS file_operations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL,
    operation_type TEXT NOT NULL,
    file_path TEXT NOT NULL,
    execution_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    duration_ms INTEGER,
    bytes_affected INTEGER,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    FOREIGN KEY (session_id) REFERENCES behavioral_sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_file_operations_session_id ON file_operations(session_id);
CREATE INDEX IF NOT EXISTS idx_file_operations_type ON file_operations(operation_type);
CREATE INDEX IF NOT EXISTS idx_file_operations_path ON file_operations(file_path);

-- Token usage table
CREATE TABLE IF NOT EXISTS token_usage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL,
    measurement_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    input_tokens INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    context_window_size INTEGER,
    FOREIGN KEY (session_id) REFERENCES behavioral_sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_token_usage_session_id ON token_usage(session_id);
CREATE INDEX IF NOT EXISTS idx_token_usage_time ON token_usage(measurement_time DESC);
`,
	},
	{
		Version:     3,
		Description: "Add ingest_offsets for JSONL file tracking",
		SQL: `
-- Ingest offsets table for tracking JSONL file read positions
CREATE TABLE IF NOT EXISTS ingest_offsets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path TEXT NOT NULL UNIQUE,
    byte_offset INTEGER NOT NULL DEFAULT 0,
    inode INTEGER,
    last_line_hash TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ingest_offsets_path ON ingest_offsets(file_path);
`,
	},
	{
		Version:     4,
		Description: "Add external_session_id for Claude session correlation",
		// This migration adds columns to behavioral_sessions for incremental ingestion.
		// The actual ALTER TABLE statements are handled by ApplyMigrations() using
		// addColumnIfNotExists() to ensure idempotency.
		SQL: `
-- Create unique index for idempotent upserts by external session ID
CREATE UNIQUE INDEX IF NOT EXISTS idx_behavioral_sessions_external_id ON behavioral_sessions(external_session_id);

-- Index for project-based queries
CREATE INDEX IF NOT EXISTS idx_behavioral_sessions_project ON behavioral_sessions(project_path);
`,
	},
	{
		Version:     5,
		Description: "Backfill aggregate counters from detail tables",
		// This migration backfills total_tool_calls, total_bash_commands, and total_file_operations
		// from the detail tables for sessions that were ingested before counter increments were added.
		SQL: `
-- Backfill aggregate counters from detail tables
UPDATE behavioral_sessions SET
  total_tool_calls = (SELECT COUNT(*) FROM tool_executions WHERE tool_executions.session_id = behavioral_sessions.id),
  total_bash_commands = (SELECT COUNT(*) FROM bash_commands WHERE bash_commands.session_id = behavioral_sessions.id),
  total_file_operations = (SELECT COUNT(*) FROM file_operations WHERE file_operations.session_id = behavioral_sessions.id)
WHERE total_tool_calls = 0 AND total_bash_commands = 0 AND total_file_operations = 0;
`,
	},
	{
		Version:     6,
		Description: "Backfill session timestamps from tool_executions",
		// This migration backfills session_start, session_end, and total_duration_seconds
		// from tool_executions for sessions that have 0 duration (before timestamp tracking was added).
		SQL: `
-- Backfill session timestamps from tool_executions (most reliable source)
UPDATE behavioral_sessions SET
  session_start = COALESCE(
    (SELECT MIN(execution_time) FROM tool_executions WHERE tool_executions.session_id = behavioral_sessions.id),
    session_start
  ),
  session_end = COALESCE(
    (SELECT MAX(execution_time) FROM tool_executions WHERE tool_executions.session_id = behavioral_sessions.id),
    session_end
  ),
  total_duration_seconds = COALESCE(
    (SELECT CAST((julianday(MAX(execution_time)) - julianday(MIN(execution_time))) * 86400 AS INTEGER)
     FROM tool_executions WHERE tool_executions.session_id = behavioral_sessions.id),
    total_duration_seconds
  )
WHERE (total_duration_seconds = 0 OR total_duration_seconds IS NULL)
  AND EXISTS (SELECT 1 FROM tool_executions WHERE tool_executions.session_id = behavioral_sessions.id);

-- Also try bash_commands for sessions without tool_executions
UPDATE behavioral_sessions SET
  session_start = COALESCE(
    (SELECT MIN(execution_time) FROM bash_commands WHERE bash_commands.session_id = behavioral_sessions.id),
    session_start
  ),
  session_end = COALESCE(
    (SELECT MAX(execution_time) FROM bash_commands WHERE bash_commands.session_id = behavioral_sessions.id),
    session_end
  ),
  total_duration_seconds = COALESCE(
    (SELECT CAST((julianday(MAX(execution_time)) - julianday(MIN(execution_time))) * 86400 AS INTEGER)
     FROM bash_commands WHERE bash_commands.session_id = behavioral_sessions.id),
    total_duration_seconds
  )
WHERE (total_duration_seconds = 0 OR total_duration_seconds IS NULL)
  AND EXISTS (SELECT 1 FROM bash_commands WHERE bash_commands.session_id = behavioral_sessions.id);

-- Finally try file_operations
UPDATE behavioral_sessions SET
  session_start = COALESCE(
    (SELECT MIN(execution_time) FROM file_operations WHERE file_operations.session_id = behavioral_sessions.id),
    session_start
  ),
  session_end = COALESCE(
    (SELECT MAX(execution_time) FROM file_operations WHERE file_operations.session_id = behavioral_sessions.id),
    session_end
  ),
  total_duration_seconds = COALESCE(
    (SELECT CAST((julianday(MAX(execution_time)) - julianday(MIN(execution_time))) * 86400 AS INTEGER)
     FROM file_operations WHERE file_operations.session_id = behavioral_sessions.id),
    total_duration_seconds
  )
WHERE (total_duration_seconds = 0 OR total_duration_seconds IS NULL)
  AND EXISTS (SELECT 1 FROM file_operations WHERE file_operations.session_id = behavioral_sessions.id);

-- Delete sessions with no events and no duration (orphaned/empty)
DELETE FROM behavioral_sessions
WHERE (total_duration_seconds = 0 OR total_duration_seconds IS NULL)
  AND total_tool_calls = 0
  AND total_bash_commands = 0
  AND total_file_operations = 0;
`,
	},
	{
		Version:     7,
		Description: "Add model name column to behavioral_sessions for cost calculation",
		// This migration adds a model_name column to track the model used in each session.
		// Required for cost calculation based on ModelCosts pricing.
		SQL: ``,
	},
}

// MigrationVersion represents a record of an applied migration
type MigrationVersion struct {
	Version   int
	AppliedAt time.Time
}

// ApplyMigrations applies all pending migrations to the database.
// Uses EXCLUSIVE transaction to serialize concurrent initialization.
func (s *Store) ApplyMigrations(ctx context.Context) error {
	// Start EXCLUSIVE transaction to serialize concurrent migrations.
	// This ensures only one connection can run migrations at a time.
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("begin exclusive transaction: %w", err)
	}
	defer tx.Rollback() // no-op if committed

	// Ensure schema_version table exists (within transaction)
	if err := s.ensureSchemaVersionTableTx(tx); err != nil {
		return fmt.Errorf("ensure schema_version table: %w", err)
	}

	// Get currently applied versions
	appliedVersions, err := s.getAppliedVersionsTx(tx)
	if err != nil {
		return fmt.Errorf("get applied versions: %w", err)
	}

	// Build map of applied versions for quick lookup
	applied := make(map[int]bool)
	for _, v := range appliedVersions {
		applied[v.Version] = true
	}

	// Apply pending migrations
	for _, migration := range migrations {
		if applied[migration.Version] {
			continue
		}

		// Handle migration 4 special case: add columns idempotently
		if migration.Version == 4 {
			if err := s.applyMigration4Tx(ctx, tx); err != nil {
				return fmt.Errorf("apply migration %d (%s): %w", migration.Version, migration.Description, err)
			}
		}

		// Handle migration 7 special case: add model_name column idempotently
		if migration.Version == 7 {
			if err := s.applyMigration7Tx(ctx, tx); err != nil {
				return fmt.Errorf("apply migration %d (%s): %w", migration.Version, migration.Description, err)
			}
		}

		// Execute migration SQL (indexes are IF NOT EXISTS, safe to re-run)
		if migration.SQL != "" {
			if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
				return fmt.Errorf("apply migration %d (%s): %w", migration.Version, migration.Description, err)
			}
		}

		// Record migration as applied
		if err := s.recordMigrationTx(ctx, tx, migration.Version); err != nil {
			return fmt.Errorf("record migration %d: %w", migration.Version, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}

	return nil
}

// applyMigration4 adds columns for external session correlation idempotently.
// SQLite doesn't support ADD COLUMN IF NOT EXISTS, so we check first.
func (s *Store) applyMigration4(ctx context.Context) error {
	columns := []struct {
		name string
		def  string
	}{
		{"external_session_id", "TEXT"},
		{"project_path", "TEXT"},
		{"agent_type", "TEXT"},
	}

	for _, col := range columns {
		if err := s.addColumnIfNotExists(ctx, "behavioral_sessions", col.name, col.def); err != nil {
			return fmt.Errorf("add column %s: %w", col.name, err)
		}
	}

	return nil
}

// addColumnIfNotExists adds a column to a table if it doesn't already exist.
// This is idempotent - concurrent calls are safe (duplicate column errors are ignored).
func (s *Store) addColumnIfNotExists(ctx context.Context, table, column, definition string) error {
	// Check if column exists using PRAGMA table_info
	query := fmt.Sprintf("PRAGMA table_info(%s)", table)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("query table info: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("scan table info: %w", err)
		}
		if name == column {
			// Column already exists
			return nil
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate table info: %w", err)
	}

	// Column doesn't exist, add it
	// Handle race condition: if another goroutine added it between our check and now,
	// SQLite returns "duplicate column name" which we treat as success
	alterSQL := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)
	if _, err := s.db.ExecContext(ctx, alterSQL); err != nil {
		if strings.Contains(err.Error(), "duplicate column name") {
			return nil // Column was added by concurrent operation - success
		}
		return fmt.Errorf("alter table: %w", err)
	}

	return nil
}

// GetAppliedVersions retrieves all applied migration versions
func (s *Store) GetAppliedVersions() ([]*MigrationVersion, error) {
	query := `SELECT version, applied_at FROM schema_version ORDER BY version ASC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query schema versions: %w", err)
	}
	defer rows.Close()

	var versions []*MigrationVersion
	for rows.Next() {
		v := &MigrationVersion{}
		if err := rows.Scan(&v.Version, &v.AppliedAt); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
		versions = append(versions, v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate versions: %w", err)
	}

	return versions, nil
}

// IsMigrationApplied checks if a specific migration version has been applied
func (s *Store) IsMigrationApplied(version int) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM schema_version WHERE version = ?`
	err := s.db.QueryRow(query, version).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check migration: %w", err)
	}
	return count > 0, nil
}

// ensureSchemaVersionTable ensures the schema_version table exists
func (s *Store) ensureSchemaVersionTable() error {
	sql := `
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`
	_, err := s.db.Exec(sql)
	if err != nil {
		return fmt.Errorf("create schema_version table: %w", err)
	}
	return nil
}

// recordMigration records that a migration has been applied
func (s *Store) recordMigration(ctx context.Context, version int) error {
	query := `INSERT OR IGNORE INTO schema_version (version) VALUES (?)`
	_, err := s.db.ExecContext(ctx, query, version)
	if err != nil {
		return fmt.Errorf("insert migration version: %w", err)
	}
	return nil
}

// GetLatestVersion returns the latest applied migration version
func (s *Store) GetLatestVersion() (int, error) {
	var version int
	query := `SELECT COALESCE(MAX(version), 0) FROM schema_version`
	err := s.db.QueryRow(query).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("query latest version: %w", err)
	}
	return version, nil
}

// Transaction-aware versions for ApplyMigrations

// ensureSchemaVersionTableTx ensures the schema_version table exists (within transaction)
func (s *Store) ensureSchemaVersionTableTx(tx *sql.Tx) error {
	sqlStr := `
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`
	_, err := tx.Exec(sqlStr)
	if err != nil {
		return fmt.Errorf("create schema_version table: %w", err)
	}
	return nil
}

// getAppliedVersionsTx retrieves all applied migration versions (within transaction)
func (s *Store) getAppliedVersionsTx(tx *sql.Tx) ([]*MigrationVersion, error) {
	query := `SELECT version, applied_at FROM schema_version ORDER BY version ASC`
	rows, err := tx.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query schema versions: %w", err)
	}
	defer rows.Close()

	var versions []*MigrationVersion
	for rows.Next() {
		v := &MigrationVersion{}
		if err := rows.Scan(&v.Version, &v.AppliedAt); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
		versions = append(versions, v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate versions: %w", err)
	}

	return versions, nil
}

// recordMigrationTx records that a migration has been applied (within transaction)
func (s *Store) recordMigrationTx(ctx context.Context, tx *sql.Tx, version int) error {
	query := `INSERT OR IGNORE INTO schema_version (version) VALUES (?)`
	_, err := tx.ExecContext(ctx, query, version)
	if err != nil {
		return fmt.Errorf("insert migration version: %w", err)
	}
	return nil
}

// applyMigration4Tx adds columns for external session correlation idempotently (within transaction).
func (s *Store) applyMigration4Tx(ctx context.Context, tx *sql.Tx) error {
	columns := []struct {
		name string
		def  string
	}{
		{"external_session_id", "TEXT"},
		{"project_path", "TEXT"},
		{"agent_type", "TEXT"},
	}

	for _, col := range columns {
		if err := s.addColumnIfNotExistsTx(ctx, tx, "behavioral_sessions", col.name, col.def); err != nil {
			return fmt.Errorf("add column %s: %w", col.name, err)
		}
	}

	return nil
}

// applyMigration7Tx adds model_name column for cost calculation (within transaction).
func (s *Store) applyMigration7Tx(ctx context.Context, tx *sql.Tx) error {
	if err := s.addColumnIfNotExistsTx(ctx, tx, "behavioral_sessions", "model_name", "TEXT"); err != nil {
		return fmt.Errorf("add column model_name: %w", err)
	}
	return nil
}

// addColumnIfNotExistsTx adds a column to a table if it doesn't already exist (within transaction).
func (s *Store) addColumnIfNotExistsTx(ctx context.Context, tx *sql.Tx, table, column, definition string) error {
	// Check if column exists using PRAGMA table_info
	query := fmt.Sprintf("PRAGMA table_info(%s)", table)
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("query table info: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("scan table info: %w", err)
		}
		if name == column {
			// Column already exists
			return nil
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate table info: %w", err)
	}

	// Column doesn't exist, add it
	alterSQL := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)
	if _, err := tx.ExecContext(ctx, alterSQL); err != nil {
		// Within a serialized transaction, this shouldn't race, but handle anyway
		if strings.Contains(err.Error(), "duplicate column name") {
			return nil
		}
		return fmt.Errorf("alter table: %w", err)
	}

	return nil
}
