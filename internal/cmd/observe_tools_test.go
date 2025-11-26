package cmd

import (
	"os"
	"testing"

	"github.com/harrison/conductor/internal/learning"
)

func TestFormatToolAnalysisTable(t *testing.T) {
	tests := []struct {
		name   string
		tools  []learning.ToolStats
		limit  int
		verify func(string) bool
	}{
		{
			name:  "empty tools",
			tools: []learning.ToolStats{},
			limit: 20,
			verify: func(s string) bool {
				return contains(s, "No tool data available")
			},
		},
		{
			name: "single tool",
			tools: []learning.ToolStats{
				{
					ToolName:      "Read",
					CallCount:     100,
					SuccessCount:  99,
					FailureCount:  1,
					AvgDurationMs: 45.5,
					SuccessRate:   0.99,
				},
			},
			limit: 20,
			verify: func(s string) bool {
				return contains(s, "Read") &&
					contains(s, "100") &&
					contains(s, "99") &&
					contains(s, "Tool Usage Analysis")
			},
		},
		{
			name: "multiple tools with limit",
			tools: []learning.ToolStats{
				{
					ToolName:      "Read",
					CallCount:     100,
					SuccessCount:  99,
					FailureCount:  1,
					AvgDurationMs: 45.5,
					SuccessRate:   0.99,
				},
				{
					ToolName:      "Edit",
					CallCount:     50,
					SuccessCount:  48,
					FailureCount:  2,
					AvgDurationMs: 120.0,
					SuccessRate:   0.96,
				},
				{
					ToolName:      "Bash",
					CallCount:     30,
					SuccessCount:  25,
					FailureCount:  5,
					AvgDurationMs: 850.0,
					SuccessRate:   0.833,
				},
			},
			limit: 2,
			verify: func(s string) bool {
				return contains(s, "Showing 2 of 3 tools")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatToolAnalysisTable(tt.tools, tt.limit)
			if !tt.verify(result) {
				t.Errorf("formatToolAnalysisTable output verification failed\nGot:\n%s", result)
			}
		})
	}
}

func TestDisplayToolAnalysis_NoData(t *testing.T) {
	// Test handles empty database gracefully (config defaults now used)
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	os.MkdirAll(".conductor", 0755)
	os.WriteFile(".conductor/config.yaml", []byte("learning:\n  db_path: .conductor/learning/test.db\n"), 0644)

	err := DisplayToolAnalysis("", 20)
	if err != nil {
		t.Errorf("Expected success with empty results, got error: %v", err)
	}
}

func TestToolStatsCalculations(t *testing.T) {
	tests := []struct {
		name         string
		callCount    int
		successCount int
		expectedRate float64
	}{
		{
			name:         "100% success",
			callCount:    10,
			successCount: 10,
			expectedRate: 1.0,
		},
		{
			name:         "50% success",
			callCount:    10,
			successCount: 5,
			expectedRate: 0.5,
		},
		{
			name:         "0% success",
			callCount:    10,
			successCount: 0,
			expectedRate: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := learning.ToolStats{
				ToolName:     "test",
				CallCount:    tt.callCount,
				SuccessCount: tt.successCount,
				FailureCount: tt.callCount - tt.successCount,
			}

			if tool.CallCount > 0 {
				tool.SuccessRate = float64(tool.SuccessCount) / float64(tool.CallCount)
			}

			if tool.SuccessRate != tt.expectedRate {
				t.Errorf("Expected success rate %.2f, got %.2f", tt.expectedRate, tool.SuccessRate)
			}
		})
	}
}
