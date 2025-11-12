package learning

import (
	"context"
	"database/sql"
	"encoding/json"
	_ "embed"
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

// initSchema executes the embedded SQL schema
func (s *Store) initSchema() error {
	_, err := s.db.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("execute schema: %w", err)
	}
	return nil
}

// getSchemaVersion retrieves the current schema version
func (s *Store) getSchemaVersion() (int, error) {
	var version int
	err := s.db.QueryRow("SELECT version FROM schema_version ORDER BY version DESC LIMIT 1").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("query schema version: %w", err)
	}
	return version, nil
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
