package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/learning"
	"github.com/harrison/conductor/internal/models"
)

// QualityController manages quality control reviews using Claude Code agents
type QualityController struct {
	Invoker       InvokerInterface
	ReviewAgent   string               // DEPRECATED: Agent name for single-agent QC (backward compat)
	AgentConfig   models.QCAgentConfig // Multi-agent QC configuration (v2.2+)
	Registry      *agent.Registry      // Agent registry for auto-selection (v2.2+)
	MaxRetries    int                  // Maximum number of retry attempts for RED responses
	LearningStore *learning.Store      // Learning store for historical context (optional)
	Logger        QCLogger             // Logger for QC events (optional, can be nil)
}

// QCLogger is the minimal interface for QC logging functionality
type QCLogger interface {
	LogQCAgentSelection(agents []string, mode string)
	LogQCIndividualVerdicts(verdicts map[string]string)
	LogQCAggregatedResult(verdict string, strategy string)
}

// InvokerInterface defines the interface for agent invocation
type InvokerInterface interface {
	Invoke(ctx context.Context, task models.Task) (*agent.InvocationResult, error)
}

// ReviewResult captures the result of a quality control review
type ReviewResult struct {
	Flag           string // "GREEN", "RED", or "YELLOW"
	Feedback       string // Detailed feedback from the reviewer
	SuggestedAgent string // Alternative agent suggestion for retry
	AgentName      string // Name of agent that produced this result (v2.2+)
}

// NewQualityController creates a new QualityController with default settings
func NewQualityController(invoker InvokerInterface) *QualityController {
	return &QualityController{
		Invoker:     invoker,
		ReviewAgent: "quality-control", // Deprecated, kept for backward compat
		AgentConfig: models.QCAgentConfig{
			Mode:             "auto",
			ExplicitList:     []string{},
			AdditionalAgents: []string{},
			BlockedAgents:    []string{},
		},
		Registry:      nil, // Set externally for auto-selection
		MaxRetries:    2,
		LearningStore: nil, // Set externally when learning is enabled
	}
}

// BuildReviewPrompt creates a comprehensive review prompt for the QC agent
func (qc *QualityController) BuildReviewPrompt(ctx context.Context, task models.Task, output string) string {
	basePrompt := fmt.Sprintf(`Review the following task execution:

Task: %s

Requirements:
%s

Agent Output:
%s
`, task.Name, task.Prompt, output)

	// Load historical context if learning enabled
	if qc.LearningStore != nil {
		if historicalContext, err := qc.LoadContext(ctx, task, qc.LearningStore); err == nil && historicalContext != "" {
			basePrompt += "\n\n" + historicalContext
		}
	}

	basePrompt += `

Provide quality control review in this format:
Quality Control: [GREEN/RED/YELLOW]
Feedback: [your detailed feedback]

Respond with GREEN if all requirements met, RED if needs rework, YELLOW if minor issues.
`

	// Add formatting instructions
	return agent.PrepareAgentPrompt(basePrompt)
}

// buildQCJSONPrompt appends JSON instruction to QC prompts
func (qc *QualityController) buildQCJSONPrompt(basePrompt string) string {
	jsonInstruction := `

IMPORTANT: Respond ONLY with valid JSON in this format:
{
  "verdict": "GREEN|RED|YELLOW",
  "feedback": "Detailed review feedback",
  "issues": [{"severity": "critical|warning|info", "description": "...", "location": "..."}],
  "recommendations": ["suggestion1"],
  "should_retry": false,
  "suggested_agent": "agent-name"
}`
	return basePrompt + jsonInstruction
}

// ParseReviewResponse extracts the QC flag and feedback from agent output
func ParseReviewResponse(output string) (flag string, feedback string) {
	// If output is JSON-wrapped (from claude CLI invocation), extract the "result" field
	if strings.Contains(output, `"result"`) && strings.Contains(output, `"type"`) {
		// Try to extract result field from JSON
		resultRegex := regexp.MustCompile(`"result":\s*"([^"]*(?:\\.[^"]*)*)"`)
		matches := resultRegex.FindStringSubmatch(output)
		if len(matches) > 1 {
			// Unescape the JSON string
			result := matches[1]
			result = strings.ReplaceAll(result, `\"`, `"`)
			result = strings.ReplaceAll(result, `\\n`, "\n")
			result = strings.ReplaceAll(result, `\\`, `\`)
			output = result
		}
	}

	// Try exact format first: "Quality Control: GREEN/RED/YELLOW"
	flagRegex := regexp.MustCompile(`Quality Control:\s*(GREEN|RED|YELLOW)`)
	matches := flagRegex.FindStringSubmatch(output)
	if len(matches) > 1 {
		flag = matches[1]
	}

	// Fallback: look for verdict keywords anywhere in output if exact format not found
	if flag == "" {
		if strings.Contains(output, "GREEN") {
			flag = "GREEN"
		} else if strings.Contains(output, "RED") {
			flag = "RED"
		} else if strings.Contains(output, "YELLOW") {
			flag = "YELLOW"
		}
	}

	// Only extract feedback if a valid flag was found
	if flag != "" {
		// Split by lines and find everything after the flag line
		lines := strings.Split(output, "\n")
		feedbackLines := []string{}
		foundFlag := false

		for _, line := range lines {
			if strings.Contains(line, "Quality Control") {
				foundFlag = true
				continue
			}
			if foundFlag {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					feedbackLines = append(feedbackLines, trimmed)
				}
			}
		}

		// Join all feedback lines preserving the multiline format
		if len(feedbackLines) > 0 {
			feedback = strings.Join(feedbackLines, "\n")
		}
	}

	return flag, feedback
}

// Review executes a quality control review of a task output
// If multi-agent QC is configured (AgentConfig.Mode != "" and not single explicit agent),
// this automatically delegates to ReviewMultiAgent for parallel review.
func (qc *QualityController) Review(ctx context.Context, task models.Task, output string) (*ReviewResult, error) {
	// Check if multi-agent QC is enabled
	if qc.shouldUseMultiAgent() {
		return qc.ReviewMultiAgent(ctx, task, output)
	}

	// Single agent review (legacy behavior)
	basePrompt := qc.BuildReviewPrompt(ctx, task, output)
	prompt := qc.buildQCJSONPrompt(basePrompt)

	// Create a review task for the invoker
	reviewTask := models.Task{
		Number: task.Number,
		Name:   fmt.Sprintf("QC Review: %s", task.Name),
		Prompt: prompt,
		Agent:  qc.ReviewAgent,
	}

	// Invoke the QC agent
	result, err := qc.Invoker.Invoke(ctx, reviewTask)
	if err != nil {
		return nil, fmt.Errorf("QC review failed: %w", err)
	}

	// Try parsing as JSON first
	qcResp, jsonErr := parseQCJSON(result.Output)
	if jsonErr == nil {
		// Success: use JSON response
		return &ReviewResult{
			Flag:           qcResp.Verdict,
			Feedback:       qcResp.Feedback,
			SuggestedAgent: qcResp.SuggestedAgent,
			AgentName:      qc.ReviewAgent,
		}, nil
	}

	// Fallback: use legacy parsing
	flag, feedback := ParseReviewResponse(result.Output)

	return &ReviewResult{
		Flag:      flag,
		Feedback:  feedback,
		AgentName: qc.ReviewAgent,
	}, nil
}

// shouldUseMultiAgent determines if multi-agent QC should be used
func (qc *QualityController) shouldUseMultiAgent() bool {
	// Use multi-agent if:
	// 1. AgentConfig.Mode is set (not empty) AND
	// 2. Either mode is "auto" or "mixed", OR mode is "explicit" with >1 agent
	if qc.AgentConfig.Mode == "" {
		return false // Legacy single-agent mode
	}

	switch qc.AgentConfig.Mode {
	case "auto", "mixed":
		return true // Auto-selection may result in multiple agents
	case "explicit":
		return len(qc.AgentConfig.ExplicitList) > 1 // Multiple explicit agents
	}

	return false
}

// ShouldRetry determines if a task should be retried based on the QC result
func (qc *QualityController) ShouldRetry(result *ReviewResult, currentAttempt int) bool {
	// Only retry if the flag is RED
	if result.Flag != models.StatusRed {
		return false
	}

	// Check if we haven't exceeded max retries
	return currentAttempt < qc.MaxRetries
}

// ReviewMultiAgent performs QC review with multiple agents in parallel (v2.2+)
//
// COST CONSIDERATION: Each agent invocation is a separate Claude API call.
// Multi-agent review multiplies API costs proportionally to the number of agents.
// For example, auto mode with 3+ language-specific agents will result in 3+ API calls per review.
// Consider the cost implications when enabling auto mode, especially for large plans.
func (qc *QualityController) ReviewMultiAgent(ctx context.Context, task models.Task, output string) (*ReviewResult, error) {
	// Select which agents to use based on configuration
	agents := SelectQCAgents(task, qc.AgentConfig, qc.Registry)

	// Check for empty agent list (all agents blocked)
	if len(agents) == 0 {
		return nil, fmt.Errorf("no QC agents available: all agents blocked by configuration")
	}

	// Log agent selection
	if qc.Logger != nil {
		qc.Logger.LogQCAgentSelection(agents, qc.AgentConfig.Mode)
	}

	// If only one agent, fall back to single-agent review for efficiency
	if len(agents) == 1 {
		// Call single-agent review directly (avoid recursion)
		return qc.reviewSingleAgent(ctx, task, output, agents[0])
	}

	// Invoke all agents in parallel
	results := qc.invokeAgentsParallel(ctx, task, output, agents)

	// Log individual verdicts from each agent
	if qc.Logger != nil {
		verdictMap := make(map[string]string)
		for _, result := range results {
			if result != nil {
				verdictMap[result.AgentName] = result.Flag
			}
		}
		qc.Logger.LogQCIndividualVerdicts(verdictMap)
	}

	// Aggregate results using strictest-wins
	final := qc.aggregateVerdicts(results, agents)

	// Log aggregated result
	if qc.Logger != nil {
		qc.Logger.LogQCAggregatedResult(final.Flag, "strictest-wins")
	}

	return final, nil
}

// reviewSingleAgent performs QC review with a single specified agent
func (qc *QualityController) reviewSingleAgent(ctx context.Context, task models.Task, output string, agentName string) (*ReviewResult, error) {
	basePrompt := qc.BuildReviewPrompt(ctx, task, output)
	prompt := qc.buildQCJSONPrompt(basePrompt)

	reviewTask := models.Task{
		Number: task.Number,
		Name:   fmt.Sprintf("QC Review: %s", task.Name),
		Prompt: prompt,
		Agent:  agentName,
	}

	result, err := qc.Invoker.Invoke(ctx, reviewTask)
	if err != nil {
		return nil, fmt.Errorf("QC review failed: %w", err)
	}

	qcResp, jsonErr := parseQCJSON(result.Output)
	if jsonErr == nil {
		return &ReviewResult{
			Flag:           qcResp.Verdict,
			Feedback:       qcResp.Feedback,
			SuggestedAgent: qcResp.SuggestedAgent,
			AgentName:      agentName,
		}, nil
	}

	flag, feedback := ParseReviewResponse(result.Output)
	return &ReviewResult{
		Flag:      flag,
		Feedback:  feedback,
		AgentName: agentName,
	}, nil
}

// invokeAgentsParallel invokes multiple QC agents concurrently
func (qc *QualityController) invokeAgentsParallel(ctx context.Context, task models.Task, output string, agents []string) []*ReviewResult {
	var wg sync.WaitGroup
	results := make([]*ReviewResult, len(agents))
	var mu sync.Mutex

	for i, agentName := range agents {
		wg.Add(1)
		go func(idx int, agent string) {
			defer wg.Done()

			// Build review prompt
			basePrompt := qc.BuildReviewPrompt(ctx, task, output)
			prompt := qc.buildQCJSONPrompt(basePrompt)

			// Create review task for this agent
			reviewTask := models.Task{
				Number: task.Number,
				Name:   fmt.Sprintf("QC Review: %s", task.Name),
				Prompt: prompt,
				Agent:  agent,
			}

			// Invoke the agent
			result, err := qc.Invoker.Invoke(ctx, reviewTask)
			if err != nil {
				mu.Lock()
				results[idx] = &ReviewResult{
					Flag:      "",
					Feedback:  fmt.Sprintf("Agent %s failed: %v", agent, err),
					AgentName: agent,
				}
				mu.Unlock()
				return
			}

			// Parse response
			var reviewResult *ReviewResult
			qcResp, jsonErr := parseQCJSON(result.Output)
			if jsonErr == nil {
				reviewResult = &ReviewResult{
					Flag:           qcResp.Verdict,
					Feedback:       qcResp.Feedback,
					SuggestedAgent: qcResp.SuggestedAgent,
					AgentName:      agent,
				}
			} else {
				flag, feedback := ParseReviewResponse(result.Output)
				reviewResult = &ReviewResult{
					Flag:      flag,
					Feedback:  feedback,
					AgentName: agent,
				}
			}

			mu.Lock()
			results[idx] = reviewResult
			mu.Unlock()
		}(i, agentName)
	}

	wg.Wait()
	return results
}

// aggregateVerdicts combines multiple QC results using strictest-wins strategy
func (qc *QualityController) aggregateVerdicts(results []*ReviewResult, agents []string) *ReviewResult {
	combined := &ReviewResult{
		Flag:     models.StatusGreen,
		Feedback: "",
	}

	var feedbackParts []string
	hasRed := false
	hasYellow := false

	for _, result := range results {
		if result == nil {
			continue
		}

		// Treat empty/invalid flag as YELLOW (indicates agent failure)
		if result.Flag == "" {
			hasYellow = true
			if result.Feedback != "" {
				feedbackParts = append(feedbackParts, fmt.Sprintf("[%s] %s", result.AgentName, result.Feedback))
			}
			continue
		}

		// Aggregate feedback from all agents
		if result.Feedback != "" {
			feedbackParts = append(feedbackParts, fmt.Sprintf("[%s] %s", result.AgentName, result.Feedback))
		}

		// Track agent suggestions (use first non-empty)
		if result.SuggestedAgent != "" && combined.SuggestedAgent == "" {
			combined.SuggestedAgent = result.SuggestedAgent
		}

		// Apply strictest-wins: RED > YELLOW > GREEN
		switch result.Flag {
		case models.StatusRed:
			hasRed = true
		case models.StatusYellow:
			hasYellow = true
		}
	}

	// Determine final verdict
	if hasRed {
		combined.Flag = models.StatusRed
	} else if hasYellow {
		combined.Flag = models.StatusYellow
	} else {
		combined.Flag = models.StatusGreen
	}

	// Combine all feedback
	combined.Feedback = strings.Join(feedbackParts, "\n")

	// Set AgentName to indicate multi-agent review
	combined.AgentName = fmt.Sprintf("multi-agent(%s)", strings.Join(agents, ","))

	return combined
}

// parseQCJSON parses JSON response from QC agent, with strict validation
func parseQCJSON(output string) (*models.QCResponse, error) {
	// Step 1: Extract "result" field from Claude CLI envelope
	// The output is wrapped in a JSON envelope with a nested "result" field
	claudeOut, _ := agent.ParseClaudeOutput(output)
	actualOutput := claudeOut.Content

	// Fallback to raw output if ParseClaudeOutput returns empty content
	if actualOutput == "" {
		actualOutput = output
	}

	// Step 1.5: Extract JSON from markdown code fences if present
	// QC agent may wrap JSON in ```json...```
	if strings.HasPrefix(strings.TrimSpace(actualOutput), "```") {
		actualOutput = extractJSONFromCodeFence(actualOutput)
	}

	var resp models.QCResponse

	// Step 2: Parse extracted JSON as QCResponse
	err := json.Unmarshal([]byte(actualOutput), &resp)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Step 3: Validate parsed JSON
	if err := resp.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &resp, nil
}

// extractJSONFromCodeFence removes markdown code fence wrappers (```json...```)
// Returns the extracted JSON or the original string if no fences found
func extractJSONFromCodeFence(content string) string {
	trimmed := strings.TrimSpace(content)

	// Look for ```json...``` pattern
	if strings.HasPrefix(trimmed, "```json") {
		// Find opening fence
		start := strings.Index(trimmed, "```json")
		if start != -1 {
			// Move past the opening fence and any newline
			start += len("```json")
			if start < len(trimmed) && trimmed[start] == '\n' {
				start++
			}

			// Find closing fence
			end := strings.LastIndex(trimmed, "```")
			if end > start {
				return strings.TrimSpace(trimmed[start:end])
			}
		}
	}

	// Fallback: look for any ``` markers
	if strings.HasPrefix(trimmed, "```") {
		start := strings.Index(trimmed, "```") + 3
		// Skip any language identifier on same line
		if newlineIdx := strings.Index(trimmed[start:], "\n"); newlineIdx != -1 {
			start += newlineIdx + 1
		}

		end := strings.LastIndex(trimmed, "```")
		if end > start {
			return strings.TrimSpace(trimmed[start:end])
		}
	}

	return content
}

// LoadContext loads execution history from database for the task
func (qc *QualityController) LoadContext(ctx context.Context, task models.Task, store *learning.Store) (string, error) {
	var sb strings.Builder

	sb.WriteString("=== Historical Attempts ===\n")

	// Load from database
	history, err := store.GetExecutionHistory(ctx, task.SourceFile, task.Number)
	if err != nil {
		return "", fmt.Errorf("get execution history: %w", err)
	}

	if len(history) == 0 {
		sb.WriteString("No previous attempts found\n")
		return sb.String(), nil
	}

	for i, exec := range history {
		sb.WriteString(fmt.Sprintf("\n--- Attempt %d ---\n", len(history)-i))
		sb.WriteString(fmt.Sprintf("Task: %s\n", exec.TaskName))
		sb.WriteString(fmt.Sprintf("Success: %v\n", exec.Success))
		sb.WriteString(fmt.Sprintf("QC Verdict: %s\n", exec.QCVerdict))
		if exec.QCFeedback != "" {
			sb.WriteString(fmt.Sprintf("QC Feedback: %s\n", exec.QCFeedback))
		}
		if exec.ErrorMessage != "" {
			sb.WriteString(fmt.Sprintf("Error: %s\n", exec.ErrorMessage))
		}
	}

	return sb.String(), nil
}
