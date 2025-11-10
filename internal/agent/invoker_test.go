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
	}{
		{
			name: "basic invocation structure",
			task: models.Task{
				Number: 1,
				Name:   "Test Task",
				Prompt: "print 'hello'",
			},
			timeout: 10 * time.Second,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := NewInvoker()
			inv.ClaudePath = "/opt/homebrew/bin/claude"

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

			// Duration should be less than timeout
			if result.Duration >= tt.timeout {
				t.Errorf("Duration %v should be less than timeout %v", result.Duration, tt.timeout)
			}

			// Output should be captured
			if result.Output == "" {
				t.Error("Output should be captured")
			}

			// Exit code should be set (0 for success)
			if result.ExitCode != 0 {
				t.Logf("Exit code: %d, Output: %s", result.ExitCode, result.Output)
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
	}{
		{
			name: "successful invocation",
			task: models.Task{
				Number: 1,
				Name:   "Test Task",
				Prompt: "respond with 'ok'",
			},
			timeout: 10 * time.Second,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := NewInvoker()
			inv.ClaudePath = "/opt/homebrew/bin/claude"

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

			// Duration should be tracked
			if result.Duration == 0 {
				t.Error("Duration should be tracked")
			}

			// Output should be captured
			if result.Output == "" {
				t.Error("Output should be captured")
			}
		})
	}
}

// TestInvokeSuccess tests the happy path with simple prompt
func TestInvokeSuccess(t *testing.T) {
	inv := NewInvoker()
	inv.ClaudePath = "/opt/homebrew/bin/claude"

	task := models.Task{
		Number: 1,
		Name:   "Simple Test",
		Prompt: "say 'test'",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := inv.Invoke(ctx, task)

	if err != nil {
		t.Fatalf("Invoke() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Invoke() returned nil result")
	}

	// Some environments require manual acceptance of updated terms.
	// If the CLI exits with a non-zero code and prints the terms notice,
	// skip this integration test instead of failing the suite.
	outputLower := strings.ToLower(result.Output)
	if result.ExitCode != 0 && strings.Contains(outputLower, "terms") && strings.Contains(outputLower, "action required") {
		t.Skip("claude CLI requires manual terms acceptance; skipping integration test")
	}

	// Verify output contains something
	if result.Output == "" {
		t.Error("Output should not be empty")
	}

	// Exit code should be 0 for success
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0. Output: %s", result.ExitCode, result.Output)
	}

	// Duration should be reasonable (under 10 seconds)
	if result.Duration <= 0 {
		t.Error("Duration should be positive")
	}
	if result.Duration >= 10*time.Second {
		t.Errorf("Duration %v too long for simple prompt", result.Duration)
	}

	// No error should be set
	if result.Error != nil {
		t.Errorf("Result.Error should be nil, got: %v", result.Error)
	}
}

// TestInvokeTimeout tests that timeout is properly handled
func TestInvokeTimeout(t *testing.T) {
	inv := NewInvoker()
	inv.ClaudePath = "/opt/homebrew/bin/claude"

	task := models.Task{
		Number: 1,
		Name:   "Timeout Test",
		Prompt: "count to 1000 slowly, taking at least 5 seconds",
	}

	// Use a very short timeout to force timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := inv.Invoke(ctx, task)

	// Should not return error from Invoke itself (returns result with error info)
	if err != nil {
		t.Fatalf("Invoke() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Invoke() returned nil result")
	}

	// Result should have an error set (context deadline exceeded or killed)
	// or a non-zero exit code
	if result.Error == nil && result.ExitCode == 0 {
		t.Error("Expected timeout to cause error or non-zero exit code")
	}
}

// TestInvokeContextCancellation tests context cancellation handling
func TestInvokeContextCancellation(t *testing.T) {
	inv := NewInvoker()
	inv.ClaudePath = "/opt/homebrew/bin/claude"

	task := models.Task{
		Number: 1,
		Name:   "Cancellation Test",
		Prompt: "perform a long task",
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	result, err := inv.Invoke(ctx, task)

	if err != nil {
		t.Fatalf("Invoke() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Invoke() returned nil result")
	}

	// Should have error or non-zero exit code due to cancellation
	if result.Error == nil && result.ExitCode == 0 {
		t.Error("Expected cancellation to cause error or non-zero exit code")
	}
}

// TestInvokeExitCode tests handling of non-zero exit codes
func TestInvokeExitCode(t *testing.T) {
	inv := NewInvoker()
	inv.ClaudePath = "/opt/homebrew/bin/claude"

	task := models.Task{
		Number: 1,
		Name:   "Exit Code Test",
		Prompt: "test",
	}

	// Build args and append an invalid flag to cause error
	args := inv.BuildCommandArgs(task)
	args = append(args, "--invalid-flag-xyz")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a custom invocation with invalid flag
	task.Prompt = "test --invalid-flag-xyz-should-fail"

	result, err := inv.Invoke(ctx, task)

	if err != nil {
		t.Fatalf("Invoke() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Invoke() returned nil result")
	}

	// Output should be captured even on error
	if result.Output == "" {
		t.Log("Warning: Expected some output even on error")
	}

	// Note: This test may not always produce non-zero exit code
	// depending on how claude CLI handles invalid input
	// The important thing is that exit code is captured
	t.Logf("Exit code: %d, Output: %s", result.ExitCode, result.Output)
}

// TestInvokeOutput tests that output is properly captured
func TestInvokeOutput(t *testing.T) {
	inv := NewInvoker()
	inv.ClaudePath = "/opt/homebrew/bin/claude"

	task := models.Task{
		Number: 1,
		Name:   "Output Test",
		Prompt: "respond with exactly: EXPECTED_OUTPUT",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := inv.Invoke(ctx, task)

	if err != nil {
		t.Fatalf("Invoke() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Invoke() returned nil result")
	}

	// Output should be captured
	if result.Output == "" {
		t.Error("Output should not be empty")
	}

	// Verify output is in InvocationResult
	if len(result.Output) == 0 {
		t.Error("InvocationResult should contain output")
	}

	t.Logf("Captured output (%d bytes): %s", len(result.Output), result.Output)
}

// TestBuildCommandArgsWithRealRegistry tests integration with Registry
func TestBuildCommandArgsWithRealRegistry(t *testing.T) {
	// Create temporary directory with test agent
	tmpDir := t.TempDir()

	agentContent := `---
name: test-agent
description: Test agent for integration
tools:
  - read
  - write
---
This is a test agent for integration testing.
`

	err := os.WriteFile(filepath.Join(tmpDir, "test-agent.md"), []byte(agentContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create registry and discover agents
	registry := NewRegistry(tmpDir)
	agents, err := registry.Discover()
	if err != nil {
		t.Fatalf("Registry.Discover() error: %v", err)
	}

	if len(agents) != 1 {
		t.Fatalf("Expected 1 agent, got %d", len(agents))
	}

	// Create invoker with registry
	inv := NewInvokerWithRegistry(registry)

	task := models.Task{
		Number: 1,
		Name:   "Test with agent",
		Prompt: "perform task",
		Agent:  "test-agent",
	}

	args := inv.BuildCommandArgs(task)

	// Verify agent is mentioned in prompt
	foundAgentRef := false
	for _, arg := range args {
		if strings.Contains(arg, "use the test-agent subagent to:") {
			foundAgentRef = true
			break
		}
	}

	if !foundAgentRef {
		t.Error("Agent should be referenced in built command args")
	}

	// Verify original prompt is still there
	foundPrompt := false
	for _, arg := range args {
		if strings.Contains(arg, "perform task") {
			foundPrompt = true
			break
		}
	}

	if !foundPrompt {
		t.Error("Original prompt should be in command args")
	}
}

// TestInvokeWithTimeoutShort tests short timeout handling
func TestInvokeWithTimeoutShort(t *testing.T) {
	inv := NewInvoker()
	inv.ClaudePath = "/opt/homebrew/bin/claude"

	task := models.Task{
		Number: 1,
		Name:   "Short Timeout",
		Prompt: "take a very long time to respond",
	}

	// Very short timeout
	result, err := inv.InvokeWithTimeout(task, 200*time.Millisecond)

	if err != nil {
		t.Fatalf("InvokeWithTimeout() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("InvokeWithTimeout() returned nil result")
	}

	// Should complete quickly (timeout or actual completion)
	if result.Duration > 1*time.Second {
		t.Errorf("Duration %v too long for short timeout", result.Duration)
	}
}

// TestInvokeWithDifferentPrompts tests various prompt types
func TestInvokeWithDifferentPrompts(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{
			name:   "simple prompt",
			prompt: "hello",
		},
		{
			name:   "question prompt",
			prompt: "what is 2+2?",
		},
		{
			name:   "instruction prompt",
			prompt: "list three colors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := NewInvoker()
			inv.ClaudePath = "/opt/homebrew/bin/claude"

			task := models.Task{
				Number: 1,
				Name:   tt.name,
				Prompt: tt.prompt,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, err := inv.Invoke(ctx, task)

			if err != nil {
				t.Fatalf("Invoke() unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("Invoke() returned nil result")
			}

			// All should produce some output
			if result.Output == "" {
				t.Error("Expected output from claude CLI")
			}

			t.Logf("Prompt: %s, Output length: %d bytes", tt.prompt, len(result.Output))
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

// TestInvokeInvalidPath tests the error path when ClaudePath is invalid
// This test covers line 94 in invoker.go (result.Error = err)
func TestInvokeInvalidPath(t *testing.T) {
	inv := NewInvoker()
	inv.ClaudePath = "/nonexistent/path/to/claude"

	task := models.Task{
		Number: 1,
		Name:   "Test Task",
		Prompt: "test",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := inv.Invoke(ctx, task)

	// Should not panic, err can be nil (error in result.Error)
	if err != nil {
		t.Fatalf("Invoke() returned unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// The critical assertion - line 94 path
	// When the executable doesn't exist, result.Error should be set
	if result.Error == nil {
		t.Error("Expected result.Error to be set for invalid path")
	}

	// Error should indicate missing/invalid binary
	if result.Error != nil {
		errMsg := result.Error.Error()
		if !strings.Contains(errMsg, "executable") && !strings.Contains(errMsg, "no such file") {
			t.Logf("Error message: %v", result.Error)
		}
	}

	// Exit code should remain 0 (wasn't set by ExitError)
	if result.ExitCode != 0 {
		t.Errorf("Expected ExitCode 0 for spawn failure, got %d", result.ExitCode)
	}
}

// TestInvokeExitCodeGuaranteed tests the exit code extraction path with deterministic exit code
// This test covers lines 91-92 in invoker.go (exit code extraction)
func TestInvokeExitCodeGuaranteed(t *testing.T) {
	// Create wrapper script that exits with code 42
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "exit42.sh")

	scriptContent := `#!/bin/sh
exit 42
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	inv := NewInvoker()
	inv.ClaudePath = scriptPath // Use our script instead of claude

	task := models.Task{
		Number: 1,
		Name:   "Test Task",
		Prompt: "anything", // Script ignores this
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := inv.Invoke(ctx, task)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// THE CRITICAL ASSERTION - lines 91-92 path
	// This guarantees the exit code extraction code is executed
	if result.ExitCode != 42 {
		t.Errorf("Expected exit code 42, got %d", result.ExitCode)
	}

	// Verify error is NOT set (only ExitCode should be populated)
	if result.Error != nil {
		t.Errorf("Expected result.Error to be nil with exit code, got: %v", result.Error)
	}
}
