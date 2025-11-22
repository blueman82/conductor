package models

import (
	"testing"
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
