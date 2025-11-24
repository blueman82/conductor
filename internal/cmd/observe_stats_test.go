package cmd

import (
	"testing"
	"time"

	"github.com/harrison/conductor/internal/behavioral"
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
	stats := &behavioral.AggregateStats{
		TotalSessions:     10,
		TotalAgents:       3,
		TotalOperations:   100,
		SuccessRate:       0.85,
		ErrorRate:         0.15,
		AverageDuration:   45 * time.Second,
		TotalCost:         2.5,
		TotalInputTokens:  5000,
		TotalOutputTokens: 2500,
		TopTools: []behavioral.ToolStatSummary{
			{Name: "Read", Count: 50, SuccessRate: 0.96, ErrorRate: 0.04},
			{Name: "Write", Count: 30, SuccessRate: 0.90, ErrorRate: 0.10},
		},
		AgentBreakdown: map[string]int{
			"agent1": 5,
			"agent2": 3,
		},
	}

	result := formatStatsTable(stats)

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

	// Should contain cost
	if !contains(result, "Total Cost:") {
		t.Error("expected Total Cost field")
	}

	// Should contain tool section
	if !contains(result, "Top Tools") {
		t.Error("expected Top Tools section")
	}

	// Should contain agent section
	if !contains(result, "Agent Performance") {
		t.Error("expected Agent Performance section")
	}
}

func TestFormatStatsTable_Empty(t *testing.T) {
	stats := &behavioral.AggregateStats{
		TopTools:       []behavioral.ToolStatSummary{},
		AgentBreakdown: map[string]int{},
	}

	result := formatStatsTable(stats)

	// Should still produce valid output with zero values
	if result == "" {
		t.Error("expected non-empty result")
	}

	// Should still contain summary section
	if !contains(result, "SUMMARY STATISTICS") {
		t.Error("expected SUMMARY STATISTICS header")
	}
}

func TestDisplaySessionUpdate(t *testing.T) {
	update := SessionUpdate{
		ID:        1,
		TaskName:  "Test Task",
		Agent:     "test-agent",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Success:   true,
		Duration:  30 * time.Second,
	}

	// This function prints to stdout, so we just verify it doesn't panic
	displaySessionUpdate(update)
}

func TestDisplaySessionUpdate_Failed(t *testing.T) {
	update := SessionUpdate{
		ID:        2,
		TaskName:  "Failed Task",
		Agent:     "",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Success:   false,
		Duration:  15 * time.Second,
	}

	// This function prints to stdout, so we just verify it doesn't panic
	displaySessionUpdate(update)
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
