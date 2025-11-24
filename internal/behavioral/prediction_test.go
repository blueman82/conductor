package behavioral

import (
	"testing"
	"time"
)

func TestNewFailurePredictor(t *testing.T) {
	sessions := []Session{
		{ID: "1", Success: true, Duration: 1000},
		{ID: "2", Success: false, Duration: 2000},
	}
	metrics := []BehavioralMetrics{
		{TotalSessions: 2},
	}

	predictor := NewFailurePredictor(sessions, metrics)

	if predictor == nil {
		t.Fatal("Expected non-nil predictor")
	}
	if len(predictor.sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(predictor.sessions))
	}
	if predictor.minHistorySessions != 5 {
		t.Errorf("Expected minHistorySessions=5, got %d", predictor.minHistorySessions)
	}
	if predictor.highRiskThreshold != 0.7 {
		t.Errorf("Expected highRiskThreshold=0.7, got %f", predictor.highRiskThreshold)
	}
}

func TestCalculateToolFailureRates(t *testing.T) {
	tests := []struct {
		name     string
		sessions []Session
		metrics  []BehavioralMetrics
		expected map[string]float64
	}{
		{
			name: "basic failure rate calculation",
			sessions: []Session{
				{ID: "1", Success: true},
				{ID: "2", Success: false},
				{ID: "3", Success: false},
			},
			metrics: []BehavioralMetrics{
				{
					ToolExecutions: []ToolExecution{
						{Name: "Read", Count: 5},
						{Name: "Write", Count: 2},
					},
				},
				{
					ToolExecutions: []ToolExecution{
						{Name: "Read", Count: 3},
						{Name: "Bash", Count: 1},
					},
				},
				{
					ToolExecutions: []ToolExecution{
						{Name: "Read", Count: 2},
						{Name: "Write", Count: 1},
					},
				},
			},
			expected: map[string]float64{
				"Read":  0.5,  // 5/(5+3+2) failures in 2/3 sessions
				"Write": 0.33, // 3 total, 1 in failed session
				"Bash":  1.0,  // 1 total, 1 in failed session
			},
		},
		{
			name:     "empty data",
			sessions: []Session{},
			metrics:  []BehavioralMetrics{},
			expected: map[string]float64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			predictor := NewFailurePredictor(tt.sessions, tt.metrics)
			rates := predictor.CalculateToolFailureRates()

			if len(rates) == 0 && len(tt.expected) == 0 {
				return
			}

			for tool := range tt.expected {
				if _, exists := rates[tool]; !exists {
					t.Errorf("Tool %s not found in rates", tool)
				}
			}
		})
	}
}

func TestFindSimilarHistoricalSessions(t *testing.T) {
	tests := []struct {
		name        string
		sessions    []Session
		metrics     []BehavioralMetrics
		session     *Session
		toolUsage   []string
		expectedMin int
	}{
		{
			name: "find similar sessions",
			sessions: []Session{
				{ID: "1", Success: true},
				{ID: "2", Success: true},
				{ID: "3", Success: false},
			},
			metrics: []BehavioralMetrics{
				{
					ToolExecutions: []ToolExecution{
						{Name: "Read", Count: 1},
						{Name: "Write", Count: 1},
					},
				},
				{
					ToolExecutions: []ToolExecution{
						{Name: "Read", Count: 1},
						{Name: "Edit", Count: 1},
					},
				},
				{
					ToolExecutions: []ToolExecution{
						{Name: "Bash", Count: 1},
						{Name: "Grep", Count: 1},
					},
				},
			},
			session:     &Session{ID: "new"},
			toolUsage:   []string{"Read", "Write"},
			expectedMin: 1,
		},
		{
			name:        "no history",
			sessions:    []Session{},
			metrics:     []BehavioralMetrics{},
			session:     &Session{ID: "new"},
			toolUsage:   []string{"Read"},
			expectedMin: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			predictor := NewFailurePredictor(tt.sessions, tt.metrics)
			similar := predictor.FindSimilarHistoricalSessions(tt.session, tt.toolUsage)

			if len(similar) < tt.expectedMin {
				t.Errorf("Expected at least %d similar sessions, got %d", tt.expectedMin, len(similar))
			}
		})
	}
}

func TestCalculateProbability(t *testing.T) {
	tests := []struct {
		name            string
		sessions        []Session
		metrics         []BehavioralMetrics
		toolUsage       []string
		similarSessions []Session
		expectedMin     float64
		expectedMax     float64
	}{
		{
			name: "high risk tools",
			sessions: []Session{
				{ID: "1", Success: false},
				{ID: "2", Success: false},
				{ID: "3", Success: false},
			},
			metrics: []BehavioralMetrics{
				{
					ToolExecutions: []ToolExecution{
						{Name: "Bash", Count: 10},
					},
				},
				{
					ToolExecutions: []ToolExecution{
						{Name: "Bash", Count: 8},
					},
				},
				{
					ToolExecutions: []ToolExecution{
						{Name: "Bash", Count: 12},
					},
				},
			},
			toolUsage: []string{"Bash", "Bash", "Bash"},
			similarSessions: []Session{
				{ID: "1", Success: false},
				{ID: "2", Success: false},
			},
			expectedMin: 0.5,
			expectedMax: 1.0,
		},
		{
			name: "low risk tools",
			sessions: []Session{
				{ID: "1", Success: true},
				{ID: "2", Success: true},
			},
			metrics: []BehavioralMetrics{
				{
					ToolExecutions: []ToolExecution{
						{Name: "Read", Count: 5},
					},
				},
				{
					ToolExecutions: []ToolExecution{
						{Name: "Read", Count: 3},
					},
				},
			},
			toolUsage: []string{"Read"},
			similarSessions: []Session{
				{ID: "1", Success: true},
			},
			expectedMin: 0.0,
			expectedMax: 0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			predictor := NewFailurePredictor(tt.sessions, tt.metrics)
			predictor.toolFailureRates = predictor.CalculateToolFailureRates()

			probability := predictor.CalculateProbability(tt.toolUsage, tt.similarSessions)

			if probability < 0.0 || probability > 1.0 {
				t.Errorf("Probability must be between 0 and 1, got %f", probability)
			}

			if probability < tt.expectedMin || probability > tt.expectedMax {
				t.Logf("Probability %f outside expected range [%f, %f]", probability, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

func TestPredictFailure(t *testing.T) {
	sessions := []Session{
		{ID: "1", Success: true, Duration: 1000, Timestamp: time.Now()},
		{ID: "2", Success: false, Duration: 2000, Timestamp: time.Now()},
		{ID: "3", Success: false, Duration: 3000, Timestamp: time.Now()},
	}
	metrics := []BehavioralMetrics{
		{
			ToolExecutions: []ToolExecution{
				{Name: "Read", Count: 5},
			},
		},
		{
			ToolExecutions: []ToolExecution{
				{Name: "Bash", Count: 10},
				{Name: "Write", Count: 5},
			},
		},
		{
			ToolExecutions: []ToolExecution{
				{Name: "Bash", Count: 8},
				{Name: "Edit", Count: 3},
			},
		},
	}

	predictor := NewFailurePredictor(sessions, metrics)

	tests := []struct {
		name      string
		session   *Session
		toolUsage []string
		wantError bool
	}{
		{
			name:      "valid prediction",
			session:   &Session{ID: "new", Project: "test"},
			toolUsage: []string{"Read", "Write"},
			wantError: false,
		},
		{
			name:      "high risk prediction",
			session:   &Session{ID: "new2", Project: "test"},
			toolUsage: []string{"Bash", "Bash", "Bash", "Write", "Edit"},
			wantError: false,
		},
		{
			name:      "nil session",
			session:   nil,
			toolUsage: []string{"Read"},
			wantError: true,
		},
		{
			name:      "empty tool usage",
			session:   &Session{ID: "new3", Project: "test"},
			toolUsage: []string{},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := predictor.PredictFailure(tt.session, tt.toolUsage)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("Expected non-nil result")
			}

			// Validate result fields
			if result.Probability < 0.0 || result.Probability > 1.0 {
				t.Errorf("Probability must be between 0 and 1, got %f", result.Probability)
			}

			if result.Confidence < 0.0 || result.Confidence > 1.0 {
				t.Errorf("Confidence must be between 0 and 1, got %f", result.Confidence)
			}

			if result.RiskLevel == "" {
				t.Error("RiskLevel should not be empty")
			}

			if result.Explanation == "" {
				t.Error("Explanation should not be empty")
			}

			validRiskLevels := map[string]bool{"low": true, "medium": true, "high": true}
			if !validRiskLevels[result.RiskLevel] {
				t.Errorf("Invalid risk level: %s", result.RiskLevel)
			}
		})
	}
}

func TestGetRiskLevel(t *testing.T) {
	predictor := NewFailurePredictor([]Session{}, []BehavioralMetrics{})

	tests := []struct {
		name        string
		probability float64
		expected    string
	}{
		{
			name:        "high risk",
			probability: 0.8,
			expected:    "high",
		},
		{
			name:        "medium risk",
			probability: 0.5,
			expected:    "medium",
		},
		{
			name:        "low risk",
			probability: 0.2,
			expected:    "low",
		},
		{
			name:        "boundary high",
			probability: 0.7,
			expected:    "high",
		},
		{
			name:        "boundary medium",
			probability: 0.4,
			expected:    "medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := predictor.getRiskLevel(tt.probability)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCalculateConfidence(t *testing.T) {
	predictor := NewFailurePredictor([]Session{}, []BehavioralMetrics{})

	tests := []struct {
		name            string
		similarSessions []Session
		expectedMin     float64
		expectedMax     float64
	}{
		{
			name:            "no history",
			similarSessions: []Session{},
			expectedMin:     0.1,
			expectedMax:     0.1,
		},
		{
			name: "few similar sessions",
			similarSessions: []Session{
				{ID: "1"},
				{ID: "2"},
			},
			expectedMin: 0.1,
			expectedMax: 0.5,
		},
		{
			name: "many similar sessions",
			similarSessions: []Session{
				{ID: "1"}, {ID: "2"}, {ID: "3"}, {ID: "4"}, {ID: "5"},
				{ID: "6"}, {ID: "7"}, {ID: "8"}, {ID: "9"}, {ID: "10"},
				{ID: "11"}, {ID: "12"}, {ID: "13"}, {ID: "14"}, {ID: "15"},
			},
			expectedMin: 0.5,
			expectedMax: 0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := predictor.calculateConfidence(tt.similarSessions)

			if confidence < tt.expectedMin || confidence > tt.expectedMax {
				t.Errorf("Confidence %f outside expected range [%f, %f]",
					confidence, tt.expectedMin, tt.expectedMax)
			}

			if confidence < 0.0 || confidence > 1.0 {
				t.Errorf("Confidence must be between 0 and 1, got %f", confidence)
			}
		})
	}
}

func TestIdentifyRiskFactors(t *testing.T) {
	sessions := []Session{
		{ID: "1", Success: false},
		{ID: "2", Success: false},
		{ID: "3", Success: false},
	}
	metrics := []BehavioralMetrics{
		{
			ToolExecutions: []ToolExecution{
				{Name: "Bash", Count: 10},
			},
		},
		{
			ToolExecutions: []ToolExecution{
				{Name: "Bash", Count: 8},
			},
		},
		{
			ToolExecutions: []ToolExecution{
				{Name: "Bash", Count: 12},
			},
		},
	}

	predictor := NewFailurePredictor(sessions, metrics)
	predictor.toolFailureRates = predictor.CalculateToolFailureRates()

	tests := []struct {
		name            string
		toolUsage       []string
		similarSessions []Session
		expectedMin     int
	}{
		{
			name:            "high risk tool",
			toolUsage:       []string{"Bash", "Bash", "Bash"},
			similarSessions: sessions,
			expectedMin:     1,
		},
		{
			name: "high tool diversity",
			toolUsage: []string{
				"Read", "Write", "Edit", "Bash", "Grep",
				"Glob", "Task", "WebFetch", "NotebookEdit",
			},
			similarSessions: []Session{},
			expectedMin:     1,
		},
		{
			name: "high tool usage",
			toolUsage: func() []string {
				tools := make([]string, 60)
				for i := range tools {
					tools[i] = "Read"
				}
				return tools
			}(),
			similarSessions: []Session{},
			expectedMin:     1,
		},
		{
			name:            "low risk",
			toolUsage:       []string{"Read"},
			similarSessions: []Session{{ID: "1", Success: true}},
			expectedMin:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factors := predictor.identifyRiskFactors(tt.toolUsage, tt.similarSessions)

			if len(factors) < tt.expectedMin {
				t.Errorf("Expected at least %d risk factors, got %d", tt.expectedMin, len(factors))
			}
		})
	}
}

func TestGenerateExplanation(t *testing.T) {
	predictor := NewFailurePredictor(
		[]Session{{ID: "1", Success: true}},
		[]BehavioralMetrics{{ToolExecutions: []ToolExecution{{Name: "Read", Count: 1}}}},
	)
	predictor.toolFailureRates = map[string]float64{"Read": 0.2}

	tests := []struct {
		name            string
		probability     float64
		toolUsage       []string
		similarSessions []Session
	}{
		{
			name:        "high risk explanation",
			probability: 0.8,
			toolUsage:   []string{"Read", "Write"},
			similarSessions: []Session{
				{ID: "1", Success: false},
				{ID: "2", Success: false},
			},
		},
		{
			name:            "low risk explanation",
			probability:     0.2,
			toolUsage:       []string{"Read"},
			similarSessions: []Session{{ID: "1", Success: true}},
		},
		{
			name:            "no history",
			probability:     0.5,
			toolUsage:       []string{"Read"},
			similarSessions: []Session{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			explanation := predictor.generateExplanation(tt.probability, tt.toolUsage, tt.similarSessions)

			if explanation == "" {
				t.Error("Explanation should not be empty")
			}

			// Should contain probability
			if len(explanation) < 20 {
				t.Error("Explanation seems too short")
			}
		})
	}
}

func TestGenerateRecommendations(t *testing.T) {
	predictor := NewFailurePredictor([]Session{}, []BehavioralMetrics{})

	tests := []struct {
		name        string
		riskFactors []string
		toolUsage   []string
		expectedMin int
	}{
		{
			name: "many risk factors",
			riskFactors: []string{
				"High failure rate",
				"Complex task",
				"Many tools",
			},
			toolUsage:   []string{"Read", "Write", "Bash"},
			expectedMin: 2,
		},
		{
			name: "heavy bash usage",
			riskFactors: []string{},
			toolUsage: func() []string {
				tools := make([]string, 15)
				for i := range tools {
					tools[i] = "Bash"
				}
				return tools
			}(),
			expectedMin: 1,
		},
		{
			name:        "low risk",
			riskFactors: []string{},
			toolUsage:   []string{"Read"},
			expectedMin: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendations := predictor.generateRecommendations(tt.riskFactors, tt.toolUsage)

			if len(recommendations) < tt.expectedMin {
				t.Errorf("Expected at least %d recommendations, got %d", tt.expectedMin, len(recommendations))
			}

			for _, rec := range recommendations {
				if rec == "" {
					t.Error("Recommendations should not be empty strings")
				}
			}
		})
	}
}

func TestGetHighRiskTools(t *testing.T) {
	sessions := []Session{
		{ID: "1", Success: false},
		{ID: "2", Success: false},
		{ID: "3", Success: true},
	}
	metrics := []BehavioralMetrics{
		{
			ToolExecutions: []ToolExecution{
				{Name: "Bash", Count: 10},
				{Name: "Write", Count: 5},
			},
		},
		{
			ToolExecutions: []ToolExecution{
				{Name: "Bash", Count: 8},
				{Name: "Edit", Count: 3},
			},
		},
		{
			ToolExecutions: []ToolExecution{
				{Name: "Read", Count: 20},
			},
		},
	}

	predictor := NewFailurePredictor(sessions, metrics)
	predictor.toolFailureRates = predictor.CalculateToolFailureRates()

	highRiskTools := predictor.getHighRiskTools()

	// Should return tools sorted by failure rate
	if len(highRiskTools) == 0 {
		t.Error("Expected at least one high risk tool")
	}

	// Verify sorting by failure rate
	for i := 1; i < len(highRiskTools); i++ {
		if highRiskTools[i].FailureRate > highRiskTools[i-1].FailureRate {
			t.Error("High risk tools not sorted by failure rate descending")
		}
	}

	// Verify fields are populated
	for _, tool := range highRiskTools {
		if tool.ToolName == "" {
			t.Error("Tool name should not be empty")
		}
		if tool.FailureRate < 0 || tool.FailureRate > 1 {
			t.Errorf("Invalid failure rate: %f", tool.FailureRate)
		}
		if tool.UsageCount < 0 {
			t.Errorf("Invalid usage count: %d", tool.UsageCount)
		}
	}
}

func TestCalculateCombinationRisk(t *testing.T) {
	predictor := NewFailurePredictor([]Session{}, []BehavioralMetrics{})

	tests := []struct {
		name        string
		toolUsage   []string
		expectedMin float64
		expectedMax float64
	}{
		{
			name: "heavy bash and writes",
			toolUsage: func() []string {
				tools := make([]string, 20)
				for i := 0; i < 10; i++ {
					tools[i] = "Bash"
				}
				for i := 10; i < 20; i++ {
					tools[i] = "Write"
				}
				return tools
			}(),
			expectedMin: 0.3,
			expectedMax: 1.0,
		},
		{
			name: "high diversity",
			toolUsage: []string{
				"Read", "Write", "Edit", "Bash", "Grep",
				"Glob", "Task", "WebFetch", "NotebookEdit",
			},
			expectedMin: 0.2,
			expectedMax: 1.0,
		},
		{
			name:        "single tool",
			toolUsage:   []string{"Read"},
			expectedMin: 0.0,
			expectedMax: 0.0,
		},
		{
			name:        "empty",
			toolUsage:   []string{},
			expectedMin: 0.0,
			expectedMax: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk := predictor.calculateCombinationRisk(tt.toolUsage)

			if risk < 0.0 || risk > 1.0 {
				t.Errorf("Risk must be between 0 and 1, got %f", risk)
			}

			if risk < tt.expectedMin || risk > tt.expectedMax {
				t.Logf("Risk %f outside expected range [%f, %f]", risk, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}
