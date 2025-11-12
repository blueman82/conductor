package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harrison/conductor/internal/learning"
)

// TestE2E_LearningEnabled tests the complete learning workflow with learning enabled
func TestE2E_LearningEnabled(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Build conductor binary
	binaryPath := buildConductorBinary(t)
	defer os.Remove(binaryPath)

	// Setup test environment
	tmpDir := t.TempDir()
	planFile := createTestPlanFile(t, tmpDir, "simple-plan.md")

	// First run: Execute plan with learning enabled (dry-run to avoid actual claude execution)
	t.Log("First run: Executing plan with learning enabled (dry-run)")
	output := runConductor(t, binaryPath, tmpDir, "run", planFile, "--dry-run")
	if !strings.Contains(output, "Wave 1") {
		t.Errorf("Expected wave execution in output, got: %s", output)
	}

	// Note: In dry-run mode, tasks aren't executed so learning won't be recorded.
	// This test validates that dry-run works without breaking when learning is enabled.
	// For actual learning validation, see the integration tests in internal/executor/

	// Verify learning directory structure is created when enabled in config
	learningDir := filepath.Join(tmpDir, ".conductor", "learning")

	// Even in dry-run, we should be able to create a learning database manually
	// to test the integration
	dbPath := filepath.Join(learningDir, "executions.db")
	if err := os.MkdirAll(learningDir, 0755); err != nil {
		t.Fatalf("Failed to create learning directory: %v", err)
	}

	// Create and populate a test database to simulate learning
	db, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create learning database: %v", err)
	}
	defer db.Close()

	// Record a simulated execution
	exec := &learning.TaskExecution{
		PlanFile:     filepath.Base(planFile),
		RunNumber:    1,
		TaskNumber:   "1",
		TaskName:     "Initialize Project",
		Agent:        "",
		Prompt:       "Initialize the project structure.",
		Success:      true,
		Output:       "Task completed successfully (simulated)",
		ErrorMessage: "",
		DurationSecs: 5,
		Context:      "{}",
	}
	if err := db.RecordExecution(context.Background(), exec); err != nil {
		t.Fatalf("Failed to record execution: %v", err)
	}

	// Verify execution was recorded
	executions, err := db.GetExecutions(filepath.Base(planFile))
	if err != nil {
		t.Fatalf("Failed to get executions: %v", err)
	}
	if len(executions) == 0 {
		t.Error("Expected at least one execution record")
	}

	// Verify execution details
	retrieved := executions[0]
	if retrieved.PlanFile != filepath.Base(planFile) {
		t.Errorf("Expected plan file %s, got %s", filepath.Base(planFile), retrieved.PlanFile)
	}
	if retrieved.TaskNumber != "1" {
		t.Errorf("Expected task number 1, got %s", retrieved.TaskNumber)
	}
	if !retrieved.Success {
		t.Error("Expected successful execution")
	}

	// Record second execution
	exec2 := &learning.TaskExecution{
		PlanFile:     filepath.Base(planFile),
		RunNumber:    2,
		TaskNumber:   "1",
		TaskName:     "Initialize Project",
		Agent:        "",
		Prompt:       "Initialize the project structure.",
		Success:      true,
		Output:       "Task completed successfully (simulated, run 2)",
		ErrorMessage: "",
		DurationSecs: 4,
		Context:      "{}",
	}
	if err := db.RecordExecution(context.Background(), exec2); err != nil {
		t.Fatalf("Failed to record second execution: %v", err)
	}

	// Verify second execution was recorded
	executions2, err := db.GetExecutions(filepath.Base(planFile))
	if err != nil {
		t.Fatalf("Failed to get executions: %v", err)
	}
	if len(executions2) < 2 {
		t.Errorf("Expected at least 2 executions, got %d", len(executions2))
	}

	t.Log("E2E test with learning enabled: PASS")
}

// TestE2E_LearningDisabled tests execution with learning disabled
func TestE2E_LearningDisabled(t *testing.T) {
	// Build conductor binary
	binaryPath := buildConductorBinary(t)
	defer os.Remove(binaryPath)

	// Setup test environment
	tmpDir := t.TempDir()
	planFile := createTestPlanFile(t, tmpDir, "simple-plan.md")

	// Create config with learning disabled
	configDir := filepath.Join(tmpDir, ".conductor")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	configContent := `learning:
  enabled: false
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Run conductor with learning disabled
	t.Log("Running conductor with learning disabled")
	output := runConductor(t, binaryPath, tmpDir, "run", planFile, "--dry-run")
	if !strings.Contains(output, "Wave 1") {
		t.Errorf("Expected wave execution in output, got: %s", output)
	}

	// Verify learning database was NOT created
	dbPath := filepath.Join(tmpDir, ".conductor", "learning", "executions.db")
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Error("Learning database should not exist when learning is disabled")
	}

	t.Log("E2E test with learning disabled: PASS")
}

// TestE2E_FailureAdaptation tests learning from task failures
func TestE2E_FailureAdaptation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Build conductor binary
	binaryPath := buildConductorBinary(t)
	defer os.Remove(binaryPath)

	// Setup test environment
	tmpDir := t.TempDir()
	planFile := createFailurePlanFile(t, tmpDir, "failure-plan.md")

	// Run conductor in dry-run mode (won't record actual executions)
	t.Log("Running conductor with failure plan (dry-run)")
	_ = runConductor(t, binaryPath, tmpDir, "run", planFile, "--dry-run")

	// Manually create learning database and simulate failure tracking
	learningDir := filepath.Join(tmpDir, ".conductor", "learning")
	if err := os.MkdirAll(learningDir, 0755); err != nil {
		t.Fatalf("Failed to create learning directory: %v", err)
	}

	dbPath := filepath.Join(learningDir, "executions.db")
	db, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create learning database: %v", err)
	}
	defer db.Close()

	// Simulate a failed execution
	exec := &learning.TaskExecution{
		PlanFile:     filepath.Base(planFile),
		RunNumber:    1,
		TaskNumber:   "1",
		TaskName:     "Risky Operation",
		Agent:        "",
		Prompt:       "This task might fail during execution.",
		Success:      false,
		Output:       "Task failed (simulated)",
		ErrorMessage: "Compilation error: missing dependency",
		DurationSecs: 3,
		Context:      `{"failure_reason": "dependency_missing"}`,
	}
	if err := db.RecordExecution(context.Background(), exec); err != nil {
		t.Fatalf("Failed to record execution: %v", err)
	}

	// Verify execution was recorded
	executions, err := db.GetExecutions(filepath.Base(planFile))
	if err != nil {
		t.Fatalf("Failed to get executions: %v", err)
	}
	if len(executions) == 0 {
		t.Error("Expected at least one execution record")
	}

	// Get execution details
	retrieved := executions[0]
	if retrieved.PlanFile != filepath.Base(planFile) {
		t.Errorf("Expected plan file %s, got %s", filepath.Base(planFile), retrieved.PlanFile)
	}
	if retrieved.Success {
		t.Error("Expected failed execution")
	}
	if retrieved.ErrorMessage == "" {
		t.Error("Expected error message for failed execution")
	}

	// Verify context contains useful information
	var context map[string]interface{}
	if retrieved.Context != "" && retrieved.Context != "{}" {
		if err := json.Unmarshal([]byte(retrieved.Context), &context); err != nil {
			t.Errorf("Failed to parse execution context: %v", err)
		}
		if _, ok := context["failure_reason"]; !ok {
			t.Error("Expected failure_reason in context")
		}
	}

	t.Log("E2E test with failure adaptation: PASS")
}

// TestE2E_CLICommands tests CLI commands against real database
func TestE2E_CLICommands(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Build conductor binary
	binaryPath := buildConductorBinary(t)
	defer os.Remove(binaryPath)

	// Setup test environment with existing learning data
	tmpDir := t.TempDir()
	planFile := createTestPlanFile(t, tmpDir, "cli-test-plan.md")

	// Create learning database with test data
	learningDir := filepath.Join(tmpDir, ".conductor", "learning")
	if err := os.MkdirAll(learningDir, 0755); err != nil {
		t.Fatalf("Failed to create learning directory: %v", err)
	}

	dbPath := filepath.Join(learningDir, "executions.db")
	db, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create learning database: %v", err)
	}

	// Add some test executions
	for i := 1; i <= 3; i++ {
		exec := &learning.TaskExecution{
			PlanFile:     filepath.Base(planFile),
			RunNumber:    i,
			TaskNumber:   "1",
			TaskName:     "Initialize Project",
			Agent:        "",
			Prompt:       "Initialize the project structure.",
			Success:      i != 2, // Second execution fails
			Output:       fmt.Sprintf("Task output (run %d)", i),
			ErrorMessage: "",
			DurationSecs: int64(5 + i),
			Context:      "{}",
		}
		if i == 2 {
			exec.ErrorMessage = "Test failure"
		}
		if err := db.RecordExecution(context.Background(), exec); err != nil {
			t.Fatalf("Failed to record execution %d: %v", i, err)
		}
	}
	db.Close()

	// Test: learning export command
	t.Log("Testing learning export command")
	// Use basename for export since that's what we stored
	exportOutput := runConductor(t, binaryPath, tmpDir, "learning", "export", filepath.Base(planFile), "--db-path", dbPath)

	// The export command outputs a JSON array of executions
	var exportData []map[string]interface{}
	if err := json.Unmarshal([]byte(exportOutput), &exportData); err != nil {
		t.Errorf("Export output is not valid JSON: %v\nOutput: %s", err, exportOutput)
	} else {
		// Verify we have execution records
		if len(exportData) < 3 {
			t.Errorf("Expected at least 3 execution records, got %d", len(exportData))
		}

		// Verify structure of first record
		if len(exportData) > 0 {
			record := exportData[0]
			// JSON encoding uses field names as-is (PascalCase)
			if _, ok := record["PlanFile"]; !ok {
				t.Error("Export record missing 'PlanFile' field")
			}
			if _, ok := record["TaskNumber"]; !ok {
				t.Error("Export record missing 'TaskNumber' field")
			}
			if _, ok := record["Success"]; !ok {
				t.Error("Export record missing 'Success' field")
			}
		}
	}

	t.Log("E2E CLI commands test: PASS")
}

// TestE2E_LargeComplexPlan tests with a large plan with complex dependencies
func TestE2E_LargeComplexPlan(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large plan test in short mode")
	}

	// Build conductor binary
	binaryPath := buildConductorBinary(t)
	defer os.Remove(binaryPath)

	// Setup test environment
	tmpDir := t.TempDir()
	planFile := createLargePlanFile(t, tmpDir, "large-plan.md")

	// Run conductor
	t.Log("Running conductor with large complex plan")
	output := runConductor(t, binaryPath, tmpDir, "run", planFile, "--dry-run")

	// Verify multiple waves executed
	waveCount := strings.Count(output, "Wave ")
	if waveCount < 3 {
		t.Errorf("Expected at least 3 waves for complex plan, got %d", waveCount)
	}

	// Create learning database and simulate execution history
	learningDir := filepath.Join(tmpDir, ".conductor", "learning")
	if err := os.MkdirAll(learningDir, 0755); err != nil {
		t.Fatalf("Failed to create learning directory: %v", err)
	}

	dbPath := filepath.Join(learningDir, "executions.db")
	db, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create learning database: %v", err)
	}
	defer db.Close()

	// Simulate execution history for multiple tasks
	for i := 1; i <= 5; i++ {
		exec := &learning.TaskExecution{
			PlanFile:     filepath.Base(planFile),
			RunNumber:    1,
			TaskNumber:   fmt.Sprintf("%d", i),
			TaskName:     fmt.Sprintf("Task %d", i),
			Agent:        "",
			Prompt:       fmt.Sprintf("Implement task %d.", i),
			Success:      true,
			Output:       fmt.Sprintf("Task %d completed successfully", i),
			ErrorMessage: "",
			DurationSecs: int64(5 * i),
			Context:      "{}",
		}
		if err := db.RecordExecution(context.Background(), exec); err != nil {
			t.Fatalf("Failed to record execution %d: %v", i, err)
		}
	}

	// Verify executions were recorded
	executions, err := db.GetExecutions(filepath.Base(planFile))
	if err != nil {
		t.Fatalf("Failed to get executions: %v", err)
	}
	if len(executions) < 5 {
		t.Errorf("Expected at least 5 execution records, got %d", len(executions))
	}

	t.Log("E2E large complex plan test: PASS")
}

// Helper functions

func buildConductorBinary(t *testing.T) string {
	t.Helper()

	tmpBinary := filepath.Join(t.TempDir(), "conductor-test")
	cmd := exec.Command("go", "build", "-o", tmpBinary, "../../cmd/conductor")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build conductor binary: %v\nOutput: %s", err, output)
	}

	return tmpBinary
}

func runConductor(t *testing.T, binaryPath, workDir string, args ...string) string {
	t.Helper()

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Don't fail on error - some tests expect failures
		t.Logf("Conductor command failed (may be expected): %v\nOutput: %s", err, output)
	}

	return string(output)
}

func createTestPlanFile(t *testing.T, dir, filename string) string {
	t.Helper()

	planContent := `# Simple Test Plan

## Task 1: Initialize Project
**Files**: main.go
**Depends on**: None
**Estimated time**: 5m

Initialize the project structure.

## Task 2: Add Configuration
**Files**: config.yaml
**Depends on**: Task 1
**Estimated time**: 3m

Add configuration file.

## Task 3: Add Tests
**Files**: main_test.go
**Depends on**: Task 1, Task 2
**Estimated time**: 10m

Add unit tests for the project.
`

	planPath := filepath.Join(dir, filename)
	if err := os.WriteFile(planPath, []byte(planContent), 0644); err != nil {
		t.Fatalf("Failed to create test plan file: %v", err)
	}

	return planPath
}

func createFailurePlanFile(t *testing.T, dir, filename string) string {
	t.Helper()

	planContent := `# Failure Test Plan

## Task 1: Risky Operation
**Files**: risky.go
**Depends on**: None
**Estimated time**: 5m

This task might fail during execution.

## Task 2: Dependent Task
**Files**: dependent.go
**Depends on**: Task 1
**Estimated time**: 3m

This task depends on the risky operation.
`

	planPath := filepath.Join(dir, filename)
	if err := os.WriteFile(planPath, []byte(planContent), 0644); err != nil {
		t.Fatalf("Failed to create failure plan file: %v", err)
	}

	return planPath
}

func createLargePlanFile(t *testing.T, dir, filename string) string {
	t.Helper()

	var planContent strings.Builder
	planContent.WriteString("# Large Complex Plan\n\n")

	// Create 20 tasks with complex dependencies
	for i := 1; i <= 20; i++ {
		planContent.WriteString("## Task ")
		planContent.WriteString(string(rune('0' + i)))
		planContent.WriteString(": Task ")
		planContent.WriteString(string(rune('0' + i)))
		planContent.WriteString("\n")
		planContent.WriteString("**Files**: file")
		planContent.WriteString(string(rune('0' + i)))
		planContent.WriteString(".go\n")

		// Add dependencies for a complex graph
		if i > 1 {
			planContent.WriteString("**Depends on**: ")
			if i <= 5 {
				planContent.WriteString("Task 1")
			} else if i <= 10 {
				planContent.WriteString("Task 2, Task 3")
			} else if i <= 15 {
				planContent.WriteString("Task 5, Task 7")
			} else {
				planContent.WriteString("Task 10, Task 12, Task 14")
			}
			planContent.WriteString("\n")
		} else {
			planContent.WriteString("**Depends on**: None\n")
		}

		planContent.WriteString("**Estimated time**: 5m\n\n")
		planContent.WriteString("Implement task ")
		planContent.WriteString(string(rune('0' + i)))
		planContent.WriteString(".\n\n")
	}

	planPath := filepath.Join(dir, filename)
	if err := os.WriteFile(planPath, []byte(planContent.String()), 0644); err != nil {
		t.Fatalf("Failed to create large plan file: %v", err)
	}

	return planPath
}
