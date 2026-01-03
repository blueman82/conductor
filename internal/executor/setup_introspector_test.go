package executor

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSetupWaiterLogger implements budget.WaiterLogger for testing
type mockSetupWaiterLogger struct {
	countdownCalls int
	announceCalls  int
}

func (m *mockSetupWaiterLogger) LogRateLimitCountdown(remaining, total time.Duration) {
	m.countdownCalls++
}

func (m *mockSetupWaiterLogger) LogRateLimitAnnounce(remaining, total time.Duration) {
	m.announceCalls++
}

func TestNewSetupIntrospector(t *testing.T) {
	si := NewSetupIntrospector(90*time.Second, nil)

	if si.Timeout != 90*time.Second {
		t.Errorf("Timeout = %v, want 90s", si.Timeout)
	}
	if si.ClaudePath != "claude" {
		t.Errorf("Default ClaudePath = %s, want claude", si.ClaudePath)
	}
	if si.Logger != nil {
		t.Errorf("Logger should be nil when not provided")
	}
}

func TestNewSetupIntrospector_CustomTimeout(t *testing.T) {
	si := NewSetupIntrospector(45*time.Second, nil)

	if si.Timeout != 45*time.Second {
		t.Errorf("Timeout = %v, want 45s", si.Timeout)
	}
	if si.ClaudePath != "claude" {
		t.Errorf("ClaudePath = %s, want claude", si.ClaudePath)
	}
	if si.Logger != nil {
		t.Errorf("Logger should be nil when not provided")
	}
}

func TestNewSetupIntrospectorWithLogger(t *testing.T) {
	logger := &mockSetupWaiterLogger{}
	si := NewSetupIntrospector(90*time.Second, logger)

	if si.Logger == nil {
		t.Error("Logger should not be nil when provided")
	}
	if si.Logger != logger {
		t.Error("Logger should be the one provided")
	}
	if si.Timeout != 90*time.Second {
		t.Errorf("Timeout = %v, want 90s", si.Timeout)
	}
}

func TestSetupSchema(t *testing.T) {
	schema := SetupSchema()

	if schema == "" {
		t.Error("SetupSchema returned empty string")
	}

	// Check required fields are in schema
	if !strings.Contains(schema, "commands") {
		t.Error("Schema should contain commands")
	}
	if !strings.Contains(schema, "reasoning") {
		t.Error("Schema should contain reasoning")
	}
	if !strings.Contains(schema, `"required"`) {
		t.Error("Schema should specify required fields")
	}
}

func TestSetupSchema_ValidJSON(t *testing.T) {
	schema := SetupSchema()

	// Verify schema is valid JSON by checking key structure
	if !strings.Contains(schema, `"type": "object"`) && !strings.Contains(schema, `"type":"object"`) {
		t.Error("Schema should define type as object")
	}
	if !strings.Contains(schema, `"properties"`) {
		t.Error("Schema should define properties")
	}
}

func TestSetupSchema_CommandsArray(t *testing.T) {
	schema := SetupSchema()

	// Commands should be defined as array
	if !strings.Contains(schema, `"commands"`) {
		t.Error("Schema should define commands property")
	}
	if !strings.Contains(schema, `"array"`) {
		t.Error("Schema should specify commands as array type")
	}
}

func TestSetupSchema_CommandItemStructure(t *testing.T) {
	schema := SetupSchema()

	// Command items should have required fields
	if !strings.Contains(schema, `"command"`) {
		t.Error("Schema should define command property in items")
	}
	if !strings.Contains(schema, `"purpose"`) {
		t.Error("Schema should define purpose property in items")
	}
	if !strings.Contains(schema, `"required"`) {
		t.Error("Schema should define required property in items")
	}
}

func TestSetupCommand_Fields(t *testing.T) {
	cmd := SetupCommand{
		Command:  "npm install",
		Purpose:  "Install project dependencies",
		Required: true,
	}

	if cmd.Command != "npm install" {
		t.Errorf("Command = %s, want 'npm install'", cmd.Command)
	}
	if cmd.Purpose != "Install project dependencies" {
		t.Errorf("Purpose = %s, want 'Install project dependencies'", cmd.Purpose)
	}
	if !cmd.Required {
		t.Errorf("Required = %v, want true", cmd.Required)
	}
}

func TestSetupResult_Fields(t *testing.T) {
	result := SetupResult{
		Commands: []SetupCommand{
			{Command: "go mod tidy", Purpose: "Tidy Go modules", Required: true},
			{Command: "go generate ./...", Purpose: "Generate code", Required: false},
		},
		Reasoning: "Go project detected with go.mod file",
	}

	if len(result.Commands) != 2 {
		t.Errorf("Commands length = %d, want 2", len(result.Commands))
	}
	if result.Reasoning != "Go project detected with go.mod file" {
		t.Errorf("Reasoning = %s, want 'Go project detected with go.mod file'", result.Reasoning)
	}
}

func TestSetupResult_JSONParsing(t *testing.T) {
	tests := []struct {
		name          string
		jsonInput     string
		wantCmdCount  int
		wantReasoning string
		wantFirstCmd  string
		wantFirstReq  bool
		wantErr       bool
	}{
		{
			name: "valid with commands",
			jsonInput: `{
				"commands": [
					{"command": "npm install", "purpose": "Install deps", "required": true}
				],
				"reasoning": "Node.js project detected"
			}`,
			wantCmdCount:  1,
			wantReasoning: "Node.js project detected",
			wantFirstCmd:  "npm install",
			wantFirstReq:  true,
			wantErr:       false,
		},
		{
			name: "valid with multiple commands",
			jsonInput: `{
				"commands": [
					{"command": "go mod tidy", "purpose": "Tidy modules", "required": true},
					{"command": "go generate ./...", "purpose": "Generate", "required": false}
				],
				"reasoning": "Go project"
			}`,
			wantCmdCount:  2,
			wantReasoning: "Go project",
			wantFirstCmd:  "go mod tidy",
			wantFirstReq:  true,
			wantErr:       false,
		},
		{
			name: "valid empty commands",
			jsonInput: `{
				"commands": [],
				"reasoning": "No setup required"
			}`,
			wantCmdCount:  0,
			wantReasoning: "No setup required",
			wantErr:       false,
		},
		{
			name:      "invalid json",
			jsonInput: `{not valid json}`,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result SetupResult
			err := json.Unmarshal([]byte(tt.jsonInput), &result)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result.Commands) != tt.wantCmdCount {
				t.Errorf("Commands count = %d, want %d", len(result.Commands), tt.wantCmdCount)
			}
			if result.Reasoning != tt.wantReasoning {
				t.Errorf("Reasoning = %s, want %s", result.Reasoning, tt.wantReasoning)
			}
			if tt.wantCmdCount > 0 {
				if result.Commands[0].Command != tt.wantFirstCmd {
					t.Errorf("First command = %s, want %s", result.Commands[0].Command, tt.wantFirstCmd)
				}
				if result.Commands[0].Required != tt.wantFirstReq {
					t.Errorf("First required = %v, want %v", result.Commands[0].Required, tt.wantFirstReq)
				}
			}
		})
	}
}

func TestSetupResult_JSONParsing_ExtractFromMixed(t *testing.T) {
	// Test extracting JSON from mixed output (like Claude sometimes returns)
	mixedOutput := `Some text before {"commands": [{"command": "npm install", "purpose": "Install deps", "required": true}], "reasoning": "Node project"} some text after`

	// Find JSON in mixed output
	start := strings.Index(mixedOutput, "{")
	end := strings.LastIndex(mixedOutput, "}")
	if start < 0 || end <= start {
		t.Fatal("Could not find JSON in mixed output")
	}

	jsonStr := mixedOutput[start : end+1]
	var result SetupResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("Failed to parse extracted JSON: %v", err)
	}

	if len(result.Commands) != 1 {
		t.Errorf("Commands count = %d, want 1", len(result.Commands))
	}
	if result.Commands[0].Command != "npm install" {
		t.Errorf("Command = %s, want 'npm install'", result.Commands[0].Command)
	}
	if result.Reasoning != "Node project" {
		t.Errorf("Reasoning = %s, want 'Node project'", result.Reasoning)
	}
}

func TestRunSetupCommands_NilResult(t *testing.T) {
	si := NewSetupIntrospector(90*time.Second, nil)
	ctx := context.Background()

	err := si.RunSetupCommands(ctx, nil)

	assert.NoError(t, err, "RunSetupCommands should handle nil result gracefully")
}

func TestRunSetupCommands_EmptyCommands(t *testing.T) {
	si := NewSetupIntrospector(90*time.Second, nil)
	ctx := context.Background()

	result := &SetupResult{
		Commands:  []SetupCommand{},
		Reasoning: "No setup needed",
	}

	err := si.RunSetupCommands(ctx, result)

	assert.NoError(t, err, "RunSetupCommands should handle empty commands gracefully")
}

func TestRunSetupCommands_SuccessfulCommand(t *testing.T) {
	si := NewSetupIntrospector(90*time.Second, nil)
	ctx := context.Background()

	result := &SetupResult{
		Commands: []SetupCommand{
			{Command: "echo 'test'", Purpose: "Test echo", Required: true},
		},
		Reasoning: "Test command",
	}

	err := si.RunSetupCommands(ctx, result)

	assert.NoError(t, err, "RunSetupCommands should succeed for valid echo command")
}

func TestRunSetupCommands_RequiredFailure(t *testing.T) {
	si := NewSetupIntrospector(90*time.Second, nil)
	ctx := context.Background()

	result := &SetupResult{
		Commands: []SetupCommand{
			{Command: "exit 1", Purpose: "Fail intentionally", Required: true},
		},
		Reasoning: "Test required failure",
	}

	err := si.RunSetupCommands(ctx, result)

	require.Error(t, err, "RunSetupCommands should fail when required command fails")
	assert.Contains(t, err.Error(), "required setup command", "Error should mention required command")
}

func TestRunSetupCommands_OptionalFailure(t *testing.T) {
	si := NewSetupIntrospector(90*time.Second, nil)
	ctx := context.Background()

	result := &SetupResult{
		Commands: []SetupCommand{
			{Command: "exit 1", Purpose: "Optional fail", Required: false},
			{Command: "echo 'success'", Purpose: "Should run", Required: true},
		},
		Reasoning: "Test optional failure continues",
	}

	err := si.RunSetupCommands(ctx, result)

	assert.NoError(t, err, "RunSetupCommands should continue when optional command fails")
}

func TestRunSetupCommands_MultipleCommands(t *testing.T) {
	si := NewSetupIntrospector(90*time.Second, nil)
	ctx := context.Background()

	result := &SetupResult{
		Commands: []SetupCommand{
			{Command: "echo 'first'", Purpose: "First command", Required: true},
			{Command: "echo 'second'", Purpose: "Second command", Required: true},
			{Command: "echo 'third'", Purpose: "Third command", Required: false},
		},
		Reasoning: "Multiple test commands",
	}

	err := si.RunSetupCommands(ctx, result)

	assert.NoError(t, err, "RunSetupCommands should run all commands successfully")
}

func TestRunSetupCommands_ContextCancellation(t *testing.T) {
	si := NewSetupIntrospector(90*time.Second, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := &SetupResult{
		Commands: []SetupCommand{
			{Command: "sleep 10", Purpose: "Long command", Required: true},
		},
		Reasoning: "Test cancellation",
	}

	err := si.RunSetupCommands(ctx, result)

	require.Error(t, err, "RunSetupCommands should fail on cancelled context")
}

// SetupHook nil-safety tests (following warmup_hook_test.go pattern)

func TestNewSetupHook_NilSafety(t *testing.T) {
	t.Run("returns nil for nil introspector", func(t *testing.T) {
		hook := NewSetupHook(nil, nil)
		assert.Nil(t, hook)
	})

	t.Run("creates hook with valid introspector", func(t *testing.T) {
		introspector := NewSetupIntrospector(90*time.Second, nil)
		hook := NewSetupHook(introspector, nil)
		assert.NotNil(t, hook)
	})
}

func TestSetupHook_Setup_NilHook(t *testing.T) {
	ctx := context.Background()

	var hook *SetupHook
	err := hook.Setup(ctx)

	assert.NoError(t, err, "Setup should gracefully handle nil hook")
}

func TestSetupHook_Setup_NilIntrospector(t *testing.T) {
	ctx := context.Background()

	// This shouldn't happen in practice due to NewSetupHook check,
	// but test defensive programming
	hook := &SetupHook{
		introspector: nil,
		logger:       nil,
	}

	err := hook.Setup(ctx)

	assert.NoError(t, err, "Setup should gracefully handle nil introspector")
}

// mockSetupLogger implements RuntimeEnforcementLogger for testing
type mockSetupLogger struct {
	warnMessages []string
	infoMessages []string
}

func (m *mockSetupLogger) LogTestCommands(entries []interface{}) {}
func (m *mockSetupLogger) LogCriterionVerifications(entries []interface{}) {
}
func (m *mockSetupLogger) LogDocTargetVerifications(entries []interface{}) {}
func (m *mockSetupLogger) LogErrorPattern(pattern interface{})             {}
func (m *mockSetupLogger) LogDetectedError(detected interface{})           {}
func (m *mockSetupLogger) Warnf(format string, args ...interface{}) {
	m.warnMessages = append(m.warnMessages, format)
}
func (m *mockSetupLogger) Info(message string) {
	m.infoMessages = append(m.infoMessages, message)
}
func (m *mockSetupLogger) Infof(format string, args ...interface{}) {
	m.infoMessages = append(m.infoMessages, format)
}

func TestLogSetupCommandOutput(t *testing.T) {
	tests := []struct {
		name     string
		cmdIndex int
		total    int
		cmd      SetupCommand
		success  bool
		want     string
	}{
		{
			name:     "successful command",
			cmdIndex: 0,
			total:    3,
			cmd:      SetupCommand{Command: "npm install", Purpose: "Install deps"},
			success:  true,
			want:     "[1/3]",
		},
		{
			name:     "failed command",
			cmdIndex: 1,
			total:    3,
			cmd:      SetupCommand{Command: "go mod tidy", Purpose: "Tidy modules"},
			success:  false,
			want:     "[2/3]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := logSetupCommandOutput(tt.cmdIndex, tt.total, tt.cmd, tt.success)

			assert.Contains(t, output, tt.want, "Output should contain command index")
			assert.Contains(t, output, tt.cmd.Purpose, "Output should contain purpose")
			assert.Contains(t, output, tt.cmd.Command, "Output should contain command")

			if tt.success {
				assert.NotContains(t, output, "\u2717", "Success should not have X mark")
			}
		})
	}
}

func TestBuildPrompt_ContainsKeyElements(t *testing.T) {
	// Note: buildPrompt uses os.Getwd() and filepath.Walk, so this tests
	// in the context of the actual working directory
	si := NewSetupIntrospector(90*time.Second, nil)

	prompt, err := si.buildPrompt()

	require.NoError(t, err, "buildPrompt should not error")
	assert.NotEmpty(t, prompt, "buildPrompt should return non-empty prompt")

	// Check key instructions are present
	assert.Contains(t, prompt, "Analyze this project", "Prompt should have analysis instruction")
	assert.Contains(t, prompt, "setup commands", "Prompt should mention setup commands")
	assert.Contains(t, prompt, "required", "Prompt should explain required field")
	assert.Contains(t, prompt, "JSON", "Prompt should request JSON response")
}

func TestBuildPrompt_MentionsProjectFiles(t *testing.T) {
	si := NewSetupIntrospector(90*time.Second, nil)

	prompt, err := si.buildPrompt()

	require.NoError(t, err, "buildPrompt should not error")

	// Should mention project files section
	assert.Contains(t, prompt, "Project files:", "Prompt should have project files section")
}

func TestBuildPrompt_MentionsPackageManagers(t *testing.T) {
	si := NewSetupIntrospector(90*time.Second, nil)

	prompt, err := si.buildPrompt()

	require.NoError(t, err, "buildPrompt should not error")

	// Should mention common package managers
	packageManagers := []string{"npm install", "go mod tidy", "pip install", "cargo build"}
	containsPackageManager := false
	for _, pm := range packageManagers {
		if strings.Contains(prompt, pm) {
			containsPackageManager = true
			break
		}
	}
	assert.True(t, containsPackageManager, "Prompt should mention at least one package manager")
}

func TestBuildPrompt_MentionsProjectIndicators(t *testing.T) {
	si := NewSetupIntrospector(90*time.Second, nil)

	prompt, err := si.buildPrompt()

	require.NoError(t, err, "buildPrompt should not error")

	// Should mention project indicators
	projectIndicators := []string{"package.json", "go.mod", "requirements.txt", "Cargo.toml"}
	containsIndicator := false
	for _, pi := range projectIndicators {
		if strings.Contains(prompt, pi) {
			containsIndicator = true
			break
		}
	}
	assert.True(t, containsIndicator, "Prompt should mention at least one project indicator file")
}

func TestSetupCommand_OptionalVsRequired(t *testing.T) {
	requiredCmd := SetupCommand{
		Command:  "npm install",
		Purpose:  "Install essential dependencies",
		Required: true,
	}

	optionalCmd := SetupCommand{
		Command:  "npm run lint",
		Purpose:  "Run linting",
		Required: false,
	}

	assert.True(t, requiredCmd.Required, "Required command should have Required=true")
	assert.False(t, optionalCmd.Required, "Optional command should have Required=false")
}

// Integration test for SetupHook with Orchestrator config
func TestSetupHook_OrchestratorIntegration(t *testing.T) {
	t.Run("orchestrator config accepts SetupHook", func(t *testing.T) {
		introspector := NewSetupIntrospector(90*time.Second, nil)
		hook := NewSetupHook(introspector, nil)

		config := OrchestratorConfig{
			SetupHook: hook,
		}

		assert.NotNil(t, config.SetupHook, "OrchestratorConfig should accept SetupHook")
	})

	t.Run("orchestrator config handles nil SetupHook", func(t *testing.T) {
		config := OrchestratorConfig{
			SetupHook: nil,
		}

		assert.Nil(t, config.SetupHook, "OrchestratorConfig should accept nil SetupHook")
	})
}
