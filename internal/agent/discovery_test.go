package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestDiscoverWithNumberedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create numbered directories with agents
	numberedDirs := []string{
		"01-core",
		"02-language",
		"03-infrastructure",
		"10-research",
	}

	expectedAgents := []string{}

	for _, dir := range numberedDirs {
		dirPath := filepath.Join(tmpDir, dir)
		err := os.Mkdir(dirPath, 0755)
		if err != nil {
			t.Fatal(err)
		}

		// Create an agent in each numbered directory
		agentName := "agent-" + dir
		agentContent := fmt.Sprintf(`---
name: %s
description: Agent in %s
---
Agent content
`, agentName, dir)
		err = os.WriteFile(filepath.Join(dirPath, agentName+".md"), []byte(agentContent), 0644)
		if err != nil {
			t.Fatal(err)
		}
		expectedAgents = append(expectedAgents, agentName)
	}

	// Also create a root-level agent
	rootAgentContent := `---
name: root-agent
description: Root level agent
---
Root agent content
`
	err := os.WriteFile(filepath.Join(tmpDir, "root-agent.md"), []byte(rootAgentContent), 0644)
	if err != nil {
		t.Fatal(err)
	}
	expectedAgents = append(expectedAgents, "root-agent")

	registry := NewRegistry(tmpDir)
	agents, err := registry.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(agents) != len(expectedAgents) {
		t.Errorf("Expected %d agents, got %d", len(expectedAgents), len(agents))
	}

	for _, agentName := range expectedAgents {
		if !registry.Exists(agentName) {
			t.Errorf("Expected agent '%s' to exist", agentName)
		}
	}
}

func TestDiscoverSkipsSpecialDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create special directories that should be skipped
	specialDirs := []string{"examples", "transcripts", "logs"}

	for _, dir := range specialDirs {
		dirPath := filepath.Join(tmpDir, dir)
		err := os.Mkdir(dirPath, 0755)
		if err != nil {
			t.Fatal(err)
		}

		// Create non-agent .md files in special directories
		nonAgentContent := `# Documentation File

This is not an agent, just documentation.
No frontmatter here.
`
		err = os.WriteFile(filepath.Join(dirPath, "README.md"), []byte(nonAgentContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		// Also create a file that looks like it could be an agent
		fakeAgentContent := `---
name: fake-agent-in-` + dir + `
description: This should be skipped
---
This should not be discovered
`
		err = os.WriteFile(filepath.Join(dirPath, "fake-agent.md"), []byte(fakeAgentContent), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Create a real agent in root
	realAgentContent := `---
name: real-agent
description: Real agent
---
Real agent content
`
	err := os.WriteFile(filepath.Join(tmpDir, "real-agent.md"), []byte(realAgentContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry(tmpDir)
	agents, err := registry.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Should only discover the real agent, not any from special directories
	if len(agents) != 1 {
		t.Errorf("Expected 1 agent (special directories should be skipped), got %d", len(agents))
		for name := range agents {
			t.Logf("Found agent: %s", name)
		}
	}

	if !registry.Exists("real-agent") {
		t.Error("real-agent should exist")
	}

	// Verify fake agents were NOT discovered
	for _, dir := range specialDirs {
		fakeAgentName := "fake-agent-in-" + dir
		if registry.Exists(fakeAgentName) {
			t.Errorf("Agent '%s' should not exist (from skipped directory)", fakeAgentName)
		}
	}
}

func TestDiscoverSkipsNonNumberedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create non-numbered directories that should be skipped
	nonNumberedDirs := []string{"custom-dir", "agents-backup", "temp"}

	for _, dir := range nonNumberedDirs {
		dirPath := filepath.Join(tmpDir, dir)
		err := os.Mkdir(dirPath, 0755)
		if err != nil {
			t.Fatal(err)
		}

		// Create agent-like files in non-numbered directories
		agentContent := fmt.Sprintf(`---
name: agent-in-%s
description: Should be skipped
---
Content
`, dir)
		err = os.WriteFile(filepath.Join(dirPath, "agent.md"), []byte(agentContent), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Create a real agent in a numbered directory
	numberedDir := filepath.Join(tmpDir, "05-data")
	err := os.Mkdir(numberedDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	realAgentContent := `---
name: real-agent
description: Real agent in numbered directory
---
Real content
`
	err = os.WriteFile(filepath.Join(numberedDir, "real-agent.md"), []byte(realAgentContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry(tmpDir)
	agents, err := registry.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Should only discover the agent from numbered directory
	if len(agents) != 1 {
		t.Errorf("Expected 1 agent (non-numbered directories should be skipped), got %d", len(agents))
		for name := range agents {
			t.Logf("Found agent: %s", name)
		}
	}

	if !registry.Exists("real-agent") {
		t.Error("real-agent should exist")
	}

	// Verify agents from non-numbered directories were NOT discovered
	for _, dir := range nonNumberedDirs {
		fakeAgentName := "agent-in-" + dir
		if registry.Exists(fakeAgentName) {
			t.Errorf("Agent '%s' should not exist (from non-numbered directory)", fakeAgentName)
		}
	}
}

func TestDiscoverMixedStructure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a realistic mixed structure
	structure := map[string][]string{
		".":             {"agent-root.md"},
		"01-core":       {"core-agent1.md", "core-agent2.md"},
		"02-language":   {"lang-agent.md"},
		"examples":      {"example-usage.md", "README.md"},
		"transcripts":   {"session-2024.md"},
		"logs":          {"debug.md"},
		"custom-folder": {"custom-agent.md"},
		"10-research":   {"research-agent.md"},
	}

	expectedAgents := []string{
		"agent-root",
		"core-agent1",
		"core-agent2",
		"lang-agent",
		"research-agent",
	}

	for dir, files := range structure {
		var dirPath string
		if dir == "." {
			dirPath = tmpDir
		} else {
			dirPath = filepath.Join(tmpDir, dir)
			err := os.Mkdir(dirPath, 0755)
			if err != nil {
				t.Fatal(err)
			}
		}

		for _, file := range files {
			// Extract agent name from filename
			agentName := strings.TrimSuffix(file, ".md")
			agentContent := fmt.Sprintf(`---
name: %s
description: Agent %s
---
Content
`, agentName, agentName)
			err := os.WriteFile(filepath.Join(dirPath, file), []byte(agentContent), 0644)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	registry := NewRegistry(tmpDir)
	agents, err := registry.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(agents) != len(expectedAgents) {
		t.Errorf("Expected %d agents, got %d", len(expectedAgents), len(agents))
		t.Logf("Expected: %v", expectedAgents)
		discoveredNames := []string{}
		for name := range agents {
			discoveredNames = append(discoveredNames, name)
		}
		t.Logf("Discovered: %v", discoveredNames)
	}

	for _, agentName := range expectedAgents {
		if !registry.Exists(agentName) {
			t.Errorf("Expected agent '%s' to exist", agentName)
		}
	}

	// Verify agents from skipped directories were NOT discovered
	skippedAgents := []string{"example-usage", "session-2024", "debug", "custom-agent"}
	for _, agentName := range skippedAgents {
		if registry.Exists(agentName) {
			t.Errorf("Agent '%s' should not exist (from skipped directory)", agentName)
		}
	}
}

func TestDiscoverNestedNumberedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create numbered directory with nested subdirectories
	numberedDir := filepath.Join(tmpDir, "03-infra")
	nestedDir := filepath.Join(numberedDir, "cloud")

	err := os.MkdirAll(nestedDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create agent in top-level numbered directory
	topLevelAgent := `---
name: infra-agent
description: Infrastructure agent
---
Content
`
	err = os.WriteFile(filepath.Join(numberedDir, "infra-agent.md"), []byte(topLevelAgent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create agent in nested subdirectory (should also be discovered)
	nestedAgent := `---
name: cloud-agent
description: Cloud agent
---
Content
`
	err = os.WriteFile(filepath.Join(nestedDir, "cloud-agent.md"), []byte(nestedAgent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry(tmpDir)
	agents, err := registry.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Should discover both agents
	if len(agents) != 2 {
		t.Errorf("Expected 2 agents (including nested), got %d", len(agents))
	}

	if !registry.Exists("infra-agent") {
		t.Error("infra-agent should exist")
	}

	if !registry.Exists("cloud-agent") {
		t.Error("cloud-agent should exist (nested in numbered directory)")
	}
}

func TestDiscoverSkipsREADMEAndFrameworkFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create numbered directory
	numberedDir := filepath.Join(tmpDir, "01-core")
	err := os.Mkdir(numberedDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create README.md in root
	rootREADME := `# Agents Directory

This is the main agents directory.
`
	err = os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte(rootREADME), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create README.md in numbered directory
	dirREADME := `# Core Development Agents

These are core development agents.
`
	err = os.WriteFile(filepath.Join(numberedDir, "README.md"), []byte(dirREADME), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create framework file in root
	frameworkContent := `# Deliberation Framework

This is a framework document, not an agent.
`
	err = os.WriteFile(filepath.Join(tmpDir, "deliberation-framework.md"), []byte(frameworkContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create a real agent
	realAgentContent := `---
name: test-agent
description: Test agent
---
Real agent
`
	err = os.WriteFile(filepath.Join(numberedDir, "test-agent.md"), []byte(realAgentContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry(tmpDir)
	agents, err := registry.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Should only discover the real agent, not READMEs or framework files
	if len(agents) != 1 {
		t.Errorf("Expected 1 agent (READMEs and framework files should be skipped), got %d", len(agents))
		for name := range agents {
			t.Logf("Found agent: %s", name)
		}
	}

	if !registry.Exists("test-agent") {
		t.Error("test-agent should exist")
	}
}

func TestDiscoverHandlesDuplicateNames(t *testing.T) {
	tmpDir := t.TempDir()

	// Create numbered directory
	numberedDir := filepath.Join(tmpDir, "01-core")
	err := os.Mkdir(numberedDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create agent with same name in root
	rootAgentContent := `---
name: duplicate-agent
description: Root version
---
Root version content
`
	err = os.WriteFile(filepath.Join(tmpDir, "duplicate-agent.md"), []byte(rootAgentContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create agent with same name in numbered directory
	numberedAgentContent := `---
name: duplicate-agent
description: Numbered directory version
---
Numbered directory version content
`
	err = os.WriteFile(filepath.Join(numberedDir, "duplicate-agent.md"), []byte(numberedAgentContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry(tmpDir)
	agents, err := registry.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Should only have one entry (last one wins)
	if len(agents) != 1 {
		t.Errorf("Expected 1 agent (duplicates should be deduplicated), got %d", len(agents))
	}

	if !registry.Exists("duplicate-agent") {
		t.Error("duplicate-agent should exist")
	}

	// Verify one of the versions was stored
	agent, exists := registry.Get("duplicate-agent")
	if !exists {
		t.Fatal("duplicate-agent should exist")
	}

	// Should have one of the descriptions
	if agent.Description != "Root version" && agent.Description != "Numbered directory version" {
		t.Errorf("Unexpected description: %s", agent.Description)
	}
}
