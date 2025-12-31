package executor

import (
	"strings"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/behavioral"
	"github.com/stretchr/testify/assert"
)

func TestFormatBehaviorContext(t *testing.T) {
	tests := []struct {
		name     string
		metrics  *behavioral.BehavioralMetrics
		wantNil  bool
		contains []string
	}{
		{
			name:    "nil metrics returns empty string",
			metrics: nil,
			wantNil: true,
		},
		{
			name: "basic metrics formatted correctly",
			metrics: &behavioral.BehavioralMetrics{
				TotalSessions:   5,
				SuccessRate:     0.8,
				AverageDuration: 30 * time.Second,
			},
			contains: []string{
				"<behavioral_analysis>",
				"</behavioral_analysis>",
				`sessions="5"`,
				`success_rate="80.0%"`,
			},
		},
		{
			name: "metrics with tools formatted",
			metrics: &behavioral.BehavioralMetrics{
				TotalSessions:   1,
				SuccessRate:     1.0,
				AverageDuration: 10 * time.Second,
				ToolExecutions: []behavioral.ToolExecution{
					{Name: "Read", Count: 10, SuccessRate: 1.0, AvgDuration: 50 * time.Millisecond},
					{Name: "Write", Count: 5, SuccessRate: 0.8, AvgDuration: 100 * time.Millisecond},
				},
			},
			contains: []string{
				"<tool_usage>",
				`name="Read"`,
				`name="Write"`,
			},
		},
		{
			name: "metrics with cost",
			metrics: &behavioral.BehavioralMetrics{
				TotalSessions:   1,
				SuccessRate:     1.0,
				AverageDuration: 5 * time.Second,
				TotalCost:       0.05,
				TokenUsage: behavioral.TokenUsage{
					InputTokens:  1000,
					OutputTokens: 500,
					CostUSD:      0.05,
					ModelName:    "claude-sonnet-4-5",
				},
			},
			contains: []string{
				"<cost_summary>",
				"<tokens",
				"claude-sonnet-4-5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBehaviorContext(tt.metrics)

			if tt.wantNil {
				assert.Empty(t, result)
				return
			}

			for _, substr := range tt.contains {
				assert.Contains(t, result, substr)
			}
		})
	}
}

func TestIdentifyAnomalies(t *testing.T) {
	tests := []struct {
		name     string
		metrics  *behavioral.BehavioralMetrics
		expected []string
	}{
		{
			name:     "nil metrics returns nil",
			metrics:  nil,
			expected: nil,
		},
		{
			name: "no anomalies in healthy metrics",
			metrics: &behavioral.BehavioralMetrics{
				TotalSessions:   5,
				SuccessRate:     0.9,
				ErrorRate:       0.1,
				TotalErrors:     1,
				TotalCost:       0.5,
				AverageDuration: 1 * time.Minute,
			},
			expected: nil,
		},
		{
			name: "high error rate detected",
			metrics: &behavioral.BehavioralMetrics{
				TotalSessions: 5,
				ErrorRate:     0.3,
			},
			expected: []string{"High error rate"},
		},
		{
			name: "low success rate detected",
			metrics: &behavioral.BehavioralMetrics{
				TotalSessions: 5,
				SuccessRate:   0.5,
			},
			expected: []string{"Low success rate"},
		},
		{
			name: "high cost detected",
			metrics: &behavioral.BehavioralMetrics{
				TotalSessions: 1,
				TotalCost:     2.0,
			},
			expected: []string{"High cost"},
		},
		{
			name: "long duration detected",
			metrics: &behavioral.BehavioralMetrics{
				TotalSessions:   1,
				AverageDuration: 10 * time.Minute,
			},
			expected: []string{"Long execution"},
		},
		{
			name: "high tool error rate detected",
			metrics: &behavioral.BehavioralMetrics{
				TotalSessions: 1,
				ToolExecutions: []behavioral.ToolExecution{
					{Name: "Bash", Count: 5, ErrorRate: 0.5, TotalErrors: 2},
				},
			},
			expected: []string{"Tool 'Bash' high error rate"},
		},
		{
			name: "multiple bash failures detected",
			metrics: &behavioral.BehavioralMetrics{
				TotalSessions: 1,
				BashCommands: []behavioral.BashCommand{
					{Command: "cmd1", Success: false},
					{Command: "cmd2", Success: false},
					{Command: "cmd3", Success: false},
					{Command: "cmd4", Success: false},
				},
			},
			expected: []string{"Multiple bash failures"},
		},
		{
			name: "file operation issues detected",
			metrics: &behavioral.BehavioralMetrics{
				TotalSessions: 1,
				FileOperations: []behavioral.FileOperation{
					{Path: "file1.go", Success: false},
					{Path: "file2.go", Success: false},
					{Path: "file3.go", Success: false},
				},
			},
			expected: []string{"File operation issues"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anomalies := IdentifyAnomalies(tt.metrics)

			if tt.expected == nil {
				assert.Empty(t, anomalies)
				return
			}

			for _, expected := range tt.expected {
				found := false
				for _, anomaly := range anomalies {
					if strings.Contains(anomaly, expected) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected anomaly containing '%s' not found in %v", expected, anomalies)
			}
		})
	}
}

func TestSummarizeCost(t *testing.T) {
	tests := []struct {
		name     string
		metrics  *behavioral.BehavioralMetrics
		contains []string
	}{
		{
			name:     "nil metrics returns empty",
			metrics:  nil,
			contains: nil,
		},
		{
			name: "token usage formatted",
			metrics: &behavioral.BehavioralMetrics{
				TokenUsage: behavioral.TokenUsage{
					InputTokens:  5000,
					OutputTokens: 2000,
					CostUSD:      0.1,
					ModelName:    "claude-sonnet-4-5",
				},
			},
			contains: []string{"<cost_summary>", `input="5.0K"`, `output="2.0K"`, "$0.1000", "claude-sonnet-4-5"},
		},
		{
			name: "large token counts formatted with M suffix",
			metrics: &behavioral.BehavioralMetrics{
				TokenUsage: behavioral.TokenUsage{
					InputTokens:  1500000,
					OutputTokens: 500000,
				},
			},
			contains: []string{`input="1.50M"`, `output="500.0K"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SummarizeCost(tt.metrics)

			if tt.contains == nil {
				assert.Empty(t, result)
				return
			}

			for _, substr := range tt.contains {
				assert.Contains(t, result, substr)
			}
		})
	}
}

func TestSummarizeToolUsage(t *testing.T) {
	tests := []struct {
		name     string
		metrics  *behavioral.BehavioralMetrics
		contains []string
	}{
		{
			name:     "nil metrics returns empty",
			metrics:  nil,
			contains: nil,
		},
		{
			name: "empty tool executions returns empty",
			metrics: &behavioral.BehavioralMetrics{
				ToolExecutions: []behavioral.ToolExecution{},
			},
			contains: nil,
		},
		{
			name: "tools sorted by count",
			metrics: &behavioral.BehavioralMetrics{
				ToolExecutions: []behavioral.ToolExecution{
					{Name: "Write", Count: 3, SuccessRate: 1.0, AvgDuration: 100 * time.Millisecond},
					{Name: "Read", Count: 10, SuccessRate: 0.9, AvgDuration: 50 * time.Millisecond},
				},
			},
			contains: []string{"<tool_usage>", `name="Read" calls="10"`, `name="Write" calls="3"`},
		},
		{
			name: "success indicators shown",
			metrics: &behavioral.BehavioralMetrics{
				ToolExecutions: []behavioral.ToolExecution{
					{Name: "Read", Count: 5, SuccessRate: 1.0, AvgDuration: 50 * time.Millisecond},
					{Name: "Bash", Count: 5, SuccessRate: 0.5, AvgDuration: 200 * time.Millisecond},
				},
			},
			contains: []string{`status="ok"`, `status="issues"`},
		},
		{
			name: "max 5 tools shown with overflow indicator",
			metrics: &behavioral.BehavioralMetrics{
				ToolExecutions: []behavioral.ToolExecution{
					{Name: "Tool1", Count: 6, SuccessRate: 1.0, AvgDuration: 10 * time.Millisecond},
					{Name: "Tool2", Count: 5, SuccessRate: 1.0, AvgDuration: 10 * time.Millisecond},
					{Name: "Tool3", Count: 4, SuccessRate: 1.0, AvgDuration: 10 * time.Millisecond},
					{Name: "Tool4", Count: 3, SuccessRate: 1.0, AvgDuration: 10 * time.Millisecond},
					{Name: "Tool5", Count: 2, SuccessRate: 1.0, AvgDuration: 10 * time.Millisecond},
					{Name: "Tool6", Count: 1, SuccessRate: 1.0, AvgDuration: 10 * time.Millisecond},
				},
			},
			contains: []string{`<more_tools count="1"/>`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SummarizeToolUsage(tt.metrics)

			if tt.contains == nil {
				assert.Empty(t, result)
				return
			}

			for _, substr := range tt.contains {
				assert.Contains(t, result, substr)
			}
		})
	}
}

func TestBuildBehaviorPromptSection(t *testing.T) {
	tests := []struct {
		name     string
		metrics  *behavioral.BehavioralMetrics
		wantNil  bool
		contains []string
	}{
		{
			name:    "nil metrics returns empty",
			metrics: nil,
			wantNil: true,
		},
		{
			name: "includes consideration note",
			metrics: &behavioral.BehavioralMetrics{
				TotalSessions:   1,
				SuccessRate:     1.0,
				AverageDuration: 5 * time.Second,
			},
			contains: []string{"Consider these behavioral patterns"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildBehaviorPromptSection(tt.metrics)

			if tt.wantNil {
				assert.Empty(t, result)
				return
			}

			for _, substr := range tt.contains {
				assert.Contains(t, result, substr)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{500 * time.Millisecond, "500ms"},
		{1500 * time.Millisecond, "1.5s"},
		{90 * time.Second, "1.5m"},
		{90 * time.Minute, "1.5h"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		tokens   int64
		expected string
	}{
		{500, "500"},
		{1500, "1.5K"},
		{1500000, "1.50M"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatTokenCount(tt.tokens)
			assert.Equal(t, tt.expected, result)
		})
	}
}
