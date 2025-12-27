package learning

import (
	"context"
	"time"
)

// LIPEventType represents the type of progress event being recorded.
// These event types capture test and build results that complement
// the existing behavioral tracking (ToolExecutionData, BashCommandData, FileOperationData).
type LIPEventType string

const (
	// LIPEventTestPass indicates a test passed during execution
	LIPEventTestPass LIPEventType = "test_pass"
	// LIPEventTestFail indicates a test failed during execution
	LIPEventTestFail LIPEventType = "test_fail"
	// LIPEventBuildSuccess indicates a build completed successfully
	LIPEventBuildSuccess LIPEventType = "build_success"
	// LIPEventBuildFail indicates a build failed
	LIPEventBuildFail LIPEventType = "build_fail"
)

// LIPEvent represents a single progress event during task execution.
// This struct captures test and build events that are NOT tracked by
// existing behavioral data (ToolExecutionData, BashCommandData, FileOperationData).
// These existing types already capture tool calls, file edits, and commands.
type LIPEvent struct {
	// ID is the unique identifier for this event (auto-generated on storage)
	ID int64 `json:"id,omitempty"`

	// TaskExecutionID links this event to a specific task execution
	TaskExecutionID int64 `json:"task_execution_id"`

	// TaskNumber identifies the task (e.g., "1", "2a")
	TaskNumber string `json:"task_number"`

	// EventType identifies what kind of progress event occurred
	EventType LIPEventType `json:"event_type"`

	// Timestamp when the event occurred
	Timestamp time.Time `json:"timestamp"`

	// Details contains event-specific information (test name, error message, etc.)
	Details string `json:"details,omitempty"`

	// Confidence indicates how certain we are about this event (0.0-1.0)
	// Higher values indicate more reliable detection
	Confidence float64 `json:"confidence"`
}

// ProgressScore represents a normalized progress value (0.0-1.0).
// This type enables consistent progress tracking across different event types
// and allows aggregation of partial progress for failed tasks.
type ProgressScore float64

const (
	// ProgressNone indicates no progress was made (0.0)
	ProgressNone ProgressScore = 0.0
	// ProgressPartial indicates some progress was made (0.5)
	ProgressPartial ProgressScore = 0.5
	// ProgressComplete indicates the task completed successfully (1.0)
	ProgressComplete ProgressScore = 1.0
)

// LIPFilter defines criteria for querying progress events.
// It supports flexible filtering across both new LIP events and
// existing behavioral data for unified warm-up context queries.
type LIPFilter struct {
	// TaskNumber filters events for a specific task
	TaskNumber string

	// TaskExecutionID filters events for a specific execution
	TaskExecutionID int64

	// TimeRange filters events within a time window
	TimeRangeStart time.Time
	TimeRangeEnd   time.Time

	// EventTypes filters to specific event types (empty = all types)
	EventTypes []LIPEventType

	// MinConfidence filters events above this confidence threshold
	MinConfidence float64

	// Limit restricts the number of results (0 = no limit)
	Limit int
}

// LIPCollector defines the interface for recording and querying progress events.
// Implementations handle storage and retrieval of LIP events alongside
// existing behavioral data for comprehensive progress tracking.
type LIPCollector interface {
	// RecordEvent stores a new progress event
	RecordEvent(ctx context.Context, event *LIPEvent) error

	// GetEvents retrieves events matching the filter criteria
	GetEvents(ctx context.Context, filter *LIPFilter) ([]LIPEvent, error)

	// CalculateProgress computes an aggregate progress score for a task execution
	// This considers both LIP events and existing behavioral data
	CalculateProgress(ctx context.Context, taskExecutionID int64) (ProgressScore, error)
}
