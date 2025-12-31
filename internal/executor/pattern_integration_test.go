package executor

import (
	"context"
	"strings"
	"testing"

	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/models"
	"github.com/harrison/conductor/internal/pattern"
)

// mockPatternIntelligence is a mock implementation of pattern.PatternIntelligence for testing
type mockPatternIntelligence struct {
	stopResult       *pattern.STOPResult
	duplicateResult  *pattern.DuplicateResult
	checkTaskError   error
	recordSuccessErr error
	recordedTasks    []models.Task
}

func (m *mockPatternIntelligence) CheckTask(_ context.Context, task models.Task) (*pattern.STOPResult, *pattern.DuplicateResult, error) {
	return m.stopResult, m.duplicateResult, m.checkTaskError
}

func (m *mockPatternIntelligence) IsInitialized() bool {
	return true
}

func (m *mockPatternIntelligence) Initialize(_ context.Context) error {
	return nil
}

// mockPatternLogger is a mock logger for testing Pattern Intelligence logging
type mockPatternLogger struct {
	warnMessages []string
	infoMessages []string
}

func (m *mockPatternLogger) LogTestCommands(entries []models.TestCommandResult) {}
func (m *mockPatternLogger) LogCriterionVerifications(entries []models.CriterionVerificationResult) {
}
func (m *mockPatternLogger) LogDocTargetVerifications(entries []models.DocTargetResult) {}
func (m *mockPatternLogger) LogErrorPattern(pattern interface{})                        {}
func (m *mockPatternLogger) LogDetectedError(detected interface{})                      {}

func (m *mockPatternLogger) Warnf(format string, args ...interface{}) {
	m.warnMessages = append(m.warnMessages, format)
}

func (m *mockPatternLogger) Info(message string) {
	m.infoMessages = append(m.infoMessages, message)
}

func (m *mockPatternLogger) Infof(format string, args ...interface{}) {
	m.infoMessages = append(m.infoMessages, format)
}

func TestNewPatternIntelligenceHook(t *testing.T) {
	tests := []struct {
		name   string
		pi     pattern.PatternIntelligence
		cfg    *config.PatternConfig
		logger RuntimeEnforcementLogger
		want   bool // true if hook should be nil
	}{
		{
			name:   "nil pi returns nil",
			pi:     nil,
			cfg:    &config.PatternConfig{Enabled: true},
			logger: nil,
			want:   true,
		},
		{
			name:   "nil config returns nil",
			pi:     &mockPatternIntelligence{},
			cfg:    nil,
			logger: nil,
			want:   true,
		},
		{
			name:   "disabled config returns nil",
			pi:     &mockPatternIntelligence{},
			cfg:    &config.PatternConfig{Enabled: false},
			logger: nil,
			want:   true,
		},
		{
			name:   "enabled config returns hook",
			pi:     &mockPatternIntelligence{},
			cfg:    &config.PatternConfig{Enabled: true},
			logger: nil,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := NewPatternIntelligenceHook(tt.pi, tt.cfg, tt.logger)
			if (hook == nil) != tt.want {
				t.Errorf("NewPatternIntelligenceHook() = nil: %v, want nil: %v", hook == nil, tt.want)
			}
		})
	}
}

func TestPatternIntelligenceHook_CheckTask_NilHook(t *testing.T) {
	var hook *PatternIntelligenceHook
	task := models.Task{Number: "1", Name: "Test task"}

	result, err := hook.CheckTask(context.Background(), task)
	if err != nil {
		t.Errorf("CheckTask() error = %v, want nil", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ShouldBlock {
		t.Error("expected ShouldBlock=false for nil hook")
	}
}

func TestPatternIntelligenceHook_CheckTask_BlockMode(t *testing.T) {
	mockPI := &mockPatternIntelligence{
		duplicateResult: &pattern.DuplicateResult{
			IsDuplicate:     true,
			SimilarityScore: 0.95,
			DuplicateOf: []pattern.DuplicateRef{
				{TaskName: "Similar task", SimilarityScore: 0.95},
			},
		},
	}
	mockLogger := &mockPatternLogger{}

	cfg := &config.PatternConfig{
		Enabled:            true,
		Mode:               config.PatternModeBlock,
		DuplicateThreshold: 0.9,
	}

	hook := NewPatternIntelligenceHook(mockPI, cfg, mockLogger)
	task := models.Task{Number: "1", Name: "Test task"}

	result, err := hook.CheckTask(context.Background(), task)
	if err != nil {
		t.Errorf("CheckTask() error = %v, want nil", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.ShouldBlock {
		t.Error("expected ShouldBlock=true in block mode with duplicate")
	}
	if result.BlockReason == "" {
		t.Error("expected BlockReason to be set")
	}
	if len(mockLogger.warnMessages) == 0 {
		t.Error("expected warning log for blocked task")
	}
}

func TestPatternIntelligenceHook_CheckTask_WarnMode(t *testing.T) {
	mockPI := &mockPatternIntelligence{
		duplicateResult: &pattern.DuplicateResult{
			IsDuplicate:     true,
			SimilarityScore: 0.85,
		},
	}
	mockLogger := &mockPatternLogger{}

	cfg := &config.PatternConfig{
		Enabled:            true,
		Mode:               config.PatternModeWarn,
		DuplicateThreshold: 0.9,
		InjectIntoPrompt:   true,
	}

	hook := NewPatternIntelligenceHook(mockPI, cfg, mockLogger)
	task := models.Task{Number: "1", Name: "Test task"}

	result, err := hook.CheckTask(context.Background(), task)
	if err != nil {
		t.Errorf("CheckTask() error = %v, want nil", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ShouldBlock {
		t.Error("expected ShouldBlock=false in warn mode")
	}
	if len(mockLogger.warnMessages) == 0 {
		t.Error("expected warning log for duplicate in warn mode")
	}
}

func TestPatternIntelligenceHook_CheckTask_SuggestMode(t *testing.T) {
	mockPI := &mockPatternIntelligence{
		stopResult: &pattern.STOPResult{
			Confidence:      0.8,
			Recommendations: []string{"Consider using existing pattern"},
		},
	}
	mockLogger := &mockPatternLogger{}

	cfg := &config.PatternConfig{
		Enabled:          true,
		Mode:             config.PatternModeSuggest,
		InjectIntoPrompt: true,
		MinConfidence:    0.7,
	}

	hook := NewPatternIntelligenceHook(mockPI, cfg, mockLogger)
	task := models.Task{Number: "1", Name: "Test task"}

	result, err := hook.CheckTask(context.Background(), task)
	if err != nil {
		t.Errorf("CheckTask() error = %v, want nil", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ShouldBlock {
		t.Error("expected ShouldBlock=false in suggest mode")
	}
	if len(mockLogger.infoMessages) == 0 && len(result.Recommendations) > 0 {
		// Info log is optional, only when there are recommendations
	}
}

func TestPatternIntelligenceHook_CheckTask_GracefulDegradation(t *testing.T) {
	mockPI := &mockPatternIntelligence{
		checkTaskError: context.DeadlineExceeded,
	}
	mockLogger := &mockPatternLogger{}

	cfg := &config.PatternConfig{
		Enabled: true,
		Mode:    config.PatternModeBlock,
	}

	hook := NewPatternIntelligenceHook(mockPI, cfg, mockLogger)
	task := models.Task{Number: "1", Name: "Test task"}

	result, err := hook.CheckTask(context.Background(), task)
	// Should not return error - graceful degradation
	if err != nil {
		t.Errorf("CheckTask() error = %v, want nil (graceful degradation)", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ShouldBlock {
		t.Error("expected ShouldBlock=false on error (graceful degradation)")
	}
	if len(mockLogger.warnMessages) == 0 {
		t.Error("expected warning log for failed check")
	}
}

func TestPatternIntelligenceHook_BuildPromptInjection(t *testing.T) {
	cfg := &config.PatternConfig{
		Enabled:            true,
		Mode:               config.PatternModeSuggest,
		InjectIntoPrompt:   true,
		MinConfidence:      0.7,
		MaxPatternsPerTask: 3,
		MaxRelatedFiles:    5,
	}

	hook := &PatternIntelligenceHook{
		config: cfg,
	}

	t.Run("empty results returns empty string", func(t *testing.T) {
		result := hook.buildPromptInjection(nil, nil)
		if result != "" {
			t.Errorf("expected empty string for nil inputs, got %q", result)
		}
	})

	t.Run("builds injection from STOP result", func(t *testing.T) {
		stopResult := &pattern.STOPResult{
			Confidence: 0.8,
			Search: pattern.SearchResult{
				SimilarPatterns: []pattern.PatternMatch{
					{Name: "Pattern1", FilePath: "file1.go", Similarity: 0.9, Description: "desc1"},
				},
				RelatedFiles: []string{"related1.go", "related2.go"},
			},
			Think: pattern.ThinkResult{
				ComplexityScore:     7,
				EstimatedEffort:     "Medium",
				ApproachSuggestions: []string{"Use existing pattern"},
				RiskFactors: []pattern.RiskFactor{
					{Name: "API changes", Severity: "Medium", Mitigation: "Review docs"},
				},
			},
			Outline: pattern.OutlineResult{
				Steps: []pattern.OutlineStep{
					{Order: 1, Description: "Read existing code", TestStrategy: "Manual review"},
				},
			},
			Prove: pattern.ProveResult{
				TestCommands: []string{"go test ./..."},
			},
			Recommendations: []string{"Check documentation"},
		}

		result := hook.buildPromptInjection(stopResult, nil)
		if result == "" {
			t.Error("expected non-empty result for valid STOP result")
		}
		if !strings.Contains(result, "<pattern_intelligence>") {
			t.Error("expected header in injection")
		}
		if !strings.Contains(result, "Pattern1") {
			t.Error("expected similar pattern in injection")
		}
		if !strings.Contains(result, `complexity="7"`) {
			t.Error("expected complexity in injection")
		}
	})

	t.Run("builds injection from duplicate result", func(t *testing.T) {
		dupResult := &pattern.DuplicateResult{
			IsDuplicate:     true,
			SimilarityScore: 0.85,
			DuplicateOf: []pattern.DuplicateRef{
				{TaskName: "Similar task", SimilarityScore: 0.85},
			},
			OverlapAreas: []string{"authentication", "validation"},
		}

		result := hook.buildPromptInjection(nil, dupResult)
		if result == "" {
			t.Error("expected non-empty result for duplicate result")
		}
		if !strings.Contains(result, "<duplicate_warning") {
			t.Error("expected duplicate warning in injection")
		}
		if !strings.Contains(result, "Similar task") {
			t.Error("expected duplicate reference in injection")
		}
	})

	t.Run("disabled injection returns empty string", func(t *testing.T) {
		hookDisabled := &PatternIntelligenceHook{
			config: &config.PatternConfig{
				Enabled:          true,
				InjectIntoPrompt: false,
			},
		}

		stopResult := &pattern.STOPResult{Confidence: 0.9}
		result := hookDisabled.buildPromptInjection(stopResult, nil)
		if result != "" {
			t.Errorf("expected empty string when injection disabled, got %q", result)
		}
	})
}

func TestPatternIntelligenceHook_RecordSuccess(t *testing.T) {
	t.Run("nil hook gracefully handles", func(t *testing.T) {
		var hook *PatternIntelligenceHook
		task := models.Task{Name: "test"}
		err := hook.RecordSuccess(context.Background(), task, "test-agent")
		if err != nil {
			t.Errorf("RecordSuccess() on nil hook should not error, got %v", err)
		}
	})

	t.Run("disabled hook gracefully handles", func(t *testing.T) {
		hook := &PatternIntelligenceHook{
			config: &config.PatternConfig{Enabled: false},
		}
		task := models.Task{Name: "test"}
		err := hook.RecordSuccess(context.Background(), task, "test-agent")
		if err != nil {
			t.Errorf("RecordSuccess() on disabled hook should not error, got %v", err)
		}
	})
}

func TestApplyPromptInjection(t *testing.T) {
	t.Run("nil result returns original task", func(t *testing.T) {
		task := models.Task{Number: "1", Prompt: "Original prompt"}
		result := ApplyPromptInjection(task, nil)
		if result.Prompt != "Original prompt" {
			t.Errorf("expected original prompt, got %q", result.Prompt)
		}
	})

	t.Run("empty injection returns original task", func(t *testing.T) {
		task := models.Task{Number: "1", Prompt: "Original prompt"}
		result := ApplyPromptInjection(task, &PreTaskCheckResult{PromptInjection: ""})
		if result.Prompt != "Original prompt" {
			t.Errorf("expected original prompt, got %q", result.Prompt)
		}
	})

	t.Run("applies injection to prompt", func(t *testing.T) {
		task := models.Task{Number: "1", Prompt: "Original prompt"}
		injection := "\n## STOP Context\nRecommendations here"
		result := ApplyPromptInjection(task, &PreTaskCheckResult{PromptInjection: injection})
		expectedPrompt := "Original prompt" + injection
		if result.Prompt != expectedPrompt {
			t.Errorf("expected %q, got %q", expectedPrompt, result.Prompt)
		}
	})
}

func TestPatternIntegration_PreTaskHook(t *testing.T) {
	// Integration test: verify pattern hook is called during task execution
	mockPI := &mockPatternIntelligence{
		stopResult: &pattern.STOPResult{
			Confidence:      0.8,
			Recommendations: []string{"Use existing pattern"},
		},
	}

	cfg := &config.PatternConfig{
		Enabled:          true,
		Mode:             config.PatternModeSuggest,
		InjectIntoPrompt: true,
		MinConfidence:    0.5,
	}

	hook := NewPatternIntelligenceHook(mockPI, cfg, nil)
	if hook == nil {
		t.Fatal("expected non-nil hook")
	}

	task := models.Task{
		Number: "1",
		Name:   "Test task",
		Files:  []string{"test.go"},
	}

	result, err := hook.CheckTask(context.Background(), task)
	if err != nil {
		t.Fatalf("CheckTask() error = %v", err)
	}

	if result.STOPResult == nil {
		t.Error("expected STOPResult to be populated")
	}
	if len(result.Recommendations) == 0 {
		t.Error("expected recommendations to be populated")
	}
}
