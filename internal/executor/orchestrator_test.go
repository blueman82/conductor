package executor

import (
	"context"
	"errors"
	"fmt"
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
			name: "successful plan execution",
			plan: &models.Plan{
				Name: "Test Plan",
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
			name: "plan with failed tasks",
			plan: &models.Plan{
				Name: "Test Plan",
				Tasks: []models.Task{
					{Number: "1", Name: "Task 1"},
					{Number: "2", Name: "Task 2"},
					{Number: "3", Name: "Task 3"},
				},
				Waves: []models.Wave{
					{Name: "Wave 1", TaskNumbers: []string{"1", "2", "3"}},
				},
			},
			results: []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
				{Task: models.Task{Number: "2"}, Status: models.StatusRed, Error: errors.New("task failed")},
				{Task: models.Task{Number: "3"}, Status: models.StatusGreen},
			},
			waveErr:        errors.New("task failed"),
			expectedErr:    errors.New("task failed"),
			expectedTotal:  3,
			expectedGreen:  2,
			expectedFailed: 1,
		},
		{
			name: "empty plan",
			plan: &models.Plan{
				Name:  "Empty Plan",
				Tasks: []models.Task{},
				Waves: []models.Wave{},
			},
			results:        []models.TaskResult{},
			waveErr:        nil,
			expectedErr:    nil,
			expectedTotal:  0,
			expectedGreen:  0,
			expectedFailed: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWave := &mockWaveExecutor{
				executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
					return tt.results, tt.waveErr
				},
			}
			mockLog := &mockLogger{}

			orch := NewOrchestrator(mockWave, mockLog)
			result, err := orch.ExecutePlan(context.Background(), tt.plan)

			if tt.expectedErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.expectedErr)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.TotalTasks != tt.expectedTotal {
				t.Errorf("expected TotalTasks=%d, got %d", tt.expectedTotal, result.TotalTasks)
			}

			greenCount := 0
			failedCount := 0
			for _, r := range tt.results {
				if r.Status == models.StatusGreen || r.Status == models.StatusYellow {
					greenCount++
				} else if r.Status == models.StatusRed || r.Status == models.StatusFailed || r.Error != nil {
					failedCount++
				}
			}

			if result.Completed != greenCount {
				t.Errorf("expected Completed=%d, got %d", greenCount, result.Completed)
			}

			if result.Failed != failedCount {
				t.Errorf("expected Failed=%d, got %d", failedCount, result.Failed)
			}

			// Verify logger was called
			if len(mockLog.summaryCalls) != 1 {
				t.Errorf("expected 1 summary log call, got %d", len(mockLog.summaryCalls))
			}
		})
	}
}

// TestOrchestratorExecutePlanEmptyInput verifies error handling for no plans
func TestOrchestratorExecutePlanEmptyInput(t *testing.T) {
	mockWave := &mockWaveExecutor{}
	mockLog := &mockLogger{}
	orch := NewOrchestrator(mockWave, mockLog)

	result, err := orch.ExecutePlan(context.Background())
	if err == nil {
		t.Error("expected error for empty plans, got nil")
	}
	if result != nil {
		t.Error("expected nil result for empty plans")
	}
}

// TestOrchestratorGracefulShutdown verifies graceful shutdown via context cancellation.
// This also covers signal handling, as SIGINT/SIGTERM trigger context cancellation.
func TestOrchestratorGracefulShutdown(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			// Simulate long-running task that respects context cancellation
			select {
			case <-time.After(5 * time.Second):
				return []models.TaskResult{
					{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
				}, nil
			case <-ctx.Done():
				return []models.TaskResult{
					{Task: models.Task{Number: "1"}, Status: models.StatusFailed, Error: ctx.Err()},
				}, ctx.Err()
			}
		},
	}
	mockLog := &mockLogger{}

	plan := &models.Plan{
		Name: "Test Plan",
		Tasks: []models.Task{
			{Number: "1", Name: "Long Task"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1"}},
		},
	}

	orch := NewOrchestrator(mockWave, mockLog)

	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Start execution in a goroutine
	resultChan := make(chan struct {
		result *models.ExecutionResult
		err    error
	}, 1)

	go func() {
		result, err := orch.ExecutePlan(ctx, plan)
		resultChan <- struct {
			result *models.ExecutionResult
			err    error
		}{result, err}
	}()

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// Simulate SIGINT by canceling context
	cancel()

	// Wait for result
	res := <-resultChan

	if res.err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", res.err)
	}

	if res.result == nil {
		t.Fatal("expected non-nil result even on cancellation")
	}
}

func TestOrchestratorResultAggregation(t *testing.T) {
	results := []models.TaskResult{
		{Task: models.Task{Number: "1"}, Status: models.StatusGreen, Duration: 100 * time.Millisecond},
		{Task: models.Task{Number: "2"}, Status: models.StatusYellow, Duration: 150 * time.Millisecond},
		{Task: models.Task{Number: "3"}, Status: models.StatusRed, Error: errors.New("failed"), Duration: 200 * time.Millisecond},
		{Task: models.Task{Number: "4"}, Status: models.StatusFailed, Error: errors.New("error"), Duration: 50 * time.Millisecond},
		{Task: models.Task{Number: "5"}, Status: models.StatusGreen, Duration: 300 * time.Millisecond},
	}

	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			return results, errors.New("some tasks failed")
		},
	}
	mockLog := &mockLogger{}

	plan := &models.Plan{
		Name: "Test Plan",
		Tasks: []models.Task{
			{Number: "1"}, {Number: "2"}, {Number: "3"}, {Number: "4"}, {Number: "5"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2", "3", "4", "5"}},
		},
	}

	orch := NewOrchestrator(mockWave, mockLog)
	result, err := orch.ExecutePlan(context.Background(), plan)

	if err == nil {
		t.Error("expected error for failed tasks")
	}

	if result.TotalTasks != 5 {
		t.Errorf("expected TotalTasks=5, got %d", result.TotalTasks)
	}

	// GREEN (1) + YELLOW (1) = 2 completed
	if result.Completed != 3 {
		t.Errorf("expected Completed=3, got %d", result.Completed)
	}

	// RED (1) + FAILED (1) = 2 failed
	if result.Failed != 2 {
		t.Errorf("expected Failed=2, got %d", result.Failed)
	}

	if len(result.FailedTasks) != 2 {
		t.Errorf("expected 2 failed tasks, got %d", len(result.FailedTasks))
	}

	// Verify failed tasks are correct
	failedNumbers := make(map[string]bool)
	for _, ft := range result.FailedTasks {
		failedNumbers[ft.Task.Number] = true
	}
	if !failedNumbers["3"] || !failedNumbers["4"] {
		t.Errorf("expected tasks 3 and 4 in failed tasks, got %v", failedNumbers)
	}
}

func TestOrchestratorErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		waveErr     error
		expectedErr error
	}{
		{
			name:        "wave executor error",
			waveErr:     errors.New("wave execution failed"),
			expectedErr: errors.New("wave execution failed"),
		},
		{
			name:        "context canceled",
			waveErr:     context.Canceled,
			expectedErr: context.Canceled,
		},
		{
			name:        "context deadline exceeded",
			waveErr:     context.DeadlineExceeded,
			expectedErr: context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWave := &mockWaveExecutor{
				executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
					return nil, tt.waveErr
				},
			}
			mockLog := &mockLogger{}

			plan := &models.Plan{
				Name:  "Test Plan",
				Tasks: []models.Task{{Number: "1"}},
				Waves: []models.Wave{{Name: "Wave 1", TaskNumbers: []string{"1"}}},
			}

			orch := NewOrchestrator(mockWave, mockLog)
			_, err := orch.ExecutePlan(context.Background(), plan)

			if err == nil {
				t.Error("expected error, got nil")
			}

			// Check that error matches expected
			if err.Error() != tt.expectedErr.Error() {
				t.Errorf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

func TestOrchestratorContextCancellation(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			<-ctx.Done()
			return []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusFailed, Error: ctx.Err()},
			}, ctx.Err()
		},
	}
	mockLog := &mockLogger{}

	plan := &models.Plan{
		Name:  "Test Plan",
		Tasks: []models.Task{{Number: "1"}},
		Waves: []models.Wave{{Name: "Wave 1", TaskNumbers: []string{"1"}}},
	}

	orch := NewOrchestrator(mockWave, mockLog)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := orch.ExecutePlan(ctx, plan)

	if err != context.DeadlineExceeded && err != context.Canceled {
		t.Errorf("expected context cancellation error, got %v", err)
	}
}

func TestOrchestratorNilInputs(t *testing.T) {
	tests := []struct {
		name        string
		waveExec    *mockWaveExecutor
		logger      Logger
		plan        *models.Plan
		expectPanic bool
	}{
		{
			name:        "nil wave executor",
			waveExec:    nil,
			logger:      &mockLogger{},
			plan:        &models.Plan{},
			expectPanic: true,
		},
		{
			name:        "nil logger",
			waveExec:    &mockWaveExecutor{},
			logger:      nil,
			plan:        &models.Plan{},
			expectPanic: false, // Logger is optional
		},
		{
			name:        "nil plan",
			waveExec:    &mockWaveExecutor{},
			logger:      &mockLogger{},
			plan:        nil,
			expectPanic: false, // Should return error, not panic
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("expected panic, got none")
					}
				}()
			}

			orch := NewOrchestrator(tt.waveExec, tt.logger)
			var result *models.ExecutionResult
			var err error
			if tt.plan != nil {
				result, err = orch.ExecutePlan(context.Background(), tt.plan)
			} else {
				result, err = orch.ExecutePlan(context.Background())
			}

			if !tt.expectPanic && tt.plan == nil && err == nil {
				t.Error("expected error for no plans")
			}
			_ = result // Suppress unused variable warning
		})
	}
}

func TestNewOrchestrator(t *testing.T) {
	tests := []struct {
		name     string
		waveExec *mockWaveExecutor
		logger   Logger
		wantNil  bool
	}{
		{
			name:     "valid inputs",
			waveExec: &mockWaveExecutor{},
			logger:   &mockLogger{},
			wantNil:  false,
		},
		{
			name:     "nil logger is ok",
			waveExec: &mockWaveExecutor{},
			logger:   nil,
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orch := NewOrchestrator(tt.waveExec, tt.logger)
			if (orch == nil) != tt.wantNil {
				t.Errorf("NewOrchestrator() nil = %v, want %v", orch == nil, tt.wantNil)
			}
		})
	}
}

func TestOrchestratorLogging(t *testing.T) {
	results := []models.TaskResult{
		{Task: models.Task{Number: "1", Name: "Task 1"}, Status: models.StatusGreen},
		{Task: models.Task{Number: "2", Name: "Task 2"}, Status: models.StatusRed, Error: fmt.Errorf("failed")},
	}

	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			return results, nil
		},
	}
	mockLog := &mockLogger{}

	plan := &models.Plan{
		Name: "Test Plan",
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1"},
			{Number: "2", Name: "Task 2"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2"}},
		},
	}

	orch := NewOrchestrator(mockWave, mockLog)
	_, _ = orch.ExecutePlan(context.Background(), plan)

	// Verify summary was logged
	if len(mockLog.summaryCalls) == 0 {
		t.Error("expected summary to be logged")
	}

	if len(mockLog.summaryCalls) > 0 {
		summary := mockLog.summaryCalls[0]
		if summary.TotalTasks != 2 {
			t.Errorf("expected 2 total tasks in summary, got %d", summary.TotalTasks)
		}
	}
}

// TestOrchestratorTimeoutErrorHandling verifies that context timeout errors
// are properly detected and can be wrapped with TimeoutError.
func TestOrchestratorTimeoutErrorHandling(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			// Wait for context to timeout
			<-ctx.Done()

			// Return timeout error wrapped in TimeoutError for better error handling
			timeoutErr := NewTimeoutError("plan execution", 50*time.Millisecond)
			return []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusFailed, Error: timeoutErr},
			}, timeoutErr
		},
	}
	mockLog := &mockLogger{}

	plan := &models.Plan{
		Name:  "Test Plan",
		Tasks: []models.Task{{Number: "1"}},
		Waves: []models.Wave{{Name: "Wave 1", TaskNumbers: []string{"1"}}},
	}

	orch := NewOrchestrator(mockWave, mockLog)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := orch.ExecutePlan(ctx, plan)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	// Verify error is a timeout error
	if !IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}

	// Verify we can unwrap to context.DeadlineExceeded
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Error("expected error to wrap context.DeadlineExceeded")
	}

	// Verify result still returned
	if result == nil {
		t.Error("expected non-nil result even on timeout")
	}
}

// TestMergeMultiFilePlans verifies that multiple plan files are merged correctly
func TestMergeMultiFilePlans(t *testing.T) {
	tests := []struct {
		name             string
		plans            []*models.Plan
		expectedTasks    int
		expectedConflict bool
		expectErr        bool
	}{
		{
			name: "merge two simple plans",
			plans: []*models.Plan{
				{
					Name:     "Plan 1",
					FilePath: "plan1.yaml",
					Tasks: []models.Task{
						{Number: "1", Name: "Task 1", Prompt: "Do task 1"},
						{Number: "2", Name: "Task 2", Prompt: "Do task 2", DependsOn: []string{"1"}},
					},
				},
				{
					Name:     "Plan 2",
					FilePath: "plan2.yaml",
					Tasks: []models.Task{
						{Number: "3", Name: "Task 3", Prompt: "Do task 3"},
						{Number: "4", Name: "Task 4", Prompt: "Do task 4", DependsOn: []string{"3"}},
					},
				},
			},
			expectedTasks:    4,
			expectedConflict: false,
			expectErr:        false,
		},
		{
			name: "conflicting task numbers",
			plans: []*models.Plan{
				{
					Name:     "Plan 1",
					FilePath: "plan1.yaml",
					Tasks: []models.Task{
						{Number: "1", Name: "Task 1", Prompt: "Do task 1"},
					},
				},
				{
					Name:     "Plan 2",
					FilePath: "plan2.yaml",
					Tasks: []models.Task{
						{Number: "1", Name: "Task 1 Duplicate", Prompt: "Do duplicate task"},
					},
				},
			},
			expectedTasks:    0,
			expectedConflict: true,
			expectErr:        true,
		},
		{
			name: "single plan",
			plans: []*models.Plan{
				{
					Name:     "Plan 1",
					FilePath: "plan1.yaml",
					Tasks: []models.Task{
						{Number: "1", Name: "Task 1", Prompt: "Do task 1"},
					},
				},
			},
			expectedTasks:    1,
			expectedConflict: false,
			expectErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged, err := MergePlans(tt.plans...)

			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectErr && merged == nil {
				t.Fatal("expected non-nil merged plan")
			}

			if !tt.expectErr && len(merged.Tasks) != tt.expectedTasks {
				t.Errorf("expected %d tasks, got %d", tt.expectedTasks, len(merged.Tasks))
			}
		})
	}
}

// TestFileToTaskMapping verifies file-to-task ownership tracking
func TestFileToTaskMapping(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			return []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
				{Task: models.Task{Number: "2"}, Status: models.StatusGreen},
				{Task: models.Task{Number: "3"}, Status: models.StatusGreen},
			}, nil
		},
	}

	orch := NewOrchestrator(mockWave, nil)

	plans := []*models.Plan{
		{
			Name:     "Plan 1",
			FilePath: "plan1.yaml",
			Tasks: []models.Task{
				{Number: "1", Name: "Task 1", Prompt: "Do task 1"},
				{Number: "2", Name: "Task 2", Prompt: "Do task 2"},
			},
			Waves: []models.Wave{
				{Name: "Wave 1", TaskNumbers: []string{"1", "2"}},
			},
		},
		{
			Name:     "Plan 2",
			FilePath: "plan2.yaml",
			Tasks: []models.Task{
				{Number: "3", Name: "Task 3", Prompt: "Do task 3"},
			},
			Waves: []models.Wave{
				{Name: "Wave 1", TaskNumbers: []string{"3"}},
			},
		},
	}

	result, err := orch.ExecutePlan(context.Background(), plans...)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}
	_ = result // Suppress unused variable warning

	// Verify file-to-task mapping is populated
	if orch.FileToTaskMapping == nil {
		t.Error("FileToTaskMapping not initialized")
	}

	tests := []struct {
		taskNum      string
		expectedFile string
	}{
		{"1", "plan1.yaml"},
		{"2", "plan1.yaml"},
		{"3", "plan2.yaml"},
	}

	for _, test := range tests {
		if file, ok := orch.FileToTaskMapping[test.taskNum]; !ok {
			t.Errorf("task %s not in mapping", test.taskNum)
		} else if file != test.expectedFile {
			t.Errorf("task %s mapped to %s, expected %s", test.taskNum, file, test.expectedFile)
		}
	}
}

// TestOrchestratorWithConfig verifies that skip/retry config is properly threaded through orchestrator
func TestOrchestratorWithConfig(t *testing.T) {
	tests := []struct {
		name          string
		skipCompleted bool
		retryFailed   bool
	}{
		{
			name:          "skip completed enabled",
			skipCompleted: true,
			retryFailed:   false,
		},
		{
			name:          "retry failed enabled",
			skipCompleted: false,
			retryFailed:   true,
		},
		{
			name:          "both enabled",
			skipCompleted: true,
			retryFailed:   true,
		},
		{
			name:          "both disabled",
			skipCompleted: false,
			retryFailed:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWave := &mockWaveExecutor{
				executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
					return []models.TaskResult{
						{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
					}, nil
				},
			}
			mockLog := &mockLogger{}

			plan := &models.Plan{
				Name:  "Test Plan",
				Tasks: []models.Task{{Number: "1", Name: "Task 1"}},
				Waves: []models.Wave{{Name: "Wave 1", TaskNumbers: []string{"1"}}},
			}

			// Create orchestrator with config
			orch := NewOrchestratorWithConfig(mockWave, mockLog, tt.skipCompleted, tt.retryFailed)

			// Verify config fields are set
			if orch.skipCompleted != tt.skipCompleted {
				t.Errorf("expected skipCompleted=%v, got %v", tt.skipCompleted, orch.skipCompleted)
			}
			if orch.retryFailed != tt.retryFailed {
				t.Errorf("expected retryFailed=%v, got %v", tt.retryFailed, orch.retryFailed)
			}

			// Execute plan
			result, err := orch.ExecutePlan(context.Background(), plan)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// mockLearningStore is a test double for learning store
type mockLearningStore struct {
	getRunCountFunc func(ctx context.Context, planFile string) (int, error)
}

func (m *mockLearningStore) GetRunCount(ctx context.Context, planFile string) (int, error) {
	if m == nil || m.getRunCountFunc == nil {
		return 0, nil
	}
	return m.getRunCountFunc(ctx, planFile)
}

// TestOrchestrator_WithLearning verifies orchestrator initializes with learning store
func TestOrchestrator_WithLearning(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			return []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
			}, nil
		},
	}
	mockLog := &mockLogger{}
	mockStore := &mockLearningStore{
		getRunCountFunc: func(ctx context.Context, planFile string) (int, error) {
			return 0, nil
		},
	}

	config := OrchestratorConfig{
		WaveExecutor:  mockWave,
		Logger:        mockLog,
		LearningStore: mockStore,
		PlanFile:      "test-plan.yaml",
		SessionID:     "test-session",
	}

	orch := NewOrchestratorFromConfig(config)

	if orch.learningStore != mockStore {
		t.Error("expected learning store to be set")
	}
	if orch.sessionID != "test-session" {
		t.Errorf("expected sessionID='test-session', got %q", orch.sessionID)
	}
	if orch.planFile != "test-plan.yaml" {
		t.Errorf("expected planFile='test-plan.yaml', got %q", orch.planFile)
	}
}

// TestOrchestrator_WithoutLearning verifies orchestrator works with nil learning store
func TestOrchestrator_WithoutLearning(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			return []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
			}, nil
		},
	}
	mockLog := &mockLogger{}

	config := OrchestratorConfig{
		WaveExecutor:  mockWave,
		Logger:        mockLog,
		LearningStore: nil, // No learning
		PlanFile:      "test-plan.yaml",
	}

	orch := NewOrchestratorFromConfig(config)

	if orch.learningStore != nil {
		t.Error("expected learning store to be nil")
	}

	// Should still execute successfully without learning
	plan := &models.Plan{
		Name:  "Test Plan",
		Tasks: []models.Task{{Number: "1", Name: "Task 1"}},
		Waves: []models.Wave{{Name: "Wave 1", TaskNumbers: []string{"1"}}},
	}

	result, err := orch.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// TestOrchestrator_SessionIDGeneration verifies session ID is generated if not provided
func TestOrchestrator_SessionIDGeneration(t *testing.T) {
	mockWave := &mockWaveExecutor{}
	mockLog := &mockLogger{}

	tests := []struct {
		name              string
		providedSessionID string
		expectGenerated   bool
	}{
		{
			name:              "session ID provided",
			providedSessionID: "custom-session-123",
			expectGenerated:   false,
		},
		{
			name:              "session ID not provided",
			providedSessionID: "",
			expectGenerated:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := OrchestratorConfig{
				WaveExecutor: mockWave,
				Logger:       mockLog,
				SessionID:    tt.providedSessionID,
			}

			orch := NewOrchestratorFromConfig(config)

			if tt.expectGenerated {
				// Should have generated a session ID
				if orch.sessionID == "" {
					t.Error("expected session ID to be generated, got empty string")
				}
				// Verify format: session-YYYYMMDD-HHMM
				if len(orch.sessionID) < 17 || orch.sessionID[:8] != "session-" {
					t.Errorf("generated session ID has unexpected format: %q", orch.sessionID)
				}
			} else {
				// Should use provided session ID
				if orch.sessionID != tt.providedSessionID {
					t.Errorf("expected sessionID=%q, got %q", tt.providedSessionID, orch.sessionID)
				}
			}
		})
	}
}

// TestOrchestrator_RunNumberTracking verifies run number is calculated from learning store
func TestOrchestrator_RunNumberTracking(t *testing.T) {
	mockWave := &mockWaveExecutor{}
	mockLog := &mockLogger{}

	tests := []struct {
		name              string
		planFile          string
		storedRunCount    int
		providedRunNumber int
		expectedRunNumber int
		learningEnabled   bool
	}{
		{
			name:              "first run",
			planFile:          "plan.yaml",
			storedRunCount:    0,
			providedRunNumber: 0,
			expectedRunNumber: 1,
			learningEnabled:   true,
		},
		{
			name:              "subsequent run",
			planFile:          "plan.yaml",
			storedRunCount:    3,
			providedRunNumber: 0,
			expectedRunNumber: 4,
			learningEnabled:   true,
		},
		{
			name:              "manual run number",
			planFile:          "plan.yaml",
			storedRunCount:    5,
			providedRunNumber: 10,
			expectedRunNumber: 10,
			learningEnabled:   true,
		},
		{
			name:              "learning disabled",
			planFile:          "plan.yaml",
			storedRunCount:    0,
			providedRunNumber: 0,
			expectedRunNumber: 0,
			learningEnabled:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := OrchestratorConfig{
				WaveExecutor: mockWave,
				Logger:       mockLog,
				PlanFile:     tt.planFile,
				RunNumber:    tt.providedRunNumber,
			}

			// Only set learning store if learning is enabled
			if tt.learningEnabled {
				config.LearningStore = &mockLearningStore{
					getRunCountFunc: func(ctx context.Context, planFile string) (int, error) {
						if planFile != tt.planFile {
							t.Errorf("expected planFile=%q, got %q", tt.planFile, planFile)
						}
						return tt.storedRunCount, nil
					},
				}
			}

			orch := NewOrchestratorFromConfig(config)

			if orch.runNumber != tt.expectedRunNumber {
				t.Errorf("expected runNumber=%d, got %d", tt.expectedRunNumber, orch.runNumber)
			}
		})
	}
}
