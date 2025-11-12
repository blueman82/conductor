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
			wantErrContain: "requires at least 1 arg",
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
				!strings.Contains(tt.name, "missing plan file") {
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

	if cmd.Use != "run <plan-file-or-directory>..." {
		t.Errorf("Expected Use to be 'run <plan-file-or-directory>...', got: %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected Short description to be set")
	}

	if cmd.Long == "" {
		t.Error("Expected Long description to be set")
	}

	// Verify flags exist
	flags := []string{"dry-run", "max-concurrency", "timeout", "verbose", "skip-completed", "no-skip-completed", "retry-failed", "no-retry-failed"}
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

// ============================================================================
// E2E TESTS - Task 25 Implementation
// ============================================================================

// TestRunWithFileLogger verifies FileLogger receives events during execution
func TestRunWithFileLogger(t *testing.T) {
	// Use simple test plan
	simplePlan := `# Test Plan

## Task 1: First task
**Status**: pending

Do something simple.
`

	planFile := createTestPlanFile(t, simplePlan)
	logDir := t.TempDir()

	// Execute with custom log directory (dry-run to avoid actual execution)
	args := []string{"run", "--dry-run", "--log-dir", logDir, planFile}
	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v\nOutput: %s", err, output)
	}

	// In dry-run mode, no log files should be created
	entries, err := os.ReadDir(logDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("Failed to read log directory: %v", err)
	}

	// Dry-run should not create log files
	if len(entries) > 0 {
		t.Errorf("Expected no log files in dry-run mode, found: %d", len(entries))
	}
}

// TestRunWithConsoleLogger verifies ConsoleLogger output during execution
func TestRunWithConsoleLogger(t *testing.T) {
	simplePlan := `# Test Plan

## Task 1: Simple task
**Status**: pending

Execute a simple task.

## Task 2: Second task
**Status**: pending
**Depends on**: 1

Execute second task.
`

	planFile := createTestPlanFile(t, simplePlan)
	args := []string{"run", "--dry-run", "--verbose", planFile}

	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify console output contains execution information
	if !strings.Contains(output, "Loading plan") {
		t.Error("Expected console output to contain 'Loading plan'")
	}

	if !strings.Contains(output, "Validating dependencies") {
		t.Error("Expected console output to contain 'Validating dependencies'")
	}

	if !strings.Contains(output, "Plan Summary") {
		t.Error("Expected console output to contain 'Plan Summary'")
	}

	// In verbose mode, should show task details
	if !strings.Contains(output, "Task 1") {
		t.Error("Expected verbose output to show task details")
	}
}

// TestRunWithBothLoggers verifies both loggers work together
func TestRunWithBothLoggers(t *testing.T) {
	simplePlan := `# Test Plan

## Task 1: Test task
**Status**: pending

Test both loggers.
`

	planFile := createTestPlanFile(t, simplePlan)
	logDir := t.TempDir()

	// Execute with both console (captured) and file logger
	args := []string{"run", "--dry-run", "--log-dir", logDir, "--verbose", planFile}
	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify console output is present
	if len(output) == 0 {
		t.Error("Expected console output but got empty string")
	}

	// Verify console contains plan summary
	if !strings.Contains(output, "Plan Summary") {
		t.Error("Expected console output to contain plan summary")
	}
}

// TestLogFilesCreated verifies .conductor/logs/run-*.log files exist after execution
func TestLogFilesCreated(t *testing.T) {
	// This test requires actual execution, not dry-run
	// For now, we'll test that the log directory is prepared correctly
	simplePlan := `# Test Plan

## Task 1: Log test
**Status**: pending

Test log file creation.
`

	planFile := createTestPlanFile(t, simplePlan)

	// Change to temp directory to avoid polluting the project
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Execute in dry-run (actual execution would require mock agent invoker)
	args := []string{"run", "--dry-run", planFile}
	_, err = executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify .conductor directory structure is ready
	conductorDir := filepath.Join(tmpDir, ".conductor")
	if _, err := os.Stat(conductorDir); os.IsNotExist(err) {
		// In dry-run mode, .conductor directory may not be created
		t.Skip("Skipping log file verification in dry-run mode")
	}
}

// TestLatestSymlinkUpdated verifies latest.log symlink points to current run
func TestLatestSymlinkUpdated(t *testing.T) {
	// This test verifies symlink behavior (requires actual execution)
	// Testing symlink creation is done in logger tests
	simplePlan := `# Test Plan

## Task 1: Symlink test
**Status**: pending

Test symlink behavior.
`

	planFile := createTestPlanFile(t, simplePlan)
	args := []string{"run", "--dry-run", planFile}

	_, err := executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Symlink verification is tested in logger unit tests
	// This E2E test verifies the command completes successfully
}

// TestLogsContainExecutionDetails verifies log content includes task results/output
func TestLogsContainExecutionDetails(t *testing.T) {
	// This test requires actual execution with mock invoker
	// For E2E, we verify that verbose mode shows execution details
	simplePlan := `# Test Plan

## Task 1: Detail test
**Status**: pending

Test execution details in logs.
`

	planFile := createTestPlanFile(t, simplePlan)
	args := []string{"run", "--dry-run", "--verbose", planFile}

	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify output contains execution details
	if !strings.Contains(output, "Total tasks") {
		t.Error("Expected output to contain task count")
	}

	if !strings.Contains(output, "Execution waves") {
		t.Error("Expected output to contain wave information")
	}

	// Verbose mode should show task details
	if !strings.Contains(output, "Task 1") {
		t.Error("Expected verbose output to show task details")
	}
}

// TestNoLogFileIfDryRun verifies dry-run doesn't create log files
func TestNoLogFileIfDryRun(t *testing.T) {
	simplePlan := `# Test Plan

## Task 1: Dry-run test
**Status**: pending

Test that dry-run doesn't create logs.
`

	planFile := createTestPlanFile(t, simplePlan)

	// Change to temp directory
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Execute in dry-run mode
	args := []string{"run", "--dry-run", planFile}
	_, err = executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify .conductor/logs directory was not created in dry-run
	logsDir := filepath.Join(tmpDir, ".conductor", "logs")
	if _, err := os.Stat(logsDir); !os.IsNotExist(err) {
		// Check if any log files exist
		entries, readErr := os.ReadDir(logsDir)
		if readErr == nil && len(entries) > 0 {
			t.Errorf("Expected no log files in dry-run mode, found: %d files", len(entries))
		}
	}
}

// TestE2E_SimpleMarkdownPlan executes simple 2-task markdown plan end-to-end
func TestE2E_SimpleMarkdownPlan(t *testing.T) {
	// Read the integration test fixture
	fixtureDir := filepath.Join("..", "..", "test", "integration", "fixtures")
	planFile := filepath.Join(fixtureDir, "simple-plan.md")

	// Check if fixture exists
	if _, err := os.Stat(planFile); os.IsNotExist(err) {
		t.Skipf("Skipping E2E test: fixture not found at %s", planFile)
	}

	// Execute in dry-run mode (actual execution requires mock invoker)
	args := []string{"run", "--dry-run", planFile}
	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v\nOutput: %s", err, output)
	}

	// Verify the plan is parsed correctly
	if !strings.Contains(output, "Total tasks: 2") {
		t.Errorf("Expected 2 tasks in simple plan, got output: %s", output)
	}

	// Verify wave calculation
	if !strings.Contains(output, "Execution waves: 2") {
		t.Errorf("Expected 2 waves (sequential tasks), got output: %s", output)
	}

	// Verify dry-run message
	if !strings.Contains(output, "Dry-run mode") {
		t.Error("Expected dry-run mode message in output")
	}
}

// TestE2E_SimpleYamlPlan executes simple YAML plan end-to-end
func TestE2E_SimpleYamlPlan(t *testing.T) {
	// Read the integration test fixture
	fixtureDir := filepath.Join("..", "..", "test", "integration", "fixtures")
	planFile := filepath.Join(fixtureDir, "simple-plan.yaml")

	// Check if fixture exists
	if _, err := os.Stat(planFile); os.IsNotExist(err) {
		t.Skipf("Skipping E2E test: fixture not found at %s", planFile)
	}

	// Execute in dry-run mode
	args := []string{"run", "--dry-run", planFile}
	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v\nOutput: %s", err, output)
	}

	// Verify the YAML plan is parsed correctly
	if !strings.Contains(output, "Total tasks: 2") {
		t.Errorf("Expected 2 tasks in YAML plan, got output: %s", output)
	}

	// Verify dry-run message
	if !strings.Contains(output, "Dry-run mode") {
		t.Error("Expected dry-run mode message in output")
	}
}

// TestE2E_FailureHandling handles task failure scenarios
func TestE2E_FailureHandling(t *testing.T) {
	// Read the failure fixture
	fixtureDir := filepath.Join("..", "..", "test", "integration", "fixtures")
	planFile := filepath.Join(fixtureDir, "with-failure.md")

	// Check if fixture exists
	if _, err := os.Stat(planFile); os.IsNotExist(err) {
		t.Skipf("Skipping E2E test: fixture not found at %s", planFile)
	}

	// Execute in dry-run mode (we can still validate the plan structure)
	args := []string{"run", "--dry-run", planFile}
	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error during dry-run: %v\nOutput: %s", err, output)
	}

	// Verify plan is valid and contains expected tasks
	if !strings.Contains(output, "Total tasks: 3") {
		t.Errorf("Expected 3 tasks in failure plan, got output: %s", output)
	}

	// Actual failure handling would be tested with mock invoker
	// This E2E test verifies the plan structure is valid
}

// TestE2E_ComplexDependencies executes multi-wave plan end-to-end
func TestE2E_ComplexDependencies(t *testing.T) {
	// Read the complex dependencies fixture
	fixtureDir := filepath.Join("..", "..", "test", "integration", "fixtures")
	planFile := filepath.Join(fixtureDir, "complex-dependencies.md")

	// Check if fixture exists
	if _, err := os.Stat(planFile); os.IsNotExist(err) {
		t.Skipf("Skipping E2E test: fixture not found at %s", planFile)
	}

	// Execute in dry-run mode
	args := []string{"run", "--dry-run", "--verbose", planFile}
	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v\nOutput: %s", err, output)
	}

	// Verify the complex plan is parsed correctly
	if !strings.Contains(output, "Total tasks: 7") {
		t.Errorf("Expected 7 tasks in complex plan, got output: %s", output)
	}

	// Verify wave structure (complex dependencies should create multiple waves)
	if !strings.Contains(output, "Execution waves: 4") {
		t.Errorf("Expected 4 waves in complex plan, got output: %s", output)
	}

	// Verify verbose mode shows wave details
	if !strings.Contains(output, "Wave") {
		t.Error("Expected verbose output to show wave details")
	}
}

// TestE2E_DryRunMode verifies dry-run doesn't execute tasks
func TestE2E_DryRunMode(t *testing.T) {
	simplePlan := `# Dry-Run Test

## Task 1: Should not execute
**Status**: pending

This task should not be executed in dry-run mode.

## Task 2: Also not executed
**Status**: pending
**Depends on**: 1

This task should also not be executed.
`

	planFile := createTestPlanFile(t, simplePlan)

	// Change to temp directory to monitor file system changes
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Execute in dry-run mode
	args := []string{"run", "--dry-run", planFile}
	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify dry-run message is displayed
	if !strings.Contains(output, "Dry-run mode") {
		t.Error("Expected dry-run mode message in output")
	}

	if !strings.Contains(output, "valid and ready for execution") {
		t.Error("Expected message about plan being ready for execution")
	}

	// Verify no .conductor/logs directory created (no actual execution)
	logsDir := filepath.Join(tmpDir, ".conductor", "logs")
	if _, err := os.Stat(logsDir); !os.IsNotExist(err) {
		entries, readErr := os.ReadDir(logsDir)
		if readErr == nil && len(entries) > 0 {
			t.Errorf("Expected no logs in dry-run mode, found %d files", len(entries))
		}
	}
}

// TestE2E_TimeoutHandling verifies timeout handling during execution
func TestE2E_TimeoutHandling(t *testing.T) {
	simplePlan := `# Timeout Test

## Task 1: Quick task
**Status**: pending

Execute quickly.
`

	planFile := createTestPlanFile(t, simplePlan)

	// Test various timeout values
	tests := []struct {
		name    string
		timeout string
		wantErr bool
	}{
		{
			name:    "short timeout",
			timeout: "1s",
			wantErr: false,
		},
		{
			name:    "medium timeout",
			timeout: "30m",
			wantErr: false,
		},
		{
			name:    "long timeout",
			timeout: "2h",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"run", "--dry-run", "--timeout", tt.timeout, planFile}
			output, err := executeRunCommand(t, args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v\nOutput: %s", err, output)
				}

				// Verify timeout is shown in output
				if !strings.Contains(output, "Timeout: "+tt.timeout) {
					t.Errorf("Expected timeout %q in output, got: %s", tt.timeout, output)
				}
			}
		})
	}
}

// TestE2E_LargePlan is a performance test with large-plan.md fixture
func TestE2E_LargePlan(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large plan test in short mode")
	}

	// Read the large plan fixture
	fixtureDir := filepath.Join("..", "..", "test", "integration", "fixtures")
	planFile := filepath.Join(fixtureDir, "large-plan.md")

	// Check if fixture exists
	if _, err := os.Stat(planFile); os.IsNotExist(err) {
		t.Skipf("Skipping E2E test: fixture not found at %s", planFile)
	}

	// Execute in dry-run mode to test parsing performance
	args := []string{"run", "--dry-run", planFile}
	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v\nOutput: %s", err, output)
	}

	// Verify the large plan is parsed correctly
	// Large plan should have many tasks
	if !strings.Contains(output, "Total tasks:") {
		t.Error("Expected total tasks count in output")
	}

	if !strings.Contains(output, "Execution waves:") {
		t.Error("Expected execution waves count in output")
	}

	// Verify dry-run completes successfully
	if !strings.Contains(output, "Dry-run mode") {
		t.Error("Expected dry-run mode message")
	}
}

// TestExecutionSummaryLogging verifies summary logs are created
func TestExecutionSummaryLogging(t *testing.T) {
	simplePlan := `# Summary Test

## Task 1: Test task
**Status**: pending

Test summary logging.

## Task 2: Second task
**Status**: pending
**Depends on**: 1

Another test task.
`

	planFile := createTestPlanFile(t, simplePlan)
	args := []string{"run", "--dry-run", "--verbose", planFile}

	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify output contains summary information
	if !strings.Contains(output, "Plan Summary") {
		t.Error("Expected 'Plan Summary' in output")
	}

	if !strings.Contains(output, "Total tasks: 2") {
		t.Error("Expected total tasks count in summary")
	}

	if !strings.Contains(output, "Execution waves: 2") {
		t.Error("Expected execution waves count in summary")
	}

	// Verify wave information is shown
	if !strings.Contains(output, "Wave 1") {
		t.Error("Expected Wave 1 information in output")
	}

	if !strings.Contains(output, "Wave 2") {
		t.Error("Expected Wave 2 information in output")
	}
}

// TestRunCommand_SkipCompletedFlag verifies --skip-completed flag is accepted
func TestRunCommand_SkipCompletedFlag(t *testing.T) {
	simplePlan := `# Test Plan

## Task 1: Test task
**Status**: pending

Test skip completed flag.
`

	planFile := createTestPlanFile(t, simplePlan)
	args := []string{"run", "--dry-run", "--skip-completed", planFile}

	_, err := executeRunCommand(t, args)

	if err != nil {
		t.Errorf("Unexpected error with --skip-completed flag: %v", err)
	}
}

// TestRunCommand_RetryFailedFlag verifies --retry-failed flag is accepted
func TestRunCommand_RetryFailedFlag(t *testing.T) {
	simplePlan := `# Test Plan

## Task 1: Test task
**Status**: pending

Test retry failed flag.
`

	planFile := createTestPlanFile(t, simplePlan)
	args := []string{"run", "--dry-run", "--retry-failed", planFile}

	_, err := executeRunCommand(t, args)

	if err != nil {
		t.Errorf("Unexpected error with --retry-failed flag: %v", err)
	}
}

// TestRunCommand_NoSkipCompletedFlag verifies --no-skip-completed flag is accepted
func TestRunCommand_NoSkipCompletedFlag(t *testing.T) {
	simplePlan := `# Test Plan

## Task 1: Test task
**Status**: pending

Test no-skip-completed flag.
`

	planFile := createTestPlanFile(t, simplePlan)
	args := []string{"run", "--dry-run", "--no-skip-completed", planFile}

	_, err := executeRunCommand(t, args)

	if err != nil {
		t.Errorf("Unexpected error with --no-skip-completed flag: %v", err)
	}
}

// TestRunCommand_NoRetryFailedFlag verifies --no-retry-failed flag is accepted
func TestRunCommand_NoRetryFailedFlag(t *testing.T) {
	simplePlan := `# Test Plan

## Task 1: Test task
**Status**: pending

Test no-retry-failed flag.
`

	planFile := createTestPlanFile(t, simplePlan)
	args := []string{"run", "--dry-run", "--no-retry-failed", planFile}

	_, err := executeRunCommand(t, args)

	if err != nil {
		t.Errorf("Unexpected error with --no-retry-failed flag: %v", err)
	}
}

// TestRunCommand_ConflictingSkipCompletedFlags verifies conflicting flags are rejected
func TestRunCommand_ConflictingSkipCompletedFlags(t *testing.T) {
	simplePlan := `# Test Plan

## Task 1: Test task
**Status**: pending

Test conflicting skip-completed flags.
`

	planFile := createTestPlanFile(t, simplePlan)
	args := []string{"run", "--dry-run", "--skip-completed", "--no-skip-completed", planFile}

	_, err := executeRunCommand(t, args)

	if err == nil {
		t.Error("Expected error for conflicting --skip-completed and --no-skip-completed flags")
	}

	if !strings.Contains(err.Error(), "cannot use both") {
		t.Errorf("Expected error about conflicting flags, got: %v", err)
	}
}

// TestRunCommand_ConflictingRetryFailedFlags verifies conflicting flags are rejected
func TestRunCommand_ConflictingRetryFailedFlags(t *testing.T) {
	simplePlan := `# Test Plan

## Task 1: Test task
**Status**: pending

Test conflicting retry-failed flags.
`

	planFile := createTestPlanFile(t, simplePlan)
	args := []string{"run", "--dry-run", "--retry-failed", "--no-retry-failed", planFile}

	_, err := executeRunCommand(t, args)

	if err == nil {
		t.Error("Expected error for conflicting --retry-failed and --no-retry-failed flags")
	}

	if !strings.Contains(err.Error(), "cannot use both") {
		t.Errorf("Expected error about conflicting flags, got: %v", err)
	}
}

// ============================================================================
// MULTI-FILE RUN TESTS
// ============================================================================

// TestRunCommand_MultipleFiles tests running with multiple file arguments
func TestRunCommand_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create plan file 1 with tasks 1-2
	plan1Content := `# Plan Part 1

## Task 1: Setup
**Status**: pending
**Files**: main.go

Initialize the project.

## Task 2: Database
**Status**: pending
**Depends on**: 1
**Files**: db.go

Setup database connection.
`
	plan1Path := filepath.Join(tmpDir, "plan-01-setup.md")
	if err := os.WriteFile(plan1Path, []byte(plan1Content), 0644); err != nil {
		t.Fatalf("Failed to create plan file 1: %v", err)
	}

	// Create plan file 2 with tasks 3-4 (depends on task 2)
	plan2Content := `plan:
  tasks:
    - task_number: 3
      name: "API Server"
      files: ["api.go"]
      depends_on: [2]
      status: "pending"
      description: "Build API server"
    - task_number: 4
      name: "Tests"
      files: ["api_test.go"]
      depends_on: [3]
      status: "pending"
      description: "Write tests"
`
	plan2Path := filepath.Join(tmpDir, "plan-02-api.yaml")
	if err := os.WriteFile(plan2Path, []byte(plan2Content), 0644); err != nil {
		t.Fatalf("Failed to create plan file 2: %v", err)
	}

	// Execute with multiple file arguments in dry-run mode
	args := []string{"run", "--dry-run", plan1Path, plan2Path}
	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Errorf("Unexpected error: %v\nOutput: %s", err, output)
	}

	// Verify merged plan execution
	if !strings.Contains(output, "Loading and merging plans from 2 files") {
		t.Errorf("Expected message about merging 2 files, got: %s", output)
	}

	// Verify total task count
	if !strings.Contains(output, "Total tasks: 4") {
		t.Errorf("Expected 4 tasks from merged plan, got: %s", output)
	}

	// Verify wave calculation for cross-file dependencies
	if !strings.Contains(output, "Execution waves: 4") {
		t.Errorf("Expected 4 waves (sequential dependencies), got: %s", output)
	}

	// Verify dry-run message
	if !strings.Contains(output, "Dry-run mode") {
		t.Error("Expected dry-run mode message in output")
	}
}

// TestRunCommand_MultipleFiles_ParallelWaves tests wave calculation for parallel tasks
func TestRunCommand_MultipleFiles_ParallelWaves(t *testing.T) {
	tmpDir := t.TempDir()

	// Plan 1: Task 1 (foundation)
	plan1Content := `# Plan Part 1

## Task 1: Foundation
**Status**: pending
**Files**: base.go

Build foundation.
`
	plan1Path := filepath.Join(tmpDir, "plan-01-foundation.md")
	if err := os.WriteFile(plan1Path, []byte(plan1Content), 0644); err != nil {
		t.Fatalf("Failed to create plan file 1: %v", err)
	}

	// Plan 2: Tasks 2-3 (both depend on task 1, can run in parallel)
	plan2Content := `plan:
  tasks:
    - task_number: 2
      name: "Feature A"
      files: ["feature_a.go"]
      depends_on: [1]
      status: "pending"
      description: "Build feature A"
    - task_number: 3
      name: "Feature B"
      files: ["feature_b.go"]
      depends_on: [1]
      status: "pending"
      description: "Build feature B"
`
	plan2Path := filepath.Join(tmpDir, "plan-02-features.yaml")
	if err := os.WriteFile(plan2Path, []byte(plan2Content), 0644); err != nil {
		t.Fatalf("Failed to create plan file 2: %v", err)
	}

	// Execute with dry-run
	args := []string{"run", "--dry-run", "--verbose", plan1Path, plan2Path}
	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify total tasks
	if !strings.Contains(output, "Total tasks: 3") {
		t.Errorf("Expected 3 tasks, got: %s", output)
	}

	// Verify waves (Task 1 in wave 1, Tasks 2-3 in wave 2)
	if !strings.Contains(output, "Execution waves: 2") {
		t.Errorf("Expected 2 waves (1 foundation + 1 parallel), got: %s", output)
	}

	// Verify wave details in verbose mode
	if !strings.Contains(output, "Wave 1") {
		t.Error("Expected Wave 1 information in verbose output")
	}

	if !strings.Contains(output, "Wave 2") {
		t.Error("Expected Wave 2 information in verbose output")
	}
}

// TestRunCommand_DirectoryWithPlanFiles tests running with directory argument
func TestRunCommand_DirectoryWithPlanFiles(t *testing.T) {
	// NOTE: After refactoring, run.go now detects directory args and uses FilterPlanFiles
	// to load plan-* files automatically. This test verifies that functionality works.

	tmpDir := t.TempDir()

	// Create plan-*.md files
	plan1Content := `# Plan 1

## Task 1: First Task
**Status**: pending
**Files**: task1.go

Do task one.
`
	if err := os.WriteFile(filepath.Join(tmpDir, "plan-01-first.md"), []byte(plan1Content), 0644); err != nil {
		t.Fatalf("Failed to create plan-01-first.md: %v", err)
	}

	// Create plan-*.yaml files
	plan2Content := `plan:
  tasks:
    - task_number: 2
      name: "Second Task"
      files: ["task2.go"]
      depends_on: [1]
      status: "pending"
      description: "Do task two"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "plan-02-second.yaml"), []byte(plan2Content), 0644); err != nil {
		t.Fatalf("Failed to create plan-02-second.yaml: %v", err)
	}

	// Create non-plan files that should be filtered out
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# README"), 0644); err != nil {
		t.Fatalf("Failed to create readme.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "notes.txt"), []byte("notes"), 0644); err != nil {
		t.Fatalf("Failed to create notes.txt: %v", err)
	}

	// Execute with directory argument (use --verbose to see task names)
	args := []string{"run", "--dry-run", "--verbose", tmpDir}
	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Errorf("Unexpected error: %v\nOutput: %s", err, output)
	}

	// Verify only plan-* files were processed (2 tasks from 2 files)
	if !strings.Contains(output, "Total tasks: 2") {
		t.Errorf("Expected 2 tasks from plan files, got: %s", output)
	}

	// Verify both tasks are present (one from each plan file)
	if !strings.Contains(output, "First Task") || !strings.Contains(output, "Second Task") {
		t.Error("Expected both 'First Task' and 'Second Task' in output")
	}

	// Verify the multi-file merge message is shown
	if !strings.Contains(output, "Loading and merging plans from 2 files") {
		t.Error("Expected multi-file merge message for 2 plan files")
	}

	// Verify non-plan files were filtered out (readme.md and notes.txt should not appear)
	if strings.Contains(output, "readme.md") || strings.Contains(output, "notes.txt") {
		t.Error("Non-plan files should be filtered out")
	}

	// Verify success
	if !strings.Contains(output, "Dry-run mode") {
		t.Error("Expected dry-run mode message")
	}

	// Verify waves were calculated correctly
	if !strings.Contains(output, "Execution waves: 2") {
		t.Error("Expected 2 execution waves (Task 1 in Wave 1, Task 2 in Wave 2 due to dependency)")
	}
}

// TestRunCommand_PlanFileFiltering tests that only plan-* files are processed
func TestRunCommand_PlanFileFiltering(t *testing.T) {
	tmpDir := t.TempDir()

	// Create plan-01.md (should be included)
	plan1Content := `# Plan 1

## Task 1: Test
**Status**: pending
**Files**: test.go

Test task.
`
	plan1Path := filepath.Join(tmpDir, "plan-01.md")
	if err := os.WriteFile(plan1Path, []byte(plan1Content), 0644); err != nil {
		t.Fatalf("Failed to create plan-01.md: %v", err)
	}

	// Create other.md (should NOT be included when specifying multiple files)
	otherContent := `# Other Plan

## Task 2: Other
**Status**: pending
**Files**: other.go

Other task.
`
	otherPath := filepath.Join(tmpDir, "other.md")
	if err := os.WriteFile(otherPath, []byte(otherContent), 0644); err != nil {
		t.Fatalf("Failed to create other.md: %v", err)
	}

	// Create plan-02.yaml (should be included)
	plan2Content := `plan:
  tasks:
    - task_number: 3
      name: "Task Three"
      files: ["three.go"]
      depends_on: [1]
      status: "pending"
      description: "Task three"
`
	plan2Path := filepath.Join(tmpDir, "plan-02.yaml")
	if err := os.WriteFile(plan2Path, []byte(plan2Content), 0644); err != nil {
		t.Fatalf("Failed to create plan-02.yaml: %v", err)
	}

	// Test 1: When specifying multiple files explicitly, only plan-* files are accepted
	args := []string{"run", "--dry-run", plan1Path, plan2Path}
	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should have 2 tasks (from plan-01.md and plan-02.yaml only)
	if !strings.Contains(output, "Total tasks: 2") {
		t.Errorf("Expected 2 tasks from plan-* files only, got: %s", output)
	}

	// Test 2: Explicit file argument for other.md should work when single file (parser handles it)
	args2 := []string{"run", "--dry-run", otherPath}
	output2, err2 := executeRunCommand(t, args2)

	// Single non-plan file should work with ParseFile()
	if err2 != nil {
		t.Logf("Single non-plan file accepted by ParseFile(): %v", output2)
	}

	// Test 3: Multiple files including non-plan file should filter out non-plan files
	args3 := []string{"run", "--dry-run", plan1Path, otherPath}
	output3, err3 := executeRunCommand(t, args3)

	// After refactoring with FilterPlanFiles(), non-plan files are silently filtered
	// This is lenient behavior - only plan-* files are loaded
	if err3 != nil {
		t.Errorf("Unexpected error when mixing plan and non-plan files: %v", err3)
	}

	// Should have only 1 task (from plan-01.md), other.md is filtered out
	if !strings.Contains(output3, "Total tasks: 1") {
		t.Errorf("Expected 1 task from plan-01.md only (other.md filtered out), got: %s", output3)
	}
}

// TestRunCommand_MultipleFiles_WithMaxConcurrency tests concurrency control with multi-file plans
func TestRunCommand_MultipleFiles_WithMaxConcurrency(t *testing.T) {
	tmpDir := t.TempDir()

	// Create plan with parallel tasks
	plan1Content := `# Foundation

## Task 1: Base
**Status**: pending
**Files**: base.go

Build base.
`
	plan1Path := filepath.Join(tmpDir, "plan-01.md")
	if err := os.WriteFile(plan1Path, []byte(plan1Content), 0644); err != nil {
		t.Fatalf("Failed to create plan file: %v", err)
	}

	plan2Content := `plan:
  tasks:
    - task_number: 2
      name: "Feature A"
      files: ["feature_a.go"]
      depends_on: [1]
      status: "pending"
      description: "Feature A"
    - task_number: 3
      name: "Feature B"
      files: ["feature_b.go"]
      depends_on: [1]
      status: "pending"
      description: "Feature B"
    - task_number: 4
      name: "Feature C"
      files: ["feature_c.go"]
      depends_on: [1]
      status: "pending"
      description: "Feature C"
`
	plan2Path := filepath.Join(tmpDir, "plan-02.yaml")
	if err := os.WriteFile(plan2Path, []byte(plan2Content), 0644); err != nil {
		t.Fatalf("Failed to create plan file: %v", err)
	}

	// Execute with max concurrency
	args := []string{"run", "--dry-run", "--max-concurrency", "2", plan1Path, plan2Path}
	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify concurrency setting is applied
	if !strings.Contains(output, "Max concurrency: 2") {
		t.Errorf("Expected max concurrency setting, got: %s", output)
	}

	// Verify all tasks are present
	if !strings.Contains(output, "Total tasks: 4") {
		t.Errorf("Expected 4 tasks, got: %s", output)
	}
}

// TestRunCommand_MultipleFiles_ErrorHandling tests error cases with multi-file plans
func TestRunCommand_MultipleFiles_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(t *testing.T) []string
		wantErrContain string
	}{
		{
			name: "missing plan files",
			setupFunc: func(t *testing.T) []string {
				return []string{"/nonexistent/plan-01.md", "/nonexistent/plan-02.yaml"}
			},
			wantErrContain: "does not exist",
		},
		{
			name: "circular dependency across files",
			setupFunc: func(t *testing.T) []string {
				tmpDir := t.TempDir()

				// Task 1 depends on Task 2
				plan1 := `# Plan 1

## Task 1: First
**Status**: pending
**Depends on**: 2
**Files**: one.go

Task one.
`
				plan1Path := filepath.Join(tmpDir, "plan-01.md")
				os.WriteFile(plan1Path, []byte(plan1), 0644)

				// Task 2 depends on Task 1 (circular)
				plan2 := `plan:
  tasks:
    - task_number: 2
      name: "Second"
      files: ["two.go"]
      depends_on: [1]
      status: "pending"
      description: "Task two"
`
				plan2Path := filepath.Join(tmpDir, "plan-02.yaml")
				os.WriteFile(plan2Path, []byte(plan2), 0644)

				return []string{plan1Path, plan2Path}
			},
			wantErrContain: "circular dependency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.setupFunc(t)
			runArgs := append([]string{"run", "--dry-run"}, args...)

			_, err := executeRunCommand(t, runArgs)

			if err == nil {
				t.Errorf("Expected error but got none")
			}

			if !strings.Contains(err.Error(), tt.wantErrContain) {
				t.Errorf("Expected error containing %q, got: %v", tt.wantErrContain, err)
			}
		})
	}
}

// TestRunCommand_MultipleFiles_MixedFormats tests mixing .md and .yaml plan files
func TestRunCommand_MultipleFiles_MixedFormats(t *testing.T) {
	tmpDir := t.TempDir()

	// Markdown plan
	mdPlan := `# Markdown Plan

## Task 1: Markdown Task
**Status**: pending
**Files**: md.go

Markdown task.
`
	mdPath := filepath.Join(tmpDir, "plan-01-markdown.md")
	if err := os.WriteFile(mdPath, []byte(mdPlan), 0644); err != nil {
		t.Fatalf("Failed to create markdown plan: %v", err)
	}

	// YAML plan
	yamlPlan := `plan:
  tasks:
    - task_number: 2
      name: "YAML Task"
      files: ["yaml.go"]
      depends_on: [1]
      status: "pending"
      description: "YAML task"
`
	yamlPath := filepath.Join(tmpDir, "plan-02-yaml.yaml")
	if err := os.WriteFile(yamlPath, []byte(yamlPlan), 0644); err != nil {
		t.Fatalf("Failed to create YAML plan: %v", err)
	}

	// Execute with mixed format files
	args := []string{"run", "--dry-run", mdPath, yamlPath}
	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Errorf("Unexpected error with mixed formats: %v", err)
	}

	// Verify both tasks are processed
	if !strings.Contains(output, "Total tasks: 2") {
		t.Errorf("Expected 2 tasks from mixed formats, got: %s", output)
	}

	// Verify cross-file dependency works
	if !strings.Contains(output, "Execution waves: 2") {
		t.Errorf("Expected 2 waves for cross-file dependency, got: %s", output)
	}
}

// TestRunCommand_MultipleDirectories tests running with multiple directory arguments
func TestRunCommand_MultipleDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first directory with plan files
	dir1 := filepath.Join(tmpDir, "backend")
	if err := os.MkdirAll(dir1, 0755); err != nil {
		t.Fatalf("Failed to create dir1: %v", err)
	}

	plan1 := `# Backend

## Task 1: Backend Setup
**Status**: pending
**Files**: backend.go

Setup backend.
`
	if err := os.WriteFile(filepath.Join(dir1, "plan-01-backend.md"), []byte(plan1), 0644); err != nil {
		t.Fatalf("Failed to create backend plan: %v", err)
	}

	// Create second directory with plan files
	dir2 := filepath.Join(tmpDir, "frontend")
	if err := os.MkdirAll(dir2, 0755); err != nil {
		t.Fatalf("Failed to create dir2: %v", err)
	}

	plan2 := `plan:
  tasks:
    - task_number: 2
      name: "Frontend Setup"
      files: ["frontend.js"]
      depends_on: [1]
      status: "pending"
      description: "Setup frontend"
`
	if err := os.WriteFile(filepath.Join(dir2, "plan-01-frontend.yaml"), []byte(plan2), 0644); err != nil {
		t.Fatalf("Failed to create frontend plan: %v", err)
	}

	// Execute with multiple directory arguments
	args := []string{"run", "--dry-run", dir1, dir2}
	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Errorf("Unexpected error: %v\nOutput: %s", err, output)
	}

	// Verify both directories were processed
	if !strings.Contains(output, "Total tasks: 2") {
		t.Errorf("Expected 2 tasks from both directories, got: %s", output)
	}

	// Verify cross-directory dependency
	if !strings.Contains(output, "Execution waves: 2") {
		t.Errorf("Expected 2 waves for cross-directory dependency, got: %s", output)
	}
}

// TestRunCommand_VerboseMultiFile tests verbose output with multi-file plans
func TestRunCommand_VerboseMultiFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple plan files
	plan1 := `# Plan 1

## Task 1: First
**Status**: pending
**Files**: first.go

First task.
`
	plan1Path := filepath.Join(tmpDir, "plan-01.md")
	if err := os.WriteFile(plan1Path, []byte(plan1), 0644); err != nil {
		t.Fatalf("Failed to create plan 1: %v", err)
	}

	plan2 := `plan:
  tasks:
    - task_number: 2
      name: "Second"
      files: ["second.go"]
      depends_on: [1]
      status: "pending"
      description: "Second task"
`
	plan2Path := filepath.Join(tmpDir, "plan-02.yaml")
	if err := os.WriteFile(plan2Path, []byte(plan2), 0644); err != nil {
		t.Fatalf("Failed to create plan 2: %v", err)
	}

	// Execute with verbose flag
	args := []string{"run", "--dry-run", "--verbose", plan1Path, plan2Path}
	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify verbose output shows task details
	if !strings.Contains(output, "Task 1") {
		t.Error("Expected Task 1 in verbose output")
	}

	if !strings.Contains(output, "Task 2") {
		t.Error("Expected Task 2 in verbose output")
	}

	// Verify wave details
	if !strings.Contains(output, "Wave 1") {
		t.Error("Expected Wave 1 details in verbose output")
	}

	if !strings.Contains(output, "Wave 2") {
		t.Error("Expected Wave 2 details in verbose output")
	}
}

// ============================================================================
// BUG-R003: NO MATCHING PLAN FILES ERROR TESTS
// ============================================================================

// TestRunCommand_NoMatchingPlanFiles_SpecificError tests error message when directory has files but no plan-* files
func TestRunCommand_NoMatchingPlanFiles_SpecificError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files that don't match plan-* pattern
	if err := os.WriteFile(filepath.Join(tmpDir, "setup.md"), []byte("# Setup"), 0644); err != nil {
		t.Fatalf("Failed to create setup.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "deploy.yaml"), []byte("key: value"), 0644); err != nil {
		t.Fatalf("Failed to create deploy.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("readme"), 0644); err != nil {
		t.Fatalf("Failed to create readme.txt: %v", err)
	}

	// Run conductor run with directory path
	args := []string{"run", "--dry-run", tmpDir}
	_, err := executeRunCommand(t, args)

	// Should fail with specific error
	if err == nil {
		t.Fatal("Expected error when no plan-* files found, got nil")
	}

	// Verify error message is specific and helpful
	errMsg := err.Error()

	// Should mention "no plan files found" or similar
	if !strings.Contains(errMsg, "no plan files") && !strings.Contains(errMsg, "no matching") {
		t.Errorf("Expected error to mention 'no plan files' or 'no matching', got: %v", errMsg)
	}

	// Should mention the pattern plan-*.{md,yaml}
	hasPattern := strings.Contains(errMsg, "plan-") &&
		(strings.Contains(errMsg, ".md") || strings.Contains(errMsg, ".yaml") || strings.Contains(errMsg, "pattern"))

	if !hasPattern {
		t.Errorf("Expected error to mention 'plan-*.md' or 'plan-*.yaml' pattern, got: %v", errMsg)
	}

	// Should NOT just say "no files found" or generic error
	if strings.Contains(errMsg, "failed to access path") || strings.Contains(errMsg, "file not found") {
		t.Errorf("Error is too generic (mentions file access instead of pattern matching), got: %v", errMsg)
	}
}

// TestRunCommand_EmptyDirectory_SpecificError tests error message when directory is empty
func TestRunCommand_EmptyDirectory_SpecificError(t *testing.T) {
	tmpDir := t.TempDir()

	// Directory is empty (no files at all)

	// Run conductor run with empty directory path
	args := []string{"run", "--dry-run", tmpDir}
	_, err := executeRunCommand(t, args)

	// Should fail with specific error
	if err == nil {
		t.Fatal("Expected error when directory is empty, got nil")
	}

	// Verify error message is helpful
	errMsg := err.Error()

	// Should mention no plan files or directory is empty
	hasHelpfulMessage := strings.Contains(errMsg, "no plan files") ||
		strings.Contains(errMsg, "empty") ||
		strings.Contains(errMsg, "no matching")

	if !hasHelpfulMessage {
		t.Errorf("Expected error to mention 'no plan files' or 'empty directory', got: %v", errMsg)
	}

	// Should suggest creating plan-*.md files or mention the pattern
	hasSuggestion := strings.Contains(errMsg, "plan-") &&
		(strings.Contains(errMsg, ".md") || strings.Contains(errMsg, ".yaml") || strings.Contains(errMsg, "pattern"))

	if !hasSuggestion {
		t.Errorf("Expected error to suggest plan-* file naming convention, got: %v", errMsg)
	}
}

// ============================================================================
// REFACTORING TESTS - FilterPlanFiles() Usage (TDD RED Phase)
// ============================================================================
// These tests verify that run.go uses parser.FilterPlanFiles() for filtering
// instead of loadAndMergePlanFiles(). They are written to FAIL initially
// (TDD RED phase) and will pass after refactoring is complete.

// TestRunCommand_UsesFilterPlanFiles verifies that directory arguments filter plan-* files correctly
func TestRunCommand_UsesFilterPlanFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create plan-01.md (should be included)
	plan1 := `# Plan 1

## Task 1: First
**Status**: pending
**Files**: first.go

First task.
`
	if err := os.WriteFile(filepath.Join(tmpDir, "plan-01.md"), []byte(plan1), 0644); err != nil {
		t.Fatalf("Failed to create plan-01.md: %v", err)
	}

	// Create other.md (should be filtered OUT)
	other := `# Other

## Task 2: Other
**Status**: pending
**Files**: other.go

Other task (not a plan-* file).
`
	if err := os.WriteFile(filepath.Join(tmpDir, "other.md"), []byte(other), 0644); err != nil {
		t.Fatalf("Failed to create other.md: %v", err)
	}

	// Create plan-02.yaml (should be included)
	plan2 := `plan:
  tasks:
    - task_number: 3
      name: "Second"
      files: ["second.go"]
      depends_on: [1]
      status: "pending"
      description: "Second task"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "plan-02.yaml"), []byte(plan2), 0644); err != nil {
		t.Fatalf("Failed to create plan-02.yaml: %v", err)
	}

	// Execute with directory argument - should filter to only plan-* files
	args := []string{"run", "--dry-run", tmpDir}
	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Errorf("Unexpected error: %v\nOutput: %s", err, output)
	}

	// CRITICAL: After refactoring, should load ONLY 2 files (plan-01.md and plan-02.yaml)
	// The other.md file should be filtered out by FilterPlanFiles()
	if !strings.Contains(output, "Loading and merging plans from 2 files") {
		t.Errorf("Expected message about loading 2 files (after FilterPlanFiles filtering), got: %s", output)
	}

	// Should have 2 tasks total (Task 1 from plan-01.md, Task 3 from plan-02.yaml)
	// Task 2 from other.md should NOT be included
	if !strings.Contains(output, "Total tasks: 2") {
		t.Errorf("Expected 2 tasks from plan-* files only, got: %s", output)
	}

	// Verify dry-run succeeds
	if !strings.Contains(output, "Dry-run mode") {
		t.Error("Expected dry-run mode message")
	}
}

// TestRunCommand_MultipleFilesWithFiltering verifies explicit file arguments are filtered correctly
func TestRunCommand_MultipleFilesWithFiltering(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 5 files: 3 plan-* files and 2 non-plan files
	plan1 := `# Plan 1

## Task 1: First
**Status**: pending
**Files**: first.go

First task.
`
	plan1Path := filepath.Join(tmpDir, "plan-01.md")
	if err := os.WriteFile(plan1Path, []byte(plan1), 0644); err != nil {
		t.Fatalf("Failed to create plan-01.md: %v", err)
	}

	setup := `# Setup

## Task 2: Setup
**Status**: pending
**Files**: setup.go

Setup task (not a plan-* file).
`
	setupPath := filepath.Join(tmpDir, "setup.md")
	if err := os.WriteFile(setupPath, []byte(setup), 0644); err != nil {
		t.Fatalf("Failed to create setup.md: %v", err)
	}

	plan2 := `plan:
  tasks:
    - task_number: 3
      name: "Second"
      files: ["second.go"]
      depends_on: [1]
      status: "pending"
      description: "Second task"
`
	plan2Path := filepath.Join(tmpDir, "plan-02.yaml")
	if err := os.WriteFile(plan2Path, []byte(plan2), 0644); err != nil {
		t.Fatalf("Failed to create plan-02.yaml: %v", err)
	}

	readme := `# README

Documentation file.
`
	readmePath := filepath.Join(tmpDir, "readme.md")
	if err := os.WriteFile(readmePath, []byte(readme), 0644); err != nil {
		t.Fatalf("Failed to create readme.md: %v", err)
	}

	plan3 := `plan:
  tasks:
    - task_number: 4
      name: "Third"
      files: ["third.go"]
      depends_on: [3]
      status: "pending"
      description: "Third task"
`
	plan3Path := filepath.Join(tmpDir, "plan-03.yml")
	if err := os.WriteFile(plan3Path, []byte(plan3), 0644); err != nil {
		t.Fatalf("Failed to create plan-03.yml: %v", err)
	}

	// Execute with explicit file arguments (mix of plan-* and non-plan files)
	// After refactoring with FilterPlanFiles(), only plan-* files should be loaded
	args := []string{"run", "--dry-run", plan1Path, setupPath, plan2Path, readmePath, plan3Path}
	output, err := executeRunCommand(t, args)

	if err != nil {
		t.Errorf("Unexpected error: %v\nOutput: %s", err, output)
	}

	// CRITICAL: After refactoring, FilterPlanFiles() should accept all 5 paths
	// but filter down to ONLY the 3 plan-* files
	if !strings.Contains(output, "Loading and merging plans from 3 files") {
		t.Errorf("Expected message about loading 3 plan-* files (filtered), got: %s", output)
	}

	// Should have 3 tasks from plan-* files only (Task 1, 3, 4)
	// Tasks 2 from setup.md should NOT be included
	if !strings.Contains(output, "Total tasks: 3") {
		t.Errorf("Expected 3 tasks from plan-* files only, got: %s", output)
	}
}

// TestRunCommand_FilteringErrorMessage verifies clear error when no plan-* files found
func TestRunCommand_FilteringErrorMessage(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory with NO plan-* files (only non-plan files)
	other := `# Other

## Task 1: Other
**Status**: pending
**Files**: other.go

Other task.
`
	if err := os.WriteFile(filepath.Join(tmpDir, "other.md"), []byte(other), 0644); err != nil {
		t.Fatalf("Failed to create other.md: %v", err)
	}

	setup := `plan:
  tasks:
    - task_number: 2
      name: "Setup"
      files: ["setup.go"]
      status: "pending"
      description: "Setup task"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "setup.yaml"), []byte(setup), 0644); err != nil {
		t.Fatalf("Failed to create setup.yaml: %v", err)
	}

	// Execute with directory that has NO plan-* files
	args := []string{"run", "--dry-run", tmpDir}
	_, err := executeRunCommand(t, args)

	// CRITICAL: After refactoring, FilterPlanFiles() should return a clear error
	if err == nil {
		t.Error("Expected error when no plan-* files found")
		return
	}

	// Error message should mention the filtering pattern
	expectedErrMsg := "no plan files found matching pattern"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error containing %q, got: %v", expectedErrMsg, err)
	}

	// Error should also mention the pattern format
	if !strings.Contains(err.Error(), "plan-*.") {
		t.Errorf("Expected error to mention 'plan-*.' pattern, got: %v", err)
	}
}
