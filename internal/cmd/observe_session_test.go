package cmd

import (
	"os"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/behavioral"
	"github.com/harrison/conductor/internal/learning"
)

func TestDisplaySessionAnalysis_NoSessionID(t *testing.T) {
	err := DisplaySessionAnalysis("", "")
	if err == nil {
		t.Error("DisplaySessionAnalysis() should error when session ID is empty")
	}
	if !containsStr(err.Error(), "session ID is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFormatSessionAnalysis_Basic(t *testing.T) {
	sessionID := "abc123"
	info := &behavioral.SessionInfo{
		Project:   "myapp",
		Filename:  "agent-abc123.jsonl",
		SessionID: "abc123",
		CreatedAt: time.Date(2025, 11, 25, 10, 30, 0, 0, time.UTC),
		FileSize:  1024,
		FilePath:  "/tmp/test/agent-abc123.jsonl",
	}
	metrics := &behavioral.BehavioralMetrics{
		TotalSessions:   1,
		SuccessRate:     0.9,
		AverageDuration: 5 * time.Minute,
		TokenUsage: behavioral.TokenUsage{
			InputTokens:  10000,
			OutputTokens: 5000,
			CostUSD:      0.42,
		},
	}

	result := formatSessionAnalysis(sessionID, "myapp", info, metrics, nil)

	wantContains := []string{
		"=== Session: abc123 ===",
		"Project:         myapp",
		"File:            agent-abc123.jsonl",
		"Duration:        5.0m",
		"Success Rate:    90.0%",
		"--- Token Usage ---",
		"Input Tokens:    10,000",
		"Output Tokens:   5,000",
		"Total:           15,000",
		"Cost:            $0.4200",
	}

	for _, want := range wantContains {
		if !containsStr(result, want) {
			t.Errorf("formatSessionAnalysis() missing %q in output:\n%s", want, result)
		}
	}
}

func TestFormatSessionAnalysis_WithTimeline(t *testing.T) {
	sessionID := "test456"
	ts := time.Date(2025, 11, 25, 10, 30, 0, 0, time.UTC)

	sessionData := &behavioral.SessionData{
		Session: behavioral.Session{
			ID:      "test456",
			Project: "testproj",
		},
		Events: []behavioral.Event{
			&behavioral.ToolCallEvent{
				BaseEvent: behavioral.BaseEvent{
					Type:      "tool_call",
					Timestamp: ts,
				},
				ToolName: "Read",
				Success:  true,
				Duration: 120,
			},
			&behavioral.BashCommandEvent{
				BaseEvent: behavioral.BaseEvent{
					Type:      "bash_command",
					Timestamp: ts.Add(5 * time.Second),
				},
				Command:  "go test ./...",
				Success:  false,
				ExitCode: 1,
				Duration: 2300,
			},
			&behavioral.FileOperationEvent{
				BaseEvent: behavioral.BaseEvent{
					Type:      "file_operation",
					Timestamp: ts.Add(10 * time.Second),
				},
				Operation: "Write",
				Path:      "src/main.go",
				Success:   true,
				Duration:  85,
			},
		},
	}

	result := formatSessionAnalysis(sessionID, "testproj", nil, nil, sessionData)

	wantContains := []string{
		"=== Session: test456 ===",
		"--- Tool Timeline ---",
		"Read",
		"success",
		"Bash",
		"failed",
		"--- Errors (1) ---",
		"exit code 1",
	}

	for _, want := range wantContains {
		if !containsStr(result, want) {
			t.Errorf("formatSessionAnalysis() missing %q in output:\n%s", want, result)
		}
	}
}

func TestFormatSessionAnalysis_WithToolUsage(t *testing.T) {
	sessionID := "xyz789"
	metrics := &behavioral.BehavioralMetrics{
		TotalSessions:   1,
		SuccessRate:     0.8,
		AverageDuration: 2 * time.Minute,
		TotalErrors:     2,
		ErrorRate:       0.2,
		ToolExecutions: []behavioral.ToolExecution{
			{Name: "Read", Count: 10, SuccessRate: 1.0},
			{Name: "Write", Count: 5, SuccessRate: 0.8},
		},
	}

	result := formatSessionAnalysis(sessionID, "", nil, metrics, nil)

	wantContains := []string{
		"=== Session: xyz789 ===",
		"--- Tool Usage ---",
		"Read",
		"10",
		"100.0%",
		"Write",
		"5",
		"80.0%",
		"--- Errors (2) ---",
	}

	for _, want := range wantContains {
		if !containsStr(result, want) {
			t.Errorf("formatSessionAnalysis() missing %q in output:\n%s", want, result)
		}
	}
}

func TestFormatDBSessionAnalysis(t *testing.T) {
	ts := time.Date(2025, 11, 25, 10, 30, 0, 0, time.UTC)
	endTs := ts.Add(5 * time.Minute)

	metrics := &learning.BehavioralSessionMetrics{
		SessionID:           1,
		TaskExecutionID:     42,
		SessionStart:        ts,
		SessionEnd:          &endTs,
		TotalDurationSecs:   300,
		TotalToolCalls:      25,
		TotalBashCommands:   10,
		TotalFileOperations: 15,
		TotalTokensUsed:     10000,
		ContextWindowUsed:   25,
		TaskNumber:          "3",
		TaskName:            "Implement feature",
		Agent:               "code-reviewer",
		Success:             true,
	}

	result := formatDBSessionAnalysis(metrics)

	wantContains := []string{
		"=== Session: 1 ===",
		"Task Number:     3",
		"Task Name:       Implement feature",
		"Agent:           code-reviewer",
		"Status:          success",
		"Duration:        5.0m",
		"--- Activity ---",
		"Tool Calls:      25",
		"Bash Commands:   10",
		"File Operations: 15",
		"--- Token Usage ---",
		"Total Tokens:    10,000",
		"Context Window:  25%",
	}

	for _, want := range wantContains {
		if !containsStr(result, want) {
			t.Errorf("formatDBSessionAnalysis() missing %q in output:\n%s", want, result)
		}
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{12345, "12,345"},
		{123456, "123,456"},
		{1234567, "1,234,567"},
		{12345678, "12,345,678"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatNumber(tt.n)
			if got != tt.want {
				t.Errorf("formatNumber(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestFormatMillis(t *testing.T) {
	tests := []struct {
		ms   int64
		want string
	}{
		{100, "100ms"},
		{999, "999ms"},
		{1000, "1.0s"},
		{1500, "1.5s"},
		{59999, "60.0s"},
		{60000, "1.0m"},
		{90000, "1.5m"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatMillis(tt.ms)
			if got != tt.want {
				t.Errorf("formatMillis(%d) = %q, want %q", tt.ms, got, tt.want)
			}
		})
	}
}

func TestNewObserveSessionCmd(t *testing.T) {
	// Set up temp directory with config for test
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)
	os.MkdirAll(".conductor", 0755)
	os.WriteFile(".conductor/config.yaml", []byte("learning:\n  db_path: .conductor/learning/test.db\n"), 0644)

	cmd := NewObserveSessionCmd()

	if cmd.Use != "session [session-id]" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Short != "Analyze a specific agent session or list recent sessions" {
		t.Errorf("unexpected Short: %s", cmd.Short)
	}

	// Test that no session ID now lists recent sessions (no error)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected success when no session ID provided (lists sessions), got error: %v", err)
	}
}
