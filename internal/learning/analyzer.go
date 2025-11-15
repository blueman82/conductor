package learning

import (
	"sort"
)

// AgentStats holds performance statistics for an agent
type AgentStats struct {
	Agent        string
	SuccessRate  float64
	TotalRuns    int
	SuccessCount int
	FailureCount int
}

// SelectBetterAgent selects the best agent for a retry attempt based on QC suggestion
// and historical performance data. Returns the selected agent and reasoning.
//
// Priority order:
// 1. Use QC suggestion if provided and different from current agent
// 2. Analyze history for best success rate (excluding current agent)
// 3. Fallback to "general-purpose"
func SelectBetterAgent(currentAgent string, history []*TaskExecution, qcSuggestion string) (string, string) {
	// Priority 1: Use QC suggestion if valid
	if qcSuggestion != "" && qcSuggestion != currentAgent {
		return qcSuggestion, "QC suggested agent"
	}

	// Priority 2: Analyze historical performance
	if len(history) > 0 {
		agentStats := analyzeAgentPerformance(history)
		candidates := filterOutAgent(agentStats, currentAgent)
		if len(candidates) > 0 {
			return candidates[0].Agent, "historical best performer"
		}
	}

	// Priority 3: Fallback to general-purpose
	return "general-purpose", "fallback agent"
}

// analyzeAgentPerformance calculates success rates for each agent in the history
func analyzeAgentPerformance(history []*TaskExecution) map[string]*AgentStats {
	stats := make(map[string]*AgentStats)

	for _, exec := range history {
		if exec.Agent == "" {
			continue
		}

		if _, exists := stats[exec.Agent]; !exists {
			stats[exec.Agent] = &AgentStats{
				Agent: exec.Agent,
			}
		}

		stats[exec.Agent].TotalRuns++
		if exec.Success {
			stats[exec.Agent].SuccessCount++
		} else {
			stats[exec.Agent].FailureCount++
		}
	}

	// Calculate success rates
	for _, stat := range stats {
		if stat.TotalRuns > 0 {
			stat.SuccessRate = float64(stat.SuccessCount) / float64(stat.TotalRuns)
		}
	}

	return stats
}

// filterOutAgent removes the excluded agent and returns candidates sorted by success rate
func filterOutAgent(stats map[string]*AgentStats, excludeAgent string) []*AgentStats {
	var candidates []*AgentStats

	for agent, stat := range stats {
		if agent != excludeAgent {
			candidates = append(candidates, stat)
		}
	}

	// Sort by success rate (descending), then by total runs (descending), then by agent name for deterministic ordering
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].SuccessRate != candidates[j].SuccessRate {
			return candidates[i].SuccessRate > candidates[j].SuccessRate
		}
		if candidates[i].TotalRuns != candidates[j].TotalRuns {
			return candidates[i].TotalRuns > candidates[j].TotalRuns
		}
		return candidates[i].Agent < candidates[j].Agent
	})

	return candidates
}
