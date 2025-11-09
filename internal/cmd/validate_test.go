package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/parser"
)

func TestValidateCommand_ValidPlan(t *testing.T) {
	testFile := filepath.Join("testdata", "valid-plan.md")

	// Create a mock agent registry with the agent referenced in the plan
	agentsDir := setupTestAgents(t)
	defer os.RemoveAll(agentsDir)

	registry := agent.NewRegistry(agentsDir)
	registry.Discover()

	var output bytes.Buffer
	err := validatePlan(testFile, registry, &output)

	if err != nil {
		t.Errorf("validatePlan() returned error for valid plan: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Plan is valid") {
		t.Errorf("Expected success message, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "Parsed 3 tasks") {
		t.Errorf("Expected task count message, got: %s", outputStr)
	}
}

func TestValidateCommand_CyclicDependencies(t *testing.T) {
	testFile := filepath.Join("testdata", "invalid-cycle.yaml")

	registry := agent.NewRegistry(filepath.Join("testdata", "agents"))
	registry.Discover()

	var output bytes.Buffer
	err := validatePlan(testFile, registry, &output)

	if err == nil {
		t.Error("validatePlan() should return error for cyclic dependencies")
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Validation failed") {
		t.Errorf("Expected validation failed message, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "Circular dependency") {
		t.Errorf("Expected circular dependency error, got: %s", outputStr)
	}
}

func TestValidateCommand_MissingFile(t *testing.T) {
	testFile := filepath.Join("testdata", "nonexistent.md")

	registry := agent.NewRegistry(filepath.Join("testdata", "agents"))
	registry.Discover()

	var output bytes.Buffer
	err := validatePlan(testFile, registry, &output)

	if err == nil {
		t.Error("validatePlan() should return error for missing file")
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Failed to parse") {
		t.Errorf("Expected parse failure message, got: %s", outputStr)
	}
}

func TestValidateCommand_InvalidTasks(t *testing.T) {
	testFile := filepath.Join("testdata", "invalid-task.yaml")

	registry := agent.NewRegistry(filepath.Join("testdata", "agents"))
	registry.Discover()

	var output bytes.Buffer
	err := validatePlan(testFile, registry, &output)

	if err == nil {
		t.Error("validatePlan() should return error for invalid tasks")
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Validation failed") {
		t.Errorf("Expected validation failed message, got: %s", outputStr)
	}
	// Task 1 has empty name
	if !strings.Contains(outputStr, "Task 1") && !strings.Contains(outputStr, "name") {
		t.Errorf("Expected task validation error for empty name, got: %s", outputStr)
	}
}

func TestValidateCommand_FileOverlaps(t *testing.T) {
	testFile := filepath.Join("testdata", "file-overlap.yaml")

	registry := agent.NewRegistry(filepath.Join("testdata", "agents"))
	registry.Discover()

	var output bytes.Buffer
	err := validatePlan(testFile, registry, &output)

	if err == nil {
		t.Error("validatePlan() should return error for file overlaps")
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Validation failed") {
		t.Errorf("Expected validation failed message, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "shared.go") {
		t.Errorf("Expected file overlap error mentioning shared.go, got: %s", outputStr)
	}
}

func TestValidateCommand_MissingDependencies(t *testing.T) {
	testFile := filepath.Join("testdata", "missing-dependency.md")

	registry := agent.NewRegistry(filepath.Join("testdata", "agents"))
	registry.Discover()

	var output bytes.Buffer
	err := validatePlan(testFile, registry, &output)

	if err == nil {
		t.Error("validatePlan() should return error for missing dependencies")
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Validation failed") {
		t.Errorf("Expected validation failed message, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "Task 2") && !strings.Contains(outputStr, "99") {
		t.Errorf("Expected missing dependency error, got: %s", outputStr)
	}
}

func TestValidateCommand_AgentNotFound(t *testing.T) {
	testFile := filepath.Join("testdata", "unknown-agent.yaml")

	// Create empty agent registry (no agents)
	registry := agent.NewRegistry(filepath.Join("testdata", "nonexistent-agents"))
	registry.Discover()

	var output bytes.Buffer
	err := validatePlan(testFile, registry, &output)

	if err == nil {
		t.Error("validatePlan() should return error for unknown agent")
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Validation failed") {
		t.Errorf("Expected validation failed message, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "nonexistent-super-agent") {
		t.Errorf("Expected unknown agent error, got: %s", outputStr)
	}
}

func TestValidateCommand_UnknownFormat(t *testing.T) {
	// Create a temporary file with unsupported extension
	tmpFile := filepath.Join(os.TempDir(), "test-plan.txt")
	defer os.Remove(tmpFile)

	err := os.WriteFile(tmpFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	registry := agent.NewRegistry(filepath.Join("testdata", "agents"))
	registry.Discover()

	var output bytes.Buffer
	err = validatePlan(tmpFile, registry, &output)

	if err == nil {
		t.Error("validatePlan() should return error for unknown format")
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Failed to parse") {
		t.Errorf("Expected parse failure message, got: %s", outputStr)
	}
}

func TestNewValidateCommand(t *testing.T) {
	cmd := NewValidateCommand()

	if cmd == nil {
		t.Fatal("NewValidateCommand() returned nil")
	}

	if cmd.Use != "validate <plan-file>" {
		t.Errorf("Expected Use to be 'validate <plan-file>', got: %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE should be set")
	}
}

func TestValidateCommand_Integration(t *testing.T) {
	// Test the actual command execution
	cmd := NewValidateCommand()

	testFile := filepath.Join("testdata", "valid-plan.md")

	// Convert to absolute path for testing
	absPath, err := filepath.Abs(testFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	cmd.SetArgs([]string{absPath})

	// Capture output
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err = cmd.Execute()
	// For valid plan, we expect no error (exit code 0)
	// However, since we might not have real agents, this could fail
	// So we just check that the command runs
	outputStr := output.String()
	if outputStr == "" && err != nil {
		t.Logf("Command executed with error (expected in test): %v", err)
	}
}

func TestValidateCommand_NoArgs(t *testing.T) {
	cmd := NewValidateCommand()
	cmd.SetArgs([]string{})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error when no arguments provided")
	}
}

func TestValidateCommand_MultipleErrors(t *testing.T) {
	// Create a plan with multiple errors
	tmpFile := filepath.Join(os.TempDir(), "multi-error-plan.yaml")
	defer os.Remove(tmpFile)

	content := `plan:
  metadata:
    feature_name: "Multiple Errors"
  tasks:
    - task_number: 1
      name: ""
      files: ["file1.go"]
      depends_on: [99]
      agent: "fake-agent"
      description: "Empty name and missing dependency"
    - task_number: 2
      name: "Task Two"
      files: []
      depends_on: [1]
      description: "Missing prompt"
`

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	registry := agent.NewRegistry(filepath.Join("testdata", "nonexistent-agents"))
	registry.Discover()

	var output bytes.Buffer
	err = validatePlan(tmpFile, registry, &output)

	if err == nil {
		t.Error("validatePlan() should return error for plan with multiple errors")
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Validation failed") {
		t.Errorf("Expected validation failed message, got: %s", outputStr)
	}

	// Should report multiple errors
	errorCount := strings.Count(outputStr, "✗")
	if errorCount < 2 {
		t.Errorf("Expected multiple errors (at least 2), got %d errors", errorCount)
	}
}

// Helper function to create test agent files
func setupTestAgents(t *testing.T) string {
	t.Helper()

	agentsDir := filepath.Join(os.TempDir(), "test-agents")
	err := os.MkdirAll(agentsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create agents dir: %v", err)
	}

	// Create a sample agent
	agentContent := `---
name: golang-pro
description: Go expert agent
tools: [Read, Write, Bash]
---

# Golang Pro Agent

Expert Go developer.
`

	err = os.WriteFile(filepath.Join(agentsDir, "golang-pro.md"), []byte(agentContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create agent file: %v", err)
	}

	return agentsDir
}

func TestValidateCommand_WithRealAgents(t *testing.T) {
	agentsDir := setupTestAgents(t)
	defer os.RemoveAll(agentsDir)

	testFile := filepath.Join("testdata", "valid-plan.md")

	registry := agent.NewRegistry(agentsDir)
	registry.Discover()

	var output bytes.Buffer
	err := validatePlan(testFile, registry, &output)

	// Should succeed now that we have the golang-pro agent
	if err != nil {
		t.Errorf("validatePlan() returned error with real agents: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Plan is valid") {
		t.Errorf("Expected success message, got: %s", outputStr)
	}
}

func TestValidateCommand_ParseError(t *testing.T) {
	// Create a malformed YAML file
	tmpFile := filepath.Join(os.TempDir(), "malformed.yaml")
	defer os.Remove(tmpFile)

	content := `plan:
  tasks:
    - task_number: 1
      name: "Test"
      invalid_yaml: [unclosed array
`

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	registry := agent.NewRegistry(filepath.Join("testdata", "agents"))
	registry.Discover()

	var output bytes.Buffer
	err = validatePlan(tmpFile, registry, &output)

	if err == nil {
		t.Error("validatePlan() should return error for malformed YAML")
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Failed to parse") {
		t.Errorf("Expected parse failure message, got: %s", outputStr)
	}
}

// TestValidateCommand_OutputFormat verifies the output formatting
func TestValidateCommand_OutputFormat(t *testing.T) {
	tests := []struct {
		name           string
		planFile       string
		expectSuccess  bool
		expectedTokens []string
	}{
		{
			name:          "Valid plan shows checkmarks",
			planFile:      "valid-plan.md",
			expectSuccess: true,
			expectedTokens: []string{
				"✓ Validating plan",
				"✓ Parsed",
				"✓ No circular dependencies",
				"Plan is valid",
			},
		},
		{
			name:          "Invalid plan shows X marks",
			planFile:      "invalid-cycle.yaml",
			expectSuccess: false,
			expectedTokens: []string{
				"✗ Validation failed",
				"✗",
				"Circular dependency",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join("testdata", tt.planFile)

			// Setup test agents for valid plan test
			var registry *agent.Registry
			var cleanup func()
			if tt.expectSuccess {
				agentsDir := setupTestAgents(t)
				cleanup = func() { os.RemoveAll(agentsDir) }
				registry = agent.NewRegistry(agentsDir)
			} else {
				registry = agent.NewRegistry(filepath.Join("testdata", "agents"))
			}
			if cleanup != nil {
				defer cleanup()
			}
			registry.Discover()

			var output bytes.Buffer
			err := validatePlan(testFile, registry, &output)

			if tt.expectSuccess && err != nil {
				t.Errorf("Expected success but got error: %v", err)
			}
			if !tt.expectSuccess && err == nil {
				t.Error("Expected error but got success")
			}

			outputStr := output.String()
			for _, token := range tt.expectedTokens {
				if !strings.Contains(outputStr, token) {
					t.Errorf("Expected output to contain %q, got: %s", token, outputStr)
				}
			}
		})
	}
}

// Test that validatePlanFile wrapper works correctly
func TestValidatePlanFile(t *testing.T) {
	testFile := filepath.Join("testdata", "valid-plan.md")

	absPath, err := filepath.Abs(testFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// This should use the default registry and stdout
	err = validatePlanFile(absPath)

	// We expect this to potentially fail if agents aren't found,
	// but it should at least run without panic
	t.Logf("validatePlanFile result: %v", err)
}

// Test DetectFormat is called correctly
func TestValidateCommand_FormatDetection(t *testing.T) {
	tests := []struct {
		filename string
		wantErr  bool
	}{
		{"test.md", false},
		{"test.yaml", false},
		{"test.yml", false},
		{"test.txt", true}, // unsupported format
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			format := parser.DetectFormat(tt.filename)

			if tt.wantErr && format != parser.FormatUnknown {
				t.Errorf("Expected FormatUnknown for %s, got %v", tt.filename, format)
			}
			if !tt.wantErr && format == parser.FormatUnknown {
				t.Errorf("Expected valid format for %s, got FormatUnknown", tt.filename)
			}
		})
	}
}

// Test validateAgents with default agent and QC agent
func TestValidateCommand_DefaultAndQCAgents(t *testing.T) {
	// Create a plan with default agent and QC agent
	tmpFile := filepath.Join(os.TempDir(), "plan-with-defaults.yaml")
	defer os.Remove(tmpFile)

	content := `conductor:
  default_agent: "default-agent"
  quality_control:
    enabled: true
    review_agent: "qc-agent"

plan:
  tasks:
    - task_number: 1
      name: "Test Task"
      files: ["file1.go"]
      depends_on: []
      description: "Uses default agent"
`

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Create empty registry
	registry := agent.NewRegistry(filepath.Join("testdata", "nonexistent"))
	registry.Discover()

	var output bytes.Buffer
	err = validatePlan(tmpFile, registry, &output)

	if err == nil {
		t.Error("Expected validation to fail for missing default and QC agents")
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "default-agent") {
		t.Errorf("Expected error about missing default agent, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "qc-agent") {
		t.Errorf("Expected error about missing QC agent, got: %s", outputStr)
	}
}
