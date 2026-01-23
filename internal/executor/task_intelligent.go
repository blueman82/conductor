// Package executor provides task execution and orchestration.
package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/budget"
	"github.com/harrison/conductor/internal/claude"
	"github.com/harrison/conductor/internal/models"
)

// TaskAgentSelector uses Claude to intelligently select an agent for task execution.
// This is used when task.Agent is empty and QC mode is "intelligent".
// Embeds BaseSelector for shared registry access and Claude invocation.
type TaskAgentSelector struct {
	BaseSelector
}

// NewTaskAgentSelector creates a new task agent selector with the specified timeout.
// The timeout controls how long to wait for Claude's agent selection response.
func NewTaskAgentSelector(registry *agent.Registry, timeout time.Duration, logger budget.WaiterLogger) *TaskAgentSelector {
	return &TaskAgentSelector{
		BaseSelector: BaseSelector{
			Service:  *claude.NewService(timeout, logger),
			Registry: registry,
		},
	}
}

// NewTaskAgentSelectorWithInvoker creates a task agent selector using an external Invoker.
// This allows sharing a single Invoker across multiple components for consistent
// configuration and rate limit handling. The invoker should already have Timeout
// and Logger configured.
func NewTaskAgentSelectorWithInvoker(registry *agent.Registry, inv *claude.Invoker) *TaskAgentSelector {
	return &TaskAgentSelector{
		BaseSelector: BaseSelector{
			Service:  *claude.NewServiceWithInvoker(inv),
			Registry: registry,
		},
	}
}

// TaskAgentSelectionResult contains the result of intelligent agent selection.
type TaskAgentSelectionResult struct {
	Agent     string `json:"agent"`
	Rationale string `json:"rationale"`
}

// TaskAgentSelectionSchema returns the JSON schema for task agent selection response.
func TaskAgentSelectionSchema() string {
	return `{
  "type": "object",
  "properties": {
    "agent": {
      "type": "string",
      "description": "Name of the selected agent from the available list"
    },
    "rationale": {
      "type": "string",
      "description": "Brief explanation of why this agent was selected"
    }
  },
  "required": ["agent", "rationale"]
}`
}

// SelectAgent uses Claude to recommend an agent for task execution.
func (tas *TaskAgentSelector) SelectAgent(ctx context.Context, task models.Task) (*TaskAgentSelectionResult, error) {
	availableAgents := tas.getAvailableAgents()
	if len(availableAgents) == 0 {
		return nil, fmt.Errorf("no agents available in registry")
	}

	prompt := tas.buildSelectionPrompt(task, availableAgents)

	// Invoke Claude for recommendation with fallback JSON extraction
	var result TaskAgentSelectionResult
	if err := tas.InvokeAndParseWithFallback(ctx, prompt, TaskAgentSelectionSchema(), &result); err != nil {
		return nil, fmt.Errorf("intelligent task agent selection failed: %w", err)
	}

	// Validate agent exists, fallback to general-purpose if not
	if !tas.agentExists(result.Agent) {
		result.Agent = "general-purpose"
		result.Rationale = fmt.Sprintf("Fallback (agent not in registry): %s", result.Rationale)
	}

	return &result, nil
}

// agentExists checks if an agent exists in the registry.
func (tas *TaskAgentSelector) agentExists(name string) bool {
	if tas.Registry == nil {
		return false
	}
	_, exists := tas.Registry.Get(name)
	return exists
}

// buildSelectionPrompt creates the prompt for Claude to recommend an agent.
func (tas *TaskAgentSelector) buildSelectionPrompt(task models.Task, availableAgents []string) string {
	filesStr := strings.Join(task.Files, ", ")
	agentsStr := strings.Join(availableAgents, ", ")

	// Include success criteria if available
	criteriaStr := ""
	if len(task.SuccessCriteria) > 0 {
		criteriaStr = "\n\nSUCCESS CRITERIA:"
		for i, criterion := range task.SuccessCriteria {
			criteriaStr += fmt.Sprintf("\n%d. %s", i+1, criterion)
		}
	}

	// Add integration task context
	integrationStr := ""
	if task.Type == "integration" || task.IsIntegration() {
		integrationStr = "\n\nThis is an INTEGRATION task requiring cross-component implementation."
	}

	prompt := fmt.Sprintf(`You are selecting the best agent to EXECUTE a task (not review it).

TASK CONTEXT:
- Task Number: %s
- Task Name: %s
- Files to Modify: [%s]
- Task Description: %s%s%s

AVAILABLE AGENTS:
%s

INSTRUCTIONS:
Select the single best agent to execute this task. Consider:
1. Domain expertise needed (e.g., golang-pro for Go files, frontend-developer for React)
2. The specific files being modified and their extensions
3. The task description and what skills are required
4. The success criteria (what needs to be achieved)
5. Integration tasks need fullstack or architect expertise

IMPORTANT:
- Only select ONE agent from the AVAILABLE AGENTS list
- Match file extensions to agent expertise (.go → golang-pro, .ts/.tsx → frontend-developer, etc.)
- For integration tasks, prefer fullstack-developer or architect agents

Return JSON with "agent" (single agent name) and "rationale" (brief explanation).`,
		task.Number, task.Name, filesStr, task.Prompt, criteriaStr, integrationStr, agentsStr)

	return prompt
}
