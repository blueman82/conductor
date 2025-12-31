// Package executor provides task execution and orchestration.
package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/claude"
	"github.com/harrison/conductor/internal/models"
)

// TaskAgentSelector uses Claude to intelligently select an agent for task execution.
// This is used when task.Agent is empty and QC mode is "intelligent".
type TaskAgentSelector struct {
	Registry   *agent.Registry
	ClaudePath string
	Timeout    time.Duration
}

// NewTaskAgentSelector creates a new task agent selector with the specified timeout.
// The timeout controls how long to wait for Claude's agent selection response.
func NewTaskAgentSelector(registry *agent.Registry, timeout time.Duration) *TaskAgentSelector {
	return &TaskAgentSelector{
		Registry:   registry,
		ClaudePath: "claude",
		Timeout:    timeout,
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
	result, err := tas.invokeClaudeForSelection(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("intelligent task agent selection failed: %w", err)
	}

	if !tas.agentExists(result.Agent) {
		result.Agent = "general-purpose"
		result.Rationale = fmt.Sprintf("Fallback (agent not in registry): %s", result.Rationale)
	}

	return result, nil
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

func (tas *TaskAgentSelector) invokeClaudeForSelection(ctx context.Context, prompt string) (*TaskAgentSelectionResult, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, tas.Timeout)
	defer cancel()

	args := []string{
		"-p", prompt,
		"--json-schema", TaskAgentSelectionSchema(),
		"--output-format", "json",
		"--settings", `{"disableAllHooks": true}`,
	}

	cmd := exec.CommandContext(ctxWithTimeout, tas.ClaudePath, args...)
	claude.SetCleanEnv(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("claude invocation failed: %w (output: %s)", err, string(output))
	}

	parsed, err := agent.ParseClaudeOutput(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse claude output: %w", err)
	}

	if parsed.Content == "" {
		return nil, fmt.Errorf("empty content in claude response")
	}

	var result TaskAgentSelectionResult
	if err := json.Unmarshal([]byte(parsed.Content), &result); err != nil {
		start := strings.Index(parsed.Content, "{")
		end := strings.LastIndex(parsed.Content, "}")
		if start >= 0 && end > start {
			if err := json.Unmarshal([]byte(parsed.Content[start:end+1]), &result); err != nil {
				return nil, fmt.Errorf("failed to extract JSON: %w", err)
			}
			return &result, nil
		}
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &result, nil
}
