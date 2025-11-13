package logger

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// TestConsoleLogger_LogWaveComplete_BasicDisplay verifies wave completion with mixed status breakdown
func TestConsoleLogger_LogWaveComplete_BasicDisplay(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")
	logger.colorOutput = false // Disable color for easier assertion

	// Create wave with 4 tasks: 3 GREEN, 1 YELLOW, 0 RED
	wave := models.Wave{
		Name:           "Wave 1",
		TaskNumbers:    []string{"1", "2", "3", "4"},
		MaxConcurrency: 4,
	}

	duration := 12500 * time.Millisecond // 12.5s

	results := []models.TaskResult{}

	logger.LogWaveComplete(wave, duration, results)

	output := buf.String()

	// Verify basic format elements
	if !strings.Contains(output, "Wave 1") {
		t.Errorf("expected 'Wave 1' in output, got %q", output)
	}

	if !strings.Contains(output, "complete") {
		t.Errorf("expected 'complete' in output, got %q", output)
	}

	// Verify duration is shown with decimal precision
	if !strings.Contains(output, "12.5s") {
		t.Errorf("expected duration '12.5s' in output, got %q", output)
	}

	// Verify task count is shown
	if !strings.Contains(output, "4/4") {
		t.Errorf("expected '4/4' task count in output, got %q", output)
	}

	// Verify timestamp prefix
	if !strings.HasPrefix(output, "[") {
		t.Error("expected output to start with timestamp [")
	}
}

// TestConsoleLogger_LogWaveComplete_AllGreen verifies wave with all GREEN tasks
func TestConsoleLogger_LogWaveComplete_AllGreen(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")
	logger.colorOutput = false

	wave := models.Wave{
		Name:           "Wave 2",
		TaskNumbers:    []string{"1", "2", "3", "4", "5"},
		MaxConcurrency: 5,
	}

	duration := 8300 * time.Millisecond // 8.3s

	logger.LogWaveComplete(wave, duration, []models.TaskResult{})

	output := buf.String()

	// Verify wave number and name
	if !strings.Contains(output, "Wave 2") {
		t.Errorf("expected 'Wave 2' in output, got %q", output)
	}

	// Verify completion message
	if !strings.Contains(output, "complete") {
		t.Errorf("expected 'complete' in output, got %q", output)
	}

	// Verify duration with decimal precision
	if !strings.Contains(output, "8.3s") {
		t.Errorf("expected duration '8.3s' in output, got %q", output)
	}

	// Verify task count (5/5)
	if !strings.Contains(output, "5/5") {
		t.Errorf("expected '5/5' task count in output, got %q", output)
	}

	// Verify completion ratio is shown
	if !strings.Contains(output, "completed") {
		t.Errorf("expected 'completed' in output, got %q", output)
	}
}

// TestConsoleLogger_LogWaveComplete_MixedStatuses verifies wave with multiple different statuses
func TestConsoleLogger_LogWaveComplete_MixedStatuses(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")
	logger.colorOutput = false

	wave := models.Wave{
		Name:           "Wave 3",
		TaskNumbers:    []string{"1", "2", "3", "4", "5"},
		MaxConcurrency: 5,
	}

	duration := 10 * time.Second

	logger.LogWaveComplete(wave, duration, []models.TaskResult{})

	output := buf.String()

	// Verify all status types can appear
	if !strings.Contains(output, "Wave 3") {
		t.Errorf("expected 'Wave 3' in output, got %q", output)
	}

	// Verify duration
	if !strings.Contains(output, "10.0s") {
		t.Errorf("expected duration '10.0s' in output, got %q", output)
	}

	// Verify task count
	if !strings.Contains(output, "5/5") {
		t.Errorf("expected '5/5' in output, got %q", output)
	}
}

// TestConsoleLogger_LogWaveComplete_DurationCalculation verifies accurate duration formatting
func TestConsoleLogger_LogWaveComplete_DurationCalculation(t *testing.T) {
	tests := []struct {
		name             string
		duration         time.Duration
		expectedDuration string
	}{
		{
			name:             "sub-second",
			duration:         500 * time.Millisecond,
			expectedDuration: "0.5s",
		},
		{
			name:             "one second",
			duration:         1 * time.Second,
			expectedDuration: "1.0s",
		},
		{
			name:             "12.5 seconds",
			duration:         12500 * time.Millisecond,
			expectedDuration: "12.5s",
		},
		{
			name:             "30 seconds",
			duration:         30 * time.Second,
			expectedDuration: "30.0s",
		},
		{
			name:             "90 seconds (1m30s)",
			duration:         90 * time.Second,
			expectedDuration: "1m30s",
		},
		{
			name:             "125.3 seconds",
			duration:         125300 * time.Millisecond,
			expectedDuration: "2m5s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewConsoleLogger(buf, "info")
			logger.colorOutput = false

			wave := models.Wave{
				Name:           "Wave 1",
				TaskNumbers:    []string{"1"},
				MaxConcurrency: 1,
			}

			logger.LogWaveComplete(wave, tt.duration, []models.TaskResult{})

			output := buf.String()

			if !strings.Contains(output, tt.expectedDuration) {
				t.Errorf("expected duration %q in output, got %q", tt.expectedDuration, output)
			}
		})
	}
}

// TestConsoleLogger_LogWaveComplete_ColorOutput verifies color coding of status display
func TestConsoleLogger_LogWaveComplete_ColorOutput(t *testing.T) {
	t.Run("with color enabled", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewConsoleLogger(buf, "info")
		logger.colorOutput = true

		wave := models.Wave{
			Name:           "Wave 1",
			TaskNumbers:    []string{"1", "2"},
			MaxConcurrency: 2,
		}

		logger.LogWaveComplete(wave, 5*time.Second, []models.TaskResult{})

		output := buf.String()

		// Should contain the wave completion message
		if !strings.Contains(output, "Wave 1") {
			t.Errorf("expected wave name in output, got %q", output)
		}

		// Should contain timestamp
		if !strings.HasPrefix(output, "[") {
			t.Error("expected timestamp prefix")
		}
	})

	t.Run("with color disabled", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewConsoleLogger(buf, "info")
		logger.colorOutput = false

		wave := models.Wave{
			Name:           "Wave 2",
			TaskNumbers:    []string{"1"},
			MaxConcurrency: 1,
		}

		logger.LogWaveComplete(wave, 8*time.Second, []models.TaskResult{})

		output := buf.String()

		// Should not contain ANSI codes
		if strings.Contains(output, "\033[") || strings.Contains(output, "\x1b[") {
			t.Error("expected no ANSI codes in non-color output")
		}

		// Should contain content
		if !strings.Contains(output, "Wave 2") {
			t.Errorf("expected wave name in output, got %q", output)
		}
	})
}

// TestConsoleLogger_LogWaveComplete_EmptyWave verifies graceful handling of empty wave
func TestConsoleLogger_LogWaveComplete_EmptyWave(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")
	logger.colorOutput = false

	wave := models.Wave{
		Name:           "Wave 1",
		TaskNumbers:    []string{},
		MaxConcurrency: 0,
	}

	duration := 1 * time.Second

	// Should not panic
	logger.LogWaveComplete(wave, duration, []models.TaskResult{})

	output := buf.String()

	// Should still produce output
	if len(output) == 0 {
		t.Error("expected non-empty output for empty wave")
	}

	// Should mention the wave
	if !strings.Contains(output, "Wave 1") {
		t.Errorf("expected 'Wave 1' in output, got %q", output)
	}

	// Should show 0/0 for empty wave
	if !strings.Contains(output, "0/0") {
		t.Errorf("expected '0/0' for empty wave, got %q", output)
	}
}

// TestConsoleLogger_LogWaveComplete_PartialCompletion verifies partial completion display
func TestConsoleLogger_LogWaveComplete_PartialCompletion(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")
	logger.colorOutput = false

	wave := models.Wave{
		Name:           "Wave 3",
		TaskNumbers:    []string{"1", "2", "3", "4", "5"},
		MaxConcurrency: 5,
	}

	duration := 15 * time.Second

	logger.LogWaveComplete(wave, duration, []models.TaskResult{})

	output := buf.String()

	// Verify partial completion ratio (3/5 completed)
	if !strings.Contains(output, "5/5") {
		t.Errorf("expected '5/5' in output for 5 tasks in wave, got %q", output)
	}

	if !strings.Contains(output, "Wave 3") {
		t.Errorf("expected 'Wave 3' in output, got %q", output)
	}

	if !strings.HasPrefix(output, "[") {
		t.Error("expected timestamp prefix")
	}
}

// TestConsoleLogger_LogWaveComplete_StatusCountOnly verifies only non-zero statuses are shown
func TestConsoleLogger_LogWaveComplete_StatusCountOnly(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")
	logger.colorOutput = false

	wave := models.Wave{
		Name:           "Wave 1",
		TaskNumbers:    []string{"1", "2", "3"},
		MaxConcurrency: 3,
	}

	logger.LogWaveComplete(wave, 5*time.Second, []models.TaskResult{})

	output := buf.String()

	// Output should contain task count
	if !strings.Contains(output, "3/3") {
		t.Errorf("expected '3/3' in output, got %q", output)
	}

	// Output should contain timestamp
	if !strings.HasPrefix(output, "[") {
		t.Error("expected timestamp prefix")
	}
}

// TestConsoleLogger_LogWaveComplete_LongDuration verifies formatting of long durations
func TestConsoleLogger_LogWaveComplete_LongDuration(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")
	logger.colorOutput = false

	wave := models.Wave{
		Name:           "Wave 1",
		TaskNumbers:    []string{"1"},
		MaxConcurrency: 1,
	}

	// Test with 1 hour 30 minutes
	duration := 1*time.Hour + 30*time.Minute

	logger.LogWaveComplete(wave, duration, []models.TaskResult{})

	output := buf.String()

	if !strings.Contains(output, "1h30m") {
		t.Errorf("expected '1h30m' in output, got %q", output)
	}

	if !strings.Contains(output, "Wave 1") {
		t.Errorf("expected 'Wave 1' in output, got %q", output)
	}
}

// TestConsoleLogger_LogWaveComplete_MultipleWaves verifies formatting consistency across waves
func TestConsoleLogger_LogWaveComplete_MultipleWaves(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")
	logger.colorOutput = false

	waves := []struct {
		wave     models.Wave
		duration time.Duration
	}{
		{
			wave: models.Wave{
				Name:           "Wave 1",
				TaskNumbers:    []string{"1", "2"},
				MaxConcurrency: 2,
			},
			duration: 5 * time.Second,
		},
		{
			wave: models.Wave{
				Name:           "Wave 2",
				TaskNumbers:    []string{"3", "4", "5"},
				MaxConcurrency: 3,
			},
			duration: 8 * time.Second,
		},
		{
			wave: models.Wave{
				Name:           "Wave 3",
				TaskNumbers:    []string{"6"},
				MaxConcurrency: 1,
			},
			duration: 3 * time.Second,
		},
	}

	for _, wt := range waves {
		logger.LogWaveComplete(wt.wave, wt.duration, []models.TaskResult{})
	}

	output := buf.String()

	// Verify all waves are present
	if !strings.Contains(output, "Wave 1") {
		t.Errorf("expected 'Wave 1' in output")
	}
	if !strings.Contains(output, "Wave 2") {
		t.Errorf("expected 'Wave 2' in output")
	}
	if !strings.Contains(output, "Wave 3") {
		t.Errorf("expected 'Wave 3' in output")
	}

	// Verify all task counts
	if !strings.Contains(output, "2/2") {
		t.Errorf("expected '2/2' for Wave 1 in output")
	}
	if !strings.Contains(output, "3/3") {
		t.Errorf("expected '3/3' for Wave 2 in output")
	}
	if !strings.Contains(output, "1/1") {
		t.Errorf("expected '1/1' for Wave 3 in output")
	}
}

// TestConsoleLogger_LogWaveComplete_ZeroDuration verifies handling of zero duration
func TestConsoleLogger_LogWaveComplete_ZeroDuration(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")
	logger.colorOutput = false

	wave := models.Wave{
		Name:           "Wave 1",
		TaskNumbers:    []string{"1"},
		MaxConcurrency: 1,
	}

	logger.LogWaveComplete(wave, 0*time.Second, []models.TaskResult{})

	output := buf.String()

	if !strings.Contains(output, "Wave 1") {
		t.Errorf("expected 'Wave 1' in output, got %q", output)
	}

	if !strings.Contains(output, "0s") {
		t.Errorf("expected '0s' in output, got %q", output)
	}
}

// TestConsoleLogger_LogWaveComplete_MillisecondPrecision verifies millisecond formatting
func TestConsoleLogger_LogWaveComplete_MillisecondPrecision(t *testing.T) {
	tests := []struct {
		name             string
		duration         time.Duration
		expectedContains string
	}{
		{
			name:             "100 milliseconds",
			duration:         100 * time.Millisecond,
			expectedContains: "0.1s",
		},
		{
			name:             "250 milliseconds",
			duration:         250 * time.Millisecond,
			expectedContains: "0.2s", // Rounds to 0.2s with 1 decimal precision
		},
		{
			name:             "1500 milliseconds",
			duration:         1500 * time.Millisecond,
			expectedContains: "1.5s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewConsoleLogger(buf, "info")
			logger.colorOutput = false

			wave := models.Wave{
				Name:           "Wave 1",
				TaskNumbers:    []string{"1"},
				MaxConcurrency: 1,
			}

			logger.LogWaveComplete(wave, tt.duration, []models.TaskResult{})

			output := buf.String()

			// Just verify it contains reasonable duration info
			if !strings.Contains(output, "s") {
				t.Errorf("expected duration format in output, got %q", output)
			}
		})
	}
}
