package executor

import (
	"bytes"
	"io"
	"os"
	"strings"
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
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "2", Name: "Task 2", DependsOn: []string{"1"}},
			},
			wantErr: false,
		},
		{
			name: "non-existent dependency",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{"999"}},
			},
			wantErr: true,
		},
		{
			name: "duplicate task numbers",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "1", Name: "Task 1 Duplicate", DependsOn: []string{}},
			},
			wantErr: true,
		},
		{
			name: "empty task number",
			tasks: []models.Task{
				{Number: "", Name: "Empty number task", DependsOn: []string{}},
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
		{Number: "1", Name: "Task 1", DependsOn: []string{}},
		{Number: "2", Name: "Task 2", DependsOn: []string{"1"}},
		{Number: "3", Name: "Task 3", DependsOn: []string{"1"}},
	}

	graph := BuildDependencyGraph(tasks)

	if graph == nil {
		t.Fatal("Graph should not be nil")
	}

	if len(graph.Tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(graph.Tasks))
	}

	if graph.InDegree["1"] != 0 {
		t.Errorf("Task 1 should have in-degree 0, got %d", graph.InDegree["1"])
	}

	if graph.InDegree["2"] != 1 {
		t.Errorf("Task 2 should have in-degree 1, got %d", graph.InDegree["2"])
	}

	if graph.InDegree["3"] != 1 {
		t.Errorf("Task 3 should have in-degree 1, got %d", graph.InDegree["3"])
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
				{Number: "1", DependsOn: []string{}},
				{Number: "2", DependsOn: []string{"1"}},
			},
			wantCycle: false,
		},
		{
			name: "simple cycle",
			tasks: []models.Task{
				{Number: "1", DependsOn: []string{"2"}},
				{Number: "2", DependsOn: []string{"1"}},
			},
			wantCycle: true,
		},
		{
			name: "self reference",
			tasks: []models.Task{
				{Number: "1", DependsOn: []string{"1"}},
			},
			wantCycle: true,
		},
		{
			name: "complex cycle",
			tasks: []models.Task{
				{Number: "1", DependsOn: []string{"2"}},
				{Number: "2", DependsOn: []string{"3"}},
				{Number: "3", DependsOn: []string{"1"}},
			},
			wantCycle: true,
		},
		{
			name: "no cycle complex",
			tasks: []models.Task{
				{Number: "1", DependsOn: []string{}},
				{Number: "2", DependsOn: []string{"1"}},
				{Number: "3", DependsOn: []string{"1"}},
				{Number: "4", DependsOn: []string{"2", "3"}},
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
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "2", Name: "Task 2", DependsOn: []string{"1"}},
			},
			wantWaveCount: 2,
			wantErr:       false,
			validateWaves: func(t *testing.T, waves []models.Wave) {
				if len(waves[0].TaskNumbers) != 1 || waves[0].TaskNumbers[0] != "1" {
					t.Error("Wave 1 should contain only task 1")
				}
				if len(waves[1].TaskNumbers) != 1 || waves[1].TaskNumbers[0] != "2" {
					t.Error("Wave 2 should contain only task 2")
				}
			},
		},
		{
			name: "parallel tasks",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "2", Name: "Task 2", DependsOn: []string{"1"}},
				{Number: "3", Name: "Task 3", DependsOn: []string{"1"}},
				{Number: "4", Name: "Task 4", DependsOn: []string{"2", "3"}},
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
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "2", Name: "Task 2", DependsOn: []string{}},
				{Number: "3", Name: "Task 3", DependsOn: []string{}},
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
				{Number: "1", Name: "Task 1", DependsOn: []string{"2"}},
				{Number: "2", Name: "Task 2", DependsOn: []string{"1"}},
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
		{Number: "1", Name: "Task 1", DependsOn: []string{}},
		{Number: "2", Name: "Task 2", DependsOn: []string{}},
		{Number: "3", Name: "Task 3", DependsOn: []string{}},
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
		{Number: "1", Name: "Task 1", DependsOn: []string{}},
		{Number: "2", Name: "Task 2", DependsOn: []string{"1"}},
		{Number: "3", Name: "Task 3", DependsOn: []string{"2"}},
	}

	waves, err := CalculateWaves(tasks)
	if err != nil {
		t.Fatalf("CalculateWaves failed: %v", err)
	}

	// Verify topological ordering: all dependencies come before dependents
	completed := make(map[string]bool)

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
					t.Errorf("Task %s executed before its dependency %s", taskNum, dep)
				}
			}

			completed[taskNum] = true
		}
	}
}

func TestValidateFileOverlaps(t *testing.T) {
	type expectations struct {
		errContains     []string
		warningContains []string
	}

	tests := []struct {
		name          string
		tasks         []models.Task
		waves         []models.Wave
		wantErr       bool
		expectWarning bool
		expectations  expectations
	}{
		{
			name: "no overlaps in single wave",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", Files: []string{"alpha.go"}},
				{Number: "2", Name: "Task 2", Files: []string{"beta.go"}},
			},
			waves: []models.Wave{
				{Name: "Wave 1", TaskNumbers: []string{"1", "2"}},
			},
			wantErr:       false,
			expectWarning: false,
		},
		{
			name: "overlap within same wave returns error",
			tasks: []models.Task{
				{Number: "1", Name: "Task A", Files: []string{"shared/file.go"}},
				{Number: "2", Name: "Task B", Files: []string{"shared/file.go"}},
			},
			waves: []models.Wave{
				{Name: "Wave 1", TaskNumbers: []string{"1", "2"}},
			},
			wantErr: true,
			expectations: expectations{
				errContains: []string{"Wave 1", "shared/file.go", "Task A", "Task B", "Move the conflicting tasks"},
			},
		},
		{
			name: "overlap across waves allowed",
			tasks: []models.Task{
				{Number: "1", Name: "Task A", Files: []string{"shared/file.go"}},
				{Number: "2", Name: "Task B", Files: []string{"shared/file.go"}},
			},
			waves: []models.Wave{
				{Name: "Wave 1", TaskNumbers: []string{"1"}},
				{Name: "Wave 2", TaskNumbers: []string{"2"}},
			},
			wantErr:       false,
			expectWarning: false,
		},
		{
			name: "skip validation when task has no files",
			tasks: []models.Task{
				{Number: "1", Name: "Task A", Files: nil},
				{Number: "2", Name: "Task B", Files: []string{"beta.go"}},
			},
			waves: []models.Wave{
				{Name: "Wave 1", TaskNumbers: []string{"1", "2"}},
			},
			wantErr:       false,
			expectWarning: true,
			expectations: expectations{
				warningContains: []string{"Wave 1", "Task A", "skipping file overlap validation"},
			},
		},
		{
			name: "path normalization treats ./file and file as same",
			tasks: []models.Task{
				{Number: "1", Name: "Task A", Files: []string{"./config.go"}},
				{Number: "2", Name: "Task B", Files: []string{"config.go"}},
			},
			waves: []models.Wave{
				{Name: "Wave 1", TaskNumbers: []string{"1", "2"}},
			},
			wantErr: true,
			expectations: expectations{
				errContains: []string{"Wave 1", "config.go", "Task A", "Task B"},
			},
		},
		{
			name: "duplicate files within same task do not error",
			tasks: []models.Task{
				{Number: "1", Name: "Task A", Files: []string{"alpha.go", "alpha.go"}},
				{Number: "2", Name: "Task B", Files: []string{"beta.go"}},
			},
			waves: []models.Wave{
				{Name: "Wave 1", TaskNumbers: []string{"1", "2"}},
			},
			wantErr:       false,
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build taskMap from tasks
			taskMap := make(map[string]*models.Task)
			for i := range tt.tasks {
				taskMap[tt.tasks[i].Number] = &tt.tasks[i]
			}

			var (
				err      error
				warnings string
			)

			if tt.expectWarning {
				warnings = captureStderr(t, func() {
					err = ValidateFileOverlaps(tt.waves, taskMap)
				})
			} else {
				err = ValidateFileOverlaps(tt.waves, taskMap)
			}

			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateFileOverlaps() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				msg := err.Error()
				for _, snippet := range tt.expectations.errContains {
					if !strings.Contains(msg, snippet) {
						t.Errorf("expected error to contain %q, got %q", snippet, msg)
					}
				}
			}

			if tt.expectWarning {
				if warnings == "" {
					t.Fatal("expected warning output but got none")
				}
				for _, snippet := range tt.expectations.warningContains {
					if !strings.Contains(warnings, snippet) {
						t.Errorf("expected warning to contain %q, got %q", snippet, warnings)
					}
				}
			} else if warnings != "" {
				t.Fatalf("did not expect warning but got: %s", warnings)
			}
		})
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	os.Stderr = w
	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}
	os.Stderr = original

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to copy stderr output: %v", err)
	}

	return buf.String()
}

func TestGraphNodeGroups(t *testing.T) {
	tests := []struct {
		name       string
		tasks      []models.Task
		wantGroups map[int]string // task number (as string) -> group ID
		wantErr    bool
	}{
		{
			name: "single group",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}, WorktreeGroup: "group-a"},
				{Number: "2", Name: "Task 2", DependsOn: []string{"1"}, WorktreeGroup: "group-a"},
			},
			wantGroups: map[int]string{1: "group-a", 2: "group-a"},
			wantErr:    false,
		},
		{
			name: "multiple groups",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}, WorktreeGroup: "group-a"},
				{Number: "2", Name: "Task 2", DependsOn: []string{}, WorktreeGroup: "group-b"},
				{Number: "3", Name: "Task 3", DependsOn: []string{"1"}, WorktreeGroup: "group-c"},
			},
			wantGroups: map[int]string{1: "group-a", 2: "group-b", 3: "group-c"},
			wantErr:    false,
		},
		{
			name: "mixed with empty group",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}, WorktreeGroup: "group-a"},
				{Number: "2", Name: "Task 2", DependsOn: []string{}, WorktreeGroup: ""},
				{Number: "3", Name: "Task 3", DependsOn: []string{"1", "2"}, WorktreeGroup: "group-a"},
			},
			wantGroups: map[int]string{1: "group-a", 2: "", 3: "group-a"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := BuildDependencyGraph(tt.tasks)

			if graph == nil {
				t.Fatal("Graph should not be nil")
			}

			if len(graph.Groups) != len(tt.wantGroups) {
				t.Fatalf("Expected %d group mappings, got %d", len(tt.wantGroups), len(graph.Groups))
			}

			for taskNum, expectedGroup := range tt.wantGroups {
				actualGroup, exists := graph.Groups[taskNum]
				if !exists {
					t.Errorf("Task %d not in Groups map", taskNum)
					continue
				}
				if actualGroup != expectedGroup {
					t.Errorf("Task %d: expected group %q, got %q", taskNum, expectedGroup, actualGroup)
				}
			}
		})
	}
}

func TestWaveGroupMetadata(t *testing.T) {
	tests := []struct {
		name                  string
		tasks                 []models.Task
		wantWaveCount         int
		wantErr               bool
		validateGroupMetadata func(*testing.T, []models.Wave)
	}{
		{
			name: "single group wave",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}, WorktreeGroup: "group-a"},
				{Number: "2", Name: "Task 2", DependsOn: []string{"1"}, WorktreeGroup: "group-a"},
			},
			wantWaveCount: 2,
			wantErr:       false,
			validateGroupMetadata: func(t *testing.T, waves []models.Wave) {
				// Wave 1: [1] from group-a
				if len(waves[0].GroupInfo) != 1 {
					t.Errorf("Wave 1: expected 1 group, got %d", len(waves[0].GroupInfo))
				}
				if _, exists := waves[0].GroupInfo["group-a"]; !exists {
					t.Error("Wave 1: expected group-a in GroupInfo")
				}
				if len(waves[0].GroupInfo["group-a"]) != 1 {
					t.Errorf("Wave 1: expected 1 task in group-a, got %d", len(waves[0].GroupInfo["group-a"]))
				}

				// Wave 2: [2] from group-a
				if len(waves[1].GroupInfo) != 1 {
					t.Errorf("Wave 2: expected 1 group, got %d", len(waves[1].GroupInfo))
				}
				if _, exists := waves[1].GroupInfo["group-a"]; !exists {
					t.Error("Wave 2: expected group-a in GroupInfo")
				}
				if len(waves[1].GroupInfo["group-a"]) != 1 {
					t.Errorf("Wave 2: expected 1 task in group-a, got %d", len(waves[1].GroupInfo["group-a"]))
				}
			},
		},
		{
			name: "mixed groups in wave",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}, WorktreeGroup: "group-a"},
				{Number: "2", Name: "Task 2", DependsOn: []string{}, WorktreeGroup: "group-b"},
				{Number: "3", Name: "Task 3", DependsOn: []string{"1", "2"}, WorktreeGroup: "group-c"},
			},
			wantWaveCount: 2,
			wantErr:       false,
			validateGroupMetadata: func(t *testing.T, waves []models.Wave) {
				// Wave 1: [1, 2] from group-a and group-b
				if len(waves[0].GroupInfo) != 2 {
					t.Errorf("Wave 1: expected 2 groups, got %d", len(waves[0].GroupInfo))
				}
				if _, exists := waves[0].GroupInfo["group-a"]; !exists {
					t.Error("Wave 1: expected group-a in GroupInfo")
				}
				if _, exists := waves[0].GroupInfo["group-b"]; !exists {
					t.Error("Wave 1: expected group-b in GroupInfo")
				}

				// Wave 2: [3] from group-c
				if len(waves[1].GroupInfo) != 1 {
					t.Errorf("Wave 2: expected 1 group, got %d", len(waves[1].GroupInfo))
				}
				if _, exists := waves[1].GroupInfo["group-c"]; !exists {
					t.Error("Wave 2: expected group-c in GroupInfo")
				}
			},
		},
		{
			name: "empty group strings",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}, WorktreeGroup: ""},
				{Number: "2", Name: "Task 2", DependsOn: []string{}, WorktreeGroup: ""},
				{Number: "3", Name: "Task 3", DependsOn: []string{"1", "2"}, WorktreeGroup: ""},
			},
			wantWaveCount: 2,
			wantErr:       false,
			validateGroupMetadata: func(t *testing.T, waves []models.Wave) {
				// Wave 1: [1, 2] both with empty group
				if len(waves[0].GroupInfo) != 1 {
					t.Errorf("Wave 1: expected 1 entry in GroupInfo (empty string key), got %d", len(waves[0].GroupInfo))
				}
				if val, exists := waves[0].GroupInfo[""]; !exists || len(val) != 2 {
					t.Error("Wave 1: expected 2 tasks under empty group key")
				}
			},
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

			if tt.validateGroupMetadata != nil {
				tt.validateGroupMetadata(t, waves)
			}
		})
	}
}
