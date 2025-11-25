package cmd

import (
	"testing"

	"github.com/harrison/conductor/internal/learning"
)

func TestFormatBashAnalysisTable(t *testing.T) {
	tests := []struct {
		name     string
		commands []learning.BashStats
		limit    int
		verify   func(string) bool
	}{
		{
			name:     "empty commands",
			commands: []learning.BashStats{},
			limit:    20,
			verify: func(s string) bool {
				return contains(s, "No bash command data available")
			},
		},
		{
			name: "single command",
			commands: []learning.BashStats{
				{
					Command:       "ls -la",
					CallCount:     50,
					SuccessCount:  49,
					FailureCount:  1,
					AvgDurationMs: 100.0,
					SuccessRate:   0.98,
				},
			},
			limit: 20,
			verify: func(s string) bool {
				return contains(s, "ls -la") &&
					contains(s, "50") &&
					contains(s, "Bash Command Analysis")
			},
		},
		{
			name: "multiple commands with pagination",
			commands: []learning.BashStats{
				{
					Command:       "git status",
					CallCount:     100,
					SuccessCount:  95,
					FailureCount:  5,
					AvgDurationMs: 500.0,
					SuccessRate:   0.95,
				},
				{
					Command:       "go test ./...",
					CallCount:     75,
					SuccessCount:  70,
					FailureCount:  5,
					AvgDurationMs: 2000.0,
					SuccessRate:   0.933,
				},
				{
					Command:       "make build",
					CallCount:     60,
					SuccessCount:  55,
					FailureCount:  5,
					AvgDurationMs: 1500.0,
					SuccessRate:   0.917,
				},
			},
			limit: 2,
			verify: func(s string) bool {
				return contains(s, "Showing 2 of 3 commands")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBashAnalysisTable(tt.commands, tt.limit)
			if !tt.verify(result) {
				t.Errorf("formatBashAnalysisTable output verification failed\nGot:\n%s", result)
			}
		})
	}
}

func TestDisplayBashAnalysis_MissingDatabase(t *testing.T) {
	// This test verifies that the function properly handles a missing database
	// In actual usage, this would fail due to missing config, which is expected
	err := DisplayBashAnalysis("", 20)
	if err == nil {
		t.Error("Expected error for missing database, got nil")
	}
}

func TestBashStatsCalculations(t *testing.T) {
	tests := []struct {
		name          string
		callCount     int
		successCount  int
		expectedRate  float64
	}{
		{
			name:          "100% success",
			callCount:     20,
			successCount:  20,
			expectedRate:  1.0,
		},
		{
			name:          "80% success",
			callCount:     10,
			successCount:  8,
			expectedRate:  0.8,
		},
		{
			name:          "0% success",
			callCount:     5,
			successCount:  0,
			expectedRate:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := learning.BashStats{
				Command:      "test_cmd",
				CallCount:    tt.callCount,
				SuccessCount: tt.successCount,
				FailureCount: tt.callCount - tt.successCount,
			}

			if cmd.CallCount > 0 {
				cmd.SuccessRate = float64(cmd.SuccessCount) / float64(cmd.CallCount)
			}

			if cmd.SuccessRate != tt.expectedRate {
				t.Errorf("Expected success rate %.2f, got %.2f", tt.expectedRate, cmd.SuccessRate)
			}
		})
	}
}
