package executor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
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
