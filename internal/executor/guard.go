package executor

import (
	"context"
	"strings"

	"github.com/harrison/conductor/internal/behavioral"
	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
	"github.com/harrison/conductor/internal/models"
)

// GuardMode defines the GUARD protocol operational mode
type GuardMode string

const (
	// GuardModeBlock blocks tasks when prediction exceeds threshold
	GuardModeBlock GuardMode = "block"

	// GuardModeWarn logs warnings but never blocks execution
	GuardModeWarn GuardMode = "warn"

	// GuardModeAdaptive blocks when both probability AND confidence thresholds are met
	GuardModeAdaptive GuardMode = "adaptive"
)

// GuardConfig defines GUARD protocol configuration
type GuardConfig struct {
	Enabled              bool      // Master switch
	Mode                 GuardMode // Operational mode
	ProbabilityThreshold float64   // Minimum probability to trigger action (0.0-1.0)
	ConfidenceThreshold  float64   // Minimum confidence for adaptive mode (0.0-1.0)
	MinHistorySessions   int       // Minimum sessions required for predictions
}

// GuardResult contains prediction analysis for a single task
type GuardResult struct {
	TaskNumber      string
	Prediction      *behavioral.PredictionResult
	ShouldBlock     bool
	BlockReason     string
	Recommendations []string
}

// GuardProtocol implements the GUARD (Guided Adaptive Risk Detection) protocol.
// It integrates the FailurePredictor from behavioral analytics into wave execution.
type GuardProtocol struct {
	config      GuardConfig
	store       *learning.Store
	predictor   *behavioral.FailurePredictor
	logger      Logger
	initialized bool
}

// NewGuardProtocol creates a new GuardProtocol instance.
// Returns nil if GUARD is disabled or store is nil (graceful degradation).
func NewGuardProtocol(config GuardConfig, store *learning.Store, logger Logger) *GuardProtocol {
	if !config.Enabled {
		return nil
	}

	if store == nil {
		return nil
	}

	return &GuardProtocol{
		config:      config,
		store:       store,
		logger:      logger,
		initialized: false,
	}
}

// Initialize performs lazy initialization of the GUARD protocol.
// Loads behavioral data from store and creates the FailurePredictor.
// Returns nil on error (graceful degradation - GUARD will be disabled).
func (gp *GuardProtocol) Initialize(ctx context.Context) error {
	if gp.initialized {
		return nil
	}

	// Load behavioral data from store
	sessions, metrics, err := gp.loadBehavioralData(ctx)
	if err != nil {
		// Graceful degradation: log error but don't fail execution
		return nil
	}

	// Create predictor with loaded data
	gp.predictor = behavioral.NewFailurePredictor(sessions, metrics)
	gp.initialized = true

	return nil
}

// CheckWave analyzes all tasks in a wave and returns prediction results.
// Returns empty map on error (graceful degradation).
func (gp *GuardProtocol) CheckWave(ctx context.Context, tasks []models.Task) (map[string]*GuardResult, error) {
	if gp == nil {
		return nil, nil
	}

	// Lazy initialization
	if !gp.initialized {
		if err := gp.Initialize(ctx); err != nil {
			return nil, nil
		}
	}

	// If predictor is still nil after initialization, return empty
	if gp.predictor == nil {
		return nil, nil
	}

	results := make(map[string]*GuardResult)

	for _, task := range tasks {
		result := gp.checkTask(ctx, task)
		if result != nil {
			results[task.Number] = result
		}
	}

	return results, nil
}

// checkTask analyzes a single task for failure risk.
// Returns nil on error (graceful degradation).
func (gp *GuardProtocol) checkTask(ctx context.Context, task models.Task) *GuardResult {
	if gp.predictor == nil {
		return nil
	}

	// Infer tool usage from task metadata
	toolUsage := gp.predictToolUsage(task)

	// Create a minimal session for prediction
	session := &behavioral.Session{
		ID:        task.Number,
		Project:   task.SourceFile,
		AgentName: task.Agent,
		Success:   false, // Predict failure
	}

	// Run prediction
	prediction, err := gp.predictor.PredictFailure(session, toolUsage)
	if err != nil {
		// Graceful degradation: return nil on prediction error
		return nil
	}

	// Evaluate whether to block based on mode
	shouldBlock, blockReason := gp.evaluateBlockDecision(prediction)

	return &GuardResult{
		TaskNumber:      task.Number,
		Prediction:      prediction,
		ShouldBlock:     shouldBlock,
		BlockReason:     blockReason,
		Recommendations: prediction.Recommendations,
	}
}

// evaluateBlockDecision applies mode-specific logic to determine if task should be blocked.
func (gp *GuardProtocol) evaluateBlockDecision(prediction *behavioral.PredictionResult) (bool, string) {
	switch gp.config.Mode {
	case GuardModeBlock:
		// Block if probability exceeds threshold
		if prediction.Probability >= gp.config.ProbabilityThreshold {
			return true, "Failure probability exceeds threshold"
		}
		return false, ""

	case GuardModeWarn:
		// Never block, always warn
		return false, ""

	case GuardModeAdaptive:
		// Block if both probability AND confidence exceed thresholds
		if prediction.Probability >= gp.config.ProbabilityThreshold &&
			prediction.Confidence >= gp.config.ConfidenceThreshold {
			return true, "High probability and high confidence failure prediction"
		}
		return false, ""

	default:
		return false, ""
	}
}

// predictToolUsage infers likely tool usage from task metadata.
// Heuristic: analyze task.Files and task.TestCommands to predict tools.
func (gp *GuardProtocol) predictToolUsage(task models.Task) []string {
	toolSet := make(map[string]bool)

	// Files suggest Read/Write/Edit tools
	if len(task.Files) > 0 {
		toolSet["Read"] = true
		toolSet["Write"] = true
	}

	// TestCommands suggest Bash tool
	if len(task.TestCommands) > 0 {
		toolSet["Bash"] = true
	}

	// Integration tasks suggest more complex tool usage
	if task.IsIntegration() {
		toolSet["Read"] = true
		toolSet["Bash"] = true
	}

	// Convert set to slice
	tools := make([]string, 0, len(toolSet))
	for tool := range toolSet {
		tools = append(tools, tool)
	}

	return tools
}

// loadBehavioralData queries the learning store for historical sessions and metrics.
// Returns empty slices on error (graceful degradation).
func (gp *GuardProtocol) loadBehavioralData(ctx context.Context) ([]behavioral.Session, []behavioral.BehavioralMetrics, error) {
	// Query all behavioral sessions from the store
	// This is a simplified implementation - in production, you might want to:
	// 1. Limit to recent sessions (e.g., last 90 days)
	// 2. Filter by project/context
	// 3. Cache results for performance

	// Get recent sessions (limit to 1000 for performance)
	recentSessions, err := gp.store.GetRecentSessions(ctx, "", 1000, 0)
	if err != nil {
		// Graceful degradation
		return nil, nil, nil
	}

	// Convert to behavioral.Session format
	sessions := make([]behavioral.Session, 0, len(recentSessions))
	for _, rs := range recentSessions {
		session := behavioral.Session{
			ID:        rs.TaskName,
			Project:   rs.TaskName, // Use task name as project identifier
			AgentName: rs.Agent,
			Success:   rs.Success,
			Duration:  rs.DurationSecs * 1000, // Convert to milliseconds
			Timestamp: rs.Timestamp,
		}
		sessions = append(sessions, session)
	}

	// Get summary stats for metrics
	summaryStats, err := gp.store.GetSummaryStats(ctx, "")
	if err != nil {
		// Graceful degradation
		return sessions, nil, nil
	}

	// Get tool stats for more detailed metrics
	toolStats, err := gp.store.GetToolStats(ctx, "", 100, 0)
	if err != nil {
		// Continue with what we have
		toolStats = nil
	}

	// Convert to behavioral.BehavioralMetrics format
	metrics := make([]behavioral.BehavioralMetrics, 0)
	if summaryStats != nil {
		// Create tool executions from tool stats
		toolExecs := make([]behavioral.ToolExecution, 0)
		if toolStats != nil {
			for _, ts := range toolStats {
				toolExec := behavioral.ToolExecution{
					Name:        ts.ToolName,
					Count:       ts.CallCount,
					SuccessRate: ts.SuccessRate,
					ErrorRate:   1.0 - ts.SuccessRate,
				}
				toolExec.CalculateRates()
				toolExecs = append(toolExecs, toolExec)
			}
		}

		metric := behavioral.BehavioralMetrics{
			TotalSessions:  summaryStats.TotalSessions,
			SuccessRate:    summaryStats.SuccessRate,
			TotalCost:      summaryStats.TotalCostUSD,
			ToolExecutions: toolExecs,
			TokenUsage: behavioral.TokenUsage{
				InputTokens:  summaryStats.TotalInputTokens,
				OutputTokens: summaryStats.TotalOutputTokens,
				CostUSD:      summaryStats.TotalCostUSD,
			},
		}
		metrics = append(metrics, metric)
	}

	return sessions, metrics, nil
}

// DefaultGuardConfig returns a GuardConfig with sensible defaults
func DefaultGuardConfig() GuardConfig {
	return GuardConfig{
		Enabled:              false, // Disabled by default
		Mode:                 GuardModeWarn,
		ProbabilityThreshold: 0.7, // 70% failure probability
		ConfidenceThreshold:  0.8, // 80% confidence
		MinHistorySessions:   5,   // Require at least 5 historical sessions
	}
}

// LoadGuardConfig loads GUARD configuration from config file
func LoadGuardConfig(cfg *config.Config) GuardConfig {
	// For now, return defaults. In future, add GUARD section to config.yaml
	return DefaultGuardConfig()
}

// FormatRiskLevel returns a human-readable risk level string
func FormatRiskLevel(level string) string {
	switch strings.ToLower(level) {
	case "low":
		return "LOW RISK"
	case "medium":
		return "MEDIUM RISK"
	case "high":
		return "HIGH RISK"
	default:
		return "UNKNOWN RISK"
	}
}

// ========== GuardResultDisplay interface implementation ==========

// GetTaskNumber returns the task number for this prediction
func (gr *GuardResult) GetTaskNumber() string {
	return gr.TaskNumber
}

// GetProbability returns the failure probability (0.0-1.0)
func (gr *GuardResult) GetProbability() float64 {
	if gr.Prediction == nil {
		return 0.0
	}
	return gr.Prediction.Probability
}

// GetConfidence returns the prediction confidence (0.0-1.0)
func (gr *GuardResult) GetConfidence() float64 {
	if gr.Prediction == nil {
		return 0.0
	}
	return gr.Prediction.Confidence
}

// GetRiskLevel returns the risk level string (low, medium, high)
func (gr *GuardResult) GetRiskLevel() string {
	if gr.Prediction == nil {
		return "low"
	}
	return gr.Prediction.RiskLevel
}

// GetShouldBlock returns whether this task should be blocked
func (gr *GuardResult) GetShouldBlock() bool {
	return gr.ShouldBlock
}

// GetBlockReason returns the reason for blocking (empty if not blocked)
func (gr *GuardResult) GetBlockReason() string {
	return gr.BlockReason
}

// GetRecommendations returns improvement suggestions
func (gr *GuardResult) GetRecommendations() []string {
	return gr.Recommendations
}
