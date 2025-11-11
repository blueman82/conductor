package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/models"
)

type sequentialMockExecutor struct {
	wave1Done chan struct{}
	mu        sync.Mutex
	wave1Seen int
}

func newSequentialMockExecutor() *sequentialMockExecutor {
	return &sequentialMockExecutor{
		wave1Done: make(chan struct{}),
	}
}

func (m *sequentialMockExecutor) Execute(ctx context.Context, task models.Task) (models.TaskResult, error) {
	if task.Number == "1" || task.Number == "2" {
		time.Sleep(10 * time.Millisecond)
		m.mu.Lock()
		m.wave1Seen++
		if m.wave1Seen == 2 {
			close(m.wave1Done)
		}
		m.mu.Unlock()
	} else {
		select {
		case <-m.wave1Done:
		default:
			return models.TaskResult{Task: task, Status: models.StatusFailed}, fmt.Errorf("wave 2 task %s executed before wave 1 completion", task.Number)
		}
	}

	return models.TaskResult{
		Task:   task,
		Status: models.StatusGreen,
	}, nil
}

func TestWaveExecutor_WavesExecuteSequentially(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Prompt: "Do task 1"},
			{Number: "2", Name: "Task 2", Prompt: "Do task 2"},
			{Number: "3", Name: "Task 3", Prompt: "Do task 3"},
			{Number: "4", Name: "Task 4", Prompt: "Do task 4"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2"}, MaxConcurrency: 2},
			{Name: "Wave 2", TaskNumbers: []string{"3", "4"}, MaxConcurrency: 2},
		},
	}

	mockExecutor := newSequentialMockExecutor()
	waveExecutor := NewWaveExecutor(mockExecutor, nil)

	results, err := waveExecutor.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan returned error: %v", err)
	}

	if len(results) != len(plan.Tasks) {
		t.Fatalf("expected %d results, got %d", len(plan.Tasks), len(results))
	}
}

type concurrencyMockExecutor struct {
	mu       sync.Mutex
	current  int
	maxSeen  int
	duration time.Duration
}

func newConcurrencyMockExecutor(duration time.Duration) *concurrencyMockExecutor {
	return &concurrencyMockExecutor{duration: duration}
}

func (m *concurrencyMockExecutor) Execute(ctx context.Context, task models.Task) (models.TaskResult, error) {
	m.mu.Lock()
	m.current++
	if m.current > m.maxSeen {
		m.maxSeen = m.current
	}
	m.mu.Unlock()

	select {
	case <-ctx.Done():
	case <-time.After(m.duration):
	}

	m.mu.Lock()
	m.current--
	m.mu.Unlock()

	return models.TaskResult{Task: task, Status: models.StatusGreen}, nil
}

func TestWaveExecutor_RespectsMaxConcurrency(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Prompt: "Do task 1"},
			{Number: "2", Name: "Task 2", Prompt: "Do task 2"},
			{Number: "3", Name: "Task 3", Prompt: "Do task 3"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2", "3"}, MaxConcurrency: 2},
		},
	}

	mockExecutor := newConcurrencyMockExecutor(25 * time.Millisecond)
	waveExecutor := NewWaveExecutor(mockExecutor, nil)

	_, err := waveExecutor.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan returned error: %v", err)
	}

	if mockExecutor.maxSeen > plan.Waves[0].MaxConcurrency {
		t.Fatalf("expected max concurrency <= %d, got %d", plan.Waves[0].MaxConcurrency, mockExecutor.maxSeen)
	}
}

type cancellableMockExecutor struct {
	cancel func()
	mu     sync.Mutex
	calls  int
}

func (m *cancellableMockExecutor) Execute(ctx context.Context, task models.Task) (models.TaskResult, error) {
	m.mu.Lock()
	m.calls++
	callNumber := m.calls
	m.mu.Unlock()

	if callNumber == 1 && m.cancel != nil {
		m.cancel()
	}

	select {
	case <-ctx.Done():
		return models.TaskResult{Task: task, Status: models.StatusFailed, Error: ctx.Err()}, ctx.Err()
	case <-time.After(5 * time.Millisecond):
		return models.TaskResult{Task: task, Status: models.StatusGreen}, nil
	}
}

func TestWaveExecutor_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	mockExecutor := &cancellableMockExecutor{cancel: cancel}
	waveExecutor := NewWaveExecutor(mockExecutor, nil)

	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Prompt: "Do task 1"},
			{Number: "2", Name: "Task 2", Prompt: "Do task 2"},
			{Number: "3", Name: "Task 3", Prompt: "Do task 3"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2", "3"}, MaxConcurrency: 2},
		},
	}

	results, err := waveExecutor.ExecutePlan(ctx, plan)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation error, got %v", err)
	}

	if len(results) == len(plan.Tasks) {
		t.Fatalf("expected fewer results due to cancellation, got %d", len(results))
	}
}

func TestWaveExecutor_ErrorsOnMissingTask(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Prompt: "Do task 1"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "99"}, MaxConcurrency: 1},
		},
	}

	waveExecutor := NewWaveExecutor(&sequentialMockExecutor{wave1Done: make(chan struct{})}, nil)

	_, err := waveExecutor.ExecutePlan(context.Background(), plan)
	if err == nil {
		t.Fatalf("expected error for missing task")
	}
	if err != nil && !strings.Contains(err.Error(), "task 99") {
		t.Fatalf("expected error mentioning missing task, got %v", err)
	}
}

func TestWaveExecutor_LoggingIntegration(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Prompt: "Do task 1"},
			{Number: "2", Name: "Task 2", Prompt: "Do task 2"},
			{Number: "3", Name: "Task 3", Prompt: "Do task 3"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2"}, MaxConcurrency: 2},
			{Name: "Wave 2", TaskNumbers: []string{"3"}, MaxConcurrency: 1},
		},
	}

	mockExecutor := newSequentialMockExecutor()
	mockLog := &mockLogger{}
	waveExecutor := NewWaveExecutor(mockExecutor, mockLog)

	results, err := waveExecutor.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan returned error: %v", err)
	}

	if len(results) != len(plan.Tasks) {
		t.Fatalf("expected %d results, got %d", len(plan.Tasks), len(results))
	}

	// Verify LogWaveStart was called for each wave
	if len(mockLog.waveStartCalls) != 2 {
		t.Errorf("expected 2 LogWaveStart calls, got %d", len(mockLog.waveStartCalls))
	}

	// Verify LogWaveComplete was called for each wave
	if len(mockLog.waveCompleteCalls) != 2 {
		t.Errorf("expected 2 LogWaveComplete calls, got %d", len(mockLog.waveCompleteCalls))
	}

	// Verify wave names are correct
	if len(mockLog.waveStartCalls) >= 2 {
		if mockLog.waveStartCalls[0].Name != "Wave 1" {
			t.Errorf("expected first wave to be 'Wave 1', got %s", mockLog.waveStartCalls[0].Name)
		}
		if mockLog.waveStartCalls[1].Name != "Wave 2" {
			t.Errorf("expected second wave to be 'Wave 2', got %s", mockLog.waveStartCalls[1].Name)
		}
	}

	// Verify durations are recorded
	if len(mockLog.waveCompleteCalls) >= 2 {
		for i, call := range mockLog.waveCompleteCalls {
			if call.duration <= 0 {
				t.Errorf("expected positive duration for wave %d, got %v", i+1, call.duration)
			}
		}
	}
}

func TestWaveExecutor_LoggingWithNilLogger(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Prompt: "Do task 1"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1"}, MaxConcurrency: 1},
		},
	}

	mockExecutor := newSequentialMockExecutor()
	waveExecutor := NewWaveExecutor(mockExecutor, nil)

	// Should not panic with nil logger
	_, err := waveExecutor.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan returned error: %v", err)
	}
}

func TestWaveExecutor_LoggingOnEmptyWave(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{},
		Waves: []models.Wave{
			{Name: "Empty Wave", TaskNumbers: []string{}, MaxConcurrency: 1},
		},
	}

	mockExecutor := newSequentialMockExecutor()
	mockLog := &mockLogger{}
	waveExecutor := NewWaveExecutor(mockExecutor, mockLog)

	_, err := waveExecutor.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan returned error: %v", err)
	}

	// Empty waves should not trigger logging
	if len(mockLog.waveStartCalls) != 0 {
		t.Errorf("expected 0 LogWaveStart calls for empty wave, got %d", len(mockLog.waveStartCalls))
	}
	if len(mockLog.waveCompleteCalls) != 0 {
		t.Errorf("expected 0 LogWaveComplete calls for empty wave, got %d", len(mockLog.waveCompleteCalls))
	}
}

// TestWaveExecutor_MissingTaskReturnsTaskError verifies that missing tasks return TaskError.
// This tests the integration of custom error types in wave.go line 110.
func TestWaveExecutor_MissingTaskReturnsTaskError(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Prompt: "Do task 1"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "99"}, MaxConcurrency: 1},
		},
	}

	waveExecutor := NewWaveExecutor(&sequentialMockExecutor{wave1Done: make(chan struct{})}, nil)

	_, err := waveExecutor.ExecutePlan(context.Background(), plan)
	if err == nil {
		t.Fatalf("expected error for missing task")
	}

	// Verify that the error is a TaskError
	if !IsTaskError(err) {
		t.Errorf("expected TaskError, got %T: %v", err, err)
	}

	// Verify error contains task information
	var taskErr *TaskError
	if errors.As(err, &taskErr) {
		if taskErr.TaskName != "99" {
			t.Errorf("expected TaskName '99', got %q", taskErr.TaskName)
		}
		if !strings.Contains(taskErr.Message, "not found") {
			t.Errorf("expected message to contain 'not found', got %q", taskErr.Message)
		}
	}
}

// TestWaveExecutor_TaskExecutionErrorsAggregated verifies that task execution errors
// are properly wrapped in ExecutionError with phase context.
func TestWaveExecutor_TaskExecutionErrorsAggregated(t *testing.T) {
	failingExecutor := &mockFailingExecutor{
		failOnTask: "2",
	}

	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Prompt: "Do task 1"},
			{Number: "2", Name: "Task 2", Prompt: "Do task 2"},
			{Number: "3", Name: "Task 3", Prompt: "Do task 3"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2", "3"}, MaxConcurrency: 2},
		},
	}

	waveExecutor := NewWaveExecutor(failingExecutor, nil)
	results, err := waveExecutor.ExecutePlan(context.Background(), plan)

	if err == nil {
		t.Fatalf("expected error from failing task")
	}

	// Verify we got results even with error
	if len(results) == 0 {
		t.Error("expected some results even with error")
	}

	// Verify the error is a TaskError
	if !IsTaskError(err) {
		t.Errorf("expected TaskError from execution, got %T: %v", err, err)
	}
}

// mockFailingExecutor fails on a specific task.
type mockFailingExecutor struct {
	failOnTask string
}

func (m *mockFailingExecutor) Execute(ctx context.Context, task models.Task) (models.TaskResult, error) {
	if task.Number == m.failOnTask {
		taskErr := NewTaskError(task.Number, "task execution failed", errors.New("simulated failure"))
		return models.TaskResult{
			Task:   task,
			Status: models.StatusFailed,
			Error:  taskErr,
		}, taskErr
	}
	return models.TaskResult{
		Task:   task,
		Status: models.StatusGreen,
	}, nil
}

// TestWaveExecutor_LogsTaskResults verifies that LogTaskResult is called for each task result.
// This test ensures that individual task execution results are logged to .conductor/logs/tasks/
func TestWaveExecutor_LogsTaskResults(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Prompt: "Do task 1"},
			{Number: "2", Name: "Task 2", Prompt: "Do task 2"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2"}, MaxConcurrency: 2},
		},
	}

	mockExecutor := newSequentialMockExecutor()
	mockLog := &mockLoggerWithTaskLogging{}
	waveExecutor := NewWaveExecutor(mockExecutor, mockLog)

	results, err := waveExecutor.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan returned error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Verify LogTaskResult was called for each task
	if len(mockLog.taskResultCalls) != 2 {
		t.Errorf("expected LogTaskResult called 2 times, got %d", len(mockLog.taskResultCalls))
	}

	// Verify task numbers match
	taskNumbersSeen := make(map[string]bool)
	for _, result := range mockLog.taskResultCalls {
		taskNumbersSeen[result.Task.Number] = true
	}
	if !taskNumbersSeen["1"] || !taskNumbersSeen["2"] {
		t.Errorf("expected to see tasks 1 and 2, got %v", taskNumbersSeen)
	}
}

// TestWaveExecutor_LogsAllTasksInWave verifies all N tasks in a wave get logged.
func TestWaveExecutor_LogsAllTasksInWave(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Prompt: "Do task 1"},
			{Number: "2", Name: "Task 2", Prompt: "Do task 2"},
			{Number: "3", Name: "Task 3", Prompt: "Do task 3"},
			{Number: "4", Name: "Task 4", Prompt: "Do task 4"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2", "3", "4"}, MaxConcurrency: 2},
		},
	}

	mockExecutor := newConcurrencyMockExecutor(5 * time.Millisecond)
	mockLog := &mockLoggerWithTaskLogging{}
	waveExecutor := NewWaveExecutor(mockExecutor, mockLog)

	results, err := waveExecutor.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan returned error: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	// Verify LogTaskResult was called for all 4 tasks
	if len(mockLog.taskResultCalls) != 4 {
		t.Errorf("expected LogTaskResult called 4 times, got %d", len(mockLog.taskResultCalls))
	}
}

// TestWaveExecutor_LogsTaskResultsAcrossMultipleWaves verifies task logging across multiple waves.
func TestWaveExecutor_LogsTaskResultsAcrossMultipleWaves(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Prompt: "Do task 1"},
			{Number: "2", Name: "Task 2", Prompt: "Do task 2"},
			{Number: "3", Name: "Task 3", Prompt: "Do task 3"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2"}, MaxConcurrency: 2},
			{Name: "Wave 2", TaskNumbers: []string{"3"}, MaxConcurrency: 1},
		},
	}

	mockExecutor := newSequentialMockExecutor()
	mockLog := &mockLoggerWithTaskLogging{}
	waveExecutor := NewWaveExecutor(mockExecutor, mockLog)

	results, err := waveExecutor.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan returned error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Verify LogTaskResult was called for all 3 tasks across both waves
	if len(mockLog.taskResultCalls) != 3 {
		t.Errorf("expected LogTaskResult called 3 times, got %d", len(mockLog.taskResultCalls))
	}
}

// mockLoggerWithTaskLogging extends mockLogger with task result logging.
type mockLoggerWithTaskLogging struct {
	mockLogger
	taskResultCalls []models.TaskResult
}

func (m *mockLoggerWithTaskLogging) LogTaskResult(result models.TaskResult) error {
	m.taskResultCalls = append(m.taskResultCalls, result)
	return nil
}

// TestWaveExecutor_NoCancellationLoggingIfNoTasks verifies that when a context is
// pre-cancelled before ExecutePlan is called, no wave logging occurs because no tasks
// are launched. This tests the tasksLaunched counter fix in wave.go.
func TestWaveExecutor_NoCancellationLoggingIfNoTasks(t *testing.T) {
	// Create a pre-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately before any execution

	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Prompt: "Do task 1"},
			{Number: "2", Name: "Task 2", Prompt: "Do task 2"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2"}, MaxConcurrency: 2},
		},
	}

	mockExecutor := newSequentialMockExecutor()
	mockLog := &mockLogger{}
	waveExecutor := NewWaveExecutor(mockExecutor, mockLog)

	// Execute with pre-cancelled context
	results, err := waveExecutor.ExecutePlan(ctx, plan)

	// Verify error is context.Canceled
	if err == nil {
		t.Fatalf("expected context cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}

	// Verify no results were returned (no tasks launched)
	if len(results) != 0 {
		t.Errorf("expected 0 results when context pre-cancelled, got %d", len(results))
	}

	// Critical verification: LogWaveStart should NOT be called
	if len(mockLog.waveStartCalls) != 0 {
		t.Errorf("expected 0 LogWaveStart calls when context pre-cancelled, got %d", len(mockLog.waveStartCalls))
	}

	// Critical verification: LogWaveComplete should NOT be called
	if len(mockLog.waveCompleteCalls) != 0 {
		t.Errorf("expected 0 LogWaveComplete calls when context pre-cancelled, got %d", len(mockLog.waveCompleteCalls))
	}
}

// TestWaveExecutor_SkipsCompletedTasks verifies that completed tasks are skipped when skipCompleted is true.
func TestWaveExecutor_SkipsCompletedTasks(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Prompt: "Do task 1", Status: "completed"},
			{Number: "2", Name: "Task 2", Prompt: "Do task 2"},
			{Number: "3", Name: "Task 3", Prompt: "Do task 3", Status: "completed"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2", "3"}, MaxConcurrency: 3},
		},
	}

	mockExecutor := newSequentialMockExecutor()
	mockLog := &mockLoggerWithTaskLogging{}
	waveExecutor := NewWaveExecutorWithConfig(mockExecutor, mockLog, true, false)

	results, err := waveExecutor.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan returned error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Verify skipped tasks have GREEN status and "Skipped" output
	if results[0].Status != models.StatusGreen {
		t.Errorf("expected skipped task 1 status GREEN, got %s", results[0].Status)
	}
	if results[0].Output != "Skipped" {
		t.Errorf("expected skipped task 1 output 'Skipped', got %s", results[0].Output)
	}

	// Verify task 2 was executed (GREEN status but not "Skipped")
	if results[1].Status != models.StatusGreen {
		t.Errorf("expected executed task 2 status GREEN, got %s", results[1].Status)
	}
	if results[1].Output == "Skipped" {
		t.Errorf("expected task 2 not to be marked as skipped")
	}

	// Verify skipped task 3
	if results[2].Status != models.StatusGreen {
		t.Errorf("expected skipped task 3 status GREEN, got %s", results[2].Status)
	}
	if results[2].Output != "Skipped" {
		t.Errorf("expected skipped task 3 output 'Skipped', got %s", results[2].Output)
	}
}

// TestWaveExecutor_DoesNotSkipFailedStatus verifies that failed tasks are not skipped by CanSkip.
func TestWaveExecutor_DoesNotSkipFailedStatus(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Prompt: "Do task 1", Status: "failed"},
			{Number: "2", Name: "Task 2", Prompt: "Do task 2"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2"}, MaxConcurrency: 2},
		},
	}

	mockExecutor := newSequentialMockExecutor()
	waveExecutor := NewWaveExecutorWithConfig(mockExecutor, nil, true, false)

	results, err := waveExecutor.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan returned error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Verify failed task is NOT skipped (CanSkip returns false for failed status)
	if results[0].Output == "Skipped" {
		t.Errorf("expected failed task NOT to be skipped")
	}
}

// TestWaveExecutor_DoesNotSkipWhenDisabled verifies that skipCompleted=false prevents skipping.
func TestWaveExecutor_DoesNotSkipWhenDisabled(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Prompt: "Do task 1", Status: "completed"},
			{Number: "2", Name: "Task 2", Prompt: "Do task 2"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2"}, MaxConcurrency: 2},
		},
	}

	mockExecutor := newSequentialMockExecutor()
	waveExecutor := NewWaveExecutorWithConfig(mockExecutor, nil, false, false) // skipCompleted=false

	results, err := waveExecutor.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan returned error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Both tasks should be executed (not skipped)
	// We check that both executed (they won't have "Skipped" output)
	for i, result := range results {
		if result.Output == "Skipped" {
			t.Errorf("task %d should not be skipped when skipCompleted=false", i+1)
		}
	}
}

// TestWaveExecutor_AllTasksSkipped verifies behavior when all tasks in a wave are skipped.
func TestWaveExecutor_AllTasksSkipped(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Prompt: "Do task 1", Status: "completed"},
			{Number: "2", Name: "Task 2", Prompt: "Do task 2", Status: "completed"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2"}, MaxConcurrency: 2},
		},
	}

	mockExecutor := newSequentialMockExecutor()
	mockLog := &mockLoggerWithTaskLogging{}
	waveExecutor := NewWaveExecutorWithConfig(mockExecutor, mockLog, true, false)

	results, err := waveExecutor.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan returned error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// All results should be GREEN with "Skipped" output
	for i, result := range results {
		if result.Status != models.StatusGreen {
			t.Errorf("task %d should be GREEN, got %s", i+1, result.Status)
		}
		if result.Output != "Skipped" {
			t.Errorf("task %d should be marked as Skipped, got %s", i+1, result.Output)
		}
	}

	// LogTaskResult should still be called for skipped tasks
	if len(mockLog.taskResultCalls) != 2 {
		t.Errorf("expected LogTaskResult called 2 times for skipped tasks, got %d", len(mockLog.taskResultCalls))
	}
}

// TestWaveExecutor_MixedSkippedAndExecuted verifies ordering of skipped and executed tasks.
func TestWaveExecutor_MixedSkippedAndExecuted(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Prompt: "Do task 1", Status: "completed"},
			{Number: "2", Name: "Task 2", Prompt: "Do task 2"},
			{Number: "3", Name: "Task 3", Prompt: "Do task 3", Status: "completed"},
			{Number: "4", Name: "Task 4", Prompt: "Do task 4"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2", "3", "4"}, MaxConcurrency: 4},
		},
	}

	mockExecutor := newConcurrencyMockExecutor(1 * time.Millisecond)
	waveExecutor := NewWaveExecutorWithConfig(mockExecutor, nil, true, false)

	results, err := waveExecutor.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan returned error: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	// Verify task 1 is skipped
	if results[0].Output != "Skipped" {
		t.Errorf("expected task 1 to be skipped")
	}

	// Verify task 3 is skipped
	if results[2].Output != "Skipped" {
		t.Errorf("expected task 3 to be skipped")
	}

	// Verify tasks 2 and 4 are executed (no "Skipped" output)
	if results[1].Output == "Skipped" {
		t.Errorf("expected task 2 to be executed, not skipped")
	}
	if results[3].Output == "Skipped" {
		t.Errorf("expected task 4 to be executed, not skipped")
	}
}
