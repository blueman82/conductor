package cmd

import (
	"testing"
	"time"

	"github.com/harrison/conductor/internal/behavioral"
)

func TestDisplayToolExecutions(t *testing.T) {
	tools := []behavioral.ToolExecution{
		{
			Name:         "Read",
			Count:        100,
			SuccessRate:  0.95,
			ErrorRate:    0.05,
			AvgDuration:  100 * time.Millisecond,
			TotalSuccess: 95,
			TotalErrors:  5,
		},
	}

	err := DisplayToolExecutions(tools, 50)
	if err != nil {
		t.Errorf("DisplayToolExecutions() failed: %v", err)
	}
}

func TestDisplayBashCommands(t *testing.T) {
	commands := []behavioral.BashCommand{
		{
			Command:      "ls -la",
			ExitCode:     0,
			OutputLength: 1024,
			Duration:     50 * time.Millisecond,
			Success:      true,
			Timestamp:    time.Now(),
		},
	}

	err := DisplayBashCommands(commands, 50)
	if err != nil {
		t.Errorf("DisplayBashCommands() failed: %v", err)
	}
}

func TestDisplayFileOperations(t *testing.T) {
	ops := []behavioral.FileOperation{
		{
			Type:      "read",
			Path:      "/path/to/file.go",
			SizeBytes: 2048,
			Success:   true,
			Timestamp: time.Now(),
			Duration:  10,
		},
	}

	err := DisplayFileOperations(ops, 50)
	if err != nil {
		t.Errorf("DisplayFileOperations() failed: %v", err)
	}
}

func TestDisplaySessions(t *testing.T) {
	sessions := []behavioral.Session{
		{
			ID:         "session-001",
			Project:    "conductor",
			Timestamp:  time.Now(),
			Status:     "completed",
			Duration:   5000,
			Success:    true,
			ErrorCount: 0,
		},
	}

	err := DisplaySessions(sessions, 50)
	if err != nil {
		t.Errorf("DisplaySessions() failed: %v", err)
	}
}

func TestDisplayPaginatedResults_Empty(t *testing.T) {
	items := []interface{}{}
	err := DisplayPaginatedResults(items, 50)
	if err != nil {
		t.Errorf("DisplayPaginatedResults() with empty items failed: %v", err)
	}
}

func TestDisplayPaginatedResults_SinglePage(t *testing.T) {
	tools := []behavioral.ToolExecution{
		{Name: "Read", Count: 10},
		{Name: "Write", Count: 5},
	}

	items := make([]interface{}, len(tools))
	for i, tool := range tools {
		items[i] = tool
	}

	err := DisplayPaginatedResults(items, 50)
	if err != nil {
		t.Errorf("DisplayPaginatedResults() failed: %v", err)
	}
}

func TestDisplayPaginatedResults_MultiplePages(t *testing.T) {
	// Create 60 items to test pagination (should be 2 pages with pageSize=50)
	tools := make([]behavioral.ToolExecution, 60)
	for i := 0; i < 60; i++ {
		tools[i] = behavioral.ToolExecution{
			Name:  "Tool" + string(rune('A'+i%26)),
			Count: i + 1,
		}
	}

	items := make([]interface{}, len(tools))
	for i, tool := range tools {
		items[i] = tool
	}

	err := DisplayPaginatedResults(items, 50)
	if err != nil {
		t.Errorf("DisplayPaginatedResults() with multiple pages failed: %v", err)
	}
}

func TestDisplayMetricsSummary(t *testing.T) {
	metrics := &behavioral.BehavioralMetrics{
		TotalSessions:   10,
		SuccessRate:     0.8,
		AverageDuration: 5 * time.Minute,
		TotalCost:       1.5,
		ErrorRate:       0.2,
		TotalErrors:     2,
		ToolExecutions: []behavioral.ToolExecution{
			{Name: "Read", Count: 100},
		},
		BashCommands: []behavioral.BashCommand{
			{Command: "ls", ExitCode: 0, Success: true},
		},
		FileOperations: []behavioral.FileOperation{
			{Type: "read", Path: "/test", Success: true},
		},
		AgentPerformance: map[string]int{
			"agent1": 5,
			"agent2": 3,
		},
	}

	err := DisplayMetricsSummary(metrics)
	if err != nil {
		t.Errorf("DisplayMetricsSummary() failed: %v", err)
	}
}

func TestDisplayMetricsSummary_Empty(t *testing.T) {
	metrics := &behavioral.BehavioralMetrics{
		TotalSessions: 0,
	}

	err := DisplayMetricsSummary(metrics)
	if err != nil {
		t.Errorf("DisplayMetricsSummary() with empty metrics failed: %v", err)
	}
}

func TestPrintTableHeader(t *testing.T) {
	// printTableHeader currently returns empty string
	// This test ensures it doesn't panic
	header := printTableHeader()
	if header != "" {
		t.Errorf("Expected empty header, got: %s", header)
	}
}
