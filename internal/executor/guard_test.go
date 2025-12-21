package executor

import (
	"context"
	"testing"

	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/models"
)

// =============================================================================
// Mode Evaluation Tests
// =============================================================================

func TestGuardModeBlock_HighProbability_ShouldBlock(t *testing.T) {
	cfg := &config.GuardConfig{
		Enabled:              true,
		Mode:                 config.GuardModeBlock,
		ProbabilityThreshold: 0.7,
		ConfidenceThreshold:  0.8,
		MinHistorySessions:   5,
	}

	guard := NewGuardProtocol(mockFailureStore(), cfg)
	if guard == nil {
		t.Fatal("expected non-nil GuardProtocol")
	}

	wave := []models.Task{
		{
			Number:    "1",
			Name:      "Task with high failure probability",
			Files:     []string{"internal/executor/task.go"},
			Prompt:    "Modify critical file with history of failures",
			Agent:     "golang-pro",
			DependsOn: []string{},
		},
	}

	ctx := context.Background()
	results, err := guard.CheckWave(ctx, wave)

	// In block mode with high probability, should return blocking results
	if err != nil {
		t.Fatalf("CheckWave should not return error in block mode, got: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected guard results, got empty slice")
	}

	result := results[0]
	if !result.ShouldBlock {
		t.Error("expected ShouldBlock=true for high probability in block mode")
	}
	if result.Probability < cfg.ProbabilityThreshold {
		t.Errorf("expected probability >= %f, got %f", cfg.ProbabilityThreshold, result.Probability)
	}
}

func TestGuardModeBlock_LowProbability_ShouldNotBlock(t *testing.T) {
	cfg := &config.GuardConfig{
		Enabled:              true,
		Mode:                 config.GuardModeBlock,
		ProbabilityThreshold: 0.7,
		ConfidenceThreshold:  0.8,
		MinHistorySessions:   5,
	}

	guard := NewGuardProtocol(mockSuccessStore(), cfg)
	if guard == nil {
		t.Fatal("expected non-nil GuardProtocol")
	}

	wave := []models.Task{
		{
			Number:    "1",
			Name:      "Task with low failure probability",
			Files:     []string{"docs/README.md"},
			Prompt:    "Update documentation",
			Agent:     "documentation-expert",
			DependsOn: []string{},
		},
	}

	ctx := context.Background()
	results, err := guard.CheckWave(ctx, wave)

	if err != nil {
		t.Fatalf("CheckWave returned unexpected error: %v", err)
	}

	if len(results) == 0 {
		// Low probability might result in no guard results
		return
	}

	result := results[0]
	if result.ShouldBlock {
		t.Errorf("expected ShouldBlock=false for low probability, got probability=%f", result.Probability)
	}
}

func TestGuardModeWarn_HighProbability_ShouldNotBlock(t *testing.T) {
	cfg := &config.GuardConfig{
		Enabled:              true,
		Mode:                 config.GuardModeWarn,
		ProbabilityThreshold: 0.7,
		ConfidenceThreshold:  0.8,
		MinHistorySessions:   5,
	}

	guard := NewGuardProtocol(mockFailureStore(), cfg)
	if guard == nil {
		t.Fatal("expected non-nil GuardProtocol")
	}

	wave := []models.Task{
		{
			Number:    "1",
			Name:      "Task with high failure probability",
			Files:     []string{"internal/executor/task.go"},
			Prompt:    "Modify critical file",
			Agent:     "golang-pro",
			DependsOn: []string{},
		},
	}

	ctx := context.Background()
	results, err := guard.CheckWave(ctx, wave)

	if err != nil {
		t.Fatalf("CheckWave returned unexpected error: %v", err)
	}

	// In warn mode, should never block
	for _, result := range results {
		if result.ShouldBlock {
			t.Error("expected ShouldBlock=false in warn mode, got true")
		}
		if result.Probability >= cfg.ProbabilityThreshold {
			t.Logf("warning generated for task %s with probability %f", result.TaskNumber, result.Probability)
		}
	}
}

func TestGuardModeAdaptive_HighProbHighConf_ShouldBlock(t *testing.T) {
	cfg := &config.GuardConfig{
		Enabled:              true,
		Mode:                 config.GuardModeAdaptive,
		ProbabilityThreshold: 0.7,
		ConfidenceThreshold:  0.85,
		MinHistorySessions:   5,
	}

	guard := NewGuardProtocol(mockHighConfidenceFailureStore(), cfg)
	if guard == nil {
		t.Fatal("expected non-nil GuardProtocol")
	}

	wave := []models.Task{
		{
			Number:    "1",
			Name:      "Task with high probability and high confidence",
			Files:     []string{"internal/executor/task.go"},
			Prompt:    "Modify frequently-failing file",
			Agent:     "golang-pro",
			DependsOn: []string{},
		},
	}

	ctx := context.Background()
	results, err := guard.CheckWave(ctx, wave)

	if err != nil {
		t.Fatalf("CheckWave returned unexpected error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected guard results for high prob + high conf")
	}

	result := results[0]
	if !result.ShouldBlock {
		t.Errorf("expected ShouldBlock=true for high prob (%.2f) and high conf (%.2f) in adaptive mode",
			result.Probability, result.Confidence)
	}
}

func TestGuardModeAdaptive_HighProbLowConf_ShouldNotBlock(t *testing.T) {
	cfg := &config.GuardConfig{
		Enabled:              true,
		Mode:                 config.GuardModeAdaptive,
		ProbabilityThreshold: 0.7,
		ConfidenceThreshold:  0.85,
		MinHistorySessions:   5,
	}

	guard := NewGuardProtocol(mockLowConfidenceFailureStore(), cfg)
	if guard == nil {
		t.Fatal("expected non-nil GuardProtocol")
	}

	wave := []models.Task{
		{
			Number:    "1",
			Name:      "Task with high probability but low confidence",
			Files:     []string{"internal/executor/task.go"},
			Prompt:    "Modify file with limited history",
			Agent:     "golang-pro",
			DependsOn: []string{},
		},
	}

	ctx := context.Background()
	results, err := guard.CheckWave(ctx, wave)

	if err != nil {
		t.Fatalf("CheckWave returned unexpected error: %v", err)
	}

	// In adaptive mode with low confidence, should not block
	for _, result := range results {
		if result.ShouldBlock && result.Confidence < cfg.ConfidenceThreshold {
			t.Errorf("expected ShouldBlock=false for low confidence (%.2f < %.2f), got true",
				result.Confidence, cfg.ConfidenceThreshold)
		}
	}
}

// =============================================================================
// Threshold Boundary Tests
// =============================================================================

func TestProbabilityThreshold_ExactlyAtThreshold(t *testing.T) {
	threshold := 0.75
	cfg := &config.GuardConfig{
		Enabled:              true,
		Mode:                 config.GuardModeBlock,
		ProbabilityThreshold: threshold,
		ConfidenceThreshold:  0.8,
		MinHistorySessions:   5,
	}

	guard := NewGuardProtocol(mockExactThresholdStore(threshold), cfg)
	if guard == nil {
		t.Fatal("expected non-nil GuardProtocol")
	}

	wave := []models.Task{
		{
			Number:    "1",
			Name:      "Task at exact threshold",
			Files:     []string{"internal/executor/task.go"},
			Prompt:    "Modify file",
			Agent:     "golang-pro",
			DependsOn: []string{},
		},
	}

	ctx := context.Background()
	results, err := guard.CheckWave(ctx, wave)

	if err != nil {
		t.Fatalf("CheckWave returned unexpected error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected guard results for exact threshold")
	}

	result := results[0]
	// P >= threshold should block
	if !result.ShouldBlock {
		t.Errorf("expected ShouldBlock=true when probability (%.2f) >= threshold (%.2f)",
			result.Probability, threshold)
	}
}

func TestProbabilityThreshold_SlightlyBelow(t *testing.T) {
	threshold := 0.75
	belowThreshold := threshold - 0.01

	cfg := &config.GuardConfig{
		Enabled:              true,
		Mode:                 config.GuardModeBlock,
		ProbabilityThreshold: threshold,
		ConfidenceThreshold:  0.8,
		MinHistorySessions:   5,
	}

	guard := NewGuardProtocol(mockExactThresholdStore(belowThreshold), cfg)
	if guard == nil {
		t.Fatal("expected non-nil GuardProtocol")
	}

	wave := []models.Task{
		{
			Number:    "1",
			Name:      "Task slightly below threshold",
			Files:     []string{"internal/executor/task.go"},
			Prompt:    "Modify file",
			Agent:     "golang-pro",
			DependsOn: []string{},
		},
	}

	ctx := context.Background()
	results, err := guard.CheckWave(ctx, wave)

	if err != nil {
		t.Fatalf("CheckWave returned unexpected error: %v", err)
	}

	// P < threshold should not block
	for _, result := range results {
		if result.ShouldBlock && result.Probability < threshold {
			t.Errorf("expected ShouldBlock=false when probability (%.2f) < threshold (%.2f)",
				result.Probability, threshold)
		}
	}
}

func TestProbabilityThreshold_SlightlyAbove(t *testing.T) {
	threshold := 0.75
	aboveThreshold := threshold + 0.01

	cfg := &config.GuardConfig{
		Enabled:              true,
		Mode:                 config.GuardModeBlock,
		ProbabilityThreshold: threshold,
		ConfidenceThreshold:  0.8,
		MinHistorySessions:   5,
	}

	guard := NewGuardProtocol(mockExactThresholdStore(aboveThreshold), cfg)
	if guard == nil {
		t.Fatal("expected non-nil GuardProtocol")
	}

	wave := []models.Task{
		{
			Number:    "1",
			Name:      "Task slightly above threshold",
			Files:     []string{"internal/executor/task.go"},
			Prompt:    "Modify file",
			Agent:     "golang-pro",
			DependsOn: []string{},
		},
	}

	ctx := context.Background()
	results, err := guard.CheckWave(ctx, wave)

	if err != nil {
		t.Fatalf("CheckWave returned unexpected error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected guard results for above threshold")
	}

	result := results[0]
	// P > threshold should block
	if !result.ShouldBlock {
		t.Errorf("expected ShouldBlock=true when probability (%.2f) > threshold (%.2f)",
			result.Probability, threshold)
	}
}

// =============================================================================
// Graceful Degradation Tests
// =============================================================================

func TestNewGuardProtocol_NilStore_ReturnsNil(t *testing.T) {
	cfg := &config.GuardConfig{
		Enabled:              true,
		Mode:                 config.GuardModeBlock,
		ProbabilityThreshold: 0.7,
		ConfidenceThreshold:  0.8,
		MinHistorySessions:   5,
	}

	guard := NewGuardProtocol(nil, cfg)
	if guard != nil {
		t.Error("expected nil GuardProtocol when store is nil")
	}
}

func TestNewGuardProtocol_DisabledConfig_ReturnsNil(t *testing.T) {
	cfg := &config.GuardConfig{
		Enabled:              false,
		Mode:                 config.GuardModeBlock,
		ProbabilityThreshold: 0.7,
		ConfidenceThreshold:  0.8,
		MinHistorySessions:   5,
	}

	guard := NewGuardProtocol(mockFailureStore(), cfg)
	if guard != nil {
		t.Error("expected nil GuardProtocol when config.Enabled is false")
	}
}

func TestCheckWave_NilGuard_ReturnsNil(t *testing.T) {
	// Simulate calling CheckWave on nil guard (graceful degradation)
	var guard *GuardProtocol = nil

	wave := []models.Task{
		{Number: "1", Name: "Task", Files: []string{"test.go"}},
	}

	ctx := context.Background()
	results, err := guard.CheckWave(ctx, wave)

	if err != nil {
		t.Errorf("expected nil error for nil guard, got: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results for nil guard, got: %v", results)
	}
}

func TestCheckWave_PredictorError_ContinuesExecution(t *testing.T) {
	cfg := &config.GuardConfig{
		Enabled:              true,
		Mode:                 config.GuardModeBlock,
		ProbabilityThreshold: 0.7,
		ConfidenceThreshold:  0.8,
		MinHistorySessions:   5,
	}

	guard := NewGuardProtocol(mockErrorStore(), cfg)
	if guard == nil {
		t.Fatal("expected non-nil GuardProtocol")
	}

	wave := []models.Task{
		{
			Number:    "1",
			Name:      "Task that triggers predictor error",
			Files:     []string{"internal/executor/task.go"},
			Prompt:    "Modify file",
			Agent:     "golang-pro",
			DependsOn: []string{},
		},
	}

	ctx := context.Background()
	results, err := guard.CheckWave(ctx, wave)

	// Should not return error even if predictor fails
	if err != nil {
		t.Errorf("expected graceful degradation on predictor error, got: %v", err)
	}

	// Results may be empty or contain warning-level results
	for _, result := range results {
		if result.ShouldBlock {
			t.Error("predictor error should not cause blocking")
		}
	}
}

// =============================================================================
// Tool Prediction Tests
// =============================================================================

func TestPredictToolUsage_GoFiles_IncludesBashWriteEdit(t *testing.T) {
	task := models.Task{
		Number:       "1",
		Name:         "Implement Go function",
		Files:        []string{"internal/executor/task.go", "internal/executor/task_test.go"},
		Prompt:       "Add new function and tests",
		Agent:        "golang-pro",
		TestCommands: []string{"go test ./internal/executor/..."},
	}

	tools := predictToolUsage(task)

	expectedTools := map[string]bool{
		"Bash":  true,
		"Write": true,
		"Edit":  true,
	}

	for tool := range expectedTools {
		found := false
		for _, predicted := range tools {
			if predicted == tool {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected tool %q to be predicted for Go file task, tools: %v", tool, tools)
		}
	}
}

func TestPredictToolUsage_TestCommands_IncludesBash(t *testing.T) {
	task := models.Task{
		Number:       "1",
		Name:         "Task with test commands",
		Files:        []string{"main.go"},
		Prompt:       "Implement feature",
		TestCommands: []string{"go test ./...", "go build"},
	}

	tools := predictToolUsage(task)

	hasBash := false
	for _, tool := range tools {
		if tool == "Bash" {
			hasBash = true
			break
		}
	}

	if !hasBash {
		t.Errorf("expected Bash tool for task with test_commands, got: %v", tools)
	}
}

func TestPredictToolUsage_MarkdownFiles_IncludesReadWrite(t *testing.T) {
	task := models.Task{
		Number: "1",
		Name:   "Update documentation",
		Files:  []string{"docs/README.md", "docs/guide.md"},
		Prompt: "Update documentation with new features",
	}

	tools := predictToolUsage(task)

	expectedTools := map[string]bool{
		"Read":  true,
		"Write": true,
	}

	for tool := range expectedTools {
		found := false
		for _, predicted := range tools {
			if predicted == tool {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected tool %q to be predicted for markdown task, tools: %v", tool, tools)
		}
	}
}

func TestPredictToolUsage_EmptyFiles_ReturnsEmpty(t *testing.T) {
	task := models.Task{
		Number: "1",
		Name:   "Task without files",
		Files:  []string{},
		Prompt: "Abstract task",
	}

	tools := predictToolUsage(task)

	// Empty files might still predict some tools based on prompt analysis
	// But should not cause errors
	if tools == nil {
		t.Error("expected non-nil tools slice, got nil")
	}
}

// =============================================================================
// DefaultGuardConfig Test
// =============================================================================

func TestDefaultGuardConfig_Values(t *testing.T) {
	cfg := config.DefaultGuardConfig()

	// Verify default values
	if cfg.Enabled {
		t.Error("expected Enabled=false by default (opt-in feature)")
	}

	if cfg.Mode != config.GuardModeWarn {
		t.Errorf("expected Mode='warn' by default, got %q", cfg.Mode)
	}

	if cfg.ProbabilityThreshold != 0.7 {
		t.Errorf("expected ProbabilityThreshold=0.7, got %f", cfg.ProbabilityThreshold)
	}

	if cfg.ConfidenceThreshold != 0.7 {
		t.Errorf("expected ConfidenceThreshold=0.7, got %f", cfg.ConfidenceThreshold)
	}

	if cfg.MinHistorySessions != 5 {
		t.Errorf("expected MinHistorySessions=5, got %d", cfg.MinHistorySessions)
	}
}

// =============================================================================
// Mock Stores for Testing
// =============================================================================

// mockFailureStore returns a store that predicts high failure probability
func mockFailureStore() FailureStore {
	return &mockStore{
		probability: 0.85,
		confidence:  0.90,
		err:         nil,
	}
}

// mockSuccessStore returns a store that predicts low failure probability
func mockSuccessStore() FailureStore {
	return &mockStore{
		probability: 0.15,
		confidence:  0.90,
		err:         nil,
	}
}

// mockHighConfidenceFailureStore returns high prob with high confidence
func mockHighConfidenceFailureStore() FailureStore {
	return &mockStore{
		probability: 0.88,
		confidence:  0.92,
		err:         nil,
	}
}

// mockLowConfidenceFailureStore returns high prob with low confidence
func mockLowConfidenceFailureStore() FailureStore {
	return &mockStore{
		probability: 0.80,
		confidence:  0.60,
		err:         nil,
	}
}

// mockExactThresholdStore returns exact probability value
func mockExactThresholdStore(prob float64) FailureStore {
	return &mockStore{
		probability: prob,
		confidence:  0.90,
		err:         nil,
	}
}

// mockErrorStore simulates predictor errors
func mockErrorStore() FailureStore {
	return &mockStore{
		probability: 0.0,
		confidence:  0.0,
		err:         &PredictorError{Message: "simulated predictor error"},
	}
}

// mockStore implements FailureStore for testing
type mockStore struct {
	probability float64
	confidence  float64
	err         error
}

func (m *mockStore) PredictFailure(ctx context.Context, task models.Task) (float64, float64, error) {
	if m.err != nil {
		return 0.0, 0.0, m.err
	}
	return m.probability, m.confidence, nil
}

// PredictorError represents a predictor error for testing
type PredictorError struct {
	Message string
}

func (e *PredictorError) Error() string {
	return e.Message
}

// =============================================================================
// Test Helper Interfaces (to be implemented in guard.go)
// =============================================================================

// GuardProtocol represents the guard protocol implementation
// This is a placeholder for testing - actual implementation in guard.go
type GuardProtocol struct {
	store     FailureStore
	cfg       *config.GuardConfig
	predictor FailurePredictor
}

// GuardResult represents the result of a guard check
type GuardResult struct {
	TaskNumber  string
	Probability float64
	Confidence  float64
	ShouldBlock bool
	Warning     string
	Reasoning   string
}

// FailureStore is the interface for accessing historical execution data
type FailureStore interface {
	PredictFailure(ctx context.Context, task models.Task) (probability float64, confidence float64, err error)
}

// FailurePredictor predicts task failure probability
type FailurePredictor interface {
	Predict(ctx context.Context, task models.Task, historicalData []ExecutionRecord) (float64, float64, error)
}

// ExecutionRecord represents a historical task execution
type ExecutionRecord struct {
	TaskName  string
	Agent     string
	Files     []string
	Outcome   string // "success" or "failure"
	Duration  int64
	Timestamp int64
}

// NewGuardProtocol creates a new guard protocol instance
// Returns nil if store is nil or config is disabled
func NewGuardProtocol(store FailureStore, cfg *config.GuardConfig) *GuardProtocol {
	if store == nil || cfg == nil || !cfg.Enabled {
		return nil
	}
	return &GuardProtocol{
		store: store,
		cfg:   cfg,
	}
}

// CheckWave analyzes a wave of tasks and returns guard results
// Returns nil results and nil error if guard is nil (graceful degradation)
func (g *GuardProtocol) CheckWave(ctx context.Context, wave []models.Task) ([]GuardResult, error) {
	if g == nil {
		return nil, nil
	}

	results := make([]GuardResult, 0)
	for _, task := range wave {
		prob, conf, err := g.store.PredictFailure(ctx, task)

		// Graceful degradation on predictor error
		if err != nil {
			continue
		}

		result := GuardResult{
			TaskNumber:  task.Number,
			Probability: prob,
			Confidence:  conf,
			ShouldBlock: g.shouldBlock(prob, conf),
			Warning:     g.generateWarning(prob, conf),
			Reasoning:   g.generateReasoning(task, prob, conf),
		}

		// Only include in results if probability exceeds threshold or mode is warn
		if prob >= g.cfg.ProbabilityThreshold || g.cfg.Mode == config.GuardModeWarn {
			results = append(results, result)
		}
	}

	return results, nil
}

// shouldBlock determines if execution should be blocked based on mode and thresholds
func (g *GuardProtocol) shouldBlock(probability, confidence float64) bool {
	if g.cfg.Mode == config.GuardModeWarn {
		return false
	}

	if g.cfg.Mode == config.GuardModeAdaptive {
		return probability >= g.cfg.ProbabilityThreshold && confidence >= g.cfg.ConfidenceThreshold
	}

	// block mode
	return probability >= g.cfg.ProbabilityThreshold
}

// generateWarning creates a warning message for the task
func (g *GuardProtocol) generateWarning(probability, confidence float64) string {
	if probability < g.cfg.ProbabilityThreshold {
		return ""
	}
	return "High failure probability detected"
}

// generateReasoning creates reasoning text explaining the guard decision
func (g *GuardProtocol) generateReasoning(task models.Task, probability, confidence float64) string {
	return "Analysis based on historical execution patterns"
}

// predictToolUsage predicts which tools will be used for a task
func predictToolUsage(task models.Task) []string {
	tools := make([]string, 0)
	toolSet := make(map[string]bool)

	// Analyze files
	for _, file := range task.Files {
		if hasGoExtension(file) {
			toolSet["Write"] = true
			toolSet["Edit"] = true
			toolSet["Bash"] = true
		} else if hasMarkdownExtension(file) {
			toolSet["Read"] = true
			toolSet["Write"] = true
		}
	}

	// Test commands require Bash
	if len(task.TestCommands) > 0 {
		toolSet["Bash"] = true
	}

	// Convert set to slice
	for tool := range toolSet {
		tools = append(tools, tool)
	}

	return tools
}

// hasGoExtension checks if file has .go extension
func hasGoExtension(file string) bool {
	return len(file) > 3 && file[len(file)-3:] == ".go"
}

// hasMarkdownExtension checks if file has .md extension
func hasMarkdownExtension(file string) bool {
	return len(file) > 3 && file[len(file)-3:] == ".md"
}
