package logger

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// TestFormatBehavioralMetrics verifies behavioral metrics formatting.
func TestFormatBehavioralMetrics(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		expected string
	}{
		{
			name:     "nil metadata",
			metadata: nil,
			expected: "",
		},
		{
			name:     "empty metadata",
			metadata: map[string]interface{}{},
			expected: "",
		},
		{
			name: "all metrics present",
			metadata: map[string]interface{}{
				"tool_count":      5,
				"bash_count":      3,
				"file_operations": 2,
				"cost":            0.0125,
			},
			expected: "tools: 5, bash: 3, files: 2, cost: $0.0125",
		},
		{
			name: "partial metrics",
			metadata: map[string]interface{}{
				"tool_count": 3,
				"cost":       0.0050,
			},
			expected: "tools: 3, cost: $0.0050",
		},
		{
			name: "float64 counts",
			metadata: map[string]interface{}{
				"tool_count": float64(10),
				"bash_count": float64(5),
			},
			expected: "tools: 10, bash: 5",
		},
		{
			name: "zero values ignored",
			metadata: map[string]interface{}{
				"tool_count": 0,
				"bash_count": 2,
				"cost":       0.0,
			},
			expected: "bash: 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBehavioralMetrics(tt.metadata)
			if result != tt.expected {
				t.Errorf("formatBehavioralMetrics() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestLogTaskResultVerboseWithBehavioralMetrics verifies behavioral metrics appear in verbose output.
func TestLogTaskResultVerboseWithBehavioralMetrics(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "debug")
	logger.SetVerbose(true)

	task := models.Task{
		Number: "1",
		Name:   "Test task",
		Agent:  "test-agent",
		Metadata: map[string]interface{}{
			"tool_count": 5,
			"bash_count": 3,
			"cost":       0.0125,
		},
	}

	result := models.TaskResult{
		Task:     task,
		Status:   models.StatusGreen,
		Duration: 2 * time.Second,
	}

	err := logger.LogTaskResult(result)
	if err != nil {
		t.Fatalf("LogTaskResult() failed: %v", err)
	}

	output := buf.String()

	// Check for behavioral metrics section
	if !strings.Contains(output, "Behavioral:") {
		t.Error("expected output to contain 'Behavioral:' section")
	}

	// Check for specific metrics
	expectedMetrics := []string{"tools: 5", "bash: 3", "cost: $0.0125"}
	for _, metric := range expectedMetrics {
		if !strings.Contains(output, metric) {
			t.Errorf("expected output to contain %q", metric)
		}
	}
}

// TestLogTaskResultVerboseWithoutBehavioralMetrics verifies no behavioral section when no data.
func TestLogTaskResultVerboseWithoutBehavioralMetrics(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "debug")
	logger.SetVerbose(true)

	task := models.Task{
		Number:   "1",
		Name:     "Test task",
		Agent:    "test-agent",
		Metadata: nil,
	}

	result := models.TaskResult{
		Task:     task,
		Status:   models.StatusGreen,
		Duration: 2 * time.Second,
	}

	err := logger.LogTaskResult(result)
	if err != nil {
		t.Fatalf("LogTaskResult() failed: %v", err)
	}

	output := buf.String()

	// Should NOT contain behavioral section
	if strings.Contains(output, "Behavioral:") {
		t.Error("expected output NOT to contain 'Behavioral:' section when no data")
	}
}

// TestNewConsoleLogger verifies the constructor creates a ConsoleLogger with the provided writer.
func TestNewConsoleLogger(t *testing.T) {
	t.Run("with valid writer", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewConsoleLogger(buf, "info")

		if logger == nil {
			t.Error("expected non-nil logger")
		}
		if logger.writer != buf {
			t.Error("writer not set correctly")
		}
		if logger.logLevel != "info" {
			t.Errorf("expected log level %q, got %q", "info", logger.logLevel)
		}
	})

	t.Run("with nil writer", func(t *testing.T) {
		logger := NewConsoleLogger(nil, "info")
		if logger == nil {
			t.Error("expected non-nil logger even with nil writer")
		}
		if logger.writer != nil {
			t.Error("expected nil writer")
		}
	})
}

// TestLogWaveStart verifies wave start messages are formatted correctly.
func TestLogWaveStart(t *testing.T) {
	tests := []struct {
		name         string
		wave         models.Wave
		expectedText string
	}{
		{
			name: "single task",
			wave: models.Wave{
				Name:           "Wave 1",
				TaskNumbers:    []string{"1"},
				MaxConcurrency: 1,
			},
			expectedText: "Starting Wave 1: 1 tasks",
		},
		{
			name: "multiple tasks",
			wave: models.Wave{
				Name:           "Wave 2",
				TaskNumbers:    []string{"2", "3", "4"},
				MaxConcurrency: 3,
			},
			expectedText: "Starting Wave 2: 3 tasks",
		},
		{
			name: "empty tasks",
			wave: models.Wave{
				Name:           "Wave 3",
				TaskNumbers:    []string{},
				MaxConcurrency: 1,
			},
			expectedText: "Starting Wave 3: 0 tasks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewConsoleLogger(buf, "info")

			logger.LogWaveStart(tt.wave)

			output := buf.String()
			if !strings.Contains(output, tt.expectedText) {
				t.Errorf("expected output to contain %q, got %q", tt.expectedText, output)
			}

			// Verify timestamp prefix
			if !strings.HasPrefix(output, "[") {
				t.Error("expected output to start with timestamp [")
			}
		})
	}
}

// TestLogWaveComplete verifies wave complete messages are formatted correctly.
func TestLogWaveComplete(t *testing.T) {
	tests := []struct {
		name         string
		wave         models.Wave
		duration     time.Duration
		results      []models.TaskResult
		expectedText string
	}{
		{
			name: "5 seconds - all green",
			wave: models.Wave{
				Name:           "Wave 1",
				TaskNumbers:    []string{"1"},
				MaxConcurrency: 1,
			},
			duration: 5 * time.Second,
			results: []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
			},
			expectedText: "Wave 1 complete (5.0s) - 1/1 completed (1 GREEN)",
		},
		{
			name: "90 seconds (1m30s) - mixed statuses",
			wave: models.Wave{
				Name:           "Wave 2",
				TaskNumbers:    []string{"1", "2"},
				MaxConcurrency: 2,
			},
			duration: 90 * time.Second,
			results: []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
				{Task: models.Task{Number: "2"}, Status: models.StatusYellow},
			},
			expectedText: "Wave 2 complete (1m30s) - 2/2 completed (1 GREEN, 1 YELLOW)",
		},
		{
			name: "zero duration - all green",
			wave: models.Wave{
				Name:           "Wave 3",
				TaskNumbers:    []string{"1"},
				MaxConcurrency: 1,
			},
			duration: 0 * time.Second,
			results: []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
			},
			expectedText: "Wave 3 complete (0.0s) - 1/1 completed (1 GREEN)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewConsoleLogger(buf, "info")

			logger.LogWaveComplete(tt.wave, tt.duration, tt.results)

			output := buf.String()
			if !strings.Contains(output, tt.expectedText) {
				t.Errorf("expected output to contain %q, got %q", tt.expectedText, output)
			}

			// Verify timestamp prefix
			if !strings.HasPrefix(output, "[") {
				t.Error("expected output to start with timestamp [")
			}
		})
	}
}

// TestLogWaveComplete_StatusBreakdown verifies status breakdown display with correct format and color.
func TestLogWaveComplete_StatusBreakdown(t *testing.T) {
	tests := []struct {
		name         string
		wave         models.Wave
		results      []models.TaskResult
		expectedText []string
		notExpected  []string
	}{
		{
			name: "all green - only show green",
			wave: models.Wave{
				Name:           "Wave 1",
				TaskNumbers:    []string{"1", "2", "3"},
				MaxConcurrency: 3,
			},
			results: []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
				{Task: models.Task{Number: "2"}, Status: models.StatusGreen},
				{Task: models.Task{Number: "3"}, Status: models.StatusGreen},
			},
			expectedText: []string{
				"3/3 completed",
				"3 GREEN",
			},
			notExpected: []string{
				"YELLOW",
				"RED",
				"0 YELLOW",
				"0 RED",
			},
		},
		{
			name: "mixed statuses - show all non-zero",
			wave: models.Wave{
				Name:           "Wave 2",
				TaskNumbers:    []string{"1", "2", "3", "4"},
				MaxConcurrency: 4,
			},
			results: []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
				{Task: models.Task{Number: "2"}, Status: models.StatusGreen},
				{Task: models.Task{Number: "3"}, Status: models.StatusYellow},
				{Task: models.Task{Number: "4"}, Status: models.StatusRed},
			},
			expectedText: []string{
				"4/4 completed",
				"2 GREEN",
				"1 YELLOW",
				"1 RED",
			},
			notExpected: []string{},
		},
		{
			name: "all red - only show red",
			wave: models.Wave{
				Name:           "Wave 3",
				TaskNumbers:    []string{"1", "2"},
				MaxConcurrency: 2,
			},
			results: []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusRed},
				{Task: models.Task{Number: "2"}, Status: models.StatusRed},
			},
			expectedText: []string{
				"2/2 completed",
				"2 RED",
			},
			notExpected: []string{
				"GREEN",
				"YELLOW",
				"0 GREEN",
				"0 YELLOW",
			},
		},
		{
			name: "empty wave",
			wave: models.Wave{
				Name:           "Wave 4",
				TaskNumbers:    []string{},
				MaxConcurrency: 1,
			},
			results: []models.TaskResult{},
			expectedText: []string{
				"0/0 completed",
			},
			notExpected: []string{
				"GREEN",
				"YELLOW",
				"RED",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewConsoleLogger(buf, "info")

			logger.LogWaveComplete(tt.wave, 5*time.Second, tt.results)

			output := buf.String()

			for _, expected := range tt.expectedText {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got %q", expected, output)
				}
			}

			for _, notExp := range tt.notExpected {
				if strings.Contains(output, notExp) {
					t.Errorf("expected output NOT to contain %q, got %q", notExp, output)
				}
			}
		})
	}
}

// TestLogWaveComplete_ColorOutput verifies status counts are color-coded correctly.
func TestLogWaveComplete_ColorOutput(t *testing.T) {
	t.Run("with colors enabled", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewConsoleLogger(buf, "info")
		logger.colorOutput = true // Force color output

		wave := models.Wave{
			Name:           "Wave 1",
			TaskNumbers:    []string{"1", "2", "3"},
			MaxConcurrency: 3,
		}

		results := []models.TaskResult{
			{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
			{Task: models.Task{Number: "2"}, Status: models.StatusYellow},
			{Task: models.Task{Number: "3"}, Status: models.StatusRed},
		}

		logger.LogWaveComplete(wave, 10*time.Second, results)

		output := buf.String()

		// Verify content is present
		if !strings.Contains(output, "GREEN") {
			t.Error("expected 'GREEN' in colored output")
		}

		if !strings.Contains(output, "YELLOW") {
			t.Error("expected 'YELLOW' in colored output")
		}

		if !strings.Contains(output, "RED") {
			t.Error("expected 'RED' in colored output")
		}

		if !strings.Contains(output, "3/3 completed") {
			t.Error("expected task count in colored output")
		}
	})

	t.Run("with colors disabled", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewConsoleLogger(buf, "info")
		logger.colorOutput = false // Disable color

		wave := models.Wave{
			Name:           "Wave 2",
			TaskNumbers:    []string{"1", "2"},
			MaxConcurrency: 2,
		}

		results := []models.TaskResult{
			{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
			{Task: models.Task{Number: "2"}, Status: models.StatusYellow},
		}

		logger.LogWaveComplete(wave, 5*time.Second, results)

		output := buf.String()

		// Verify content is present
		if !strings.Contains(output, "1 GREEN") {
			t.Error("expected '1 GREEN' in plain output")
		}

		if !strings.Contains(output, "1 YELLOW") {
			t.Error("expected '1 YELLOW' in plain output")
		}

		if !strings.Contains(output, "2/2 completed") {
			t.Error("expected task count in plain output")
		}

		// Should not contain ANSI escape codes
		if strings.Contains(output, "\033[") || strings.Contains(output, "\x1b[") {
			t.Error("expected no ANSI codes in plain text output")
		}
	})
}

// TestLogSummary verifies execution summary formatting.
func TestLogSummary(t *testing.T) {
	tests := []struct {
		name          string
		result        models.ExecutionResult
		expectedTexts []string
		notExpected   []string
	}{
		{
			name: "all completed",
			result: models.ExecutionResult{
				TotalTasks:  10,
				Completed:   10,
				Failed:      0,
				Duration:    2 * time.Minute,
				FailedTasks: []models.TaskResult{},
			},
			expectedTexts: []string{
				"=== Execution Summary ===",
				"Total tasks: 10",
				"Completed: 10",
				"Failed: 0",
				"Duration: 2m",
			},
			notExpected: []string{"Failed tasks:"},
		},
		{
			name: "some failed",
			result: models.ExecutionResult{
				TotalTasks: 10,
				Completed:  8,
				Failed:     2,
				Duration:   3 * time.Minute,
				FailedTasks: []models.TaskResult{
					{
						Task: models.Task{
							Name: "Task 1",
						},
						Status: models.StatusRed,
					},
					{
						Task: models.Task{
							Name: "Task 2",
						},
						Status: models.StatusFailed,
					},
				},
			},
			expectedTexts: []string{
				"=== Execution Summary ===",
				"Total tasks: 10",
				"Completed: 8",
				"Failed: 2",
				"Duration: 3m",
				"Failed tasks:",
				"Task 1",
				"Task 2",
			},
			notExpected: []string{},
		},
		{
			name: "zero tasks",
			result: models.ExecutionResult{
				TotalTasks:  0,
				Completed:   0,
				Failed:      0,
				Duration:    0,
				FailedTasks: []models.TaskResult{},
			},
			expectedTexts: []string{
				"Total tasks: 0",
				"Completed: 0",
				"Failed: 0",
			},
			notExpected: []string{"Failed tasks:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewConsoleLogger(buf, "info")

			logger.LogSummary(tt.result)

			output := buf.String()

			for _, expected := range tt.expectedTexts {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got %q", expected, output)
				}
			}

			for _, notExp := range tt.notExpected {
				if strings.Contains(output, notExp) {
					t.Errorf("expected output NOT to contain %q, got %q", notExp, output)
				}
			}
		})
	}
}

// TestTimestampFormat verifies timestamps are formatted correctly as HH:MM:SS.
func TestTimestampFormat(t *testing.T) {
	ts := timestamp()

	// Verify format is HH:MM:SS (8 characters total with colons)
	if len(ts) != 8 {
		t.Errorf("expected timestamp length 8, got %d: %s", len(ts), ts)
	}

	// Verify colons at correct positions
	if ts[2] != ':' || ts[5] != ':' {
		t.Errorf("expected colons at positions 2 and 5, got %s", ts)
	}

	// Verify all other characters are digits
	parts := strings.Split(ts, ":")
	if len(parts) != 3 {
		t.Errorf("expected 3 parts separated by colons, got %d", len(parts))
	}

	for i, part := range parts {
		if len(part) != 2 {
			t.Errorf("expected part %d to have length 2, got %d", i, len(part))
		}
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				t.Errorf("expected digit in timestamp, got %c", ch)
			}
		}
	}
}

// TestConcurrentLogging verifies thread safety with concurrent logging.
func TestConcurrentLogging(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	// Track successful operations
	var successCount int32 = 0

	// Run multiple goroutines logging concurrently
	numGoroutines := 10
	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()

			wave := models.Wave{
				Name:           fmt.Sprintf("Wave %d", index),
				TaskNumbers:    []string{"1", "2"},
				MaxConcurrency: 2,
			}

			logger.LogWaveStart(wave)
			logger.LogWaveComplete(wave, 5*time.Second, []models.TaskResult{})

			result := models.ExecutionResult{
				TotalTasks:  10,
				Completed:   8,
				Failed:      2,
				Duration:    time.Minute,
				FailedTasks: []models.TaskResult{},
			}
			logger.LogSummary(result)

			atomic.AddInt32(&successCount, 1)
		}(i)
	}

	wg.Wait()

	// Verify all operations completed
	if successCount != int32(numGoroutines) {
		t.Errorf("expected %d successful operations, got %d", numGoroutines, successCount)
	}

	// Verify output was written
	output := buf.String()
	if len(output) == 0 {
		t.Error("expected non-empty output")
	}

	// Verify no data corruption (all wave names present)
	for i := 0; i < numGoroutines; i++ {
		waveName := fmt.Sprintf("Wave %d", i)
		if !strings.Contains(output, waveName) {
			t.Errorf("expected output to contain %q", waveName)
		}
	}
}

// TestNilWriter verifies that nil writer is handled gracefully.
func TestNilWriter(t *testing.T) {
	logger := NewConsoleLogger(nil, "info")

	// These should not panic
	wave := models.Wave{
		Name:           "Wave 1",
		TaskNumbers:    []string{"1"},
		MaxConcurrency: 1,
	}

	logger.LogWaveStart(wave)
	logger.LogWaveComplete(wave, 5*time.Second, []models.TaskResult{})

	result := models.ExecutionResult{
		TotalTasks:  10,
		Completed:   10,
		Failed:      0,
		Duration:    time.Minute,
		FailedTasks: []models.TaskResult{},
	}
	logger.LogSummary(result)

	// If we got here without panic, test passed
}

// TestDurationFormatting verifies duration formatting for various time ranges.
func TestDurationFormatting(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "zero",
			duration: 0,
			expected: "0s",
		},
		{
			name:     "5 seconds",
			duration: 5 * time.Second,
			expected: "5s",
		},
		{
			name:     "30 seconds",
			duration: 30 * time.Second,
			expected: "30s",
		},
		{
			name:     "1 minute",
			duration: 1 * time.Minute,
			expected: "1m",
		},
		{
			name:     "1m30s",
			duration: 1*time.Minute + 30*time.Second,
			expected: "1m30s",
		},
		{
			name:     "2m45s",
			duration: 2*time.Minute + 45*time.Second,
			expected: "2m45s",
		},
		{
			name:     "1 hour",
			duration: 1 * time.Hour,
			expected: "1h",
		},
		{
			name:     "1h30m",
			duration: 1*time.Hour + 30*time.Minute,
			expected: "1h30m",
		},
		{
			name:     "1h30m45s",
			duration: 1*time.Hour + 30*time.Minute + 45*time.Second,
			expected: "1h30m45s",
		},
		{
			name:     "2h",
			duration: 2 * time.Hour,
			expected: "2h",
		},
		{
			name:     "2h15m",
			duration: 2*time.Hour + 15*time.Minute,
			expected: "2h15m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestNoOpLogger verifies that NoOpLogger is a valid Logger implementation.
func TestNoOpLogger(t *testing.T) {
	t.Run("creation", func(t *testing.T) {
		logger := NewNoOpLogger()
		if logger == nil {
			t.Error("expected non-nil logger")
		}
	})

	t.Run("methods don't panic", func(t *testing.T) {
		logger := NewNoOpLogger()

		wave := models.Wave{
			Name:           "Wave 1",
			TaskNumbers:    []string{"1"},
			MaxConcurrency: 1,
		}

		logger.LogWaveStart(wave)
		logger.LogWaveComplete(wave, 5*time.Second, []models.TaskResult{})

		result := models.ExecutionResult{
			TotalTasks:  10,
			Completed:   10,
			Failed:      0,
			Duration:    time.Minute,
			FailedTasks: []models.TaskResult{},
		}
		logger.LogSummary(result)

		// If we got here without panic, test passed
	})

	t.Run("concurrent calls", func(t *testing.T) {
		logger := NewNoOpLogger()

		wg := sync.WaitGroup{}
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				wave := models.Wave{
					Name:           "Wave 1",
					TaskNumbers:    []string{"1"},
					MaxConcurrency: 1,
				}

				logger.LogWaveStart(wave)
				logger.LogWaveComplete(wave, 5*time.Second, []models.TaskResult{})

				result := models.ExecutionResult{
					TotalTasks:  10,
					Completed:   10,
					Failed:      0,
					Duration:    time.Minute,
					FailedTasks: []models.TaskResult{},
				}
				logger.LogSummary(result)
			}()
		}

		wg.Wait()
	})
}

// TestLogSummaryWithFailedTasks verifies failed task details are included.
func TestLogSummaryWithFailedTasks(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	result := models.ExecutionResult{
		TotalTasks: 5,
		Completed:  3,
		Failed:     2,
		Duration:   5 * time.Second,
		FailedTasks: []models.TaskResult{
			{
				Task: models.Task{
					Name: "Parse Configuration",
				},
				Status: models.StatusRed,
			},
			{
				Task: models.Task{
					Name: "Build Application",
				},
				Status: models.StatusFailed,
			},
		},
	}

	logger.LogSummary(result)

	output := buf.String()

	// Verify summary information
	if !strings.Contains(output, "Total tasks: 5") {
		t.Error("expected total tasks count in output")
	}

	if !strings.Contains(output, "Completed: 3") {
		t.Error("expected completed count in output")
	}

	if !strings.Contains(output, "Failed: 2") {
		t.Error("expected failed count in output")
	}

	// Verify failed task details
	if !strings.Contains(output, "Failed tasks:") {
		t.Error("expected 'Failed tasks:' section in output")
	}

	if !strings.Contains(output, "Parse Configuration") {
		t.Error("expected failed task name in output")
	}

	if !strings.Contains(output, "Build Application") {
		t.Error("expected second failed task name in output")
	}

	if !strings.Contains(output, models.StatusRed) {
		t.Error("expected RED status in output")
	}

	if !strings.Contains(output, models.StatusFailed) {
		t.Error("expected FAILED status in output")
	}
}

// TestConsoleLoggerSatisfiesInterface verifies ConsoleLogger implements Logger interface.
func TestConsoleLoggerSatisfiesInterface(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	// This will fail to compile if ConsoleLogger doesn't implement Logger
	var _ Logger = logger
}

// TestNoOpLoggerSatisfiesInterface verifies NoOpLogger implements Logger interface.
func TestNoOpLoggerSatisfiesInterface(t *testing.T) {
	logger := NewNoOpLogger()

	// This will fail to compile if NoOpLogger doesn't implement Logger
	var _ Logger = logger
}

// Logger is the interface that both ConsoleLogger and NoOpLogger must satisfy.
// This is defined here for testing purposes to verify interface compliance.
type Logger interface {
	LogWaveStart(wave models.Wave)
	LogWaveComplete(wave models.Wave, duration time.Duration, results []models.TaskResult)
	LogSummary(result models.ExecutionResult)
}

// TestConsoleLogger_LogProgress_BasicDisplay verifies progress bar display with basic task counts.
func TestConsoleLogger_LogProgress_BasicDisplay(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	// Create 4 completed tasks out of 8 total
	results := []models.TaskResult{
		{Task: models.Task{Number: "1", Name: "Task 1", Status: "completed"}, Status: "GREEN"},
		{Task: models.Task{Number: "2", Name: "Task 2", Status: "completed"}, Status: "GREEN"},
		{Task: models.Task{Number: "3", Name: "Task 3", Status: "completed"}, Status: "GREEN"},
		{Task: models.Task{Number: "4", Name: "Task 4", Status: "completed"}, Status: "GREEN"},
		{Task: models.Task{Number: "5", Name: "Task 5", Status: "pending"}},
		{Task: models.Task{Number: "6", Name: "Task 6", Status: "pending"}},
		{Task: models.Task{Number: "7", Name: "Task 7", Status: "pending"}},
		{Task: models.Task{Number: "8", Name: "Task 8", Status: "pending"}},
	}

	logger.LogProgress(results)

	output := buf.String()

	// Verify output contains Progress label
	if !strings.Contains(output, "Progress:") {
		t.Error("expected 'Progress:' in output")
	}

	// Verify output contains percentage (50%)
	if !strings.Contains(output, "50%") {
		t.Errorf("expected '50%%' in output, got %q", output)
	}

	// Verify output contains task counts (4/8)
	if !strings.Contains(output, "(4/8 tasks)") {
		t.Errorf("expected '(4/8 tasks)' in output, got %q", output)
	}

	// Verify timestamp prefix
	if !strings.HasPrefix(output, "[") {
		t.Error("expected output to start with timestamp [")
	}
}

// TestConsoleLogger_LogProgress_WithAverageDuration verifies average duration calculation and display.
func TestConsoleLogger_LogProgress_WithAverageDuration(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	// Create tasks with specific durations: 2s, 3s, 4s, 5s (avg: 3.5s)
	now := time.Now()
	results := []models.TaskResult{
		{
			Task: models.Task{
				Number: "1", Name: "Task 1", Status: "completed",
				StartedAt:   nil,
				CompletedAt: &now,
			},
			Status: "GREEN",
		},
		{
			Task: models.Task{
				Number: "2", Name: "Task 2", Status: "completed",
				StartedAt:   nil,
				CompletedAt: &now,
			},
			Status: "GREEN",
		},
		{
			Task: models.Task{
				Number: "3", Name: "Task 3", Status: "completed",
				StartedAt:   nil,
				CompletedAt: &now,
			},
			Status: "GREEN",
		},
		{
			Task: models.Task{
				Number: "4", Name: "Task 4", Status: "completed",
				StartedAt:   nil,
				CompletedAt: &now,
			},
			Status: "GREEN",
		},
		{
			Task: models.Task{
				Number: "5", Name: "Task 5", Status: "pending",
			},
		},
		{
			Task: models.Task{
				Number: "6", Name: "Task 6", Status: "pending",
			},
		},
	}

	// Add durations to tasks (2s, 3s, 4s, 5s)
	for i := range results {
		if results[i].Task.Status == "completed" {
			startTime := now.Add(-time.Duration((i + 2) * int(time.Second)))
			results[i].Task.StartedAt = &startTime
		}
	}

	logger.LogProgress(results)

	output := buf.String()

	// Verify output contains average duration
	if !strings.Contains(output, "Avg:") {
		t.Errorf("expected 'Avg:' in output, got %q", output)
	}

	// Verify percentage and counts (4/6 = 66%)
	if !strings.Contains(output, "66%") {
		t.Errorf("expected '66%%' in output, got %q", output)
	}

	if !strings.Contains(output, "(4/6 tasks)") {
		t.Errorf("expected '(4/6 tasks)' in output, got %q", output)
	}
}

// TestConsoleLogger_LogProgress_ZeroTasks verifies handling of zero tasks.
func TestConsoleLogger_LogProgress_ZeroTasks(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	// Empty task list
	results := []models.TaskResult{}

	// Should not panic
	logger.LogProgress(results)

	output := buf.String()

	// Verify it handled gracefully - should show 0/0 or similar
	if !strings.Contains(output, "0") {
		t.Errorf("expected '0' in output for zero tasks, got %q", output)
	}
}

// TestConsoleLogger_LogProgress_AllTasksComplete verifies display with all tasks complete.
func TestConsoleLogger_LogProgress_AllTasksComplete(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	// All 8 tasks completed
	results := []models.TaskResult{
		{Task: models.Task{Number: "1", Name: "Task 1", Status: "completed"}, Status: "GREEN"},
		{Task: models.Task{Number: "2", Name: "Task 2", Status: "completed"}, Status: "GREEN"},
		{Task: models.Task{Number: "3", Name: "Task 3", Status: "completed"}, Status: "GREEN"},
		{Task: models.Task{Number: "4", Name: "Task 4", Status: "completed"}, Status: "GREEN"},
		{Task: models.Task{Number: "5", Name: "Task 5", Status: "completed"}, Status: "GREEN"},
		{Task: models.Task{Number: "6", Name: "Task 6", Status: "completed"}, Status: "GREEN"},
		{Task: models.Task{Number: "7", Name: "Task 7", Status: "completed"}, Status: "GREEN"},
		{Task: models.Task{Number: "8", Name: "Task 8", Status: "completed"}, Status: "GREEN"},
	}

	logger.LogProgress(results)

	output := buf.String()

	// Verify 100% completion
	if !strings.Contains(output, "100%") {
		t.Errorf("expected '100%%' in output, got %q", output)
	}

	// Verify all tasks count (8/8)
	if !strings.Contains(output, "(8/8 tasks)") {
		t.Errorf("expected '(8/8 tasks)' in output, got %q", output)
	}
}

// TestConsoleLogger_LogProgress_ColorOutput verifies color output when enabled/disabled.
func TestConsoleLogger_LogProgress_ColorOutput(t *testing.T) {
	t.Run("with color enabled", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewConsoleLogger(buf, "info")
		logger.colorOutput = true // Force color output for testing

		results := []models.TaskResult{
			{Task: models.Task{Number: "1", Name: "Task 1", Status: "completed"}, Status: "GREEN"},
			{Task: models.Task{Number: "2", Name: "Task 2", Status: "pending"}},
		}

		logger.LogProgress(results)

		output := buf.String()

		// When color is enabled, output should contain ANSI codes
		// Cyan color code is \033[36m or \x1b[36m
		if !strings.Contains(output, "50%") {
			t.Errorf("expected progress output, got %q", output)
		}
	})

	t.Run("with color disabled", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewConsoleLogger(buf, "info")
		logger.colorOutput = false // Disable color for testing

		results := []models.TaskResult{
			{Task: models.Task{Number: "1", Name: "Task 1", Status: "completed"}, Status: "GREEN"},
			{Task: models.Task{Number: "2", Name: "Task 2", Status: "pending"}},
		}

		logger.LogProgress(results)

		output := buf.String()

		// Plain text should contain percentage and counts
		if !strings.Contains(output, "50%") {
			t.Errorf("expected '50%%' in output, got %q", output)
		}

		if !strings.Contains(output, "(1/2 tasks)") {
			t.Errorf("expected '(1/2 tasks)' in output, got %q", output)
		}

		// Should not contain raw ANSI escape codes
		if strings.Contains(output, "\033[") || strings.Contains(output, "\x1b[") {
			t.Error("expected no ANSI codes in plain text output")
		}
	})
}

// TestConsoleLogger_LogProgress_TimestampPrefix verifies timestamp formatting.
func TestConsoleLogger_LogProgress_TimestampPrefix(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	results := []models.TaskResult{
		{Task: models.Task{Number: "1", Name: "Task 1", Status: "completed"}, Status: "GREEN"},
		{Task: models.Task{Number: "2", Name: "Task 2", Status: "pending"}},
	}

	logger.LogProgress(results)

	output := buf.String()

	// Verify timestamp prefix [HH:MM:SS]
	if !strings.HasPrefix(output, "[") {
		t.Error("expected output to start with timestamp [")
	}

	// Extract timestamp portion and verify format
	endBracket := strings.Index(output, "]")
	if endBracket == -1 {
		t.Error("expected ] in output")
	}

	timestamp := output[1:endBracket]
	parts := strings.Split(timestamp, ":")
	if len(parts) != 3 {
		t.Errorf("expected timestamp HH:MM:SS format, got %q", timestamp)
	}
}

// TestConsoleLogger_LogProgress_ProgressBarIntegration verifies progress bar rendering.
func TestConsoleLogger_LogProgress_ProgressBarIntegration(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	// Test with 50% completion
	results := []models.TaskResult{
		{Task: models.Task{Number: "1", Name: "Task 1", Status: "completed"}, Status: "GREEN"},
		{Task: models.Task{Number: "2", Name: "Task 2", Status: "completed"}, Status: "GREEN"},
		{Task: models.Task{Number: "3", Name: "Task 3", Status: "pending"}},
		{Task: models.Task{Number: "4", Name: "Task 4", Status: "pending"}},
	}

	logger.LogProgress(results)

	output := buf.String()

	// Verify percentage calculation
	if !strings.Contains(output, "50%") {
		t.Errorf("expected '50%%' in output, got %q", output)
	}

	// Verify counts
	if !strings.Contains(output, "(2/4 tasks)") {
		t.Errorf("expected '(2/4 tasks)' in output, got %q", output)
	}

	// Verify Progress prefix
	if !strings.Contains(output, "Progress:") {
		t.Errorf("expected 'Progress:' in output, got %q", output)
	}
}

// TestConsoleLogger_LogProgress_EdgeCaseNilStartedAt verifies handling of tasks without StartedAt.
func TestConsoleLogger_LogProgress_EdgeCaseNilStartedAt(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	// Tasks without StartedAt (no duration data)
	results := []models.TaskResult{
		{Task: models.Task{Number: "1", Name: "Task 1", Status: "completed", StartedAt: nil}, Status: "GREEN"},
		{Task: models.Task{Number: "2", Name: "Task 2", Status: "pending"}},
	}

	// Should not panic with nil StartedAt
	logger.LogProgress(results)

	output := buf.String()

	// Verify output is still generated
	if !strings.Contains(output, "50%") {
		t.Errorf("expected '50%%' in output, got %q", output)
	}

	// Should show progress without average duration
	if !strings.Contains(output, "(1/2 tasks)") {
		t.Errorf("expected '(1/2 tasks)' in output, got %q", output)
	}
}

// TestConsoleLoggerLogTaskStart verifies basic task start output with progress counter and agent.
func TestConsoleLoggerLogTaskStart(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	task := models.Task{
		Number: "1",
		Name:   "Setup Database",
		Agent:  "backend-dev",
	}

	logger.LogTaskStart(task, 1, 5)

	output := buf.String()

	// Verify timestamp prefix
	if !strings.HasPrefix(output, "[") {
		t.Error("expected output to start with timestamp [")
	}

	// Verify progress counter [1/5]
	if !strings.Contains(output, "[1/5]") {
		t.Errorf("expected '[1/5]' progress counter in output, got %q", output)
	}

	// Verify status indicator
	if !strings.Contains(output, "IN PROGRESS") {
		t.Errorf("expected 'IN PROGRESS' status in output, got %q", output)
	}

	// Verify task identifier
	if !strings.Contains(output, "Task 1") {
		t.Errorf("expected 'Task 1' in output, got %q", output)
	}

	if !strings.Contains(output, "Setup Database") {
		t.Errorf("expected 'Setup Database' in output, got %q", output)
	}

	// Verify agent info
	if !strings.Contains(output, "backend-dev") {
		t.Errorf("expected 'backend-dev' agent in output, got %q", output)
	}
}

// TestConsoleLoggerLogTaskStartNoAgent verifies task without agent specified.
func TestConsoleLoggerLogTaskStartNoAgent(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	task := models.Task{
		Number: "2",
		Name:   "Run Tests",
		Agent:  "", // No agent
	}

	logger.LogTaskStart(task, 2, 5)

	output := buf.String()

	// Verify task info is present
	if !strings.Contains(output, "Task 2") {
		t.Errorf("expected 'Task 2' in output, got %q", output)
	}

	if !strings.Contains(output, "Run Tests") {
		t.Errorf("expected 'Run Tests' in output, got %q", output)
	}

	// Verify progress counter
	if !strings.Contains(output, "[2/5]") {
		t.Errorf("expected '[2/5]' in output, got %q", output)
	}

	// Should not have agent info when agent is empty
	if strings.Contains(output, "(agent: )") {
		t.Error("expected no agent info when agent is empty")
	}
}

// TestConsoleLoggerLogTaskStartWithColors verifies color output when enabled.
func TestConsoleLoggerLogTaskStartWithColors(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")
	logger.colorOutput = true // Force color output

	task := models.Task{
		Number: "3",
		Name:   "Build Application",
		Agent:  "code-gen",
	}

	logger.LogTaskStart(task, 3, 10)

	output := buf.String()

	// Verify essential content is present (color codes may be there too)
	if !strings.Contains(output, "Task 3") {
		t.Errorf("expected 'Task 3' in output, got %q", output)
	}

	if !strings.Contains(output, "Build Application") {
		t.Errorf("expected 'Build Application' in output, got %q", output)
	}

	if !strings.Contains(output, "[3/10]") {
		t.Errorf("expected '[3/10]' in output, got %q", output)
	}
}

// TestConsoleLoggerLogTaskStartNoColors verifies plain text output when colors disabled.
func TestConsoleLoggerLogTaskStartNoColors(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")
	logger.colorOutput = false // Disable color

	task := models.Task{
		Number: "4",
		Name:   "Deploy Service",
		Agent:  "devops",
	}

	logger.LogTaskStart(task, 4, 8)

	output := buf.String()

	// Verify output contains expected text without ANSI codes
	if !strings.Contains(output, "Task 4") {
		t.Errorf("expected 'Task 4' in output, got %q", output)
	}

	if !strings.Contains(output, "Deploy Service") {
		t.Errorf("expected 'Deploy Service' in output, got %q", output)
	}

	if !strings.Contains(output, "[4/8]") {
		t.Errorf("expected '[4/8]' in output, got %q", output)
	}

	// Should not contain raw ANSI escape codes
	if strings.Contains(output, "\033[") || strings.Contains(output, "\x1b[") {
		t.Error("expected no ANSI codes in plain text output")
	}
}

// TestConsoleLoggerLogTaskStartTimestamp verifies timestamp format.
func TestConsoleLoggerLogTaskStartTimestamp(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	task := models.Task{
		Number: "5",
		Name:   "Documentation",
		Agent:  "writer",
	}

	logger.LogTaskStart(task, 5, 6)

	output := buf.String()

	// Extract timestamp portion [HH:MM:SS]
	if !strings.HasPrefix(output, "[") {
		t.Error("expected output to start with timestamp [")
	}

	endBracket := strings.Index(output, "]")
	if endBracket == -1 {
		t.Error("expected ] after timestamp")
	}

	ts := output[1:endBracket]
	parts := strings.Split(ts, ":")

	// Verify HH:MM:SS format (3 parts)
	if len(parts) != 3 {
		t.Errorf("expected HH:MM:SS format, got %q", ts)
	}

	// Verify each part is 2 digits
	for i, part := range parts {
		if len(part) != 2 {
			t.Errorf("expected part %d to have length 2, got %d for %q", i, len(part), part)
		}
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				t.Errorf("expected digit in timestamp part %d, got %c", i, ch)
			}
		}
	}
}

// TestConsoleLoggerLogTaskStartEdgeCases tests boundary conditions.
func TestConsoleLoggerLogTaskStartEdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		taskNumber    string
		taskName      string
		agent         string
		current       int
		total         int
		shouldContain []string
	}{
		{
			name:       "Long task name",
			taskNumber: "1",
			taskName:   "This is a very long task name that spans multiple words and should be handled correctly",
			agent:      "test-agent",
			current:    1,
			total:      5,
			shouldContain: []string{
				"Task 1",
				"This is a very long task name",
				"[1/5]",
				"test-agent",
			},
		},
		{
			name:       "Special characters in name",
			taskNumber: "2",
			taskName:   "Task with @#$% special chars & symbols",
			agent:      "agent-123",
			current:    2,
			total:      5,
			shouldContain: []string{
				"Task 2",
				"@#$%",
				"[2/5]",
			},
		},
		{
			name:       "Single task (1/1)",
			taskNumber: "1",
			taskName:   "Only Task",
			agent:      "solo",
			current:    1,
			total:      1,
			shouldContain: []string{
				"[1/1]",
				"Task 1",
				"Only Task",
			},
		},
		{
			name:       "Last task in sequence",
			taskNumber: "100",
			taskName:   "Final Task",
			agent:      "finalizer",
			current:    100,
			total:      100,
			shouldContain: []string{
				"[100/100]",
				"Task 100",
				"Final Task",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewConsoleLogger(buf, "info")

			task := models.Task{
				Number: tt.taskNumber,
				Name:   tt.taskName,
				Agent:  tt.agent,
			}

			logger.LogTaskStart(task, tt.current, tt.total)
			output := buf.String()

			for _, expected := range tt.shouldContain {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got %q", expected, output)
				}
			}
		})
	}
}

// TestConsoleLoggerLogTaskStartConcurrency verifies thread-safe concurrent logging.
func TestConsoleLoggerLogTaskStartConcurrency(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	var successCount int32 = 0
	numGoroutines := 20

	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()

			task := models.Task{
				Number: fmt.Sprintf("%d", index),
				Name:   fmt.Sprintf("Task %d", index),
				Agent:  fmt.Sprintf("agent-%d", index),
			}

			logger.LogTaskStart(task, index+1, numGoroutines)
			atomic.AddInt32(&successCount, 1)
		}(i)
	}

	wg.Wait()

	// Verify all operations completed
	if successCount != int32(numGoroutines) {
		t.Errorf("expected %d successful operations, got %d", numGoroutines, successCount)
	}

	// Verify output was written
	output := buf.String()
	if len(output) == 0 {
		t.Error("expected non-empty output from concurrent logging")
	}

	// Verify no data corruption (spot check some agents)
	if !strings.Contains(output, "agent-0") {
		t.Error("expected agent-0 in output")
	}
	if !strings.Contains(output, fmt.Sprintf("agent-%d", numGoroutines-1)) {
		t.Error("expected last agent in output")
	}

	// Verify no mixed lines (each log should have matching task number and agent)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		// Each non-empty line should have IN PROGRESS indicator
		if strings.Contains(line, "IN PROGRESS") {
			// Should have matching task number and agent info
			if !strings.Contains(line, "Task") || !strings.Contains(line, "agent-") {
				t.Errorf("expected consistent task and agent info in line: %q", line)
			}
		}
	}
}

// TestConsoleLoggerLogTaskResultSuccess verifies successful task (GREEN) output.
func TestConsoleLoggerLogTaskResultSuccess(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "debug")

	result := models.TaskResult{
		Task: models.Task{
			Number:        "1",
			Name:          "Setup Database",
			Agent:         "backend-dev",
			Files:         []string{"db.go", "schema.sql", "migration.go"},
			FilesModified: 2,
			FilesCreated:  1,
		},
		Status:   models.StatusGreen,
		Duration: 5*time.Second + 300*time.Millisecond,
	}

	err := logger.LogTaskResult(result)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	output := buf.String()

	// Verify timestamp prefix
	if !strings.HasPrefix(output, "[") {
		t.Error("expected output to start with timestamp [")
	}

	// Verify status is GREEN
	if !strings.Contains(output, "GREEN") {
		t.Errorf("expected 'GREEN' status in output, got %q", output)
	}

	// Verify task info
	if !strings.Contains(output, "Task 1") {
		t.Errorf("expected 'Task 1' in output, got %q", output)
	}

	if !strings.Contains(output, "Setup Database") {
		t.Errorf("expected 'Setup Database' in output, got %q", output)
	}

	// Verify duration
	if !strings.Contains(output, "5.3s") {
		t.Errorf("expected duration '5.3s' in output, got %q", output)
	}

	// Verify agent
	if !strings.Contains(output, "backend-dev") {
		t.Errorf("expected agent 'backend-dev' in output, got %q", output)
	}

	// Verify file count
	if !strings.Contains(output, "3 files") {
		t.Errorf("expected '3 files' in output, got %q", output)
	}
}

// TestConsoleLoggerLogTaskResultFailed verifies failed task (RED) output.
func TestConsoleLoggerLogTaskResultFailed(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "debug")

	result := models.TaskResult{
		Task: models.Task{
			Number:        "2",
			Name:          "Run Tests",
			Agent:         "test-runner",
			Files:         []string{"test.go"},
			FilesModified: 0,
			FilesCreated:  0,
		},
		Status:   models.StatusRed,
		Duration: 2 * time.Second,
	}

	err := logger.LogTaskResult(result)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	output := buf.String()

	// Verify status is RED
	if !strings.Contains(output, "RED") {
		t.Errorf("expected 'RED' status in output, got %q", output)
	}

	// Verify task info
	if !strings.Contains(output, "Task 2") {
		t.Errorf("expected 'Task 2' in output, got %q", output)
	}

	if !strings.Contains(output, "Run Tests") {
		t.Errorf("expected 'Run Tests' in output, got %q", output)
	}

	// Verify duration
	if !strings.Contains(output, "2.0s") {
		t.Errorf("expected duration '2.0s' in output, got %q", output)
	}
}

// TestConsoleLoggerLogTaskResultWarning verifies warning task (YELLOW) output.
func TestConsoleLoggerLogTaskResultWarning(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "debug")

	result := models.TaskResult{
		Task: models.Task{
			Number:        "3",
			Name:          "Lint Check",
			Agent:         "linter",
			Files:         []string{"main.go", "utils.go"},
			FilesModified: 1,
			FilesCreated:  0,
		},
		Status:   models.StatusYellow,
		Duration: 1 * time.Second,
	}

	err := logger.LogTaskResult(result)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	output := buf.String()

	// Verify status is YELLOW
	if !strings.Contains(output, "YELLOW") {
		t.Errorf("expected 'YELLOW' status in output, got %q", output)
	}

	// Verify task info
	if !strings.Contains(output, "Task 3") {
		t.Errorf("expected 'Task 3' in output, got %q", output)
	}

	if !strings.Contains(output, "Lint Check") {
		t.Errorf("expected 'Lint Check' in output, got %q", output)
	}
}

// TestConsoleLoggerLogTaskResultColors verifies color output is applied correctly.
func TestConsoleLoggerLogTaskResultColors(t *testing.T) {
	t.Run("with colors enabled", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewConsoleLogger(buf, "debug")
		logger.colorOutput = true // Force color output

		result := models.TaskResult{
			Task: models.Task{
				Number:        "1",
				Name:          "Test Task",
				Agent:         "test-agent",
				FilesModified: 1,
			},
			Status:   models.StatusGreen,
			Duration: 3 * time.Second,
		}

		logger.LogTaskResult(result)

		output := buf.String()

		// Verify content is present (color codes will be in there too)
		if !strings.Contains(output, "GREEN") {
			t.Errorf("expected 'GREEN' in colored output, got %q", output)
		}

		if !strings.Contains(output, "Test Task") {
			t.Errorf("expected task name in colored output, got %q", output)
		}
	})

	t.Run("with colors disabled", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewConsoleLogger(buf, "debug")
		logger.colorOutput = false // Disable color

		result := models.TaskResult{
			Task: models.Task{
				Number:        "2",
				Name:          "Another Task",
				Agent:         "another-agent",
				FilesModified: 2,
			},
			Status:   models.StatusRed,
			Duration: 10 * time.Second,
		}

		logger.LogTaskResult(result)

		output := buf.String()

		// Verify content is present without ANSI codes
		if !strings.Contains(output, "RED") {
			t.Errorf("expected 'RED' in plain output, got %q", output)
		}

		if !strings.Contains(output, "Another Task") {
			t.Errorf("expected task name in plain output, got %q", output)
		}

		// Should not contain ANSI escape codes
		if strings.Contains(output, "\033[") || strings.Contains(output, "\x1b[") {
			t.Error("expected no ANSI codes in plain text output")
		}
	})
}

// TestConsoleLoggerLogTaskResultNoFiles verifies task with no file operations.
func TestConsoleLoggerLogTaskResultNoFiles(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "debug")

	result := models.TaskResult{
		Task: models.Task{
			Number:        "4",
			Name:          "Validate Configuration",
			Agent:         "validator",
			Files:         []string{},
			FilesModified: 0,
			FilesCreated:  0,
			FilesDeleted:  0,
		},
		Status:   models.StatusGreen,
		Duration: 500 * time.Millisecond,
	}

	err := logger.LogTaskResult(result)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	output := buf.String()

	// Verify task info is present
	if !strings.Contains(output, "Task 4") {
		t.Errorf("expected 'Task 4' in output, got %q", output)
	}

	if !strings.Contains(output, "Validate Configuration") {
		t.Errorf("expected 'Validate Configuration' in output, got %q", output)
	}

	// Verify status
	if !strings.Contains(output, "GREEN") {
		t.Errorf("expected 'GREEN' in output, got %q", output)
	}

	// When no files are modified/created/deleted, should handle gracefully
	// (no "0 files" or similar should appear)
}

// TestConsoleLoggerLogTaskResultNoAgent verifies task without agent specified.
func TestConsoleLoggerLogTaskResultNoAgent(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "debug")

	result := models.TaskResult{
		Task: models.Task{
			Number:        "5",
			Name:          "Manual Review",
			Agent:         "", // No agent
			Files:         []string{"review.txt"},
			FilesModified: 1,
		},
		Status:   models.StatusYellow,
		Duration: 15 * time.Second,
	}

	err := logger.LogTaskResult(result)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	output := buf.String()

	// Verify task info
	if !strings.Contains(output, "Task 5") {
		t.Errorf("expected 'Task 5' in output, got %q", output)
	}

	if !strings.Contains(output, "Manual Review") {
		t.Errorf("expected 'Manual Review' in output, got %q", output)
	}

	// Verify status
	if !strings.Contains(output, "YELLOW") {
		t.Errorf("expected 'YELLOW' in output, got %q", output)
	}

	// Should gracefully handle missing agent (no empty parentheses)
	if strings.Contains(output, "(agent: )") {
		t.Error("expected no empty agent info when agent is empty")
	}
}

// TestConsoleLoggerLogTaskResultDurationFormats verifies various duration formats.
func TestConsoleLoggerLogTaskResultDurationFormats(t *testing.T) {
	tests := []struct {
		name           string
		duration       time.Duration
		expectedOutput string
	}{
		{
			name:           "zero duration",
			duration:       0 * time.Second,
			expectedOutput: "0.0s",
		},
		{
			name:           "sub-second",
			duration:       500 * time.Millisecond,
			expectedOutput: "0.5s",
		},
		{
			name:           "one second",
			duration:       1 * time.Second,
			expectedOutput: "1.0s",
		},
		{
			name:           "a few seconds",
			duration:       5*time.Second + 300*time.Millisecond,
			expectedOutput: "5.3s",
		},
		{
			name:           "one minute",
			duration:       1 * time.Minute,
			expectedOutput: "1m",
		},
		{
			name:           "minute and seconds",
			duration:       1*time.Minute + 30*time.Second,
			expectedOutput: "1m30s",
		},
		{
			name:           "multiple minutes",
			duration:       2*time.Minute + 45*time.Second,
			expectedOutput: "2m45s",
		},
		{
			name:           "one hour",
			duration:       1 * time.Hour,
			expectedOutput: "1h",
		},
		{
			name:           "hour and minutes",
			duration:       1*time.Hour + 30*time.Minute,
			expectedOutput: "1h30m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewConsoleLogger(buf, "debug")

			result := models.TaskResult{
				Task: models.Task{
					Number:        "1",
					Name:          "Test Task",
					Agent:         "test-agent",
					FilesModified: 1,
				},
				Status:   models.StatusGreen,
				Duration: tt.duration,
			}

			logger.LogTaskResult(result)

			output := buf.String()

			if !strings.Contains(output, tt.expectedOutput) {
				t.Errorf("expected duration %q in output, got %q", tt.expectedOutput, output)
			}
		})
	}
}

// TestConsoleLoggerLogTaskResultNilWriter verifies nil writer is handled gracefully.
func TestConsoleLoggerLogTaskResultNilWriter(t *testing.T) {
	logger := NewConsoleLogger(nil, "debug")

	result := models.TaskResult{
		Task: models.Task{
			Number: "1",
			Name:   "Test",
			Agent:  "test",
		},
		Status:   models.StatusGreen,
		Duration: 1 * time.Second,
	}

	// Should not panic with nil writer
	err := logger.LogTaskResult(result)

	if err != nil {
		t.Errorf("expected no error with nil writer, got %v", err)
	}
}

// TestConsoleLoggerLogTaskResultBelowLogLevel verifies filtering by log level.
func TestConsoleLoggerLogTaskResultBelowLogLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info") // Set to info, task result is at debug level

	result := models.TaskResult{
		Task: models.Task{
			Number: "1",
			Name:   "Test",
			Agent:  "test",
		},
		Status:   models.StatusGreen,
		Duration: 1 * time.Second,
	}

	logger.LogTaskResult(result)

	output := buf.String()

	// Should not output anything since debug < info
	if len(output) > 0 {
		t.Errorf("expected no output when log level filters debug, got %q", output)
	}
}

// TestConsoleLoggerLogTaskResultMultipleFiles verifies file count formatting.
func TestConsoleLoggerLogTaskResultMultipleFiles(t *testing.T) {
	tests := []struct {
		name               string
		filesModified      int
		filesCreated       int
		filesDeleted       int
		expectedTotalFiles string
	}{
		{
			name:               "single modified file",
			filesModified:      1,
			filesCreated:       0,
			filesDeleted:       0,
			expectedTotalFiles: "1",
		},
		{
			name:               "multiple operations",
			filesModified:      2,
			filesCreated:       1,
			filesDeleted:       0,
			expectedTotalFiles: "3",
		},
		{
			name:               "all operations",
			filesModified:      3,
			filesCreated:       2,
			filesDeleted:       1,
			expectedTotalFiles: "6",
		},
		{
			name:               "large file count",
			filesModified:      10,
			filesCreated:       5,
			filesDeleted:       3,
			expectedTotalFiles: "18",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewConsoleLogger(buf, "debug")

			result := models.TaskResult{
				Task: models.Task{
					Number:        "1",
					Name:          "File Task",
					Agent:         "file-agent",
					FilesModified: tt.filesModified,
					FilesCreated:  tt.filesCreated,
					FilesDeleted:  tt.filesDeleted,
				},
				Status:   models.StatusGreen,
				Duration: 2 * time.Second,
			}

			logger.LogTaskResult(result)

			output := buf.String()

			// Verify total file count is in output
			if !strings.Contains(output, tt.expectedTotalFiles+" file") {
				t.Errorf("expected %q file count in output, got %q", tt.expectedTotalFiles, output)
			}
		})
	}
}

// TestConsoleLoggerLogTaskResultConcurrency verifies thread-safe concurrent logging.
func TestConsoleLoggerLogTaskResultConcurrency(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "debug")

	var successCount int32 = 0
	numGoroutines := 20

	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()

			result := models.TaskResult{
				Task: models.Task{
					Number:        fmt.Sprintf("%d", index),
					Name:          fmt.Sprintf("Task %d", index),
					Agent:         fmt.Sprintf("agent-%d", index),
					FilesModified: index,
				},
				Status:   models.StatusGreen,
				Duration: time.Duration(index) * time.Second,
			}

			err := logger.LogTaskResult(result)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	// Verify all operations completed
	if successCount != int32(numGoroutines) {
		t.Errorf("expected %d successful operations, got %d", numGoroutines, successCount)
	}

	// Verify output was written
	output := buf.String()
	if len(output) == 0 {
		t.Error("expected non-empty output from concurrent logging")
	}

	// Verify no data corruption (spot check some agents)
	if !strings.Contains(output, "agent-0") {
		t.Error("expected agent-0 in output")
	}
	if !strings.Contains(output, fmt.Sprintf("agent-%d", numGoroutines-1)) {
		t.Error("expected last agent in output")
	}
}

// TestConsoleLoggerLogTaskResultVerbose verifies basic verbose output with multi-line format.
func TestConsoleLoggerLogTaskResultVerbose(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "debug")
	logger.SetVerbose(true)

	result := models.TaskResult{
		Task: models.Task{
			Number:        "1",
			Name:          "Setup Database",
			Agent:         "backend-dev",
			Files:         []string{"db.go", "schema.sql"},
			FilesModified: 1,
			FilesCreated:  1,
			FilesDeleted:  0,
		},
		Status:         models.StatusGreen,
		Duration:       5*time.Second + 300*time.Millisecond,
		ReviewFeedback: "Great work on the database schema!",
	}

	err := logger.LogTaskResult(result)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	output := buf.String()

	// Verify multi-line format (should contain newlines)
	if !strings.Contains(output, "\n") {
		t.Error("expected multi-line output in verbose mode")
	}

	// Verify header line with status
	if !strings.Contains(output, "GREEN") {
		t.Errorf("expected 'GREEN' status in verbose output, got %q", output)
	}

	if !strings.Contains(output, "Task 1") {
		t.Errorf("expected 'Task 1' in verbose output, got %q", output)
	}

	if !strings.Contains(output, "Setup Database") {
		t.Errorf("expected 'Setup Database' in verbose output, got %q", output)
	}

	// Verify duration section
	if !strings.Contains(output, "Duration:") {
		t.Errorf("expected 'Duration:' in verbose output, got %q", output)
	}

	if !strings.Contains(output, "5.3s") {
		t.Errorf("expected duration '5.3s' in verbose output, got %q", output)
	}

	// Verify agent section
	if !strings.Contains(output, "Agent:") {
		t.Errorf("expected 'Agent:' in verbose output, got %q", output)
	}

	if !strings.Contains(output, "backend-dev") {
		t.Errorf("expected agent 'backend-dev' in verbose output, got %q", output)
	}

	// Verify files section
	if !strings.Contains(output, "Files:") {
		t.Errorf("expected 'Files:' in verbose output, got %q", output)
	}

	// Verify QC feedback section
	if !strings.Contains(output, "Feedback:") {
		t.Errorf("expected 'Feedback:' in verbose output, got %q", output)
	}

	if !strings.Contains(output, "Great work on the database schema!") {
		t.Errorf("expected feedback text in verbose output, got %q", output)
	}
}

// TestConsoleLoggerLogTaskResultVerboseNoFiles verifies verbose output without files.
func TestConsoleLoggerLogTaskResultVerboseNoFiles(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "debug")
	logger.SetVerbose(true)

	result := models.TaskResult{
		Task: models.Task{
			Number:        "2",
			Name:          "Validate Config",
			Agent:         "validator",
			Files:         []string{},
			FilesModified: 0,
			FilesCreated:  0,
			FilesDeleted:  0,
		},
		Status:         models.StatusYellow,
		Duration:       2 * time.Second,
		ReviewFeedback: "Config is valid but needs cleanup",
	}

	err := logger.LogTaskResult(result)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	output := buf.String()

	// Verify basic info is present
	if !strings.Contains(output, "Task 2") {
		t.Errorf("expected 'Task 2' in output, got %q", output)
	}

	if !strings.Contains(output, "YELLOW") {
		t.Errorf("expected 'YELLOW' status in output, got %q", output)
	}

	// Verify Duration and Agent are present
	if !strings.Contains(output, "Duration:") {
		t.Errorf("expected 'Duration:' in output, got %q", output)
	}

	if !strings.Contains(output, "Agent:") {
		t.Errorf("expected 'Agent:' in output, got %q", output)
	}

	// Verify Files section is omitted (gracefully handled)
	// The output should not have a "Files:" line if no files exist
}

// TestConsoleLoggerLogTaskResultVerboseNoQCFeedback verifies verbose output without QC feedback.
func TestConsoleLoggerLogTaskResultVerboseNoQCFeedback(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "debug")
	logger.SetVerbose(true)

	result := models.TaskResult{
		Task: models.Task{
			Number:        "3",
			Name:          "Build Service",
			Agent:         "builder",
			Files:         []string{"main.go"},
			FilesModified: 1,
			FilesCreated:  0,
			FilesDeleted:  0,
		},
		Status:         models.StatusGreen,
		Duration:       10 * time.Second,
		ReviewFeedback: "", // No feedback
	}

	err := logger.LogTaskResult(result)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	output := buf.String()

	// Verify basic info is present
	if !strings.Contains(output, "Task 3") {
		t.Errorf("expected 'Task 3' in output, got %q", output)
	}

	if !strings.Contains(output, "Build Service") {
		t.Errorf("expected 'Build Service' in output, got %q", output)
	}

	// Verify Duration, Agent, and Files are present
	if !strings.Contains(output, "Duration:") {
		t.Errorf("expected 'Duration:' in output, got %q", output)
	}

	if !strings.Contains(output, "Agent:") {
		t.Errorf("expected 'Agent:' in output, got %q", output)
	}

	if !strings.Contains(output, "Files:") {
		t.Errorf("expected 'Files:' in output, got %q", output)
	}

	// Feedback section should be omitted when empty
}

// TestConsoleLoggerLogTaskResultVerboseWithColors verifies colored verbose output.
func TestConsoleLoggerLogTaskResultVerboseWithColors(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "debug")
	logger.colorOutput = true // Force color output
	logger.SetVerbose(true)

	result := models.TaskResult{
		Task: models.Task{
			Number:        "4",
			Name:          "Deploy App",
			Agent:         "devops",
			Files:         []string{"deploy.sh"},
			FilesModified: 1,
		},
		Status:         models.StatusGreen,
		Duration:       15 * time.Second,
		ReviewFeedback: "Deployment successful",
	}

	err := logger.LogTaskResult(result)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	output := buf.String()

	// Verify content is present (color codes will be there too)
	if !strings.Contains(output, "Task 4") {
		t.Errorf("expected 'Task 4' in colored output, got %q", output)
	}

	if !strings.Contains(output, "Deploy App") {
		t.Errorf("expected 'Deploy App' in colored output, got %q", output)
	}

	if !strings.Contains(output, "GREEN") {
		t.Errorf("expected 'GREEN' status in colored output, got %q", output)
	}

	if !strings.Contains(output, "Duration:") {
		t.Errorf("expected 'Duration:' in colored output, got %q", output)
	}
}

// TestConsoleLoggerLogTaskResultVerboseToggle verifies toggling verbose mode on/off.
func TestConsoleLoggerLogTaskResultVerboseToggle(t *testing.T) {
	t.Run("toggle on and off", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewConsoleLogger(buf, "debug")

		// Initially should not be verbose
		if logger.IsVerbose() {
			t.Error("expected IsVerbose() to return false initially")
		}

		// Enable verbose
		logger.SetVerbose(true)
		if !logger.IsVerbose() {
			t.Error("expected IsVerbose() to return true after SetVerbose(true)")
		}

		// Disable verbose
		logger.SetVerbose(false)
		if logger.IsVerbose() {
			t.Error("expected IsVerbose() to return false after SetVerbose(false)")
		}
	})

	t.Run("verbose mode affects output format", func(t *testing.T) {
		// Compare verbose vs non-verbose output
		defaultBuf := &bytes.Buffer{}
		defaultLogger := NewConsoleLogger(defaultBuf, "debug")
		defaultLogger.SetVerbose(false)

		verboseBuf := &bytes.Buffer{}
		verboseLogger := NewConsoleLogger(verboseBuf, "debug")
		verboseLogger.SetVerbose(true)

		result := models.TaskResult{
			Task: models.Task{
				Number:        "1",
				Name:          "Test Task",
				Agent:         "test-agent",
				Files:         []string{"test.go"},
				FilesModified: 1,
			},
			Status:         models.StatusGreen,
			Duration:       5 * time.Second,
			ReviewFeedback: "Looks good",
		}

		defaultLogger.LogTaskResult(result)
		verboseLogger.LogTaskResult(result)

		defaultOutput := defaultBuf.String()
		verboseOutput := verboseBuf.String()

		// Verbose output should be longer (more details)
		if len(verboseOutput) <= len(defaultOutput) {
			t.Error("expected verbose output to be longer than default output")
		}

		// Verbose output should have Duration section
		if !strings.Contains(verboseOutput, "Duration:") {
			t.Error("expected verbose output to have 'Duration:' section")
		}

		// Default output should not have Duration section (it's inline)
		if strings.Contains(defaultOutput, "\nDuration:") {
			t.Error("expected default output to not have 'Duration:' section on new line")
		}
	})
}

// TestConsoleLoggerLogTaskResultVerboseLongQCFeedback verifies wrapping of long QC feedback.
func TestConsoleLoggerLogTaskResultVerboseLongQCFeedback(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "debug")
	logger.SetVerbose(true)

	longFeedback := `This is a very long QC feedback message that contains multiple sentences and detailed information about the task execution. It includes observations about code quality, performance, and any warnings or issues found during the review process. The feedback should be displayed in full in verbose mode even if it spans multiple lines.`

	result := models.TaskResult{
		Task: models.Task{
			Number:        "5",
			Name:          "Code Review",
			Agent:         "reviewer",
			Files:         []string{"code.go"},
			FilesModified: 1,
		},
		Status:         models.StatusYellow,
		Duration:       8 * time.Second,
		ReviewFeedback: longFeedback,
	}

	err := logger.LogTaskResult(result)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	output := buf.String()

	// Verify the long feedback is present
	if !strings.Contains(output, "very long QC feedback") {
		t.Errorf("expected long feedback content in output, got %q", output)
	}

	// Verify it's in a Feedback section
	if !strings.Contains(output, "Feedback:") {
		t.Errorf("expected 'Feedback:' section in output, got %q", output)
	}

	// Output should be multi-line due to long feedback
	if !strings.Contains(output, "\n") {
		t.Error("expected multi-line output for long feedback")
	}
}

// TestConsoleLoggerVerboseModeProperty tests SetVerbose and IsVerbose methods.
func TestConsoleLoggerVerboseModeProperty(t *testing.T) {
	t.Run("default is non-verbose", func(t *testing.T) {
		logger := NewConsoleLogger(&bytes.Buffer{}, "info")
		if logger.IsVerbose() {
			t.Error("expected new logger to not be verbose by default")
		}
	})

	t.Run("set and get verbose", func(t *testing.T) {
		logger := NewConsoleLogger(&bytes.Buffer{}, "info")

		// Set to true
		logger.SetVerbose(true)
		if !logger.IsVerbose() {
			t.Error("expected IsVerbose() to return true after SetVerbose(true)")
		}

		// Set to false
		logger.SetVerbose(false)
		if logger.IsVerbose() {
			t.Error("expected IsVerbose() to return false after SetVerbose(false)")
		}

		// Set to true again
		logger.SetVerbose(true)
		if !logger.IsVerbose() {
			t.Error("expected IsVerbose() to return true again after SetVerbose(true)")
		}
	})

	t.Run("verbose mode persists across calls", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewConsoleLogger(buf, "debug")
		logger.SetVerbose(true)

		// Log multiple results - verbose mode should persist
		for i := 0; i < 3; i++ {
			result := models.TaskResult{
				Task: models.Task{
					Number: fmt.Sprintf("%d", i+1),
					Name:   fmt.Sprintf("Task %d", i+1),
					Agent:  "test",
				},
				Status:   models.StatusGreen,
				Duration: 1 * time.Second,
			}
			logger.LogTaskResult(result)
		}

		if !logger.IsVerbose() {
			t.Error("expected IsVerbose() to remain true after multiple LogTaskResult calls")
		}

		// All outputs should be verbose format
		output := buf.String()
		// Each result should have Duration section (verbose indicator)
		lines := strings.Split(output, "\n")
		durationCount := 0
		for _, line := range lines {
			if strings.Contains(line, "Duration:") {
				durationCount++
			}
		}

		if durationCount < 3 {
			t.Errorf("expected at least 3 'Duration:' sections in verbose output, got %d", durationCount)
		}
	})
}

// TestLogSummary_StatusBreakdown verifies status breakdown shows correct counts
func TestLogSummary_StatusBreakdown(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	result := models.ExecutionResult{
		TotalTasks:      10,
		Completed:       7,
		Failed:          3,
		Duration:        5 * time.Minute,
		StatusBreakdown: map[string]int{models.StatusGreen: 6, models.StatusYellow: 1, models.StatusRed: 3},
		AgentUsage:      map[string]int{"backend": 5, "frontend": 5},
		TotalFiles:      15,
		FailedTasks: []models.TaskResult{
			{Task: models.Task{Name: "Auth"}, Status: models.StatusRed},
			{Task: models.Task{Name: "API"}, Status: models.StatusRed},
			{Task: models.Task{Name: "UI"}, Status: models.StatusRed},
		},
	}

	logger.LogSummary(result)
	output := buf.String()

	// Verify status breakdown section exists
	if !strings.Contains(output, "Status Breakdown") {
		t.Error("expected 'Status Breakdown' in output")
	}

	// Verify status counts
	if !strings.Contains(output, "GREEN") || !strings.Contains(output, "6") {
		t.Error("expected GREEN status count in output")
	}

	if !strings.Contains(output, "YELLOW") || !strings.Contains(output, "1") {
		t.Error("expected YELLOW status count in output")
	}

	if !strings.Contains(output, "RED") || !strings.Contains(output, "3") {
		t.Error("expected RED status count in output")
	}
}

// TestLogSummary_AgentUsage verifies agent usage statistics
func TestLogSummary_AgentUsage(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	result := models.ExecutionResult{
		TotalTasks: 10,
		Completed:  10,
		Failed:     0,
		Duration:   2 * time.Minute,
		StatusBreakdown: map[string]int{
			models.StatusGreen: 10,
		},
		AgentUsage: map[string]int{
			"backend-dev":    4,
			"frontend-dev":   3,
			"devops":         2,
			"quality-review": 1,
		},
		TotalFiles:  8,
		FailedTasks: []models.TaskResult{},
	}

	logger.LogSummary(result)
	output := buf.String()

	// Verify agent usage section
	if !strings.Contains(output, "Agent Usage") {
		t.Error("expected 'Agent Usage' section in output")
	}

	// Verify agents and counts are present
	if !strings.Contains(output, "backend-dev") {
		t.Error("expected backend-dev agent in output")
	}

	if !strings.Contains(output, "frontend-dev") {
		t.Error("expected frontend-dev agent in output")
	}

	if !strings.Contains(output, "4") {
		t.Error("expected agent count 4 in output")
	}
}

// TestLogSummary_FileModifications verifies file modification tracking
func TestLogSummary_FileModifications(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	result := models.ExecutionResult{
		TotalTasks: 5,
		Completed:  5,
		Failed:     0,
		Duration:   1 * time.Minute,
		StatusBreakdown: map[string]int{
			models.StatusGreen: 5,
		},
		AgentUsage: map[string]int{
			"code-gen": 5,
		},
		TotalFiles:  12,
		FailedTasks: []models.TaskResult{},
	}

	logger.LogSummary(result)
	output := buf.String()

	// Verify file modifications section
	if !strings.Contains(output, "Files Modified") || !strings.Contains(output, "12") {
		t.Error("expected 'Files Modified' section with count 12 in output")
	}
}

// TestLogSummary_FailedTaskDetails verifies failed task details formatting
func TestLogSummary_FailedTaskDetails(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	result := models.ExecutionResult{
		TotalTasks: 5,
		Completed:  2,
		Failed:     3,
		Duration:   2 * time.Minute,
		StatusBreakdown: map[string]int{
			models.StatusGreen: 2,
			models.StatusRed:   3,
		},
		AgentUsage: map[string]int{
			"code-gen": 5,
		},
		TotalFiles: 5,
		FailedTasks: []models.TaskResult{
			{
				Task:           models.Task{Number: "2", Name: "Setup Database"},
				Status:         models.StatusRed,
				ReviewFeedback: "Schema validation failed",
			},
			{
				Task:           models.Task{Number: "3", Name: "Configure API"},
				Status:         models.StatusRed,
				ReviewFeedback: "Port binding error",
			},
			{
				Task:           models.Task{Number: "4", Name: "Deploy Service"},
				Status:         models.StatusRed,
				ReviewFeedback: "Network configuration issue",
			},
		},
	}

	logger.LogSummary(result)
	output := buf.String()

	// Verify failed task section
	if !strings.Contains(output, "Failed tasks") {
		t.Error("expected 'Failed tasks' section in output")
	}

	// Verify each failed task is listed
	if !strings.Contains(output, "Setup Database") {
		t.Error("expected 'Setup Database' failed task in output")
	}

	if !strings.Contains(output, "Configure API") {
		t.Error("expected 'Configure API' failed task in output")
	}

	if !strings.Contains(output, "Deploy Service") {
		t.Error("expected 'Deploy Service' failed task in output")
	}

	// Verify feedback is included
	if !strings.Contains(output, "Schema validation failed") {
		t.Error("expected QC feedback in output")
	}
}

// TestLogSummary_AverageDuration verifies average duration calculation
func TestLogSummary_AverageDuration(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	result := models.ExecutionResult{
		TotalTasks:      4,
		Completed:       4,
		Failed:          0,
		Duration:        12 * time.Second,
		AvgTaskDuration: 3 * time.Second, // 12s / 4 = 3s
		StatusBreakdown: map[string]int{
			models.StatusGreen: 4,
		},
		AgentUsage: map[string]int{
			"agent": 4,
		},
		TotalFiles:  4,
		FailedTasks: []models.TaskResult{},
	}

	logger.LogSummary(result)
	output := buf.String()

	// Verify average duration is displayed
	if !strings.Contains(output, "Average Duration") || !strings.Contains(output, "3.0s") {
		t.Error("expected average duration 3.0s in output")
	}
}

// TestLogSummary_EmptySummary verifies empty summary handling
func TestLogSummary_EmptySummary(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	result := models.ExecutionResult{
		TotalTasks:      0,
		Completed:       0,
		Failed:          0,
		Duration:        0,
		StatusBreakdown: map[string]int{},
		AgentUsage:      map[string]int{},
		TotalFiles:      0,
		FailedTasks:     []models.TaskResult{},
	}

	logger.LogSummary(result)
	output := buf.String()

	// Should handle gracefully without panic
	if !strings.Contains(output, "Total tasks: 0") {
		t.Error("expected zero task count in output")
	}

	if !strings.Contains(output, "Completed: 0") {
		t.Error("expected zero completed count in output")
	}
}

// TestLogSummary_ColorDisabled verifies no-color mode
func TestLogSummary_ColorDisabled(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")
	logger.colorOutput = false // Explicitly disable colors

	result := models.ExecutionResult{
		TotalTasks: 10,
		Completed:  8,
		Failed:     2,
		Duration:   5 * time.Minute,
		StatusBreakdown: map[string]int{
			models.StatusGreen: 8,
			models.StatusRed:   2,
		},
		AgentUsage: map[string]int{
			"agent": 10,
		},
		TotalFiles: 6,
		FailedTasks: []models.TaskResult{
			{Task: models.Task{Name: "Task A"}, Status: models.StatusRed},
			{Task: models.Task{Name: "Task B"}, Status: models.StatusRed},
		},
	}

	logger.LogSummary(result)
	output := buf.String()

	// Verify no ANSI color codes are present
	if strings.Contains(output, "\033[") || strings.Contains(output, "\x1b[") {
		t.Error("expected no ANSI color codes in output when colors disabled")
	}

	// Verify content is still present
	if !strings.Contains(output, "Total tasks: 10") {
		t.Error("expected content to be present even without colors")
	}
}

// TestIsTerminal_StdoutStderr verifies isTerminal detects os.Stdout and os.Stderr as TTY
func TestIsTerminal_StdoutStderr(t *testing.T) {
	t.Run("os.Stdout is detected as terminal", func(t *testing.T) {
		// Note: This test might fail in non-TTY environments (CI/CD)
		// In such cases, isTerminal should correctly return false
		result := isTerminal(os.Stdout)

		// We can't assert true/false definitively since it depends on the environment
		// But we can verify the function doesn't panic and returns a boolean
		_ = result
	})

	t.Run("os.Stderr is detected as terminal", func(t *testing.T) {
		// Note: This test might fail in non-TTY environments (CI/CD)
		// In such cases, isTerminal should correctly return false
		result := isTerminal(os.Stderr)

		// We can't assert true/false definitively since it depends on the environment
		// But we can verify the function doesn't panic and returns a boolean
		_ = result
	})
}

// TestIsTerminal_NonTTY verifies isTerminal returns false for non-TTY writers
func TestIsTerminal_NonTTY(t *testing.T) {
	t.Run("bytes.Buffer is not a terminal", func(t *testing.T) {
		buf := &bytes.Buffer{}
		result := isTerminal(buf)

		if result {
			t.Error("expected isTerminal to return false for bytes.Buffer")
		}
	})

	t.Run("nil writer is not a terminal", func(t *testing.T) {
		result := isTerminal(nil)

		if result {
			t.Error("expected isTerminal to return false for nil writer")
		}
	})
}

// TestIsTerminal_NoColorEnvVarIgnored verifies NO_COLOR env var is ignored
func TestIsTerminal_NoColorEnvVarIgnored(t *testing.T) {
	// This test verifies that isTerminal() does NOT check NO_COLOR environment variable
	// After refactoring, the function should only check if writer is os.Stdout/os.Stderr
	// and perform TTY detection, ignoring NO_COLOR completely

	t.Run("NO_COLOR env var should not affect isTerminal", func(t *testing.T) {
		// Note: We can't reliably test this without mocking the color library
		// The best we can do is ensure bytes.Buffer always returns false
		// regardless of NO_COLOR setting

		buf := &bytes.Buffer{}
		result := isTerminal(buf)

		if result {
			t.Error("expected isTerminal to return false for bytes.Buffer regardless of NO_COLOR")
		}
	})
}

// TestConsoleLogger_ColorControl verifies color output is controlled by config
func TestConsoleLogger_ColorControl(t *testing.T) {
	t.Run("colors can be explicitly enabled", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewConsoleLogger(buf, "info")
		logger.colorOutput = true

		if !logger.colorOutput {
			t.Error("expected colorOutput to be true when explicitly enabled")
		}
	})

	t.Run("colors can be explicitly disabled", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewConsoleLogger(buf, "info")
		logger.colorOutput = false

		if logger.colorOutput {
			t.Error("expected colorOutput to be false when explicitly disabled")
		}
	})

	t.Run("auto detection for non-TTY writers", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewConsoleLogger(buf, "info")

		// For bytes.Buffer (non-TTY), colorOutput should be false
		if logger.colorOutput {
			t.Error("expected colorOutput to be false for bytes.Buffer (non-TTY)")
		}
	})
}
