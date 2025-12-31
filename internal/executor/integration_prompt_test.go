package executor

import (
	"testing"

	"github.com/harrison/conductor/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestBuildIntegrationPrompt_SingleDependency(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{
				Number: "1",
				Name:   "Auth Module",
				Files:  []string{"internal/auth/jwt.go", "internal/auth/types.go"},
			},
		},
	}

	task := models.Task{
		Number:    "2",
		Name:      "Wire auth to router",
		DependsOn: []string{"1"},
		Prompt:    "Wire the auth module to the router.",
	}

	result := buildIntegrationPrompt(task, plan)

	// Should contain dependency context header
	assert.Contains(t, result, "INTEGRATION TASK CONTEXT")
	// Should list dependency files
	assert.Contains(t, result, "internal/auth/jwt.go")
	assert.Contains(t, result, "internal/auth/types.go")
	// Should contain justification section (XML format)
	assert.Contains(t, result, "<justification>")
	// Should contain original prompt
	assert.Contains(t, result, "Wire the auth module to the router")
}

func TestBuildIntegrationPrompt_MultipleDependencies(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Auth", Files: []string{"auth.go"}},
			{Number: "2", Name: "Router", Files: []string{"router.go"}},
		},
	}

	task := models.Task{
		Number:    "3",
		Name:      "Integration",
		DependsOn: []string{"1", "2"},
		Prompt:    "Integrate",
	}

	result := buildIntegrationPrompt(task, plan)

	// Should mention both dependencies
	assert.Contains(t, result, "Auth")
	assert.Contains(t, result, "Router")
	assert.Contains(t, result, "auth.go")
	assert.Contains(t, result, "router.go")
}

func TestBuildIntegrationPrompt_NoDependencies(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{},
	}

	task := models.Task{
		Number:    "1",
		Name:      "Standalone",
		DependsOn: []string{},
		Prompt:    "Do standalone work",
	}

	result := buildIntegrationPrompt(task, plan)

	// Should return original prompt without modification
	assert.Equal(t, "Do standalone work", result)
}

func TestBuildIntegrationPrompt_MissingDependency(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Existing", Files: []string{"existing.go"}},
		},
	}

	task := models.Task{
		Number:    "2",
		Name:      "Integration",
		DependsOn: []string{"1", "999"}, // 999 doesn't exist
		Prompt:    "Integrate stuff",
	}

	result := buildIntegrationPrompt(task, plan)

	// Should contain existing dependency
	assert.Contains(t, result, "Existing")
	assert.Contains(t, result, "existing.go")
	// Should still contain original prompt
	assert.Contains(t, result, "Integrate stuff")
	// Should not panic or error
}

// TestBuildIntegrationPrompt_TypeIntegrationWithDependencies
// verifies that integration type tasks with dependencies get prompt enhancement
func TestBuildIntegrationPrompt_TypeIntegrationWithDependencies(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Setup", Files: []string{"setup.go"}},
		},
	}

	task := models.Task{
		Number:    "2",
		Name:      "Integration Task",
		Type:      "integration",
		DependsOn: []string{"1"},
		Prompt:    "Integrate after setup",
	}

	result := buildIntegrationPrompt(task, plan)

	// Should contain dependency context for integration type with dependencies
	assert.Contains(t, result, "INTEGRATION TASK CONTEXT")
	assert.Contains(t, result, "Setup")
	assert.Contains(t, result, "setup.go")
	assert.Contains(t, result, "Integrate after setup")
}

// TestBuildIntegrationPrompt_TypeIntegrationWithoutDependencies
// verifies that integration type tasks WITHOUT dependencies return original prompt
// (no context enhancement when there are no dependencies to document)
func TestBuildIntegrationPrompt_TypeIntegrationWithoutDependencies(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{},
	}

	task := models.Task{
		Number:    "1",
		Name:      "Standalone Integration",
		Type:      "integration",
		DependsOn: []string{},
		Prompt:    "Do standalone integration work",
	}

	result := buildIntegrationPrompt(task, plan)

	// Should return original prompt without enhancement (no dependencies to document)
	assert.Equal(t, "Do standalone integration work", result)
	assert.NotContains(t, result, "INTEGRATION TASK CONTEXT")
}

// TestBuildIntegrationPrompt_RegularTypeWithDependencies
// verifies that regular (non-integration) type tasks WITH dependencies still get prompt enhancement
// This validates the OR condition: Type=="integration" OR len(DependsOn)>0
func TestBuildIntegrationPrompt_RegularTypeWithDependencies(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Feature", Files: []string{"feature.go"}},
		},
	}

	task := models.Task{
		Number:    "2",
		Name:      "Regular Integration",
		Type:      "regular",
		DependsOn: []string{"1"},
		Prompt:    "Implement with dependency",
	}

	result := buildIntegrationPrompt(task, plan)

	// Regular type with dependencies should still get context enhancement
	assert.Contains(t, result, "INTEGRATION TASK CONTEXT")
	assert.Contains(t, result, "Feature")
	assert.Contains(t, result, "feature.go")
	assert.Contains(t, result, "Implement with dependency")
}

// TestBuildIntegrationPrompt_RegularTypeNoDependencies
// verifies that regular (non-integration) type tasks WITHOUT dependencies return original prompt
func TestBuildIntegrationPrompt_RegularTypeNoDependencies(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{},
	}

	task := models.Task{
		Number:    "1",
		Name:      "Standalone Regular",
		Type:      "regular",
		DependsOn: []string{},
		Prompt:    "Do standalone work",
	}

	result := buildIntegrationPrompt(task, plan)

	// Should return original prompt without enhancement
	assert.Equal(t, "Do standalone work", result)
	assert.NotContains(t, result, "INTEGRATION TASK CONTEXT")
}

// TestBuildIntegrationPrompt_EmptyTypeWithDependencies
// verifies that tasks with empty type BUT WITH dependencies get prompt enhancement
func TestBuildIntegrationPrompt_EmptyTypeWithDependencies(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: "1", Name: "Core", Files: []string{"core.go"}},
		},
	}

	task := models.Task{
		Number:    "2",
		Name:      "Dependent Task",
		Type:      "",
		DependsOn: []string{"1"},
		Prompt:    "Depend on core",
	}

	result := buildIntegrationPrompt(task, plan)

	// Empty type but with dependencies should still get context enhancement
	assert.Contains(t, result, "INTEGRATION TASK CONTEXT")
	assert.Contains(t, result, "Core")
	assert.Contains(t, result, "core.go")
	assert.Contains(t, result, "Depend on core")
}

// TestBuildIntegrationPrompt_EmptyTypeNoDependencies
// verifies that tasks with empty type and NO dependencies return original prompt
func TestBuildIntegrationPrompt_EmptyTypeNoDependencies(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{},
	}

	task := models.Task{
		Number:    "1",
		Name:      "Independent",
		Type:      "",
		DependsOn: []string{},
		Prompt:    "Do independent work",
	}

	result := buildIntegrationPrompt(task, plan)

	// Should return original prompt without enhancement
	assert.Equal(t, "Do independent work", result)
	assert.NotContains(t, result, "INTEGRATION TASK CONTEXT")
}
