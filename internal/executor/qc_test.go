package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/learning"
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
				Number: "1",
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
				Number: "2",
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
			ctx := context.Background()
			prompt := qc.BuildReviewPrompt(ctx, tt.task, tt.output)

			for _, want := range tt.wantContains {
				if !contains(prompt, want) {
					t.Errorf("BuildReviewPrompt() missing expected content %q\nGot: %s", want, prompt)
				}
			}
		})
	}
}

func TestBuildReviewPromptWithLearning(t *testing.T) {
	// Create in-memory store
	store, err := learning.NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	// Add some historical data
	task := models.Task{
		Number:     "1",
		Name:       "Test task",
		SourceFile: "plan.md",
	}

	exec := &learning.TaskExecution{
		PlanFile:     "plan.md",
		RunNumber:    1,
		TaskNumber:   "1",
		TaskName:     "Test task",
		Success:      false,
		QCVerdict:    "RED",
		QCFeedback:   "Tests failed",
		ErrorMessage: "compilation error",
	}

	if err := store.RecordExecution(context.Background(), exec); err != nil {
		t.Fatalf("RecordExecution() error = %v", err)
	}

	// Test with learning enabled
	t.Run("includes historical context when learning enabled", func(t *testing.T) {
		qc := NewQualityController(nil)
		qc.LearningStore = store

		ctx := context.Background()
		prompt := qc.BuildReviewPrompt(ctx, task, "Task output")

		// Should include historical context
		if !contains(prompt, "=== Historical Attempts ===") {
			t.Error("BuildReviewPrompt() missing historical attempts header")
		}
		if !contains(prompt, "Tests failed") {
			t.Error("BuildReviewPrompt() missing historical feedback")
		}
		if !contains(prompt, "compilation error") {
			t.Error("BuildReviewPrompt() missing historical error")
		}
	})

	// Test without learning
	t.Run("no historical context when learning disabled", func(t *testing.T) {
		qc := NewQualityController(nil)
		qc.LearningStore = nil

		ctx := context.Background()
		prompt := qc.BuildReviewPrompt(ctx, task, "Task output")

		// Should NOT include historical context
		if contains(prompt, "=== Historical Attempts ===") {
			t.Error("BuildReviewPrompt() should not include historical context when learning disabled")
		}
	})
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
			wantFlag:     models.StatusGreen,
			wantFeedback: "All tests pass",
		},
		{
			name:         "RED response",
			output:       "Quality Control: RED\n\nTests are failing",
			wantFlag:     models.StatusRed,
			wantFeedback: "Tests are failing",
		},
		{
			name:         "YELLOW response",
			output:       "Quality Control: YELLOW\n\nMinor issues detected",
			wantFlag:     models.StatusYellow,
			wantFeedback: "Minor issues detected",
		},
		{
			name:         "GREEN with extra whitespace",
			output:       "Quality Control:   GREEN  \n\n  Looks good!  ",
			wantFlag:     models.StatusGreen,
			wantFeedback: "Looks good!",
		},
		{
			name:         "multiline feedback",
			output:       "Quality Control: RED\n\nIssue 1: Tests failing\nIssue 2: Coverage low\nIssue 3: Linting errors",
			wantFlag:     models.StatusRed,
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
			wantFlag:     models.StatusGreen,
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
				Flag:     models.StatusRed,
				Feedback: "Needs fixes",
			},
			currentAttempt: 0,
			maxRetries:     2,
			wantRetry:      true,
		},
		{
			name: "RED flag at max retries",
			result: &ReviewResult{
				Flag:     models.StatusRed,
				Feedback: "Needs fixes",
			},
			currentAttempt: 2,
			maxRetries:     2,
			wantRetry:      false,
		},
		{
			name: "GREEN flag",
			result: &ReviewResult{
				Flag:     models.StatusGreen,
				Feedback: "All good",
			},
			currentAttempt: 0,
			maxRetries:     2,
			wantRetry:      false,
		},
		{
			name: "YELLOW flag",
			result: &ReviewResult{
				Flag:     models.StatusYellow,
				Feedback: "Minor issues",
			},
			currentAttempt: 0,
			maxRetries:     2,
			wantRetry:      false,
		},
		{
			name: "RED flag first attempt",
			result: &ReviewResult{
				Flag:     models.StatusRed,
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
				Number: "1",
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
			wantFlag:  models.StatusGreen,
			wantErr:   false,
		},
		{
			name: "RED review requiring fixes",
			task: models.Task{
				Number: "2",
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
			wantFlag:  models.StatusRed,
			wantErr:   false,
		},
		{
			name: "invoker error",
			task: models.Task{
				Number: "3",
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
				Number: "4",
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
			wantFlag:  models.StatusYellow,
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

func TestReviewWithLearning(t *testing.T) {
	// Create in-memory store
	store, err := learning.NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	task := models.Task{
		Number:     "1",
		Name:       "Test task",
		SourceFile: "plan.md",
		Prompt:     "Do something",
	}

	// Add historical failure
	exec := &learning.TaskExecution{
		PlanFile:     "plan.md",
		RunNumber:    1,
		TaskNumber:   "1",
		TaskName:     "Test task",
		Success:      false,
		QCVerdict:    "RED",
		QCFeedback:   "Previous attempt failed",
		ErrorMessage: "compilation error",
	}

	if err := store.RecordExecution(context.Background(), exec); err != nil {
		t.Fatalf("RecordExecution() error = %v", err)
	}

	// Mock that captures the prompt
	var capturedPrompt string
	mock := &mockInvoker{
		mockInvoke: func(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
			capturedPrompt = task.Prompt
			return &agent.InvocationResult{
				Output:   "Quality Control: GREEN\n\nLooks good now!",
				ExitCode: 0,
				Duration: 100 * time.Millisecond,
			}, nil
		},
	}

	qc := NewQualityController(mock)
	qc.LearningStore = store

	ctx := context.Background()
	result, err := qc.Review(ctx, task, "Task output")

	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	if result.Flag != models.StatusGreen {
		t.Errorf("Review() Flag = %q, want GREEN", result.Flag)
	}

	// Verify historical context was included in prompt
	if !contains(capturedPrompt, "=== Historical Attempts ===") {
		t.Error("Review() prompt missing historical context header")
	}
	if !contains(capturedPrompt, "Previous attempt failed") {
		t.Error("Review() prompt missing previous QC feedback")
	}
	if !contains(capturedPrompt, "compilation error") {
		t.Error("Review() prompt missing previous error message")
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
			Number: "1",
			Name:   "Implement feature",
			Prompt: "Add new feature",
			Agent:  "developer",
		}

		// Execute review
		result, err := qc.Review(context.Background(), task, "Feature implemented")
		if err != nil {
			t.Fatalf("Review() error = %v", err)
		}

		if result.Flag != models.StatusGreen {
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
			Number: "1",
			Name:   "Fix bug",
			Prompt: "Fix the bug",
		}

		// Execute review
		result, err := qc.Review(context.Background(), task, "Bug fixed")
		if err != nil {
			t.Fatalf("Review() error = %v", err)
		}

		if result.Flag != models.StatusRed {
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

	if qc.LearningStore != nil {
		t.Errorf("NewQualityController() LearningStore = %v, want nil", qc.LearningStore)
	}
}

// Helper function to check if a string contains a substring
func TestLoadContext(t *testing.T) {
	tests := []struct {
		name         string
		task         models.Task
		dbHistory    []*learning.TaskExecution
		wantContains []string
		wantErr      bool
	}{
		{
			name: "empty history",
			task: models.Task{
				Number:     "1",
				Name:       "Test task",
				SourceFile: "plan.md",
			},
			dbHistory:    []*learning.TaskExecution{},
			wantContains: []string{"=== Historical Attempts ===", "No previous attempts found"},
			wantErr:      false,
		},
		{
			name: "single failure in history",
			task: models.Task{
				Number:     "2",
				Name:       "Fix bug",
				SourceFile: "plan.md",
			},
			dbHistory: []*learning.TaskExecution{
				{
					TaskNumber:   "2",
					TaskName:     "Fix bug",
					Success:      false,
					ErrorMessage: "compilation error",
					QCVerdict:    "RED",
					QCFeedback:   "Tests failing",
				},
			},
			wantContains: []string{
				"=== Historical Attempts ===",
				"Fix bug",
				"compilation error",
				"RED",
				"Tests failing",
			},
			wantErr: false,
		},
		{
			name: "multiple attempts in history",
			task: models.Task{
				Number:     "3",
				Name:       "Implement feature",
				SourceFile: "plan.md",
			},
			dbHistory: []*learning.TaskExecution{
				{
					TaskNumber: "3",
					TaskName:   "Implement feature",
					Success:    true,
					QCVerdict:  "GREEN",
					QCFeedback: "All good",
				},
				{
					TaskNumber:   "3",
					TaskName:     "Implement feature",
					Success:      false,
					ErrorMessage: "missing import",
					QCVerdict:    "RED",
					QCFeedback:   "Import missing",
				},
			},
			wantContains: []string{
				"=== Historical Attempts ===",
				"Implement feature",
				"GREEN",
				"All good",
				"missing import",
				"RED",
				"Import missing",
			},
			wantErr: false,
		},
		{
			name: "history with success only",
			task: models.Task{
				Number:     "4",
				Name:       "Task success",
				SourceFile: "plan.md",
			},
			dbHistory: []*learning.TaskExecution{
				{
					TaskNumber: "4",
					TaskName:   "Task success",
					Success:    true,
					QCVerdict:  "GREEN",
					QCFeedback: "Perfect implementation",
				},
			},
			wantContains: []string{
				"=== Historical Attempts ===",
				"Task success",
				"Success: true",
				"GREEN",
				"Perfect implementation",
			},
			wantErr: false,
		},
		{
			name: "history without QC feedback",
			task: models.Task{
				Number:     "5",
				Name:       "No feedback task",
				SourceFile: "plan.md",
			},
			dbHistory: []*learning.TaskExecution{
				{
					TaskNumber: "5",
					TaskName:   "No feedback task",
					Success:    true,
					QCVerdict:  "GREEN",
					QCFeedback: "",
				},
			},
			wantContains: []string{
				"=== Historical Attempts ===",
				"No feedback task",
				"Success: true",
				"GREEN",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create in-memory store
			store, err := learning.NewStore(":memory:")
			if err != nil {
				t.Fatalf("NewStore() error = %v", err)
			}
			defer store.Close()

			// Populate test data
			for _, exec := range tt.dbHistory {
				exec.PlanFile = tt.task.SourceFile
				exec.RunNumber = 1
				if err := store.RecordExecution(context.Background(), exec); err != nil {
					t.Fatalf("RecordExecution() error = %v", err)
				}
			}

			qc := NewQualityController(nil)
			ctx := context.Background()

			context, err := qc.LoadContext(ctx, tt.task, store)

			if tt.wantErr {
				if err == nil {
					t.Errorf("LoadContext() error = nil, wantErr = true")
				}
				return
			}

			if err != nil {
				t.Errorf("LoadContext() unexpected error = %v", err)
				return
			}

			for _, want := range tt.wantContains {
				if !contains(context, want) {
					t.Errorf("LoadContext() missing expected content %q\nGot: %s", want, context)
				}
			}
		})
	}
}

func TestBuildQCJSONPrompt(t *testing.T) {
	tests := []struct {
		name         string
		basePrompt   string
		wantContains []string
	}{
		{
			name:       "adds JSON instruction",
			basePrompt: "Review this task execution",
			wantContains: []string{
				"Review this task execution",
				"IMPORTANT: Respond ONLY with valid JSON",
				"verdict",
				"feedback",
				"issues",
				"recommendations",
				"should_retry",
				"suggested_agent",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qc := NewQualityController(nil)
			prompt := qc.buildQCJSONPrompt(tt.basePrompt)

			for _, want := range tt.wantContains {
				if !contains(prompt, want) {
					t.Errorf("buildQCJSONPrompt() missing expected content %q\nGot: %s", want, prompt)
				}
			}
		})
	}
}

func TestParseQCJSON(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantVerdict string
		wantAgent   string
		wantErr     bool
	}{
		{
			name:        "GREEN verdict",
			output:      `{"verdict":"GREEN","feedback":"Good"}`,
			wantVerdict: "GREEN",
			wantAgent:   "",
			wantErr:     false,
		},
		{
			name:        "RED with suggestion",
			output:      `{"verdict":"RED","feedback":"Needs work","suggested_agent":"golang-pro"}`,
			wantVerdict: "RED",
			wantAgent:   "golang-pro",
			wantErr:     false,
		},
		{
			name:        "YELLOW verdict",
			output:      `{"verdict":"YELLOW","feedback":"Minor issues"}`,
			wantVerdict: "YELLOW",
			wantAgent:   "",
			wantErr:     false,
		},
		{
			name:        "full response with issues",
			output:      `{"verdict":"RED","feedback":"Failed","issues":[{"severity":"critical","description":"Test failed","location":"foo.go:10"}],"recommendations":["Fix test"],"should_retry":true,"suggested_agent":"test-expert"}`,
			wantVerdict: "RED",
			wantAgent:   "test-expert",
			wantErr:     false,
		},
		{
			name:        "invalid JSON",
			output:      `not json`,
			wantVerdict: "",
			wantAgent:   "",
			wantErr:     true,
		},
		{
			name:        "missing verdict",
			output:      `{"feedback":"Something"}`,
			wantVerdict: "",
			wantAgent:   "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := parseQCJSON(tt.output)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseQCJSON() error = nil, wantErr = true")
				}
				return
			}

			if err != nil {
				t.Errorf("parseQCJSON() unexpected error = %v", err)
				return
			}

			if resp.Verdict != tt.wantVerdict {
				t.Errorf("parseQCJSON() verdict = %q, want %q", resp.Verdict, tt.wantVerdict)
			}

			if resp.SuggestedAgent != tt.wantAgent {
				t.Errorf("parseQCJSON() suggested_agent = %q, want %q", resp.SuggestedAgent, tt.wantAgent)
			}
		})
	}
}

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
