package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/models"
	"github.com/harrison/conductor/internal/similarity"
)

// mockWaveExecutor is a test double for WaveExecutor.
type mockWaveExecutor struct {
	executePlanFunc func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error)
}

func (m *mockWaveExecutor) ExecutePlan(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
	if m.executePlanFunc != nil {
		return m.executePlanFunc(ctx, plan)
	}
	return nil, nil
}

// mockLogger captures logging calls for testing.
type mockLogger struct {
	waveStartCalls    []models.Wave
	waveCompleteCalls []struct {
		wave     models.Wave
		duration time.Duration
		results  []models.TaskResult
	}
	summaryCalls []models.ExecutionResult
}

func (m *mockLogger) LogWaveStart(wave models.Wave) {
	m.waveStartCalls = append(m.waveStartCalls, wave)
}

func (m *mockLogger) LogWaveComplete(wave models.Wave, duration time.Duration, results []models.TaskResult) {
	m.waveCompleteCalls = append(m.waveCompleteCalls, struct {
		wave     models.Wave
		duration time.Duration
		results  []models.TaskResult
	}{wave: wave, duration: duration, results: results})
}

func (m *mockLogger) LogTaskResult(result models.TaskResult) error {
	return nil
}

func (m *mockLogger) LogProgress(results []models.TaskResult) {}

func (m *mockLogger) LogSummary(result models.ExecutionResult) {
	m.summaryCalls = append(m.summaryCalls, result)
}

func (m *mockLogger) LogTaskAgentInvoke(task models.Task)                   {}
func (m *mockLogger) LogQCAgentSelection(agents []string, mode string)      {}
func (m *mockLogger) LogQCIndividualVerdicts(verdicts map[string]string)    {}
func (m *mockLogger) LogQCAggregatedResult(verdict string, strategy string) {}
func (m *mockLogger) LogQCCriteriaResults(agentName string, results []models.CriterionResult) {
}
func (m *mockLogger) LogQCIntelligentSelectionMetadata(rationale string, fallback bool, fallbackReason string) {
}
func (m *mockLogger) LogGuardPrediction(taskNumber string, result interface{})         {}
func (m *mockLogger) LogAgentSwap(taskNumber string, fromAgent string, toAgent string) {}
func (m *mockLogger) LogAnomaly(anomaly interface{})                                   {}
func (m *mockLogger) LogBudgetStatus(status interface{})                               {}
func (m *mockLogger) LogBudgetWarning(percentUsed float64)                             {}
func (m *mockLogger) LogRateLimitPause(delay time.Duration)                            {}
func (m *mockLogger) LogRateLimitResume()                                              {}
func (m *mockLogger) LogRateLimitCountdown(remaining, total time.Duration)             {}
func (m *mockLogger) LogRateLimitAnnounce(remaining, total time.Duration)              {}

func TestOrchestratorExecutePlan(t *testing.T) {
	tests := []struct {
		name           string
		plan           *models.Plan
		results        []models.TaskResult
		waveErr        error
		expectedErr    error
		expectedTotal  int
		expectedGreen  int
		expectedFailed int
	}{
		{
			name: "successful execution",
			plan: &models.Plan{
				Tasks: []models.Task{
					{Number: "1", Name: "Task 1"},
					{Number: "2", Name: "Task 2"},
				},
				Waves: []models.Wave{
					{Name: "Wave 1", TaskNumbers: []string{"1", "2"}},
				},
			},
			results: []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
				{Task: models.Task{Number: "2"}, Status: models.StatusGreen},
			},
			waveErr:        nil,
			expectedErr:    nil,
			expectedTotal:  2,
			expectedGreen:  2,
			expectedFailed: 0,
		},
		{
			name: "wave execution error",
			plan: &models.Plan{
				Tasks: []models.Task{
					{Number: "1", Name: "Task 1"},
				},
				Waves: []models.Wave{
					{Name: "Wave 1", TaskNumbers: []string{"1"}},
				},
			},
			results: []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusFailed},
			},
			waveErr:        errors.New("wave failed"),
			expectedErr:    errors.New("wave failed"),
			expectedTotal:  1,
			expectedGreen:  0,
			expectedFailed: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWE := &mockWaveExecutor{
				executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
					return tt.results, tt.waveErr
				},
			}

			mockLog := &mockLogger{}
			orchestrator := NewOrchestrator(mockWE, mockLog)

			result, err := orchestrator.ExecutePlan(context.Background(), tt.plan)

			// Check error
			if tt.expectedErr != nil {
				if err == nil {
					t.Errorf("expected error %q, got nil", tt.expectedErr)
				} else if err.Error() != tt.expectedErr.Error() {
					t.Errorf("expected error %q, got %q", tt.expectedErr, err)
				}
			} else if err != nil {
				t.Errorf("expected no error, got %q", err)
			}

			// Check total tasks from plan
			if result.TotalTasks != tt.expectedTotal {
				t.Errorf("expected %d total tasks, got %d", tt.expectedTotal, result.TotalTasks)
			}

			// Check completed count (GREEN tasks)
			if result.Completed != tt.expectedGreen {
				t.Errorf("expected %d completed, got %d", tt.expectedGreen, result.Completed)
			}

			// Check failed count
			if result.Failed != tt.expectedFailed {
				t.Errorf("expected %d failed, got %d", tt.expectedFailed, result.Failed)
			}

			// Check that summary was logged
			if len(mockLog.summaryCalls) != 1 {
				t.Errorf("expected 1 summary log call, got %d", len(mockLog.summaryCalls))
			}
		})
	}
}

func TestOrchestratorContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockWE := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return []models.TaskResult{}, nil
			}
		},
	}

	mockLog := &mockLogger{}
	orchestrator := NewOrchestrator(mockWE, mockLog)

	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1"}},
		},
	}

	_, err := orchestrator.ExecutePlan(ctx, plan)

	if err == nil {
		t.Error("expected context cancellation error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

func TestOrchestratorNoPlan(t *testing.T) {
	mockWE := &mockWaveExecutor{}
	mockLog := &mockLogger{}
	orchestrator := NewOrchestrator(mockWE, mockLog)

	// Pass no plans (empty variadic)
	_, err := orchestrator.ExecutePlan(context.Background())

	if err == nil {
		t.Error("expected error for no plans, got nil")
	}
}

func TestOrchestratorNilLogger(t *testing.T) {
	mockWE := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			return []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
			}, nil
		},
	}

	orchestrator := NewOrchestrator(mockWE, nil)

	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1"}},
		},
	}

	result, err := orchestrator.ExecutePlan(context.Background(), plan)

	if err != nil {
		t.Errorf("expected no error with nil logger, got %v", err)
	}

	if result.Completed != 1 {
		t.Errorf("expected 1 completed task, got %d", result.Completed)
	}
}

func TestOrchestratorExecutionMetrics(t *testing.T) {
	mockWE := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			time.Sleep(10 * time.Millisecond)
			return []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
			}, nil
		},
	}

	mockLog := &mockLogger{}
	orchestrator := NewOrchestrator(mockWE, mockLog)

	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1"}},
		},
	}

	result, _ := orchestrator.ExecutePlan(context.Background(), plan)

	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}

	if result.Duration < 10*time.Millisecond {
		t.Errorf("expected duration >= 10ms, got %v", result.Duration)
	}
}

func (m *mockLogger) SetGuardVerbose(verbose bool) {}

// TestOrchestratorFromConfigWithPatternHook verifies orchestrator configuration
// with PatternHook integration (ClaudeSimilarity wiring test).
func TestOrchestratorFromConfigWithPatternHook(t *testing.T) {
	mockWE := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			return []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
			}, nil
		},
	}

	mockLog := &mockLogger{}

	// Test that OrchestratorConfig properly wires the pattern hook
	config := OrchestratorConfig{
		WaveExecutor:  mockWE,
		Logger:        mockLog,
		LearningStore: nil, // Can be nil
		SessionID:     "test-session-123",
		RunNumber:     1,
		PlanFile:      "test-plan.md",
		SkipCompleted: false,
		RetryFailed:   false,
		PatternHook:   nil, // Pattern hook is optional
	}

	orchestrator := NewOrchestratorFromConfig(config)

	// Verify orchestrator was created successfully
	if orchestrator == nil {
		t.Fatal("expected non-nil orchestrator")
	}

	// Verify config values were applied
	if orchestrator.planFile != "test-plan.md" {
		t.Errorf("expected planFile 'test-plan.md', got %q", orchestrator.planFile)
	}

	if orchestrator.sessionID != "test-session-123" {
		t.Errorf("expected sessionID 'test-session-123', got %q", orchestrator.sessionID)
	}

	// Execute plan to verify basic functionality
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1"}},
		},
	}

	result, err := orchestrator.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Completed != 1 {
		t.Errorf("expected 1 completed task, got %d", result.Completed)
	}
}

// TestOrchestratorFromConfigWithWarmUpHook verifies that orchestrator config
// integrates properly with WarmUpHook-enabled WaveExecutor.
func TestOrchestratorFromConfigWithWarmUpHook(t *testing.T) {
	// This test verifies that the WarmUpHook integration point in
	// the wiring (run.go) will work correctly with the orchestrator.
	// The actual WarmUpHook is tested in warmup_hook_test.go.

	t.Run("config with learning store and run number calculation", func(t *testing.T) {
		// Mock learning store that returns a specific run count
		mockStore := &mockLearningStore{
			runCount: 5,
		}

		mockWE := &mockWaveExecutor{
			executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
				return []models.TaskResult{
					{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
				}, nil
			},
		}

		config := OrchestratorConfig{
			WaveExecutor:  mockWE,
			Logger:        &mockLogger{},
			LearningStore: mockStore,
			RunNumber:     0, // Should be calculated from store
			PlanFile:      "test-plan.md",
		}

		orchestrator := NewOrchestratorFromConfig(config)

		// RunNumber should be previous count + 1 = 6
		if orchestrator.runNumber != 6 {
			t.Errorf("expected runNumber 6, got %d", orchestrator.runNumber)
		}
	})

	t.Run("config with explicit run number", func(t *testing.T) {
		mockWE := &mockWaveExecutor{}

		config := OrchestratorConfig{
			WaveExecutor: mockWE,
			Logger:       &mockLogger{},
			RunNumber:    42, // Explicit run number
			PlanFile:     "test-plan.md",
		}

		orchestrator := NewOrchestratorFromConfig(config)

		// Explicit RunNumber should be preserved
		if orchestrator.runNumber != 42 {
			t.Errorf("expected runNumber 42, got %d", orchestrator.runNumber)
		}
	})
}

// mockLearningStore implements LearningStoreInterface for testing.
type mockLearningStore struct {
	runCount int
}

func (m *mockLearningStore) GetRunCount(ctx context.Context, planFile string) (int, error) {
	return m.runCount, nil
}

// TestOrchestratorFromConfigWithSimilarity verifies that OrchestratorConfig
// properly wires ClaudeSimilarity into the orchestrator (v2.32+).
// This tests the shared similarity instance used by PatternIntelligence and WarmUpProvider.
func TestOrchestratorFromConfigWithSimilarity(t *testing.T) {
	t.Run("config with ClaudeSimilarity wired", func(t *testing.T) {
		mockWE := &mockWaveExecutor{
			executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
				return []models.TaskResult{
					{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
				}, nil
			},
		}

		// Create ClaudeSimilarity instance (normally created in run.go)
		claudeSim := similarity.NewClaudeSimilarity(90*time.Second, nil)

		config := OrchestratorConfig{
			WaveExecutor:  mockWE,
			Logger:        &mockLogger{},
			LearningStore: nil,
			SessionID:     "test-session-similarity",
			RunNumber:     1,
			PlanFile:      "test-plan.md",
			Similarity:    claudeSim,
		}

		orchestrator := NewOrchestratorFromConfig(config)

		// Verify orchestrator was created successfully
		if orchestrator == nil {
			t.Fatal("expected non-nil orchestrator")
		}

		// Verify similarity was wired correctly
		if orchestrator.Similarity() != claudeSim {
			t.Error("expected Similarity() to return the configured ClaudeSimilarity instance")
		}

		// Verify other config values were applied
		if orchestrator.sessionID != "test-session-similarity" {
			t.Errorf("expected sessionID 'test-session-similarity', got %q", orchestrator.sessionID)
		}
	})

	t.Run("config without ClaudeSimilarity (nil)", func(t *testing.T) {
		mockWE := &mockWaveExecutor{}

		config := OrchestratorConfig{
			WaveExecutor: mockWE,
			Logger:       &mockLogger{},
			Similarity:   nil, // No similarity configured
		}

		orchestrator := NewOrchestratorFromConfig(config)

		// Verify orchestrator was created successfully
		if orchestrator == nil {
			t.Fatal("expected non-nil orchestrator")
		}

		// Verify Similarity() returns nil when not configured
		if orchestrator.Similarity() != nil {
			t.Error("expected Similarity() to return nil when not configured")
		}
	})

	t.Run("orchestrator execution with similarity configured", func(t *testing.T) {
		mockWE := &mockWaveExecutor{
			executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
				return []models.TaskResult{
					{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
				}, nil
			},
		}

		claudeSim := similarity.NewClaudeSimilarity(90*time.Second, nil)

		config := OrchestratorConfig{
			WaveExecutor: mockWE,
			Logger:       &mockLogger{},
			Similarity:   claudeSim,
		}

		orchestrator := NewOrchestratorFromConfig(config)

		plan := &models.Plan{
			Tasks: []models.Task{
				{Number: "1", Name: "Task 1"},
			},
			Waves: []models.Wave{
				{Name: "Wave 1", TaskNumbers: []string{"1"}},
			},
		}

		result, err := orchestrator.ExecutePlan(context.Background(), plan)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if result.Completed != 1 {
			t.Errorf("expected 1 completed task, got %d", result.Completed)
		}

		// Verify similarity is still accessible after execution
		if orchestrator.Similarity() != claudeSim {
			t.Error("similarity should remain accessible after execution")
		}
	})
}
