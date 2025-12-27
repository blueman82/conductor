package learning

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Compile-time assertion that Store implements LIPCollector
var _ LIPCollector = (*Store)(nil)

// Progress scoring weights for different event types
const (
	// Weights for existing behavioral data
	toolExecutionWeight   = 0.1
	fileOperationWeight   = 0.2

	// Weights for new LIP events
	testPassWeight        = 0.3
	testFailWeight        = 0.1 // Partial credit for attempting tests
	buildSuccessWeight    = 0.3
	buildFailWeight       = 0.1 // Partial credit for attempting builds
)

// RecordEvent stores a new LIP progress event in the database.
// Implements LIPCollector.RecordEvent.
func (s *Store) RecordEvent(ctx context.Context, event *LIPEvent) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// Validate event type
	switch event.EventType {
	case LIPEventTestPass, LIPEventTestFail, LIPEventBuildSuccess, LIPEventBuildFail:
		// Valid event types
	default:
		return fmt.Errorf("invalid event type: %s", event.EventType)
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Default confidence to 1.0 if not set
	if event.Confidence == 0 {
		event.Confidence = 1.0
	}

	query := `INSERT INTO lip_events
		(task_execution_id, task_number, event_type, timestamp, details, confidence)
		VALUES (?, ?, ?, ?, ?, ?)`

	result, err := s.db.ExecContext(ctx, query,
		event.TaskExecutionID,
		event.TaskNumber,
		string(event.EventType),
		event.Timestamp,
		event.Details,
		event.Confidence,
	)
	if err != nil {
		return fmt.Errorf("insert LIP event: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	event.ID = id

	return nil
}

// GetEvents retrieves LIP events matching the filter criteria.
// Implements LIPCollector.GetEvents.
// This method aggregates events from both the lip_events table and
// existing behavioral tables (tool_executions, file_operations).
func (s *Store) GetEvents(ctx context.Context, filter *LIPFilter) ([]LIPEvent, error) {
	if filter == nil {
		filter = &LIPFilter{}
	}

	var events []LIPEvent

	// Query LIP events table
	lipEvents, err := s.getLIPEventsFromTable(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("get LIP events: %w", err)
	}
	events = append(events, lipEvents...)

	// If no specific event types requested, or if we want all events,
	// also include derived events from behavioral tables
	if len(filter.EventTypes) == 0 {
		behavioralEvents, err := s.getDerivedEventsFromBehavioral(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("get behavioral events: %w", err)
		}
		events = append(events, behavioralEvents...)
	}

	// Sort by timestamp descending
	sortLIPEventsByTimestamp(events)

	// Apply limit if specified
	if filter.Limit > 0 && len(events) > filter.Limit {
		events = events[:filter.Limit]
	}

	return events, nil
}

// getLIPEventsFromTable queries the lip_events table directly.
func (s *Store) getLIPEventsFromTable(ctx context.Context, filter *LIPFilter) ([]LIPEvent, error) {
	query := `SELECT id, task_execution_id, task_number, event_type, timestamp, details, confidence
		FROM lip_events WHERE 1=1`
	args := []interface{}{}

	if filter.TaskNumber != "" {
		query += " AND task_number = ?"
		args = append(args, filter.TaskNumber)
	}

	if filter.TaskExecutionID > 0 {
		query += " AND task_execution_id = ?"
		args = append(args, filter.TaskExecutionID)
	}

	if !filter.TimeRangeStart.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filter.TimeRangeStart)
	}

	if !filter.TimeRangeEnd.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, filter.TimeRangeEnd)
	}

	if len(filter.EventTypes) > 0 {
		placeholders := make([]string, len(filter.EventTypes))
		for i, et := range filter.EventTypes {
			placeholders[i] = "?"
			args = append(args, string(et))
		}
		query += fmt.Sprintf(" AND event_type IN (%s)", joinStrings(placeholders, ","))
	}

	if filter.MinConfidence > 0 {
		query += " AND confidence >= ?"
		args = append(args, filter.MinConfidence)
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query LIP events: %w", err)
	}
	defer rows.Close()

	var events []LIPEvent
	for rows.Next() {
		var event LIPEvent
		var eventType string
		var details sql.NullString

		err := rows.Scan(
			&event.ID,
			&event.TaskExecutionID,
			&event.TaskNumber,
			&eventType,
			&event.Timestamp,
			&details,
			&event.Confidence,
		)
		if err != nil {
			return nil, fmt.Errorf("scan LIP event: %w", err)
		}

		event.EventType = LIPEventType(eventType)
		if details.Valid {
			event.Details = details.String
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate LIP events: %w", err)
	}

	return events, nil
}

// getDerivedEventsFromBehavioral creates LIP events from existing behavioral data.
// This provides backward compatibility with sessions recorded before LIP was implemented.
func (s *Store) getDerivedEventsFromBehavioral(ctx context.Context, filter *LIPFilter) ([]LIPEvent, error) {
	// For now, return empty - behavioral data doesn't directly map to LIP events
	// (test_pass, test_fail, build_success, build_fail)
	// Tool executions and file operations contribute to progress score but don't
	// represent discrete LIP events
	return nil, nil
}

// CalculateProgress computes an aggregate progress score for a task execution.
// Implements LIPCollector.CalculateProgress.
// This considers both LIP events and existing behavioral data (tool_executions, file_operations).
func (s *Store) CalculateProgress(ctx context.Context, taskExecutionID int64) (ProgressScore, error) {
	if taskExecutionID <= 0 {
		return ProgressNone, fmt.Errorf("invalid task execution ID: %d", taskExecutionID)
	}

	var totalScore float64 = 0.0
	var maxScore float64 = 1.0

	// 1. Score from LIP events (test/build results)
	lipScore, err := s.calculateLIPEventScore(ctx, taskExecutionID)
	if err != nil {
		return ProgressNone, fmt.Errorf("calculate LIP score: %w", err)
	}
	totalScore += lipScore

	// 2. Score from tool executions (weighted less, as they're intermediate steps)
	toolScore, err := s.calculateToolExecutionScore(ctx, taskExecutionID)
	if err != nil {
		return ProgressNone, fmt.Errorf("calculate tool score: %w", err)
	}
	totalScore += toolScore

	// 3. Score from file operations (weighted moderately, as they're tangible progress)
	fileScore, err := s.calculateFileOperationScore(ctx, taskExecutionID)
	if err != nil {
		return ProgressNone, fmt.Errorf("calculate file score: %w", err)
	}
	totalScore += fileScore

	// Normalize to 0.0-1.0 range
	if totalScore > maxScore {
		totalScore = maxScore
	}

	return ProgressScore(totalScore), nil
}

// calculateLIPEventScore computes the score contribution from LIP events.
func (s *Store) calculateLIPEventScore(ctx context.Context, taskExecutionID int64) (float64, error) {
	query := `SELECT event_type, COUNT(*) as count, AVG(confidence) as avg_confidence
		FROM lip_events
		WHERE task_execution_id = ?
		GROUP BY event_type`

	rows, err := s.db.QueryContext(ctx, query, taskExecutionID)
	if err != nil {
		return 0, fmt.Errorf("query LIP event scores: %w", err)
	}
	defer rows.Close()

	var score float64 = 0.0
	for rows.Next() {
		var eventType string
		var count int
		var avgConfidence float64

		if err := rows.Scan(&eventType, &count, &avgConfidence); err != nil {
			return 0, fmt.Errorf("scan LIP event score: %w", err)
		}

		// Apply weights based on event type
		switch LIPEventType(eventType) {
		case LIPEventTestPass:
			score += testPassWeight * avgConfidence
		case LIPEventTestFail:
			score += testFailWeight * avgConfidence
		case LIPEventBuildSuccess:
			score += buildSuccessWeight * avgConfidence
		case LIPEventBuildFail:
			score += buildFailWeight * avgConfidence
		}
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate LIP event scores: %w", err)
	}

	return score, nil
}

// calculateToolExecutionScore computes the score contribution from tool executions.
func (s *Store) calculateToolExecutionScore(ctx context.Context, taskExecutionID int64) (float64, error) {
	query := `SELECT COUNT(*) as total, COALESCE(SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END), 0) as success_count
		FROM tool_executions te
		JOIN behavioral_sessions bs ON te.session_id = bs.id
		WHERE bs.task_execution_id = ?`

	var total, successCount int
	err := s.db.QueryRowContext(ctx, query, taskExecutionID).Scan(&total, &successCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("query tool execution score: %w", err)
	}

	if total == 0 {
		return 0, nil
	}

	// Score based on success rate of tool executions, weighted by the constant
	successRate := float64(successCount) / float64(total)
	return toolExecutionWeight * successRate, nil
}

// calculateFileOperationScore computes the score contribution from file operations.
func (s *Store) calculateFileOperationScore(ctx context.Context, taskExecutionID int64) (float64, error) {
	query := `SELECT COUNT(*) as total, COALESCE(SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END), 0) as success_count
		FROM file_operations fo
		JOIN behavioral_sessions bs ON fo.session_id = bs.id
		WHERE bs.task_execution_id = ?`

	var total, successCount int
	err := s.db.QueryRowContext(ctx, query, taskExecutionID).Scan(&total, &successCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("query file operation score: %w", err)
	}

	if total == 0 {
		return 0, nil
	}

	// Score based on success rate of file operations, weighted by the constant
	successRate := float64(successCount) / float64(total)
	return fileOperationWeight * successRate, nil
}

// Helper functions

// sortLIPEventsByTimestamp sorts events by timestamp descending (most recent first)
func sortLIPEventsByTimestamp(events []LIPEvent) {
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			if events[j].Timestamp.After(events[i].Timestamp) {
				events[i], events[j] = events[j], events[i]
			}
		}
	}
}

// joinStrings joins strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// GetLIPEventsByTask retrieves all LIP events for a specific task number.
// This is a convenience method that wraps GetEvents.
func (s *Store) GetLIPEventsByTask(ctx context.Context, taskNumber string) ([]LIPEvent, error) {
	return s.GetEvents(ctx, &LIPFilter{
		TaskNumber: taskNumber,
	})
}

// GetLIPEventsByExecution retrieves all LIP events for a specific task execution.
// This is a convenience method that wraps GetEvents.
func (s *Store) GetLIPEventsByExecution(ctx context.Context, taskExecutionID int64) ([]LIPEvent, error) {
	return s.GetEvents(ctx, &LIPFilter{
		TaskExecutionID: taskExecutionID,
	})
}

// RecordTestResult is a convenience method for recording test results.
func (s *Store) RecordTestResult(ctx context.Context, taskExecutionID int64, taskNumber string, passed bool, details string) error {
	eventType := LIPEventTestPass
	if !passed {
		eventType = LIPEventTestFail
	}

	return s.RecordEvent(ctx, &LIPEvent{
		TaskExecutionID: taskExecutionID,
		TaskNumber:      taskNumber,
		EventType:       eventType,
		Details:         details,
		Confidence:      1.0,
	})
}

// RecordBuildResult is a convenience method for recording build results.
func (s *Store) RecordBuildResult(ctx context.Context, taskExecutionID int64, taskNumber string, success bool, details string) error {
	eventType := LIPEventBuildSuccess
	if !success {
		eventType = LIPEventBuildFail
	}

	return s.RecordEvent(ctx, &LIPEvent{
		TaskExecutionID: taskExecutionID,
		TaskNumber:      taskNumber,
		EventType:       eventType,
		Details:         details,
		Confidence:      1.0,
	})
}
