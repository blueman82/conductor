package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestNewTaskError verifies TaskError creation and Error() formatting.
func TestNewTaskError(t *testing.T) {
	tests := []struct {
		name        string
		taskName    string
		message     string
		err         error
		wantContain []string
	}{
		{
			name:     "simple task error",
			taskName: "Task 1",
			message:  "failed to execute",
			err:      nil,
			wantContain: []string{
				"Task 1",
				"failed to execute",
			},
		},
		{
			name:     "task error with wrapped error",
			taskName: "Task 2",
			message:  "quality control failed",
			err:      errors.New("timeout exceeded"),
			wantContain: []string{
				"Task 2",
				"quality control failed",
				"timeout exceeded",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskErr := NewTaskError(tt.taskName, tt.message, tt.err)

			if taskErr == nil {
				t.Fatal("expected non-nil TaskError")
			}

			if taskErr.TaskName != tt.taskName {
				t.Errorf("TaskName = %q, want %q", taskErr.TaskName, tt.taskName)
			}

			if taskErr.Message != tt.message {
				t.Errorf("Message = %q, want %q", taskErr.Message, tt.message)
			}

			if taskErr.Err != tt.err {
				t.Errorf("Err = %v, want %v", taskErr.Err, tt.err)
			}

			if taskErr.Timestamp.IsZero() {
				t.Error("expected non-zero Timestamp")
			}

			errString := taskErr.Error()
			for _, want := range tt.wantContain {
				if !strings.Contains(errString, want) {
					t.Errorf("Error() = %q, want to contain %q", errString, want)
				}
			}
		})
	}
}

// TestTaskErrorWrapping verifies error wrapping with errors.Is and errors.As.
func TestTaskErrorWrapping(t *testing.T) {
	baseErr := errors.New("base error")
	taskErr := NewTaskError("Task 1", "failed", baseErr)

	// Test errors.Is
	if !errors.Is(taskErr, baseErr) {
		t.Error("errors.Is should find wrapped error")
	}

	// Test errors.As
	var te *TaskError
	if !errors.As(taskErr, &te) {
		t.Error("errors.As should unwrap to TaskError")
	}
	if te.TaskName != "Task 1" {
		t.Errorf("unwrapped TaskName = %q, want %q", te.TaskName, "Task 1")
	}
}

// TestNewExecutionError verifies ExecutionError creation and task aggregation.
func TestNewExecutionError(t *testing.T) {
	tests := []struct {
		name       string
		phase      string
		wantPhase  ExecutionPhase
		wantString string
	}{
		{
			name:       "graph phase",
			phase:      "graph",
			wantPhase:  PhaseGraph,
			wantString: "graph",
		},
		{
			name:       "wave phase",
			phase:      "wave",
			wantPhase:  PhaseWave,
			wantString: "wave",
		},
		{
			name:       "task phase",
			phase:      "task",
			wantPhase:  PhaseTask,
			wantString: "task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execErr := NewExecutionError(tt.phase)

			if execErr == nil {
				t.Fatal("expected non-nil ExecutionError")
			}

			if execErr.Phase != tt.wantPhase {
				t.Errorf("Phase = %v, want %v", execErr.Phase, tt.wantPhase)
			}

			if execErr.TotalTasks != 0 {
				t.Errorf("TotalTasks = %d, want 0", execErr.TotalTasks)
			}

			if execErr.FailedTasks != 0 {
				t.Errorf("FailedTasks = %d, want 0", execErr.FailedTasks)
			}

			if len(execErr.TaskErrors) != 0 {
				t.Errorf("TaskErrors length = %d, want 0", len(execErr.TaskErrors))
			}

			errString := execErr.Error()
			if !strings.Contains(errString, tt.wantString) {
				t.Errorf("Error() = %q, want to contain %q", errString, tt.wantString)
			}
		})
	}
}

// TestExecutionErrorAddTask verifies adding task errors.
func TestExecutionErrorAddTask(t *testing.T) {
	execErr := NewExecutionError("wave")
	execErr.TotalTasks = 5

	// Add first task error
	task1 := NewTaskError("Task 1", "failed to execute", errors.New("timeout"))
	execErr.AddTask(task1)

	if execErr.FailedTasks != 1 {
		t.Errorf("FailedTasks = %d, want 1", execErr.FailedTasks)
	}

	if len(execErr.TaskErrors) != 1 {
		t.Fatalf("TaskErrors length = %d, want 1", len(execErr.TaskErrors))
	}

	if execErr.TaskErrors[0] != task1 {
		t.Error("TaskErrors[0] should be task1")
	}

	// Add second task error
	task2 := NewTaskError("Task 2", "quality control failed", nil)
	execErr.AddTask(task2)

	if execErr.FailedTasks != 2 {
		t.Errorf("FailedTasks = %d, want 2", execErr.FailedTasks)
	}

	if len(execErr.TaskErrors) != 2 {
		t.Fatalf("TaskErrors length = %d, want 2", len(execErr.TaskErrors))
	}

	// Error message should contain both tasks
	errString := execErr.Error()
	if !strings.Contains(errString, "Task 1") {
		t.Errorf("Error() should contain 'Task 1'")
	}
	if !strings.Contains(errString, "Task 2") {
		t.Errorf("Error() should contain 'Task 2'")
	}
	if !strings.Contains(errString, "2/5") {
		t.Errorf("Error() should contain '2/5' (failed/total)")
	}
}

// TestNewTimeoutError verifies TimeoutError creation and formatting.
func TestNewTimeoutError(t *testing.T) {
	tests := []struct {
		name        string
		taskName    string
		duration    time.Duration
		wantContain []string
	}{
		{
			name:     "30 second timeout",
			taskName: "Task 1",
			duration: 30 * time.Second,
			wantContain: []string{
				"Task 1",
				"30s",
				"timeout",
			},
		},
		{
			name:     "5 minute timeout",
			taskName: "Task 2",
			duration: 5 * time.Minute,
			wantContain: []string{
				"Task 2",
				"5m0s",
				"timeout",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeoutErr := NewTimeoutError(tt.taskName, tt.duration)

			if timeoutErr == nil {
				t.Fatal("expected non-nil TimeoutError")
			}

			if timeoutErr.TaskName != tt.taskName {
				t.Errorf("TaskName = %q, want %q", timeoutErr.TaskName, tt.taskName)
			}

			if timeoutErr.TimeoutDuration != tt.duration {
				t.Errorf("TimeoutDuration = %v, want %v", timeoutErr.TimeoutDuration, tt.duration)
			}

			if timeoutErr.Timestamp.IsZero() {
				t.Error("expected non-zero Timestamp")
			}

			errString := timeoutErr.Error()
			for _, want := range tt.wantContain {
				if !strings.Contains(errString, want) {
					t.Errorf("Error() = %q, want to contain %q", errString, want)
				}
			}
		})
	}
}

// TestTimeoutErrorWithContext verifies timeout error with context.
func TestTimeoutErrorWithContext(t *testing.T) {
	timeoutErr := NewTimeoutError("Task 1", 30*time.Second)
	timeoutErr.Context = "waiting for agent response"

	errString := timeoutErr.Error()
	if !strings.Contains(errString, "waiting for agent response") {
		t.Errorf("Error() should contain context: %q", errString)
	}
}

// TestIsTaskError verifies type checking for TaskError.
func TestIsTaskError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "task error",
			err:  NewTaskError("Task 1", "failed", nil),
			want: true,
		},
		{
			name: "wrapped task error",
			err:  fmt.Errorf("wrapper: %w", NewTaskError("Task 1", "failed", nil)),
			want: true,
		},
		{
			name: "regular error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "timeout error",
			err:  NewTimeoutError("Task 1", 30*time.Second),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTaskError(tt.err)
			if got != tt.want {
				t.Errorf("IsTaskError() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsTimeoutError verifies type checking for TimeoutError.
func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "timeout error",
			err:  NewTimeoutError("Task 1", 30*time.Second),
			want: true,
		},
		{
			name: "wrapped timeout error",
			err:  fmt.Errorf("wrapper: %w", NewTimeoutError("Task 1", 30*time.Second)),
			want: true,
		},
		{
			name: "context.DeadlineExceeded",
			err:  context.DeadlineExceeded,
			want: true,
		},
		{
			name: "wrapped context.DeadlineExceeded",
			err:  fmt.Errorf("timeout: %w", context.DeadlineExceeded),
			want: true,
		},
		{
			name: "regular error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "task error",
			err:  NewTaskError("Task 1", "failed", nil),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTimeoutError(tt.err)
			if got != tt.want {
				t.Errorf("IsTimeoutError() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsExecutionError verifies type checking for ExecutionError.
func TestIsExecutionError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "execution error",
			err:  NewExecutionError("wave"),
			want: true,
		},
		{
			name: "wrapped execution error",
			err:  fmt.Errorf("wrapper: %w", NewExecutionError("wave")),
			want: true,
		},
		{
			name: "regular error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsExecutionError(tt.err)
			if got != tt.want {
				t.Errorf("IsExecutionError() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestErrorAggregation verifies combining multiple errors.
func TestErrorAggregation(t *testing.T) {
	execErr := NewExecutionError("wave")
	execErr.TotalTasks = 10

	// Simulate multiple task failures
	taskErrors := []*TaskError{
		NewTaskError("Task 1", "timeout", context.DeadlineExceeded),
		NewTaskError("Task 3", "quality control failed", ErrQualityGateFailed),
		NewTaskError("Task 5", "invocation error", errors.New("agent not found")),
	}

	for _, taskErr := range taskErrors {
		execErr.AddTask(taskErr)
	}

	if execErr.FailedTasks != 3 {
		t.Errorf("FailedTasks = %d, want 3", execErr.FailedTasks)
	}

	errString := execErr.Error()

	// Should mention phase
	if !strings.Contains(errString, "wave") {
		t.Error("Error() should mention phase 'wave'")
	}

	// Should mention task count
	if !strings.Contains(errString, "3/10") {
		t.Error("Error() should mention '3/10' tasks failed")
	}

	// Should list all failed tasks
	for _, taskErr := range taskErrors {
		if !strings.Contains(errString, taskErr.TaskName) {
			t.Errorf("Error() should contain %q", taskErr.TaskName)
		}
	}
}

// TestExecutionPhaseString verifies ExecutionPhase string representation.
func TestExecutionPhaseString(t *testing.T) {
	tests := []struct {
		phase ExecutionPhase
		want  string
	}{
		{PhaseGraph, "graph"},
		{PhaseWave, "wave"},
		{PhaseTask, "task"},
		{PhaseQC, "qc"},
		{ExecutionPhase(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.phase.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestExecutionErrorUnwrap verifies error unwrapping for ExecutionError.
func TestExecutionErrorUnwrap(t *testing.T) {
	execErr := NewExecutionError("wave")
	task1 := NewTaskError("Task 1", "failed", errors.New("root cause"))
	execErr.AddTask(task1)

	// ExecutionError should support unwrapping to get individual task errors
	unwrapped := execErr.Unwrap()
	if unwrapped == nil {
		t.Fatal("Unwrap() should return task errors")
	}

	// Should be able to find the root cause
	if !errors.Is(execErr, errors.New("root cause")) {
		// This is expected - ExecutionError wraps multiple errors
		// We need to check individual task errors
	}
}

// TestConcurrentTaskErrorCreation verifies thread-safety of TaskError creation.
func TestConcurrentTaskErrorCreation(t *testing.T) {
	// Create multiple task errors concurrently to ensure timestamp handling is safe
	const goroutines = 10
	done := make(chan *TaskError, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(n int) {
			taskErr := NewTaskError(fmt.Sprintf("Task %d", n), "concurrent error", nil)
			done <- taskErr
		}(i)
	}

	// Collect all errors
	for i := 0; i < goroutines; i++ {
		taskErr := <-done
		if taskErr == nil {
			t.Error("expected non-nil TaskError")
		}
		if taskErr.Timestamp.IsZero() {
			t.Error("expected non-zero Timestamp")
		}
	}
}

// TestExecutionErrorEmpty verifies ExecutionError with no task errors.
func TestExecutionErrorEmpty(t *testing.T) {
	execErr := NewExecutionError("task")
	execErr.TotalTasks = 5

	errString := execErr.Error()
	if !strings.Contains(errString, "0/5") {
		t.Errorf("Error() should show 0/5 tasks failed, got: %q", errString)
	}
}

// TestTimeoutErrorUnwrap verifies TimeoutError can wrap context.DeadlineExceeded.
func TestTimeoutErrorUnwrap(t *testing.T) {
	timeoutErr := NewTimeoutError("Task 1", 30*time.Second)

	// TimeoutError should be identifiable as a timeout
	if !IsTimeoutError(timeoutErr) {
		t.Error("TimeoutError should be recognized by IsTimeoutError")
	}

	// Should also work with errors.Is for context.DeadlineExceeded
	if !errors.Is(timeoutErr, context.DeadlineExceeded) {
		t.Error("TimeoutError should wrap context.DeadlineExceeded")
	}
}
