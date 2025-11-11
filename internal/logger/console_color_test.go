package logger

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/harrison/conductor/internal/models"
)

// TestColorOutputDetection verifies color output is enabled for terminal, disabled for buffers
func TestColorOutputDetection(t *testing.T) {
	tests := []struct {
		name                string
		writer              io.Writer
		expectedColorOutput bool
	}{
		{
			name:                "buffer should disable colors",
			writer:              &bytes.Buffer{},
			expectedColorOutput: false,
		},
		{
			name:                "nil writer should disable colors",
			writer:              nil,
			expectedColorOutput: false,
		},
		// Note: os.Stdout/os.Stderr detection is tested separately
		// as it depends on TTY presence which varies by environment
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewConsoleLogger(tt.writer, "info")
			if logger.colorOutput != tt.expectedColorOutput {
				t.Errorf("expected colorOutput=%v, got %v", tt.expectedColorOutput, logger.colorOutput)
			}
		})
	}
}

// TestColorOutputFormatting verifies color codes are applied correctly
func TestColorOutputFormatting(t *testing.T) {
	// Use a buffer to capture output
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "trace")

	// Force color output ON for testing (even though buffer normally disables it)
	logger.colorOutput = true

	// Temporarily enable color globally for the color library
	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	// Log messages at different levels
	logger.LogTrace("trace message")
	logger.LogDebug("debug message")
	logger.LogInfo("info message")
	logger.LogWarn("warn message")
	logger.LogError("error message")

	output := buf.String()

	// Verify messages appear (color codes will be present but we check for content)
	if !strings.Contains(output, "trace message") {
		t.Error("expected trace message in output")
	}
	if !strings.Contains(output, "debug message") {
		t.Error("expected debug message in output")
	}
	if !strings.Contains(output, "info message") {
		t.Error("expected info message in output")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("expected warn message in output")
	}
	if !strings.Contains(output, "error message") {
		t.Error("expected error message in output")
	}

	// Verify ANSI color codes are present (basic check)
	if !strings.Contains(output, "\x1b[") {
		t.Error("expected ANSI color codes in output when colorOutput=true")
	}
}

// TestPlainTextOutputFormatting verifies no color codes when colorOutput is false
func TestPlainTextOutputFormatting(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "trace")

	// colorOutput should default to false for buffer
	if logger.colorOutput {
		t.Error("expected colorOutput=false for buffer writer")
	}

	// Log messages at different levels
	logger.LogTrace("trace message")
	logger.LogDebug("debug message")
	logger.LogInfo("info message")
	logger.LogWarn("warn message")
	logger.LogError("error message")

	output := buf.String()

	// Verify messages appear
	if !strings.Contains(output, "trace message") {
		t.Error("expected trace message in output")
	}

	// Verify NO ANSI color codes are present
	if strings.Contains(output, "\x1b[") {
		t.Error("expected NO ANSI color codes in output when colorOutput=false")
	}
}

// TestColorInWaveMessages verifies color codes in wave logging
func TestColorInWaveMessages(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	// Force color output ON for testing
	logger.colorOutput = true

	// Temporarily enable color globally for the color library
	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	wave := models.Wave{
		Name:           "Wave 1",
		TaskNumbers:    []string{"1", "2"},
		MaxConcurrency: 2,
	}

	logger.LogWaveStart(wave)
	logger.LogWaveComplete(wave, 5*time.Second)

	output := buf.String()

	// Verify wave name appears
	if !strings.Contains(output, "Wave 1") {
		t.Error("expected wave name in output")
	}

	// Verify ANSI color codes are present
	if !strings.Contains(output, "\x1b[") {
		t.Error("expected ANSI color codes in wave output when colorOutput=true")
	}
}

// TestColorInSummary verifies color codes in summary logging
func TestColorInSummary(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	// Force color output ON for testing
	logger.colorOutput = true

	// Temporarily enable color globally for the color library
	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	result := models.ExecutionResult{
		TotalTasks: 10,
		Completed:  8,
		Failed:     2,
		Duration:   5 * time.Second,
		FailedTasks: []models.TaskResult{
			{
				Task: models.Task{
					Name: "Task 1",
				},
				Status: models.StatusRed,
			},
		},
	}

	logger.LogSummary(result)

	output := buf.String()

	// Verify summary content appears
	if !strings.Contains(output, "Execution Summary") {
		t.Error("expected summary header in output")
	}
	if !strings.Contains(output, "Task 1") {
		t.Error("expected failed task in output")
	}

	// Verify ANSI color codes are present
	if !strings.Contains(output, "\x1b[") {
		t.Error("expected ANSI color codes in summary output when colorOutput=true")
	}
}

// TestPlainTextInWaveMessages verifies no color codes in wave logging when disabled
func TestPlainTextInWaveMessages(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	// colorOutput should be false by default for buffers
	if logger.colorOutput {
		t.Error("expected colorOutput=false for buffer writer")
	}

	wave := models.Wave{
		Name:           "Wave 1",
		TaskNumbers:    []string{"1", "2"},
		MaxConcurrency: 2,
	}

	logger.LogWaveStart(wave)
	logger.LogWaveComplete(wave, 5*time.Second)

	output := buf.String()

	// Verify wave name appears
	if !strings.Contains(output, "Wave 1") {
		t.Error("expected wave name in output")
	}

	// Verify NO ANSI color codes are present
	if strings.Contains(output, "\x1b[") {
		t.Error("expected NO ANSI color codes in wave output when colorOutput=false")
	}
}

// TestPlainTextInSummary verifies no color codes in summary logging when disabled
func TestPlainTextInSummary(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	// colorOutput should be false by default for buffers
	if logger.colorOutput {
		t.Error("expected colorOutput=false for buffer writer")
	}

	result := models.ExecutionResult{
		TotalTasks: 10,
		Completed:  8,
		Failed:     2,
		Duration:   5 * time.Second,
		FailedTasks: []models.TaskResult{
			{
				Task: models.Task{
					Name: "Task 1",
				},
				Status: models.StatusRed,
			},
		},
	}

	logger.LogSummary(result)

	output := buf.String()

	// Verify summary content appears
	if !strings.Contains(output, "Execution Summary") {
		t.Error("expected summary header in output")
	}

	// Verify NO ANSI color codes are present
	if strings.Contains(output, "\x1b[") {
		t.Error("expected NO ANSI color codes in summary output when colorOutput=false")
	}
}

// TestColorLevelFormatting verifies each log level gets the correct color
func TestColorLevelFormatting(t *testing.T) {
	tests := []struct {
		name     string
		logFunc  func(*ConsoleLogger, string)
		message  string
		contains string // substring to verify the level appears
	}{
		{
			name:     "trace level",
			logFunc:  (*ConsoleLogger).LogTrace,
			message:  "trace test",
			contains: "TRACE",
		},
		{
			name:     "debug level",
			logFunc:  (*ConsoleLogger).LogDebug,
			message:  "debug test",
			contains: "DEBUG",
		},
		{
			name:     "info level",
			logFunc:  (*ConsoleLogger).LogInfo,
			message:  "info test",
			contains: "INFO",
		},
		{
			name:     "warn level",
			logFunc:  (*ConsoleLogger).LogWarn,
			message:  "warn test",
			contains: "WARN",
		},
		{
			name:     "error level",
			logFunc:  (*ConsoleLogger).LogError,
			message:  "error test",
			contains: "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewConsoleLogger(buf, "trace")
			logger.colorOutput = true

			// Temporarily enable color globally for the color library
			oldNoColor := color.NoColor
			color.NoColor = false
			defer func() { color.NoColor = oldNoColor }()

			tt.logFunc(logger, tt.message)

			output := buf.String()

			// Verify message appears
			if !strings.Contains(output, tt.message) {
				t.Errorf("expected message %q in output", tt.message)
			}

			// Verify level appears
			if !strings.Contains(output, tt.contains) {
				t.Errorf("expected level %q in output", tt.contains)
			}

			// Verify ANSI codes present
			if !strings.Contains(output, "\x1b[") {
				t.Error("expected ANSI color codes in output")
			}
		})
	}
}
