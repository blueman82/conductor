package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverAgents(t *testing.T) {
	// Create temp directory with test agent files
	tmpDir := t.TempDir()

	// Write sample agent file
	agentContent := `---
name: test-agent
description: Test agent for unit testing
tools:
  - Read
  - Write
---

# Test Agent

This is a test agent used for unit testing agent discovery.
`
	err := os.WriteFile(filepath.Join(tmpDir, "test-agent.md"), []byte(agentContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry(tmpDir)
	agents, err := registry.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(agents) != 1 {
		t.Errorf("Expected 1 agent, got %d", len(agents))
	}

	if _, exists := agents["test-agent"]; !exists {
		t.Error("Expected test-agent to exist")
	}
}

func TestDiscoverMultipleAgents(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple agent files
	agents := []struct {
		name    string
		content string
	}{
		{
			name: "agent-one.md",
			content: `---
name: agent-one
description: First agent
---
Agent one content
`,
		},
		{
			name: "agent-two.md",
			content: `---
name: agent-two
description: Second agent
tools:
  - Read
  - Bash
---
Agent two content
`,
		},
	}

	for _, a := range agents {
		err := os.WriteFile(filepath.Join(tmpDir, a.name), []byte(a.content), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	registry := NewRegistry(tmpDir)
	discovered, err := registry.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(discovered) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(discovered))
	}

	if !registry.Exists("agent-one") {
		t.Error("agent-one should exist")
	}

	if !registry.Exists("agent-two") {
		t.Error("agent-two should exist")
	}
}

func TestAgentExists(t *testing.T) {
	tmpDir := t.TempDir()

	agentContent := `---
name: swiftdev
description: Swift development agent
---
Swift agent content
`
	err := os.WriteFile(filepath.Join(tmpDir, "swiftdev.md"), []byte(agentContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry(tmpDir)
	_, err = registry.Discover()
	if err != nil {
		t.Fatal(err)
	}

	if !registry.Exists("swiftdev") {
		t.Error("swiftdev agent should exist")
	}

	if registry.Exists("nonexistent-agent") {
		t.Error("nonexistent-agent should not exist")
	}
}

func TestGetAgent(t *testing.T) {
	tmpDir := t.TempDir()

	agentContent := `---
name: test-agent
description: Test agent description
tools:
  - Read
  - Write
  - Edit
---
Test agent body
`
	err := os.WriteFile(filepath.Join(tmpDir, "test-agent.md"), []byte(agentContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry(tmpDir)
	_, err = registry.Discover()
	if err != nil {
		t.Fatal(err)
	}

	agent, exists := registry.Get("test-agent")
	if !exists {
		t.Fatal("test-agent should exist")
	}

	if agent.Name != "test-agent" {
		t.Errorf("Expected name 'test-agent', got '%s'", agent.Name)
	}

	if agent.Description != "Test agent description" {
		t.Errorf("Expected description 'Test agent description', got '%s'", agent.Description)
	}

	if len(agent.Tools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(agent.Tools))
	}

	expectedTools := []string{"Read", "Write", "Edit"}
	for i, tool := range agent.Tools {
		if tool != expectedTools[i] {
			t.Errorf("Expected tool '%s' at index %d, got '%s'", expectedTools[i], i, tool)
		}
	}
}

func TestDiscoverNoAgentsDirectory(t *testing.T) {
	// Use a directory that doesn't exist
	nonExistentDir := filepath.Join(t.TempDir(), "does-not-exist")

	registry := NewRegistry(nonExistentDir)
	agents, err := registry.Discover()

	// Should not return an error, just an empty map
	if err != nil {
		t.Errorf("Expected no error when directory doesn't exist, got: %v", err)
	}

	if len(agents) != 0 {
		t.Errorf("Expected 0 agents from non-existent directory, got %d", len(agents))
	}
}

func TestParseAgentFileInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Invalid YAML frontmatter
	invalidContent := `---
name: test-agent
description: missing closing delimiter
Test agent body
`
	err := os.WriteFile(filepath.Join(tmpDir, "invalid.md"), []byte(invalidContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry(tmpDir)
	agents, err := registry.Discover()

	// Should not fail completely, just skip the invalid file
	if err != nil {
		t.Errorf("Discover should not fail on invalid files, got: %v", err)
	}

	if len(agents) != 0 {
		t.Errorf("Expected 0 agents when file is invalid, got %d", len(agents))
	}
}

func TestParseAgentFileMissingName(t *testing.T) {
	tmpDir := t.TempDir()

	// Missing required 'name' field
	missingNameContent := `---
description: Agent without name
---
Agent body
`
	err := os.WriteFile(filepath.Join(tmpDir, "no-name.md"), []byte(missingNameContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry(tmpDir)
	agents, err := registry.Discover()

	// Should skip files without required name field
	if err != nil {
		t.Errorf("Discover should not fail on files missing name, got: %v", err)
	}

	if len(agents) != 0 {
		t.Errorf("Expected 0 agents when name is missing, got %d", len(agents))
	}
}

func TestDefaultAgentsDirectory(t *testing.T) {
	// Test that NewRegistry uses ~/.claude/agents when empty string is passed
	registry := NewRegistry("")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(home, ".claude", "agents")
	if registry.AgentsDir != expectedDir {
		t.Errorf("Expected AgentsDir '%s', got '%s'", expectedDir, registry.AgentsDir)
	}
}

func TestDiscoverOnlyMarkdownFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .md agent file
	mdContent := `---
name: md-agent
description: Markdown agent
---
MD agent content
`
	err := os.WriteFile(filepath.Join(tmpDir, "agent.md"), []byte(mdContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create non-.md files that should be ignored
	err = os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("Not an agent"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte("Not an agent"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry(tmpDir)
	agents, err := registry.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Should only discover the .md file
	if len(agents) != 1 {
		t.Errorf("Expected 1 agent (only .md files), got %d", len(agents))
	}

	if !registry.Exists("md-agent") {
		t.Error("md-agent should exist")
	}
}

func TestAgentFilePath(t *testing.T) {
	tmpDir := t.TempDir()

	agentFile := filepath.Join(tmpDir, "test-agent.md")
	agentContent := `---
name: test-agent
description: Test agent
---
Test content
`
	err := os.WriteFile(agentFile, []byte(agentContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry(tmpDir)
	_, err = registry.Discover()
	if err != nil {
		t.Fatal(err)
	}

	agent, exists := registry.Get("test-agent")
	if !exists {
		t.Fatal("test-agent should exist")
	}

	if agent.FilePath != agentFile {
		t.Errorf("Expected FilePath '%s', got '%s'", agentFile, agent.FilePath)
	}
}
