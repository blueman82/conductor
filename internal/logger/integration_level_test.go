package logger

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// TestConsoleLoggerIntegration verifies full console logger behavior with log levels
func TestConsoleLoggerIntegration(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "warn")

	// Log at various levels
	logger.LogTrace("This is a trace message")
	logger.LogDebug("This is a debug message")
	logger.LogInfo("This is an info message")
	logger.LogWarn("This is a warning message")
	logger.LogError("This is an error message")

	// Wave and summary should be filtered (they're at INFO level)
	wave := models.Wave{
		Name:           "Test Wave",
		TaskNumbers:    []string{"1"},
		MaxConcurrency: 1,
	}
	logger.LogWaveStart(wave)
	logger.LogWaveComplete(wave, 5*time.Second)

	result := models.ExecutionResult{
		TotalTasks:  1,
		Completed:   1,
		Failed:      0,
		Duration:    5 * time.Second,
		FailedTasks: []models.TaskResult{},
	}
	logger.LogSummary(result)

	output := buf.String()

	// Only WARN and ERROR level messages should appear
	if strings.Contains(output, "trace message") {
		t.Error("trace message should be filtered at warn level")
	}
	if strings.Contains(output, "debug message") {
		t.Error("debug message should be filtered at warn level")
	}
	if strings.Contains(output, "info message") {
		t.Error("info message should be filtered at warn level")
	}
	if !strings.Contains(output, "warning message") {
		t.Error("warning message should appear at warn level")
	}
	if !strings.Contains(output, "error message") {
		t.Error("error message should appear at warn level")
	}

	// Wave and summary should be filtered
	if strings.Contains(output, "Test Wave") {
		t.Error("wave messages should be filtered at warn level")
	}
	if strings.Contains(output, "Execution Summary") {
		t.Error("summary should be filtered at warn level")
	}
}

// TestFileLoggerIntegration verifies full file logger behavior with log levels
func TestFileLoggerIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewFileLoggerWithDirAndLevel(tmpDir, "warn")
	if err != nil {
		t.Fatalf("NewFileLoggerWithDirAndLevel() error = %v", err)
	}
	defer logger.Close()

	// Log at various levels
	logger.LogTrace("This is a trace message")
	logger.LogDebug("This is a debug message")
	logger.LogInfo("This is an info message")
	logger.LogWarn("This is a warning message")
	logger.LogError("This is an error message")

	// Wave and summary should be filtered (they're at INFO level)
	wave := models.Wave{
		Name:           "Test Wave",
		TaskNumbers:    []string{"1"},
		MaxConcurrency: 1,
	}
	logger.LogWaveStart(wave)
	logger.LogWaveComplete(wave, 5*time.Second)

	result := models.ExecutionResult{
		TotalTasks:  1,
		Completed:   1,
		Failed:      0,
		Duration:    5 * time.Second,
		FailedTasks: []models.TaskResult{},
	}
	logger.LogSummary(result)

	// Flush and read the output
	logger.runLog.Sync()
	content := readFileLoggerOutput(t, logger)

	// Only WARN and ERROR level messages should appear
	if strings.Contains(content, "trace message") {
		t.Error("trace message should be filtered at warn level")
	}
	if strings.Contains(content, "debug message") {
		t.Error("debug message should be filtered at warn level")
	}
	if strings.Contains(content, "info message") {
		t.Error("info message should be filtered at warn level")
	}
	if !strings.Contains(content, "warning message") {
		t.Error("warning message should appear at warn level")
	}
	if !strings.Contains(content, "error message") {
		t.Error("error message should appear at warn level")
	}

	// Wave and summary should be filtered
	if strings.Contains(content, "Test Wave") {
		t.Error("wave messages should be filtered at warn level")
	}
	if strings.Contains(content, "EXECUTION SUMMARY") {
		t.Error("summary should be filtered at warn level")
	}
}

// TestLogLevelMessageFormatting verifies message formatting includes level tags
func TestLogLevelMessageFormatting(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "trace")

	logger.LogTrace("trace test")
	logger.LogDebug("debug test")
	logger.LogInfo("info test")
	logger.LogWarn("warn test")
	logger.LogError("error test")

	output := buf.String()

	// Verify level tags are present
	if !strings.Contains(output, "[TRACE]") {
		t.Error("expected [TRACE] tag in output")
	}
	if !strings.Contains(output, "[DEBUG]") {
		t.Error("expected [DEBUG] tag in output")
	}
	if !strings.Contains(output, "[INFO]") {
		t.Error("expected [INFO] tag in output")
	}
	if !strings.Contains(output, "[WARN]") {
		t.Error("expected [WARN] tag in output")
	}
	if !strings.Contains(output, "[ERROR]") {
		t.Error("expected [ERROR] tag in output")
	}

	// Verify all messages present
	if !strings.Contains(output, "trace test") {
		t.Error("expected trace message in output")
	}
	if !strings.Contains(output, "debug test") {
		t.Error("expected debug message in output")
	}
	if !strings.Contains(output, "info test") {
		t.Error("expected info message in output")
	}
	if !strings.Contains(output, "warn test") {
		t.Error("expected warn message in output")
	}
	if !strings.Contains(output, "error test") {
		t.Error("expected error message in output")
	}
}
