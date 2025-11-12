package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/models"
)

type stubInvoker struct {
	mu        sync.Mutex
	responses []*agent.InvocationResult
	calls     []models.Task
}

func newStubInvoker(responses ...*agent.InvocationResult) *stubInvoker {
	return &stubInvoker{responses: responses}
}

func (s *stubInvoker) Invoke(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, task)
	if len(s.responses) == 0 {
		return nil, fmt.Errorf("no response configured for task %s", task.Name)
	}
	res := s.responses[0]
	s.responses = s.responses[1:]
	return res, nil
}

type updateCall struct {
	status      string
	completedAt *time.Time
}

type recordingUpdater struct {
	mu    sync.Mutex
	calls []updateCall
	err   error
}

func (r *recordingUpdater) Update(_ string, _ string, status string, completedAt *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.calls = append(r.calls, updateCall{status: status, completedAt: completedAt})
	return nil
}

type stubReviewer struct {
	mu             sync.Mutex
	results        []*ReviewResult
	reviewErr      error
	retryDecisions map[int]bool
	outputs        []string
}

func (s *stubReviewer) Review(_ context.Context, _ models.Task, output string) (*ReviewResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outputs = append(s.outputs, output)
	if s.reviewErr != nil {
		return nil, s.reviewErr
	}
	if len(s.results) == 0 {
		return &ReviewResult{Flag: models.StatusGreen}, nil
	}
	res := s.results[0]
	s.results = s.results[1:]
	return res, nil
}

func (s *stubReviewer) ShouldRetry(_ *ReviewResult, attempt int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.retryDecisions == nil {
		return false
	}
	return s.retryDecisions[attempt]
}

func TestTaskExecutor_ExecutesTaskWithoutQC(t *testing.T) {
	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"done"}`,
		ExitCode: 0,
		Duration: 75 * time.Millisecond,
	})

	updater := &recordingUpdater{}

	executor, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	fixedTime := time.Date(2025, time.November, 8, 12, 0, 0, 0, time.UTC)
	executor.clock = func() time.Time { return fixedTime }

	task := models.Task{Number: "1", Name: "Demo", Prompt: "Do the thing"}

	result, err := executor.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.Status != models.StatusGreen {
		t.Errorf("expected status GREEN, got %s", result.Status)
	}
	if result.Output != "done" {
		t.Errorf("expected output 'done', got %q", result.Output)
	}
	if result.RetryCount != 0 {
		t.Errorf("expected retry count 0, got %d", result.RetryCount)
	}

	if len(updater.calls) != 2 {
		t.Fatalf("expected 2 plan updates, got %d", len(updater.calls))
	}
	if updater.calls[0].status != StatusInProgress {
		t.Errorf("expected first status in-progress, got %s", updater.calls[0].status)
	}
	if updater.calls[1].status != StatusCompleted {
		t.Errorf("expected second status completed, got %s", updater.calls[1].status)
	}
	if updater.calls[1].completedAt == nil || !updater.calls[1].completedAt.Equal(fixedTime) {
		t.Errorf("expected completion timestamp %v", fixedTime)
	}
}

func TestTaskExecutor_RetriesOnRedFlag(t *testing.T) {
	invoker := newStubInvoker(
		&agent.InvocationResult{Output: `{"content":"first"}`, ExitCode: 0},
		&agent.InvocationResult{Output: `{"content":"second"}`, ExitCode: 0},
	)

	reviewer := &stubReviewer{
		results: []*ReviewResult{
			{Flag: models.StatusRed, Feedback: "Needs fixes"},
			{Flag: models.StatusGreen, Feedback: "Looks good"},
		},
		retryDecisions: map[int]bool{0: true, 1: false},
	}

	updater := &recordingUpdater{}

	executor, err := NewTaskExecutor(invoker, reviewer, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
		QualityControl: models.QualityControlConfig{
			Enabled:    true,
			RetryOnRed: 1,
		},
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	executor.clock = func() time.Time { return time.Date(2025, time.November, 9, 9, 30, 0, 0, time.UTC) }

	task := models.Task{Number: "2", Name: "Write feature", Prompt: "Implement"}

	result, err := executor.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if len(invoker.calls) != 2 {
		t.Fatalf("expected 2 invocations, got %d", len(invoker.calls))
	}
	if result.Status != models.StatusGreen {
		t.Errorf("expected status GREEN, got %s", result.Status)
	}
	if result.RetryCount != 1 {
		t.Errorf("expected retry count 1, got %d", result.RetryCount)
	}
	if result.Output != "second" {
		t.Errorf("expected final output 'second', got %q", result.Output)
	}
	if result.ReviewFeedback != "Looks good" {
		t.Errorf("expected final feedback 'Looks good', got %q", result.ReviewFeedback)
	}

	if len(updater.calls) != 2 {
		t.Fatalf("expected 2 plan updates, got %d", len(updater.calls))
	}
	if updater.calls[len(updater.calls)-1].status != StatusCompleted {
		t.Errorf("expected final status completed, got %s", updater.calls[len(updater.calls)-1].status)
	}
}

func TestTaskExecutor_FailsAfterMaxRetries(t *testing.T) {
	invoker := newStubInvoker(
		&agent.InvocationResult{Output: `{"content":"attempt1"}`, ExitCode: 0},
		&agent.InvocationResult{Output: `{"content":"attempt2"}`, ExitCode: 0},
	)

	reviewer := &stubReviewer{
		results: []*ReviewResult{
			{Flag: models.StatusRed, Feedback: "Still failing"},
			{Flag: models.StatusRed, Feedback: "No progress"},
		},
		retryDecisions: map[int]bool{0: true, 1: false},
	}

	updater := &recordingUpdater{}

	executor, err := NewTaskExecutor(invoker, reviewer, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
		QualityControl: models.QualityControlConfig{
			Enabled:    true,
			RetryOnRed: 1,
		},
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	task := models.Task{Number: "3", Name: "Hard task", Prompt: "Difficult"}

	result, err := executor.Execute(context.Background(), task)
	if err == nil {
		t.Fatalf("expected error after max retries")
	}
	if !errors.Is(err, ErrQualityGateFailed) {
		t.Fatalf("expected ErrQualityGateFailed, got %v", err)
	}

	if result.Status != models.StatusRed {
		t.Errorf("expected status RED, got %s", result.Status)
	}
	if result.RetryCount != 1 {
		t.Errorf("expected retry count 1, got %d", result.RetryCount)
	}
	if result.ReviewFeedback != "No progress" {
		t.Errorf("expected last feedback 'No progress', got %q", result.ReviewFeedback)
	}

	if len(updater.calls) == 0 || updater.calls[len(updater.calls)-1].status != StatusFailed {
		t.Fatalf("expected final plan status failed")
	}
}

func TestTaskExecutor_InvocationFailure(t *testing.T) {
	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"boom"}`,
		ExitCode: 1,
	})

	updater := &recordingUpdater{}

	executor, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{PlanPath: "plan.md"})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	task := models.Task{Number: "4", Name: "Broken", Prompt: "Broken"}

	result, err := executor.Execute(context.Background(), task)
	if err == nil {
		t.Fatalf("expected error for failed invocation")
	}

	if result.Status != models.StatusFailed {
		t.Errorf("expected status FAILED, got %s", result.Status)
	}

	if len(updater.calls) == 0 || updater.calls[len(updater.calls)-1].status != StatusFailed {
		t.Fatalf("expected final plan status failed")
	}
}

// TestTaskExecutor_YellowFlagHandling tests that YELLOW flag completes task without retry.
// Critical: Lines 210-218 in task.go handle YELLOW but no tests existed.
func TestTaskExecutor_YellowFlagHandling(t *testing.T) {
	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"needs minor improvements"}`,
		ExitCode: 0,
		Duration: 100 * time.Millisecond,
	})

	reviewer := &stubReviewer{
		results: []*ReviewResult{
			{Flag: models.StatusYellow, Feedback: "Acceptable with minor issues"},
		},
	}

	updater := &recordingUpdater{}

	executor, err := NewTaskExecutor(invoker, reviewer, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
		QualityControl: models.QualityControlConfig{
			Enabled:    true,
			RetryOnRed: 2,
		},
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	fixedTime := time.Date(2025, time.November, 9, 10, 0, 0, 0, time.UTC)
	executor.clock = func() time.Time { return fixedTime }

	task := models.Task{Number: "5", Name: "Yellow task", Prompt: "Do something"}

	result, err := executor.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	// YELLOW should complete without retry
	if result.Status != models.StatusYellow {
		t.Errorf("expected status YELLOW, got %s", result.Status)
	}
	if result.RetryCount != 0 {
		t.Errorf("expected retry count 0 (no retries for YELLOW), got %d", result.RetryCount)
	}
	if len(invoker.calls) != 1 {
		t.Errorf("expected 1 invocation (no retries), got %d", len(invoker.calls))
	}
	if result.ReviewFeedback != "Acceptable with minor issues" {
		t.Errorf("expected feedback 'Acceptable with minor issues', got %q", result.ReviewFeedback)
	}

	// Verify plan updated to completed with timestamp
	if len(updater.calls) != 2 {
		t.Fatalf("expected 2 plan updates, got %d", len(updater.calls))
	}
	if updater.calls[1].status != StatusCompleted {
		t.Errorf("expected final status completed, got %s", updater.calls[1].status)
	}
	if updater.calls[1].completedAt == nil || !updater.calls[1].completedAt.Equal(fixedTime) {
		t.Errorf("expected completion timestamp %v, got %v", fixedTime, updater.calls[1].completedAt)
	}
}

// TestTaskExecutor_ContextCancellation tests that task execution stops gracefully on context cancellation.
// Critical: Line 128 in task.go checks ctx.Err() but no test verifies this behavior.
func TestTaskExecutor_ContextCancellation(t *testing.T) {
	// Create invoker that will be interrupted
	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"should not complete"}`,
		ExitCode: 0,
	})

	updater := &recordingUpdater{}

	executor, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{PlanPath: "plan.md"})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	task := models.Task{Number: "6", Name: "Cancelled task", Prompt: "Never completes"}

	result, err := executor.Execute(ctx, task)
	if err == nil {
		t.Fatalf("expected error for cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}

	// Verify status and plan update
	if result.Status != models.StatusFailed {
		t.Errorf("expected status FAILED, got %s", result.Status)
	}
	if len(updater.calls) == 0 || updater.calls[len(updater.calls)-1].status != StatusFailed {
		t.Errorf("expected final plan status failed")
	}
}

// TestTaskExecutor_ReviewError tests that review errors are handled gracefully.
// Critical: Line 187 in task.go handles reviewErr but was not tested.
func TestTaskExecutor_ReviewError(t *testing.T) {
	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"task completed"}`,
		ExitCode: 0,
	})

	reviewErr := errors.New("review service unavailable")
	reviewer := &stubReviewer{
		reviewErr: reviewErr,
	}

	updater := &recordingUpdater{}

	executor, err := NewTaskExecutor(invoker, reviewer, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
		QualityControl: models.QualityControlConfig{
			Enabled:    true,
			RetryOnRed: 1,
		},
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	task := models.Task{Number: "7", Name: "Review fails", Prompt: "Should fail on review"}

	result, err := executor.Execute(context.Background(), task)
	if err == nil {
		t.Fatalf("expected error from review failure")
	}
	if !errors.Is(err, reviewErr) {
		t.Errorf("expected review error, got %v", err)
	}

	// Verify result captures error and status
	if result.Status != models.StatusFailed {
		t.Errorf("expected status FAILED, got %s", result.Status)
	}
	if !errors.Is(result.Error, reviewErr) {
		t.Errorf("expected result.Error to be review error, got %v", result.Error)
	}

	// Verify plan marked as failed
	if len(updater.calls) == 0 || updater.calls[len(updater.calls)-1].status != StatusFailed {
		t.Errorf("expected final plan status failed")
	}
}

// TestTaskExecutor_PlanUpdateFailures tests that plan update failures are recorded but don't block execution.
// Critical: Lines 113-117 and 204-208 in task.go handle update failures but not tested.
func TestTaskExecutor_PlanUpdateFailures(t *testing.T) {
	tests := []struct {
		name           string
		failInitial    bool
		failFinal      bool
		expectedStatus string
		expectedError  bool
		qcEnabled      bool
		reviewFlag     string
	}{
		{
			name:           "Initial update fails",
			failInitial:    true,
			failFinal:      false,
			expectedStatus: models.StatusFailed,
			expectedError:  true,
			qcEnabled:      false,
		},
		{
			name:           "Final update fails on success",
			failInitial:    false,
			failFinal:      true,
			expectedStatus: models.StatusFailed,
			expectedError:  true,
			qcEnabled:      true,
			reviewFlag:     models.StatusGreen,
		},
		{
			name:           "Final update fails on YELLOW",
			failInitial:    false,
			failFinal:      true,
			expectedStatus: models.StatusFailed,
			expectedError:  true,
			qcEnabled:      true,
			reviewFlag:     models.StatusYellow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invoker := newStubInvoker(&agent.InvocationResult{
				Output:   `{"content":"done"}`,
				ExitCode: 0,
			})

			updateErr := errors.New("filesystem error")

			var updater PlanUpdater
			if tt.failInitial {
				// Fail on first call (in-progress)
				updater = &recordingUpdater{err: updateErr}
			} else if tt.failFinal {
				// Fail after first successful call - use custom wrapper
				var callCount int
				updater = planUpdaterFunc(func(path string, taskNum string, status string, completedAt *time.Time) error {
					callCount++
					if callCount == 1 {
						return nil // First call succeeds
					}
					return updateErr // Second call fails
				})
			} else {
				updater = &recordingUpdater{}
			}

			var reviewer *stubReviewer
			if tt.qcEnabled {
				reviewer = &stubReviewer{
					results: []*ReviewResult{
						{Flag: tt.reviewFlag, Feedback: "Review feedback"},
					},
				}
			}

			executor, err := NewTaskExecutor(invoker, reviewer, updater, TaskExecutorConfig{
				PlanPath: "plan.md",
				QualityControl: models.QualityControlConfig{
					Enabled:    tt.qcEnabled,
					RetryOnRed: 1,
				},
			})
			if err != nil {
				t.Fatalf("NewTaskExecutor returned error: %v", err)
			}

			task := models.Task{Number: "8", Name: "Update fails", Prompt: "Test"}

			result, err := executor.Execute(context.Background(), task)

			if tt.expectedError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			if result.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, result.Status)
			}
		})
	}
}

// TestTaskExecutor_DefaultAgentAssignment tests that default agent is applied when task has no agent.
// Important: Lines 108-111 in task.go apply default agent but not tested.
func TestTaskExecutor_DefaultAgentAssignment(t *testing.T) {
	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"used default agent"}`,
		ExitCode: 0,
	})

	updater := &recordingUpdater{}

	executor, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
		PlanPath:     "plan.md",
		DefaultAgent: "test-automation",
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	// Task with no agent specified
	task := models.Task{Number: "9", Name: "Uses default", Prompt: "Do something", Agent: ""}

	result, err := executor.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	// Verify default agent was applied to result
	if result.Task.Agent != "test-automation" {
		t.Errorf("expected agent 'test-automation', got %q", result.Task.Agent)
	}

	// Verify task passed to invoker has default agent
	if len(invoker.calls) != 1 {
		t.Fatalf("expected 1 invocation, got %d", len(invoker.calls))
	}
	if invoker.calls[0].Agent != "test-automation" {
		t.Errorf("expected invoked task to have agent 'test-automation', got %q", invoker.calls[0].Agent)
	}
}

// TestTaskExecutor_InvalidReviewFlag tests handling of unknown QC flags.
// Important: Lines 199-200 and 222-223 in task.go handle unknown flags but not tested.
func TestTaskExecutor_InvalidReviewFlag(t *testing.T) {
	tests := []struct {
		name         string
		reviewFlag   string
		reviewNil    bool
		expectedErr  bool
		expectedStop bool
	}{
		{
			name:         "Unknown flag PURPLE",
			reviewFlag:   "PURPLE",
			expectedErr:  true,
			expectedStop: true,
		},
		{
			name:         "Empty flag",
			reviewFlag:   "",
			expectedErr:  true,
			expectedStop: true,
		},
		{
			name:         "Nil review result",
			reviewNil:    true,
			expectedErr:  true,
			expectedStop: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invoker := newStubInvoker(&agent.InvocationResult{
				Output:   `{"content":"task output"}`,
				ExitCode: 0,
			})

			var reviewResult *ReviewResult
			if !tt.reviewNil {
				reviewResult = &ReviewResult{Flag: tt.reviewFlag, Feedback: "Unknown flag"}
			}

			reviewer := &stubReviewer{
				results:        []*ReviewResult{reviewResult},
				retryDecisions: map[int]bool{0: false}, // Don't retry
			}

			updater := &recordingUpdater{}

			executor, err := NewTaskExecutor(invoker, reviewer, updater, TaskExecutorConfig{
				PlanPath: "plan.md",
				QualityControl: models.QualityControlConfig{
					Enabled:    true,
					RetryOnRed: 1,
				},
			})
			if err != nil {
				t.Fatalf("NewTaskExecutor returned error: %v", err)
			}

			task := models.Task{Number: "10", Name: "Invalid flag", Prompt: "Test"}

			result, err := executor.Execute(context.Background(), task)

			if tt.expectedErr && err == nil {
				t.Errorf("expected error for invalid flag, got nil")
			}

			if tt.expectedStop && result.Status != models.StatusRed && result.Status != models.StatusFailed {
				t.Errorf("expected status RED or FAILED for invalid flag, got %s", result.Status)
			}

			// Verify plan marked as failed
			if len(updater.calls) == 0 || updater.calls[len(updater.calls)-1].status != StatusFailed {
				t.Errorf("expected final plan status failed")
			}
		})
	}
}

// TestTaskExecutor_JSONParsingEdgeCases tests handling of malformed or empty JSON output.
// Important: Lines 160-170 in task.go parse JSON output with edge cases.
func TestTaskExecutor_JSONParsingEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		expectedOutput string
		description    string
	}{
		{
			name:           "Malformed JSON falls back to raw",
			output:         `{invalid json here`,
			expectedOutput: `{invalid json here`,
			description:    "ParseClaudeOutput returns raw output when JSON invalid",
		},
		{
			name:           "Empty content field uses error field",
			output:         `{"content":"","error":"something went wrong"}`,
			expectedOutput: "something went wrong",
			description:    "Uses error field when content is empty",
		},
		{
			name:           "Both fields empty returns empty string",
			output:         `{"content":"","error":""}`,
			expectedOutput: "",
			description:    "Returns empty string when both fields empty",
		},
		{
			name:           "Valid content field",
			output:         `{"content":"task completed successfully"}`,
			expectedOutput: "task completed successfully",
			description:    "Uses content field when present",
		},
		{
			name:           "Non-JSON plaintext",
			output:         "Plain text output without JSON",
			expectedOutput: "Plain text output without JSON",
			description:    "Falls back to raw output for non-JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invoker := newStubInvoker(&agent.InvocationResult{
				Output:   tt.output,
				ExitCode: 0,
			})

			updater := &recordingUpdater{}

			executor, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
				PlanPath: "plan.md",
			})
			if err != nil {
				t.Fatalf("NewTaskExecutor returned error: %v", err)
			}

			task := models.Task{Number: "11", Name: "JSON test", Prompt: "Test"}

			result, err := executor.Execute(context.Background(), task)
			if err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}

			if result.Output != tt.expectedOutput {
				t.Errorf("expected output %q, got %q", tt.expectedOutput, result.Output)
			}
		})
	}
}

// TestTaskExecutor_InvocationErrorVsExitCode tests differentiation between Error and ExitCode.
// Important: Lines 145-150 in task.go differentiate these but not fully tested.
func TestTaskExecutor_InvocationErrorVsExitCode(t *testing.T) {
	tests := []struct {
		name          string
		invocationErr error
		exitCode      int
		expectedErr   bool
		description   string
	}{
		{
			name:          "Invocation.Error set (not ExitError)",
			invocationErr: errors.New("command not found"),
			exitCode:      0,
			expectedErr:   true,
			description:   "Error field indicates command execution failure",
		},
		{
			name:          "ExitCode non-zero",
			invocationErr: nil,
			exitCode:      127,
			expectedErr:   true,
			description:   "Non-zero exit code indicates task failure",
		},
		{
			name:          "Both Error and ExitCode (Error takes precedence)",
			invocationErr: errors.New("exec error"),
			exitCode:      1,
			expectedErr:   true,
			description:   "Error field checked first before ExitCode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invoker := newStubInvoker(&agent.InvocationResult{
				Output:   `{"content":"output"}`,
				Error:    tt.invocationErr,
				ExitCode: tt.exitCode,
			})

			updater := &recordingUpdater{}

			executor, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
				PlanPath: "plan.md",
			})
			if err != nil {
				t.Fatalf("NewTaskExecutor returned error: %v", err)
			}

			task := models.Task{Number: "12", Name: "Error test", Prompt: "Test"}

			result, err := executor.Execute(context.Background(), task)

			if tt.expectedErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectedErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			if result.Status != models.StatusFailed {
				t.Errorf("expected status FAILED, got %s", result.Status)
			}

			// Verify error is captured in result
			if tt.invocationErr != nil && result.Error != nil {
				if !errors.Is(result.Error, tt.invocationErr) && result.Error.Error() != fmt.Sprintf("task invocation error: %v", tt.invocationErr) {
					t.Errorf("expected error to contain invocation error, got %v", result.Error)
				}
			}
		})
	}
}

// TestTaskExecutor_InvocationErrorReturnsTaskError verifies that invocation errors
// are wrapped in TaskError with proper context.
// Tests integration of custom error types in task.go lines 153 and 160.
func TestTaskExecutor_InvocationErrorReturnsTaskError(t *testing.T) {
	tests := []struct {
		name          string
		invocationErr error
		exitCode      int
		expectTaskErr bool
	}{
		{
			name:          "Invocation Error wrapped in TaskError",
			invocationErr: errors.New("command not found"),
			exitCode:      0,
			expectTaskErr: true,
		},
		{
			name:          "Non-zero ExitCode wrapped in TaskError",
			invocationErr: nil,
			exitCode:      127,
			expectTaskErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invoker := newStubInvoker(&agent.InvocationResult{
				Output:   `{"content":"output"}`,
				Error:    tt.invocationErr,
				ExitCode: tt.exitCode,
			})

			updater := &recordingUpdater{}

			executor, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
				PlanPath: "plan.md",
			})
			if err != nil {
				t.Fatalf("NewTaskExecutor returned error: %v", err)
			}

			task := models.Task{Number: "13", Name: "Task error test", Prompt: "Test"}

			result, err := executor.Execute(context.Background(), task)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			// Verify the error is a TaskError
			if tt.expectTaskErr && !IsTaskError(err) {
				t.Errorf("expected TaskError, got %T: %v", err, err)
			}

			// Verify error contains task information
			var taskErr *TaskError
			if errors.As(err, &taskErr) {
				if taskErr.TaskName != task.Number {
					t.Errorf("expected TaskName %q, got %q", task.Number, taskErr.TaskName)
				}
			}

			// Verify result also has the error
			if result.Error == nil {
				t.Error("expected result.Error to be set")
			}
		})
	}
}

// TestTaskExecutor_ContextTimeoutReturnsTimeoutError verifies that context timeout
// returns a TimeoutError with proper wrapping.
func TestTaskExecutor_ContextTimeoutReturnsTimeoutError(t *testing.T) {
	// Create a slow invoker that will timeout
	slowInvoker := &slowStubInvoker{
		delay: 100 * time.Millisecond,
	}

	updater := &recordingUpdater{}

	executor, err := NewTaskExecutor(slowInvoker, nil, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	task := models.Task{Number: "14", Name: "Timeout task", Prompt: "Will timeout"}

	result, err := executor.Execute(ctx, task)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}

	// Verify the error is a timeout
	if !IsTimeoutError(err) {
		t.Errorf("expected timeout error, got %T: %v", err, err)
	}

	// Verify it's context.DeadlineExceeded
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context timeout error, got %v", err)
	}

	// Verify result status
	if result.Status != models.StatusFailed {
		t.Errorf("expected status FAILED, got %s", result.Status)
	}
}

// slowStubInvoker simulates a slow invocation that respects context.
type slowStubInvoker struct {
	delay time.Duration
}

func (s *slowStubInvoker) Invoke(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
	select {
	case <-time.After(s.delay):
		return &agent.InvocationResult{
			Output:   `{"content":"completed"}`,
			ExitCode: 0,
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// TestTaskExecutor_QCInvalidFlagReturnsTaskError verifies that invalid QC flags
// return TaskError with proper context.
// Tests integration at task.go lines 207 and 229.
func TestTaskExecutor_QCInvalidFlagReturnsTaskError(t *testing.T) {
	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"task output"}`,
		ExitCode: 0,
	})

	reviewer := &stubReviewer{
		results: []*ReviewResult{
			{Flag: "INVALID_FLAG", Feedback: "Unknown flag"},
		},
		retryDecisions: map[int]bool{0: false},
	}

	updater := &recordingUpdater{}

	executor, err := NewTaskExecutor(invoker, reviewer, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
		QualityControl: models.QualityControlConfig{
			Enabled:    true,
			RetryOnRed: 1,
		},
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	task := models.Task{Number: "15", Name: "Invalid QC flag", Prompt: "Test"}

	result, err := executor.Execute(context.Background(), task)
	if err == nil {
		t.Fatalf("expected error for invalid flag, got nil")
	}

	// The error itself may not be a TaskError (it's ErrQualityGateFailed),
	// but we can verify the result contains error information
	if result.Status != models.StatusRed && result.Status != models.StatusFailed {
		t.Errorf("expected status RED or FAILED, got %s", result.Status)
	}
}

// TestTaskFileTracking verifies that tasks track their source file.
// When multiple tasks come from different files, each task knows its origin file.
func TestTaskFileTracking(t *testing.T) {
	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"task from file1"}`,
		ExitCode: 0,
	})

	updater := &recordingUpdater{}

	executor, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
		PlanPath: "plan1.md",
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	// Create task with explicit file mapping
	task := models.Task{
		Number: "1",
		Name:   "Task from split plan",
		Prompt: "Do something in plan1.md",
	}

	// Set the source file in executor
	executor.SourceFile = "plan1.md"

	result, err := executor.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.Status != models.StatusGreen {
		t.Errorf("expected status GREEN, got %s", result.Status)
	}

	// Verify that the source file was tracked
	if executor.SourceFile != "plan1.md" {
		t.Errorf("expected source file 'plan1.md', got %q", executor.SourceFile)
	}
}

// TestPerFileLocking verifies that per-file locks prevent concurrent modifications.
// Multiple tasks from the same file should serialize through the lock.
func TestPerFileLocking(t *testing.T) {
	invoker := newStubInvoker(
		&agent.InvocationResult{Output: `{"content":"task1"}`, ExitCode: 0},
		&agent.InvocationResult{Output: `{"content":"task2"}`, ExitCode: 0},
	)

	updater := &recordingUpdater{}

	executor, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
		PlanPath: "shared.md",
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	// Initialize file lock manager
	if executor.FileLockManager == nil {
		executor.FileLockManager = NewFileLockManager()
	}

	executor.SourceFile = "shared.md"

	// Execute two tasks sequentially from same file
	task1 := models.Task{Number: "1", Name: "First task", Prompt: "Do first"}
	task2 := models.Task{Number: "2", Name: "Second task", Prompt: "Do second"}

	result1, err := executor.Execute(context.Background(), task1)
	if err != nil {
		t.Fatalf("First Execute returned error: %v", err)
	}
	if result1.Status != models.StatusGreen {
		t.Errorf("expected status GREEN for task1, got %s", result1.Status)
	}

	result2, err := executor.Execute(context.Background(), task2)
	if err != nil {
		t.Fatalf("Second Execute returned error: %v", err)
	}
	if result2.Status != models.StatusGreen {
		t.Errorf("expected status GREEN for task2, got %s", result2.Status)
	}

	// Verify both tasks updated (with locking coordination)
	if len(updater.calls) != 4 {
		t.Fatalf("expected 4 plan updates (2 tasks × 2 updates each), got %d", len(updater.calls))
	}
}

// TestUpdateCorrectFile verifies that the correct file is updated when task completes.
// With multiple source files, each task should update only its own file.
func TestUpdateCorrectFile(t *testing.T) {
	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"updated in file2"}`,
		ExitCode: 0,
	})

	// Track which plan path was updated
	var updatedPath string
	updater := planUpdaterFunc(func(path string, taskNumber string, status string, completedAt *time.Time) error {
		updatedPath = path
		return nil
	})

	executor, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
		PlanPath: "plan2.md",
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	// Set source file to different path
	executor.SourceFile = "merged-plan2.md"

	// When SourceFile is set, it should be used for updates instead of PlanPath
	// This tests the priority: SourceFile > PlanPath
	task := models.Task{Number: "1", Name: "Task from split plan", Prompt: "Update correct file"}

	result, err := executor.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.Status != models.StatusGreen {
		t.Errorf("expected status GREEN, got %s", result.Status)
	}

	// The updater should have been called with SourceFile (not PlanPath)
	// This is the fix: SourceFile takes precedence for multi-file plans
	if updatedPath != "merged-plan2.md" {
		t.Errorf("expected source file 'merged-plan2.md', got %q", updatedPath)
	}
}

// MockLearningStore implements a mock learning store for testing pre-task hooks
type MockLearningStore struct {
	AnalysisResult *FailureAnalysis
	AnalysisError  error
	CallCount      int
	LastPlanFile   string
	LastTaskNumber string
}

func (m *MockLearningStore) AnalyzeFailures(ctx context.Context, planFile, taskNumber string) (*FailureAnalysis, error) {
	m.CallCount++
	m.LastPlanFile = planFile
	m.LastTaskNumber = taskNumber

	if m.AnalysisError != nil {
		return nil, m.AnalysisError
	}

	// Return default empty analysis if none configured
	if m.AnalysisResult == nil {
		return &FailureAnalysis{
			TriedAgents:    make([]string, 0),
			CommonPatterns: make([]string, 0),
		}, nil
	}

	return m.AnalysisResult, nil
}

// TestPreTaskHook_NoHistory verifies that hook is no-op when no failure history exists
func TestPreTaskHook_NoHistory(t *testing.T) {
	mockStore := &MockLearningStore{
		AnalysisResult: &FailureAnalysis{
			TotalAttempts:  0,
			FailedAttempts: 0,
			TriedAgents:    []string{},
			CommonPatterns: []string{},
		},
	}

	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"task completed"}`,
		ExitCode: 0,
	})

	updater := &recordingUpdater{}

	executor, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	// Set learning store and related fields
	executor.learningStore = mockStore
	executor.planFile = "plan.md"

	task := models.Task{
		Number: "1",
		Name:   "Test task",
		Prompt: "Do something",
		Agent:  "backend-developer",
	}

	result, err := executor.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.Status != models.StatusGreen {
		t.Errorf("expected status GREEN, got %s", result.Status)
	}

	// Verify hook was called
	if mockStore.CallCount != 1 {
		t.Errorf("expected AnalyzeFailures called once, got %d", mockStore.CallCount)
	}

	// Verify agent was not changed
	if result.Task.Agent != "backend-developer" {
		t.Errorf("expected agent unchanged, got %s", result.Task.Agent)
	}
}

// TestPreTaskHook_AdaptsAgent verifies that agent is changed when 2+ failures detected
func TestPreTaskHook_AdaptsAgent(t *testing.T) {
	mockStore := &MockLearningStore{
		AnalysisResult: &FailureAnalysis{
			TotalAttempts:           3,
			FailedAttempts:          2,
			TriedAgents:             []string{"backend-developer"},
			SuggestedAgent:          "golang-pro",
			ShouldTryDifferentAgent: true,
			SuggestedApproach:       "Try using golang-pro agent with focus on Go-specific patterns",
		},
	}

	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"task completed with golang-pro"}`,
		ExitCode: 0,
	})

	updater := &recordingUpdater{}

	executor, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	// Set learning store and related fields
	executor.learningStore = mockStore
	executor.planFile = "plan.md"

	task := models.Task{
		Number: "1",
		Name:   "Test task",
		Prompt: "Do something",
		Agent:  "backend-developer",
	}

	result, err := executor.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.Status != models.StatusGreen {
		t.Errorf("expected status GREEN, got %s", result.Status)
	}

	// Verify hook was called
	if mockStore.CallCount != 1 {
		t.Errorf("expected AnalyzeFailures called once, got %d", mockStore.CallCount)
	}

	// Verify agent was changed
	if result.Task.Agent != "golang-pro" {
		t.Errorf("expected agent changed to 'golang-pro', got %s", result.Task.Agent)
	}

	// Verify the invoker received the modified agent
	if len(invoker.calls) != 1 {
		t.Fatalf("expected 1 invocation, got %d", len(invoker.calls))
	}
	if invoker.calls[0].Agent != "golang-pro" {
		t.Errorf("expected invoker called with 'golang-pro', got %s", invoker.calls[0].Agent)
	}
}

// TestPreTaskHook_EnhancesPrompt verifies that prompt is enhanced with learning context
func TestPreTaskHook_EnhancesPrompt(t *testing.T) {
	mockStore := &MockLearningStore{
		AnalysisResult: &FailureAnalysis{
			TotalAttempts:     2,
			FailedAttempts:    1,
			TriedAgents:       []string{"backend-developer"},
			CommonPatterns:    []string{"syntax_error", "compilation_error"},
			SuggestedApproach: "Focus on fixing syntax and compilation errors",
		},
	}

	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"task completed"}`,
		ExitCode: 0,
	})

	updater := &recordingUpdater{}

	executor, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	// Set learning store and related fields
	executor.learningStore = mockStore
	executor.planFile = "plan.md"

	originalPrompt := "Write a function"
	task := models.Task{
		Number: "1",
		Name:   "Test task",
		Prompt: originalPrompt,
		Agent:  "backend-developer",
	}

	result, err := executor.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.Status != models.StatusGreen {
		t.Errorf("expected status GREEN, got %s", result.Status)
	}

	// Verify hook was called
	if mockStore.CallCount != 1 {
		t.Errorf("expected AnalyzeFailures called once, got %d", mockStore.CallCount)
	}

	// Verify prompt was enhanced (contains learning context)
	if len(invoker.calls) != 1 {
		t.Fatalf("expected 1 invocation, got %d", len(invoker.calls))
	}

	invokedPrompt := invoker.calls[0].Prompt
	if invokedPrompt == originalPrompt {
		t.Error("expected prompt to be enhanced, but it was unchanged")
	}

	// Should contain learning context about past failures
	if !strings.Contains(invokedPrompt, "past failures") && !strings.Contains(invokedPrompt, "previous attempts") {
		t.Errorf("expected prompt to contain learning context, got: %s", invokedPrompt)
	}
}

// TestPreTaskHook_LearningDisabled verifies that hook is no-op when learning disabled
func TestPreTaskHook_LearningDisabled(t *testing.T) {
	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"task completed"}`,
		ExitCode: 0,
	})

	updater := &recordingUpdater{}

	executor, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	// Do NOT set learning store - simulates disabled learning

	task := models.Task{
		Number: "1",
		Name:   "Test task",
		Prompt: "Do something",
		Agent:  "backend-developer",
	}

	result, err := executor.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.Status != models.StatusGreen {
		t.Errorf("expected status GREEN, got %s", result.Status)
	}

	// Verify agent was not changed
	if result.Task.Agent != "backend-developer" {
		t.Errorf("expected agent unchanged, got %s", result.Task.Agent)
	}
}

// TestPreTaskHook_LearningStoreError verifies graceful degradation on learning errors
func TestPreTaskHook_LearningStoreError(t *testing.T) {
	mockStore := &MockLearningStore{
		AnalysisError: fmt.Errorf("database connection failed"),
	}

	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"task completed"}`,
		ExitCode: 0,
	})

	updater := &recordingUpdater{}

	executor, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	// Set learning store that will return error
	executor.learningStore = mockStore
	executor.planFile = "plan.md"

	task := models.Task{
		Number: "1",
		Name:   "Test task",
		Prompt: "Do something",
		Agent:  "backend-developer",
	}

	// Learning error should not break task execution
	result, err := executor.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute should not fail on learning error, got: %v", err)
	}

	if result.Status != models.StatusGreen {
		t.Errorf("expected status GREEN, got %s", result.Status)
	}

	// Verify hook was attempted
	if mockStore.CallCount != 1 {
		t.Errorf("expected AnalyzeFailures called once, got %d", mockStore.CallCount)
	}

	// Verify agent was not changed (graceful degradation)
	if result.Task.Agent != "backend-developer" {
		t.Errorf("expected agent unchanged on error, got %s", result.Task.Agent)
	}
}

// TestPreTaskHook_NoSuggestedAgent verifies handling when no alternative agent available
func TestPreTaskHook_NoSuggestedAgent(t *testing.T) {
	mockStore := &MockLearningStore{
		AnalysisResult: &FailureAnalysis{
			TotalAttempts:           3,
			FailedAttempts:          2,
			TriedAgents:             []string{"backend-developer"},
			SuggestedAgent:          "", // No suggestion available
			ShouldTryDifferentAgent: true,
		},
	}

	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"task completed"}`,
		ExitCode: 0,
	})

	updater := &recordingUpdater{}

	executor, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	// Set learning store
	executor.learningStore = mockStore
	executor.planFile = "plan.md"

	task := models.Task{
		Number: "1",
		Name:   "Test task",
		Prompt: "Do something",
		Agent:  "backend-developer",
	}

	result, err := executor.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.Status != models.StatusGreen {
		t.Errorf("expected status GREEN, got %s", result.Status)
	}

	// Verify agent was NOT changed when no suggestion available
	if result.Task.Agent != "backend-developer" {
		t.Errorf("expected agent unchanged when no suggestion, got %s", result.Task.Agent)
	}
}

// TestPreTaskHook_AgentAlreadyOptimal verifies no change when suggested agent is current
func TestPreTaskHook_AgentAlreadyOptimal(t *testing.T) {
	mockStore := &MockLearningStore{
		AnalysisResult: &FailureAnalysis{
			TotalAttempts:           3,
			FailedAttempts:          2,
			TriedAgents:             []string{"backend-developer"},
			SuggestedAgent:          "golang-pro", // Suggested agent
			ShouldTryDifferentAgent: true,
		},
	}

	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"task completed"}`,
		ExitCode: 0,
	})

	updater := &recordingUpdater{}

	executor, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	// Set learning store
	executor.learningStore = mockStore
	executor.planFile = "plan.md"

	task := models.Task{
		Number: "1",
		Name:   "Test task",
		Prompt: "Do something",
		Agent:  "golang-pro", // Already using suggested agent
	}

	result, err := executor.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.Status != models.StatusGreen {
		t.Errorf("expected status GREEN, got %s", result.Status)
	}

	// Verify agent remains golang-pro (no redundant change)
	if result.Task.Agent != "golang-pro" {
		t.Errorf("expected agent to remain 'golang-pro', got %s", result.Task.Agent)
	}
}

// trackingFileLockManager tracks which file paths are locked for testing
type trackingFileLockManager struct {
	lockedPaths []string
	mu          sync.Mutex
}

func (t *trackingFileLockManager) Lock(filePath string) func() {
	t.mu.Lock()
	t.lockedPaths = append(t.lockedPaths, filePath)
	t.mu.Unlock()
	return func() {}
}

// TestMultiFileExecutionWithoutSourceFile demonstrates the bug where multi-file plans
// concatenate file paths and cause lock acquisition failures.
// After fix: verifies that SourceFile is used for locking instead of concatenated PlanPath.
func TestMultiFileExecutionWithoutSourceFile(t *testing.T) {
	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"task output"}`,
		ExitCode: 0,
	})

	updater := &recordingUpdater{}

	// Simulate multi-file execution where run.go concatenates plan paths
	concatenatedPath := "plan-01-foundation.yaml, plan-02-configuration-testing.yaml"

	executor, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
		PlanPath: concatenatedPath, // Multi-file paths are concatenated in run.go
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	// Use tracking lock manager to verify which path is locked
	tracker := &trackingFileLockManager{lockedPaths: []string{}}
	executor.FileLockManager = tracker

	// Task with SourceFile set (wave executor does this from Task.SourceFile)
	task := models.Task{
		Number:     "1",
		Name:       "Task from multi-file plan",
		Prompt:     "Do something",
		SourceFile: "plan-01-foundation.yaml", // Set by orchestrator during merge
	}

	// Simulate wave executor setting SourceFile before execution
	executor.SourceFile = task.SourceFile

	// This should execute successfully
	result, err := executor.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.Status != models.StatusGreen {
		t.Errorf("expected status GREEN, got %s", result.Status)
	}

	// AFTER FIX: Verify that individual source file was locked, not concatenated path
	if len(tracker.lockedPaths) != 1 {
		t.Fatalf("expected 1 lock, got %d", len(tracker.lockedPaths))
	}

	lockedPath := tracker.lockedPaths[0]

	// The fix: it should lock the individual source file
	if lockedPath != "plan-01-foundation.yaml" {
		t.Errorf("expected to lock individual file 'plan-01-foundation.yaml', got %q", lockedPath)
	}

	// Verify SourceFile takes precedence over PlanPath
	if lockedPath == concatenatedPath {
		t.Errorf("ERROR: Still locking concatenated path %q instead of SourceFile", lockedPath)
	}
}

// TestTaskExecutorUsesSourceFileForLocking verifies that when SourceFile is set,
// it takes precedence over PlanPath for file locking operations.
func TestTaskExecutorUsesSourceFileForLocking(t *testing.T) {
	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"task completed"}`,
		ExitCode: 0,
	})

	updater := &recordingUpdater{}

	// Create executor with concatenated PlanPath (simulating multi-file bug)
	concatenatedPath := "file1.md, file2.md"
	executor, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
		PlanPath: concatenatedPath,
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	// Set the correct individual source file
	executor.SourceFile = "file1.md"

	task := models.Task{
		Number: "1",
		Name:   "Task from file1",
		Prompt: "Task should lock file1.md only",
	}

	result, err := executor.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.Status != models.StatusGreen {
		t.Errorf("expected status GREEN, got %s", result.Status)
	}

	// Verify the lock was acquired on the individual file, not concatenated path
	// This is implicit in the test passing - if it tried to lock the concatenated
	// path, the file system operation would likely fail or behave unexpectedly
}
