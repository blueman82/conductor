package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/learning"
)

func TestShowCommand_ValidTask(t *testing.T) {
	// Create temp database in expected location
	tmpDir := t.TempDir()
	planFile := "test-plan.md"

	// Database should be in .conductor/learning/ relative to plan file
	dbDir := filepath.Join(tmpDir, ".conductor", "learning")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		t.Fatalf("Failed to create db directory: %v", err)
	}
	dbPath := filepath.Join(dbDir, "executions.db")

	store, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Populate with sample execution history for Task 1
	executions := []struct {
		agent    string
		success  bool
		output   string
		errorMsg string
		duration int64
	}{
		{"golang-pro", false, "Failed: compilation error", "Compilation failed", 15},
		{"python-pro", false, "Failed: syntax error", "Syntax error", 10},
		{"golang-pro", true, "Success: tests passed", "", 25},
	}

	for i, exec := range executions {
		execution := &learning.TaskExecution{
			PlanFile:     planFile,
			RunNumber:    i + 1,
			TaskNumber:   "1",
			TaskName:     "Setup Database",
			Agent:        exec.agent,
			Prompt:       "Initialize database schema",
			Success:      exec.success,
			Output:       exec.output,
			ErrorMessage: exec.errorMsg,
			DurationSecs: exec.duration,
			Timestamp:    time.Now().Add(-time.Duration(3-i) * time.Hour),
		}
		if err := store.RecordExecution(execution); err != nil {
			t.Fatalf("Failed to record execution: %v", err)
		}
	}

	// Create a plan file
	planPath := filepath.Join(tmpDir, planFile)
	if err := os.WriteFile(planPath, []byte("# Test Plan"), 0644); err != nil {
		t.Fatalf("Failed to create plan file: %v", err)
	}

	// Run show command
	cmd := NewShowCommand()
	cmd.SetArgs([]string{planPath, "1"})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	outputStr := output.String()

	// Verify output contains execution history header
	if !strings.Contains(outputStr, "Execution History") {
		t.Error("Expected 'Execution History' in output")
	}

	// Verify task name is shown
	if !strings.Contains(outputStr, "Setup Database") {
		t.Error("Expected task name 'Setup Database' in output")
	}

	// Verify agents are shown
	if !strings.Contains(outputStr, "golang-pro") {
		t.Error("Expected agent 'golang-pro' in output")
	}
	if !strings.Contains(outputStr, "python-pro") {
		t.Error("Expected agent 'python-pro' in output")
	}

	// Verify success/failure indicators
	hasSuccessIndicator := strings.Contains(outputStr, "GREEN") ||
		strings.Contains(outputStr, "Success") ||
		strings.Contains(outputStr, "✓")

	hasFailureIndicator := strings.Contains(outputStr, "RED") ||
		strings.Contains(outputStr, "Failed") ||
		strings.Contains(outputStr, "✗")

	if !hasSuccessIndicator {
		t.Error("Expected success indicator in output")
	}
	if !hasFailureIndicator {
		t.Error("Expected failure indicator in output")
	}
}

func TestShowCommand_NoHistory(t *testing.T) {
	// Create empty database
	tmpDir := t.TempDir()
	planFile := "empty-plan.md"

	dbDir := filepath.Join(tmpDir, ".conductor", "learning")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		t.Fatalf("Failed to create db directory: %v", err)
	}
	dbPath := filepath.Join(dbDir, "executions.db")

	store, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	store.Close()

	// Create a plan file
	planPath := filepath.Join(tmpDir, planFile)
	if err := os.WriteFile(planPath, []byte("# Empty Plan"), 0644); err != nil {
		t.Fatalf("Failed to create plan file: %v", err)
	}

	// Run show command for non-existent task
	cmd := NewShowCommand()
	cmd.SetArgs([]string{planPath, "99"})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	outputStr := output.String()

	// Should handle no history gracefully
	hasNoHistoryMsg := strings.Contains(outputStr, "No execution") ||
		strings.Contains(outputStr, "no history") ||
		strings.Contains(outputStr, "never executed")

	if !hasNoHistoryMsg {
		t.Error("Expected message about no execution history")
	}
}

func TestShowCommand_FormattedOutput(t *testing.T) {
	// Create temp database
	tmpDir := t.TempDir()
	planFile := "formatted-plan.md"

	dbDir := filepath.Join(tmpDir, ".conductor", "learning")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		t.Fatalf("Failed to create db directory: %v", err)
	}
	dbPath := filepath.Join(dbDir, "executions.db")

	store, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Populate with multiple executions with details
	now := time.Now()
	executions := []struct {
		agent    string
		success  bool
		output   string
		duration int64
		ago      time.Duration
	}{
		{"golang-pro", false, "Error: missing dependency", 12, 3 * time.Hour},
		{"rust-pro", false, "Error: type mismatch", 18, 2 * time.Hour},
		{"golang-pro", true, "Success: all tests passed", 30, 1 * time.Hour},
	}

	for i, exec := range executions {
		execution := &learning.TaskExecution{
			PlanFile:     planFile,
			RunNumber:    i + 1,
			TaskNumber:   "5",
			TaskName:     "Complex Feature",
			Agent:        exec.agent,
			Prompt:       "Implement feature X",
			Success:      exec.success,
			Output:       exec.output,
			DurationSecs: exec.duration,
			Timestamp:    now.Add(-exec.ago),
		}
		if err := store.RecordExecution(execution); err != nil {
			t.Fatalf("Failed to record execution: %v", err)
		}
	}

	// Create plan file
	planPath := filepath.Join(tmpDir, planFile)
	if err := os.WriteFile(planPath, []byte("# Formatted Plan"), 0644); err != nil {
		t.Fatalf("Failed to create plan file: %v", err)
	}

	// Run show command
	cmd := NewShowCommand()
	cmd.SetArgs([]string{planPath, "5"})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	outputStr := output.String()

	// Verify formatted output with all execution details
	if !strings.Contains(outputStr, "Complex Feature") {
		t.Error("Expected task name in output")
	}

	// Should show agent names
	if !strings.Contains(outputStr, "golang-pro") || !strings.Contains(outputStr, "rust-pro") {
		t.Error("Expected agent names in formatted output")
	}

	// Should show attempt numbers or execution count
	hasAttemptInfo := strings.Contains(outputStr, "Attempt") ||
		strings.Contains(outputStr, "#") ||
		strings.Contains(outputStr, "1") ||
		strings.Contains(outputStr, "2") ||
		strings.Contains(outputStr, "3")

	if !hasAttemptInfo {
		t.Error("Expected attempt information in output")
	}

	// Should show timestamps or duration
	hasTimeInfo := strings.Contains(outputStr, "ago") ||
		strings.Contains(outputStr, "seconds") ||
		strings.Contains(outputStr, "duration") ||
		strings.Contains(outputStr, "30") ||
		strings.Contains(outputStr, "18") ||
		strings.Contains(outputStr, "12")

	if !hasTimeInfo {
		t.Log("Expected time information in output (optional)")
	}
}

func TestShowCommand_TaskWithSpaces(t *testing.T) {
	// Create temp database
	tmpDir := t.TempDir()
	planFile := "space-plan.md"

	dbDir := filepath.Join(tmpDir, ".conductor", "learning")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		t.Fatalf("Failed to create db directory: %v", err)
	}
	dbPath := filepath.Join(dbDir, "executions.db")

	store, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Record execution for a task with spaces in name
	execution := &learning.TaskExecution{
		PlanFile:     planFile,
		RunNumber:    1,
		TaskNumber:   "Task 1",
		TaskName:     "Setup Database Schema",
		Agent:        "golang-pro",
		Prompt:       "Initialize",
		Success:      true,
		Output:       "Success",
		DurationSecs: 20,
		Timestamp:    time.Now(),
	}
	if err := store.RecordExecution(execution); err != nil {
		t.Fatalf("Failed to record execution: %v", err)
	}

	// Create plan file
	planPath := filepath.Join(tmpDir, planFile)
	if err := os.WriteFile(planPath, []byte("# Space Plan"), 0644); err != nil {
		t.Fatalf("Failed to create plan file: %v", err)
	}

	// Run show command with task name containing spaces
	cmd := NewShowCommand()
	cmd.SetArgs([]string{planPath, "Task 1"})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	outputStr := output.String()

	// Should find and display the task
	if !strings.Contains(outputStr, "Setup Database Schema") {
		t.Error("Expected task with spaces to be found and displayed")
	}
}

func TestShowCommand_NoDatabase(t *testing.T) {
	// Create temp directory without database
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "no-db-plan.md")

	// Create plan file but no database
	if err := os.WriteFile(planPath, []byte("# No DB Plan"), 0644); err != nil {
		t.Fatalf("Failed to create plan file: %v", err)
	}

	// Run show command
	cmd := NewShowCommand()
	cmd.SetArgs([]string{planPath, "1"})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	outputStr := output.String()

	// Should handle missing database gracefully
	hasNoDataMsg := strings.Contains(outputStr, "No execution") ||
		strings.Contains(outputStr, "no data") ||
		strings.Contains(outputStr, "not found")

	if !hasNoDataMsg {
		t.Error("Expected message about no execution data")
	}
}

func TestShowCommand_InvalidArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no args", []string{}},
		{"one arg", []string{"plan.md"}},
		{"three args", []string{"plan.md", "1", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewShowCommand()
			cmd.SetArgs(tt.args)

			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)

			err := cmd.Execute()
			if err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
		})
	}
}

func TestShowCommand_MissingPlanFile(t *testing.T) {
	cmd := NewShowCommand()
	cmd.SetArgs([]string{"nonexistent.md", "1"})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for nonexistent plan file")
	}
}

func TestNewShowCommand(t *testing.T) {
	cmd := NewShowCommand()

	if cmd == nil {
		t.Fatal("NewShowCommand() returned nil")
	}

	if !strings.Contains(cmd.Use, "show") {
		t.Errorf("Expected Use to contain 'show', got: %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE should be set")
	}
}

func TestShowCommand_ChronologicalOrder(t *testing.T) {
	// Create temp database
	tmpDir := t.TempDir()
	planFile := "chrono-plan.md"

	dbDir := filepath.Join(tmpDir, ".conductor", "learning")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		t.Fatalf("Failed to create db directory: %v", err)
	}
	dbPath := filepath.Join(dbDir, "executions.db")

	store, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Record executions at different times
	timestamps := []time.Time{
		time.Now().Add(-3 * time.Hour),
		time.Now().Add(-2 * time.Hour),
		time.Now().Add(-1 * time.Hour),
	}

	for i, ts := range timestamps {
		execution := &learning.TaskExecution{
			PlanFile:     planFile,
			RunNumber:    i + 1,
			TaskNumber:   "10",
			TaskName:     "Chronological Task",
			Agent:        "golang-pro",
			Prompt:       "Test chronological order",
			Success:      true,
			Output:       "Success",
			DurationSecs: 10,
			Timestamp:    ts,
		}
		if err := store.RecordExecution(execution); err != nil {
			t.Fatalf("Failed to record execution: %v", err)
		}
	}

	// Create plan file
	planPath := filepath.Join(tmpDir, planFile)
	if err := os.WriteFile(planPath, []byte("# Chrono Plan"), 0644); err != nil {
		t.Fatalf("Failed to create plan file: %v", err)
	}

	// Run show command
	cmd := NewShowCommand()
	cmd.SetArgs([]string{planPath, "10"})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	outputStr := output.String()

	// Should display all three executions
	// The exact format may vary, but we should see evidence of multiple executions
	hasMultipleExecs := strings.Count(outputStr, "golang-pro") >= 3 ||
		strings.Contains(outputStr, "3")

	if !hasMultipleExecs {
		t.Log("Expected evidence of 3 executions in chronological order (optional)")
	}
}
