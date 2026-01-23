package executor

import (
	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/claude"
)

// BaseSelector provides common functionality for intelligent agent selection.
// It embeds claude.Service for CLI invocation and manages registry access.
// Both IntelligentSelector (QC agent selection) and TaskAgentSelector (task execution)
// embed this type to share the common getAvailableAgents() logic.
type BaseSelector struct {
	claude.Service
	Registry *agent.Registry
}

// getAvailableAgents returns a list of agent names from the registry.
// This is the shared implementation used by both QC and task selectors.
func (bs *BaseSelector) getAvailableAgents() []string {
	if bs.Registry == nil {
		return []string{}
	}

	agents := bs.Registry.List()
	names := make([]string, 0, len(agents))
	for _, a := range agents {
		names = append(names, a.Name)
	}
	return names
}
