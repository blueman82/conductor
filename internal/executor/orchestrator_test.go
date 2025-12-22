package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/models"
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
	// Base mockLogger doesn't track task results; this is a no-op.
	// Use mockLoggerWithTaskLogging for tests that need task logging.
	return nil
}

func (m *mockLogger) LogProgress(results []models.TaskResult) {
	// no-op for mock
}

func (m *mockLogger) LogSummary(result models.ExecutionResult) {
	m.summaryCalls = append(m.summaryCalls, result)
}

func (m *mockLogger) LogTaskAgentInvoke(task models.Task) {
	// no-op for mock
}

func (m *mockLogger) LogQCAgentSelection(agents []string, mode string) {
	// no-op for mock
}

func (m *mockLogger) LogQCIndividualVerdicts(verdicts map[string]string) {
	// no-op for mock
}

func (m *mockLogger) LogQCAggregatedResult(verdict string, strategy string) {
	// no-op for mock
}

func (m *mockLogger) LogQCCriteriaResults(agentName string, results []models.CriterionResult) {
	// no-op for mock
}

func (m *mockLogger) LogQCIntelligentSelectionMetadata(rationale string, fallback bool, fallbackReason string) {
	// no-op for mock
}

func (m *mockLogger) LogGuardPrediction(taskNumber string, result interface{}) {
	// no-op for mock
}

func (m *mockLogger) LogAgentSwap(taskNumber string, fromAgent string, toAgent string) {
	// no-op for mock
}

func (m *mockLogger) LogAnomaly(anomaly interface{}) {
	// no-op for mock
}

// Budget tracking methods (v2.19+)
func (m *mockLogger) LogBudgetStatus(status interface{}) {
	// no-op for mock
}

func (m *mockLogger) LogBudgetWarning(percentUsed float64) {
	// no-op for mock
}

func (m *mockLogger) LogRateLimitPause(delay time.Duration) {
	// no-op for mock
}

func (m *mockLogger) LogRateLimitResume() {
	// no-op for mock
}

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
		{
			name: "multiple waves - stops at first error",
			plan: &models.Plan{
				Tasks: []models.Task{
					{Number: "1", Name: "Task 1"},
					{Number: "2", Name: "Task 2"},
				},
				Waves: []models.Wave{
					{Name: "Wave 1", TaskNumbers: []string{"1"}},
					{Name: "Wave 2", TaskNumbers: []string{"2"}},
				},
			},
			results: []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusFailed},
			},
			waveErr:        errors.New("wave 1 failed"),
			expectedErr:    errors.New("wave 1 failed"),
			expectedTotal:  1,
			expectedGreen:  0,
			expectedFailed: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock wave executor that returns test results
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

			// Check total tasks
			if len(result.TaskResults) != tt.expectedTotal {
				t.Errorf("expected %d task results, got %d", tt.expectedTotal, len(result.TaskResults))
			}

			// Check green count
			greenCount := 0
			for _, r := range result.TaskResults {
				if r.Status == models.StatusGreen {
					greenCount++
				}
			}
			if greenCount != tt.expectedGreen {
				t.Errorf("expected %d GREEN tasks, got %d", tt.expectedGreen, greenCount)
			}

			// Check that summary was logged
			if len(mockLog.summaryCalls) != 1 {
				t.Errorf("expected 1 summary log call, got %d", len(mockLog.summaryCalls))
			}
		})
	}
}

func TestOrchestratorSessionID(t *testing.T) {
	mockWE := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			return []models.TaskResult{}, nil
		},
	}

	mockLog := &mockLogger{}

	// Test with explicit session ID
	config := OrchestratorConfig{
		WaveExecutor: mockWE,
		Logger:       mockLog,
		SessionID:    "test-session-123",
	}
	orchestrator := NewOrchestratorFromConfig(config)

	plan := &models.Plan{
		Tasks: []models.Task{},
		Waves: []models.Wave{},
	}

	result, _ := orchestrator.ExecutePlan(context.Background(), plan)

	if result.SessionID != "test-session-123" {
		t.Errorf("expected session ID %q, got %q", "test-session-123", result.SessionID)
	}

	// Test with auto-generated session ID
	config2 := OrchestratorConfig{
		WaveExecutor: mockWE,
		Logger:       mockLog,
		SessionID:    "", // Empty = auto-generate
	}
	orchestrator2 := NewOrchestratorFromConfig(config2)

	result2, _ := orchestrator2.ExecutePlan(context.Background(), plan)

	if result2.SessionID == "" {
		t.Error("expected auto-generated session ID, got empty string")
	}

	// Session IDs should be different for different orchestrators
	if result.SessionID == result2.SessionID {
		t.Error("expected different session IDs for different orchestrators")
	}
}

func TestOrchestratorContextCancellation(t *testing.T) {
	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockWE := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			// Check if context is cancelled
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

func TestOrchestratorNilPlan(t *testing.T) {
	mockWE := &mockWaveExecutor{}
	mockLog := &mockLogger{}
	orchestrator := NewOrchestrator(mockWE, mockLog)

	// ExecutePlan should handle nil plan gracefully - it returns an error
	_, err := orchestrator.ExecutePlan(context.Background(), nil)

	// Should return an error for nil plan (at least one plan is required)
	if err == nil {
		t.Error("expected error for nil plan, got nil")
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

	// Create orchestrator with nil logger - should not panic
	orchestrator := NewOrchestrator(mockWE, nil)

	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1"}},
		},
	}

	result := orchestrator.ExecutePlan(context.Background(), plan)

	if result.Error != nil {
		t.Errorf("expected no error with nil logger, got %v", result.Error)
	}

	if len(result.TaskResults) != 1 {
		t.Errorf("expected 1 task result, got %d", len(result.TaskResults))
	}
}

func TestOrchestratorExecutionMetrics(t *testing.T) {
	startTime := time.Now()

	mockWE := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			// Simulate some work
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

	result := orchestrator.ExecutePlan(context.Background(), plan)

	duration := result.Duration

	if duration <= 0 {
		t.Error("expected positive duration")
	}

	// Duration should be at least as long as our simulated work
	if duration < 10*time.Millisecond {
		t.Errorf("expected duration >= 10ms, got %v", duration)
	}

	// Duration should be measured from start
	expectedMin := time.Since(startTime)
	if duration > expectedMin+100*time.Millisecond {
		t.Errorf("duration %v seems too long compared to expected %v", duration, expectedMin)
	}
}
