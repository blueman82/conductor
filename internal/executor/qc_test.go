package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/behavioral"
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
				"Agent Output:",
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
				Output:   `{"verdict":"GREEN","feedback":"Excellent work!","issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
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
				Output:   `{"verdict":"RED","feedback":"Tests still failing","issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
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
				Output:   `{"verdict":"YELLOW","feedback":"Minor documentation issues","issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
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
			// Disable multi-agent mode for legacy single-agent testing
			qc.AgentConfig.Mode = ""
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
				Output:   `{"verdict":"GREEN","feedback":"Looks good now!","issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
				ExitCode: 0,
				Duration: 100 * time.Millisecond,
			}, nil
		},
	}

	qc := NewQualityController(mock)
	// Disable multi-agent mode for legacy single-agent testing
	qc.AgentConfig.Mode = ""
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
					Output:   `{"verdict":"GREEN","feedback":"All requirements met","issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
					ExitCode: 0,
					Duration: 100 * time.Millisecond,
				}, nil
			},
		}

		qc := NewQualityController(mock)
		// Disable multi-agent mode for legacy single-agent testing
		qc.AgentConfig.Mode = ""
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
					Output:   `{"verdict":"RED","feedback":"Tests failing, needs rework","issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
					ExitCode: 0,
					Duration: 100 * time.Millisecond,
				}, nil
			},
		}

		qc := NewQualityController(mock)
		// Disable multi-agent mode for legacy single-agent testing
		qc.AgentConfig.Mode = ""
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
			resp, err := parseQCJSON(tt.output) // No criteria expected for legacy tests

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
		name   string
		config models.QCAgentConfig
		want   bool
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
		name      string
		task      models.Task
		output    string
		agents    []string
		responses map[string]*agent.InvocationResult
		wantFlag  string
		wantErr   bool
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

// mockQCLogger for testing QC logging calls
type mockQCLogger struct {
	agentSelectionCalls []struct {
		agents []string
		mode   string
	}
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
		verdict  string
		strategy string
	}{verdict: verdict, strategy: strategy})
}

func (m *mockQCLogger) LogQCCriteriaResults(agentName string, results []models.CriterionResult) {
	// no-op for mock
}

func (m *mockQCLogger) LogQCIntelligentSelectionMetadata(rationale string, fallback bool, fallbackReason string) {
	// no-op for mock
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

func TestBuildStructuredReviewPrompt_WithCriteria(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		Name:   "Test Task",
		Prompt: "Implement feature",
		SuccessCriteria: []string{
			"Criterion A",
			"Criterion B",
		},
		TestCommands: []string{"go test ./..."},
	}

	prompt := qc.BuildStructuredReviewPrompt(context.Background(), task, "agent output")

	if !contains(prompt, "0. [ ] Criterion A") {
		t.Error("should include numbered criterion 0")
	}
	if !contains(prompt, "1. [ ] Criterion B") {
		t.Error("should include numbered criterion 1")
	}
	if !contains(prompt, "go test ./...") {
		t.Error("should include test commands")
	}
	if !contains(prompt, "agent output") {
		t.Error("should include agent output")
	}
	if !contains(prompt, "Test Task") {
		t.Error("should include task name")
	}
	if !contains(prompt, "Implement feature") {
		t.Error("should include task prompt")
	}
}

func TestBuildStructuredReviewPrompt_NoCriteria_UsesLegacy(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		Name:   "Legacy Task",
		Prompt: "Do something",
	}

	prompt := qc.BuildStructuredReviewPrompt(context.Background(), task, "output here")

	// Should still produce a valid prompt but without criteria section
	if !contains(prompt, "Legacy Task") {
		t.Error("should include task name")
	}
	if !contains(prompt, "Do something") {
		t.Error("should include task prompt")
	}
	if !contains(prompt, "output here") {
		t.Error("should include agent output")
	}
	// Should NOT have numbered criteria
	if contains(prompt, "0. [ ]") {
		t.Error("should not include numbered criteria when SuccessCriteria is empty")
	}
}

func TestBuildStructuredReviewPrompt_MultipleTestCommands(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		Name:   "Multi Command Task",
		Prompt: "Build feature",
		SuccessCriteria: []string{
			"Tests pass",
		},
		TestCommands: []string{
			"go test ./...",
			"go vet ./...",
			"golangci-lint run",
		},
	}

	prompt := qc.BuildStructuredReviewPrompt(context.Background(), task, "output")

	if !contains(prompt, "go test ./...") {
		t.Error("should include first test command")
	}
	if !contains(prompt, "go vet ./...") {
		t.Error("should include second test command")
	}
	if !contains(prompt, "golangci-lint run") {
		t.Error("should include third test command")
	}
}

func TestBuildStructuredReviewPrompt_CriteriaIndexing(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		Name:   "Indexing Test",
		Prompt: "Check indexing",
		SuccessCriteria: []string{
			"First criterion",
			"Second criterion",
			"Third criterion",
			"Fourth criterion",
		},
		TestCommands: []string{},
	}

	prompt := qc.BuildStructuredReviewPrompt(context.Background(), task, "output")

	// Verify 0-based indexing for machine parsing
	if !contains(prompt, "0. [ ] First criterion") {
		t.Error("criterion 0 not found with correct format")
	}
	if !contains(prompt, "1. [ ] Second criterion") {
		t.Error("criterion 1 not found with correct format")
	}
	if !contains(prompt, "2. [ ] Third criterion") {
		t.Error("criterion 2 not found with correct format")
	}
	if !contains(prompt, "3. [ ] Fourth criterion") {
		t.Error("criterion 3 not found with correct format")
	}
}

func TestBuildStructuredReviewPrompt_WithIntegrationCriteria(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		Name:   "Integration Test",
		Prompt: "Test integration",
		Type:   "integration",
		SuccessCriteria: []string{
			"Feature A works",
			"Feature B works",
		},
		IntegrationCriteria: []string{
			"A integrates with B",
			"No regression in C",
			"Performance acceptable",
		},
		TestCommands: []string{},
	}

	prompt := qc.BuildStructuredReviewPrompt(context.Background(), task, "output")

	// Verify success criteria (0-1)
	if !contains(prompt, "0. [ ] Feature A works") {
		t.Error("success criterion 0 not found")
	}
	if !contains(prompt, "1. [ ] Feature B works") {
		t.Error("success criterion 1 not found")
	}

	// Verify integration criteria start at index 2
	if !contains(prompt, "2. [ ] A integrates with B") {
		t.Error("integration criterion 0 (index 2) not found")
	}
	if !contains(prompt, "3. [ ] No regression in C") {
		t.Error("integration criterion 1 (index 3) not found")
	}
	if !contains(prompt, "4. [ ] Performance acceptable") {
		t.Error("integration criterion 2 (index 4) not found")
	}

	// Verify INTEGRATION CRITERIA section header
	if !contains(prompt, "## INTEGRATION CRITERIA") {
		t.Error("INTEGRATION CRITERIA section header not found")
	}
}

func TestBuildStructuredReviewPrompt_IntegrationWithoutSuccessCriteria(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		Name:            "Integration Only",
		Prompt:          "Integration test",
		Type:            "integration",
		SuccessCriteria: []string{}, // No success criteria
		IntegrationCriteria: []string{
			"Service A calls Service B",
			"Data consistency maintained",
		},
		TestCommands: []string{},
	}

	prompt := qc.BuildStructuredReviewPrompt(context.Background(), task, "output")

	// Verify integration criteria start at index 0 when no success criteria
	if !contains(prompt, "0. [ ] Service A calls Service B") {
		t.Error("integration criterion 0 (index 0) not found")
	}
	if !contains(prompt, "1. [ ] Data consistency maintained") {
		t.Error("integration criterion 1 (index 1) not found")
	}

	// Verify INTEGRATION CRITERIA section header
	if !contains(prompt, "## INTEGRATION CRITERIA") {
		t.Error("INTEGRATION CRITERIA section header not found")
	}
}

func TestGetCombinedCriteria_SuccessOnly(t *testing.T) {
	task := models.Task{
		SuccessCriteria: []string{"A", "B"},
	}
	combined := getCombinedCriteria(task)
	if len(combined) != 2 {
		t.Errorf("expected 2 criteria, got %d", len(combined))
	}
	if combined[0] != "A" || combined[1] != "B" {
		t.Errorf("unexpected combined criteria: %v", combined)
	}
}

func TestGetCombinedCriteria_IntegrationOnly(t *testing.T) {
	task := models.Task{
		Type:                "integration",
		IntegrationCriteria: []string{"X", "Y", "Z"},
	}
	combined := getCombinedCriteria(task)
	if len(combined) != 3 {
		t.Errorf("expected 3 criteria, got %d", len(combined))
	}
	if combined[0] != "X" || combined[1] != "Y" || combined[2] != "Z" {
		t.Errorf("unexpected combined criteria: %v", combined)
	}
}

func TestGetCombinedCriteria_Both(t *testing.T) {
	task := models.Task{
		Type:                "integration",
		SuccessCriteria:     []string{"A", "B"},
		IntegrationCriteria: []string{"X", "Y"},
	}
	combined := getCombinedCriteria(task)
	if len(combined) != 4 {
		t.Errorf("expected 4 criteria, got %d", len(combined))
	}
	if combined[0] != "A" || combined[1] != "B" || combined[2] != "X" || combined[3] != "Y" {
		t.Errorf("unexpected combined criteria: %v", combined)
	}
}

func TestGetCombinedCriteria_NonIntegrationIgnoresIntegrationCriteria(t *testing.T) {
	task := models.Task{
		Type:                "regular", // Not integration
		SuccessCriteria:     []string{"A"},
		IntegrationCriteria: []string{"X", "Y"}, // Should be ignored
	}
	combined := getCombinedCriteria(task)
	if len(combined) != 1 {
		t.Errorf("expected 1 criterion (non-integration should ignore integration criteria), got %d", len(combined))
	}
	if combined[0] != "A" {
		t.Errorf("unexpected combined criteria: %v", combined)
	}
}

func TestAggregateCriteriaResults_AllPass(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		SuccessCriteria: []string{"A", "B"},
	}
	resp := &models.QCResponse{
		Verdict: "GREEN",
		CriteriaResults: []models.CriterionResult{
			{Index: 0, Passed: true},
			{Index: 1, Passed: true},
		},
	}

	verdict := qc.aggregateCriteriaResults(resp, task, 2)
	if verdict != models.StatusGreen {
		t.Errorf("expected GREEN, got %s", verdict)
	}
}

func TestAggregateCriteriaResults_OneFails_ReturnsRed(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		SuccessCriteria: []string{"A", "B"},
	}
	resp := &models.QCResponse{
		CriteriaResults: []models.CriterionResult{
			{Index: 0, Passed: true},
			{Index: 1, Passed: false},
		},
	}

	verdict := qc.aggregateCriteriaResults(resp, task, 2)
	if verdict != models.StatusRed {
		t.Errorf("expected RED when criterion fails, got %s", verdict)
	}
}

func TestAggregateCriteriaResults_NoCriteria_UsesAgentVerdict(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		SuccessCriteria: []string{},
	}
	resp := &models.QCResponse{
		Verdict:         "YELLOW",
		CriteriaResults: []models.CriterionResult{},
	}

	verdict := qc.aggregateCriteriaResults(resp, task, 0)
	if verdict != "YELLOW" {
		t.Errorf("expected YELLOW (agent verdict), got %s", verdict)
	}
}

func TestAggregateCriteriaResults_AllFail(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		SuccessCriteria: []string{"A", "B", "C"},
	}
	resp := &models.QCResponse{
		Verdict: "GREEN", // Agent says GREEN but criteria fail
		CriteriaResults: []models.CriterionResult{
			{Index: 0, Passed: false},
			{Index: 1, Passed: false},
			{Index: 2, Passed: false},
		},
	}

	verdict := qc.aggregateCriteriaResults(resp, task, 3)
	if verdict != models.StatusRed {
		t.Errorf("expected RED when all criteria fail, got %s", verdict)
	}
}

func TestAggregateCriteriaResults_FirstFails(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		SuccessCriteria: []string{"A", "B"},
	}
	resp := &models.QCResponse{
		Verdict: "GREEN",
		CriteriaResults: []models.CriterionResult{
			{Index: 0, Passed: false},
			{Index: 1, Passed: true},
		},
	}

	verdict := qc.aggregateCriteriaResults(resp, task, 2)
	if verdict != models.StatusRed {
		t.Errorf("expected RED when first criterion fails, got %s", verdict)
	}
}

func TestMultiAgentCriteriaConsensus_Unanimous(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		SuccessCriteria: []string{"A", "B"},
	}
	results := []*ReviewResult{
		{
			AgentName: "agent1",
			CriteriaResults: []models.CriterionResult{
				{Index: 0, Passed: true},
				{Index: 1, Passed: true},
			},
		},
		{
			AgentName: "agent2",
			CriteriaResults: []models.CriterionResult{
				{Index: 0, Passed: true},
				{Index: 1, Passed: true},
			},
		},
	}

	final := qc.aggregateMultiAgentCriteria(results, task, 2)
	if final.Flag != models.StatusGreen {
		t.Errorf("expected GREEN (all unanimous), got %s", final.Flag)
	}
}

func TestMultiAgentCriteriaConsensus_OneAgentFails(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		SuccessCriteria: []string{"A", "B"},
	}
	results := []*ReviewResult{
		{
			AgentName: "agent1",
			CriteriaResults: []models.CriterionResult{
				{Index: 0, Passed: true},
				{Index: 1, Passed: true},
			},
		},
		{
			AgentName: "agent2",
			CriteriaResults: []models.CriterionResult{
				{Index: 0, Passed: true},
				{Index: 1, Passed: false}, // One agent fails B
			},
		},
	}

	final := qc.aggregateMultiAgentCriteria(results, task, 2)
	if final.Flag != models.StatusRed {
		t.Errorf("expected RED (not unanimous on B), got %s", final.Flag)
	}
}

func TestMultiAgentCriteriaConsensus_MixedResults(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		SuccessCriteria: []string{"A", "B", "C"},
	}
	results := []*ReviewResult{
		{
			AgentName: "agent1",
			CriteriaResults: []models.CriterionResult{
				{Index: 0, Passed: true},
				{Index: 1, Passed: false}, // Agent1 fails B
				{Index: 2, Passed: true},
			},
		},
		{
			AgentName: "agent2",
			CriteriaResults: []models.CriterionResult{
				{Index: 0, Passed: true},
				{Index: 1, Passed: true},
				{Index: 2, Passed: false}, // Agent2 fails C
			},
		},
		{
			AgentName: "agent3",
			CriteriaResults: []models.CriterionResult{
				{Index: 0, Passed: true},
				{Index: 1, Passed: true},
				{Index: 2, Passed: true},
			},
		},
	}

	final := qc.aggregateMultiAgentCriteria(results, task, 3)
	if final.Flag != models.StatusRed {
		t.Errorf("expected RED (mixed failures), got %s", final.Flag)
	}
}

func TestMultiAgentCriteriaConsensus_SkipsAgentsWithoutCriteria(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		SuccessCriteria: []string{"A", "B"},
	}
	results := []*ReviewResult{
		{
			AgentName: "agent1",
			CriteriaResults: []models.CriterionResult{
				{Index: 0, Passed: true},
				{Index: 1, Passed: true},
			},
		},
		{
			AgentName:       "agent2",
			CriteriaResults: []models.CriterionResult{}, // No criteria results
		},
		{
			AgentName: "agent3",
			CriteriaResults: []models.CriterionResult{
				{Index: 0, Passed: true},
				{Index: 1, Passed: true},
			},
		},
	}

	final := qc.aggregateMultiAgentCriteria(results, task, 2)
	// Should be GREEN as agent2 is skipped (non-compliant)
	if final.Flag != models.StatusGreen {
		t.Errorf("expected GREEN (skips non-compliant agent), got %s", final.Flag)
	}
}

func TestMultiAgentCriteriaConsensus_NoCriteria_FallsBackToAggregateVerdicts(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		SuccessCriteria: []string{},
	}
	results := []*ReviewResult{
		{
			AgentName: "agent1",
			Flag:      models.StatusGreen,
			Feedback:  "Good",
		},
		{
			AgentName: "agent2",
			Flag:      models.StatusYellow,
			Feedback:  "Minor issues",
		},
	}

	final := qc.aggregateMultiAgentCriteria(results, task, 0)
	// Should fall back to aggregateVerdicts (strictest-wins)
	if final.Flag != models.StatusYellow {
		t.Errorf("expected YELLOW (fallback to aggregateVerdicts), got %s", final.Flag)
	}
}

func TestMultiAgentCriteriaConsensus_NilResult(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		SuccessCriteria: []string{"A"},
	}
	results := []*ReviewResult{
		{
			AgentName: "agent1",
			CriteriaResults: []models.CriterionResult{
				{Index: 0, Passed: true},
			},
		},
		nil, // Nil result
		{
			AgentName: "agent3",
			CriteriaResults: []models.CriterionResult{
				{Index: 0, Passed: true},
			},
		},
	}

	final := qc.aggregateMultiAgentCriteria(results, task, 1)
	if final.Flag != models.StatusGreen {
		t.Errorf("expected GREEN (skips nil result), got %s", final.Flag)
	}
}

func TestMultiAgentCriteriaConsensus_AllAgentsNonCompliant(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		SuccessCriteria: []string{"A", "B"},
	}
	results := []*ReviewResult{
		{
			AgentName:       "agent1",
			CriteriaResults: []models.CriterionResult{}, // No criteria
		},
		{
			AgentName:       "agent2",
			CriteriaResults: []models.CriterionResult{}, // No criteria
		},
	}

	final := qc.aggregateMultiAgentCriteria(results, task, 2)
	// No votes means fail (no consensus possible)
	if final.Flag != models.StatusRed {
		t.Errorf("expected RED (no compliant agents), got %s", final.Flag)
	}
}

func TestReview_WithStructuredCriteria(t *testing.T) {
	tests := []struct {
		name         string
		task         models.Task
		response     string
		wantFlag     string
		wantCriteria bool
	}{
		{
			name: "task with criteria uses structured prompt and criteria override",
			task: models.Task{
				Number: "1",
				Name:   "Structured task",
				Prompt: "Implement feature",
				SuccessCriteria: []string{
					"Tests pass",
					"Code compiles",
				},
			},
			response:     `{"verdict":"GREEN","feedback":"All good","criteria_results":[{"index":0,"passed":true,"evidence":"Tests pass"},{"index":1,"passed":true,"evidence":"Compiles"}]}`,
			wantFlag:     models.StatusGreen,
			wantCriteria: true,
		},
		{
			name: "criteria failure overrides agent GREEN verdict",
			task: models.Task{
				Number: "2",
				Name:   "Failing criteria",
				Prompt: "Implement feature",
				SuccessCriteria: []string{
					"Tests pass",
					"Code compiles",
				},
			},
			response:     `{"verdict":"GREEN","feedback":"Looks good","criteria_results":[{"index":0,"passed":true,"evidence":"Tests pass"},{"index":1,"passed":false,"fail_reason":"Does not compile"}]}`,
			wantFlag:     models.StatusRed,
			wantCriteria: true,
		},
		{
			name: "all criteria pass returns GREEN",
			task: models.Task{
				Number: "3",
				Name:   "All pass",
				Prompt: "Do it",
				SuccessCriteria: []string{
					"A",
					"B",
					"C",
				},
			},
			response:     `{"verdict":"YELLOW","feedback":"Minor issues","criteria_results":[{"index":0,"passed":true},{"index":1,"passed":true},{"index":2,"passed":true}]}`,
			wantFlag:     models.StatusGreen,
			wantCriteria: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPrompt string
			mock := &mockInvoker{
				mockInvoke: func(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
					capturedPrompt = task.Prompt
					return &agent.InvocationResult{
						Output:   tt.response,
						ExitCode: 0,
						Duration: 100 * time.Millisecond,
					}, nil
				},
			}

			qc := NewQualityController(mock)
			// Disable multi-agent mode to test single-agent path
			qc.AgentConfig.Mode = ""
			ctx := context.Background()

			result, err := qc.Review(ctx, tt.task, "agent output")
			if err != nil {
				t.Fatalf("Review() error = %v", err)
			}

			if result.Flag != tt.wantFlag {
				t.Errorf("Review() Flag = %q, want %q", result.Flag, tt.wantFlag)
			}

			// Verify structured prompt was used
			if tt.wantCriteria {
				if !contains(capturedPrompt, "SUCCESS CRITERIA") {
					t.Error("should use structured prompt with SUCCESS CRITERIA section")
				}
			}
		})
	}
}

func TestReview_LegacyTaskWithoutCriteria(t *testing.T) {
	mock := &mockInvoker{
		mockInvoke: func(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
			return &agent.InvocationResult{
				Output:   `{"verdict":"GREEN","feedback":"Looks good"}`,
				ExitCode: 0,
				Duration: 100 * time.Millisecond,
			}, nil
		},
	}

	qc := NewQualityController(mock)
	// Disable multi-agent mode to test single-agent path
	qc.AgentConfig.Mode = ""
	task := models.Task{
		Number: "1",
		Name:   "Legacy task",
		Prompt: "Do something",
		// No SuccessCriteria
	}

	ctx := context.Background()
	result, err := qc.Review(ctx, task, "output")
	if err != nil {
		t.Fatalf("Review() unexpected error: %v", err)
	}

	if result.Flag != models.StatusGreen {
		t.Errorf("legacy task should use agent verdict, got %s", result.Flag)
	}
}

func TestReview_FallbackToYellowOnMalformedResponse(t *testing.T) {
	mock := &mockInvoker{
		mockInvoke: func(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
			// Agent returns valid JSON but no criteria_results (non-compliant)
			return &agent.InvocationResult{
				Output:   `{"verdict":"GREEN","feedback":"Looks good","issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
				ExitCode: 0,
				Duration: 100 * time.Millisecond,
			}, nil
		},
	}

	qc := NewQualityController(mock)
	// Disable multi-agent mode to test single-agent path
	qc.AgentConfig.Mode = ""
	task := models.Task{
		Number: "1",
		Name:   "Task with criteria",
		Prompt: "Do something",
		SuccessCriteria: []string{
			"Criterion A",
			"Criterion B",
		},
	}

	ctx := context.Background()
	result, err := qc.Review(ctx, task, "output")
	if err != nil {
		t.Fatalf("Review() unexpected error: %v", err)
	}

	// Should be YELLOW when agent doesn't return criteria_results for task with criteria
	if result.Flag != models.StatusYellow {
		t.Errorf("should fallback to YELLOW on missing criteria_results, got %s", result.Flag)
	}

	// Feedback should contain warning
	if !contains(result.Feedback, "criteria_results") {
		t.Error("feedback should mention missing criteria_results")
	}
}

func TestReviewMultiAgent_WithCriteriaResults(t *testing.T) {
	tests := []struct {
		name         string
		task         models.Task
		output       string
		agents       []string
		responses    map[string]*agent.InvocationResult
		wantFlag     string
		wantStrategy string
	}{
		{
			name: "all agents unanimous GREEN on all criteria",
			task: models.Task{
				Number: "1",
				Name:   "Feature with criteria",
				Prompt: "Implement feature",
				SuccessCriteria: []string{
					"Tests pass",
					"Code compiles",
				},
			},
			output: "Task completed",
			agents: []string{"quality-control", "golang-pro"},
			responses: map[string]*agent.InvocationResult{
				"quality-control": {
					Output:   `{"verdict":"GREEN","feedback":"All criteria met","criteria_results":[{"index":0,"passed":true,"evidence":"Tests pass"},{"index":1,"passed":true,"evidence":"Compiles"}],"issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
					ExitCode: 0,
				},
				"golang-pro": {
					Output:   `{"verdict":"GREEN","feedback":"Go code is excellent","criteria_results":[{"index":0,"passed":true,"evidence":"All tests pass"},{"index":1,"passed":true,"evidence":"No compilation errors"}],"issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
					ExitCode: 0,
				},
			},
			wantFlag:     models.StatusGreen,
			wantStrategy: "multi-agent-criteria-consensus",
		},
		{
			name: "agent disagreement on criterion -> RED",
			task: models.Task{
				Number: "2",
				Name:   "Disagreement task",
				Prompt: "Implement feature",
				SuccessCriteria: []string{
					"Tests pass",
					"Code compiles",
				},
			},
			output: "Task output",
			agents: []string{"quality-control", "golang-pro"},
			responses: map[string]*agent.InvocationResult{
				"quality-control": {
					Output:   `{"verdict":"GREEN","feedback":"All good","criteria_results":[{"index":0,"passed":true,"evidence":"Tests pass"},{"index":1,"passed":true,"evidence":"Compiles"}],"issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
					ExitCode: 0,
				},
				"golang-pro": {
					Output:   `{"verdict":"RED","feedback":"Compilation issues","criteria_results":[{"index":0,"passed":true,"evidence":"Tests pass"},{"index":1,"passed":false,"fail_reason":"Does not compile"}],"issues":[],"recommendations":[],"should_retry":true,"suggested_agent":""}`,
					ExitCode: 0,
				},
			},
			wantFlag:     models.StatusRed,
			wantStrategy: "multi-agent-criteria-consensus",
		},
		{
			name: "legacy task without criteria uses strictest-wins",
			task: models.Task{
				Number: "3",
				Name:   "Legacy task",
				Prompt: "Do something",
				// No SuccessCriteria
			},
			output: "Task output",
			agents: []string{"quality-control", "golang-pro"},
			responses: map[string]*agent.InvocationResult{
				"quality-control": {
					Output:   `{"verdict":"GREEN","feedback":"All good","issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
					ExitCode: 0,
				},
				"golang-pro": {
					Output:   `{"verdict":"YELLOW","feedback":"Minor issues","issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
					ExitCode: 0,
				},
			},
			wantFlag:     models.StatusYellow,
			wantStrategy: "strictest-wins",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := &mockQCLogger{}
			mock := &mockInvoker{
				mockInvoke: func(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
					return tt.responses[task.Agent], nil
				},
			}

			qc := NewQualityController(mock)
			qc.Logger = mockLogger
			qc.AgentConfig = models.QCAgentConfig{
				Mode:         "explicit",
				ExplicitList: tt.agents,
			}

			ctx := context.Background()
			result, err := qc.ReviewMultiAgent(ctx, tt.task, tt.output)

			if err != nil {
				t.Errorf("ReviewMultiAgent() error = %v", err)
				return
			}

			if result.Flag != tt.wantFlag {
				t.Errorf("ReviewMultiAgent() Flag = %q, want %q", result.Flag, tt.wantFlag)
			}

			// Verify logging strategy
			if len(mockLogger.aggregatedResultCalls) > 0 {
				strategy := mockLogger.aggregatedResultCalls[0].strategy
				if strategy != tt.wantStrategy {
					t.Errorf("ReviewMultiAgent() strategy = %q, want %q", strategy, tt.wantStrategy)
				}
			}

			// Verify agent name includes multi-agent prefix
			if !strings.Contains(result.AgentName, "multi-agent") {
				t.Errorf("ReviewMultiAgent() AgentName should include multi-agent prefix, got %q", result.AgentName)
			}
		})
	}
}

func TestReviewMultiAgent_WithoutCriteria_BackwardCompat(t *testing.T) {
	// Verify backward compatibility: tasks without SuccessCriteria use strictest-wins
	mock := &mockInvoker{
		mockInvoke: func(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
			if task.Agent == "quality-control" {
				return &agent.InvocationResult{
					Output:   `{"verdict":"GREEN","feedback":"Good","issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
					ExitCode: 0,
				}, nil
			}
			return &agent.InvocationResult{
				Output:   `{"verdict":"RED","feedback":"Bad","issues":[],"recommendations":[],"should_retry":true,"suggested_agent":""}`,
				ExitCode: 0,
			}, nil
		},
	}

	mockLogger := &mockQCLogger{}
	qc := NewQualityController(mock)
	qc.Logger = mockLogger
	qc.AgentConfig = models.QCAgentConfig{
		Mode:         "explicit",
		ExplicitList: []string{"quality-control", "golang-pro"},
	}

	task := models.Task{
		Number: "1",
		Name:   "Legacy task",
		Prompt: "Do something",
		// NO SuccessCriteria
	}

	ctx := context.Background()
	result, err := qc.ReviewMultiAgent(ctx, task, "output")

	if err != nil {
		t.Fatalf("ReviewMultiAgent() error = %v", err)
	}

	// Should be RED (strictest-wins: RED > GREEN)
	if result.Flag != models.StatusRed {
		t.Errorf("ReviewMultiAgent() Flag = %q, want RED", result.Flag)
	}

	// Should log strictest-wins strategy
	if len(mockLogger.aggregatedResultCalls) > 0 {
		strategy := mockLogger.aggregatedResultCalls[0].strategy
		if strategy != "strictest-wins" {
			t.Errorf("ReviewMultiAgent() should use strictest-wins for legacy tasks, got %q", strategy)
		}
	}
}

func TestReviewMultiAgent_CriteriaConsensusWithThreeAgents(t *testing.T) {
	// Test unanimous consensus with three agents
	task := models.Task{
		Number: "1",
		Name:   "Three agent consensus",
		Prompt: "Implement feature",
		SuccessCriteria: []string{
			"Criterion A",
			"Criterion B",
			"Criterion C",
		},
	}

	responses := map[string]*agent.InvocationResult{
		"agent1": {
			Output:   `{"verdict":"GREEN","feedback":"All pass","criteria_results":[{"index":0,"passed":true},{"index":1,"passed":true},{"index":2,"passed":true}],"issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
			ExitCode: 0,
		},
		"agent2": {
			Output:   `{"verdict":"GREEN","feedback":"All pass","criteria_results":[{"index":0,"passed":true},{"index":1,"passed":true},{"index":2,"passed":true}],"issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
			ExitCode: 0,
		},
		"agent3": {
			Output:   `{"verdict":"GREEN","feedback":"All pass","criteria_results":[{"index":0,"passed":true},{"index":1,"passed":true},{"index":2,"passed":true}],"issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
			ExitCode: 0,
		},
	}

	mock := &mockInvoker{
		mockInvoke: func(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
			return responses[task.Agent], nil
		},
	}

	qc := NewQualityController(mock)
	qc.AgentConfig = models.QCAgentConfig{
		Mode:         "explicit",
		ExplicitList: []string{"agent1", "agent2", "agent3"},
	}

	ctx := context.Background()
	result, err := qc.ReviewMultiAgent(ctx, task, "output")

	if err != nil {
		t.Fatalf("ReviewMultiAgent() error = %v", err)
	}

	if result.Flag != models.StatusGreen {
		t.Errorf("ReviewMultiAgent() Flag = %q, want GREEN (unanimous consensus)", result.Flag)
	}
}

func TestReviewMultiAgent_CriteriaPartialDisagreement(t *testing.T) {
	// Test when agents disagree on some criteria (not all unanimous)
	task := models.Task{
		Number: "1",
		Name:   "Partial disagreement",
		Prompt: "Implement feature",
		SuccessCriteria: []string{
			"Tests pass",
			"Compiles",
		},
	}

	responses := map[string]*agent.InvocationResult{
		"agent1": {
			Output:   `{"verdict":"GREEN","feedback":"Both pass","criteria_results":[{"index":0,"passed":true},{"index":1,"passed":true}],"issues":[],"recommendations":[],"should_retry":false,"suggested_agent":""}`,
			ExitCode: 0,
		},
		"agent2": {
			Output:   `{"verdict":"RED","feedback":"Tests fail","criteria_results":[{"index":0,"passed":false},{"index":1,"passed":true}],"issues":[],"recommendations":[],"should_retry":true,"suggested_agent":""}`,
			ExitCode: 0,
		},
	}

	mock := &mockInvoker{
		mockInvoke: func(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
			return responses[task.Agent], nil
		},
	}

	qc := NewQualityController(mock)
	qc.AgentConfig = models.QCAgentConfig{
		Mode:         "explicit",
		ExplicitList: []string{"agent1", "agent2"},
	}

	ctx := context.Background()
	result, err := qc.ReviewMultiAgent(ctx, task, "output")

	if err != nil {
		t.Fatalf("ReviewMultiAgent() error = %v", err)
	}

	// Should be RED because agent2 says tests fail, not unanimous
	if result.Flag != models.StatusRed {
		t.Errorf("ReviewMultiAgent() Flag = %q, want RED (not unanimous on tests)", result.Flag)
	}
}

func TestReview_StructuredPromptContainsRequiredSections(t *testing.T) {
	var capturedPrompt string
	mock := &mockInvoker{
		mockInvoke: func(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
			capturedPrompt = task.Prompt
			return &agent.InvocationResult{
				Output:   `{"verdict":"GREEN","feedback":"OK","criteria_results":[{"index":0,"passed":true}]}`,
				ExitCode: 0,
				Duration: 100 * time.Millisecond,
			}, nil
		},
	}

	qc := NewQualityController(mock)
	// Disable multi-agent mode to test single-agent path
	qc.AgentConfig.Mode = ""
	inputTask := models.Task{
		Number: "1",
		Name:   "Structured Review",
		Prompt: "Build the feature",
		SuccessCriteria: []string{
			"Feature works correctly",
		},
		TestCommands: []string{
			"go test ./...",
		},
	}

	ctx := context.Background()
	_, err := qc.Review(ctx, inputTask, "agent output here")
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	// Verify structured prompt sections (they are combined with JSON instructions)
	if !contains(capturedPrompt, "# Quality Control Review: Structured Review") {
		t.Errorf("prompt missing task name header, got: %s", capturedPrompt[:min(500, len(capturedPrompt))])
	}
	if !contains(capturedPrompt, "SUCCESS CRITERIA - VERIFY EACH ONE") {
		t.Error("prompt missing success criteria section")
	}
	if !contains(capturedPrompt, "0. [ ] Feature works correctly") {
		t.Error("prompt missing numbered criterion")
	}
	if !contains(capturedPrompt, "TEST COMMANDS") {
		t.Error("prompt missing test commands section")
	}
	if !contains(capturedPrompt, "go test ./...") {
		t.Error("prompt missing test command")
	}
	if !contains(capturedPrompt, "agent output here") {
		t.Error("prompt missing agent output")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

// TestInvokeAndParseQCAgent_FailsOnPersistentInvalidJSON tests that invalid JSON responses fail immediately
// With --json-schema flag enforcement, invalid JSON should not reach this point, but we test failure handling.
func TestInvokeAndParseQCAgent_FailsOnPersistentInvalidJSON(t *testing.T) {
	callCount := 0
	mockInv := &mockInvoker{
		mockInvoke: func(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
			callCount++
			// Always return invalid JSON (never fixes the schema)
			return &agent.InvocationResult{
				Output:   `{"metadata": {"verdict": "GREEN"}}`,
				Duration: 100 * time.Millisecond,
			}, nil
		},
	}

	qc := &QualityController{
		Invoker: mockInv,
	}

	task := models.Task{
		Number: "1",
		Name:   "Test task",
		Prompt: "original prompt",
	}

	// Test that invokeAndParseQCAgent fails on invalid JSON (single attempt with schema enforcement)
	resp, err := qc.invokeAndParseQCAgent(context.Background(), task, "test-agent")

	if err == nil {
		t.Error("Expected error on persistent invalid JSON, got nil")
	}

	if callCount == 0 {
		t.Error("Expected at least one invocation attempt")
	}

	if resp != nil {
		t.Error("Expected nil response on error, got non-nil response")
	}
}

// TestInvokeAndParseQCAgent_SucceedsWithValidJSONFirstAttempt tests happy path
// When JSON is valid on first attempt, no retry occurs.
func TestInvokeAndParseQCAgent_SucceedsWithValidJSONFirstAttempt(t *testing.T) {
	callCount := 0
	mockInv := &mockInvoker{
		mockInvoke: func(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
			callCount++
			// Return valid JSON immediately
			return &agent.InvocationResult{
				Output:   `{"verdict":"GREEN","feedback":"excellent work","issues":[],"recommendations":["add tests"],"should_retry":false,"suggested_agent":""}`,
				Duration: 100 * time.Millisecond,
			}, nil
		},
	}

	qc := &QualityController{
		Invoker: mockInv,
	}

	task := models.Task{
		Number: "1",
		Name:   "Test task",
		Prompt: "original prompt",
	}

	resp, err := qc.invokeAndParseQCAgent(context.Background(), task, "test-agent")

	if err != nil {
		t.Fatalf("Expected success on valid JSON, got error: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 invocation (no retry), got %d", callCount)
	}

	if resp.Verdict != "GREEN" {
		t.Errorf("Expected verdict GREEN, got %s", resp.Verdict)
	}

	if resp.Feedback != "excellent work" {
		t.Errorf("Expected feedback 'excellent work', got %q", resp.Feedback)
	}

	if len(resp.Recommendations) != 1 || resp.Recommendations[0] != "add tests" {
		t.Errorf("Expected recommendation 'add tests', got %v", resp.Recommendations)
	}
}

// TestInvokeAndParseQCAgent_HandlesContextCancellation tests cancellation handling
// When context is cancelled during invocation, the system should propagate the error.
func TestInvokeAndParseQCAgent_HandlesContextCancellation(t *testing.T) {
	mockInv := &mockInvoker{
		mockInvoke: func(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
			// Simulate context cancellation
			return nil, context.Canceled
		},
	}

	qc := &QualityController{
		Invoker: mockInv,
	}

	task := models.Task{
		Number: "1",
		Name:   "Test task",
		Prompt: "original prompt",
	}

	resp, err := qc.invokeAndParseQCAgent(context.Background(), task, "test-agent")

	if err == nil {
		t.Error("Expected error on context cancellation, got nil")
	}

	if resp != nil {
		t.Error("Expected nil response on context cancellation, got non-nil response")
	}
}

// TestBuildStructuredReviewPrompt_DualCriteria tests structured review prompts with both
// success criteria and integration criteria for integration tasks.
// Verifies that criteria are properly indexed sequentially and sections are clearly
// separated with appropriate headers.
func TestBuildStructuredReviewPrompt_DualCriteria(t *testing.T) {
	tests := []struct {
		name                    string
		taskType                string
		successCriteria         []string
		integrationCriteria     []string
		wantSuccessCriteria     bool
		wantIntegrationCriteria bool
		wantExactCriteria       []struct {
			index int
			text  string
		}
	}{
		{
			name:     "success and integration criteria both present",
			taskType: "integration",
			successCriteria: []string{
				"Component works",
				"Unit tests pass",
			},
			integrationCriteria: []string{
				"Integrates properly",
				"No regression",
			},
			wantSuccessCriteria:     true,
			wantIntegrationCriteria: true,
			wantExactCriteria: []struct {
				index int
				text  string
			}{
				{0, "Component works"},
				{1, "Unit tests pass"},
				{2, "Integrates properly"},
				{3, "No regression"},
			},
		},
		{
			name:                    "only success criteria (non-integration task)",
			taskType:                "regular",
			successCriteria:         []string{"Feature implemented"},
			integrationCriteria:     []string{}, // No integration criteria for regular tasks
			wantSuccessCriteria:     true,
			wantIntegrationCriteria: false,
			wantExactCriteria: []struct {
				index int
				text  string
			}{
				{0, "Feature implemented"},
			},
		},
		{
			name:                    "only integration criteria",
			taskType:                "integration",
			successCriteria:         []string{},
			integrationCriteria:     []string{"Services communicate"},
			wantSuccessCriteria:     true, // SUCCESS CRITERIA header still shown
			wantIntegrationCriteria: true,
			wantExactCriteria: []struct {
				index int
				text  string
			}{
				{0, "Services communicate"},
			},
		},
		{
			name:                    "multiple criteria with special characters",
			taskType:                "integration",
			successCriteria:         []string{"API returns HTTP 200"},
			integrationCriteria:     []string{"Response contains required fields: id, name, email"},
			wantSuccessCriteria:     true,
			wantIntegrationCriteria: true,
			wantExactCriteria: []struct {
				index int
				text  string
			}{
				{0, "API returns HTTP 200"},
				{1, "Response contains required fields: id, name, email"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qc := NewQualityController(nil)
			task := models.Task{
				Number:              "5",
				Name:                "Test Integration Task",
				Prompt:              "Implement integration",
				Type:                tt.taskType,
				SuccessCriteria:     tt.successCriteria,
				IntegrationCriteria: tt.integrationCriteria,
				Files:               []string{"internal/integration/service.go"},
			}

			ctx := context.Background()
			prompt := qc.BuildStructuredReviewPrompt(ctx, task, "agent output")

			// Verify SUCCESS CRITERIA section is present if there are any criteria
			if (tt.successCriteria != nil && len(tt.successCriteria) > 0) || (tt.integrationCriteria != nil && len(tt.integrationCriteria) > 0) {
				if !contains(prompt, "## SUCCESS CRITERIA") {
					t.Error("SUCCESS CRITERIA section header missing")
				}
			}

			// Verify INTEGRATION CRITERIA section presence
			if tt.wantIntegrationCriteria && tt.taskType == "integration" && len(tt.integrationCriteria) > 0 {
				if !contains(prompt, "## INTEGRATION CRITERIA") {
					t.Error("INTEGRATION CRITERIA section header missing")
				}
			} else if tt.wantIntegrationCriteria && tt.taskType == "integration" && len(tt.integrationCriteria) == 0 {
				// Integration type but no integration criteria - should NOT have the header
				if contains(prompt, "## INTEGRATION CRITERIA - VERIFY EACH ONE") {
					t.Error("INTEGRATION CRITERIA section header should not be present when no integration criteria")
				}
			}

			// Verify exact criterion indices and text
			for _, criterion := range tt.wantExactCriteria {
				expectedLine := fmt.Sprintf("%d. [ ] %s", criterion.index, criterion.text)
				if !contains(prompt, expectedLine) {
					t.Errorf("Criterion not found: %q\nFull prompt:\n%s", expectedLine, prompt)
				}
			}

			// Verify criteria are numbered sequentially
			allCriteria := append(tt.successCriteria, tt.integrationCriteria...)
			if len(allCriteria) > 0 {
				for i, criterion := range allCriteria {
					if !contains(prompt, fmt.Sprintf("%d. [ ] %s", i, criterion)) {
						t.Errorf("Criterion %d (%q) not found with correct index", i, criterion)
					}
				}
			}

			// Verify task name is in the prompt
			if !contains(prompt, task.Name) {
				t.Error("Task name not found in prompt")
			}

			// Verify agent output section
			if !contains(prompt, "## AGENT OUTPUT") {
				t.Error("AGENT OUTPUT section missing")
			}
			if !contains(prompt, "agent output") {
				t.Error("Agent output content missing")
			}
		})
	}
}

// TestBuildStructuredReviewPrompt_DualCriteria_Ordering verifies that success criteria
// and integration criteria are properly ordered with correct sequential indices.
func TestBuildStructuredReviewPrompt_DualCriteria_Ordering(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		Number: "7",
		Name:   "Complex Integration",
		Prompt: "Implement complex integration system",
		Type:   "integration",
		SuccessCriteria: []string{
			"Database schema created",
			"Migrations work forward and backward",
			"Indexes created on foreign keys",
		},
		IntegrationCriteria: []string{
			"API server starts without errors",
			"Connections to database established",
			"Health check endpoint responds",
			"Metrics exported correctly",
		},
		Files: []string{
			"internal/db/schema.go",
			"internal/api/server.go",
			"internal/metrics/exporter.go",
		},
	}

	ctx := context.Background()
	prompt := qc.BuildStructuredReviewPrompt(ctx, task, "implementation output")

	// Verify SUCCESS CRITERIA section
	if !contains(prompt, "## SUCCESS CRITERIA") {
		t.Error("SUCCESS CRITERIA header missing")
	}

	// Verify INTEGRATION CRITERIA section
	if !contains(prompt, "## INTEGRATION CRITERIA") {
		t.Error("INTEGRATION CRITERIA header missing")
	}

	// Verify success criteria indices (0-2)
	successTests := []struct {
		index int
		text  string
	}{
		{0, "Database schema created"},
		{1, "Migrations work forward and backward"},
		{2, "Indexes created on foreign keys"},
	}

	for _, st := range successTests {
		expectedLine := fmt.Sprintf("%d. [ ] %s", st.index, st.text)
		if !contains(prompt, expectedLine) {
			t.Errorf("Success criterion %d not found: %q", st.index, expectedLine)
		}
	}

	// Verify integration criteria indices (3-6)
	integrationTests := []struct {
		index int
		text  string
	}{
		{3, "API server starts without errors"},
		{4, "Connections to database established"},
		{5, "Health check endpoint responds"},
		{6, "Metrics exported correctly"},
	}

	for _, it := range integrationTests {
		expectedLine := fmt.Sprintf("%d. [ ] %s", it.index, it.text)
		if !contains(prompt, expectedLine) {
			t.Errorf("Integration criterion %d not found: %q", it.index, expectedLine)
		}
	}

	// Verify SUCCESS CRITERIA comes before INTEGRATION CRITERIA
	successIdx := strings.Index(prompt, "## SUCCESS CRITERIA")
	integrationIdx := strings.Index(prompt, "## INTEGRATION CRITERIA")
	if successIdx < 0 {
		t.Fatal("SUCCESS CRITERIA section not found")
	}
	if integrationIdx < 0 {
		t.Fatal("INTEGRATION CRITERIA section not found")
	}
	if successIdx >= integrationIdx {
		t.Error("SUCCESS CRITERIA section should appear before INTEGRATION CRITERIA section")
	}
}

// TestBuildStructuredReviewPrompt_DualCriteria_NonIntegrationIgnoresIntegration
// verifies that integration criteria are not included for non-integration task types.
func TestBuildStructuredReviewPrompt_DualCriteria_NonIntegrationIgnoresIntegration(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		Number: "3",
		Name:   "Regular Task",
		Prompt: "Implement feature",
		Type:   "regular", // Not integration
		SuccessCriteria: []string{
			"Feature works",
			"Tests pass",
		},
		IntegrationCriteria: []string{
			"Should be ignored",
			"This should not appear",
		},
		Files: []string{"internal/feature/impl.go"},
	}

	ctx := context.Background()
	prompt := qc.BuildStructuredReviewPrompt(ctx, task, "output")

	// Verify SUCCESS CRITERIA is present
	if !contains(prompt, "## SUCCESS CRITERIA") {
		t.Error("SUCCESS CRITERIA header should be present")
	}

	// Verify success criteria are indexed
	if !contains(prompt, "0. [ ] Feature works") {
		t.Error("Success criterion 0 not found")
	}
	if !contains(prompt, "1. [ ] Tests pass") {
		t.Error("Success criterion 1 not found")
	}

	// Verify INTEGRATION CRITERIA section is NOT present
	if contains(prompt, "## INTEGRATION CRITERIA - VERIFY EACH ONE") {
		t.Error("INTEGRATION CRITERIA section should NOT be present for non-integration tasks")
	}

	// Verify integration criteria text does not appear
	if contains(prompt, "Should be ignored") {
		t.Error("Integration criteria should not be included in non-integration task prompts")
	}
	if contains(prompt, "This should not appear") {
		t.Error("Integration criteria should not be included in non-integration task prompts")
	}
}

// TestInjectBehaviorContext_WithProvider tests behavioral context injection with a provider
func TestInjectBehaviorContext_WithProvider(t *testing.T) {
	// Create mock metrics provider
	mockProvider := &mockBehavioralMetricsProvider{
		metrics: &behavioral.BehavioralMetrics{
			TotalSessions:   3,
			SuccessRate:     0.75,
			AverageDuration: 30 * time.Second,
			TotalCost:       0.25,
			ErrorRate:       0.1,
			ToolExecutions: []behavioral.ToolExecution{
				{Name: "Read", Count: 15, SuccessRate: 0.95, AvgDuration: 50 * time.Millisecond},
				{Name: "Bash", Count: 8, SuccessRate: 0.7, AvgDuration: 200 * time.Millisecond},
			},
			TokenUsage: behavioral.TokenUsage{
				InputTokens:  5000,
				OutputTokens: 2000,
				CostUSD:      0.25,
				ModelName:    "claude-sonnet-4-5",
			},
		},
	}

	var provider BehavioralMetricsProvider = mockProvider
	qc := NewQualityController(nil)
	qc.BehavioralMetrics = &provider

	task := models.Task{
		Number: "1",
		Name:   "Test task",
		Prompt: "Do something",
	}

	ctx := context.Background()
	prompt := qc.BuildReviewPrompt(ctx, task, "Task output")

	// Verify behavioral context is injected
	if !contains(prompt, "BEHAVIORAL ANALYSIS CONTEXT") {
		t.Error("BuildReviewPrompt() should include BEHAVIORAL ANALYSIS CONTEXT when provider is set")
	}
	if !contains(prompt, "Sessions**: 3") {
		t.Error("BuildReviewPrompt() should include session count")
	}
	if !contains(prompt, "Success Rate**: 75.0%") {
		t.Error("BuildReviewPrompt() should include success rate")
	}
	if !contains(prompt, "Tool Usage") {
		t.Error("BuildReviewPrompt() should include tool usage section")
	}
	if !contains(prompt, "Cost Summary") {
		t.Error("BuildReviewPrompt() should include cost summary")
	}
}

// TestInjectBehaviorContext_WithoutProvider tests behavioral context injection without provider
func TestInjectBehaviorContext_WithoutProvider(t *testing.T) {
	qc := NewQualityController(nil)
	// BehavioralMetrics is nil by default

	task := models.Task{
		Number: "1",
		Name:   "Test task",
		Prompt: "Do something",
	}

	ctx := context.Background()
	prompt := qc.BuildReviewPrompt(ctx, task, "Task output")

	// Verify no behavioral context when provider is not set
	if contains(prompt, "BEHAVIORAL ANALYSIS CONTEXT") {
		t.Error("BuildReviewPrompt() should not include BEHAVIORAL ANALYSIS CONTEXT when provider is nil")
	}
}

// TestInjectBehaviorContext_StructuredPrompt tests behavioral context in structured prompts
func TestInjectBehaviorContext_StructuredPrompt(t *testing.T) {
	mockProvider := &mockBehavioralMetricsProvider{
		metrics: &behavioral.BehavioralMetrics{
			TotalSessions:   5,
			SuccessRate:     0.8,
			AverageDuration: 1 * time.Minute,
			TotalCost:       1.5,  // High cost - should trigger anomaly
			ErrorRate:       0.25, // High error rate - should trigger anomaly
			BashCommands: []behavioral.BashCommand{
				{Command: "cmd1", Success: false},
				{Command: "cmd2", Success: false},
				{Command: "cmd3", Success: false},
				{Command: "cmd4", Success: false},
			},
		},
	}

	var provider BehavioralMetricsProvider = mockProvider
	qc := NewQualityController(nil)
	qc.BehavioralMetrics = &provider

	task := models.Task{
		Number: "2",
		Name:   "Task with criteria",
		Prompt: "Do something",
		SuccessCriteria: []string{
			"Criterion 1",
			"Criterion 2",
		},
	}

	ctx := context.Background()
	prompt := qc.BuildStructuredReviewPrompt(ctx, task, "Task output")

	// Verify behavioral context is in structured prompt
	if !contains(prompt, "BEHAVIORAL ANALYSIS CONTEXT") {
		t.Error("BuildStructuredReviewPrompt() should include BEHAVIORAL ANALYSIS CONTEXT")
	}
	// Verify anomalies are detected and shown
	if !contains(prompt, "Anomalies Detected") {
		t.Error("BuildStructuredReviewPrompt() should include anomalies section when there are anomalies")
	}
	if !contains(prompt, "High cost") {
		t.Error("BuildStructuredReviewPrompt() should flag high cost anomaly")
	}
	if !contains(prompt, "High error rate") {
		t.Error("BuildStructuredReviewPrompt() should flag high error rate anomaly")
	}
	if !contains(prompt, "bash failures") {
		t.Error("BuildStructuredReviewPrompt() should flag multiple bash failures")
	}
	// Verify consideration note
	if !contains(prompt, "Consider these behavioral patterns") {
		t.Error("BuildStructuredReviewPrompt() should include behavioral consideration note")
	}
}

// TestInjectBehaviorContext_NilMetrics tests handling when provider returns nil metrics
func TestInjectBehaviorContext_NilMetrics(t *testing.T) {
	mockProvider := &mockBehavioralMetricsProvider{
		metrics: nil, // Provider returns nil
	}

	var provider BehavioralMetricsProvider = mockProvider
	qc := NewQualityController(nil)
	qc.BehavioralMetrics = &provider

	task := models.Task{
		Number: "1",
		Name:   "Test task",
		Prompt: "Do something",
	}

	ctx := context.Background()
	prompt := qc.BuildReviewPrompt(ctx, task, "Task output")

	// Verify no behavioral context when metrics are nil
	if contains(prompt, "BEHAVIORAL ANALYSIS CONTEXT") {
		t.Error("BuildReviewPrompt() should not include BEHAVIORAL ANALYSIS CONTEXT when metrics are nil")
	}
}

// mockBehavioralMetricsProvider is a test mock for BehavioralMetricsProvider
type mockBehavioralMetricsProvider struct {
	metrics *behavioral.BehavioralMetrics
}

func (m *mockBehavioralMetricsProvider) GetMetrics(ctx context.Context, taskNumber string) *behavioral.BehavioralMetrics {
	return m.metrics
}

// TestBuildStructuredReviewPrompt_KeyPointsSection verifies key_points section is included
// when task has key_points.
func TestBuildStructuredReviewPrompt_KeyPointsSection(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		Number: "1",
		Name:   "Task with key points",
		Prompt: "Implement feature",
		SuccessCriteria: []string{
			"Feature works",
			"Tests pass",
			"Code is clean",
		},
		KeyPoints: []models.KeyPoint{
			{Point: "Use dependency injection"},
			{Point: "Follow naming conventions", Details: "Use camelCase for variables"},
			{Point: "Add error handling", Details: "Wrap errors with context", Reference: "internal/errors/errors.go"},
		},
	}

	ctx := context.Background()
	prompt := qc.BuildStructuredReviewPrompt(ctx, task, "output")

	// Verify KEY POINTS section is present
	if !contains(prompt, "## KEY POINTS (GUIDANCE - NOT SCORED)") {
		t.Error("KEY POINTS section should be present when task has key_points")
	}

	// Verify key point content
	if !contains(prompt, "Use dependency injection") {
		t.Error("First key point should be included")
	}
	if !contains(prompt, "Follow naming conventions: Use camelCase for variables") {
		t.Error("Second key point with details should be included")
	}
	if !contains(prompt, "Add error handling: Wrap errors with context (ref: internal/errors/errors.go)") {
		t.Error("Third key point with details and reference should be included")
	}

	// Verify guidance text is present (not the warning since criteria >= key_points)
	if !contains(prompt, "These implementation key points are provided for context") {
		t.Error("Standard guidance text should be present when criteria >= key_points")
	}
}

// TestBuildStructuredReviewPrompt_KeyPointsWarning verifies warning appears when
// key_points count exceeds criteria count.
func TestBuildStructuredReviewPrompt_KeyPointsWarning(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		Number: "2",
		Name:   "Task with more key points than criteria",
		Prompt: "Implement feature",
		SuccessCriteria: []string{
			"Feature works",
		},
		KeyPoints: []models.KeyPoint{
			{Point: "Key point 1"},
			{Point: "Key point 2"},
			{Point: "Key point 3"},
			{Point: "Key point 4"},
		},
	}

	ctx := context.Background()
	prompt := qc.BuildStructuredReviewPrompt(ctx, task, "output")

	// Verify KEY POINTS section is present
	if !contains(prompt, "## KEY POINTS (GUIDANCE - NOT SCORED)") {
		t.Error("KEY POINTS section should be present")
	}

	// Verify warning is shown (4 key_points - 1 criteria = 3 without explicit criteria)
	if !contains(prompt, "3 key point(s) do not have explicit success criteria") {
		t.Error("Warning about key points without explicit criteria should be present")
	}

	// Verify all key points are still included
	if !contains(prompt, "Key point 1") {
		t.Error("Key point 1 should be included")
	}
	if !contains(prompt, "Key point 4") {
		t.Error("Key point 4 should be included")
	}
}

// TestBuildStructuredReviewPrompt_NoKeyPoints verifies KEY POINTS section is NOT included
// when task has no key_points.
func TestBuildStructuredReviewPrompt_NoKeyPoints(t *testing.T) {
	tests := []struct {
		name      string
		keyPoints []models.KeyPoint
	}{
		{
			name:      "nil key_points",
			keyPoints: nil,
		},
		{
			name:      "empty key_points slice",
			keyPoints: []models.KeyPoint{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qc := NewQualityController(nil)
			task := models.Task{
				Number: "3",
				Name:   "Task without key points",
				Prompt: "Implement feature",
				SuccessCriteria: []string{
					"Feature works",
					"Tests pass",
				},
				KeyPoints: tt.keyPoints,
			}

			ctx := context.Background()
			prompt := qc.BuildStructuredReviewPrompt(ctx, task, "output")

			// Verify KEY POINTS section is NOT present
			if contains(prompt, "## KEY POINTS") {
				t.Error("KEY POINTS section should NOT be present when task has no key_points")
			}
			if contains(prompt, "GUIDANCE - NOT SCORED") {
				t.Error("KEY POINTS header should NOT be present when task has no key_points")
			}
		})
	}
}

// TestBuildStructuredReviewPrompt_KeyPointsWithIntegrationTask verifies key_points work
// correctly with integration tasks that have both success and integration criteria.
func TestBuildStructuredReviewPrompt_KeyPointsWithIntegrationTask(t *testing.T) {
	qc := NewQualityController(nil)
	task := models.Task{
		Number: "4",
		Name:   "Integration task with key points",
		Prompt: "Wire components together",
		Type:   "integration",
		SuccessCriteria: []string{
			"Component A works",
			"Component B works",
		},
		IntegrationCriteria: []string{
			"A and B integrate properly",
			"Data flows correctly",
		},
		KeyPoints: []models.KeyPoint{
			{Point: "Check interface compatibility"},
			{Point: "Verify error propagation"},
		},
	}

	ctx := context.Background()
	prompt := qc.BuildStructuredReviewPrompt(ctx, task, "output")

	// Verify KEY POINTS section is present
	if !contains(prompt, "## KEY POINTS (GUIDANCE - NOT SCORED)") {
		t.Error("KEY POINTS section should be present for integration task")
	}

	// With 4 total criteria (2 success + 2 integration) and 2 key points,
	// no warning should appear
	if contains(prompt, "do not have explicit success criteria") {
		t.Error("Warning should NOT appear when total criteria >= key_points count")
	}

	// Verify key points are included
	if !contains(prompt, "Check interface compatibility") {
		t.Error("First key point should be included")
	}
	if !contains(prompt, "Verify error propagation") {
		t.Error("Second key point should be included")
	}
}

// TestFormatDetectedErrors tests the formatting of error classifications
func TestFormatDetectedErrors(t *testing.T) {
	tests := []struct {
		name         string
		errors       []*DetectedError
		wantContains []string
		wantEmpty    bool
	}{
		{
			name:      "nil errors",
			errors:    nil,
			wantEmpty: true,
		},
		{
			name:      "empty errors",
			errors:    []*DetectedError{},
			wantEmpty: true,
		},
		{
			name: "single regex error",
			errors: []*DetectedError{
				{
					Pattern: &ErrorPattern{
						Category:                  CODE_LEVEL,
						Suggestion:                "Fix syntax error",
						AgentCanFix:               true,
						RequiresHumanIntervention: false,
					},
					Method:     "regex",
					Confidence: 1.0,
				},
			},
			wantContains: []string{
				"## Error Classification Analysis",
				"Error 1: CODE_LEVEL",
				"**Method**: regex",
				"**Agent Can Fix**: true",
				"**Requires Human**: false",
				"**Suggestion**: Fix syntax error",
				"Use this context when reviewing code quality",
			},
		},
		{
			name: "claude error with confidence",
			errors: []*DetectedError{
				{
					Pattern: &ErrorPattern{
						Category:                  CODE_LEVEL,
						Suggestion:                "Install missing package",
						AgentCanFix:               false,
						RequiresHumanIntervention: true,
					},
					Method:     "claude",
					Confidence: 0.85,
				},
			},
			wantContains: []string{
				"**Method**: claude (confidence: 85%)",
				"**Agent Can Fix**: false",
				"**Requires Human**: true",
				"**Suggestion**: Install missing package",
			},
		},
		{
			name: "multiple errors",
			errors: []*DetectedError{
				{
					Pattern: &ErrorPattern{
						Category:                  CODE_LEVEL,
						Suggestion:                "Fix syntax",
						AgentCanFix:               true,
						RequiresHumanIntervention: false,
					},
					Method:     "regex",
					Confidence: 1.0,
				},
				{
					Pattern: &ErrorPattern{
						Category:                  CODE_LEVEL,
						Suggestion:                "Update test assertions",
						AgentCanFix:               true,
						RequiresHumanIntervention: false,
					},
					Method:     "claude",
					Confidence: 0.92,
				},
			},
			wantContains: []string{
				"Error 1: CODE_LEVEL",
				"Error 2: CODE_LEVEL",
				"confidence: 92%",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDetectedErrors(tt.errors)

			if tt.wantEmpty {
				if result != "" {
					t.Errorf("FormatDetectedErrors() expected empty string, got: %s", result)
				}
				return
			}

			for _, want := range tt.wantContains {
				if !contains(result, want) {
					t.Errorf("FormatDetectedErrors() missing expected content %q\nGot: %s", want, result)
				}
			}
		})
	}
}

// TestBuildStructuredReviewPrompt_WithErrorClassification tests error classification injection in QC prompt
func TestBuildStructuredReviewPrompt_WithErrorClassification(t *testing.T) {
	qc := NewQualityController(nil)

	t.Run("includes error classification when present", func(t *testing.T) {
		task := models.Task{
			Number: "1",
			Name:   "Task with detected errors",
			Prompt: "Implement feature",
			SuccessCriteria: []string{
				"Feature works",
				"Tests pass",
			},
			Metadata: map[string]interface{}{
				"detected_errors": []*DetectedError{
					{
						Pattern: &ErrorPattern{
							Category:                  CODE_LEVEL,
							Suggestion:                "Fix test assertions",
							AgentCanFix:               true,
							RequiresHumanIntervention: false,
						},
						Method:     "claude",
						Confidence: 0.9,
					},
				},
			},
		}

		ctx := context.Background()
		prompt := qc.BuildStructuredReviewPrompt(ctx, task, "output")

		// Should include error classification section
		if !contains(prompt, "## Error Classification Analysis") {
			t.Error("BuildStructuredReviewPrompt() missing error classification section")
		}
		if !contains(prompt, "Error 1: CODE_LEVEL") {
			t.Error("BuildStructuredReviewPrompt() missing error category")
		}
		if !contains(prompt, "confidence: 90%") {
			t.Error("BuildStructuredReviewPrompt() missing confidence percentage")
		}
		if !contains(prompt, "Fix test assertions") {
			t.Error("BuildStructuredReviewPrompt() missing suggestion")
		}
	})

	t.Run("no error section when metadata missing", func(t *testing.T) {
		task := models.Task{
			Number:          "2",
			Name:            "Task without errors",
			Prompt:          "Implement feature",
			SuccessCriteria: []string{"Feature works"},
			Metadata:        nil,
		}

		ctx := context.Background()
		prompt := qc.BuildStructuredReviewPrompt(ctx, task, "output")

		// Should NOT include error classification section
		if contains(prompt, "## Error Classification Analysis") {
			t.Error("BuildStructuredReviewPrompt() should not include error section when no metadata")
		}
	})

	t.Run("no error section when detected_errors empty", func(t *testing.T) {
		task := models.Task{
			Number: "3",
			Name:   "Task with empty detected errors",
			Prompt: "Implement feature",
			Metadata: map[string]interface{}{
				"detected_errors": []*DetectedError{},
			},
		}

		ctx := context.Background()
		prompt := qc.BuildStructuredReviewPrompt(ctx, task, "output")

		// Should NOT include error classification section
		if contains(prompt, "## Error Classification Analysis") {
			t.Error("BuildStructuredReviewPrompt() should not include error section when detected_errors empty")
		}
	})
}

func (m *mockLogger) SetGuardVerbose(verbose bool) {}
