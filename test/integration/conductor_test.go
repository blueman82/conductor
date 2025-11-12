package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/executor"
	"github.com/harrison/conductor/internal/models"
	"github.com/harrison/conductor/internal/parser"
)

func TestParseAndValidateSimpleMarkdown(t *testing.T) {
	planPath := filepath.Join("fixtures", "simple-plan.md")

	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("Failed to parse plan: %v", err)
	}

	if len(plan.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(plan.Tasks))
	}

	if plan.Tasks[0].Number != "1" || plan.Tasks[0].Name != "First Task" {
		t.Errorf("First task not parsed correctly")
	}

	if plan.Tasks[1].DependsOn[0] != "1" {
		t.Error("Task 2 dependency incorrect")
	}
}

func TestParseAndValidateSimpleYAML(t *testing.T) {
	planPath := filepath.Join("fixtures", "simple-plan.yaml")

	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("Failed to parse YAML plan: %v", err)
	}

	if len(plan.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(plan.Tasks))
	}
}

func TestCalculateWavesSimplePlan(t *testing.T) {
	planPath := filepath.Join("fixtures", "simple-plan.md")
	plan, _ := parser.ParseFile(planPath)

	waves, err := executor.CalculateWaves(plan.Tasks)
	if err != nil {
		t.Fatalf("Failed to calculate waves: %v", err)
	}

	if len(waves) != 2 {
		t.Errorf("Expected 2 waves, got %d", len(waves))
	}
}

func TestCalculateWavesMultiWave(t *testing.T) {
	planPath := filepath.Join("fixtures", "multi-wave.md")
	plan, _ := parser.ParseFile(planPath)

	waves, err := executor.CalculateWaves(plan.Tasks)
	if err != nil {
		t.Fatalf("Failed to calculate waves: %v", err)
	}

	if len(waves) != 4 {
		t.Errorf("Expected 4 waves, got %d", len(waves))
	}
}

func TestCyclicDependencyDetection(t *testing.T) {
	planPath := filepath.Join("fixtures", "cyclic.md")
	plan, _ := parser.ParseFile(planPath)

	_, err := executor.CalculateWaves(plan.Tasks)
	if err == nil {
		t.Error("Expected error for cyclic dependencies")
	}
}

func TestInvalidDependencies(t *testing.T) {
	planPath := filepath.Join("fixtures", "invalid.md")
	plan, _ := parser.ParseFile(planPath)

	_, err := executor.CalculateWaves(plan.Tasks)
	if err == nil {
		t.Error("Expected error for invalid dependency")
	}
}

func TestDryRunValidation(t *testing.T) {
	planPath := filepath.Join("fixtures", "simple-plan.md")
	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("Failed to parse plan: %v", err)
	}

	waves, err := executor.CalculateWaves(plan.Tasks)
	if err != nil {
		t.Fatalf("Failed to calculate waves: %v", err)
	}

	if len(waves) == 0 {
		t.Error("Expected waves")
	}
}

func TestTimeoutContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	select {
	case <-time.After(200 * time.Millisecond):
		t.Error("Expected timeout")
	case <-ctx.Done():
		// OK
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	select {
	case <-ctx.Done():
		// OK
	default:
		t.Error("Expected context to be cancelled")
	}
}

func TestMultipleFormatParsing(t *testing.T) {
	tests := []struct {
		name      string
		fixture   string
		taskCount int
	}{
		{"Markdown", filepath.Join("fixtures", "simple-plan.md"), 2},
		{"YAML", filepath.Join("fixtures", "simple-plan.yaml"), 2},
		{"MultiWave", filepath.Join("fixtures", "multi-wave.md"), 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := parser.ParseFile(tt.fixture)
			if err != nil {
				t.Fatalf("ParseFile() error = %v", err)
			}

			if len(plan.Tasks) != tt.taskCount {
				t.Errorf("Expected %d tasks, got %d", tt.taskCount, len(plan.Tasks))
			}
		})
	}
}

func TestWaveExecutionOrder(t *testing.T) {
	planPath := filepath.Join("fixtures", "multi-wave.md")
	plan, _ := parser.ParseFile(planPath)
	waves, _ := executor.CalculateWaves(plan.Tasks)

	expectedWaveStructure := []int{1, 2, 1, 1}
	for i, wave := range waves {
		if i >= len(expectedWaveStructure) {
			break
		}
		if len(wave.TaskNumbers) != expectedWaveStructure[i] {
			t.Errorf("Wave %d: expected %d tasks, got %d", i+1, expectedWaveStructure[i], len(wave.TaskNumbers))
		}
	}
}

func TestFileExists(t *testing.T) {
	fixtures := []string{
		"fixtures/simple-plan.md",
		"fixtures/simple-plan.yaml",
		"fixtures/multi-wave.md",
		"fixtures/cyclic.md",
		"fixtures/invalid.md",
	}

	for _, fixture := range fixtures {
		if _, err := os.Stat(fixture); os.IsNotExist(err) {
			t.Errorf("Fixture file missing: %s", fixture)
		}
	}
}

// ============================================================================
// Skip-Completed Feature Integration Tests
// ============================================================================

// TestSkipMarkdownCompletedTasks tests parsing and skipping of completed tasks in Markdown
func TestSkipMarkdownCompletedTasks(t *testing.T) {
	planPath := filepath.Join("fixtures", "skip-markdown-completed.md")

	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("Failed to parse plan: %v", err)
	}

	if len(plan.Tasks) != 4 {
		t.Errorf("Expected 4 tasks, got %d", len(plan.Tasks))
	}

	// Verify status extraction
	if !plan.Tasks[0].IsCompleted() {
		t.Error("Task 1 should be completed")
	}
	if plan.Tasks[1].IsCompleted() {
		t.Error("Task 2 should not be completed")
	}
	if !plan.Tasks[2].IsCompleted() {
		t.Error("Task 3 should be completed")
	}
	if plan.Tasks[3].IsCompleted() {
		t.Error("Task 4 should not be completed")
	}

	// Verify CanSkip logic
	if !plan.Tasks[0].CanSkip() {
		t.Error("Task 1 should be skippable (completed)")
	}
	if plan.Tasks[1].CanSkip() {
		t.Error("Task 2 should not be skippable (pending)")
	}
	if !plan.Tasks[2].CanSkip() {
		t.Error("Task 3 should be skippable (completed)")
	}
	if plan.Tasks[3].CanSkip() {
		t.Error("Task 4 should not be skippable (pending)")
	}
}

// TestSkipYAMLAllCompleted tests YAML plan with all tasks marked completed
func TestSkipYAMLAllCompleted(t *testing.T) {
	planPath := filepath.Join("fixtures", "skip-yaml-all-completed.yaml")

	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("Failed to parse YAML plan: %v", err)
	}

	if len(plan.Tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(plan.Tasks))
	}

	// All tasks should be completed
	for i, task := range plan.Tasks {
		if !task.IsCompleted() {
			t.Errorf("Task %d should be completed", i+1)
		}
		if !task.CanSkip() {
			t.Errorf("Task %d should be skippable", i+1)
		}
		// Verify timestamp extraction
		if task.CompletedAt == nil {
			t.Errorf("Task %d should have CompletedAt timestamp", i+1)
		}
	}
}

// TestSkipYAMLMixedStatus tests YAML plan with mixed task statuses
func TestSkipYAMLMixedStatus(t *testing.T) {
	planPath := filepath.Join("fixtures", "skip-yaml-mixed-status.yaml")

	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("Failed to parse YAML plan: %v", err)
	}

	if len(plan.Tasks) != 5 {
		t.Errorf("Expected 5 tasks, got %d", len(plan.Tasks))
	}

	completedIndices := []int{0, 2}     // Tasks 1 and 3 are completed
	skippableIndices := []int{0, 2}     // Completed tasks are skippable
	executableIndices := []int{1, 3, 4} // Pending tasks are executable

	for i, task := range plan.Tasks {
		isCompleted := contains(completedIndices, i)
		if task.IsCompleted() != isCompleted {
			t.Errorf("Task %d completion status mismatch", i+1)
		}

		isSkippable := contains(skippableIndices, i)
		if task.CanSkip() != isSkippable {
			t.Errorf("Task %d skipability mismatch", i+1)
		}

		isExecutable := contains(executableIndices, i)
		if task.CanSkip() == isExecutable {
			t.Errorf("Task %d executability mismatch", i+1)
		}
	}
}

// TestSkipMarkdownParallelCompleted tests skip behavior with parallel tasks
func TestSkipMarkdownParallelCompleted(t *testing.T) {
	planPath := filepath.Join("fixtures", "skip-markdown-parallel-completed.md")

	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("Failed to parse plan: %v", err)
	}

	if len(plan.Tasks) != 5 {
		t.Errorf("Expected 5 tasks, got %d", len(plan.Tasks))
	}

	waves, err := executor.CalculateWaves(plan.Tasks)
	if err != nil {
		t.Fatalf("Failed to calculate waves: %v", err)
	}

	// Expected structure: 1 task in wave 1, 3 parallel tasks in wave 2, 1 in wave 3
	if len(waves) != 3 {
		t.Errorf("Expected 3 waves, got %d", len(waves))
	}

	// Wave 1: Task 1 (completed)
	if len(waves[0].TaskNumbers) != 1 {
		t.Errorf("Wave 1: expected 1 task, got %d", len(waves[0].TaskNumbers))
	}

	// Wave 2: Tasks 2, 3, 4 (mix of pending and completed)
	if len(waves[1].TaskNumbers) != 3 {
		t.Errorf("Wave 2: expected 3 tasks, got %d", len(waves[1].TaskNumbers))
	}

	// Verify statuses in parallel wave
	taskMap := make(map[string]models.Task)
	for _, task := range plan.Tasks {
		taskMap[task.Number] = task
	}

	// Task 2 should be pending, Task 3 completed, Task 4 pending
	if taskMap["2"].Status == "completed" {
		t.Error("Task 2 should not be completed")
	}
	if taskMap["3"].Status != "completed" {
		t.Error("Task 3 should be completed")
	}
	if taskMap["4"].Status == "completed" {
		t.Error("Task 4 should not be completed")
	}
}

// TestWaveExecutorSkipCompleted tests wave execution with skip flag enabled
func TestWaveExecutorSkipCompleted(t *testing.T) {
	planPath := filepath.Join("fixtures", "skip-yaml-mixed-status.yaml")
	plan, _ := parser.ParseFile(planPath)

	// Calculate waves from tasks
	waves, err := executor.CalculateWaves(plan.Tasks)
	if err != nil {
		t.Fatalf("Failed to calculate waves: %v", err)
	}
	plan.Waves = waves

	// Create a mock task executor that returns GREEN for all tasks
	mockExecutor := &mockTaskExecutor{
		executeFunc: func(ctx context.Context, task models.Task) (models.TaskResult, error) {
			return models.TaskResult{
				Task:   task,
				Status: models.StatusGreen,
				Output: "Mock execution completed",
			}, nil
		},
	}

	// Execute with skip enabled
	waveExec := executor.NewWaveExecutorWithConfig(mockExecutor, nil, true, false)
	results, err := waveExec.ExecutePlan(context.Background(), plan)

	if err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}

	// Verify that completed tasks have GREEN status and Skipped output
	completedTaskNums := map[string]bool{"1": true, "3": true}
	for _, result := range results {
		if completedTaskNums[result.Task.Number] {
			if result.Status != models.StatusGreen {
				t.Errorf("Task %s should have GREEN status", result.Task.Number)
			}
			if result.Output != "Skipped" {
				t.Errorf("Task %s should have 'Skipped' output, got %q", result.Task.Number, result.Output)
			}
		}
	}
}

// TestWaveExecutorNoSkip tests that tasks aren't skipped when skip flag is disabled
func TestWaveExecutorNoSkip(t *testing.T) {
	planPath := filepath.Join("fixtures", "skip-yaml-mixed-status.yaml")
	plan, _ := parser.ParseFile(planPath)

	// Calculate waves from tasks
	waves, err := executor.CalculateWaves(plan.Tasks)
	if err != nil {
		t.Fatalf("Failed to calculate waves: %v", err)
	}
	plan.Waves = waves

	executeCount := 0

	// Create a mock task executor that counts executions
	mockExecutor := &mockTaskExecutor{
		executeFunc: func(ctx context.Context, task models.Task) (models.TaskResult, error) {
			executeCount++
			return models.TaskResult{
				Task:   task,
				Status: models.StatusGreen,
				Output: "Executed",
			}, nil
		},
	}

	// Execute with skip disabled (default)
	waveExec := executor.NewWaveExecutorWithConfig(mockExecutor, nil, false, false)
	results, err := waveExec.ExecutePlan(context.Background(), plan)

	if err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	// All 5 tasks should be executed when skip is disabled
	if executeCount != 5 {
		t.Errorf("Expected 5 executions, got %d", executeCount)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}
}

// TestWaveExecutorSkipAllCompleted tests skipping when all tasks are completed
func TestWaveExecutorSkipAllCompleted(t *testing.T) {
	planPath := filepath.Join("fixtures", "skip-yaml-all-completed.yaml")
	plan, _ := parser.ParseFile(planPath)

	// Calculate waves from tasks
	waves, err := executor.CalculateWaves(plan.Tasks)
	if err != nil {
		t.Fatalf("Failed to calculate waves: %v", err)
	}
	plan.Waves = waves

	executeCount := 0

	// Create a mock task executor that shouldn't be called
	mockExecutor := &mockTaskExecutor{
		executeFunc: func(ctx context.Context, task models.Task) (models.TaskResult, error) {
			executeCount++
			return models.TaskResult{
				Task:   task,
				Status: models.StatusGreen,
				Output: "Executed",
			}, nil
		},
	}

	// Execute with skip enabled
	waveExec := executor.NewWaveExecutorWithConfig(mockExecutor, nil, true, false)
	results, err := waveExec.ExecutePlan(context.Background(), plan)

	if err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	// No tasks should be executed when all are completed and skip is enabled
	if executeCount != 0 {
		t.Errorf("Expected 0 executions (all skipped), got %d", executeCount)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// All results should be GREEN with Skipped output
	for _, result := range results {
		if result.Status != models.StatusGreen {
			t.Errorf("Task %s should have GREEN status", result.Task.Number)
		}
		if result.Output != "Skipped" {
			t.Errorf("Task %s should have 'Skipped' output", result.Task.Number)
		}
	}
}

// TestSkipPreservesWaveDependencies verifies that skipping doesn't break dependency graph
func TestSkipPreservesWaveDependencies(t *testing.T) {
	planPath := filepath.Join("fixtures", "skip-markdown-parallel-completed.md")
	plan, _ := parser.ParseFile(planPath)

	waves, err := executor.CalculateWaves(plan.Tasks)
	if err != nil {
		t.Fatalf("Failed to calculate waves: %v", err)
	}

	// Wave structure should be preserved regardless of skip
	if len(waves) != 3 {
		t.Errorf("Expected 3 waves, got %d", len(waves))
		return
	}

	// Each wave should have expected number of tasks
	expectedCounts := []int{1, 3, 1}
	for i, wave := range waves {
		if len(wave.TaskNumbers) != expectedCounts[i] {
			t.Errorf("Wave %d: expected %d tasks, got %d", i+1, expectedCounts[i], len(wave.TaskNumbers))
		}
	}
}

// TestSkipMixedParallelTasks verifies correct skip for mixed parallel task statuses
func TestSkipMixedParallelTasks(t *testing.T) {
	planPath := filepath.Join("fixtures", "skip-markdown-parallel-completed.md")
	plan, _ := parser.ParseFile(planPath)

	// Count tasks by status
	completedCount := 0
	pendingCount := 0

	for _, task := range plan.Tasks {
		if task.IsCompleted() {
			completedCount++
		} else {
			pendingCount++
		}
	}

	// Should have mixed statuses
	if completedCount == 0 {
		t.Error("Expected at least one completed task")
	}
	if pendingCount == 0 {
		t.Error("Expected at least one pending task")
	}

	// Total should match
	if completedCount+pendingCount != len(plan.Tasks) {
		t.Errorf("Status count mismatch: completed %d + pending %d != total %d", completedCount, pendingCount, len(plan.Tasks))
	}
}

// TestSkipStatusPersistence verifies status is correctly read from fixtures
func TestSkipStatusPersistence(t *testing.T) {
	tests := []struct {
		name       string
		fixture    string
		taskNum    string
		shouldSkip bool
	}{
		{"Markdown-Task1-Completed", filepath.Join("fixtures", "skip-markdown-completed.md"), "1", true},
		{"Markdown-Task2-Pending", filepath.Join("fixtures", "skip-markdown-completed.md"), "2", false},
		{"YAML-Task1-Completed", filepath.Join("fixtures", "skip-yaml-all-completed.yaml"), "1", true},
		{"YAML-Task3-Completed", filepath.Join("fixtures", "skip-yaml-all-completed.yaml"), "3", true},
		{"Mixed-Task1-Completed", filepath.Join("fixtures", "skip-yaml-mixed-status.yaml"), "1", true},
		{"Mixed-Task2-Pending", filepath.Join("fixtures", "skip-yaml-mixed-status.yaml"), "2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := parser.ParseFile(tt.fixture)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			var task *models.Task
			for i := range plan.Tasks {
				if plan.Tasks[i].Number == tt.taskNum {
					task = &plan.Tasks[i]
					break
				}
			}

			if task == nil {
				t.Fatalf("Task %s not found", tt.taskNum)
			}

			if task.CanSkip() != tt.shouldSkip {
				t.Errorf("Task %s skipability: expected %v, got %v", tt.taskNum, tt.shouldSkip, task.CanSkip())
			}
		})
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// mockTaskExecutor implements the TaskExecutor interface for testing
type mockTaskExecutor struct {
	executeFunc func(context.Context, models.Task) (models.TaskResult, error)
}

func (m *mockTaskExecutor) Execute(ctx context.Context, task models.Task) (models.TaskResult, error) {
	return m.executeFunc(ctx, task)
}

// contains checks if an index exists in a slice
func contains(slice []int, item int) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// stringToIndex converts a string task number to index for comparison
func stringToIndex(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			return int(s[0]-'0') - 1
		}
	}
	return -1
}

// makeTaskMap creates a map of task number to task for wave execution
func makeTaskMap(tasks []models.Task) map[string]models.Task {
	taskMap := make(map[string]models.Task)
	for _, task := range tasks {
		taskMap[task.Number] = task
	}
	return taskMap
}

// ctx is a helper for context in tests
var ctx = context.Background()
