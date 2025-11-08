package executor

import (
	"testing"

	"github.com/harrison/conductor/internal/models"
)

func TestValidateTasks(t *testing.T) {
	tests := []struct {
		name    string
		tasks   []models.Task
		wantErr bool
	}{
		{
			name: "valid tasks",
			tasks: []models.Task{
				{Number: 1, Name: "Task 1", DependsOn: []int{}},
				{Number: 2, Name: "Task 2", DependsOn: []int{1}},
			},
			wantErr: false,
		},
		{
			name: "non-existent dependency",
			tasks: []models.Task{
				{Number: 1, Name: "Task 1", DependsOn: []int{999}},
			},
			wantErr: true,
		},
		{
			name: "duplicate task numbers",
			tasks: []models.Task{
				{Number: 1, Name: "Task 1", DependsOn: []int{}},
				{Number: 1, Name: "Task 1 Duplicate", DependsOn: []int{}},
			},
			wantErr: true,
		},
		{
			name: "invalid task number (zero)",
			tasks: []models.Task{
				{Number: 0, Name: "Task 0", DependsOn: []int{}},
			},
			wantErr: true,
		},
		{
			name: "invalid task number (negative)",
			tasks: []models.Task{
				{Number: -1, Name: "Task -1", DependsOn: []int{}},
			},
			wantErr: true,
		},
		{
			name:    "empty task list",
			tasks:   []models.Task{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTasks(tt.tasks)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTasks() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildDependencyGraph(t *testing.T) {
	tasks := []models.Task{
		{Number: 1, Name: "Task 1", DependsOn: []int{}},
		{Number: 2, Name: "Task 2", DependsOn: []int{1}},
		{Number: 3, Name: "Task 3", DependsOn: []int{1}},
	}

	graph := BuildDependencyGraph(tasks)

	if graph == nil {
		t.Fatal("Graph should not be nil")
	}

	if len(graph.Tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(graph.Tasks))
	}

	if graph.InDegree[1] != 0 {
		t.Errorf("Task 1 should have in-degree 0, got %d", graph.InDegree[1])
	}

	if graph.InDegree[2] != 1 {
		t.Errorf("Task 2 should have in-degree 1, got %d", graph.InDegree[2])
	}

	if graph.InDegree[3] != 1 {
		t.Errorf("Task 3 should have in-degree 1, got %d", graph.InDegree[3])
	}
}

func TestDetectCycle(t *testing.T) {
	tests := []struct {
		name      string
		tasks     []models.Task
		wantCycle bool
	}{
		{
			name: "no cycle",
			tasks: []models.Task{
				{Number: 1, DependsOn: []int{}},
				{Number: 2, DependsOn: []int{1}},
			},
			wantCycle: false,
		},
		{
			name: "simple cycle",
			tasks: []models.Task{
				{Number: 1, DependsOn: []int{2}},
				{Number: 2, DependsOn: []int{1}},
			},
			wantCycle: true,
		},
		{
			name: "self reference",
			tasks: []models.Task{
				{Number: 1, DependsOn: []int{1}},
			},
			wantCycle: true,
		},
		{
			name: "complex cycle",
			tasks: []models.Task{
				{Number: 1, DependsOn: []int{2}},
				{Number: 2, DependsOn: []int{3}},
				{Number: 3, DependsOn: []int{1}},
			},
			wantCycle: true,
		},
		{
			name: "no cycle complex",
			tasks: []models.Task{
				{Number: 1, DependsOn: []int{}},
				{Number: 2, DependsOn: []int{1}},
				{Number: 3, DependsOn: []int{1}},
				{Number: 4, DependsOn: []int{2, 3}},
			},
			wantCycle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := BuildDependencyGraph(tt.tasks)
			hasCycle := graph.HasCycle()
			if hasCycle != tt.wantCycle {
				t.Errorf("HasCycle() = %v, want %v", hasCycle, tt.wantCycle)
			}
		})
	}
}

func TestCalculateWaves(t *testing.T) {
	tests := []struct {
		name          string
		tasks         []models.Task
		wantWaveCount int
		wantErr       bool
		validateWaves func(*testing.T, []models.Wave)
	}{
		{
			name: "simple linear",
			tasks: []models.Task{
				{Number: 1, Name: "Task 1", DependsOn: []int{}},
				{Number: 2, Name: "Task 2", DependsOn: []int{1}},
			},
			wantWaveCount: 2,
			wantErr:       false,
			validateWaves: func(t *testing.T, waves []models.Wave) {
				if len(waves[0].TaskNumbers) != 1 || waves[0].TaskNumbers[0] != 1 {
					t.Error("Wave 1 should contain only task 1")
				}
				if len(waves[1].TaskNumbers) != 1 || waves[1].TaskNumbers[0] != 2 {
					t.Error("Wave 2 should contain only task 2")
				}
			},
		},
		{
			name: "parallel tasks",
			tasks: []models.Task{
				{Number: 1, Name: "Task 1", DependsOn: []int{}},
				{Number: 2, Name: "Task 2", DependsOn: []int{1}},
				{Number: 3, Name: "Task 3", DependsOn: []int{1}},
				{Number: 4, Name: "Task 4", DependsOn: []int{2, 3}},
			},
			wantWaveCount: 3,
			wantErr:       false,
			validateWaves: func(t *testing.T, waves []models.Wave) {
				// Wave 1: [1]
				if len(waves[0].TaskNumbers) != 1 {
					t.Errorf("Wave 1 should have 1 task, got %d", len(waves[0].TaskNumbers))
				}
				// Wave 2: [2, 3]
				if len(waves[1].TaskNumbers) != 2 {
					t.Errorf("Wave 2 should have 2 tasks, got %d", len(waves[1].TaskNumbers))
				}
				// Wave 3: [4]
				if len(waves[2].TaskNumbers) != 1 {
					t.Errorf("Wave 3 should have 1 task, got %d", len(waves[2].TaskNumbers))
				}
			},
		},
		{
			name: "all independent",
			tasks: []models.Task{
				{Number: 1, Name: "Task 1", DependsOn: []int{}},
				{Number: 2, Name: "Task 2", DependsOn: []int{}},
				{Number: 3, Name: "Task 3", DependsOn: []int{}},
			},
			wantWaveCount: 1,
			wantErr:       false,
			validateWaves: func(t *testing.T, waves []models.Wave) {
				if len(waves[0].TaskNumbers) != 3 {
					t.Errorf("Wave 1 should have 3 tasks, got %d", len(waves[0].TaskNumbers))
				}
			},
		},
		{
			name: "circular dependency",
			tasks: []models.Task{
				{Number: 1, Name: "Task 1", DependsOn: []int{2}},
				{Number: 2, Name: "Task 2", DependsOn: []int{1}},
			},
			wantWaveCount: 0,
			wantErr:       true,
			validateWaves: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			waves, err := CalculateWaves(tt.tasks)

			if (err != nil) != tt.wantErr {
				t.Errorf("CalculateWaves() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if len(waves) != tt.wantWaveCount {
				t.Errorf("Expected %d waves, got %d", tt.wantWaveCount, len(waves))
			}

			if tt.validateWaves != nil {
				tt.validateWaves(t, waves)
			}

			// Verify all waves have names
			for i, wave := range waves {
				if wave.Name == "" {
					t.Errorf("Wave %d has no name", i+1)
				}
			}

			// Verify default concurrency is set
			for i, wave := range waves {
				if wave.MaxConcurrency == 0 {
					t.Errorf("Wave %d has no MaxConcurrency set", i+1)
				}
			}
		})
	}
}

func TestIndependentTasks(t *testing.T) {
	tasks := []models.Task{
		{Number: 1, Name: "Task 1", DependsOn: []int{}},
		{Number: 2, Name: "Task 2", DependsOn: []int{}},
		{Number: 3, Name: "Task 3", DependsOn: []int{}},
	}

	waves, err := CalculateWaves(tasks)
	if err != nil {
		t.Fatalf("CalculateWaves failed: %v", err)
	}

	if len(waves) != 1 {
		t.Errorf("All independent tasks should be in one wave, got %d waves", len(waves))
	}

	if len(waves[0].TaskNumbers) != 3 {
		t.Errorf("Wave should contain 3 tasks, got %d", len(waves[0].TaskNumbers))
	}
}

func TestTopologicalSort(t *testing.T) {
	tasks := []models.Task{
		{Number: 1, Name: "Task 1", DependsOn: []int{}},
		{Number: 2, Name: "Task 2", DependsOn: []int{1}},
		{Number: 3, Name: "Task 3", DependsOn: []int{2}},
	}

	waves, err := CalculateWaves(tasks)
	if err != nil {
		t.Fatalf("CalculateWaves failed: %v", err)
	}

	// Verify topological ordering: all dependencies come before dependents
	completed := make(map[int]bool)

	for _, wave := range waves {
		for _, taskNum := range wave.TaskNumbers {
			// Find the task
			var task models.Task
			for _, t := range tasks {
				if t.Number == taskNum {
					task = t
					break
				}
			}

			// Verify all dependencies are completed
			for _, dep := range task.DependsOn {
				if !completed[dep] {
					t.Errorf("Task %d executed before its dependency %d", taskNum, dep)
				}
			}

			completed[taskNum] = true
		}
	}
}
