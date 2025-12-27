package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/budget"
	"github.com/harrison/conductor/internal/models"
)

func TestNewInvoker(t *testing.T) {
	tests := []struct {
		name            string
		wantClaudePath  string
		wantRegistryNil bool
	}{
		{
			name:            "default invoker",
			wantClaudePath:  "claude",
			wantRegistryNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := NewInvoker()
			if inv == nil {
				t.Fatal("NewInvoker() returned nil")
			}
			if inv.ClaudePath != tt.wantClaudePath {
				t.Errorf("ClaudePath = %s, want %s", inv.ClaudePath, tt.wantClaudePath)
			}
			if (inv.Registry == nil) != tt.wantRegistryNil {
				t.Errorf("Registry nil = %v, want nil = %v", inv.Registry == nil, tt.wantRegistryNil)
			}
		})
	}
}

func TestNewInvokerWithRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry(tmpDir)

	inv := NewInvokerWithRegistry(registry)
	if inv == nil {
		t.Fatal("NewInvokerWithRegistry() returned nil")
	}
	if inv.Registry != registry {
		t.Error("Registry not set correctly")
	}
	if inv.ClaudePath != "claude" {
		t.Errorf("ClaudePath = %s, want 'claude'", inv.ClaudePath)
	}
}

func TestBuildCommandArgs(t *testing.T) {
	tests := []struct {
		name          string
		task          models.Task
		setupRegistry func(*testing.T) *Registry
		wantChecks    []func(*testing.T, []string)
	}{
		{
			name: "basic task without agent",
			task: models.Task{
				Number:        "1",
				Name:          "Test Task",
				Prompt:        "Do something",
				Agent:         "",
				EstimatedTime: 30 * time.Minute,
			},
			setupRegistry: nil,
			wantChecks: []func(*testing.T, []string){
				func(t *testing.T, args []string) {
					// Check for -p (print mode) flag - REQUIRED for non-interactive automation
					hasPrintFlag := false
					for _, arg := range args {
						if arg == "-p" {
							hasPrintFlag = true
							break
						}
					}
					if !hasPrintFlag {
						t.Error("Command must have -p flag for non-interactive print mode (essential for automation)")
					}
				},
				func(t *testing.T, args []string) {
					// Check for --permission-mode bypassPermissions flag
					hasPermissionMode := false
					for i, arg := range args {
						if arg == "--permission-mode" && i+1 < len(args) {
							if args[i+1] == "bypassPermissions" {
								hasPermissionMode = true
								break
							}
						}
					}
					if !hasPermissionMode {
						t.Error("Command should have --permission-mode bypassPermissions flag")
					}
				},
				func(t *testing.T, args []string) {
					// Check for --settings with disableAllHooks
					hasSettings := false
					for i, arg := range args {
						if arg == "--settings" && i+1 < len(args) {
							if strings.Contains(args[i+1], "disableAllHooks") {
								hasSettings = true
								break
							}
						}
					}
					if !hasSettings {
						t.Error("Command should have --settings with disableAllHooks")
					}
				},
				func(t *testing.T, args []string) {
					// Check for --output-format json
					hasOutputFormat := false
					for i, arg := range args {
						if arg == "--output-format" && i+1 < len(args) {
							if args[i+1] == "json" {
								hasOutputFormat = true
								break
							}
						}
					}
					if !hasOutputFormat {
						t.Error("Command should have --output-format json")
					}
				},
				func(t *testing.T, args []string) {
					// Verify prompt includes JSON instruction suffix and task prompt
					hasPrompt := false
					for _, arg := range args {
						if strings.Contains(arg, "Respond with ONLY this exact JSON") &&
							strings.Contains(arg, "Do something") {
							hasPrompt = true
							break
						}
					}
					if !hasPrompt {
						t.Error("Command should include JSON instruction suffix and the task prompt")
					}
				},
			},
		},
		{
			name: "task with agent that exists in registry",
			task: models.Task{
				Number:        "2",
				Name:          "Swift Task",
				Prompt:        "Build iOS app",
				Agent:         "swiftdev",
				EstimatedTime: 60 * time.Minute,
			},
			setupRegistry: func(t *testing.T) *Registry {
				tmpDir := t.TempDir()
				agentContent := `---
name: swiftdev
description: Swift development agent
---
Swift agent content
`
				err := os.WriteFile(filepath.Join(tmpDir, "swiftdev.md"), []byte(agentContent), 0644)
				if err != nil {
					t.Fatal(err)
				}
				registry := NewRegistry(tmpDir)
				_, err = registry.Discover()
				if err != nil {
					t.Fatal(err)
				}
				return registry
			},
			wantChecks: []func(*testing.T, []string){
				func(t *testing.T, args []string) {
					// Verify JSON instruction suffix (formatting instruction removed)
					hasFormatting := false
					for _, arg := range args {
						if strings.Contains(arg, "Respond with ONLY this exact JSON") {
							hasFormatting = true
							break
						}
					}
					if !hasFormatting {
						t.Error("Prompt should include JSON instruction suffix")
					}
				},
			},
		},
		{
			name: "task with agent that doesn't exist",
			task: models.Task{
				Number:        "3",
				Name:          "Invalid Agent Task",
				Prompt:        "Do work",
				Agent:         "nonexistent",
				EstimatedTime: 30 * time.Minute,
			},
			setupRegistry: func(t *testing.T) *Registry {
				tmpDir := t.TempDir()
				registry := NewRegistry(tmpDir)
				return registry
			},
			wantChecks: []func(*testing.T, []string){
				func(t *testing.T, args []string) {
					// Verify prompt includes JSON instruction suffix
					hasPrompt := false
					for _, arg := range args {
						if strings.Contains(arg, "Respond with ONLY this exact JSON") &&
							strings.Contains(arg, "Do work") {
							hasPrompt = true
							break
						}
					}
					if !hasPrompt {
						t.Error("Prompt should include formatting instructions when agent doesn't exist in registry")
					}
				},
				func(t *testing.T, args []string) {
					// Verify new JSON instruction format is present
					hasNewFormat := false
					for _, arg := range args {
						if strings.Contains(arg, "Respond with ONLY this exact JSON") {
							hasNewFormat = true
							break
						}
					}
					if !hasNewFormat {
						t.Error("Prompt should include new JSON instruction format")
					}
				},
			},
		},
		{
			name: "task with agent but no registry",
			task: models.Task{
				Number:        "4",
				Name:          "Task Without Registry",
				Prompt:        "Create something",
				Agent:         "someagent",
				EstimatedTime: 30 * time.Minute,
			},
			setupRegistry: nil,
			wantChecks: []func(*testing.T, []string){
				func(t *testing.T, args []string) {
					// Verify prompt includes JSON instruction suffix
					hasPrompt := false
					for _, arg := range args {
						if strings.Contains(arg, "Respond with ONLY this exact JSON") &&
							strings.Contains(arg, "Create something") {
							hasPrompt = true
							break
						}
					}
					if !hasPrompt {
						t.Error("Prompt should include JSON instruction when no registry is available")
					}
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up invoker with or without registry
			var inv *Invoker
			if tt.setupRegistry != nil {
				registry := tt.setupRegistry(t)
				inv = NewInvokerWithRegistry(registry)
			} else {
				inv = NewInvoker()
			}

			// Build command args
			args := inv.BuildCommandArgs(tt.task)

			// Run all validation checks
			for _, check := range tt.wantChecks {
				check(t, args)
			}
		})
	}
}

func TestBuildCommandArgsWithRealRegistry(t *testing.T) {
	// Create a temporary directory for agent files
	tmpDir := t.TempDir()

	// Create test agent files
	agentContent := `---
name: golang-pro
description: Go development expert
---
Go development content
`
	err := os.WriteFile(filepath.Join(tmpDir, "golang-pro.md"), []byte(agentContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create registry and discover agents
	registry := NewRegistry(tmpDir)
	agents, err := registry.Discover()
	if err != nil {
		t.Fatal(err)
	}

	if len(agents) != 1 {
		t.Fatalf("Expected 1 agent, got %d", len(agents))
	}

	// Create invoker with registry
	inv := NewInvokerWithRegistry(registry)

	// Test task with known agent
	task := models.Task{
		Number:        "1",
		Name:          "Go Task",
		Prompt:        "Write Go code",
		Agent:         "golang-pro",
		EstimatedTime: 30 * time.Minute,
	}

	args := inv.BuildCommandArgs(task)

	// Verify --agents flag includes golang-pro agent (agent reference prefix removed in e431bfd)
	found := false
	for i, arg := range args {
		if arg == "--agents" && i+1 < len(args) {
			if strings.Contains(args[i+1], "golang-pro") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("Expected --agents flag to include golang-pro agent definition")
	}
}

func TestParseClaudeOutput(t *testing.T) {
	tests := []struct {
		name          string
		output        string
		wantContent   string
		wantError     string
		wantSessionID string
		wantErr       bool
	}{
		{
			name:          "valid JSON output",
			output:        `{"content":"Hello World","error":""}`,
			wantContent:   "Hello World",
			wantError:     "",
			wantSessionID: "",
			wantErr:       false,
		},
		{
			name:          "valid JSON with session_id",
			output:        `{"content":"Task done","error":"","session_id":"abc-123-xyz"}`,
			wantContent:   "Task done",
			wantError:     "",
			wantSessionID: "abc-123-xyz",
			wantErr:       false,
		},
		{
			name:          "valid JSON with error",
			output:        `{"content":"","error":"Something went wrong"}`,
			wantContent:   "",
			wantError:     "Something went wrong",
			wantSessionID: "",
			wantErr:       false,
		},
		{
			name:          "non-JSON output",
			output:        "Plain text output without JSON",
			wantContent:   "Plain text output without JSON",
			wantError:     "",
			wantSessionID: "",
			wantErr:       false,
		},
		{
			name:          "empty output",
			output:        "",
			wantContent:   "",
			wantError:     "",
			wantSessionID: "",
			wantErr:       false,
		},
		{
			name:          "structured_output from --json-schema",
			output:        `{"type":"result","result":"","session_id":"test-123","structured_output":{"status":"success","summary":"Task done","output":"Created file","errors":[],"files_modified":["test.go"]}}`,
			wantContent:   `{"errors":[],"files_modified":["test.go"],"output":"Created file","status":"success","summary":"Task done"}`,
			wantError:     "",
			wantSessionID: "test-123",
			wantErr:       false,
		},
		{
			name:          "structured_output null - falls through to content",
			output:        `{"type":"result","content":"Task completed via content field","session_id":"test-456","structured_output":null}`,
			wantContent:   "Task completed via content field",
			wantError:     "",
			wantSessionID: "test-456",
			wantErr:       false,
		},
		{
			name:          "structured_output empty object - falls through to content",
			output:        `{"type":"result","content":"Task completed via content field","session_id":"test-789","structured_output":{}}`,
			wantContent:   "Task completed via content field",
			wantError:     "",
			wantSessionID: "test-789",
			wantErr:       false,
		},
		{
			name:          "structured_output null without content - falls through to result",
			output:        `{"type":"result","result":"Agent response text","session_id":"test-abc","structured_output":null}`,
			wantContent:   "Agent response text",
			wantError:     "",
			wantSessionID: "test-abc",
			wantErr:       false,
		},
		{
			name:          "mixed output with error prefix before JSON",
			output:        "Error: EOPNOTSUPP: unknown error\nSome other warning\n" + `{"type":"result","session_id":"mixed-123","structured_output":{"status":"success","summary":"Done","output":"Result","errors":[],"files_modified":[]}}`,
			wantContent:   `{"errors":[],"files_modified":[],"output":"Result","status":"success","summary":"Done"}`,
			wantError:     "",
			wantSessionID: "mixed-123",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseClaudeOutput(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseClaudeOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result.Content != tt.wantContent {
				t.Errorf("Content = %q, want %q", result.Content, tt.wantContent)
			}
			if result.Error != tt.wantError {
				t.Errorf("Error = %q, want %q", result.Error, tt.wantError)
			}
			if result.SessionID != tt.wantSessionID {
				t.Errorf("SessionID = %q, want %q", result.SessionID, tt.wantSessionID)
			}
		})
	}
}

func TestInvoke(t *testing.T) {
	tests := []struct {
		name              string
		scriptOutput      string
		wantSessionID     string
		wantContentPart   string
		wantSessionIDLogs bool
	}{
		{
			name:              "output with session_id",
			scriptOutput:      `{"content":"Task completed successfully","error":"","session_id":"session-abc-123"}`,
			wantSessionID:     "session-abc-123",
			wantContentPart:   "Task completed successfully",
			wantSessionIDLogs: true,
		},
		{
			name:              "output without session_id",
			scriptOutput:      `{"content":"Task completed successfully","error":""}`,
			wantSessionID:     "",
			wantContentPart:   "Task completed successfully",
			wantSessionIDLogs: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test files
			tmpDir := t.TempDir()

			// Create a mock executable script that simulates claude CLI
			mockScript := filepath.Join(tmpDir, "mock-claude")
			scriptContent := fmt.Sprintf(`#!/bin/sh
echo '%s'
exit 0
`, tt.scriptOutput)
			err := os.WriteFile(mockScript, []byte(scriptContent), 0755)
			if err != nil {
				t.Fatal(err)
			}

			// Create invoker with mock claude path
			inv := &Invoker{
				ClaudePath: mockScript,
			}

			// Create test task
			task := models.Task{
				Number:        "1",
				Name:          "Test Task",
				Prompt:        "Do something",
				EstimatedTime: 30 * time.Minute,
			}

			// Invoke with context
			ctx := context.Background()
			result, err := inv.Invoke(ctx, task)

			if err != nil {
				t.Fatalf("Invoke() error = %v", err)
			}

			if result == nil {
				t.Fatal("Invoke() returned nil result")
			}

			// Verify SessionID in result
			if result.SessionID != tt.wantSessionID {
				t.Errorf("result.SessionID = %q, want %q", result.SessionID, tt.wantSessionID)
			}

			// Verify SessionID in AgentResponse
			if result.AgentResponse != nil && result.AgentResponse.SessionID != tt.wantSessionID {
				t.Errorf("result.AgentResponse.SessionID = %q, want %q", result.AgentResponse.SessionID, tt.wantSessionID)
			}

			// Parse and verify output
			output, err := ParseClaudeOutput(result.Output)
			if err != nil {
				t.Fatalf("ParseClaudeOutput() error = %v", err)
			}

			if !strings.Contains(output.Content, tt.wantContentPart) {
				t.Errorf("Output content = %q, want to contain %q", output.Content, tt.wantContentPart)
			}

			// Verify session_id in parsed output
			if output.SessionID != tt.wantSessionID {
				t.Errorf("output.SessionID = %q, want %q", output.SessionID, tt.wantSessionID)
			}
		})
	}
}

func TestInvokeWithTimeout(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()

	// Create a mock script that sleeps
	mockScript := filepath.Join(tmpDir, "mock-claude-slow")
	scriptContent := `#!/bin/sh
sleep 10
echo '{"content":"Done","error":""}'
exit 0
`
	err := os.WriteFile(mockScript, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create invoker with mock claude path
	inv := &Invoker{
		ClaudePath: mockScript,
	}

	// Create test task
	task := models.Task{
		Number:        "1",
		Name:          "Slow Task",
		Prompt:        "Do something slow",
		EstimatedTime: 30 * time.Minute,
	}

	// Invoke with very short timeout
	result, err := inv.InvokeWithTimeout(task, 100*time.Millisecond)

	if err != nil {
		t.Fatalf("InvokeWithTimeout() error = %v", err)
	}

	if result == nil {
		t.Fatal("InvokeWithTimeout() returned nil result")
	}

	// Verify that it timed out (non-zero exit code or error)
	if result.ExitCode == 0 && result.Error == nil {
		t.Error("Expected timeout to cause non-zero exit code or error")
	}
}

func TestInvokeContext(t *testing.T) {
	// Create a mock executable that responds to cancellation
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-claude")
	scriptContent := `#!/bin/sh
echo '{"content":"Result","error":""}'
exit 0
`
	err := os.WriteFile(mockScript, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatal(err)
	}

	inv := &Invoker{
		ClaudePath: mockScript,
	}

	task := models.Task{
		Number:        "1",
		Name:          "Test",
		Prompt:        "Test prompt",
		EstimatedTime: 1 * time.Minute,
	}

	// Test with normal context
	ctx := context.Background()
	result, err := inv.Invoke(ctx, task)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestClaudeOutputMarshaling(t *testing.T) {
	// Test that ClaudeOutput can be marshaled/unmarshaled correctly
	original := &ClaudeOutput{
		Content:   "Test content",
		Error:     "Test error",
		SessionID: "test-session-123",
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	// Unmarshal back
	var parsed ClaudeOutput
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	// Verify
	if parsed.Content != original.Content {
		t.Errorf("Content = %q, want %q", parsed.Content, original.Content)
	}
	if parsed.Error != original.Error {
		t.Errorf("Error = %q, want %q", parsed.Error, original.Error)
	}
	if parsed.SessionID != original.SessionID {
		t.Errorf("SessionID = %q, want %q", parsed.SessionID, original.SessionID)
	}
}

func TestParseAgentJSON(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    *models.AgentResponse
		wantErr bool
	}{
		{
			name:   "valid JSON",
			output: `{"status":"success","summary":"Done","output":"result","errors":[],"files_modified":[],"metadata":{}}`,
			want: &models.AgentResponse{
				Status:   "success",
				Summary:  "Done",
				Output:   "result",
				Errors:   []string{},
				Files:    []string{},
				Metadata: map[string]interface{}{},
			},
			wantErr: false,
		},
		{
			name:   "valid JSON with errors",
			output: `{"status":"failed","summary":"Build failed","output":"compilation errors","errors":["undefined variable"],"files_modified":["main.go"],"metadata":{}}`,
			want: &models.AgentResponse{
				Status:   "failed",
				Summary:  "Build failed",
				Output:   "compilation errors",
				Errors:   []string{"undefined variable"},
				Files:    []string{"main.go"},
				Metadata: map[string]interface{}{},
			},
			wantErr: false,
		},
		{
			name:    "malformed JSON - schema enforcement prevents this",
			output:  "plain text output",
			wantErr: true,
		},
		{
			name:    "empty response - schema enforcement prevents this",
			output:  "",
			wantErr: true,
		},
		{
			name:    "partial JSON - schema enforcement prevents this",
			output:  `{"status":"success"`,
			wantErr: true,
		},
		{
			name:   "JSON with extra fields",
			output: `{"status":"success","summary":"Done","output":"result","errors":[],"files_modified":[],"metadata":{},"extra":"ignored"}`,
			want: &models.AgentResponse{
				Status:   "success",
				Summary:  "Done",
				Output:   "result",
				Errors:   []string{},
				Files:    []string{},
				Metadata: map[string]interface{}{},
			},
			wantErr: false,
		},
		{
			name:    "invalid status value - schema enforcement prevents this",
			output:  `{"status":"invalid","summary":"Done","output":"result","errors":[],"files_modified":[],"metadata":{}}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAgentJSON(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAgentJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Skip assertion checks if we expected an error
			if tt.wantErr {
				return
			}
			if got.Status != tt.want.Status {
				t.Errorf("Status = %v, want %v", got.Status, tt.want.Status)
			}
			if got.Summary != tt.want.Summary {
				t.Errorf("Summary = %v, want %v", got.Summary, tt.want.Summary)
			}
			if got.Output != tt.want.Output {
				t.Errorf("Output = %v, want %v", got.Output, tt.want.Output)
			}
			if len(got.Errors) != len(tt.want.Errors) {
				t.Errorf("Errors length = %v, want %v", len(got.Errors), len(tt.want.Errors))
			}
			if len(got.Files) != len(tt.want.Files) {
				t.Errorf("Files length = %v, want %v", len(got.Files), len(tt.want.Files))
			}
		})
	}
}

func TestBuildCommandArgsWithAgentsFlag(t *testing.T) {
	tests := []struct {
		name          string
		task          models.Task
		setupRegistry func(*testing.T) *Registry
		wantChecks    []func(*testing.T, []string)
	}{
		{
			name: "task with agent - includes --agents flag with JSON definition",
			task: models.Task{
				Number:        "1",
				Name:          "Go Development Task",
				Prompt:        "Write Go code",
				Agent:         "golang-pro",
				EstimatedTime: 30 * time.Minute,
			},
			setupRegistry: func(t *testing.T) *Registry {
				tmpDir := t.TempDir()
				agentContent := `---
name: golang-pro
description: Go development expert
tools:
  - Read
  - Write
  - Edit
---
Go development content
`
				err := os.WriteFile(filepath.Join(tmpDir, "golang-pro.md"), []byte(agentContent), 0644)
				if err != nil {
					t.Fatal(err)
				}
				registry := NewRegistry(tmpDir)
				_, err = registry.Discover()
				if err != nil {
					t.Fatal(err)
				}
				return registry
			},
			wantChecks: []func(*testing.T, []string){
				// Check 1: --agents flag must be present
				func(t *testing.T, args []string) {
					hasAgentsFlag := false
					for _, arg := range args {
						if arg == "--agents" {
							hasAgentsFlag = true
							break
						}
					}
					if !hasAgentsFlag {
						t.Error("Command must have --agents flag when agent is specified")
					}
				},
				// Check 1.5: --json-schema flag must be present
				func(t *testing.T, args []string) {
					hasJsonSchemaFlag := false
					for _, arg := range args {
						if arg == "--json-schema" {
							hasJsonSchemaFlag = true
							break
						}
					}
					if !hasJsonSchemaFlag {
						t.Error("Command must have --json-schema flag for structured responses")
					}
				},
				// Check 2: --agents must come before -p flag
				func(t *testing.T, args []string) {
					agentsFlagIdx := -1
					pFlagIdx := -1
					for i, arg := range args {
						if arg == "--agents" {
							agentsFlagIdx = i
						}
						if arg == "-p" {
							pFlagIdx = i
						}
					}
					if agentsFlagIdx >= 0 && pFlagIdx >= 0 && agentsFlagIdx >= pFlagIdx {
						t.Error("--agents flag must come before -p flag")
					}
				},
				// Check 2.5: --json-schema must come before -p flag
				func(t *testing.T, args []string) {
					jsonSchemaIdx := -1
					pFlagIdx := -1
					for i, arg := range args {
						if arg == "--json-schema" {
							jsonSchemaIdx = i
						}
						if arg == "-p" {
							pFlagIdx = i
						}
					}
					if jsonSchemaIdx >= 0 && pFlagIdx >= 0 && jsonSchemaIdx >= pFlagIdx {
						t.Error("--json-schema flag must come before -p flag")
					}
				},
				// Check 3: --agents value must be valid JSON
				func(t *testing.T, args []string) {
					for i, arg := range args {
						if arg == "--agents" && i+1 < len(args) {
							var agentMap map[string]interface{}
							err := json.Unmarshal([]byte(args[i+1]), &agentMap)
							if err != nil {
								t.Errorf("--agents value must be valid JSON: %v", err)
							}
							return
						}
					}
					t.Error("--agents flag present but no value found")
				},
				// Check 4: JSON must contain agent name as key
				func(t *testing.T, args []string) {
					for i, arg := range args {
						if arg == "--agents" && i+1 < len(args) {
							var agentMap map[string]interface{}
							json.Unmarshal([]byte(args[i+1]), &agentMap)
							if _, exists := agentMap["golang-pro"]; !exists {
								t.Error("--agents JSON must contain 'golang-pro' as key")
							}
							return
						}
					}
				},
				// Check 5: Agent definition must include description field
				func(t *testing.T, args []string) {
					for i, arg := range args {
						if arg == "--agents" && i+1 < len(args) {
							var agentMap map[string]map[string]interface{}
							json.Unmarshal([]byte(args[i+1]), &agentMap)
							if agent, exists := agentMap["golang-pro"]; exists {
								if _, hasDesc := agent["description"]; !hasDesc {
									t.Error("Agent definition must include 'description' field")
								}
							}
							return
						}
					}
				},
				// Check 6: Agent definition must include tools field
				func(t *testing.T, args []string) {
					for i, arg := range args {
						if arg == "--agents" && i+1 < len(args) {
							var agentMap map[string]map[string]interface{}
							json.Unmarshal([]byte(args[i+1]), &agentMap)
							if agent, exists := agentMap["golang-pro"]; exists {
								if _, hasTools := agent["tools"]; !hasTools {
									t.Error("Agent definition must include 'tools' field")
								}
							}
							return
						}
					}
				},
				// Check 7: Agent reference prefix removed in e431bfd - no longer needed
				// Previous checks already verify --agents flag structure
			},
		},
		{
			name: "task without agent - no --agents flag but has --json-schema",
			task: models.Task{
				Number:        "2",
				Name:          "Simple Task",
				Prompt:        "Do something",
				Agent:         "",
				EstimatedTime: 30 * time.Minute,
			},
			setupRegistry: nil,
			wantChecks: []func(*testing.T, []string){
				func(t *testing.T, args []string) {
					hasAgentsFlag := false
					for _, arg := range args {
						if arg == "--agents" {
							hasAgentsFlag = true
							break
						}
					}
					if hasAgentsFlag {
						t.Error("Command should not have --agents flag when no agent specified")
					}
				},
				func(t *testing.T, args []string) {
					hasJsonSchemaFlag := false
					for _, arg := range args {
						if arg == "--json-schema" {
							hasJsonSchemaFlag = true
							break
						}
					}
					if !hasJsonSchemaFlag {
						t.Error("Command must have --json-schema flag even without agent")
					}
				},
			},
		},
		{
			name: "task with agent that doesn't exist - no --agents flag",
			task: models.Task{
				Number:        "3",
				Name:          "Task with missing agent",
				Prompt:        "Do work",
				Agent:         "nonexistent",
				EstimatedTime: 30 * time.Minute,
			},
			setupRegistry: func(t *testing.T) *Registry {
				tmpDir := t.TempDir()
				registry := NewRegistry(tmpDir)
				return registry
			},
			wantChecks: []func(*testing.T, []string){
				func(t *testing.T, args []string) {
					hasAgentsFlag := false
					for _, arg := range args {
						if arg == "--agents" {
							hasAgentsFlag = true
							break
						}
					}
					if hasAgentsFlag {
						t.Error("Command should not have --agents flag when agent doesn't exist in registry")
					}
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up invoker with or without registry
			var inv *Invoker
			if tt.setupRegistry != nil {
				registry := tt.setupRegistry(t)
				inv = NewInvokerWithRegistry(registry)
			} else {
				inv = NewInvoker()
			}

			// Build command args
			args := inv.BuildCommandArgs(tt.task)

			// Run all validation checks
			for _, check := range tt.wantChecks {
				check(t, args)
			}
		})
	}
}

// TestRateLimitDetection verifies rate limit detection via budget package
// (Consolidated to use budget.ParseRateLimitFromOutput as single source of truth v2.28+)
func TestRateLimitDetection(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name:   "out of extra usage message",
			output: "You're out of extra usage · resets 1am (Europe/Dublin)",
			want:   true,
		},
		{
			name:   "out of usage without extra",
			output: "You're out of usage for this period",
			want:   true,
		},
		{
			name:   "rate limit message",
			output: "rate limit exceeded",
			want:   true,
		},
		{
			name:   "rate_limit with underscore",
			output: "Error: rate_limit_error",
			want:   true,
		},
		{
			name:   "429 error",
			output: "HTTP 429: Too Many Requests",
			want:   true,
		},
		{
			name:   "usage limit",
			output: "usage limit reached",
			want:   true,
		},
		{
			name:   "too many requests",
			output: "too many requests, please try again",
			want:   true,
		},
		{
			name:   "normal JSON response",
			output: `{"status":"success","summary":"Done"}`,
			want:   false,
		},
		{
			name:   "normal error message",
			output: "Error: file not found",
			want:   false,
		},
		{
			name:   "empty output",
			output: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := budget.ParseRateLimitFromOutput(tt.output) != nil
			if got != tt.want {
				t.Errorf("budget.ParseRateLimitFromOutput(%q) detected=%v, want %v", tt.output, got, tt.want)
			}
		})
	}
}

func TestErrRateLimit(t *testing.T) {
	err := &ErrRateLimit{RawMessage: "You're out of extra usage · resets 1am (Europe/Dublin)"}

	expected := "rate limit: You're out of extra usage · resets 1am (Europe/Dublin)"
	if err.Error() != expected {
		t.Errorf("ErrRateLimit.Error() = %q, want %q", err.Error(), expected)
	}
}

func TestBuildCommandArgsSystemPrompt(t *testing.T) {
	tests := []struct {
		name     string
		taskName string
	}{
		{
			name:     "QC Review task includes --system-prompt",
			taskName: "QC Review: Task 1 Implementation",
		},
		{
			name:     "Regular task also includes --system-prompt",
			taskName: "Implement Feature",
		},
		{
			name:     "Another regular task includes --system-prompt",
			taskName: "Add Unit Tests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := NewInvoker()
			task := models.Task{
				Number:        "1",
				Name:          tt.taskName,
				Prompt:        "Do work",
				EstimatedTime: 30 * time.Minute,
			}

			args := inv.BuildCommandArgs(task)

			// Check if --system-prompt is present
			hasSystemPrompt := false
			for i, arg := range args {
				if arg == "--system-prompt" {
					hasSystemPrompt = true
					// Verify there's a value after it that mentions JSON
					if i+1 < len(args) && strings.Contains(args[i+1], "JSON") {
						// Expected format
						break
					}
				}
			}

			if !hasSystemPrompt {
				t.Errorf("Task %q should have --system-prompt flag", tt.taskName)
			}

			// Verify --system-prompt comes after --json-schema
			systemPromptIdx := -1
			jsonSchemaIdx := -1
			for i, arg := range args {
				if arg == "--system-prompt" {
					systemPromptIdx = i
				}
				if arg == "--json-schema" {
					jsonSchemaIdx = i
				}
			}
			if jsonSchemaIdx >= 0 && systemPromptIdx >= 0 && jsonSchemaIdx >= systemPromptIdx {
				t.Error("--system-prompt should come after --json-schema")
			}
		})
	}
}

func TestBuildCommandArgsWithResumeSessionID(t *testing.T) {
	tests := []struct {
		name            string
		resumeSessionID string
		wantResume      bool
	}{
		{
			name:            "task with resume session ID includes --resume flag",
			resumeSessionID: "session-abc-123",
			wantResume:      true,
		},
		{
			name:            "task without resume session ID has no --resume flag",
			resumeSessionID: "",
			wantResume:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := NewInvoker()
			task := models.Task{
				Number:          "1",
				Name:            "Test Task",
				Prompt:          "Do work",
				EstimatedTime:   30 * time.Minute,
				ResumeSessionID: tt.resumeSessionID,
			}

			args := inv.BuildCommandArgs(task)

			// Check if --resume is present with correct value
			hasResume := false
			for i, arg := range args {
				if arg == "--resume" {
					hasResume = true
					if i+1 < len(args) && args[i+1] == tt.resumeSessionID {
						// Correct session ID
					} else if tt.wantResume {
						t.Errorf("--resume flag has wrong value, got %v, want %s", args[i+1:], tt.resumeSessionID)
					}
				}
			}

			if hasResume != tt.wantResume {
				t.Errorf("--resume flag present = %v, want %v", hasResume, tt.wantResume)
			}

			// Verify --resume comes first (before other flags)
			if tt.wantResume && len(args) > 0 && args[0] != "--resume" {
				t.Errorf("--resume should be first flag, got %s", args[0])
			}
		})
	}
}
