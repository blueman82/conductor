package models

import (
	"encoding/json"
)

// AgentResponseSchema returns a JSON Schema for the AgentResponse struct.
// This schema enforces the structure expected from Claude CLI agent responses.
// It requires 'status' and 'summary' fields, uses enum constraints for status,
// and supports dynamic metadata through additionalProperties.
func AgentResponseSchema() string {
	return `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Agent Response",
  "description": "Structured JSON output from an agent task execution",
  "type": "object",
  "required": ["status", "summary"],
  "properties": {
    "status": {
      "type": "string",
      "enum": ["success", "failed"],
      "description": "Task execution status"
    },
    "summary": {
      "type": "string",
      "description": "Brief description of the result"
    },
    "output": {
      "type": "string",
      "description": "Full execution output"
    },
    "errors": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "List of error messages"
    },
    "files_modified": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "Paths of files modified during execution"
    },
    "metadata": {
      "type": "object",
      "additionalProperties": true,
      "description": "Additional execution metadata"
    },
    "session_id": {
      "type": "string",
      "description": "Claude CLI session ID (optional)"
    }
  },
  "additionalProperties": false
}`
}

// QCResponseSchema returns a JSON Schema for the QCResponse struct.
// This schema enforces the structure expected from Quality Control reviews.
// When hasSuccessCriteria is true, it includes the criteria_results field schema
// for per-criterion verification results.
// It requires 'verdict' and 'feedback' fields, uses enum constraints for verdict.
func QCResponseSchema(hasSuccessCriteria bool) string {
	baseSchema := map[string]interface{}{
		"$schema":      "http://json-schema.org/draft-07/schema#",
		"title":        "QC Response",
		"description":  "Structured JSON output from Quality Control review",
		"type":         "object",
		"required":     []string{"verdict", "feedback"},
		"properties": map[string]interface{}{
			"verdict": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"GREEN", "RED", "YELLOW"},
				"description": "QC verdict status",
			},
			"feedback": map[string]interface{}{
				"type":        "string",
				"description": "Detailed review feedback",
			},
			"issues": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"required": []string{"severity", "description"},
					"properties": map[string]interface{}{
						"severity": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"critical", "warning", "info"},
							"description": "Issue severity level",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "Issue description",
						},
						"location": map[string]interface{}{
							"type":        "string",
							"description": "Location of issue (file:line or component)",
						},
					},
					"additionalProperties": false,
				},
				"description": "List of specific issues found",
			},
			"recommendations": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "Suggested improvements",
			},
			"should_retry": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether the task should be retried",
			},
			"suggested_agent": map[string]interface{}{
				"type":        "string",
				"description": "Alternative agent recommendation for retry",
			},
		},
		"additionalProperties": false,
	}

	// Conditionally add criteria_results field
	if hasSuccessCriteria {
		props := baseSchema["properties"].(map[string]interface{})
		props["criteria_results"] = map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"required": []string{"index", "criterion", "passed"},
				"properties": map[string]interface{}{
					"index": map[string]interface{}{
						"type":        "integer",
						"description": "Machine-parseable criterion index",
					},
					"criterion": map[string]interface{}{
						"type":        "string",
						"description": "Human-readable success criterion text",
					},
					"passed": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether the criterion was satisfied",
					},
					"evidence": map[string]interface{}{
						"type":        "string",
						"description": "Evidence supporting the pass result",
					},
					"fail_reason": map[string]interface{}{
						"type":        "string",
						"description": "Reason explaining the failure",
					},
				},
				"additionalProperties": false,
			},
			"description": "Per-criterion verification results",
		}
	}

	jsonBytes, _ := json.Marshal(baseSchema)
	return string(jsonBytes)
}

// IntelligentSelectionSchema returns a JSON Schema for intelligent agent selection responses.
// This schema enforces the structure expected from Claude when recommending QC agents.
// It requires both 'agents' (array of strings) and 'rationale' (string) fields.
func IntelligentSelectionSchema() string {
	schema := map[string]interface{}{
		"$schema":      "http://json-schema.org/draft-07/schema#",
		"title":        "Intelligent Agent Selection",
		"description":  "Claude's recommended QC agents for task review based on context",
		"type":         "object",
		"required":     []string{"agents", "rationale"},
		"properties": map[string]interface{}{
			"agents": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "Recommended agent names from registry, ordered by priority",
			},
			"rationale": map[string]interface{}{
				"type":        "string",
				"description": "Brief explanation of why these agents were selected",
			},
		},
		"additionalProperties": false,
	}

	jsonBytes, _ := json.Marshal(schema)
	return string(jsonBytes)
}
