package pattern

import (
	"testing"
)

func TestDefaultPatternConfig(t *testing.T) {
	cfg := DefaultPatternConfig()

	// Verify disabled by default
	if cfg.Enabled {
		t.Error("expected pattern intelligence to be disabled by default")
	}

	// Verify default mode
	if cfg.Mode != PatternModeWarn {
		t.Errorf("expected default mode to be 'warn', got %q", cfg.Mode)
	}

	// Verify threshold defaults
	if cfg.SimilarityThreshold != 0.8 {
		t.Errorf("expected similarity_threshold 0.8, got %f", cfg.SimilarityThreshold)
	}
	if cfg.DuplicateThreshold != 0.9 {
		t.Errorf("expected duplicate_threshold 0.9, got %f", cfg.DuplicateThreshold)
	}
	if cfg.MinConfidence != 0.7 {
		t.Errorf("expected min_confidence 0.7, got %f", cfg.MinConfidence)
	}

	// Verify feature defaults
	if !cfg.EnableSTOP {
		t.Error("expected enable_stop to be true by default")
	}
	if !cfg.EnableDuplicateDetection {
		t.Error("expected enable_duplicate_detection to be true by default")
	}
	if !cfg.InjectIntoPrompt {
		t.Error("expected inject_into_prompt to be true by default")
	}

	// Verify limits
	if cfg.MaxPatternsPerTask != 5 {
		t.Errorf("expected max_patterns_per_task 5, got %d", cfg.MaxPatternsPerTask)
	}
	if cfg.MaxRelatedFiles != 10 {
		t.Errorf("expected max_related_files 10, got %d", cfg.MaxRelatedFiles)
	}
	if cfg.CacheTTLSeconds != 3600 {
		t.Errorf("expected cache_ttl_seconds 3600, got %d", cfg.CacheTTLSeconds)
	}
}

func TestPatternConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  PatternConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "disabled config always valid",
			config:  PatternConfig{Enabled: false},
			wantErr: false,
		},
		{
			name: "valid enabled config",
			config: PatternConfig{
				Enabled:                  true,
				Mode:                     PatternModeWarn,
				SimilarityThreshold:      0.8,
				DuplicateThreshold:       0.9,
				MinConfidence:            0.7,
				EnableSTOP:               true,
				EnableDuplicateDetection: true,
				MaxPatternsPerTask:       5,
				MaxRelatedFiles:          10,
				CacheTTLSeconds:          3600,
			},
			wantErr: false,
		},
		{
			name: "invalid similarity_threshold negative",
			config: PatternConfig{
				Enabled:             true,
				Mode:                PatternModeWarn,
				SimilarityThreshold: -0.1,
			},
			wantErr: true,
			errMsg:  "similarity_threshold",
		},
		{
			name: "invalid similarity_threshold over 1",
			config: PatternConfig{
				Enabled:             true,
				Mode:                PatternModeWarn,
				SimilarityThreshold: 1.1,
			},
			wantErr: true,
			errMsg:  "similarity_threshold",
		},
		{
			name: "invalid duplicate_threshold negative",
			config: PatternConfig{
				Enabled:             true,
				Mode:                PatternModeWarn,
				SimilarityThreshold: 0.8,
				DuplicateThreshold:  -0.1,
			},
			wantErr: true,
			errMsg:  "duplicate_threshold",
		},
		{
			name: "invalid min_confidence over 1",
			config: PatternConfig{
				Enabled:             true,
				Mode:                PatternModeWarn,
				SimilarityThreshold: 0.8,
				DuplicateThreshold:  0.9,
				MinConfidence:       1.5,
			},
			wantErr: true,
			errMsg:  "min_confidence",
		},
		{
			name: "invalid max_patterns_per_task negative",
			config: PatternConfig{
				Enabled:             true,
				Mode:                PatternModeWarn,
				SimilarityThreshold: 0.8,
				DuplicateThreshold:  0.9,
				MinConfidence:       0.7,
				MaxPatternsPerTask:  -1,
			},
			wantErr: true,
			errMsg:  "max_patterns_per_task",
		},
		{
			name: "invalid max_related_files negative",
			config: PatternConfig{
				Enabled:             true,
				Mode:                PatternModeWarn,
				SimilarityThreshold: 0.8,
				DuplicateThreshold:  0.9,
				MinConfidence:       0.7,
				MaxPatternsPerTask:  5,
				MaxRelatedFiles:     -1,
			},
			wantErr: true,
			errMsg:  "max_related_files",
		},
		{
			name: "invalid cache_ttl_seconds negative",
			config: PatternConfig{
				Enabled:             true,
				Mode:                PatternModeWarn,
				SimilarityThreshold: 0.8,
				DuplicateThreshold:  0.9,
				MinConfidence:       0.7,
				MaxPatternsPerTask:  5,
				MaxRelatedFiles:     10,
				CacheTTLSeconds:     -1,
			},
			wantErr: true,
			errMsg:  "cache_ttl_seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPatternConfig_ShouldBlock(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		mode     PatternMode
		expected bool
	}{
		{"disabled", false, PatternModeBlock, false},
		{"enabled block", true, PatternModeBlock, true},
		{"enabled warn", true, PatternModeWarn, false},
		{"enabled suggest", true, PatternModeSuggest, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := PatternConfig{Enabled: tt.enabled, Mode: tt.mode}
			if got := cfg.ShouldBlock(); got != tt.expected {
				t.Errorf("ShouldBlock() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPatternConfig_ShouldWarn(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		mode     PatternMode
		expected bool
	}{
		{"disabled", false, PatternModeWarn, false},
		{"enabled block", true, PatternModeBlock, true},
		{"enabled warn", true, PatternModeWarn, true},
		{"enabled suggest", true, PatternModeSuggest, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := PatternConfig{Enabled: tt.enabled, Mode: tt.mode}
			if got := cfg.ShouldWarn(); got != tt.expected {
				t.Errorf("ShouldWarn() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPatternConfig_ShouldInject(t *testing.T) {
	tests := []struct {
		name           string
		enabled        bool
		injectIntoPrompt bool
		expected       bool
	}{
		{"disabled no inject", false, false, false},
		{"disabled with inject", false, true, false},
		{"enabled no inject", true, false, false},
		{"enabled with inject", true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := PatternConfig{Enabled: tt.enabled, InjectIntoPrompt: tt.injectIntoPrompt}
			if got := cfg.ShouldInject(); got != tt.expected {
				t.Errorf("ShouldInject() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfigError_Error(t *testing.T) {
	err := &ConfigError{
		Field:   "similarity_threshold",
		Message: "must be between 0 and 1",
		Value:   1.5,
	}
	expected := "pattern.similarity_threshold: must be between 0 and 1"
	if err.Error() != expected {
		t.Errorf("ConfigError.Error() = %q, want %q", err.Error(), expected)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
