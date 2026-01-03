// Package executor provides task execution and orchestration.
package executor

import (
	"context"
	"encoding/json"
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
// Uses claude.Invoker for CLI invocation with rate limit handling.
type TaskAgentSelector struct {
	Registry *agent.Registry
	inv      *claude.Invoker     // Invoker handles CLI invocation and rate limit retry
	Logger   budget.WaiterLogger // For TTS + visual during rate limit wait (passed to Invoker)
}

// NewTaskAgentSelector creates a new task agent selector with the specified timeout.
// The timeout controls how long to wait for Claude's agent selection response.
func NewTaskAgentSelector(registry *agent.Registry, timeout time.Duration, logger budget.WaiterLogger) *TaskAgentSelector {
	inv := claude.NewInvoker()
	inv.Timeout = timeout
	inv.Logger = logger
	return &TaskAgentSelector{
		Registry: registry,
		inv:      inv,
		Logger:   logger,
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

	// Invoke Claude for recommendation via Invoker (rate limit handling is automatic)
	req := claude.Request{
		Prompt: prompt,
		Schema: TaskAgentSelectionSchema(),
	}

	resp, err := tas.inv.Invoke(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("intelligent task agent selection failed: %w", err)
	}

	// Parse the response
	content, _, err := claude.ParseResponse(resp.RawOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to parse claude output: %w", err)
	}

	if content == "" {
		return nil, fmt.Errorf("empty content in claude response")
	}

	// Parse as selection result - schema guarantees valid JSON
	var result TaskAgentSelectionResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// Fallback: try to extract JSON from content (handles edge cases)
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start >= 0 && end > start {
			if err := json.Unmarshal([]byte(content[start:end+1]), &result); err != nil {
				return nil, fmt.Errorf("schema enforcement failed: %w (content: %s)", err, content)
			}
		} else {
			return nil, fmt.Errorf("schema enforcement failed: %w (content: %s)", err, content)
		}
	}

	if !tas.agentExists(result.Agent) {
		result.Agent = "general-purpose"
		result.Rationale = fmt.Sprintf("Fallback (agent not in registry): %s", result.Rationale)
	}

	return &result, nil
}

// getAvailableAgents returns a list of agent names from the registry.
func (tas *TaskAgentSelector) getAvailableAgents() []string {
	if tas.Registry == nil {
		return []string{}
	}

	agents := tas.Registry.List()
	names := make([]string, 0, len(agents))
	for _, a := range agents {
		names = append(names, a.Name)
	}
	return names
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

