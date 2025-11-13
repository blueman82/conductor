package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// FileLogger logs orchestrator events to files in .conductor/logs/ directory.
// It creates timestamped per-run log files, per-task detailed logs,
// and maintains a latest.log symlink pointing to the most recent run.
// It is thread-safe and implements the executor.Logger interface.
// It supports log level filtering to control message verbosity.
type FileLogger struct {
	logDir   string
	runLog   *os.File
	runFile  string
	tasksDir string
	logLevel string
	mu       sync.Mutex
}

// NewFileLogger creates a new FileLogger that writes to .conductor/logs/.
// It creates the log directory if it doesn't exist, opens a timestamped
// run log file, and creates/updates the latest.log symlink.
// Uses default log level "info".
func NewFileLogger() (*FileLogger, error) {
	// Default log directory is .conductor/logs/ in current working directory
	logDir := filepath.Join(".conductor", "logs")
	return NewFileLoggerWithDirAndLevel(logDir, "info")
}

// NewFileLoggerWithDir creates a new FileLogger with a custom log directory.
// This is useful for testing or custom deployments.
// Uses default log level "info".
func NewFileLoggerWithDir(logDir string) (*FileLogger, error) {
	return NewFileLoggerWithDirAndLevel(logDir, "info")
}

// NewFileLoggerWithDirAndLevel creates a new FileLogger with a custom log directory and log level.
// This is useful for testing or custom deployments.
func NewFileLoggerWithDirAndLevel(logDir string, logLevel string) (*FileLogger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create tasks subdirectory
	tasksDir := filepath.Join(logDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create tasks directory: %w", err)
	}

	// Generate timestamped filename: run-YYYYMMDD-HHMMSS.log
	timestamp := time.Now().Format("20060102-150405")
	runFile := filepath.Join(logDir, fmt.Sprintf("run-%s.log", timestamp))

	// Open run log file for writing
	file, err := os.OpenFile(runFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create run log file: %w", err)
	}

	// Create/update latest.log symlink
	symlinkPath := filepath.Join(logDir, "latest.log")

	// Remove existing symlink if it exists
	if _, err := os.Lstat(symlinkPath); err == nil {
		if err := os.Remove(symlinkPath); err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to remove old symlink: %w", err)
		}
	}

	// Create new symlink pointing to current run log
	if err := os.Symlink(filepath.Base(runFile), symlinkPath); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to create symlink: %w", err)
	}

	// Normalize and validate log level
	normalizedLevel := normalizeLogLevel(logLevel)

	logger := &FileLogger{
		logDir:   logDir,
		runLog:   file,
		runFile:  runFile,
		tasksDir: tasksDir,
		logLevel: normalizedLevel,
		mu:       sync.Mutex{},
	}

	// Write header to run log
	logger.writeRunLog("=== Conductor Run Log ===\n")
	logger.writeRunLog(fmt.Sprintf("Started at: %s\n\n", time.Now().Format(time.RFC3339)))

	return logger, nil
}

// shouldLog checks if a message at the given level should be logged.
// Returns true if messageLevel >= configured logLevel.
func (fl *FileLogger) shouldLog(messageLevel string) bool {
	configuredLevel := logLevelToInt(fl.logLevel)
	msgLevel := logLevelToInt(messageLevel)
	return msgLevel >= configuredLevel
}

// LogTrace logs a trace-level message (most verbose).
func (fl *FileLogger) LogTrace(message string) {
	fl.logWithLevel("TRACE", message)
}

// LogDebug logs a debug-level message.
func (fl *FileLogger) LogDebug(message string) {
	fl.logWithLevel("DEBUG", message)
}

// LogInfo logs an info-level message.
func (fl *FileLogger) LogInfo(message string) {
	fl.logWithLevel("INFO", message)
}

// LogWarn logs a warning-level message.
func (fl *FileLogger) LogWarn(message string) {
	fl.logWithLevel("WARN", message)
}

// LogError logs an error-level message.
func (fl *FileLogger) LogError(message string) {
	fl.logWithLevel("ERROR", message)
}

// logWithLevel is a helper that logs a message at the specified level if filtering allows it.
func (fl *FileLogger) logWithLevel(level string, message string) {
	// Check if this level should be logged
	levelLower := strings.ToLower(level)
	if !fl.shouldLog(levelLower) {
		return
	}

	formatted := fmt.Sprintf("[%s] [%s] %s\n", time.Now().Format("15:04:05"), level, message)
	fl.writeRunLog(formatted)
}

// LogWaveStart logs the start of a wave execution at INFO level.
// It displays the wave name, number of tasks, and max concurrency.
func (fl *FileLogger) LogWaveStart(wave models.Wave) {
	// Wave logging is at INFO level
	if !fl.shouldLog("info") {
		return
	}

	taskCount := len(wave.TaskNumbers)
	taskLabel := "task"
	if taskCount != 1 {
		taskLabel = "tasks"
	}

	message := fmt.Sprintf(
		"[%s] Starting %s: %d %s (max concurrency: %d)\n",
		time.Now().Format("15:04:05"),
		wave.Name,
		taskCount,
		taskLabel,
		wave.MaxConcurrency,
	)

	fl.writeRunLog(message)
}

// LogWaveComplete logs the completion of a wave execution at INFO level.
// It displays the wave name and duration.
func (fl *FileLogger) LogWaveComplete(wave models.Wave, duration time.Duration, results []models.TaskResult) {
	// Wave logging is at INFO level
	if !fl.shouldLog("info") {
		return
	}

	message := fmt.Sprintf(
		"[%s] %s complete: duration %.1fs\n",
		time.Now().Format("15:04:05"),
		wave.Name,
		duration.Seconds(),
	)

	fl.writeRunLog(message)
}

// LogSummary logs the execution summary with final statistics at INFO level.
// It displays total tasks, completed, failed, duration, and overall status.
func (fl *FileLogger) LogSummary(result models.ExecutionResult) {
	// Summary logging is at INFO level
	if !fl.shouldLog("info") {
		return
	}

	timestamp := time.Now().Format("15:04:05")

	// Determine status
	status := "SUCCESS"
	if result.Failed > 0 {
		if result.Completed == 0 {
			status = "FAILED"
		} else {
			status = "PARTIAL"
		}
	}

	// Build summary output
	message := fmt.Sprintf(
		"\n[%s] === EXECUTION SUMMARY ===\n"+
			"[%s] Total tasks:  %d\n"+
			"[%s] Completed:    %d\n"+
			"[%s] Failed:       %d\n"+
			"[%s] Total time:   %.1fs\n"+
			"[%s] Status:       %s (%d/%d tasks passed)\n"+
			"[%s] Completed at: %s\n",
		timestamp,
		timestamp,
		result.TotalTasks,
		timestamp,
		result.Completed,
		timestamp,
		result.Failed,
		timestamp,
		result.Duration.Seconds(),
		timestamp,
		status,
		result.Completed,
		result.TotalTasks,
		timestamp,
		time.Now().Format(time.RFC3339),
	)

	fl.writeRunLog(message)
}

// LogTaskResult logs detailed information about a task execution.
// It creates a separate log file for each task in the tasks/ subdirectory.
func (fl *FileLogger) LogTaskResult(result models.TaskResult) error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	// Create task log file: tasks/task-N.log
	taskLogPath := filepath.Join(fl.tasksDir, fmt.Sprintf("task-%s.log", result.Task.Number))

	file, err := os.OpenFile(taskLogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create task log file: %w", err)
	}
	defer file.Close()

	// Write task details
	content := fmt.Sprintf("=== Task %s: %s ===\n", result.Task.Number, result.Task.Name)
	content += fmt.Sprintf("Status: %s\n", result.Status)
	content += fmt.Sprintf("Duration: %.1fs\n", result.Duration.Seconds())
	content += fmt.Sprintf("Retry Count: %d\n", result.RetryCount)
	content += "\n"

	if result.Task.Prompt != "" {
		content += fmt.Sprintf("Prompt:\n%s\n\n", result.Task.Prompt)
	}

	if result.Output != "" {
		content += fmt.Sprintf("Output:\n%s\n\n", result.Output)
	}

	if result.ReviewFeedback != "" {
		content += fmt.Sprintf("QC Feedback:\n%s\n\n", result.ReviewFeedback)
	}

	if result.Error != nil {
		content += fmt.Sprintf("Error:\n%v\n\n", result.Error)
	}

	content += fmt.Sprintf("Completed at: %s\n", time.Now().Format(time.RFC3339))

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write task log: %w", err)
	}

	return nil
}

// LogProgress logs the current execution progress (no-op for file logger).
// Progress is displayed on console but not written to log files.
func (fl *FileLogger) LogProgress(results []models.TaskResult) {
	// No-op: progress bars are console-only for now
}

// Close flushes and closes the run log file.
// It should be called when the logger is no longer needed.
func (fl *FileLogger) Close() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	if fl.runLog != nil {
		if err := fl.runLog.Sync(); err != nil {
			return fmt.Errorf("failed to sync run log: %w", err)
		}
		if err := fl.runLog.Close(); err != nil {
			return fmt.Errorf("failed to close run log: %w", err)
		}
		fl.runLog = nil
	}

	return nil
}

// writeRunLog is a thread-safe helper to write to the run log file.
func (fl *FileLogger) writeRunLog(message string) {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	if fl.runLog != nil {
		fl.runLog.WriteString(message)
		// Flush after each write for real-time logging
		fl.runLog.Sync()
	}
}
