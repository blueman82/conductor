package executor

import (
	"context"
	"testing"

	"github.com/harrison/conductor/internal/behavioral"
	"github.com/harrison/conductor/internal/learning"
	"github.com/harrison/conductor/internal/models"
)

func TestNewGuardProtocol_DisabledConfig(t *testing.T) {
	config := GuardConfig{Enabled: false}
	store := &learning.Store{}
	logger := &mockLogger{}

	gp := NewGuardProtocol(config, store, logger)

	if gp != nil {
		t.Error("Expected nil GuardProtocol when disabled")
	}
}

func TestNewGuardProtocol_NilStore(t *testing.T) {
	config := GuardConfig{Enabled: true}
	logger := &mockLogger{}

	gp := NewGuardProtocol(config, nil, logger)

	if gp != nil {
		t.Error("Expected nil GuardProtocol when store is nil")
	}
}

func TestNewGuardProtocol_ValidConfig(t *testing.T) {
	config := GuardConfig{Enabled: true}
	store := &learning.Store{}
	logger := &mockLogger{}

	gp := NewGuardProtocol(config, store, logger)

	if gp == nil {
		t.Fatal("Expected non-nil GuardProtocol")
	}

	if gp.initialized {
		t.Error("Expected initialized to be false initially")
	}

	if gp.config.Enabled != true {
		t.Error("Expected config.Enabled to be true")
	}
}

func TestEvaluateBlockDecision_BlockMode(t *testing.T) {
	config := GuardConfig{
		Enabled:              true,
		Mode:                 GuardModeBlock,
		ProbabilityThreshold: 0.7,
	}
	gp := &GuardProtocol{config: config}

	tests := []struct {
		name        string
		probability float64
		confidence  float64
		wantBlock   bool
		wantReason  string
	}{
		{
			name:        "below threshold",
			probability: 0.6,
			confidence:  0.9,
			wantBlock:   false,
			wantReason:  "",
		},
		{
			name:        "at threshold",
			probability: 0.7,
			confidence:  0.9,
			wantBlock:   true,
			wantReason:  "Failure probability exceeds threshold",
		},
		{
			name:        "above threshold",
			probability: 0.8,
			confidence:  0.5,
			wantBlock:   true,
			wantReason:  "Failure probability exceeds threshold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prediction := &behavioral.PredictionResult{
				Probability: tt.probability,
				Confidence:  tt.confidence,
			}

			shouldBlock, blockReason := gp.evaluateBlockDecision(prediction)

			if shouldBlock != tt.wantBlock {
				t.Errorf("shouldBlock = %v, want %v", shouldBlock, tt.wantBlock)
			}

			if blockReason != tt.wantReason {
				t.Errorf("blockReason = %q, want %q", blockReason, tt.wantReason)
			}
		})
	}
}

func TestEvaluateBlockDecision_WarnMode(t *testing.T) {
	config := GuardConfig{
		Enabled:              true,
		Mode:                 GuardModeWarn,
		ProbabilityThreshold: 0.7,
	}
	gp := &GuardProtocol{config: config}

	tests := []struct {
		name        string
		probability float64
		confidence  float64
	}{
		{"low probability", 0.3, 0.9},
		{"high probability", 0.9, 0.9},
		{"at threshold", 0.7, 0.9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prediction := &behavioral.PredictionResult{
				Probability: tt.probability,
				Confidence:  tt.confidence,
			}

			shouldBlock, blockReason := gp.evaluateBlockDecision(prediction)

			if shouldBlock {
				t.Error("Expected shouldBlock to be false in warn mode")
			}

			if blockReason != "" {
				t.Errorf("Expected empty blockReason in warn mode, got %q", blockReason)
			}
		})
	}
}

func TestEvaluateBlockDecision_AdaptiveMode(t *testing.T) {
	config := GuardConfig{
		Enabled:              true,
		Mode:                 GuardModeAdaptive,
		ProbabilityThreshold: 0.7,
		ConfidenceThreshold:  0.8,
	}
	gp := &GuardProtocol{config: config}

	tests := []struct {
		name        string
		probability float64
		confidence  float64
		wantBlock   bool
		wantReason  string
	}{
		{
			name:        "low probability, high confidence",
			probability: 0.5,
			confidence:  0.9,
			wantBlock:   false,
			wantReason:  "",
		},
		{
			name:        "high probability, low confidence",
			probability: 0.8,
			confidence:  0.6,
			wantBlock:   false,
			wantReason:  "",
		},
		{
			name:        "high probability, high confidence",
			probability: 0.8,
			confidence:  0.9,
			wantBlock:   true,
			wantReason:  "High probability and high confidence failure prediction",
		},
		{
			name:        "at both thresholds",
			probability: 0.7,
			confidence:  0.8,
			wantBlock:   true,
			wantReason:  "High probability and high confidence failure prediction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prediction := &behavioral.PredictionResult{
				Probability: tt.probability,
				Confidence:  tt.confidence,
			}

			shouldBlock, blockReason := gp.evaluateBlockDecision(prediction)

			if shouldBlock != tt.wantBlock {
				t.Errorf("shouldBlock = %v, want %v", shouldBlock, tt.wantBlock)
			}

			if blockReason != tt.wantReason {
				t.Errorf("blockReason = %q, want %q", blockReason, tt.wantReason)
			}
		})
	}
}

func TestPredictToolUsage(t *testing.T) {
	gp := &GuardProtocol{}

	tests := []struct {
		name     string
		task     models.Task
		wantAny  []string // Any of these tools should be present
		wantNone []string // None of these tools should be present
	}{
		{
			name: "task with files",
			task: models.Task{
				Files: []string{"main.go", "test.go"},
			},
			wantAny: []string{"Read", "Write"},
		},
		{
			name: "task with test commands",
			task: models.Task{
				TestCommands: []string{"go test"},
			},
			wantAny: []string{"Bash"},
		},
		{
			name: "integration task",
			task: models.Task{
				Type: "integration",
			},
			wantAny: []string{"Read", "Bash"},
		},
		{
			name: "task with files and tests",
			task: models.Task{
				Files:        []string{"main.go"},
				TestCommands: []string{"go test"},
			},
			wantAny: []string{"Read", "Write", "Bash"},
		},
		{
			name:     "empty task",
			task:     models.Task{},
			wantNone: []string{"Read", "Write", "Bash"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tools := gp.predictToolUsage(tt.task)

			// Convert to map for easy lookup
			toolMap := make(map[string]bool)
			for _, tool := range tools {
				toolMap[tool] = true
			}

			// Check that expected tools are present
			for _, expectedTool := range tt.wantAny {
				if !toolMap[expectedTool] {
					t.Errorf("Expected tool %q to be predicted, but it wasn't. Got: %v", expectedTool, tools)
				}
			}

			// Check that unwanted tools are absent
			for _, unwantedTool := range tt.wantNone {
				if toolMap[unwantedTool] {
					t.Errorf("Expected tool %q NOT to be predicted, but it was. Got: %v", unwantedTool, tools)
				}
			}
		})
	}
}

func TestCheckWave_NilGuardProtocol(t *testing.T) {
	var gp *GuardProtocol
	ctx := context.Background()
	tasks := []models.Task{
		{Number: "1", Name: "Test Task"},
	}

	results, err := gp.CheckWave(ctx, tasks)

	if err != nil {
		t.Errorf("Expected nil error for nil GuardProtocol, got %v", err)
	}

	if results != nil {
		t.Error("Expected nil results for nil GuardProtocol")
	}
}

func TestDefaultGuardConfig(t *testing.T) {
	config := DefaultGuardConfig()

	if config.Enabled {
		t.Error("Expected Enabled to be false by default")
	}

	if config.Mode != GuardModeWarn {
		t.Errorf("Expected Mode to be %q, got %q", GuardModeWarn, config.Mode)
	}

	if config.ProbabilityThreshold != 0.7 {
		t.Errorf("Expected ProbabilityThreshold to be 0.7, got %f", config.ProbabilityThreshold)
	}

	if config.ConfidenceThreshold != 0.8 {
		t.Errorf("Expected ConfidenceThreshold to be 0.8, got %f", config.ConfidenceThreshold)
	}

	if config.MinHistorySessions != 5 {
		t.Errorf("Expected MinHistorySessions to be 5, got %d", config.MinHistorySessions)
	}
}

func TestFormatRiskLevel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"low", "LOW RISK"},
		{"Low", "LOW RISK"},
		{"LOW", "LOW RISK"},
		{"medium", "MEDIUM RISK"},
		{"Medium", "MEDIUM RISK"},
		{"high", "HIGH RISK"},
		{"High", "HIGH RISK"},
		{"unknown", "UNKNOWN RISK"},
		{"", "UNKNOWN RISK"},
		{"invalid", "UNKNOWN RISK"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := FormatRiskLevel(tt.input)
			if got != tt.want {
				t.Errorf("FormatRiskLevel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGuardResult_Structure(t *testing.T) {
	// Test that GuardResult can be created and accessed
	result := &GuardResult{
		TaskNumber: "1",
		Prediction: &behavioral.PredictionResult{
			Probability: 0.75,
			Confidence:  0.85,
			RiskLevel:   "high",
			Explanation: "Test explanation",
		},
		ShouldBlock:     true,
		BlockReason:     "Test block reason",
		Recommendations: []string{"Test recommendation 1", "Test recommendation 2"},
	}

	if result.TaskNumber != "1" {
		t.Errorf("Expected TaskNumber to be '1', got %q", result.TaskNumber)
	}

	if result.Prediction.Probability != 0.75 {
		t.Errorf("Expected Probability to be 0.75, got %f", result.Prediction.Probability)
	}

	if !result.ShouldBlock {
		t.Error("Expected ShouldBlock to be true")
	}

	if len(result.Recommendations) != 2 {
		t.Errorf("Expected 2 recommendations, got %d", len(result.Recommendations))
	}
}

func TestGuardMode_Constants(t *testing.T) {
	// Verify mode constants are defined correctly
	if GuardModeBlock != "block" {
		t.Errorf("GuardModeBlock should be 'block', got %q", GuardModeBlock)
	}

	if GuardModeWarn != "warn" {
		t.Errorf("GuardModeWarn should be 'warn', got %q", GuardModeWarn)
	}

	if GuardModeAdaptive != "adaptive" {
		t.Errorf("GuardModeAdaptive should be 'adaptive', got %q", GuardModeAdaptive)
	}
}
