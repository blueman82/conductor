package learning

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/budget"
	"github.com/harrison/conductor/internal/claude"
)

// AgentSwapRecommendation represents Claude's response for agent swap selection
type AgentSwapRecommendation struct {
	// RecommendedAgent is the agent Claude recommends for the task
	RecommendedAgent string `json:"recommended_agent"`

	// Rationale explains why this agent was selected
	Rationale string `json:"rationale"`

	// Confidence indicates how certain Claude is about this recommendation (0.0-1.0)
	Confidence float64 `json:"confidence"`

	// Alternatives lists other agents that could also handle the task
	Alternatives []string `json:"alternatives,omitempty"`
}

// SwapContext contains context for intelligent agent selection
type SwapContext struct {
	// TaskNumber is the unique identifier for the task
	TaskNumber string

	// TaskName is the descriptive name of the task
	TaskName string

	// TaskDescription is the full task description/prompt
	TaskDescription string

	// Files are the files this task modifies
	Files []string

	// CurrentAgent is the agent that failed
	CurrentAgent string

	// ErrorContext contains error details from the failed attempt
	ErrorContext string

	// AttemptNumber is which retry attempt this is (1-indexed)
	AttemptNumber int
}

// IntelligentAgentSwapper provides AI-powered agent selection based on comprehensive context
type IntelligentAgentSwapper struct {
	// Registry provides access to available agents
	Registry *agent.Registry

	// KnowledgeGraph provides historical context about agent-file relationships
	KnowledgeGraph KnowledgeGraph

	// LIPStore provides progress scores from previous attempts
	LIPStore LIPCollector

	// ClaudePath is the path to the Claude CLI executable
	ClaudePath string

	// Timeout is the maximum time to wait for Claude response
	Timeout time.Duration

	// Logger is for TTS + visual during rate limit wait
	Logger budget.WaiterLogger
}

// NewIntelligentAgentSwapper creates a new IntelligentAgentSwapper with defaults
func NewIntelligentAgentSwapper(registry *agent.Registry, kg KnowledgeGraph, lipStore LIPCollector, logger budget.WaiterLogger) *IntelligentAgentSwapper {
	return &IntelligentAgentSwapper{
		Registry:       registry,
		KnowledgeGraph: kg,
		LIPStore:       lipStore,
		ClaudePath:     "claude",
		Timeout:        90 * time.Second,
		Logger:         logger,
	}
}

// NewIntelligentAgentSwapperWithTimeout creates a swapper with custom timeout
func NewIntelligentAgentSwapperWithTimeout(registry *agent.Registry, kg KnowledgeGraph, lipStore LIPCollector, timeout time.Duration, logger budget.WaiterLogger) *IntelligentAgentSwapper {
	swapper := NewIntelligentAgentSwapper(registry, kg, lipStore, logger)
	swapper.Timeout = timeout
	return swapper
}

// SelectAgent uses Claude to recommend the best agent for a task retry
// It considers file extensions, knowledge graph history, LIP progress, and error context
func (ias *IntelligentAgentSwapper) SelectAgent(ctx context.Context, swapCtx *SwapContext) (*AgentSwapRecommendation, error) {
	if swapCtx == nil {
		return nil, fmt.Errorf("swap context cannot be nil")
	}

	// Build comprehensive prompt with all available context
	prompt, err := ias.buildSwapPrompt(ctx, swapCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to build swap prompt: %w", err)
	}

	// Invoke Claude for recommendation with rate limit handling
	recommendation, err := ias.invokeClaudeForSwap(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Apply guardrails to ensure valid agent selection
	return ias.applyGuardrails(recommendation), nil
}

// buildSwapPrompt creates a comprehensive prompt for Claude with all available context
func (ias *IntelligentAgentSwapper) buildSwapPrompt(ctx context.Context, swapCtx *SwapContext) (string, error) {
	var sb strings.Builder

	sb.WriteString("You are selecting a replacement agent for a failed task retry.\n\n")

	// Task context
	sb.WriteString("## TASK CONTEXT\n")
	sb.WriteString(fmt.Sprintf("- Task Number: %s\n", swapCtx.TaskNumber))
	sb.WriteString(fmt.Sprintf("- Task Name: %s\n", swapCtx.TaskName))
	sb.WriteString(fmt.Sprintf("- Current Agent (failed): %s\n", swapCtx.CurrentAgent))
	sb.WriteString(fmt.Sprintf("- Retry Attempt: %d\n", swapCtx.AttemptNumber))

	// File extensions analysis
	if len(swapCtx.Files) > 0 {
		sb.WriteString("\n## FILE CONTEXT\n")
		sb.WriteString(fmt.Sprintf("Files to modify: %s\n", strings.Join(swapCtx.Files, ", ")))

		// Extract and list file extensions
		extensions := extractFileExtensions(swapCtx.Files)
		if len(extensions) > 0 {
			sb.WriteString(fmt.Sprintf("File extensions: %s\n", strings.Join(extensions, ", ")))
		}
	}

	// Error context from failed attempt
	if swapCtx.ErrorContext != "" {
		sb.WriteString("\n## ERROR CONTEXT (from failed attempt)\n")
		// Truncate if too long
		errorContext := swapCtx.ErrorContext
		if len(errorContext) > 2000 {
			errorContext = errorContext[:2000] + "... (truncated)"
		}
		sb.WriteString(errorContext)
		sb.WriteString("\n")
	}

	// Task description
	if swapCtx.TaskDescription != "" {
		sb.WriteString("\n## TASK DESCRIPTION\n")
		// Truncate if too long
		taskDesc := swapCtx.TaskDescription
		if len(taskDesc) > 1500 {
			taskDesc = taskDesc[:1500] + "... (truncated)"
		}
		sb.WriteString(taskDesc)
		sb.WriteString("\n")
	}

	// Knowledge graph context - agents that succeeded with similar files
	if ias.KnowledgeGraph != nil && len(swapCtx.Files) > 0 {
		graphContext := ias.getKnowledgeGraphContext(ctx, swapCtx.Files)
		if graphContext != "" {
			sb.WriteString("\n## HISTORICAL CONTEXT (from knowledge graph)\n")
			sb.WriteString(graphContext)
			sb.WriteString("\n")
		}
	}

	// LIP progress score context
	if ias.LIPStore != nil {
		lipContext := ias.getLIPContext(ctx, swapCtx)
		if lipContext != "" {
			sb.WriteString("\n## PROGRESS CONTEXT (from LIP)\n")
			sb.WriteString(lipContext)
			sb.WriteString("\n")
		}
	}

	// Available agents
	sb.WriteString("\n## AVAILABLE AGENTS\n")
	availableAgents := ias.getAvailableAgents()
	if len(availableAgents) > 0 {
		for _, agentName := range availableAgents {
			if a, exists := ias.Registry.Get(agentName); exists && a.Description != "" {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", agentName, a.Description))
			} else {
				sb.WriteString(fmt.Sprintf("- %s\n", agentName))
			}
		}
	} else {
		sb.WriteString("(no agents available in registry)\n")
	}

	// Instructions
	sb.WriteString(`
## INSTRUCTIONS
Analyze the task context, error patterns, and file types to recommend the best agent.

Consider:
1. File extensions - match agent expertise to the language/framework
2. Error patterns - what went wrong and which agent can fix it
3. Historical success - agents that succeeded with similar files
4. Current agent weaknesses - don't recommend the same agent unless no alternative
5. Progress made - if significant progress, maybe same approach with different agent

IMPORTANT:
- Only select agents from the AVAILABLE AGENTS list
- Prioritize language/framework specialists for the file types involved
- Consider the error context to understand what expertise is needed
- If the current agent is specialized for these files, consider if the error suggests a different approach

Return a JSON object with:
- "recommended_agent": the single best agent name
- "rationale": brief explanation of why this agent was selected
- "confidence": how certain you are (0.0-1.0)
- "alternatives": array of 1-2 alternative agent names (optional)

RESPONSE FORMAT (JSON only):
{"recommended_agent":"agent-name","rationale":"Selected because...","confidence":0.85,"alternatives":["alt-agent"]}
`)

	return sb.String(), nil
}

// getKnowledgeGraphContext queries the knowledge graph for agents that succeeded with similar files
func (ias *IntelligentAgentSwapper) getKnowledgeGraphContext(ctx context.Context, files []string) string {
	var sb strings.Builder
	agentSuccessCount := make(map[string]int)

	// For each file, find agents that succeeded
	for _, filePath := range files {
		// Create a file node ID (normalized path)
		fileID := normalizeFileID(filePath)

		// Query knowledge graph for agents related to this file
		agents, err := ias.KnowledgeGraph.GetRelated(ctx, fileID, 2, []EdgeType{EdgeTypeSucceededWith, EdgeTypeModifies})
		if err != nil {
			continue
		}

		for _, node := range agents {
			if node.NodeType == NodeTypeAgent {
				// Extract agent name from properties
				if name, ok := node.Properties["name"].(string); ok {
					agentSuccessCount[name]++
				}
			}
		}
	}

	if len(agentSuccessCount) > 0 {
		sb.WriteString("Agents with historical success on similar files:\n")
		for agentName, count := range agentSuccessCount {
			sb.WriteString(fmt.Sprintf("- %s: %d successful tasks\n", agentName, count))
		}
	}

	return sb.String()
}

// getLIPContext gets progress information from LIP store
func (ias *IntelligentAgentSwapper) getLIPContext(ctx context.Context, swapCtx *SwapContext) string {
	// We don't have task execution ID here directly, so provide general guidance
	// In a full implementation, we'd look up the task execution ID
	return "Previous attempt analysis available - consider if partial progress was made."
}

// getAvailableAgents returns a list of agent names from the registry
func (ias *IntelligentAgentSwapper) getAvailableAgents() []string {
	if ias.Registry == nil {
		return []string{}
	}

	return ias.Registry.ListNames()
}

// invokeClaudeForSwap calls Claude CLI to get agent swap recommendation
func (ias *IntelligentAgentSwapper) invokeClaudeForSwap(ctx context.Context, prompt string) (*AgentSwapRecommendation, error) {
	result, err := ias.invoke(ctx, prompt)

	// Handle rate limit with retry (TTS + visual countdown)
	if err != nil {
		if info := budget.ParseRateLimitFromError(err.Error()); info != nil {
			// Use 24h as max - waiter uses actual reset time from info
			waiter := budget.NewRateLimitWaiter(24*time.Hour, 15*time.Second, 30*time.Second, ias.Logger)
			if waiter.ShouldWait(info) {
				if waitErr := waiter.WaitForReset(ctx, info); waitErr != nil {
					return nil, waitErr
				}
				// Retry once after wait
				return ias.invoke(ctx, prompt)
			}
		}
		return nil, err
	}

	return result, nil
}

// invoke performs the actual Claude CLI call (follows qc_intelligent.go pattern)
func (ias *IntelligentAgentSwapper) invoke(ctx context.Context, prompt string) (*AgentSwapRecommendation, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, ias.Timeout)
	defer cancel()

	args := []string{
		"-p", prompt,
		"--json-schema", AgentSwapSchema(),
		"--output-format", "json",
		"--settings", `{"disableAllHooks": true}`,
	}

	cmd := exec.CommandContext(ctxWithTimeout, ias.ClaudePath, args...)
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
		return nil, fmt.Errorf("empty response from claude")
	}

	var result AgentSwapRecommendation
	if err := json.Unmarshal([]byte(parsed.Content), &result); err != nil {
		// Try extracting JSON from mixed output (fallback)
		start := strings.Index(parsed.Content, "{")
		end := strings.LastIndex(parsed.Content, "}")
		if start >= 0 && end > start {
			if err := json.Unmarshal([]byte(parsed.Content[start:end+1]), &result); err != nil {
				return nil, fmt.Errorf("failed to extract JSON: %w", err)
			}
			return &result, nil
		}
		return nil, fmt.Errorf("failed to parse swap recommendation: %w", err)
	}

	return &result, nil
}

// applyGuardrails validates and constrains the agent selection
func (ias *IntelligentAgentSwapper) applyGuardrails(recommendation *AgentSwapRecommendation) *AgentSwapRecommendation {
	if recommendation == nil {
		return &AgentSwapRecommendation{
			RecommendedAgent: "",
			Rationale:        "No recommendation provided",
			Confidence:       0.0,
		}
	}

	// Validate recommended agent exists in registry
	if ias.Registry != nil && recommendation.RecommendedAgent != "" {
		if !ias.Registry.Exists(recommendation.RecommendedAgent) {
			// Try to find a valid alternative
			for _, alt := range recommendation.Alternatives {
				if ias.Registry.Exists(alt) {
					recommendation.RecommendedAgent = alt
					recommendation.Rationale = fmt.Sprintf("Fallback to %s (original recommendation not in registry)", alt)
					recommendation.Confidence = recommendation.Confidence * 0.8 // Reduce confidence
					break
				}
			}

			// If still invalid, clear recommendation
			if !ias.Registry.Exists(recommendation.RecommendedAgent) {
				recommendation.RecommendedAgent = ""
				recommendation.Rationale = "No valid agent found in registry"
				recommendation.Confidence = 0.0
			}
		}
	}

	// Filter alternatives to only valid agents
	if len(recommendation.Alternatives) > 0 && ias.Registry != nil {
		validAlts := make([]string, 0, len(recommendation.Alternatives))
		for _, alt := range recommendation.Alternatives {
			if ias.Registry.Exists(alt) {
				validAlts = append(validAlts, alt)
			}
		}
		recommendation.Alternatives = validAlts
	}

	// Cap confidence to valid range
	if recommendation.Confidence < 0.0 {
		recommendation.Confidence = 0.0
	}
	if recommendation.Confidence > 1.0 {
		recommendation.Confidence = 1.0
	}

	return recommendation
}

// extractFileExtensions extracts unique file extensions from a list of file paths
func extractFileExtensions(files []string) []string {
	extMap := make(map[string]bool)
	for _, file := range files {
		ext := filepath.Ext(file)
		if ext != "" {
			// Remove leading dot and normalize to lowercase
			ext = strings.ToLower(strings.TrimPrefix(ext, "."))
			extMap[ext] = true
		}
	}

	extensions := make([]string, 0, len(extMap))
	for ext := range extMap {
		extensions = append(extensions, ext)
	}
	return extensions
}

// normalizeFileID creates a normalized file ID for knowledge graph lookup
func normalizeFileID(filePath string) string {
	// Normalize path separators and clean the path
	filePath = filepath.Clean(filePath)
	filePath = strings.ToLower(filePath)
	return "file:" + filePath
}

// AgentSwapSchema returns the JSON schema for agent swap recommendations
func AgentSwapSchema() string {
	return `{"type":"object","properties":{"recommended_agent":{"type":"string"},"rationale":{"type":"string"},"confidence":{"type":"number","minimum":0,"maximum":1},"alternatives":{"type":"array","items":{"type":"string"}}},"required":["recommended_agent","rationale","confidence"]}`
}
