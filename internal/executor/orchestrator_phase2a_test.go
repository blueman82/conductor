package executor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// TestMergePlansSimple verifies basic plan merging with no conflicts
func TestMergePlansSimple(t *testing.T) {
	plan1 := &models.Plan{
		Name:     "Plan 1",
		FilePath: "plan1.yaml",
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1"},
			{Number: "2", Name: "Task 2", DependsOn: []string{"1"}},
		},
		WorktreeGroups: []models.WorktreeGroup{
			{GroupID: "group-a", Description: "Group A"},
		},
	}

	plan2 := &models.Plan{
		Name:     "Plan 2",
		FilePath: "plan2.yaml",
		Tasks: []models.Task{
			{Number: "3", Name: "Task 3", DependsOn: []string{"2"}},
			{Number: "4", Name: "Task 4", DependsOn: []string{"3"}},
		},
		WorktreeGroups: []models.WorktreeGroup{
			{GroupID: "group-b", Description: "Group B"},
		},
	}

	merged, err := MergePlans(plan1, plan2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(merged.Tasks) != 4 {
		t.Errorf("expected 4 tasks, got %d", len(merged.Tasks))
	}

	if len(merged.WorktreeGroups) != 2 {
		t.Errorf("expected 2 worktree groups, got %d", len(merged.WorktreeGroups))
	}

	// Verify FileToTaskMap
	if len(merged.FileToTaskMap) != 2 {
		t.Errorf("expected 2 file mappings, got %d", len(merged.FileToTaskMap))
	}
	if len(merged.FileToTaskMap["plan1.yaml"]) != 2 {
		t.Errorf("expected 2 tasks for plan1.yaml, got %d", len(merged.FileToTaskMap["plan1.yaml"]))
	}
	if len(merged.FileToTaskMap["plan2.yaml"]) != 2 {
		t.Errorf("expected 2 tasks for plan2.yaml, got %d", len(merged.FileToTaskMap["plan2.yaml"]))
	}
}

// TestMergePlansConflictingTaskNumbers verifies error on duplicate task numbers
func TestMergePlansConflictingTaskNumbers(t *testing.T) {
	plan1 := &models.Plan{
		Name:     "Plan 1",
		FilePath: "plan1.yaml",
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1"},
			{Number: "2", Name: "Task 2"},
		},
	}

	plan2 := &models.Plan{
		Name:     "Plan 2",
		FilePath: "plan2.yaml",
		Tasks: []models.Task{
			{Number: "2", Name: "Conflicting Task"},
			{Number: "3", Name: "Task 3"},
		},
	}

	merged, err := MergePlans(plan1, plan2)
	if err == nil {
		t.Error("expected error for conflicting task numbers, got nil")
	}
	if merged != nil {
		t.Error("expected nil merged plan on conflict")
	}
}

// TestMergePlansWithCyclicDependencies verifies cycle detection in merged plans
func TestMergePlansWithCyclicDependencies(t *testing.T) {
	plan1 := &models.Plan{
		Name:     "Plan 1",
		FilePath: "plan1.yaml",
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", DependsOn: []string{"3"}},
		},
	}

	plan2 := &models.Plan{
		Name:     "Plan 2",
		FilePath: "plan2.yaml",
		Tasks: []models.Task{
			{Number: "2", Name: "Task 2", DependsOn: []string{"1"}},
			{Number: "3", Name: "Task 3", DependsOn: []string{"2"}},
		},
	}

	merged, err := MergePlans(plan1, plan2)
	if err == nil {
		t.Error("expected error for cyclic dependencies, got nil")
	}
	if merged != nil {
		t.Error("expected nil merged plan on cycle detection")
	}
}

// TestMergePlansMultipleFiles tests merging 3+ files with complex dependencies
func TestMergePlansMultipleFiles(t *testing.T) {
	plan1 := &models.Plan{
		Name:     "Part 1",
		FilePath: "part1.yaml",
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1"},
			{Number: "2", Name: "Task 2", DependsOn: []string{"1"}},
		},
	}

	plan2 := &models.Plan{
		Name:     "Part 2",
		FilePath: "part2.yaml",
		Tasks: []models.Task{
			{Number: "3", Name: "Task 3", DependsOn: []string{"2"}},
		},
	}

	plan3 := &models.Plan{
		Name:     "Part 3",
		FilePath: "part3.yaml",
		Tasks: []models.Task{
			{Number: "4", Name: "Task 4", DependsOn: []string{"3"}},
		},
	}

	merged, err := MergePlans(plan1, plan2, plan3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(merged.Tasks) != 4 {
		t.Errorf("expected 4 tasks, got %d", len(merged.Tasks))
	}

	if len(merged.FileToTaskMap) != 3 {
		t.Errorf("expected 3 file mappings, got %d", len(merged.FileToTaskMap))
	}
}

// TestMergePlansWorktreeGroupDeduplication tests group deduplication
func TestMergePlansWorktreeGroupDeduplication(t *testing.T) {
	group := models.WorktreeGroup{
		GroupID:        "shared-group",
		Description:    "Shared group",
		ExecutionModel: "parallel",
		Isolation:      "weak",
	}

	plan1 := &models.Plan{
		Name:           "Plan 1",
		FilePath:       "plan1.yaml",
		Tasks:          []models.Task{{Number: "1", Name: "Task 1"}},
		WorktreeGroups: []models.WorktreeGroup{group},
	}

	plan2 := &models.Plan{
		Name:           "Plan 2",
		FilePath:       "plan2.yaml",
		Tasks:          []models.Task{{Number: "2", Name: "Task 2"}},
		WorktreeGroups: []models.WorktreeGroup{group}, // Same group
	}

	merged, err := MergePlans(plan1, plan2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(merged.WorktreeGroups) != 1 {
		t.Errorf("expected 1 deduplicated group, got %d", len(merged.WorktreeGroups))
	}
}

// TestMergePlansEmptyPlans verifies error handling for empty plan lists
func TestMergePlansEmptyPlans(t *testing.T) {
	merged, err := MergePlans()
	if err == nil {
		t.Error("expected error for empty plans, got nil")
	}
	if merged != nil {
		t.Error("expected nil merged plan")
	}
}

// TestMergePlansNilPlans verifies handling of nil plans in list
func TestMergePlansNilPlans(t *testing.T) {
	plan1 := &models.Plan{
		Name:  "Plan 1",
		Tasks: []models.Task{{Number: "1", Name: "Task 1"}},
		Waves: []models.Wave{},
	}

	merged, err := MergePlans(plan1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(merged.Tasks) != 1 {
		t.Errorf("expected 1 task after filtering nil, got %d", len(merged.Tasks))
	}
}

// TestMergePlansWithDefaultAgent tests inheriting first plan's defaults
func TestMergePlansWithDefaultAgent(t *testing.T) {
	plan1 := &models.Plan{
		Name:         "Plan 1",
		FilePath:     "plan1.yaml",
		DefaultAgent: "golang-pro",
		Tasks:        []models.Task{{Number: "1", Name: "Task 1"}},
	}

	plan2 := &models.Plan{
		Name:         "Plan 2",
		FilePath:     "plan2.yaml",
		DefaultAgent: "python-pro",
		Tasks:        []models.Task{{Number: "2", Name: "Task 2"}},
	}

	merged, err := MergePlans(plan1, plan2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if merged.DefaultAgent != "golang-pro" {
		t.Errorf("expected DefaultAgent=golang-pro, got %s", merged.DefaultAgent)
	}
}

// TestOrchestratorFileToTaskMapping verifies file-to-task mapping population
func TestOrchestratorFileToTaskMapping(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			return []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
				{Task: models.Task{Number: "2"}, Status: models.StatusGreen},
			}, nil
		},
	}

	plan1 := &models.Plan{
		Name:          "Part 1",
		FilePath:      "part1.yaml",
		Tasks:         []models.Task{{Number: "1", Name: "Task 1"}},
		FileToTaskMap: map[string][]string{"part1.yaml": {"1"}},
	}

	plan2 := &models.Plan{
		Name:          "Part 2",
		FilePath:      "part2.yaml",
		Tasks:         []models.Task{{Number: "2", Name: "Task 2"}},
		FileToTaskMap: map[string][]string{"part2.yaml": {"2"}},
	}

	orch := NewOrchestrator(mockWave, nil)
	result, err := orch.ExecutePlan(context.Background(), plan1, plan2)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file mappings were populated
	if orch.FileToTaskMapping["1"] != "part1.yaml" {
		t.Errorf("expected task 1 to map to part1.yaml, got %s", orch.FileToTaskMapping["1"])
	}
	if orch.FileToTaskMapping["2"] != "part2.yaml" {
		t.Errorf("expected task 2 to map to part2.yaml, got %s", orch.FileToTaskMapping["2"])
	}

	if result.Completed != 2 {
		t.Errorf("expected 2 completed tasks, got %d", result.Completed)
	}
}

// TestOrchestratorMultiplePlansExecution tests full execution with multiple merged plans
func TestOrchestratorMultiplePlansExecution(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			var results []models.TaskResult
			for _, task := range plan.Tasks {
				results = append(results, models.TaskResult{
					Task:   task,
					Status: models.StatusGreen,
				})
			}
			return results, nil
		},
	}

	plan1 := &models.Plan{
		Name:  "Plan 1",
		Tasks: []models.Task{{Number: "1", Name: "Task 1"}},
		Waves: []models.Wave{{Name: "Wave 1", TaskNumbers: []string{"1"}}},
	}

	plan2 := &models.Plan{
		Name:  "Plan 2",
		Tasks: []models.Task{{Number: "2", Name: "Task 2", DependsOn: []string{"1"}}},
		Waves: []models.Wave{{Name: "Wave 2", TaskNumbers: []string{"2"}}},
	}

	orch := NewOrchestrator(mockWave, nil)
	result, err := orch.ExecutePlan(context.Background(), plan1, plan2)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalTasks != 2 {
		t.Errorf("expected 2 total tasks, got %d", result.TotalTasks)
	}
	if result.Completed != 2 {
		t.Errorf("expected 2 completed tasks, got %d", result.Completed)
	}
}

// TestOrchestratorCrossFileDependencies tests dependencies spanning multiple files
func TestOrchestratorCrossFileDependencies(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			// Verify that cross-file dependencies are preserved
			for _, task := range plan.Tasks {
				if task.Number == "3" && len(task.DependsOn) != 2 {
					t.Errorf("task 3 should have 2 dependencies, got %d", len(task.DependsOn))
				}
			}

			var results []models.TaskResult
			for _, task := range plan.Tasks {
				results = append(results, models.TaskResult{
					Task:   task,
					Status: models.StatusGreen,
				})
			}
			return results, nil
		},
	}

	plan1 := &models.Plan{
		Name:     "Part 1",
		FilePath: "part1.yaml",
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1"},
			{Number: "2", Name: "Task 2", DependsOn: []string{"1"}},
		},
	}

	plan2 := &models.Plan{
		Name:     "Part 2",
		FilePath: "part2.yaml",
		Tasks: []models.Task{
			{Number: "3", Name: "Task 3", DependsOn: []string{"1", "2"}}, // Cross-file deps
		},
	}

	orch := NewOrchestrator(mockWave, nil)
	result, err := orch.ExecutePlan(context.Background(), plan1, plan2)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalTasks != 3 {
		t.Errorf("expected 3 tasks with cross-file deps, got %d", result.TotalTasks)
	}
}

// TestOrchestratorWithWorktreeGroups tests execution with worktree group information
func TestOrchestratorWithWorktreeGroups(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			// Verify groups are preserved in merged plan
			if len(plan.WorktreeGroups) != 2 {
				t.Errorf("expected 2 worktree groups, got %d", len(plan.WorktreeGroups))
			}

			var results []models.TaskResult
			for _, task := range plan.Tasks {
				results = append(results, models.TaskResult{
					Task:   task,
					Status: models.StatusGreen,
				})
			}
			return results, nil
		},
	}

	plan1 := &models.Plan{
		Name:     "Part 1",
		FilePath: "part1.yaml",
		Tasks:    []models.Task{{Number: "1", Name: "Task 1"}},
		WorktreeGroups: []models.WorktreeGroup{
			{GroupID: "group-a", ExecutionModel: "sequential"},
		},
	}

	plan2 := &models.Plan{
		Name:     "Part 2",
		FilePath: "part2.yaml",
		Tasks:    []models.Task{{Number: "2", Name: "Task 2"}},
		WorktreeGroups: []models.WorktreeGroup{
			{GroupID: "group-b", ExecutionModel: "parallel"},
		},
	}

	orch := NewOrchestrator(mockWave, nil)
	result, err := orch.ExecutePlan(context.Background(), plan1, plan2)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Completed != 2 {
		t.Errorf("expected 2 completed tasks, got %d", result.Completed)
	}
}

// TestOrchestratorSkipCompleted tests skipping already-completed tasks in merged plans
func TestOrchestratorSkipCompleted(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			var results []models.TaskResult
			for _, task := range plan.Tasks {
				results = append(results, models.TaskResult{
					Task:   task,
					Status: models.StatusGreen,
				})
			}
			return results, nil
		},
	}

	now := time.Now()
	plan1 := &models.Plan{
		Name:     "Part 1",
		FilePath: "part1.yaml",
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", Status: models.StatusGreen, CompletedAt: &now},
			{Number: "2", Name: "Task 2"},
		},
	}

	plan2 := &models.Plan{
		Name:     "Part 2",
		FilePath: "part2.yaml",
		Tasks:    []models.Task{{Number: "3", Name: "Task 3"}},
	}

	orch := NewOrchestratorWithConfig(mockWave, nil, true, false)
	result, err := orch.ExecutePlan(context.Background(), plan1, plan2)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalTasks != 3 {
		t.Errorf("expected 3 total tasks, got %d", result.TotalTasks)
	}
}

// TestMergePlansSinglePlan verifies single plan is returned unchanged
func TestMergePlansSinglePlan(t *testing.T) {
	plan := &models.Plan{
		Name:  "Single Plan",
		Tasks: []models.Task{{Number: "1", Name: "Task 1"}},
	}

	merged, err := MergePlans(plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if merged != plan {
		t.Error("expected single plan to be returned unchanged")
	}
}

// TestOrchestratorExecutePlanLarge tests execution with many merged plans
func TestOrchestratorExecutePlanLarge(t *testing.T) {
	// Create 5 plan parts with cross-file dependencies
	plans := make([]*models.Plan, 5)

	for i := 0; i < 5; i++ {
		tasks := make([]models.Task, 3)
		base := i * 3

		for j := 0; j < 3; j++ {
			taskNum := fmt.Sprintf("%d", base+j+1)
			tasks[j] = models.Task{
				Number: taskNum,
				Name:   fmt.Sprintf("Task %s", taskNum),
			}

			// Add dependencies to previous part's last task
			if i > 0 && j == 0 {
				tasks[j].DependsOn = []string{fmt.Sprintf("%d", base)}
			}
		}

		plans[i] = &models.Plan{
			Name:     fmt.Sprintf("Part %d", i+1),
			FilePath: fmt.Sprintf("part%d.yaml", i+1),
			Tasks:    tasks,
		}
	}

	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			var results []models.TaskResult
			for _, task := range plan.Tasks {
				results = append(results, models.TaskResult{
					Task:   task,
					Status: models.StatusGreen,
				})
			}
			return results, nil
		},
	}

	orch := NewOrchestrator(mockWave, nil)
	result, err := orch.ExecutePlan(context.Background(), plans...)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalTasks != 15 {
		t.Errorf("expected 15 total tasks, got %d", result.TotalTasks)
	}
	if result.Completed != 15 {
		t.Errorf("expected 15 completed tasks, got %d", result.Completed)
	}
}

// TestOrchestratorWithQualityControl tests QC config inheritance in merged plans
func TestOrchestratorWithQualityControl(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			// Verify QC config is preserved
			if !plan.QualityControl.Enabled {
				t.Error("expected QC to be enabled in merged plan")
			}
			if len(plan.QualityControl.Agents.ExplicitList) == 0 || plan.QualityControl.Agents.ExplicitList[0] != "quality-control" {
				t.Errorf("expected QC agent to be quality-control, got %v", plan.QualityControl.Agents.ExplicitList)
			}

			var results []models.TaskResult
			for _, task := range plan.Tasks {
				results = append(results, models.TaskResult{
					Task:   task,
					Status: models.StatusGreen,
				})
			}
			return results, nil
		},
	}

	plan1 := &models.Plan{
		Name:  "Plan 1",
		Tasks: []models.Task{{Number: "1", Name: "Task 1"}},
		QualityControl: models.QualityControlConfig{
			Enabled: true,
			Agents: models.QCAgentConfig{
				Mode:         "explicit",
				ExplicitList: []string{"quality-control"},
			},
			RetryOnRed: 2,
		},
	}

	plan2 := &models.Plan{
		Name:  "Plan 2",
		Tasks: []models.Task{{Number: "2", Name: "Task 2"}},
	}

	orch := NewOrchestrator(mockWave, nil)
	result, err := orch.ExecutePlan(context.Background(), plan1, plan2)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Completed != 2 {
		t.Errorf("expected 2 completed tasks, got %d", result.Completed)
	}
}

// TestMergePlansEmptyTaskList tests merging when all plans have no tasks
func TestMergePlansEmptyTaskList(t *testing.T) {
	plan1 := &models.Plan{
		Name:     "Plan 1",
		FilePath: "plan1.yaml",
		Tasks:    []models.Task{},
	}

	plan2 := &models.Plan{
		Name:     "Plan 2",
		FilePath: "plan2.yaml",
		Tasks:    []models.Task{},
	}

	_, err := MergePlans(plan1, plan2)
	if err == nil {
		t.Error("expected error for merged plan with no tasks")
	}
}

// TestMergePlansLargeScaleStress tests merging many plans with many tasks
func TestMergePlansLargeScaleStress(t *testing.T) {
	const numPlans = 10
	const tasksPerPlan = 20

	plans := make([]*models.Plan, numPlans)

	taskCounter := 1
	for p := 0; p < numPlans; p++ {
		tasks := make([]models.Task, tasksPerPlan)
		for j := 0; j < tasksPerPlan; j++ {
			taskNum := fmt.Sprintf("%d", taskCounter)
			tasks[j] = models.Task{
				Number: taskNum,
				Name:   fmt.Sprintf("Task %d", taskCounter),
			}

			// Add cross-plan dependency for some tasks
			if taskCounter > tasksPerPlan {
				tasks[j].DependsOn = []string{fmt.Sprintf("%d", taskCounter-tasksPerPlan)}
			}
			taskCounter++
		}

		plans[p] = &models.Plan{
			Name:     fmt.Sprintf("Plan %d", p+1),
			FilePath: fmt.Sprintf("plan%d.yaml", p+1),
			Tasks:    tasks,
		}
	}

	merged, err := MergePlans(plans...)
	if err != nil {
		t.Fatalf("unexpected error in large scale merge: %v", err)
	}

	if len(merged.Tasks) != numPlans*tasksPerPlan {
		t.Errorf("expected %d tasks, got %d", numPlans*tasksPerPlan, len(merged.Tasks))
	}

	if len(merged.FileToTaskMap) != numPlans {
		t.Errorf("expected %d file mappings, got %d", numPlans, len(merged.FileToTaskMap))
	}
}

// TestMergePlansComplexDependencyGraph tests realistic complex dependency graphs
func TestMergePlansComplexDependencyGraph(t *testing.T) {
	// Simulates real-world microservices architecture:
	// Part 1: Core infrastructure
	plan1 := &models.Plan{
		Name:     "Infrastructure (Part 1)",
		FilePath: "1-infrastructure.yaml",
		Tasks: []models.Task{
			{Number: "1", Name: "Database Setup"},
			{Number: "2", Name: "Cache Setup", DependsOn: []string{"1"}},
			{Number: "3", Name: "Message Queue", DependsOn: []string{"1"}},
		},
	}

	// Part 2: Core services
	plan2 := &models.Plan{
		Name:     "Core Services (Part 2)",
		FilePath: "2-core-services.yaml",
		Tasks: []models.Task{
			{Number: "4", Name: "User Service", DependsOn: []string{"1"}},
			{Number: "5", Name: "Auth Service", DependsOn: []string{"4"}},
			{Number: "6", Name: "Product Service", DependsOn: []string{"1"}},
		},
	}

	// Part 3: Integration
	plan3 := &models.Plan{
		Name:     "Integration (Part 3)",
		FilePath: "3-integration.yaml",
		Tasks: []models.Task{
			{Number: "7", Name: "API Gateway", DependsOn: []string{"2", "3", "4", "5", "6"}},
			{Number: "8", Name: "Load Balancer", DependsOn: []string{"7"}},
			{Number: "9", Name: "Tests", DependsOn: []string{"8"}},
		},
	}

	merged, err := MergePlans(plan1, plan2, plan3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(merged.Tasks) != 9 {
		t.Errorf("expected 9 tasks, got %d", len(merged.Tasks))
	}

	// Verify critical task has correct dependencies
	var task7 *models.Task
	for i := range merged.Tasks {
		if merged.Tasks[i].Number == "7" {
			task7 = &merged.Tasks[i]
			break
		}
	}

	if task7 == nil {
		t.Fatal("task 7 not found")
	}

	if len(task7.DependsOn) != 5 {
		t.Errorf("task 7 should have 5 dependencies, got %d", len(task7.DependsOn))
	}
}

// TestOrchestratorBackwardCompatibility tests that single-file plans still work
func TestOrchestratorBackwardCompatibility(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			var results []models.TaskResult
			for _, task := range plan.Tasks {
				results = append(results, models.TaskResult{
					Task:   task,
					Status: models.StatusGreen,
				})
			}
			return results, nil
		},
	}

	// V1-style single plan
	plan := &models.Plan{
		Name: "Single File Plan",
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1"},
			{Number: "2", Name: "Task 2", DependsOn: []string{"1"}},
			{Number: "3", Name: "Task 3", DependsOn: []string{"2"}},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []string{"1"}},
			{Name: "Wave 2", TaskNumbers: []string{"2"}},
			{Name: "Wave 3", TaskNumbers: []string{"3"}},
		},
	}

	orch := NewOrchestrator(mockWave, nil)
	result, err := orch.ExecutePlan(context.Background(), plan)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalTasks != 3 {
		t.Errorf("expected 3 total tasks, got %d", result.TotalTasks)
	}

	if result.Completed != 3 {
		t.Errorf("expected 3 completed tasks, got %d", result.Completed)
	}
}

// TestMergePlansWithNilFileToTaskMap tests handling of nil FileToTaskMap
func TestMergePlansWithNilFileToTaskMap(t *testing.T) {
	plan1 := &models.Plan{
		Name:          "Plan 1",
		FilePath:      "plan1.yaml",
		Tasks:         []models.Task{{Number: "1", Name: "Task 1"}},
		FileToTaskMap: nil,
	}

	plan2 := &models.Plan{
		Name:          "Plan 2",
		FilePath:      "plan2.yaml",
		Tasks:         []models.Task{{Number: "2", Name: "Task 2"}},
		FileToTaskMap: map[string][]string{"plan2.yaml": {"2"}},
	}

	merged, err := MergePlans(plan1, plan2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Merge always populates FileToTaskMap from FilePath, so both should be present
	if len(merged.FileToTaskMap) != 2 {
		t.Errorf("expected 2 file mappings after merge, got %d", len(merged.FileToTaskMap))
	}

	// Verify both files are in the mapping
	if _, hasFile1 := merged.FileToTaskMap["plan1.yaml"]; !hasFile1 {
		t.Error("expected plan1.yaml in FileToTaskMap")
	}
	if _, hasFile2 := merged.FileToTaskMap["plan2.yaml"]; !hasFile2 {
		t.Error("expected plan2.yaml in FileToTaskMap")
	}
}

// TestOrchestratorWithFailedTasksMerged tests result aggregation with mixed status
func TestOrchestratorWithFailedTasksMerged(t *testing.T) {
	mockWave := &mockWaveExecutor{
		executePlanFunc: func(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
			results := []models.TaskResult{
				{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
				{Task: models.Task{Number: "2"}, Status: models.StatusRed, Error: fmt.Errorf("task failed")},
				{Task: models.Task{Number: "3"}, Status: models.StatusYellow},
				{Task: models.Task{Number: "4"}, Status: models.StatusFailed},
			}
			return results, nil
		},
	}

	plan1 := &models.Plan{
		Name:  "Part 1",
		Tasks: []models.Task{{Number: "1"}, {Number: "2"}},
	}

	plan2 := &models.Plan{
		Name:  "Part 2",
		Tasks: []models.Task{{Number: "3"}, {Number: "4"}},
	}

	orch := NewOrchestrator(mockWave, nil)
	result, err := orch.ExecutePlan(context.Background(), plan1, plan2)

	if err != nil {
		// Error from wave execution is expected
	}

	if result.TotalTasks != 4 {
		t.Errorf("expected 4 total tasks, got %d", result.TotalTasks)
	}

	// GREEN (1) + YELLOW (1) = 2 completed
	if result.Completed != 2 {
		t.Errorf("expected 2 completed (GREEN+YELLOW), got %d", result.Completed)
	}

	// RED (1) + FAILED (1) = 2 failed
	if result.Failed != 2 {
		t.Errorf("expected 2 failed, got %d", result.Failed)
	}
}
