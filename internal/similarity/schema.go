package similarity

// SimilaritySchema returns the JSON schema for Claude CLI enforcement.
// This schema ensures Claude returns the expected SimilarityResult structure.
func SimilaritySchema() string {
	return `{
		"type": "object",
		"properties": {
			"score": {
				"type": "number",
				"minimum": 0,
				"maximum": 1,
				"description": "Similarity score from 0.0 (completely different) to 1.0 (essentially identical)"
			},
			"reasoning": {
				"type": "string",
				"description": "Explanation of the similarity assessment"
			},
			"semantic_match": {
				"type": "boolean",
				"description": "True if similarity meets threshold for semantic equivalence"
			}
		},
		"required": ["score", "reasoning", "semantic_match"],
		"additionalProperties": false
	}`
}

// BatchSimilaritySchema returns the JSON schema for batch similarity comparison.
// Returns an array of scores in the same order as input candidates.
func BatchSimilaritySchema() string {
	return `{
		"type": "object",
		"properties": {
			"scores": {
				"type": "array",
				"items": {
					"type": "number",
					"minimum": 0,
					"maximum": 1
				},
				"description": "Array of similarity scores (0.0-1.0) in same order as input candidates"
			}
		},
		"required": ["scores"],
		"additionalProperties": false
	}`
}
