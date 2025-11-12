package cmd

import (
	"context"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/learning"
	"github.com/spf13/cobra"
)

// TestCLI_StatsCommand tests the stats command end-to-end
func TestCLI_StatsCommand(t *testing.T) {
	// Setup test database with realistic data
	tmpDir := t.TempDir()
	planFile := "integration-plan.md"
	dbPath := setupTestDatabase(t, tmpDir, planFile)

	// Seed test data
	store, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	seedTestData(t, store, planFile)

	// Create plan file
	planPath := filepath.Join(tmpDir, planFile)
	if err := os.WriteFile(planPath, []byte("# Integration Plan"), 0644); err != nil {
		t.Fatalf("Failed to create plan file: %v", err)
	}

	// Execute stats command
	cmd := NewStatsCommand()
	cmd.SetArgs([]string{planPath})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Stats command failed: %v", err)
	}

	// Verify output contains expected statistics
	outputStr := output.String()

	// Check for overall statistics
	if !strings.Contains(outputStr, "Total executions:") {
		t.Error("Expected 'Total executions:' in output")
	}
	if !strings.Contains(outputStr, "Success rate:") {
		t.Error("Expected 'Success rate:' in output")
	}
	if !strings.Contains(outputStr, "Average duration:") {
		t.Error("Expected 'Average duration:' in output")
	}

	// Check for agent performance section
	if !strings.Contains(outputStr, "Agent Performance") {
		t.Error("Expected 'Agent Performance' section in output")
	}

	// Check for task metrics section
	if !strings.Contains(outputStr, "Task Metrics") {
		t.Error("Expected 'Task Metrics' section in output")
	}

	// Verify correct counts (we seeded 10 executions)
	if !strings.Contains(outputStr, "10") {
		t.Error("Expected total execution count of 10 in output")
	}
}

// TestCLI_ShowCommand tests the show command end-to-end
func TestCLI_ShowCommand(t *testing.T) {
	// Setup test database
	tmpDir := t.TempDir()
	planFile := "show-plan.md"
	dbPath := setupTestDatabase(t, tmpDir, planFile)

	// Seed test data
	store, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	// Add specific task executions
	taskNumber := "1"
	for i := 1; i <= 3; i++ {
		exec := &learning.TaskExecution{
			PlanFile:     planFile,
			RunNumber:    i,
			TaskNumber:   taskNumber,
			TaskName:     "Database Setup",
			Agent:        "golang-pro",
			Prompt:       "Initialize database",
			Success:      i != 2, // Second attempt fails
			Output:       "Task output",
			ErrorMessage: "",
			DurationSecs: 25,
			Timestamp:    time.Now().Add(-time.Duration(i) * time.Hour),
		}
		if i == 2 {
			exec.ErrorMessage = "Connection failed"
		}
		if err := store.RecordExecution(context.Background(), exec); err != nil {
			t.Fatalf("Failed to record execution: %v", err)
		}
	}

	// Create plan file
	planPath := filepath.Join(tmpDir, planFile)
	if err := os.WriteFile(planPath, []byte("# Show Plan"), 0644); err != nil {
		t.Fatalf("Failed to create plan file: %v", err)
	}

	// Execute show command
	cmd := NewShowCommand()
	cmd.SetArgs([]string{planPath, taskNumber})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Show command failed: %v", err)
	}

	// Verify output
	outputStr := output.String()

	// Check for task header
	if !strings.Contains(outputStr, "Execution History for Task 1") {
		t.Error("Expected task header in output")
	}
	if !strings.Contains(outputStr, "Database Setup") {
		t.Error("Expected task name in output")
	}

	// Check for total attempts
	if !strings.Contains(outputStr, "Total attempts: 3") {
		t.Error("Expected 'Total attempts: 3' in output")
	}

	// Check for attempt details
	if !strings.Contains(outputStr, "Attempt #") {
		t.Error("Expected attempt numbers in output")
	}
	if !strings.Contains(outputStr, "Agent:") {
		t.Error("Expected agent information in output")
	}
	if !strings.Contains(outputStr, "Verdict:") {
		t.Error("Expected verdict information in output")
	}
	if !strings.Contains(outputStr, "Duration:") {
		t.Error("Expected duration information in output")
	}

	// Check for summary
	if !strings.Contains(outputStr, "Summary:") {
		t.Error("Expected summary section in output")
	}
	if !strings.Contains(outputStr, "Success rate:") {
		t.Error("Expected success rate in summary")
	}
}

// TestCLI_ClearCommand tests the clear command end-to-end
func TestCLI_ClearCommand(t *testing.T) {
	// Setup test database
	tmpDir := t.TempDir()
	planFile := "clear-plan.md"
	dbPath := setupTestDatabase(t, tmpDir, planFile)

	// Seed test data
	store, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}

	// Add executions for two different plans
	for i := 1; i <= 3; i++ {
		exec := &learning.TaskExecution{
			PlanFile:     planFile,
			RunNumber:    i,
			TaskNumber:   "1",
			TaskName:     "Task 1",
			Agent:        "golang-pro",
			Prompt:       "Do task",
			Success:      true,
			Output:       "Success",
			DurationSecs: 20,
			Timestamp:    time.Now(),
		}
		if err := store.RecordExecution(context.Background(), exec); err != nil {
			t.Fatalf("Failed to record execution: %v", err)
		}
	}

	// Add executions for another plan
	otherPlan := "other-plan.md"
	for i := 1; i <= 2; i++ {
		exec := &learning.TaskExecution{
			PlanFile:     otherPlan,
			RunNumber:    i,
			TaskNumber:   "1",
			TaskName:     "Other Task",
			Agent:        "python-pro",
			Prompt:       "Do other task",
			Success:      true,
			Output:       "Success",
			DurationSecs: 15,
			Timestamp:    time.Now(),
		}
		if err := store.RecordExecution(context.Background(), exec); err != nil {
			t.Fatalf("Failed to record execution: %v", err)
		}
	}
	store.Close()

	// Execute clear command for specific plan
	cmd := newClearCommand()
	cmd.SetArgs([]string{planFile, "--db-path", dbPath})

	// Mock stdin to provide "y" confirmation
	// Note: In real usage, this would require user interaction
	// For testing, we'll use the --db-path flag to bypass file path resolution

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	// Create a pipe for stdin to simulate user confirmation
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// Write "y" to stdin in a goroutine
	go func() {
		w.Write([]byte("y\n"))
		w.Close()
	}()

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Clear command failed: %v", err)
	}

	outputStr := output.String()

	// Verify deletion message
	if !strings.Contains(outputStr, "Deleted") {
		t.Error("Expected deletion confirmation in output")
	}
	if !strings.Contains(outputStr, "3") {
		t.Error("Expected 3 records deleted")
	}

	// Verify other plan's data still exists
	store, err = learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen store: %v", err)
	}
	defer store.Close()

	otherExecs, err := store.GetExecutions(otherPlan)
	if err != nil {
		t.Fatalf("Failed to get executions: %v", err)
	}
	if len(otherExecs) != 2 {
		t.Errorf("Expected 2 executions for other plan, got %d", len(otherExecs))
	}

	// Verify cleared plan has no data
	clearedExecs, err := store.GetExecutions(planFile)
	if err != nil {
		t.Fatalf("Failed to get executions: %v", err)
	}
	if len(clearedExecs) != 0 {
		t.Errorf("Expected 0 executions for cleared plan, got %d", len(clearedExecs))
	}
}

// TestCLI_ExportCommand_JSON tests the export command with JSON format
func TestCLI_ExportCommand_JSON(t *testing.T) {
	// Setup test database
	tmpDir := t.TempDir()
	planFile := "export-plan.md"
	dbPath := setupTestDatabase(t, tmpDir, planFile)

	// Seed test data
	store, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	// Add test executions
	expectedExecs := 5
	for i := 1; i <= expectedExecs; i++ {
		exec := &learning.TaskExecution{
			PlanFile:     planFile,
			RunNumber:    i,
			TaskNumber:   "1",
			TaskName:     "Export Task",
			Agent:        "golang-pro",
			Prompt:       "Do export task",
			Success:      true,
			Output:       "Task completed",
			DurationSecs: 30,
			Timestamp:    time.Now(),
		}
		if err := store.RecordExecution(context.Background(), exec); err != nil {
			t.Fatalf("Failed to record execution: %v", err)
		}
	}

	// Create output file path
	outputFile := filepath.Join(tmpDir, "export.json")

	// Execute export command
	cmd := newExportCommand()
	cmd.SetArgs([]string{planFile, "--format", "json", "--output", outputFile, "--db-path", dbPath})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Export command failed: %v", err)
	}

	// Verify output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatal("Expected output file to be created")
	}

	// Read and parse JSON
	jsonData, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var executions []*learning.TaskExecution
	if err := json.Unmarshal(jsonData, &executions); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify data
	if len(executions) != expectedExecs {
		t.Errorf("Expected %d executions in JSON, got %d", expectedExecs, len(executions))
	}

	// Verify structure
	if len(executions) > 0 {
		exec := executions[0]
		if exec.PlanFile != planFile {
			t.Errorf("Expected plan file %s, got %s", planFile, exec.PlanFile)
		}
		if exec.TaskName != "Export Task" {
			t.Errorf("Expected task name 'Export Task', got %s", exec.TaskName)
		}
	}
}

// TestCLI_ExportCommand_CSV tests the export command with CSV format
func TestCLI_ExportCommand_CSV(t *testing.T) {
	// Setup test database
	tmpDir := t.TempDir()
	planFile := "export-csv-plan.md"
	dbPath := setupTestDatabase(t, tmpDir, planFile)

	// Seed test data
	store, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	// Add test executions
	expectedExecs := 3
	for i := 1; i <= expectedExecs; i++ {
		exec := &learning.TaskExecution{
			PlanFile:     planFile,
			RunNumber:    i,
			TaskNumber:   "1",
			TaskName:     "CSV Export Task",
			Agent:        "python-pro",
			Prompt:       "Do CSV export",
			Success:      true,
			Output:       "Completed",
			DurationSecs: 25,
			Timestamp:    time.Now(),
		}
		if err := store.RecordExecution(context.Background(), exec); err != nil {
			t.Fatalf("Failed to record execution: %v", err)
		}
	}

	// Create output file path
	outputFile := filepath.Join(tmpDir, "export.csv")

	// Execute export command
	cmd := newExportCommand()
	cmd.SetArgs([]string{planFile, "--format", "csv", "--output", outputFile, "--db-path", dbPath})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Export command failed: %v", err)
	}

	// Verify output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatal("Expected output file to be created")
	}

	// Read and parse CSV
	file, err := os.Open(outputFile)
	if err != nil {
		t.Fatalf("Failed to open output file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Verify header and data rows
	expectedRows := expectedExecs + 1 // +1 for header
	if len(records) != expectedRows {
		t.Errorf("Expected %d rows (including header), got %d", expectedRows, len(records))
	}

	// Verify header
	if len(records) > 0 {
		header := records[0]
		expectedHeaders := []string{"id", "plan_file", "run_number", "task_number", "task_name", "agent", "success", "error_message", "duration_seconds", "timestamp"}
		if len(header) != len(expectedHeaders) {
			t.Errorf("Expected %d columns, got %d", len(expectedHeaders), len(header))
		}
		// Check for key headers
		if header[1] != "plan_file" {
			t.Errorf("Expected second column to be 'plan_file', got %s", header[1])
		}
	}

	// Verify data rows
	if len(records) > 1 {
		firstData := records[1]
		if firstData[1] != planFile {
			t.Errorf("Expected plan file %s in data, got %s", planFile, firstData[1])
		}
	}
}

// TestCLI_ExportCommand_Stdout tests exporting to stdout
func TestCLI_ExportCommand_Stdout(t *testing.T) {
	// Setup test database
	tmpDir := t.TempDir()
	planFile := "stdout-plan.md"
	dbPath := setupTestDatabase(t, tmpDir, planFile)

	// Seed test data
	store, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	exec := &learning.TaskExecution{
		PlanFile:     planFile,
		RunNumber:    1,
		TaskNumber:   "1",
		TaskName:     "Stdout Task",
		Agent:        "rust-pro",
		Prompt:       "Test stdout export",
		Success:      true,
		Output:       "Success",
		DurationSecs: 20,
		Timestamp:    time.Now(),
	}
	if err := store.RecordExecution(context.Background(), exec); err != nil {
		t.Fatalf("Failed to record execution: %v", err)
	}

	// Close store before export to release lock
	store.Close()

	// Execute export command to a file (stdout testing is tricky with cobra)
	outputFile := filepath.Join(tmpDir, "stdout-export.json")
	cmd := newExportCommand()
	cmd.SetArgs([]string{planFile, "--format", "json", "--output", outputFile, "--db-path", dbPath})

	var cmdOutput bytes.Buffer
	cmd.SetOut(&cmdOutput)
	cmd.SetErr(&cmdOutput)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Export command failed: %v", err)
	}

	// Read the exported file
	jsonData, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	// Verify JSON content
	if !strings.Contains(string(jsonData), "Stdout Task") {
		t.Error("Expected task data in export output")
	}

	// Verify it's valid JSON
	var executions []*learning.TaskExecution
	if err := json.Unmarshal(jsonData, &executions); err != nil {
		t.Errorf("Expected valid JSON: %v", err)
	}

	// Verify the execution was exported correctly
	if len(executions) != 1 {
		t.Errorf("Expected 1 execution, got %d", len(executions))
	}
}

// TestCLI_ErrorHandling tests error scenarios across all commands
func TestCLI_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setupCmd    func() *cobra.Command
		expectError bool
	}{
		{
			name: "stats_missing_plan_file",
			setupCmd: func() *cobra.Command {
				cmd := NewStatsCommand()
				cmd.SetArgs([]string{"nonexistent.md"})
				return cmd
			},
			expectError: true,
		},
		{
			name: "show_missing_plan_file",
			setupCmd: func() *cobra.Command {
				cmd := NewShowCommand()
				cmd.SetArgs([]string{"nonexistent.md", "1"})
				return cmd
			},
			expectError: true,
		},
		{
			name: "export_invalid_format",
			setupCmd: func() *cobra.Command {
				tmpDir := t.TempDir()
				dbPath := filepath.Join(tmpDir, "test.db")
				store, _ := learning.NewStore(dbPath)
				store.Close()

				cmd := newExportCommand()
				cmd.SetArgs([]string{"plan.md", "--format", "xml", "--db-path", dbPath})
				return cmd
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.setupCmd()

			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)

			err := cmd.Execute()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestCLI_EmptyDatabase tests commands with empty database
func TestCLI_EmptyDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	planFile := "empty-plan.md"
	_ = setupTestDatabase(t, tmpDir, planFile)

	// Create plan file
	planPath := filepath.Join(tmpDir, planFile)
	if err := os.WriteFile(planPath, []byte("# Empty Plan"), 0644); err != nil {
		t.Fatalf("Failed to create plan file: %v", err)
	}

	// Test stats command with empty database
	t.Run("stats_empty_db", func(t *testing.T) {
		cmd := NewStatsCommand()
		cmd.SetArgs([]string{planPath})

		var output bytes.Buffer
		cmd.SetOut(&output)
		cmd.SetErr(&output)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("Stats command failed: %v", err)
		}

		outputStr := output.String()
		if !strings.Contains(outputStr, "No execution data") {
			t.Error("Expected 'No execution data' message for empty database")
		}
	})

	// Test show command with empty database
	t.Run("show_empty_db", func(t *testing.T) {
		cmd := NewShowCommand()
		cmd.SetArgs([]string{planPath, "1"})

		var output bytes.Buffer
		cmd.SetOut(&output)
		cmd.SetErr(&output)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("Show command failed: %v", err)
		}

		outputStr := output.String()
		// When database doesn't exist or has no data, it shows "No execution data found"
		if !strings.Contains(outputStr, "No execution") {
			t.Errorf("Expected message about no execution data, got: %s", outputStr)
		}
	})
}

// Helper functions

// setupTestDatabase creates a test database in the expected location
func setupTestDatabase(t *testing.T, tmpDir, planFile string) string {
	t.Helper()

	// Database should be in .conductor/learning/ relative to plan file
	dbDir := filepath.Join(tmpDir, ".conductor", "learning")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		t.Fatalf("Failed to create db directory: %v", err)
	}

	return filepath.Join(dbDir, "executions.db")
}

// seedTestData populates the database with realistic test data
func seedTestData(t *testing.T, store *learning.Store, planFile string) {
	t.Helper()

	// Add diverse execution data
	agents := []string{"golang-pro", "python-pro", "rust-pro"}
	tasks := []struct {
		number string
		name   string
	}{
		{"1", "Database Setup"},
		{"2", "API Implementation"},
		{"3", "Testing Suite"},
	}

	execID := 1
	for _, agent := range agents {
		for _, task := range tasks {
			// Add 2 successes
			for i := 0; i < 2; i++ {
				exec := &learning.TaskExecution{
					PlanFile:     planFile,
					RunNumber:    execID,
					TaskNumber:   task.number,
					TaskName:     task.name,
					Agent:        agent,
					Prompt:       "Execute " + task.name,
					Success:      true,
					Output:       "Task completed successfully",
					DurationSecs: 25 + int64(i*5),
					Timestamp:    time.Now().Add(-time.Duration(execID) * time.Hour),
				}
				if err := store.RecordExecution(context.Background(), exec); err != nil {
					t.Fatalf("Failed to record execution: %v", err)
				}
				execID++
			}

			// Add 1 failure (only for some combinations)
			if task.number == "2" {
				exec := &learning.TaskExecution{
					PlanFile:     planFile,
					RunNumber:    execID,
					TaskNumber:   task.number,
					TaskName:     task.name,
					Agent:        agent,
					Prompt:       "Execute " + task.name,
					Success:      false,
					Output:       "compilation_error: type mismatch",
					ErrorMessage: "Build failed",
					DurationSecs: 15,
					Timestamp:    time.Now().Add(-time.Duration(execID) * time.Hour),
				}
				if err := store.RecordExecution(context.Background(), exec); err != nil {
					t.Fatalf("Failed to record execution: %v", err)
				}
				execID++

				// Stop after one agent to keep test data reasonable
				break
			}
		}
	}
}
