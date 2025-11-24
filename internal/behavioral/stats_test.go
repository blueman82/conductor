package behavioral

import (
	"testing"
	"time"
)

func TestCalculateStats_Empty(t *testing.T) {
	metrics := []BehavioralMetrics{}
	stats := CalculateStats(metrics)

	if stats.TotalSessions != 0 {
		t.Errorf("expected 0 sessions, got %d", stats.TotalSessions)
	}
	if stats.TotalAgents != 0 {
		t.Errorf("expected 0 agents, got %d", stats.TotalAgents)
	}
	if stats.TotalCost != 0 {
		t.Errorf("expected 0 cost, got %f", stats.TotalCost)
	}
	if len(stats.TopTools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(stats.TopTools))
	}
}

func TestCalculateStats_SingleMetric(t *testing.T) {
	metrics := []BehavioralMetrics{
		{
			TotalSessions:   5,
			SuccessRate:     0.8,
			AverageDuration: 10 * time.Second,
			TotalCost:       1.5,
			TokenUsage: TokenUsage{
				InputTokens:  1000,
				OutputTokens: 500,
				CostUSD:      1.5,
			},
			AgentPerformance: map[string]int{
				"agent1": 4,
			},
			ToolExecutions: []ToolExecution{
				{
					Name:         "Read",
					Count:        10,
					TotalSuccess: 9,
					TotalErrors:  1,
				},
				{
					Name:         "Write",
					Count:        5,
					TotalSuccess: 5,
					TotalErrors:  0,
				},
			},
		},
	}

	stats := CalculateStats(metrics)

	if stats.TotalSessions != 5 {
		t.Errorf("expected 5 sessions, got %d", stats.TotalSessions)
	}
	if stats.TotalAgents != 1 {
		t.Errorf("expected 1 agent, got %d", stats.TotalAgents)
	}
	if stats.TotalCost != 1.5 {
		t.Errorf("expected cost 1.5, got %f", stats.TotalCost)
	}
	if stats.SuccessRate != 0.8 {
		t.Errorf("expected success rate 0.8, got %f", stats.SuccessRate)
	}
	expectedErrorRate := 0.2
	if diff := stats.ErrorRate - expectedErrorRate; diff < -0.0001 || diff > 0.0001 {
		t.Errorf("expected error rate %f, got %f", expectedErrorRate, stats.ErrorRate)
	}
	if stats.AverageDuration != 10*time.Second {
		t.Errorf("expected avg duration 10s, got %v", stats.AverageDuration)
	}
	if stats.TotalInputTokens != 1000 {
		t.Errorf("expected 1000 input tokens, got %d", stats.TotalInputTokens)
	}
	if stats.TotalOutputTokens != 500 {
		t.Errorf("expected 500 output tokens, got %d", stats.TotalOutputTokens)
	}

	// Check agent breakdown
	if stats.AgentBreakdown["agent1"] != 4 {
		t.Errorf("expected agent1 count 4, got %d", stats.AgentBreakdown["agent1"])
	}

	// Check top tools
	if len(stats.TopTools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(stats.TopTools))
	}

	// Tools should be sorted by count (Read=10, Write=5)
	if stats.TopTools[0].Name != "Read" {
		t.Errorf("expected top tool 'Read', got %s", stats.TopTools[0].Name)
	}
	if stats.TopTools[0].Count != 10 {
		t.Errorf("expected top tool count 10, got %d", stats.TopTools[0].Count)
	}
	if stats.TopTools[0].SuccessRate != 0.9 {
		t.Errorf("expected Read success rate 0.9, got %f", stats.TopTools[0].SuccessRate)
	}
	if stats.TopTools[0].ErrorRate != 0.1 {
		t.Errorf("expected Read error rate 0.1, got %f", stats.TopTools[0].ErrorRate)
	}
}

func TestCalculateStats_MultipleMetrics(t *testing.T) {
	metrics := []BehavioralMetrics{
		{
			TotalSessions:   3,
			SuccessRate:     1.0,
			AverageDuration: 5 * time.Second,
			TotalCost:       0.5,
			TokenUsage: TokenUsage{
				InputTokens:  500,
				OutputTokens: 250,
				CostUSD:      0.5,
			},
			AgentPerformance: map[string]int{
				"agent1": 3,
			},
			ToolExecutions: []ToolExecution{
				{
					Name:         "Read",
					Count:        5,
					TotalSuccess: 5,
					TotalErrors:  0,
				},
			},
		},
		{
			TotalSessions:   2,
			SuccessRate:     0.5,
			AverageDuration: 10 * time.Second,
			TotalCost:       1.0,
			TokenUsage: TokenUsage{
				InputTokens:  1000,
				OutputTokens: 500,
				CostUSD:      1.0,
			},
			AgentPerformance: map[string]int{
				"agent2": 1,
			},
			ToolExecutions: []ToolExecution{
				{
					Name:         "Read",
					Count:        3,
					TotalSuccess: 2,
					TotalErrors:  1,
				},
				{
					Name:         "Bash",
					Count:        2,
					TotalSuccess: 1,
					TotalErrors:  1,
				},
			},
		},
	}

	stats := CalculateStats(metrics)

	// Total sessions: 3 + 2 = 5
	if stats.TotalSessions != 5 {
		t.Errorf("expected 5 sessions, got %d", stats.TotalSessions)
	}

	// Total agents: 2 unique
	if stats.TotalAgents != 2 {
		t.Errorf("expected 2 agents, got %d", stats.TotalAgents)
	}

	// Total cost: 0.5 + 1.0 = 1.5
	if stats.TotalCost != 1.5 {
		t.Errorf("expected cost 1.5, got %f", stats.TotalCost)
	}

	// Success rate: (3*1.0 + 2*0.5) / 5 = 4/5 = 0.8
	if stats.SuccessRate != 0.8 {
		t.Errorf("expected success rate 0.8, got %f", stats.SuccessRate)
	}

	// Average duration: (3*5s + 2*10s) / 5 = 35/5 = 7s
	expectedDuration := 7 * time.Second
	if stats.AverageDuration != expectedDuration {
		t.Errorf("expected avg duration 7s, got %v", stats.AverageDuration)
	}

	// Total input tokens: 500 + 1000 = 1500
	if stats.TotalInputTokens != 1500 {
		t.Errorf("expected 1500 input tokens, got %d", stats.TotalInputTokens)
	}

	// Total output tokens: 250 + 500 = 750
	if stats.TotalOutputTokens != 750 {
		t.Errorf("expected 750 output tokens, got %d", stats.TotalOutputTokens)
	}

	// Check agent breakdown
	if stats.AgentBreakdown["agent1"] != 3 {
		t.Errorf("expected agent1 count 3, got %d", stats.AgentBreakdown["agent1"])
	}
	if stats.AgentBreakdown["agent2"] != 1 {
		t.Errorf("expected agent2 count 1, got %d", stats.AgentBreakdown["agent2"])
	}

	// Check tool aggregation: Read count should be 5+3=8
	if len(stats.TopTools) < 1 {
		t.Fatalf("expected at least 1 tool, got %d", len(stats.TopTools))
	}

	// Find Read tool
	var readTool *ToolStatSummary
	for i := range stats.TopTools {
		if stats.TopTools[i].Name == "Read" {
			readTool = &stats.TopTools[i]
			break
		}
	}

	if readTool == nil {
		t.Fatal("Read tool not found in top tools")
	}

	if readTool.Count != 8 {
		t.Errorf("expected Read count 8, got %d", readTool.Count)
	}

	// Read success: (5+2)/(5+3) = 7/8 = 0.875
	expectedSuccessRate := 7.0 / 8.0
	if readTool.SuccessRate != expectedSuccessRate {
		t.Errorf("expected Read success rate %f, got %f", expectedSuccessRate, readTool.SuccessRate)
	}
}

func TestGetTopTools(t *testing.T) {
	stats := &AggregateStats{
		TopTools: []ToolStatSummary{
			{Name: "Read", Count: 100},
			{Name: "Write", Count: 50},
			{Name: "Bash", Count: 30},
			{Name: "Edit", Count: 20},
			{Name: "Grep", Count: 10},
		},
	}

	// Get top 3
	top3 := stats.GetTopTools(3)
	if len(top3) != 3 {
		t.Errorf("expected 3 tools, got %d", len(top3))
	}
	if top3[0].Name != "Read" {
		t.Errorf("expected 'Read' as top tool, got %s", top3[0].Name)
	}

	// Get all
	topAll := stats.GetTopTools(0)
	if len(topAll) != 5 {
		t.Errorf("expected 5 tools, got %d", len(topAll))
	}

	// Get more than available
	topMore := stats.GetTopTools(10)
	if len(topMore) != 5 {
		t.Errorf("expected 5 tools, got %d", len(topMore))
	}
}

func TestGetAgentTypeBreakdown(t *testing.T) {
	stats := &AggregateStats{
		AgentBreakdown: map[string]int{
			"agent1": 10,
			"agent2": 5,
			"agent3": 3,
		},
	}

	breakdown := stats.GetAgentTypeBreakdown()
	if len(breakdown) != 3 {
		t.Errorf("expected 3 agents, got %d", len(breakdown))
	}
	if breakdown["agent1"] != 10 {
		t.Errorf("expected agent1 count 10, got %d", breakdown["agent1"])
	}
	if breakdown["agent2"] != 5 {
		t.Errorf("expected agent2 count 5, got %d", breakdown["agent2"])
	}
	if breakdown["agent3"] != 3 {
		t.Errorf("expected agent3 count 3, got %d", breakdown["agent3"])
	}
}

func TestBuildToolSummaries_Sorting(t *testing.T) {
	toolMap := map[string]*ToolExecution{
		"Read": {
			Name:         "Read",
			Count:        100,
			TotalSuccess: 95,
			TotalErrors:  5,
		},
		"Write": {
			Name:         "Write",
			Count:        150,
			TotalSuccess: 145,
			TotalErrors:  5,
		},
		"Bash": {
			Name:         "Bash",
			Count:        50,
			TotalSuccess: 40,
			TotalErrors:  10,
		},
	}

	summaries := buildToolSummaries(toolMap)

	if len(summaries) != 3 {
		t.Fatalf("expected 3 summaries, got %d", len(summaries))
	}

	// Should be sorted by count descending: Write(150), Read(100), Bash(50)
	if summaries[0].Name != "Write" {
		t.Errorf("expected first tool 'Write', got %s", summaries[0].Name)
	}
	if summaries[1].Name != "Read" {
		t.Errorf("expected second tool 'Read', got %s", summaries[1].Name)
	}
	if summaries[2].Name != "Bash" {
		t.Errorf("expected third tool 'Bash', got %s", summaries[2].Name)
	}

	// Check rates calculation
	if summaries[0].SuccessRate != 145.0/150.0 {
		t.Errorf("expected Write success rate %f, got %f", 145.0/150.0, summaries[0].SuccessRate)
	}
	if summaries[2].ErrorRate != 10.0/50.0 {
		t.Errorf("expected Bash error rate %f, got %f", 10.0/50.0, summaries[2].ErrorRate)
	}
}

func TestCalculateStats_ZeroSessions(t *testing.T) {
	metrics := []BehavioralMetrics{
		{
			TotalSessions: 0,
		},
	}

	stats := CalculateStats(metrics)

	// Should handle division by zero gracefully
	if stats.SuccessRate != 0 {
		t.Errorf("expected success rate 0, got %f", stats.SuccessRate)
	}
	if stats.ErrorRate != 0 {
		t.Errorf("expected error rate 0, got %f", stats.ErrorRate)
	}
	if stats.AverageDuration != 0 {
		t.Errorf("expected avg duration 0, got %v", stats.AverageDuration)
	}
}
