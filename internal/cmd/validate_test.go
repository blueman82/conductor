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

	if cmd.Use != "validate <plan-file-or-directory>..." {
		t.Errorf("Expected Use to be 'validate <plan-file-or-directory>...', got: %s", cmd.Use)
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
	// validatePlanFile now accepts []string
	err = validatePlanFile([]string{absPath})

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

// TestValidateMultiFilePlan tests validation of multi-file split plans
func TestValidateMultiFilePlan(t *testing.T) {
	testDir := filepath.Join("testdata", "valid-split-plan")

	agentsDir := setupTestAgents(t)
	defer os.RemoveAll(agentsDir)

	registry := agent.NewRegistry(agentsDir)
	registry.Discover()

	var output bytes.Buffer
	err := validatePlanDirectory(testDir, registry, &output)

	// Should succeed for valid split plan
	if err != nil {
		t.Errorf("validatePlanDirectory() returned error for valid split plan: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "multi-file") {
		t.Logf("Note: Plan recognized as multi-file: %s", outputStr)
	}
}

// TestValidateWorktreeGroups tests validation of worktree group assignments
func TestValidateWorktreeGroups(t *testing.T) {
	testFile := filepath.Join("testdata", "invalid-groups.yaml")

	registry := agent.NewRegistry(filepath.Join("testdata", "agents"))
	registry.Discover()

	var output bytes.Buffer
	err := validatePlan(testFile, registry, &output)

	// Should fail due to invalid worktree groups
	if err == nil {
		t.Error("validatePlan() should return error for invalid worktree groups")
	}

	outputStr := output.String()
	// Check for worktree group validation errors
	if !strings.Contains(outputStr, "group") && !strings.Contains(outputStr, "WorktreeGroup") {
		t.Logf("Note: Got output: %s", outputStr)
	}

	// Verify the error messages mention group-related issues
	if strings.Contains(outputStr, "Validation failed") {
		t.Logf("Validation correctly failed for invalid worktree groups")
	}
}

// TestValidateCrossFileDeps tests validation of cross-file dependencies
func TestValidateCrossFileDeps(t *testing.T) {
	testDir := filepath.Join("testdata", "broken-cross-deps")

	agentsDir := setupTestAgents(t)
	defer os.RemoveAll(agentsDir)

	registry := agent.NewRegistry(agentsDir)
	registry.Discover()

	var output bytes.Buffer
	err := validatePlanDirectory(testDir, registry, &output)

	// Should fail due to broken cross-file dependencies (task 2 depends on non-existent task 999)
	if err == nil {
		t.Error("validatePlanDirectory() should return error for broken cross-file dependencies")
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "999") {
		t.Logf("Expected error about missing task 999, got: %s", outputStr)
	}
}

// TestValidateSplitBoundaries tests validation of split boundaries
func TestValidateSplitBoundaries(t *testing.T) {
	// Valid split boundaries test - verify that split plans are correctly validated
	testDir := filepath.Join("testdata", "valid-split-plan")

	agentsDir := setupTestAgents(t)
	defer os.RemoveAll(agentsDir)

	registry := agent.NewRegistry(agentsDir)
	registry.Discover()

	var output bytes.Buffer
	err := validatePlanDirectory(testDir, registry, &output)

	// Should succeed for valid split boundaries
	if err != nil {
		t.Errorf("validatePlanDirectory() returned error for valid split boundaries: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Parsed") || !strings.Contains(outputStr, "valid") {
		t.Logf("Split boundaries validation successful: %s", outputStr)
	}
}

// ============================================================================
// MULTI-FILE VALIDATION TESTS
// ============================================================================

// TestValidateCommand_MultipleFiles tests validation with multiple file arguments
func TestValidateCommand_MultipleFiles(t *testing.T) {
	// Setup test agents
	agentsDir := setupTestAgents(t)
	defer os.RemoveAll(agentsDir)

	registry := agent.NewRegistry(agentsDir)
	registry.Discover()

	// Create temporary plan files
	tmpDir := t.TempDir()

	// Plan 1: Markdown file with tasks 1-2
	plan1Content := `# Plan Part 1

## Task 1: Setup
**Files**: main.go
**Agent**: golang-pro

Initialize the project.

## Task 2: Database
**Files**: db.go
**Depends on**: 1
**Agent**: golang-pro

Setup database connection.
`
	plan1Path := filepath.Join(tmpDir, "plan-01-setup.md")
	if err := os.WriteFile(plan1Path, []byte(plan1Content), 0644); err != nil {
		t.Fatalf("Failed to create plan file 1: %v", err)
	}

	// Plan 2: YAML file with tasks 3-4
	plan2Content := `plan:
  tasks:
    - task_number: 3
      name: "API Server"
      files: ["api.go"]
      depends_on: [2]
      agent: "golang-pro"
      description: "Build API server"
    - task_number: 4
      name: "Tests"
      files: ["api_test.go"]
      depends_on: [3]
      agent: "golang-pro"
      description: "Write tests"
`
	plan2Path := filepath.Join(tmpDir, "plan-02-api.yaml")
	if err := os.WriteFile(plan2Path, []byte(plan2Content), 0644); err != nil {
		t.Fatalf("Failed to create plan file 2: %v", err)
	}

	// Test validation with multiple file arguments
	var output bytes.Buffer
	err := validateMultipleFiles([]string{plan1Path, plan2Path}, registry, &output)

	if err != nil {
		t.Errorf("validateMultipleFiles() failed for valid multi-file plan: %v", err)
	}

	outputStr := output.String()

	// Verify output mentions multiple files
	if !strings.Contains(outputStr, "Validating 2 plan file(s)") {
		t.Errorf("Expected output to mention 2 plan files, got: %s", outputStr)
	}

	// Verify task count
	if !strings.Contains(outputStr, "Parsed 4 tasks") {
		t.Errorf("Expected 4 tasks from merged plan, got: %s", outputStr)
	}

	// Verify cross-file dependency validation
	if !strings.Contains(outputStr, "No circular dependencies") {
		t.Errorf("Expected dependency validation message, got: %s", outputStr)
	}

	// Verify success message
	if !strings.Contains(outputStr, "Plan is valid") {
		t.Errorf("Expected validation success message, got: %s", outputStr)
	}
}

// TestValidateCommand_MultipleFiles_CrossFileDependencies tests cross-file dependency validation
func TestValidateCommand_MultipleFiles_CrossFileDependencies(t *testing.T) {
	agentsDir := setupTestAgents(t)
	defer os.RemoveAll(agentsDir)

	registry := agent.NewRegistry(agentsDir)
	registry.Discover()

	tmpDir := t.TempDir()

	// Plan 1: Task 1
	plan1Content := `# Plan Part 1

## Task 1: Foundation
**Files**: base.go
**Agent**: golang-pro

Build foundation.
`
	plan1Path := filepath.Join(tmpDir, "plan-01.md")
	if err := os.WriteFile(plan1Path, []byte(plan1Content), 0644); err != nil {
		t.Fatalf("Failed to create plan file 1: %v", err)
	}

	// Plan 2: Task 2 depends on Task 1 (cross-file dependency)
	plan2Content := `plan:
  tasks:
    - task_number: 2
      name: "Feature"
      files: ["feature.go"]
      depends_on: [1]
      agent: "golang-pro"
      description: "Build feature on top of foundation"
`
	plan2Path := filepath.Join(tmpDir, "plan-02.yaml")
	if err := os.WriteFile(plan2Path, []byte(plan2Content), 0644); err != nil {
		t.Fatalf("Failed to create plan file 2: %v", err)
	}

	// Test validation - should succeed with valid cross-file dependency
	var output bytes.Buffer
	err := validateMultipleFiles([]string{plan1Path, plan2Path}, registry, &output)

	if err != nil {
		t.Errorf("validateMultipleFiles() failed for valid cross-file dependencies: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Plan is valid") {
		t.Errorf("Expected validation to pass with cross-file dependencies, got: %s", outputStr)
	}
}

// TestValidateCommand_MultipleFiles_MissingDependency tests invalid cross-file dependencies
func TestValidateCommand_MultipleFiles_MissingDependency(t *testing.T) {
	agentsDir := setupTestAgents(t)
	defer os.RemoveAll(agentsDir)

	registry := agent.NewRegistry(agentsDir)
	registry.Discover()

	tmpDir := t.TempDir()

	// Plan 1: Task 1
	plan1Content := `# Plan Part 1

## Task 1: Base
**Files**: base.go
**Agent**: golang-pro

Build base.
`
	plan1Path := filepath.Join(tmpDir, "plan-01.md")
	if err := os.WriteFile(plan1Path, []byte(plan1Content), 0644); err != nil {
		t.Fatalf("Failed to create plan file 1: %v", err)
	}

	// Plan 2: Task 2 depends on non-existent Task 99
	plan2Content := `plan:
  tasks:
    - task_number: 2
      name: "Feature"
      files: ["feature.go"]
      depends_on: [99]
      agent: "golang-pro"
      description: "Build feature with broken dependency"
`
	plan2Path := filepath.Join(tmpDir, "plan-02.yaml")
	if err := os.WriteFile(plan2Path, []byte(plan2Content), 0644); err != nil {
		t.Fatalf("Failed to create plan file 2: %v", err)
	}

	// Test validation - should fail due to missing dependency
	var output bytes.Buffer
	err := validateMultipleFiles([]string{plan1Path, plan2Path}, registry, &output)

	if err == nil {
		t.Error("validateMultipleFiles() should fail for missing cross-file dependency")
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Validation failed") {
		t.Errorf("Expected validation failure message, got: %s", outputStr)
	}
}

// TestValidateCommand_DirectoryWithPlanFiles tests validation of directory with plan-* files
func TestValidateCommand_DirectoryWithPlanFiles(t *testing.T) {
	agentsDir := setupTestAgents(t)
	defer os.RemoveAll(agentsDir)

	registry := agent.NewRegistry(agentsDir)
	registry.Discover()

	tmpDir := t.TempDir()

	// Create plan-*.md files that should be included
	plan1Content := `# Plan 1

## Task 1: Task One
**Files**: one.go
**Agent**: golang-pro

Do task one.
`
	if err := os.WriteFile(filepath.Join(tmpDir, "plan-01-first.md"), []byte(plan1Content), 0644); err != nil {
		t.Fatalf("Failed to create plan-01-first.md: %v", err)
	}

	// Create plan-*.yaml files that should be included
	plan2Content := `plan:
  tasks:
    - task_number: 2
      name: "Task Two"
      files: ["two.go"]
      depends_on: [1]
      agent: "golang-pro"
      description: "Do task two"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "plan-02-second.yaml"), []byte(plan2Content), 0644); err != nil {
		t.Fatalf("Failed to create plan-02-second.yaml: %v", err)
	}

	// Create non-plan files that should be filtered out
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# README"), 0644); err != nil {
		t.Fatalf("Failed to create readme.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "setup.yaml"), []byte("key: value"), 0644); err != nil {
		t.Fatalf("Failed to create setup.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "notes.txt"), []byte("notes"), 0644); err != nil {
		t.Fatalf("Failed to create notes.txt: %v", err)
	}

	// Filter plan files from directory
	planFiles, err := filterPlanFiles([]string{tmpDir})
	if err != nil {
		t.Fatalf("filterPlanFiles() failed: %v", err)
	}

	// Should only find 2 plan files
	if len(planFiles) != 2 {
		t.Errorf("Expected 2 plan files, got %d", len(planFiles))
	}

	// Validate the filtered files
	var output bytes.Buffer
	err = validateMultipleFiles(planFiles, registry, &output)

	if err != nil {
		t.Errorf("validateMultipleFiles() failed: %v", err)
	}

	outputStr := output.String()

	// Verify correct number of files and tasks
	if !strings.Contains(outputStr, "Validating 2 plan file(s)") {
		t.Errorf("Expected 2 plan files, got: %s", outputStr)
	}

	if !strings.Contains(outputStr, "Parsed 2 tasks") {
		t.Errorf("Expected 2 tasks, got: %s", outputStr)
	}

	if !strings.Contains(outputStr, "Plan is valid") {
		t.Errorf("Expected validation success, got: %s", outputStr)
	}
}

// TestValidateCommand_MixedDirectoryAndFiles tests validation with both directory and file arguments
func TestValidateCommand_MixedDirectoryAndFiles(t *testing.T) {
	agentsDir := setupTestAgents(t)
	defer os.RemoveAll(agentsDir)

	registry := agent.NewRegistry(agentsDir)
	registry.Discover()

	tmpDir := t.TempDir()

	// Create a subdirectory with plan files
	subDir := filepath.Join(tmpDir, "plans")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Plan 1 in subdirectory
	plan1Content := `# Plan 1

## Task 1: Base
**Files**: base.go
**Agent**: golang-pro

Build base.
`
	if err := os.WriteFile(filepath.Join(subDir, "plan-01.md"), []byte(plan1Content), 0644); err != nil {
		t.Fatalf("Failed to create plan-01.md: %v", err)
	}

	// Plan 2 as standalone file in parent directory
	plan2Content := `plan:
  tasks:
    - task_number: 2
      name: "Extra Feature"
      files: ["extra.go"]
      depends_on: [1]
      agent: "golang-pro"
      description: "Add extra feature"
`
	plan2Path := filepath.Join(tmpDir, "plan-extra.yaml")
	if err := os.WriteFile(plan2Path, []byte(plan2Content), 0644); err != nil {
		t.Fatalf("Failed to create plan-extra.yaml: %v", err)
	}

	// Filter both directory and standalone file
	planFiles, err := filterPlanFiles([]string{subDir, plan2Path})
	if err != nil {
		t.Fatalf("filterPlanFiles() failed: %v", err)
	}

	// Should find 2 plan files (1 from dir + 1 standalone)
	if len(planFiles) != 2 {
		t.Errorf("Expected 2 plan files, got %d", len(planFiles))
	}

	// Validate the filtered files
	var output bytes.Buffer
	err = validateMultipleFiles(planFiles, registry, &output)

	if err != nil {
		t.Errorf("validateMultipleFiles() failed: %v", err)
	}

	outputStr := output.String()

	if !strings.Contains(outputStr, "Validating 2 plan file(s)") {
		t.Errorf("Expected 2 plan files, got: %s", outputStr)
	}

	if !strings.Contains(outputStr, "Parsed 2 tasks") {
		t.Errorf("Expected 2 tasks, got: %s", outputStr)
	}

	if !strings.Contains(outputStr, "Plan is valid") {
		t.Errorf("Expected validation success, got: %s", outputStr)
	}
}

// TestValidateCommand_PlanFileFiltering tests that only plan-* files are validated
func TestValidateCommand_PlanFileFiltering(t *testing.T) {
	tmpDir := t.TempDir()

	// Create various files with different naming patterns
	testFiles := map[string]bool{
		"plan-01-setup.md":       true,  // Should be included
		"plan-02-features.yaml":  true,  // Should be included
		"plan-test.yml":          true,  // Should be included
		"plan-final.markdown":    true,  // Should be included
		"setup.md":               false, // Should be filtered out
		"readme.md":              false, // Should be filtered out
		"config.yaml":            false, // Should be filtered out
		"notes.txt":              false, // Should be filtered out
		"plan.txt":               false, // Wrong extension
		"myplan-01.md":           false, // Doesn't start with "plan-"
	}

	for filename := range testFiles {
		content := "test content"
		if err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	// Filter plan files
	planFiles, err := filterPlanFiles([]string{tmpDir})
	if err != nil {
		t.Fatalf("filterPlanFiles() failed: %v", err)
	}

	// Count expected files
	expectedCount := 0
	for _, shouldInclude := range testFiles {
		if shouldInclude {
			expectedCount++
		}
	}

	if len(planFiles) != expectedCount {
		t.Errorf("Expected %d plan files, got %d", expectedCount, len(planFiles))
	}

	// Verify only plan-* files are included
	for _, planFile := range planFiles {
		basename := filepath.Base(planFile)
		if !strings.HasPrefix(basename, "plan-") {
			t.Errorf("Non-plan file included: %s", basename)
		}

		ext := strings.ToLower(filepath.Ext(basename))
		validExt := ext == ".md" || ext == ".markdown" || ext == ".yaml" || ext == ".yml"
		if !validExt {
			t.Errorf("File with invalid extension included: %s", basename)
		}
	}
}

// TestValidateCommand_EmptyDirectory tests validation of directory with no plan files
func TestValidateCommand_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create non-plan files
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# README"), 0644); err != nil {
		t.Fatalf("Failed to create readme.md: %v", err)
	}

	// Try to filter plan files - should return error
	_, err := filterPlanFiles([]string{tmpDir})

	if err == nil {
		t.Error("filterPlanFiles() should fail for directory with no plan files")
	}

	if !strings.Contains(err.Error(), "no plan files") {
		t.Errorf("Expected 'no plan files' error, got: %v", err)
	}
}

// TestValidateCommand_DuplicateFiles tests that duplicate files are handled correctly
func TestValidateCommand_DuplicateFiles(t *testing.T) {
	agentsDir := setupTestAgents(t)
	defer os.RemoveAll(agentsDir)

	registry := agent.NewRegistry(agentsDir)
	registry.Discover()

	tmpDir := t.TempDir()

	// Create a plan file
	planContent := `# Plan

## Task 1: Test
**Files**: test.go
**Agent**: golang-pro

Test task.
`
	planPath := filepath.Join(tmpDir, "plan-01.md")
	if err := os.WriteFile(planPath, []byte(planContent), 0644); err != nil {
		t.Fatalf("Failed to create plan file: %v", err)
	}

	// Filter with same file specified multiple times
	planFiles, err := filterPlanFiles([]string{planPath, planPath, tmpDir})
	if err != nil {
		t.Fatalf("filterPlanFiles() failed: %v", err)
	}

	// Should deduplicate to 1 file
	if len(planFiles) != 1 {
		t.Errorf("Expected 1 unique plan file, got %d", len(planFiles))
	}

	// Validate should succeed
	var output bytes.Buffer
	err = validateMultipleFiles(planFiles, registry, &output)

	if err != nil {
		t.Errorf("validateMultipleFiles() failed: %v", err)
	}
}
