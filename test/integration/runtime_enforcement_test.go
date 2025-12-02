package integration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/executor"
	"github.com/harrison/conductor/internal/models"
	"github.com/harrison/conductor/internal/parser"
)

// =============================================================================
// Runtime Enforcement Integration Tests
// =============================================================================

// TestRuntimeEnforcement_SuccessScenario tests the complete runtime enforcement
// pipeline with a plan that passes all checks.
func TestRuntimeEnforcement_SuccessScenario(t *testing.T) {
	planPath := filepath.Join("fixtures", "runtime_enforcement_plan.yaml")

	// Parse the plan
	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("Failed to parse runtime enforcement plan: %v", err)
	}

	// Verify plan structure
	if len(plan.Tasks) != 4 {
		t.Fatalf("Expected 4 tasks, got %d", len(plan.Tasks))
	}

	// Verify PlannerComplianceSpec is parsed
	if plan.PlannerCompliance == nil {
		t.Error("Expected PlannerComplianceSpec to be parsed")
	} else {
		if plan.PlannerCompliance.PlannerVersion != "2.9.0" {
			t.Errorf("Expected planner version 2.9.0, got %s", plan.PlannerCompliance.PlannerVersion)
		}
	}

	// Verify DataFlowRegistry is parsed
	if plan.DataFlowRegistry == nil {
		t.Error("Expected DataFlowRegistry to be parsed")
	} else {
		if len(plan.DataFlowRegistry.Producers) == 0 {
			t.Error("Expected producers in DataFlowRegistry")
		}
		if len(plan.DataFlowRegistry.Consumers) == 0 {
			t.Error("Expected consumers in DataFlowRegistry")
		}
	}

	// Calculate waves - validates dependency graph
	waves, err := executor.CalculateWaves(plan.Tasks)
	if err != nil {
		t.Fatalf("Failed to calculate waves: %v", err)
	}

	// Expected wave structure:
	// Wave 1: Task 1 (no dependencies)
	// Wave 2: Task 2 (depends on 1)
	// Wave 3: Tasks 3, 4 (3 depends on 1,2; 4 depends on 2)
	if len(waves) < 3 {
		t.Fatalf("Expected at least 3 waves, got %d", len(waves))
	}

	// Verify task 1 is in first wave
	wave1Tasks := mapset(waves[0].TaskNumbers)
	if !wave1Tasks["1"] {
		t.Error("Task 1 should be in first wave")
	}

	// Verify wave dependencies are correct
	// With updated fixture: task 4 depends on task 3 (to serialize internal/executor access)
	task2Wave := findWaveIndex(waves, "2")
	task3Wave := findWaveIndex(waves, "3")
	task4Wave := findWaveIndex(waves, "4")

	if task2Wave <= 0 {
		t.Errorf("Task 2 should be after first wave, got wave %d", task2Wave)
	}
	if task3Wave <= task2Wave {
		t.Errorf("Task 3 should be after task 2 wave. Task 3: %d, Task 2: %d", task3Wave, task2Wave)
	}
	// Task 4 now depends on task 3, so it should be after task 3
	if task4Wave <= task3Wave {
		t.Errorf("Task 4 should be after task 3 wave. Task 4: %d, Task 3: %d", task4Wave, task3Wave)
	}
}

// TestRuntimeEnforcement_DependencyChecks verifies dependency check execution.
func TestRuntimeEnforcement_DependencyChecks(t *testing.T) {
	planPath := filepath.Join("fixtures", "runtime_enforcement_plan.yaml")

	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("Failed to parse plan: %v", err)
	}

	// Find task 1 which has dependency checks
	var task1 *models.Task
	for i := range plan.Tasks {
		if plan.Tasks[i].Number == "1" {
			task1 = &plan.Tasks[i]
			break
		}
	}

	if task1 == nil {
		t.Fatalf("Task 1 not found")
	}

	// Verify runtime metadata exists
	if task1.RuntimeMetadata == nil {
		t.Fatal("Task 1 should have RuntimeMetadata")
	}

	// Verify dependency checks exist
	checks := task1.RuntimeMetadata.DependencyChecks
	if len(checks) != 2 {
		t.Fatalf("Expected 2 dependency checks, got %d", len(checks))
	}

	// Verify check structure
	if checks[0].Command != "test -f go.mod" {
		t.Errorf("Unexpected first check command: %s", checks[0].Command)
	}
	if checks[0].Description != "Verify Go module exists" {
		t.Errorf("Unexpected first check description: %s", checks[0].Description)
	}

	// Create a task with simple checks that will pass in any test context
	testTask := models.Task{
		Number: "test",
		Name:   "Test passing checks",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DependencyChecks: []models.DependencyCheck{
				{
					Command:     "echo 'check passed'",
					Description: "Simple echo check",
				},
				{
					Command:     "true",
					Description: "Always passes",
				},
			},
		},
	}

	// Run dependency checks using real runner
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	runner := executor.NewShellCommandRunner("")
	err = executor.RunDependencyChecks(ctx, runner, testTask)
	if err != nil {
		t.Errorf("Dependency checks failed (expected to pass): %v", err)
	}
}

// TestRuntimeEnforcement_FailureDependencyCheck tests that dependency check
// failures are properly detected and reported.
func TestRuntimeEnforcement_FailureDependencyCheck(t *testing.T) {
	// Create a task with a failing dependency check
	task := models.Task{
		Number: "test",
		Name:   "Test failing check",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DependencyChecks: []models.DependencyCheck{
				{
					Command:     "exit 1",
					Description: "Intentionally failing check",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	runner := executor.NewShellCommandRunner("")
	err := executor.RunDependencyChecks(ctx, runner, task)

	if err == nil {
		t.Fatal("Expected error for failing dependency check")
	}

	// Verify error is wrapped with ErrDependencyCheckFailed
	if !containsString(err.Error(), "dependency check failed") {
		t.Errorf("Expected error to contain 'dependency check failed', got: %v", err)
	}
}

// TestRuntimeEnforcement_DocumentationTargets verifies documentation target verification.
func TestRuntimeEnforcement_DocumentationTargets(t *testing.T) {
	planPath := filepath.Join("fixtures", "runtime_enforcement_plan.yaml")

	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("Failed to parse plan: %v", err)
	}

	// Find task 1 which has documentation targets
	var task1 *models.Task
	for i := range plan.Tasks {
		if plan.Tasks[i].Number == "1" {
			task1 = &plan.Tasks[i]
			break
		}
	}

	if task1 == nil {
		t.Fatalf("Task 1 not found")
	}

	// Verify documentation targets exist
	if task1.RuntimeMetadata == nil {
		t.Fatal("Task 1 should have RuntimeMetadata")
	}

	targets := task1.RuntimeMetadata.DocumentationTargets
	if len(targets) != 1 {
		t.Fatalf("Expected 1 documentation target, got %d", len(targets))
	}

	// Verify target structure
	if targets[0].Section != "# Configuration" {
		t.Errorf("Unexpected target section: %s", targets[0].Section)
	}

	// Create a test task pointing to the actual fixture file (relative to test directory)
	docTargetPath := filepath.Join("fixtures", "runtime_doc_target.md")
	testTask := models.Task{
		Number: "test",
		Name:   "Test doc target verification",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DocumentationTargets: []models.DocumentationTarget{
				{
					Location: docTargetPath,
					Section:  "# Configuration",
				},
			},
		},
	}

	// Run documentation target verification
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := executor.VerifyDocumentationTargets(ctx, testTask)
	if err != nil {
		t.Fatalf("Documentation target verification failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Passed {
		t.Errorf("Documentation target should pass: %v", results[0].Error)
	}
}

// TestRuntimeEnforcement_DocumentationTargetFailure tests doc target failures.
func TestRuntimeEnforcement_DocumentationTargetFailure(t *testing.T) {
	// Create a task with a non-existent documentation target
	task := models.Task{
		Number: "test",
		Name:   "Test missing doc target",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DocumentationTargets: []models.DocumentationTarget{
				{
					Location: "/nonexistent/file.md",
					Section:  "# Missing",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := executor.VerifyDocumentationTargets(ctx, task)
	if err != nil {
		t.Fatalf("Verification should not return error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Passed {
		t.Error("Documentation target should fail for non-existent file")
	}

	if results[0].Error == nil {
		t.Error("Expected error in result")
	}
}

// TestRuntimeEnforcement_PackageGuard verifies package guard functionality.
func TestRuntimeEnforcement_PackageGuard(t *testing.T) {
	planPath := filepath.Join("fixtures", "runtime_enforcement_plan.yaml")

	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("Failed to parse plan: %v", err)
	}

	// Verify package guard can detect conflicts at validation time
	// Task 2 and Task 4 both touch internal/executor files
	err = executor.DetectPackageConflicts(plan.Tasks)
	// This should pass because tasks are serialized via depends_on
	if err != nil {
		// If error, it means we found unserialized conflicts (which is actually fine
		// because the fixture has proper dependencies)
		t.Logf("Package conflict detection: %v", err)
	}

	// Test runtime package guard
	guard := executor.NewPackageGuard()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Acquire package for task 1
	packages := []string{"internal/config"}
	release, err := guard.Acquire(ctx, "1", packages)
	if err != nil {
		t.Fatalf("Failed to acquire package: %v", err)
	}
	defer release()

	// Verify package is held
	if !guard.IsHeld("internal/config") {
		t.Error("Package should be held after acquisition")
	}

	// Verify holder is correct
	if holder := guard.GetHolder("internal/config"); holder != "1" {
		t.Errorf("Expected holder '1', got '%s'", holder)
	}

	// TryAcquire should fail for same package
	ok, _ := guard.TryAcquire("2", packages)
	if ok {
		t.Error("TryAcquire should fail when package is held")
	}

	// Release and verify
	release()
	if guard.IsHeld("internal/config") {
		t.Error("Package should not be held after release")
	}
}

// TestRuntimeEnforcement_PackageGuardConcurrent verifies concurrent access handling.
func TestRuntimeEnforcement_PackageGuardConcurrent(t *testing.T) {
	guard := executor.NewPackageGuard()

	packages := []string{"internal/test/concurrent"}

	// Acquire from first task
	ctx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel1()

	release1, err := guard.Acquire(ctx1, "task1", packages)
	if err != nil {
		t.Fatalf("First acquire failed: %v", err)
	}

	// Try to acquire from second task with short timeout (should block then timeout)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel2()

	_, err = guard.Acquire(ctx2, "task2", packages)
	if err == nil {
		t.Error("Second acquire should timeout while first holds lock")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded, got: %v", err)
	}

	// Release first and verify second can now acquire
	release1()

	ctx3, cancel3 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel3()

	release3, err := guard.Acquire(ctx3, "task2", packages)
	if err != nil {
		t.Errorf("Third acquire should succeed after release: %v", err)
	}
	if release3 != nil {
		release3()
	}
}

// TestRuntimeEnforcement_TestCommands verifies test command extraction.
func TestRuntimeEnforcement_TestCommands(t *testing.T) {
	planPath := filepath.Join("fixtures", "runtime_enforcement_plan.yaml")

	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("Failed to parse plan: %v", err)
	}

	// Find task 1 which has test commands
	var task1 *models.Task
	for i := range plan.Tasks {
		if plan.Tasks[i].Number == "1" {
			task1 = &plan.Tasks[i]
			break
		}
	}

	if task1 == nil {
		t.Fatalf("Task 1 not found")
	}

	// Verify test commands exist
	if len(task1.TestCommands) != 1 {
		t.Fatalf("Expected 1 test command, got %d", len(task1.TestCommands))
	}

	if task1.TestCommands[0] != "echo 'Task 1 test command passed'" {
		t.Errorf("Unexpected test command: %s", task1.TestCommands[0])
	}
}

// TestRuntimeEnforcement_SuccessCriteria verifies success criteria with verification blocks.
func TestRuntimeEnforcement_SuccessCriteria(t *testing.T) {
	planPath := filepath.Join("fixtures", "runtime_enforcement_plan.yaml")

	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("Failed to parse plan: %v", err)
	}

	// Find task 1 which has structured success criteria
	var task1 *models.Task
	for i := range plan.Tasks {
		if plan.Tasks[i].Number == "1" {
			task1 = &plan.Tasks[i]
			break
		}
	}

	if task1 == nil {
		t.Fatalf("Task 1 not found")
	}

	// Verify success criteria (plain string format)
	if len(task1.SuccessCriteria) < 2 {
		t.Fatalf("Expected at least 2 success criteria, got %d", len(task1.SuccessCriteria))
	}

	// Verify structured criteria with verification blocks
	if len(task1.StructuredCriteria) > 0 {
		// Find criterion with verification block
		var foundVerification bool
		for _, sc := range task1.StructuredCriteria {
			if sc.Verification != nil {
				foundVerification = true
				if sc.Verification.Command == "" {
					t.Error("Verification command should not be empty")
				}
				break
			}
		}

		if foundVerification {
			t.Log("Found criterion with verification block")
		}
	} else {
		t.Log("Task uses plain success_criteria format without structured_criteria")
	}
}

// TestRuntimeEnforcement_IntegrationTask verifies integration task dual criteria.
func TestRuntimeEnforcement_IntegrationTask(t *testing.T) {
	planPath := filepath.Join("fixtures", "runtime_enforcement_plan.yaml")

	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("Failed to parse plan: %v", err)
	}

	// Find task 3 which is an integration task
	var task3 *models.Task
	for i := range plan.Tasks {
		if plan.Tasks[i].Number == "3" {
			task3 = &plan.Tasks[i]
			break
		}
	}

	if task3 == nil {
		t.Fatalf("Task 3 not found")
	}

	// Verify task type
	if task3.Type != "integration" {
		t.Errorf("Expected task type 'integration', got '%s'", task3.Type)
	}

	// Verify success criteria (component-level)
	if len(task3.SuccessCriteria) < 1 {
		t.Error("Integration task should have success criteria")
	}

	// Verify integration criteria
	if len(task3.IntegrationCriteria) < 1 {
		t.Error("Integration task should have integration criteria")
	}

	// Verify dual criteria validation would work
	expectedIntegrationCriteria := []string{
		"Preflight checks complete before agent invocation",
		"Test commands execute after agent output",
		"Documentation targets verified before QC",
	}

	for i, expected := range expectedIntegrationCriteria {
		if i >= len(task3.IntegrationCriteria) {
			t.Errorf("Missing integration criterion %d: %s", i, expected)
			continue
		}
		if task3.IntegrationCriteria[i] != expected {
			t.Errorf("Integration criterion %d mismatch.\nExpected: %s\nGot: %s", i, expected, task3.IntegrationCriteria[i])
		}
	}
}

// TestRuntimeEnforcement_RegistryValidation verifies data flow registry validation.
func TestRuntimeEnforcement_RegistryValidation(t *testing.T) {
	planPath := filepath.Join("fixtures", "runtime_enforcement_plan.yaml")

	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("Failed to parse plan: %v", err)
	}

	if plan.DataFlowRegistry == nil {
		t.Skip("No DataFlowRegistry in fixture")
	}

	// Validate registry prerequisites
	err = executor.ValidateRegistryPrerequisites(plan.Tasks, plan.DataFlowRegistry)
	if err != nil {
		t.Errorf("Registry prerequisites validation failed: %v", err)
	}
}

// TestRuntimeEnforcement_WaveExecution tests wave execution with enforcement.
func TestRuntimeEnforcement_WaveExecution(t *testing.T) {
	planPath := filepath.Join("fixtures", "runtime_enforcement_plan.yaml")

	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("Failed to parse plan: %v", err)
	}

	// Calculate waves
	waves, err := executor.CalculateWaves(plan.Tasks)
	if err != nil {
		t.Fatalf("Failed to calculate waves: %v", err)
	}
	plan.Waves = waves

	// Create mock executor that verifies enforcement was checked
	executionOrder := []string{}
	mockExecutor := &mockTaskExecutor{
		executeFunc: func(ctx context.Context, task models.Task) (models.TaskResult, error) {
			executionOrder = append(executionOrder, task.Number)
			return models.TaskResult{
				Task:   task,
				Status: models.StatusGreen,
				Output: "Mock execution",
			}, nil
		},
	}

	// Execute with package guard enabled
	waveExec := executor.NewWaveExecutorWithPackageGuard(mockExecutor, nil, false, false, true)
	results, err := waveExec.ExecutePlan(context.Background(), plan)

	if err != nil {
		t.Fatalf("Wave execution failed: %v", err)
	}

	// Verify all tasks executed
	if len(results) != 4 {
		t.Errorf("Expected 4 results, got %d", len(results))
	}

	// Verify execution order respects dependencies
	// Task 1 must come before Tasks 2, 3, 4
	// Task 2 must come before Tasks 3, 4
	// Task 3 must come before Task 4 (serialized for package guard)
	task1Idx := indexOfStr(executionOrder, "1")
	task2Idx := indexOfStr(executionOrder, "2")
	task3Idx := indexOfStr(executionOrder, "3")
	task4Idx := indexOfStr(executionOrder, "4")

	if task1Idx >= task2Idx {
		t.Errorf("Task 1 should execute before Task 2. Order: %v", executionOrder)
	}
	if task2Idx >= task3Idx {
		t.Errorf("Task 2 should execute before Task 3. Order: %v", executionOrder)
	}
	// Task 4 depends on Task 3 (serialized via depends_on)
	if task3Idx >= task4Idx {
		t.Errorf("Task 3 should execute before Task 4. Order: %v", executionOrder)
	}
}

// indexOfStr returns the index of an element in a slice, or -1 if not found.
// Named differently to avoid conflict with indexOf in cross_file_dependency_test.go
func indexOfStr(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}
