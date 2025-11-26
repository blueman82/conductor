package cmd

import (
	"testing"
	"time"

	"github.com/harrison/conductor/internal/learning"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "seconds",
			duration: 45 * time.Second,
			expected: "45.0s",
		},
		{
			name:     "minutes",
			duration: 90 * time.Second,
			expected: "1.5m",
		},
		{
			name:     "hours",
			duration: 2*time.Hour + 30*time.Minute,
			expected: "2.5h",
		},
		{
			name:     "zero",
			duration: 0,
			expected: "0.0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestFormatStatsTable(t *testing.T) {
	summary := &learning.SummaryStats{
		TotalSessions:      10,
		TotalAgents:        3,
		TotalSuccesses:     8,
		TotalFailures:      2,
		SuccessRate:        0.85,
		AvgDurationSeconds: 45,
		TotalInputTokens:   5000,
		TotalOutputTokens:  2500,
		TotalTokens:        7500,
	}

	agents := []learning.AgentTypeStats{
		{AgentType: "agent1", SuccessCount: 5, FailureCount: 0, TotalSessions: 5, AvgDurationSeconds: 45},
		{AgentType: "agent2", SuccessCount: 3, FailureCount: 2, TotalSessions: 5, AvgDurationSeconds: 45},
	}

	result := formatStatsTable(summary, agents, 20)

	// Check that key information is present
	if result == "" {
		t.Error("expected non-empty result")
	}

	// Should contain summary section
	if !contains(result, "SUMMARY STATISTICS") {
		t.Error("expected SUMMARY STATISTICS header")
	}

	// Should contain session count
	if !contains(result, "Total Sessions:") {
		t.Error("expected Total Sessions field")
	}

	// Should contain agent section
	if !contains(result, "Agent Performance") {
		t.Error("expected Agent Performance section")
	}

	// Should contain agent names
	if !contains(result, "agent1") {
		t.Error("expected agent1 in output")
	}
}

func TestFormatStatsTable_Empty(t *testing.T) {
	summary := &learning.SummaryStats{
		TotalSessions: 0,
		TotalAgents:   0,
	}

	result := formatStatsTable(summary, []learning.AgentTypeStats{}, 20)

	// Should still produce valid output with zero values
	if result == "" {
		t.Error("expected non-empty result")
	}

	// Should still contain summary section
	if !contains(result, "SUMMARY STATISTICS") {
		t.Error("expected SUMMARY STATISTICS header")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
