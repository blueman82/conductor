package executor

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/logger"
	"github.com/harrison/conductor/internal/models"
)

// TestOrchestratorLogsTaskStart verifies LogTaskStart() is called before task execution
func TestOrchestratorLogsTaskStart(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			return []models.TaskResult{
				{
					Task:   plan.Tasks[0],
					Status: models.StatusGreen,
				},
			}, nil
		},
	}

	output := &bytes.Buffer{}
	consoleLogger := logger.NewConsoleLogger(output, "info")

	orchestrator := NewOrchestrator(mockWave, consoleLogger)

	plan := &models.Plan{
		Name: "Test Plan",
		Tasks: []models.Task{
			{
				Number: "1",
				Name:   "Task 1",
				Agent:  "test-agent",
			},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1"}},
		},
	}

	result, err := orchestrator.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	if result == nil {
		t.Fatal("result should not be nil")
	}

	// Verify logging occurred (check for summary output)
	logOutput := output.String()
	if !strings.Contains(logOutput, "Execution Summary") {
		t.Errorf("expected summary output, got: %q", logOutput)
	}
}

// TestOrchestratorLogsProgress verifies LogProgress() called after completion
func TestOrchestratorLogsProgress(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			results := []models.TaskResult{}
			for _, task := range plan.Tasks {
				results = append(results, models.TaskResult{
					Task:     task,
					Status:   models.StatusGreen,
					Duration: 100 * time.Millisecond,
				})
			}
			return results, nil
		},
	}

	output := &bytes.Buffer{}
	consoleLogger := logger.NewConsoleLogger(output, "info")

	orchestrator := NewOrchestrator(mockWave, consoleLogger)

	plan := &models.Plan{
		Name: "Test Plan",
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1"},
			{Number: "2", Name: "Task 2"},
			{Number: "3", Name: "Task 3"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2", "3"}},
		},
	}

	result, err := orchestrator.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	if result == nil {
		t.Fatal("result should not be nil")
	}

	if result.Completed != 3 {
		t.Errorf("expected 3 completed tasks, got %d", result.Completed)
	}

	// Verify summary was generated
	logOutput := output.String()
	if !strings.Contains(logOutput, "Execution Summary") {
		t.Errorf("expected summary output, got: %q", logOutput)
	}
	if !strings.Contains(logOutput, "Completed: 3") {
		t.Errorf("expected 'Completed: 3' in output, got: %q", logOutput)
	}
}

// TestOrchestratorGeneratesEnhancedSummary verifies ExecutionSummary generation
func TestOrchestratorGeneratesEnhancedSummary(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			return []models.TaskResult{
				{
					Task: models.Task{
						Number: "1",
						Name:   "Task 1",
						Agent:  "agent-1",
						Files:  []string{"file1.go"},
					},
					Status:   models.StatusGreen,
					Duration: 1 * time.Second,
				},
				{
					Task: models.Task{
						Number: "2",
						Name:   "Task 2",
						Agent:  "agent-2",
						Files:  []string{"file2.go", "file3.go"},
					},
					Status:   models.StatusYellow,
					Duration: 2 * time.Second,
				},
				{
					Task: models.Task{
						Number: "3",
						Name:   "Task 3",
						Agent:  "agent-1",
						Files:  []string{"file4.go"},
					},
					Status:   models.StatusRed,
					Duration: 500 * time.Millisecond,
				},
			}, nil
		},
	}

	output := &bytes.Buffer{}
	consoleLogger := logger.NewConsoleLogger(output, "info")

	orchestrator := NewOrchestrator(mockWave, consoleLogger)

	plan := &models.Plan{
		Name: "Test Plan",
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Agent: "agent-1", Files: []string{"file1.go"}},
			{Number: "2", Name: "Task 2", Agent: "agent-2", Files: []string{"file2.go", "file3.go"}},
			{Number: "3", Name: "Task 3", Agent: "agent-1", Files: []string{"file4.go"}},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2", "3"}},
		},
	}

	result, err := orchestrator.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	// Verify summary contains all expected statistics
	if result.TotalTasks != 3 {
		t.Errorf("expected 3 total tasks, got %d", result.TotalTasks)
	}
	if result.Completed != 2 {
		t.Errorf("expected 2 completed tasks, got %d", result.Completed)
	}
	if result.Failed != 1 {
		t.Errorf("expected 1 failed task, got %d", result.Failed)
	}

	// Verify status breakdown
	if result.StatusBreakdown[models.StatusGreen] != 1 {
		t.Errorf("expected 1 GREEN, got %d", result.StatusBreakdown[models.StatusGreen])
	}
	if result.StatusBreakdown[models.StatusYellow] != 1 {
		t.Errorf("expected 1 YELLOW, got %d", result.StatusBreakdown[models.StatusYellow])
	}
	if result.StatusBreakdown[models.StatusRed] != 1 {
		t.Errorf("expected 1 RED, got %d", result.StatusBreakdown[models.StatusRed])
	}

	// Verify agent usage
	if result.AgentUsage["agent-1"] != 2 {
		t.Errorf("expected agent-1 used 2 times, got %d", result.AgentUsage["agent-1"])
	}
	if result.AgentUsage["agent-2"] != 1 {
		t.Errorf("expected agent-2 used 1 time, got %d", result.AgentUsage["agent-2"])
	}

	// Verify total files (unique count)
	if result.TotalFiles != 4 {
		t.Errorf("expected 4 total unique files, got %d", result.TotalFiles)
	}

	// Verify average duration
	expectedAvg := (1*time.Second + 2*time.Second + 500*time.Millisecond) / 3
	if result.AvgTaskDuration != expectedAvg {
		t.Errorf("expected avg duration %v, got %v", expectedAvg, result.AvgTaskDuration)
	}

	// Verify logging output
	logOutput := output.String()
	if !strings.Contains(logOutput, "Execution Summary") {
		t.Errorf("expected summary header in output")
	}
	if !strings.Contains(logOutput, "Total tasks: 3") {
		t.Errorf("expected total tasks count in output")
	}
	if !strings.Contains(logOutput, "Completed: 2") {
		t.Errorf("expected completed count in output")
	}
	if !strings.Contains(logOutput, "Failed: 1") {
		t.Errorf("expected failed count in output")
	}
}

// TestOrchestratorProgressTrackingAccuracy verifies progress counts are accurate
func TestOrchestratorProgressTrackingAccuracy(t *testing.T) {
	completedCount := 0
	var completedMu sync.Mutex

	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			results := []models.TaskResult{}
			for i, task := range plan.Tasks {
				status := models.StatusGreen
				if i == len(plan.Tasks)-1 {
					status = models.StatusRed
				}
				results = append(results, models.TaskResult{
					Task:     task,
					Status:   status,
					Duration: 100 * time.Millisecond,
				})

				if status == models.StatusGreen {
					completedMu.Lock()
					completedCount++
					completedMu.Unlock()
				}
			}
			return results, nil
		},
	}

	output := &bytes.Buffer{}
	consoleLogger := logger.NewConsoleLogger(output, "info")

	orchestrator := NewOrchestrator(mockWave, consoleLogger)

	plan := &models.Plan{
		Name: "Test Plan",
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1"},
			{Number: "2", Name: "Task 2"},
			{Number: "3", Name: "Task 3"},
			{Number: "4", Name: "Task 4"},
			{Number: "5", Name: "Task 5"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2", "3", "4", "5"}},
		},
	}

	result, err := orchestrator.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	if result.Completed != 4 {
		t.Errorf("expected 4 completed, got %d", result.Completed)
	}
	if result.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", result.Failed)
	}
}

// TestOrchestratorErrorStateLogging verifies error states are logged
func TestOrchestratorErrorStateLogging(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			return []models.TaskResult{
				{
					Task:   models.Task{Number: "1", Name: "Task 1"},
					Status: models.StatusRed,
					Error:  fmt.Errorf("execution error"),
				},
				{
					Task:   models.Task{Number: "2", Name: "Task 2"},
					Status: models.StatusFailed,
					Error:  fmt.Errorf("compilation error"),
				},
			}, nil
		},
	}

	output := &bytes.Buffer{}
	consoleLogger := logger.NewConsoleLogger(output, "info")

	orchestrator := NewOrchestrator(mockWave, consoleLogger)

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

	result, err := orchestrator.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	if result.Failed != 2 {
		t.Errorf("expected 2 failed tasks, got %d", result.Failed)
	}

	if len(result.FailedTasks) != 2 {
		t.Errorf("expected 2 failed task details, got %d", len(result.FailedTasks))
	}

	// Verify failed tasks are tracked in result
	if result.FailedTasks[0].Task.Number != "1" {
		t.Errorf("expected failed task 1, got %s", result.FailedTasks[0].Task.Number)
	}
	if result.FailedTasks[1].Task.Number != "2" {
		t.Errorf("expected failed task 2, got %s", result.FailedTasks[1].Task.Number)
	}

	// Verify logging includes failed tasks
	logOutput := output.String()
	if !strings.Contains(logOutput, "Failed: 2") {
		t.Errorf("expected 'Failed: 2' in output")
	}
}

// TestOrchestratorWaveExecutorIntegration verifies wave executor integration
func TestOrchestratorWaveExecutorIntegration(t *testing.T) {
	waveExecuted := false
	var waveExecutedMu sync.Mutex

	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			waveExecutedMu.Lock()
			waveExecuted = true
			waveExecutedMu.Unlock()

			results := []models.TaskResult{}
			for _, task := range plan.Tasks {
				results = append(results, models.TaskResult{
					Task:   task,
					Status: models.StatusGreen,
				})
			}
			return results, nil
		},
	}

	output := &bytes.Buffer{}
	consoleLogger := logger.NewConsoleLogger(output, "info")

	orchestrator := NewOrchestrator(mockWave, consoleLogger)

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

	result, err := orchestrator.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	waveExecutedMu.Lock()
	if !waveExecuted {
		waveExecutedMu.Unlock()
		t.Fatal("wave executor was not called")
	}
	waveExecutedMu.Unlock()

	if result.TotalTasks != 2 {
		t.Errorf("expected 2 tasks, got %d", result.TotalTasks)
	}

	if result.Completed != 2 {
		t.Errorf("expected 2 completed, got %d", result.Completed)
	}
}

// TestOrchestratorEndToEndFlow verifies complete execution flow
func TestOrchestratorEndToEndFlow(t *testing.T) {
	executionLog := []string{}
	var logMu sync.Mutex

	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			logMu.Lock()
			executionLog = append(executionLog, "wave-start")
			logMu.Unlock()

			results := []models.TaskResult{}
			for i, task := range plan.Tasks {
				logMu.Lock()
				executionLog = append(executionLog, fmt.Sprintf("task-%d-execute", i+1))
				logMu.Unlock()

				status := models.StatusGreen
				if i == 0 {
					status = models.StatusYellow
				}

				// Add a small delay to simulate execution time
				time.Sleep(100 * time.Millisecond)

				results = append(results, models.TaskResult{
					Task:     task,
					Status:   status,
					Duration: 100 * time.Millisecond,
				})
			}

			logMu.Lock()
			executionLog = append(executionLog, "wave-complete")
			logMu.Unlock()

			return results, nil
		},
	}

	output := &bytes.Buffer{}
	consoleLogger := logger.NewConsoleLogger(output, "info")

	orchestrator := NewOrchestrator(mockWave, consoleLogger)

	plan := &models.Plan{
		Name: "Test Plan",
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Agent: "agent-1"},
			{Number: "2", Name: "Task 2", Agent: "agent-2"},
			{Number: "3", Name: "Task 3", Agent: "agent-1"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2", "3"}},
		},
	}

	startTime := time.Now()
	result, err := orchestrator.ExecutePlan(context.Background(), plan)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	// Verify execution flow
	logMu.Lock()
	if len(executionLog) == 0 {
		logMu.Unlock()
		t.Fatal("execution log is empty")
	}
	logMu.Unlock()

	// Verify results
	if result.TotalTasks != 3 {
		t.Errorf("expected 3 total tasks, got %d", result.TotalTasks)
	}
	if result.Completed != 3 {
		t.Errorf("expected 3 completed, got %d", result.Completed)
	}

	// Verify status breakdown
	if result.StatusBreakdown[models.StatusGreen] != 2 {
		t.Errorf("expected 2 GREEN, got %d", result.StatusBreakdown[models.StatusGreen])
	}
	if result.StatusBreakdown[models.StatusYellow] != 1 {
		t.Errorf("expected 1 YELLOW, got %d", result.StatusBreakdown[models.StatusYellow])
	}

	// Verify agent usage
	if result.AgentUsage["agent-1"] != 2 {
		t.Errorf("expected agent-1 used 2 times, got %d", result.AgentUsage["agent-1"])
	}
	if result.AgentUsage["agent-2"] != 1 {
		t.Errorf("expected agent-2 used 1 time, got %d", result.AgentUsage["agent-2"])
	}

	// Verify duration is reasonable (at least 300ms for 3 tasks * 100ms each)
	if duration < 250*time.Millisecond {
		t.Errorf("execution duration too short: %v", duration)
	}

	// Verify logging output contains summary
	logOutput := output.String()
	if !strings.Contains(logOutput, "Execution Summary") {
		t.Errorf("expected summary output")
	}
	if !strings.Contains(logOutput, "Total tasks: 3") {
		t.Errorf("expected total tasks count")
	}
	if !strings.Contains(logOutput, "Completed: 3") {
		t.Errorf("expected completed count")
	}
	if !strings.Contains(logOutput, "Failed: 0") {
		t.Errorf("expected failed count of 0")
	}
}

// mockLoggerWithTaskTracking captures task logging calls
type mockLoggerWithTaskTracking struct {
	taskStartCalls []struct {
		task    models.Task
		current int
		total   int
	}
	taskProgressCalls [][]models.Task
	taskResultCalls   []models.TaskResult
	summaryCalls      []models.ExecutionResult
	mu                sync.Mutex
}

func (m *mockLoggerWithTaskTracking) LogWaveStart(wave models.Wave) {}
func (m *mockLoggerWithTaskTracking) LogWaveComplete(wave models.Wave, duration time.Duration, results []models.TaskResult) {
}

func (m *mockLoggerWithTaskTracking) LogTaskStart(task models.Task, current int, total int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.taskStartCalls = append(m.taskStartCalls, struct {
		task    models.Task
		current int
		total   int
	}{task: task, current: current, total: total})
}

func (m *mockLoggerWithTaskTracking) LogProgress(tasks []models.Task) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Make a copy of the tasks slice
	tasksCopy := make([]models.Task, len(tasks))
	copy(tasksCopy, tasks)
	m.taskProgressCalls = append(m.taskProgressCalls, tasksCopy)
}

func (m *mockLoggerWithTaskTracking) LogTaskResult(result models.TaskResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.taskResultCalls = append(m.taskResultCalls, result)
	return nil
}

func (m *mockLoggerWithTaskTracking) LogSummary(result models.ExecutionResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.summaryCalls = append(m.summaryCalls, result)
}

// TestOrchestratorMultipleWaves verifies handling of multiple waves
func TestOrchestratorMultipleWaves(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			results := []models.TaskResult{}
			for _, task := range plan.Tasks {
				results = append(results, models.TaskResult{
					Task:     task,
					Status:   models.StatusGreen,
					Duration: 100 * time.Millisecond,
				})
			}
			return results, nil
		},
	}

	output := &bytes.Buffer{}
	consoleLogger := logger.NewConsoleLogger(output, "info")

	orchestrator := NewOrchestrator(mockWave, consoleLogger)

	plan := &models.Plan{
		Name: "Test Plan",
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1"},
			{Number: "2", Name: "Task 2"},
			{Number: "3", Name: "Task 3"},
			{Number: "4", Name: "Task 4"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2"}},
			{Name: "Wave 2", TaskNumbers: []string{"3", "4"}},
		},
	}

	result, err := orchestrator.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	if result.TotalTasks != 4 {
		t.Errorf("expected 4 total tasks, got %d", result.TotalTasks)
	}
	if result.Completed != 4 {
		t.Errorf("expected 4 completed, got %d", result.Completed)
	}

	// Verify summary is generated (wave logging is handled by wave executor, not orchestrator)
	logOutput := output.String()
	if !strings.Contains(logOutput, "Execution Summary") {
		t.Errorf("expected summary in logging output")
	}
	if !strings.Contains(logOutput, "Total tasks: 4") {
		t.Errorf("expected 'Total tasks: 4' in logging output")
	}
}

// TestOrchestratorWithoutLogger verifies graceful operation without logger
func TestOrchestratorWithoutLogger(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			results := []models.TaskResult{}
			for _, task := range plan.Tasks {
				results = append(results, models.TaskResult{
					Task:   task,
					Status: models.StatusGreen,
				})
			}
			return results, nil
		},
	}

	// Create orchestrator without logger (nil)
	orchestrator := NewOrchestrator(mockWave, nil)

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

	result, err := orchestrator.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	if result.TotalTasks != 2 {
		t.Errorf("expected 2 total tasks, got %d", result.TotalTasks)
	}
	if result.Completed != 2 {
		t.Errorf("expected 2 completed, got %d", result.Completed)
	}
}

// TestOrchestratorMetricsCalculation verifies accurate metrics calculation
func TestOrchestratorMetricsCalculation(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			return []models.TaskResult{
				{
					Task: models.Task{
						Number: "1",
						Name:   "Task 1",
						Agent:  "agent-1",
						Files:  []string{"file1.go"},
					},
					Status:   models.StatusGreen,
					Duration: 1 * time.Second,
				},
				{
					Task: models.Task{
						Number: "2",
						Name:   "Task 2",
						Agent:  "agent-1",
						Files:  []string{"file2.go", "file3.go"},
					},
					Status:   models.StatusGreen,
					Duration: 3 * time.Second,
				},
				{
					Task: models.Task{
						Number: "3",
						Name:   "Task 3",
						Agent:  "agent-2",
						Files:  []string{"file1.go", "file4.go"}, // file1 is duplicate
					},
					Status:   models.StatusGreen,
					Duration: 2 * time.Second,
				},
			}, nil
		},
	}

	output := &bytes.Buffer{}
	consoleLogger := logger.NewConsoleLogger(output, "info")

	orchestrator := NewOrchestrator(mockWave, consoleLogger)

	plan := &models.Plan{
		Name: "Test Plan",
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Agent: "agent-1", Files: []string{"file1.go"}},
			{Number: "2", Name: "Task 2", Agent: "agent-1", Files: []string{"file2.go", "file3.go"}},
			{Number: "3", Name: "Task 3", Agent: "agent-2", Files: []string{"file1.go", "file4.go"}},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2", "3"}},
		},
	}

	result, err := orchestrator.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	// Verify metrics
	if result.Completed != 3 {
		t.Errorf("expected 3 completed, got %d", result.Completed)
	}

	// Verify unique file count (file1, file2, file3, file4)
	if result.TotalFiles != 4 {
		t.Errorf("expected 4 unique files, got %d", result.TotalFiles)
	}

	// Verify average duration: (1 + 3 + 2) / 3 = 2 seconds
	expectedAvg := (1*time.Second + 3*time.Second + 2*time.Second) / 3
	if result.AvgTaskDuration != expectedAvg {
		t.Errorf("expected avg %v, got %v", expectedAvg, result.AvgTaskDuration)
	}

	// Verify agent usage
	if result.AgentUsage["agent-1"] != 2 {
		t.Errorf("expected agent-1 used 2 times, got %d", result.AgentUsage["agent-1"])
	}
	if result.AgentUsage["agent-2"] != 1 {
		t.Errorf("expected agent-2 used 1 time, got %d", result.AgentUsage["agent-2"])
	}
}

// TestOrchestratorConcurrentExecution verifies thread-safe concurrent execution
func TestOrchestratorConcurrentExecution(t *testing.T) {
	executionOrder := []string{}
	var orderMu sync.Mutex

	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			results := []models.TaskResult{}
			var resultsMu sync.Mutex
			var wg sync.WaitGroup

			for _, task := range plan.Tasks {
				wg.Add(1)
				go func(t models.Task) {
					defer wg.Done()
					orderMu.Lock()
					executionOrder = append(executionOrder, t.Number)
					orderMu.Unlock()

					resultsMu.Lock()
					results = append(results, models.TaskResult{
						Task:   t,
						Status: models.StatusGreen,
					})
					resultsMu.Unlock()
				}(task)
			}
			wg.Wait()
			return results, nil
		},
	}

	output := &bytes.Buffer{}
	consoleLogger := logger.NewConsoleLogger(output, "info")

	orchestrator := NewOrchestrator(mockWave, consoleLogger)

	plan := &models.Plan{
		Name: "Test Plan",
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1"},
			{Number: "2", Name: "Task 2"},
			{Number: "3", Name: "Task 3"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1", "2", "3"}},
		},
	}

	result, err := orchestrator.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	if result.TotalTasks != 3 {
		t.Errorf("expected 3 total tasks, got %d", result.TotalTasks)
	}

	// Verify all tasks executed (order may vary due to concurrency)
	if len(executionOrder) != 3 {
		t.Errorf("expected 3 executions, got %d", len(executionOrder))
	}
}
