package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Helper function to create a test plan file
func createTestPlanFile(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "test-plan.md")

	err := os.WriteFile(planFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test plan file: %v", err)
	}

	return planFile
}

// Helper function to execute run command with args
func executeRunCommand(t *testing.T, args []string) (string, error) {
	t.Helper()

	// Create a new root command and run command
	rootCmd := &cobra.Command{Use: "conductor"}
	runCmd := NewRunCommand()
	rootCmd.AddCommand(runCmd)

	// Capture output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Set args
	rootCmd.SetArgs(args)

	// Execute
	err := rootCmd.Execute()
	return buf.String(), err
}

func TestRunCommand_Basic(t *testing.T) {
	validPlan := `# Test Plan

## Task 1: First task
**Status**: pending

Do something simple.

## Task 2: Second task
**Status**: pending

Do something else.
`

	tests := []struct {
		name           string
		planContent    string
		args           []string
		wantErr        bool
		wantErrContain string
	}{
		{
			name:        "valid plan with dry-run",
			planContent: validPlan,
			args:        []string{"run", "--dry-run"},
			wantErr:     false,
		},
		{
			name:        "custom max concurrency",
			planContent: validPlan,
			args:        []string{"run", "--dry-run", "--max-concurrency", "5"},
			wantErr:     false,
		},
		{
			name:        "custom timeout",
			planContent: validPlan,
			args:        []string{"run", "--dry-run", "--timeout", "10m"},
			wantErr:     false,
		},
		{
			name:        "verbose mode",
			planContent: validPlan,
			args:        []string{"run", "--dry-run", "--verbose"},
			wantErr:     false,
		},
		{
			name:        "all flags combined",
			planContent: validPlan,
			args:        []string{"run", "--dry-run", "--max-concurrency", "3", "--timeout", "15m", "--verbose"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test plan file
			planFile := createTestPlanFile(t, tt.planContent)

			// Append plan file to args
			args := append(tt.args, planFile)

			// Execute command
			output, err := executeRunCommand(t, args)

			// Check error
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if tt.wantErrContain != "" && !strings.Contains(err.Error(), tt.wantErrContain) {
					t.Errorf("Expected error containing %q, got: %v", tt.wantErrContain, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v\nOutput: %s", err, output)
				}
			}
		})
	}
}

func TestRunCommand_ErrorCases(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantErrContain string
	}{
		{
			name:           "missing plan file argument",
			args:           []string{"run"},
			wantErrContain: "accepts 1 arg",
		},
		{
			name:           "too many arguments",
			args:           []string{"run", "file1.md", "file2.md"},
			wantErrContain: "accepts 1 arg",
		},
		{
			name:           "plan file not found",
			args:           []string{"run", "/nonexistent/plan.md"},
			wantErrContain: "failed to load plan file",
		},
		{
			name:           "invalid timeout format",
			args:           []string{"run", "--timeout", "invalid", "dummy.md"},
			wantErrContain: "invalid timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a valid plan file if args don't already have one
			args := tt.args
			if !strings.Contains(tt.name, "plan file not found") &&
				!strings.Contains(tt.name, "missing plan file") &&
				!strings.Contains(tt.name, "too many arguments") {
				// Create a valid plan file for other cases
				planFile := createTestPlanFile(t, "# Test\n## Task 1: Test\n**Status**: pending\n")
				// Replace dummy.md with actual plan file
				if len(args) > 0 && strings.Contains(args[len(args)-1], "dummy.md") {
					args[len(args)-1] = planFile
				} else if !strings.Contains(strings.Join(args, " "), ".md") {
					args = append(args, planFile)
				}
			}

			_, err := executeRunCommand(t, args)

			if err == nil {
				t.Errorf("Expected error but got none")
			}
			if !strings.Contains(err.Error(), tt.wantErrContain) {
				t.Errorf("Expected error containing %q, got: %v", tt.wantErrContain, err)
			}
		})
	}
}

func TestRunCommand_DryRunOutput(t *testing.T) {
	validPlan := `# Test Plan

## Task 1: First task
**Status**: pending

Do something.

## Task 2: Second task
**Status**: pending

Do something else.
`

	planFile := createTestPlanFile(t, validPlan)
	args := []string{"run", "--dry-run", planFile}

	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify dry-run output contains expected information
	if !strings.Contains(output, "Dry-run mode") && !strings.Contains(output, "valid") {
		t.Logf("Dry-run output: %s", output)
	}

	// Verify output mentions plan validation
	if !strings.Contains(output, "valid") && !strings.Contains(output, "Plan") {
		t.Error("Expected dry-run output to mention plan validation")
	}
}

func TestRunCommand_TimeoutParsing(t *testing.T) {
	validPlan := `# Test Plan

## Task 1: Test timeout
**Status**: pending

Test timeout parsing.
`

	tests := []struct {
		name    string
		timeout string
		wantErr bool
	}{
		{
			name:    "valid minutes",
			timeout: "5m",
			wantErr: false,
		},
		{
			name:    "valid hours",
			timeout: "2h",
			wantErr: false,
		},
		{
			name:    "invalid format",
			timeout: "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planFile := createTestPlanFile(t, validPlan)
			args := []string{"run", "--dry-run", "--timeout", tt.timeout, planFile}

			_, err := executeRunCommand(t, args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for timeout %q but got none", tt.timeout)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for timeout %q: %v", tt.timeout, err)
				}
			}
		})
	}
}

func TestNewRunCommand(t *testing.T) {
	cmd := NewRunCommand()

	if cmd.Use != "run [plan-file]" {
		t.Errorf("Expected Use to be 'run [plan-file]', got: %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected Short description to be set")
	}

	if cmd.Long == "" {
		t.Error("Expected Long description to be set")
	}

	// Verify flags exist
	flags := []string{"dry-run", "max-concurrency", "timeout", "verbose"}
	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag %q to exist", flagName)
		}
	}
}

func TestRunCommand_VerboseOutput(t *testing.T) {
	validPlan := `# Test Plan

## Task 1: Verbose test
**Status**: pending

Test verbose output.
`

	planFile := createTestPlanFile(t, validPlan)
	args := []string{"run", "--dry-run", "--verbose", planFile}

	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(output) == 0 {
		t.Error("Expected verbose output but got empty string")
	}
}

func TestRunCommand_PlanValidation(t *testing.T) {
	validPlan := `# Valid Plan

## Task 1: Valid task
**Status**: pending

Do something.
`

	planFile := createTestPlanFile(t, validPlan)
	args := []string{"run", "--dry-run", planFile}

	_, err := executeRunCommand(t, args)

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
}

func TestRunCommand_LogDirFlag(t *testing.T) {
	validPlan := `# Test Plan

## Task 1: First task
**Status**: pending

Do something.
`

	planFile := createTestPlanFile(t, validPlan)

	tests := []struct {
		name   string
		logDir string
	}{
		{
			name:   "default log directory",
			logDir: "",
		},
		{
			name:   "custom log directory",
			logDir: filepath.Join(t.TempDir(), "custom-logs"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"run", "--dry-run", planFile}
			if tt.logDir != "" {
				args = append(args, "--log-dir", tt.logDir)
			}

			_, err := executeRunCommand(t, args)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestRunCommand_MaxConcurrency(t *testing.T) {
	validPlan := `# Test Plan

## Task 1: First task
**Status**: pending

Do something.

## Task 2: Second task
**Status**: pending

Do something else.
`

	planFile := createTestPlanFile(t, validPlan)
	args := []string{"run", "--dry-run", "--max-concurrency", "2", planFile}

	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(output, "Max concurrency: 2") {
		t.Error("Expected output to mention max concurrency setting")
	}
}

func TestRunCommand_EmptyPlan(t *testing.T) {
	emptyPlan := `# Empty Plan

No tasks defined.
`

	planFile := createTestPlanFile(t, emptyPlan)
	args := []string{"run", "--dry-run", planFile}

	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !strings.Contains(output, "no tasks") {
		t.Error("Expected output to mention no tasks")
	}
}

func TestRunCommand_CircularDependency(t *testing.T) {
	cyclicPlan := `# Cyclic Plan

## Task 1: First
**Status**: pending
**Depends on**: 2

Do something.

## Task 2: Second
**Status**: pending
**Depends on**: 1

Do something else.
`

	planFile := createTestPlanFile(t, cyclicPlan)
	args := []string{"run", "--dry-run", planFile}

	_, err := executeRunCommand(t, args)

	if err == nil {
		t.Error("Expected error for circular dependency")
	}

	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("Expected circular dependency error, got: %v", err)
	}
}

func TestRunCommand_VerboseDryRun(t *testing.T) {
	validPlan := `# Test Plan

## Task 1: First task
**Status**: pending

Do something.

## Task 2: Second task
**Status**: pending
**Depends on**: 1

Do something else.
`

	planFile := createTestPlanFile(t, validPlan)
	args := []string{"run", "--dry-run", "--verbose", planFile}

	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verbose mode should show task details
	if !strings.Contains(output, "Task 1") || !strings.Contains(output, "Task 2") {
		t.Error("Expected verbose output to show task details")
	}

	// Should show wave structure
	if !strings.Contains(output, "Wave") {
		t.Error("Expected output to show wave structure")
	}
}

func TestRunCommand_ComplexDependencies(t *testing.T) {
	complexPlan := `# Complex Plan

## Task 1: Foundation
**Status**: pending

Build foundation.

## Task 2: Branch A
**Status**: pending
**Depends on**: 1

Build branch A.

## Task 3: Branch B
**Status**: pending
**Depends on**: 1

Build branch B.

## Task 4: Convergence
**Status**: pending
**Depends on**: 2, 3

Merge branches.
`

	planFile := createTestPlanFile(t, complexPlan)
	args := []string{"run", "--dry-run", planFile}

	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should calculate 3 waves
	if !strings.Contains(output, "Execution waves: 3") {
		t.Errorf("Expected 3 waves for complex dependencies, got output: %s", output)
	}
}

func TestRunCommand_YAMLFormat(t *testing.T) {
	yamlPlan := `conductor:
  max_concurrency: 2
  timeout: "5m"
  default_agent: "general-purpose"

plan:
  metadata:
    name: "YAML Test Plan"
    estimated_tasks: 2
  tasks:
    - task_number: 1
      name: "First YAML Task"
      prompt: "Do the first task"
      files: ["main.go"]
      depends_on: []
      estimated_time: "10m"
    - task_number: 2
      name: "Second YAML Task"
      prompt: "Do the second task"
      files: ["output.go"]
      depends_on: [1]
      estimated_time: "10m"
`

	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "test-plan.yaml")

	err := os.WriteFile(planFile, []byte(yamlPlan), 0644)
	if err != nil {
		t.Fatalf("Failed to create YAML plan file: %v", err)
	}

	args := []string{"run", "--dry-run", planFile}

	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(output, "Total tasks: 2") {
		t.Errorf("Expected YAML plan to be parsed correctly, got output: %s", output)
	}
}

func TestRunCommand_LongTimeout(t *testing.T) {
	validPlan := `# Test Plan

## Task 1: Test task
**Status**: pending

Test long timeout.
`

	planFile := createTestPlanFile(t, validPlan)
	args := []string{"run", "--dry-run", "--timeout", "24h", planFile}

	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(output, "Timeout: 24h") {
		t.Error("Expected output to show 24h timeout")
	}
}
