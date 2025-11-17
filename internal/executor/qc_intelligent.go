package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/models"
)

// IntelligentAgentRecommendation represents Claude's response for agent selection
type IntelligentAgentRecommendation struct {
	Agents    []string `json:"agents"`
	Rationale string   `json:"rationale"`
}

// IntelligentSelector manages intelligent QC agent selection using Claude
type IntelligentSelector struct {
	Registry   *agent.Registry
	Cache      *QCSelectionCache
	ClaudePath string
	MaxAgents  int
	Timeout    time.Duration
}

// NewIntelligentSelector creates a new intelligent selector with defaults
func NewIntelligentSelector(registry *agent.Registry, cacheTTLSeconds int) *IntelligentSelector {
	return &IntelligentSelector{
		Registry:   registry,
		Cache:      NewQCSelectionCache(cacheTTLSeconds),
		ClaudePath: "claude",
		MaxAgents:  4,
		Timeout:    30 * time.Second,
	}
}

// SelectAgents uses Claude to recommend QC agents based on task context and executing agent
func (is *IntelligentSelector) SelectAgents(
	ctx context.Context,
	task models.Task,
	executingAgent string,
	config models.QCAgentConfig,
) (*IntelligentSelectionResult, error) {
	// Check cache first
	cacheKey := GenerateCacheKey(task, executingAgent)
	if cached, found := is.Cache.Get(cacheKey); found {
		return cached, nil
	}

	// Build list of available agents from registry
	availableAgents := is.getAvailableAgents()

	// Build prompt for Claude
	prompt := is.buildSelectionPrompt(task, executingAgent, availableAgents, config)

	// Invoke Claude for recommendation
	recommendation, err := is.invokeClaudeForSelection(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("intelligent selection failed: %w", err)
	}

	// Apply guardrails
	result := is.applyGuardrails(recommendation, config)

	// Cache the result
	is.Cache.Set(cacheKey, result)

	return result, nil
}

// getAvailableAgents returns a list of agent names from the registry
func (is *IntelligentSelector) getAvailableAgents() []string {
	if is.Registry == nil {
		return []string{}
	}

	agents := is.Registry.List()
	names := make([]string, 0, len(agents))
	for _, a := range agents {
		names = append(names, a.Name)
	}
	return names
}

// buildSelectionPrompt creates the prompt for Claude to recommend agents
func (is *IntelligentSelector) buildSelectionPrompt(
	task models.Task,
	executingAgent string,
	availableAgents []string,
	config models.QCAgentConfig,
) string {
	maxAgents := config.MaxAgents
	if maxAgents <= 0 {
		maxAgents = is.MaxAgents
	}

	filesStr := strings.Join(task.Files, ", ")
	agentsStr := strings.Join(availableAgents, ", ")

	prompt := fmt.Sprintf(`You are selecting QC (Quality Control) reviewer agents for a completed task.

TASK CONTEXT:
- Task Number: %s
- Task Name: %s
- Files Modified: [%s]
- Executing Agent: %s
- Task Description: %s

AVAILABLE QC AGENTS (from registry):
%s

INSTRUCTIONS:
Analyze the task context and recommend the best QC agents to review this work. Consider:
1. Domain expertise needed (security, performance, database, API design, etc.)
2. Language/stack expertise based on files modified
3. The executing agent's strengths and potential blind spots
4. Complementary perspectives for thorough review

Return a JSON object with:
- "agents": array of %d or fewer agent names (from the available list)
- "rationale": brief explanation of why these agents were selected

IMPORTANT:
- Only select agents from the AVAILABLE QC AGENTS list
- Prioritize domain specialists (security-auditor, database-optimizer, etc.) for relevant tasks
- Include language experts for stack-specific review
- Maximum %d agents total
- Order by priority (most important first)

RESPONSE FORMAT (JSON only):
{
  "agents": ["agent-name-1", "agent-name-2"],
  "rationale": "Selected security-auditor for JWT implementation review, golang-pro for Go-specific patterns"
}`,
		task.Number,
		task.Name,
		filesStr,
		executingAgent,
		task.Prompt,
		agentsStr,
		maxAgents,
		maxAgents,
	)

	return prompt
}

// invokeClaudeForSelection calls Claude CLI to get agent recommendations
func (is *IntelligentSelector) invokeClaudeForSelection(ctx context.Context, prompt string) (*IntelligentAgentRecommendation, error) {
	// Create context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, is.Timeout)
	defer cancel()

	// Build command args - minimal flags for quick selection
	args := []string{
		"-p", prompt,
		"--output-format", "json",
		"--settings", `{"disableAllHooks": true}`,
	}

	cmd := exec.CommandContext(ctxWithTimeout, is.ClaudePath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("claude invocation failed: %w (output: %s)", err, string(output))
	}

	// Parse Claude's JSON envelope
	parsed, err := agent.ParseClaudeOutput(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse claude output: %w", err)
	}

	// Extract the actual content
	content := parsed.Content
	if content == "" {
		return nil, fmt.Errorf("empty response from claude")
	}

	// Strip markdown code fences if present
	content = stripCodeFences(content)

	// Parse as recommendation
	var recommendation IntelligentAgentRecommendation
	if err := json.Unmarshal([]byte(content), &recommendation); err != nil {
		return nil, fmt.Errorf("failed to parse recommendation JSON: %w (content: %s)", err, content)
	}

	return &recommendation, nil
}

// stripCodeFences removes markdown code fences from JSON response
func stripCodeFences(content string) string {
	content = strings.TrimSpace(content)

	// Remove ```json ... ``` or ``` ... ```
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		if len(lines) >= 2 {
			// Skip first line (```json or ```)
			start := 1
			end := len(lines) - 1

			// Skip last line if it's closing ```
			if end > start && strings.TrimSpace(lines[end]) == "```" {
				end--
			}

			// Rejoin the middle lines
			if start <= end {
				content = strings.Join(lines[start:end+1], "\n")
			}
		}
	}

	return strings.TrimSpace(content)
}

// applyGuardrails validates and constrains the agent selection
func (is *IntelligentSelector) applyGuardrails(
	recommendation *IntelligentAgentRecommendation,
	config models.QCAgentConfig,
) *IntelligentSelectionResult {
	result := &IntelligentSelectionResult{
		Agents:    []string{},
		Rationale: recommendation.Rationale,
	}

	maxAgents := config.MaxAgents
	if maxAgents <= 0 {
		maxAgents = is.MaxAgents
	}

	// Ensure code-reviewer baseline if required
	hasCodeReviewer := false
	if config.RequireCodeReview {
		// Check if code-reviewer exists in registry
		if is.Registry != nil && is.Registry.Exists("code-reviewer") {
			result.Agents = append(result.Agents, "code-reviewer")
			hasCodeReviewer = true
		}
	}

	// Add recommended agents (validate against registry and cap)
	for _, agentName := range recommendation.Agents {
		// Skip if already added
		if agentName == "code-reviewer" && hasCodeReviewer {
			continue
		}

		// Validate against registry
		if is.Registry != nil && !is.Registry.Exists(agentName) {
			continue
		}

		// Check if blocked
		isBlocked := false
		for _, blocked := range config.BlockedAgents {
			if agentName == blocked {
				isBlocked = true
				break
			}
		}
		if isBlocked {
			continue
		}

		// Add if under cap
		if len(result.Agents) < maxAgents {
			result.Agents = append(result.Agents, agentName)
		}
	}

	// Fallback: if no agents selected, add quality-control as baseline
	if len(result.Agents) == 0 {
		if is.Registry != nil && is.Registry.Exists("quality-control") {
			result.Agents = append(result.Agents, "quality-control")
			result.Rationale = "Fallback to quality-control agent"
		}
	}

	return result
}

// IntelligentSelectQCAgents is the main entry point for intelligent selection
// It uses task context and executing agent to recommend QC reviewers
func IntelligentSelectQCAgents(
	ctx context.Context,
	task models.Task,
	executingAgent string,
	config models.QCAgentConfig,
	registry *agent.Registry,
	selector *IntelligentSelector,
) ([]string, string, error) {
	// Use provided selector or create new one
	if selector == nil {
		selector = NewIntelligentSelector(registry, config.CacheTTLSeconds)
	}

	// Ensure registry is set
	if selector.Registry == nil {
		selector.Registry = registry
	}

	// Get intelligent selection
	result, err := selector.SelectAgents(ctx, task, executingAgent, config)
	if err != nil {
		return nil, "", err
	}

	return result.Agents, result.Rationale, nil
}
