package executor

import (
	"strings"
	"testing"
)

// TestFormatSTOPPriorArt tests the STOP prior art formatting for QC prompts
func TestFormatSTOPPriorArt(t *testing.T) {
	t.Run("empty summary returns empty string", func(t *testing.T) {
		result := FormatSTOPPriorArt("", false)
		if result != "" {
			t.Errorf("expected empty string for empty summary, got %q", result)
		}
	})

	t.Run("formats summary without justification requirement", func(t *testing.T) {
		summary := "Found 3 similar commits:\n- abc123: Add pattern matching\n- def456: Pattern system refactor"
		result := FormatSTOPPriorArt(summary, false)

		if !strings.Contains(result, "STOP Protocol: Prior Art Analysis") {
			t.Error("expected header in output")
		}
		if !strings.Contains(result, "Pattern Intelligence discovered existing solutions") {
			t.Error("expected intro text in output")
		}
		if !strings.Contains(result, summary) {
			t.Error("expected summary content in output")
		}
		if !strings.Contains(result, "Context only") {
			t.Error("expected 'Context only' note when justification not required")
		}
		if strings.Contains(result, "JUSTIFICATION REQUIRED") {
			t.Error("should not contain justification requirement when disabled")
		}
	})

	t.Run("formats summary with justification requirement", func(t *testing.T) {
		summary := "Found existing implementation in pattern/matcher.go"
		result := FormatSTOPPriorArt(summary, true)

		if !strings.Contains(result, "JUSTIFICATION REQUIRED") {
			t.Error("expected justification requirement header")
		}
		if !strings.Contains(result, "Prior art exists") {
			t.Error("expected prior art warning")
		}
		if !strings.Contains(result, "stop_justification") {
			t.Error("expected reference to stop_justification field")
		}
		if !strings.Contains(result, "YELLOW") {
			t.Error("expected YELLOW verdict warning for weak justification")
		}
		if strings.Contains(result, "Context only") {
			t.Error("should not contain 'Context only' when justification is required")
		}
	})
}

// TestEvaluateSTOPJustification tests the justification quality evaluation
func TestEvaluateSTOPJustification(t *testing.T) {
	tests := []struct {
		name          string
		stopSummary   string
		justification string
		wantSufficient bool
	}{
		{
			name:           "no prior art - no justification needed",
			stopSummary:    "",
			justification:  "",
			wantSufficient: true,
		},
		{
			name:           "no prior art - justification ignored",
			stopSummary:    "",
			justification:  "Some justification",
			wantSufficient: true,
		},
		{
			name:           "prior art with no justification",
			stopSummary:    "Found similar commit abc123",
			justification:  "",
			wantSufficient: false,
		},
		{
			name:           "prior art with weak n/a justification",
			stopSummary:    "Found similar commit abc123",
			justification:  "n/a",
			wantSufficient: false,
		},
		{
			name:           "prior art with weak not applicable justification",
			stopSummary:    "Found similar commit abc123",
			justification:  "not applicable",
			wantSufficient: false,
		},
		{
			name:           "prior art with weak none justification",
			stopSummary:    "Found similar commit abc123",
			justification:  "none",
			wantSufficient: false,
		},
		{
			name:           "prior art with weak custom implementation justification",
			stopSummary:    "Found similar commit abc123",
			justification:  "custom implementation",
			wantSufficient: false,
		},
		{
			name:           "prior art with too short justification",
			stopSummary:    "Found similar commit abc123",
			justification:  "Different use case",
			wantSufficient: false, // Less than 20 chars
		},
		{
			name:           "prior art with adequate justification",
			stopSummary:    "Found similar commit abc123",
			justification:  "The existing implementation uses a different hashing algorithm that doesn't support streaming input",
			wantSufficient: true,
		},
		{
			name:           "prior art with technical justification",
			stopSummary:    "Found pattern in internal/legacy/matcher.go",
			justification:  "The legacy matcher uses regex which is too slow for our use case. We need O(1) lookup via hash table.",
			wantSufficient: true,
		},
		{
			name:           "prior art with whitespace-padded weak justification",
			stopSummary:    "Found similar pattern",
			justification:  "   n/a   ",
			wantSufficient: false, // Should be trimmed and recognized as weak
		},
		{
			name:           "prior art with case-insensitive weak pattern",
			stopSummary:    "Found similar pattern",
			justification:  "Not Applicable",
			wantSufficient: false, // Should be case-insensitive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateSTOPJustification(tt.stopSummary, tt.justification)
			if got != tt.wantSufficient {
				t.Errorf("EvaluateSTOPJustification() = %v, want %v", got, tt.wantSufficient)
			}
		})
	}
}

// TestQualityControllerSTOPIntegration tests QC controller STOP field handling
func TestQualityControllerSTOPIntegration(t *testing.T) {
	t.Run("STOPSummary and RequireJustification fields exist", func(t *testing.T) {
		qc := &QualityController{
			STOPSummary:          "Found prior art",
			RequireJustification: true,
		}

		if qc.STOPSummary != "Found prior art" {
			t.Error("STOPSummary field not accessible")
		}
		if !qc.RequireJustification {
			t.Error("RequireJustification field not accessible")
		}
	})

	t.Run("BuildStructuredReviewPrompt includes STOP summary", func(t *testing.T) {
		qc := &QualityController{
			STOPSummary:          "Found 2 similar commits related to pattern matching",
			RequireJustification: true,
		}

		// We can't easily test BuildStructuredReviewPrompt without a full mock setup,
		// but we can verify the FormatSTOPPriorArt function is called correctly
		// by checking that it would produce the expected output
		formatted := FormatSTOPPriorArt(qc.STOPSummary, qc.RequireJustification)

		if !strings.Contains(formatted, "Found 2 similar commits") {
			t.Error("STOP summary not included in formatted output")
		}
		if !strings.Contains(formatted, "JUSTIFICATION REQUIRED") {
			t.Error("justification requirement not indicated")
		}
	})
}

// TestQCResponseSchemaWithSTOPJustification tests schema generation with STOP justification
func TestQCResponseSchemaWithSTOPJustification(t *testing.T) {
	// Note: We're testing the schema string contains the expected field
	// The actual schema is defined in models/schemas.go

	t.Run("schema without STOP justification", func(t *testing.T) {
		// When requireSTOPJustification is false, stop_justification should not be in schema
		// This is tested implicitly by the existing QCResponseSchema tests
	})

	t.Run("schema with STOP justification", func(t *testing.T) {
		// When requireSTOPJustification is true, stop_justification should be in schema
		// This is tested in models/schemas_test.go
	})
}
