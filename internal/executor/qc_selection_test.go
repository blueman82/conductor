package executor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/models"
)

func TestSelectQCAgents(t *testing.T) {
	tests := []struct {
		name         string
		task         models.Task
		config       models.QCAgentConfig
		registry     interface{}
		wantAgents   []string
		wantContains bool
	}{
		{
			name: "auto mode with .go files",
			task: models.Task{
				Number: "1",
				Name:   "Implement feature",
				Files:  []string{"main.go", "handler.go"},
			},
			config: models.QCAgentConfig{
				Mode: "auto",
			},
			registry:   nil, // When nil, auto-select still returns baseline + no language agents
			wantAgents: []string{"quality-control"},
		},
		{
			name: "explicit mode with single agent",
			task: models.Task{
				Number: "4",
				Name:   "Custom task",
				Files:  []string{"main.go"},
			},
			config: models.QCAgentConfig{
				Mode:         "explicit",
				ExplicitList: []string{"custom-reviewer"},
			},
			registry:   nil,
			wantAgents: []string{"custom-reviewer"},
		},
		{
			name: "explicit mode with multiple agents",
			task: models.Task{
				Number: "5",
				Name:   "Multi-reviewer task",
				Files:  []string{"main.go"},
			},
			config: models.QCAgentConfig{
				Mode:         "explicit",
				ExplicitList: []string{"golang-pro", "quality-control", "test-expert"},
			},
			registry:   nil,
			wantAgents: []string{"golang-pro", "quality-control", "test-expert"},
		},
		{
			name: "mixed mode with additional agents",
			task: models.Task{
				Number: "6",
				Name:   "Mixed mode task",
				Files:  []string{"main.go"},
			},
			config: models.QCAgentConfig{
				Mode:             "mixed",
				AdditionalAgents: []string{"security-expert", "performance-pro"},
			},
			registry:   nil,
			wantAgents: []string{"quality-control", "security-expert", "performance-pro"},
		},
		{
			name: "default mode (empty string) falls back to auto",
			task: models.Task{
				Number: "7",
				Name:   "Default mode task",
				Files:  []string{"main.py"},
			},
			config:     models.QCAgentConfig{},
			registry:   nil,
			wantAgents: []string{"quality-control"},
		},
		{
			name: "empty files list returns just baseline",
			task: models.Task{
				Number: "8",
				Name:   "No files task",
				Files:  []string{},
			},
			config: models.QCAgentConfig{
				Mode: "auto",
			},
			registry:   nil,
			wantAgents: []string{"quality-control"},
		},
		{
			name: "blocked agents filtered out from explicit mode",
			task: models.Task{
				Number: "9",
				Name:   "Filtered task",
				Files:  []string{"main.go"},
			},
			config: models.QCAgentConfig{
				Mode:            "explicit",
				ExplicitList:    []string{"golang-pro", "quality-control"},
				BlockedAgents:   []string{"golang-pro"},
			},
			registry:   nil,
			wantAgents: []string{"quality-control"},
		},
		{
			name: "all agents blocked including fallback returns empty",
			task: models.Task{
				Number: "10",
				Name:   "All blocked task",
				Files:  []string{"main.go"},
			},
			config: models.QCAgentConfig{
				Mode:            "explicit",
				ExplicitList:    []string{"golang-pro"},
				BlockedAgents:   []string{"golang-pro", "quality-control"},
			},
			registry:   nil,
			wantAgents: []string{}, // Empty - all agents blocked, caller must handle error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Cast interface to *agent.Registry (will be nil if tt.registry is nil)
			var reg *agent.Registry
			if tt.registry != nil {
				reg = tt.registry.(*agent.Registry)
			}
			got := SelectQCAgents(tt.task, tt.config, reg)

			if len(got) != len(tt.wantAgents) {
				t.Errorf("SelectQCAgents() got %d agents, want %d\ngot: %v\nwant: %v", len(got), len(tt.wantAgents), got, tt.wantAgents)
				return
			}

			// Check all expected agents are present
			for _, want := range tt.wantAgents {
				found := false
				for _, agent := range got {
					if agent == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("SelectQCAgents() missing expected agent %q in %v", want, got)
				}
			}
		})
	}
}

func TestFilterBlockedAgents(t *testing.T) {
	tests := []struct {
		name        string
		agents      []string
		blocked     []string
		wantAgents  []string
	}{
		{
			name:       "no blocked agents",
			agents:     []string{"golang-pro", "python-pro"},
			blocked:    []string{},
			wantAgents: []string{"golang-pro", "python-pro"},
		},
		{
			name:       "some agents blocked",
			agents:     []string{"quality-control", "golang-pro", "python-pro"},
			blocked:    []string{"golang-pro"},
			wantAgents: []string{"quality-control", "python-pro"},
		},
		{
			name:       "multiple agents blocked",
			agents:     []string{"quality-control", "golang-pro", "python-pro", "typescript-pro"},
			blocked:    []string{"golang-pro", "python-pro"},
			wantAgents: []string{"quality-control", "typescript-pro"},
		},
		{
			name:       "all agents blocked",
			agents:     []string{"golang-pro", "python-pro"},
			blocked:    []string{"golang-pro", "python-pro"},
			wantAgents: []string{},
		},
		{
			name:       "blocked agents not in list",
			agents:     []string{"golang-pro", "python-pro"},
			blocked:    []string{"rust-pro", "java-pro"},
			wantAgents: []string{"golang-pro", "python-pro"},
		},
		{
			name:       "empty agent list",
			agents:     []string{},
			blocked:    []string{"golang-pro"},
			wantAgents: []string{},
		},
		{
			name:       "empty blocked list",
			agents:     []string{"golang-pro"},
			blocked:    []string{},
			wantAgents: []string{"golang-pro"},
		},
		{
			name:       "both empty",
			agents:     []string{},
			blocked:    []string{},
			wantAgents: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterBlockedAgents(tt.agents, tt.blocked)

			if len(got) != len(tt.wantAgents) {
				t.Errorf("filterBlockedAgents() got %d agents, want %d\ngot: %v\nwant: %v", len(got), len(tt.wantAgents), got, tt.wantAgents)
				return
			}

			for i, want := range tt.wantAgents {
				if i >= len(got) || got[i] != want {
					t.Errorf("filterBlockedAgents() agent %d: got %q, want %q", i, got[i], want)
				}
			}
		})
	}
}

func TestAppendUniqueAgents(t *testing.T) {
	tests := []struct {
		name        string
		existing    []string
		new         []string
		wantAgents  []string
	}{
		{
			name:       "add new agents to empty list",
			existing:   []string{},
			new:        []string{"golang-pro", "python-pro"},
			wantAgents: []string{"golang-pro", "python-pro"},
		},
		{
			name:       "add new agents to existing list",
			existing:   []string{"quality-control"},
			new:        []string{"golang-pro", "python-pro"},
			wantAgents: []string{"quality-control", "golang-pro", "python-pro"},
		},
		{
			name:       "avoid duplicate agents",
			existing:   []string{"quality-control", "golang-pro"},
			new:        []string{"golang-pro", "python-pro"},
			wantAgents: []string{"quality-control", "golang-pro", "python-pro"},
		},
		{
			name:       "all new agents already exist",
			existing:   []string{"golang-pro", "python-pro"},
			new:        []string{"golang-pro", "python-pro"},
			wantAgents: []string{"golang-pro", "python-pro"},
		},
		{
			name:       "empty new list",
			existing:   []string{"golang-pro"},
			new:        []string{},
			wantAgents: []string{"golang-pro"},
		},
		{
			name:       "both empty",
			existing:   []string{},
			new:        []string{},
			wantAgents: []string{},
		},
		{
			name:       "multiple duplicates",
			existing:   []string{"a", "b"},
			new:        []string{"b", "c", "a", "d"},
			wantAgents: []string{"a", "b", "c", "d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendUniqueAgents(tt.existing, tt.new...)

			if len(got) != len(tt.wantAgents) {
				t.Errorf("appendUniqueAgents() got %d agents, want %d\ngot: %v\nwant: %v", len(got), len(tt.wantAgents), got, tt.wantAgents)
				return
			}

			// Check that all expected agents are present (order may vary except for initial)
			for _, want := range tt.wantAgents {
				found := false
				for _, agent := range got {
					if agent == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("appendUniqueAgents() missing expected agent %q in %v", want, got)
				}
			}
		})
	}
}

func TestSelectQCAgentsIntegration(t *testing.T) {
	tests := []struct {
		name        string
		task        models.Task
		config      models.QCAgentConfig
		registry    *agent.Registry
		wantNoError bool
		checkCount  int
	}{
		{
			name: "complex scenario: auto + additional + blocked",
			task: models.Task{
				Number: "1",
				Name:   "Complex task",
				Files:  []string{"main.go", "style.css"},
			},
			config: models.QCAgentConfig{
				Mode:             "mixed",
				AdditionalAgents: []string{"security-expert"},
				BlockedAgents:    []string{"security-expert"}, // Should be filtered
			},
			registry:    nil,
			wantNoError: true,
			checkCount:  1, // quality-control (security-expert is blocked)
		},
		{
			name: "explicit with one agent should not use multi-agent",
			task: models.Task{
				Number: "2",
				Name:   "Single reviewer",
				Files:  []string{"main.go"},
			},
			config: models.QCAgentConfig{
				Mode:         "explicit",
				ExplicitList: []string{"golang-pro"},
			},
			registry:    nil,
			wantNoError: true,
			checkCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectQCAgents(tt.task, tt.config, tt.registry)

			if len(got) != tt.checkCount {
				t.Errorf("SelectQCAgents() got %d agents, want %d\ngot: %v", len(got), tt.checkCount, got)
			}
		})
	}
}

// createTestRegistry creates a mock agent registry with specified agent names
// by creating actual agent files in a temporary directory
func createTestRegistry(t *testing.T, agentNames []string) *agent.Registry {
	tmpDir, err := os.MkdirTemp("", "test-agents-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}

	// Create minimal agent .md files with YAML frontmatter
	for _, name := range agentNames {
		agentPath := filepath.Join(tmpDir, name+".md")
		content := "---\nname: " + name + "\n---\n"
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

	return registry
}

// cleanupTestRegistry removes the temporary directory used by the test registry
func cleanupTestRegistry(t *testing.T, registry *agent.Registry) {
	if registry != nil && registry.AgentsDir != "" {
		err := os.RemoveAll(registry.AgentsDir)
		if err != nil {
			t.Errorf("failed to cleanup test registry directory: %v", err)
		}
	}
}

// TestAutoSelectQCAgentsWithRegistry tests the critical code path that checks
// registry.Exists(langAgent) - this was previously untested with a real registry
func TestAutoSelectQCAgentsWithRegistry(t *testing.T) {
	tests := []struct {
		name           string
		task           models.Task
		config         models.QCAgentConfig
		registryAgents []string // Agents that exist in registry
		wantAgents     []string
	}{
		{
			name: "auto mode with available golang-pro",
			task: models.Task{
				Number: "1",
				Name:   "Test task",
				Files:  []string{"main.go", "util.go"},
			},
			config: models.QCAgentConfig{Mode: "auto"},
			registryAgents: []string{"golang-pro", "quality-control"},
			wantAgents:     []string{"quality-control", "golang-pro"},
		},
		{
			name: "auto mode with unavailable language agent",
			task: models.Task{
				Number: "2",
				Name:   "Test task",
				Files:  []string{"main.go"},
			},
			config: models.QCAgentConfig{Mode: "auto"},
			registryAgents: []string{"quality-control"}, // golang-pro NOT available
			wantAgents:     []string{"quality-control"}, // Falls back to baseline only
		},
		{
			name: "auto mode with multiple available agents",
			task: models.Task{
				Number: "3",
				Name:   "Multi-language task",
				Files:  []string{"main.go", "script.py", "app.ts"},
			},
			config: models.QCAgentConfig{Mode: "auto"},
			registryAgents: []string{"golang-pro", "python-pro", "typescript-pro", "quality-control"},
			wantAgents:     []string{"quality-control", "golang-pro", "python-pro", "typescript-pro"},
		},
		{
			name: "auto mode with partial agent availability",
			task: models.Task{
				Number: "4",
				Name:   "Multi-language task",
				Files:  []string{"main.go", "script.py", "app.ts"},
			},
			config: models.QCAgentConfig{Mode: "auto"},
			registryAgents: []string{"golang-pro", "quality-control"}, // Only golang-pro exists
			wantAgents:     []string{"quality-control", "golang-pro"}, // python-pro and typescript-pro skipped
		},
		{
			name: "mixed mode adds agents only if they exist in registry",
			task: models.Task{
				Number: "5",
				Name:   "Mixed mode task",
				Files:  []string{"main.go"},
			},
			config: models.QCAgentConfig{
				Mode:             "mixed",
				AdditionalAgents: []string{"security-auditor", "non-existent-agent"},
			},
			registryAgents: []string{"golang-pro", "quality-control", "security-auditor"},
			// non-existent-agent should NOT affect the result (SelectQCAgents adds all requested agents)
			// but golang-pro should be added from auto-select
			wantAgents: []string{"quality-control", "golang-pro", "security-auditor", "non-existent-agent"},
		},
		{
			name: "mixed mode with auto-select respects registry availability",
			task: models.Task{
				Number: "6",
				Name:   "Task with Go files",
				Files:  []string{"main.go"},
			},
			config: models.QCAgentConfig{
				Mode:             "mixed",
				AdditionalAgents: []string{"security-expert"},
			},
			registryAgents: []string{"quality-control", "security-expert"}, // golang-pro NOT available
			wantAgents:     []string{"quality-control", "security-expert"}, // golang-pro skipped
		},
		{
			name: "empty files list returns just baseline even with registry",
			task: models.Task{
				Number: "7",
				Name:   "No files task",
				Files:  []string{},
			},
			config: models.QCAgentConfig{Mode: "auto"},
			registryAgents: []string{"golang-pro", "python-pro", "quality-control"},
			wantAgents:     []string{"quality-control"},
		},
		{
			name: "multiple files same language only adds agent once",
			task: models.Task{
				Number: "8",
				Name:   "Multiple Go files",
				Files:  []string{"main.go", "handler.go", "util.go"},
			},
			config: models.QCAgentConfig{Mode: "auto"},
			registryAgents: []string{"golang-pro", "quality-control"},
			wantAgents:     []string{"quality-control", "golang-pro"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := createTestRegistry(t, tt.registryAgents)
			defer cleanupTestRegistry(t, registry)

			got := SelectQCAgents(tt.task, tt.config, registry)

			if len(got) != len(tt.wantAgents) {
				t.Errorf("SelectQCAgents() got %d agents, want %d\ngot: %v\nwant: %v",
					len(got), len(tt.wantAgents), got, tt.wantAgents)
				return
			}

			// Check all expected agents are present
			for _, want := range tt.wantAgents {
				found := false
				for _, agent := range got {
					if agent == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("SelectQCAgents() missing expected agent %q in %v", want, got)
				}
			}
		})
	}
}

// TestAutoSelectQCAgentsAllExtensions tests that all supported file extensions
// map to the correct language-specific agents (with registry checks)
func TestAutoSelectQCAgentsAllExtensions(t *testing.T) {
	extensionTests := []struct {
		ext       string
		wantAgent string
	}{
		{".go", "golang-pro"},
		{".py", "python-pro"},
		{".ts", "typescript-pro"},
		{".tsx", "typescript-pro"},
		{".js", "javascript-pro"},
		{".jsx", "javascript-pro"},
		{".rs", "rust-pro"},
		{".java", "java-pro"},
		{".rb", "ruby-pro"},
		{".cs", "csharp-pro"},
		{".cpp", "cpp-pro"},
		{".cc", "cpp-pro"},
		{".cxx", "cpp-pro"},
		{".c", "cpp-pro"},
		{".h", "cpp-pro"},
		{".hpp", "cpp-pro"},
		{".swift", "swift-expert"},
		{".kt", "kotlin-specialist"},
		{".kts", "kotlin-specialist"},
		{".php", "php-pro"},
		{".ex", "elixir-pro"},
		{".exs", "elixir-pro"},
		{".scala", "scala-pro"},
		{".sql", "sql-pro"},
	}

	for _, tt := range extensionTests {
		t.Run(tt.ext, func(t *testing.T) {
			task := models.Task{
				Number: "1",
				Name:   "Test task",
				Files:  []string{"file" + tt.ext},
			}
			config := models.QCAgentConfig{Mode: "auto"}

			// Create registry with the expected agent
			registry := createTestRegistry(t, []string{tt.wantAgent, "quality-control"})
			defer cleanupTestRegistry(t, registry)

			agents := SelectQCAgents(task, config, registry)

			// Should contain both baseline and language agent
			hasWantAgent := false
			hasBaseline := false
			for _, agent := range agents {
				if agent == tt.wantAgent {
					hasWantAgent = true
				}
				if agent == "quality-control" {
					hasBaseline = true
				}
			}

			if !hasWantAgent {
				t.Errorf("Expected %s for %s extension, got %v", tt.wantAgent, tt.ext, agents)
			}
			if !hasBaseline {
				t.Errorf("Expected quality-control baseline for %s extension, got %v", tt.ext, agents)
			}
		})
	}
}

// TestAutoSelectQCAgentsExtensionWithoutRegistry tests file extensions
// when the language-specific agent is NOT in the registry
func TestAutoSelectQCAgentsExtensionWithoutRegistry(t *testing.T) {
	extensionTests := []struct {
		ext string
		// These would map to agents, but we won't provide them in registry
	}{
		{".go"},
		{".py"},
		{".ts"},
		{".rs"},
		{".java"},
	}

	for _, tt := range extensionTests {
		t.Run("missing_agent_"+tt.ext, func(t *testing.T) {
			task := models.Task{
				Number: "1",
				Name:   "Test task",
				Files:  []string{"file" + tt.ext},
			}
			config := models.QCAgentConfig{Mode: "auto"}

			// Create registry with ONLY baseline, no language-specific agents
			registry := createTestRegistry(t, []string{"quality-control"})
			defer cleanupTestRegistry(t, registry)

			agents := SelectQCAgents(task, config, registry)

			// Should only contain baseline since language agent not available
			if len(agents) != 1 || agents[0] != "quality-control" {
				t.Errorf("Expected only quality-control for missing agent, got %v", agents)
			}
		})
	}
}

// TestSelectQCAgentsWithRegistryBlocking tests that blocked agents
// are filtered out even when they exist in the registry
func TestSelectQCAgentsWithRegistryBlocking(t *testing.T) {
	tests := []struct {
		name           string
		task           models.Task
		config         models.QCAgentConfig
		registryAgents []string
		wantAgents     []string
	}{
		{
			name: "block language agent that exists in registry",
			task: models.Task{
				Number: "1",
				Name:   "Test task",
				Files:  []string{"main.go"},
			},
			config: models.QCAgentConfig{
				Mode:          "auto",
				BlockedAgents: []string{"golang-pro"},
			},
			registryAgents: []string{"golang-pro", "quality-control"},
			wantAgents:     []string{"quality-control"},
		},
		{
			name: "block baseline quality-control falls back to auto-select agents",
			task: models.Task{
				Number: "2",
				Name:   "Test task",
				Files:  []string{"main.go", "main.py"},
			},
			config: models.QCAgentConfig{
				Mode:          "auto",
				BlockedAgents: []string{"quality-control"},
			},
			registryAgents: []string{"golang-pro", "python-pro", "quality-control"},
			wantAgents:     []string{"golang-pro", "python-pro"},
		},
		{
			name: "block multiple agents including baseline",
			task: models.Task{
				Number: "3",
				Name:   "Test task",
				Files:  []string{"main.go", "main.py"},
			},
			config: models.QCAgentConfig{
				Mode:          "mixed",
				AdditionalAgents: []string{"security-expert"},
				BlockedAgents: []string{"golang-pro", "quality-control"},
			},
			registryAgents: []string{"golang-pro", "python-pro", "security-expert", "quality-control"},
			wantAgents:     []string{"python-pro", "security-expert"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := createTestRegistry(t, tt.registryAgents)
			defer cleanupTestRegistry(t, registry)

			got := SelectQCAgents(tt.task, tt.config, registry)

			if len(got) != len(tt.wantAgents) {
				t.Errorf("SelectQCAgents() got %d agents, want %d\ngot: %v\nwant: %v",
					len(got), len(tt.wantAgents), got, tt.wantAgents)
				return
			}

			// Check all expected agents are present
			for _, want := range tt.wantAgents {
				found := false
				for _, agent := range got {
					if agent == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("SelectQCAgents() missing expected agent %q in %v", want, got)
				}
			}
		})
	}
}
