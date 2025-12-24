// +build integration

package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// TestLLMFailurePrediction uses gpt-oss via Ollama to predict task failure probability.
// Run with: go test -tags=integration -run TestLLMFailurePrediction -v ./internal/executor/
func TestLLMFailurePrediction(t *testing.T) {
	// Check Ollama is running
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		t.Skipf("Ollama not running: %v", err)
	}
	resp.Body.Close()

	// Load plan
	planPath := "../../docs/plans/pattern-intelligence.yaml"
	planData, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("Failed to read plan: %v", err)
	}

	var plan struct {
		Plan struct {
			Tasks []struct {
				TaskNumber      string   `yaml:"task_number"`
				Name            string   `yaml:"name"`
				Agent           string   `yaml:"agent"`
				Files           []string `yaml:"files"`
				DependsOn       []any    `yaml:"depends_on"`
				EstimatedTime   string   `yaml:"estimated_time"`
				SuccessCriteria []string `yaml:"success_criteria"`
				Description     string   `yaml:"description"`
			} `yaml:"tasks"`
		} `yaml:"plan"`
	}
	if err := yaml.Unmarshal(planData, &plan); err != nil {
		t.Fatalf("Failed to parse plan: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("LLM-Enhanced Failure Prediction (gpt-oss with reasoning)")
	fmt.Println(strings.Repeat("=", 80))

	for _, task := range plan.Plan.Tasks {
		prediction := predictTaskFailure(t, task.TaskNumber, task.Name, task.Agent, task.Files, task.SuccessCriteria, task.Description)

		// Color code based on probability
		var indicator string
		switch {
		case prediction.Probability < 0.3:
			indicator = "🟢"
		case prediction.Probability < 0.6:
			indicator = "🟡"
		default:
			indicator = "🔴"
		}

		fmt.Printf("\n%s Task %s: %s\n", indicator, task.TaskNumber, task.Name)
		fmt.Printf("   Probability: %.1f%%\n", prediction.Probability*100)
		fmt.Printf("   Confidence:  %.1f%%\n", prediction.Confidence*100)
		fmt.Printf("   Reasoning:   %s\n", truncate(prediction.Reasoning, 200))
		if len(prediction.RiskFactors) > 0 {
			fmt.Printf("   Risks:       %s\n", strings.Join(prediction.RiskFactors, "; "))
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
}

type FailurePrediction struct {
	Probability float64  `json:"probability"`
	Confidence  float64  `json:"confidence"`
	Reasoning   string   `json:"reasoning"`
	RiskFactors []string `json:"risk_factors"`
}

func predictTaskFailure(t *testing.T, taskNum, name, agent string, files, criteria []string, description string) FailurePrediction {
	prompt := fmt.Sprintf(`You are an expert at predicting software task failures.

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
		taskNum, name, agent,
		strings.Join(files, ", "),
		formatCriteria(criteria),
		truncate(description, 500))

	// Call Ollama with gpt-oss and reasoning
	reqBody := map[string]any{
		"model":  "gpt-oss:latest",
		"prompt": prompt,
		"stream": false,
		"options": map[string]any{
			"temperature": 0.3,
		},
		"think": "medium", // Use medium reasoning level
	}

	jsonBody, _ := json.Marshal(reqBody)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "POST", "http://localhost:11434/api/generate", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("Task %s: Ollama request failed: %v", taskNum, err)
		return FailurePrediction{Probability: 0.5, Confidence: 0.0, Reasoning: "LLM unavailable"}
	}
	defer resp.Body.Close()

	var result struct {
		Response string `json:"response"`
		Thinking string `json:"thinking"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Logf("Task %s: Failed to decode response: %v", taskNum, err)
		return FailurePrediction{Probability: 0.5, Confidence: 0.0, Reasoning: "Parse error"}
	}

	// Log thinking trace if present
	if result.Thinking != "" {
		t.Logf("Task %s thinking: %s", taskNum, truncate(result.Thinking, 300))
	}

	// Extract JSON from response
	response := result.Response
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start == -1 || end == -1 || end <= start {
		t.Logf("Task %s: No JSON in response: %s", taskNum, truncate(response, 200))
		return FailurePrediction{Probability: 0.5, Confidence: 0.0, Reasoning: "No JSON in response"}
	}

	var prediction FailurePrediction
	if err := json.Unmarshal([]byte(response[start:end+1]), &prediction); err != nil {
		t.Logf("Task %s: Failed to parse prediction JSON: %v", taskNum, err)
		return FailurePrediction{Probability: 0.5, Confidence: 0.0, Reasoning: "JSON parse error"}
	}

	return prediction
}

func formatCriteria(criteria []string) string {
	var sb strings.Builder
	for i, c := range criteria {
		if i < 5 { // Limit to first 5 criteria
			sb.WriteString(fmt.Sprintf("- %s\n", c))
		}
	}
	if len(criteria) > 5 {
		sb.WriteString(fmt.Sprintf("... and %d more\n", len(criteria)-5))
	}
	return sb.String()
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
