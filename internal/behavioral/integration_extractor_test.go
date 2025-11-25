package behavioral

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestExtractMetricsIntegration tests the full flow from file to metrics
func TestExtractMetricsIntegration(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, "test-session.jsonl")

	// Write test JSONL data
	content := `{"type":"session_start","id":"test-123","project":"test-project","timestamp":"2024-01-01T12:00:00Z","status":"completed","agent_name":"test-agent","duration":15000,"success":true,"error_count":2}
{"type":"tool_call","timestamp":"2024-01-01T12:00:01Z","tool_name":"Read","parameters":{},"result":"success","success":true,"duration":150}
{"type":"tool_call","timestamp":"2024-01-01T12:00:02Z","tool_name":"Read","parameters":{},"result":"success","success":true,"duration":200}
{"type":"tool_call","timestamp":"2024-01-01T12:00:03Z","tool_name":"Write","parameters":{},"result":"error","success":false,"duration":50,"error":"file not found"}
{"type":"bash_command","timestamp":"2024-01-01T12:00:04Z","command":"go test ./...","exit_code":0,"output":"PASS","output_length":1024,"duration":5000,"success":true}
{"type":"bash_command","timestamp":"2024-01-01T12:00:09Z","command":"git status","exit_code":1,"output":"error","output_length":256,"duration":100,"success":false}
{"type":"file_operation","timestamp":"2024-01-01T12:00:10Z","operation":"read","path":"/test/file.go","success":true,"size_bytes":4096,"duration":200}
{"type":"file_operation","timestamp":"2024-01-01T12:00:11Z","operation":"write","path":"/test/output.go","success":true,"size_bytes":8192,"duration":300}
{"type":"token_usage","timestamp":"2024-01-01T12:00:12Z","input_tokens":10000,"output_tokens":5000,"cost_usd":0.105,"model_name":"claude-sonnet-4-5"}
`

	if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Parse the session file
	sessionData, err := ParseSessionFile(sessionFile)
	if err != nil {
		t.Fatalf("ParseSessionFile failed: %v", err)
	}

	// Extract metrics
	metrics := ExtractMetrics(sessionData)

	// Validate session-level metrics
	if metrics.TotalSessions != 1 {
		t.Errorf("expected 1 session, got %d", metrics.TotalSessions)
	}
	if metrics.SuccessRate != 1.0 {
		t.Errorf("expected success rate 1.0, got %f", metrics.SuccessRate)
	}
	if metrics.AverageDuration != 15*time.Second {
		t.Errorf("expected duration 15s, got %v", metrics.AverageDuration)
	}
	if metrics.TotalErrors != 2 {
		t.Errorf("expected 2 errors, got %d", metrics.TotalErrors)
	}

	// Validate tool executions
	if len(metrics.ToolExecutions) != 2 {
		t.Fatalf("expected 2 tool types, got %d", len(metrics.ToolExecutions))
	}

	var readTool, writeTool *ToolExecution
	for i := range metrics.ToolExecutions {
		if metrics.ToolExecutions[i].Name == "Read" {
			readTool = &metrics.ToolExecutions[i]
		} else if metrics.ToolExecutions[i].Name == "Write" {
			writeTool = &metrics.ToolExecutions[i]
		}
	}

	if readTool == nil || writeTool == nil {
		t.Fatal("missing Read or Write tool in metrics")
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

	if writeTool.Count != 1 {
		t.Errorf("expected Write count 1, got %d", writeTool.Count)
	}
	if writeTool.TotalErrors != 1 {
		t.Errorf("expected Write errors 1, got %d", writeTool.TotalErrors)
	}
	if writeTool.ErrorRate != 1.0 {
		t.Errorf("expected Write error rate 1.0, got %f", writeTool.ErrorRate)
	}

	// Validate bash commands
	if len(metrics.BashCommands) != 2 {
		t.Fatalf("expected 2 bash commands, got %d", len(metrics.BashCommands))
	}

	cmd1 := metrics.BashCommands[0]
	if cmd1.Command != "go test ./..." {
		t.Errorf("expected 'go test ./...', got %s", cmd1.Command)
	}
	if cmd1.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", cmd1.ExitCode)
	}
	if !cmd1.Success {
		t.Error("expected command to succeed")
	}
	if cmd1.Duration != 5*time.Second {
		t.Errorf("expected duration 5s, got %v", cmd1.Duration)
	}

	cmd2 := metrics.BashCommands[1]
	if cmd2.Command != "git status" {
		t.Errorf("expected 'git status', got %s", cmd2.Command)
	}
	if cmd2.Success {
		t.Error("expected command to fail")
	}

	// Validate file operations
	if len(metrics.FileOperations) != 2 {
		t.Fatalf("expected 2 file operations, got %d", len(metrics.FileOperations))
	}

	op1 := metrics.FileOperations[0]
	if op1.Type != "read" {
		t.Errorf("expected type 'read', got %s", op1.Type)
	}
	if op1.Path != "/test/file.go" {
		t.Errorf("expected path '/test/file.go', got %s", op1.Path)
	}
	if op1.SizeBytes != 4096 {
		t.Errorf("expected size 4096, got %d", op1.SizeBytes)
	}

	op2 := metrics.FileOperations[1]
	if op2.Type != "write" {
		t.Errorf("expected type 'write', got %s", op2.Type)
	}
	if op2.SizeBytes != 8192 {
		t.Errorf("expected size 8192, got %d", op2.SizeBytes)
	}

	// Validate token usage
	if metrics.TokenUsage.InputTokens != 10000 {
		t.Errorf("expected 10000 input tokens, got %d", metrics.TokenUsage.InputTokens)
	}
	if metrics.TokenUsage.OutputTokens != 5000 {
		t.Errorf("expected 5000 output tokens, got %d", metrics.TokenUsage.OutputTokens)
	}
	if metrics.TokenUsage.CostUSD != 0.105 {
		t.Errorf("expected cost 0.105, got %f", metrics.TokenUsage.CostUSD)
	}
	if metrics.TokenUsage.ModelName != "claude-sonnet-4-5" {
		t.Errorf("expected model 'claude-sonnet-4-5', got %s", metrics.TokenUsage.ModelName)
	}

	// Validate total cost calculation
	expectedCost := 0.105
	if metrics.TotalCost != expectedCost {
		t.Errorf("expected total cost %f, got %f", expectedCost, metrics.TotalCost)
	}

	// Validate metrics are internally consistent
	if err := metrics.Validate(); err != nil {
		t.Errorf("metrics validation failed: %v", err)
	}
}

// TestExtractMetricsWithDiscovery tests integration with discovery
func TestExtractMetricsWithDiscovery(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create test session file
	sessionFile := filepath.Join(projectDir, "agent-12345678.jsonl")
	content := `{"type":"session_metadata","id":"12345678","project":"test-project","timestamp":"2024-01-01T12:00:00Z","status":"completed","agent_name":"general-agent","duration":10000,"success":true,"error_count":0}
{"type":"tool_call","timestamp":"2024-01-01T12:00:01Z","tool_name":"Read","parameters":{},"result":"success","success":true,"duration":100}
{"type":"token_usage","timestamp":"2024-01-01T12:00:02Z","input_tokens":1000,"output_tokens":500,"cost_usd":0.0105,"model_name":"claude-sonnet-4-5"}
`

	if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Discover sessions
	sessions, err := DiscoverProjectSessions(tmpDir, "test-project")
	if err != nil {
		t.Fatalf("DiscoverProjectSessions failed: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	session := sessions[0]
	if session.SessionID != "12345678" {
		t.Errorf("expected session ID '12345678', got %s", session.SessionID)
	}

	// Parse the discovered session
	sessionData, err := ParseSessionFile(session.FilePath)
	if err != nil {
		t.Fatalf("ParseSessionFile failed: %v", err)
	}

	// Extract metrics
	metrics := ExtractMetrics(sessionData)

	// Validate
	if metrics.TotalSessions != 1 {
		t.Errorf("expected 1 session, got %d", metrics.TotalSessions)
	}
	if len(metrics.ToolExecutions) != 1 {
		t.Errorf("expected 1 tool execution, got %d", len(metrics.ToolExecutions))
	}
	if metrics.TokenUsage.InputTokens != 1000 {
		t.Errorf("expected 1000 input tokens, got %d", metrics.TokenUsage.InputTokens)
	}
}

// TestExtractMetricsRobustness tests error resilience
func TestExtractMetricsRobustness(t *testing.T) {
	// Create temporary test file with mixed valid/invalid data
	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, "robust-session.jsonl")

	content := `{"type":"session_start","id":"robust-test","project":"test","timestamp":"2024-01-01T12:00:00Z","status":"completed","agent_name":"test","duration":5000,"success":true,"error_count":0}
{"type":"tool_call","timestamp":"2024-01-01T12:00:01Z","tool_name":"Read","success":true,"duration":100}
invalid json line
{"type":"tool_call","timestamp":"2024-01-01T12:00:02Z","tool_name":"Write","success":false,"duration":0}
{"type":"bash_command","timestamp":"2024-01-01T12:00:03Z","command":"ls","exit_code":0,"output":"","output_length":0,"duration":50,"success":true}
`

	if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Parse should succeed despite invalid line
	sessionData, err := ParseSessionFile(sessionFile)
	if err != nil {
		t.Fatalf("ParseSessionFile failed: %v", err)
	}

	// Extract metrics should handle gracefully
	metrics := ExtractMetrics(sessionData)

	// Should have valid events only
	if len(metrics.ToolExecutions) != 2 {
		t.Errorf("expected 2 tool executions, got %d", len(metrics.ToolExecutions))
	}
	if len(metrics.BashCommands) != 1 {
		t.Errorf("expected 1 bash command, got %d", len(metrics.BashCommands))
	}
}
