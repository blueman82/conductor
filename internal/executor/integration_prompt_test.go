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
	// Should contain WHY justification
	assert.Contains(t, result, "WHY")
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
