package parser

import (
	"strings"
	"testing"

	"github.com/harrison/conductor/internal/models"
)

func TestValidateTaskType(t *testing.T) {
	tests := []struct {
		name      string
		taskType  string
		expectErr bool
	}{
		{"empty type is valid", "", false},
		{"regular type", "regular", false},
		{"integration type", "integration", false},
		{"case insensitive regular", "REGULAR", false},
		{"case insensitive integration", "Integration", false},
		{"whitespace trimmed", "  regular  ", false},
		{"invalid type", "invalid", true},
		{"unknown type", "custom", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &models.Task{
				Number: "1",
				Name:   "Test Task",
				Prompt: "Test",
				Type:   tt.taskType,
			}

			err := ValidateTaskType(task)
			if tt.expectErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}

			// Check normalization for valid types
			if !tt.expectErr && tt.taskType != "" {
				expectedNormalized := "regular"
				if tt.taskType == "integration" || tt.taskType == "Integration" || tt.taskType == "INTEGRATION" {
					expectedNormalized = "integration"
				}
				if task.Type != expectedNormalized {
					t.Errorf("expected normalized type %q, got %q", expectedNormalized, task.Type)
				}
			}
		})
	}
}

func TestValidateIntegrationTask(t *testing.T) {
	tests := []struct {
		name      string
		taskType  string
		dependsOn []string
		expectErr bool
	}{
		{"non-integration task with no deps", "regular", []string{}, false},
		{"non-integration task with deps", "regular", []string{"1"}, false},
		{"integration task with deps", "integration", []string{"1", "2"}, false},
		{"integration task without deps", "integration", []string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &models.Task{
				Number:    "3",
				Name:      "Test Task",
				Prompt:    "Test",
				Type:      tt.taskType,
				DependsOn: tt.dependsOn,
			}

			err := ValidateIntegrationTask(task)
			if tt.expectErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestYAMLParserWithTypeField(t *testing.T) {
	tests := []struct {
		name         string
		yamlContent  string
		expectErr    bool
		expectedType string
	}{
		{
			name: "regular task type",
			yamlContent: `
plan:
  tasks:
    - task_number: 1
      name: "Regular Task"
      files: [test.go]
      depends_on: []
      estimated_time: "30m"
      description: "Regular task"
      type: "regular"
`,
			expectErr:    false,
			expectedType: "regular",
		},
		{
			name: "integration task type with deps",
			yamlContent: `
plan:
  tasks:
    - task_number: 1
      name: "First Task"
      files: [test1.go]
      depends_on: []
      estimated_time: "30m"
      description: "First"
    - task_number: 2
      name: "Integration Task"
      files: [test2.go]
      depends_on: [1]
      estimated_time: "45m"
      description: "Integration task"
      type: "integration"
`,
			expectErr:    false,
			expectedType: "integration",
		},
		{
			name: "integration task without deps - should error",
			yamlContent: `
plan:
  tasks:
    - task_number: 1
      name: "Integration Task"
      files: [test.go]
      depends_on: []
      estimated_time: "30m"
      description: "Integration without deps"
      type: "integration"
`,
			expectErr: true,
		},
		{
			name: "invalid task type",
			yamlContent: `
plan:
  tasks:
    - task_number: 1
      name: "Invalid Task"
      files: [test.go]
      depends_on: []
      estimated_time: "30m"
      description: "Invalid type"
      type: "invalid"
`,
			expectErr: true,
		},
		{
			name: "task without type field",
			yamlContent: `
plan:
  tasks:
    - task_number: 1
      name: "No Type Task"
      files: [test.go]
      depends_on: []
      estimated_time: "30m"
      description: "No type specified"
`,
			expectErr:    false,
			expectedType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewYAMLParser()
			plan, err := parser.Parse(strings.NewReader(tt.yamlContent))

			if tt.expectErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(plan.Tasks) == 0 {
				t.Fatal("expected at least one task")
			}

			// Check the last task (for multi-task tests)
			task := plan.Tasks[len(plan.Tasks)-1]
			if task.Type != tt.expectedType {
				t.Errorf("expected type %q, got %q", tt.expectedType, task.Type)
			}
		})
	}
}

func TestYAMLParserWithIntegrationCriteria(t *testing.T) {
	tests := []struct {
		name                    string
		yamlContent             string
		expectedCriteriaCount   int
		expectedFirstCriterion  string
		expectedSecondCriterion string
	}{
		{
			name: "integration task with criteria",
			yamlContent: `
plan:
  tasks:
    - task_number: 1
      name: "First Task"
      files: [test1.go]
      depends_on: []
      estimated_time: "30m"
      description: "First"
    - task_number: 2
      name: "Integration Task"
      files: [test2.go]
      depends_on: [1]
      estimated_time: "45m"
      description: "Integration task"
      type: "integration"
      integration_criteria:
        - "Must read dependency files before implementation"
        - "Must verify interface compatibility"
`,
			expectedCriteriaCount:   2,
			expectedFirstCriterion:  "Must read dependency files before implementation",
			expectedSecondCriterion: "Must verify interface compatibility",
		},
		{
			name: "task without integration criteria",
			yamlContent: `
plan:
  tasks:
    - task_number: 1
      name: "Regular Task"
      files: [test.go]
      depends_on: []
      estimated_time: "30m"
      description: "Regular task"
`,
			expectedCriteriaCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewYAMLParser()
			plan, err := parser.Parse(strings.NewReader(tt.yamlContent))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check the last task
			task := plan.Tasks[len(plan.Tasks)-1]
			if len(task.IntegrationCriteria) != tt.expectedCriteriaCount {
				t.Errorf("expected %d criteria, got %d", tt.expectedCriteriaCount, len(task.IntegrationCriteria))
			}

			if tt.expectedCriteriaCount > 0 {
				if task.IntegrationCriteria[0] != tt.expectedFirstCriterion {
					t.Errorf("first criterion mismatch: expected %q, got %q", tt.expectedFirstCriterion, task.IntegrationCriteria[0])
				}
			}

			if tt.expectedCriteriaCount > 1 {
				if task.IntegrationCriteria[1] != tt.expectedSecondCriterion {
					t.Errorf("second criterion mismatch: expected %q, got %q", tt.expectedSecondCriterion, task.IntegrationCriteria[1])
				}
			}
		})
	}
}
