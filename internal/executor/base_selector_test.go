package executor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/agent"
)

// createTestRegistryForBaseSelector creates a test registry with the given agent names.
func createTestRegistryForBaseSelector(t *testing.T, agentNames []string) (*agent.Registry, func()) {
	tmpDir, err := os.MkdirTemp("", "test-agents-base-*")
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

func TestBaseSelector_GetAvailableAgents(t *testing.T) {
	t.Run("with populated registry", func(t *testing.T) {
		registry, cleanup := createTestRegistryForBaseSelector(t, []string{"golang-pro", "frontend-developer", "security-auditor"})
		defer cleanup()

		selector := &BaseSelector{Registry: registry}
		agents := selector.getAvailableAgents()

		if len(agents) != 3 {
			t.Errorf("expected 3 agents, got %d", len(agents))
		}

		// Check that all expected agents are present (order may vary)
		agentSet := make(map[string]bool)
		for _, a := range agents {
			agentSet[a] = true
		}
		for _, expected := range []string{"golang-pro", "frontend-developer", "security-auditor"} {
			if !agentSet[expected] {
				t.Errorf("expected agent %q not found", expected)
			}
		}
	})

	t.Run("with nil registry", func(t *testing.T) {
		selector := &BaseSelector{Registry: nil}
		agents := selector.getAvailableAgents()

		if len(agents) != 0 {
			t.Errorf("expected 0 agents with nil registry, got %d", len(agents))
		}
	})

	t.Run("with empty registry", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-empty-agents-*")
		if err != nil {
			t.Fatalf("failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		registry := agent.NewRegistry(tmpDir)
		selector := &BaseSelector{Registry: registry}
		agents := selector.getAvailableAgents()

		if len(agents) != 0 {
			t.Errorf("expected 0 agents with empty registry, got %d", len(agents))
		}
	})
}

func TestBaseSelector_InheritedByIntelligentSelector(t *testing.T) {
	registry, cleanup := createTestRegistryForBaseSelector(t, []string{"golang-pro", "code-reviewer"})
	defer cleanup()

	// IntelligentSelector should inherit getAvailableAgents from BaseSelector
	selector := NewIntelligentSelector(registry, 3600, 90*time.Second, nil)
	agents := selector.getAvailableAgents()

	if len(agents) != 2 {
		t.Errorf("IntelligentSelector: expected 2 agents, got %d", len(agents))
	}
}

func TestBaseSelector_InheritedByTaskAgentSelector(t *testing.T) {
	registry, cleanup := createTestRegistryForBaseSelector(t, []string{"golang-pro", "frontend-developer"})
	defer cleanup()

	// TaskAgentSelector should inherit getAvailableAgents from BaseSelector
	selector := NewTaskAgentSelector(registry, 90*time.Second, nil)
	agents := selector.getAvailableAgents()

	if len(agents) != 2 {
		t.Errorf("TaskAgentSelector: expected 2 agents, got %d", len(agents))
	}
}
