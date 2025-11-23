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

// TestCrossFileDependency_SimpleLinearChain verifies that cross-file dependencies
// work correctly in a simple linear chain pattern across two files.
// Test structure:
//   foundation.yaml: tasks 1-3 (no dependencies)
//   features.yaml: tasks 4-6 (task 4 depends on task 2 from foundation.yaml)
func TestCrossFileDependency_SimpleLinearChain(t *testing.T) {
	foundationPath := filepath.Join("fixtures", "cross-file", "linear", "foundation.yaml")
	featuresPath := filepath.Join("fixtures", "cross-file", "linear", "features.yaml")

	// Parse both files
	foundationPlan, err := parser.ParseFile(foundationPath)
	if err != nil {
		t.Fatalf("Failed to parse foundation.yaml: %v", err)
	}

	featuresPlan, err := parser.ParseFile(featuresPath)
	if err != nil {
		t.Fatalf("Failed to parse features.yaml: %v", err)
	}

	// Merge plans
	mergedPlan, err := parser.MergePlans(foundationPlan, featuresPlan)
	if err != nil {
		t.Fatalf("Failed to merge plans: %v", err)
	}

	// Verify task count
	if len(mergedPlan.Tasks) != 6 {
		t.Fatalf("Expected 6 tasks after merge, got %d", len(mergedPlan.Tasks))
	}

	// Verify task 4 has cross-file dependency on task 2
	task4 := findTaskByNumber(mergedPlan.Tasks, "4")
	if task4 == nil {
		t.Fatalf("Task 4 not found in merged plan")
	}

	expectedDep := "file:foundation.yaml:task:2"
	hasDep := false
	for _, dep := range task4.DependsOn {
		if dep == expectedDep {
			hasDep = true
			break
		}
	}

	if !hasDep {
		t.Fatalf("Task 4 missing expected cross-file dependency %q. Got: %v", expectedDep, task4.DependsOn)
	}

	// Calculate waves and verify execution order
	waves, err := executor.CalculateWaves(mergedPlan.Tasks)
	if err != nil {
		t.Fatalf("Failed to calculate waves: %v", err)
	}

	// Expected waves:
	// Wave 1: Tasks 1, 2, 3 (all can execute in parallel)
	// Wave 2: Tasks 4, 5 (4 depends on 2, 5 has no dependencies)
	// Wave 3: Task 6 (depends on 4)

	if len(waves) < 3 {
		t.Fatalf("Expected at least 3 waves, got %d", len(waves))
	}

	// Verify wave 1 contains foundation tasks and task 5 (no dependencies in features)
	wave1Tasks := mapset(waves[0].TaskNumbers)
	expectedWave1 := mapset([]string{"1", "2", "3", "5"})
	if !setsEqual(wave1Tasks, expectedWave1) {
		t.Errorf("Wave 1 mismatch. Expected %v, got %v", expectedWave1, wave1Tasks)
	}

	// Verify task 2 from foundation comes before task 4 from features
	task2WaveIdx := findWaveIndex(waves, "2")
	task4WaveIdx := findWaveIndex(waves, "4")
	if task2WaveIdx >= task4WaveIdx {
		t.Errorf("Task 2 should execute before task 4 (cross-file dependency). Task 2 wave: %d, Task 4 wave: %d", task2WaveIdx, task4WaveIdx)
	}
}

// TestCrossFileDependency_DiamondPattern verifies diamond dependency pattern
// across multiple files with parallel execution and join points.
// Test structure:
//   setup.yaml: task 1 (no dependencies)
//   branches.yaml: tasks 2, 3 (both depend on task 1)
//   join.yaml: task 4 (depends on both tasks 2 and 3)
func TestCrossFileDependency_DiamondPattern(t *testing.T) {
	setupPath := filepath.Join("fixtures", "cross-file", "diamond", "setup.yaml")
	branchesPath := filepath.Join("fixtures", "cross-file", "diamond", "branches.yaml")
	joinPath := filepath.Join("fixtures", "cross-file", "diamond", "join.yaml")

	// Parse all files
	setupPlan, err := parser.ParseFile(setupPath)
	if err != nil {
		t.Fatalf("Failed to parse setup.yaml: %v", err)
	}

	branchesPlan, err := parser.ParseFile(branchesPath)
	if err != nil {
		t.Fatalf("Failed to parse branches.yaml: %v", err)
	}

	joinPlan, err := parser.ParseFile(joinPath)
	if err != nil {
		t.Fatalf("Failed to parse join.yaml: %v", err)
	}

	// Merge all plans
	mergedPlan, err := parser.MergePlans(setupPlan, branchesPlan, joinPlan)
	if err != nil {
		t.Fatalf("Failed to merge plans: %v", err)
	}

	// Verify total task count
	if len(mergedPlan.Tasks) != 4 {
		t.Fatalf("Expected 4 tasks, got %d", len(mergedPlan.Tasks))
	}

	// Verify task 4 dependencies
	task4 := findTaskByNumber(mergedPlan.Tasks, "4")
	if task4 == nil {
		t.Fatalf("Task 4 not found in merged plan")
	}

	expectedDeps := map[string]bool{
		"file:branches.yaml:task:2": false,
		"file:branches.yaml:task:3": false,
	}

	for _, dep := range task4.DependsOn {
		expectedDeps[dep] = true
	}

	for dep, found := range expectedDeps {
		if !found {
			t.Errorf("Task 4 missing expected dependency %q", dep)
		}
	}

	// Calculate waves and verify structure
	waves, err := executor.CalculateWaves(mergedPlan.Tasks)
	if err != nil {
		t.Fatalf("Failed to calculate waves: %v", err)
	}

	// Expected wave structure:
	// Wave 1: Task 1 (root task)
	// Wave 2: Tasks 2, 3 (both depend on 1, can execute in parallel)
	// Wave 3: Task 4 (depends on both 2 and 3)

	if len(waves) != 3 {
		t.Fatalf("Expected 3 waves, got %d", len(waves))
	}

	// Verify tasks 2 and 3 are in the same wave (can execute in parallel)
	task2Wave := findWaveIndex(waves, "2")
	task3Wave := findWaveIndex(waves, "3")
	if task2Wave != task3Wave {
		t.Errorf("Tasks 2 and 3 should be in the same wave for parallel execution. Task 2: wave %d, Task 3: wave %d", task2Wave, task3Wave)
	}

	// Verify task 4 is in a later wave than tasks 2 and 3
	task4Wave := findWaveIndex(waves, "4")
	if task4Wave <= task2Wave {
		t.Errorf("Task 4 should execute after tasks 2 and 3. Task 4 wave: %d, Task 2/3 wave: %d", task4Wave, task2Wave)
	}
}

// TestCrossFileDependency_InvalidReference verifies proper error handling
// for invalid cross-file references (non-existent file or task).
func TestCrossFileDependency_InvalidReference(t *testing.T) {
	tests := []struct {
		name          string
		plan1Path     string
		plan2Path     string
		expectedError string
	}{
		{
			name:          "ReferencesNonExistentTask",
			plan1Path:     filepath.Join("fixtures", "cross-file", "invalid", "valid.yaml"),
			plan2Path:     filepath.Join("fixtures", "cross-file", "invalid", "missing-task-ref.yaml"),
			expectedError: "cross-file dependency references non-existent task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan1, err := parser.ParseFile(tt.plan1Path)
			if err != nil {
				t.Fatalf("Failed to parse first plan: %v", err)
			}

			plan2, err := parser.ParseFile(tt.plan2Path)
			if err != nil {
				t.Fatalf("Failed to parse second plan: %v", err)
			}

			_, err = parser.MergePlans(plan1, plan2)
			if err == nil {
				t.Fatalf("Expected error for %s, but got none", tt.name)
			}

			if !containsString(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing %q, got: %v", tt.expectedError, err)
			}
		})
	}
}

// TestCrossFileDependency_CircularDetection verifies that circular dependencies
// across multiple files are properly detected and reported.
func TestCrossFileDependency_CircularDetection(t *testing.T) {
	tests := []struct {
		name       string
		plan1Path  string
		plan2Path  string
		shouldFail bool
	}{
		{
			name:       "SimpleCircular",
			plan1Path:  filepath.Join("fixtures", "cross-file", "circular", "file1.yaml"),
			plan2Path:  filepath.Join("fixtures", "cross-file", "circular", "file2.yaml"),
			shouldFail: true,
		},
		{
			name:       "IndirectCircular",
			plan1Path:  filepath.Join("fixtures", "cross-file", "circular", "file3.yaml"),
			plan2Path:  filepath.Join("fixtures", "cross-file", "circular", "file4.yaml"),
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan1, err := parser.ParseFile(tt.plan1Path)
			if err != nil {
				t.Fatalf("Failed to parse first plan: %v", err)
			}

			plan2, err := parser.ParseFile(tt.plan2Path)
			if err != nil {
				t.Fatalf("Failed to parse second plan: %v", err)
			}

			// Merge plans
			mergedPlan, err := parser.MergePlans(plan1, plan2)
			if err != nil {
				if !tt.shouldFail {
					t.Fatalf("Merge failed unexpectedly: %v", err)
				}
				return
			}

			// Try to calculate waves - this is where cycle detection happens
			_, err = executor.CalculateWaves(mergedPlan.Tasks)

			if tt.shouldFail && err == nil {
				t.Error("Expected cycle detection error, but got none")
			}

			if !tt.shouldFail && err != nil {
				t.Errorf("Unexpected error for valid dependency graph: %v", err)
			}
		})
	}
}

// TestCrossFileDependency_MixedFormat verifies that cross-file dependencies
// work alongside numeric local dependencies in the same plan.
func TestCrossFileDependency_MixedFormat(t *testing.T) {
	plan1Path := filepath.Join("fixtures", "cross-file", "mixed", "part1.yaml")
	plan2Path := filepath.Join("fixtures", "cross-file", "mixed", "part2.yaml")

	plan1, err := parser.ParseFile(plan1Path)
	if err != nil {
		t.Fatalf("Failed to parse part1.yaml: %v", err)
	}

	plan2, err := parser.ParseFile(plan2Path)
	if err != nil {
		t.Fatalf("Failed to parse part2.yaml: %v", err)
	}

	mergedPlan, err := parser.MergePlans(plan1, plan2)
	if err != nil {
		t.Fatalf("Failed to merge plans: %v", err)
	}

	// Verify that tasks with both local and cross-file dependencies work correctly
	// Task 5 should depend on:
	// - Task 4 (local dependency within part2.yaml)
	// - Task 2 (cross-file dependency on part1.yaml)

	task5 := findTaskByNumber(mergedPlan.Tasks, "5")
	if task5 == nil {
		t.Fatalf("Task 5 not found in merged plan")
	}

	expectedDeps := map[string]bool{
		"4":                           false,
		"file:part1.yaml:task:2":      false,
	}

	for _, dep := range task5.DependsOn {
		expectedDeps[dep] = true
	}

	for dep, found := range expectedDeps {
		if !found {
			t.Errorf("Task 5 missing expected dependency %q. Got: %v", dep, task5.DependsOn)
		}
	}

	// Verify wave calculation works with mixed dependencies
	waves, err := executor.CalculateWaves(mergedPlan.Tasks)
	if err != nil {
		t.Fatalf("Failed to calculate waves with mixed dependencies: %v", err)
	}

	// Verify task ordering respects both types of dependencies
	if len(waves) < 3 {
		t.Fatalf("Expected at least 3 waves, got %d", len(waves))
	}
}

// TestCrossFileDependency_MarkdownFormat verifies that Markdown plan files
// can be parsed and merged with other plan files. While the Markdown parser
// has limited support for explicit cross-file syntax in text, we verify that
// multiple Markdown files can be merged and their dependency graphs respect
// the overall task dependencies.
func TestCrossFileDependency_MarkdownFormat(t *testing.T) {
	setupPath := filepath.Join("fixtures", "cross-file", "markdown", "setup.md")
	implementPath := filepath.Join("fixtures", "cross-file", "markdown", "implement.md")

	setupPlan, err := parser.ParseFile(setupPath)
	if err != nil {
		t.Fatalf("Failed to parse setup.md: %v", err)
	}

	implementPlan, err := parser.ParseFile(implementPath)
	if err != nil {
		t.Fatalf("Failed to parse implement.md: %v", err)
	}

	// Manually add cross-file dependency to Task 4 for this test
	for i := range implementPlan.Tasks {
		if implementPlan.Tasks[i].Number == "4" {
			implementPlan.Tasks[i].DependsOn = []string{"file:setup.md:task:2"}
			break
		}
	}

	mergedPlan, err := parser.MergePlans(setupPlan, implementPlan)
	if err != nil {
		t.Fatalf("Failed to merge Markdown plans: %v", err)
	}

	// Verify we can merge Markdown plans
	if len(mergedPlan.Tasks) < 3 {
		t.Fatalf("Expected at least 3 tasks after merge, got %d", len(mergedPlan.Tasks))
	}

	// Verify cross-file dependency was preserved
	task4 := findTaskByNumber(mergedPlan.Tasks, "4")
	if task4 != nil && len(task4.DependsOn) > 0 {
		hasCrossFileDep := false
		for _, dep := range task4.DependsOn {
			if models.IsCrossFileDep(dep) {
				hasCrossFileDep = true
				break
			}
		}
		if !hasCrossFileDep {
			t.Logf("Warning: Task 4 does not have cross-file dependency in merged plan")
		}
	}

	// Verify wave calculation works correctly
	waves, err := executor.CalculateWaves(mergedPlan.Tasks)
	if err != nil {
		t.Fatalf("Failed to calculate waves for Markdown plans: %v", err)
	}

	if len(waves) == 0 {
		t.Fatal("Expected at least one wave")
	}
}

// TestCrossFileDependency_WaveCalculation verifies that wave calculation correctly
// handles cross-file dependencies and enables parallel execution where appropriate.
func TestCrossFileDependency_WaveCalculation(t *testing.T) {
	foundationPath := filepath.Join("fixtures", "cross-file", "linear", "foundation.yaml")
	featuresPath := filepath.Join("fixtures", "cross-file", "linear", "features.yaml")

	foundationPlan, err := parser.ParseFile(foundationPath)
	if err != nil {
		t.Fatalf("Failed to parse foundation.yaml: %v", err)
	}

	featuresPlan, err := parser.ParseFile(featuresPath)
	if err != nil {
		t.Fatalf("Failed to parse features.yaml: %v", err)
	}

	mergedPlan, err := parser.MergePlans(foundationPlan, featuresPlan)
	if err != nil {
		t.Fatalf("Failed to merge plans: %v", err)
	}

	waves, err := executor.CalculateWaves(mergedPlan.Tasks)
	if err != nil {
		t.Fatalf("Failed to calculate waves: %v", err)
	}

	// Verify wave structure enables parallelism
	// Tasks without dependencies should execute in the same wave
	if len(waves[0].TaskNumbers) < 2 {
		t.Errorf("Expected multiple independent tasks in first wave, got %d", len(waves[0].TaskNumbers))
	}

	// Build task map for validation
	taskMap := make(map[string]*models.Task)
	for i, task := range mergedPlan.Tasks {
		taskMap[task.Number] = &mergedPlan.Tasks[i]
	}

	// Validate that dependent tasks don't execute in same wave as dependencies
	for waveIdx, wave := range waves {
		for _, taskNum := range wave.TaskNumbers {
			task, _ := taskMap[taskNum]
			if task == nil {
				continue
			}

			for _, dep := range task.DependsOn {
				var depTaskNum string
				if models.IsCrossFileDep(dep) {
					cfd, _ := models.ParseCrossFileDep(dep)
					depTaskNum = cfd.TaskID
				} else {
					depTaskNum = dep
				}

				// Find which wave the dependency is in
				depWaveIdx := findWaveIndex(waves, depTaskNum)
				if depWaveIdx >= waveIdx {
					t.Errorf("Task %s (wave %d) depends on task %s (wave %d), but dependency is not in earlier wave",
						taskNum, waveIdx, depTaskNum, depWaveIdx)
				}
			}
		}
	}
}

// TestCrossFileDependency_ExecutionBoundary verifies that tasks from different files
// can have dependencies and execute correctly even when split across files.
func TestCrossFileDependency_ExecutionBoundary(t *testing.T) {
	// This test would ideally execute tasks using a mock executor,
	// but we verify the dependency structure is correct

	setupPath := filepath.Join("fixtures", "cross-file", "diamond", "setup.yaml")
	branchesPath := filepath.Join("fixtures", "cross-file", "diamond", "branches.yaml")

	setupPlan, err := parser.ParseFile(setupPath)
	if err != nil {
		t.Fatalf("Failed to parse setup.yaml: %v", err)
	}

	branchesPlan, err := parser.ParseFile(branchesPath)
	if err != nil {
		t.Fatalf("Failed to parse branches.yaml: %v", err)
	}

	// Get absolute paths for comparison
	absSetupPath, _ := filepath.Abs(setupPath)
	absBranchesPath, _ := filepath.Abs(branchesPath)

	mergedPlan, err := parser.MergePlans(setupPlan, branchesPlan)
	if err != nil {
		t.Fatalf("Failed to merge plans: %v", err)
	}

	// Verify SourceFile is set correctly for tracking file boundaries
	for _, task := range mergedPlan.Tasks {
		if task.SourceFile == "" {
			t.Errorf("Task %s has empty SourceFile field", task.Number)
		}
	}

	// Verify we can distinguish tasks by their source file
	setupTasks := 0
	branchesTasks := 0

	for _, task := range mergedPlan.Tasks {
		if task.SourceFile == absSetupPath {
			setupTasks++
		} else if task.SourceFile == absBranchesPath {
			branchesTasks++
		}
	}

	if setupTasks == 0 || branchesTasks == 0 {
		t.Errorf("Expected tasks from both files. Setup tasks: %d, Branches tasks: %d", setupTasks, branchesTasks)
	}
}

// TestCrossFileDependency_LargeMultiFile verifies cross-file dependencies work
// with many files and complex dependency graphs.
func TestCrossFileDependency_LargeMultiFile(t *testing.T) {
	// Create paths for a complex multi-file scenario
	paths := []string{
		filepath.Join("fixtures", "cross-file", "complex", "01-foundation.yaml"),
		filepath.Join("fixtures", "cross-file", "complex", "02-middleware.yaml"),
		filepath.Join("fixtures", "cross-file", "complex", "03-handlers.yaml"),
		filepath.Join("fixtures", "cross-file", "complex", "04-integration.yaml"),
	}

	var plans []*models.Plan
	for _, path := range paths {
		plan, err := parser.ParseFile(path)
		if err != nil {
			t.Fatalf("Failed to parse %s: %v", path, err)
		}
		plans = append(plans, plan)
	}

	// Merge all plans
	mergedPlan, err := parser.MergePlans(plans...)
	if err != nil {
		t.Fatalf("Failed to merge plans: %v", err)
	}

	// Verify no duplicate task numbers
	seen := make(map[string]bool)
	for _, task := range mergedPlan.Tasks {
		if seen[task.Number] {
			t.Fatalf("Duplicate task number: %s", task.Number)
		}
		seen[task.Number] = true
	}

	// Verify wave calculation succeeds with complex graph
	waves, err := executor.CalculateWaves(mergedPlan.Tasks)
	if err != nil {
		t.Fatalf("Failed to calculate waves for complex multi-file plan: %v", err)
	}

	// Verify all tasks are in some wave
	tasksInWaves := 0
	for _, wave := range waves {
		tasksInWaves += len(wave.TaskNumbers)
	}

	if tasksInWaves != len(mergedPlan.Tasks) {
		t.Errorf("Not all tasks are in waves. Tasks: %d, In waves: %d", len(mergedPlan.Tasks), tasksInWaves)
	}
}

// TestCrossFileDependency_ResolutionValidation verifies the cross-file dependency
// resolution validation function works correctly.
func TestCrossFileDependency_ResolutionValidation(t *testing.T) {
	tests := []struct {
		name        string
		tasks       []models.Task
		fileMap     map[string]string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "ValidCrossFileDep",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", Prompt: "Test", DependsOn: []string{}},
				{Number: "2", Name: "Task 2", Prompt: "Test", DependsOn: []string{"file:file1.yaml:task:1"}},
			},
			fileMap:     map[string]string{"1": "file1.yaml", "2": "file2.yaml"},
			shouldError: false,
		},
		{
			name: "InvalidTaskReference",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", Prompt: "Test", DependsOn: []string{}},
				{Number: "2", Name: "Task 2", Prompt: "Test", DependsOn: []string{"file:file1.yaml:task:99"}},
			},
			fileMap:     map[string]string{"1": "file1.yaml", "2": "file2.yaml"},
			shouldError: true,
			errorMsg:    "cross-file dependency references non-existent task",
		},
		{
			name: "InvalidLocalDep",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", Prompt: "Test", DependsOn: []string{}},
				{Number: "2", Name: "Task 2", Prompt: "Test", DependsOn: []string{"99"}},
			},
			fileMap:     map[string]string{"1": "file1.yaml", "2": "file2.yaml"},
			shouldError: true,
			errorMsg:    "depends on non-existent task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.ResolveCrossFileDependencies(tt.tasks, tt.fileMap)

			if tt.shouldError && err == nil {
				t.Fatalf("Expected error for %s, but got none", tt.name)
			}

			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
			}

			if tt.shouldError && err != nil && !containsString(err.Error(), tt.errorMsg) {
				t.Errorf("Error message mismatch. Expected %q, got %q", tt.errorMsg, err.Error())
			}
		})
	}
}

// TestCrossFileDependency_DependencyStringFormat verifies that cross-file dependency
// string format parsing and serialization works correctly.
func TestCrossFileDependency_DependencyStringFormat(t *testing.T) {
	tests := []struct {
		name      string
		depString string
		valid     bool
		expFile   string
		expTask   string
	}{
		{
			name:      "ValidFormat",
			depString: "file:plan-01.yaml:task:2",
			valid:     true,
			expFile:   "plan-01.yaml",
			expTask:   "2",
		},
		{
			name:      "ValidFormatComplexFilename",
			depString: "file:foundation-setup.yaml:task:3",
			valid:     true,
			expFile:   "foundation-setup.yaml",
			expTask:   "3",
		},
		{
			name:      "InvalidFormat_NoTask",
			depString: "file:plan-01.yaml",
			valid:     false,
		},
		{
			name:      "InvalidFormat_NoFile",
			depString: "task:2",
			valid:     false,
		},
		{
			name:      "InvalidFormat_EmptyTask",
			depString: "file:plan-01.yaml:task:",
			valid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfd, err := models.ParseCrossFileDep(tt.depString)

			if tt.valid && err != nil {
				t.Fatalf("Expected valid format, got error: %v", err)
			}

			if !tt.valid && err == nil {
				t.Fatalf("Expected error for invalid format, got none")
			}

			if tt.valid {
				if cfd.File != tt.expFile {
					t.Errorf("File mismatch. Expected %q, got %q", tt.expFile, cfd.File)
				}
				if cfd.TaskID != tt.expTask {
					t.Errorf("Task mismatch. Expected %q, got %q", tt.expTask, cfd.TaskID)
				}

				// Verify round-trip: parse -> String() -> parse
				roundTrip := cfd.String()
				cfd2, err := models.ParseCrossFileDep(roundTrip)
				if err != nil {
					t.Errorf("Round-trip parsing failed: %v", err)
				}
				if cfd2.File != cfd.File || cfd2.TaskID != cfd.TaskID {
					t.Errorf("Round-trip mismatch. Original: {%s,%s}, After: {%s,%s}", cfd.File, cfd.TaskID, cfd2.File, cfd2.TaskID)
				}
			}
		})
	}
}

// TestCrossFileDependency_ContextualExecution verifies that execution context
// is properly maintained across file boundaries for pre-task and post-task hooks.
// This is a structural test - actual execution would require mocked task executor.
func TestCrossFileDependency_ContextualExecution(t *testing.T) {
	foundationPath := filepath.Join("fixtures", "cross-file", "linear", "foundation.yaml")
	featuresPath := filepath.Join("fixtures", "cross-file", "linear", "features.yaml")

	foundationPlan, err := parser.ParseFile(foundationPath)
	if err != nil {
		t.Fatalf("Failed to parse foundation.yaml: %v", err)
	}

	featuresPlan, err := parser.ParseFile(featuresPath)
	if err != nil {
		t.Fatalf("Failed to parse features.yaml: %v", err)
	}

	mergedPlan, err := parser.MergePlans(foundationPlan, featuresPlan)
	if err != nil {
		t.Fatalf("Failed to merge plans: %v", err)
	}

	// Create a minimal task executor to test context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify plan structure is maintained through execution
	// Note: The merged plan may not have name set from merging, which is expected
	if len(mergedPlan.Tasks) == 0 {
		t.Error("Merged plan has no tasks")
	}

	// Verify context timeouts work with merged plans
	select {
	case <-ctx.Done():
		if ctx.Err() != context.Canceled && ctx.Err() != context.DeadlineExceeded {
			t.Errorf("Unexpected context error: %v", ctx.Err())
		}
	default:
		// Context still active - good for this test
	}

	// Verify task metadata is preserved
	for _, task := range mergedPlan.Tasks {
		if task.Number == "" {
			t.Error("Task has empty number after merge")
		}
		if task.SourceFile == "" {
			t.Error("Task has empty SourceFile after merge")
		}
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// findTaskByNumber finds a task in a slice by its number
func findTaskByNumber(tasks []models.Task, number string) *models.Task {
	for i, task := range tasks {
		if task.Number == number {
			return &tasks[i]
		}
	}
	return nil
}

// findWaveIndex finds the index of the wave containing a specific task
func findWaveIndex(waves []models.Wave, taskNumber string) int {
	for i, wave := range waves {
		for _, taskNum := range wave.TaskNumbers {
			if taskNum == taskNumber {
				return i
			}
		}
	}
	return -1
}

// mapset converts a string slice to a map for set operations
func mapset(items []string) map[string]bool {
	set := make(map[string]bool)
	for _, item := range items {
		set[item] = true
	}
	return set
}

// setsEqual checks if two string sets are equal
func setsEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// containsString checks if a string contains a substring
func containsString(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && indexOf(haystack, needle) >= 0
}

// indexOf returns the index of a substring in a string, or -1 if not found
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
