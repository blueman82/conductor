package executor

import (
	"path/filepath"
	"strings"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/models"
)

// SelectQCAgents determines which QC agents to use based on configuration and task context
func SelectQCAgents(task models.Task, agentConfig models.QCAgentConfig, registry *agent.Registry) []string {
	var agents []string

	switch agentConfig.Mode {
	case "explicit":
		// Use only explicitly listed agents
		agents = agentConfig.ExplicitList
	case "mixed":
		// Start with auto-selected agents, then add additional
		agents = AutoSelectQCAgents(task, registry)
		agents = appendUniqueAgents(agents, agentConfig.AdditionalAgents...)
	case "auto":
		fallthrough
	default:
		// Auto-select based on task context
		agents = AutoSelectQCAgents(task, registry)
	}

	// Remove blocked agents
	agents = filterBlockedAgents(agents, agentConfig.BlockedAgents)

	// Ensure at least one agent is selected
	if len(agents) == 0 {
		// Check if fallback is also blocked
		fallback := "quality-control"
		isBlocked := false
		for _, b := range agentConfig.BlockedAgents {
			if b == fallback {
				isBlocked = true
				break
			}
		}

		if !isBlocked {
			agents = []string{fallback}
		}
		// If blocked, return empty list - caller must handle error
	}

	return agents
}

// AutoSelectQCAgents automatically selects QC agents based on task file types
func AutoSelectQCAgents(task models.Task, registry *agent.Registry) []string {
	agents := []string{"quality-control"} // Always include baseline QC agent

	// Track which language-specific agents we've added
	addedLangAgents := make(map[string]bool)

	for _, file := range task.Files {
		ext := strings.ToLower(filepath.Ext(file))

		// Map file extensions to specialized agents
		var langAgent string
		switch ext {
		case ".go":
			langAgent = "golang-pro"
		case ".py":
			langAgent = "python-pro"
		case ".ts", ".tsx":
			langAgent = "typescript-pro"
		case ".js", ".jsx":
			langAgent = "javascript-pro"
		case ".rs":
			langAgent = "rust-pro"
		case ".java":
			langAgent = "java-pro"
		case ".rb":
			langAgent = "ruby-pro"
		case ".cs":
			langAgent = "csharp-pro"
		case ".cpp", ".cc", ".cxx", ".c", ".h", ".hpp":
			langAgent = "cpp-pro"
		case ".swift":
			langAgent = "swift-expert"
		case ".kt", ".kts":
			langAgent = "kotlin-specialist"
		case ".php":
			langAgent = "php-pro"
		case ".ex", ".exs":
			langAgent = "elixir-pro"
		case ".scala":
			langAgent = "scala-pro"
		case ".sql":
			langAgent = "sql-pro"
		}

		// Add language agent if available and not already added
		if langAgent != "" && !addedLangAgents[langAgent] {
			if registry != nil && registry.Exists(langAgent) {
				agents = appendUniqueAgents(agents, langAgent)
				addedLangAgents[langAgent] = true
			}
		}
	}

	return agents
}

// appendUniqueAgents appends agents to the list, avoiding duplicates
func appendUniqueAgents(agents []string, newAgents ...string) []string {
	existing := make(map[string]bool)
	for _, a := range agents {
		existing[a] = true
	}

	for _, agent := range newAgents {
		if !existing[agent] {
			agents = append(agents, agent)
			existing[agent] = true
		}
	}

	return agents
}

// filterBlockedAgents removes blocked agents from the list
func filterBlockedAgents(agents []string, blocked []string) []string {
	if len(blocked) == 0 {
		return agents
	}

	blockedSet := make(map[string]bool)
	for _, b := range blocked {
		blockedSet[b] = true
	}

	result := make([]string, 0, len(agents))
	for _, agent := range agents {
		if !blockedSet[agent] {
			result = append(result, agent)
		}
	}

	return result
}
