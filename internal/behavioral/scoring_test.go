package behavioral

import (
	"testing"
	"time"
)

func TestNewPerformanceScorer(t *testing.T) {
	sessions := []Session{
		{ID: "1", AgentName: "agent-a", Success: true, Duration: 1000},
		{ID: "2", AgentName: "agent-b", Success: false, Duration: 2000},
	}
	metrics := []BehavioralMetrics{
		{TotalSessions: 2},
	}

	scorer := NewPerformanceScorer(sessions, metrics)

	if scorer == nil {
		t.Fatal("Expected non-nil scorer")
	}
	if len(scorer.sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(scorer.sessions))
	}

	// Verify default weights sum to 1.0
	total := scorer.weights.Success + scorer.weights.CostEff + scorer.weights.Speed + scorer.weights.ErrorRecov
	if total < 0.99 || total > 1.01 {
		t.Errorf("Expected weights to sum to 1.0, got %f", total)
	}
}

func TestSetWeights(t *testing.T) {
	scorer := NewPerformanceScorer([]Session{}, []BehavioralMetrics{})

	customWeights := ScoreWeights{
		Success:    0.5,
		CostEff:    0.3,
		Speed:      0.1,
		ErrorRecov: 0.1,
	}

	scorer.SetWeights(customWeights)

	// Verify weights are normalized
	total := scorer.weights.Success + scorer.weights.CostEff + scorer.weights.Speed + scorer.weights.ErrorRecov
	if total < 0.99 || total > 1.01 {
		t.Errorf("Expected weights to sum to 1.0, got %f", total)
	}

	if scorer.weights.Success != 0.5 {
		t.Errorf("Expected Success weight 0.5, got %f", scorer.weights.Success)
	}
}

func TestScoreAgent(t *testing.T) {
	tests := []struct {
		name              string
		sessions          []Session
		metrics           []BehavioralMetrics
		agentName         string
		expectedSampleSize int
		minCompositeScore float64
		maxCompositeScore float64
	}{
		{
			name: "perfect agent",
			sessions: []Session{
				{ID: "1", AgentName: "perfect-agent", Success: true, Duration: 1000, ErrorCount: 0},
				{ID: "2", AgentName: "perfect-agent", Success: true, Duration: 1000, ErrorCount: 0},
				{ID: "3", AgentName: "perfect-agent", Success: true, Duration: 1000, ErrorCount: 0},
			},
			metrics: []BehavioralMetrics{
				{
					TotalSessions: 3,
					AgentPerformance: map[string]int{"perfect-agent": 3},
					TokenUsage: TokenUsage{CostUSD: 0.01},
				},
			},
			agentName:          "perfect-agent",
			expectedSampleSize: 3,
			minCompositeScore:  0.5,
			maxCompositeScore:  1.0,
		},
		{
			name: "failing agent",
			sessions: []Session{
				{ID: "1", AgentName: "failing-agent", Success: false, Duration: 5000, ErrorCount: 10},
				{ID: "2", AgentName: "failing-agent", Success: false, Duration: 5000, ErrorCount: 10},
			},
			metrics: []BehavioralMetrics{
				{
					TotalSessions: 2,
					AgentPerformance: map[string]int{"failing-agent": 0},
					TokenUsage: TokenUsage{CostUSD: 1.0},
				},
			},
			agentName:          "failing-agent",
			expectedSampleSize: 2,
			minCompositeScore:  0.0,
			maxCompositeScore:  0.5,
		},
		{
			name: "agent with recovery",
			sessions: []Session{
				{ID: "1", AgentName: "recovery-agent", Success: true, Duration: 2000, ErrorCount: 2},
				{ID: "2", AgentName: "recovery-agent", Success: true, Duration: 2000, ErrorCount: 1},
				{ID: "3", AgentName: "recovery-agent", Success: false, Duration: 2000, ErrorCount: 5},
			},
			metrics: []BehavioralMetrics{
				{
					TotalSessions: 3,
					AgentPerformance: map[string]int{"recovery-agent": 2},
					TokenUsage: TokenUsage{CostUSD: 0.05},
				},
			},
			agentName:          "recovery-agent",
			expectedSampleSize: 3,
			minCompositeScore:  0.3,
			maxCompositeScore:  0.9,
		},
		{
			name:               "non-existent agent",
			sessions:           []Session{},
			metrics:            []BehavioralMetrics{},
			agentName:          "ghost-agent",
			expectedSampleSize: 0,
			minCompositeScore:  0.0,
			maxCompositeScore:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := NewPerformanceScorer(tt.sessions, tt.metrics)
			score := scorer.ScoreAgent(tt.agentName)

			if score.AgentName != tt.agentName {
				t.Errorf("Expected agent name %s, got %s", tt.agentName, score.AgentName)
			}

			if score.SampleSize != tt.expectedSampleSize {
				t.Errorf("Expected sample size %d, got %d", tt.expectedSampleSize, score.SampleSize)
			}

			if score.SampleSize > 0 {
				// Verify all scores are in [0, 1] range
				if score.SuccessScore < 0 || score.SuccessScore > 1 {
					t.Errorf("SuccessScore out of range: %f", score.SuccessScore)
				}
				if score.CostEffScore < 0 || score.CostEffScore > 1 {
					t.Errorf("CostEffScore out of range: %f", score.CostEffScore)
				}
				if score.SpeedScore < 0 || score.SpeedScore > 1 {
					t.Errorf("SpeedScore out of range: %f", score.SpeedScore)
				}
				if score.ErrorRecovScore < 0 || score.ErrorRecovScore > 1 {
					t.Errorf("ErrorRecovScore out of range: %f", score.ErrorRecovScore)
				}
				if score.CompositeScore < 0 || score.CompositeScore > 1 {
					t.Errorf("CompositeScore out of range: %f", score.CompositeScore)
				}

				// Verify composite score is within expected range
				if score.CompositeScore < tt.minCompositeScore || score.CompositeScore > tt.maxCompositeScore {
					t.Errorf("CompositeScore %f not in expected range [%f, %f]",
						score.CompositeScore, tt.minCompositeScore, tt.maxCompositeScore)
				}
			}
		})
	}
}

func TestCalculateSuccessScore(t *testing.T) {
	scorer := NewPerformanceScorer([]Session{}, []BehavioralMetrics{})

	tests := []struct {
		name     string
		stats    *agentStats
		expected float64
	}{
		{
			name: "100% success",
			stats: &agentStats{
				totalSessions: 10,
				successCount:  10,
			},
			expected: 1.0,
		},
		{
			name: "50% success",
			stats: &agentStats{
				totalSessions: 10,
				successCount:  5,
			},
			expected: 0.5,
		},
		{
			name: "0% success",
			stats: &agentStats{
				totalSessions: 10,
				successCount:  0,
			},
			expected: 0.0,
		},
		{
			name:     "no sessions",
			stats:    &agentStats{},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scorer.calculateSuccessScore(tt.stats)
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestRankAgents(t *testing.T) {
	sessions := []Session{
		{ID: "1", AgentName: "agent-a", Success: true, Duration: 1000, ErrorCount: 0},
		{ID: "2", AgentName: "agent-a", Success: true, Duration: 1000, ErrorCount: 0},
		{ID: "3", AgentName: "agent-b", Success: true, Duration: 2000, ErrorCount: 1},
		{ID: "4", AgentName: "agent-c", Success: false, Duration: 5000, ErrorCount: 10},
	}

	metrics := []BehavioralMetrics{
		{
			TotalSessions: 4,
			AgentPerformance: map[string]int{
				"agent-a": 2,
				"agent-b": 1,
				"agent-c": 0,
			},
			TokenUsage: TokenUsage{CostUSD: 0.04},
		},
	}

	scorer := NewPerformanceScorer(sessions, metrics)
	ranked := scorer.RankAgents()

	if len(ranked) != 3 {
		t.Errorf("Expected 3 ranked agents, got %d", len(ranked))
	}

	// Verify ranking is in descending order
	for i := 1; i < len(ranked); i++ {
		if ranked[i].Score > ranked[i-1].Score {
			t.Error("Agents not ranked in descending order by score")
		}
		if ranked[i].Rank != i+1 {
			t.Errorf("Agent at index %d has rank %d, expected %d", i, ranked[i].Rank, i+1)
		}
	}

	// First ranked agent should be agent-a (best performance)
	if ranked[0].AgentName != "agent-a" {
		t.Errorf("Expected agent-a to be ranked first, got %s", ranked[0].AgentName)
	}
}

func TestCompareWithinDomain(t *testing.T) {
	sessions := []Session{
		{ID: "1", AgentName: "backend-api", Success: true, Duration: 1000, ErrorCount: 0},
		{ID: "2", AgentName: "backend-database", Success: true, Duration: 2000, ErrorCount: 0},
		{ID: "3", AgentName: "frontend-react", Success: true, Duration: 1500, ErrorCount: 1},
		{ID: "4", AgentName: "frontend-ui", Success: false, Duration: 3000, ErrorCount: 5},
	}

	metrics := []BehavioralMetrics{
		{
			TotalSessions: 4,
			AgentPerformance: map[string]int{
				"backend-api":      1,
				"backend-database": 1,
				"frontend-react":   1,
				"frontend-ui":      0,
			},
			TokenUsage: TokenUsage{CostUSD: 0.04},
		},
	}

	scorer := NewPerformanceScorer(sessions, metrics)
	domains := scorer.CompareWithinDomain()

	// Verify we have backend and frontend domains
	if _, exists := domains["backend"]; !exists {
		t.Error("Expected backend domain")
	}
	if _, exists := domains["frontend"]; !exists {
		t.Error("Expected frontend domain")
	}

	// Verify each domain has correct agents
	backendAgents := domains["backend"]
	if len(backendAgents) != 2 {
		t.Errorf("Expected 2 backend agents, got %d", len(backendAgents))
	}

	frontendAgents := domains["frontend"]
	if len(frontendAgents) != 2 {
		t.Errorf("Expected 2 frontend agents, got %d", len(frontendAgents))
	}

	// Verify ranking within each domain
	for domain, agents := range domains {
		for i := 1; i < len(agents); i++ {
			if agents[i].Score > agents[i-1].Score {
				t.Errorf("Domain %s: agents not ranked in descending order", domain)
			}
			if agents[i].Rank != i+1 {
				t.Errorf("Domain %s: agent at index %d has rank %d, expected %d",
					domain, i, agents[i].Rank, i+1)
			}
		}
	}
}

func TestInferDomain(t *testing.T) {
	scorer := NewPerformanceScorer([]Session{}, []BehavioralMetrics{})

	tests := []struct {
		name     string
		agent    string
		expected string
	}{
		{
			name:     "backend agent",
			agent:    "backend-api",
			expected: "backend",
		},
		{
			name:     "frontend agent",
			agent:    "frontend-react",
			expected: "frontend",
		},
		{
			name:     "devops agent",
			agent:    "devops-deploy",
			expected: "devops",
		},
		{
			name:     "test agent",
			agent:    "test-automation",
			expected: "testing",
		},
		{
			name:     "security agent",
			agent:    "security-audit",
			expected: "security",
		},
		{
			name:     "general agent",
			agent:    "general-purpose",
			expected: "general",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scorer.inferDomain(tt.agent)
			if result != tt.expected {
				t.Errorf("Expected domain %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestAdjustForSampleSize(t *testing.T) {
	scorer := NewPerformanceScorer([]Session{}, []BehavioralMetrics{})

	tests := []struct {
		name       string
		score      float64
		sampleSize int
		minScore   float64
		maxScore   float64
	}{
		{
			name:       "large sample - no adjustment",
			score:      0.8,
			sampleSize: 20,
			minScore:   0.79,
			maxScore:   0.81,
		},
		{
			name:       "small sample - regress to mean",
			score:      1.0,
			sampleSize: 1,
			minScore:   0.5,
			maxScore:   0.7,
		},
		{
			name:       "medium sample - partial adjustment",
			score:      0.9,
			sampleSize: 5,
			minScore:   0.65,
			maxScore:   0.75,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scorer.adjustForSampleSize(tt.score, tt.sampleSize)

			if result < tt.minScore || result > tt.maxScore {
				t.Errorf("Adjusted score %f not in range [%f, %f]",
					result, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestCollectAgentStats(t *testing.T) {
	sessions := []Session{
		{ID: "1", AgentName: "test-agent", Success: true, Duration: 1000, ErrorCount: 0},
		{ID: "2", AgentName: "test-agent", Success: true, Duration: 2000, ErrorCount: 1},
		{ID: "3", AgentName: "test-agent", Success: false, Duration: 3000, ErrorCount: 5},
		{ID: "4", AgentName: "other-agent", Success: true, Duration: 1000, ErrorCount: 0},
	}

	metrics := []BehavioralMetrics{
		{
			TotalSessions: 4,
			AgentPerformance: map[string]int{"test-agent": 2},
			TokenUsage: TokenUsage{CostUSD: 0.10},
		},
	}

	scorer := NewPerformanceScorer(sessions, metrics)
	stats := scorer.collectAgentStats("test-agent")

	if stats.totalSessions != 3 {
		t.Errorf("Expected 3 sessions, got %d", stats.totalSessions)
	}

	if stats.successCount != 2 {
		t.Errorf("Expected 2 successes, got %d", stats.successCount)
	}

	if stats.totalDuration != 6000 {
		t.Errorf("Expected total duration 6000, got %d", stats.totalDuration)
	}

	if stats.errorCount != 2 {
		t.Errorf("Expected 2 error sessions, got %d", stats.errorCount)
	}

	if stats.recoveryCount != 1 {
		t.Errorf("Expected 1 recovery, got %d", stats.recoveryCount)
	}
}

func TestCalculateGlobalAverageCost(t *testing.T) {
	metrics := []BehavioralMetrics{
		{
			TotalSessions: 5,
			TokenUsage: TokenUsage{CostUSD: 0.10},
		},
		{
			TotalSessions: 5,
			TokenUsage: TokenUsage{CostUSD: 0.20},
		},
	}

	scorer := NewPerformanceScorer([]Session{}, metrics)
	avgCost := scorer.calculateGlobalAverageCost()

	expectedAvg := 0.30 / 10.0 // Total cost / total sessions
	tolerance := 0.0001
	if avgCost < expectedAvg-tolerance || avgCost > expectedAvg+tolerance {
		t.Errorf("Expected average cost %f, got %f", expectedAvg, avgCost)
	}
}

func TestCalculateGlobalAverageDuration(t *testing.T) {
	sessions := []Session{
		{Duration: 1000},
		{Duration: 2000},
		{Duration: 3000},
	}

	scorer := NewPerformanceScorer(sessions, []BehavioralMetrics{})
	avgDuration := scorer.calculateGlobalAverageDuration()

	expectedAvg := 2000.0
	if avgDuration != expectedAvg {
		t.Errorf("Expected average duration %f, got %f", expectedAvg, avgDuration)
	}
}

func TestGetUniqueAgents(t *testing.T) {
	sessions := []Session{
		{AgentName: "agent-a"},
		{AgentName: "agent-b"},
		{AgentName: "agent-a"},
		{AgentName: "agent-c"},
		{AgentName: "agent-b"},
	}

	scorer := NewPerformanceScorer(sessions, []BehavioralMetrics{})
	agents := scorer.getUniqueAgents()

	if len(agents) != 3 {
		t.Errorf("Expected 3 unique agents, got %d", len(agents))
	}

	// Verify sorted order
	expected := []string{"agent-a", "agent-b", "agent-c"}
	for i, agent := range agents {
		if agent != expected[i] {
			t.Errorf("Expected agent %s at index %d, got %s", expected[i], i, agent)
		}
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "contains at start",
			s:        "backend-api",
			substr:   "backend",
			expected: true,
		},
		{
			name:     "contains at end",
			s:        "api-backend",
			substr:   "backend",
			expected: true,
		},
		{
			name:     "exact match",
			s:        "backend",
			substr:   "backend",
			expected: true,
		},
		{
			name:     "does not contain",
			s:        "frontend-ui",
			substr:   "backend",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func BenchmarkScoreAgent(b *testing.B) {
	// Setup test data
	sessions := make([]Session, 1000)
	for i := 0; i < 1000; i++ {
		sessions[i] = Session{
			ID:         "session-" + string(rune(i)),
			AgentName:  "benchmark-agent",
			Success:    i%2 == 0,
			Duration:   int64(1000 + i),
			ErrorCount: i % 5,
			Timestamp:  time.Now(),
		}
	}

	metrics := []BehavioralMetrics{
		{
			TotalSessions: 1000,
			AgentPerformance: map[string]int{"benchmark-agent": 500},
			TokenUsage: TokenUsage{CostUSD: 10.0},
		},
	}

	scorer := NewPerformanceScorer(sessions, metrics)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scorer.ScoreAgent("benchmark-agent")
	}
}

func BenchmarkRankAgents(b *testing.B) {
	// Setup test data with multiple agents
	sessions := make([]Session, 1000)
	agentNames := []string{"agent-a", "agent-b", "agent-c", "agent-d", "agent-e"}

	for i := 0; i < 1000; i++ {
		sessions[i] = Session{
			ID:         "session-" + string(rune(i)),
			AgentName:  agentNames[i%len(agentNames)],
			Success:    i%3 == 0,
			Duration:   int64(1000 + i),
			ErrorCount: i % 7,
			Timestamp:  time.Now(),
		}
	}

	metrics := []BehavioralMetrics{
		{
			TotalSessions: 1000,
			AgentPerformance: map[string]int{
				"agent-a": 100,
				"agent-b": 80,
				"agent-c": 90,
				"agent-d": 70,
				"agent-e": 60,
			},
			TokenUsage: TokenUsage{CostUSD: 10.0},
		},
	}

	scorer := NewPerformanceScorer(sessions, metrics)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scorer.RankAgents()
	}
}
