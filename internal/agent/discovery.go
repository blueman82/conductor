package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Agent represents a Claude Code agent with metadata
type Agent struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	Tools       ToolList `yaml:"tools" json:"tools,omitempty"` // Omit if empty = all tools available
	FilePath    string   `yaml:"-" json:"-"`                   // Not parsed from YAML, not included in JSON
}

// ToolList is a custom type that handles both comma-separated strings
// and YAML arrays for the tools field in agent frontmatter
type ToolList []string

// UnmarshalYAML implements custom unmarshaling for ToolList
// Accepts both formats:
// - Comma-separated string: "Read, Write, Edit"
// - YAML array: [Read, Write, Edit]
func (t *ToolList) UnmarshalYAML(value *yaml.Node) error {
	// Try to unmarshal as string first (Claude Code format)
	var str string
	if err := value.Decode(&str); err == nil {
		// Split by comma and trim whitespace
		parts := strings.Split(str, ",")
		*t = make(ToolList, 0, len(parts))
		for _, part := range parts {
			tool := strings.TrimSpace(part)
			if tool != "" {
				*t = append(*t, tool)
			}
		}
		return nil
	}

	// Try to unmarshal as array (YAML array format)
	var arr []string
	if err := value.Decode(&arr); err == nil {
		*t = ToolList(arr)
		return nil
	}

	return fmt.Errorf("tools must be either a comma-separated string or an array")
}

// MarshalJSON implements custom marshaling for ToolList
// Always serializes as a JSON array for consistency with claude CLI --agents flag
// Example: ["Read", "Write", "Edit"]
func (t ToolList) MarshalJSON() ([]byte, error) {
	// Convert ToolList to []string and marshal as array
	return json.Marshal([]string(t))
}

// Registry manages discovered agents
type Registry struct {
	AgentsDir string
	agents    map[string]*Agent
}

// NewRegistry creates a new agent registry
// If agentsDir is empty, uses ~/.claude/agents as default
func NewRegistry(agentsDir string) *Registry {
	if agentsDir == "" {
		// Default to ~/.claude/agents
		home, _ := os.UserHomeDir()
		agentsDir = filepath.Join(home, ".claude", "agents")
	}

	return &Registry{
		AgentsDir: agentsDir,
		agents:    make(map[string]*Agent),
	}
}

// Discover scans the agents directory and parses agent files
// Returns a map of agent names to Agent structs
// Returns an empty map (not an error) if the directory doesn't exist
//
// Strategy: Directory whitelisting + file filtering to reduce false warnings
// - Scans root level .md files (agent definitions)
// - Scans numbered subdirectories: 01-*, 02-*, ..., 10-* (categorized agents)
// - Skips special directories: examples/, transcripts/, logs/ (documentation/metadata)
// - Skips README.md files (category documentation, not agent definitions)
func (r *Registry) Discover() (map[string]*Agent, error) {
	// Check if directory exists
	if _, err := os.Stat(r.AgentsDir); os.IsNotExist(err) {
		// No agents directory - return empty map, not an error
		return r.agents, nil
	}

	// Walk directory
	err := filepath.Walk(r.AgentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Handle directories with whitelist filtering
		if info.IsDir() {
			// Always allow the root agents directory itself
			if path == r.AgentsDir {
				return nil
			}

			// Get directory name relative to agents directory
			relPath, err := filepath.Rel(r.AgentsDir, path)
			if err != nil {
				return err
			}

			// Extract the directory name (first component of relative path)
			dirName := strings.Split(relPath, string(filepath.Separator))[0]

			// Skip special documentation/metadata directories
			// These contain examples, transcripts, logs - not agent definitions
			if dirName == "examples" || dirName == "transcripts" || dirName == "logs" {
				return filepath.SkipDir
			}

			// Allow numbered directories (01-*, 02-*, ..., 10-*)
			// These contain categorized agent definitions
			if len(dirName) >= 3 && dirName[0] >= '0' && dirName[0] <= '9' && dirName[1] >= '0' && dirName[1] <= '9' && dirName[2] == '-' {
				return nil
			}

			// Skip any other subdirectories not matching our whitelist
			return filepath.SkipDir
		}

		// Only process .md files
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		// Skip non-agent documentation files
		// - README.md: Category documentation
		// - *-framework.md: Framework/methodology documentation
		basename := filepath.Base(path)
		if basename == "README.md" || strings.HasSuffix(basename, "-framework.md") {
			return nil
		}

		agent, err := parseAgentFile(path)
		if err != nil {
			// Log warning but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, err)
			return nil
		}

		r.agents[agent.Name] = agent
		return nil
	})

	return r.agents, err
}

// Exists checks if an agent with the given name exists in the registry
func (r *Registry) Exists(agentName string) bool {
	_, exists := r.agents[agentName]
	return exists
}

// Get retrieves an agent by name
// Returns the agent and true if found, nil and false otherwise
func (r *Registry) Get(agentName string) (*Agent, bool) {
	agent, exists := r.agents[agentName]
	return agent, exists
}

// List returns all agents in the registry
func (r *Registry) List() []*Agent {
	agents := make([]*Agent, 0, len(r.agents))
	for _, agent := range r.agents {
		agents = append(agents, agent)
	}
	return agents
}

// parseAgentFile parses a single agent file
func parseAgentFile(path string) (*Agent, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Extract YAML frontmatter between --- markers
	frontmatter, _ := extractFrontmatter(content)
	if frontmatter == nil {
		return nil, fmt.Errorf("no frontmatter found in %s", path)
	}

	var agent Agent
	if err := yaml.Unmarshal(frontmatter, &agent); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	agent.FilePath = path

	if agent.Name == "" {
		return nil, fmt.Errorf("agent name is required")
	}

	return &agent, nil
}

// extractFrontmatter extracts YAML frontmatter from markdown content
// Returns the frontmatter and the remaining body
func extractFrontmatter(content []byte) ([]byte, []byte) {
	lines := strings.Split(string(content), "\n")
	if len(lines) < 3 || lines[0] != "---" {
		return nil, content
	}

	// Find closing ---
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			frontmatter := []byte(strings.Join(lines[1:i], "\n"))
			body := []byte(strings.Join(lines[i+1:], "\n"))
			return frontmatter, body
		}
	}

	return nil, content
}
