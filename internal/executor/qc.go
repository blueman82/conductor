package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/behavioral"
	"github.com/harrison/conductor/internal/learning"
	"github.com/harrison/conductor/internal/models"
)

// domainSpecificChecks maps file extensions to domain-specific QC review criteria
var domainSpecificChecks = map[string]string{
	".go": `## GO-SPECIFIC REVIEW CRITERIA
- Error handling follows if err != nil pattern
- No nil pointer dereferences on interfaces/pointers
- Context cancellation respected (no goroutine leaks)
- strings.Builder used for string concatenation (not +=)
`,
	".sql": `## SQL-SPECIFIC REVIEW CRITERIA
- No SQL injection vulnerabilities (parameterized queries)
- Indexes exist for WHERE/JOIN columns
- Foreign keys have explicit ON DELETE behavior
- Rollback migration reverses all changes safely
`,
	".ts": `## TYPESCRIPT-SPECIFIC REVIEW CRITERIA
- No 'any' types unless explicitly justified
- Async/await properly handled (no floating promises)
- Type narrowing used correctly
- No unused imports or variables
`,
	".tsx": `## REACT-SPECIFIC REVIEW CRITERIA
- React hooks follow rules (no conditional hooks)
- useEffect has cleanup functions where needed
- Props properly typed (no implicit any)
- Event handlers properly typed
`,
	".py": `## PYTHON-SPECIFIC REVIEW CRITERIA
- Type hints used for function signatures
- Exception handling is specific (not bare except)
- No mutable default arguments
- Resource cleanup with context managers
`,
}

// inferDomainChecks infers domain-specific review criteria from task file extensions
func inferDomainChecks(files []string) string {
	var checks []string
	seenDomains := map[string]bool{}

	for _, file := range files {
		ext := filepath.Ext(file)
		if !seenDomains[ext] {
			if criteria, ok := domainSpecificChecks[ext]; ok {
				checks = append(checks, criteria)
				seenDomains[ext] = true
			}
		}
	}
	return strings.Join(checks, "\n")
}

// QualityController manages quality control reviews using Claude Code agents
type QualityController struct {
	Invoker             InvokerInterface
	ReviewAgent         string               // DEPRECATED: Agent name for single-agent QC (backward compat)
	AgentConfig         models.QCAgentConfig // Multi-agent QC configuration (v2.2+)
	Registry            *agent.Registry      // Agent registry for auto-selection (v2.2+)
	MaxRetries          int                  // Maximum number of retry attempts for RED responses
	LearningStore       *learning.Store      // Learning store for historical context (optional)
	Logger              QCLogger             // Logger for QC events (optional, can be nil)
	IntelligentSelector *IntelligentSelector // Intelligent selector for mode="intelligent" (v2.4+)
	BehavioralMetrics   *BehavioralMetricsProvider // Provider for behavioral metrics context (v2.7+)
}

// QCLogger is the minimal interface for QC logging functionality
type QCLogger interface {
	LogQCAgentSelection(agents []string, mode string)
	LogQCIndividualVerdicts(verdicts map[string]string)
	LogQCAggregatedResult(verdict string, strategy string)
	LogQCCriteriaResults(agentName string, results []models.CriterionResult)
	LogQCIntelligentSelectionMetadata(rationale string, fallback bool, fallbackReason string)
}

// BehavioralMetricsProvider provides behavioral metrics for QC context injection
type BehavioralMetricsProvider interface {
	// GetMetrics returns behavioral metrics for the current task execution
	GetMetrics(ctx context.Context, taskNumber string) *behavioral.BehavioralMetrics
}

// InvokerInterface defines the interface for agent invocation
type InvokerInterface interface {
	Invoke(ctx context.Context, task models.Task) (*agent.InvocationResult, error)
}

// ReviewResult captures the result of a quality control review
type ReviewResult struct {
	Flag            string                   // "GREEN", "RED", or "YELLOW"
	Feedback        string                   // Detailed feedback from the reviewer
	SuggestedAgent  string                   // Alternative agent suggestion for retry
	AgentName       string                   // Name of agent that produced this result (v2.2+)
	CriteriaResults []models.CriterionResult // Per-criterion verification results (v2.2+)
}

// NewQualityController creates a new QualityController with default settings
func NewQualityController(invoker InvokerInterface) *QualityController {
	return &QualityController{
		Invoker:     invoker,
		ReviewAgent: "quality-control", // Deprecated, kept for backward compat
		AgentConfig: models.QCAgentConfig{
			Mode:              "auto",
			ExplicitList:      []string{},
			AdditionalAgents:  []string{},
			BlockedAgents:     []string{},
			MaxAgents:         4,
			CacheTTLSeconds:   3600,
			RequireCodeReview: true,
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

	// Inject behavioral metrics context if provider configured
	if behaviorContext := qc.injectBehaviorContext(ctx, task); behaviorContext != "" {
		basePrompt += "\n\n" + behaviorContext
	}

	// Add formatting instructions
	return agent.PrepareAgentPrompt(basePrompt)
}

// getCombinedCriteria merges success_criteria and integration_criteria into single array
// for structured review prompts. Integration criteria appended after success criteria.
func getCombinedCriteria(task models.Task) []string {
	criteria := make([]string, 0, len(task.SuccessCriteria)+len(task.IntegrationCriteria))
	criteria = append(criteria, task.SuccessCriteria...)
	if task.Type == "integration" {
		criteria = append(criteria, task.IntegrationCriteria...)
	}
	return criteria
}

// BuildStructuredReviewPrompt creates a QC prompt with numbered success criteria for verification.
// Falls back to legacy prompt structure for tasks without explicit criteria.
func (qc *QualityController) BuildStructuredReviewPrompt(ctx context.Context, task models.Task, output string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Quality Control Review: %s\n\n", task.Name))
	sb.WriteString("## Task Requirements\n")
	sb.WriteString(task.Prompt)
	sb.WriteString("\n\n")

	// Get combined criteria (success + integration)
	allCriteria := getCombinedCriteria(task)

	if len(allCriteria) > 0 {
		sb.WriteString("## SUCCESS CRITERIA - VERIFY EACH ONE\n\n")
		for i, criterion := range allCriteria {
			sb.WriteString(fmt.Sprintf("%d. [ ] %s\n", i, criterion))
		}
		sb.WriteString("\n")

		// Add integration note if this is an integration task with integration criteria
		if task.Type == "integration" && len(task.IntegrationCriteria) > 0 {
			sb.WriteString("## INTEGRATION CRITERIA - VERIFY EACH ONE\n\n")
			startIdx := len(task.SuccessCriteria)
			for i, criterion := range task.IntegrationCriteria {
				sb.WriteString(fmt.Sprintf("%d. [ ] %s\n", startIdx+i, criterion))
			}
			sb.WriteString("\n")
		}
	}

	if len(task.TestCommands) > 0 {
		sb.WriteString("## TEST COMMANDS\n")
		for _, cmd := range task.TestCommands {
			sb.WriteString(fmt.Sprintf("- `%s`\n", cmd))
		}
		sb.WriteString("\n")
	}

	// Add expected file paths for verification
	if len(task.Files) > 0 {
		sb.WriteString("## EXPECTED FILE PATHS - VERIFY EXACT MATCH\n")
		sb.WriteString("Agent output must show files_modified matching these paths:\n")
		for _, file := range task.Files {
			sb.WriteString(fmt.Sprintf("- [ ] %s\n", file))
		}
		sb.WriteString("\n")
	}

	// Add domain-specific review criteria based on file extensions
	if domainCriteria := inferDomainChecks(task.Files); domainCriteria != "" {
		sb.WriteString(domainCriteria)
		sb.WriteString("\n")
	}

	sb.WriteString("## AGENT OUTPUT\n```\n")
	sb.WriteString(output)
	sb.WriteString("\n```\n\n")

	// Load historical context if learning enabled
	if qc.LearningStore != nil {
		if historicalContext, err := qc.LoadContext(ctx, task, qc.LearningStore); err == nil && historicalContext != "" {
			sb.WriteString(historicalContext)
			sb.WriteString("\n\n")
		}
	}

	// Inject behavioral metrics context if provider configured
	if behaviorContext := qc.injectBehaviorContext(ctx, task); behaviorContext != "" {
		sb.WriteString(behaviorContext)
		sb.WriteString("\n")
	}

	sb.WriteString("## RESPONSE FORMAT\nYou MUST ONLY respond with JSON including criteria_results array.\n")

	return sb.String()
}

// buildQCJSONPrompt appends JSON instruction to QC prompts
func (qc *QualityController) buildQCJSONPrompt(basePrompt string) string {
	return qc.buildQCJSONPromptWithCriteria(basePrompt, false)
}

// buildQCJSONPromptWithCriteria appends JSON instruction with optional criteria_results schema
func (qc *QualityController) buildQCJSONPromptWithCriteria(basePrompt string, hasSuccessCriteria bool) string {
	var jsonInstruction string
	if hasSuccessCriteria {
		jsonInstruction = `

IMPORTANT: You MUST ONLY respond with valid JSON in this format:
{
  "verdict": "GREEN|RED|YELLOW",
  "feedback": "Detailed review feedback",
  "criteria_results": [
    {"index": 0, "criterion": "first criterion text", "passed": true, "evidence": "evidence or fail reason"},
    {"index": 1, "criterion": "second criterion text", "passed": false, "evidence": "why it failed"}
  ],
  "issues": [{"severity": "critical|warning|info", "description": "...", "location": "..."}],
  "recommendations": ["suggestion1"],
  "should_retry": false,
  "suggested_agent": "agent-name"
}

Each criterion MUST have a corresponding entry in criteria_results with its index and pass/fail status.`
	} else {
		jsonInstruction = `

IMPORTANT: You MUST ONLY respond with valid JSON in this format:
{
  "verdict": "GREEN|RED|YELLOW",
  "feedback": "Detailed review feedback",
  "issues": [{"severity": "critical|warning|info", "description": "...", "location": "..."}],
  "recommendations": ["suggestion1"],
  "should_retry": false,
  "suggested_agent": "agent-name"
}`
	}
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
	var basePrompt string
	allCriteria := getCombinedCriteria(task)
	hasSuccessCriteria := len(allCriteria) > 0
	if hasSuccessCriteria {
		// Use structured prompt for tasks with explicit success criteria
		basePrompt = qc.BuildStructuredReviewPrompt(ctx, task, output)
	} else {
		// Use legacy prompt for backward compatibility
		basePrompt = qc.BuildReviewPrompt(ctx, task, output)
	}
	prompt := qc.buildQCJSONPromptWithCriteria(basePrompt, hasSuccessCriteria)

	// Create a review task for the invoker
	reviewTask := models.Task{
		Number: task.Number,
		Name:   fmt.Sprintf("QC Review: %s", task.Name),
		Prompt: prompt,
		Agent:  qc.ReviewAgent,
	}

	// Invoke the QC agent and parse response with automatic retry on JSON validation failure
	qcResp, jsonErr := qc.invokeAndParseQCAgent(ctx, reviewTask, qc.ReviewAgent)
	if jsonErr != nil {
		// Invocation failed (not just JSON parsing)
		return nil, fmt.Errorf("QC review failed: %w", jsonErr)
	}

	// Success: use JSON response
	reviewResult := &ReviewResult{
		Flag:            qcResp.Verdict,
		Feedback:        qcResp.Feedback,
		SuggestedAgent:  qcResp.SuggestedAgent,
		AgentName:       qc.ReviewAgent,
		CriteriaResults: qcResp.CriteriaResults,
	}

	// Apply criteria aggregation if task has success criteria or integration criteria
	if len(allCriteria) > 0 {
		if len(qcResp.CriteriaResults) == 0 {
			// Agent didn't follow schema - treat as YELLOW (AI Counsel consensus)
			reviewResult.Flag = models.StatusYellow
			reviewResult.Feedback += " (Warning: Agent did not return criteria_results)"
		} else {
			// Override verdict based on criteria aggregation
			reviewResult.Flag = qc.aggregateCriteriaResults(qcResp, task, len(allCriteria))
		}
	}

	return reviewResult, nil
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
	case "auto", "mixed", "intelligent":
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
	var agents []string
	var selCtx *SelectionContext

	// Use intelligent selection if mode is "intelligent" and selector is available
	if qc.AgentConfig.Mode == "intelligent" {
		// Ensure intelligent selector is initialized
		if qc.IntelligentSelector == nil && qc.Registry != nil {
			qc.IntelligentSelector = NewIntelligentSelector(qc.Registry, qc.AgentConfig.CacheTTLSeconds)
		}

		selCtx = &SelectionContext{
			ExecutingAgent:      task.Agent,
			IntelligentSelector: qc.IntelligentSelector,
		}
		agents = SelectQCAgentsWithContext(ctx, task, qc.AgentConfig, qc.Registry, selCtx)
	} else {
		agents = SelectQCAgents(task, qc.AgentConfig, qc.Registry)
	}

	// Check for empty agent list (all agents blocked)
	if len(agents) == 0 {
		return nil, fmt.Errorf("no QC agents available: all agents blocked by configuration")
	}

	// Log agent selection
	if qc.Logger != nil {
		qc.Logger.LogQCAgentSelection(agents, qc.AgentConfig.Mode)

		// Log intelligent selection metadata if available
		if selCtx != nil && qc.AgentConfig.Mode == "intelligent" {
			qc.Logger.LogQCIntelligentSelectionMetadata(selCtx.Rationale, selCtx.FallbackOccured, selCtx.FallbackReason)
		}
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

	// Aggregate results: use per-criterion consensus for tasks with success or integration criteria,
	// otherwise use strictest-wins for legacy tasks
	var final *ReviewResult
	allCriteria := getCombinedCriteria(task)
	if len(allCriteria) > 0 {
		// Use multi-agent criteria aggregation (unanimous consensus)
		final = qc.aggregateMultiAgentCriteria(results, task, len(allCriteria))
		final.AgentName = fmt.Sprintf("multi-agent(%s)", strings.Join(agents, ","))

		// Log aggregated result
		if qc.Logger != nil {
			qc.Logger.LogQCAggregatedResult(final.Flag, "multi-agent-criteria-consensus")
		}
	} else {
		// Use strictest-wins for legacy tasks without success criteria
		final = qc.aggregateVerdicts(results, agents)

		// Log aggregated result
		if qc.Logger != nil {
			qc.Logger.LogQCAggregatedResult(final.Flag, "strictest-wins")
		}
	}

	return final, nil
}

// reviewSingleAgent performs QC review with a single specified agent
func (qc *QualityController) reviewSingleAgent(ctx context.Context, task models.Task, output string, agentName string) (*ReviewResult, error) {
	var basePrompt string
	allCriteria := getCombinedCriteria(task)
	hasSuccessCriteria := len(allCriteria) > 0
	if hasSuccessCriteria {
		basePrompt = qc.BuildStructuredReviewPrompt(ctx, task, output)
	} else {
		basePrompt = qc.BuildReviewPrompt(ctx, task, output)
	}
	prompt := qc.buildQCJSONPromptWithCriteria(basePrompt, hasSuccessCriteria)

	reviewTask := models.Task{
		Number: task.Number,
		Name:   fmt.Sprintf("QC Review: %s", task.Name),
		Prompt: prompt,
		Agent:  agentName,
	}

	qcResp, jsonErr := qc.invokeAndParseQCAgent(ctx, reviewTask, agentName)
	if jsonErr != nil {
		// Invocation or JSON parsing failed - return error
		return nil, fmt.Errorf("QC review failed: %w", jsonErr)
	}

	return &ReviewResult{
		Flag:            qcResp.Verdict,
		Feedback:        qcResp.Feedback,
		SuggestedAgent:  qcResp.SuggestedAgent,
		AgentName:       agentName,
		CriteriaResults: qcResp.CriteriaResults,
	}, nil
}

// invokeAgentsParallel invokes multiple QC agents concurrently
func (qc *QualityController) invokeAgentsParallel(ctx context.Context, task models.Task, output string, agents []string) []*ReviewResult {
	var wg sync.WaitGroup
	results := make([]*ReviewResult, len(agents))
	var mu sync.Mutex

	// Get combined criteria once (same for all agents)
	allCriteria := getCombinedCriteria(task)
	hasSuccessCriteria := len(allCriteria) > 0

	for i, agentName := range agents {
		wg.Add(1)
		go func(idx int, agent string) {
			defer wg.Done()

			// Build review prompt
			var basePrompt string
			if hasSuccessCriteria {
				basePrompt = qc.BuildStructuredReviewPrompt(ctx, task, output)
			} else {
				basePrompt = qc.BuildReviewPrompt(ctx, task, output)
			}
			prompt := qc.buildQCJSONPromptWithCriteria(basePrompt, hasSuccessCriteria)

			// Create review task for this agent
			reviewTask := models.Task{
				Number: task.Number,
				Name:   fmt.Sprintf("QC Review: %s", task.Name),
				Prompt: prompt,
				Agent:  agent,
			}

			// Invoke agent and parse response
			qcResp, jsonErr := qc.invokeAndParseQCAgent(ctx, reviewTask, agent)
			if jsonErr != nil {
				mu.Lock()
				results[idx] = &ReviewResult{
					Flag:      "",
					Feedback:  fmt.Sprintf("Agent %s failed: %v", agent, jsonErr),
					AgentName: agent,
				}
				mu.Unlock()
				return
			}

			// Parse successful
			var reviewResult *ReviewResult
			reviewResult = &ReviewResult{
				Flag:            qcResp.Verdict,
				Feedback:        qcResp.Feedback,
				SuggestedAgent:  qcResp.SuggestedAgent,
				AgentName:       agent,
				CriteriaResults: qcResp.CriteriaResults,
			}
			// Log criteria results if available
			if qc.Logger != nil && len(qcResp.CriteriaResults) > 0 {
				qc.Logger.LogQCCriteriaResults(agent, qcResp.CriteriaResults)
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
	// QC agent may wrap JSON in ```json...``` anywhere in output (not just at start)
	// Use case-insensitive check for ```json (handles ```JSON, ```Json, etc.)
	lowerOutput := strings.ToLower(actualOutput)
	if strings.Contains(lowerOutput, "```json") || strings.HasPrefix(strings.TrimSpace(actualOutput), "```") {
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

	// Look for ```json...``` pattern ANYWHERE in content (case-insensitive)
	// This handles both ```json and ```JSON variants
	lowerContent := strings.ToLower(trimmed)
	if strings.Contains(lowerContent, "```json") {
		// Find opening fence (case-insensitive)
		start := strings.Index(lowerContent, "```json")
		if start != -1 {
			// Move past the opening fence and any newline
			start += len("```json")
			if start < len(trimmed) && trimmed[start] == '\n' {
				start++
			}

			// Find closing fence after the opening
			remaining := trimmed[start:]
			end := strings.Index(remaining, "```")
			if end > 0 {
				return strings.TrimSpace(remaining[:end])
			}
		}
	}

	// Fallback: look for any ``` markers at START of content
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

// invokeAndParseQCAgent invokes a QC agent and parses the response, with automatic retry on JSON validation failure.
// This method implements a robust JSON validation retry mechanism:
// 1. First invocation attempts JSON parsing
// 2. If parsing fails, retries with a schema reminder injected into the prompt
// 3. Returns parsed response on success, error on persistent failure
func (qc *QualityController) invokeAndParseQCAgent(ctx context.Context, task models.Task, agentName string) (*models.QCResponse, error) {
	// First attempt: invoke agent and parse response
	result, err := qc.Invoker.Invoke(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("QC review failed: %w", err)
	}

	// Try to parse JSON response
	qcResp, parseErr := parseQCJSON(result.Output)
	if parseErr == nil {
		return qcResp, nil // Success on first try
	}

	// JSON parsing failed - prepare retry with schema reminder
	retryTask := task
	retryTask.Prompt = task.Prompt + fmt.Sprintf(`

PREVIOUS JSON INVALID

Your previous response had invalid JSON.
Error: %s

You MUST respond with this exact JSON schema (verdict must be at root level, NOT nested):
{
  "verdict": "GREEN|RED|YELLOW",
  "feedback": "detailed review feedback",
  "issues": [{"severity": "critical|warning|info", "description": "...", "location": "..."}],
  "recommendations": ["suggestion"],
  "should_retry": false,
  "suggested_agent": "agent-name"
}`, parseErr.Error())

	// Retry invocation
	retryResult, err := qc.Invoker.Invoke(ctx, retryTask)
	if err != nil {
		return nil, fmt.Errorf("QC review retry failed: %w", err)
	}

	// Parse retry response - return error if still invalid
	return parseQCJSON(retryResult.Output)
}

// aggregateCriteriaResults combines per-criterion verdicts using unanimous consensus.
// If ANY criterion fails, returns RED. Empty CriteriaResults returns agent's original verdict for backward compatibility.
func (qc *QualityController) aggregateCriteriaResults(resp *models.QCResponse, task models.Task, expectedCriteriaCount int) string {
	if len(resp.CriteriaResults) == 0 {
		return resp.Verdict // Backward compatible
	}

	for _, cr := range resp.CriteriaResults {
		if !cr.Passed {
			return models.StatusRed // Unanimous - any fail = RED
		}
	}

	return models.StatusGreen
}

// aggregateMultiAgentCriteria combines per-criterion verdicts across multiple agents using unanimous consensus.
// A criterion passes only if ALL agents unanimously agree it passes.
// Falls back to aggregateVerdicts for tasks without success or integration criteria.
func (qc *QualityController) aggregateMultiAgentCriteria(results []*ReviewResult, task models.Task, expectedCriteriaCount int) *ReviewResult {
	// Fall back to standard aggregation if no criteria defined
	if expectedCriteriaCount == 0 {
		return qc.aggregateVerdicts(results, []string{})
	}

	// Collect votes per criterion from all agents
	criterionVotes := make(map[int][]bool)
	for i := 0; i < expectedCriteriaCount; i++ {
		criterionVotes[i] = []bool{}
	}

	// Gather votes from each agent
	for _, result := range results {
		if result == nil || len(result.CriteriaResults) == 0 {
			continue // Skip non-compliant agents (no criteria results)
		}
		for _, cr := range result.CriteriaResults {
			if votes, ok := criterionVotes[cr.Index]; ok {
				criterionVotes[cr.Index] = append(votes, cr.Passed)
			}
		}
	}

	// Check unanimous consensus per criterion and log results
	allPassed := true
	var consensusResults []string
	for idx := 0; idx < expectedCriteriaCount; idx++ {
		votes := criterionVotes[idx]
		if len(votes) == 0 {
			allPassed = false // No votes = fail (no consensus possible)
			consensusResults = append(consensusResults, fmt.Sprintf("%d:FAIL", idx))
		} else {
			// Check if all votes are true (unanimous consensus)
			criterion_passed := true
			for _, passed := range votes {
				if !passed {
					criterion_passed = false
					allPassed = false
					break
				}
			}
			if criterion_passed {
				consensusResults = append(consensusResults, fmt.Sprintf("%d:PASS", idx))
			} else {
				consensusResults = append(consensusResults, fmt.Sprintf("%d:FAIL", idx))
			}
		}
	}

	// Log consensus results if logger available
	if qc.Logger != nil && len(consensusResults) > 0 {
		// Create a synthetic CriterionResult slice for logging
		var logResults []models.CriterionResult
		for _, consensusStr := range consensusResults {
			// Parse "idx:PASS/FAIL" format
			parts := strings.Split(consensusStr, ":")
			if len(parts) == 2 {
				var idx int
				fmt.Sscanf(parts[0], "%d", &idx)
				passed := parts[1] == "PASS"
				logResults = append(logResults, models.CriterionResult{
					Index:  idx,
					Passed: passed,
				})
			}
		}
		if len(logResults) > 0 {
			qc.Logger.LogQCCriteriaResults("consensus", logResults)
		}
	}

	verdict := models.StatusGreen
	if !allPassed {
		verdict = models.StatusRed
	}

	// If all criteria pass but agents returned RED, respect their verdict
	// They likely caught issues outside explicit criteria (file paths, etc.)
	if allPassed {
		for _, result := range results {
			if result != nil && result.Flag == models.StatusRed {
				verdict = models.StatusRed
				break
			}
		}
	}

	// Preserve individual agent feedback instead of generic message
	var feedbackBuilder strings.Builder
	for _, result := range results {
		if result != nil && result.Feedback != "" {
			if result.AgentName != "" {
				feedbackBuilder.WriteString(fmt.Sprintf("[%s] %s\n", result.AgentName, result.Feedback))
			} else {
				feedbackBuilder.WriteString(fmt.Sprintf("%s\n", result.Feedback))
			}
		}
	}

	feedback := strings.TrimSpace(feedbackBuilder.String())
	if feedback == "" {
		feedback = "Multi-agent criteria consensus" // Fallback if no feedback
	}

	return &ReviewResult{
		Flag:     verdict,
		Feedback: feedback,
	}
}

// injectBehaviorContext retrieves and formats behavioral metrics for QC prompt injection
func (qc *QualityController) injectBehaviorContext(ctx context.Context, task models.Task) string {
	if qc.BehavioralMetrics == nil {
		return ""
	}

	provider := *qc.BehavioralMetrics
	if provider == nil {
		return ""
	}

	metrics := provider.GetMetrics(ctx, task.Number)
	if metrics == nil {
		return ""
	}

	return BuildBehaviorPromptSection(metrics)
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
