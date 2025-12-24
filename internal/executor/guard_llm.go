package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// LLMGuardConfig defines LLM-enhanced GUARD configuration
type LLMGuardConfig struct {
	// Enabled enables LLM-based failure prediction
	Enabled bool `yaml:"enabled"`

	// Model specifies the Ollama model to use (e.g., "gpt-oss:latest")
	Model string `yaml:"model"`

	// ThinkLevel specifies reasoning depth: "low", "medium", "high"
	ThinkLevel string `yaml:"think_level"`

	// BaseURL is the Ollama API endpoint (default: http://localhost:11434)
	BaseURL string `yaml:"base_url"`

	// Timeout for LLM requests (default: 60s)
	Timeout time.Duration `yaml:"timeout"`

	// FallbackToStats uses statistical prediction if LLM fails
	FallbackToStats bool `yaml:"fallback_to_stats"`

	// MinProbabilityForLLM only uses LLM when stats probability is uncertain (0.3-0.7)
	MinProbabilityForLLM float64 `yaml:"min_probability_for_llm"`
	MaxProbabilityForLLM float64 `yaml:"max_probability_for_llm"`
}

// DefaultLLMGuardConfig returns default LLM GUARD configuration
func DefaultLLMGuardConfig() LLMGuardConfig {
	return LLMGuardConfig{
		Enabled:              false,
		Model:                "gpt-oss:latest",
		ThinkLevel:           "medium",
		BaseURL:              "http://localhost:11434",
		Timeout:              60 * time.Second,
		FallbackToStats:      true,
		MinProbabilityForLLM: 0.3,
		MaxProbabilityForLLM: 0.7,
	}
}

// LLMPrediction contains the LLM's failure prediction
type LLMPrediction struct {
	Probability float64  `json:"probability"`
	Confidence  float64  `json:"confidence"`
	Reasoning   string   `json:"reasoning"`
	RiskFactors []string `json:"risk_factors"`
	Thinking    string   `json:"-"` // Chain-of-thought trace (not in JSON)
}

// OllamaPredictor uses Ollama with gpt-oss for LLM-based failure prediction
type OllamaPredictor struct {
	config LLMGuardConfig
	client *http.Client
	logger Logger
}

// NewOllamaPredictor creates a new Ollama-based predictor
func NewOllamaPredictor(config LLMGuardConfig, logger Logger) *OllamaPredictor {
	if !config.Enabled {
		return nil
	}

	return &OllamaPredictor{
		config: config,
		client: &http.Client{Timeout: config.Timeout},
		logger: logger,
	}
}

// IsAvailable checks if Ollama is running and the model is available
func (op *OllamaPredictor) IsAvailable(ctx context.Context) bool {
	if op == nil {
		return false
	}

	req, err := http.NewRequestWithContext(ctx, "GET", op.config.BaseURL+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := op.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// PredictFailure uses the LLM to predict task failure probability
func (op *OllamaPredictor) PredictFailure(ctx context.Context, task models.Task) (*LLMPrediction, error) {
	if op == nil {
		return nil, fmt.Errorf("ollama predictor not initialized")
	}

	prompt := op.buildPrompt(task)

	reqBody := map[string]any{
		"model":  op.config.Model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]any{
			"temperature": 0.3,
		},
		"think": op.config.ThinkLevel,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", op.config.BaseURL+"/api/generate", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := op.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var result struct {
		Response string `json:"response"`
		Thinking string `json:"thinking"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	prediction, err := op.parseResponse(result.Response)
	if err != nil {
		return nil, fmt.Errorf("parse prediction: %w", err)
	}

	prediction.Thinking = result.Thinking
	return prediction, nil
}

// buildPrompt constructs the failure prediction prompt for the LLM
func (op *OllamaPredictor) buildPrompt(task models.Task) string {
	var criteriaStr strings.Builder
	for i, c := range task.SuccessCriteria {
		if i < 5 {
			criteriaStr.WriteString(fmt.Sprintf("- %s\n", c))
		}
	}
	if len(task.SuccessCriteria) > 5 {
		criteriaStr.WriteString(fmt.Sprintf("... and %d more\n", len(task.SuccessCriteria)-5))
	}

	description := task.Prompt
	if len(description) > 500 {
		description = description[:500] + "..."
	}

	return fmt.Sprintf(`You are an expert at predicting software task failures.

Analyze this task and predict the probability of failure:

Task: %s - %s
Agent: %s
Files to modify: %s
Success Criteria:
%s

Task Description (first 500 chars):
%s

Respond ONLY with valid JSON in this exact format:
{
  "probability": 0.0 to 1.0,
  "confidence": 0.0 to 1.0,
  "reasoning": "brief explanation",
  "risk_factors": ["factor1", "factor2"]
}

Consider:
- Complexity of files being modified
- Clarity of success criteria
- Agent capability match
- Potential integration issues
- Test coverage requirements`,
		task.Number, task.Name, task.Agent,
		strings.Join(task.Files, ", "),
		criteriaStr.String(),
		description)
}

// parseResponse extracts the prediction JSON from the LLM response
func (op *OllamaPredictor) parseResponse(response string) (*LLMPrediction, error) {
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var prediction LLMPrediction
	if err := json.Unmarshal([]byte(response[start:end+1]), &prediction); err != nil {
		return nil, fmt.Errorf("unmarshal prediction: %w", err)
	}

	// Clamp values to valid range
	if prediction.Probability < 0 {
		prediction.Probability = 0
	}
	if prediction.Probability > 1 {
		prediction.Probability = 1
	}
	if prediction.Confidence < 0 {
		prediction.Confidence = 0
	}
	if prediction.Confidence > 1 {
		prediction.Confidence = 1
	}

	return &prediction, nil
}

// ShouldUseLLM determines if LLM prediction should be used based on stats uncertainty
func (op *OllamaPredictor) ShouldUseLLM(statsProbability float64) bool {
	if op == nil || !op.config.Enabled {
		return false
	}

	// Use LLM when stats are uncertain (between min and max thresholds)
	return statsProbability >= op.config.MinProbabilityForLLM &&
		statsProbability <= op.config.MaxProbabilityForLLM
}
