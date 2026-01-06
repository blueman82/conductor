package models

import (
	"encoding/json"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestExecutionResult_StatusBreakdown(t *testing.T) {
	tests := []struct {
		name         string
		results      []TaskResult
		expectGreen  int
		expectYellow int
		expectRed    int
	}{
		{
			name: "mixed task statuses",
			results: []TaskResult{
				{Status: StatusGreen, Task: Task{Number: "1", Name: "Task 1", Prompt: "test"}},
				{Status: StatusGreen, Task: Task{Number: "2", Name: "Task 2", Prompt: "test"}},
				{Status: StatusGreen, Task: Task{Number: "3", Name: "Task 3", Prompt: "test"}},
				{Status: StatusGreen, Task: Task{Number: "4", Name: "Task 4", Prompt: "test"}},
				{Status: StatusGreen, Task: Task{Number: "5", Name: "Task 5", Prompt: "test"}},
				{Status: StatusGreen, Task: Task{Number: "6", Name: "Task 6", Prompt: "test"}},
				{Status: StatusGreen, Task: Task{Number: "7", Name: "Task 7", Prompt: "test"}},
				{Status: StatusGreen, Task: Task{Number: "8", Name: "Task 8", Prompt: "test"}},
				{Status: StatusYellow, Task: Task{Number: "9", Name: "Task 9", Prompt: "test"}},
				{Status: StatusYellow, Task: Task{Number: "10", Name: "Task 10", Prompt: "test"}},
			},
			expectGreen:  8,
			expectYellow: 2,
			expectRed:    0,
		},
		{
			name: "all green",
			results: []TaskResult{
				{Status: StatusGreen, Task: Task{Number: "1", Name: "Task 1", Prompt: "test"}},
				{Status: StatusGreen, Task: Task{Number: "2", Name: "Task 2", Prompt: "test"}},
			},
			expectGreen:  2,
			expectYellow: 0,
			expectRed:    0,
		},
		{
			name: "all red",
			results: []TaskResult{
				{Status: StatusRed, Task: Task{Number: "1", Name: "Task 1", Prompt: "test"}},
				{Status: StatusRed, Task: Task{Number: "2", Name: "Task 2", Prompt: "test"}},
			},
			expectGreen:  0,
			expectYellow: 0,
			expectRed:    2,
		},
		{
			name:         "empty results",
			results:      []TaskResult{},
			expectGreen:  0,
			expectYellow: 0,
			expectRed:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewExecutionResult(tt.results, true, 1*time.Minute)

			if result.StatusBreakdown[StatusGreen] != tt.expectGreen {
				t.Errorf("StatusBreakdown[GREEN] = %v, want %v", result.StatusBreakdown[StatusGreen], tt.expectGreen)
			}
			if result.StatusBreakdown[StatusYellow] != tt.expectYellow {
				t.Errorf("StatusBreakdown[YELLOW] = %v, want %v", result.StatusBreakdown[StatusYellow], tt.expectYellow)
			}
			if result.StatusBreakdown[StatusRed] != tt.expectRed {
				t.Errorf("StatusBreakdown[RED] = %v, want %v", result.StatusBreakdown[StatusRed], tt.expectRed)
			}

			// Verify zero values are included
			if _, hasGreen := result.StatusBreakdown[StatusGreen]; !hasGreen {
				t.Error("StatusBreakdown should include GREEN key even if count is 0")
			}
			if _, hasYellow := result.StatusBreakdown[StatusYellow]; !hasYellow {
				t.Error("StatusBreakdown should include YELLOW key even if count is 0")
			}
			if _, hasRed := result.StatusBreakdown[StatusRed]; !hasRed {
				t.Error("StatusBreakdown should include RED key even if count is 0")
			}
		})
	}
}

func TestExecutionResult_AgentUsage(t *testing.T) {
	tests := []struct {
		name         string
		results      []TaskResult
		expectAgents map[string]int
	}{
		{
			name: "multiple agents with different usage",
			results: []TaskResult{
				{Task: Task{Number: "1", Name: "T1", Prompt: "test", Agent: "golang-pro"}},
				{Task: Task{Number: "2", Name: "T2", Prompt: "test", Agent: "golang-pro"}},
				{Task: Task{Number: "3", Name: "T3", Prompt: "test", Agent: "golang-pro"}},
				{Task: Task{Number: "4", Name: "T4", Prompt: "test", Agent: "golang-pro"}},
				{Task: Task{Number: "5", Name: "T5", Prompt: "test", Agent: "golang-pro"}},
				{Task: Task{Number: "6", Name: "T6", Prompt: "test", Agent: "devops"}},
				{Task: Task{Number: "7", Name: "T7", Prompt: "test", Agent: "devops"}},
				{Task: Task{Number: "8", Name: "T8", Prompt: "test", Agent: "devops"}},
				{Task: Task{Number: "9", Name: "T9", Prompt: "test", Agent: "quality-control"}},
				{Task: Task{Number: "10", Name: "T10", Prompt: "test", Agent: "quality-control"}},
			},
			expectAgents: map[string]int{
				"golang-pro":      5,
				"devops":          3,
				"quality-control": 2,
			},
		},
		{
			name: "single agent",
			results: []TaskResult{
				{Task: Task{Number: "1", Name: "T1", Prompt: "test", Agent: "python-pro"}},
				{Task: Task{Number: "2", Name: "T2", Prompt: "test", Agent: "python-pro"}},
			},
			expectAgents: map[string]int{
				"python-pro": 2,
			},
		},
		{
			name: "tasks with no agent specified",
			results: []TaskResult{
				{Task: Task{Number: "1", Name: "T1", Prompt: "test", Agent: ""}},
				{Task: Task{Number: "2", Name: "T2", Prompt: "test", Agent: ""}},
				{Task: Task{Number: "3", Name: "T3", Prompt: "test", Agent: "golang-pro"}},
			},
			expectAgents: map[string]int{
				"":           2,
				"golang-pro": 1,
			},
		},
		{
			name:         "empty results",
			results:      []TaskResult{},
			expectAgents: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewExecutionResult(tt.results, true, 1*time.Minute)

			if len(result.AgentUsage) != len(tt.expectAgents) {
				t.Errorf("AgentUsage length = %v, want %v", len(result.AgentUsage), len(tt.expectAgents))
			}

			for agent, expectCount := range tt.expectAgents {
				if result.AgentUsage[agent] != expectCount {
					t.Errorf("AgentUsage[%q] = %v, want %v", agent, result.AgentUsage[agent], expectCount)
				}
			}
		})
	}
}

func TestExecutionResult_TotalFiles(t *testing.T) {
	tests := []struct {
		name        string
		results     []TaskResult
		expectTotal int
	}{
		{
			name: "unique files across tasks",
			results: []TaskResult{
				{Task: Task{Number: "1", Name: "T1", Prompt: "test", Files: []string{"file1.go", "file2.go"}}},
				{Task: Task{Number: "2", Name: "T2", Prompt: "test", Files: []string{"file2.go", "file3.go"}}},
				{Task: Task{Number: "3", Name: "T3", Prompt: "test", Files: []string{"file4.go"}}},
			},
			expectTotal: 4, // file1.go, file2.go, file3.go, file4.go
		},
		{
			name: "same files in multiple tasks",
			results: []TaskResult{
				{Task: Task{Number: "1", Name: "T1", Prompt: "test", Files: []string{"file1.go"}}},
				{Task: Task{Number: "2", Name: "T2", Prompt: "test", Files: []string{"file1.go"}}},
				{Task: Task{Number: "3", Name: "T3", Prompt: "test", Files: []string{"file1.go"}}},
			},
			expectTotal: 1, // Only counted once
		},
		{
			name: "mixed files with deduplication",
			results: []TaskResult{
				{Task: Task{Number: "1", Name: "T1", Prompt: "test", Files: []string{"api.go", "handler.go", "utils.go"}}},
				{Task: Task{Number: "2", Name: "T2", Prompt: "test", Files: []string{"utils.go", "config.go"}}},
				{Task: Task{Number: "3", Name: "T3", Prompt: "test", Files: []string{"handler.go", "middleware.go"}}},
			},
			expectTotal: 5, // api.go, handler.go, utils.go, config.go, middleware.go
		},
		{
			name: "empty file lists",
			results: []TaskResult{
				{Task: Task{Number: "1", Name: "T1", Prompt: "test", Files: []string{}}},
				{Task: Task{Number: "2", Name: "T2", Prompt: "test", Files: []string{}}},
			},
			expectTotal: 0,
		},
		{
			name:        "no results",
			results:     []TaskResult{},
			expectTotal: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewExecutionResult(tt.results, true, 1*time.Minute)

			if result.TotalFiles != tt.expectTotal {
				t.Errorf("TotalFiles = %v, want %v", result.TotalFiles, tt.expectTotal)
			}
		})
	}
}

func TestExecutionResult_LOC(t *testing.T) {
	tests := []struct {
		name                string
		results             []TaskResult
		expectTotalAdded    int
		expectTotalDeleted  int
	}{
		{
			name: "aggregate LOC across tasks",
			results: []TaskResult{
				{Task: Task{Number: "1", Name: "T1", Prompt: "test", LinesAdded: 100, LinesDeleted: 20}},
				{Task: Task{Number: "2", Name: "T2", Prompt: "test", LinesAdded: 50, LinesDeleted: 30}},
				{Task: Task{Number: "3", Name: "T3", Prompt: "test", LinesAdded: 75, LinesDeleted: 10}},
			},
			expectTotalAdded:   225, // 100 + 50 + 75
			expectTotalDeleted: 60,  // 20 + 30 + 10
		},
		{
			name: "tasks with zero LOC",
			results: []TaskResult{
				{Task: Task{Number: "1", Name: "T1", Prompt: "test", LinesAdded: 0, LinesDeleted: 0}},
				{Task: Task{Number: "2", Name: "T2", Prompt: "test", LinesAdded: 0, LinesDeleted: 0}},
			},
			expectTotalAdded:   0,
			expectTotalDeleted: 0,
		},
		{
			name: "mixed LOC values",
			results: []TaskResult{
				{Task: Task{Number: "1", Name: "T1", Prompt: "test", LinesAdded: 500, LinesDeleted: 0}},
				{Task: Task{Number: "2", Name: "T2", Prompt: "test", LinesAdded: 0, LinesDeleted: 200}},
				{Task: Task{Number: "3", Name: "T3", Prompt: "test", LinesAdded: 100, LinesDeleted: 100}},
			},
			expectTotalAdded:   600, // 500 + 0 + 100
			expectTotalDeleted: 300, // 0 + 200 + 100
		},
		{
			name:                "empty results",
			results:             []TaskResult{},
			expectTotalAdded:    0,
			expectTotalDeleted:  0,
		},
		{
			name: "single task with LOC",
			results: []TaskResult{
				{Task: Task{Number: "1", Name: "T1", Prompt: "test", LinesAdded: 42, LinesDeleted: 17}},
			},
			expectTotalAdded:   42,
			expectTotalDeleted: 17,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewExecutionResult(tt.results, true, 1*time.Minute)

			if result.TotalLinesAdded != tt.expectTotalAdded {
				t.Errorf("TotalLinesAdded = %v, want %v", result.TotalLinesAdded, tt.expectTotalAdded)
			}
			if result.TotalLinesDeleted != tt.expectTotalDeleted {
				t.Errorf("TotalLinesDeleted = %v, want %v", result.TotalLinesDeleted, tt.expectTotalDeleted)
			}
		})
	}
}

func TestExecutionResult_AvgTaskDuration(t *testing.T) {
	tests := []struct {
		name         string
		results      []TaskResult
		expectAvgDur time.Duration
	}{
		{
			name: "multiple tasks with varying durations",
			results: []TaskResult{
				{Status: StatusGreen, Duration: 2 * time.Second, Task: Task{Number: "1", Name: "T1", Prompt: "test"}},
				{Status: StatusGreen, Duration: 4 * time.Second, Task: Task{Number: "2", Name: "T2", Prompt: "test"}},
				{Status: StatusGreen, Duration: 6 * time.Second, Task: Task{Number: "3", Name: "T3", Prompt: "test"}},
				{Status: StatusGreen, Duration: 8 * time.Second, Task: Task{Number: "4", Name: "T4", Prompt: "test"}},
			},
			expectAvgDur: 5 * time.Second, // (2+4+6+8) / 4 = 20 / 4 = 5
		},
		{
			name: "single task",
			results: []TaskResult{
				{Status: StatusGreen, Duration: 10 * time.Second, Task: Task{Number: "1", Name: "T1", Prompt: "test"}},
			},
			expectAvgDur: 10 * time.Second,
		},
		{
			name: "zero durations",
			results: []TaskResult{
				{Status: StatusGreen, Duration: 0, Task: Task{Number: "1", Name: "T1", Prompt: "test"}},
				{Status: StatusGreen, Duration: 0, Task: Task{Number: "2", Name: "T2", Prompt: "test"}},
			},
			expectAvgDur: 0,
		},
		{
			name:         "empty results",
			results:      []TaskResult{},
			expectAvgDur: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewExecutionResult(tt.results, true, 1*time.Minute)

			if result.AvgTaskDuration != tt.expectAvgDur {
				t.Errorf("AvgTaskDuration = %v, want %v", result.AvgTaskDuration, tt.expectAvgDur)
			}
		})
	}
}

func TestExecutionResult_CalculateMetrics(t *testing.T) {
	tests := []struct {
		name           string
		results        []TaskResult
		validateMetric func(*ExecutionResult) error
	}{
		{
			name: "realistic task data",
			results: []TaskResult{
				{
					Status:   StatusGreen,
					Duration: 3 * time.Second,
					Task:     Task{Number: "1", Name: "T1", Prompt: "test", Agent: "golang-pro", Files: []string{"main.go", "config.go"}},
				},
				{
					Status:   StatusGreen,
					Duration: 5 * time.Second,
					Task:     Task{Number: "2", Name: "T2", Prompt: "test", Agent: "golang-pro", Files: []string{"main.go", "utils.go"}},
				},
				{
					Status:   StatusYellow,
					Duration: 2 * time.Second,
					Task:     Task{Number: "3", Name: "T3", Prompt: "test", Agent: "devops", Files: []string{"docker.yaml"}},
				},
			},
			validateMetric: func(er *ExecutionResult) error {
				if er.TotalTasks != 3 {
					t.Errorf("TotalTasks = %v, want 3", er.TotalTasks)
				}
				if er.StatusBreakdown[StatusGreen] != 2 {
					t.Errorf("StatusBreakdown[GREEN] = %v, want 2", er.StatusBreakdown[StatusGreen])
				}
				if er.AgentUsage["golang-pro"] != 2 {
					t.Errorf("AgentUsage[golang-pro] = %v, want 2", er.AgentUsage["golang-pro"])
				}
				if er.TotalFiles != 4 {
					t.Errorf("TotalFiles = %v, want 4", er.TotalFiles)
				}
				// Average: (3s + 5s + 2s) / 3 = 10/3 = 3.333...s
				expectedAvg := time.Duration((3 + 5 + 2) * 1000000000 / 3)
				if er.AvgTaskDuration != expectedAvg {
					t.Errorf("AvgTaskDuration = %v, want %v", er.AvgTaskDuration, expectedAvg)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewExecutionResult(tt.results, true, 1*time.Minute)
			if err := tt.validateMetric(result); err != nil {
				t.Errorf("metric validation failed: %v", err)
			}
		})
	}
}

func TestExecutionResult_JSONSerialization(t *testing.T) {
	t.Run("JSON marshaling and unmarshaling", func(t *testing.T) {
		results := []TaskResult{
			{
				Status:   StatusGreen,
				Duration: 3 * time.Second,
				Task:     Task{Number: "1", Name: "T1", Prompt: "test", Agent: "golang-pro", Files: []string{"file1.go"}, LinesAdded: 100, LinesDeleted: 25},
			},
			{
				Status:   StatusYellow,
				Duration: 2 * time.Second,
				Task:     Task{Number: "2", Name: "T2", Prompt: "test", Agent: "devops", Files: []string{"file2.yaml"}, LinesAdded: 50, LinesDeleted: 10},
			},
		}

		original := NewExecutionResult(results, true, 10*time.Second)

		// Marshal to JSON
		jsonData, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("JSON marshal failed: %v", err)
		}

		// Unmarshal back
		var unmarshaled ExecutionResult
		err = json.Unmarshal(jsonData, &unmarshaled)
		if err != nil {
			t.Fatalf("JSON unmarshal failed: %v", err)
		}

		// Verify key fields are preserved
		if unmarshaled.TotalTasks != original.TotalTasks {
			t.Errorf("TotalTasks = %v, want %v", unmarshaled.TotalTasks, original.TotalTasks)
		}
		if unmarshaled.StatusBreakdown[StatusGreen] != original.StatusBreakdown[StatusGreen] {
			t.Errorf("StatusBreakdown[GREEN] mismatch after JSON round-trip")
		}
		if len(unmarshaled.AgentUsage) != len(original.AgentUsage) {
			t.Errorf("AgentUsage length mismatch after JSON round-trip")
		}
		if unmarshaled.TotalFiles != original.TotalFiles {
			t.Errorf("TotalFiles = %v, want %v", unmarshaled.TotalFiles, original.TotalFiles)
		}
		if unmarshaled.TotalLinesAdded != original.TotalLinesAdded {
			t.Errorf("TotalLinesAdded = %v, want %v", unmarshaled.TotalLinesAdded, original.TotalLinesAdded)
		}
		if unmarshaled.TotalLinesDeleted != original.TotalLinesDeleted {
			t.Errorf("TotalLinesDeleted = %v, want %v", unmarshaled.TotalLinesDeleted, original.TotalLinesDeleted)
		}
	})
}

func TestExecutionResult_YAMLSerialization(t *testing.T) {
	t.Run("YAML marshaling and unmarshaling", func(t *testing.T) {
		results := []TaskResult{
			{
				Status:   StatusGreen,
				Duration: 3 * time.Second,
				Task:     Task{Number: "1", Name: "T1", Prompt: "test", Agent: "golang-pro", Files: []string{"file1.go"}, LinesAdded: 100, LinesDeleted: 25},
			},
			{
				Status:   StatusYellow,
				Duration: 2 * time.Second,
				Task:     Task{Number: "2", Name: "T2", Prompt: "test", Agent: "devops", Files: []string{"file2.yaml"}, LinesAdded: 50, LinesDeleted: 10},
			},
		}

		original := NewExecutionResult(results, true, 10*time.Second)

		// Marshal to YAML
		yamlData, err := yaml.Marshal(original)
		if err != nil {
			t.Fatalf("YAML marshal failed: %v", err)
		}

		// Unmarshal back
		var unmarshaled ExecutionResult
		err = yaml.Unmarshal(yamlData, &unmarshaled)
		if err != nil {
			t.Fatalf("YAML unmarshal failed: %v", err)
		}

		// Verify key fields are preserved
		if unmarshaled.TotalTasks != original.TotalTasks {
			t.Errorf("TotalTasks = %v, want %v", unmarshaled.TotalTasks, original.TotalTasks)
		}
		if unmarshaled.StatusBreakdown[StatusGreen] != original.StatusBreakdown[StatusGreen] {
			t.Errorf("StatusBreakdown[GREEN] mismatch after YAML round-trip")
		}
		if len(unmarshaled.AgentUsage) != len(original.AgentUsage) {
			t.Errorf("AgentUsage length mismatch after YAML round-trip")
		}
		if unmarshaled.TotalFiles != original.TotalFiles {
			t.Errorf("TotalFiles = %v, want %v", unmarshaled.TotalFiles, original.TotalFiles)
		}
		if unmarshaled.TotalLinesAdded != original.TotalLinesAdded {
			t.Errorf("TotalLinesAdded = %v, want %v", unmarshaled.TotalLinesAdded, original.TotalLinesAdded)
		}
		if unmarshaled.TotalLinesDeleted != original.TotalLinesDeleted {
			t.Errorf("TotalLinesDeleted = %v, want %v", unmarshaled.TotalLinesDeleted, original.TotalLinesDeleted)
		}
	})
}

func TestExecutionResult_CalculateMetricsMethod(t *testing.T) {
	t.Run("CalculateMetrics updates existing result", func(t *testing.T) {
		// Create a result with initial empty values
		result := &ExecutionResult{
			TotalTasks: 0,
			Duration:   0,
		}

		// Call CalculateMetrics with task data
		results := []TaskResult{
			{
				Status:   StatusGreen,
				Duration: 2 * time.Second,
				Task:     Task{Number: "1", Name: "T1", Prompt: "test", Agent: "golang-pro", Files: []string{"file1.go"}},
			},
			{
				Status:   StatusGreen,
				Duration: 4 * time.Second,
				Task:     Task{Number: "2", Name: "T2", Prompt: "test", Agent: "devops", Files: []string{"file2.yaml"}},
			},
		}

		result.CalculateMetrics(results)

		// Verify all metrics are calculated
		if result.Completed != 2 {
			t.Errorf("Completed = %v, want 2", result.Completed)
		}
		if result.StatusBreakdown[StatusGreen] != 2 {
			t.Errorf("StatusBreakdown[GREEN] = %v, want 2", result.StatusBreakdown[StatusGreen])
		}
		if result.AgentUsage["golang-pro"] != 1 {
			t.Errorf("AgentUsage[golang-pro] = %v, want 1", result.AgentUsage["golang-pro"])
		}
		if result.AgentUsage["devops"] != 1 {
			t.Errorf("AgentUsage[devops] = %v, want 1", result.AgentUsage["devops"])
		}
		if result.TotalFiles != 2 {
			t.Errorf("TotalFiles = %v, want 2", result.TotalFiles)
		}
		if result.AvgTaskDuration != 3*time.Second {
			t.Errorf("AvgTaskDuration = %v, want 3s", result.AvgTaskDuration)
		}
	})
}

func TestExecutionResult_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		results []TaskResult
	}{
		{
			name:    "empty task list",
			results: []TaskResult{},
		},
		{
			name: "tasks with nil files",
			results: []TaskResult{
				{Status: StatusGreen, Duration: 1 * time.Second, Task: Task{Number: "1", Name: "T1", Prompt: "test", Files: nil}},
			},
		},
		{
			name: "tasks with empty files",
			results: []TaskResult{
				{Status: StatusGreen, Duration: 1 * time.Second, Task: Task{Number: "1", Name: "T1", Prompt: "test", Files: []string{}}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			result := NewExecutionResult(tt.results, true, 1*time.Minute)
			if result == nil {
				t.Error("NewExecutionResult returned nil")
			}
		})
	}
}

// TestExecutionResult_MetricConsistency verifies that NewExecutionResult and CalculateMetrics
// produce identical metric calculations from the same input data.
// This test ensures DRY refactoring doesn't introduce behavioral differences.
func TestExecutionResult_MetricConsistency(t *testing.T) {
	tests := []struct {
		name    string
		results []TaskResult
	}{
		{
			name: "mixed statuses and agents",
			results: []TaskResult{
				{
					Status:   StatusGreen,
					Duration: 2 * time.Second,
					Task:     Task{Number: "1", Name: "T1", Prompt: "test", Agent: "golang-pro", Files: []string{"file1.go", "file2.go"}},
				},
				{
					Status:   StatusGreen,
					Duration: 3 * time.Second,
					Task:     Task{Number: "2", Name: "T2", Prompt: "test", Agent: "golang-pro", Files: []string{"file2.go", "file3.go"}},
				},
				{
					Status:   StatusYellow,
					Duration: 4 * time.Second,
					Task:     Task{Number: "3", Name: "T3", Prompt: "test", Agent: "devops", Files: []string{"file4.yaml"}},
				},
				{
					Status:   StatusRed,
					Duration: 1 * time.Second,
					Task:     Task{Number: "4", Name: "T4", Prompt: "test", Agent: "golang-pro", Files: []string{"file5.go"}},
				},
			},
		},
		{
			name: "empty agents",
			results: []TaskResult{
				{Status: StatusGreen, Duration: 1 * time.Second, Task: Task{Number: "1", Name: "T1", Prompt: "test", Agent: "", Files: []string{"file1.go"}}},
				{Status: StatusGreen, Duration: 2 * time.Second, Task: Task{Number: "2", Name: "T2", Prompt: "test", Agent: "", Files: []string{"file2.go"}}},
			},
		},
		{
			name: "all same status",
			results: []TaskResult{
				{Status: StatusGreen, Duration: 1 * time.Second, Task: Task{Number: "1", Name: "T1", Prompt: "test", Agent: "agent1"}},
				{Status: StatusGreen, Duration: 2 * time.Second, Task: Task{Number: "2", Name: "T2", Prompt: "test", Agent: "agent1"}},
				{Status: StatusGreen, Duration: 3 * time.Second, Task: Task{Number: "3", Name: "T3", Prompt: "test", Agent: "agent1"}},
			},
		},
		{
			name: "no files",
			results: []TaskResult{
				{Status: StatusGreen, Duration: 1 * time.Second, Task: Task{Number: "1", Name: "T1", Prompt: "test", Agent: "agent1", Files: []string{}}},
				{Status: StatusYellow, Duration: 2 * time.Second, Task: Task{Number: "2", Name: "T2", Prompt: "test", Agent: "agent2", Files: nil}},
			},
		},
		{
			name:    "empty results",
			results: []TaskResult{},
		},
		{
			name: "zero durations",
			results: []TaskResult{
				{Status: StatusGreen, Duration: 0, Task: Task{Number: "1", Name: "T1", Prompt: "test", Agent: "agent1"}},
				{Status: StatusYellow, Duration: 0, Task: Task{Number: "2", Name: "T2", Prompt: "test", Agent: "agent2"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create result using NewExecutionResult
			resultFromNew := NewExecutionResult(tt.results, true, 10*time.Second)

			// Create result using CalculateMetrics
			resultFromCalc := &ExecutionResult{
				TotalTasks:  len(tt.results),
				Duration:    10 * time.Second,
				FailedTasks: []TaskResult{},
			}
			resultFromCalc.CalculateMetrics(tt.results)

			// Verify StatusBreakdown is identical
			if len(resultFromNew.StatusBreakdown) != len(resultFromCalc.StatusBreakdown) {
				t.Errorf("StatusBreakdown length mismatch: NewExecutionResult=%d, CalculateMetrics=%d",
					len(resultFromNew.StatusBreakdown), len(resultFromCalc.StatusBreakdown))
			}
			for status, count := range resultFromNew.StatusBreakdown {
				if resultFromCalc.StatusBreakdown[status] != count {
					t.Errorf("StatusBreakdown[%s] mismatch: NewExecutionResult=%d, CalculateMetrics=%d",
						status, count, resultFromCalc.StatusBreakdown[status])
				}
			}

			// Verify AgentUsage is identical (this will currently fail due to empty agent handling difference)
			if len(resultFromNew.AgentUsage) != len(resultFromCalc.AgentUsage) {
				t.Errorf("AgentUsage length mismatch: NewExecutionResult=%d, CalculateMetrics=%d",
					len(resultFromNew.AgentUsage), len(resultFromCalc.AgentUsage))
			}
			for agent, count := range resultFromNew.AgentUsage {
				if resultFromCalc.AgentUsage[agent] != count {
					t.Errorf("AgentUsage[%s] mismatch: NewExecutionResult=%d, CalculateMetrics=%d",
						agent, count, resultFromCalc.AgentUsage[agent])
				}
			}
			// Check reverse (CalculateMetrics has agents that NewExecutionResult doesn't)
			for agent, count := range resultFromCalc.AgentUsage {
				if resultFromNew.AgentUsage[agent] != count {
					t.Errorf("AgentUsage[%s] in CalculateMetrics but not in NewExecutionResult: CalculateMetrics=%d",
						agent, count)
				}
			}

			// Verify TotalFiles is identical
			if resultFromNew.TotalFiles != resultFromCalc.TotalFiles {
				t.Errorf("TotalFiles mismatch: NewExecutionResult=%d, CalculateMetrics=%d",
					resultFromNew.TotalFiles, resultFromCalc.TotalFiles)
			}

			// Verify AvgTaskDuration is identical
			if resultFromNew.AvgTaskDuration != resultFromCalc.AvgTaskDuration {
				t.Errorf("AvgTaskDuration mismatch: NewExecutionResult=%v, CalculateMetrics=%v",
					resultFromNew.AvgTaskDuration, resultFromCalc.AvgTaskDuration)
			}

			// Verify Completed/Failed counts are identical
			if resultFromNew.Completed != resultFromCalc.Completed {
				t.Errorf("Completed mismatch: NewExecutionResult=%d, CalculateMetrics=%d",
					resultFromNew.Completed, resultFromCalc.Completed)
			}
			if resultFromNew.Failed != resultFromCalc.Failed {
				t.Errorf("Failed mismatch: NewExecutionResult=%d, CalculateMetrics=%d",
					resultFromNew.Failed, resultFromCalc.Failed)
			}

			// Verify LOC aggregates are identical
			if resultFromNew.TotalLinesAdded != resultFromCalc.TotalLinesAdded {
				t.Errorf("TotalLinesAdded mismatch: NewExecutionResult=%d, CalculateMetrics=%d",
					resultFromNew.TotalLinesAdded, resultFromCalc.TotalLinesAdded)
			}
			if resultFromNew.TotalLinesDeleted != resultFromCalc.TotalLinesDeleted {
				t.Errorf("TotalLinesDeleted mismatch: NewExecutionResult=%d, CalculateMetrics=%d",
					resultFromNew.TotalLinesDeleted, resultFromCalc.TotalLinesDeleted)
			}
		})
	}
}

// TestExecutionResult_HelperFunction tests the private helper function behavior
// by verifying it through the public API after refactoring.
func TestExecutionResult_HelperFunction(t *testing.T) {
	tests := []struct {
		name                string
		results             []TaskResult
		expectedCompleted   int
		expectedFailed      int
		expectedGreen       int
		expectedYellow      int
		expectedRed         int
		expectedFiles       int
		expectedAvgDuration time.Duration
		expectedAgents      map[string]int
	}{
		{
			name: "comprehensive test case",
			results: []TaskResult{
				{
					Status:   StatusGreen,
					Duration: 2 * time.Second,
					Task:     Task{Number: "1", Name: "T1", Prompt: "test", Agent: "golang-pro", Files: []string{"a.go", "b.go"}},
				},
				{
					Status:   StatusGreen,
					Duration: 4 * time.Second,
					Task:     Task{Number: "2", Name: "T2", Prompt: "test", Agent: "golang-pro", Files: []string{"b.go", "c.go"}},
				},
				{
					Status:   StatusYellow,
					Duration: 6 * time.Second,
					Task:     Task{Number: "3", Name: "T3", Prompt: "test", Agent: "devops", Files: []string{"d.yaml"}},
				},
				{
					Status:   StatusRed,
					Duration: 8 * time.Second,
					Task:     Task{Number: "4", Name: "T4", Prompt: "test", Agent: "golang-pro", Files: []string{"e.go"}},
				},
			},
			expectedCompleted:   3, // GREEN + YELLOW
			expectedFailed:      1, // RED
			expectedGreen:       2,
			expectedYellow:      1,
			expectedRed:         1,
			expectedFiles:       5,               // a.go, b.go, c.go, d.yaml, e.go
			expectedAvgDuration: 5 * time.Second, // (2+4+6+8)/4 = 20/4 = 5
			expectedAgents: map[string]int{
				"golang-pro": 3,
				"devops":     1,
			},
		},
		{
			name: "empty agent handling",
			results: []TaskResult{
				{Status: StatusGreen, Duration: 1 * time.Second, Task: Task{Number: "1", Name: "T1", Prompt: "test", Agent: "", Files: []string{"file1.go"}}},
				{Status: StatusGreen, Duration: 2 * time.Second, Task: Task{Number: "2", Name: "T2", Prompt: "test", Agent: "golang-pro", Files: []string{"file2.go"}}},
			},
			expectedCompleted:   2,
			expectedFailed:      0,
			expectedGreen:       2,
			expectedYellow:      0,
			expectedRed:         0,
			expectedFiles:       2,
			expectedAvgDuration: 1500 * time.Millisecond, // (1+2)/2 = 1.5s
			expectedAgents: map[string]int{
				"":           1,
				"golang-pro": 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewExecutionResult(tt.results, true, 10*time.Second)

			if result.Completed != tt.expectedCompleted {
				t.Errorf("Completed = %d, want %d", result.Completed, tt.expectedCompleted)
			}
			if result.Failed != tt.expectedFailed {
				t.Errorf("Failed = %d, want %d", result.Failed, tt.expectedFailed)
			}
			if result.StatusBreakdown[StatusGreen] != tt.expectedGreen {
				t.Errorf("StatusBreakdown[GREEN] = %d, want %d", result.StatusBreakdown[StatusGreen], tt.expectedGreen)
			}
			if result.StatusBreakdown[StatusYellow] != tt.expectedYellow {
				t.Errorf("StatusBreakdown[YELLOW] = %d, want %d", result.StatusBreakdown[StatusYellow], tt.expectedYellow)
			}
			if result.StatusBreakdown[StatusRed] != tt.expectedRed {
				t.Errorf("StatusBreakdown[RED] = %d, want %d", result.StatusBreakdown[StatusRed], tt.expectedRed)
			}
			if result.TotalFiles != tt.expectedFiles {
				t.Errorf("TotalFiles = %d, want %d", result.TotalFiles, tt.expectedFiles)
			}
			if result.AvgTaskDuration != tt.expectedAvgDuration {
				t.Errorf("AvgTaskDuration = %v, want %v", result.AvgTaskDuration, tt.expectedAvgDuration)
			}

			// Verify agent usage
			if len(result.AgentUsage) != len(tt.expectedAgents) {
				t.Errorf("AgentUsage length = %d, want %d", len(result.AgentUsage), len(tt.expectedAgents))
			}
			for agent, expectedCount := range tt.expectedAgents {
				if result.AgentUsage[agent] != expectedCount {
					t.Errorf("AgentUsage[%s] = %d, want %d", agent, result.AgentUsage[agent], expectedCount)
				}
			}
		})
	}
}
