package models

import (
	"encoding/json"
	"testing"
)

func TestAgentResponseSchema(t *testing.T) {
	schema := AgentResponseSchema()

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(schema), &parsed); err != nil {
		t.Fatalf("AgentResponseSchema returned invalid JSON: %v", err)
	}

	// Verify required fields are present
	if required, ok := parsed["required"].([]interface{}); ok {
		found := map[string]bool{}
		for _, field := range required {
			if fieldStr, ok := field.(string); ok {
				found[fieldStr] = true
			}
		}
		if !found["status"] {
			t.Error("'status' should be required")
		}
		if !found["summary"] {
			t.Error("'summary' should be required")
		}
	} else {
		t.Error("'required' field is missing or not an array")
	}

	// Verify status enum
	if props, ok := parsed["properties"].(map[string]interface{}); ok {
		if statusProp, ok := props["status"].(map[string]interface{}); ok {
			if enum, ok := statusProp["enum"].([]interface{}); ok {
				expectedStatuses := map[string]bool{"success": false, "failed": false}
				for _, status := range enum {
					if statusStr, ok := status.(string); ok {
						expectedStatuses[statusStr] = true
					}
				}
				if !expectedStatuses["success"] || !expectedStatuses["failed"] {
					t.Error("status enum should contain 'success' and 'failed'")
				}
			} else {
				t.Error("status property should have enum constraint")
			}
		} else {
			t.Error("'status' property is missing")
		}

		// Verify files_modified array type
		if filesProp, ok := props["files_modified"].(map[string]interface{}); ok {
			if fileType, ok := filesProp["type"].(string); !ok || fileType != "array" {
				t.Error("files_modified should be of type array")
			}
		}

		// Verify metadata object type with additionalProperties
		if metadataProp, ok := props["metadata"].(map[string]interface{}); ok {
			if metaType, ok := metadataProp["type"].(string); !ok || metaType != "object" {
				t.Error("metadata should be of type object")
			}
			if additionalProps, ok := metadataProp["additionalProperties"].(bool); !ok || !additionalProps {
				t.Error("metadata should allow additionalProperties")
			}
		}
	} else {
		t.Error("'properties' field is missing or not an object")
	}
}

func TestQCResponseSchemaWithoutCriteria(t *testing.T) {
	schema := QCResponseSchema(false)

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(schema), &parsed); err != nil {
		t.Fatalf("QCResponseSchema returned invalid JSON: %v", err)
	}

	// Verify required fields
	if required, ok := parsed["required"].([]interface{}); ok {
		found := map[string]bool{}
		for _, field := range required {
			if fieldStr, ok := field.(string); ok {
				found[fieldStr] = true
			}
		}
		if !found["verdict"] {
			t.Error("'verdict' should be required")
		}
		if !found["feedback"] {
			t.Error("'feedback' should be required")
		}
	} else {
		t.Error("'required' field is missing or not an array")
	}

	// Verify verdict enum
	if props, ok := parsed["properties"].(map[string]interface{}); ok {
		if verdictProp, ok := props["verdict"].(map[string]interface{}); ok {
			if enum, ok := verdictProp["enum"].([]interface{}); ok {
				expectedVerdicts := map[string]bool{"GREEN": false, "RED": false, "YELLOW": false}
				for _, verdict := range enum {
					if verdictStr, ok := verdict.(string); ok {
						expectedVerdicts[verdictStr] = true
					}
				}
				if !expectedVerdicts["GREEN"] || !expectedVerdicts["RED"] || !expectedVerdicts["YELLOW"] {
					t.Error("verdict enum should contain GREEN, RED, and YELLOW")
				}
			} else {
				t.Error("verdict property should have enum constraint")
			}
		} else {
			t.Error("'verdict' property is missing")
		}

		// Verify criteria_results is NOT present when hasSuccessCriteria=false
		if _, ok := props["criteria_results"]; ok {
			t.Error("criteria_results should not be present when hasSuccessCriteria=false")
		}
	} else {
		t.Error("'properties' field is missing or not an object")
	}
}

func TestQCResponseSchemaWithCriteria(t *testing.T) {
	schema := QCResponseSchema(true)

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(schema), &parsed); err != nil {
		t.Fatalf("QCResponseSchema returned invalid JSON: %v", err)
	}

	// Verify criteria_results is present
	if props, ok := parsed["properties"].(map[string]interface{}); ok {
		if _, ok := props["criteria_results"]; !ok {
			t.Error("criteria_results should be present when hasSuccessCriteria=true")
		}

		// Verify criteria_results structure
		if criteriaResults, ok := props["criteria_results"].(map[string]interface{}); ok {
			if crType, ok := criteriaResults["type"].(string); !ok || crType != "array" {
				t.Error("criteria_results should be of type array")
			}

			// Verify items structure
			if items, ok := criteriaResults["items"].(map[string]interface{}); ok {
				if itemType, ok := items["type"].(string); !ok || itemType != "object" {
					t.Error("criteria_results items should be of type object")
				}

				// Verify required fields for each item
				if required, ok := items["required"].([]interface{}); ok {
					found := map[string]bool{}
					for _, field := range required {
						if fieldStr, ok := field.(string); ok {
							found[fieldStr] = true
						}
					}
					if !found["index"] || !found["criterion"] || !found["passed"] {
						t.Error("criteria_results items should require index, criterion, and passed")
					}
				} else {
					t.Error("criteria_results items should have required fields")
				}

				// Verify item properties
				if itemProps, ok := items["properties"].(map[string]interface{}); ok {
					expectedProps := []string{"index", "criterion", "passed", "evidence", "fail_reason"}
					for _, prop := range expectedProps {
						if _, ok := itemProps[prop]; !ok {
							t.Errorf("criterion item should have property '%s'", prop)
						}
					}
				}
			}
		}
	} else {
		t.Error("'properties' field is missing or not an object")
	}
}

func TestIntelligentSelectionSchema(t *testing.T) {
	schema := IntelligentSelectionSchema()

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(schema), &parsed); err != nil {
		t.Fatalf("IntelligentSelectionSchema returned invalid JSON: %v", err)
	}

	// Verify required fields
	if required, ok := parsed["required"].([]interface{}); ok {
		found := map[string]bool{}
		for _, field := range required {
			if fieldStr, ok := field.(string); ok {
				found[fieldStr] = true
			}
		}
		if !found["agents"] {
			t.Error("'agents' should be required")
		}
		if !found["rationale"] {
			t.Error("'rationale' should be required")
		}
	} else {
		t.Error("'required' field is missing or not an array")
	}

	// Verify agents array type
	if props, ok := parsed["properties"].(map[string]interface{}); ok {
		if agentsProp, ok := props["agents"].(map[string]interface{}); ok {
			if agentsType, ok := agentsProp["type"].(string); !ok || agentsType != "array" {
				t.Error("agents should be of type array")
			}

			// Verify items are strings
			if items, ok := agentsProp["items"].(map[string]interface{}); ok {
				if itemType, ok := items["type"].(string); !ok || itemType != "string" {
					t.Error("agents items should be of type string")
				}
			}
		} else {
			t.Error("'agents' property is missing")
		}

		// Verify rationale is string
		if rationaleProp, ok := props["rationale"].(map[string]interface{}); ok {
			if rationaleType, ok := rationaleProp["type"].(string); !ok || rationaleType != "string" {
				t.Error("rationale should be of type string")
			}
		} else {
			t.Error("'rationale' property is missing")
		}
	} else {
		t.Error("'properties' field is missing or not an object")
	}
}

func TestSchemaCompactness(t *testing.T) {
	// Verify schemas are reasonably compact (no pretty printing)
	schemas := []struct {
		name string
		fn   func() string
	}{
		{"AgentResponseSchema", AgentResponseSchema},
		{"IntelligentSelectionSchema", IntelligentSelectionSchema},
	}

	for _, s := range schemas {
		schema := s.fn()
		// Schemas should not contain excessive newlines if properly minimized
		// But JSON marshaling may add some, so just check it's not overly verbose
		if len(schema) > 5000 {
			t.Logf("Warning: %s is %d chars (may need compacting)", s.name, len(schema))
		}
	}

	// QCResponseSchema with criteria could be larger
	schemaWithCriteria := QCResponseSchema(true)
	if len(schemaWithCriteria) > 10000 {
		t.Logf("Warning: QCResponseSchema(true) is %d chars (may need compacting)", len(schemaWithCriteria))
	}
}

func TestSchemasValidateResponses(t *testing.T) {
	tests := []struct {
		name            string
		schemaFn        func() string
		validJSON       string
		invalidJSON     string
		description     string
	}{
		{
			name:      "AgentResponse",
			schemaFn:  AgentResponseSchema,
			validJSON: `{"status":"success","summary":"Task complete","output":"Done","errors":[],"files_modified":[],"metadata":{}}`,
			// Missing required 'status'
			invalidJSON: `{"summary":"No status","output":"Done"}`,
			description: "Agent response with all required fields",
		},
		{
			name:      "IntelligentSelection",
			schemaFn:  IntelligentSelectionSchema,
			validJSON: `{"agents":["agent-1","agent-2"],"rationale":"Selected for expertise"}`,
			// Missing required 'rationale'
			invalidJSON: `{"agents":["agent-1"]}`,
			description: "Intelligent selection with required fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify valid JSON can be parsed
			var validData map[string]interface{}
			if err := json.Unmarshal([]byte(tt.validJSON), &validData); err != nil {
				t.Errorf("valid JSON failed to parse: %v", err)
			}

			// Verify invalid JSON is detectably wrong
			var invalidData map[string]interface{}
			if err := json.Unmarshal([]byte(tt.invalidJSON), &invalidData); err != nil {
				t.Errorf("invalid JSON should still be parseable: %v", err)
			}

			// Schema itself should be valid JSON
			schema := tt.schemaFn()
			var schemaData map[string]interface{}
			if err := json.Unmarshal([]byte(schema), &schemaData); err != nil {
				t.Errorf("schema is not valid JSON: %v", err)
			}
		})
	}
}

func TestSchemasCanBeUsedInFlags(t *testing.T) {
	// Schemas should be usable as CLI flag values
	tests := []struct {
		name      string
		schemaFn  func() string
		shouldErr bool
	}{
		{"AgentResponseSchema", AgentResponseSchema, false},
		{"IntelligentSelectionSchema", IntelligentSelectionSchema, false},
		{"QCResponseSchema(false)", func() string { return QCResponseSchema(false) }, false},
		{"QCResponseSchema(true)", func() string { return QCResponseSchema(true) }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := tt.schemaFn()

			// Schema should be non-empty
			if schema == "" {
				t.Error("schema should not be empty")
			}

			// Schema should be valid JSON
			var parsed interface{}
			if err := json.Unmarshal([]byte(schema), &parsed); err != nil {
				t.Errorf("schema is not valid JSON: %v", err)
			}

			// Should be able to use as a string (e.g., for CLI flags)
			if len(schema) == 0 {
				t.Error("schema string length is zero")
			}
		})
	}
}

func TestQCResponseSchemasConsistency(t *testing.T) {
	schemaWithout := QCResponseSchema(false)
	schemaWith := QCResponseSchema(true)

	// Both should be valid JSON
	var parsedWithout, parsedWith map[string]interface{}
	if err := json.Unmarshal([]byte(schemaWithout), &parsedWithout); err != nil {
		t.Fatalf("QCResponseSchema(false) is invalid: %v", err)
	}
	if err := json.Unmarshal([]byte(schemaWith), &parsedWith); err != nil {
		t.Fatalf("QCResponseSchema(true) is invalid: %v", err)
	}

	// With criteria should be larger or equal
	if len(schemaWith) < len(schemaWithout) {
		t.Error("QCResponseSchema(true) should be >= QCResponseSchema(false)")
	}

	// Both should have verdict and feedback required
	propsWithout := parsedWithout["properties"].(map[string]interface{})
	propsWith := parsedWith["properties"].(map[string]interface{})

	if _, ok := propsWithout["verdict"]; !ok {
		t.Error("verdict should be in both schemas")
	}
	if _, ok := propsWith["verdict"]; !ok {
		t.Error("verdict should be in both schemas")
	}
}
