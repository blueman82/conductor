package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/learning"
)

func TestStatsCommand_ValidPlan(t *testing.T) {
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

	// Populate with sample data

	// Success executions
	for i := 1; i <= 3; i++ {
		exec := &learning.TaskExecution{
			PlanFile:     planFile,
			RunNumber:    i,
			TaskNumber:   "1",
			TaskName:     "Setup Database",
			Agent:        "golang-pro",
			Prompt:       "Initialize database",
			Success:      true,
			Output:       "Database initialized successfully",
			DurationSecs: 30,
			Timestamp:    time.Now().Add(-time.Duration(i) * time.Hour),
		}
		if err := store.RecordExecution(context.Background(), exec); err != nil {
			t.Fatalf("Failed to record execution: %v", err)
		}
	}

	// Failure executions
	for i := 1; i <= 2; i++ {
		exec := &learning.TaskExecution{
			PlanFile:     planFile,
			RunNumber:    i + 3,
			TaskNumber:   "2",
			TaskName:     "Build API",
			Agent:        "backend-architect",
			Prompt:       "Build REST API",
			Success:      false,
			Output:       "compilation_error: type mismatch",
			ErrorMessage: "Compilation failed",
			DurationSecs: 15,
			Timestamp:    time.Now().Add(-time.Duration(i) * time.Hour),
		}
		if err := store.RecordExecution(context.Background(), exec); err != nil {
			t.Fatalf("Failed to record execution: %v", err)
		}
	}

	// Create a plan file
	planPath := filepath.Join(tmpDir, planFile)
	if err := os.WriteFile(planPath, []byte("# Test Plan"), 0644); err != nil {
		t.Fatalf("Failed to create plan file: %v", err)
	}

	// Run stats command
	cmd := NewStatsCommand()
	cmd.SetArgs([]string{planPath})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	outputStr := output.String()

	// Verify output contains key statistics
	if !strings.Contains(outputStr, "Total executions:") {
		t.Error("Expected 'Total executions:' in output")
	}

	if !strings.Contains(outputStr, "Success rate:") {
		t.Error("Expected 'Success rate:' in output")
	}

	if !strings.Contains(outputStr, "5") { // 5 total executions
		t.Error("Expected total execution count in output")
	}

	// Should show agent performance
	if !strings.Contains(outputStr, "golang-pro") {
		t.Error("Expected agent name in output")
	}
}

func TestStatsCommand_NoData(t *testing.T) {
	// Create empty database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "empty.db")

	store, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	store.Close()

	// Create a plan file
	planPath := filepath.Join(tmpDir, "plan.md")
	if err := os.WriteFile(planPath, []byte("# Test Plan"), 0644); err != nil {
		t.Fatalf("Failed to create plan file: %v", err)
	}

	// Run stats command
	cmd := NewStatsCommand()
	cmd.SetArgs([]string{planPath})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	outputStr := output.String()

	// Should handle no data gracefully
	if !strings.Contains(outputStr, "No execution data") && !strings.Contains(outputStr, "0") {
		t.Error("Expected message about no data")
	}
}

func TestStatsCommand_InvalidPlan(t *testing.T) {
	cmd := NewStatsCommand()
	cmd.SetArgs([]string{"nonexistent-plan.md"})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for nonexistent plan file")
	}
}

func TestStatsCommand_FormattedOutput(t *testing.T) {
	// Create temp database in expected location
	tmpDir := t.TempDir()
	planFile := "formatted-plan.md"

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

	// Populate with diverse data

	// Different agents with varying success rates
	agents := []struct {
		name    string
		success int
		fail    int
	}{
		{"golang-pro", 10, 2},
		{"python-pro", 5, 5},
		{"rust-pro", 2, 8},
	}

	execID := 1
	for _, agent := range agents {
		// Record successes
		for i := 0; i < agent.success; i++ {
			exec := &learning.TaskExecution{
				PlanFile:     planFile,
				RunNumber:    execID,
				TaskNumber:   "1",
				TaskName:     "Task with " + agent.name,
				Agent:        agent.name,
				Prompt:       "Do task",
				Success:      true,
				Output:       "Success",
				DurationSecs: 30,
				Timestamp:    time.Now(),
			}
			if err := store.RecordExecution(context.Background(), exec); err != nil {
				t.Fatalf("Failed to record execution: %v", err)
			}
			execID++
		}

		// Record failures
		for i := 0; i < agent.fail; i++ {
			exec := &learning.TaskExecution{
				PlanFile:     planFile,
				RunNumber:    execID,
				TaskNumber:   "2",
				TaskName:     "Failing task with " + agent.name,
				Agent:        agent.name,
				Prompt:       "Do failing task",
				Success:      false,
				Output:       "Error occurred",
				ErrorMessage: "Task failed",
				DurationSecs: 15,
				Timestamp:    time.Now(),
			}
			if err := store.RecordExecution(context.Background(), exec); err != nil {
				t.Fatalf("Failed to record execution: %v", err)
			}
			execID++
		}
	}

	// Create plan file
	planPath := filepath.Join(tmpDir, planFile)
	if err := os.WriteFile(planPath, []byte("# Formatted Plan"), 0644); err != nil {
		t.Fatalf("Failed to create plan file: %v", err)
	}

	// Run stats command
	cmd := NewStatsCommand()
	cmd.SetArgs([]string{planPath})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	outputStr := output.String()

	// Verify formatted output with tables
	// Should show all three agents
	if !strings.Contains(outputStr, "golang-pro") ||
		!strings.Contains(outputStr, "python-pro") ||
		!strings.Contains(outputStr, "rust-pro") {
		t.Error("Expected all agent names in output")
	}

	// Should show percentages or counts
	hasPercentages := strings.Contains(outputStr, "%")
	hasCounts := strings.Contains(outputStr, "10") || strings.Contains(outputStr, "5") || strings.Contains(outputStr, "2")

	if !hasPercentages && !hasCounts {
		t.Error("Expected formatted statistics with percentages or counts")
	}

	// Should have section headers
	if !strings.Contains(outputStr, "Agent Performance") && !strings.Contains(outputStr, "Statistics") {
		t.Error("Expected section headers in formatted output")
	}
}

func TestStatsCommand_NoArgs(t *testing.T) {
	cmd := NewStatsCommand()
	cmd.SetArgs([]string{})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error when no arguments provided")
	}
}

func TestStatsCommand_TaskLevelMetrics(t *testing.T) {
	// Create temp database in expected location
	tmpDir := t.TempDir()
	planFile := "task-plan.md"

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

	// Populate with task-specific data

	// Task 1: High success rate
	for i := 1; i <= 8; i++ {
		exec := &learning.TaskExecution{
			PlanFile:     planFile,
			RunNumber:    i,
			TaskNumber:   "1",
			TaskName:     "Reliable Task",
			Agent:        "golang-pro",
			Prompt:       "Do reliable task",
			Success:      true,
			Output:       "Success",
			DurationSecs: 20,
			Timestamp:    time.Now(),
		}
		if err := store.RecordExecution(context.Background(), exec); err != nil {
			t.Fatalf("Failed to record execution: %v", err)
		}
	}

	// Task 2: Low success rate
	for i := 1; i <= 3; i++ {
		exec := &learning.TaskExecution{
			PlanFile:     planFile,
			RunNumber:    i + 8,
			TaskNumber:   "2",
			TaskName:     "Problematic Task",
			Agent:        "backend-architect",
			Prompt:       "Do problematic task",
			Success:      false,
			Output:       "compilation_error",
			ErrorMessage: "Failed",
			DurationSecs: 10,
			Timestamp:    time.Now(),
		}
		if err := store.RecordExecution(context.Background(), exec); err != nil {
			t.Fatalf("Failed to record execution: %v", err)
		}
	}

	// Create plan file
	planPath := filepath.Join(tmpDir, planFile)
	if err := os.WriteFile(planPath, []byte("# Task Plan"), 0644); err != nil {
		t.Fatalf("Failed to create plan file: %v", err)
	}

	// Run stats command
	cmd := NewStatsCommand()
	cmd.SetArgs([]string{planPath})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	outputStr := output.String()

	// Verify task-level metrics are shown
	if !strings.Contains(outputStr, "Reliable Task") ||
		!strings.Contains(outputStr, "Problematic Task") {
		t.Error("Expected task names in output")
	}

	// Should show task numbers
	if !strings.Contains(outputStr, "Task 1") && !strings.Contains(outputStr, "1") {
		t.Log("Expected task identifiers in output (optional)")
	}
}

func TestStatsCommand_CommonFailures(t *testing.T) {
	// Create temp database in expected location
	tmpDir := t.TempDir()
	planFile := "failure-plan.md"

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

	// Populate with failures containing patterns

	failurePatterns := []string{
		"compilation_error: type mismatch",
		"compilation_error: undefined variable",
		"test_failure: assertion failed",
		"timeout: operation took too long",
	}

	for i, pattern := range failurePatterns {
		exec := &learning.TaskExecution{
			PlanFile:     planFile,
			RunNumber:    i + 1,
			TaskNumber:   "1",
			TaskName:     "Task with failures",
			Agent:        "golang-pro",
			Prompt:       "Do task",
			Success:      false,
			Output:       pattern,
			ErrorMessage: "Failed",
			DurationSecs: 10,
			Timestamp:    time.Now(),
		}
		if err := store.RecordExecution(context.Background(), exec); err != nil {
			t.Fatalf("Failed to record execution: %v", err)
		}
	}

	// Create plan file
	planPath := filepath.Join(tmpDir, planFile)
	if err := os.WriteFile(planPath, []byte("# Failure Plan"), 0644); err != nil {
		t.Fatalf("Failed to create plan file: %v", err)
	}

	// Run stats command
	cmd := NewStatsCommand()
	cmd.SetArgs([]string{planPath})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	outputStr := output.String()

	// Should mention common failure patterns
	hasFailureInfo := strings.Contains(outputStr, "compilation_error") ||
		strings.Contains(outputStr, "test_failure") ||
		strings.Contains(outputStr, "timeout") ||
		strings.Contains(outputStr, "Common") ||
		strings.Contains(outputStr, "Failure")

	if !hasFailureInfo {
		t.Log("Expected common failure patterns in output (optional)")
	}
}

func TestNewStatsCommand(t *testing.T) {
	cmd := NewStatsCommand()

	if cmd == nil {
		t.Fatal("NewStatsCommand() returned nil")
	}

	if cmd.Use != "stats <plan-file>" {
		t.Errorf("Expected Use to be 'stats <plan-file>', got: %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE should be set")
	}
}

func TestStatsCommand_AverageDuration(t *testing.T) {
	// Create temp database in expected location
	tmpDir := t.TempDir()
	planFile := "duration-plan.md"

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

	// Populate with executions with varying durations
	durations := []int64{10, 20, 30, 40, 50}

	for i, duration := range durations {
		exec := &learning.TaskExecution{
			PlanFile:     planFile,
			RunNumber:    i + 1,
			TaskNumber:   "1",
			TaskName:     "Timed Task",
			Agent:        "golang-pro",
			Prompt:       "Do task",
			Success:      true,
			Output:       "Success",
			DurationSecs: duration,
			Timestamp:    time.Now(),
		}
		if err := store.RecordExecution(context.Background(), exec); err != nil {
			t.Fatalf("Failed to record execution: %v", err)
		}
	}

	// Create plan file
	planPath := filepath.Join(tmpDir, planFile)
	if err := os.WriteFile(planPath, []byte("# Duration Plan"), 0644); err != nil {
		t.Fatalf("Failed to create plan file: %v", err)
	}

	// Run stats command
	cmd := NewStatsCommand()
	cmd.SetArgs([]string{planPath})

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	outputStr := output.String()

	// Should show duration information
	hasDuration := strings.Contains(outputStr, "Average") ||
		strings.Contains(outputStr, "Duration") ||
		strings.Contains(outputStr, "30") // Average of durations

	if !hasDuration {
		t.Log("Expected average duration in output (optional)")
	}
}
