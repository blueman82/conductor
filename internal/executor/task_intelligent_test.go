package executor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/models"
)

// createTestRegistryForTaskSelection creates a test registry with the given agent names.
// Returns the registry and a cleanup function to remove the temp directory.
func createTestRegistryForTaskSelection(t *testing.T, agentNames []string) (*agent.Registry, func()) {
	tmpDir, err := os.MkdirTemp("", "test-agents-task-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}

	// Create minimal agent .md files with YAML frontmatter
	for _, name := range agentNames {
		agentPath := filepath.Join(tmpDir, name+".md")
		content := "---\nname: " + name + "\ndescription: Test agent for " + name + "\n---\n"
		err := os.WriteFile(agentPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write agent file for %s: %v", name, err)
		}
	}

	// Create registry and discover agents
	registry := agent.NewRegistry(tmpDir)
	_, err = registry.Discover()
	if err != nil {
		t.Fatalf("failed to discover agents: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return registry, cleanup
}

func TestNewTaskAgentSelector(t *testing.T) {
	registry, cleanup := createTestRegistryForTaskSelection(t, []string{"golang-pro", "frontend-developer"})
	defer cleanup()

	selector := NewTaskAgentSelector(registry, 90*time.Second, nil)

	if selector == nil {
		t.Fatal("expected non-nil selector")
	}
	if selector.Registry != registry {
		t.Error("expected registry to be set")
	}
	if selector.Invoker() == nil {
		t.Error("expected invoker to be set")
	}
}

func TestNewTaskAgentSelector_CustomTimeout(t *testing.T) {
	registry, cleanup := createTestRegistryForTaskSelection(t, []string{"golang-pro"})
	defer cleanup()

	selector := NewTaskAgentSelector(registry, 45*time.Second, nil)

	// Timeout is set on the internal invoker, verify via method accessor
	if selector.Invoker() == nil {
		t.Fatal("expected invoker to be set")
	}
}

func TestTaskAgentSelector_GetAvailableAgents(t *testing.T) {
	t.Run("with registry", func(t *testing.T) {
		registry, cleanup := createTestRegistryForTaskSelection(t, []string{"golang-pro", "frontend-developer", "code-reviewer"})
		defer cleanup()

		selector := NewTaskAgentSelector(registry, 90*time.Second, nil)
		agents := selector.getAvailableAgents()

		if len(agents) != 3 {
			t.Errorf("expected 3 agents, got %d", len(agents))
		}

		// Check that all expected agents are present (order may vary)
		agentSet := make(map[string]bool)
		for _, a := range agents {
			agentSet[a] = true
		}
		for _, expected := range []string{"golang-pro", "frontend-developer", "code-reviewer"} {
			if !agentSet[expected] {
				t.Errorf("expected agent %q not found", expected)
			}
		}
	})

	t.Run("with nil registry", func(t *testing.T) {
		selector := &TaskAgentSelector{Registry: nil}
		agents := selector.getAvailableAgents()

		if len(agents) != 0 {
			t.Errorf("expected 0 agents with nil registry, got %d", len(agents))
		}
	})
}

func TestTaskAgentSelector_AgentExists(t *testing.T) {
	registry, cleanup := createTestRegistryForTaskSelection(t, []string{"golang-pro", "frontend-developer"})
	defer cleanup()

	selector := NewTaskAgentSelector(registry, 90*time.Second, nil)

	t.Run("existing agent", func(t *testing.T) {
		if !selector.agentExists("golang-pro") {
			t.Error("expected golang-pro to exist")
		}
		if !selector.agentExists("frontend-developer") {
			t.Error("expected frontend-developer to exist")
		}
	})

	t.Run("non-existing agent", func(t *testing.T) {
		if selector.agentExists("nonexistent-agent") {
			t.Error("expected nonexistent-agent to not exist")
		}
	})

	t.Run("with nil registry", func(t *testing.T) {
		nilSelector := &TaskAgentSelector{Registry: nil}
		if nilSelector.agentExists("any-agent") {
			t.Error("expected agent to not exist with nil registry")
		}
	})
}

func TestTaskAgentSelector_BuildSelectionPrompt(t *testing.T) {
	registry, cleanup := createTestRegistryForTaskSelection(t, []string{"golang-pro", "frontend-developer"})
	defer cleanup()

	selector := NewTaskAgentSelector(registry, 90*time.Second, nil)

	t.Run("basic task", func(t *testing.T) {
		task := models.Task{
			Number: "1",
			Name:   "Implement login API",
			Files:  []string{"internal/api/auth.go", "internal/api/handlers.go"},
			Prompt: "Implement the user authentication endpoint",
		}

		prompt := selector.buildSelectionPrompt(task, []string{"golang-pro", "frontend-developer"})

		// Verify prompt contains key information
		if !strings.Contains(prompt, "Task Number: 1") {
			t.Error("expected prompt to contain task number")
		}
		if !strings.Contains(prompt, "Task Name: Implement login API") {
			t.Error("expected prompt to contain task name")
		}
		if !strings.Contains(prompt, "internal/api/auth.go") {
			t.Error("expected prompt to contain files")
		}
		if !strings.Contains(prompt, "golang-pro") {
			t.Error("expected prompt to contain available agents")
		}
		if !strings.Contains(prompt, "frontend-developer") {
			t.Error("expected prompt to contain available agents")
		}
	})

	t.Run("task with success criteria", func(t *testing.T) {
		task := models.Task{
			Number: "2",
			Name:   "Add validation",
			Files:  []string{"src/validate.ts"},
			Prompt: "Add input validation",
			SuccessCriteria: []string{
				"Validates email format",
				"Validates password strength",
			},
		}

		prompt := selector.buildSelectionPrompt(task, []string{"frontend-developer"})

		if !strings.Contains(prompt, "SUCCESS CRITERIA") {
			t.Error("expected prompt to contain success criteria section")
		}
		if !strings.Contains(prompt, "Validates email format") {
			t.Error("expected prompt to contain first criterion")
		}
		if !strings.Contains(prompt, "Validates password strength") {
			t.Error("expected prompt to contain second criterion")
		}
	})

	t.Run("integration task", func(t *testing.T) {
		task := models.Task{
			Number: "3",
			Name:   "Wire auth with API",
			Type:   "integration",
			Files:  []string{"internal/api/main.go"},
			Prompt: "Connect authentication with API handlers",
		}

		prompt := selector.buildSelectionPrompt(task, []string{"fullstack-developer"})

		if !strings.Contains(prompt, "INTEGRATION task") {
			t.Error("expected prompt to mention integration task")
		}
	})
}

func TestTaskAgentSelectionSchema(t *testing.T) {
	schema := TaskAgentSelectionSchema()

	// Verify schema contains required fields
	if !strings.Contains(schema, `"type": "object"`) {
		t.Error("expected schema to define object type")
	}
	if !strings.Contains(schema, `"agent"`) {
		t.Error("expected schema to contain agent field")
	}
	if !strings.Contains(schema, `"rationale"`) {
		t.Error("expected schema to contain rationale field")
	}
	if !strings.Contains(schema, `"required"`) {
		t.Error("expected schema to have required fields")
	}
}

func TestTaskAgentSelector_SelectAgent_EmptyRegistry(t *testing.T) {
	// Create empty temp directory
	tmpDir, err := os.MkdirTemp("", "test-empty-agents-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create registry with no agents
	registry := agent.NewRegistry(tmpDir)
	selector := NewTaskAgentSelector(registry, 90*time.Second, nil)

	task := models.Task{
		Number: "1",
		Name:   "Test task",
		Files:  []string{"test.go"},
	}

	_, err = selector.SelectAgent(context.Background(), task)
	if err == nil {
		t.Error("expected error with empty registry")
	}
	if !strings.Contains(err.Error(), "no agents available") {
		t.Errorf("expected 'no agents available' error, got: %v", err)
	}
}

func TestTaskAgentSelector_SelectAgent_WithNilRegistry(t *testing.T) {
	// Create selector with nil registry
	// The embedded Service will have zero values (nil invoker, nil logger)
	// But with nil registry, it errors early before invoking Claude
	selector := &TaskAgentSelector{
		Registry: nil,
	}

	task := models.Task{
		Number: "1",
		Name:   "Test task",
		Files:  []string{"test.go"},
	}

	_, err := selector.SelectAgent(context.Background(), task)
	if err == nil {
		t.Error("expected error with nil registry")
	}
	if !strings.Contains(err.Error(), "no agents available") {
		t.Errorf("expected 'no agents available' error, got: %v", err)
	}
}

func TestTaskAgentSelectionResult_Struct(t *testing.T) {
	// Test that the struct can be created and fields are accessible
	result := TaskAgentSelectionResult{
		Agent:     "golang-pro",
		Rationale: "Selected because task involves Go files",
	}

	if result.Agent != "golang-pro" {
		t.Errorf("expected Agent='golang-pro', got %q", result.Agent)
	}
	if result.Rationale != "Selected because task involves Go files" {
		t.Errorf("expected Rationale to be set, got %q", result.Rationale)
	}
}

func TestTaskAgentSelector_FallbackValidation(t *testing.T) {
	// This tests the code path where Claude recommends an agent that doesn't exist
	// The selector should fall back to general-purpose

	registry, cleanup := createTestRegistryForTaskSelection(t, []string{"general-purpose", "golang-pro"})
	defer cleanup()

	selector := NewTaskAgentSelector(registry, 90*time.Second, nil)

	// Simulate result from Claude with non-existent agent
	result := &TaskAgentSelectionResult{
		Agent:     "nonexistent-agent",
		Rationale: "Claude recommended this agent",
	}

	// Verify agentExists check
	if selector.agentExists(result.Agent) {
		t.Error("expected nonexistent-agent to not exist")
	}

	// The actual fallback logic would happen in SelectAgent after Claude invocation
	// Here we just verify the agentExists check works correctly
	if !selector.agentExists("general-purpose") {
		t.Error("expected general-purpose to exist for fallback")
	}
}
