package pattern

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/harrison/conductor/internal/learning"
)

// SearchTimeout is the default timeout for each individual search operation.
const SearchTimeout = 5 * time.Second

// STOPSearcher orchestrates parallel searches across multiple sources
// as part of the STOP (Search/Think/Outline/Prove) protocol.
type STOPSearcher struct {
	store   *learning.Store
	hasher  *TaskHasher
	timeout time.Duration
}

// NewSTOPSearcher creates a new STOPSearcher with the given learning store.
// If store is nil, history search will be skipped gracefully.
func NewSTOPSearcher(store *learning.Store) *STOPSearcher {
	return &STOPSearcher{
		store:   store,
		hasher:  NewTaskHasher(),
		timeout: SearchTimeout,
	}
}

// WithTimeout sets a custom timeout for individual search operations.
func (s *STOPSearcher) WithTimeout(timeout time.Duration) *STOPSearcher {
	s.timeout = timeout
	return s
}

// GitCommit represents a matching git commit.
type GitCommit struct {
	Hash    string `json:"hash"`
	Subject string `json:"subject"`
	Author  string `json:"author"`
	Date    string `json:"date"`
}

// GitHubIssue represents a matching GitHub issue.
type GitHubIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	URL    string `json:"url"`
}

// DocMatch represents a documentation search match.
type DocMatch struct {
	FilePath   string `json:"file_path"`
	LineNumber int    `json:"line_number"`
	LineText   string `json:"line_text"`
}

// HistoryMatch represents a matching pattern from execution history.
type HistoryMatch struct {
	TaskHash           string    `json:"task_hash"`
	PatternDescription string    `json:"pattern_description"`
	SuccessCount       int       `json:"success_count"`
	LastAgent          string    `json:"last_agent"`
	LastUsed           time.Time `json:"last_used"`
	Similarity         float64   `json:"similarity"`
}

// SearchResults aggregates results from all search sources.
type SearchResults struct {
	// GitMatches contains matching commits from git history
	GitMatches []GitCommit `json:"git_matches"`

	// IssueMatches contains matching GitHub issues (if gh CLI available)
	IssueMatches []GitHubIssue `json:"issue_matches"`

	// DocMatches contains documentation search results
	DocMatches []DocMatch `json:"doc_matches"`

	// HistoryMatches contains similar patterns from execution history
	HistoryMatches []HistoryMatch `json:"history_matches"`

	// Errors contains any non-fatal errors that occurred during search
	Errors []string `json:"errors,omitempty"`

	// SearchDuration is the total time spent searching
	SearchDuration time.Duration `json:"search_duration"`
}

// Search performs parallel searches across all sources for the given task keywords.
// It uses the task description to extract keywords and search for relevant context.
// Each search runs with its own timeout; failures are gracefully handled.
func (s *STOPSearcher) Search(ctx context.Context, taskDescription string, files []string) SearchResults {
	startTime := time.Now()
	results := SearchResults{
		GitMatches:     []GitCommit{},
		IssueMatches:   []GitHubIssue{},
		DocMatches:     []DocMatch{},
		HistoryMatches: []HistoryMatch{},
		Errors:         []string{},
	}

	// Extract keywords for searching
	hashResult := s.hasher.Hash(taskDescription, files)
	keywords := hashResult.Keywords

	if len(keywords) == 0 {
		results.Errors = append(results.Errors, "no keywords extracted from task description")
		results.SearchDuration = time.Since(startTime)
		return results
	}

	// Create search query from keywords
	searchQuery := strings.Join(keywords[:min(5, len(keywords))], " ")

	// Run searches in parallel with individual timeouts
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Git search
	wg.Add(1)
	go func() {
		defer wg.Done()
		searchCtx, cancel := context.WithTimeout(ctx, s.timeout)
		defer cancel()

		commits, err := s.searchGit(searchCtx, keywords)
		mu.Lock()
		if err != nil {
			results.Errors = append(results.Errors, fmt.Sprintf("git search: %v", err))
		} else {
			results.GitMatches = commits
		}
		mu.Unlock()
	}()

	// GitHub issue search
	wg.Add(1)
	go func() {
		defer wg.Done()
		searchCtx, cancel := context.WithTimeout(ctx, s.timeout)
		defer cancel()

		issues, err := s.searchIssues(searchCtx, searchQuery)
		mu.Lock()
		if err != nil {
			// gh CLI not available is expected - only log if actual error
			if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "not installed") {
				results.Errors = append(results.Errors, fmt.Sprintf("issue search: %v", err))
			}
		} else {
			results.IssueMatches = issues
		}
		mu.Unlock()
	}()

	// Documentation search
	wg.Add(1)
	go func() {
		defer wg.Done()
		searchCtx, cancel := context.WithTimeout(ctx, s.timeout)
		defer cancel()

		matches, err := s.searchDocs(searchCtx, keywords)
		mu.Lock()
		if err != nil {
			results.Errors = append(results.Errors, fmt.Sprintf("doc search: %v", err))
		} else {
			results.DocMatches = matches
		}
		mu.Unlock()
	}()

	// Execution history search
	wg.Add(1)
	go func() {
		defer wg.Done()
		searchCtx, cancel := context.WithTimeout(ctx, s.timeout)
		defer cancel()

		matches, err := s.searchHistory(searchCtx, hashResult)
		mu.Lock()
		if err != nil {
			results.Errors = append(results.Errors, fmt.Sprintf("history search: %v", err))
		} else {
			results.HistoryMatches = matches
		}
		mu.Unlock()
	}()

	wg.Wait()
	results.SearchDuration = time.Since(startTime)

	return results
}

// searchGit searches git commit history for matching keywords.
// Uses 'git log --grep' with '--all --name-only' to find matching commits
// across all branches and include file information for commit context.
func (s *STOPSearcher) searchGit(ctx context.Context, keywords []string) ([]GitCommit, error) {
	if len(keywords) == 0 {
		return []GitCommit{}, nil
	}

	// Build grep pattern with OR for all keywords
	grepPattern := strings.Join(keywords[:min(3, len(keywords))], "\\|")

	// Use git log with --grep to search commit messages
	// --all: search across all branches (not just current)
	// --name-only: include file names for context about what changed
	cmd := exec.CommandContext(ctx, "git", "log",
		"--all",
		"--name-only",
		"--grep="+grepPattern,
		"--format=%H|%s|%an|%ai",
		"-n", "10", // Limit to 10 results
		"--regexp-ignore-case",
	)

	output, err := cmd.Output()
	if err != nil {
		// Check if we're not in a git repo
		if exitErr, ok := err.(*exec.ExitError); ok {
			if strings.Contains(string(exitErr.Stderr), "not a git repository") {
				return []GitCommit{}, nil // Not an error, just no git repo
			}
		}
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	commits := []GitCommit{}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Check if line is a commit line (contains |) or a file name
		parts := strings.SplitN(line, "|", 4)
		if len(parts) >= 4 {
			commits = append(commits, GitCommit{
				Hash:    parts[0][:min(7, len(parts[0]))], // Short hash
				Subject: parts[1],
				Author:  parts[2],
				Date:    parts[3],
			})
		}
		// File names from --name-only are on separate lines without |
		// We skip them here as we primarily care about commit metadata
	}

	return commits, nil
}

// searchIssues searches GitHub issues using the gh CLI.
// Returns empty result gracefully if gh CLI is not available.
func (s *STOPSearcher) searchIssues(ctx context.Context, query string) ([]GitHubIssue, error) {
	// Check if gh CLI is available
	if _, err := exec.LookPath("gh"); err != nil {
		return []GitHubIssue{}, nil // gh not installed, graceful fallback
	}

	// Use gh issue list with search
	cmd := exec.CommandContext(ctx, "gh", "issue", "list",
		"--search", query,
		"--limit", "5",
		"--json", "number,title,state,url",
	)

	output, err := cmd.Output()
	if err != nil {
		// gh may fail for various reasons (not in repo, not authenticated, etc.)
		return []GitHubIssue{}, nil // Graceful fallback
	}

	// Parse JSON output manually (avoid importing encoding/json in multiple places)
	issues := parseGHIssuesOutput(string(output))
	return issues, nil
}

// parseGHIssuesOutput parses the JSON output from gh issue list.
// Simple parser to avoid complex JSON unmarshaling.
func parseGHIssuesOutput(output string) []GitHubIssue {
	issues := []GitHubIssue{}

	// Handle empty or error output
	output = strings.TrimSpace(output)
	if output == "" || output == "[]" {
		return issues
	}

	// Simple extraction: look for patterns in the JSON
	// Format: [{"number":1,"title":"...","state":"...","url":"..."},...]
	lines := strings.Split(output, "},{")
	for _, line := range lines {
		line = strings.Trim(line, "[]{}")

		issue := GitHubIssue{}

		// Extract number
		if numIdx := strings.Index(line, `"number":`); numIdx >= 0 {
			numStr := line[numIdx+9:]
			if commaIdx := strings.Index(numStr, ","); commaIdx > 0 {
				numStr = numStr[:commaIdx]
				fmt.Sscanf(numStr, "%d", &issue.Number)
			}
		}

		// Extract title
		if titleIdx := strings.Index(line, `"title":"`); titleIdx >= 0 {
			titleStr := line[titleIdx+9:]
			if endIdx := strings.Index(titleStr, `"`); endIdx > 0 {
				issue.Title = titleStr[:endIdx]
			}
		}

		// Extract state
		if stateIdx := strings.Index(line, `"state":"`); stateIdx >= 0 {
			stateStr := line[stateIdx+9:]
			if endIdx := strings.Index(stateStr, `"`); endIdx > 0 {
				issue.State = stateStr[:endIdx]
			}
		}

		// Extract URL
		if urlIdx := strings.Index(line, `"url":"`); urlIdx >= 0 {
			urlStr := line[urlIdx+7:]
			if endIdx := strings.Index(urlStr, `"`); endIdx > 0 {
				issue.URL = urlStr[:endIdx]
			}
		}

		if issue.Number > 0 {
			issues = append(issues, issue)
		}
	}

	return issues
}

// searchDocs searches the docs/ directory for keyword matches.
func (s *STOPSearcher) searchDocs(ctx context.Context, keywords []string) ([]DocMatch, error) {
	if len(keywords) == 0 {
		return []DocMatch{}, nil
	}

	// Build grep pattern
	pattern := strings.Join(keywords[:min(3, len(keywords))], "\\|")

	// Use grep to search docs/
	cmd := exec.CommandContext(ctx, "grep",
		"-r", // Recursive
		"-n", // Show line numbers
		"-i", // Case insensitive
		"-l", // List files only first to check if docs/ exists
		pattern,
		"docs/",
	)

	// First check if docs/ exists by listing matching files
	fileOutput, err := cmd.Output()
	if err != nil {
		// docs/ directory may not exist
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "No such file or directory") ||
				strings.Contains(stderr, "docs/") {
				return []DocMatch{}, nil // No docs dir, graceful fallback
			}
		}
		// grep returns exit code 1 when no matches found
		return []DocMatch{}, nil
	}

	if len(fileOutput) == 0 {
		return []DocMatch{}, nil
	}

	// Now get actual line matches with context
	cmd = exec.CommandContext(ctx, "grep",
		"-r",
		"-n",
		"-i",
		"--include=*.md",
		"--include=*.txt",
		"--include=*.rst",
		pattern,
		"docs/",
	)

	output, err := cmd.Output()
	if err != nil {
		// grep returns exit code 1 when no matches found
		return []DocMatch{}, nil
	}

	matches := []DocMatch{}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for i, line := range lines {
		if line == "" || i >= 10 { // Limit to 10 matches
			continue
		}

		// Format: file:linenum:text
		parts := strings.SplitN(line, ":", 3)
		if len(parts) >= 3 {
			var lineNum int
			fmt.Sscanf(parts[1], "%d", &lineNum)
			matches = append(matches, DocMatch{
				FilePath:   parts[0],
				LineNumber: lineNum,
				LineText:   strings.TrimSpace(parts[2]),
			})
		}
	}

	return matches, nil
}

// searchHistory searches execution history for similar patterns.
func (s *STOPSearcher) searchHistory(ctx context.Context, hashResult HashResult) ([]HistoryMatch, error) {
	if s.store == nil {
		return []HistoryMatch{}, nil // No store available, graceful fallback
	}

	// Use normalized hash prefix for fuzzy matching
	// Take first 8 characters for prefix matching
	hashPrefix := hashResult.NormalizedHash
	if len(hashPrefix) > 8 {
		hashPrefix = hashPrefix[:8]
	}

	// Query similar patterns from store
	patterns, err := s.store.GetSimilarPatterns(ctx, hashPrefix, 10)
	if err != nil {
		return nil, fmt.Errorf("query similar patterns: %w", err)
	}

	matches := []HistoryMatch{}
	for _, p := range patterns {
		// Calculate actual similarity using Jaccard
		// We need to extract keywords from the pattern description
		patternHash := s.hasher.Hash(p.PatternDescription, nil)
		similarity := JaccardSimilarity(hashResult.Keywords, patternHash.Keywords)

		// Only include if reasonably similar
		if similarity >= 0.3 {
			matches = append(matches, HistoryMatch{
				TaskHash:           p.TaskHash,
				PatternDescription: p.PatternDescription,
				SuccessCount:       p.SuccessCount,
				LastAgent:          p.LastAgent,
				LastUsed:           p.LastUsed,
				Similarity:         similarity,
			})
		}
	}

	return matches, nil
}

// HasRelevantResults returns true if any meaningful results were found.
func (r *SearchResults) HasRelevantResults() bool {
	return len(r.GitMatches) > 0 ||
		len(r.IssueMatches) > 0 ||
		len(r.DocMatches) > 0 ||
		len(r.HistoryMatches) > 0
}

// ToSearchResult converts SearchResults to the pattern.SearchResult type
// used in STOPResult for integration with the Pattern Intelligence system.
func (r *SearchResults) ToSearchResult() SearchResult {
	result := SearchResult{
		SimilarPatterns:         []PatternMatch{},
		RelatedFiles:            []string{},
		ExistingImplementations: []ImplementationRef{},
	}

	// Convert git commits to pattern matches
	for _, commit := range r.GitMatches {
		result.SimilarPatterns = append(result.SimilarPatterns, PatternMatch{
			Name:        commit.Subject,
			Description: fmt.Sprintf("Git commit %s by %s", commit.Hash, commit.Author),
			Similarity:  0.7, // Git matches are moderately similar
		})
	}

	// Convert doc matches to related files
	seenFiles := make(map[string]bool)
	for _, match := range r.DocMatches {
		if !seenFiles[match.FilePath] {
			result.RelatedFiles = append(result.RelatedFiles, match.FilePath)
			seenFiles[match.FilePath] = true
		}
	}

	// Convert history matches to existing implementations
	for _, match := range r.HistoryMatches {
		result.ExistingImplementations = append(result.ExistingImplementations, ImplementationRef{
			Name:      match.PatternDescription,
			Type:      "historical_pattern",
			Relevance: match.Similarity,
		})
	}

	// Calculate overall search confidence
	if r.HasRelevantResults() {
		// Higher confidence with more results
		result.SearchConfidence = float64(len(r.GitMatches)+len(r.DocMatches)+len(r.HistoryMatches)) / 15.0
		if result.SearchConfidence > 1.0 {
			result.SearchConfidence = 1.0
		}
	}

	return result
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CommandRunner interface for testing command execution.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// RealCommandRunner executes commands using os/exec.
type RealCommandRunner struct{}

// Run executes a command and returns its output.
func (r *RealCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}
