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

func TestTaskNumericalSorting(t *testing.T) {
	tests := []struct {
		name         string
		tasks        []models.Task
		expectedWave []string
	}{
		{
			name: "sort multi-digit task numbers",
			tasks: []models.Task{
				{Number: "21", Name: "Task 21", DependsOn: []string{}},
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "16", Name: "Task 16", DependsOn: []string{}},
				{Number: "17", Name: "Task 17", DependsOn: []string{}},
				{Number: "13", Name: "Task 13", DependsOn: []string{}},
				{Number: "14", Name: "Task 14", DependsOn: []string{}},
			},
			expectedWave: []string{"1", "13", "14", "16", "17", "21"},
		},
		{
			name: "sort single-digit task numbers",
			tasks: []models.Task{
				{Number: "3", Name: "Task 3", DependsOn: []string{}},
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "2", Name: "Task 2", DependsOn: []string{}},
			},
			expectedWave: []string{"1", "2", "3"},
		},
		{
			name: "sort mixed format task numbers",
			tasks: []models.Task{
				{Number: "Task 5", Name: "Task 5", DependsOn: []string{}},
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "Task 10", Name: "Task 10", DependsOn: []string{}},
				{Number: "3", Name: "Task 3", DependsOn: []string{}},
			},
			expectedWave: []string{"1", "3", "Task 5", "Task 10"},
		},
		{
			name: "unparseable strings sort last",
			tasks: []models.Task{
				{Number: "invalid", Name: "Invalid Task", DependsOn: []string{}},
				{Number: "5", Name: "Task 5", DependsOn: []string{}},
				{Number: "alpha", Name: "Alpha Task", DependsOn: []string{}},
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
			},
			// Only validate that parseable numbers come first in correct order
			// Unparseable strings order is non-deterministic (both have same sort key)
			expectedWave: []string{"1", "5"}, // Check first two only
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			waves, err := CalculateWaves(tt.tasks)
			if err != nil {
				t.Fatalf("CalculateWaves() failed: %v", err)
			}

			if len(waves) != 1 {
				t.Fatalf("Expected 1 wave, got %d", len(waves))
			}

			actualWave := waves[0].TaskNumbers

			// Check at least the expected tasks are present and in order
			// (for tests that only check partial ordering)
			if len(actualWave) < len(tt.expectedWave) {
				t.Fatalf("Expected at least %d tasks in wave, got %d", len(tt.expectedWave), len(actualWave))
			}

			for i, expected := range tt.expectedWave {
				if actualWave[i] != expected {
					t.Errorf("Task at position %d: expected %q, got %q", i, expected, actualWave[i])
				}
			}
		})
	}
}

func TestParseTaskNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "simple number",
			input:    "1",
			expected: 1,
		},
		{
			name:     "multi-digit number",
			input:    "42",
			expected: 42,
		},
		{
			name:     "Task prefix format",
			input:    "Task 5",
			expected: 5,
		},
		{
			name:     "Task prefix with multi-digit",
			input:    "Task 100",
			expected: 100,
		},
		{
			name:     "unparseable string",
			input:    "invalid",
			expected: 999999,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 999999,
		},
		{
			name:     "alphabetic string",
			input:    "alpha",
			expected: 999999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTaskNumber(tt.input)
			if result != tt.expected {
				t.Errorf("parseTaskNumber(%q) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

// TestValidateTasks_CrossFileReferences tests validation of cross-file dependencies
func TestValidateTasks_CrossFileReferences(t *testing.T) {
	tests := []struct {
		name    string
		tasks   []models.Task
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid cross-file reference",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "5", Name: "Task 5", DependsOn: []string{"file:plan-01.md:task:1"}},
			},
			wantErr: false,
		},
		{
			name: "multiple cross-file references",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "5", Name: "Task 5", DependsOn: []string{"file:plan-01.md:task:1"}},
				{Number: "10", Name: "Task 10", DependsOn: []string{"file:plan-02.md:task:5"}},
			},
			wantErr: false,
		},
		{
			name: "cross-file reference to non-existent task",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "5", Name: "Task 5", DependsOn: []string{"file:plan-01.md:task:999"}},
			},
			wantErr: true,
			errMsg:  "non-existent task",
		},
		{
			name: "invalid cross-file format",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "5", Name: "Task 5", DependsOn: []string{"file:plan-01.md"}}, // missing task:
			},
			wantErr: true,
			errMsg:  "non-existent", // Treated as a local dependency that doesn't exist
		},
		{
			name: "mixed local and cross-file dependencies",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "2", Name: "Task 2", DependsOn: []string{"1"}},
				{Number: "5", Name: "Task 5", DependsOn: []string{"file:plan-01.md:task:2", "2"}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTasks(tt.tasks)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTasks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error to contain %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

// TestBuildFileAwareDependencyGraph_Valid tests building a file-aware dependency graph
func TestBuildFileAwareDependencyGraph_Valid(t *testing.T) {
	tasks := []models.Task{
		{Number: "1", Name: "Task 1", DependsOn: []string{}},
		{Number: "2", Name: "Task 2", DependsOn: []string{"1"}},
		{Number: "5", Name: "Task 5", DependsOn: []string{"file:plan-01.md:task:1"}},
		{Number: "10", Name: "Task 10", DependsOn: []string{"file:plan-02.md:task:5", "2"}},
	}

	depGraph, err := BuildFileAwareDependencyGraph(tasks)
	if err != nil {
		t.Fatalf("BuildFileAwareDependencyGraph() error = %v", err)
	}

	if depGraph == nil {
		t.Fatal("BuildFileAwareDependencyGraph() returned nil graph")
	}

	// Verify all tasks are in the graph
	if len(depGraph) != 4 {
		t.Errorf("Expected 4 tasks in graph, got %d", len(depGraph))
	}

	// Verify dependencies are preserved with full cross-file information
	if len(depGraph["5"]) != 1 || depGraph["5"][0] != "file:plan-01.md:task:1" {
		t.Errorf("Task 5: expected ['file:plan-01.md:task:1'], got %v", depGraph["5"])
	}

	if len(depGraph["10"]) != 2 {
		t.Errorf("Task 10: expected 2 dependencies, got %d", len(depGraph["10"]))
	}
}

// TestBuildFileAwareDependencyGraph_InvalidReference tests error handling for invalid cross-file refs
func TestBuildFileAwareDependencyGraph_InvalidReference(t *testing.T) {
	tests := []struct {
		name    string
		tasks   []models.Task
		wantErr bool
		errMsg  string
	}{
		{
			name: "reference to non-existent task",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "5", Name: "Task 5", DependsOn: []string{"file:plan-01.md:task:999"}},
			},
			wantErr: true,
			errMsg:  "non-existent",
		},
		{
			name: "malformed cross-file dependency",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "5", Name: "Task 5", DependsOn: []string{"file:plan-01.md"}},
			},
			wantErr: true,
			errMsg:  "non-existent", // Treated as a local dependency that doesn't exist
		},
		{
			name: "local reference to non-existent task",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "5", Name: "Task 5", DependsOn: []string{"999"}},
			},
			wantErr: true,
			errMsg:  "non-existent",
		},
		{
			name: "duplicate task numbers",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "1", Name: "Task 1 Duplicate", DependsOn: []string{}},
			},
			wantErr: true,
			errMsg:  "duplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			depGraph, err := BuildFileAwareDependencyGraph(tt.tasks)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildFileAwareDependencyGraph() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to contain %q, got %q", tt.errMsg, err.Error())
				}
				if depGraph != nil {
					t.Error("Expected nil graph on error")
				}
			}
		})
	}
}

// TestBuildDependencyGraph_CrossFileResolution tests that BuildDependencyGraph properly resolves cross-file deps
func TestBuildDependencyGraph_CrossFileResolution(t *testing.T) {
	tasks := []models.Task{
		{Number: "1", Name: "Task 1", DependsOn: []string{}},
		{Number: "5", Name: "Task 5", DependsOn: []string{"file:plan-01.md:task:1"}},
		{Number: "10", Name: "Task 10", DependsOn: []string{"5"}},
	}

	graph := BuildDependencyGraph(tasks)
	if graph == nil {
		t.Fatal("BuildDependencyGraph() returned nil")
	}

	// Verify that cross-file dependencies are resolved to extract task ID
	// Task 5 depends on "file:plan-01.md:task:1" which resolves to task "1"
	// So the in-degree of task "5" should be 1, with "1" as its dependency
	if graph.InDegree["5"] != 1 {
		t.Errorf("Task 5 in-degree: expected 1, got %d", graph.InDegree["5"])
	}

	// Verify that task "1" has task "5" as a dependent
	if len(graph.Edges["1"]) != 1 || graph.Edges["1"][0] != "5" {
		t.Errorf("Task 1 edges: expected ['5'], got %v", graph.Edges["1"])
	}

	// Verify that task "10" depends on task "5" (local dependency)
	if graph.InDegree["10"] != 1 {
		t.Errorf("Task 10 in-degree: expected 1, got %d", graph.InDegree["10"])
	}
}

// TestCalculateWaves_CrossFileDependencies tests wave calculation with cross-file dependencies
func TestCalculateWaves_CrossFileDependencies(t *testing.T) {
	tests := []struct {
		name          string
		tasks         []models.Task
		wantWaveCount int
		wantErr       bool
		validate      func(*testing.T, []models.Wave)
	}{
		{
			name: "linear cross-file chain",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "5", Name: "Task 5", DependsOn: []string{"file:plan-01.md:task:1"}},
				{Number: "10", Name: "Task 10", DependsOn: []string{"5"}},
			},
			wantWaveCount: 3,
			wantErr:       false,
			validate: func(t *testing.T, waves []models.Wave) {
				// Wave 1: [1]
				// Wave 2: [5] (depends on 1)
				// Wave 3: [10] (depends on 5)
				if len(waves[0].TaskNumbers) != 1 || waves[0].TaskNumbers[0] != "1" {
					t.Error("Wave 1 should contain only task 1")
				}
				if len(waves[1].TaskNumbers) != 1 || waves[1].TaskNumbers[0] != "5" {
					t.Error("Wave 2 should contain only task 5")
				}
				if len(waves[2].TaskNumbers) != 1 || waves[2].TaskNumbers[0] != "10" {
					t.Error("Wave 3 should contain only task 10")
				}
			},
		},
		{
			name: "diamond pattern with cross-file",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "2", Name: "Task 2", DependsOn: []string{"file:plan-01.md:task:1"}},
				{Number: "3", Name: "Task 3", DependsOn: []string{"file:plan-01.md:task:1"}},
				{Number: "4", Name: "Task 4", DependsOn: []string{"2", "3"}},
			},
			wantWaveCount: 3,
			wantErr:       false,
			validate: func(t *testing.T, waves []models.Wave) {
				// Wave 1: [1]
				// Wave 2: [2, 3]
				// Wave 3: [4]
				if len(waves[0].TaskNumbers) != 1 {
					t.Errorf("Wave 1 should have 1 task, got %d", len(waves[0].TaskNumbers))
				}
				if len(waves[1].TaskNumbers) != 2 {
					t.Errorf("Wave 2 should have 2 tasks, got %d", len(waves[1].TaskNumbers))
				}
				if len(waves[2].TaskNumbers) != 1 {
					t.Errorf("Wave 3 should have 1 task, got %d", len(waves[2].TaskNumbers))
				}
			},
		},
		{
			name: "cross-file reference to non-existent task",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "5", Name: "Task 5", DependsOn: []string{"file:plan-01.md:task:999"}},
			},
			wantWaveCount: 0,
			wantErr:       true,
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

			if tt.validate != nil {
				tt.validate(t, waves)
			}
		})
	}
}

// =============================================================================
// Registry Prerequisite Validation Tests
// =============================================================================

// TestValidateRegistryPrerequisites_Valid verifies valid registry passes
func TestValidateRegistryPrerequisites_Valid(t *testing.T) {
	registry := &models.DataFlowRegistry{
		Producers: map[string][]models.DataFlowEntry{
			"SymbolA": {{TaskNumber: "1", Description: "Produces A"}},
		},
		Consumers: map[string][]models.DataFlowEntry{
			"SymbolA": {{TaskNumber: "2", Description: "Consumes A"}},
		},
	}

	tasks := []models.Task{
		{Number: "1", Name: "Task 1", DependsOn: []string{}},
		{Number: "2", Name: "Task 2", DependsOn: []string{"1"}}, // Depends on producer
	}

	err := ValidateRegistryPrerequisites(tasks, registry)
	if err != nil {
		t.Errorf("Expected no error for valid registry, got: %v", err)
	}
}

// TestValidateRegistryPrerequisites_MissingDependency verifies consumer must depend on producer
func TestValidateRegistryPrerequisites_MissingDependency(t *testing.T) {
	registry := &models.DataFlowRegistry{
		Producers: map[string][]models.DataFlowEntry{
			"SymbolA": {{TaskNumber: "1", Description: "Produces A"}},
		},
		Consumers: map[string][]models.DataFlowEntry{
			"SymbolA": {{TaskNumber: "2", Description: "Consumes A"}},
		},
	}

	tasks := []models.Task{
		{Number: "1", Name: "Task 1", DependsOn: []string{}},
		{Number: "2", Name: "Task 2", DependsOn: []string{}}, // No dependency on producer!
	}

	err := ValidateRegistryPrerequisites(tasks, registry)
	if err == nil {
		t.Fatal("Expected error for missing dependency on producer")
	}
	if !strings.Contains(err.Error(), "task 2") {
		t.Errorf("Expected error to mention task 2, got: %v", err)
	}
}

// TestValidateRegistryPrerequisites_TransitiveDependency verifies transitive deps work
func TestValidateRegistryPrerequisites_TransitiveDependency(t *testing.T) {
	registry := &models.DataFlowRegistry{
		Producers: map[string][]models.DataFlowEntry{
			"SymbolA": {{TaskNumber: "1", Description: "Produces A"}},
		},
		Consumers: map[string][]models.DataFlowEntry{
			"SymbolA": {{TaskNumber: "3", Description: "Consumes A"}},
		},
	}

	tasks := []models.Task{
		{Number: "1", Name: "Task 1", DependsOn: []string{}},
		{Number: "2", Name: "Task 2", DependsOn: []string{"1"}},
		{Number: "3", Name: "Task 3", DependsOn: []string{"2"}}, // Transitively depends on 1
	}

	err := ValidateRegistryPrerequisites(tasks, registry)
	if err != nil {
		t.Errorf("Expected transitive dependency to satisfy registry, got: %v", err)
	}
}

// TestValidateRegistryPrerequisites_NilRegistry verifies nil registry passes
func TestValidateRegistryPrerequisites_NilRegistry(t *testing.T) {
	tasks := []models.Task{
		{Number: "1", Name: "Task 1", DependsOn: []string{}},
	}

	err := ValidateRegistryPrerequisites(tasks, nil)
	if err != nil {
		t.Errorf("Expected no error for nil registry, got: %v", err)
	}
}

// TestValidateRegistryPrerequisites_CrossFileDependency verifies cross-file deps work
func TestValidateRegistryPrerequisites_CrossFileDependency(t *testing.T) {
	registry := &models.DataFlowRegistry{
		Producers: map[string][]models.DataFlowEntry{
			"SymbolA": {{TaskNumber: "1", Description: "Produces A"}},
		},
		Consumers: map[string][]models.DataFlowEntry{
			"SymbolA": {{TaskNumber: "2", Description: "Consumes A"}},
		},
	}

	tasks := []models.Task{
		{Number: "1", Name: "Task 1", DependsOn: []string{}},
		{Number: "2", Name: "Task 2", DependsOn: []string{"file:plan-01.yaml:task:1"}}, // Cross-file dep
	}

	err := ValidateRegistryPrerequisites(tasks, registry)
	if err != nil {
		t.Errorf("Expected cross-file dependency to satisfy registry, got: %v", err)
	}
}

// TestDetectCycle_CrossFileReferences tests cycle detection with cross-file dependencies
func TestDetectCycle_CrossFileReferences(t *testing.T) {
	tests := []struct {
		name      string
		tasks     []models.Task
		wantCycle bool
	}{
		{
			name: "no cycle with cross-file",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{}},
				{Number: "5", Name: "Task 5", DependsOn: []string{"file:plan-01.md:task:1"}},
			},
			wantCycle: false,
		},
		{
			name: "simple cycle across files",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{"file:plan-02.md:task:5"}},
				{Number: "5", Name: "Task 5", DependsOn: []string{"1"}},
			},
			wantCycle: true,
		},
		{
			name: "self-reference via cross-file",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{"file:plan-01.md:task:1"}},
			},
			wantCycle: true,
		},
		{
			name: "complex cycle with cross-file",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", DependsOn: []string{"file:plan-02.md:task:5"}},
				{Number: "5", Name: "Task 5", DependsOn: []string{"file:plan-03.md:task:10"}},
				{Number: "10", Name: "Task 10", DependsOn: []string{"1"}},
			},
			wantCycle: true,
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

// =============================================================================
// Package Conflict Detection in Waves Tests
// =============================================================================

func TestCalculateWaves_PackageConflictDetection(t *testing.T) {
	tests := []struct {
		name    string
		tasks   []models.Task
		wantErr bool
		errMsg  string
	}{
		{
			name: "no conflict - different packages",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", Files: []string{"internal/executor/task.go"}},
				{Number: "2", Name: "Task 2", Files: []string{"internal/parser/yaml.go"}},
			},
			wantErr: false,
		},
		{
			name: "conflict - same package no dependency",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", Files: []string{"internal/executor/task.go"}},
				{Number: "2", Name: "Task 2", Files: []string{"internal/executor/wave.go"}},
			},
			wantErr: true,
			errMsg:  "internal/executor",
		},
		{
			name: "no conflict - same package with dependency",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", Files: []string{"internal/executor/task.go"}},
				{Number: "2", Name: "Task 2", DependsOn: []string{"1"}, Files: []string{"internal/executor/wave.go"}},
			},
			wantErr: false,
		},
		{
			name: "no conflict - non-Go files same directory",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", Files: []string{"docs/README.md"}},
				{Number: "2", Name: "Task 2", Files: []string{"docs/guide.md"}},
			},
			wantErr: false,
		},
		{
			name: "conflict in wave 2 - same package",
			tasks: []models.Task{
				{Number: "1", Name: "Task 1", Files: []string{"internal/parser/yaml.go"}},
				{Number: "2", Name: "Task 2", DependsOn: []string{"1"}, Files: []string{"internal/executor/a.go"}},
				{Number: "3", Name: "Task 3", DependsOn: []string{"1"}, Files: []string{"internal/executor/b.go"}},
			},
			wantErr: true,
			errMsg:  "internal/executor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateWaves(tt.tasks)
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculateWaves() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error to contain %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}
