package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/models"
)

// Mock invoker for testing
type mockInvoker struct {
	mockInvoke func(ctx context.Context, task models.Task) (*agent.InvocationResult, error)
}

func (m *mockInvoker) Invoke(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
	if m.mockInvoke != nil {
		return m.mockInvoke(ctx, task)
	}
	return &agent.InvocationResult{
		Output:   "Quality Control: GREEN\n\nAll tests pass",
		ExitCode: 0,
		Duration: 100 * time.Millisecond,
	}, nil
}

func TestBuildReviewPrompt(t *testing.T) {
	tests := []struct {
		name         string
		task         models.Task
		output       string
		wantContains []string
	}{
		{
			name: "basic task review",
			task: models.Task{
				Number: 1,
				Name:   "Implement feature X",
				Prompt: "Add feature X to the system",
			},
			output: "Feature X implemented successfully",
			wantContains: []string{
				"Implement feature X",
				"Feature X implemented successfully",
				"Quality Control:",
				"GREEN/RED/YELLOW",
			},
		},
		{
			name: "task with complex output",
			task: models.Task{
				Number: 2,
				Name:   "Fix bug Y",
				Prompt: "Fix the Y bug",
			},
			output: "Bug Y fixed\nTests pass\nCoverage: 85%",
			wantContains: []string{
				"Fix bug Y",
				"Bug Y fixed",
				"Tests pass",
				"Coverage: 85%",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qc := NewQualityController(nil)
			prompt := qc.BuildReviewPrompt(tt.task, tt.output)

			for _, want := range tt.wantContains {
				if !contains(prompt, want) {
					t.Errorf("BuildReviewPrompt() missing expected content %q\nGot: %s", want, prompt)
				}
			}
		})
	}
}

func TestParseReviewResponse(t *testing.T) {
	tests := []struct {
		name         string
		output       string
		wantFlag     string
		wantFeedback string
	}{
		{
			name:         "GREEN response",
			output:       "Quality Control: GREEN\n\nAll tests pass",
			wantFlag:     "GREEN",
			wantFeedback: "All tests pass",
		},
		{
			name:         "RED response",
			output:       "Quality Control: RED\n\nTests are failing",
			wantFlag:     "RED",
			wantFeedback: "Tests are failing",
		},
		{
			name:         "YELLOW response",
			output:       "Quality Control: YELLOW\n\nMinor issues detected",
			wantFlag:     "YELLOW",
			wantFeedback: "Minor issues detected",
		},
		{
			name:         "GREEN with extra whitespace",
			output:       "Quality Control:   GREEN  \n\n  Looks good!  ",
			wantFlag:     "GREEN",
			wantFeedback: "Looks good!",
		},
		{
			name:         "multiline feedback",
			output:       "Quality Control: RED\n\nIssue 1: Tests failing\nIssue 2: Coverage low\nIssue 3: Linting errors",
			wantFlag:     "RED",
			wantFeedback: "Issue 1: Tests failing\nIssue 2: Coverage low\nIssue 3: Linting errors",
		},
		{
			name:         "no flag found",
			output:       "Some random output",
			wantFlag:     "",
			wantFeedback: "",
		},
		{
			name:         "flag without feedback",
			output:       "Quality Control: GREEN",
			wantFlag:     "GREEN",
			wantFeedback: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag, feedback := ParseReviewResponse(tt.output)
			if flag != tt.wantFlag {
				t.Errorf("ParseReviewResponse() flag = %q, want %q", flag, tt.wantFlag)
			}
			if feedback != tt.wantFeedback {
				t.Errorf("ParseReviewResponse() feedback = %q, want %q", feedback, tt.wantFeedback)
			}
		})
	}
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name           string
		result         *ReviewResult
		currentAttempt int
		maxRetries     int
		wantRetry      bool
	}{
		{
			name: "RED flag under max retries",
			result: &ReviewResult{
				Flag:     "RED",
				Feedback: "Needs fixes",
			},
			currentAttempt: 0,
			maxRetries:     2,
			wantRetry:      true,
		},
		{
			name: "RED flag at max retries",
			result: &ReviewResult{
				Flag:     "RED",
				Feedback: "Needs fixes",
			},
			currentAttempt: 2,
			maxRetries:     2,
			wantRetry:      false,
		},
		{
			name: "GREEN flag",
			result: &ReviewResult{
				Flag:     "GREEN",
				Feedback: "All good",
			},
			currentAttempt: 0,
			maxRetries:     2,
			wantRetry:      false,
		},
		{
			name: "YELLOW flag",
			result: &ReviewResult{
				Flag:     "YELLOW",
				Feedback: "Minor issues",
			},
			currentAttempt: 0,
			maxRetries:     2,
			wantRetry:      false,
		},
		{
			name: "RED flag first attempt",
			result: &ReviewResult{
				Flag:     "RED",
				Feedback: "Fails",
			},
			currentAttempt: 1,
			maxRetries:     2,
			wantRetry:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qc := NewQualityController(nil)
			qc.MaxRetries = tt.maxRetries

			got := qc.ShouldRetry(tt.result, tt.currentAttempt)
			if got != tt.wantRetry {
				t.Errorf("ShouldRetry() = %v, want %v", got, tt.wantRetry)
			}
		})
	}
}

func TestReview(t *testing.T) {
	tests := []struct {
		name       string
		task       models.Task
		output     string
		mockResult *agent.InvocationResult
		mockError  error
		wantFlag   string
		wantErr    bool
	}{
		{
			name: "successful GREEN review",
			task: models.Task{
				Number: 1,
				Name:   "Test task",
				Prompt: "Do something",
				Agent:  "developer",
			},
			output: "Task completed",
			mockResult: &agent.InvocationResult{
				Output:   "Quality Control: GREEN\n\nExcellent work!",
				ExitCode: 0,
				Duration: 50 * time.Millisecond,
			},
			mockError: nil,
			wantFlag:  "GREEN",
			wantErr:   false,
		},
		{
			name: "RED review requiring fixes",
			task: models.Task{
				Number: 2,
				Name:   "Fix bug",
				Prompt: "Fix the bug",
			},
			output: "Bug fixed",
			mockResult: &agent.InvocationResult{
				Output:   "Quality Control: RED\n\nTests still failing",
				ExitCode: 0,
				Duration: 75 * time.Millisecond,
			},
			mockError: nil,
			wantFlag:  "RED",
			wantErr:   false,
		},
		{
			name: "invoker error",
			task: models.Task{
				Number: 3,
				Name:   "Task",
				Prompt: "Do it",
			},
			output:     "Output",
			mockResult: nil,
			mockError:  errors.New("invocation failed"),
			wantFlag:   "",
			wantErr:    true,
		},
		{
			name: "YELLOW review",
			task: models.Task{
				Number: 4,
				Name:   "Implement feature",
				Prompt: "Add feature",
			},
			output: "Feature added",
			mockResult: &agent.InvocationResult{
				Output:   "Quality Control: YELLOW\n\nMinor documentation issues",
				ExitCode: 0,
				Duration: 60 * time.Millisecond,
			},
			mockError: nil,
			wantFlag:  "YELLOW",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockInvoker{
				mockInvoke: func(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
					// Verify the review task was constructed correctly
					if task.Agent != "quality-control" {
						t.Errorf("Review() task.Agent = %q, want %q", task.Agent, "quality-control")
					}
					if !contains(task.Prompt, tt.output) {
						t.Errorf("Review() task.Prompt missing original output")
					}

					return tt.mockResult, tt.mockError
				},
			}

			qc := NewQualityController(mock)
			ctx := context.Background()

			result, err := qc.Review(ctx, tt.task, tt.output)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Review() error = nil, wantErr = true")
				}
				return
			}

			if err != nil {
				t.Errorf("Review() unexpected error = %v", err)
				return
			}

			if result.Flag != tt.wantFlag {
				t.Errorf("Review() Flag = %q, want %q", result.Flag, tt.wantFlag)
			}
		})
	}
}

func TestQualityControlFlow(t *testing.T) {
	// Integration test of full QC workflow
	t.Run("complete workflow with GREEN response", func(t *testing.T) {
		mock := &mockInvoker{
			mockInvoke: func(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
				return &agent.InvocationResult{
					Output:   "Quality Control: GREEN\n\nAll requirements met",
					ExitCode: 0,
					Duration: 100 * time.Millisecond,
				}, nil
			},
		}

		qc := NewQualityController(mock)
		task := models.Task{
			Number: 1,
			Name:   "Implement feature",
			Prompt: "Add new feature",
			Agent:  "developer",
		}

		// Execute review
		result, err := qc.Review(context.Background(), task, "Feature implemented")
		if err != nil {
			t.Fatalf("Review() error = %v", err)
		}

		if result.Flag != "GREEN" {
			t.Errorf("Review() Flag = %q, want GREEN", result.Flag)
		}

		// Should not retry on GREEN
		if qc.ShouldRetry(result, 0) {
			t.Error("ShouldRetry() = true for GREEN, want false")
		}
	})

	t.Run("complete workflow with RED response and retry logic", func(t *testing.T) {
		mock := &mockInvoker{
			mockInvoke: func(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
				return &agent.InvocationResult{
					Output:   "Quality Control: RED\n\nTests failing, needs rework",
					ExitCode: 0,
					Duration: 100 * time.Millisecond,
				}, nil
			},
		}

		qc := NewQualityController(mock)
		qc.MaxRetries = 2
		task := models.Task{
			Number: 1,
			Name:   "Fix bug",
			Prompt: "Fix the bug",
		}

		// Execute review
		result, err := qc.Review(context.Background(), task, "Bug fixed")
		if err != nil {
			t.Fatalf("Review() error = %v", err)
		}

		if result.Flag != "RED" {
			t.Errorf("Review() Flag = %q, want RED", result.Flag)
		}

		// Should retry on first attempt
		if !qc.ShouldRetry(result, 0) {
			t.Error("ShouldRetry() = false for RED at attempt 0, want true")
		}

		// Should retry on second attempt (under max)
		if !qc.ShouldRetry(result, 1) {
			t.Error("ShouldRetry() = false for RED at attempt 1, want true")
		}

		// Should NOT retry when max retries reached
		if qc.ShouldRetry(result, 2) {
			t.Error("ShouldRetry() = true for RED at attempt 2 (max retries), want false")
		}
	})
}

func TestNewQualityController(t *testing.T) {
	mock := &mockInvoker{}
	qc := NewQualityController(mock)

	if qc.Invoker == nil {
		t.Error("NewQualityController() Invoker is nil")
	}

	if qc.ReviewAgent != "quality-control" {
		t.Errorf("NewQualityController() ReviewAgent = %q, want %q", qc.ReviewAgent, "quality-control")
	}

	if qc.MaxRetries != 2 {
		t.Errorf("NewQualityController() MaxRetries = %d, want 2", qc.MaxRetries)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
