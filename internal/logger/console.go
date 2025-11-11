// Package logger provides logging implementations for Conductor execution.
//
// The logger package offers structured logging of execution progress at the
// wave and summary levels. Implementations are thread-safe and support various
// output destinations (console, file, etc.).
package logger

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// Log level constants for filtering
const (
	levelTrace int = 0
	levelDebug int = 1
	levelInfo  int = 2
	levelWarn  int = 3
	levelError int = 4
)

// ConsoleLogger logs execution progress to a writer with timestamps and thread safety.
// All output is prefixed with [HH:MM:SS] timestamps for tracking execution flow.
// It supports log level filtering to control message verbosity.
type ConsoleLogger struct {
	writer   io.Writer
	logLevel string
	mutex    sync.Mutex
}

// NewConsoleLogger creates a ConsoleLogger that writes to the provided io.Writer.
// If writer is nil, messages are silently discarded.
// logLevel determines the minimum log level for messages to be output.
// Valid levels: trace, debug, info, warn, error (case-insensitive).
// If logLevel is empty or invalid, defaults to "info".
func NewConsoleLogger(writer io.Writer, logLevel string) *ConsoleLogger {
	// Normalize and validate log level
	normalizedLevel := normalizeLogLevel(logLevel)

	return &ConsoleLogger{
		writer:   writer,
		logLevel: normalizedLevel,
		mutex:    sync.Mutex{},
	}
}

// normalizeLogLevel converts a log level string to lowercase and validates it.
// Returns "info" as default for empty or invalid levels.
func normalizeLogLevel(level string) string {
	normalized := strings.ToLower(strings.TrimSpace(level))

	validLevels := map[string]bool{
		"trace": true,
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if validLevels[normalized] {
		return normalized
	}

	return "info" // Default level
}

// shouldLog checks if a message at the given level should be logged.
// Returns true if messageLevel >= configured logLevel.
func (cl *ConsoleLogger) shouldLog(messageLevel string) bool {
	configuredLevel := logLevelToInt(cl.logLevel)
	msgLevel := logLevelToInt(messageLevel)
	return msgLevel >= configuredLevel
}

// logLevelToInt converts a log level string to its numeric value.
func logLevelToInt(level string) int {
	switch level {
	case "trace":
		return levelTrace
	case "debug":
		return levelDebug
	case "info":
		return levelInfo
	case "warn":
		return levelWarn
	case "error":
		return levelError
	default:
		return levelInfo // Default to info if unknown
	}
}

// LogTrace logs a trace-level message (most verbose).
// Format: "[HH:MM:SS] [TRACE] <message>"
func (cl *ConsoleLogger) LogTrace(message string) {
	cl.logWithLevel("TRACE", message)
}

// LogDebug logs a debug-level message.
// Format: "[HH:MM:SS] [DEBUG] <message>"
func (cl *ConsoleLogger) LogDebug(message string) {
	cl.logWithLevel("DEBUG", message)
}

// LogInfo logs an info-level message.
// Format: "[HH:MM:SS] [INFO] <message>"
func (cl *ConsoleLogger) LogInfo(message string) {
	cl.logWithLevel("INFO", message)
}

// LogWarn logs a warning-level message.
// Format: "[HH:MM:SS] [WARN] <message>"
func (cl *ConsoleLogger) LogWarn(message string) {
	cl.logWithLevel("WARN", message)
}

// LogError logs an error-level message.
// Format: "[HH:MM:SS] [ERROR] <message>"
func (cl *ConsoleLogger) LogError(message string) {
	cl.logWithLevel("ERROR", message)
}

// logWithLevel is a helper that logs a message at the specified level if filtering allows it.
func (cl *ConsoleLogger) logWithLevel(level string, message string) {
	if cl.writer == nil {
		return
	}

	// Check if this level should be logged
	if !cl.shouldLog(strings.ToLower(level)) {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	formatted := fmt.Sprintf("[%s] [%s] %s\n", timestamp(), level, message)
	cl.writer.Write([]byte(formatted))
}

// LogWaveStart logs the start of a wave execution at INFO level.
// Format: "[HH:MM:SS] Starting Wave <name>: <count> tasks"
func (cl *ConsoleLogger) LogWaveStart(wave models.Wave) {
	if cl.writer == nil {
		return
	}

	// Wave logging is at INFO level
	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	taskCount := len(wave.TaskNumbers)
	message := fmt.Sprintf("[%s] Starting Wave %s: %d tasks\n", timestamp(), wave.Name, taskCount)
	cl.writer.Write([]byte(message))
}

// LogWaveComplete logs the completion of a wave execution at INFO level.
// Format: "[HH:MM:SS] Wave <name> complete (<duration>)"
func (cl *ConsoleLogger) LogWaveComplete(wave models.Wave, duration time.Duration) {
	if cl.writer == nil {
		return
	}

	// Wave logging is at INFO level
	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	durationStr := formatDuration(duration)
	message := fmt.Sprintf("[%s] Wave %s complete (%s)\n", timestamp(), wave.Name, durationStr)
	cl.writer.Write([]byte(message))
}

// LogSummary logs the execution summary with completion statistics at INFO level.
// Format: "[HH:MM:SS] === Execution Summary ===\n[HH:MM:SS] Total tasks: <n>\n[HH:MM:SS] Completed: <n>\n[HH:MM:SS] Failed: <n>\n[HH:MM:SS] Duration: <d>\n"
func (cl *ConsoleLogger) LogSummary(result models.ExecutionResult) {
	if cl.writer == nil {
		return
	}

	// Summary logging is at INFO level
	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ts := timestamp()
	durationStr := formatDuration(result.Duration)

	output := fmt.Sprintf("[%s] === Execution Summary ===\n", ts)
	output += fmt.Sprintf("[%s] Total tasks: %d\n", ts, result.TotalTasks)
	output += fmt.Sprintf("[%s] Completed: %d\n", ts, result.Completed)
	output += fmt.Sprintf("[%s] Failed: %d\n", ts, result.Failed)
	output += fmt.Sprintf("[%s] Duration: %s\n", ts, durationStr)

	if result.Failed > 0 && len(result.FailedTasks) > 0 {
		output += fmt.Sprintf("[%s] Failed tasks:\n", ts)
		for _, failedTask := range result.FailedTasks {
			output += fmt.Sprintf("[%s]   - Task %s: %s\n", ts, failedTask.Task.Name, failedTask.Status)
		}
	}

	cl.writer.Write([]byte(output))
}

// timestamp returns the current time formatted as "15:04:05" (HH:MM:SS).
func timestamp() string {
	return time.Now().Format("15:04:05")
}

// formatDuration converts a time.Duration to a human-readable string.
// Examples: "5s", "1m30s", "2h15m"
func formatDuration(d time.Duration) string {
	switch {
	case d >= time.Hour:
		hours := d / time.Hour
		remainder := d % time.Hour
		if remainder == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		minutes := remainder / time.Minute
		remainder = remainder % time.Minute
		if remainder == 0 {
			return fmt.Sprintf("%dh%dm", hours, minutes)
		}
		seconds := remainder / time.Second
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	case d >= time.Minute:
		minutes := d / time.Minute
		remainder := d % time.Minute
		if remainder == 0 {
			return fmt.Sprintf("%dm", minutes)
		}
		seconds := remainder / time.Second
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	default:
		return fmt.Sprintf("%ds", int64(d.Seconds()))
	}
}

// NoOpLogger is a Logger implementation that discards all log messages.
// Useful for testing or when logging is disabled.
type NoOpLogger struct{}

// NewNoOpLogger creates a NoOpLogger instance.
func NewNoOpLogger() *NoOpLogger {
	return &NoOpLogger{}
}

// LogWaveStart is a no-op implementation.
func (n *NoOpLogger) LogWaveStart(wave models.Wave) {
}

// LogWaveComplete is a no-op implementation.
func (n *NoOpLogger) LogWaveComplete(wave models.Wave, duration time.Duration) {
}

// LogSummary is a no-op implementation.
func (n *NoOpLogger) LogSummary(result models.ExecutionResult) {
}
