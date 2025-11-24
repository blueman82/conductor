package logger

import (
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestNewColorScheme(t *testing.T) {
	scheme := newColorScheme()

	if scheme == nil {
		t.Fatal("Expected non-nil color scheme")
	}

	if scheme.success == nil {
		t.Error("Expected success color to be initialized")
	}
	if scheme.fail == nil {
		t.Error("Expected fail color to be initialized")
	}
	if scheme.warn == nil {
		t.Error("Expected warn color to be initialized")
	}
	if scheme.label == nil {
		t.Error("Expected label color to be initialized")
	}
	if scheme.value == nil {
		t.Error("Expected value color to be initialized")
	}
}

func TestFormatColorizedMetric(t *testing.T) {
	scheme := newColorScheme()

	tests := []struct {
		name  string
		label string
		value interface{}
	}{
		{"integer value", "tools", 5},
		{"float value", "cost", 0.1234},
		{"string value", "type", "test"},
		{"zero value", "count", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatColorizedMetric(tt.label, tt.value, scheme)

			if result == "" {
				t.Error("Expected non-empty result")
			}

			// Result should contain the label
			if !strings.Contains(result, tt.label) {
				t.Errorf("Expected result to contain label %q, got %q", tt.label, result)
			}

			// Result should be in format "label: value" (plus ANSI codes)
			if !strings.Contains(result, ":") {
				t.Errorf("Expected result to contain colon separator, got %q", result)
			}
		})
	}
}

func TestFormatColorizedBehavioralMetrics_EmptyMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
	}{
		{"nil metadata", nil},
		{"empty metadata", map[string]interface{}{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatColorizedBehavioralMetrics(tt.metadata)
			if result != "" {
				t.Errorf("Expected empty string for %s, got %q", tt.name, result)
			}
		})
	}
}

func TestFormatColorizedBehavioralMetrics_ToolCount(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		expected string
	}{
		{
			"tool_count as int",
			map[string]interface{}{"tool_count": 5},
			"tools",
		},
		{
			"tool_count as float64",
			map[string]interface{}{"tool_count": 5.0},
			"tools",
		},
		{
			"tool_count zero",
			map[string]interface{}{"tool_count": 0},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatColorizedBehavioralMetrics(tt.metadata)

			if tt.expected == "" {
				if result != "" {
					t.Errorf("Expected empty result for zero value, got %q", result)
				}
			} else {
				if !strings.Contains(result, tt.expected) {
					t.Errorf("Expected result to contain %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestFormatColorizedBehavioralMetrics_BashCount(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		expected string
	}{
		{
			"bash_count as int",
			map[string]interface{}{"bash_count": 3},
			"bash",
		},
		{
			"bash_count as float64",
			map[string]interface{}{"bash_count": 3.0},
			"bash",
		},
		{
			"bash_count zero",
			map[string]interface{}{"bash_count": 0},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatColorizedBehavioralMetrics(tt.metadata)

			if tt.expected == "" {
				if result != "" {
					t.Errorf("Expected empty result for zero value, got %q", result)
				}
			} else {
				if !strings.Contains(result, tt.expected) {
					t.Errorf("Expected result to contain %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestFormatColorizedBehavioralMetrics_FileOperations(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		expected string
	}{
		{
			"file_operations as int",
			map[string]interface{}{"file_operations": 7},
			"files",
		},
		{
			"file_operations as float64",
			map[string]interface{}{"file_operations": 7.0},
			"files",
		},
		{
			"file_operations zero",
			map[string]interface{}{"file_operations": 0},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatColorizedBehavioralMetrics(tt.metadata)

			if tt.expected == "" {
				if result != "" {
					t.Errorf("Expected empty result for zero value, got %q", result)
				}
			} else {
				if !strings.Contains(result, tt.expected) {
					t.Errorf("Expected result to contain %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestFormatColorizedBehavioralMetrics_Cost(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		expected string
	}{
		{
			"cost as float64",
			map[string]interface{}{"cost": 0.1234},
			"cost",
		},
		{
			"cost zero",
			map[string]interface{}{"cost": 0.0},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatColorizedBehavioralMetrics(tt.metadata)

			if tt.expected == "" {
				if result != "" {
					t.Errorf("Expected empty result for zero value, got %q", result)
				}
			} else {
				if !strings.Contains(result, tt.expected) {
					t.Errorf("Expected result to contain %q, got %q", tt.expected, result)
				}
				// Check for dollar sign
				if !strings.Contains(result, "$") {
					t.Errorf("Expected result to contain $ for cost, got %q", result)
				}
			}
		})
	}
}

func TestFormatColorizedBehavioralMetrics_AllMetrics(t *testing.T) {
	metadata := map[string]interface{}{
		"tool_count":      5,
		"bash_count":      3,
		"file_operations": 7,
		"cost":            0.1234,
	}

	result := formatColorizedBehavioralMetrics(metadata)

	expected := []string{"tools", "bash", "files", "cost", "$"}
	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("Expected result to contain %q, got %q", exp, result)
		}
	}

	// Check for comma separators (should have 3 commas for 4 metrics)
	commaCount := strings.Count(result, ",")
	if commaCount != 3 {
		t.Errorf("Expected 3 comma separators, got %d in %q", commaCount, result)
	}
}

func TestFormatColorizedBehavioralMetrics_MixedTypes(t *testing.T) {
	metadata := map[string]interface{}{
		"tool_count":      5.0,   // float64
		"bash_count":      3,     // int
		"file_operations": 7.0,   // float64
		"cost":            0.1234, // float64
	}

	result := formatColorizedBehavioralMetrics(metadata)

	expected := []string{"tools", "bash", "files", "cost"}
	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("Expected result to contain %q, got %q", exp, result)
		}
	}
}

func TestFormatColorizedBehavioralMetrics_PartialData(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		expected []string
		notExp   []string
	}{
		{
			"only tool_count",
			map[string]interface{}{"tool_count": 5},
			[]string{"tools"},
			[]string{"bash", "files", "cost"},
		},
		{
			"only cost",
			map[string]interface{}{"cost": 0.5},
			[]string{"cost", "$"},
			[]string{"tools", "bash", "files"},
		},
		{
			"tools and bash",
			map[string]interface{}{"tool_count": 5, "bash_count": 3},
			[]string{"tools", "bash"},
			[]string{"files", "cost"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatColorizedBehavioralMetrics(tt.metadata)

			for _, exp := range tt.expected {
				if !strings.Contains(result, exp) {
					t.Errorf("Expected result to contain %q, got %q", exp, result)
				}
			}

			for _, notExp := range tt.notExp {
				if strings.Contains(result, notExp) {
					t.Errorf("Expected result NOT to contain %q, got %q", notExp, result)
				}
			}
		})
	}
}

func TestFormatColorizedBehavioralMetrics_ColorCodes(t *testing.T) {
	// Enable colors for this test
	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	metadata := map[string]interface{}{
		"tool_count": 5,
	}

	result := formatColorizedBehavioralMetrics(metadata)

	// Result should contain ANSI color codes
	// Color codes start with ESC[ which is \x1b[
	if !strings.Contains(result, "\x1b[") {
		t.Errorf("Expected result to contain ANSI color codes, got %q", result)
	}
}

func TestColorScheme_RedForFailures(t *testing.T) {
	scheme := newColorScheme()

	// Verify red color is assigned for failures
	if scheme.fail == nil {
		t.Fatal("Expected fail color to be initialized")
	}

	// Format a failure metric and verify red color code is present
	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	result := scheme.fail.Sprint("error")
	// Red ANSI code is \x1b[31m
	if !strings.Contains(result, "\x1b[31m") {
		t.Errorf("Expected red ANSI code in failure output, got %q", result)
	}
}

func TestColorScheme_GreenForSuccess(t *testing.T) {
	scheme := newColorScheme()

	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	result := scheme.success.Sprint("success")
	// Green ANSI code is \x1b[32m
	if !strings.Contains(result, "\x1b[32m") {
		t.Errorf("Expected green ANSI code in success output, got %q", result)
	}
}

func TestColorScheme_YellowForWarnings(t *testing.T) {
	scheme := newColorScheme()

	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	result := scheme.warn.Sprint("warning")
	// Yellow ANSI code is \x1b[33m
	if !strings.Contains(result, "\x1b[33m") {
		t.Errorf("Expected yellow ANSI code in warning output, got %q", result)
	}
}

func TestColorScheme_CyanForLabels(t *testing.T) {
	scheme := newColorScheme()

	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	result := scheme.label.Sprint("label")
	// Cyan ANSI code is \x1b[36m
	if !strings.Contains(result, "\x1b[36m") {
		t.Errorf("Expected cyan ANSI code in label output, got %q", result)
	}
}

func TestColorScheme_DisabledWhenNoColor(t *testing.T) {
	// Disable colors
	oldNoColor := color.NoColor
	color.NoColor = true
	defer func() { color.NoColor = oldNoColor }()

	metadata := map[string]interface{}{
		"tool_count": 5,
		"bash_count": 3,
	}

	result := formatColorizedBehavioralMetrics(metadata)

	// Result should NOT contain ANSI color codes when NoColor is true
	if strings.Contains(result, "\x1b[") {
		t.Errorf("Expected no ANSI color codes when NoColor=true, got %q", result)
	}

	// But should still contain the content
	if !strings.Contains(result, "tools") || !strings.Contains(result, "bash") {
		t.Errorf("Expected content to be present even without colors, got %q", result)
	}
}

func TestFormatColorizedBehavioralMetrics_ErrorCount(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		expected string
	}{
		{
			"error_count as int",
			map[string]interface{}{"error_count": 2},
			"errors",
		},
		{
			"error_count as float64",
			map[string]interface{}{"error_count": 2.0},
			"errors",
		},
		{
			"error_count zero",
			map[string]interface{}{"error_count": 0},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatColorizedBehavioralMetrics(tt.metadata)

			if tt.expected == "" {
				if result != "" {
					t.Errorf("Expected empty result for zero value, got %q", result)
				}
			} else {
				if !strings.Contains(result, tt.expected) {
					t.Errorf("Expected result to contain %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestFormatColorizedBehavioralMetrics_FailedCommands(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		expected string
	}{
		{
			"failed_commands as int",
			map[string]interface{}{"failed_commands": 3},
			"failed",
		},
		{
			"failed_commands as float64",
			map[string]interface{}{"failed_commands": 3.0},
			"failed",
		},
		{
			"failed_commands zero",
			map[string]interface{}{"failed_commands": 0},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatColorizedBehavioralMetrics(tt.metadata)

			if tt.expected == "" {
				if result != "" {
					t.Errorf("Expected empty result for zero value, got %q", result)
				}
			} else {
				if !strings.Contains(result, tt.expected) {
					t.Errorf("Expected result to contain %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestFormatColorizedBehavioralMetrics_ErrorsRedColor(t *testing.T) {
	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	metadata := map[string]interface{}{
		"error_count": 5,
	}

	result := formatColorizedBehavioralMetrics(metadata)

	// Verify red ANSI code is present for errors
	if !strings.Contains(result, "\x1b[31m") {
		t.Errorf("Expected red ANSI code for errors, got %q", result)
	}

	if !strings.Contains(result, "errors") {
		t.Errorf("Expected 'errors' label in output, got %q", result)
	}
}

func TestFormatColorizedBehavioralMetrics_WarningYellowColor(t *testing.T) {
	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	metadata := map[string]interface{}{
		"cost": 0.15, // High cost > 0.10
	}

	result := formatColorizedBehavioralMetrics(metadata)

	// Verify yellow ANSI code is present for warnings
	if !strings.Contains(result, "\x1b[33m") {
		t.Errorf("Expected yellow ANSI code for high cost warning, got %q", result)
	}

	if !strings.Contains(result, "cost") {
		t.Errorf("Expected 'cost' label in output, got %q", result)
	}
}

func TestFormatColorizedBehavioralMetrics_SuccessGreenColor(t *testing.T) {
	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	metadata := map[string]interface{}{
		"tool_count":      5,
		"file_operations": 3,
	}

	result := formatColorizedBehavioralMetrics(metadata)

	// Verify green ANSI code is present for success metrics
	if !strings.Contains(result, "\x1b[32m") {
		t.Errorf("Expected green ANSI code for success metrics, got %q", result)
	}

	if !strings.Contains(result, "tools") || !strings.Contains(result, "files") {
		t.Errorf("Expected success metric labels in output, got %q", result)
	}
}
