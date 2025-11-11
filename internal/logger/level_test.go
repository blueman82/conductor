package logger

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// TestLogLevelFiltering verifies that messages are filtered based on log level
func TestLogLevelFiltering(t *testing.T) {
	tests := []struct {
		name            string
		logLevel        string
		messageLevel    string
		message         string
		shouldAppear    bool
	}{
		// trace level - should see everything
		{name: "trace sees trace", logLevel: "trace", messageLevel: "trace", message: "trace msg", shouldAppear: true},
		{name: "trace sees debug", logLevel: "trace", messageLevel: "debug", message: "debug msg", shouldAppear: true},
		{name: "trace sees info", logLevel: "trace", messageLevel: "info", message: "info msg", shouldAppear: true},
		{name: "trace sees warn", logLevel: "trace", messageLevel: "warn", message: "warn msg", shouldAppear: true},
		{name: "trace sees error", logLevel: "trace", messageLevel: "error", message: "error msg", shouldAppear: true},

		// debug level - should not see trace
		{name: "debug blocks trace", logLevel: "debug", messageLevel: "trace", message: "trace msg", shouldAppear: false},
		{name: "debug sees debug", logLevel: "debug", messageLevel: "debug", message: "debug msg", shouldAppear: true},
		{name: "debug sees info", logLevel: "debug", messageLevel: "info", message: "info msg", shouldAppear: true},
		{name: "debug sees warn", logLevel: "debug", messageLevel: "warn", message: "warn msg", shouldAppear: true},
		{name: "debug sees error", logLevel: "debug", messageLevel: "error", message: "error msg", shouldAppear: true},

		// info level - should not see trace/debug
		{name: "info blocks trace", logLevel: "info", messageLevel: "trace", message: "trace msg", shouldAppear: false},
		{name: "info blocks debug", logLevel: "info", messageLevel: "debug", message: "debug msg", shouldAppear: false},
		{name: "info sees info", logLevel: "info", messageLevel: "info", message: "info msg", shouldAppear: true},
		{name: "info sees warn", logLevel: "info", messageLevel: "warn", message: "warn msg", shouldAppear: true},
		{name: "info sees error", logLevel: "info", messageLevel: "error", message: "error msg", shouldAppear: true},

		// warn level - should only see warn/error
		{name: "warn blocks trace", logLevel: "warn", messageLevel: "trace", message: "trace msg", shouldAppear: false},
		{name: "warn blocks debug", logLevel: "warn", messageLevel: "debug", message: "debug msg", shouldAppear: false},
		{name: "warn blocks info", logLevel: "warn", messageLevel: "info", message: "info msg", shouldAppear: false},
		{name: "warn sees warn", logLevel: "warn", messageLevel: "warn", message: "warn msg", shouldAppear: true},
		{name: "warn sees error", logLevel: "warn", messageLevel: "error", message: "error msg", shouldAppear: true},

		// error level - should only see error
		{name: "error blocks trace", logLevel: "error", messageLevel: "trace", message: "trace msg", shouldAppear: false},
		{name: "error blocks debug", logLevel: "error", messageLevel: "debug", message: "debug msg", shouldAppear: false},
		{name: "error blocks info", logLevel: "error", messageLevel: "info", message: "info msg", shouldAppear: false},
		{name: "error blocks warn", logLevel: "error", messageLevel: "warn", message: "warn msg", shouldAppear: false},
		{name: "error sees error", logLevel: "error", messageLevel: "error", message: "error msg", shouldAppear: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewConsoleLogger(buf, tt.logLevel)

			// Log message at specified level
			switch tt.messageLevel {
			case "trace":
				logger.LogTrace(tt.message)
			case "debug":
				logger.LogDebug(tt.message)
			case "info":
				logger.LogInfo(tt.message)
			case "warn":
				logger.LogWarn(tt.message)
			case "error":
				logger.LogError(tt.message)
			}

			output := buf.String()
			contains := strings.Contains(output, tt.message)

			if tt.shouldAppear && !contains {
				t.Errorf("Expected message %q to appear in output, but it didn't. Output: %q", tt.message, output)
			}
			if !tt.shouldAppear && contains {
				t.Errorf("Expected message %q NOT to appear in output, but it did. Output: %q", tt.message, output)
			}
		})
	}
}

// TestConsoleLoggerWithLogLevel verifies ConsoleLogger respects log level
func TestConsoleLoggerWithLogLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "warn")

	// These should be filtered out
	logger.LogTrace("trace message")
	logger.LogDebug("debug message")
	logger.LogInfo("info message")

	// These should appear
	logger.LogWarn("warn message")
	logger.LogError("error message")

	output := buf.String()

	// Verify filtered messages don't appear
	if strings.Contains(output, "trace message") {
		t.Error("trace message should be filtered at warn level")
	}
	if strings.Contains(output, "debug message") {
		t.Error("debug message should be filtered at warn level")
	}
	if strings.Contains(output, "info message") {
		t.Error("info message should be filtered at warn level")
	}

	// Verify unfiltered messages appear
	if !strings.Contains(output, "warn message") {
		t.Error("warn message should appear at warn level")
	}
	if !strings.Contains(output, "error message") {
		t.Error("error message should appear at warn level")
	}
}

// TestExistingMethodsRespectLogLevel verifies LogWaveStart/LogWaveComplete/LogSummary respect log level
func TestExistingMethodsRespectLogLevel(t *testing.T) {
	tests := []struct {
		name          string
		logLevel      string
		shouldAppear  bool
	}{
		{name: "info level shows waves", logLevel: "info", shouldAppear: true},
		{name: "warn level hides waves", logLevel: "warn", shouldAppear: false},
		{name: "error level hides waves", logLevel: "error", shouldAppear: false},
		{name: "trace level shows waves", logLevel: "trace", shouldAppear: true},
		{name: "debug level shows waves", logLevel: "debug", shouldAppear: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewConsoleLogger(buf, tt.logLevel)

			wave := models.Wave{
				Name:           "Wave 1",
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
			contains := strings.Contains(output, "Wave 1")

			if tt.shouldAppear && !contains {
				t.Errorf("Expected wave messages to appear at %s level, but they didn't", tt.logLevel)
			}
			if !tt.shouldAppear && contains {
				t.Errorf("Expected wave messages NOT to appear at %s level, but they did", tt.logLevel)
			}
		})
	}
}

// TestLogLevelEdgeCases verifies handling of invalid/unknown log levels
func TestLogLevelEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		logLevel       string
		expectedLevel  string
	}{
		{name: "empty string defaults to info", logLevel: "", expectedLevel: "info"},
		{name: "unknown level defaults to info", logLevel: "unknown", expectedLevel: "info"},
		{name: "uppercase level normalized", logLevel: "DEBUG", expectedLevel: "debug"},
		{name: "mixed case normalized", logLevel: "WaRn", expectedLevel: "warn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewConsoleLogger(buf, tt.logLevel)

			// Test that the logger behaves as if it has the expected level
			// If expected level is info, debug should be filtered, info should pass
			logger.LogDebug("debug message")
			logger.LogInfo("info message")

			output := buf.String()

			if tt.expectedLevel == "info" {
				if strings.Contains(output, "debug message") {
					t.Error("debug message should be filtered when defaulting to info level")
				}
				if !strings.Contains(output, "info message") {
					t.Error("info message should appear when defaulting to info level")
				}
			}
		})
	}
}

// TestFileLoggerWithLogLevel verifies FileLogger respects log level
func TestFileLoggerWithLogLevel(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewFileLoggerWithDirAndLevel(tmpDir, "warn")
	if err != nil {
		t.Fatalf("NewFileLoggerWithDirAndLevel() error = %v", err)
	}
	defer logger.Close()

	// These should be filtered out
	logger.LogTrace("trace message")
	logger.LogDebug("debug message")
	logger.LogInfo("info message")

	// These should appear
	logger.LogWarn("warn message")
	logger.LogError("error message")

	// Verify by reading the run log
	content := readFileLoggerOutput(t, logger)

	// Verify filtered messages don't appear
	if strings.Contains(content, "trace message") {
		t.Error("trace message should be filtered at warn level")
	}
	if strings.Contains(content, "debug message") {
		t.Error("debug message should be filtered at warn level")
	}
	if strings.Contains(content, "info message") {
		t.Error("info message should be filtered at warn level")
	}

	// Verify unfiltered messages appear
	if !strings.Contains(content, "warn message") {
		t.Error("warn message should appear at warn level")
	}
	if !strings.Contains(content, "error message") {
		t.Error("error message should appear at warn level")
	}
}

// TestNewConsoleLoggerWithLevel verifies constructor with log level
func TestNewConsoleLoggerWithLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "debug")

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	// Verify log level is set correctly by testing filtering behavior
	logger.LogTrace("should not appear")
	logger.LogDebug("should appear")

	output := buf.String()
	if strings.Contains(output, "should not appear") {
		t.Error("trace should be filtered at debug level")
	}
	if !strings.Contains(output, "should appear") {
		t.Error("debug should appear at debug level")
	}
}

// TestNewFileLoggerUsesDefaultLevel verifies NewFileLogger uses default info level
func TestNewFileLoggerUsesDefaultLevel(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewFileLoggerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("NewFileLoggerWithDir() error = %v", err)
	}
	defer logger.Close()

	// At info level, debug should be filtered, info should pass
	logger.LogDebug("debug message")
	logger.LogInfo("info message")

	content := readFileLoggerOutput(t, logger)

	if strings.Contains(content, "debug message") {
		t.Error("debug should be filtered at default info level")
	}
	if !strings.Contains(content, "info message") {
		t.Error("info should appear at default info level")
	}
}

// Helper to read FileLogger output
func readFileLoggerOutput(t *testing.T, logger *FileLogger) string {
	t.Helper()

	// Sync to ensure everything is written
	logger.runLog.Sync()

	// Read the run log file
	content, err := os.ReadFile(logger.runFile)
	if err != nil {
		t.Fatalf("Failed to read run log: %v", err)
	}

	return string(content)
}
