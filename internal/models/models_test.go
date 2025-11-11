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
			Enabled:     true,
			ReviewAgent: "reviewer",
			RetryOnRed:  3,
		}

		if !qc.Enabled {
			t.Error("QualityControlConfig.Enabled should be true")
		}
		if qc.ReviewAgent != "reviewer" {
			t.Errorf("QualityControlConfig.ReviewAgent = %v, want reviewer", qc.ReviewAgent)
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
		name            string
		task            Task
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
