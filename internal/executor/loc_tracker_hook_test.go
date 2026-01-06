package executor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/harrison/conductor/internal/models"
)

// locMockLogger implements RuntimeEnforcementLogger for testing.
type locMockLogger struct {
	infos    []string
	warnings []string
}

func (l *locMockLogger) LogTestCommands(entries []models.TestCommandResult)                     {}
func (l *locMockLogger) LogCriterionVerifications(entries []models.CriterionVerificationResult) {}
func (l *locMockLogger) LogDocTargetVerifications(entries []models.DocTargetResult)             {}
func (l *locMockLogger) LogErrorPattern(pattern interface{})                                    {}
func (l *locMockLogger) LogDetectedError(detected interface{})                                  {}

func (l *locMockLogger) Warnf(format string, args ...interface{}) {
	l.warnings = append(l.warnings, format)
}

func (l *locMockLogger) Info(message string) {
	l.infos = append(l.infos, message)
}

func (l *locMockLogger) Infof(format string, args ...interface{}) {
	l.infos = append(l.infos, format)
}

// setupGitRepo creates a temporary directory with an initialized git repo.
func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to configure git email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to configure git name: %v", err)
	}

	// Create initial commit
	initialFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(initialFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add initial file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}

	return dir
}

func TestNewLOCTrackerHook(t *testing.T) {
	t.Run("returns nil when disabled", func(t *testing.T) {
		hook := NewLOCTrackerHook(false, "/some/path", nil)
		if hook != nil {
			t.Error("expected nil when disabled")
		}
	})

	t.Run("returns hook when enabled", func(t *testing.T) {
		logger := &locMockLogger{}
		hook := NewLOCTrackerHook(true, "/some/path", logger)
		if hook == nil {
			t.Fatal("expected non-nil hook when enabled")
		}
		if !hook.Enabled {
			t.Error("expected Enabled to be true")
		}
		if hook.WorkDir != "/some/path" {
			t.Errorf("expected WorkDir to be /some/path, got %s", hook.WorkDir)
		}
		if hook.Logger != logger {
			t.Error("expected Logger to match")
		}
	})
}

func TestLOCTrackerHook_PreTask(t *testing.T) {
	t.Run("nil hook gracefully returns nil", func(t *testing.T) {
		var hook *LOCTrackerHook
		task := &models.Task{Number: "1", Name: "Test", Prompt: "test"}
		err := hook.PreTask(context.Background(), task)
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("disabled hook gracefully returns nil", func(t *testing.T) {
		hook := &LOCTrackerHook{Enabled: false}
		task := &models.Task{Number: "1", Name: "Test", Prompt: "test"}
		err := hook.PreTask(context.Background(), task)
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("captures baseline commit in git repo", func(t *testing.T) {
		dir := setupGitRepo(t)
		logger := &locMockLogger{}
		hook := NewLOCTrackerHook(true, dir, logger)

		task := &models.Task{Number: "1", Name: "Test", Prompt: "test"}
		err := hook.PreTask(context.Background(), task)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if task.Metadata == nil {
			t.Fatal("expected Metadata to be initialized")
		}

		baseline, ok := task.Metadata["loc_baseline_commit"].(string)
		if !ok {
			t.Fatal("expected loc_baseline_commit in Metadata")
		}

		// Commit hash should be 40 hex characters
		if len(baseline) != 40 {
			t.Errorf("expected 40-char commit hash, got %d chars: %s", len(baseline), baseline)
		}
	})

	t.Run("initializes nil Metadata", func(t *testing.T) {
		dir := setupGitRepo(t)
		hook := NewLOCTrackerHook(true, dir, nil)

		task := &models.Task{Number: "1", Name: "Test", Prompt: "test", Metadata: nil}
		err := hook.PreTask(context.Background(), task)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if task.Metadata == nil {
			t.Error("expected Metadata to be initialized")
		}
	})

	t.Run("graceful degradation on non-git directory", func(t *testing.T) {
		dir := t.TempDir() // Not a git repo
		logger := &locMockLogger{}
		hook := NewLOCTrackerHook(true, dir, logger)

		task := &models.Task{Number: "1", Name: "Test", Prompt: "test"}
		err := hook.PreTask(context.Background(), task)
		if err != nil {
			t.Errorf("expected nil error for graceful degradation, got: %v", err)
		}

		// Should have logged a warning
		if len(logger.warnings) == 0 {
			t.Error("expected warning to be logged")
		}
	})
}

func TestLOCTrackerHook_PostTask(t *testing.T) {
	t.Run("nil hook gracefully returns nil", func(t *testing.T) {
		var hook *LOCTrackerHook
		task := &models.Task{Number: "1", Name: "Test", Prompt: "test"}
		metrics, err := hook.PostTask(context.Background(), task)
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
		if metrics != nil {
			t.Error("expected nil metrics")
		}
	})

	t.Run("disabled hook gracefully returns nil", func(t *testing.T) {
		hook := &LOCTrackerHook{Enabled: false}
		task := &models.Task{Number: "1", Name: "Test", Prompt: "test"}
		metrics, err := hook.PostTask(context.Background(), task)
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
		if metrics != nil {
			t.Error("expected nil metrics")
		}
	})

	t.Run("nil task gracefully returns nil", func(t *testing.T) {
		hook := NewLOCTrackerHook(true, "/some/path", nil)
		metrics, err := hook.PostTask(context.Background(), nil)
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
		if metrics != nil {
			t.Error("expected nil metrics")
		}
	})

	t.Run("returns nil when no baseline commit", func(t *testing.T) {
		hook := NewLOCTrackerHook(true, "/some/path", nil)
		task := &models.Task{Number: "1", Name: "Test", Prompt: "test"}
		metrics, err := hook.PostTask(context.Background(), task)
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
		if metrics != nil {
			t.Error("expected nil metrics when no baseline")
		}
	})

	t.Run("calculates LOC changes", func(t *testing.T) {
		dir := setupGitRepo(t)
		logger := &locMockLogger{}
		hook := NewLOCTrackerHook(true, dir, logger)

		task := &models.Task{Number: "1", Name: "Test", Prompt: "test"}

		// Capture baseline
		if err := hook.PreTask(context.Background(), task); err != nil {
			t.Fatalf("PreTask failed: %v", err)
		}

		// Make changes and commit
		testFile := filepath.Join(dir, "test.go")
		content := "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		cmd := exec.Command("git", "add", ".")
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add test file: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "Add test file")
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit test file: %v", err)
		}

		// Calculate LOC
		metrics, err := hook.PostTask(context.Background(), task)
		if err != nil {
			t.Fatalf("PostTask failed: %v", err)
		}

		if metrics == nil {
			t.Fatal("expected non-nil metrics")
		}

		// 5 lines added (the content we wrote)
		if metrics.LinesAdded != 5 {
			t.Errorf("expected 5 lines added, got %d", metrics.LinesAdded)
		}
		if metrics.LinesDeleted != 0 {
			t.Errorf("expected 0 lines deleted, got %d", metrics.LinesDeleted)
		}
		if metrics.FileCount != 1 {
			t.Errorf("expected 1 file, got %d", metrics.FileCount)
		}

		// Verify task fields were updated
		if task.LinesAdded != 5 {
			t.Errorf("expected task.LinesAdded = 5, got %d", task.LinesAdded)
		}
		if task.LinesDeleted != 0 {
			t.Errorf("expected task.LinesDeleted = 0, got %d", task.LinesDeleted)
		}

		// Verify logging
		if len(logger.infos) == 0 {
			t.Error("expected info log for LOC tracking")
		}
	})

	t.Run("calculates deletions correctly", func(t *testing.T) {
		dir := setupGitRepo(t)
		hook := NewLOCTrackerHook(true, dir, nil)

		task := &models.Task{Number: "1", Name: "Test", Prompt: "test"}

		// Capture baseline
		if err := hook.PreTask(context.Background(), task); err != nil {
			t.Fatalf("PreTask failed: %v", err)
		}

		// Delete content from README.md (2 lines -> 0 lines = 2 deleted, but we have 1 line)
		readme := filepath.Join(dir, "README.md")
		if err := os.WriteFile(readme, []byte(""), 0644); err != nil {
			t.Fatalf("failed to truncate readme: %v", err)
		}

		cmd := exec.Command("git", "add", ".")
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add changes: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "Clear readme")
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit changes: %v", err)
		}

		// Calculate LOC
		metrics, err := hook.PostTask(context.Background(), task)
		if err != nil {
			t.Fatalf("PostTask failed: %v", err)
		}

		if metrics == nil {
			t.Fatal("expected non-nil metrics")
		}

		// Original README had "# Test\n" = 1 line deleted
		if metrics.LinesDeleted != 1 {
			t.Errorf("expected 1 line deleted, got %d", metrics.LinesDeleted)
		}
		if metrics.LinesAdded != 0 {
			t.Errorf("expected 0 lines added, got %d", metrics.LinesAdded)
		}
	})

	t.Run("handles multiple file changes", func(t *testing.T) {
		dir := setupGitRepo(t)
		hook := NewLOCTrackerHook(true, dir, nil)

		task := &models.Task{Number: "1", Name: "Test", Prompt: "test"}

		// Capture baseline
		if err := hook.PreTask(context.Background(), task); err != nil {
			t.Fatalf("PreTask failed: %v", err)
		}

		// Create multiple files
		file1 := filepath.Join(dir, "file1.go")
		if err := os.WriteFile(file1, []byte("package main\n\n"), 0644); err != nil {
			t.Fatalf("failed to write file1: %v", err)
		}

		file2 := filepath.Join(dir, "file2.go")
		if err := os.WriteFile(file2, []byte("package main\n\nfunc test() {}\n"), 0644); err != nil {
			t.Fatalf("failed to write file2: %v", err)
		}

		cmd := exec.Command("git", "add", ".")
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add files: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "Add multiple files")
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit files: %v", err)
		}

		// Calculate LOC
		metrics, err := hook.PostTask(context.Background(), task)
		if err != nil {
			t.Fatalf("PostTask failed: %v", err)
		}

		if metrics == nil {
			t.Fatal("expected non-nil metrics")
		}

		// file1: 2 lines (package main\n\n)
		// file2: 3 lines (package main\n\nfunc test() {}\n)
		if metrics.LinesAdded != 5 {
			t.Errorf("expected 5 lines added, got %d", metrics.LinesAdded)
		}
		if metrics.FileCount != 2 {
			t.Errorf("expected 2 files, got %d", metrics.FileCount)
		}
	})

	t.Run("no changes returns zero metrics", func(t *testing.T) {
		dir := setupGitRepo(t)
		hook := NewLOCTrackerHook(true, dir, nil)

		task := &models.Task{Number: "1", Name: "Test", Prompt: "test"}

		// Capture baseline
		if err := hook.PreTask(context.Background(), task); err != nil {
			t.Fatalf("PreTask failed: %v", err)
		}

		// No changes made

		// Calculate LOC
		metrics, err := hook.PostTask(context.Background(), task)
		if err != nil {
			t.Fatalf("PostTask failed: %v", err)
		}

		if metrics == nil {
			t.Fatal("expected non-nil metrics")
		}

		if metrics.LinesAdded != 0 {
			t.Errorf("expected 0 lines added, got %d", metrics.LinesAdded)
		}
		if metrics.LinesDeleted != 0 {
			t.Errorf("expected 0 lines deleted, got %d", metrics.LinesDeleted)
		}
		if metrics.FileCount != 0 {
			t.Errorf("expected 0 files, got %d", metrics.FileCount)
		}
	})
}

func TestLOCTrackerHook_parseNumstat(t *testing.T) {
	hook := &LOCTrackerHook{}

	t.Run("empty output", func(t *testing.T) {
		metrics := hook.parseNumstat("")
		if metrics.LinesAdded != 0 || metrics.LinesDeleted != 0 || metrics.FileCount != 0 {
			t.Errorf("expected all zeros for empty output, got: +%d/-%d in %d files",
				metrics.LinesAdded, metrics.LinesDeleted, metrics.FileCount)
		}
	})

	t.Run("single file with additions only", func(t *testing.T) {
		output := "10\t0\tfile.go"
		metrics := hook.parseNumstat(output)
		if metrics.LinesAdded != 10 {
			t.Errorf("expected 10 lines added, got %d", metrics.LinesAdded)
		}
		if metrics.LinesDeleted != 0 {
			t.Errorf("expected 0 lines deleted, got %d", metrics.LinesDeleted)
		}
		if metrics.FileCount != 1 {
			t.Errorf("expected 1 file, got %d", metrics.FileCount)
		}
	})

	t.Run("single file with deletions only", func(t *testing.T) {
		output := "0\t5\tfile.go"
		metrics := hook.parseNumstat(output)
		if metrics.LinesAdded != 0 {
			t.Errorf("expected 0 lines added, got %d", metrics.LinesAdded)
		}
		if metrics.LinesDeleted != 5 {
			t.Errorf("expected 5 lines deleted, got %d", metrics.LinesDeleted)
		}
	})

	t.Run("mixed additions and deletions", func(t *testing.T) {
		output := "15\t8\tfile.go"
		metrics := hook.parseNumstat(output)
		if metrics.LinesAdded != 15 {
			t.Errorf("expected 15 lines added, got %d", metrics.LinesAdded)
		}
		if metrics.LinesDeleted != 8 {
			t.Errorf("expected 8 lines deleted, got %d", metrics.LinesDeleted)
		}
	})

	t.Run("multiple files", func(t *testing.T) {
		output := "10\t5\tfile1.go\n20\t3\tfile2.go\n5\t0\tfile3.go"
		metrics := hook.parseNumstat(output)
		if metrics.LinesAdded != 35 {
			t.Errorf("expected 35 lines added (10+20+5), got %d", metrics.LinesAdded)
		}
		if metrics.LinesDeleted != 8 {
			t.Errorf("expected 8 lines deleted (5+3+0), got %d", metrics.LinesDeleted)
		}
		if metrics.FileCount != 3 {
			t.Errorf("expected 3 files, got %d", metrics.FileCount)
		}
	})

	t.Run("binary file (shown as dashes)", func(t *testing.T) {
		output := "-\t-\timage.png"
		metrics := hook.parseNumstat(output)
		if metrics.LinesAdded != 0 {
			t.Errorf("expected 0 lines added for binary, got %d", metrics.LinesAdded)
		}
		if metrics.LinesDeleted != 0 {
			t.Errorf("expected 0 lines deleted for binary, got %d", metrics.LinesDeleted)
		}
		if metrics.FileCount != 1 {
			t.Errorf("expected 1 file (binary still counts), got %d", metrics.FileCount)
		}
	})

	t.Run("mixed text and binary files", func(t *testing.T) {
		output := "10\t5\tfile.go\n-\t-\timage.png\n20\t0\tother.go"
		metrics := hook.parseNumstat(output)
		if metrics.LinesAdded != 30 {
			t.Errorf("expected 30 lines added (10+20), got %d", metrics.LinesAdded)
		}
		if metrics.LinesDeleted != 5 {
			t.Errorf("expected 5 lines deleted, got %d", metrics.LinesDeleted)
		}
		if metrics.FileCount != 3 {
			t.Errorf("expected 3 files, got %d", metrics.FileCount)
		}
	})

	t.Run("whitespace in output", func(t *testing.T) {
		output := "\n10\t5\tfile.go\n\n"
		metrics := hook.parseNumstat(output)
		if metrics.LinesAdded != 10 {
			t.Errorf("expected 10 lines added, got %d", metrics.LinesAdded)
		}
		if metrics.FileCount != 1 {
			t.Errorf("expected 1 file, got %d", metrics.FileCount)
		}
	})

	t.Run("malformed line (less than 3 parts)", func(t *testing.T) {
		output := "10\tfile.go"
		metrics := hook.parseNumstat(output)
		// Should skip malformed lines
		if metrics.FileCount != 0 {
			t.Errorf("expected 0 files for malformed output, got %d", metrics.FileCount)
		}
	})

	t.Run("file path with spaces", func(t *testing.T) {
		output := "10\t5\tpath with spaces/file.go"
		metrics := hook.parseNumstat(output)
		if metrics.LinesAdded != 10 {
			t.Errorf("expected 10 lines added, got %d", metrics.LinesAdded)
		}
		if metrics.FileCount != 1 {
			t.Errorf("expected 1 file, got %d", metrics.FileCount)
		}
	})
}

func TestLOCTrackerHook_EndToEnd(t *testing.T) {
	t.Run("full workflow: PreTask -> changes -> PostTask", func(t *testing.T) {
		dir := setupGitRepo(t)
		logger := &locMockLogger{}
		hook := NewLOCTrackerHook(true, dir, logger)

		task := &models.Task{Number: "42", Name: "Integration Test", Prompt: "test"}

		// 1. PreTask captures baseline
		if err := hook.PreTask(context.Background(), task); err != nil {
			t.Fatalf("PreTask failed: %v", err)
		}

		baseline, ok := task.Metadata["loc_baseline_commit"].(string)
		if !ok || len(baseline) != 40 {
			t.Fatal("baseline commit not captured correctly")
		}

		// 2. Simulate agent making changes
		testFile := filepath.Join(dir, "feature.go")
		content := "package feature\n\n// Feature does something\nfunc Feature() string {\n\treturn \"feature\"\n}\n"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write feature file: %v", err)
		}

		// Modify existing README
		readme := filepath.Join(dir, "README.md")
		if err := os.WriteFile(readme, []byte("# Test\n\nThis is a test project.\n"), 0644); err != nil {
			t.Fatalf("failed to update readme: %v", err)
		}

		cmd := exec.Command("git", "add", ".")
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add changes: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "Add feature and update readme")
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// 3. PostTask calculates diff
		metrics, err := hook.PostTask(context.Background(), task)
		if err != nil {
			t.Fatalf("PostTask failed: %v", err)
		}

		if metrics == nil {
			t.Fatal("expected metrics to be returned")
		}

		// Verify non-zero metrics (actual counts may vary by git version)
		if metrics.LinesAdded == 0 {
			t.Error("expected non-zero lines added")
		}
		if metrics.FileCount != 2 {
			t.Errorf("expected 2 files, got %d", metrics.FileCount)
		}

		// 4. Verify task fields were updated to match metrics
		if task.LinesAdded != metrics.LinesAdded {
			t.Errorf("task.LinesAdded = %d, expected %d", task.LinesAdded, metrics.LinesAdded)
		}
		if task.LinesDeleted != metrics.LinesDeleted {
			t.Errorf("task.LinesDeleted = %d, expected %d", task.LinesDeleted, metrics.LinesDeleted)
		}

		// 5. Verify helper methods work
		netLOC := task.NetLOC()
		expectedNet := metrics.LinesAdded - metrics.LinesDeleted
		if netLOC != expectedNet {
			t.Errorf("NetLOC() = %d, expected %d", netLOC, expectedNet)
		}

		totalLOC := task.TotalLOC()
		expectedTotal := metrics.LinesAdded + metrics.LinesDeleted
		if totalLOC != expectedTotal {
			t.Errorf("TotalLOC() = %d, expected %d", totalLOC, expectedTotal)
		}

		// Sanity check: we added content, so additions should be positive
		if metrics.LinesAdded < 5 {
			t.Errorf("expected at least 5 lines added (feature.go has 6 lines content), got %d", metrics.LinesAdded)
		}
	})
}
