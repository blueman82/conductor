package behavioral

import (
	"math"
	"sort"
)

// PerformanceScorer scores agents based on multiple performance dimensions
type PerformanceScorer struct {
	sessions []Session
	metrics  []BehavioralMetrics
	weights  ScoreWeights
}

// ScoreWeights defines how to weight different scoring dimensions
type ScoreWeights struct {
	Success      float64 // Weight for success rate
	CostEff      float64 // Weight for cost efficiency
	Speed        float64 // Weight for speed
	ErrorRecov   float64 // Weight for error recovery
}

// AgentScore represents multi-dimensional performance score for an agent
type AgentScore struct {
	AgentName        string  `json:"agent_name"`
	SuccessScore     float64 `json:"success_score"`      // 0-1 scale
	CostEffScore     float64 `json:"cost_eff_score"`     // 0-1 scale
	SpeedScore       float64 `json:"speed_score"`        // 0-1 scale
	ErrorRecovScore  float64 `json:"error_recov_score"`  // 0-1 scale
	CompositeScore   float64 `json:"composite_score"`    // Weighted combination
	SampleSize       int     `json:"sample_size"`        // Number of sessions
	Domain           string  `json:"domain"`             // Agent domain (e.g., "backend", "frontend")
}

// RankedAgent represents an agent with its rank
type RankedAgent struct {
	AgentName string  `json:"agent_name"`
	Score     float64 `json:"score"`
	Rank      int     `json:"rank"`
}

// agentStats holds aggregate statistics for an agent
type agentStats struct {
	totalSessions   int
	successCount    int
	totalCost       float64
	totalDuration   int64
	recoveryCount   int  // Sessions that recovered from errors
	errorCount      int
}

// NewPerformanceScorer creates a new performance scorer with default weights
func NewPerformanceScorer(sessions []Session, metrics []BehavioralMetrics) *PerformanceScorer {
	return &PerformanceScorer{
		sessions: sessions,
		metrics:  metrics,
		weights: ScoreWeights{
			Success:    0.40, // Success rate most important
			CostEff:    0.25,
			Speed:      0.20,
			ErrorRecov: 0.15,
		},
	}
}

// SetWeights allows custom weighting of score dimensions
func (ps *PerformanceScorer) SetWeights(weights ScoreWeights) {
	// Normalize weights to sum to 1.0
	total := weights.Success + weights.CostEff + weights.Speed + weights.ErrorRecov
	if total > 0 {
		ps.weights.Success = weights.Success / total
		ps.weights.CostEff = weights.CostEff / total
		ps.weights.Speed = weights.Speed / total
		ps.weights.ErrorRecov = weights.ErrorRecov / total
	}
}

// ScoreAgent calculates multi-dimensional score for a specific agent
func (ps *PerformanceScorer) ScoreAgent(agentName string) *AgentScore {
	stats := ps.collectAgentStats(agentName)

	if stats.totalSessions == 0 {
		return &AgentScore{
			AgentName:  agentName,
			SampleSize: 0,
		}
	}

	score := &AgentScore{
		AgentName:       agentName,
		SuccessScore:    ps.calculateSuccessScore(stats),
		CostEffScore:    ps.calculateCostEfficiencyScore(stats),
		SpeedScore:      ps.calculateSpeedScore(stats),
		ErrorRecovScore: ps.calculateErrorRecoveryScore(stats),
		SampleSize:      stats.totalSessions,
		Domain:          ps.inferDomain(agentName),
	}

	// Calculate composite score with weights
	score.CompositeScore = (score.SuccessScore * ps.weights.Success) +
		(score.CostEffScore * ps.weights.CostEff) +
		(score.SpeedScore * ps.weights.Speed) +
		(score.ErrorRecovScore * ps.weights.ErrorRecov)

	// Adjust for sample size confidence
	score.CompositeScore = ps.adjustForSampleSize(score.CompositeScore, stats.totalSessions)

	return score
}

// CalculateSuccessScore returns success rate normalized to 0-1
func (ps *PerformanceScorer) calculateSuccessScore(stats *agentStats) float64 {
	if stats.totalSessions == 0 {
		return 0.0
	}
	return float64(stats.successCount) / float64(stats.totalSessions)
}

// CalculateCostEfficiencyScore scores cost efficiency (lower cost = higher score)
func (ps *PerformanceScorer) calculateCostEfficiencyScore(stats *agentStats) float64 {
	if stats.totalSessions == 0 {
		return 0.0
	}

	avgCost := stats.totalCost / float64(stats.totalSessions)

	// Get global average cost for normalization
	globalAvgCost := ps.calculateGlobalAverageCost()
	if globalAvgCost == 0 {
		return 0.5 // Neutral score if no baseline
	}

	// Score: 1.0 if cost is 50% below average, 0.0 if 2x above average
	// Formula: 1 - (agentCost / (2 * globalAvg))
	ratio := avgCost / globalAvgCost
	score := 1.0 - math.Min(ratio/2.0, 1.0)

	return math.Max(0.0, math.Min(1.0, score))
}

// CalculateSpeedScore scores execution speed (faster = higher score)
func (ps *PerformanceScorer) calculateSpeedScore(stats *agentStats) float64 {
	if stats.totalSessions == 0 {
		return 0.0
	}

	avgDuration := float64(stats.totalDuration) / float64(stats.totalSessions)

	// Get global average duration for normalization
	globalAvgDuration := ps.calculateGlobalAverageDuration()
	if globalAvgDuration == 0 {
		return 0.5 // Neutral score if no baseline
	}

	// Score: 1.0 if 50% faster than average, 0.0 if 2x slower
	ratio := avgDuration / globalAvgDuration
	score := 1.0 - math.Min(ratio/2.0, 1.0)

	return math.Max(0.0, math.Min(1.0, score))
}

// CalculateErrorRecoveryScore scores ability to recover from errors
func (ps *PerformanceScorer) calculateErrorRecoveryScore(stats *agentStats) float64 {
	if stats.errorCount == 0 {
		return 1.0 // Perfect score if no errors
	}

	// Recovery rate: sessions that had errors but still succeeded
	recoveryRate := float64(stats.recoveryCount) / float64(stats.errorCount)

	return math.Max(0.0, math.Min(1.0, recoveryRate))
}

// RankAgents ranks all agents by composite score
func (ps *PerformanceScorer) RankAgents() []RankedAgent {
	agentNames := ps.getUniqueAgents()
	scored := make([]RankedAgent, 0, len(agentNames))

	for _, name := range agentNames {
		score := ps.ScoreAgent(name)
		if score.SampleSize > 0 {
			scored = append(scored, RankedAgent{
				AgentName: name,
				Score:     score.CompositeScore,
			})
		}
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Assign ranks
	for i := range scored {
		scored[i].Rank = i + 1
	}

	return scored
}

// CompareWithinDomain compares agents within the same domain
func (ps *PerformanceScorer) CompareWithinDomain() map[string][]RankedAgent {
	allAgents := ps.getUniqueAgents()
	domains := make(map[string][]RankedAgent)

	// Group agents by domain
	for _, agentName := range allAgents {
		domain := ps.inferDomain(agentName)
		score := ps.ScoreAgent(agentName)

		if score.SampleSize > 0 {
			domains[domain] = append(domains[domain], RankedAgent{
				AgentName: agentName,
				Score:     score.CompositeScore,
			})
		}
	}

	// Sort each domain's agents
	for domain := range domains {
		agents := domains[domain]
		sort.Slice(agents, func(i, j int) bool {
			return agents[i].Score > agents[j].Score
		})

		// Assign ranks within domain
		for i := range agents {
			agents[i].Rank = i + 1
		}

		domains[domain] = agents
	}

	return domains
}

// Helper functions

func (ps *PerformanceScorer) collectAgentStats(agentName string) *agentStats {
	stats := &agentStats{}

	for _, session := range ps.sessions {
		if session.AgentName != agentName {
			continue
		}

		stats.totalSessions++
		if session.Success {
			stats.successCount++
		}
		stats.totalDuration += session.Duration

		if session.ErrorCount > 0 {
			stats.errorCount++
			if session.Success {
				stats.recoveryCount++ // Had errors but still succeeded
			}
		}
	}

	// Add cost from metrics
	for _, metric := range ps.metrics {
		if perf, exists := metric.AgentPerformance[agentName]; exists && perf > 0 {
			stats.totalCost += metric.CalculateTotalCost()
		}
	}

	return stats
}

func (ps *PerformanceScorer) calculateGlobalAverageCost() float64 {
	totalCost := 0.0
	totalSessions := 0

	for _, metric := range ps.metrics {
		totalCost += metric.CalculateTotalCost()
		totalSessions += metric.TotalSessions
	}

	if totalSessions == 0 {
		return 0.0
	}
	return totalCost / float64(totalSessions)
}

func (ps *PerformanceScorer) calculateGlobalAverageDuration() float64 {
	var totalDuration int64
	totalSessions := 0

	for _, session := range ps.sessions {
		totalDuration += session.Duration
		totalSessions++
	}

	if totalSessions == 0 {
		return 0.0
	}
	return float64(totalDuration) / float64(totalSessions)
}

func (ps *PerformanceScorer) getUniqueAgents() []string {
	agentSet := make(map[string]bool)

	for _, session := range ps.sessions {
		if session.AgentName != "" {
			agentSet[session.AgentName] = true
		}
	}

	agents := make([]string, 0, len(agentSet))
	for agent := range agentSet {
		agents = append(agents, agent)
	}

	sort.Strings(agents) // Consistent ordering
	return agents
}

func (ps *PerformanceScorer) inferDomain(agentName string) string {
	// Simple domain inference based on agent name
	// In production, this could be more sophisticated
	switch {
	case contains(agentName, "backend") || contains(agentName, "api") || contains(agentName, "database"):
		return "backend"
	case contains(agentName, "frontend") || contains(agentName, "ui") || contains(agentName, "react"):
		return "frontend"
	case contains(agentName, "devops") || contains(agentName, "deploy") || contains(agentName, "infra"):
		return "devops"
	case contains(agentName, "test") || contains(agentName, "qa"):
		return "testing"
	case contains(agentName, "security") || contains(agentName, "audit"):
		return "security"
	default:
		return "general"
	}
}

func (ps *PerformanceScorer) adjustForSampleSize(score float64, sampleSize int) float64 {
	// Reduce confidence for small sample sizes
	// Full confidence at 10+ samples, reduced below
	if sampleSize >= 10 {
		return score
	}

	confidence := float64(sampleSize) / 10.0
	// Regress towards mean (0.5) for small samples
	return score*confidence + 0.5*(1.0-confidence)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
		 len(s) > len(substr) &&
		 (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr))
}
