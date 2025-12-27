package executor

import (
	"context"
	"os"
	"testing"

	"github.com/harrison/conductor/internal/learning"
	"github.com/harrison/conductor/internal/models"
)

func TestLIPCollectorHook_NilGracefulDegradation(t *testing.T) {
	// Test that nil hook doesn't panic
	var hook *LIPCollectorHook
	ctx := context.Background()

	// All methods should handle nil gracefully
	err := hook.RecordTestResult(ctx, 1, "1", true, "test passed")
	if err != nil {
		t.Errorf("RecordTestResult should not error on nil hook: %v", err)
	}

	err = hook.RecordBuildResult(ctx, 1, "1", true, "build success")
	if err != nil {
		t.Errorf("RecordBuildResult should not error on nil hook: %v", err)
	}

	err = hook.RecordTestResults(ctx, 1, "1", []TestCommandResult{
		{Command: "go test", Passed: true},
	})
	if err != nil {
		t.Errorf("RecordTestResults should not error on nil hook: %v", err)
	}

	err = hook.RecordTaskFileRelation(ctx, "task:1", "/path/to/file.go", 1.0)
	if err != nil {
		t.Errorf("RecordTaskFileRelation should not error on nil hook: %v", err)
	}

	err = hook.RecordTaskAgentRelation(ctx, "task:1", "agent-x", true, 1.0)
	if err != nil {
		t.Errorf("RecordTaskAgentRelation should not error on nil hook: %v", err)
	}

	// PostTaskHook should not panic
	hook.PostTaskHook(ctx, models.Task{Number: "1"}, nil, true)

	// GetProgressScore should return zero values
	score, err := hook.GetProgressScore(ctx, 1)
	if err != nil {
		t.Errorf("GetProgressScore should not error on nil hook: %v", err)
	}
	if score != learning.ProgressNone {
		t.Errorf("GetProgressScore expected ProgressNone, got %v", score)
	}
}

func TestNewLIPCollectorHook_NilStore(t *testing.T) {
	hook := NewLIPCollectorHook(nil, nil)
	if hook != nil {
		t.Error("NewLIPCollectorHook should return nil for nil store")
	}
}

func TestLIPCollectorHook_RecordTestResult(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "lip_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Initialize store with migrations
	store, err := learning.NewStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.ApplyMigrations(ctx); err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	// Create a task execution record first
	exec := &learning.TaskExecution{
		PlanFile:   "test.yaml",
		RunNumber:  1,
		TaskNumber: "1",
		TaskName:   "Test Task",
		Success:    false,
		Output:     "testing",
	}
	if err := store.RecordExecution(ctx, exec); err != nil {
		t.Fatalf("Failed to record execution: %v", err)
	}
	taskExecID := exec.ID

	// Create hook
	hook := NewLIPCollectorHook(store, nil)
	if hook == nil {
		t.Fatal("NewLIPCollectorHook returned nil for valid store")
	}

	// Record test results
	err = hook.RecordTestResult(ctx, taskExecID, "1", true, "test passed")
	if err != nil {
		t.Errorf("RecordTestResult failed: %v", err)
	}

	err = hook.RecordTestResult(ctx, taskExecID, "1", false, "test failed with assertion error")
	if err != nil {
		t.Errorf("RecordTestResult failed: %v", err)
	}

	// Verify events were recorded by querying
	events, err := store.GetEvents(ctx, &learning.LIPFilter{
		TaskExecutionID: taskExecID,
	})
	if err != nil {
		t.Fatalf("Failed to get events: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 LIP events, got %d", len(events))
	}

	// Check event types
	hasPass := false
	hasFail := false
	for _, e := range events {
		if e.EventType == learning.LIPEventTestPass {
			hasPass = true
		}
		if e.EventType == learning.LIPEventTestFail {
			hasFail = true
		}
	}
	if !hasPass {
		t.Error("Expected test_pass event")
	}
	if !hasFail {
		t.Error("Expected test_fail event")
	}
}

func TestLIPCollectorHook_RecordBuildResult(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "lip_build_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := learning.NewStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.ApplyMigrations(ctx); err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	exec := &learning.TaskExecution{
		PlanFile:   "test.yaml",
		RunNumber:  1,
		TaskNumber: "2",
		TaskName:   "Build Task",
		Success:    false,
		Output:     "building",
	}
	if err := store.RecordExecution(ctx, exec); err != nil {
		t.Fatalf("Failed to record execution: %v", err)
	}
	taskExecID := exec.ID

	hook := NewLIPCollectorHook(store, nil)

	// Record build results
	err = hook.RecordBuildResult(ctx, taskExecID, "2", true, "build succeeded")
	if err != nil {
		t.Errorf("RecordBuildResult failed: %v", err)
	}

	err = hook.RecordBuildResult(ctx, taskExecID, "2", false, "compilation error: undefined reference")
	if err != nil {
		t.Errorf("RecordBuildResult failed: %v", err)
	}

	events, err := store.GetEvents(ctx, &learning.LIPFilter{
		TaskExecutionID: taskExecID,
	})
	if err != nil {
		t.Fatalf("Failed to get events: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 LIP events, got %d", len(events))
	}
}

func TestLIPCollectorHook_RecordTestResults(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "lip_results_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := learning.NewStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.ApplyMigrations(ctx); err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	exec := &learning.TaskExecution{
		PlanFile:   "test.yaml",
		RunNumber:  1,
		TaskNumber: "3",
		TaskName:   "Multi-Test Task",
		Success:    false,
		Output:     "testing",
	}
	if err := store.RecordExecution(ctx, exec); err != nil {
		t.Fatalf("Failed to record execution: %v", err)
	}
	taskExecID := exec.ID

	hook := NewLIPCollectorHook(store, nil)

	// Record multiple test results at once
	testResults := []TestCommandResult{
		{Command: "go test ./pkg/...", Passed: true, Output: "ok  pkg  0.5s"},
		{Command: "go test ./internal/...", Passed: false, Output: "FAIL"},
		{Command: "make lint", Passed: true, Output: "no issues"},
	}

	err = hook.RecordTestResults(ctx, taskExecID, "3", testResults)
	if err != nil {
		t.Errorf("RecordTestResults failed: %v", err)
	}

	events, err := store.GetEvents(ctx, &learning.LIPFilter{
		TaskExecutionID: taskExecID,
	})
	if err != nil {
		t.Fatalf("Failed to get events: %v", err)
	}

	if len(events) != 3 {
		t.Errorf("Expected 3 LIP events, got %d", len(events))
	}

	// Count pass/fail
	passCount := 0
	failCount := 0
	for _, e := range events {
		switch e.EventType {
		case learning.LIPEventTestPass:
			passCount++
		case learning.LIPEventTestFail:
			failCount++
		}
	}
	if passCount != 2 {
		t.Errorf("Expected 2 pass events, got %d", passCount)
	}
	if failCount != 1 {
		t.Errorf("Expected 1 fail event, got %d", failCount)
	}
}

func TestLIPCollectorHook_KnowledgeGraph(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "lip_kg_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := learning.NewStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.ApplyMigrations(ctx); err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	hook := NewLIPCollectorHook(store, nil)

	// Record task-agent relationship
	err = hook.RecordTaskAgentRelation(ctx, "task:1", "backend-developer", true, 1.0)
	if err != nil {
		t.Errorf("RecordTaskAgentRelation failed: %v", err)
	}

	err = hook.RecordTaskAgentRelation(ctx, "task:2", "backend-developer", false, 0.5)
	if err != nil {
		t.Errorf("RecordTaskAgentRelation failed: %v", err)
	}

	// Record file relationships
	err = hook.RecordTaskFileRelation(ctx, "task:1", "/pkg/handler.go", 1.0)
	if err != nil {
		t.Errorf("RecordTaskFileRelation failed: %v", err)
	}

	// Verify nodes exist
	kg := store.NewKnowledgeGraph()

	taskNode, err := kg.GetNode(ctx, "task:1")
	if err != nil {
		t.Errorf("GetNode failed: %v", err)
	}
	if taskNode == nil {
		t.Error("Task node should exist")
	}

	agentNode, err := kg.GetNode(ctx, "backend-developer")
	if err != nil {
		t.Errorf("GetNode failed: %v", err)
	}
	if agentNode == nil {
		t.Error("Agent node should exist")
	}

	// Verify edges
	edges, err := kg.GetEdges(ctx, "task:1", nil)
	if err != nil {
		t.Errorf("GetEdges failed: %v", err)
	}
	if len(edges) < 2 {
		t.Errorf("Expected at least 2 edges from task:1, got %d", len(edges))
	}
}

func TestLIPCollectorHook_PostTaskHook(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "lip_posthook_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := learning.NewStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.ApplyMigrations(ctx); err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	hook := NewLIPCollectorHook(store, nil)

	// Test successful task
	task := models.Task{
		Number: "4",
		Name:   "Test Post Hook",
		Agent:  "frontend-developer",
	}
	result := &models.TaskResult{
		Output: "completed",
	}

	// Should not panic
	hook.PostTaskHook(ctx, task, result, true)

	// Verify knowledge graph edges were created
	kg := store.NewKnowledgeGraph()
	taskNode, _ := kg.GetNode(ctx, "task:4")
	if taskNode == nil {
		t.Error("Task node should be created by PostTaskHook")
	}

	edges, err := kg.GetEdges(ctx, "task:4", []learning.EdgeType{learning.EdgeTypeSucceededWith})
	if err != nil {
		t.Errorf("GetEdges failed: %v", err)
	}
	if len(edges) != 1 {
		t.Errorf("Expected 1 succeeded_with edge, got %d", len(edges))
	}

	// Test failed task
	task2 := models.Task{
		Number: "5",
		Name:   "Test Post Hook Failed",
		Agent:  "backend-developer",
	}
	hook.PostTaskHook(ctx, task2, nil, false)

	edges2, _ := kg.GetEdges(ctx, "task:5", []learning.EdgeType{learning.EdgeTypeUsedBy})
	if len(edges2) != 1 {
		t.Errorf("Expected 1 used_by edge for failed task, got %d", len(edges2))
	}

	// Test with empty agent (should use "default")
	task3 := models.Task{
		Number: "6",
		Name:   "Test Post Hook No Agent",
		Agent:  "", // Empty agent
	}
	hook.PostTaskHook(ctx, task3, nil, true)

	defaultNode, _ := kg.GetNode(ctx, "default")
	if defaultNode == nil {
		t.Error("Default agent node should be created for empty agent")
	}
}

func TestLIPCollectorHook_GetProgressScore(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "lip_progress_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := learning.NewStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.ApplyMigrations(ctx); err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	// Create a task execution
	exec := &learning.TaskExecution{
		PlanFile:   "test.yaml",
		RunNumber:  1,
		TaskNumber: "7",
		TaskName:   "Progress Task",
		Success:    true,
		Output:     "done",
	}
	if err := store.RecordExecution(ctx, exec); err != nil {
		t.Fatalf("Failed to record execution: %v", err)
	}
	taskExecID := exec.ID

	hook := NewLIPCollectorHook(store, nil)

	// Record test pass (adds to progress score)
	_ = hook.RecordTestResult(ctx, taskExecID, "7", true, "test passed")
	_ = hook.RecordBuildResult(ctx, taskExecID, "7", true, "build success")

	// Get progress score
	score, err := hook.GetProgressScore(ctx, taskExecID)
	if err != nil {
		t.Errorf("GetProgressScore failed: %v", err)
	}

	// Score should be > 0 since we have test+build success events
	if score <= learning.ProgressNone {
		t.Errorf("Expected progress score > 0, got %v", score)
	}
}
