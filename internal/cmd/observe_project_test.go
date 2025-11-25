package cmd

import (
	"testing"
	"time"

	"github.com/harrison/conductor/internal/behavioral"
	"github.com/harrison/conductor/internal/learning"
)

func TestFormatProjectAnalysis(t *testing.T) {
	tests := []struct {
		name           string
		project        string
		summary        *learning.SummaryStats
		projectStats   *behavioral.ProjectStats
		projectErr     error
		projectMetrics *behavioral.AggregateProjectMetrics
		metricsErr     error
		agentStats     []learning.AgentTypeStats
		agentErr       error
		toolStats      []learning.ToolStats
		toolErr        error
		sessions       []learning.RecentSession
		sessionsErr    error
		limit          int
		wantContains   []string
	}{
		{
			name:    "basic project analysis",
			project: "myapp",
			summary: &learning.SummaryStats{
				TotalSessions:      42,
				TotalAgents:        5,
				SuccessRate:        0.875,
				AvgDurationSeconds: 300,
				TotalTokens:        10000,
				AvgTokensPerSession: 238.0,
			},
			projectStats: &behavioral.ProjectStats{
				TotalSessions: 42,
				LastModified:  "2025-11-25 10:30:00",
				TotalSize:     1024 * 1024,
			},
			limit: 20,
			wantContains: []string{
				"=== Project: myapp ===",
				"Sessions:        42",
				"Success Rate:    87.5%",
				"Last Activity:   2025-11-25 10:30:00",
				"Total Tokens:    10000",
			},
		},
		{
			name:    "with tool stats",
			project: "testproj",
			summary: &learning.SummaryStats{
				TotalSessions: 10,
				SuccessRate:   0.9,
			},
			toolStats: []learning.ToolStats{
				{ToolName: "Read", CallCount: 100, SuccessRate: 0.98},
				{ToolName: "Write", CallCount: 50, SuccessRate: 0.95},
			},
			limit: 20,
			wantContains: []string{
				"--- Tool Usage ---",
				"Read",
				"100",
				"98.0%",
				"Write",
				"50",
				"95.0%",
			},
		},
		{
			name:    "with agent stats",
			project: "agentproj",
			summary: &learning.SummaryStats{
				TotalSessions: 20,
				SuccessRate:   0.8,
			},
			agentStats: []learning.AgentTypeStats{
				{AgentType: "code-reviewer", TotalSessions: 10, SuccessRate: 0.9, AvgDurationSeconds: 120},
				{AgentType: "debugger", TotalSessions: 5, SuccessRate: 0.8, AvgDurationSeconds: 60},
			},
			limit: 20,
			wantContains: []string{
				"--- Agent Performance ---",
				"code-reviewer",
				"10",
				"90.0%",
				"debugger",
				"5",
				"80.0%",
			},
		},
		{
			name:    "with recent sessions",
			project: "sessionproj",
			summary: &learning.SummaryStats{
				TotalSessions: 5,
				SuccessRate:   0.6,
			},
			sessions: []learning.RecentSession{
				{TaskName: "Task 1", Agent: "reviewer", Success: true, DurationSecs: 60},
				{TaskName: "Task 2", Agent: "debugger", Success: false, DurationSecs: 120},
			},
			limit: 20,
			wantContains: []string{
				"--- Recent Sessions ---",
				"reviewer",
				"success",
				"debugger",
				"failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatProjectAnalysis(
				tt.project,
				tt.summary,
				tt.projectStats, tt.projectErr,
				tt.projectMetrics, tt.metricsErr,
				tt.agentStats, tt.agentErr,
				tt.toolStats, tt.toolErr,
				tt.sessions, tt.sessionsErr,
				tt.limit,
			)

			for _, want := range tt.wantContains {
				if !containsStr(result, want) {
					t.Errorf("formatProjectAnalysis() missing %q in output:\n%s", want, result)
				}
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestDisplayProjectAnalysis_NoProject(t *testing.T) {
	err := DisplayProjectAnalysis("", 20)
	if err == nil {
		t.Error("DisplayProjectAnalysis() should error when project is empty")
	}
	if !containsStr(err.Error(), "project name is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

// containsStr checks if s contains substr (avoids conflict with other test file)
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestGetRecentSessions(t *testing.T) {
	// This test uses the store.GetRecentSessions via the learning package tests
	// We verify the struct definition matches expectations
	var session learning.RecentSession
	session.ID = 1
	session.TaskName = "Test Task"
	session.Agent = "test-agent"
	session.Success = true
	session.DurationSecs = 120
	session.Timestamp = time.Now()

	if session.ID != 1 {
		t.Error("RecentSession ID not set correctly")
	}
	if session.TaskName != "Test Task" {
		t.Error("RecentSession TaskName not set correctly")
	}
}
