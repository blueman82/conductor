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
			}
		},
		"required": ["score", "reasoning"],
		"additionalProperties": false
	}`
}
