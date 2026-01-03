package learning

import (
	"context"
	"encoding/json"
	"fmt"
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

	// inv handles Claude CLI invocation with rate limit retry
	inv *claude.Invoker

	// Logger is for TTS + visual during rate limit wait (passed to Invoker)
	Logger budget.WaiterLogger
}

// NewIntelligentAgentSwapper creates a new IntelligentAgentSwapper with the specified timeout.
// The timeout parameter controls how long to wait for Claude CLI responses.
// Use config.Timeouts.LLM from configuration for consistent timeout values.
func NewIntelligentAgentSwapper(registry *agent.Registry, kg KnowledgeGraph, lipStore LIPCollector, timeout time.Duration, logger budget.WaiterLogger) *IntelligentAgentSwapper {
	inv := claude.NewInvoker()
	inv.Timeout = timeout
	inv.Logger = logger
	return &IntelligentAgentSwapper{
		Registry:       registry,
		KnowledgeGraph: kg,
		LIPStore:       lipStore,
		inv:            inv,
		Logger:         logger,
	}
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
// Uses XML format for Claude 4 optimization
func (ias *IntelligentAgentSwapper) buildSwapPrompt(ctx context.Context, swapCtx *SwapContext) (string, error) {
	var sb strings.Builder

	sb.WriteString("<agent_swap_context>\n")
	sb.WriteString("<role>You are selecting a replacement agent for a failed task retry.</role>\n\n")

	// Task context with nested elements
	sb.WriteString("<task_context>\n")
	sb.WriteString(fmt.Sprintf("<number>%s</number>\n", swapCtx.TaskNumber))
	sb.WriteString(fmt.Sprintf("<name>%s</name>\n", swapCtx.TaskName))
	sb.WriteString(fmt.Sprintf("<current_agent status=\"failed\">%s</current_agent>\n", swapCtx.CurrentAgent))
	sb.WriteString(fmt.Sprintf("<retry_attempt>%d</retry_attempt>\n", swapCtx.AttemptNumber))
	sb.WriteString("</task_context>\n")

	// File extensions analysis
	if len(swapCtx.Files) > 0 {
		sb.WriteString("\n<file_context>\n")
		sb.WriteString("<files>\n")
		for _, file := range swapCtx.Files {
			sb.WriteString(fmt.Sprintf("<file>%s</file>\n", file))
		}
		sb.WriteString("</files>\n")

		// Extract and list file extensions
		extensions := extractFileExtensions(swapCtx.Files)
		if len(extensions) > 0 {
			sb.WriteString("<extensions>\n")
			for _, ext := range extensions {
				sb.WriteString(fmt.Sprintf("<ext>%s</ext>\n", ext))
			}
			sb.WriteString("</extensions>\n")
		}
		sb.WriteString("</file_context>\n")
	}

	// Error context from failed attempt
	if swapCtx.ErrorContext != "" {
		sb.WriteString("\n<error_context source=\"failed_attempt\">\n")
		// Truncate if too long
		errorContext := swapCtx.ErrorContext
		if len(errorContext) > 2000 {
			errorContext = errorContext[:2000] + "... (truncated)"
		}
		sb.WriteString(errorContext)
		sb.WriteString("\n</error_context>\n")
	}

	// Task description
	if swapCtx.TaskDescription != "" {
		sb.WriteString("\n<task_description>\n")
		// Truncate if too long
		taskDesc := swapCtx.TaskDescription
		if len(taskDesc) > 1500 {
			taskDesc = taskDesc[:1500] + "... (truncated)"
		}
		sb.WriteString(taskDesc)
		sb.WriteString("\n</task_description>\n")
	}

	// Knowledge graph context - agents that succeeded with similar files
	if ias.KnowledgeGraph != nil && len(swapCtx.Files) > 0 {
		graphContext := ias.getKnowledgeGraphContext(ctx, swapCtx.Files)
		if graphContext != "" {
			sb.WriteString("\n<historical_context source=\"knowledge_graph\">\n")
			sb.WriteString(graphContext)
			sb.WriteString("</historical_context>\n")
		}
	}

	// LIP progress score context
	if ias.LIPStore != nil {
		lipContext := ias.getLIPContext(ctx, swapCtx)
		if lipContext != "" {
			sb.WriteString("\n<progress_context source=\"lip\">\n")
			sb.WriteString(lipContext)
			sb.WriteString("\n</progress_context>\n")
		}
	}

	// Available agents with name attributes
	sb.WriteString("\n<available_agents>\n")
	availableAgents := ias.getAvailableAgents()
	if len(availableAgents) > 0 {
		for _, agentName := range availableAgents {
			if a, exists := ias.Registry.Get(agentName); exists && a.Description != "" {
				sb.WriteString(fmt.Sprintf("<agent name=\"%s\">%s</agent>\n", agentName, a.Description))
			} else {
				sb.WriteString(fmt.Sprintf("<agent name=\"%s\"/>\n", agentName))
			}
		}
	} else {
		sb.WriteString("<none>No agents available in registry</none>\n")
	}
	sb.WriteString("</available_agents>\n")

	// Instructions
	sb.WriteString(`
<instructions>
<objective>Analyze the task context, error patterns, and file types to recommend the best agent.</objective>

<considerations>
<item priority="1">File extensions - match agent expertise to the language/framework</item>
<item priority="2">Error patterns - what went wrong and which agent can fix it</item>
<item priority="3">Historical success - agents that succeeded with similar files</item>
<item priority="4">Current agent weaknesses - don't recommend the same agent unless no alternative</item>
<item priority="5">Progress made - if significant progress, maybe same approach with different agent</item>
</considerations>

<constraints>
<constraint>Only select agents from the available_agents list</constraint>
<constraint>Prioritize language/framework specialists for the file types involved</constraint>
<constraint>Consider the error context to understand what expertise is needed</constraint>
<constraint>If the current agent is specialized for these files, consider if the error suggests a different approach</constraint>
</constraints>

<response_format type="json">
<field name="recommended_agent" required="true">The single best agent name</field>
<field name="rationale" required="true">Brief explanation of why this agent was selected</field>
<field name="confidence" required="true">How certain you are (0.0-1.0)</field>
<field name="alternatives" required="false">Array of 1-2 alternative agent names</field>
<example>{"recommended_agent":"agent-name","rationale":"Selected because...","confidence":0.85,"alternatives":["alt-agent"]}</example>
</response_format>
</instructions>

</agent_swap_context>
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
// Uses claude.Invoker for CLI invocation with built-in rate limit handling.
func (ias *IntelligentAgentSwapper) invokeClaudeForSwap(ctx context.Context, prompt string) (*AgentSwapRecommendation, error) {
	req := claude.Request{
		Prompt: prompt,
		Schema: AgentSwapSchema(),
	}

	// Invoke Claude CLI (rate limit handling is in Invoker)
	resp, err := ias.inv.Invoke(ctx, req)
	if err != nil {
		return nil, err
	}

	// Parse the response
	content, _, err := claude.ParseResponse(resp.RawOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to parse claude output: %w", err)
	}

	if content == "" {
		return nil, fmt.Errorf("empty response from claude")
	}

	var result AgentSwapRecommendation
	if err := json.Unmarshal([]byte(content), &result); err != nil {
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
