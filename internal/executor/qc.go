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
	"github.com/harrison/conductor/internal/pattern"
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
	ReviewAgent         string                     // DEPRECATED: Agent name for single-agent QC (backward compat)
	AgentConfig         models.QCAgentConfig       // Multi-agent QC configuration (v2.2+)
	Registry            *agent.Registry            // Agent registry for auto-selection (v2.2+)
	MaxRetries          int                        // Maximum number of retry attempts for RED responses
	LearningStore       *learning.Store            // Learning store for historical context (optional)
	Logger              QCLogger                   // Logger for QC events (optional, can be nil)
	IntelligentSelector *IntelligentSelector       // Intelligent selector for mode="intelligent" (v2.4+)
	BehavioralMetrics   *BehavioralMetricsProvider // Provider for behavioral metrics context (v2.7+)

	// Test/verification results for QC prompt injection (v2.9+)
	TestCommandResults     []TestCommandResult           // Results from RunTestCommands
	CriterionVerifyResults []CriterionVerificationResult // Results from RunCriterionVerifications
	DocTargetResults       []DocTargetResult             // Results from VerifyDocumentationTargets

	// STOP Protocol integration (v2.24+)
	STOPSummary          string // Prior art summary from Pattern Intelligence (injected into prompt)
	RequireJustification bool   // Whether to require justification for custom implementations
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

	// Add formatting instructions (QC-specific JSON format)
	return agent.PrepareQCPrompt(basePrompt)
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

	if len(task.KeyPoints) > 0 {
		sb.WriteString("## KEY POINTS (GUIDANCE - NOT SCORED)\n\n")
		keyPointCount := len(task.KeyPoints)
		criteriaCount := len(allCriteria)
		if criteriaCount < keyPointCount {
			sb.WriteString(fmt.Sprintf("⚠️ %d key point(s) do not have explicit success criteria above. Treat these as context while scoring only against listed criteria.\n\n", keyPointCount-criteriaCount))
		} else {
			sb.WriteString("These implementation key points are provided for context. Score strictly against the success criteria above.\n\n")
		}
		for _, kp := range task.KeyPoints {
			line := kp.Point
			if kp.Details != "" {
				line += fmt.Sprintf(": %s", kp.Details)
			}
			if kp.Reference != "" {
				line += fmt.Sprintf(" (ref: %s)", kp.Reference)
			}
			sb.WriteString(fmt.Sprintf("- %s\n", line))
		}
		sb.WriteString("\n")
	}

	if len(task.TestCommands) > 0 {
		sb.WriteString("## TEST COMMANDS\n")
		for _, cmd := range task.TestCommands {
			sb.WriteString(fmt.Sprintf("- `%s`\n", cmd))
		}
		sb.WriteString("\n")
	}

	// Add expected file paths for verification - CRITICAL for agent compliance
	if len(task.Files) > 0 {
		sb.WriteString("## EXPECTED FILE PATHS - CRITICAL VERIFICATION\n")
		sb.WriteString("⚠️ **MANDATORY**: Agent MUST create/modify these EXACT file paths:\n")
		for _, file := range task.Files {
			sb.WriteString(fmt.Sprintf("- [ ] `%s`\n", file))
		}
		sb.WriteString("\n**Verdict Rules:**\n")
		sb.WriteString("- If agent created files with DIFFERENT paths/names → **RED** (wrong files)\n")
		sb.WriteString("- If expected files are missing → **RED** (incomplete)\n")
		sb.WriteString("- Only GREEN if ALL expected files exist at EXACT paths\n\n")
	}

	// Add domain-specific review criteria based on file extensions
	if domainCriteria := inferDomainChecks(task.Files); domainCriteria != "" {
		sb.WriteString(domainCriteria)
		sb.WriteString("\n")
	}

	sb.WriteString("## AGENT OUTPUT\n```\n")
	sb.WriteString(output)
	sb.WriteString("\n```\n\n")

	// Inject test command results (v2.9+)
	if len(qc.TestCommandResults) > 0 {
		sb.WriteString(FormatTestResults(qc.TestCommandResults))
		sb.WriteString("\n")
	}

	// Inject criterion verification results (v2.9+)
	if len(qc.CriterionVerifyResults) > 0 {
		sb.WriteString(FormatCriterionResults(qc.CriterionVerifyResults))
		sb.WriteString("\n")
	}

	// Inject error classification context (v2.12+)
	if detectedErrors := getDetectedErrors(&task); len(detectedErrors) > 0 {
		sb.WriteString(FormatDetectedErrors(detectedErrors))
		sb.WriteString("\n")
	}

	// Inject documentation target verification results (v2.9+)
	if len(qc.DocTargetResults) > 0 {
		sb.WriteString(FormatDocTargetResults(qc.DocTargetResults))
		sb.WriteString("\n")
	}

	// Inject STOP protocol prior art context (v2.24+)
	if qc.STOPSummary != "" {
		sb.WriteString(FormatSTOPPriorArt(qc.STOPSummary, qc.RequireJustification))
		sb.WriteString("\n")
	}

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

	// Add formatting instructions (QC-specific JSON format)
	return agent.PrepareQCPrompt(sb.String())
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

	// Determine if STOP justification is required (prior art exists and config enables it)
	requireSTOPJustification := qc.RequireJustification && qc.STOPSummary != ""

	// Create a review task for the invoker with JSON schema enforcement
	reviewTask := models.Task{
		Number:     task.Number,
		Name:       fmt.Sprintf("QC Review: %s", task.Name),
		Prompt:     basePrompt,
		Agent:      qc.ReviewAgent,
		JSONSchema: models.QCResponseSchemaWithOptions(hasSuccessCriteria, requireSTOPJustification), // Enforce QC response structure via schema
	}

	// Invoke the QC agent and parse response
	qcResp, jsonErr := qc.invokeAndParseQCAgent(ctx, reviewTask, qc.ReviewAgent)
	if jsonErr != nil {
		// Invocation or parsing failed
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

	// Apply STOP justification check if required (v2.24+)
	// Weak/missing justification when prior art exists → YELLOW
	if requireSTOPJustification && reviewResult.Flag == models.StatusGreen {
		if !EvaluateSTOPJustification(qc.STOPSummary, qcResp.STOPJustification) {
			reviewResult.Flag = models.StatusYellow
			reviewResult.Feedback += " (Warning: Prior art exists but justification for custom implementation is weak or missing)"
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

	// Determine if STOP justification is required (prior art exists and config enables it)
	requireSTOPJustification := qc.RequireJustification && qc.STOPSummary != ""

	reviewTask := models.Task{
		Number:     task.Number,
		Name:       fmt.Sprintf("QC Review: %s", task.Name),
		Prompt:     basePrompt,
		Agent:      agentName,
		JSONSchema: models.QCResponseSchemaWithOptions(hasSuccessCriteria, requireSTOPJustification), // Enforce QC response structure via schema
	}

	qcResp, jsonErr := qc.invokeAndParseQCAgent(ctx, reviewTask, agentName)
	if jsonErr != nil {
		// Invocation or parsing failed - return error
		return nil, fmt.Errorf("QC review failed: %w", jsonErr)
	}

	reviewResult := &ReviewResult{
		Flag:            qcResp.Verdict,
		Feedback:        qcResp.Feedback,
		SuggestedAgent:  qcResp.SuggestedAgent,
		AgentName:       agentName,
		CriteriaResults: qcResp.CriteriaResults,
	}

	// Apply STOP justification check if required (v2.24+)
	if requireSTOPJustification && reviewResult.Flag == models.StatusGreen {
		if !EvaluateSTOPJustification(qc.STOPSummary, qcResp.STOPJustification) {
			reviewResult.Flag = models.StatusYellow
			reviewResult.Feedback += " (Warning: Prior art exists but justification for custom implementation is weak or missing)"
		}
	}

	return reviewResult, nil
}

// invokeAgentsParallel invokes multiple QC agents concurrently
func (qc *QualityController) invokeAgentsParallel(ctx context.Context, task models.Task, output string, agents []string) []*ReviewResult {
	var wg sync.WaitGroup
	results := make([]*ReviewResult, len(agents))
	var mu sync.Mutex

	// Get combined criteria once (same for all agents)
	allCriteria := getCombinedCriteria(task)
	hasSuccessCriteria := len(allCriteria) > 0

	// Determine if STOP justification is required (prior art exists and config enables it)
	requireSTOPJustification := qc.RequireJustification && qc.STOPSummary != ""

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

			// Create review task for this agent with JSON schema enforcement
			reviewTask := models.Task{
				Number:     task.Number,
				Name:       fmt.Sprintf("QC Review: %s", task.Name),
				Prompt:     basePrompt,
				Agent:      agent,
				JSONSchema: models.QCResponseSchemaWithOptions(hasSuccessCriteria, requireSTOPJustification), // Enforce QC response structure via schema
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

// parseQCJSON parses JSON response from QC agent with schema enforcement
// With --json-schema flag, responses are guaranteed to match QCResponse structure
func parseQCJSON(output string) (*models.QCResponse, error) {
	// Extract content from Claude CLI envelope
	claudeOut, _ := agent.ParseClaudeOutput(output)
	actualOutput := claudeOut.Content

	// Fallback to raw output if ParseClaudeOutput returns empty content
	if actualOutput == "" {
		actualOutput = output
	}

	var resp models.QCResponse

	// Extract JSON portion - find opening and closing braces
	// This handles agents outputting prose before JSON, or wrapping in markdown code blocks
	jsonStart := strings.Index(actualOutput, "{")
	jsonEnd := strings.LastIndex(actualOutput, "}")
	if jsonStart < 0 || jsonEnd < 0 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("failed to parse QC response as JSON: no complete JSON object found in output")
	}
	jsonStr := actualOutput[jsonStart : jsonEnd+1]

	// Parse JSON
	err := json.Unmarshal([]byte(jsonStr), &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse QC response as JSON: %w", err)
	}

	// Validate parsed JSON structure
	if err := resp.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed despite schema: %w", err)
	}

	return &resp, nil
}

// invokeAndParseQCAgent invokes a QC agent and parses the response with schema enforcement.
// With --json-schema flag, invalid JSON is prevented at the CLI level, eliminating need for retries.
func (qc *QualityController) invokeAndParseQCAgent(ctx context.Context, task models.Task, agentName string) (*models.QCResponse, error) {
	result, err := qc.Invoker.Invoke(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("QC review failed: %w", err)
	}

	return parseQCJSON(result.Output)
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

// FormatDetectedErrors formats error classifications for QC agent context
func FormatDetectedErrors(errors []*DetectedError) string {
	if len(errors) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n## Error Classification Analysis\n\n")
	sb.WriteString("Claude has analyzed test failures and classified them:\n\n")

	for i, err := range errors {
		sb.WriteString(fmt.Sprintf("### Error %d: %s\n", i+1, err.Pattern.Category))
		sb.WriteString(fmt.Sprintf("- **Method**: %s", err.Method))
		if err.Method == "claude" {
			sb.WriteString(fmt.Sprintf(" (confidence: %.0f%%)", err.Confidence*100))
		}
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("- **Agent Can Fix**: %v\n", err.Pattern.AgentCanFix))
		sb.WriteString(fmt.Sprintf("- **Requires Human**: %v\n", err.Pattern.RequiresHumanIntervention))
		sb.WriteString(fmt.Sprintf("- **Suggestion**: %s\n\n", err.Pattern.Suggestion))
	}

	sb.WriteString("**Use this context when reviewing code quality and deciding retry strategy.**\n\n")
	return sb.String()
}

// FormatSTOPPriorArt formats STOP protocol prior art findings for QC prompt injection.
// When requireJustification is true, the prompt instructs the QC agent to request
// justification from the implementing agent for why custom implementation was needed.
func FormatSTOPPriorArt(stopSummary string, requireJustification bool) string {
	if stopSummary == "" {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n## STOP Protocol: Prior Art Analysis\n\n")
	sb.WriteString("Pattern Intelligence discovered existing solutions related to this task:\n\n")
	sb.WriteString(stopSummary)
	sb.WriteString("\n")

	if requireJustification {
		sb.WriteString("\n### JUSTIFICATION REQUIRED\n")
		sb.WriteString("⚠️ **Prior art exists.** The implementing agent MUST justify why a custom implementation was needed.\n")
		sb.WriteString("Evaluate the `stop_justification` field in the agent's response:\n")
		sb.WriteString("- **Strong justification**: Prior solutions are insufficient due to specific technical reasons → Proceed normally\n")
		sb.WriteString("- **Weak/missing justification**: Agent did not explain why existing solutions don't work → **YELLOW** verdict\n")
		sb.WriteString("- **No justification when prior art exists**: Consider this a quality concern that should be flagged\n\n")
	} else {
		sb.WriteString("\n**Context only**: Prior art is provided for informational purposes. No justification required.\n\n")
	}

	return sb.String()
}

// BuildSTOPSummaryFromSTOPResult extracts the STOP summary from a pattern.STOPResult.
// This is called from task.go after Pattern Intelligence check to wire STOP context to QC.
// The summary is injected into the QC prompt to request justification for custom implementations.
func BuildSTOPSummaryFromSTOPResult(stopResult *pattern.STOPResult) string {
	if stopResult == nil {
		return ""
	}

	var sb strings.Builder
	hasPriorArt := false

	// Check for similar patterns
	if len(stopResult.Search.SimilarPatterns) > 0 {
		hasPriorArt = true
		sb.WriteString("**Similar Patterns Found:**\n")
		for i, p := range stopResult.Search.SimilarPatterns {
			if i >= 3 { // Limit to top 3
				break
			}
			sb.WriteString(fmt.Sprintf("- %s (%s): %.0f%% similar - %s\n",
				p.Name, p.FilePath, p.Similarity*100, p.Description))
		}
		sb.WriteString("\n")
	}

	// Check for existing implementations
	if len(stopResult.Search.ExistingImplementations) > 0 {
		hasPriorArt = true
		sb.WriteString("**Existing Implementations:**\n")
		for i, impl := range stopResult.Search.ExistingImplementations {
			if i >= 3 { // Limit to top 3
				break
			}
			sb.WriteString(fmt.Sprintf("- %s (%s) in %s: %.0f%% relevant\n",
				impl.Name, impl.Type, impl.FilePath, impl.Relevance*100))
		}
		sb.WriteString("\n")
	}

	// Check for related files
	if len(stopResult.Search.RelatedFiles) > 0 {
		hasPriorArt = true
		sb.WriteString("**Related Files:** ")
		maxFiles := 5
		if maxFiles > len(stopResult.Search.RelatedFiles) {
			maxFiles = len(stopResult.Search.RelatedFiles)
		}
		sb.WriteString(strings.Join(stopResult.Search.RelatedFiles[:maxFiles], ", "))
		sb.WriteString("\n\n")
	}

	// Add recommendations if any
	if len(stopResult.Recommendations) > 0 {
		sb.WriteString("**Recommendations:**\n")
		for _, rec := range stopResult.Recommendations {
			sb.WriteString(fmt.Sprintf("- %s\n", rec))
		}
	}

	if !hasPriorArt {
		return "" // No prior art found
	}

	return sb.String()
}

// EvaluateSTOPJustification assesses the quality of STOP justification when prior art exists.
// Returns true if justification is sufficient, false if weak/missing and verdict should be YELLOW.
// Empty stopSummary (no prior art) always returns true (no justification needed).
func EvaluateSTOPJustification(stopSummary, justification string) bool {
	// No prior art found - no justification needed
	if stopSummary == "" {
		return true
	}

	// Prior art exists but no justification provided
	if justification == "" {
		return false
	}

	// Check for weak justifications (common non-answers)
	weakPatterns := []string{
		"n/a",
		"not applicable",
		"none",
		"no justification needed",
		"custom implementation",
		"new implementation",
	}

	lowerJustification := strings.ToLower(strings.TrimSpace(justification))

	// Reject very short justifications (likely inadequate)
	if len(lowerJustification) < 20 {
		return false
	}

	// Check for weak/dismissive patterns
	for _, pattern := range weakPatterns {
		if lowerJustification == pattern {
			return false
		}
	}

	// Justification exists and isn't a known weak pattern
	return true
}
