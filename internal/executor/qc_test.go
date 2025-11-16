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
		{
			name:        "Claude CLI envelope format",
			output:      `{"type":"result","subtype":"success","result":"{\"verdict\":\"GREEN\",\"feedback\":\"Good work\",\"issues\":[],\"recommendations\":[],\"should_retry\":false,\"suggested_agent\":\"\"}"}`,
			wantVerdict: "GREEN",
			wantAgent:   "",
			wantErr:     false,
		},
		{
			name:        "Claude envelope with RED and agent suggestion",
			output:      `{"type":"result","subtype":"success","result":"{\"verdict\":\"RED\",\"feedback\":\"Tests failing\",\"issues\":[],\"recommendations\":[],\"should_retry\":true,\"suggested_agent\":\"golang-pro\"}"}`,
			wantVerdict: "RED",
			wantAgent:   "golang-pro",
			wantErr:     false,
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

func TestShouldUseMultiAgent(t *testing.T) {
	tests := []struct {
		name    string
		config  models.QCAgentConfig
		want    bool
	}{
		{
			name: "auto mode always uses multi-agent",
			config: models.QCAgentConfig{
				Mode: "auto",
			},
			want: true,
		},
		{
			name: "mixed mode always uses multi-agent",
			config: models.QCAgentConfig{
				Mode: "mixed",
			},
			want: true,
		},
		{
			name: "explicit mode with single agent",
			config: models.QCAgentConfig{
				Mode:         "explicit",
				ExplicitList: []string{"quality-control"},
			},
			want: false,
		},
		{
			name: "explicit mode with multiple agents",
			config: models.QCAgentConfig{
				Mode:         "explicit",
				ExplicitList: []string{"quality-control", "golang-pro"},
			},
			want: true,
		},
		{
			name: "empty mode defaults to single-agent",
			config: models.QCAgentConfig{
				Mode: "",
			},
			want: false,
		},
		{
			name: "unknown mode defaults to false",
			config: models.QCAgentConfig{
				Mode: "unknown",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qc := NewQualityController(nil)
			qc.AgentConfig = tt.config

			got := qc.shouldUseMultiAgent()
			if got != tt.want {
				t.Errorf("shouldUseMultiAgent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReviewMultiAgent(t *testing.T) {
	tests := []struct {
		name       string
		task       models.Task
		output     string
		agents     []string
		responses  map[string]*agent.InvocationResult
		wantFlag   string
		wantErr    bool
	}{
		{
			name: "all GREEN responses",
			task: models.Task{
				Number: "1",
				Name:   "Good task",
				Prompt: "Do something",
				Files:  []string{"main.go"},
			},
			output: "Task completed",
			agents: []string{"quality-control", "golang-pro"},
			responses: map[string]*agent.InvocationResult{
				"quality-control": {
					Output:   `{"verdict":"GREEN","feedback":"All good","issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
					ExitCode: 0,
				},
				"golang-pro": {
					Output:   `{"verdict":"GREEN","feedback":"Excellent Go code","issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
					ExitCode: 0,
				},
			},
			wantFlag: models.StatusGreen,
			wantErr:  false,
		},
		{
			name: "one RED one GREEN -> RED (strictest wins)",
			task: models.Task{
				Number: "2",
				Name:   "Mixed result",
				Prompt: "Do something",
				Files:  []string{"main.go"},
			},
			output: "Task output",
			agents: []string{"quality-control", "golang-pro"},
			responses: map[string]*agent.InvocationResult{
				"quality-control": {
					Output:   `{"verdict":"RED","feedback":"Tests failing","issues":[],"recommendations":[],"should_retry":true,"suggested_agent":""}`,
					ExitCode: 0,
				},
				"golang-pro": {
					Output:   `{"verdict":"GREEN","feedback":"Code is good","issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
					ExitCode: 0,
				},
			},
			wantFlag: models.StatusRed,
			wantErr:  false,
		},
		{
			name: "one YELLOW one GREEN -> YELLOW",
			task: models.Task{
				Number: "3",
				Name:   "Minor issues",
				Prompt: "Do something",
				Files:  []string{"main.go"},
			},
			output: "Task output",
			agents: []string{"quality-control", "golang-pro"},
			responses: map[string]*agent.InvocationResult{
				"quality-control": {
					Output:   `{"verdict":"YELLOW","feedback":"Minor doc issues","issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
					ExitCode: 0,
				},
				"golang-pro": {
					Output:   `{"verdict":"GREEN","feedback":"Code structure good","issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
					ExitCode: 0,
				},
			},
			wantFlag: models.StatusYellow,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockInvoker{
				mockInvoke: func(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
					return tt.responses[task.Agent], nil
				},
			}

			registry := &agent.Registry{
				AgentsDir: "/tmp/test-agents",
			}
			// For testing purposes, we just use nil registry since we can't modify unexported fields
			// The multi-agent test doesn't rely on language detection anyway

			qc := NewQualityController(mock)
			qc.Registry = registry
			qc.AgentConfig = models.QCAgentConfig{
				Mode: "auto",
			}

			ctx := context.Background()
			result, err := qc.ReviewMultiAgent(ctx, tt.task, tt.output)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ReviewMultiAgent() error = nil, wantErr = true")
				}
				return
			}

			if err != nil {
				t.Errorf("ReviewMultiAgent() unexpected error = %v", err)
				return
			}

			if result.Flag != tt.wantFlag {
				t.Errorf("ReviewMultiAgent() Flag = %q, want %q", result.Flag, tt.wantFlag)
			}

		})
	}
}

func TestAggregateVerdicts(t *testing.T) {
	tests := []struct {
		name    string
		results []*ReviewResult
		agents  []string
		want    string
	}{
		{
			name: "all GREEN",
			results: []*ReviewResult{
				{Flag: models.StatusGreen, Feedback: "Agent 1: Good", AgentName: "agent1"},
				{Flag: models.StatusGreen, Feedback: "Agent 2: Good", AgentName: "agent2"},
			},
			agents: []string{"agent1", "agent2"},
			want:   models.StatusGreen,
		},
		{
			name: "RED overrides GREEN",
			results: []*ReviewResult{
				{Flag: models.StatusGreen, Feedback: "Agent 1: Good", AgentName: "agent1"},
				{Flag: models.StatusRed, Feedback: "Agent 2: Bad", AgentName: "agent2"},
			},
			agents: []string{"agent1", "agent2"},
			want:   models.StatusRed,
		},
		{
			name: "RED overrides YELLOW",
			results: []*ReviewResult{
				{Flag: models.StatusYellow, Feedback: "Agent 1: Minor", AgentName: "agent1"},
				{Flag: models.StatusRed, Feedback: "Agent 2: Bad", AgentName: "agent2"},
			},
			agents: []string{"agent1", "agent2"},
			want:   models.StatusRed,
		},
		{
			name: "YELLOW overrides GREEN",
			results: []*ReviewResult{
				{Flag: models.StatusGreen, Feedback: "Agent 1: Good", AgentName: "agent1"},
				{Flag: models.StatusYellow, Feedback: "Agent 2: Minor", AgentName: "agent2"},
			},
			agents: []string{"agent1", "agent2"},
			want:   models.StatusYellow,
		},
		{
			name: "all YELLOW",
			results: []*ReviewResult{
				{Flag: models.StatusYellow, Feedback: "Agent 1: Minor", AgentName: "agent1"},
				{Flag: models.StatusYellow, Feedback: "Agent 2: Minor", AgentName: "agent2"},
			},
			agents: []string{"agent1", "agent2"},
			want:   models.StatusYellow,
		},
		{
			name: "all RED",
			results: []*ReviewResult{
				{Flag: models.StatusRed, Feedback: "Agent 1: Bad", AgentName: "agent1"},
				{Flag: models.StatusRed, Feedback: "Agent 2: Bad", AgentName: "agent2"},
			},
			agents: []string{"agent1", "agent2"},
			want:   models.StatusRed,
		},
		{
			name: "single GREEN",
			results: []*ReviewResult{
				{Flag: models.StatusGreen, Feedback: "All good", AgentName: "agent1"},
			},
			agents: []string{"agent1"},
			want:   models.StatusGreen,
		},
		{
			name: "single RED",
			results: []*ReviewResult{
				{Flag: models.StatusRed, Feedback: "Failed", AgentName: "agent1"},
			},
			agents: []string{"agent1"},
			want:   models.StatusRed,
		},
		{
			name: "single YELLOW",
			results: []*ReviewResult{
				{Flag: models.StatusYellow, Feedback: "Minor", AgentName: "agent1"},
			},
			agents: []string{"agent1"},
			want:   models.StatusYellow,
		},
		{
			name: "nil result in mix",
			results: []*ReviewResult{
				{Flag: models.StatusGreen, Feedback: "Agent 1: Good", AgentName: "agent1"},
				nil,
				{Flag: models.StatusGreen, Feedback: "Agent 3: Good", AgentName: "agent3"},
			},
			agents: []string{"agent1", "agent2", "agent3"},
			want:   models.StatusGreen,
		},
		{
			name: "RED with agent suggestion",
			results: []*ReviewResult{
				{Flag: models.StatusRed, Feedback: "Failed", AgentName: "agent1", SuggestedAgent: "golang-pro"},
				{Flag: models.StatusGreen, Feedback: "Good", AgentName: "agent2"},
			},
			agents: []string{"agent1", "agent2"},
			want:   models.StatusRed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qc := NewQualityController(nil)
			result := qc.aggregateVerdicts(tt.results, tt.agents)

			if result.Flag != tt.want {
				t.Errorf("aggregateVerdicts() Flag = %q, want %q", result.Flag, tt.want)
			}

			// Check that feedback is combined
			if len(tt.results) > 1 && result.Feedback == "" {
				t.Error("aggregateVerdicts() should combine feedback from multiple agents")
			}

		})
	}
}

func TestExtractJSONFromCodeFence(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid JSON with ```json fence",
			input: "```json\n{\"verdict\":\"GREEN\",\"feedback\":\"Good\"}\n```",
			want:  "{\"verdict\":\"GREEN\",\"feedback\":\"Good\"}",
		},
		{
			name:  "valid JSON with just ``` fence",
			input: "```\n{\"verdict\":\"RED\",\"feedback\":\"Bad\"}\n```",
			want:  "{\"verdict\":\"RED\",\"feedback\":\"Bad\"}",
		},
		{
			name:  "JSON without fences",
			input: "{\"verdict\":\"GREEN\",\"feedback\":\"Good\"}",
			want:  "{\"verdict\":\"GREEN\",\"feedback\":\"Good\"}",
		},
		{
			name:  "multiline JSON in fence",
			input: "```json\n{\n  \"verdict\": \"GREEN\",\n  \"feedback\": \"Good\"\n}\n```",
			want:  "{\n  \"verdict\": \"GREEN\",\n  \"feedback\": \"Good\"\n}",
		},
		{
			name:  "fence with language identifier",
			input: "```javascript\n{\"verdict\":\"YELLOW\",\"feedback\":\"OK\"}\n```",
			want:  "{\"verdict\":\"YELLOW\",\"feedback\":\"OK\"}",
		},
		{
			name:  "text with fence - returns all text",
			input: "Here's the result:\n```json\n{\"verdict\":\"GREEN\"}\n```\nDone",
			want:  "Here's the result:\n```json\n{\"verdict\":\"GREEN\"}\n```\nDone", // No fence found at start/end
		},
		{
			name:  "empty fence - returns empty string",
			input: "```json\n\n```",
			want:  "", // No JSON found in fence
		},
		{
			name:  "whitespace around JSON",
			input: "```json\n  {\"verdict\":\"GREEN\"}  \n```",
			want:  "{\"verdict\":\"GREEN\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSONFromCodeFence(tt.input)
			if got != tt.want {
				t.Errorf("extractJSONFromCodeFence() = %q, want %q", got, tt.want)
			}
		})
	}
}

// mockQCLogger for testing QC logging calls
type mockQCLogger struct {
	agentSelectionCalls     []struct{ agents []string; mode string }
	individualVerdictsCalls []map[string]string
	aggregatedResultCalls   []struct{ verdict, strategy string }
}

func (m *mockQCLogger) LogQCAgentSelection(agents []string, mode string) {
	m.agentSelectionCalls = append(m.agentSelectionCalls, struct {
		agents []string
		mode   string
	}{agents: agents, mode: mode})
}

func (m *mockQCLogger) LogQCIndividualVerdicts(verdicts map[string]string) {
	m.individualVerdictsCalls = append(m.individualVerdictsCalls, verdicts)
}

func (m *mockQCLogger) LogQCAggregatedResult(verdict string, strategy string) {
	m.aggregatedResultCalls = append(m.aggregatedResultCalls, struct {
		verdict   string
		strategy  string
	}{verdict: verdict, strategy: strategy})
}

func TestReviewMultiAgent_LoggingCalls(t *testing.T) {
	tests := []struct {
		name          string
		agents        []string
		verdicts      []string
		expectedCount int
		expectLogging bool
	}{
		{
			name:          "logs agent selection and results with multiple agents",
			agents:        []string{"quality-control", "code-reviewer"},
			verdicts:      []string{models.StatusGreen, models.StatusGreen},
			expectedCount: 2,
			expectLogging: true,
		},
		{
			name:          "logs agent selection with single agent",
			agents:        []string{"quality-control"},
			verdicts:      []string{models.StatusGreen},
			expectedCount: 1,
			expectLogging: true,
		},
		{
			name:          "logs mixed verdicts",
			agents:        []string{"quality-control", "code-reviewer"},
			verdicts:      []string{models.StatusGreen, models.StatusRed},
			expectedCount: 2,
			expectLogging: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := &mockQCLogger{}
			mockInv := &mockInvoker{
				mockInvoke: func(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
					// Find which agent is being invoked
					idx := 0
					for i, a := range tt.agents {
						if task.Agent == a {
							idx = i
							break
						}
					}
					verdict := models.StatusGreen
					if idx < len(tt.verdicts) {
						verdict = tt.verdicts[idx]
					}
					return &agent.InvocationResult{
						Output: `{
							"verdict": "` + verdict + `",
							"feedback": "Test feedback",
							"issues": [],
							"recommendations": [],
							"should_retry": false,
							"suggested_agent": ""
						}`,
						ExitCode: 0,
						Duration: 100 * time.Millisecond,
					}, nil
				},
			}

			qc := NewQualityController(mockInv)
			qc.Logger = mockLogger
			qc.AgentConfig = models.QCAgentConfig{
				Mode:         "explicit",
				ExplicitList: tt.agents,
			}

			ctx := context.Background()
			task := models.Task{
				Number: "1",
				Name:   "Test",
				Prompt: "Review this",
				Files:  []string{"test.go"},
			}

			result, err := qc.ReviewMultiAgent(ctx, task, "output")

			if err != nil {
				t.Errorf("ReviewMultiAgent() error = %v", err)
			}

			if result == nil {
				t.Errorf("ReviewMultiAgent() returned nil result")
			}

			// Verify logging calls
			if tt.expectLogging {
				// For single agent, we don't log verdicts since it falls back to single-agent review
				if len(tt.agents) == 1 {
					if len(mockLogger.agentSelectionCalls) != 1 {
						t.Errorf("expected 1 agent selection call, got %d", len(mockLogger.agentSelectionCalls))
					}
					// Single agent path doesn't call LogQCIndividualVerdicts
					if len(mockLogger.individualVerdictsCalls) != 0 {
						t.Errorf("single-agent path should not call LogQCIndividualVerdicts, got %d calls",
							len(mockLogger.individualVerdictsCalls))
					}
				} else {
					// Multi-agent path
					if len(mockLogger.agentSelectionCalls) != 1 {
						t.Errorf("expected 1 agent selection call, got %d", len(mockLogger.agentSelectionCalls))
					}
					if mockLogger.agentSelectionCalls[0].mode != "explicit" {
						t.Errorf("expected mode 'explicit', got %q", mockLogger.agentSelectionCalls[0].mode)
					}
					if len(mockLogger.individualVerdictsCalls) != 1 {
						t.Errorf("expected 1 verdicts call, got %d", len(mockLogger.individualVerdictsCalls))
					}
					if len(mockLogger.aggregatedResultCalls) != 1 {
						t.Errorf("expected 1 aggregated result call, got %d", len(mockLogger.aggregatedResultCalls))
					}
					if mockLogger.aggregatedResultCalls[0].strategy != "strictest-wins" {
						t.Errorf("expected strategy 'strictest-wins', got %q", mockLogger.aggregatedResultCalls[0].strategy)
					}
				}
			}
		})
	}
}

func TestReviewMultiAgent_NoLoggingWithoutLogger(t *testing.T) {
	mockInv := &mockInvoker{
		mockInvoke: func(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
			return &agent.InvocationResult{
				Output: `{
					"verdict": "GREEN",
					"feedback": "Good",
					"issues": [],
					"recommendations": [],
					"should_retry": false,
					"suggested_agent": ""
				}`,
				ExitCode: 0,
				Duration: 100 * time.Millisecond,
			}, nil
		},
	}

	qc := NewQualityController(mockInv)
	qc.Logger = nil // Explicitly no logger
	qc.AgentConfig = models.QCAgentConfig{
		Mode:         "explicit",
		ExplicitList: []string{"quality-control", "code-reviewer"},
	}

	ctx := context.Background()
	task := models.Task{
		Number: "1",
		Name:   "Test",
		Prompt: "Review this",
		Files:  []string{"test.go"},
	}

	result, err := qc.ReviewMultiAgent(ctx, task, "output")

	if err != nil {
		t.Errorf("ReviewMultiAgent() error = %v", err)
	}

	if result == nil {
		t.Errorf("ReviewMultiAgent() returned nil result")
	}

	// Should not panic even with nil logger
	if result.Flag != models.StatusGreen {
		t.Errorf("expected GREEN verdict, got %q", result.Flag)
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
