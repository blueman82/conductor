// Package logger provides logging implementations for Conductor execution.
//
// The logger package offers structured logging of execution progress at the
// wave and summary levels. Implementations are thread-safe and support various
// output destinations (console, file, etc.).
package logger

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// ConsoleLogger logs execution progress to a writer with timestamps and thread safety.
// All output is prefixed with [HH:MM:SS] timestamps for tracking execution flow.
type ConsoleLogger struct {
	writer io.Writer
	mutex  sync.Mutex
}

// NewConsoleLogger creates a ConsoleLogger that writes to the provided io.Writer.
// If writer is nil, messages are silently discarded.
func NewConsoleLogger(writer io.Writer) *ConsoleLogger {
	return &ConsoleLogger{
		writer: writer,
		mutex:  sync.Mutex{},
	}
}

// LogWaveStart logs the start of a wave execution.
// Format: "[HH:MM:SS] Starting Wave <name>: <count> tasks"
func (cl *ConsoleLogger) LogWaveStart(wave models.Wave) {
	if cl.writer == nil {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	taskCount := len(wave.TaskNumbers)
	message := fmt.Sprintf("[%s] Starting Wave %s: %d tasks\n", timestamp(), wave.Name, taskCount)
	cl.writer.Write([]byte(message))
}

// LogWaveComplete logs the completion of a wave execution.
// Format: "[HH:MM:SS] Wave <name> complete (<duration>)"
func (cl *ConsoleLogger) LogWaveComplete(wave models.Wave, duration time.Duration) {
	if cl.writer == nil {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	durationStr := formatDuration(duration)
	message := fmt.Sprintf("[%s] Wave %s complete (%s)\n", timestamp(), wave.Name, durationStr)
	cl.writer.Write([]byte(message))
}

// LogSummary logs the execution summary with completion statistics.
// Format: "[HH:MM:SS] === Execution Summary ===\n[HH:MM:SS] Total tasks: <n>\n[HH:MM:SS] Completed: <n>\n[HH:MM:SS] Failed: <n>\n[HH:MM:SS] Duration: <d>\n"
func (cl *ConsoleLogger) LogSummary(result models.ExecutionResult) {
	if cl.writer == nil {
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
