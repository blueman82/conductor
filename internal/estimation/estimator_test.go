package estimation

import (
	"testing"
	"time"

	"github.com/harrison/conductor/internal/models"
)

func TestEstimationSchema(t *testing.T) {
	schema := EstimationSchema()
	if schema == "" {
		t.Error("EstimationSchema() returned empty string")
	}

	// Verify it contains expected fields
	if !contains(schema, "estimate_minutes") {
		t.Error("Schema missing estimate_minutes field")
	}
	if !contains(schema, "reasoning") {
		t.Error("Schema missing reasoning field")
	}
	if !contains(schema, "confidence") {
		t.Error("Schema missing confidence field")
	}
}

func TestNewEstimator(t *testing.T) {
	estimator := NewEstimator(30*time.Second, nil)
	if estimator == nil {
		t.Fatal("NewEstimator returned nil")
	}
}

func TestBuildPrompt(t *testing.T) {
	estimator := NewEstimator(30*time.Second, nil)

	task := &models.Task{
		Name:   "Add user authentication",
		Prompt: "Implement JWT-based authentication for the API",
		Files:  []string{"internal/auth/jwt.go", "internal/auth/middleware.go"},
		SuccessCriteria: []string{
			"JWT tokens are generated on login",
			"Protected routes require valid token",
		},
	}

	prompt := estimator.buildPrompt(task)

	// Verify prompt contains task details
	if !contains(prompt, task.Name) {
		t.Error("Prompt missing task name")
	}
	if !contains(prompt, task.Prompt) {
		t.Error("Prompt missing task description")
	}
	if !contains(prompt, "jwt.go") {
		t.Error("Prompt missing files")
	}
	if !contains(prompt, "Reading") {
		t.Error("Prompt missing development cycle considerations")
	}
}

func TestBuildPromptLongPromptTruncation(t *testing.T) {
	estimator := NewEstimator(30*time.Second, nil)

	// Create a very long prompt
	longDescription := ""
	for i := 0; i < 500; i++ {
		longDescription += "This is a long description. "
	}

	task := &models.Task{
		Name:   "Long task",
		Prompt: longDescription,
	}

	prompt := estimator.buildPrompt(task)

	// Verify prompt is truncated (should have "..." at end of description)
	if !contains(prompt, "...") {
		t.Error("Long prompt should be truncated")
	}
}

func TestBuildPromptMinimalTask(t *testing.T) {
	estimator := NewEstimator(30*time.Second, nil)

	task := &models.Task{
		Name: "Minimal task",
	}

	prompt := estimator.buildPrompt(task)

	// Should still generate valid prompt
	if prompt == "" {
		t.Error("buildPrompt returned empty for minimal task")
	}
	if !contains(prompt, task.Name) {
		t.Error("Prompt missing task name")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
