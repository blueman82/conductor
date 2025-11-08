package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
				Number:        1,
				Name:          "Test Task",
				Prompt:        "Do something",
				Agent:         "",
				EstimatedTime: 30 * time.Minute,
			},
			setupRegistry: nil,
			wantChecks: []func(*testing.T, []string){
				func(t *testing.T, args []string) {
					// Check for -p flag
					hasP := false
					for _, arg := range args {
						if arg == "-p" {
							hasP = true
							break
						}
					}
					if !hasP {
						t.Error("Command should have -p flag")
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
					// Verify prompt is included
					hasPrompt := false
					for _, arg := range args {
						if strings.Contains(arg, "Do something") {
							hasPrompt = true
							break
						}
					}
					if !hasPrompt {
						t.Error("Command should include the task prompt")
					}
				},
			},
		},
		{
			name: "task with agent that exists in registry",
			task: models.Task{
				Number:        2,
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
					// Should have agent reference in prompt
					hasAgentRef := false
					for _, arg := range args {
						if strings.Contains(arg, "use the swiftdev subagent to:") {
							hasAgentRef = true
							break
						}
					}
					if !hasAgentRef {
						t.Error("Command should have agent reference in prompt")
					}
				},
				func(t *testing.T, args []string) {
					// Verify original prompt is included
					hasOriginalPrompt := false
					for _, arg := range args {
						if strings.Contains(arg, "Build iOS app") {
							hasOriginalPrompt = true
							break
						}
					}
					if !hasOriginalPrompt {
						t.Error("Command should include the original task prompt")
					}
				},
			},
		},
		{
			name: "task with agent that doesn't exist",
			task: models.Task{
				Number: 3,
				Name:   "Task with nonexistent agent",
				Prompt: "Do work",
				Agent:  "nonexistent-agent",
			},
			setupRegistry: func(t *testing.T) *Registry {
				tmpDir := t.TempDir()
				return NewRegistry(tmpDir)
			},
			wantChecks: []func(*testing.T, []string){
				func(t *testing.T, args []string) {
					// Should NOT have agent reference
					for _, arg := range args {
						if strings.Contains(arg, "use the nonexistent-agent subagent to:") {
							t.Error("Should not have agent reference for non-existent agent")
						}
					}
				},
			},
		},
		{
			name: "task with agent but no registry",
			task: models.Task{
				Number: 4,
				Name:   "Task with agent but no registry",
				Prompt: "Do work",
				Agent:  "some-agent",
			},
			setupRegistry: nil,
			wantChecks: []func(*testing.T, []string){
				func(t *testing.T, args []string) {
					// Should NOT have agent reference without registry
					for _, arg := range args {
						if strings.Contains(arg, "use the some-agent subagent to:") {
							t.Error("Should not have agent reference when registry is nil")
						}
					}
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := NewInvoker()
			if tt.setupRegistry != nil {
				inv.Registry = tt.setupRegistry(t)
			}

			args := inv.BuildCommandArgs(tt.task)

			if len(args) == 0 {
				t.Fatal("BuildCommandArgs returned empty args")
			}

			for _, check := range tt.wantChecks {
				check(t, args)
			}
		})
	}
}

func TestParseClaudeOutput(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantContent string
		wantError   string
		wantErr     bool
	}{
		{
			name:        "valid JSON output",
			output:      `{"content":"Hello world","error":""}`,
			wantContent: "Hello world",
			wantError:   "",
			wantErr:     false,
		},
		{
			name:        "JSON with error",
			output:      `{"content":"","error":"Something went wrong"}`,
			wantContent: "",
			wantError:   "Something went wrong",
			wantErr:     false,
		},
		{
			name:        "invalid JSON - fallback to raw text",
			output:      "This is not JSON",
			wantContent: "This is not JSON",
			wantError:   "",
			wantErr:     false,
		},
		{
			name:        "empty output",
			output:      "",
			wantContent: "",
			wantError:   "",
			wantErr:     false,
		},
		{
			name:        "partial JSON",
			output:      `{"content":"incomplete"`,
			wantContent: `{"content":"incomplete"`,
			wantError:   "",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseClaudeOutput(tt.output)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseClaudeOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result == nil {
				t.Fatal("ParseClaudeOutput() returned nil result")
			}

			if result.Content != tt.wantContent {
				t.Errorf("Content = %q, want %q", result.Content, tt.wantContent)
			}

			if result.Error != tt.wantError {
				t.Errorf("Error = %q, want %q", result.Error, tt.wantError)
			}
		})
	}
}

func TestInvokeWithTimeout(t *testing.T) {
	tests := []struct {
		name    string
		task    models.Task
		timeout time.Duration
		wantErr bool
		skip    string
	}{
		{
			name: "basic invocation structure",
			task: models.Task{
				Number: 1,
				Name:   "Test Task",
				Prompt: "echo test",
			},
			timeout: 5 * time.Second,
			wantErr: false,
			skip:    "requires claude CLI",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip != "" {
				t.Skip(tt.skip)
			}

			inv := NewInvoker()
			result, err := inv.InvokeWithTimeout(tt.task, tt.timeout)

			if (err != nil) != tt.wantErr {
				t.Errorf("InvokeWithTimeout() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify result structure
			if result == nil {
				t.Fatal("InvokeWithTimeout() returned nil result")
			}

			// Duration should be recorded
			if result.Duration == 0 {
				t.Error("Duration should be recorded")
			}
		})
	}
}

func TestInvoke(t *testing.T) {
	tests := []struct {
		name    string
		task    models.Task
		timeout time.Duration
		wantErr bool
		skip    string
	}{
		{
			name: "context cancellation",
			task: models.Task{
				Number: 1,
				Name:   "Test Task",
				Prompt: "sleep 10",
			},
			timeout: 100 * time.Millisecond,
			wantErr: false,
			skip:    "requires claude CLI",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip != "" {
				t.Skip(tt.skip)
			}

			inv := NewInvoker()
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			result, err := inv.Invoke(ctx, tt.task)

			if (err != nil) != tt.wantErr {
				t.Errorf("Invoke() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result == nil {
				t.Fatal("Invoke() returned nil result")
			}
		})
	}
}

func TestInvocationResult(t *testing.T) {
	// Test that InvocationResult can be created and has expected fields
	result := &InvocationResult{
		Output:   "test output",
		ExitCode: 0,
		Duration: 100 * time.Millisecond,
		Error:    nil,
	}

	if result.Output != "test output" {
		t.Errorf("Output = %s, want 'test output'", result.Output)
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}

	if result.Duration != 100*time.Millisecond {
		t.Errorf("Duration = %v, want 100ms", result.Duration)
	}

	if result.Error != nil {
		t.Errorf("Error = %v, want nil", result.Error)
	}
}

func TestClaudeOutput(t *testing.T) {
	// Test that ClaudeOutput can be marshaled/unmarshaled
	co := ClaudeOutput{
		Content: "test content",
		Error:   "test error",
	}

	data, err := json.Marshal(co)
	if err != nil {
		t.Fatalf("Failed to marshal ClaudeOutput: %v", err)
	}

	var parsed ClaudeOutput
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal ClaudeOutput: %v", err)
	}

	if parsed.Content != co.Content {
		t.Errorf("Content = %s, want %s", parsed.Content, co.Content)
	}

	if parsed.Error != co.Error {
		t.Errorf("Error = %s, want %s", parsed.Error, co.Error)
	}
}
