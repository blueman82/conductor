package executor

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
	"github.com/harrison/conductor/internal/logger"
	"github.com/harrison/conductor/internal/models"
)

// TestLOCTrackingIntegration tests the end-to-end LOC tracking workflow.
// This integration test creates an isolated git repo and verifies:
// 1. PreTask captures baseline commit
// 2. PostTask calculates LOC diff after changes
// 3. Task fields are correctly updated
// 4. Result aggregation works correctly
// 5. Database persistence functions if store available
func TestLOCTrackingIntegration(t *testing.T) {
	t.Run("end-to-end LOC tracking workflow", func(t *testing.T) {
		// 1. Create temp directory with git init
		dir := t.TempDir()
		initGitRepo(t, dir)

		// 2. Create initial file and commit
		initialFile := filepath.Join(dir, "main.go")
		initialContent := "package main\n\nfunc main() {\n}\n"
		writeFile(t, initialFile, initialContent)
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Initial commit")

		// 3. Configure LOCTrackerHook with temp dir
		logger := &locMockLogger{}
		hook := NewLOCTrackerHook(true, dir, logger)
		if hook == nil {
			t.Fatal("expected non-nil hook when enabled")
		}

		task := &models.Task{Number: "1", Name: "Integration test task", Prompt: "Test task"}

		// 4. Call PreTask to capture baseline
		if err := hook.PreTask(context.Background(), task); err != nil {
			t.Fatalf("PreTask failed: %v", err)
		}

		// Verify baseline was captured
		baseline, ok := task.Metadata["loc_baseline_commit"].(string)
		if !ok || len(baseline) != 40 {
			t.Fatalf("expected 40-char baseline commit, got: %v", task.Metadata["loc_baseline_commit"])
		}

		// 5. Modify files (add/delete lines)
		// Add new file with 10 lines
		newFile := filepath.Join(dir, "feature.go")
		newContent := "package main\n\n// Feature implements a feature\nfunc Feature() string {\n\treturn \"feature\"\n}\n\nfunc helper() {\n\treturn\n}\n"
		writeFile(t, newFile, newContent)

		// Modify existing file: delete 2 lines, add 3 lines
		modifiedContent := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"
		writeFile(t, initialFile, modifiedContent)

		// 6. Commit changes
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Add feature and update main")

		// 7. Call PostTask to calculate diff
		metrics, err := hook.PostTask(context.Background(), task)
		if err != nil {
			t.Fatalf("PostTask failed: %v", err)
		}

		if metrics == nil {
			t.Fatal("expected non-nil metrics")
		}

		// 8. Verify task.LinesAdded matches actual additions
		// We expect: 10 lines in new file + 5 lines added in modified file (new lines)
		// The exact count depends on git diff behavior
		if task.LinesAdded != metrics.LinesAdded {
			t.Errorf("task.LinesAdded (%d) != metrics.LinesAdded (%d)", task.LinesAdded, metrics.LinesAdded)
		}
		if task.LinesAdded <= 0 {
			t.Errorf("expected positive lines added, got %d", task.LinesAdded)
		}

		// 9. Verify task.LinesDeleted matches actual deletions
		if task.LinesDeleted != metrics.LinesDeleted {
			t.Errorf("task.LinesDeleted (%d) != metrics.LinesDeleted (%d)", task.LinesDeleted, metrics.LinesDeleted)
		}

		// 10. Verify file count
		if metrics.FileCount != 2 {
			t.Errorf("expected 2 files modified, got %d", metrics.FileCount)
		}

		// Verify helper methods work correctly
		expectedNet := metrics.LinesAdded - metrics.LinesDeleted
		if task.NetLOC() != expectedNet {
			t.Errorf("NetLOC() = %d, expected %d", task.NetLOC(), expectedNet)
		}

		expectedTotal := metrics.LinesAdded + metrics.LinesDeleted
		if task.TotalLOC() != expectedTotal {
			t.Errorf("TotalLOC() = %d, expected %d", task.TotalLOC(), expectedTotal)
		}

		// Verify logger was called
		if len(logger.infos) == 0 {
			t.Error("expected info log for LOC tracking result")
		}
	})

	t.Run("result aggregation across multiple tasks", func(t *testing.T) {
		// Create git repo
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Initial commit
		writeFile(t, filepath.Join(dir, "README.md"), "# Test\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Initial")

		hook := NewLOCTrackerHook(true, dir, nil)

		// Task 1: Add 5 lines
		task1 := &models.Task{Number: "1", Name: "Task 1", Prompt: "test"}
		hook.PreTask(context.Background(), task1)
		writeFile(t, filepath.Join(dir, "file1.go"), "package main\n\nfunc One() {\n\treturn\n}\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Task 1")
		metrics1, _ := hook.PostTask(context.Background(), task1)

		// Task 2: Add 3 lines, delete 1 line
		task2 := &models.Task{Number: "2", Name: "Task 2", Prompt: "test"}
		hook.PreTask(context.Background(), task2)
		writeFile(t, filepath.Join(dir, "file2.go"), "package main\n\nfunc Two() {}\n")
		writeFile(t, filepath.Join(dir, "README.md"), "# Updated\n\nNew content.\n") // Modify existing
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Task 2")
		metrics2, _ := hook.PostTask(context.Background(), task2)

		// Verify aggregation: sum of both tasks
		totalAdded := task1.LinesAdded + task2.LinesAdded
		totalDeleted := task1.LinesDeleted + task2.LinesDeleted

		// The individual metrics should be non-zero
		if metrics1 == nil || metrics2 == nil {
			t.Fatal("expected non-nil metrics for both tasks")
		}
		if metrics1.LinesAdded != task1.LinesAdded {
			t.Errorf("task1 lines added mismatch: metrics=%d, task=%d", metrics1.LinesAdded, task1.LinesAdded)
		}
		if metrics2.LinesAdded != task2.LinesAdded {
			t.Errorf("task2 lines added mismatch: metrics=%d, task=%d", metrics2.LinesAdded, task2.LinesAdded)
		}

		// Aggregate should be meaningful
		if totalAdded <= 0 {
			t.Errorf("expected positive total lines added, got %d", totalAdded)
		}

		t.Logf("Task aggregation: Task1(+%d/-%d) + Task2(+%d/-%d) = Total(+%d/-%d)",
			task1.LinesAdded, task1.LinesDeleted,
			task2.LinesAdded, task2.LinesDeleted,
			totalAdded, totalDeleted)
	})

	t.Run("database persistence when store available", func(t *testing.T) {
		// Create git repo
		dir := t.TempDir()
		initGitRepo(t, dir)
		writeFile(t, filepath.Join(dir, "init.go"), "package main\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Initial")

		// Create in-memory learning store
		store, err := learning.NewStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer store.Close()

		// Run task with LOC tracking
		hook := NewLOCTrackerHook(true, dir, nil)
		task := &models.Task{Number: "1", Name: "DB Test Task", Prompt: "test"}

		hook.PreTask(context.Background(), task)
		writeFile(t, filepath.Join(dir, "new.go"), "package main\n\nfunc New() {}\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Add new file")
		hook.PostTask(context.Background(), task)

		// Persist to database
		exec := &learning.TaskExecution{
			PlanFile:     "test-plan.yaml",
			RunNumber:    1,
			TaskNumber:   task.Number,
			TaskName:     task.Name,
			Agent:        "test-agent",
			Prompt:       task.Prompt,
			Success:      true,
			LinesAdded:   task.LinesAdded,
			LinesDeleted: task.LinesDeleted,
		}

		if err := store.RecordExecution(context.Background(), exec); err != nil {
			t.Fatalf("failed to record execution: %v", err)
		}

		// Verify persistence
		if exec.ID == 0 {
			t.Error("expected non-zero execution ID after insert")
		}
		if task.LinesAdded <= 0 {
			t.Errorf("expected positive lines added for verification, got %d", task.LinesAdded)
		}

		t.Logf("Persisted LOC metrics: +%d/-%d (exec ID: %d)", task.LinesAdded, task.LinesDeleted, exec.ID)
	})
}

// TestLOCTrackingDisabled tests that disabled config prevents LOC capture.
func TestLOCTrackingDisabled(t *testing.T) {
	t.Run("disabled config returns nil hook", func(t *testing.T) {
		// 12. Set LOCTracking: false
		hook := NewLOCTrackerHook(false, "/any/path", nil)

		// 13. Verify hook returns nil and no LOC captured
		if hook != nil {
			t.Error("expected nil hook when disabled")
		}
	})

	t.Run("nil hook gracefully handles PreTask", func(t *testing.T) {
		var hook *LOCTrackerHook = nil
		task := &models.Task{Number: "1", Name: "Test", Prompt: "test"}

		err := hook.PreTask(context.Background(), task)
		if err != nil {
			t.Errorf("expected nil error from nil hook, got: %v", err)
		}
	})

	t.Run("nil hook gracefully handles PostTask", func(t *testing.T) {
		var hook *LOCTrackerHook = nil
		task := &models.Task{Number: "1", Name: "Test", Prompt: "test"}

		metrics, err := hook.PostTask(context.Background(), task)
		if err != nil {
			t.Errorf("expected nil error from nil hook, got: %v", err)
		}
		if metrics != nil {
			t.Error("expected nil metrics from nil hook")
		}
	})

	t.Run("disabled hook does not capture LOC", func(t *testing.T) {
		// Create real git repo to ensure the failure is due to disabled, not git issues
		dir := t.TempDir()
		initGitRepo(t, dir)
		writeFile(t, filepath.Join(dir, "test.go"), "package main\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Initial")

		hook := &LOCTrackerHook{
			Enabled: false,
			WorkDir: dir,
		}

		task := &models.Task{Number: "1", Name: "Test", Prompt: "test"}

		// PreTask should be no-op
		if err := hook.PreTask(context.Background(), task); err != nil {
			t.Fatalf("PreTask failed: %v", err)
		}

		// Metadata should NOT be set
		if task.Metadata != nil && task.Metadata["loc_baseline_commit"] != nil {
			t.Error("disabled hook should not capture baseline commit")
		}

		// Make changes
		writeFile(t, filepath.Join(dir, "new.go"), "package main\n\nfunc New() {}\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Add file")

		// PostTask should return nil
		metrics, err := hook.PostTask(context.Background(), task)
		if err != nil {
			t.Fatalf("PostTask failed: %v", err)
		}
		if metrics != nil {
			t.Error("disabled hook should return nil metrics")
		}

		// Task should have zero LOC
		if task.LinesAdded != 0 || task.LinesDeleted != 0 {
			t.Errorf("disabled hook should not update task LOC, got +%d/-%d", task.LinesAdded, task.LinesDeleted)
		}
	})

	t.Run("config toggle integrates with hook creation", func(t *testing.T) {
		// Simulate config with LOCTracking disabled
		cfg := config.DefaultConfig()
		cfg.Metrics.LOCTracking = false

		// This is how run.go creates the hook
		var hook *LOCTrackerHook
		if cfg.Metrics.LOCTracking {
			hook = NewLOCTrackerHook(true, "", nil)
		}

		if hook != nil {
			t.Error("hook should be nil when config.Metrics.LOCTracking is false")
		}

		// With enabled config
		cfg.Metrics.LOCTracking = true
		if cfg.Metrics.LOCTracking {
			hook = NewLOCTrackerHook(true, "", nil)
		}

		if hook == nil {
			t.Error("hook should be non-nil when config.Metrics.LOCTracking is true")
		}
	})
}

// TestLOCTrackingEdgeCases tests edge cases in LOC tracking.
func TestLOCTrackingEdgeCases(t *testing.T) {
	t.Run("no changes between PreTask and PostTask", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)
		writeFile(t, filepath.Join(dir, "test.go"), "package main\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Initial")

		hook := NewLOCTrackerHook(true, dir, nil)
		task := &models.Task{Number: "1", Name: "Test", Prompt: "test"}

		hook.PreTask(context.Background(), task)
		// No changes made
		metrics, err := hook.PostTask(context.Background(), task)

		if err != nil {
			t.Fatalf("PostTask failed: %v", err)
		}
		if metrics == nil {
			t.Fatal("expected non-nil metrics even with no changes")
		}
		if metrics.LinesAdded != 0 || metrics.LinesDeleted != 0 {
			t.Errorf("expected zero changes, got +%d/-%d", metrics.LinesAdded, metrics.LinesDeleted)
		}
		if metrics.FileCount != 0 {
			t.Errorf("expected 0 files, got %d", metrics.FileCount)
		}
	})

	t.Run("binary file handling", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)
		writeFile(t, filepath.Join(dir, "test.go"), "package main\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Initial")

		hook := NewLOCTrackerHook(true, dir, nil)
		task := &models.Task{Number: "1", Name: "Test", Prompt: "test"}

		hook.PreTask(context.Background(), task)

		// Add binary file
		binaryFile := filepath.Join(dir, "image.png")
		if err := os.WriteFile(binaryFile, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, 0644); err != nil {
			t.Fatalf("failed to write binary file: %v", err)
		}
		// Also add a text file for comparison
		writeFile(t, filepath.Join(dir, "new.go"), "package main\n\nfunc New() {}\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Add files")

		metrics, _ := hook.PostTask(context.Background(), task)

		if metrics == nil {
			t.Fatal("expected metrics")
		}
		// Binary files show as "-" in numstat and should be counted as files but not as lines
		if metrics.FileCount != 2 {
			t.Errorf("expected 2 files (binary + text), got %d", metrics.FileCount)
		}
		// Only the text file contributes to line count (3 lines)
		if metrics.LinesAdded < 3 {
			t.Errorf("expected at least 3 lines added from text file, got %d", metrics.LinesAdded)
		}
	})

	t.Run("multiple commits between PreTask and PostTask", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)
		writeFile(t, filepath.Join(dir, "base.go"), "package main\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Initial")

		hook := NewLOCTrackerHook(true, dir, nil)
		task := &models.Task{Number: "1", Name: "Test", Prompt: "test"}

		hook.PreTask(context.Background(), task)

		// First commit
		writeFile(t, filepath.Join(dir, "file1.go"), "package main\n\nfunc One() {}\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Commit 1")

		// Second commit
		writeFile(t, filepath.Join(dir, "file2.go"), "package main\n\nfunc Two() {}\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Commit 2")

		// Third commit (modify existing)
		writeFile(t, filepath.Join(dir, "file1.go"), "package main\n\nfunc One() {\n\treturn\n}\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Commit 3")

		metrics, _ := hook.PostTask(context.Background(), task)

		if metrics == nil {
			t.Fatal("expected metrics")
		}
		// Should capture cumulative changes across all commits
		if metrics.FileCount != 2 {
			t.Errorf("expected 2 files modified, got %d", metrics.FileCount)
		}
		if metrics.LinesAdded <= 0 {
			t.Error("expected positive lines added across multiple commits")
		}
	})

	t.Run("graceful degradation in non-git directory", func(t *testing.T) {
		dir := t.TempDir() // Not a git repo
		logger := &locMockLogger{}
		hook := NewLOCTrackerHook(true, dir, logger)
		task := &models.Task{Number: "1", Name: "Test", Prompt: "test"}

		// Should not fail, just log warning
		err := hook.PreTask(context.Background(), task)
		if err != nil {
			t.Errorf("expected graceful degradation, got error: %v", err)
		}

		if len(logger.warnings) == 0 {
			t.Error("expected warning to be logged for non-git directory")
		}

		// PostTask should also gracefully handle missing baseline
		metrics, err := hook.PostTask(context.Background(), task)
		if err != nil {
			t.Errorf("expected graceful degradation, got error: %v", err)
		}
		if metrics != nil {
			t.Error("expected nil metrics when baseline not captured")
		}
	})
}

// TestLOCTrackingWaveCompleteLOCDisplay verifies that wave completion log includes LOC metrics.
// This satisfies success criteria 3: "Verify wave completion log includes LOC display"
func TestLOCTrackingWaveCompleteLOCDisplay(t *testing.T) {
	t.Run("wave completion shows LOC from task results", func(t *testing.T) {
		// Create git repo with real LOC tracking
		dir := t.TempDir()
		initGitRepo(t, dir)
		writeFile(t, filepath.Join(dir, "init.go"), "package main\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Initial")

		// Create LOC hook
		hook := NewLOCTrackerHook(true, dir, nil)

		// Task with LOC tracking
		task := &models.Task{Number: "1", Name: "Feature task", Prompt: "test"}
		hook.PreTask(context.Background(), task)

		// Make changes
		writeFile(t, filepath.Join(dir, "feature.go"), "package main\n\nfunc Feature() {}\n\nfunc Helper() {}\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Add feature")
		hook.PostTask(context.Background(), task)

		// Verify task has LOC data
		if task.LinesAdded <= 0 {
			t.Fatalf("expected positive LinesAdded, got %d", task.LinesAdded)
		}

		// Create task result with LOC data (simulates what wave executor produces)
		taskResult := models.TaskResult{
			Task:     *task,
			Status:   models.StatusGreen,
			Duration: 10 * time.Second,
		}

		// Create console logger with buffer to capture output
		buf := &bytes.Buffer{}
		consoleLogger := logger.NewConsoleLogger(buf, "info")

		// Create a wave
		wave := models.Wave{
			Name:        "Wave 1",
			TaskNumbers: []string{"1"},
		}

		// Call LogWaveComplete and verify LOC is included in output
		consoleLogger.LogWaveComplete(wave, 10*time.Second, []models.TaskResult{taskResult})

		output := buf.String()

		// Verify the wave completion log contains LOC metrics
		// Format from console.go: " | +%d/-%d LOC"
		if !strings.Contains(output, "LOC") {
			t.Errorf("expected wave completion output to contain LOC metrics, got:\n%s", output)
		}
		// Verify the actual numbers appear
		if !strings.Contains(output, "+") {
			t.Errorf("expected wave completion output to contain '+' for lines added, got:\n%s", output)
		}

		t.Logf("Wave completion output: %s", output)
	})

	t.Run("wave completion omits LOC when zero", func(t *testing.T) {
		// Create task with zero LOC
		task := models.Task{Number: "1", Name: "No changes task", LinesAdded: 0, LinesDeleted: 0}
		taskResult := models.TaskResult{
			Task:     task,
			Status:   models.StatusGreen,
			Duration: 5 * time.Second,
		}

		buf := &bytes.Buffer{}
		consoleLogger := logger.NewConsoleLogger(buf, "info")

		wave := models.Wave{Name: "Wave 1", TaskNumbers: []string{"1"}}
		consoleLogger.LogWaveComplete(wave, 5*time.Second, []models.TaskResult{taskResult})

		output := buf.String()

		// Verify LOC is NOT shown when zero
		if strings.Contains(output, "LOC") {
			t.Errorf("expected wave completion output to NOT contain LOC when zero, got:\n%s", output)
		}
	})

	t.Run("wave completion aggregates LOC from multiple tasks", func(t *testing.T) {
		// Create multiple tasks with LOC
		task1 := models.Task{Number: "1", Name: "Task 1", LinesAdded: 100, LinesDeleted: 25}
		task2 := models.Task{Number: "2", Name: "Task 2", LinesAdded: 50, LinesDeleted: 10}

		results := []models.TaskResult{
			{Task: task1, Status: models.StatusGreen, Duration: 10 * time.Second},
			{Task: task2, Status: models.StatusGreen, Duration: 15 * time.Second},
		}

		buf := &bytes.Buffer{}
		consoleLogger := logger.NewConsoleLogger(buf, "info")

		wave := models.Wave{Name: "Wave 1", TaskNumbers: []string{"1", "2"}}
		consoleLogger.LogWaveComplete(wave, 25*time.Second, results)

		output := buf.String()

		// Verify aggregated LOC is shown (100+50=150 added, 25+10=35 deleted)
		if !strings.Contains(output, "+150") {
			t.Errorf("expected aggregated lines added (+150), got:\n%s", output)
		}
		if !strings.Contains(output, "-35") {
			t.Errorf("expected aggregated lines deleted (-35), got:\n%s", output)
		}
	})
}

// TestLOCTrackingSummaryLOCDisplay verifies that execution summary includes LOC totals.
// This satisfies success criteria 4: "Verify execution summary includes LOC totals"
func TestLOCTrackingSummaryLOCDisplay(t *testing.T) {
	t.Run("execution summary shows LOC totals from ExecutionResult", func(t *testing.T) {
		// Create git repo with real LOC tracking
		dir := t.TempDir()
		initGitRepo(t, dir)
		writeFile(t, filepath.Join(dir, "init.go"), "package main\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Initial")

		// Create LOC hook
		hook := NewLOCTrackerHook(true, dir, nil)

		// Task 1 with LOC tracking
		task1 := &models.Task{Number: "1", Name: "Feature 1", Prompt: "test"}
		hook.PreTask(context.Background(), task1)
		writeFile(t, filepath.Join(dir, "feature1.go"), "package main\n\nfunc F1() {}\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Add feature 1")
		hook.PostTask(context.Background(), task1)

		// Task 2 with LOC tracking
		task2 := &models.Task{Number: "2", Name: "Feature 2", Prompt: "test"}
		hook.PreTask(context.Background(), task2)
		writeFile(t, filepath.Join(dir, "feature2.go"), "package main\n\nfunc F2() {}\n\nfunc F2Helper() {}\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Add feature 2")
		hook.PostTask(context.Background(), task2)

		// Create task results
		results := []models.TaskResult{
			{Task: *task1, Status: models.StatusGreen, Duration: 10 * time.Second},
			{Task: *task2, Status: models.StatusGreen, Duration: 15 * time.Second},
		}

		// Create ExecutionResult using models.NewExecutionResult (which aggregates LOC)
		execResult := models.NewExecutionResult(results, true, 25*time.Second)

		// Verify ExecutionResult has aggregated LOC
		expectedAdded := task1.LinesAdded + task2.LinesAdded
		expectedDeleted := task1.LinesDeleted + task2.LinesDeleted
		if execResult.TotalLinesAdded != expectedAdded {
			t.Errorf("ExecutionResult.TotalLinesAdded = %d, expected %d (task1=%d + task2=%d)",
				execResult.TotalLinesAdded, expectedAdded, task1.LinesAdded, task2.LinesAdded)
		}
		if execResult.TotalLinesDeleted != expectedDeleted {
			t.Errorf("ExecutionResult.TotalLinesDeleted = %d, expected %d",
				execResult.TotalLinesDeleted, expectedDeleted)
		}

		// Create console logger and verify output
		buf := &bytes.Buffer{}
		consoleLogger := logger.NewConsoleLogger(buf, "info")
		consoleLogger.LogSummary(*execResult)

		output := buf.String()

		// Verify summary contains "Lines of Code:" section
		if !strings.Contains(output, "Lines of Code:") {
			t.Errorf("expected summary to contain 'Lines of Code:' section, got:\n%s", output)
		}

		// Verify Added/Deleted/Net are shown
		if !strings.Contains(output, "Added:") {
			t.Errorf("expected summary to contain 'Added:' line, got:\n%s", output)
		}
		if !strings.Contains(output, "Deleted:") {
			t.Errorf("expected summary to contain 'Deleted:' line, got:\n%s", output)
		}
		if !strings.Contains(output, "Net:") {
			t.Errorf("expected summary to contain 'Net:' line, got:\n%s", output)
		}

		t.Logf("Execution summary LOC output:\n%s", output)
		t.Logf("Aggregated LOC: +%d/-%d (Net: %+d)",
			execResult.TotalLinesAdded, execResult.TotalLinesDeleted,
			execResult.TotalLinesAdded-execResult.TotalLinesDeleted)
	})

	t.Run("execution summary omits LOC section when zero", func(t *testing.T) {
		// Create ExecutionResult with zero LOC
		results := []models.TaskResult{
			{Task: models.Task{Number: "1", LinesAdded: 0, LinesDeleted: 0}, Status: models.StatusGreen},
		}
		execResult := models.NewExecutionResult(results, true, 10*time.Second)

		buf := &bytes.Buffer{}
		consoleLogger := logger.NewConsoleLogger(buf, "info")
		consoleLogger.LogSummary(*execResult)

		output := buf.String()

		// Verify LOC section is NOT shown when zero
		if strings.Contains(output, "Lines of Code:") {
			t.Errorf("expected summary to NOT contain 'Lines of Code:' when zero, got:\n%s", output)
		}
	})

	t.Run("execution summary shows correct net (added - deleted)", func(t *testing.T) {
		// Create ExecutionResult with specific LOC values
		results := []models.TaskResult{
			{Task: models.Task{Number: "1", LinesAdded: 200, LinesDeleted: 50}, Status: models.StatusGreen},
		}
		execResult := models.NewExecutionResult(results, true, 10*time.Second)

		buf := &bytes.Buffer{}
		consoleLogger := logger.NewConsoleLogger(buf, "info")
		consoleLogger.LogSummary(*execResult)

		output := buf.String()

		// Net should be 200 - 50 = +150
		if !strings.Contains(output, "Net: +150") {
			t.Errorf("expected 'Net: +150' in summary, got:\n%s", output)
		}
	})

	t.Run("execution summary shows negative net when more deleted", func(t *testing.T) {
		// Create ExecutionResult with more deletions than additions
		results := []models.TaskResult{
			{Task: models.Task{Number: "1", LinesAdded: 30, LinesDeleted: 100}, Status: models.StatusGreen},
		}
		execResult := models.NewExecutionResult(results, true, 10*time.Second)

		buf := &bytes.Buffer{}
		consoleLogger := logger.NewConsoleLogger(buf, "info")
		consoleLogger.LogSummary(*execResult)

		output := buf.String()

		// Net should be 30 - 100 = -70
		if !strings.Contains(output, "Net: -70") {
			t.Errorf("expected 'Net: -70' in summary, got:\n%s", output)
		}
	})
}

// TestLOCTrackingEndToEndWithLogger verifies the full flow from hook to logger output.
// This tests the complete integration: hook -> task -> result -> ExecutionResult -> logger
func TestLOCTrackingEndToEndWithLogger(t *testing.T) {
	t.Run("complete flow: hook -> task -> result -> logger display", func(t *testing.T) {
		// Setup isolated git repo
		dir := t.TempDir()
		initGitRepo(t, dir)
		writeFile(t, filepath.Join(dir, "init.go"), "package main\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Initial")

		// Create LOC hook
		hook := NewLOCTrackerHook(true, dir, nil)

		// Run task through LOC tracking
		task := &models.Task{Number: "1", Name: "Full flow test", Prompt: "test"}
		hook.PreTask(context.Background(), task)
		writeFile(t, filepath.Join(dir, "feature.go"), "package main\n\nfunc Feature() {\n\treturn\n}\n")
		gitAdd(t, dir, ".")
		gitCommit(t, dir, "Add feature")
		hook.PostTask(context.Background(), task)

		// Create result (simulates what wave executor produces)
		taskResult := models.TaskResult{
			Task:     *task,
			Status:   models.StatusGreen,
			Duration: 10 * time.Second,
		}

		// Create ExecutionResult (simulates what orchestrator produces)
		execResult := models.NewExecutionResult([]models.TaskResult{taskResult}, true, 10*time.Second)

		// Verify the chain: task LOC -> ExecutionResult LOC
		if task.LinesAdded <= 0 {
			t.Fatal("task should have positive LinesAdded from hook")
		}
		if execResult.TotalLinesAdded != task.LinesAdded {
			t.Errorf("ExecutionResult.TotalLinesAdded (%d) should match task.LinesAdded (%d)",
				execResult.TotalLinesAdded, task.LinesAdded)
		}

		// Verify wave completion log shows LOC
		waveBuf := &bytes.Buffer{}
		waveLogger := logger.NewConsoleLogger(waveBuf, "info")
		wave := models.Wave{Name: "Wave 1", TaskNumbers: []string{"1"}}
		waveLogger.LogWaveComplete(wave, 10*time.Second, []models.TaskResult{taskResult})

		waveOutput := waveBuf.String()
		if !strings.Contains(waveOutput, "LOC") {
			t.Errorf("wave completion should include LOC, got:\n%s", waveOutput)
		}

		// Verify execution summary shows LOC
		summaryBuf := &bytes.Buffer{}
		summaryLogger := logger.NewConsoleLogger(summaryBuf, "info")
		summaryLogger.LogSummary(*execResult)

		summaryOutput := summaryBuf.String()
		if !strings.Contains(summaryOutput, "Lines of Code:") {
			t.Errorf("execution summary should include 'Lines of Code:' section, got:\n%s", summaryOutput)
		}

		t.Logf("Task LOC: +%d/-%d", task.LinesAdded, task.LinesDeleted)
		t.Logf("ExecutionResult LOC: +%d/-%d", execResult.TotalLinesAdded, execResult.TotalLinesDeleted)
		t.Logf("Wave output:\n%s", waveOutput)
		t.Logf("Summary output:\n%s", summaryOutput)
	})
}

// Helper functions for git operations in tests

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test User")
}

func gitAdd(t *testing.T, dir string, paths ...string) {
	t.Helper()
	args := append([]string{"add"}, paths...)
	runGit(t, dir, args...)
}

func gitCommit(t *testing.T, dir, message string) {
	t.Helper()
	runGit(t, dir, "commit", "-m", message)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, output)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}
