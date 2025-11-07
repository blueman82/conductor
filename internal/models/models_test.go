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
				Number: 1,
				Name:   "Test Task",
				Prompt: "Do something",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			task: Task{
				Number: 1,
				Prompt: "Do something",
			},
			wantErr: true,
		},
		{
			name: "missing prompt",
			task: Task{
				Number: 1,
				Name:   "Test Task",
			},
			wantErr: true,
		},
		{
			name: "invalid number (zero)",
			task: Task{
				Number: 0,
				Name:   "Test Task",
				Prompt: "Do something",
			},
			wantErr: true,
		},
		{
			name: "invalid number (negative)",
			task: Task{
				Number: -1,
				Name:   "Test Task",
				Prompt: "Do something",
			},
			wantErr: true,
		},
		{
			name: "valid task with optional fields",
			task: Task{
				Number:        1,
				Name:          "Test Task",
				Prompt:        "Do something",
				Files:         []string{"file1.go", "file2.go"},
				DependsOn:     []int{},
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
				{Number: 1, Name: "Task 1", Prompt: "test", DependsOn: []int{}},
				{Number: 2, Name: "Task 2", Prompt: "test", DependsOn: []int{1}},
			},
			wantCycle: false,
		},
		{
			name: "simple cycle",
			tasks: []Task{
				{Number: 1, Name: "Task 1", Prompt: "test", DependsOn: []int{2}},
				{Number: 2, Name: "Task 2", Prompt: "test", DependsOn: []int{1}},
			},
			wantCycle: true,
		},
		{
			name: "self reference",
			tasks: []Task{
				{Number: 1, Name: "Task 1", Prompt: "test", DependsOn: []int{1}},
			},
			wantCycle: true,
		},
		{
			name: "no cycle - multiple dependencies",
			tasks: []Task{
				{Number: 1, Name: "Task 1", Prompt: "test", DependsOn: []int{}},
				{Number: 2, Name: "Task 2", Prompt: "test", DependsOn: []int{1}},
				{Number: 3, Name: "Task 3", Prompt: "test", DependsOn: []int{1}},
				{Number: 4, Name: "Task 4", Prompt: "test", DependsOn: []int{2, 3}},
			},
			wantCycle: false,
		},
		{
			name: "cycle in chain",
			tasks: []Task{
				{Number: 1, Name: "Task 1", Prompt: "test", DependsOn: []int{}},
				{Number: 2, Name: "Task 2", Prompt: "test", DependsOn: []int{1}},
				{Number: 3, Name: "Task 3", Prompt: "test", DependsOn: []int{2}},
				{Number: 4, Name: "Task 4", Prompt: "test", DependsOn: []int{3}},
				{Number: 1, Name: "Task 1", Prompt: "test", DependsOn: []int{4}}, // This creates cycle
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
				{Number: 1, Name: "Task 1", Prompt: "test", DependsOn: []int{}},
				{Number: 2, Name: "Task 2", Prompt: "test", DependsOn: []int{}},
				{Number: 3, Name: "Task 3", Prompt: "test", DependsOn: []int{}},
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
			TaskNumbers:    []int{1, 2, 3},
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
				{Number: 1, Name: "Task 1", Prompt: "test"},
				{Number: 2, Name: "Task 2", Prompt: "test"},
			},
			Waves: []Wave{
				{Name: "Wave 1", TaskNumbers: []int{1}},
				{Name: "Wave 2", TaskNumbers: []int{2}},
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
		task := Task{Number: 1, Name: "Test", Prompt: "test"}
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
