package models

import (
	"testing"
	"time"
)

func TestTaskValidation(t *testing.T) {
	tests := []struct {
		name    string
		task    Task
		wantErr bool
	}{
		{
			name: "valid task",
			task: Task{
				Number: "1",
				Name:   "Test Task",
				Prompt: "Do something",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			task: Task{
				Number: "1",
				Prompt: "Do something",
			},
			wantErr: true,
		},
		{
			name: "missing prompt",
			task: Task{
				Number: "1",
				Name:   "Test Task",
			},
			wantErr: true,
		},
		{
			name: "empty number string",
			task: Task{
				Number: "",
				Name:   "Test Task",
				Prompt: "Do something",
			},
			wantErr: true,
		},
		{
			name: "valid task with optional fields",
			task: Task{
				Number:        "1",
				Name:          "Test Task",
				Prompt:        "Do something",
				Files:         []string{"file1.go", "file2.go"},
				DependsOn:     []string{},
				EstimatedTime: 30 * time.Minute,
				Agent:         "test-agent",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Task.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDetectCycles(t *testing.T) {
	tests := []struct {
		name      string
		tasks     []Task
		wantCycle bool
	}{
		{
			name: "no cycle - linear dependency",
			tasks: []Task{
				{Number: "1", Name: "Task 1", Prompt: "test", DependsOn: []string{}},
				{Number: "2", Name: "Task 2", Prompt: "test", DependsOn: []string{"1"}},
			},
			wantCycle: false,
		},
		{
			name: "simple cycle",
			tasks: []Task{
				{Number: "1", Name: "Task 1", Prompt: "test", DependsOn: []string{"2"}},
				{Number: "2", Name: "Task 2", Prompt: "test", DependsOn: []string{"1"}},
			},
			wantCycle: true,
		},
		{
			name: "self reference",
			tasks: []Task{
				{Number: "1", Name: "Task 1", Prompt: "test", DependsOn: []string{"1"}},
			},
			wantCycle: true,
		},
		{
			name: "no cycle - multiple dependencies",
			tasks: []Task{
				{Number: "1", Name: "Task 1", Prompt: "test", DependsOn: []string{}},
				{Number: "2", Name: "Task 2", Prompt: "test", DependsOn: []string{"1"}},
				{Number: "3", Name: "Task 3", Prompt: "test", DependsOn: []string{"1"}},
				{Number: "4", Name: "Task 4", Prompt: "test", DependsOn: []string{"2", "3"}},
			},
			wantCycle: false,
		},
		{
			name: "cycle in chain",
			tasks: []Task{
				{Number: "1", Name: "Task 1", Prompt: "test", DependsOn: []string{}},
				{Number: "2", Name: "Task 2", Prompt: "test", DependsOn: []string{"1"}},
				{Number: "3", Name: "Task 3", Prompt: "test", DependsOn: []string{"2"}},
				{Number: "4", Name: "Task 4", Prompt: "test", DependsOn: []string{"3"}},
				{Number: "1", Name: "Task 1", Prompt: "test", DependsOn: []string{"4"}}, // This creates cycle
			},
			wantCycle: true,
		},
		{
			name:      "empty tasks",
			tasks:     []Task{},
			wantCycle: false,
		},
		{
			name: "no dependencies",
			tasks: []Task{
				{Number: "1", Name: "Task 1", Prompt: "test", DependsOn: []string{}},
				{Number: "2", Name: "Task 2", Prompt: "test", DependsOn: []string{}},
				{Number: "3", Name: "Task 3", Prompt: "test", DependsOn: []string{}},
			},
			wantCycle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasCycle := HasCyclicDependencies(tt.tasks)
			if hasCycle != tt.wantCycle {
				t.Errorf("HasCyclicDependencies() = %v, want %v", hasCycle, tt.wantCycle)
			}
		})
	}
}

func TestWaveCalculation(t *testing.T) {
	// This test verifies the basic structures needed for wave calculation
	// The actual wave calculation logic will be in the executor package

	t.Run("wave struct has required fields", func(t *testing.T) {
		wave := Wave{
			Name:           "Wave 1",
			TaskNumbers:    []string{"1", "2", "3"},
			MaxConcurrency: 5,
		}

		if wave.Name != "Wave 1" {
			t.Errorf("Wave.Name = %v, want Wave 1", wave.Name)
		}
		if len(wave.TaskNumbers) != 3 {
			t.Errorf("len(Wave.TaskNumbers) = %v, want 3", len(wave.TaskNumbers))
		}
		if wave.MaxConcurrency != 5 {
			t.Errorf("Wave.MaxConcurrency = %v, want 5", wave.MaxConcurrency)
		}
	})

	t.Run("plan can hold multiple waves", func(t *testing.T) {
		plan := Plan{
			Name: "Test Plan",
			Tasks: []Task{
				{Number: "1", Name: "Task 1", Prompt: "test"},
				{Number: "2", Name: "Task 2", Prompt: "test"},
			},
			Waves: []Wave{
				{Name: "Wave 1", TaskNumbers: []string{"1"}},
				{Name: "Wave 2", TaskNumbers: []string{"2"}},
			},
			DefaultAgent: "general",
		}

		if len(plan.Waves) != 2 {
			t.Errorf("len(Plan.Waves) = %v, want 2", len(plan.Waves))
		}
	})
}

func TestQualityControlConfig(t *testing.T) {
	t.Run("quality control config has required fields", func(t *testing.T) {
		qc := QualityControlConfig{
			Enabled: true,
			Agents: QCAgentConfig{
				Mode:         "explicit",
				ExplicitList: []string{"reviewer"},
			},
			RetryOnRed: 3,
		}

		if !qc.Enabled {
			t.Error("QualityControlConfig.Enabled should be true")
		}
		if len(qc.Agents.ExplicitList) == 0 || qc.Agents.ExplicitList[0] != "reviewer" {
			t.Errorf("QualityControlConfig.Agents.ExplicitList = %v, want [reviewer]", qc.Agents.ExplicitList)
		}
		if qc.RetryOnRed != 3 {
			t.Errorf("QualityControlConfig.RetryOnRed = %v, want 3", qc.RetryOnRed)
		}
	})
}

func TestTaskResult(t *testing.T) {
	t.Run("task result has required fields", func(t *testing.T) {
		task := Task{Number: "1", Name: "Test", Prompt: "test"}
		result := TaskResult{
			Task:           task,
			Status:         "GREEN",
			Output:         "test output",
			Error:          nil,
			Duration:       5 * time.Second,
			RetryCount:     0,
			ReviewFeedback: "Looks good",
		}

		if result.Status != "GREEN" {
			t.Errorf("TaskResult.Status = %v, want GREEN", result.Status)
		}
		if result.Duration != 5*time.Second {
			t.Errorf("TaskResult.Duration = %v, want 5s", result.Duration)
		}
	})
}

func TestExecutionResult(t *testing.T) {
	t.Run("execution result aggregates task results", func(t *testing.T) {
		result := ExecutionResult{
			TotalTasks:  10,
			Completed:   8,
			Failed:      2,
			Duration:    30 * time.Minute,
			FailedTasks: []TaskResult{},
		}

		if result.TotalTasks != 10 {
			t.Errorf("ExecutionResult.TotalTasks = %v, want 10", result.TotalTasks)
		}
		if result.Completed != 8 {
			t.Errorf("ExecutionResult.Completed = %v, want 8", result.Completed)
		}
		if result.Failed != 2 {
			t.Errorf("ExecutionResult.Failed = %v, want 2", result.Failed)
		}
	})
}

func TestTaskWorktreeGroup(t *testing.T) {
	tests := []struct {
		name             string
		task             Task
		expectedWorktree string
	}{
		{
			name: "task with worktree group",
			task: Task{
				Number:        "1",
				Name:          "Test Task",
				Prompt:        "Do something",
				WorktreeGroup: "frontend",
			},
			expectedWorktree: "frontend",
		},
		{
			name: "task without worktree group",
			task: Task{
				Number:        "2",
				Name:          "Backend Task",
				Prompt:        "Backend work",
				WorktreeGroup: "",
			},
			expectedWorktree: "",
		},
		{
			name: "task with complex worktree group name",
			task: Task{
				Number:        "3",
				Name:          "Complex Task",
				Prompt:        "Do work",
				WorktreeGroup: "group-v1-2-3",
			},
			expectedWorktree: "group-v1-2-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.task.WorktreeGroup != tt.expectedWorktree {
				t.Errorf("Task.WorktreeGroup = %v, want %v", tt.task.WorktreeGroup, tt.expectedWorktree)
			}
		})
	}
}

func TestPlanWorktreeGroups(t *testing.T) {
	tests := []struct {
		name             string
		plan             Plan
		expectedGroupLen int
		expectedGroupIDs []string
	}{
		{
			name: "plan with worktree groups",
			plan: Plan{
				Name: "Test Plan",
				WorktreeGroups: []WorktreeGroup{
					{
						GroupID:        "frontend",
						Description:    "Frontend tasks",
						ExecutionModel: "parallel",
						Isolation:      "strong",
						Rationale:      "UI components need isolation",
					},
					{
						GroupID:        "backend",
						Description:    "Backend tasks",
						ExecutionModel: "sequential",
						Isolation:      "weak",
						Rationale:      "Database consistency",
					},
				},
			},
			expectedGroupLen: 2,
			expectedGroupIDs: []string{"frontend", "backend"},
		},
		{
			name: "plan with empty worktree groups",
			plan: Plan{
				Name:           "Empty Plan",
				WorktreeGroups: []WorktreeGroup{},
			},
			expectedGroupLen: 0,
			expectedGroupIDs: []string{},
		},
		{
			name: "plan with single worktree group",
			plan: Plan{
				Name: "Single Group Plan",
				WorktreeGroups: []WorktreeGroup{
					{
						GroupID:        "unified",
						Description:    "All tasks run together",
						ExecutionModel: "parallel",
						Isolation:      "none",
						Rationale:      "Monolithic application",
					},
				},
			},
			expectedGroupLen: 1,
			expectedGroupIDs: []string{"unified"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.plan.WorktreeGroups) != tt.expectedGroupLen {
				t.Errorf("len(Plan.WorktreeGroups) = %v, want %v", len(tt.plan.WorktreeGroups), tt.expectedGroupLen)
			}

			for i, expectedID := range tt.expectedGroupIDs {
				if i >= len(tt.plan.WorktreeGroups) {
					t.Fatalf("WorktreeGroups index out of range: %d", i)
				}
				if tt.plan.WorktreeGroups[i].GroupID != expectedID {
					t.Errorf("WorktreeGroups[%d].GroupID = %v, want %v", i, tt.plan.WorktreeGroups[i].GroupID, expectedID)
				}
			}
		})
	}
}

func TestWorktreeGroupMetadata(t *testing.T) {
	t.Run("worktree group is serializable", func(t *testing.T) {
		group := WorktreeGroup{
			GroupID:        "test-group",
			Description:    "A test group",
			ExecutionModel: "parallel",
			Isolation:      "strong",
			Rationale:      "Testing isolation",
		}

		if group.GroupID != "test-group" {
			t.Errorf("WorktreeGroup.GroupID = %v, want test-group", group.GroupID)
		}
		if group.Description != "A test group" {
			t.Errorf("WorktreeGroup.Description = %v, want A test group", group.Description)
		}
		if group.ExecutionModel != "parallel" {
			t.Errorf("WorktreeGroup.ExecutionModel = %v, want parallel", group.ExecutionModel)
		}
		if group.Isolation != "strong" {
			t.Errorf("WorktreeGroup.Isolation = %v, want strong", group.Isolation)
		}
		if group.Rationale != "Testing isolation" {
			t.Errorf("WorktreeGroup.Rationale = %v, want Testing isolation", group.Rationale)
		}
	})
}

func TestTask_StatusFields(t *testing.T) {
	tests := []struct {
		name      string
		task      Task
		wantError bool
	}{
		{
			name: "task with status field",
			task: Task{
				Number: "1",
				Name:   "Test Task",
				Prompt: "Do something",
				Status: "pending",
			},
			wantError: false,
		},
		{
			name: "task with completed status and time",
			task: Task{
				Number:      "1",
				Name:        "Test Task",
				Prompt:      "Do something",
				Status:      "completed",
				CompletedAt: timePtr(time.Now()),
			},
			wantError: false,
		},
		{
			name: "task with in_progress status",
			task: Task{
				Number: "1",
				Name:   "Test Task",
				Prompt: "Do something",
				Status: "in_progress",
			},
			wantError: false,
		},
		{
			name: "task with skipped status",
			task: Task{
				Number: "1",
				Name:   "Test Task",
				Prompt: "Do something",
				Status: "skipped",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.task.Status == "" && tt.task.CompletedAt != nil {
				t.Error("task has CompletedAt but empty Status")
			}
			if tt.task.Status == "completed" && tt.task.CompletedAt == nil {
				t.Error("task has completed status but no CompletedAt time")
			}
		})
	}
}

func TestTask_IsCompleted(t *testing.T) {
	tests := []struct {
		name     string
		task     Task
		expected bool
	}{
		{
			name:     "task with completed status",
			task:     Task{Number: "1", Name: "Test", Prompt: "test", Status: "completed"},
			expected: true,
		},
		{
			name:     "task with pending status",
			task:     Task{Number: "1", Name: "Test", Prompt: "test", Status: "pending"},
			expected: false,
		},
		{
			name:     "task with in_progress status",
			task:     Task{Number: "1", Name: "Test", Prompt: "test", Status: "in_progress"},
			expected: false,
		},
		{
			name:     "task with skipped status",
			task:     Task{Number: "1", Name: "Test", Prompt: "test", Status: "skipped"},
			expected: false,
		},
		{
			name:     "task with empty status",
			task:     Task{Number: "1", Name: "Test", Prompt: "test", Status: ""},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.task.IsCompleted()
			if result != tt.expected {
				t.Errorf("Task.IsCompleted() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTask_CanSkip(t *testing.T) {
	tests := []struct {
		name     string
		task     Task
		expected bool
	}{
		{
			name:     "completed task can be skipped",
			task:     Task{Number: "1", Name: "Test", Prompt: "test", Status: "completed"},
			expected: true,
		},
		{
			name:     "skipped task can be skipped",
			task:     Task{Number: "1", Name: "Test", Prompt: "test", Status: "skipped"},
			expected: true,
		},
		{
			name:     "pending task cannot be skipped",
			task:     Task{Number: "1", Name: "Test", Prompt: "test", Status: "pending"},
			expected: false,
		},
		{
			name:     "in_progress task cannot be skipped",
			task:     Task{Number: "1", Name: "Test", Prompt: "test", Status: "in_progress"},
			expected: false,
		},
		{
			name:     "empty status cannot be skipped",
			task:     Task{Number: "1", Name: "Test", Prompt: "test", Status: ""},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.task.CanSkip()
			if result != tt.expected {
				t.Errorf("Task.CanSkip() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Helper function for tests
func timePtr(t time.Time) *time.Time {
	return &t
}

// TestWorktreeGroupValidation tests worktree group structure and validation
func TestWorktreeGroupValidation(t *testing.T) {
	tests := []struct {
		name  string
		group WorktreeGroup
		valid bool
	}{
		{
			name: "valid group with parallel execution",
			group: WorktreeGroup{
				GroupID:        "group-a",
				Description:    "Parallel execution group",
				ExecutionModel: "parallel",
				Isolation:      "weak",
			},
			valid: true,
		},
		{
			name: "valid group with sequential execution",
			group: WorktreeGroup{
				GroupID:        "group-b",
				Description:    "Sequential execution group",
				ExecutionModel: "sequential",
				Isolation:      "strong",
			},
			valid: true,
		},
		{
			name: "group with no isolation",
			group: WorktreeGroup{
				GroupID:        "group-c",
				Description:    "No isolation",
				ExecutionModel: "parallel",
				Isolation:      "none",
			},
			valid: true,
		},
		{
			name: "missing group ID",
			group: WorktreeGroup{
				GroupID:        "",
				Description:    "Missing ID",
				ExecutionModel: "parallel",
			},
			valid: false, // Would fail in real validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - GroupID should not be empty
			if tt.group.GroupID == "" && tt.valid {
				t.Error("group ID should not be empty for valid groups")
			}

			// Verify execution model is one of known values
			if tt.group.ExecutionModel != "parallel" && tt.group.ExecutionModel != "sequential" {
				t.Errorf("unknown execution model: %s", tt.group.ExecutionModel)
			}

			// Verify isolation level is one of known values
			validIsolations := map[string]bool{"none": true, "weak": true, "strong": true}
			if tt.group.Isolation != "" && !validIsolations[tt.group.Isolation] {
				t.Errorf("unknown isolation level: %s", tt.group.Isolation)
			}
		})
	}
}

// TestFileToTaskMapping tests file-to-task mapping structure
func TestFileToTaskMapping(t *testing.T) {
	tests := []struct {
		name     string
		mapping  map[string][]string
		validate func(map[string][]string) bool
	}{
		{
			name:    "empty mapping",
			mapping: map[string][]string{},
			validate: func(m map[string][]string) bool {
				return len(m) == 0
			},
		},
		{
			name: "single file with multiple tasks",
			mapping: map[string][]string{
				"part1.yaml": {"1", "2", "3"},
			},
			validate: func(m map[string][]string) bool {
				return len(m) == 1 && len(m["part1.yaml"]) == 3
			},
		},
		{
			name: "multiple files with mixed task counts",
			mapping: map[string][]string{
				"part1.yaml": {"1", "2"},
				"part2.yaml": {"3", "4", "5"},
				"part3.yaml": {"6"},
			},
			validate: func(m map[string][]string) bool {
				return len(m) == 3 && len(m["part1.yaml"]) == 2 && len(m["part2.yaml"]) == 3 && len(m["part3.yaml"]) == 1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.validate(tt.mapping) {
				t.Error("mapping validation failed")
			}
		})
	}
}

// TestCrossFileDependencies tests handling of dependencies spanning files
func TestCrossFileDependencies(t *testing.T) {
	tasks := []Task{
		// Part 1 tasks
		{Number: "1", Name: "Task 1", DependsOn: []string{}},
		{Number: "2", Name: "Task 2", DependsOn: []string{"1"}},
		// Part 2 tasks
		{Number: "3", Name: "Task 3", DependsOn: []string{"2"}},      // Cross-file dep
		{Number: "4", Name: "Task 4", DependsOn: []string{"1", "3"}}, // Multiple cross-file deps
		// Part 3 tasks
		{Number: "5", Name: "Task 5", DependsOn: []string{"3", "4"}}, // Multiple cross-file deps
	}

	tests := []struct {
		name      string
		taskNum   string
		expectDep int
	}{
		{
			name:      "task 1 has no deps",
			taskNum:   "1",
			expectDep: 0,
		},
		{
			name:      "task 3 has cross-file dep",
			taskNum:   "3",
			expectDep: 1,
		},
		{
			name:      "task 4 has multiple cross-file deps",
			taskNum:   "4",
			expectDep: 2,
		},
		{
			name:      "task 5 has cross-file deps from different files",
			taskNum:   "5",
			expectDep: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var found *Task
			for i := range tasks {
				if tasks[i].Number == tt.taskNum {
					found = &tasks[i]
					break
				}
			}

			if found == nil {
				t.Fatalf("task %s not found", tt.taskNum)
			}

			if len(found.DependsOn) != tt.expectDep {
				t.Errorf("task %s expected %d deps, got %d", tt.taskNum, tt.expectDep, len(found.DependsOn))
			}
		})
	}
}

// TestPlanMergeStructure tests merged plan structure preservation
func TestPlanMergeStructure(t *testing.T) {
	tests := []struct {
		name           string
		plans          []Plan
		expectTasks    int
		expectWaves    int
		expectGroups   int
		expectFileMaps int
	}{
		{
			name: "single plan unchanged",
			plans: []Plan{
				{
					Name:           "Plan 1",
					Tasks:          []Task{{Number: "1"}},
					Waves:          []Wave{{Name: "Wave 1"}},
					WorktreeGroups: []WorktreeGroup{{GroupID: "g1"}},
					FileToTaskMap:  map[string][]string{"plan1.yaml": {"1"}},
				},
			},
			expectTasks:    1,
			expectWaves:    1,
			expectGroups:   1,
			expectFileMaps: 1,
		},
		{
			name: "two plans combined",
			plans: []Plan{
				{
					Name:           "Part 1",
					Tasks:          []Task{{Number: "1"}, {Number: "2"}},
					Waves:          []Wave{{Name: "Wave 1"}},
					WorktreeGroups: []WorktreeGroup{{GroupID: "g1"}},
					FileToTaskMap:  map[string][]string{"part1.yaml": {"1", "2"}},
				},
				{
					Name:           "Part 2",
					Tasks:          []Task{{Number: "3"}},
					Waves:          []Wave{{Name: "Wave 2"}},
					WorktreeGroups: []WorktreeGroup{{GroupID: "g2"}},
					FileToTaskMap:  map[string][]string{"part2.yaml": {"3"}},
				},
			},
			expectTasks:    3,
			expectWaves:    2,
			expectGroups:   2,
			expectFileMaps: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulating merge logic
			totalTasks := 0
			totalWaves := 0
			totalGroups := 0
			totalFileMaps := 0

			for _, p := range tt.plans {
				totalTasks += len(p.Tasks)
				totalWaves += len(p.Waves)
				totalGroups += len(p.WorktreeGroups)
				totalFileMaps += len(p.FileToTaskMap)
			}

			if totalTasks != tt.expectTasks {
				t.Errorf("expected %d total tasks, got %d", tt.expectTasks, totalTasks)
			}
			if totalWaves != tt.expectWaves {
				t.Errorf("expected %d total waves, got %d", tt.expectWaves, totalWaves)
			}
			if totalGroups != tt.expectGroups {
				t.Errorf("expected %d total groups, got %d", tt.expectGroups, totalGroups)
			}
			if totalFileMaps != tt.expectFileMaps {
				t.Errorf("expected %d total file maps, got %d", tt.expectFileMaps, totalFileMaps)
			}
		})
	}
}

// TestTaskExecutionMetadata verifies that execution metadata fields are accessible and properly stored
func TestTaskExecutionMetadata(t *testing.T) {
	tests := []struct {
		name     string
		task     Task
		validate func(t *testing.T, task *Task)
	}{
		{
			name: "task with all execution metadata",
			task: Task{
				Number:             "1",
				Name:               "Test Task",
				Prompt:             "Do something",
				ExecutionStartTime: time.Now(),
				ExecutedBy:         "test-agent",
				FilesModified:      3,
				FilesCreated:       2,
				FilesDeleted:       1,
			},
			validate: func(t *testing.T, task *Task) {
				if task.ExecutedBy != "test-agent" {
					t.Errorf("ExecutedBy = %v, want test-agent", task.ExecutedBy)
				}
				if task.FilesModified != 3 {
					t.Errorf("FilesModified = %v, want 3", task.FilesModified)
				}
				if task.FilesCreated != 2 {
					t.Errorf("FilesCreated = %v, want 2", task.FilesCreated)
				}
				if task.FilesDeleted != 1 {
					t.Errorf("FilesDeleted = %v, want 1", task.FilesDeleted)
				}
				if task.ExecutionStartTime.IsZero() {
					t.Error("ExecutionStartTime should not be zero")
				}
			},
		},
		{
			name: "task without execution metadata",
			task: Task{
				Number: "2",
				Name:   "Another Task",
				Prompt: "Do something else",
			},
			validate: func(t *testing.T, task *Task) {
				if task.ExecutedBy != "" {
					t.Errorf("ExecutedBy should be empty, got %v", task.ExecutedBy)
				}
				if task.FilesModified != 0 {
					t.Errorf("FilesModified should be 0, got %v", task.FilesModified)
				}
				if task.FilesCreated != 0 {
					t.Errorf("FilesCreated should be 0, got %v", task.FilesCreated)
				}
				if task.FilesDeleted != 0 {
					t.Errorf("FilesDeleted should be 0, got %v", task.FilesDeleted)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, &tt.task)
		})
	}
}

// TestTaskCalculateDuration verifies duration calculation from start and end times
func TestTaskCalculateDuration(t *testing.T) {
	tests := []struct {
		name             string
		task             Task
		expectedDur      time.Duration
		shouldBeZero     bool
		shouldBeNegative bool
	}{
		{
			name: "normal duration calculation",
			task: Task{
				Number:             "1",
				Name:               "Test Task",
				Prompt:             "Do something",
				ExecutionStartTime: time.Unix(0, 0),
				ExecutionEndTime:   time.Unix(10, 0),
			},
			expectedDur: 10 * time.Second,
		},
		{
			name: "zero duration - same start and end time",
			task: Task{
				Number:             "2",
				Name:               "Instant Task",
				Prompt:             "Do something fast",
				ExecutionStartTime: time.Unix(100, 0),
				ExecutionEndTime:   time.Unix(100, 0),
			},
			shouldBeZero: true,
		},
		{
			name: "negative duration - end before start",
			task: Task{
				Number:             "3",
				Name:               "Invalid Task",
				Prompt:             "Time travel",
				ExecutionStartTime: time.Unix(100, 0),
				ExecutionEndTime:   time.Unix(50, 0),
			},
			shouldBeNegative: true,
		},
		{
			name: "missing start time",
			task: Task{
				Number:           "4",
				Name:             "No Start",
				Prompt:           "Missing start",
				ExecutionEndTime: time.Unix(100, 0),
			},
			shouldBeZero: true,
		},
		{
			name: "missing end time",
			task: Task{
				Number:             "5",
				Name:               "No End",
				Prompt:             "Missing end",
				ExecutionStartTime: time.Unix(0, 0),
			},
			shouldBeZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := tt.task.CalculateDuration()

			if tt.shouldBeZero && duration != 0 {
				t.Errorf("expected zero duration, got %v", duration)
			}
			if tt.shouldBeNegative && duration >= 0 {
				t.Errorf("expected negative duration, got %v", duration)
			}
			if !tt.shouldBeZero && !tt.shouldBeNegative && duration != tt.expectedDur {
				t.Errorf("duration = %v, want %v", duration, tt.expectedDur)
			}
		})
	}
}

// TestTaskExecutionMetadataDefaults verifies new task has zero values for metadata
func TestTaskExecutionMetadataDefaults(t *testing.T) {
	task := Task{
		Number: "1",
		Name:   "New Task",
		Prompt: "Do something",
	}

	if !task.ExecutionStartTime.IsZero() {
		t.Errorf("ExecutionStartTime should be zero, got %v", task.ExecutionStartTime)
	}
	if !task.ExecutionEndTime.IsZero() {
		t.Errorf("ExecutionEndTime should be zero, got %v", task.ExecutionEndTime)
	}
	if task.ExecutionDuration != 0 {
		t.Errorf("ExecutionDuration should be 0, got %v", task.ExecutionDuration)
	}
	if task.ExecutedBy != "" {
		t.Errorf("ExecutedBy should be empty, got %v", task.ExecutedBy)
	}
	if task.FilesModified != 0 {
		t.Errorf("FilesModified should be 0, got %v", task.FilesModified)
	}
	if task.FilesCreated != 0 {
		t.Errorf("FilesCreated should be 0, got %v", task.FilesCreated)
	}
	if task.FilesDeleted != 0 {
		t.Errorf("FilesDeleted should be 0, got %v", task.FilesDeleted)
	}
}

// TestTaskFileOperationCounts verifies file operation tracking and counting
func TestTaskFileOperationCounts(t *testing.T) {
	tests := []struct {
		name             string
		operations       []string
		expectedModified int
		expectedCreated  int
		expectedDeleted  int
		expectedTotal    int
	}{
		{
			name:             "no operations",
			operations:       []string{},
			expectedModified: 0,
			expectedCreated:  0,
			expectedDeleted:  0,
			expectedTotal:    0,
		},
		{
			name:             "single modified",
			operations:       []string{"modified"},
			expectedModified: 1,
			expectedCreated:  0,
			expectedDeleted:  0,
			expectedTotal:    1,
		},
		{
			name:             "single created",
			operations:       []string{"created"},
			expectedModified: 0,
			expectedCreated:  1,
			expectedDeleted:  0,
			expectedTotal:    1,
		},
		{
			name:             "single deleted",
			operations:       []string{"deleted"},
			expectedModified: 0,
			expectedCreated:  0,
			expectedDeleted:  1,
			expectedTotal:    1,
		},
		{
			name:             "multiple operations",
			operations:       []string{"modified", "modified", "created", "deleted", "modified"},
			expectedModified: 3,
			expectedCreated:  1,
			expectedDeleted:  1,
			expectedTotal:    5,
		},
		{
			name:             "unknown operation type",
			operations:       []string{"unknown", "modified"},
			expectedModified: 1,
			expectedCreated:  0,
			expectedDeleted:  0,
			expectedTotal:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := Task{
				Number: "1",
				Name:   "Test Task",
				Prompt: "Do something",
			}

			for _, op := range tt.operations {
				task.RecordFileOperation(op)
			}

			if task.FilesModified != tt.expectedModified {
				t.Errorf("FilesModified = %d, want %d", task.FilesModified, tt.expectedModified)
			}
			if task.FilesCreated != tt.expectedCreated {
				t.Errorf("FilesCreated = %d, want %d", task.FilesCreated, tt.expectedCreated)
			}
			if task.FilesDeleted != tt.expectedDeleted {
				t.Errorf("FilesDeleted = %d, want %d", task.FilesDeleted, tt.expectedDeleted)
			}
			if task.TotalFileOperations() != tt.expectedTotal {
				t.Errorf("TotalFileOperations = %d, want %d", task.TotalFileOperations(), tt.expectedTotal)
			}
		})
	}
}

// TestTaskGetFormattedDuration verifies human-readable duration formatting
func TestTaskGetFormattedDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "zero duration",
			duration: 0,
			expected: "0s",
		},
		{
			name:     "seconds",
			duration: 45 * time.Second,
			expected: "45s",
		},
		{
			name:     "minutes and seconds",
			duration: 2*time.Minute + 30*time.Second,
			expected: "2m30s",
		},
		{
			name:     "hours",
			duration: 1*time.Hour + 15*time.Minute,
			expected: "1h15m0s",
		},
		{
			name:     "milliseconds",
			duration: 500 * time.Millisecond,
			expected: "500ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := Task{
				Number:            "1",
				Name:              "Test Task",
				Prompt:            "Do something",
				ExecutionDuration: tt.duration,
			}
			result := task.GetFormattedDuration()
			if result != tt.expected {
				t.Errorf("GetFormattedDuration = %q, want %q", result, tt.expected)
			}
		})
	}
}
