package learning

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

// TaskExecution represents a single task execution record
type TaskExecution struct {
	ID              int64
	PlanFile        string
	RunNumber       int
	TaskNumber      string
	TaskName        string
	Agent           string
	Prompt          string
	Success         bool
	Output          string
	ErrorMessage    string
	DurationSecs    int64
	QCVerdict       string   // Quality control verdict: GREEN, RED, YELLOW
	QCFeedback      string   // Detailed feedback from QC review
	FailurePatterns []string // Identified failure patterns
	Timestamp       time.Time
	Context         string // JSON blob for additional context
}

// ApproachHistory tracks different approaches tried for recurring task patterns
type ApproachHistory struct {
	ID                  int64
	TaskPattern         string
	ApproachDescription string
	SuccessCount        int
	FailureCount        int
	LastUsed            time.Time
	CreatedAt           time.Time
	Metadata            string // JSON blob for additional data
}

// Store manages the SQLite database for adaptive learning
type Store struct {
	db     *sql.DB
	dbPath string
}

// NewStore creates a new Store instance and initializes the database
func NewStore(dbPath string) (*Store, error) {
	// Handle in-memory database
	if dbPath == ":memory:" {
		return openAndInitStore(dbPath)
	}

	// Ensure parent directory exists for file-based databases
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	return openAndInitStore(dbPath)
}

// openAndInitStore opens the database connection and initializes schema
func openAndInitStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Configure SQLite for concurrent access with retry logic.
	// Set busy_timeout FIRST so subsequent operations wait on locks.
	// Use retry with backoff for "database is locked" errors that can occur
	// during concurrent initialization of the same database file.
	pragmas := []string{
		"PRAGMA busy_timeout=5000", // Must be first
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-64000", // 64MB cache
	}

	for _, pragma := range pragmas {
		if err := execWithRetry(db, pragma, 5, 10*time.Millisecond); err != nil {
			db.Close()
			return nil, fmt.Errorf("set %s: %w", pragma, err)
		}
	}

	store := &Store{
		db:     db,
		dbPath: dbPath,
	}

	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return store, nil
}

// execWithRetry executes a SQL statement with exponential backoff retry on lock errors.
func execWithRetry(db *sql.DB, sql string, maxRetries int, baseDelay time.Duration) error {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		_, err := db.Exec(sql)
		if err == nil {
			return nil
		}

		// Only retry on "database is locked" errors
		if !strings.Contains(err.Error(), "database is locked") {
			return err
		}

		lastErr = err
		// Exponential backoff with jitter
		delay := baseDelay * time.Duration(1<<attempt)
		time.Sleep(delay)
	}
	return lastErr
}

// Close closes the database connection
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// initSchema initializes the database schema using migrations
func (s *Store) initSchema() error {
	ctx := context.Background()

	// Apply all pending migrations
	if err := s.ApplyMigrations(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

// getSchemaVersion retrieves the current schema version (delegates to GetLatestVersion)
func (s *Store) getSchemaVersion() (int, error) {
	return s.GetLatestVersion()
}

// tableExists checks if a table exists in the database
func (s *Store) tableExists(tableName string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`
	err := s.db.QueryRow(query, tableName).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check table existence: %w", err)
	}
	return count > 0, nil
}

// indexExists checks if an index exists in the database
func (s *Store) indexExists(indexName string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`
	err := s.db.QueryRow(query, indexName).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check index existence: %w", err)
	}
	return count > 0, nil
}

// RecordExecution records a task execution in the database
func (s *Store) RecordExecution(ctx context.Context, exec *TaskExecution) error {
	// Marshal context if needed (currently unused, but available for future use)
	contextJSON := exec.Context
	if contextJSON == "" {
		contextJSON = "{}"
	}

	// Marshal failure patterns to JSON
	failurePatternsJSON := "[]"
	if len(exec.FailurePatterns) > 0 {
		data, err := json.Marshal(exec.FailurePatterns)
		if err != nil {
			return fmt.Errorf("marshal failure patterns: %w", err)
		}
		failurePatternsJSON = string(data)
	}

	query := `INSERT INTO task_executions
		(plan_file, run_number, task_number, task_name, agent, prompt, success, output, error_message, duration_seconds, qc_verdict, qc_feedback, failure_patterns, context)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := s.db.ExecContext(ctx, query,
		exec.PlanFile,
		exec.RunNumber,
		exec.TaskNumber,
		exec.TaskName,
		exec.Agent,
		exec.Prompt,
		exec.Success,
		exec.Output,
		exec.ErrorMessage,
		exec.DurationSecs,
		exec.QCVerdict,
		exec.QCFeedback,
		failurePatternsJSON,
		contextJSON,
	)
	if err != nil {
		return fmt.Errorf("insert task execution: %w", err)
	}

	// Get the inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	exec.ID = id

	return nil
}

// GetExecutions retrieves all executions for a specific plan file, ordered by most recent first
func (s *Store) GetExecutions(planFile string) ([]*TaskExecution, error) {
	query := `SELECT id, plan_file, run_number, task_number, task_name, agent, prompt, success, output, error_message, duration_seconds, qc_verdict, qc_feedback, failure_patterns, timestamp, context
		FROM task_executions
		WHERE plan_file = ?
		ORDER BY id DESC`

	rows, err := s.db.Query(query, planFile)
	if err != nil {
		return nil, fmt.Errorf("query executions: %w", err)
	}
	defer rows.Close()

	var executions []*TaskExecution
	for rows.Next() {
		exec := &TaskExecution{}
		var planFileVal, agent, output, errorMessage, qcVerdict, qcFeedback, failurePatterns, contextVal sql.NullString
		var runNumber sql.NullInt64
		err := rows.Scan(
			&exec.ID,
			&planFileVal,
			&runNumber,
			&exec.TaskNumber,
			&exec.TaskName,
			&agent,
			&exec.Prompt,
			&exec.Success,
			&output,
			&errorMessage,
			&exec.DurationSecs,
			&qcVerdict,
			&qcFeedback,
			&failurePatterns,
			&exec.Timestamp,
			&contextVal,
		)
		if err != nil {
			return nil, fmt.Errorf("scan execution row: %w", err)
		}

		// Handle nullable fields
		if planFileVal.Valid {
			exec.PlanFile = planFileVal.String
		}
		if runNumber.Valid {
			exec.RunNumber = int(runNumber.Int64)
		}
		if agent.Valid {
			exec.Agent = agent.String
		}
		if output.Valid {
			exec.Output = output.String
		}
		if errorMessage.Valid {
			exec.ErrorMessage = errorMessage.String
		}
		if qcVerdict.Valid {
			exec.QCVerdict = qcVerdict.String
		}
		if qcFeedback.Valid {
			exec.QCFeedback = qcFeedback.String
		}
		if failurePatterns.Valid && failurePatterns.String != "" {
			if err := json.Unmarshal([]byte(failurePatterns.String), &exec.FailurePatterns); err != nil {
				return nil, fmt.Errorf("unmarshal failure patterns: %w", err)
			}
		}
		if contextVal.Valid {
			exec.Context = contextVal.String
		}

		executions = append(executions, exec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate execution rows: %w", err)
	}

	return executions, nil
}

// GetExecutionHistory retrieves all executions for a specific task, ordered by most recent first
func (s *Store) GetExecutionHistory(ctx context.Context, planFile, taskNumber string) ([]*TaskExecution, error) {
	query := `SELECT id, plan_file, run_number, task_number, task_name, agent, prompt, success, output, error_message, duration_seconds, qc_verdict, qc_feedback, failure_patterns, timestamp, context
		FROM task_executions
		WHERE plan_file = ? AND task_number = ?
		ORDER BY id DESC`

	rows, err := s.db.QueryContext(ctx, query, planFile, taskNumber)
	if err != nil {
		return nil, fmt.Errorf("query execution history: %w", err)
	}
	defer rows.Close()

	var executions []*TaskExecution
	for rows.Next() {
		exec := &TaskExecution{}
		var planFileVal, agent, output, errorMessage, qcVerdict, qcFeedback, failurePatterns, contextVal sql.NullString
		var runNumber sql.NullInt64
		err := rows.Scan(
			&exec.ID,
			&planFileVal,
			&runNumber,
			&exec.TaskNumber,
			&exec.TaskName,
			&agent,
			&exec.Prompt,
			&exec.Success,
			&output,
			&errorMessage,
			&exec.DurationSecs,
			&qcVerdict,
			&qcFeedback,
			&failurePatterns,
			&exec.Timestamp,
			&contextVal,
		)
		if err != nil {
			return nil, fmt.Errorf("scan execution row: %w", err)
		}

		// Handle nullable fields
		if planFileVal.Valid {
			exec.PlanFile = planFileVal.String
		}
		if runNumber.Valid {
			exec.RunNumber = int(runNumber.Int64)
		}
		if agent.Valid {
			exec.Agent = agent.String
		}
		if output.Valid {
			exec.Output = output.String
		}
		if errorMessage.Valid {
			exec.ErrorMessage = errorMessage.String
		}
		if qcVerdict.Valid {
			exec.QCVerdict = qcVerdict.String
		}
		if qcFeedback.Valid {
			exec.QCFeedback = qcFeedback.String
		}
		if failurePatterns.Valid && failurePatterns.String != "" {
			if err := json.Unmarshal([]byte(failurePatterns.String), &exec.FailurePatterns); err != nil {
				return nil, fmt.Errorf("unmarshal failure patterns: %w", err)
			}
		}
		if contextVal.Valid {
			exec.Context = contextVal.String
		}

		executions = append(executions, exec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate execution rows: %w", err)
	}

	return executions, nil
}

// RecordApproach records or updates an approach in the approach history
func (s *Store) RecordApproach(ctx context.Context, approach *ApproachHistory) error {
	// Marshal metadata if needed
	metadataJSON := approach.Metadata
	if metadataJSON == "" {
		metadataJSON = "{}"
	}

	query := `INSERT INTO approach_history
		(task_pattern, approach_description, success_count, failure_count, metadata)
		VALUES (?, ?, ?, ?, ?)`

	result, err := s.db.ExecContext(ctx, query,
		approach.TaskPattern,
		approach.ApproachDescription,
		approach.SuccessCount,
		approach.FailureCount,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("insert approach history: %w", err)
	}

	// Get the inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	approach.ID = id

	return nil
}

// GetRunCount returns the maximum run number for a given plan file.
// Returns 0 if the plan has never been executed.
func (s *Store) GetRunCount(ctx context.Context, planFile string) (int, error) {
	var maxRun int
	query := `SELECT COALESCE(MAX(run_number), 0) FROM task_executions WHERE plan_file = ?`
	err := s.db.QueryRowContext(ctx, query, planFile).Scan(&maxRun)
	if err != nil {
		return 0, fmt.Errorf("query run count: %w", err)
	}
	return maxRun, nil
}

// QueryRows executes a query that returns multiple rows
func (s *Store) QueryRows(query string, args ...interface{}) (*sql.Rows, error) {
	return s.db.Query(query, args...)
}

// Exec executes a query that doesn't return rows (INSERT, UPDATE, DELETE)
func (s *Store) Exec(query string, args ...interface{}) (sql.Result, error) {
	return s.db.Exec(query, args...)
}

// BehavioralSessionData represents data for a behavioral session with metrics
type BehavioralSessionData struct {
	TaskExecutionID     int64
	SessionStart        time.Time
	SessionEnd          *time.Time
	TotalDurationSecs   int64
	TotalToolCalls      int
	TotalBashCommands   int
	TotalFileOperations int
	TotalTokensUsed     int64
	ContextWindowUsed   int
}

// BehavioralSessionMetrics represents complete session metrics with execution context
type BehavioralSessionMetrics struct {
	SessionID           int64
	TaskExecutionID     int64
	SessionStart        time.Time
	SessionEnd          *time.Time
	TotalDurationSecs   int64
	TotalToolCalls      int
	TotalBashCommands   int
	TotalFileOperations int
	TotalTokensUsed     int64
	ContextWindowUsed   int
	// Execution context
	TaskNumber string
	TaskName   string
	Agent      string
	Success    bool
	ModelName  string // Claude model used (e.g., claude-opus-4-5-20251101)
}

// ToolExecutionData represents a tool execution record
type ToolExecutionData struct {
	SessionID    int64
	ToolName     string
	Parameters   string
	DurationMs   int64
	Success      bool
	ErrorMessage string
}

// BashCommandData represents a bash command execution
type BashCommandData struct {
	SessionID    int64
	Command      string
	DurationMs   int64
	ExitCode     int
	StdoutLength int
	StderrLength int
	Success      bool
}

// FileOperationData represents a file operation
type FileOperationData struct {
	SessionID     int64
	OperationType string
	FilePath      string
	DurationMs    int64
	BytesAffected int64
	Success       bool
	ErrorMessage  string
}

// TokenUsageData represents token usage measurement
type TokenUsageData struct {
	SessionID         int64
	InputTokens       int64
	OutputTokens      int64
	TotalTokens       int64
	ContextWindowSize int
}

// RecordSessionMetrics records a behavioral session and its metrics in a transaction.
// This is an atomic operation that inserts the session and all related metrics.
func (s *Store) RecordSessionMetrics(ctx context.Context, sessionData *BehavioralSessionData,
	tools []ToolExecutionData, bashCmds []BashCommandData,
	fileOps []FileOperationData, tokenUsage []TokenUsageData) (int64, error) {

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert behavioral session
	sessionQuery := `INSERT INTO behavioral_sessions
		(task_execution_id, session_start, session_end, total_duration_seconds,
		 total_tool_calls, total_bash_commands, total_file_operations,
		 total_tokens_used, context_window_used)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := tx.ExecContext(ctx, sessionQuery,
		sessionData.TaskExecutionID,
		sessionData.SessionStart,
		sessionData.SessionEnd,
		sessionData.TotalDurationSecs,
		sessionData.TotalToolCalls,
		sessionData.TotalBashCommands,
		sessionData.TotalFileOperations,
		sessionData.TotalTokensUsed,
		sessionData.ContextWindowUsed,
	)
	if err != nil {
		return 0, fmt.Errorf("insert behavioral session: %w", err)
	}

	sessionID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get session id: %w", err)
	}

	// Insert tool executions
	if len(tools) > 0 {
		toolStmt, err := tx.PrepareContext(ctx, `INSERT INTO tool_executions
			(session_id, tool_name, parameters, duration_ms, success, error_message)
			VALUES (?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return 0, fmt.Errorf("prepare tool statement: %w", err)
		}
		defer toolStmt.Close()

		for _, tool := range tools {
			_, err := toolStmt.ExecContext(ctx, sessionID, tool.ToolName,
				tool.Parameters, tool.DurationMs, tool.Success, tool.ErrorMessage)
			if err != nil {
				return 0, fmt.Errorf("insert tool execution: %w", err)
			}
		}
	}

	// Insert bash commands
	if len(bashCmds) > 0 {
		bashStmt, err := tx.PrepareContext(ctx, `INSERT INTO bash_commands
			(session_id, command, duration_ms, exit_code, stdout_length, stderr_length, success)
			VALUES (?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return 0, fmt.Errorf("prepare bash statement: %w", err)
		}
		defer bashStmt.Close()

		for _, bash := range bashCmds {
			_, err := bashStmt.ExecContext(ctx, sessionID, bash.Command,
				bash.DurationMs, bash.ExitCode, bash.StdoutLength, bash.StderrLength, bash.Success)
			if err != nil {
				return 0, fmt.Errorf("insert bash command: %w", err)
			}
		}
	}

	// Insert file operations
	if len(fileOps) > 0 {
		fileStmt, err := tx.PrepareContext(ctx, `INSERT INTO file_operations
			(session_id, operation_type, file_path, duration_ms, bytes_affected, success, error_message)
			VALUES (?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return 0, fmt.Errorf("prepare file statement: %w", err)
		}
		defer fileStmt.Close()

		for _, fileOp := range fileOps {
			_, err := fileStmt.ExecContext(ctx, sessionID, fileOp.OperationType,
				fileOp.FilePath, fileOp.DurationMs, fileOp.BytesAffected,
				fileOp.Success, fileOp.ErrorMessage)
			if err != nil {
				return 0, fmt.Errorf("insert file operation: %w", err)
			}
		}
	}

	// Insert token usage
	if len(tokenUsage) > 0 {
		tokenStmt, err := tx.PrepareContext(ctx, `INSERT INTO token_usage
			(session_id, input_tokens, output_tokens, total_tokens, context_window_size)
			VALUES (?, ?, ?, ?, ?)`)
		if err != nil {
			return 0, fmt.Errorf("prepare token statement: %w", err)
		}
		defer tokenStmt.Close()

		for _, token := range tokenUsage {
			_, err := tokenStmt.ExecContext(ctx, sessionID, token.InputTokens,
				token.OutputTokens, token.TotalTokens, token.ContextWindowSize)
			if err != nil {
				return 0, fmt.Errorf("insert token usage: %w", err)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	return sessionID, nil
}

// GetSessionMetrics retrieves behavioral metrics for a specific session
func (s *Store) GetSessionMetrics(ctx context.Context, sessionID int64) (*BehavioralSessionMetrics, error) {
	query := `SELECT
		bs.id, bs.task_execution_id, bs.session_start, bs.session_end,
		bs.total_duration_seconds, bs.total_tool_calls, bs.total_bash_commands,
		bs.total_file_operations, bs.total_tokens_used, bs.context_window_used,
		COALESCE(te.task_number, SUBSTR(bs.external_session_id, 1, 8)),
		COALESCE(te.task_name, 'Agent Session: ' || SUBSTR(bs.external_session_id, 1, 8)),
		COALESCE(te.agent, bs.agent_type),
		COALESCE(te.success, 1),
		COALESCE(bs.model_name, '')
		FROM behavioral_sessions bs
		LEFT JOIN task_executions te ON bs.task_execution_id = te.id AND bs.task_execution_id > 0
		WHERE bs.id = ?`

	row := s.db.QueryRowContext(ctx, query, sessionID)

	metrics := &BehavioralSessionMetrics{}
	var sessionEnd sql.NullTime

	err := row.Scan(
		&metrics.SessionID,
		&metrics.TaskExecutionID,
		&metrics.SessionStart,
		&sessionEnd,
		&metrics.TotalDurationSecs,
		&metrics.TotalToolCalls,
		&metrics.TotalBashCommands,
		&metrics.TotalFileOperations,
		&metrics.TotalTokensUsed,
		&metrics.ContextWindowUsed,
		&metrics.TaskNumber,
		&metrics.TaskName,
		&metrics.Agent,
		&metrics.Success,
		&metrics.ModelName,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found: %d", sessionID)
		}
		return nil, fmt.Errorf("query session metrics: %w", err)
	}

	if sessionEnd.Valid {
		metrics.SessionEnd = &sessionEnd.Time
	}

	return metrics, nil
}

// GetTaskBehavior retrieves aggregated behavioral metrics for a specific task
func (s *Store) GetTaskBehavior(ctx context.Context, taskNumber string) (map[string]interface{}, error) {
	query := `SELECT
		COUNT(bs.id) as session_count,
		AVG(bs.total_duration_seconds) as avg_duration,
		AVG(bs.total_tool_calls) as avg_tool_calls,
		AVG(bs.total_bash_commands) as avg_bash_commands,
		AVG(bs.total_file_operations) as avg_file_operations,
		SUM(bs.total_tokens_used) as total_tokens,
		AVG(bs.context_window_used) as avg_context_window
		FROM behavioral_sessions bs
		JOIN task_executions te ON bs.task_execution_id = te.id
		WHERE te.task_number = ?`

	var sessionCount int
	var avgDuration, avgToolCalls, avgBashCmds, avgFileOps, avgContextWindow sql.NullFloat64
	var totalTokens sql.NullInt64

	err := s.db.QueryRowContext(ctx, query, taskNumber).Scan(
		&sessionCount,
		&avgDuration,
		&avgToolCalls,
		&avgBashCmds,
		&avgFileOps,
		&totalTokens,
		&avgContextWindow,
	)
	if err != nil {
		return nil, fmt.Errorf("query task behavior: %w", err)
	}

	result := map[string]interface{}{
		"session_count":        sessionCount,
		"avg_duration_seconds": 0.0,
		"avg_tool_calls":       0.0,
		"avg_bash_commands":    0.0,
		"avg_file_operations":  0.0,
		"total_tokens":         int64(0),
		"avg_context_window":   0.0,
	}

	if avgDuration.Valid {
		result["avg_duration_seconds"] = avgDuration.Float64
	}
	if avgToolCalls.Valid {
		result["avg_tool_calls"] = avgToolCalls.Float64
	}
	if avgBashCmds.Valid {
		result["avg_bash_commands"] = avgBashCmds.Float64
	}
	if avgFileOps.Valid {
		result["avg_file_operations"] = avgFileOps.Float64
	}
	if totalTokens.Valid {
		result["total_tokens"] = totalTokens.Int64
	}
	if avgContextWindow.Valid {
		result["avg_context_window"] = avgContextWindow.Float64
	}

	return result, nil
}

// GetProjectBehavior retrieves aggregated behavioral metrics for a project
func (s *Store) GetProjectBehavior(ctx context.Context, project string) (map[string]interface{}, error) {
	query := `SELECT
		COUNT(bs.id) as session_count,
		AVG(bs.total_duration_seconds) as avg_duration,
		SUM(bs.total_tool_calls) as total_tool_calls,
		SUM(bs.total_bash_commands) as total_bash_commands,
		SUM(bs.total_file_operations) as total_file_operations,
		SUM(bs.total_tokens_used) as total_tokens,
		AVG(bs.context_window_used) as avg_context_window,
		COUNT(CASE WHEN te.success = 1 THEN 1 END) as success_count
		FROM behavioral_sessions bs
		JOIN task_executions te ON bs.task_execution_id = te.id
		WHERE te.plan_file LIKE ?`

	var sessionCount, successCount int
	var totalToolCalls, totalBashCmds, totalFileOps, totalTokens sql.NullInt64
	var avgDuration, avgContextWindow sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query, "%"+project+"%").Scan(
		&sessionCount,
		&avgDuration,
		&totalToolCalls,
		&totalBashCmds,
		&totalFileOps,
		&totalTokens,
		&avgContextWindow,
		&successCount,
	)
	if err != nil {
		return nil, fmt.Errorf("query project behavior: %w", err)
	}

	result := map[string]interface{}{
		"session_count":         sessionCount,
		"success_count":         successCount,
		"avg_duration_seconds":  0.0,
		"total_tool_calls":      int64(0),
		"total_bash_commands":   int64(0),
		"total_file_operations": int64(0),
		"total_tokens":          int64(0),
		"avg_context_window":    0.0,
		"success_rate":          0.0,
	}

	if avgDuration.Valid {
		result["avg_duration_seconds"] = avgDuration.Float64
	}
	if totalToolCalls.Valid {
		result["total_tool_calls"] = totalToolCalls.Int64
	}
	if totalBashCmds.Valid {
		result["total_bash_commands"] = totalBashCmds.Int64
	}
	if totalFileOps.Valid {
		result["total_file_operations"] = totalFileOps.Int64
	}
	if totalTokens.Valid {
		result["total_tokens"] = totalTokens.Int64
	}
	if avgContextWindow.Valid {
		result["avg_context_window"] = avgContextWindow.Float64
	}
	if sessionCount > 0 {
		result["success_rate"] = float64(successCount) / float64(sessionCount)
	}

	return result, nil
}

// DeleteOldBehavioralData removes behavioral data older than the specified duration.
// This is useful for cleanup and maintaining database size.
func (s *Store) DeleteOldBehavioralData(ctx context.Context, olderThan time.Time) (int64, error) {
	// Delete cascades to child tables via foreign key constraints
	query := `DELETE FROM behavioral_sessions
		WHERE id IN (
			SELECT bs.id FROM behavioral_sessions bs
			JOIN task_executions te ON bs.task_execution_id = te.id
			WHERE te.timestamp < ?
		)`

	result, err := s.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("delete old behavioral data: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	return deleted, nil
}

// CleanupOldExecutions removes task execution records older than the specified number of days.
// Also cleans up related behavioral data via cascade.
// Returns the number of deleted execution records.
func (s *Store) CleanupOldExecutions(ctx context.Context, keepDays int) (int64, error) {
	if keepDays <= 0 {
		return 0, nil // 0 or negative means keep forever
	}

	cutoff := time.Now().AddDate(0, 0, -keepDays)

	query := `DELETE FROM task_executions WHERE timestamp < ?`
	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("cleanup old executions: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	return deleted, nil
}

// AgentTypeStats represents aggregated statistics for an agent type
type AgentTypeStats struct {
	AgentType           string
	SuccessCount        int
	FailureCount        int
	TotalSessions       int
	AvgDurationSeconds  float64
	TotalInputTokens    int64
	TotalOutputTokens   int64
	SuccessRate         float64
	AvgTokensPerSession float64
}

// GetAgentTypeStats returns aggregated statistics grouped by agent type (from te.agent column)
// Returns top agents by success count, with pagination support
func (s *Store) GetAgentTypeStats(ctx context.Context, project string, limit, offset int) ([]AgentTypeStats, error) {
	query := `
		SELECT
			COALESCE(te.agent, bs.agent_type, 'unknown') as agent_type,
			COUNT(CASE WHEN COALESCE(te.success, 1) = 1 THEN 1 END) as success_count,
			COUNT(CASE WHEN COALESCE(te.success, 1) = 0 THEN 1 END) as failure_count,
			COUNT(bs.id) as total_sessions,
			AVG(bs.total_duration_seconds) as avg_duration,
			SUM(tu.input_tokens) as total_input_tokens,
			SUM(tu.output_tokens) as total_output_tokens
		FROM behavioral_sessions bs
		LEFT JOIN task_executions te ON bs.task_execution_id = te.id AND bs.task_execution_id > 0
		LEFT JOIN token_usage tu ON bs.id = tu.session_id
	`

	args := []interface{}{}

	if project != "" {
		query += " WHERE (te.plan_file LIKE ? OR bs.project_path LIKE ?)"
		args = append(args, "%"+project+"%", "%"+project+"%")
	}

	query += `
		GROUP BY COALESCE(te.agent, bs.agent_type, 'unknown')
		ORDER BY total_sessions DESC
	`

	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query agent type stats: %w", err)
	}
	defer rows.Close()

	var stats []AgentTypeStats
	for rows.Next() {
		var s AgentTypeStats
		var avgDuration sql.NullFloat64
		var totalInputTokens, totalOutputTokens sql.NullInt64

		if err := rows.Scan(
			&s.AgentType,
			&s.SuccessCount,
			&s.FailureCount,
			&s.TotalSessions,
			&avgDuration,
			&totalInputTokens,
			&totalOutputTokens,
		); err != nil {
			return nil, fmt.Errorf("scan agent type stats row: %w", err)
		}

		s.TotalSessions = s.SuccessCount + s.FailureCount
		if s.TotalSessions > 0 {
			s.SuccessRate = float64(s.SuccessCount) / float64(s.TotalSessions)
		}

		if avgDuration.Valid {
			s.AvgDurationSeconds = avgDuration.Float64
		}

		if totalInputTokens.Valid {
			s.TotalInputTokens = totalInputTokens.Int64
		}
		if totalOutputTokens.Valid {
			s.TotalOutputTokens = totalOutputTokens.Int64
		}

		if s.TotalSessions > 0 && (s.TotalInputTokens > 0 || s.TotalOutputTokens > 0) {
			s.AvgTokensPerSession = float64(s.TotalInputTokens+s.TotalOutputTokens) / float64(s.TotalSessions)
		}

		stats = append(stats, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent type stats: %w", err)
	}

	return stats, nil
}

// SummaryStats represents overall summary statistics
type SummaryStats struct {
	TotalAgents         int
	TotalSessions       int
	TotalSuccesses      int
	TotalFailures       int
	SuccessRate         float64
	AvgDurationSeconds  float64
	TotalInputTokens    int64
	TotalOutputTokens   int64
	TotalTokens         int64
	AvgTokensPerSession float64
	TotalCostUSD        float64 // Estimated cost based on default model pricing
}

// GetSummaryStats returns overall summary statistics for a project
func (s *Store) GetSummaryStats(ctx context.Context, project string) (*SummaryStats, error) {
	query := `
		SELECT
			COUNT(DISTINCT COALESCE(te.agent, bs.agent_type, 'unknown')) as total_agents,
			COUNT(bs.id) as total_sessions,
			COUNT(CASE WHEN COALESCE(te.success, 1) = 1 THEN 1 END) as total_successes,
			COUNT(CASE WHEN COALESCE(te.success, 1) = 0 THEN 1 END) as total_failures,
			AVG(bs.total_duration_seconds) as avg_duration,
			SUM(tu.input_tokens) as total_input_tokens,
			SUM(tu.output_tokens) as total_output_tokens
		FROM behavioral_sessions bs
		LEFT JOIN task_executions te ON bs.task_execution_id = te.id AND bs.task_execution_id > 0
		LEFT JOIN token_usage tu ON bs.id = tu.session_id
	`

	args := []interface{}{}

	if project != "" {
		query += " WHERE (te.plan_file LIKE ? OR bs.project_path LIKE ?)"
		args = append(args, "%"+project+"%", "%"+project+"%")
	}

	row := s.db.QueryRowContext(ctx, query, args...)

	stats := &SummaryStats{}
	var avgDuration sql.NullFloat64
	var totalInputTokens, totalOutputTokens sql.NullInt64

	if err := row.Scan(
		&stats.TotalAgents,
		&stats.TotalSessions,
		&stats.TotalSuccesses,
		&stats.TotalFailures,
		&avgDuration,
		&totalInputTokens,
		&totalOutputTokens,
	); err != nil {
		if err == sql.ErrNoRows {
			// Return empty stats if no data found
			return &SummaryStats{}, nil
		}
		return nil, fmt.Errorf("query summary stats: %w", err)
	}

	if stats.TotalSessions > 0 {
		stats.SuccessRate = float64(stats.TotalSuccesses) / float64(stats.TotalSessions)
	}

	if avgDuration.Valid {
		stats.AvgDurationSeconds = avgDuration.Float64
	}

	if totalInputTokens.Valid {
		stats.TotalInputTokens = totalInputTokens.Int64
	}
	if totalOutputTokens.Valid {
		stats.TotalOutputTokens = totalOutputTokens.Int64
	}

	stats.TotalTokens = stats.TotalInputTokens + stats.TotalOutputTokens
	if stats.TotalSessions > 0 {
		stats.AvgTokensPerSession = float64(stats.TotalTokens) / float64(stats.TotalSessions)
	}

	// Calculate estimated cost using default Sonnet pricing ($3/1M input, $15/1M output)
	inputCost := float64(stats.TotalInputTokens) / 1_000_000 * 3.0
	outputCost := float64(stats.TotalOutputTokens) / 1_000_000 * 15.0
	stats.TotalCostUSD = inputCost + outputCost

	return stats, nil
}

// ToolStats represents aggregated statistics for a specific tool
type ToolStats struct {
	ToolName      string
	CallCount     int
	SuccessCount  int
	FailureCount  int
	AvgDurationMs float64
	SuccessRate   float64
}

// GetToolStats returns aggregated statistics grouped by tool name
// Ordered by call count descending, with limit/offset support
func (s *Store) GetToolStats(ctx context.Context, project string, limit, offset int) ([]ToolStats, error) {
	query := `
		SELECT
			te.tool_name,
			COUNT(*) as call_count,
			COUNT(CASE WHEN te.success = 1 THEN 1 END) as success_count,
			COUNT(CASE WHEN te.success = 0 THEN 1 END) as failure_count,
			AVG(te.duration_ms) as avg_duration
		FROM tool_executions te
		JOIN behavioral_sessions bs ON te.session_id = bs.id
		JOIN task_executions tex ON bs.task_execution_id = tex.id
	`

	args := []interface{}{}

	if project != "" {
		query += " WHERE tex.plan_file LIKE ?"
		args = append(args, "%"+project+"%")
	}

	query += `
		GROUP BY te.tool_name
		ORDER BY call_count DESC
	`

	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query tool stats: %w", err)
	}
	defer rows.Close()

	var stats []ToolStats
	for rows.Next() {
		var ts ToolStats
		var avgDuration sql.NullFloat64

		if err := rows.Scan(
			&ts.ToolName,
			&ts.CallCount,
			&ts.SuccessCount,
			&ts.FailureCount,
			&avgDuration,
		); err != nil {
			return nil, fmt.Errorf("scan tool stats row: %w", err)
		}

		if ts.CallCount > 0 {
			ts.SuccessRate = float64(ts.SuccessCount) / float64(ts.CallCount)
		}

		if avgDuration.Valid {
			ts.AvgDurationMs = avgDuration.Float64
		}

		stats = append(stats, ts)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tool stats: %w", err)
	}

	return stats, nil
}

// BashStats represents aggregated statistics for bash commands
type BashStats struct {
	Command       string
	CallCount     int
	SuccessCount  int
	FailureCount  int
	AvgDurationMs float64
	SuccessRate   float64
}

// GetBashStats returns aggregated statistics grouped by bash command
// Ordered by call count descending, with limit/offset support
func (s *Store) GetBashStats(ctx context.Context, project string, limit, offset int) ([]BashStats, error) {
	query := `
		SELECT
			bc.command,
			COUNT(*) as call_count,
			COUNT(CASE WHEN bc.success = 1 THEN 1 END) as success_count,
			COUNT(CASE WHEN bc.success = 0 THEN 1 END) as failure_count,
			AVG(bc.duration_ms) as avg_duration
		FROM bash_commands bc
		JOIN behavioral_sessions bs ON bc.session_id = bs.id
		JOIN task_executions te ON bs.task_execution_id = te.id
	`

	args := []interface{}{}

	if project != "" {
		query += " WHERE te.plan_file LIKE ?"
		args = append(args, "%"+project+"%")
	}

	query += `
		GROUP BY bc.command
		ORDER BY call_count DESC
	`

	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query bash stats: %w", err)
	}
	defer rows.Close()

	var stats []BashStats
	for rows.Next() {
		var bs BashStats
		var avgDuration sql.NullFloat64

		if err := rows.Scan(
			&bs.Command,
			&bs.CallCount,
			&bs.SuccessCount,
			&bs.FailureCount,
			&avgDuration,
		); err != nil {
			return nil, fmt.Errorf("scan bash stats row: %w", err)
		}

		if bs.CallCount > 0 {
			bs.SuccessRate = float64(bs.SuccessCount) / float64(bs.CallCount)
		}

		if avgDuration.Valid {
			bs.AvgDurationMs = avgDuration.Float64
		}

		stats = append(stats, bs)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bash stats: %w", err)
	}

	return stats, nil
}

// FileStats represents aggregated statistics for file operations
type FileStats struct {
	FilePath      string
	OperationType string
	OpCount       int
	SuccessCount  int
	FailureCount  int
	AvgDurationMs float64
	TotalBytes    int64
	SuccessRate   float64
}

// GetFileStats returns aggregated statistics grouped by file path and operation type
// Ordered by operation count descending, with limit/offset support
func (s *Store) GetFileStats(ctx context.Context, project string, limit, offset int) ([]FileStats, error) {
	query := `
		SELECT
			fo.file_path,
			fo.operation_type,
			COUNT(*) as op_count,
			COUNT(CASE WHEN fo.success = 1 THEN 1 END) as success_count,
			COUNT(CASE WHEN fo.success = 0 THEN 1 END) as failure_count,
			AVG(fo.duration_ms) as avg_duration,
			COALESCE(SUM(fo.bytes_affected), 0) as total_bytes
		FROM file_operations fo
		JOIN behavioral_sessions bs ON fo.session_id = bs.id
		JOIN task_executions te ON bs.task_execution_id = te.id
	`

	args := []interface{}{}

	if project != "" {
		query += " WHERE te.plan_file LIKE ?"
		args = append(args, "%"+project+"%")
	}

	query += `
		GROUP BY fo.file_path, fo.operation_type
		ORDER BY op_count DESC
	`

	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query file stats: %w", err)
	}
	defer rows.Close()

	var stats []FileStats
	for rows.Next() {
		var fs FileStats
		var avgDuration sql.NullFloat64

		if err := rows.Scan(
			&fs.FilePath,
			&fs.OperationType,
			&fs.OpCount,
			&fs.SuccessCount,
			&fs.FailureCount,
			&avgDuration,
			&fs.TotalBytes,
		); err != nil {
			return nil, fmt.Errorf("scan file stats row: %w", err)
		}

		if fs.OpCount > 0 {
			fs.SuccessRate = float64(fs.SuccessCount) / float64(fs.OpCount)
		}

		if avgDuration.Valid {
			fs.AvgDurationMs = avgDuration.Float64
		}

		stats = append(stats, fs)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate file stats: %w", err)
	}

	return stats, nil
}

// ErrorPattern represents an error pattern with context
type ErrorPattern struct {
	ErrorType     string // "tool", "bash", "file"
	ErrorMessage  string
	Tool          string // For tool errors
	Command       string // For bash errors
	FilePath      string // For file errors
	OperationType string // For file errors
	Count         int
	LastOccurred  time.Time
}

// GetErrorPatterns returns aggregated error statistics
// Ordered by error count descending, with limit/offset support
func (s *Store) GetErrorPatterns(ctx context.Context, project string, limit, offset int) ([]ErrorPattern, error) {
	patterns := []ErrorPattern{}

	// Get tool errors
	toolErrors, err := s.getToolErrors(ctx, project, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get tool errors: %w", err)
	}
	patterns = append(patterns, toolErrors...)

	// Get bash errors
	bashErrors, err := s.getBashErrors(ctx, project, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get bash errors: %w", err)
	}
	patterns = append(patterns, bashErrors...)

	// Get file operation errors
	fileErrors, err := s.getFileErrors(ctx, project, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get file errors: %w", err)
	}
	patterns = append(patterns, fileErrors...)

	// Sort by count descending
	sortErrorPatterns(patterns)

	// Apply limit to combined results
	if limit > 0 && len(patterns) > limit {
		patterns = patterns[:limit]
	}

	return patterns, nil
}

// getToolErrors returns error patterns from tool executions
func (s *Store) getToolErrors(ctx context.Context, project string, limit, offset int) ([]ErrorPattern, error) {
	query := `
		SELECT
			te.tool_name,
			te.error_message,
			COUNT(*) as error_count,
			MAX(te.execution_time) as last_occurred
		FROM tool_executions te
		JOIN behavioral_sessions bs ON te.session_id = bs.id
		JOIN task_executions tex ON bs.task_execution_id = tex.id
		WHERE te.success = 0 AND te.error_message IS NOT NULL
	`

	args := []interface{}{}

	if project != "" {
		query += " AND tex.plan_file LIKE ?"
		args = append(args, "%"+project+"%")
	}

	query += `
		GROUP BY te.tool_name, te.error_message
		ORDER BY error_count DESC
	`

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query tool errors: %w", err)
	}
	defer rows.Close()

	var patterns []ErrorPattern
	for rows.Next() {
		var ep ErrorPattern
		var lastOccurredStr sql.NullString

		if err := rows.Scan(
			&ep.Tool,
			&ep.ErrorMessage,
			&ep.Count,
			&lastOccurredStr,
		); err != nil {
			return nil, fmt.Errorf("scan tool error row: %w", err)
		}

		ep.ErrorType = "tool"
		if lastOccurredStr.Valid {
			if t, err := time.Parse(time.RFC3339, lastOccurredStr.String); err == nil {
				ep.LastOccurred = t
			} else if t, err := time.Parse("2006-01-02 15:04:05", lastOccurredStr.String); err == nil {
				ep.LastOccurred = t
			}
		}

		patterns = append(patterns, ep)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tool errors: %w", err)
	}

	return patterns, nil
}

// getBashErrors returns error patterns from bash commands
func (s *Store) getBashErrors(ctx context.Context, project string, limit, offset int) ([]ErrorPattern, error) {
	query := `
		SELECT
			bc.command,
			bc.exit_code,
			COUNT(*) as error_count,
			MAX(bc.execution_time) as last_occurred
		FROM bash_commands bc
		JOIN behavioral_sessions bs ON bc.session_id = bs.id
		JOIN task_executions te ON bs.task_execution_id = te.id
		WHERE bc.success = 0
	`

	args := []interface{}{}

	if project != "" {
		query += " AND te.plan_file LIKE ?"
		args = append(args, "%"+project+"%")
	}

	query += `
		GROUP BY bc.command, bc.exit_code
		ORDER BY error_count DESC
	`

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query bash errors: %w", err)
	}
	defer rows.Close()

	var patterns []ErrorPattern
	for rows.Next() {
		var ep ErrorPattern
		var exitCode sql.NullInt64
		var lastOccurredStr sql.NullString

		if err := rows.Scan(
			&ep.Command,
			&exitCode,
			&ep.Count,
			&lastOccurredStr,
		); err != nil {
			return nil, fmt.Errorf("scan bash error row: %w", err)
		}

		ep.ErrorType = "bash"
		if exitCode.Valid {
			ep.ErrorMessage = fmt.Sprintf("exit code %d", exitCode.Int64)
		}
		if lastOccurredStr.Valid {
			if t, err := time.Parse(time.RFC3339, lastOccurredStr.String); err == nil {
				ep.LastOccurred = t
			} else if t, err := time.Parse("2006-01-02 15:04:05", lastOccurredStr.String); err == nil {
				ep.LastOccurred = t
			}
		}

		patterns = append(patterns, ep)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bash errors: %w", err)
	}

	return patterns, nil
}

// getFileErrors returns error patterns from file operations
func (s *Store) getFileErrors(ctx context.Context, project string, limit, offset int) ([]ErrorPattern, error) {
	query := `
		SELECT
			fo.file_path,
			fo.operation_type,
			fo.error_message,
			COUNT(*) as error_count,
			MAX(fo.execution_time) as last_occurred
		FROM file_operations fo
		JOIN behavioral_sessions bs ON fo.session_id = bs.id
		JOIN task_executions te ON bs.task_execution_id = te.id
		WHERE fo.success = 0 AND fo.error_message IS NOT NULL
	`

	args := []interface{}{}

	if project != "" {
		query += " AND te.plan_file LIKE ?"
		args = append(args, "%"+project+"%")
	}

	query += `
		GROUP BY fo.file_path, fo.operation_type, fo.error_message
		ORDER BY error_count DESC
	`

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query file errors: %w", err)
	}
	defer rows.Close()

	var patterns []ErrorPattern
	for rows.Next() {
		var ep ErrorPattern
		var lastOccurredStr sql.NullString

		if err := rows.Scan(
			&ep.FilePath,
			&ep.OperationType,
			&ep.ErrorMessage,
			&ep.Count,
			&lastOccurredStr,
		); err != nil {
			return nil, fmt.Errorf("scan file error row: %w", err)
		}

		ep.ErrorType = "file"
		if lastOccurredStr.Valid {
			if t, err := time.Parse(time.RFC3339, lastOccurredStr.String); err == nil {
				ep.LastOccurred = t
			} else if t, err := time.Parse("2006-01-02 15:04:05", lastOccurredStr.String); err == nil {
				ep.LastOccurred = t
			}
		}

		patterns = append(patterns, ep)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate file errors: %w", err)
	}

	return patterns, nil
}

// sortErrorPatterns sorts error patterns by count descending
func sortErrorPatterns(patterns []ErrorPattern) {
	for i := 0; i < len(patterns); i++ {
		for j := i + 1; j < len(patterns); j++ {
			if patterns[j].Count > patterns[i].Count {
				patterns[i], patterns[j] = patterns[j], patterns[i]
			}
		}
	}
}

// IngestOffset tracks read position for a JSONL file
type IngestOffset struct {
	ID           int64
	FilePath     string
	ByteOffset   int64
	Inode        uint64
	LastLineHash string
	UpdatedAt    time.Time
}

// GetFileOffset retrieves the tracking offset for a specific file.
// Returns (nil, nil) if the file is not being tracked.
func (s *Store) GetFileOffset(ctx context.Context, filePath string) (*IngestOffset, error) {
	query := `SELECT id, file_path, byte_offset, inode, last_line_hash, updated_at
		FROM ingest_offsets WHERE file_path = ?`

	row := s.db.QueryRowContext(ctx, query, filePath)

	offset := &IngestOffset{}
	var inode sql.NullInt64
	var lastLineHash sql.NullString

	err := row.Scan(
		&offset.ID,
		&offset.FilePath,
		&offset.ByteOffset,
		&inode,
		&lastLineHash,
		&offset.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query file offset: %w", err)
	}

	if inode.Valid {
		offset.Inode = uint64(inode.Int64)
	}
	if lastLineHash.Valid {
		offset.LastLineHash = lastLineHash.String
	}

	return offset, nil
}

// SetFileOffset creates or updates the tracking offset for a file.
// Uses INSERT OR REPLACE for atomic upsert based on UNIQUE file_path constraint.
func (s *Store) SetFileOffset(ctx context.Context, offset *IngestOffset) error {
	query := `INSERT OR REPLACE INTO ingest_offsets
		(file_path, byte_offset, inode, last_line_hash, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`

	result, err := s.db.ExecContext(ctx, query,
		offset.FilePath,
		offset.ByteOffset,
		offset.Inode,
		offset.LastLineHash,
	)
	if err != nil {
		return fmt.Errorf("set file offset: %w", err)
	}

	// Get the inserted/updated ID
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	offset.ID = id

	return nil
}

// DeleteFileOffset removes the tracking offset for a specific file.
func (s *Store) DeleteFileOffset(ctx context.Context, filePath string) error {
	query := `DELETE FROM ingest_offsets WHERE file_path = ?`

	_, err := s.db.ExecContext(ctx, query, filePath)
	if err != nil {
		return fmt.Errorf("delete file offset: %w", err)
	}

	return nil
}

// UpsertClaudeSession creates or updates a session by external ID.
// Returns internal session_id and whether a new session was created.
// Uses INSERT OR IGNORE + SELECT pattern for idempotent upserts.
func (s *Store) UpsertClaudeSession(ctx context.Context,
	externalID string,
	projectPath string,
	agentType string,
) (sessionID int64, isNew bool, err error) {
	// Try to insert new session (will be ignored if external_session_id already exists)
	insertQuery := `INSERT OR IGNORE INTO behavioral_sessions
		(external_session_id, project_path, agent_type, session_start, task_execution_id,
		 total_tool_calls, total_bash_commands, total_file_operations, total_tokens_used, context_window_used)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, 0, 0, 0, 0, 0, 0)`

	result, err := s.db.ExecContext(ctx, insertQuery, externalID, projectPath, agentType)
	if err != nil {
		return 0, false, fmt.Errorf("insert session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, false, fmt.Errorf("get rows affected: %w", err)
	}
	isNew = rowsAffected > 0

	// Always SELECT to get the session ID (whether just inserted or existing)
	selectQuery := `SELECT id FROM behavioral_sessions WHERE external_session_id = ?`
	err = s.db.QueryRowContext(ctx, selectQuery, externalID).Scan(&sessionID)
	if err != nil {
		return 0, false, fmt.Errorf("get session id: %w", err)
	}

	return sessionID, isNew, nil
}

// AppendToolExecution adds a tool execution to an existing session.
// Inserts the tool record and increments the session's tool call counter.
func (s *Store) AppendToolExecution(ctx context.Context,
	sessionID int64,
	tool ToolExecutionData,
) error {
	query := `INSERT INTO tool_executions
		(session_id, tool_name, parameters, duration_ms, success, error_message)
		VALUES (?, ?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query,
		sessionID, tool.ToolName, tool.Parameters,
		tool.DurationMs, tool.Success, tool.ErrorMessage)
	if err != nil {
		return fmt.Errorf("append tool execution: %w", err)
	}

	// Increment session's tool call counter
	updateQuery := `UPDATE behavioral_sessions SET total_tool_calls = total_tool_calls + 1 WHERE id = ?`
	_, err = s.db.ExecContext(ctx, updateQuery, sessionID)
	if err != nil {
		return fmt.Errorf("increment tool call counter: %w", err)
	}

	return nil
}

// AppendBashCommand adds a bash command to an existing session.
// Inserts the bash record and increments the session's bash command counter.
func (s *Store) AppendBashCommand(ctx context.Context,
	sessionID int64,
	bash BashCommandData,
) error {
	query := `INSERT INTO bash_commands
		(session_id, command, duration_ms, exit_code, stdout_length, stderr_length, success)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query,
		sessionID, bash.Command, bash.DurationMs,
		bash.ExitCode, bash.StdoutLength, bash.StderrLength, bash.Success)
	if err != nil {
		return fmt.Errorf("append bash command: %w", err)
	}

	// Increment session's bash command counter
	updateQuery := `UPDATE behavioral_sessions SET total_bash_commands = total_bash_commands + 1 WHERE id = ?`
	_, err = s.db.ExecContext(ctx, updateQuery, sessionID)
	if err != nil {
		return fmt.Errorf("increment bash command counter: %w", err)
	}

	return nil
}

// AppendFileOperation adds a file operation to an existing session.
// Inserts the file op record and increments the session's file operations counter.
func (s *Store) AppendFileOperation(ctx context.Context,
	sessionID int64,
	fileOp FileOperationData,
) error {
	query := `INSERT INTO file_operations
		(session_id, operation_type, file_path, duration_ms, bytes_affected, success, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query,
		sessionID, fileOp.OperationType, fileOp.FilePath,
		fileOp.DurationMs, fileOp.BytesAffected, fileOp.Success, fileOp.ErrorMessage)
	if err != nil {
		return fmt.Errorf("append file operation: %w", err)
	}

	// Increment session's file operations counter
	updateQuery := `UPDATE behavioral_sessions SET total_file_operations = total_file_operations + 1 WHERE id = ?`
	_, err = s.db.ExecContext(ctx, updateQuery, sessionID)
	if err != nil {
		return fmt.Errorf("increment file operations counter: %w", err)
	}

	return nil
}

// UpdateSessionAggregates updates session totals (tokens, duration).
// Increments existing values atomically.
func (s *Store) UpdateSessionAggregates(ctx context.Context,
	sessionID int64,
	tokens int64,
	duration int64,
) error {
	query := `UPDATE behavioral_sessions SET
		total_tokens_used = total_tokens_used + ?,
		total_duration_seconds = COALESCE(total_duration_seconds, 0) + ?
		WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, tokens, duration, sessionID)
	if err != nil {
		return fmt.Errorf("update session aggregates: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found: %d", sessionID)
	}

	return nil
}

// UpdateSessionTimestamps updates a session's start/end times, duration, and model.
func (s *Store) UpdateSessionTimestamps(ctx context.Context,
	sessionID int64,
	startTime time.Time,
	endTime time.Time,
	durationSeconds int64,
	model string,
) error {
	// Build dynamic query based on what fields we have
	query := `UPDATE behavioral_sessions SET
		session_start = COALESCE(?, session_start),
		session_end = COALESCE(?, session_end),
		total_duration_seconds = ?,
		model_name = COALESCE(?, model_name)
		WHERE id = ?`

	var startVal, endVal, modelVal interface{}
	if !startTime.IsZero() {
		startVal = startTime
	}
	if !endTime.IsZero() {
		endVal = endTime
	}
	if model != "" {
		modelVal = model
	}

	_, err := s.db.ExecContext(ctx, query, startVal, endVal, durationSeconds, modelVal, sessionID)
	if err != nil {
		return fmt.Errorf("update session timestamps: %w", err)
	}

	return nil
}

// GetSessionByExternalID looks up a session by its Claude external session ID.
// Returns nil, nil if session not found (not an error - just not ingested yet).
func (s *Store) GetSessionByExternalID(ctx context.Context, externalID string) (*BehavioralSessionMetrics, error) {
	query := `SELECT
		bs.id, bs.task_execution_id, bs.session_start, bs.session_end,
		bs.total_duration_seconds, bs.total_tool_calls, bs.total_bash_commands,
		bs.total_file_operations, bs.total_tokens_used, bs.context_window_used,
		bs.external_session_id, bs.project_path, bs.agent_type
		FROM behavioral_sessions bs
		WHERE bs.external_session_id = ?`

	row := s.db.QueryRowContext(ctx, query, externalID)

	metrics := &BehavioralSessionMetrics{}
	var sessionEnd sql.NullTime
	var totalDuration, totalToolCalls, totalBashCmds, totalFileOps, totalTokens, contextWindow sql.NullInt64
	var externalSessionID, projectPath, agentType sql.NullString

	err := row.Scan(
		&metrics.SessionID,
		&metrics.TaskExecutionID,
		&metrics.SessionStart,
		&sessionEnd,
		&totalDuration,
		&totalToolCalls,
		&totalBashCmds,
		&totalFileOps,
		&totalTokens,
		&contextWindow,
		&externalSessionID,
		&projectPath,
		&agentType,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found - not an error
		}
		return nil, fmt.Errorf("query session by external id: %w", err)
	}

	if sessionEnd.Valid {
		metrics.SessionEnd = &sessionEnd.Time
	}
	if totalDuration.Valid {
		metrics.TotalDurationSecs = totalDuration.Int64
	}
	if totalToolCalls.Valid {
		metrics.TotalToolCalls = int(totalToolCalls.Int64)
	}
	if totalBashCmds.Valid {
		metrics.TotalBashCommands = int(totalBashCmds.Int64)
	}
	if totalFileOps.Valid {
		metrics.TotalFileOperations = int(totalFileOps.Int64)
	}
	if totalTokens.Valid {
		metrics.TotalTokensUsed = totalTokens.Int64
	}
	if contextWindow.Valid {
		metrics.ContextWindowUsed = int(contextWindow.Int64)
	}
	if agentType.Valid {
		metrics.Agent = agentType.String
	}

	return metrics, nil
}

// ListFileOffsets retrieves all tracked file offsets.
func (s *Store) ListFileOffsets(ctx context.Context) ([]*IngestOffset, error) {
	query := `SELECT id, file_path, byte_offset, inode, last_line_hash, updated_at
		FROM ingest_offsets ORDER BY file_path ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query file offsets: %w", err)
	}
	defer rows.Close()

	var offsets []*IngestOffset
	for rows.Next() {
		offset := &IngestOffset{}
		var inode sql.NullInt64
		var lastLineHash sql.NullString

		err := rows.Scan(
			&offset.ID,
			&offset.FilePath,
			&offset.ByteOffset,
			&inode,
			&lastLineHash,
			&offset.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan file offset row: %w", err)
		}

		if inode.Valid {
			offset.Inode = uint64(inode.Int64)
		}
		if lastLineHash.Valid {
			offset.LastLineHash = lastLineHash.String
		}

		offsets = append(offsets, offset)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate file offset rows: %w", err)
	}

	return offsets, nil
}

// RecentSession represents a recent session entry for display
type RecentSession struct {
	ID           int64
	TaskName     string
	Agent        string
	Success      bool
	DurationSecs int64
	Timestamp    time.Time
}

// GetRecentSessions returns recent sessions for a project, ordered by timestamp descending
func (s *Store) GetRecentSessions(ctx context.Context, project string, limit, offset int) ([]RecentSession, error) {
	query := `
		SELECT
			bs.id,
			COALESCE(te.task_name, 'Agent Session: ' || SUBSTR(bs.external_session_id, 1, 8)),
			COALESCE(te.agent, bs.agent_type),
			COALESCE(te.success, 1),
			bs.total_duration_seconds,
			bs.session_start
		FROM behavioral_sessions bs
		LEFT JOIN task_executions te ON bs.task_execution_id = te.id AND bs.task_execution_id > 0
	`

	args := []interface{}{}

	if project != "" {
		query += " WHERE (te.plan_file LIKE ? OR bs.project_path LIKE ?)"
		args = append(args, "%"+project+"%", "%"+project+"%")
	}

	query += " ORDER BY bs.session_start DESC"

	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query recent sessions: %w", err)
	}
	defer rows.Close()

	var sessions []RecentSession
	for rows.Next() {
		var rs RecentSession
		var agent sql.NullString
		var durationSecs sql.NullInt64

		if err := rows.Scan(
			&rs.ID,
			&rs.TaskName,
			&agent,
			&rs.Success,
			&durationSecs,
			&rs.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("scan recent session row: %w", err)
		}

		if agent.Valid {
			rs.Agent = agent.String
		}
		if durationSecs.Valid {
			rs.DurationSecs = durationSecs.Int64
		}

		sessions = append(sessions, rs)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent sessions: %w", err)
	}

	return sessions, nil
}

// SuccessfulPattern represents a stored successful task pattern
type SuccessfulPattern struct {
	TaskHash           string
	PatternDescription string
	SuccessCount       int
	LastAgent          string
	LastUsed           time.Time
	CreatedAt          time.Time
	Metadata           string // JSON blob
}

// AddPattern inserts or updates a successful pattern with success count increment
func (s *Store) AddPattern(ctx context.Context, pattern *SuccessfulPattern) error {
	// Use INSERT OR REPLACE to upsert, incrementing success_count if exists
	query := `INSERT INTO successful_patterns
		(task_hash, pattern_description, success_count, last_agent, last_used, metadata)
		VALUES (?, ?,
			COALESCE((SELECT success_count FROM successful_patterns WHERE task_hash = ?), 0) + 1,
			?, CURRENT_TIMESTAMP, ?)
		ON CONFLICT(task_hash) DO UPDATE SET
			pattern_description = excluded.pattern_description,
			success_count = successful_patterns.success_count + 1,
			last_agent = excluded.last_agent,
			last_used = CURRENT_TIMESTAMP,
			metadata = excluded.metadata`

	metadataJSON := pattern.Metadata
	if metadataJSON == "" {
		metadataJSON = "{}"
	}

	_, err := s.db.ExecContext(ctx, query,
		pattern.TaskHash,
		pattern.PatternDescription,
		pattern.TaskHash, // For the subquery
		pattern.LastAgent,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("add pattern: %w", err)
	}

	return nil
}

// GetPattern retrieves a specific pattern by hash
func (s *Store) GetPattern(ctx context.Context, taskHash string) (*SuccessfulPattern, error) {
	query := `SELECT task_hash, pattern_description, success_count, last_agent, last_used, created_at, metadata
		FROM successful_patterns WHERE task_hash = ?`

	row := s.db.QueryRowContext(ctx, query, taskHash)

	pattern := &SuccessfulPattern{}
	var lastAgent, metadata sql.NullString

	err := row.Scan(
		&pattern.TaskHash,
		&pattern.PatternDescription,
		&pattern.SuccessCount,
		&lastAgent,
		&pattern.LastUsed,
		&pattern.CreatedAt,
		&metadata,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get pattern: %w", err)
	}

	if lastAgent.Valid {
		pattern.LastAgent = lastAgent.String
	}
	if metadata.Valid {
		pattern.Metadata = metadata.String
	}

	return pattern, nil
}

// GetSimilarPatterns returns patterns with similar hash prefix (fuzzy matching)
func (s *Store) GetSimilarPatterns(ctx context.Context, hashPrefix string, limit int) ([]*SuccessfulPattern, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `SELECT task_hash, pattern_description, success_count, last_agent, last_used, created_at, metadata
		FROM successful_patterns
		WHERE task_hash LIKE ?
		ORDER BY success_count DESC, last_used DESC
		LIMIT ?`

	rows, err := s.db.QueryContext(ctx, query, hashPrefix+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("get similar patterns: %w", err)
	}
	defer rows.Close()

	var patterns []*SuccessfulPattern
	for rows.Next() {
		pattern := &SuccessfulPattern{}
		var lastAgent, metadata sql.NullString

		err := rows.Scan(
			&pattern.TaskHash,
			&pattern.PatternDescription,
			&pattern.SuccessCount,
			&lastAgent,
			&pattern.LastUsed,
			&pattern.CreatedAt,
			&metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("scan pattern row: %w", err)
		}

		if lastAgent.Valid {
			pattern.LastAgent = lastAgent.String
		}
		if metadata.Valid {
			pattern.Metadata = metadata.String
		}

		patterns = append(patterns, pattern)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pattern rows: %w", err)
	}

	return patterns, nil
}

// GetTopPatterns returns the most successful patterns ordered by success count
func (s *Store) GetTopPatterns(ctx context.Context, limit int) ([]*SuccessfulPattern, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `SELECT task_hash, pattern_description, success_count, last_agent, last_used, created_at, metadata
		FROM successful_patterns
		ORDER BY success_count DESC, last_used DESC
		LIMIT ?`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("get top patterns: %w", err)
	}
	defer rows.Close()

	var patterns []*SuccessfulPattern
	for rows.Next() {
		pattern := &SuccessfulPattern{}
		var lastAgent, metadata sql.NullString

		err := rows.Scan(
			&pattern.TaskHash,
			&pattern.PatternDescription,
			&pattern.SuccessCount,
			&lastAgent,
			&pattern.LastUsed,
			&pattern.CreatedAt,
			&metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("scan pattern row: %w", err)
		}

		if lastAgent.Valid {
			pattern.LastAgent = lastAgent.String
		}
		if metadata.Valid {
			pattern.Metadata = metadata.String
		}

		patterns = append(patterns, pattern)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pattern rows: %w", err)
	}

	return patterns, nil
}

// DuplicateDetection represents a recorded duplicate detection event
type DuplicateDetection struct {
	ID          int64
	SourceHash  string
	MatchedHash string
	Similarity  float64
	Action      string // 'blocked', 'warned', 'suggested'
	TaskName    string
	DetectedAt  time.Time
	Metadata    string // JSON blob
}

// RecordDuplicateDetection records when a duplicate was detected
func (s *Store) RecordDuplicateDetection(ctx context.Context, detection *DuplicateDetection) error {
	query := `INSERT INTO duplicate_detections
		(source_hash, matched_hash, similarity, action, task_name, metadata)
		VALUES (?, ?, ?, ?, ?, ?)`

	metadataJSON := detection.Metadata
	if metadataJSON == "" {
		metadataJSON = "{}"
	}

	result, err := s.db.ExecContext(ctx, query,
		detection.SourceHash,
		detection.MatchedHash,
		detection.Similarity,
		detection.Action,
		detection.TaskName,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("record duplicate detection: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	detection.ID = id

	return nil
}

// GetDuplicateDetections retrieves duplicate detections for a source hash
func (s *Store) GetDuplicateDetections(ctx context.Context, sourceHash string, limit int) ([]*DuplicateDetection, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `SELECT id, source_hash, matched_hash, similarity, action, task_name, detected_at, metadata
		FROM duplicate_detections
		WHERE source_hash = ?
		ORDER BY detected_at DESC
		LIMIT ?`

	rows, err := s.db.QueryContext(ctx, query, sourceHash, limit)
	if err != nil {
		return nil, fmt.Errorf("get duplicate detections: %w", err)
	}
	defer rows.Close()

	var detections []*DuplicateDetection
	for rows.Next() {
		detection := &DuplicateDetection{}
		var taskName, metadata sql.NullString

		err := rows.Scan(
			&detection.ID,
			&detection.SourceHash,
			&detection.MatchedHash,
			&detection.Similarity,
			&detection.Action,
			&taskName,
			&detection.DetectedAt,
			&metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("scan duplicate detection row: %w", err)
		}

		if taskName.Valid {
			detection.TaskName = taskName.String
		}
		if metadata.Valid {
			detection.Metadata = metadata.String
		}

		detections = append(detections, detection)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate duplicate detection rows: %w", err)
	}

	return detections, nil
}

// STOPAnalysis represents a stored STOP protocol analysis result
type STOPAnalysis struct {
	ID                 int64
	TaskHash           string
	TaskName           string
	SearchResults      string // JSON blob
	ThinkAnalysis      string // JSON blob
	OutlinePlan        string // JSON blob
	ProveJustification string // JSON blob
	FinalDecision      string // 'proceed', 'skip', 'modify'
	Confidence         float64
	AnalyzedAt         time.Time
	Metadata           string // JSON blob
}

// SaveSTOPAnalysis stores a STOP protocol analysis result
func (s *Store) SaveSTOPAnalysis(ctx context.Context, analysis *STOPAnalysis) error {
	query := `INSERT INTO stop_analyses
		(task_hash, task_name, search_results, think_analysis, outline_plan, prove_justification, final_decision, confidence, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	metadataJSON := analysis.Metadata
	if metadataJSON == "" {
		metadataJSON = "{}"
	}

	result, err := s.db.ExecContext(ctx, query,
		analysis.TaskHash,
		analysis.TaskName,
		analysis.SearchResults,
		analysis.ThinkAnalysis,
		analysis.OutlinePlan,
		analysis.ProveJustification,
		analysis.FinalDecision,
		analysis.Confidence,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("record STOP analysis: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	analysis.ID = id

	return nil
}

// GetSTOPAnalysis retrieves the most recent STOP analysis for a task hash
func (s *Store) GetSTOPAnalysis(ctx context.Context, taskHash string) (*STOPAnalysis, error) {
	query := `SELECT id, task_hash, task_name, search_results, think_analysis, outline_plan, prove_justification, final_decision, confidence, analyzed_at, metadata
		FROM stop_analyses
		WHERE task_hash = ?
		ORDER BY id DESC
		LIMIT 1`

	row := s.db.QueryRowContext(ctx, query, taskHash)

	analysis := &STOPAnalysis{}
	var taskName, searchResults, thinkAnalysis, outlinePlan, proveJustification, finalDecision, metadata sql.NullString
	var confidence sql.NullFloat64

	err := row.Scan(
		&analysis.ID,
		&analysis.TaskHash,
		&taskName,
		&searchResults,
		&thinkAnalysis,
		&outlinePlan,
		&proveJustification,
		&finalDecision,
		&confidence,
		&analysis.AnalyzedAt,
		&metadata,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get STOP analysis: %w", err)
	}

	if taskName.Valid {
		analysis.TaskName = taskName.String
	}
	if searchResults.Valid {
		analysis.SearchResults = searchResults.String
	}
	if thinkAnalysis.Valid {
		analysis.ThinkAnalysis = thinkAnalysis.String
	}
	if outlinePlan.Valid {
		analysis.OutlinePlan = outlinePlan.String
	}
	if proveJustification.Valid {
		analysis.ProveJustification = proveJustification.String
	}
	if finalDecision.Valid {
		analysis.FinalDecision = finalDecision.String
	}
	if confidence.Valid {
		analysis.Confidence = confidence.Float64
	}
	if metadata.Valid {
		analysis.Metadata = metadata.String
	}

	return analysis, nil
}

// GetRecentSTOPAnalyses retrieves recent STOP analyses
func (s *Store) GetRecentSTOPAnalyses(ctx context.Context, limit int) ([]*STOPAnalysis, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `SELECT id, task_hash, task_name, search_results, think_analysis, outline_plan, prove_justification, final_decision, confidence, analyzed_at, metadata
		FROM stop_analyses
		ORDER BY id DESC
		LIMIT ?`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("get recent STOP analyses: %w", err)
	}
	defer rows.Close()

	var analyses []*STOPAnalysis
	for rows.Next() {
		analysis := &STOPAnalysis{}
		var taskName, searchResults, thinkAnalysis, outlinePlan, proveJustification, finalDecision, metadata sql.NullString
		var confidence sql.NullFloat64

		err := rows.Scan(
			&analysis.ID,
			&analysis.TaskHash,
			&taskName,
			&searchResults,
			&thinkAnalysis,
			&outlinePlan,
			&proveJustification,
			&finalDecision,
			&confidence,
			&analysis.AnalyzedAt,
			&metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("scan STOP analysis row: %w", err)
		}

		if taskName.Valid {
			analysis.TaskName = taskName.String
		}
		if searchResults.Valid {
			analysis.SearchResults = searchResults.String
		}
		if thinkAnalysis.Valid {
			analysis.ThinkAnalysis = thinkAnalysis.String
		}
		if outlinePlan.Valid {
			analysis.OutlinePlan = outlinePlan.String
		}
		if proveJustification.Valid {
			analysis.ProveJustification = proveJustification.String
		}
		if finalDecision.Valid {
			analysis.FinalDecision = finalDecision.String
		}
		if confidence.Valid {
			analysis.Confidence = confidence.Float64
		}
		if metadata.Valid {
			analysis.Metadata = metadata.String
		}

		analyses = append(analyses, analysis)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate STOP analysis rows: %w", err)
	}

	return analyses, nil
}

// UpdateCommitVerification updates the commit verification columns for a task execution.
// This is called after commit verification completes to persist results for analytics.
// Uses the most recent execution for the given plan_file, task_number, and run_number.
func (s *Store) UpdateCommitVerification(ctx context.Context, planFile, taskNumber string, runNumber int, verified bool, commitHash string) error {
	query := `UPDATE task_executions
		SET commit_verified = ?, commit_hash = ?
		WHERE plan_file = ? AND task_number = ? AND run_number = ?
		AND id = (
			SELECT id FROM task_executions
			WHERE plan_file = ? AND task_number = ? AND run_number = ?
			ORDER BY id DESC LIMIT 1
		)`

	_, err := s.db.ExecContext(ctx, query,
		verified, commitHash,
		planFile, taskNumber, runNumber,
		planFile, taskNumber, runNumber,
	)
	if err != nil {
		return fmt.Errorf("update commit verification: %w", err)
	}

	return nil
}
