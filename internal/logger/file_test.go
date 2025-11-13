package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// TestLogDirectoryCreation verifies .conductor/logs/ directory is created on initialization
func TestLogDirectoryCreation(t *testing.T) {
	// Create a temporary working directory for this test
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create FileLogger
	logger, err := NewFileLogger()
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}
	defer logger.Close()

	// Verify .conductor/logs directory exists
	logDir := filepath.Join(tmpDir, ".conductor", "logs")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Errorf("Expected log directory %s to exist, but it doesn't", logDir)
	}
}

// TestPerRunLogFile verifies a timestamped log file is created per run
func TestPerRunLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	logger, err := NewFileLogger()
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}
	defer logger.Close()

	// Verify a timestamped log file exists
	logDir := filepath.Join(tmpDir, ".conductor", "logs")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("Failed to read log directory: %v", err)
	}

	// Should have at least one log file (excluding symlinks initially)
	logFileFound := false
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".log") && entry.Name() != "latest.log" {
			logFileFound = true
			// Verify filename format: run-YYYYMMDD-HHMMSS.log
			if !strings.HasPrefix(entry.Name(), "run-") {
				t.Errorf("Expected log file to start with 'run-', got %s", entry.Name())
			}
		}
	}

	if !logFileFound {
		t.Error("Expected to find a timestamped log file")
	}
}

// TestLatestSymlink verifies latest.log symlink is created and points to current run
func TestLatestSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	logger, err := NewFileLogger()
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}
	defer logger.Close()

	// Verify latest.log symlink exists
	symlinkPath := filepath.Join(tmpDir, ".conductor", "logs", "latest.log")
	linkInfo, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Fatalf("Expected latest.log symlink to exist: %v", err)
	}

	// Verify it's a symlink
	if linkInfo.Mode()&os.ModeSymlink == 0 {
		t.Error("Expected latest.log to be a symlink")
	}

	// Verify symlink points to a valid file
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("Failed to read symlink: %v", err)
	}

	if !strings.HasPrefix(filepath.Base(target), "run-") {
		t.Errorf("Expected symlink to point to run-*.log file, got %s", target)
	}
}

// TestSymlinkUpdate verifies symlink updates on new run
func TestSymlinkUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create first logger
	logger1, err := NewFileLogger()
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}

	symlinkPath := filepath.Join(tmpDir, ".conductor", "logs", "latest.log")
	target1, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("Failed to read symlink: %v", err)
	}

	logger1.Close()

	// Wait a bit to ensure different timestamp
	time.Sleep(time.Second)

	// Create second logger
	logger2, err := NewFileLogger()
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}
	defer logger2.Close()

	// Verify symlink was updated
	target2, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("Failed to read symlink: %v", err)
	}

	if target1 == target2 {
		t.Error("Expected symlink to point to new log file, but it still points to old one")
	}
}

// TestFileLogWaveStart verifies wave start is logged correctly
func TestFileLogWaveStart(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	logger, err := NewFileLogger()
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}
	defer logger.Close()

	wave := models.Wave{
		Name:           "Wave 1",
		TaskNumbers:    []string{"1", "2"},
		MaxConcurrency: 2,
	}

	logger.LogWaveStart(wave)

	// Read log file content
	content := readRunLog(t, tmpDir)

	// Verify wave start is logged
	if !strings.Contains(content, "Wave 1") {
		t.Error("Expected log to contain wave name")
	}
	if !strings.Contains(content, "Starting") {
		t.Error("Expected log to indicate wave is starting")
	}
}

// TestFileLogWaveComplete verifies wave completion is logged with duration
func TestFileLogWaveComplete(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	logger, err := NewFileLogger()
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}
	defer logger.Close()

	wave := models.Wave{
		Name:           "Wave 1",
		TaskNumbers:    []string{"1", "2"},
		MaxConcurrency: 2,
	}

	duration := 5 * time.Second
	logger.LogWaveComplete(wave, duration, []models.TaskResult{})

	// Read log file content
	content := readRunLog(t, tmpDir)

	// Verify wave completion is logged
	if !strings.Contains(content, "Wave 1") {
		t.Error("Expected log to contain wave name")
	}
	if !strings.Contains(content, "complete") {
		t.Error("Expected log to indicate wave completion")
	}
	if !strings.Contains(content, "5.0") {
		t.Error("Expected log to contain duration")
	}
}

// TestFileLogSummary verifies execution summary is logged correctly
func TestFileLogSummary(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	logger, err := NewFileLogger()
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}
	defer logger.Close()

	result := models.ExecutionResult{
		TotalTasks: 10,
		Completed:  8,
		Failed:     2,
		Duration:   2 * time.Minute,
		FailedTasks: []models.TaskResult{
			{
				Task: models.Task{
					Number: "3",
					Name:   "Failed Task",
				},
				Status: models.StatusRed,
			},
		},
	}

	logger.LogSummary(result)

	// Read log file content
	content := readRunLog(t, tmpDir)

	// Verify summary contains key metrics
	if !strings.Contains(content, "10") {
		t.Error("Expected log to contain total tasks count")
	}
	if !strings.Contains(content, "8") {
		t.Error("Expected log to contain completed count")
	}
	if !strings.Contains(content, "2") {
		t.Error("Expected log to contain failed count")
	}
	if !strings.Contains(content, "SUMMARY") {
		t.Error("Expected log to contain summary header")
	}
}

// TestPerTaskLogs verifies detailed per-task logs are created
func TestPerTaskLogs(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	logger, err := NewFileLogger()
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}
	defer logger.Close()

	taskResult := models.TaskResult{
		Task: models.Task{
			Number: "5",
			Name:   "Test Task",
			Prompt: "Do something",
		},
		Status:         models.StatusGreen,
		Output:         "Task completed successfully",
		Duration:       30 * time.Second,
		RetryCount:     0,
		ReviewFeedback: "Looks good!",
	}

	err = logger.LogTaskResult(taskResult)
	if err != nil {
		t.Fatalf("LogTaskResult() error = %v", err)
	}

	// Verify task log file exists
	taskLogPath := filepath.Join(tmpDir, ".conductor", "logs", "tasks", "task-5.log")
	if _, err := os.Stat(taskLogPath); os.IsNotExist(err) {
		t.Errorf("Expected task log file %s to exist", taskLogPath)
	}

	// Read task log content
	content, err := os.ReadFile(taskLogPath)
	if err != nil {
		t.Fatalf("Failed to read task log: %v", err)
	}

	contentStr := string(content)

	// Verify task log contains key information
	if !strings.Contains(contentStr, "Test Task") {
		t.Error("Expected task log to contain task name")
	}
	if !strings.Contains(contentStr, "GREEN") {
		t.Error("Expected task log to contain status")
	}
	if !strings.Contains(contentStr, "Task completed successfully") {
		t.Error("Expected task log to contain output")
	}
	if !strings.Contains(contentStr, "30s") && !strings.Contains(contentStr, "30.0") {
		t.Error("Expected task log to contain duration")
	}
}

// TestCloseFlushesLogs verifies Close() properly flushes and closes log files
func TestCloseFlushesLogs(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	logger, err := NewFileLogger()
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}

	wave := models.Wave{
		Name:           "Wave 1",
		TaskNumbers:    []string{"1"},
		MaxConcurrency: 1,
	}

	logger.LogWaveStart(wave)

	// Close the logger
	err = logger.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify content was flushed to disk
	content := readRunLog(t, tmpDir)
	if !strings.Contains(content, "Wave 1") {
		t.Error("Expected log content to be flushed to disk after Close()")
	}
}

// TestNewFileLoggerWithCustomDir verifies FileLogger can use custom directory
func TestNewFileLoggerWithCustomDir(t *testing.T) {
	tmpDir := t.TempDir()
	customLogDir := filepath.Join(tmpDir, "custom", "logs")

	logger, err := NewFileLoggerWithDir(customLogDir)
	if err != nil {
		t.Fatalf("NewFileLoggerWithDir() error = %v", err)
	}
	defer logger.Close()

	// Verify custom directory was created
	if _, err := os.Stat(customLogDir); os.IsNotExist(err) {
		t.Errorf("Expected custom log directory %s to exist", customLogDir)
	}

	// Verify symlink exists in custom directory
	symlinkPath := filepath.Join(customLogDir, "latest.log")
	if _, err := os.Lstat(symlinkPath); err != nil {
		t.Errorf("Expected latest.log symlink in custom directory: %v", err)
	}
}

// TestConcurrentLogWrites verifies thread-safe logging
func TestConcurrentLogWrites(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	logger, err := NewFileLogger()
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}
	defer logger.Close()

	// Launch multiple goroutines logging concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			wave := models.Wave{
				Name:           "Wave " + string(rune('0'+n)),
				TaskNumbers:    []string{"1"},
				MaxConcurrency: 1,
			}
			logger.LogWaveStart(wave)
			logger.LogWaveComplete(wave, time.Second, []models.TaskResult{})
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify log file is readable and contains entries
	content := readRunLog(t, tmpDir)
	if len(content) == 0 {
		t.Error("Expected log file to contain entries from concurrent writes")
	}
}

// TestFileLoggerImplementsInterface verifies the FileLogger implements executor.Logger
func TestFileLoggerImplementsInterface(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	logger, err := NewFileLogger()
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}
	defer logger.Close()

	// Compile-time interface verification
	var _ interface {
		LogWaveStart(models.Wave)
		LogWaveComplete(models.Wave, time.Duration, []models.TaskResult)
		LogSummary(models.ExecutionResult)
	} = logger

	// Verify methods are callable
	wave := models.Wave{Name: "Test", TaskNumbers: []string{"1"}, MaxConcurrency: 1}
	result := models.ExecutionResult{TotalTasks: 1, Completed: 1, Failed: 0, Duration: 1 * time.Second}

	logger.LogWaveStart(wave)
	logger.LogWaveComplete(wave, 1*time.Second, []models.TaskResult{})
	logger.LogSummary(result)
}

// TestNewFileLoggerInvalidPath verifies error handling for invalid paths
func TestNewFileLoggerInvalidPath(t *testing.T) {
	// Try to create logger in a path that doesn't exist and can't be created
	// Use a path with null byte which is invalid on most file systems
	_, err := NewFileLoggerWithDir("/tmp/conductor-test\x00/logs")
	if err == nil {
		t.Error("Expected error when creating logger with invalid path")
	}
}

// TestLogTaskResultWithAllFields verifies task logging with all fields populated
func TestLogTaskResultWithAllFields(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	logger, err := NewFileLogger()
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}
	defer logger.Close()

	taskResult := models.TaskResult{
		Task: models.Task{
			Number: "10",
			Name:   "Complete Task",
			Prompt: "Test prompt",
		},
		Status:         models.StatusYellow,
		Output:         "Output text",
		Duration:       45 * time.Second,
		RetryCount:     1,
		ReviewFeedback: "Some feedback",
		Error:          fmt.Errorf("test error"),
	}

	err = logger.LogTaskResult(taskResult)
	if err != nil {
		t.Fatalf("LogTaskResult() error = %v", err)
	}

	// Verify task log file exists and contains all fields
	taskLogPath := filepath.Join(tmpDir, ".conductor", "logs", "tasks", "task-10.log")
	content, err := os.ReadFile(taskLogPath)
	if err != nil {
		t.Fatalf("Failed to read task log: %v", err)
	}

	contentStr := string(content)
	expectedFields := []string{
		"Complete Task",
		"YELLOW",
		"45.0",
		"Retry Count: 1",
		"Test prompt",
		"Output text",
		"Some feedback",
		"test error",
	}

	for _, field := range expectedFields {
		if !strings.Contains(contentStr, field) {
			t.Errorf("Expected task log to contain '%s'", field)
		}
	}
}

// TestCloseTwice verifies closing logger twice doesn't error
func TestCloseTwice(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	logger, err := NewFileLogger()
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}

	err = logger.Close()
	if err != nil {
		t.Errorf("First Close() error = %v", err)
	}

	// Second close should not error
	err = logger.Close()
	if err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}

// Helper function to read the current run log file
func readRunLog(t *testing.T, tmpDir string) string {
	t.Helper()

	symlinkPath := filepath.Join(tmpDir, ".conductor", "logs", "latest.log")
	content, err := os.ReadFile(symlinkPath)
	if err != nil {
		t.Fatalf("Failed to read run log: %v", err)
	}
	return string(content)
}
