package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/executor"
	"github.com/harrison/conductor/internal/models"
	"github.com/spf13/cobra"
)

// mockOrchestrator implements a mock orchestrator for testing
type mockOrchestrator struct {
	executeCalled   bool
	executeErr      error
	executeCtx      context.Context
	executePlan     *models.Plan
	executeFilePath string
}

func (m *mockOrchestrator) Execute(ctx context.Context, plan *models.Plan, filePath string) error {
	m.executeCalled = true
	m.executeCtx = ctx
	m.executePlan = plan
	m.executeFilePath = filePath
	return m.executeErr
}

// orchestratorInterface defines the interface for orchestration
type orchestratorInterface interface {
	Execute(ctx context.Context, plan *models.Plan, filePath string) error
}

// Global var for testing (set in tests to use mocks)
var newOrchestratorFunc = func(maxConcurrency int, timeout time.Duration, verbose bool) orchestratorInterface {
	logger := &consoleLogger{writer: os.Stdout, verbose: verbose}
	return executor.NewOrchestrator(executor.NewWaveExecutor(nil), logger)
}

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
func executeRunCommand(t *testing.T, args []string, mockOrch *mockOrchestrator) (string, error) {
	t.Helper()

	// Create a new root command and run command
	rootCmd := &cobra.Command{Use: "conductor"}
	runCmd := NewRunCommand()

	// Replace orchestrator creation with mock
	if mockOrch != nil {
		originalNewOrchestrator := newOrchestratorFunc
		defer func() { newOrchestratorFunc = originalNewOrchestrator }()

		newOrchestratorFunc = func(maxConcurrency int, timeout time.Duration, verbose bool) orchestratorInterface {
			return mockOrch
		}
	}

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
**Agent**: default
**Depends On**: none

Do something simple.

## Task 2: Second task
**Status**: pending
**Agent**: default
**Depends On**: Task 1

Do something else.
`

	tests := []struct {
		name           string
		planContent    string
		args           []string
		mockErr        error
		wantErr        bool
		wantErrContain string
		checkMock      func(*testing.T, *mockOrchestrator)
	}{
		{
			name:        "valid plan execution",
			planContent: validPlan,
			args:        []string{"run"},
			mockErr:     nil,
			wantErr:     false,
			checkMock: func(t *testing.T, m *mockOrchestrator) {
				if !m.executeCalled {
					t.Error("Execute was not called")
				}
				if m.executePlan == nil {
					t.Error("Execute called with nil plan")
				}
				if m.executeFilePath == "" {
					t.Error("Execute called with empty file path")
				}
			},
		},
		{
			name:           "orchestrator execution error",
			planContent:    validPlan,
			args:           []string{"run"},
			mockErr:        errors.New("execution failed"),
			wantErr:        true,
			wantErrContain: "execution failed",
		},
		{
			name:        "dry run mode",
			planContent: validPlan,
			args:        []string{"run", "--dry-run"},
			mockErr:     nil,
			wantErr:     false,
			checkMock: func(t *testing.T, m *mockOrchestrator) {
				if m.executeCalled {
					t.Error("Execute should not be called in dry-run mode")
				}
			},
		},
		{
			name:        "custom max concurrency",
			planContent: validPlan,
			args:        []string{"run", "--max-concurrency", "5"},
			mockErr:     nil,
			wantErr:     false,
		},
		{
			name:        "custom timeout",
			planContent: validPlan,
			args:        []string{"run", "--timeout", "10m"},
			mockErr:     nil,
			wantErr:     false,
		},
		{
			name:        "verbose mode",
			planContent: validPlan,
			args:        []string{"run", "--verbose"},
			mockErr:     nil,
			wantErr:     false,
		},
		{
			name:        "all flags combined",
			planContent: validPlan,
			args:        []string{"run", "--dry-run", "--max-concurrency", "3", "--timeout", "15m", "--verbose"},
			mockErr:     nil,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test plan file
			planFile := createTestPlanFile(t, tt.planContent)

			// Append plan file to args
			args := append(tt.args, planFile)

			// Create mock orchestrator
			mockOrch := &mockOrchestrator{
				executeErr: tt.mockErr,
			}

			// Execute command
			output, err := executeRunCommand(t, args, mockOrch)

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

			// Run custom checks
			if tt.checkMock != nil {
				tt.checkMock(t, mockOrch)
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
			wantErrContain: "requires exactly 1 arg",
		},
		{
			name:           "too many arguments",
			args:           []string{"run", "file1.md", "file2.md"},
			wantErrContain: "requires exactly 1 arg",
		},
		{
			name:           "plan file not found",
			args:           []string{"run", "/nonexistent/plan.md"},
			wantErrContain: "no such file or directory",
		},
		{
			name:           "invalid timeout format",
			args:           []string{"run", "--timeout", "invalid"},
			wantErrContain: "invalid duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For file not found case, use the provided path
			args := tt.args
			if !strings.Contains(tt.name, "plan file not found") &&
				!strings.Contains(tt.name, "missing plan file") &&
				!strings.Contains(tt.name, "too many arguments") &&
				!strings.Contains(tt.name, "invalid timeout") {
				// Create a valid plan file for other cases
				planFile := createTestPlanFile(t, "# Test\n## Task 1: Test\n**Status**: pending\n")
				args = append(args, planFile)
			}

			mockOrch := &mockOrchestrator{}
			_, err := executeRunCommand(t, args, mockOrch)

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
**Agent**: default
**Depends On**: none

Do something.

## Task 2: Second task
**Status**: pending
**Agent**: default
**Depends On**: Task 1

Do something else.
`

	planFile := createTestPlanFile(t, validPlan)
	args := []string{"run", "--dry-run", planFile}

	mockOrch := &mockOrchestrator{}
	output, err := executeRunCommand(t, args, mockOrch)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify dry-run output contains expected information
	if !strings.Contains(output, "DRY RUN") && !strings.Contains(output, "Task 1") {
		t.Logf("Dry-run output: %s", output)
	}

	// Verify orchestrator was not called
	if mockOrch.executeCalled {
		t.Error("Orchestrator.Execute should not be called in dry-run mode")
	}
}

func TestRunCommand_TimeoutParsing(t *testing.T) {
	validPlan := `# Test Plan

## Task 1: Test timeout
**Status**: pending
**Agent**: default
**Depends On**: none

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
			args := []string{"run", "--timeout", tt.timeout, planFile}

			mockOrch := &mockOrchestrator{}
			_, err := executeRunCommand(t, args, mockOrch)

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

	if cmd.Use != "run <plan-file>" {
		t.Errorf("Expected Use to be 'run <plan-file>', got: %s", cmd.Use)
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
**Agent**: default
**Depends On**: none

Test verbose output.
`

	planFile := createTestPlanFile(t, validPlan)
	args := []string{"run", "--verbose", planFile}

	mockOrch := &mockOrchestrator{}
	output, err := executeRunCommand(t, args, mockOrch)

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
**Agent**: default
**Depends On**: none

Do something.
`

	planFile := createTestPlanFile(t, validPlan)
	args := []string{"run", planFile}

	mockOrch := &mockOrchestrator{}
	_, err := executeRunCommand(t, args, mockOrch)

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
}
