package learning

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	TaskExecutionID      int64
	SessionStart         time.Time
	SessionEnd           *time.Time
	TotalDurationSecs    int64
	TotalToolCalls       int
	TotalBashCommands    int
	TotalFileOperations  int
	TotalTokensUsed      int64
	ContextWindowUsed    int
}

// BehavioralSessionMetrics represents complete session metrics with execution context
type BehavioralSessionMetrics struct {
	SessionID            int64
	TaskExecutionID      int64
	SessionStart         time.Time
	SessionEnd           *time.Time
	TotalDurationSecs    int64
	TotalToolCalls       int
	TotalBashCommands    int
	TotalFileOperations  int
	TotalTokensUsed      int64
	ContextWindowUsed    int
	// Execution context
	TaskNumber           string
	TaskName             string
	Agent                string
	Success              bool
}

// ToolExecutionData represents a tool execution record
type ToolExecutionData struct {
	SessionID      int64
	ToolName       string
	Parameters     string
	DurationMs     int64
	Success        bool
	ErrorMessage   string
}

// BashCommandData represents a bash command execution
type BashCommandData struct {
	SessionID      int64
	Command        string
	DurationMs     int64
	ExitCode       int
	StdoutLength   int
	StderrLength   int
	Success        bool
}

// FileOperationData represents a file operation
type FileOperationData struct {
	SessionID      int64
	OperationType  string
	FilePath       string
	DurationMs     int64
	BytesAffected  int64
	Success        bool
	ErrorMessage   string
}

// TokenUsageData represents token usage measurement
type TokenUsageData struct {
	SessionID          int64
	InputTokens        int64
	OutputTokens       int64
	TotalTokens        int64
	ContextWindowSize  int
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
		te.task_number, te.task_name, te.agent, te.success
		FROM behavioral_sessions bs
		JOIN task_executions te ON bs.task_execution_id = te.id
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
		"session_count": sessionCount,
		"avg_duration_seconds": 0.0,
		"avg_tool_calls": 0.0,
		"avg_bash_commands": 0.0,
		"avg_file_operations": 0.0,
		"total_tokens": int64(0),
		"avg_context_window": 0.0,
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
		"session_count": sessionCount,
		"success_count": successCount,
		"avg_duration_seconds": 0.0,
		"total_tool_calls": int64(0),
		"total_bash_commands": int64(0),
		"total_file_operations": int64(0),
		"total_tokens": int64(0),
		"avg_context_window": 0.0,
		"success_rate": 0.0,
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
