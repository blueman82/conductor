package learning

import (
	"testing"
	"time"
)

func TestSelectBetterAgent(t *testing.T) {
	// Helper to create execution history
	makeHistory := func(agent string, success bool) *TaskExecution {
		return &TaskExecution{
			Agent:     agent,
			Success:   success,
			Timestamp: time.Now(),
		}
	}

	tests := []struct {
		name          string
		currentAgent  string
		qcSuggestion  string
		history       []*TaskExecution
		wantAgent     string
		wantReasoning string
	}{
		{
			name:          "QC suggestion provided and different from current",
			currentAgent:  "backend-dev",
			qcSuggestion:  "golang-pro",
			history:       nil,
			wantAgent:     "golang-pro",
			wantReasoning: "QC suggested agent",
		},
		{
			name:          "QC suggestion same as current agent (ignore)",
			currentAgent:  "backend-dev",
			qcSuggestion:  "backend-dev",
			history:       nil,
			wantAgent:     "general-purpose",
			wantReasoning: "fallback agent",
		},
		{
			name:         "best performer from history",
			currentAgent: "backend-dev",
			qcSuggestion: "",
			history: []*TaskExecution{
				makeHistory("test-automator", true),
				makeHistory("test-automator", true),
				makeHistory("test-automator", true),
				makeHistory("golang-pro", true),
				makeHistory("golang-pro", false),
				makeHistory("backend-dev", false), // Current agent
			},
			wantAgent:     "test-automator",
			wantReasoning: "historical best performer",
		},
		{
			name:         "exclude current failed agent",
			currentAgent: "backend-dev",
			qcSuggestion: "",
			history: []*TaskExecution{
				makeHistory("backend-dev", true),
				makeHistory("backend-dev", true),
				makeHistory("backend-dev", false), // Most recent failure
			},
			wantAgent:     "general-purpose",
			wantReasoning: "fallback agent",
		},
		{
			name:          "empty history fallback",
			currentAgent:  "backend-dev",
			qcSuggestion:  "",
			history:       []*TaskExecution{},
			wantAgent:     "general-purpose",
			wantReasoning: "fallback agent",
		},
		{
			name:          "nil history fallback",
			currentAgent:  "backend-dev",
			qcSuggestion:  "",
			history:       nil,
			wantAgent:     "general-purpose",
			wantReasoning: "fallback agent",
		},
		{
			name:         "multiple agents with same success rate - deterministic order",
			currentAgent: "backend-dev",
			qcSuggestion: "",
			history: []*TaskExecution{
				makeHistory("golang-pro", true),
				makeHistory("test-automator", true),
				makeHistory("backend-dev", false),
			},
			wantAgent:     "golang-pro", // Both 100%, alphabetically first
			wantReasoning: "historical best performer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAgent, gotReasoning := SelectBetterAgent(tt.currentAgent, tt.history, tt.qcSuggestion)
			if gotAgent != tt.wantAgent {
				t.Errorf("SelectBetterAgent() agent = %v, want %v", gotAgent, tt.wantAgent)
			}
			if gotReasoning != tt.wantReasoning {
				t.Errorf("SelectBetterAgent() reasoning = %v, want %v", gotReasoning, tt.wantReasoning)
			}
		})
	}
}

func TestAnalyzeAgentPerformance(t *testing.T) {
	history := []*TaskExecution{
		{Agent: "golang-pro", Success: true},
		{Agent: "golang-pro", Success: true},
		{Agent: "golang-pro", Success: false},
		{Agent: "test-automator", Success: true},
		{Agent: "test-automator", Success: true},
		{Agent: "test-automator", Success: true},
		{Agent: "backend-dev", Success: false},
		{Agent: "backend-dev", Success: false},
	}

	stats := analyzeAgentPerformance(history)

	// Check test-automator (100% success rate)
	if stats["test-automator"].SuccessRate != 1.0 {
		t.Errorf("test-automator success rate = %v, want 1.0", stats["test-automator"].SuccessRate)
	}
	if stats["test-automator"].TotalRuns != 3 {
		t.Errorf("test-automator total runs = %v, want 3", stats["test-automator"].TotalRuns)
	}

	// Check golang-pro (66.7% success rate)
	expectedRate := 2.0 / 3.0
	if stats["golang-pro"].SuccessRate < expectedRate-0.01 || stats["golang-pro"].SuccessRate > expectedRate+0.01 {
		t.Errorf("golang-pro success rate = %v, want ~%v", stats["golang-pro"].SuccessRate, expectedRate)
	}

	// Check backend-dev (0% success rate)
	if stats["backend-dev"].SuccessRate != 0.0 {
		t.Errorf("backend-dev success rate = %v, want 0.0", stats["backend-dev"].SuccessRate)
	}
}

func TestFilterOutAgent(t *testing.T) {
	stats := map[string]*AgentStats{
		"test-automator": {Agent: "test-automator", SuccessRate: 1.0, TotalRuns: 3},
		"golang-pro":     {Agent: "golang-pro", SuccessRate: 0.67, TotalRuns: 3},
		"backend-dev":    {Agent: "backend-dev", SuccessRate: 0.0, TotalRuns: 2},
	}

	filtered := filterOutAgent(stats, "backend-dev")

	// Should exclude backend-dev
	if len(filtered) != 2 {
		t.Errorf("filtered length = %v, want 2", len(filtered))
	}

	// Verify backend-dev is not in filtered results
	for _, agent := range filtered {
		if agent.Agent == "backend-dev" {
			t.Errorf("filtered list contains excluded agent: backend-dev")
		}
	}

	// Verify sorted by success rate (test-automator first)
	if filtered[0].Agent != "test-automator" {
		t.Errorf("first agent = %v, want test-automator", filtered[0].Agent)
	}
}

func TestFilterOutAgent_EmptyStats(t *testing.T) {
	stats := map[string]*AgentStats{}
	filtered := filterOutAgent(stats, "backend-dev")

	if len(filtered) != 0 {
		t.Errorf("filtered length = %v, want 0", len(filtered))
	}
}

func TestFilterOutAgent_OnlyExcludedAgent(t *testing.T) {
	stats := map[string]*AgentStats{
		"backend-dev": {Agent: "backend-dev", SuccessRate: 0.5, TotalRuns: 2},
	}
	filtered := filterOutAgent(stats, "backend-dev")

	if len(filtered) != 0 {
		t.Errorf("filtered length = %v, want 0", len(filtered))
	}
}
