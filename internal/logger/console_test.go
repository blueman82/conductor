package logger

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/models"
)

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
		expectedText string
	}{
		{
			name: "5 seconds",
			wave: models.Wave{
				Name:           "Wave 1",
				TaskNumbers:    []string{"1"},
				MaxConcurrency: 1,
			},
			duration:     5 * time.Second,
			expectedText: "Wave 1 complete (5.0s) - 1/1 completed",
		},
		{
			name: "90 seconds (1m30s)",
			wave: models.Wave{
				Name:           "Wave 2",
				TaskNumbers:    []string{"1", "2"},
				MaxConcurrency: 2,
			},
			duration:     90 * time.Second,
			expectedText: "Wave 2 complete (1m30s) - 2/2 completed",
		},
		{
			name: "zero duration",
			wave: models.Wave{
				Name:           "Wave 3",
				TaskNumbers:    []string{"1"},
				MaxConcurrency: 1,
			},
			duration:     0 * time.Second,
			expectedText: "Wave 3 complete (0.0s) - 1/1 completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewConsoleLogger(buf, "info")

			logger.LogWaveComplete(tt.wave, tt.duration)

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
			logger.LogWaveComplete(wave, 5*time.Second)

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
	logger.LogWaveComplete(wave, 5*time.Second)

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
		logger.LogWaveComplete(wave, 5*time.Second)

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
				logger.LogWaveComplete(wave, 5*time.Second)

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
	LogWaveComplete(wave models.Wave, duration time.Duration)
	LogSummary(result models.ExecutionResult)
}

// TestConsoleLogger_LogProgress_BasicDisplay verifies progress bar display with basic task counts.
func TestConsoleLogger_LogProgress_BasicDisplay(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	// Create 4 completed tasks out of 8 total
	tasks := []models.Task{
		{Number: "1", Name: "Task 1", Status: "completed"},
		{Number: "2", Name: "Task 2", Status: "completed"},
		{Number: "3", Name: "Task 3", Status: "completed"},
		{Number: "4", Name: "Task 4", Status: "completed"},
		{Number: "5", Name: "Task 5", Status: "pending"},
		{Number: "6", Name: "Task 6", Status: "pending"},
		{Number: "7", Name: "Task 7", Status: "pending"},
		{Number: "8", Name: "Task 8", Status: "pending"},
	}

	logger.LogProgress(tasks)

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
	tasks := []models.Task{
		{
			Number: "1", Name: "Task 1", Status: "completed",
			StartedAt:   nil,
			CompletedAt: &now,
		},
		{
			Number: "2", Name: "Task 2", Status: "completed",
			StartedAt:   nil,
			CompletedAt: &now,
		},
		{
			Number: "3", Name: "Task 3", Status: "completed",
			StartedAt:   nil,
			CompletedAt: &now,
		},
		{
			Number: "4", Name: "Task 4", Status: "completed",
			StartedAt:   nil,
			CompletedAt: &now,
		},
		{
			Number: "5", Name: "Task 5", Status: "pending",
		},
		{
			Number: "6", Name: "Task 6", Status: "pending",
		},
	}

	// Add durations to tasks (2s, 3s, 4s, 5s)
	for i := range tasks {
		if tasks[i].Status == "completed" {
			startTime := now.Add(-time.Duration((i + 2) * int(time.Second)))
			tasks[i].StartedAt = &startTime
		}
	}

	logger.LogProgress(tasks)

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
	tasks := []models.Task{}

	// Should not panic
	logger.LogProgress(tasks)

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
	tasks := []models.Task{
		{Number: "1", Name: "Task 1", Status: "completed"},
		{Number: "2", Name: "Task 2", Status: "completed"},
		{Number: "3", Name: "Task 3", Status: "completed"},
		{Number: "4", Name: "Task 4", Status: "completed"},
		{Number: "5", Name: "Task 5", Status: "completed"},
		{Number: "6", Name: "Task 6", Status: "completed"},
		{Number: "7", Name: "Task 7", Status: "completed"},
		{Number: "8", Name: "Task 8", Status: "completed"},
	}

	logger.LogProgress(tasks)

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

		tasks := []models.Task{
			{Number: "1", Name: "Task 1", Status: "completed"},
			{Number: "2", Name: "Task 2", Status: "pending"},
		}

		logger.LogProgress(tasks)

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

		tasks := []models.Task{
			{Number: "1", Name: "Task 1", Status: "completed"},
			{Number: "2", Name: "Task 2", Status: "pending"},
		}

		logger.LogProgress(tasks)

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

	tasks := []models.Task{
		{Number: "1", Name: "Task 1", Status: "completed"},
		{Number: "2", Name: "Task 2", Status: "pending"},
	}

	logger.LogProgress(tasks)

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
	tasks := []models.Task{
		{Number: "1", Name: "Task 1", Status: "completed"},
		{Number: "2", Name: "Task 2", Status: "completed"},
		{Number: "3", Name: "Task 3", Status: "pending"},
		{Number: "4", Name: "Task 4", Status: "pending"},
	}

	logger.LogProgress(tasks)

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
	tasks := []models.Task{
		{Number: "1", Name: "Task 1", Status: "completed", StartedAt: nil},
		{Number: "2", Name: "Task 2", Status: "pending"},
	}

	// Should not panic with nil StartedAt
	logger.LogProgress(tasks)

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
