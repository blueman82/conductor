package behavioral

import (
	"testing"
	"time"
)

func TestExtractMetrics(t *testing.T) {
	tests := []struct {
		name     string
		input    *SessionData
		validate func(*testing.T, *BehavioralMetrics)
	}{
		{
			name:  "nil session data",
			input: nil,
			validate: func(t *testing.T, metrics *BehavioralMetrics) {
				if metrics == nil {
					t.Fatal("expected non-nil metrics")
				}
				if len(metrics.ToolExecutions) != 0 {
					t.Errorf("expected empty tool executions, got %d", len(metrics.ToolExecutions))
				}
				if len(metrics.BashCommands) != 0 {
					t.Errorf("expected empty bash commands, got %d", len(metrics.BashCommands))
				}
				if len(metrics.FileOperations) != 0 {
					t.Errorf("expected empty file operations, got %d", len(metrics.FileOperations))
				}
			},
		},
		{
			name: "empty session",
			input: &SessionData{
				Session: Session{
					ID:        "test-session",
					Project:   "test-project",
					Timestamp: time.Now(),
					Success:   true,
					Duration:  5000,
				},
				Events: []Event{},
			},
			validate: func(t *testing.T, metrics *BehavioralMetrics) {
				if metrics.TotalSessions != 1 {
					t.Errorf("expected 1 session, got %d", metrics.TotalSessions)
				}
				if metrics.SuccessRate != 1.0 {
					t.Errorf("expected success rate 1.0, got %f", metrics.SuccessRate)
				}
				if metrics.AverageDuration != 5*time.Second {
					t.Errorf("expected 5s duration, got %v", metrics.AverageDuration)
				}
			},
		},
		{
			name: "session with tool calls",
			input: &SessionData{
				Session: Session{
					ID:        "test-session",
					Project:   "test-project",
					Timestamp: time.Now(),
					Success:   true,
					Duration:  10000,
				},
				Events: []Event{
					&ToolCallEvent{
						BaseEvent: BaseEvent{
							Type:      "tool_call",
							Timestamp: time.Now(),
						},
						ToolName: "Read",
						Success:  true,
						Duration: 100,
					},
					&ToolCallEvent{
						BaseEvent: BaseEvent{
							Type:      "tool_call",
							Timestamp: time.Now(),
						},
						ToolName: "Read",
						Success:  true,
						Duration: 200,
					},
					&ToolCallEvent{
						BaseEvent: BaseEvent{
							Type:      "tool_call",
							Timestamp: time.Now(),
						},
						ToolName: "Write",
						Success:  false,
						Duration: 50,
					},
				},
			},
			validate: func(t *testing.T, metrics *BehavioralMetrics) {
				if len(metrics.ToolExecutions) != 2 {
					t.Fatalf("expected 2 tool types, got %d", len(metrics.ToolExecutions))
				}

				// Find Read tool
				var readTool *ToolExecution
				for i := range metrics.ToolExecutions {
					if metrics.ToolExecutions[i].Name == "Read" {
						readTool = &metrics.ToolExecutions[i]
						break
					}
				}

				if readTool == nil {
					t.Fatal("Read tool not found")
				}
				if readTool.Count != 2 {
					t.Errorf("expected Read count 2, got %d", readTool.Count)
				}
				if readTool.TotalSuccess != 2 {
					t.Errorf("expected Read success 2, got %d", readTool.TotalSuccess)
				}
				if readTool.SuccessRate != 1.0 {
					t.Errorf("expected Read success rate 1.0, got %f", readTool.SuccessRate)
				}

				// Find Write tool
				var writeTool *ToolExecution
				for i := range metrics.ToolExecutions {
					if metrics.ToolExecutions[i].Name == "Write" {
						writeTool = &metrics.ToolExecutions[i]
						break
					}
				}

				if writeTool == nil {
					t.Fatal("Write tool not found")
				}
				if writeTool.Count != 1 {
					t.Errorf("expected Write count 1, got %d", writeTool.Count)
				}
				if writeTool.TotalErrors != 1 {
					t.Errorf("expected Write errors 1, got %d", writeTool.TotalErrors)
				}
				if writeTool.SuccessRate != 0.0 {
					t.Errorf("expected Write success rate 0.0, got %f", writeTool.SuccessRate)
				}
			},
		},
		{
			name: "session with bash commands",
			input: &SessionData{
				Session: Session{
					ID:        "test-session",
					Project:   "test-project",
					Timestamp: time.Now(),
					Success:   true,
					Duration:  8000,
				},
				Events: []Event{
					&BashCommandEvent{
						BaseEvent: BaseEvent{
							Type:      "bash_command",
							Timestamp: time.Now(),
						},
						Command:      "ls -la",
						ExitCode:     0,
						OutputLength: 1024,
						Duration:     250,
						Success:      true,
					},
					&BashCommandEvent{
						BaseEvent: BaseEvent{
							Type:      "bash_command",
							Timestamp: time.Now(),
						},
						Command:      "git status",
						ExitCode:     1,
						OutputLength: 512,
						Duration:     100,
						Success:      false,
					},
				},
			},
			validate: func(t *testing.T, metrics *BehavioralMetrics) {
				if len(metrics.BashCommands) != 2 {
					t.Fatalf("expected 2 bash commands, got %d", len(metrics.BashCommands))
				}

				cmd1 := metrics.BashCommands[0]
				if cmd1.Command != "ls -la" {
					t.Errorf("expected 'ls -la', got %s", cmd1.Command)
				}
				if cmd1.ExitCode != 0 {
					t.Errorf("expected exit code 0, got %d", cmd1.ExitCode)
				}
				if !cmd1.Success {
					t.Error("expected command to be successful")
				}
				if cmd1.Duration != 250*time.Millisecond {
					t.Errorf("expected duration 250ms, got %v", cmd1.Duration)
				}

				cmd2 := metrics.BashCommands[1]
				if cmd2.Command != "git status" {
					t.Errorf("expected 'git status', got %s", cmd2.Command)
				}
				if cmd2.ExitCode != 1 {
					t.Errorf("expected exit code 1, got %d", cmd2.ExitCode)
				}
				if cmd2.Success {
					t.Error("expected command to fail")
				}
			},
		},
		{
			name: "session with file operations",
			input: &SessionData{
				Session: Session{
					ID:        "test-session",
					Project:   "test-project",
					Timestamp: time.Now(),
					Success:   true,
					Duration:  6000,
				},
				Events: []Event{
					&FileOperationEvent{
						BaseEvent: BaseEvent{
							Type:      "file_operation",
							Timestamp: time.Now(),
						},
						Operation: "read",
						Path:      "/path/to/file.txt",
						Success:   true,
						SizeBytes: 2048,
						Duration:  150,
					},
					&FileOperationEvent{
						BaseEvent: BaseEvent{
							Type:      "file_operation",
							Timestamp: time.Now(),
						},
						Operation: "write",
						Path:      "/path/to/output.txt",
						Success:   true,
						SizeBytes: 4096,
						Duration:  300,
					},
				},
			},
			validate: func(t *testing.T, metrics *BehavioralMetrics) {
				if len(metrics.FileOperations) != 2 {
					t.Fatalf("expected 2 file operations, got %d", len(metrics.FileOperations))
				}

				op1 := metrics.FileOperations[0]
				if op1.Type != "read" {
					t.Errorf("expected type 'read', got %s", op1.Type)
				}
				if op1.Path != "/path/to/file.txt" {
					t.Errorf("expected path '/path/to/file.txt', got %s", op1.Path)
				}
				if op1.SizeBytes != 2048 {
					t.Errorf("expected size 2048, got %d", op1.SizeBytes)
				}
				if !op1.Success {
					t.Error("expected operation to be successful")
				}

				op2 := metrics.FileOperations[1]
				if op2.Type != "write" {
					t.Errorf("expected type 'write', got %s", op2.Type)
				}
				if op2.SizeBytes != 4096 {
					t.Errorf("expected size 4096, got %d", op2.SizeBytes)
				}
			},
		},
		{
			name: "session with token usage",
			input: &SessionData{
				Session: Session{
					ID:        "test-session",
					Project:   "test-project",
					Timestamp: time.Now(),
					Success:   true,
					Duration:  12000,
				},
				Events: []Event{
					&TokenUsageEvent{
						BaseEvent: BaseEvent{
							Type:      "token_usage",
							Timestamp: time.Now(),
						},
						InputTokens:  1000,
						OutputTokens: 500,
						CostUSD:      0.01,
						ModelName:    "claude-sonnet-4-5",
					},
					&TokenUsageEvent{
						BaseEvent: BaseEvent{
							Type:      "token_usage",
							Timestamp: time.Now(),
						},
						InputTokens:  2000,
						OutputTokens: 1000,
						CostUSD:      0.02,
						ModelName:    "claude-sonnet-4-5",
					},
				},
			},
			validate: func(t *testing.T, metrics *BehavioralMetrics) {
				if metrics.TokenUsage.InputTokens != 3000 {
					t.Errorf("expected input tokens 3000, got %d", metrics.TokenUsage.InputTokens)
				}
				if metrics.TokenUsage.OutputTokens != 1500 {
					t.Errorf("expected output tokens 1500, got %d", metrics.TokenUsage.OutputTokens)
				}
				if metrics.TokenUsage.CostUSD != 0.03 {
					t.Errorf("expected cost 0.03, got %f", metrics.TokenUsage.CostUSD)
				}
				if metrics.TokenUsage.ModelName != "claude-sonnet-4-5" {
					t.Errorf("expected model 'claude-sonnet-4-5', got %s", metrics.TokenUsage.ModelName)
				}
			},
		},
		{
			name: "session with mixed events",
			input: &SessionData{
				Session: Session{
					ID:         "test-session",
					Project:    "test-project",
					Timestamp:  time.Now(),
					Success:    false,
					Duration:   15000,
					AgentName:  "test-agent",
					ErrorCount: 3,
				},
				Events: []Event{
					&ToolCallEvent{
						BaseEvent: BaseEvent{Type: "tool_call", Timestamp: time.Now()},
						ToolName:  "Read",
						Success:   true,
						Duration:  100,
					},
					&BashCommandEvent{
						BaseEvent:    BaseEvent{Type: "bash_command", Timestamp: time.Now()},
						Command:      "npm test",
						ExitCode:     0,
						OutputLength: 2048,
						Duration:     5000,
						Success:      true,
					},
					&FileOperationEvent{
						BaseEvent: BaseEvent{Type: "file_operation", Timestamp: time.Now()},
						Operation: "edit",
						Path:      "/path/to/code.go",
						Success:   true,
						SizeBytes: 8192,
						Duration:  200,
					},
					&TokenUsageEvent{
						BaseEvent:    BaseEvent{Type: "token_usage", Timestamp: time.Now()},
						InputTokens:  5000,
						OutputTokens: 2500,
						CostUSD:      0.05,
						ModelName:    "claude-sonnet-4-5",
					},
				},
			},
			validate: func(t *testing.T, metrics *BehavioralMetrics) {
				if metrics.TotalSessions != 1 {
					t.Errorf("expected 1 session, got %d", metrics.TotalSessions)
				}
				if metrics.SuccessRate != 0.0 {
					t.Errorf("expected success rate 0.0, got %f", metrics.SuccessRate)
				}
				if metrics.TotalErrors != 3 {
					t.Errorf("expected 3 errors, got %d", metrics.TotalErrors)
				}
				if len(metrics.ToolExecutions) != 1 {
					t.Errorf("expected 1 tool execution, got %d", len(metrics.ToolExecutions))
				}
				if len(metrics.BashCommands) != 1 {
					t.Errorf("expected 1 bash command, got %d", len(metrics.BashCommands))
				}
				if len(metrics.FileOperations) != 1 {
					t.Errorf("expected 1 file operation, got %d", len(metrics.FileOperations))
				}
				if metrics.TokenUsage.InputTokens != 5000 {
					t.Errorf("expected 5000 input tokens, got %d", metrics.TokenUsage.InputTokens)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := ExtractMetrics(tt.input)
			tt.validate(t, metrics)
		})
	}
}

func TestAggregateToolExecutions(t *testing.T) {
	tests := []struct {
		name     string
		input    []*ToolCallEvent
		validate func(*testing.T, []ToolExecution)
	}{
		{
			name:  "nil events",
			input: nil,
			validate: func(t *testing.T, result []ToolExecution) {
				if len(result) != 0 {
					t.Errorf("expected empty result, got %d items", len(result))
				}
			},
		},
		{
			name:  "empty events",
			input: []*ToolCallEvent{},
			validate: func(t *testing.T, result []ToolExecution) {
				if len(result) != 0 {
					t.Errorf("expected empty result, got %d items", len(result))
				}
			},
		},
		{
			name: "single tool, multiple calls",
			input: []*ToolCallEvent{
				{
					BaseEvent: BaseEvent{Type: "tool_call", Timestamp: time.Now()},
					ToolName:  "Read",
					Success:   true,
					Duration:  100,
				},
				{
					BaseEvent: BaseEvent{Type: "tool_call", Timestamp: time.Now()},
					ToolName:  "Read",
					Success:   true,
					Duration:  200,
				},
				{
					BaseEvent: BaseEvent{Type: "tool_call", Timestamp: time.Now()},
					ToolName:  "Read",
					Success:   false,
					Duration:  50,
				},
			},
			validate: func(t *testing.T, result []ToolExecution) {
				if len(result) != 1 {
					t.Fatalf("expected 1 tool, got %d", len(result))
				}
				tool := result[0]
				if tool.Name != "Read" {
					t.Errorf("expected name 'Read', got %s", tool.Name)
				}
				if tool.Count != 3 {
					t.Errorf("expected count 3, got %d", tool.Count)
				}
				if tool.TotalSuccess != 2 {
					t.Errorf("expected 2 successes, got %d", tool.TotalSuccess)
				}
				if tool.TotalErrors != 1 {
					t.Errorf("expected 1 error, got %d", tool.TotalErrors)
				}
				expectedRate := 2.0 / 3.0
				if tool.SuccessRate != expectedRate {
					t.Errorf("expected success rate %f, got %f", expectedRate, tool.SuccessRate)
				}
			},
		},
		{
			name: "multiple tools",
			input: []*ToolCallEvent{
				{
					BaseEvent: BaseEvent{Type: "tool_call", Timestamp: time.Now()},
					ToolName:  "Read",
					Success:   true,
					Duration:  100,
				},
				{
					BaseEvent: BaseEvent{Type: "tool_call", Timestamp: time.Now()},
					ToolName:  "Write",
					Success:   true,
					Duration:  200,
				},
				{
					BaseEvent: BaseEvent{Type: "tool_call", Timestamp: time.Now()},
					ToolName:  "Bash",
					Success:   false,
					Duration:  50,
				},
			},
			validate: func(t *testing.T, result []ToolExecution) {
				if len(result) != 3 {
					t.Fatalf("expected 3 tools, got %d", len(result))
				}
				// Each tool should have count 1
				for _, tool := range result {
					if tool.Count != 1 {
						t.Errorf("expected count 1 for %s, got %d", tool.Name, tool.Count)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := aggregateToolExecutions(tt.input)
			tt.validate(t, result)
		})
	}
}

func TestCalculateSuccessRate(t *testing.T) {
	tests := []struct {
		name      string
		successes int
		total     int
		expected  float64
	}{
		{"zero total", 0, 0, 0.0},
		{"all success", 10, 10, 1.0},
		{"no success", 0, 10, 0.0},
		{"half success", 5, 10, 0.5},
		{"one third", 1, 3, 0.3333333333333333},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateSuccessRate(tt.successes, tt.total)
			if result != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestCalculateAverageDuration(t *testing.T) {
	tests := []struct {
		name      string
		durations []int64
		expected  time.Duration
	}{
		{"empty", []int64{}, 0},
		{"single", []int64{1000}, 1000 * time.Millisecond},
		{"multiple", []int64{1000, 2000, 3000}, 2000 * time.Millisecond},
		{"with zero", []int64{0, 1000, 2000}, 1000 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateAverageDuration(tt.durations)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCalculateTokenCost(t *testing.T) {
	tests := []struct {
		name         string
		inputTokens  int64
		outputTokens int64
		model        string
		expected     float64
	}{
		{"zero tokens", 0, 0, "claude-sonnet-4-5", 0.0},
		{"only input", 1_000_000, 0, "claude-sonnet-4-5", 3.0},
		{"only output", 0, 1_000_000, "claude-sonnet-4-5", 15.0},
		{"both", 1_000_000, 1_000_000, "claude-sonnet-4-5", 18.0},
		{"small amounts", 10_000, 5_000, "claude-sonnet-4-5", 0.105},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateTokenCost(tt.inputTokens, tt.outputTokens, tt.model)
			if result != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestAggregateBashCommands(t *testing.T) {
	now := time.Now()
	events := []*BashCommandEvent{
		{
			BaseEvent:    BaseEvent{Type: "bash_command", Timestamp: now},
			Command:      "ls -la",
			ExitCode:     0,
			OutputLength: 1024,
			Duration:     250,
			Success:      true,
		},
		nil, // Test nil handling
		{
			BaseEvent:    BaseEvent{Type: "bash_command", Timestamp: now},
			Command:      "git status",
			ExitCode:     1,
			OutputLength: 512,
			Duration:     100,
			Success:      false,
		},
	}

	result := aggregateBashCommands(events)

	if len(result) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(result))
	}

	cmd1 := result[0]
	if cmd1.Command != "ls -la" {
		t.Errorf("expected 'ls -la', got %s", cmd1.Command)
	}
	if cmd1.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", cmd1.ExitCode)
	}
}

func TestAggregateFileOperations(t *testing.T) {
	now := time.Now()
	events := []*FileOperationEvent{
		{
			BaseEvent: BaseEvent{Type: "file_operation", Timestamp: now},
			Operation: "read",
			Path:      "/test/file.txt",
			Success:   true,
			SizeBytes: 2048,
			Duration:  150,
		},
		nil, // Test nil handling
		{
			BaseEvent: BaseEvent{Type: "file_operation", Timestamp: now},
			Operation: "write",
			Path:      "/test/output.txt",
			Success:   true,
			SizeBytes: 4096,
			Duration:  300,
		},
	}

	result := aggregateFileOperations(events)

	if len(result) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(result))
	}

	op1 := result[0]
	if op1.Type != "read" {
		t.Errorf("expected type 'read', got %s", op1.Type)
	}
	if op1.Path != "/test/file.txt" {
		t.Errorf("expected path '/test/file.txt', got %s", op1.Path)
	}
}

func TestAggregateTokenUsage(t *testing.T) {
	events := []*TokenUsageEvent{
		{
			BaseEvent:    BaseEvent{Type: "token_usage", Timestamp: time.Now()},
			InputTokens:  1000,
			OutputTokens: 500,
			CostUSD:      0.01,
			ModelName:    "claude-sonnet-4-5",
		},
		nil, // Test nil handling
		{
			BaseEvent:    BaseEvent{Type: "token_usage", Timestamp: time.Now()},
			InputTokens:  2000,
			OutputTokens: 1000,
			CostUSD:      0.02,
			ModelName:    "claude-sonnet-4-5",
		},
	}

	result := aggregateTokenUsage(events)

	if result.InputTokens != 3000 {
		t.Errorf("expected 3000 input tokens, got %d", result.InputTokens)
	}
	if result.OutputTokens != 1500 {
		t.Errorf("expected 1500 output tokens, got %d", result.OutputTokens)
	}
	if result.CostUSD != 0.03 {
		t.Errorf("expected cost 0.03, got %f", result.CostUSD)
	}
	if result.ModelName != "claude-sonnet-4-5" {
		t.Errorf("expected model 'claude-sonnet-4-5', got %s", result.ModelName)
	}
}
