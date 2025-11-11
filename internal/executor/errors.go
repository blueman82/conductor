package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ExecutionPhase represents the phase of execution where an error occurred.
type ExecutionPhase int

const (
	// PhaseGraph represents errors during dependency graph calculation.
	PhaseGraph ExecutionPhase = iota
	// PhaseWave represents errors during wave execution.
	PhaseWave
	// PhaseTask represents errors during task execution.
	PhaseTask
	// PhaseQC represents errors during quality control review.
	PhaseQC
)

// String returns the string representation of ExecutionPhase.
func (p ExecutionPhase) String() string {
	switch p {
	case PhaseGraph:
		return "graph"
	case PhaseWave:
		return "wave"
	case PhaseTask:
		return "task"
	case PhaseQC:
		return "qc"
	default:
		return "unknown"
	}
}

// TaskError represents an error that occurred during task execution.
// It includes context about which task failed and when.
type TaskError struct {
	TaskName  string    // Name or number of the task that failed
	Message   string    // Human-readable error message
	Err       error     // Underlying error (optional)
	Timestamp time.Time // When the error occurred
}

// NewTaskError creates a new TaskError with the current timestamp.
func NewTaskError(name, msg string, err error) *TaskError {
	return &TaskError{
		TaskName:  name,
		Message:   msg,
		Err:       err,
		Timestamp: time.Now(),
	}
}

// Error implements the error interface for TaskError.
func (e *TaskError) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("task %s: %s", e.TaskName, e.Message))
	if e.Err != nil {
		sb.WriteString(fmt.Sprintf(": %v", e.Err))
	}
	return sb.String()
}

// Unwrap returns the underlying error for error wrapping support.
func (e *TaskError) Unwrap() error {
	return e.Err
}

// ExecutionError aggregates multiple task errors that occurred during execution.
// It provides context about which phase of execution failed and how many tasks were affected.
type ExecutionError struct {
	Phase       ExecutionPhase // Execution phase where errors occurred
	TaskErrors  []*TaskError   // Individual task errors
	TotalTasks  int            // Total number of tasks attempted
	FailedTasks int            // Number of tasks that failed
}

// NewExecutionError creates a new ExecutionError for the given phase.
func NewExecutionError(phase string) *ExecutionError {
	var p ExecutionPhase
	switch phase {
	case "graph":
		p = PhaseGraph
	case "wave":
		p = PhaseWave
	case "task":
		p = PhaseTask
	case "qc":
		p = PhaseQC
	default:
		p = PhaseTask // default to task phase
	}

	return &ExecutionError{
		Phase:      p,
		TaskErrors: []*TaskError{},
	}
}

// AddTask adds a task error to the execution error and increments the failed task count.
func (e *ExecutionError) AddTask(taskErr *TaskError) {
	e.TaskErrors = append(e.TaskErrors, taskErr)
	e.FailedTasks++
}

// Error implements the error interface for ExecutionError.
func (e *ExecutionError) Error() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("execution failed in %s phase: %d/%d tasks failed",
		e.Phase, e.FailedTasks, e.TotalTasks))

	if len(e.TaskErrors) > 0 {
		sb.WriteString(":")
		for _, taskErr := range e.TaskErrors {
			sb.WriteString(fmt.Sprintf("\n  - %s", taskErr.Error()))
		}
	}

	return sb.String()
}

// Unwrap returns the task errors for error unwrapping support.
// This allows errors.Is and errors.As to traverse the error chain.
func (e *ExecutionError) Unwrap() []error {
	if len(e.TaskErrors) == 0 {
		return nil
	}

	errs := make([]error, len(e.TaskErrors))
	for i, taskErr := range e.TaskErrors {
		errs[i] = taskErr
	}
	return errs
}

// TimeoutError represents a timeout error during task execution.
type TimeoutError struct {
	TaskName        string        // Name or number of the task that timed out
	TimeoutDuration time.Duration // Duration after which timeout occurred
	Context         string        // Additional context about what was happening (optional)
	Timestamp       time.Time     // When the timeout occurred
}

// NewTimeoutError creates a new TimeoutError with the current timestamp.
func NewTimeoutError(name string, duration time.Duration) *TimeoutError {
	return &TimeoutError{
		TaskName:        name,
		TimeoutDuration: duration,
		Timestamp:       time.Now(),
	}
}

// Error implements the error interface for TimeoutError.
func (e *TimeoutError) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("task %s: timeout after %v", e.TaskName, e.TimeoutDuration))
	if e.Context != "" {
		sb.WriteString(fmt.Sprintf(" (%s)", e.Context))
	}
	return sb.String()
}

// Unwrap returns context.DeadlineExceeded to support error wrapping.
func (e *TimeoutError) Unwrap() error {
	return context.DeadlineExceeded
}

// IsTaskError checks if the error is or wraps a TaskError.
func IsTaskError(err error) bool {
	if err == nil {
		return false
	}
	var te *TaskError
	return errors.As(err, &te)
}

// IsTimeoutError checks if the error is or wraps a TimeoutError or context.DeadlineExceeded.
func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	// Check for TimeoutError
	var te *TimeoutError
	if errors.As(err, &te) {
		return true
	}

	// Check for context.DeadlineExceeded
	return errors.Is(err, context.DeadlineExceeded)
}

// IsExecutionError checks if the error is or wraps an ExecutionError.
func IsExecutionError(err error) bool {
	if err == nil {
		return false
	}
	var ee *ExecutionError
	return errors.As(err, &ee)
}
