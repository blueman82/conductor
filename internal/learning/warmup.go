package learning

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// WarmUpContext contains historical context for priming agents before task execution.
// This struct captures relevant history, patterns, and recommendations derived
// from similar past tasks.
type WarmUpContext struct {
	// RelevantHistory contains similar past task executions
	RelevantHistory []TaskExecution `json:"relevant_history"`

	// SimilarPatterns contains patterns from tasks with similar file paths or names
	SimilarPatterns []SuccessfulPattern `json:"similar_patterns"`

	// RecommendedApproach suggests the best approach based on historical success
	RecommendedApproach string `json:"recommended_approach"`

	// Confidence indicates how reliable the warm-up context is (0.0-1.0)
	// Higher values indicate more relevant historical data was found
	Confidence float64 `json:"confidence"`

	// ProgressScores maps task execution IDs to their calculated progress scores
	ProgressScores map[int64]ProgressScore `json:"progress_scores,omitempty"`

	// SimilarTaskIDs contains IDs of tasks found to be similar
	SimilarTaskIDs []int64 `json:"similar_task_ids,omitempty"`
}

// SimilarTask represents a past task with its similarity score
type SimilarTask struct {
	Execution  *TaskExecution
	Similarity float64
}

// WarmUpProvider defines the interface for building warm-up context.
// Implementations query historical data to prime agents with relevant context.
type WarmUpProvider interface {
	// BuildContext creates a warm-up context for a task.
	// The task parameter contains metadata about the task being executed,
	// including task number, name, and file paths it will modify.
	BuildContext(ctx context.Context, task *TaskInfo) (*WarmUpContext, error)
}

// TaskInfo contains metadata about a task for similarity matching.
// This is used as input to the warm-up provider.
type TaskInfo struct {
	// TaskNumber is the unique identifier for the task (e.g., "1", "2a")
	TaskNumber string

	// TaskName is the descriptive name of the task
	TaskName string

	// FilePaths are the files this task will modify or depend on
	FilePaths []string

	// PlanFile is the path to the plan file containing this task
	PlanFile string
}

// DefaultWarmUpProvider implements WarmUpProvider using the Store and LIPCollector.
type DefaultWarmUpProvider struct {
	store        *Store
	lipCollector LIPCollector
}

// NewWarmUpProvider creates a new DefaultWarmUpProvider.
func NewWarmUpProvider(store *Store) *DefaultWarmUpProvider {
	return &DefaultWarmUpProvider{
		store:        store,
		lipCollector: store, // Store implements LIPCollector
	}
}

// BuildContext creates a warm-up context by finding similar tasks and extracting patterns.
func (p *DefaultWarmUpProvider) BuildContext(ctx context.Context, task *TaskInfo) (*WarmUpContext, error) {
	if task == nil {
		return &WarmUpContext{Confidence: 0.0}, nil
	}

	warmUp := &WarmUpContext{
		RelevantHistory: []TaskExecution{},
		SimilarPatterns: []SuccessfulPattern{},
		ProgressScores:  make(map[int64]ProgressScore),
		SimilarTaskIDs:  []int64{},
	}

	// Step 1: Find similar tasks based on file path overlap and name similarity
	similarTasks, err := p.findSimilarTasks(ctx, task)
	if err != nil {
		return warmUp, err
	}

	// Step 2: Build relevant history from similar tasks
	for _, st := range similarTasks {
		if st.Execution != nil {
			warmUp.RelevantHistory = append(warmUp.RelevantHistory, *st.Execution)
			warmUp.SimilarTaskIDs = append(warmUp.SimilarTaskIDs, st.Execution.ID)
		}
	}

	// Step 3: Calculate progress scores for similar task executions
	for _, exec := range warmUp.RelevantHistory {
		score, err := p.lipCollector.CalculateProgress(ctx, exec.ID)
		if err != nil {
			// Continue with other tasks even if one fails
			continue
		}
		warmUp.ProgressScores[exec.ID] = score
	}

	// Step 4: Extract patterns from successful similar tasks
	warmUp.SimilarPatterns, err = p.extractPatterns(ctx, task, similarTasks)
	if err != nil {
		// Non-fatal: continue without patterns
		warmUp.SimilarPatterns = []SuccessfulPattern{}
	}

	// Step 5: Determine recommended approach from top successful execution
	warmUp.RecommendedApproach = p.extractRecommendedApproach(warmUp.RelevantHistory)

	// Step 6: Calculate overall confidence
	warmUp.Confidence = p.calculateConfidence(similarTasks, warmUp.ProgressScores)

	return warmUp, nil
}

// findSimilarTasks finds tasks similar to the given task based on file overlap and name similarity.
// Uses Jaccard similarity for file paths and Levenshtein-based similarity for task names.
// Returns tasks with combined similarity >= 0.6 threshold.
func (p *DefaultWarmUpProvider) findSimilarTasks(ctx context.Context, task *TaskInfo) ([]SimilarTask, error) {
	const similarityThreshold = 0.6

	// Query all recent task executions from the store
	// We limit to recent history to keep context relevant
	query := `SELECT id, plan_file, run_number, task_number, task_name, agent, prompt,
		success, output, error_message, duration_seconds, qc_verdict, qc_feedback,
		failure_patterns, timestamp, context
		FROM task_executions
		ORDER BY timestamp DESC
		LIMIT 100`

	rows, err := p.store.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allTasks []*TaskExecution
	for rows.Next() {
		exec, err := scanTaskExecution(rows)
		if err != nil {
			continue
		}
		allTasks = append(allTasks, exec)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Calculate similarity for each task
	var similarTasks []SimilarTask
	taskFilePaths := normalizeFilePaths(task.FilePaths)

	for _, exec := range allTasks {
		// Skip the exact same task number (we want similar, not identical)
		if exec.TaskNumber == task.TaskNumber && exec.PlanFile == task.PlanFile {
			continue
		}

		// Get file paths for this execution
		execFilePaths := extractFilePathsFromExecution(ctx, p.store, exec)
		execFilePaths = normalizeFilePaths(execFilePaths)

		// Calculate Jaccard similarity for file paths
		filePathSimilarity := jaccardSimilarity(taskFilePaths, execFilePaths)

		// Calculate name similarity using normalized Levenshtein
		nameSimilarity := normalizedLevenshteinSimilarity(task.TaskName, exec.TaskName)

		// Combined similarity (weighted average: 60% files, 40% name)
		combinedSimilarity := 0.6*filePathSimilarity + 0.4*nameSimilarity

		if combinedSimilarity >= similarityThreshold {
			similarTasks = append(similarTasks, SimilarTask{
				Execution:  exec,
				Similarity: combinedSimilarity,
			})
		}
	}

	// Sort by similarity descending
	sort.Slice(similarTasks, func(i, j int) bool {
		return similarTasks[i].Similarity > similarTasks[j].Similarity
	})

	// Limit to top 10 similar tasks
	if len(similarTasks) > 10 {
		similarTasks = similarTasks[:10]
	}

	return similarTasks, nil
}

// extractPatterns extracts successful patterns from similar tasks.
func (p *DefaultWarmUpProvider) extractPatterns(ctx context.Context, task *TaskInfo, similarTasks []SimilarTask) ([]SuccessfulPattern, error) {
	var patterns []SuccessfulPattern

	// Get top patterns from the store
	topPatterns, err := p.store.GetTopPatterns(ctx, 20)
	if err != nil {
		return patterns, err
	}

	// Build a set of task hashes from similar tasks
	similarHashes := make(map[string]bool)
	for _, st := range similarTasks {
		if st.Execution != nil && st.Execution.Success {
			// Create a simple hash from task name for pattern matching
			hash := normalizeForHash(st.Execution.TaskName)
			similarHashes[hash] = true
		}
	}

	// Filter patterns to those relevant to similar tasks
	for _, pattern := range topPatterns {
		// Check if pattern hash overlaps with similar task hashes (prefix match)
		for hash := range similarHashes {
			if strings.HasPrefix(pattern.TaskHash, hash[:min(len(hash), 8)]) ||
				strings.HasPrefix(hash, pattern.TaskHash[:min(len(pattern.TaskHash), 8)]) {
				patterns = append(patterns, *pattern)
				break
			}
		}
	}

	// Limit to top 5 patterns
	if len(patterns) > 5 {
		patterns = patterns[:5]
	}

	return patterns, nil
}

// extractRecommendedApproach finds the best approach from successful historical executions.
func (p *DefaultWarmUpProvider) extractRecommendedApproach(history []TaskExecution) string {
	// Find the most recent successful execution with high-quality output
	for _, exec := range history {
		if exec.Success && exec.QCVerdict == "GREEN" && exec.Output != "" {
			// Extract approach description from the output (first 500 chars as summary)
			approach := exec.Output
			if len(approach) > 500 {
				approach = approach[:500] + "..."
			}
			return "Based on successful execution of similar task '" + exec.TaskName +
				"' with agent " + exec.Agent + ": " + approach
		}
	}

	// Fall back to any successful execution
	for _, exec := range history {
		if exec.Success && exec.Output != "" {
			approach := exec.Output
			if len(approach) > 300 {
				approach = approach[:300] + "..."
			}
			return "Previously successful approach for '" + exec.TaskName + "': " + approach
		}
	}

	return ""
}

// calculateConfidence computes overall confidence based on similar tasks and progress scores.
func (p *DefaultWarmUpProvider) calculateConfidence(similarTasks []SimilarTask, progressScores map[int64]ProgressScore) float64 {
	if len(similarTasks) == 0 {
		return 0.0
	}

	// Base confidence on average similarity of found tasks
	var totalSimilarity float64
	for _, st := range similarTasks {
		totalSimilarity += st.Similarity
	}
	avgSimilarity := totalSimilarity / float64(len(similarTasks))

	// Boost confidence if we have progress scores
	progressBoost := 0.0
	if len(progressScores) > 0 {
		var totalProgress float64
		for _, score := range progressScores {
			totalProgress += float64(score)
		}
		avgProgress := totalProgress / float64(len(progressScores))
		progressBoost = avgProgress * 0.2 // Up to 0.2 boost from progress data
	}

	// Boost for having multiple similar tasks (up to 0.1 boost)
	countBoost := float64(len(similarTasks)) / 10.0
	if countBoost > 0.1 {
		countBoost = 0.1
	}

	confidence := avgSimilarity + progressBoost + countBoost

	// Clamp to [0.0, 1.0]
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

// jaccardSimilarity calculates the Jaccard similarity coefficient between two sets.
// Returns a value between 0.0 (no overlap) and 1.0 (identical sets).
func jaccardSimilarity(set1, set2 []string) float64 {
	if len(set1) == 0 && len(set2) == 0 {
		return 0.0
	}

	// Build maps for O(n) lookup
	map1 := make(map[string]bool)
	for _, s := range set1 {
		map1[s] = true
	}

	map2 := make(map[string]bool)
	for _, s := range set2 {
		map2[s] = true
	}

	// Calculate intersection and union
	intersection := 0
	for s := range map1 {
		if map2[s] {
			intersection++
		}
	}

	// Union = |A| + |B| - |A ∩ B|
	union := len(map1) + len(map2) - intersection

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// normalizedLevenshteinSimilarity calculates similarity based on Levenshtein distance.
// Returns a value between 0.0 (completely different) and 1.0 (identical).
func normalizedLevenshteinSimilarity(s1, s2 string) float64 {
	// Normalize strings for comparison
	s1 = normalizeString(s1)
	s2 = normalizeString(s2)

	if s1 == s2 {
		return 1.0
	}

	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	distance := levenshteinDistance(s1, s2)
	maxLen := max(len(s1), len(s2))

	return 1.0 - float64(distance)/float64(maxLen)
}

// levenshteinDistance calculates the edit distance between two strings.
func levenshteinDistance(s1, s2 string) int {
	r1 := []rune(s1)
	r2 := []rune(s2)
	len1 := len(r1)
	len2 := len(r2)

	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	// Create distance matrix
	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
	}

	// Initialize first row and column
	for i := 0; i <= len1; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j] = j
	}

	// Fill in the rest of the matrix
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 1
			if r1[i-1] == r2[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len1][len2]
}

// normalizeString normalizes a string for comparison.
func normalizeString(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)

	// Remove common noise words and punctuation
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// normalizeFilePaths normalizes file paths for comparison.
func normalizeFilePaths(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	for _, p := range paths {
		// Normalize path separators and clean the path
		p = filepath.Clean(p)
		p = strings.ToLower(p)
		if p != "" && p != "." {
			normalized = append(normalized, p)
		}
	}
	return normalized
}

// normalizeForHash creates a normalized string for hash-based matching.
func normalizeForHash(s string) string {
	s = strings.ToLower(s)
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// extractFilePathsFromExecution extracts file paths from a task execution.
// This queries file_operations table for the execution's behavioral session.
func extractFilePathsFromExecution(ctx context.Context, store *Store, exec *TaskExecution) []string {
	query := `SELECT DISTINCT fo.file_path
		FROM file_operations fo
		JOIN behavioral_sessions bs ON fo.session_id = bs.id
		WHERE bs.task_execution_id = ?`

	rows, err := store.db.QueryContext(ctx, query, exec.ID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			continue
		}
		paths = append(paths, path)
	}

	return paths
}

// scanTaskExecution scans a task execution row into a TaskExecution struct.
// This is a helper for query results.
func scanTaskExecution(rows interface{ Scan(...interface{}) error }) (*TaskExecution, error) {
	exec := &TaskExecution{}
	var planFile, agent, output, errorMessage, qcVerdict, qcFeedback, failurePatterns, contextVal interface{}
	var runNumber int

	err := rows.Scan(
		&exec.ID,
		&planFile,
		&runNumber,
		&exec.TaskNumber,
		&exec.TaskName,
		&agent,
		&exec.Prompt,
		&exec.Success,
		&output,
		&errorMessage,
		&exec.DurationSecs,
		&qcVerdict,
		&qcFeedback,
		&failurePatterns,
		&exec.Timestamp,
		&contextVal,
	)
	if err != nil {
		return nil, err
	}

	exec.RunNumber = runNumber

	if s, ok := planFile.(string); ok {
		exec.PlanFile = s
	}
	if s, ok := agent.(string); ok {
		exec.Agent = s
	}
	if s, ok := output.(string); ok {
		exec.Output = s
	}
	if s, ok := errorMessage.(string); ok {
		exec.ErrorMessage = s
	}
	if s, ok := qcVerdict.(string); ok {
		exec.QCVerdict = s
	}
	if s, ok := qcFeedback.(string); ok {
		exec.QCFeedback = s
	}
	if s, ok := contextVal.(string); ok {
		exec.Context = s
	}

	return exec, nil
}

// min returns the minimum of the provided integers.
func min(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// max returns the maximum of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
