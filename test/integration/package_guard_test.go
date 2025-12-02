package integration

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/executor"
	"github.com/harrison/conductor/internal/models"
)

// =============================================================================
// Package Guard Integration Tests
// =============================================================================

// TestPackageGuard_WaveBuilderRejectsConflict tests that CalculateWaves rejects
// plans with package conflicts in the same wave.
func TestPackageGuard_WaveBuilderRejectsConflict(t *testing.T) {
	// Two tasks touching same package without serialization
	tasks := []models.Task{
		{
			Number: "1",
			Name:   "Modify executor task.go",
			Files:  []string{"internal/executor/task.go", "internal/executor/task_test.go"},
			Prompt: "Implement task.go changes",
		},
		{
			Number: "2",
			Name:   "Modify executor wave.go",
			Files:  []string{"internal/executor/wave.go", "internal/executor/wave_test.go"},
			Prompt: "Implement wave.go changes",
		},
	}

	// CalculateWaves should reject due to package conflict
	_, err := executor.CalculateWaves(tasks)
	if err == nil {
		t.Fatal("expected error for package conflict, got nil")
	}

	// Error should mention the conflicting package
	if !containsSubstring(err.Error(), "internal/executor") {
		t.Errorf("error should mention 'internal/executor', got: %v", err)
	}
}

// TestPackageGuard_WaveBuilderAcceptsSerializedTasks tests that CalculateWaves accepts
// plans where conflicting packages are serialized via depends_on.
func TestPackageGuard_WaveBuilderAcceptsSerializedTasks(t *testing.T) {
	// Two tasks touching same package WITH serialization
	tasks := []models.Task{
		{
			Number: "1",
			Name:   "Modify executor task.go",
			Files:  []string{"internal/executor/task.go"},
			Prompt: "Implement task.go changes",
		},
		{
			Number:    "2",
			Name:      "Modify executor wave.go",
			DependsOn: []string{"1"}, // Serializes execution
			Files:     []string{"internal/executor/wave.go"},
			Prompt:    "Implement wave.go changes",
		},
	}

	// Should succeed
	waves, err := executor.CalculateWaves(tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should produce 2 waves (one task per wave)
	if len(waves) != 2 {
		t.Errorf("expected 2 waves, got %d", len(waves))
	}
}

// TestPackageGuard_RuntimeEnforcementBlocks tests that runtime enforcement
// prevents concurrent package modifications.
func TestPackageGuard_RuntimeEnforcementBlocks(t *testing.T) {
	guard := executor.NewPackageGuard()
	ctx := context.Background()

	// Task 1 acquires internal/executor
	release1, err := guard.Acquire(ctx, "1", []string{"internal/executor"})
	if err != nil {
		t.Fatalf("Task 1 acquire failed: %v", err)
	}

	// Task 2 should be blocked (TryAcquire returns false)
	acquired, _ := guard.TryAcquire("2", []string{"internal/executor"})
	if acquired {
		t.Error("Task 2 should be blocked from acquiring internal/executor")
	}

	// Release task 1
	release1()

	// Now task 2 should succeed
	release2, err := guard.Acquire(ctx, "2", []string{"internal/executor"})
	if err != nil {
		t.Fatalf("Task 2 acquire after release failed: %v", err)
	}
	release2()
}

// TestPackageGuard_DifferentPackagesParallel tests that tasks modifying
// different packages can run in parallel.
func TestPackageGuard_DifferentPackagesParallel(t *testing.T) {
	guard := executor.NewPackageGuard()
	ctx := context.Background()

	var wg sync.WaitGroup
	var errCount int32

	// Task 1: internal/executor
	wg.Add(1)
	go func() {
		defer wg.Done()
		release, err := guard.Acquire(ctx, "1", []string{"internal/executor"})
		if err != nil {
			atomic.AddInt32(&errCount, 1)
			return
		}
		time.Sleep(10 * time.Millisecond) // Simulate work
		release()
	}()

	// Task 2: internal/parser (different package - should not block)
	wg.Add(1)
	go func() {
		defer wg.Done()
		release, err := guard.Acquire(ctx, "2", []string{"internal/parser"})
		if err != nil {
			atomic.AddInt32(&errCount, 1)
			return
		}
		time.Sleep(10 * time.Millisecond) // Simulate work
		release()
	}()

	wg.Wait()

	if errCount > 0 {
		t.Errorf("expected no errors, got %d", errCount)
	}
}

// TestPackageGuard_ContextCancellation tests that context cancellation
// properly aborts waiting for package locks.
func TestPackageGuard_ContextCancellation(t *testing.T) {
	guard := executor.NewPackageGuard()
	ctx := context.Background()

	// Task 1 holds the lock
	release1, _ := guard.Acquire(ctx, "1", []string{"internal/executor"})
	defer release1()

	// Task 2 tries with cancelled context
	ctx2, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := guard.Acquire(ctx2, "2", []string{"internal/executor"})
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

// TestPackageGuard_NonGoFilesSkipped tests that non-Go files are not
// subject to package conflict detection.
func TestPackageGuard_NonGoFilesSkipped(t *testing.T) {
	tasks := []models.Task{
		{
			Number: "1",
			Name:   "Update README",
			Files:  []string{"docs/README.md", "docs/guide.md"},
			Prompt: "Update documentation",
		},
		{
			Number: "2",
			Name:   "Update API docs",
			Files:  []string{"docs/api.md"},
			Prompt: "Update API documentation",
		},
	}

	// Should succeed - non-Go files don't trigger package guard
	waves, err := executor.CalculateWaves(tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both tasks should be in same wave
	if len(waves) != 1 {
		t.Errorf("expected 1 wave for non-Go files, got %d", len(waves))
	}
}

// TestPackageGuard_MixedGoAndNonGoFiles tests handling of tasks with
// both Go and non-Go files.
func TestPackageGuard_MixedGoAndNonGoFiles(t *testing.T) {
	tasks := []models.Task{
		{
			Number: "1",
			Name:   "Task 1",
			Files:  []string{"internal/executor/task.go", "README.md"},
			Prompt: "Implement task with readme",
		},
		{
			Number: "2",
			Name:   "Task 2",
			Files:  []string{"internal/executor/wave.go", "CHANGELOG.md"},
			Prompt: "Implement wave with changelog",
		},
	}

	// Should fail - both tasks touch internal/executor package
	_, err := executor.CalculateWaves(tasks)
	if err == nil {
		t.Fatal("expected error for package conflict")
	}
}

// TestPackageGuard_SessionLockManager tests the session-level lock manager.
func TestPackageGuard_SessionLockManager(t *testing.T) {
	sm := executor.NewSessionPackageLockManager()

	// Task 1 locks packages
	acquired, release1 := sm.TryLock("1", []string{"internal/executor", "internal/parser"})
	if !acquired {
		t.Fatal("Task 1 should acquire locks")
	}

	// Verify packages are held
	if !sm.IsPackageHeld("internal/executor") {
		t.Error("internal/executor should be held")
	}
	if sm.GetPackageHolder("internal/executor") != "1" {
		t.Error("internal/executor should be held by task 1")
	}

	// Task 2 cannot acquire overlapping package
	acquired2, _ := sm.TryLock("2", []string{"internal/executor"})
	if acquired2 {
		t.Error("Task 2 should not acquire internal/executor")
	}

	// Task 2 can acquire non-overlapping package
	acquired3, release3 := sm.TryLock("2", []string{"internal/config"})
	if !acquired3 {
		t.Error("Task 2 should acquire internal/config")
	}
	release3()

	// Release task 1
	release1()

	// Now task 2 should succeed
	acquired4, release4 := sm.TryLock("2", []string{"internal/executor"})
	if !acquired4 {
		t.Error("Task 2 should acquire internal/executor after release")
	}
	release4()
}

// TestPackageGuard_TransitiveDependencyResolution tests that transitive
// dependencies properly serialize package access.
func TestPackageGuard_TransitiveDependencyResolution(t *testing.T) {
	// Task 1 -> Task 2 -> Task 3, all touching same package
	tasks := []models.Task{
		{
			Number: "1",
			Name:   "Task 1",
			Files:  []string{"internal/executor/a.go"},
			Prompt: "First change",
		},
		{
			Number:    "2",
			Name:      "Task 2",
			DependsOn: []string{"1"},
			Files:     []string{"internal/executor/b.go"},
			Prompt:    "Second change",
		},
		{
			Number:    "3",
			Name:      "Task 3",
			DependsOn: []string{"2"},
			Files:     []string{"internal/executor/c.go"},
			Prompt:    "Third change",
		},
	}

	// Should succeed - transitive dependency serializes all tasks
	waves, err := executor.CalculateWaves(tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should produce 3 waves
	if len(waves) != 3 {
		t.Errorf("expected 3 waves, got %d", len(waves))
	}
}

// TestPackageGuard_EnforcePackageIsolation tests that EnforcePackageIsolation
// correctly validates git diff against declared files.
func TestPackageGuard_EnforcePackageIsolation(t *testing.T) {
	// This test uses a mock runner since we can't control actual git state
	runner := &mockCommandRunner{
		outputs: map[string]string{
			"git diff --name-only HEAD":      "internal/executor/task.go\ninternal/executor/wave.go\n",
			"git diff --name-only --cached":  "",
		},
	}

	task := models.Task{
		Number: "1",
		Name:   "Modify executor",
		Files:  []string{"internal/executor/task.go", "internal/executor/wave.go"},
	}

	result, err := executor.EnforcePackageIsolation(context.Background(), runner, task)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected Passed to be true, got false with undeclared: %v", result.UndeclaredFiles)
	}
}

// TestPackageGuard_EnforcePackageIsolation_Violation tests that violations
// are detected and actionable remediation is provided.
func TestPackageGuard_EnforcePackageIsolation_Violation(t *testing.T) {
	runner := &mockCommandRunner{
		outputs: map[string]string{
			"git diff --name-only HEAD":      "internal/executor/task.go\ninternal/parser/yaml.go\n",
			"git diff --name-only --cached":  "",
		},
	}

	task := models.Task{
		Number: "1",
		Name:   "Modify executor only",
		Files:  []string{"internal/executor/task.go"},
	}

	result, err := executor.EnforcePackageIsolation(context.Background(), runner, task)
	if err == nil {
		t.Fatal("expected error for isolation violation")
	}
	if result.Passed {
		t.Error("expected Passed to be false")
	}
	if len(result.UndeclaredFiles) != 1 {
		t.Errorf("expected 1 undeclared file, got: %v", result.UndeclaredFiles)
	}
	if result.Remediation == "" {
		t.Error("expected non-empty remediation message")
	}
}

// TestPackageGuard_EnforcePackageIsolation_SamePackage tests that modifying
// other files in the same Go package is allowed.
func TestPackageGuard_EnforcePackageIsolation_SamePackage(t *testing.T) {
	runner := &mockCommandRunner{
		outputs: map[string]string{
			"git diff --name-only HEAD":      "internal/executor/task.go\ninternal/executor/qc.go\n",
			"git diff --name-only --cached":  "",
		},
	}

	task := models.Task{
		Number: "1",
		Name:   "Modify executor package",
		Files:  []string{"internal/executor/task.go"}, // Only task.go declared
	}

	result, err := executor.EnforcePackageIsolation(context.Background(), runner, task)
	if err != nil {
		t.Fatalf("expected no error for same package, got: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected Passed for same package modifications, got undeclared: %v", result.UndeclaredFiles)
	}
}

// mockCommandRunner implements executor.CommandRunner for testing
type mockCommandRunner struct {
	outputs map[string]string
}

func (m *mockCommandRunner) Run(ctx context.Context, command string) (string, error) {
	if output, ok := m.outputs[command]; ok {
		return output, nil
	}
	return "", nil
}

// =============================================================================
// Helpers
// =============================================================================

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
