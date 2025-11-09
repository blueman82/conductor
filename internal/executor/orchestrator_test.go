package executor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
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
	}
	taskStartCalls    []models.Task
	taskCompleteCalls []models.TaskResult
	taskFailCalls     []models.TaskResult
	summaryCalls      []models.ExecutionResult
}

func (m *mockLogger) LogWaveStart(wave models.Wave) {
	m.waveStartCalls = append(m.waveStartCalls, wave)
}

func (m *mockLogger) LogWaveComplete(wave models.Wave, duration time.Duration) {
	m.waveCompleteCalls = append(m.waveCompleteCalls, struct {
		wave     models.Wave
		duration time.Duration
	}{wave: wave, duration: duration})
}

func (m *mockLogger) LogTaskStart(task models.Task) {
	m.taskStartCalls = append(m.taskStartCalls, task)
}

func (m *mockLogger) LogTaskComplete(result models.TaskResult) {
	m.taskCompleteCalls = append(m.taskCompleteCalls, result)
}

func (m *mockLogger) LogTaskFail(result models.TaskResult) {
	m.taskFailCalls = append(m.taskFailCalls, result)
}

func (m *mockLogger) LogSummary(result models.ExecutionResult) {
	m.summaryCalls = append(m.summaryCalls, result)
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
					{Number: 1, Name: "Task 1"},
					{Number: 2, Name: "Task 2"},
				},
				Waves: []models.Wave{
					{Name: "Wave 1", TaskNumbers: []int{1, 2}},
				},
			},
			results: []models.TaskResult{
				{Task: models.Task{Number: 1}, Status: "GREEN"},
				{Task: models.Task{Number: 2}, Status: "GREEN"},
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
					{Number: 1, Name: "Task 1"},
					{Number: 2, Name: "Task 2"},
					{Number: 3, Name: "Task 3"},
				},
				Waves: []models.Wave{
					{Name: "Wave 1", TaskNumbers: []int{1, 2, 3}},
				},
			},
			results: []models.TaskResult{
				{Task: models.Task{Number: 1}, Status: "GREEN"},
				{Task: models.Task{Number: 2}, Status: "RED", Error: errors.New("task failed")},
				{Task: models.Task{Number: 3}, Status: "GREEN"},
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
				if r.Status == "GREEN" || r.Status == "YELLOW" {
					greenCount++
				} else if r.Status == "RED" || r.Status == "FAILED" || r.Error != nil {
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

func TestOrchestratorGracefulShutdown(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			// Simulate long-running task that respects context cancellation
			select {
			case <-time.After(5 * time.Second):
				return []models.TaskResult{
					{Task: models.Task{Number: 1}, Status: "GREEN"},
				}, nil
			case <-ctx.Done():
				return []models.TaskResult{
					{Task: models.Task{Number: 1}, Status: "FAILED", Error: ctx.Err()},
				}, ctx.Err()
			}
		},
	}
	mockLog := &mockLogger{}

	plan := &models.Plan{
		Name: "Test Plan",
		Tasks: []models.Task{
			{Number: 1, Name: "Long Task"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []int{1}},
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
		{Task: models.Task{Number: 1}, Status: "GREEN", Duration: 100 * time.Millisecond},
		{Task: models.Task{Number: 2}, Status: "YELLOW", Duration: 150 * time.Millisecond},
		{Task: models.Task{Number: 3}, Status: "RED", Error: errors.New("failed"), Duration: 200 * time.Millisecond},
		{Task: models.Task{Number: 4}, Status: "FAILED", Error: errors.New("error"), Duration: 50 * time.Millisecond},
		{Task: models.Task{Number: 5}, Status: "GREEN", Duration: 300 * time.Millisecond},
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
			{Number: 1}, {Number: 2}, {Number: 3}, {Number: 4}, {Number: 5},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []int{1, 2, 3, 4, 5}},
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
	failedNumbers := make(map[int]bool)
	for _, ft := range result.FailedTasks {
		failedNumbers[ft.Task.Number] = true
	}
	if !failedNumbers[3] || !failedNumbers[4] {
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
				Tasks: []models.Task{{Number: 1}},
				Waves: []models.Wave{{Name: "Wave 1", TaskNumbers: []int{1}}},
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
				{Task: models.Task{Number: 1}, Status: "FAILED", Error: ctx.Err()},
			}, ctx.Err()
		},
	}
	mockLog := &mockLogger{}

	plan := &models.Plan{
		Name:  "Test Plan",
		Tasks: []models.Task{{Number: 1}},
		Waves: []models.Wave{{Name: "Wave 1", TaskNumbers: []int{1}}},
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
			_, err := orch.ExecutePlan(context.Background(), tt.plan)

			if !tt.expectPanic && tt.plan == nil && err == nil {
				t.Error("expected error for nil plan")
			}
		})
	}
}

// TestOrchestratorSignalHandling tests actual signal handling with real signals.
// This test is more integration-style and may be flaky in some environments.
func TestOrchestratorSignalHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping signal handling test in short mode")
	}

	// Create a mock wave executor that blocks until context is canceled
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			<-ctx.Done()
			return []models.TaskResult{
				{Task: models.Task{Number: 1}, Status: "FAILED", Error: ctx.Err()},
			}, ctx.Err()
		},
	}
	mockLog := &mockLogger{}

	plan := &models.Plan{
		Name:  "Test Plan",
		Tasks: []models.Task{{Number: 1}},
		Waves: []models.Wave{{Name: "Wave 1", TaskNumbers: []int{1}}},
	}

	orch := NewOrchestrator(mockWave, mockLog)

	// Start execution in background
	resultChan := make(chan error, 1)
	go func() {
		_, err := orch.ExecutePlan(context.Background(), plan)
		resultChan <- err
	}()

	// Give orchestrator time to set up signal handling
	time.Sleep(50 * time.Millisecond)

	// Send SIGINT to self
	proc, _ := os.FindProcess(os.Getpid())
	if err := proc.Signal(syscall.SIGINT); err != nil {
		t.Fatalf("failed to send SIGINT: %v", err)
	}

	// Wait for result
	select {
	case err := <-resultChan:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("orchestrator did not respond to SIGINT")
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
		{Task: models.Task{Number: 1, Name: "Task 1"}, Status: "GREEN"},
		{Task: models.Task{Number: 2, Name: "Task 2"}, Status: "RED", Error: fmt.Errorf("failed")},
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
			{Number: 1, Name: "Task 1"},
			{Number: 2, Name: "Task 2"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []int{1, 2}},
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
