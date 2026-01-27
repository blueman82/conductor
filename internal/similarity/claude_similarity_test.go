package similarity

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewClaudeSimilarity(t *testing.T) {
	cs := NewClaudeSimilarity(90*time.Second, nil)

	if cs.Invoker() == nil {
		t.Error("Invoker should not be nil")
	}
	if cs.Invoker().Timeout != 90*time.Second {
		t.Errorf("Invoker Timeout = %v, want 90s", cs.Invoker().Timeout)
	}
	if cs.Invoker().ClaudePath != "claude" {
		t.Errorf("Default ClaudePath = %s, want claude", cs.Invoker().ClaudePath)
	}
	if cs.Logger != nil {
		t.Errorf("Logger should be nil when not provided")
	}
}

func TestNewClaudeSimilarity_CustomTimeout(t *testing.T) {
	cs := NewClaudeSimilarity(45*time.Second, nil)

	if cs.Invoker() == nil {
		t.Error("Invoker should not be nil")
	}
	if cs.Invoker().Timeout != 45*time.Second {
		t.Errorf("Invoker Timeout = %v, want 45s", cs.Invoker().Timeout)
	}
	if cs.Invoker().ClaudePath != "claude" {
		t.Errorf("ClaudePath = %s, want claude", cs.Invoker().ClaudePath)
	}
	if cs.Logger != nil {
		t.Errorf("Logger should be nil when not provided")
	}
}

func TestBuildPrompt(t *testing.T) {
	cs := NewClaudeSimilarity(30*time.Second, nil)

	prompt := cs.buildPrompt("First description of a task", "Second description of a task")

	if prompt == "" {
		t.Error("buildPrompt returned empty string")
	}

	// Check key elements are present
	if !strings.Contains(prompt, "First description of a task") {
		t.Error("Prompt should contain first description")
	}
	if !strings.Contains(prompt, "Second description of a task") {
		t.Error("Prompt should contain second description")
	}
	if !strings.Contains(prompt, "semantic similarity") {
		t.Error("Prompt should mention semantic similarity")
	}
	if !strings.Contains(prompt, "0.0") && !strings.Contains(prompt, "1.0") {
		t.Error("Prompt should mention score range")
	}
}

func TestSimilaritySchema(t *testing.T) {
	schema := SimilaritySchema()

	if schema == "" {
		t.Error("SimilaritySchema returned empty string")
	}

	// Check required fields are in schema
	if !strings.Contains(schema, "score") {
		t.Error("Schema should contain score")
	}
	if !strings.Contains(schema, "reasoning") {
		t.Error("Schema should contain reasoning")
	}
	if !strings.Contains(schema, `"minimum": 0`) && !strings.Contains(schema, `"minimum":0`) {
		t.Error("Schema should have minimum 0 for score")
	}
	if !strings.Contains(schema, `"maximum": 1`) && !strings.Contains(schema, `"maximum":1`) {
		t.Error("Schema should have maximum 1 for score")
	}
	if !strings.Contains(schema, `"required"`) {
		t.Error("Schema should specify required fields")
	}
}

func TestSimilaritySchema_ValidJSON(t *testing.T) {
	schema := SimilaritySchema()

	// Verify schema is valid JSON by checking key structure
	if !strings.Contains(schema, `"type": "object"`) && !strings.Contains(schema, `"type":"object"`) {
		t.Error("Schema should define type as object")
	}
	if !strings.Contains(schema, `"properties"`) {
		t.Error("Schema should define properties")
	}
}

func TestSimilaritySchema_ScoreConstraints(t *testing.T) {
	schema := SimilaritySchema()

	// Score should be a number with min/max constraints
	if !strings.Contains(schema, `"score"`) {
		t.Error("Schema should define score property")
	}
	if !strings.Contains(schema, "number") {
		t.Error("Schema should specify score as number type")
	}
}

func TestClaudeSimilarity_ImplementsSimilarityInterface(t *testing.T) {
	// Compile-time check that ClaudeSimilarity implements Similarity interface
	var _ Similarity = (*ClaudeSimilarity)(nil)
}

func TestSimilarityResult_Fields(t *testing.T) {
	result := SimilarityResult{
		Score:         0.85,
		Reasoning:     "Both descriptions refer to user authentication",
		SemanticMatch: true,
	}

	if result.Score != 0.85 {
		t.Errorf("Score = %f, want 0.85", result.Score)
	}
	if result.Reasoning != "Both descriptions refer to user authentication" {
		t.Errorf("Reasoning = %s, want 'Both descriptions refer to user authentication'", result.Reasoning)
	}
	if !result.SemanticMatch {
		t.Errorf("SemanticMatch = %v, want true", result.SemanticMatch)
	}
}

// mockWaiterLogger implements budget.WaiterLogger for testing
type mockWaiterLogger struct {
	countdownCalls int
	announceCalls  int
}

func (m *mockWaiterLogger) LogRateLimitCountdown(remaining, total time.Duration) {
	m.countdownCalls++
}

func (m *mockWaiterLogger) LogRateLimitAnnounce(remaining, total time.Duration) {
	m.announceCalls++
}

func TestNewClaudeSimilarityWithLogger(t *testing.T) {
	logger := &mockWaiterLogger{}
	cs := NewClaudeSimilarity(90*time.Second, logger)

	if cs.Logger == nil {
		t.Error("Logger should not be nil when provided")
	}
	if cs.Logger != logger {
		t.Error("Logger should be the one provided")
	}
	if cs.Invoker() == nil {
		t.Error("Invoker should not be nil")
	}
	if cs.Invoker().Timeout != 90*time.Second {
		t.Errorf("Invoker Timeout = %v, want 90s", cs.Invoker().Timeout)
	}
	if cs.Invoker().Logger != logger {
		t.Error("Invoker Logger should be the one provided")
	}
}

func TestSimilaritySchema_HasSemanticMatch(t *testing.T) {
	schema := SimilaritySchema()

	if !strings.Contains(schema, "semantic_match") {
		t.Error("Schema should contain semantic_match property")
	}
	if !strings.Contains(schema, "boolean") {
		t.Error("Schema should specify semantic_match as boolean type")
	}
}

func TestSimilarityResult_JSONParsing(t *testing.T) {
	tests := []struct {
		name          string
		jsonInput     string
		wantScore     float64
		wantReasoning string
		wantMatch     bool
		wantErr       bool
	}{
		{
			name:          "valid high similarity",
			jsonInput:     `{"score": 0.95, "reasoning": "Both describe user login functionality", "semantic_match": true}`,
			wantScore:     0.95,
			wantReasoning: "Both describe user login functionality",
			wantMatch:     true,
			wantErr:       false,
		},
		{
			name:          "valid low similarity",
			jsonInput:     `{"score": 0.15, "reasoning": "Completely different concepts", "semantic_match": false}`,
			wantScore:     0.15,
			wantReasoning: "Completely different concepts",
			wantMatch:     false,
			wantErr:       false,
		},
		{
			name:          "valid medium similarity",
			jsonInput:     `{"score": 0.65, "reasoning": "Some shared concepts", "semantic_match": false}`,
			wantScore:     0.65,
			wantReasoning: "Some shared concepts",
			wantMatch:     false,
			wantErr:       false,
		},
		{
			name:      "invalid json",
			jsonInput: `{not valid json}`,
			wantErr:   true,
		},
		{
			name:          "partial fields",
			jsonInput:     `{"score": 0.5, "reasoning": "", "semantic_match": false}`,
			wantScore:     0.5,
			wantReasoning: "",
			wantMatch:     false,
			wantErr:       false, // JSON unmarshal allows empty values
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result SimilarityResult
			err := json.Unmarshal([]byte(tt.jsonInput), &result)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.Score != tt.wantScore {
				t.Errorf("Score = %f, want %f", result.Score, tt.wantScore)
			}
			if result.Reasoning != tt.wantReasoning {
				t.Errorf("Reasoning = %s, want %s", result.Reasoning, tt.wantReasoning)
			}
			if result.SemanticMatch != tt.wantMatch {
				t.Errorf("SemanticMatch = %v, want %v", result.SemanticMatch, tt.wantMatch)
			}
		})
	}
}

func TestSimilarityResult_JSONParsing_ExtractFromMixed(t *testing.T) {
	// Test extracting JSON from mixed output (like Claude sometimes returns)
	mixedOutput := `Some text before {"score": 0.78, "reasoning": "Similar functionality", "semantic_match": true} some text after`

	// Find JSON in mixed output
	start := strings.Index(mixedOutput, "{")
	end := strings.LastIndex(mixedOutput, "}")
	if start < 0 || end <= start {
		t.Fatal("Could not find JSON in mixed output")
	}

	jsonStr := mixedOutput[start : end+1]
	var result SimilarityResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("Failed to parse extracted JSON: %v", err)
	}

	if result.Score != 0.78 {
		t.Errorf("Score = %f, want 0.78", result.Score)
	}
	if result.Reasoning != "Similar functionality" {
		t.Errorf("Reasoning = %s, want 'Similar functionality'", result.Reasoning)
	}
	if !result.SemanticMatch {
		t.Errorf("SemanticMatch = %v, want true", result.SemanticMatch)
	}
}

func TestSimilarityResult_SemanticMatchFalse(t *testing.T) {
	result := SimilarityResult{
		Score:         0.25,
		Reasoning:     "Descriptions are unrelated",
		SemanticMatch: false,
	}

	if result.SemanticMatch {
		t.Errorf("SemanticMatch should be false for low similarity")
	}
}

func TestBatchSimilaritySchema(t *testing.T) {
	schema := BatchSimilaritySchema()

	if schema == "" {
		t.Error("BatchSimilaritySchema returned empty string")
	}
	if !strings.Contains(schema, "scores") {
		t.Error("Schema should contain scores property")
	}
	if !strings.Contains(schema, "array") {
		t.Error("Schema should specify scores as array type")
	}
}

func TestBuildBatchPrompt(t *testing.T) {
	cs := NewClaudeSimilarity(30*time.Second, nil)

	tests := []struct {
		name       string
		query      string
		candidates []string
		wantParts  []string
	}{
		{
			name:       "single candidate",
			query:      "implement user auth",
			candidates: []string{"add login feature"},
			wantParts:  []string{"implement user auth", "[0] add login feature", "scores"},
		},
		{
			name:       "multiple candidates",
			query:      "fix database bug",
			candidates: []string{"repair SQL query", "update schema", "add index"},
			wantParts:  []string{"fix database bug", "[0] repair SQL query", "[1] update schema", "[2] add index"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := cs.buildBatchPrompt(tt.query, tt.candidates)
			for _, part := range tt.wantParts {
				if !strings.Contains(prompt, part) {
					t.Errorf("Prompt should contain %q", part)
				}
			}
		})
	}
}

func TestCompareBatch_EmptyInput(t *testing.T) {
	cs := NewClaudeSimilarity(30*time.Second, nil)

	scores, err := cs.CompareBatch(nil, "query", nil)
	if err != nil {
		t.Errorf("CompareBatch with nil candidates should not error: %v", err)
	}
	if scores != nil {
		t.Errorf("CompareBatch with nil candidates should return nil scores")
	}

	scores, err = cs.CompareBatch(nil, "query", []string{})
	if err != nil {
		t.Errorf("CompareBatch with empty candidates should not error: %v", err)
	}
	if scores != nil {
		t.Errorf("CompareBatch with empty candidates should return nil scores")
	}
}

func TestBatchSimilarityResponse_JSONParsing(t *testing.T) {
	tests := []struct {
		name       string
		jsonInput  string
		wantScores []float64
		wantErr    bool
	}{
		{
			name:       "valid scores",
			jsonInput:  `{"scores": [0.9, 0.5, 0.1]}`,
			wantScores: []float64{0.9, 0.5, 0.1},
			wantErr:    false,
		},
		{
			name:       "single score",
			jsonInput:  `{"scores": [0.75]}`,
			wantScores: []float64{0.75},
			wantErr:    false,
		},
		{
			name:       "empty scores",
			jsonInput:  `{"scores": []}`,
			wantScores: []float64{},
			wantErr:    false,
		},
		{
			name:      "invalid json",
			jsonInput: `{not valid}`,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result batchSimilarityResponse
			err := json.Unmarshal([]byte(tt.jsonInput), &result)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result.Scores) != len(tt.wantScores) {
				t.Errorf("Scores length = %d, want %d", len(result.Scores), len(tt.wantScores))
				return
			}

			for i, score := range result.Scores {
				if score != tt.wantScores[i] {
					t.Errorf("Scores[%d] = %f, want %f", i, score, tt.wantScores[i])
				}
			}
		})
	}
}
