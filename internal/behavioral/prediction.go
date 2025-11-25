package behavioral

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// FailurePredictor predicts task failure based on historical patterns
type FailurePredictor struct {
	sessions           []Session
	metrics            []BehavioralMetrics
	patternDetector    *PatternDetector
	toolFailureRates   map[string]float64
	minHistorySessions int
	highRiskThreshold  float64
}

// PredictionResult contains failure prediction analysis
type PredictionResult struct {
	Probability     float64  `json:"probability"`     // Failure probability (0.0 to 1.0)
	Confidence      float64  `json:"confidence"`      // Confidence in prediction (0.0 to 1.0)
	RiskLevel       string   `json:"risk_level"`      // low, medium, high
	Explanation     string   `json:"explanation"`     // Human-readable explanation
	RiskFactors     []string `json:"risk_factors"`    // Specific risk factors identified
	Recommendations []string `json:"recommendations"` // Suggested mitigations
}

// ToolRisk represents risk metrics for a specific tool
type ToolRisk struct {
	ToolName    string  `json:"tool_name"`
	FailureRate float64 `json:"failure_rate"`
	UsageCount  int     `json:"usage_count"`
	RiskScore   float64 `json:"risk_score"`
}

// NewFailurePredictor creates a new failure predictor
func NewFailurePredictor(sessions []Session, metrics []BehavioralMetrics) *FailurePredictor {
	return &FailurePredictor{
		sessions:           sessions,
		metrics:            metrics,
		patternDetector:    NewPatternDetector(sessions, metrics),
		minHistorySessions: 5,
		highRiskThreshold:  0.7,
	}
}

// PredictFailure analyzes a session and predicts failure probability
func (fp *FailurePredictor) PredictFailure(session *Session, toolUsage []string) (*PredictionResult, error) {
	if session == nil {
		return nil, fmt.Errorf("session cannot be nil")
	}

	// Calculate tool failure rates if not cached
	if fp.toolFailureRates == nil {
		fp.toolFailureRates = fp.CalculateToolFailureRates()
	}

	// Find similar historical sessions
	similarSessions := fp.FindSimilarHistoricalSessions(session, toolUsage)

	// Calculate failure probability
	probability := fp.CalculateProbability(toolUsage, similarSessions)

	// Calculate confidence based on historical data size
	confidence := fp.calculateConfidence(similarSessions)

	// Determine risk level
	riskLevel := fp.getRiskLevel(probability)

	// Identify risk factors
	riskFactors := fp.identifyRiskFactors(toolUsage, similarSessions)

	// Generate explanation
	explanation := fp.generateExplanation(probability, toolUsage, similarSessions)

	// Generate recommendations
	recommendations := fp.generateRecommendations(riskFactors, toolUsage)

	return &PredictionResult{
		Probability:     probability,
		Confidence:      confidence,
		RiskLevel:       riskLevel,
		Explanation:     explanation,
		RiskFactors:     riskFactors,
		Recommendations: recommendations,
	}, nil
}

// CalculateToolFailureRates computes failure rate for each tool based on history
func (fp *FailurePredictor) CalculateToolFailureRates() map[string]float64 {
	statsMap := make(map[string]*toolStats)

	// Aggregate tool usage and failures
	for i, metric := range fp.metrics {
		if i >= len(fp.sessions) {
			break
		}

		session := fp.sessions[i]
		for _, tool := range metric.ToolExecutions {
			if _, exists := statsMap[tool.Name]; !exists {
				statsMap[tool.Name] = &toolStats{}
			}

			stats := statsMap[tool.Name]
			stats.totalUsage += tool.Count

			if !session.Success {
				stats.failureCount += tool.Count
			}
		}
	}

	// Calculate failure rates
	rates := make(map[string]float64)
	for toolName, stats := range statsMap {
		if stats.totalUsage > 0 {
			rates[toolName] = float64(stats.failureCount) / float64(stats.totalUsage)
		} else {
			rates[toolName] = 0.0
		}
	}

	return rates
}

// FindSimilarHistoricalSessions finds sessions with similar tool usage patterns
func (fp *FailurePredictor) FindSimilarHistoricalSessions(session *Session, toolUsage []string) []Session {
	if len(fp.sessions) == 0 || len(toolUsage) == 0 {
		return []Session{}
	}

	// Create tool usage set for quick lookup
	toolSet := make(map[string]bool)
	for _, tool := range toolUsage {
		toolSet[tool] = true
	}

	type scoredSession struct {
		session    Session
		similarity float64
	}

	scored := make([]scoredSession, 0)

	// Score each historical session by tool overlap
	for i, histSession := range fp.sessions {
		if i >= len(fp.metrics) {
			break
		}

		metric := fp.metrics[i]
		histTools := make(map[string]bool)
		for _, tool := range metric.ToolExecutions {
			histTools[tool.Name] = true
		}

		// Calculate Jaccard similarity
		intersection := 0
		union := len(toolSet)

		for tool := range histTools {
			if toolSet[tool] {
				intersection++
			} else {
				union++
			}
		}

		similarity := 0.0
		if union > 0 {
			similarity = float64(intersection) / float64(union)
		}

		// Only include sessions with some similarity
		if similarity > 0.2 {
			scored = append(scored, scoredSession{
				session:    histSession,
				similarity: similarity,
			})
		}
	}

	// Sort by similarity descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].similarity > scored[j].similarity
	})

	// Return top matches (up to 20)
	maxResults := 20
	if len(scored) > maxResults {
		scored = scored[:maxResults]
	}

	results := make([]Session, len(scored))
	for i, s := range scored {
		results[i] = s.session
	}

	return results
}

// CalculateProbability computes failure probability based on tool usage and history
func (fp *FailurePredictor) CalculateProbability(toolUsage []string, similarSessions []Session) float64 {
	if len(toolUsage) == 0 {
		return 0.0
	}

	// Method 1: Tool-based probability (weighted by failure rates)
	toolProbability := 0.0
	totalWeight := 0.0

	for _, tool := range toolUsage {
		if rate, exists := fp.toolFailureRates[tool]; exists {
			toolProbability += rate
			totalWeight += 1.0
		}
	}

	if totalWeight > 0 {
		toolProbability /= totalWeight
	}

	// Method 2: Historical session failure rate
	historyProbability := 0.0
	if len(similarSessions) > 0 {
		failureCount := 0
		for _, session := range similarSessions {
			if !session.Success {
				failureCount++
			}
		}
		historyProbability = float64(failureCount) / float64(len(similarSessions))
	}

	// Method 3: Tool combination risk (detect high-risk combinations)
	combinationRisk := fp.calculateCombinationRisk(toolUsage)

	// Weighted average of methods
	// More weight on history if we have enough similar sessions
	historyWeight := 0.3
	if len(similarSessions) >= fp.minHistorySessions {
		historyWeight = 0.5
	}

	probability := (toolProbability * 0.3) +
		(historyProbability * historyWeight) +
		(combinationRisk * (1.0 - 0.3 - historyWeight))

	// Clamp to [0, 1]
	return math.Max(0.0, math.Min(1.0, probability))
}

// calculateCombinationRisk detects high-risk tool combinations
func (fp *FailurePredictor) calculateCombinationRisk(toolUsage []string) float64 {
	if len(toolUsage) < 2 {
		return 0.0
	}

	// Detect known risky patterns
	riskScore := 0.0
	toolSet := make(map[string]bool)
	for _, tool := range toolUsage {
		toolSet[tool] = true
	}

	// Heavy bash usage with writes is risky
	if toolSet["Bash"] && (toolSet["Write"] || toolSet["Edit"]) {
		bashCount := 0
		writeCount := 0
		for _, tool := range toolUsage {
			if tool == "Bash" {
				bashCount++
			}
			if tool == "Write" || tool == "Edit" {
				writeCount++
			}
		}
		if bashCount > 5 && writeCount > 3 {
			riskScore += 0.3
		}
	}

	// Many tool types suggests complexity
	if len(toolSet) > 8 {
		riskScore += 0.2
	}

	// High total tool usage suggests long session
	if len(toolUsage) > 50 {
		riskScore += 0.2
	}

	return math.Min(1.0, riskScore)
}

// calculateConfidence computes confidence in prediction based on data quality
func (fp *FailurePredictor) calculateConfidence(similarSessions []Session) float64 {
	// Confidence increases with more similar historical sessions
	if len(similarSessions) == 0 {
		return 0.1 // Very low confidence without history
	}

	// Sigmoid function for confidence based on similar session count
	// Confidence approaches 0.9 as sessions increase
	x := float64(len(similarSessions))
	confidence := 0.9 / (1.0 + math.Exp(-0.3*(x-10)))

	// Minimum confidence floor
	return math.Max(0.1, math.Min(0.9, confidence))
}

// getRiskLevel converts probability to risk level category
func (fp *FailurePredictor) getRiskLevel(probability float64) string {
	if probability >= fp.highRiskThreshold {
		return "high"
	} else if probability >= 0.4 {
		return "medium"
	}
	return "low"
}

// identifyRiskFactors finds specific risk factors in the tool usage
func (fp *FailurePredictor) identifyRiskFactors(toolUsage []string, similarSessions []Session) []string {
	factors := make([]string, 0)

	// High-risk tools
	highRiskTools := fp.getHighRiskTools()
	for _, tool := range toolUsage {
		for _, riskTool := range highRiskTools {
			if tool == riskTool.ToolName && riskTool.FailureRate > 0.5 {
				factors = append(factors, fmt.Sprintf("High failure rate for %s tool (%.1f%%)",
					tool, riskTool.FailureRate*100))
				break
			}
		}
	}

	// Session complexity
	uniqueTools := make(map[string]bool)
	for _, tool := range toolUsage {
		uniqueTools[tool] = true
	}
	if len(uniqueTools) > 8 {
		factors = append(factors, fmt.Sprintf("High tool diversity (%d different tools)", len(uniqueTools)))
	}

	if len(toolUsage) > 50 {
		factors = append(factors, fmt.Sprintf("High tool usage count (%d executions)", len(toolUsage)))
	}

	// Historical pattern
	if len(similarSessions) >= fp.minHistorySessions {
		failureCount := 0
		for _, session := range similarSessions {
			if !session.Success {
				failureCount++
			}
		}
		failureRate := float64(failureCount) / float64(len(similarSessions))
		if failureRate > 0.5 {
			factors = append(factors, fmt.Sprintf("Similar sessions have %.1f%% failure rate", failureRate*100))
		}
	}

	return factors
}

// generateExplanation creates human-readable explanation of prediction
func (fp *FailurePredictor) generateExplanation(probability float64, toolUsage []string, similarSessions []Session) string {
	var parts []string

	// Overall assessment
	riskLevel := fp.getRiskLevel(probability)
	parts = append(parts, fmt.Sprintf("This task has %s risk of failure (%.1f%% probability).",
		riskLevel, probability*100))

	// Tool analysis
	if len(toolUsage) > 0 {
		avgRate := 0.0
		count := 0
		for _, tool := range toolUsage {
			if rate, exists := fp.toolFailureRates[tool]; exists {
				avgRate += rate
				count++
			}
		}
		if count > 0 {
			avgRate /= float64(count)
			parts = append(parts, fmt.Sprintf("Tools used have average %.1f%% failure rate.",
				avgRate*100))
		}
	}

	// Historical context
	if len(similarSessions) >= fp.minHistorySessions {
		failureCount := 0
		for _, session := range similarSessions {
			if !session.Success {
				failureCount++
			}
		}
		parts = append(parts, fmt.Sprintf("Found %d similar historical sessions with %d failures.",
			len(similarSessions), failureCount))
	} else if len(similarSessions) > 0 {
		parts = append(parts, fmt.Sprintf("Limited historical data (%d similar sessions found).",
			len(similarSessions)))
	} else {
		parts = append(parts, "No similar historical sessions found.")
	}

	return strings.Join(parts, " ")
}

// generateRecommendations suggests risk mitigation strategies
func (fp *FailurePredictor) generateRecommendations(riskFactors []string, toolUsage []string) []string {
	recommendations := make([]string, 0)

	// If high-risk tools detected
	highRiskTools := fp.getHighRiskTools()
	for _, tool := range toolUsage {
		for _, riskTool := range highRiskTools {
			if tool == riskTool.ToolName && riskTool.FailureRate > 0.5 {
				recommendations = append(recommendations,
					fmt.Sprintf("Review %s tool usage carefully", tool))
				break
			}
		}
	}

	// If session is complex
	uniqueTools := make(map[string]bool)
	for _, tool := range toolUsage {
		uniqueTools[tool] = true
	}
	if len(uniqueTools) > 8 {
		recommendations = append(recommendations, "Consider breaking task into smaller subtasks")
	}

	if len(toolUsage) > 50 {
		recommendations = append(recommendations, "Monitor execution closely due to high complexity")
	}

	// Generic recommendations based on risk factors
	if len(riskFactors) > 2 {
		recommendations = append(recommendations, "Enable verbose logging for debugging")
		recommendations = append(recommendations, "Plan for potential retry with different approach")
	}

	// If bash is used heavily
	bashCount := 0
	for _, tool := range toolUsage {
		if tool == "Bash" {
			bashCount++
		}
	}
	if bashCount > 10 {
		recommendations = append(recommendations, "Validate bash commands before execution")
	}

	// Default if no specific recommendations
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Proceed with normal monitoring")
	}

	return recommendations
}

// getHighRiskTools returns tools with highest failure rates
func (fp *FailurePredictor) getHighRiskTools() []ToolRisk {
	risks := make([]ToolRisk, 0)

	for toolName, failureRate := range fp.toolFailureRates {
		// Calculate usage count
		usageCount := 0
		for _, metric := range fp.metrics {
			for _, tool := range metric.ToolExecutions {
				if tool.Name == toolName {
					usageCount += tool.Count
				}
			}
		}

		// Risk score combines failure rate and usage frequency
		riskScore := failureRate * math.Log10(float64(usageCount+1))

		risks = append(risks, ToolRisk{
			ToolName:    toolName,
			FailureRate: failureRate,
			UsageCount:  usageCount,
			RiskScore:   riskScore,
		})
	}

	// Sort by failure rate descending
	sort.Slice(risks, func(i, j int) bool {
		return risks[i].FailureRate > risks[j].FailureRate
	})

	// Return top 5 highest risk
	if len(risks) > 5 {
		risks = risks[:5]
	}

	return risks
}

// toolStats tracks tool usage and failure statistics
type toolStats struct {
	totalUsage   int
	failureCount int
}
