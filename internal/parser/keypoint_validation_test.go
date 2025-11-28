package parser

import (
	"testing"

	"github.com/harrison/conductor/internal/models"
)

func TestValidateKeyPointCriteriaAlignment_WarnMode(t *testing.T) {
	tests := []struct {
		name             string
		tasks            []models.Task
		mode             string
		expectedWarnings int
		expectedErrors   int
	}{
		{
			name: "more key_points than criteria returns warning",
			tasks: []models.Task{
				{
					Number: "1",
					Name:   "Task with misalignment",
					KeyPoints: []models.KeyPoint{
						{Point: "key point 1"},
						{Point: "key point 2"},
						{Point: "key point 3"},
					},
					SuccessCriteria: []string{"criterion 1"},
				},
			},
			mode:             CriteriaAlignmentWarn,
			expectedWarnings: 1,
			expectedErrors:   0,
		},
		{
			name: "empty mode defaults to warn",
			tasks: []models.Task{
				{
					Number: "1",
					Name:   "Task with misalignment",
					KeyPoints: []models.KeyPoint{
						{Point: "key point 1"},
						{Point: "key point 2"},
					},
					SuccessCriteria: []string{"criterion 1"},
				},
			},
			mode:             "",
			expectedWarnings: 1,
			expectedErrors:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, errors := ValidateKeyPointCriteriaAlignment(tt.tasks, tt.mode)
			if len(warnings) != tt.expectedWarnings {
				t.Errorf("expected %d warnings, got %d: %v", tt.expectedWarnings, len(warnings), warnings)
			}
			if len(errors) != tt.expectedErrors {
				t.Errorf("expected %d errors, got %d: %v", tt.expectedErrors, len(errors), errors)
			}
		})
	}
}

func TestValidateKeyPointCriteriaAlignment_StrictMode(t *testing.T) {
	tests := []struct {
		name             string
		tasks            []models.Task
		expectedWarnings int
		expectedErrors   int
	}{
		{
			name: "more key_points than criteria returns error in strict mode",
			tasks: []models.Task{
				{
					Number: "1",
					Name:   "Task with misalignment",
					KeyPoints: []models.KeyPoint{
						{Point: "key point 1"},
						{Point: "key point 2"},
						{Point: "key point 3"},
					},
					SuccessCriteria: []string{"criterion 1"},
				},
			},
			expectedWarnings: 0,
			expectedErrors:   1,
		},
		{
			name: "multiple tasks with misalignment returns multiple errors",
			tasks: []models.Task{
				{
					Number: "1",
					Name:   "Task 1",
					KeyPoints: []models.KeyPoint{
						{Point: "key point 1"},
						{Point: "key point 2"},
					},
					SuccessCriteria: []string{"criterion 1"},
				},
				{
					Number: "2",
					Name:   "Task 2",
					KeyPoints: []models.KeyPoint{
						{Point: "key point 1"},
						{Point: "key point 2"},
						{Point: "key point 3"},
					},
					SuccessCriteria: []string{},
				},
			},
			expectedWarnings: 0,
			expectedErrors:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, errors := ValidateKeyPointCriteriaAlignment(tt.tasks, CriteriaAlignmentStrict)
			if len(warnings) != tt.expectedWarnings {
				t.Errorf("expected %d warnings, got %d: %v", tt.expectedWarnings, len(warnings), warnings)
			}
			if len(errors) != tt.expectedErrors {
				t.Errorf("expected %d errors, got %d: %v", tt.expectedErrors, len(errors), errors)
			}
		})
	}
}

func TestValidateKeyPointCriteriaAlignment_OffMode(t *testing.T) {
	tasks := []models.Task{
		{
			Number: "1",
			Name:   "Task with misalignment",
			KeyPoints: []models.KeyPoint{
				{Point: "key point 1"},
				{Point: "key point 2"},
				{Point: "key point 3"},
			},
			SuccessCriteria: []string{},
		},
	}

	warnings, errors := ValidateKeyPointCriteriaAlignment(tasks, CriteriaAlignmentOff)
	if warnings != nil {
		t.Errorf("expected nil warnings in off mode, got %v", warnings)
	}
	if errors != nil {
		t.Errorf("expected nil errors in off mode, got %v", errors)
	}
}

func TestValidateKeyPointCriteriaAlignment_NoKeyPoints(t *testing.T) {
	tests := []struct {
		name  string
		tasks []models.Task
	}{
		{
			name: "task with no key_points is skipped",
			tasks: []models.Task{
				{
					Number:          "1",
					Name:            "Task without key points",
					KeyPoints:       []models.KeyPoint{},
					SuccessCriteria: []string{"criterion 1"},
				},
			},
		},
		{
			name: "task with nil key_points is skipped",
			tasks: []models.Task{
				{
					Number:          "1",
					Name:            "Task without key points",
					KeyPoints:       nil,
					SuccessCriteria: []string{"criterion 1"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, errors := ValidateKeyPointCriteriaAlignment(tt.tasks, CriteriaAlignmentWarn)
			if len(warnings) != 0 {
				t.Errorf("expected 0 warnings for task without key_points, got %d: %v", len(warnings), warnings)
			}
			if len(errors) != 0 {
				t.Errorf("expected 0 errors for task without key_points, got %d: %v", len(errors), errors)
			}
		})
	}
}

func TestValidateKeyPointCriteriaAlignment_EqualCounts(t *testing.T) {
	tasks := []models.Task{
		{
			Number: "1",
			Name:   "Task with equal counts",
			KeyPoints: []models.KeyPoint{
				{Point: "key point 1"},
				{Point: "key point 2"},
			},
			SuccessCriteria: []string{"criterion 1", "criterion 2"},
		},
	}

	warnings, errors := ValidateKeyPointCriteriaAlignment(tasks, CriteriaAlignmentWarn)
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for equal counts, got %d: %v", len(warnings), warnings)
	}
	if len(errors) != 0 {
		t.Errorf("expected 0 errors for equal counts, got %d: %v", len(errors), errors)
	}

	// Also test strict mode
	warnings, errors = ValidateKeyPointCriteriaAlignment(tasks, CriteriaAlignmentStrict)
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings in strict mode for equal counts, got %d: %v", len(warnings), warnings)
	}
	if len(errors) != 0 {
		t.Errorf("expected 0 errors in strict mode for equal counts, got %d: %v", len(errors), errors)
	}
}

func TestValidateKeyPointCriteriaAlignment_MoreCriteria(t *testing.T) {
	tasks := []models.Task{
		{
			Number: "1",
			Name:   "Task with more criteria than key_points",
			KeyPoints: []models.KeyPoint{
				{Point: "key point 1"},
			},
			SuccessCriteria: []string{"criterion 1", "criterion 2", "criterion 3"},
		},
	}

	warnings, errors := ValidateKeyPointCriteriaAlignment(tasks, CriteriaAlignmentWarn)
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings when more criteria than key_points, got %d: %v", len(warnings), warnings)
	}
	if len(errors) != 0 {
		t.Errorf("expected 0 errors when more criteria than key_points, got %d: %v", len(errors), errors)
	}

	// Also test strict mode
	warnings, errors = ValidateKeyPointCriteriaAlignment(tasks, CriteriaAlignmentStrict)
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings in strict mode when more criteria, got %d: %v", len(warnings), warnings)
	}
	if len(errors) != 0 {
		t.Errorf("expected 0 errors in strict mode when more criteria, got %d: %v", len(errors), errors)
	}
}

func TestValidateKeyPointCriteriaAlignment_IntegrationTask(t *testing.T) {
	tests := []struct {
		name               string
		keyPointCount      int
		successCriteriaLen int
		integrationCritLen int
		expectedWarnings   int
		expectedErrors     int
		description        string
	}{
		{
			name:               "integration task counts both criteria types",
			keyPointCount:      3,
			successCriteriaLen: 2,
			integrationCritLen: 2,
			expectedWarnings:   0, // 3 key_points <= 4 total criteria
			expectedErrors:     0,
			description:        "3 key_points with 2 success + 2 integration = 4 total criteria (valid)",
		},
		{
			name:               "integration task still warns when insufficient",
			keyPointCount:      5,
			successCriteriaLen: 2,
			integrationCritLen: 1,
			expectedWarnings:   1, // 5 key_points > 3 total criteria
			expectedErrors:     0,
			description:        "5 key_points with 2 success + 1 integration = 3 total criteria (warning)",
		},
		{
			name:               "integration task with only integration criteria",
			keyPointCount:      2,
			successCriteriaLen: 0,
			integrationCritLen: 3,
			expectedWarnings:   0, // 2 key_points <= 3 integration criteria
			expectedErrors:     0,
			description:        "2 key_points with 0 success + 3 integration = 3 total criteria (valid)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyPoints := make([]models.KeyPoint, tt.keyPointCount)
			for i := range keyPoints {
				keyPoints[i] = models.KeyPoint{Point: "key point"}
			}

			successCriteria := make([]string, tt.successCriteriaLen)
			for i := range successCriteria {
				successCriteria[i] = "success criterion"
			}

			integrationCriteria := make([]string, tt.integrationCritLen)
			for i := range integrationCriteria {
				integrationCriteria[i] = "integration criterion"
			}

			tasks := []models.Task{
				{
					Number:              "1",
					Name:                "Integration Task",
					Type:                "integration",
					KeyPoints:           keyPoints,
					SuccessCriteria:     successCriteria,
					IntegrationCriteria: integrationCriteria,
				},
			}

			warnings, errors := ValidateKeyPointCriteriaAlignment(tasks, CriteriaAlignmentWarn)
			if len(warnings) != tt.expectedWarnings {
				t.Errorf("%s: expected %d warnings, got %d: %v", tt.description, tt.expectedWarnings, len(warnings), warnings)
			}
			if len(errors) != tt.expectedErrors {
				t.Errorf("%s: expected %d errors, got %d: %v", tt.description, tt.expectedErrors, len(errors), errors)
			}
		})
	}
}

func TestValidateKeyPointCriteriaAlignment_NonIntegrationTaskIgnoresIntegrationCriteria(t *testing.T) {
	// Non-integration tasks should NOT count integration_criteria
	tasks := []models.Task{
		{
			Number: "1",
			Name:   "Regular Task",
			Type:   "", // Not integration type
			KeyPoints: []models.KeyPoint{
				{Point: "key point 1"},
				{Point: "key point 2"},
				{Point: "key point 3"},
			},
			SuccessCriteria:     []string{"criterion 1"},
			IntegrationCriteria: []string{"int criterion 1", "int criterion 2"}, // Should be ignored
		},
	}

	warnings, errors := ValidateKeyPointCriteriaAlignment(tasks, CriteriaAlignmentWarn)
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning (3 key_points > 1 success_criteria), got %d: %v", len(warnings), warnings)
	}
	if len(errors) != 0 {
		t.Errorf("expected 0 errors, got %d: %v", len(errors), errors)
	}
}

func TestValidateKeyPointCriteriaAlignment_MessageFormat(t *testing.T) {
	tasks := []models.Task{
		{
			Number: "42",
			Name:   "Test Task Name",
			KeyPoints: []models.KeyPoint{
				{Point: "key point 1"},
				{Point: "key point 2"},
				{Point: "key point 3"},
			},
			SuccessCriteria: []string{"criterion 1"},
		},
	}

	warnings, _ := ValidateKeyPointCriteriaAlignment(tasks, CriteriaAlignmentWarn)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}

	warning := warnings[0]
	// Check message contains task number and name
	if warning == "" {
		t.Error("warning message should not be empty")
	}
	// The message format: "Task %s (%s): %d key_points but only %d success_criteria..."
	expectedSubstrings := []string{"42", "Test Task Name", "3 key_points", "1 success_criteria", "2 key_point(s)"}
	for _, substr := range expectedSubstrings {
		if !containsString(warning, substr) {
			t.Errorf("warning message should contain '%s', got: %s", substr, warning)
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
