package pattern

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"unicode"

	"github.com/harrison/conductor/internal/similarity"
)

// HashResult contains both exact and normalized hashes for a task.
// FullHash is the exact SHA256 hash of the input.
// NormalizedHash is a hash of the normalized (lowercase, no punctuation, sorted words) input.
type HashResult struct {
	// FullHash is the SHA256 hash of the original input
	FullHash string `json:"full_hash"`

	// NormalizedHash is the SHA256 hash of the normalized input (for fuzzy matching)
	NormalizedHash string `json:"normalized_hash"`
}

// TaskHasher provides hashing functionality for task descriptions to enable duplicate detection.
type TaskHasher struct {
	// stopwords contains words to filter out during normalization
	stopwords map[string]bool
}

// NewTaskHasher creates a new TaskHasher with default stopwords.
func NewTaskHasher() *TaskHasher {
	return &TaskHasher{
		stopwords: defaultStopwords(),
	}
}

// defaultStopwords returns a set of common English stopwords to filter out.
func defaultStopwords() map[string]bool {
	words := []string{
		"the", "a", "an", "is", "are", "was", "were", "be", "been", "being",
		"have", "has", "had", "do", "does", "did", "will", "would", "could",
		"should", "may", "might", "must", "shall", "can", "need", "dare",
		"to", "of", "in", "for", "on", "with", "at", "by", "from", "as",
		"into", "through", "during", "before", "after", "above", "below",
		"between", "under", "over", "out", "up", "down", "off", "about",
		"and", "but", "or", "nor", "so", "yet", "both", "either", "neither",
		"not", "only", "also", "just", "than", "too", "very", "much",
		"this", "that", "these", "those", "it", "its", "itself",
		"i", "me", "my", "we", "us", "our", "you", "your", "he", "she",
		"him", "her", "his", "they", "them", "their", "who", "which", "what",
		"all", "each", "every", "any", "some", "no", "none", "one", "two",
	}

	stopwords := make(map[string]bool, len(words))
	for _, w := range words {
		stopwords[w] = true
	}
	return stopwords
}

// Hash produces a HashResult from a task description and file list.
// The description and files are combined into a single input string.
func (h *TaskHasher) Hash(description string, files []string) HashResult {
	// Combine description and files into input
	input := h.buildInput(description, files)

	// Calculate full hash
	fullHash := h.sha256Hash(input)

	// Normalize and calculate normalized hash
	normalized := h.normalize(input)
	normalizedHash := h.sha256Hash(normalized)

	return HashResult{
		FullHash:       fullHash,
		NormalizedHash: normalizedHash,
	}
}

// buildInput combines description and files into a single string for hashing.
func (h *TaskHasher) buildInput(description string, files []string) string {
	var builder strings.Builder
	builder.WriteString(description)

	if len(files) > 0 {
		// Sort files for consistent ordering
		sortedFiles := make([]string, len(files))
		copy(sortedFiles, files)
		sort.Strings(sortedFiles)

		builder.WriteString("\n---files---\n")
		for _, f := range sortedFiles {
			builder.WriteString(f)
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// sha256Hash calculates the SHA256 hash of a string and returns it as a hex string.
func (h *TaskHasher) sha256Hash(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// normalize applies the normalization pipeline:
// 1. Convert to lowercase
// 2. Remove punctuation
// 3. Split into words
// 4. Remove stopwords
// 5. Sort words alphabetically
// 6. Join back into a single string
func (h *TaskHasher) normalize(input string) string {
	// Convert to lowercase
	lower := strings.ToLower(input)

	// Remove punctuation and keep only alphanumeric and whitespace
	var cleaned strings.Builder
	for _, r := range lower {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			cleaned.WriteRune(r)
		} else {
			cleaned.WriteRune(' ')
		}
	}

	// Split into words
	words := strings.Fields(cleaned.String())

	// Filter stopwords and empty strings
	filtered := make([]string, 0, len(words))
	for _, w := range words {
		if len(w) > 0 && !h.stopwords[w] {
			filtered = append(filtered, w)
		}
	}

	// Sort words for consistent ordering
	sort.Strings(filtered)

	// Join back
	return strings.Join(filtered, " ")
}


// CompareTasks compares two HashResults using ClaudeSimilarity for semantic matching.
// Returns 1.0 for exact hash matches, otherwise uses Claude for semantic comparison.
// Requires a ClaudeSimilarity instance for semantic comparison.
func CompareTasks(ctx context.Context, desc1, desc2 string, hash1, hash2 HashResult, sim similarity.Similarity) float64 {
	// If full hashes match, tasks are identical
	if hash1.FullHash == hash2.FullHash {
		return 1.0
	}

	// If normalized hashes match, tasks are semantically identical
	if hash1.NormalizedHash == hash2.NormalizedHash {
		return 1.0
	}

	// Use ClaudeSimilarity for semantic comparison
	if sim != nil {
		result, err := sim.Compare(ctx, desc1, desc2)
		if err == nil && result != nil {
			return result.Score
		}
	}

	// No similarity available - return 0 (cannot determine similarity without Claude)
	return 0.0
}

// IsDuplicate checks if two tasks are duplicates based on similarity threshold.
// Requires ClaudeSimilarity for semantic comparison.
func IsDuplicate(ctx context.Context, desc1, desc2 string, hash1, hash2 HashResult, threshold float64, sim similarity.Similarity) bool {
	return CompareTasks(ctx, desc1, desc2, hash1, hash2, sim) >= threshold
}
