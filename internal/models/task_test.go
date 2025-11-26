package models

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestTask_WithSuccessCriteria(t *testing.T) {
	task := Task{
		Number: "1",
		Name:   "Test task",
		Prompt: "Do something",
		SuccessCriteria: []string{
			"Criterion 1",
			"Criterion 2",
		},
		TestCommands: []string{"go test ./..."},
	}

	err := task.Validate()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if len(task.SuccessCriteria) != 2 {
		t.Errorf("expected 2 criteria, got: %d", len(task.SuccessCriteria))
	}
	if len(task.TestCommands) != 1 {
		t.Errorf("expected 1 test command, got: %d", len(task.TestCommands))
	}
}

func TestTask_EmptySuccessCriteria_BackwardCompatible(t *testing.T) {
	task := Task{
		Number: "1",
		Name:   "Legacy task",
		Prompt: "Do something",
	}

	err := task.Validate()
	if err != nil {
		t.Errorf("legacy task should be valid: %v", err)
	}
	if task.SuccessCriteria != nil && len(task.SuccessCriteria) > 0 {
		t.Error("empty criteria should be nil or empty slice")
	}
	if task.TestCommands != nil && len(task.TestCommands) > 0 {
		t.Error("empty test commands should be nil or empty slice")
	}
}

func TestTask_Validate_RequiresNumber(t *testing.T) {
	task := Task{
		Name:   "Test",
		Prompt: "Do something",
	}
	err := task.Validate()
	if err == nil {
		t.Error("expected error for missing number")
	}
}

func TestTask_Validate_RequiresName(t *testing.T) {
	task := Task{
		Number: "1",
		Prompt: "Do something",
	}
	err := task.Validate()
	if err == nil {
		t.Error("expected error for missing name")
	}
}

func TestTask_Validate_RequiresPrompt(t *testing.T) {
	task := Task{
		Number: "1",
		Name:   "Test",
	}
	err := task.Validate()
	if err == nil {
		t.Error("expected error for missing prompt")
	}
}

func TestTask_IsIntegration(t *testing.T) {
	tests := []struct {
		name     string
		taskType string
		expected bool
	}{
		{
			name:     "integration type",
			taskType: "integration",
			expected: true,
		},
		{
			name:     "regular type",
			taskType: "regular",
			expected: false,
		},
		{
			name:     "empty type",
			taskType: "",
			expected: false,
		},
		{
			name:     "other type",
			taskType: "setup",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := Task{
				Number: "1",
				Name:   "Task Name",
				Prompt: "test prompt",
				Type:   tt.taskType,
			}
			result := task.IsIntegration()
			if result != tt.expected {
				t.Errorf("IsIntegration() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// CrossFileDependency Tests

func TestCrossFileDependency_String(t *testing.T) {
	tests := []struct {
		name     string
		dep      CrossFileDependency
		expected string
	}{
		{
			name: "basic cross-file dependency",
			dep: CrossFileDependency{
				File:   "plan-01-foundation.yaml",
				TaskID: "2",
			},
			expected: "file:plan-01-foundation.yaml:task:2",
		},
		{
			name: "another cross-file dependency",
			dep: CrossFileDependency{
				File:   "setup.md",
				TaskID: "5",
			},
			expected: "file:setup.md:task:5",
		},
		{
			name: "alphanumeric task ID",
			dep: CrossFileDependency{
				File:   "plan-alpha.yaml",
				TaskID: "task-a",
			},
			expected: "file:plan-alpha.yaml:task:task-a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dep.String()
			if result != tt.expected {
				t.Errorf("String() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

// Dependency Normalization Tests

func TestIsCrossFileDep(t *testing.T) {
	tests := []struct {
		name     string
		dep      string
		expected bool
	}{
		{
			name:     "valid cross-file dependency",
			dep:      "file:plan-01.yaml:task:2",
			expected: true,
		},
		{
			name:     "another valid cross-file dependency",
			dep:      "file:setup.md:task:5",
			expected: true,
		},
		{
			name:     "numeric dependency",
			dep:      "1",
			expected: false,
		},
		{
			name:     "string dependency",
			dep:      "my-task",
			expected: false,
		},
		{
			name:     "file prefix only",
			dep:      "file:plan-01.yaml",
			expected: false,
		},
		{
			name:     "empty string",
			dep:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCrossFileDep(tt.dep)
			if result != tt.expected {
				t.Errorf("IsCrossFileDep(%q) = %v, expected %v", tt.dep, result, tt.expected)
			}
		})
	}
}

func TestParseCrossFileDep(t *testing.T) {
	tests := []struct {
		name          string
		dep           string
		expectedFile  string
		expectedTask  string
		shouldError   bool
		errorContains string
	}{
		{
			name:         "valid cross-file dependency",
			dep:          "file:plan-01-foundation.yaml:task:2",
			expectedFile: "plan-01-foundation.yaml",
			expectedTask: "2",
			shouldError:  false,
		},
		{
			name:         "cross-file with different task ID",
			dep:          "file:setup.md:task:5",
			expectedFile: "setup.md",
			expectedTask: "5",
			shouldError:  false,
		},
		{
			name:         "cross-file with alphanumeric task",
			dep:          "file:complex-plan.yaml:task:task-a",
			expectedFile: "complex-plan.yaml",
			expectedTask: "task-a",
			shouldError:  false,
		},
		{
			name:          "numeric dependency (not cross-file)",
			dep:           "1",
			shouldError:   true,
			errorContains: "not a valid cross-file dependency",
		},
		{
			name:          "string dependency (not cross-file)",
			dep:           "my-task",
			shouldError:   true,
			errorContains: "not a valid cross-file dependency",
		},
		{
			name:          "missing file field",
			dep:           "task:2",
			shouldError:   true,
			errorContains: "not a valid cross-file dependency",
		},
		{
			name:          "empty file name",
			dep:           "file::task:2",
			shouldError:   true,
			errorContains: "empty filename",
		},
		{
			name:          "empty task ID",
			dep:           "file:plan.yaml:task:",
			shouldError:   true,
			errorContains: "empty task ID",
		},
		{
			name:          "empty string",
			dep:           "",
			shouldError:   true,
			errorContains: "not a valid cross-file dependency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseCrossFileDep(tt.dep)
			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result.File != tt.expectedFile {
					t.Errorf("File = %q, expected %q", result.File, tt.expectedFile)
				}
				if result.TaskID != tt.expectedTask {
					t.Errorf("TaskID = %q, expected %q", result.TaskID, tt.expectedTask)
				}
			}
		})
	}
}

func TestNormalizeDependency(t *testing.T) {
	tests := []struct {
		name          string
		dep           interface{}
		expected      string
		shouldError   bool
		errorContains string
	}{
		{
			name:        "integer dependency",
			dep:         1,
			expected:    "1",
			shouldError: false,
		},
		{
			name:        "float64 whole number",
			dep:         2.0,
			expected:    "2",
			shouldError: false,
		},
		{
			name:        "float64 with decimal",
			dep:         2.5,
			expected:    "2.5",
			shouldError: false,
		},
		{
			name:        "string dependency",
			dep:         "task-a",
			expected:    "task-a",
			shouldError: false,
		},
		{
			name: "CrossFileDependency struct",
			dep: CrossFileDependency{
				File:   "plan-01.yaml",
				TaskID: "2",
			},
			expected:    "file:plan-01.yaml:task:2",
			shouldError: false,
		},
		{
			name: "CrossFileDependency pointer",
			dep: &CrossFileDependency{
				File:   "setup.md",
				TaskID: "5",
			},
			expected:    "file:setup.md:task:5",
			shouldError: false,
		},
		{
			name: "map representation of cross-file dep",
			dep: map[string]interface{}{
				"file": "plan-02.yaml",
				"task": 3,
			},
			expected:    "file:plan-02.yaml:task:3",
			shouldError: false,
		},
		{
			name:          "unsupported type (bool)",
			dep:           true,
			shouldError:   true,
			errorContains: "unsupported dependency format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeDependency(tt.dep)
			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("normalized = %q, expected %q", result, tt.expected)
				}
			}
		})
	}
}

// YAML Unmarshaling Tests

func TestTask_UnmarshalYAML_NumericOnly(t *testing.T) {
	yamlContent := `
number: "1"
name: "Test Task"
prompt: "Do something"
depends_on: [1, 2, 3]
`
	var task Task
	err := yaml.Unmarshal([]byte(yamlContent), &task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(task.DependsOn) != 3 {
		t.Errorf("expected 3 dependencies, got %d", len(task.DependsOn))
	}
	if task.DependsOn[0] != "1" || task.DependsOn[1] != "2" || task.DependsOn[2] != "3" {
		t.Errorf("unexpected dependencies: %v", task.DependsOn)
	}
}

func TestTask_UnmarshalYAML_CrossFileDep(t *testing.T) {
	yamlContent := `
number: "2"
name: "Another Task"
prompt: "Do another thing"
depends_on:
  - file: plan-01-foundation.yaml
    task: 2
`
	var task Task
	err := yaml.Unmarshal([]byte(yamlContent), &task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(task.DependsOn) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(task.DependsOn))
	}

	expected := "file:plan-01-foundation.yaml:task:2"
	if task.DependsOn[0] != expected {
		t.Errorf("dependency = %q, expected %q", task.DependsOn[0], expected)
	}

	// Verify we can parse it back
	parsed, err := ParseCrossFileDep(task.DependsOn[0])
	if err != nil {
		t.Errorf("failed to parse cross-file dependency: %v", err)
	}
	if parsed.File != "plan-01-foundation.yaml" || parsed.TaskID != "2" {
		t.Errorf("parsed incorrectly: %+v", parsed)
	}
}

func TestTask_UnmarshalYAML_MixedDeps(t *testing.T) {
	yamlContent := `
number: "4"
name: "Integration Task"
prompt: "Integrate things"
depends_on:
  - 1
  - file: plan-01-foundation.yaml
    task: 2
  - 3
  - file: setup.md
    task: 5
`
	var task Task
	err := yaml.Unmarshal([]byte(yamlContent), &task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(task.DependsOn) != 4 {
		t.Errorf("expected 4 dependencies, got %d", len(task.DependsOn))
	}

	// Check first dependency (local)
	if task.DependsOn[0] != "1" {
		t.Errorf("first dep = %q, expected %q", task.DependsOn[0], "1")
	}

	// Check second dependency (cross-file)
	if task.DependsOn[1] != "file:plan-01-foundation.yaml:task:2" {
		t.Errorf("second dep = %q, expected cross-file format", task.DependsOn[1])
	}

	// Check third dependency (local)
	if task.DependsOn[2] != "3" {
		t.Errorf("third dep = %q, expected %q", task.DependsOn[2], "3")
	}

	// Check fourth dependency (cross-file)
	if task.DependsOn[3] != "file:setup.md:task:5" {
		t.Errorf("fourth dep = %q, expected cross-file format", task.DependsOn[3])
	}
}

func TestTask_UnmarshalYAML_FloatDeps(t *testing.T) {
	yamlContent := `
number: "1"
name: "Float Dep Task"
prompt: "Test floats"
depends_on: [1.0, 2.0, 3.0]
`
	var task Task
	err := yaml.Unmarshal([]byte(yamlContent), &task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(task.DependsOn) != 3 {
		t.Errorf("expected 3 dependencies, got %d", len(task.DependsOn))
	}
	// Float 1.0 should be converted to string "1"
	if task.DependsOn[0] != "1" {
		t.Errorf("first dep = %q, expected %q", task.DependsOn[0], "1")
	}
}

func TestTask_UnmarshalYAML_NoDeps(t *testing.T) {
	yamlContent := `
number: "1"
name: "No Deps Task"
prompt: "Simple task"
`
	var task Task
	err := yaml.Unmarshal([]byte(yamlContent), &task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.DependsOn != nil && len(task.DependsOn) != 0 {
		t.Errorf("expected no dependencies, got %v", task.DependsOn)
	}
}

func TestTask_UnmarshalYAML_StringDeps(t *testing.T) {
	yamlContent := `
number: "1"
name: "String Dep Task"
prompt: "Test strings"
depends_on: ["task-a", "task-b"]
`
	var task Task
	err := yaml.Unmarshal([]byte(yamlContent), &task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(task.DependsOn) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(task.DependsOn))
	}
	if task.DependsOn[0] != "task-a" || task.DependsOn[1] != "task-b" {
		t.Errorf("unexpected dependencies: %v", task.DependsOn)
	}
}

func TestTask_UnmarshalYAML_InvalidCrossFileDep(t *testing.T) {
	tests := []struct {
		name          string
		yamlContent   string
		errorContains string
	}{
		{
			name: "missing file field",
			yamlContent: `
number: "1"
name: "Bad Task"
prompt: "Test"
depends_on:
  - task: 2
`,
			errorContains: "missing required 'file'",
		},
		{
			name: "missing task field",
			yamlContent: `
number: "1"
name: "Bad Task"
prompt: "Test"
depends_on:
  - file: plan-01.yaml
`,
			errorContains: "missing required 'task'",
		},
		{
			name: "non-string file",
			yamlContent: `
number: "1"
name: "Bad Task"
prompt: "Test"
depends_on:
  - file: 123
    task: 2
`,
			errorContains: "'file' must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var task Task
			err := yaml.Unmarshal([]byte(tt.yamlContent), &task)
			if err == nil {
				t.Errorf("expected error, got nil")
			} else if !contains(err.Error(), tt.errorContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errorContains)
			}
		})
	}
}

// Backward Compatibility Tests

func TestTask_BackwardCompat_NumericOnlyYAML(t *testing.T) {
	// Test backward compatibility: numeric-only dependencies in YAML
	// Verifies that existing plans with pure numeric dependencies continue to work
	yamlContent := `
number: "5"
name: "Old Style Task"
prompt: "An old task"
depends_on: [1, 2]
agent: golang-pro
`
	var task Task
	err := yaml.Unmarshal([]byte(yamlContent), &task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Custom unmarshaler should handle depends_on normalization
	// even if other fields aren't populated (they don't have YAML tags)
	if len(task.DependsOn) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(task.DependsOn))
	}
	if task.DependsOn[0] != "1" || task.DependsOn[1] != "2" {
		t.Errorf("dependencies not correctly parsed: expected [1 2], got %v", task.DependsOn)
	}
}

func TestTask_BackwardCompat_NoBreakingChanges(t *testing.T) {
	// Ensure existing Task struct fields still work
	task := Task{
		Number:          "1",
		Name:            "Test",
		Prompt:          "Test prompt",
		Files:           []string{"file.go"},
		DependsOn:       []string{"2", "3"},
		Agent:           "golang-pro",
		WorktreeGroup:   "backend",
		Status:          "pending",
		SuccessCriteria: []string{"Criterion 1"},
	}

	// These should all work as before
	if task.Number != "1" {
		t.Error("Number field broken")
	}
	if len(task.DependsOn) != 2 {
		t.Error("DependsOn field broken")
	}
	if task.Agent != "golang-pro" {
		t.Error("Agent field broken")
	}
	if task.IsCompleted() {
		t.Error("IsCompleted() broken")
	}
	if !task.CanSkip() && task.Status != "pending" {
		t.Error("CanSkip() logic broken")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexStr(s, substr) >= 0))
}

func indexStr(s, substr string) int {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
