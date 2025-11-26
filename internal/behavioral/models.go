package behavioral

import (
	"errors"
	"time"
)

// ModelCosts defines pricing for different Claude models
// Values are per 1 million tokens (USD)
var ModelCosts = map[string]struct {
	Input  float64
	Output float64
}{
	// Sonnet variants
	"claude-sonnet-4-5":          {Input: 3.0, Output: 15.0},
	"claude-sonnet-4-5-20250929": {Input: 3.0, Output: 15.0},
	"sonnet-4-5":                 {Input: 3.0, Output: 15.0},
	"claude-3-5-sonnet-20241022": {Input: 3.0, Output: 15.0},
	"claude-3-5-sonnet":          {Input: 3.0, Output: 15.0},
	// Haiku variants
	"claude-haiku-3-5":          {Input: 0.80, Output: 4.0},
	"claude-haiku-4-5-20251001": {Input: 0.80, Output: 4.0},
	"claude-3-5-haiku-20241022": {Input: 0.80, Output: 4.0},
	// Opus variants
	"claude-opus-4":            {Input: 15.0, Output: 75.0},
	"claude-opus-4-5-20251101": {Input: 15.0, Output: 75.0},
	"claude-3-opus-20240229":   {Input: 15.0, Output: 75.0},
	// Default fallback
	"default": {Input: 3.0, Output: 15.0},
}

// CalculateCost calculates the cost in USD based on model and token usage
func CalculateCost(model string, inputTokens, outputTokens int64) float64 {
	costs, ok := ModelCosts[model]
	if !ok {
		costs = ModelCosts["default"]
	}
	// Convert to millions of tokens
	inputCost := float64(inputTokens) / 1_000_000 * costs.Input
	outputCost := float64(outputTokens) / 1_000_000 * costs.Output
	return inputCost + outputCost
}

// Session represents a Claude Code agent session extracted from JSONL files
type Session struct {
	ID         string    `json:"id"`          // Session UUID
	Project    string    `json:"project"`     // Project name
	Timestamp  time.Time `json:"timestamp"`   // Session start time
	Status     string    `json:"status"`      // Session status: completed, failed, in_progress
	AgentName  string    `json:"agent_name"`  // Name of the agent used
	Duration   int64     `json:"duration"`    // Session duration in milliseconds
	Success    bool      `json:"success"`     // Overall session success
	ErrorCount int       `json:"error_count"` // Number of errors encountered
}

// Validate checks if the session has all required fields
func (s *Session) Validate() error {
	if s.ID == "" {
		return errors.New("session ID is required")
	}
	if s.Project == "" {
		return errors.New("session project is required")
	}
	if s.Timestamp.IsZero() {
		return errors.New("session timestamp is required")
	}
	return nil
}

// GetDuration returns the session duration as a time.Duration
func (s *Session) GetDuration() time.Duration {
	return time.Duration(s.Duration) * time.Millisecond
}

// BehavioralMetrics represents aggregate metrics extracted from agent sessions
type BehavioralMetrics struct {
	TotalSessions    int             `json:"total_sessions"`
	SuccessRate      float64         `json:"success_rate"`
	AverageDuration  time.Duration   `json:"average_duration"`
	TotalCost        float64         `json:"total_cost"`
	ToolExecutions   []ToolExecution `json:"tool_executions"`
	BashCommands     []BashCommand   `json:"bash_commands"`
	FileOperations   []FileOperation `json:"file_operations"`
	TokenUsage       TokenUsage      `json:"token_usage"`
	ErrorRate        float64         `json:"error_rate"`
	TotalErrors      int             `json:"total_errors"`
	AgentPerformance map[string]int  `json:"agent_performance"` // Agent name -> success count
}

// Validate checks if the behavioral metrics are valid
func (bm *BehavioralMetrics) Validate() error {
	if bm.TotalSessions < 0 {
		return errors.New("total sessions cannot be negative")
	}
	if bm.SuccessRate < 0 || bm.SuccessRate > 1 {
		return errors.New("success rate must be between 0 and 1")
	}
	if bm.ErrorRate < 0 || bm.ErrorRate > 1 {
		return errors.New("error rate must be between 0 and 1")
	}
	if bm.TotalCost < 0 {
		return errors.New("total cost cannot be negative")
	}

	// Validate tool executions
	for i, te := range bm.ToolExecutions {
		if err := te.Validate(); err != nil {
			return errors.New("invalid tool execution at index " + string(rune(i+'0')) + ": " + err.Error())
		}
	}

	// Validate bash commands
	for i, bc := range bm.BashCommands {
		if err := bc.Validate(); err != nil {
			return errors.New("invalid bash command at index " + string(rune(i+'0')) + ": " + err.Error())
		}
	}

	// Validate file operations
	for i, fo := range bm.FileOperations {
		if err := fo.Validate(); err != nil {
			return errors.New("invalid file operation at index " + string(rune(i+'0')) + ": " + err.Error())
		}
	}

	// Validate token usage
	if err := bm.TokenUsage.Validate(); err != nil {
		return err
	}

	return nil
}

// CalculateTotalCost sums up all token usage costs
func (bm *BehavioralMetrics) CalculateTotalCost() float64 {
	return bm.TokenUsage.CostUSD
}

// AggregateMetrics combines metrics from multiple sessions
func (bm *BehavioralMetrics) AggregateMetrics(sessions []Session) {
	bm.TotalSessions = len(sessions)

	if bm.TotalSessions == 0 {
		return
	}

	successCount := 0
	var totalDuration int64
	totalErrors := 0

	for _, session := range sessions {
		if session.Success {
			successCount++
		}
		totalDuration += session.Duration
		totalErrors += session.ErrorCount

		// Track agent performance
		if bm.AgentPerformance == nil {
			bm.AgentPerformance = make(map[string]int)
		}
		if session.Success {
			bm.AgentPerformance[session.AgentName]++
		}
	}

	bm.SuccessRate = float64(successCount) / float64(bm.TotalSessions)
	bm.AverageDuration = time.Duration(totalDuration/int64(bm.TotalSessions)) * time.Millisecond
	bm.TotalErrors = totalErrors
	if bm.TotalSessions > 0 {
		bm.ErrorRate = float64(totalErrors) / float64(bm.TotalSessions)
	}
}

// ToolExecution represents metrics for a specific tool usage
type ToolExecution struct {
	Name         string        `json:"name"`          // Tool name (e.g., "Read", "Write", "Bash")
	Count        int           `json:"count"`         // Number of times executed
	SuccessRate  float64       `json:"success_rate"`  // Success rate (0.0 to 1.0)
	ErrorRate    float64       `json:"error_rate"`    // Error rate (0.0 to 1.0)
	AvgDuration  time.Duration `json:"avg_duration"`  // Average execution duration
	TotalSuccess int           `json:"total_success"` // Total successful executions
	TotalErrors  int           `json:"total_errors"`  // Total failed executions
}

// Validate checks if the tool execution metrics are valid
func (te *ToolExecution) Validate() error {
	if te.Name == "" {
		return errors.New("tool name is required")
	}
	if te.Count < 0 {
		return errors.New("tool count cannot be negative")
	}
	if te.SuccessRate < 0 || te.SuccessRate > 1 {
		return errors.New("tool success rate must be between 0 and 1")
	}
	if te.ErrorRate < 0 || te.ErrorRate > 1 {
		return errors.New("tool error rate must be between 0 and 1")
	}
	return nil
}

// CalculateRates computes success and error rates from counts
func (te *ToolExecution) CalculateRates() {
	if te.Count > 0 {
		te.SuccessRate = float64(te.TotalSuccess) / float64(te.Count)
		te.ErrorRate = float64(te.TotalErrors) / float64(te.Count)
	} else {
		te.SuccessRate = 0
		te.ErrorRate = 0
	}
}

// BashCommand represents a bash command execution with metrics
type BashCommand struct {
	Command      string        `json:"command"`       // Command string
	ExitCode     int           `json:"exit_code"`     // Exit code (0 = success)
	OutputLength int           `json:"output_length"` // Length of output in bytes
	Duration     time.Duration `json:"duration"`      // Execution duration
	Success      bool          `json:"success"`       // Whether command succeeded
	Timestamp    time.Time     `json:"timestamp"`     // When command was executed
}

// Validate checks if the bash command metrics are valid
func (bc *BashCommand) Validate() error {
	if bc.Command == "" {
		return errors.New("bash command is required")
	}
	if bc.OutputLength < 0 {
		return errors.New("output length cannot be negative")
	}
	return nil
}

// IsSuccess returns true if the command executed successfully (exit code 0)
func (bc *BashCommand) IsSuccess() bool {
	return bc.ExitCode == 0
}

// FileOperation represents a file operation (read, write, edit, delete)
type FileOperation struct {
	Type      string    `json:"type"`       // Operation type: read, write, edit, delete
	Path      string    `json:"path"`       // File path
	SizeBytes int64     `json:"size_bytes"` // File size in bytes
	Success   bool      `json:"success"`    // Whether operation succeeded
	Timestamp time.Time `json:"timestamp"`  // When operation occurred
	Duration  int64     `json:"duration"`   // Operation duration in milliseconds
}

// Validate checks if the file operation metrics are valid
func (fo *FileOperation) Validate() error {
	if fo.Type == "" {
		return errors.New("file operation type is required")
	}
	if fo.Path == "" {
		return errors.New("file path is required")
	}
	if fo.SizeBytes < 0 {
		return errors.New("file size cannot be negative")
	}
	return nil
}

// GetDuration returns the operation duration as a time.Duration
func (fo *FileOperation) GetDuration() time.Duration {
	return time.Duration(fo.Duration) * time.Millisecond
}

// IsWrite returns true if the operation is a write or edit
func (fo *FileOperation) IsWrite() bool {
	return fo.Type == "write" || fo.Type == "edit"
}

// IsRead returns true if the operation is a read
func (fo *FileOperation) IsRead() bool {
	return fo.Type == "read"
}

// TokenUsage represents token consumption and cost metrics
type TokenUsage struct {
	InputTokens  int64   `json:"input_tokens"`  // Total input tokens
	OutputTokens int64   `json:"output_tokens"` // Total output tokens
	CostUSD      float64 `json:"cost_usd"`      // Total cost in USD
	ModelName    string  `json:"model_name"`    // Model used (e.g., "claude-sonnet-4-5")
}

// Validate checks if the token usage metrics are valid
func (tu *TokenUsage) Validate() error {
	if tu.InputTokens < 0 {
		return errors.New("input tokens cannot be negative")
	}
	if tu.OutputTokens < 0 {
		return errors.New("output tokens cannot be negative")
	}
	if tu.CostUSD < 0 {
		return errors.New("cost cannot be negative")
	}
	return nil
}

// TotalTokens returns the sum of input and output tokens
func (tu *TokenUsage) TotalTokens() int64 {
	return tu.InputTokens + tu.OutputTokens
}

// CalculateCost computes cost based on token usage and model rates
// Default rates for claude-sonnet-4-5: $3 per 1M input tokens, $15 per 1M output tokens
func (tu *TokenUsage) CalculateCost() {
	const (
		inputCostPerMillion  = 3.0
		outputCostPerMillion = 15.0
	)

	inputCost := (float64(tu.InputTokens) / 1_000_000) * inputCostPerMillion
	outputCost := (float64(tu.OutputTokens) / 1_000_000) * outputCostPerMillion
	tu.CostUSD = inputCost + outputCost
}

// AggregateMetrics combines multiple BehavioralMetrics into summary statistics
// Returns aggregated data including tool usage, cost, duration, and error rates
func AggregateMetrics(metrics []BehavioralMetrics) map[string]interface{} {
	if len(metrics) == 0 {
		return map[string]interface{}{}
	}

	totalCost := 0.0
	var totalDuration time.Duration
	totalErrors := 0
	toolCounts := make(map[string]int)
	var totalInputTokens int64
	var totalOutputTokens int64
	totalSessions := 0
	agentPerformance := make(map[string]int)

	for _, m := range metrics {
		totalCost += m.CalculateTotalCost()
		totalDuration += m.AverageDuration * time.Duration(m.TotalSessions)
		totalErrors += m.TotalErrors
		totalSessions += m.TotalSessions
		totalInputTokens += m.TokenUsage.InputTokens
		totalOutputTokens += m.TokenUsage.OutputTokens

		// Aggregate tool usage
		for _, tool := range m.ToolExecutions {
			toolCounts[tool.Name] += tool.Count
		}

		// Aggregate agent performance
		for agent, count := range m.AgentPerformance {
			agentPerformance[agent] += count
		}
	}

	avgDuration := time.Duration(0)
	if totalSessions > 0 {
		avgDuration = totalDuration / time.Duration(totalSessions)
	}

	return map[string]interface{}{
		"session_count":       totalSessions,
		"total_cost_usd":      totalCost,
		"avg_cost_usd":        totalCost / float64(len(metrics)),
		"total_duration":      totalDuration,
		"avg_duration":        avgDuration,
		"total_errors":        totalErrors,
		"tool_usage_counts":   toolCounts,
		"total_input_tokens":  totalInputTokens,
		"total_output_tokens": totalOutputTokens,
		"agent_performance":   agentPerformance,
	}
}
