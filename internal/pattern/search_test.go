package pattern

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/learning"
)

func TestNewSTOPSearcher(t *testing.T) {
	t.Run("with nil store", func(t *testing.T) {
		s := NewSTOPSearcher(nil)
		if s == nil {
			t.Fatal("NewSTOPSearcher returned nil")
		}
		if s.store != nil {
			t.Error("store should be nil")
		}
		if s.hasher == nil {
			t.Error("hasher should not be nil")
		}
		if s.timeout != SearchTimeout {
			t.Errorf("timeout should be %v, got %v", SearchTimeout, s.timeout)
		}
	})

	t.Run("with store", func(t *testing.T) {
		store, err := learning.NewStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer store.Close()

		s := NewSTOPSearcher(store)
		if s == nil {
			t.Fatal("NewSTOPSearcher returned nil")
		}
		if s.store != store {
			t.Error("store should be set")
		}
	})
}

func TestSTOPSearcher_WithTimeout(t *testing.T) {
	s := NewSTOPSearcher(nil)
	customTimeout := 10 * time.Second

	result := s.WithTimeout(customTimeout)

	if result != s {
		t.Error("WithTimeout should return same instance")
	}
	if s.timeout != customTimeout {
		t.Errorf("timeout should be %v, got %v", customTimeout, s.timeout)
	}
}

func TestSTOPSearcher_Search_EmptyDescription(t *testing.T) {
	s := NewSTOPSearcher(nil)
	ctx := context.Background()

	results := s.Search(ctx, "", nil)

	// Should return with error about no keywords
	if len(results.Errors) == 0 {
		t.Error("should have error about no keywords")
	}
	if !sliceContainsString(results.Errors, "no keywords extracted from task description") {
		t.Errorf("expected 'no keywords' error, got: %v", results.Errors)
	}
}

func TestSTOPSearcher_Search_WithValidDescription(t *testing.T) {
	s := NewSTOPSearcher(nil)
	s.timeout = 1 * time.Second // Short timeout for tests
	ctx := context.Background()

	results := s.Search(ctx, "Create user authentication service", nil)

	// Should have a duration set
	if results.SearchDuration == 0 {
		t.Error("SearchDuration should be set")
	}

	// Results should have initialized slices (not nil)
	if results.GitMatches == nil {
		t.Error("GitMatches should not be nil")
	}
	if results.IssueMatches == nil {
		t.Error("IssueMatches should not be nil")
	}
	if results.DocMatches == nil {
		t.Error("DocMatches should not be nil")
	}
	if results.HistoryMatches == nil {
		t.Error("HistoryMatches should not be nil")
	}
}

func TestSTOPSearcher_Search_ContextCancellation(t *testing.T) {
	s := NewSTOPSearcher(nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	results := s.Search(ctx, "Create user authentication service", nil)

	// Should complete (may have errors due to cancellation)
	if results.SearchDuration == 0 {
		t.Error("SearchDuration should be set even with cancelled context")
	}
}

func TestSTOPSearcher_searchGit_EmptyKeywords(t *testing.T) {
	s := NewSTOPSearcher(nil)
	ctx := context.Background()

	commits, err := s.searchGit(ctx, []string{})

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("expected empty result, got: %v", commits)
	}
}

func TestSTOPSearcher_searchGit_InGitRepo(t *testing.T) {
	// Skip if not in a git repo
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		t.Skip("not in a git repository")
	}

	s := NewSTOPSearcher(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Search for common git keywords that should exist in most repos
	commits, err := s.searchGit(ctx, []string{"add", "fix", "update"})

	if err != nil {
		t.Logf("git search returned error: %v", err)
		// Not a failure - may be in a repo without matching commits
	}

	// Verify commit structure if we got results
	for _, commit := range commits {
		if commit.Hash == "" {
			t.Error("commit hash should not be empty")
		}
		if len(commit.Hash) > 7 {
			t.Error("commit hash should be truncated to 7 chars")
		}
	}
}

func TestSTOPSearcher_searchIssues_GHNotAvailable(t *testing.T) {
	// This test checks graceful fallback when gh is not available
	// We don't mock the path lookup, just verify behavior

	s := NewSTOPSearcher(nil)
	ctx := context.Background()

	issues, err := s.searchIssues(ctx, "test query")

	// Should not error - graceful fallback
	if err != nil {
		t.Errorf("expected graceful fallback, got error: %v", err)
	}

	// May or may not have issues depending on gh availability
	// Just verify the slice is initialized
	if issues == nil {
		t.Error("issues should not be nil")
	}
}

func TestSTOPSearcher_searchDocs_NoDocsDir(t *testing.T) {
	// Create a temp directory without docs/
	tmpDir, err := os.MkdirTemp("", "search_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	s := NewSTOPSearcher(nil)
	ctx := context.Background()

	matches, err := s.searchDocs(ctx, []string{"test", "keyword"})

	// Should gracefully return empty
	if err != nil {
		t.Errorf("expected graceful fallback, got error: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("expected empty result, got: %v", matches)
	}
}

func TestSTOPSearcher_searchDocs_WithDocsDir(t *testing.T) {
	// Skip if grep not available
	if _, err := exec.LookPath("grep"); err != nil {
		t.Skip("grep not available")
	}

	// Create a temp directory with docs/
	tmpDir, err := os.MkdirTemp("", "search_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create docs directory with a file
	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.Mkdir(docsDir, 0755); err != nil {
		t.Fatalf("failed to create docs dir: %v", err)
	}

	// Create a test markdown file
	testContent := "# Authentication Guide\n\nThis document describes user authentication.\n"
	testFile := filepath.Join(docsDir, "auth.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	s := NewSTOPSearcher(nil)
	ctx := context.Background()

	matches, err := s.searchDocs(ctx, []string{"authentication"})

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	// Should find the match
	if len(matches) == 0 {
		t.Error("expected to find matches")
	}

	// Verify match structure
	for _, match := range matches {
		if match.FilePath == "" {
			t.Error("match file path should not be empty")
		}
		if match.LineNumber == 0 {
			t.Error("match line number should not be 0")
		}
	}
}

func TestSTOPSearcher_searchDocs_EmptyKeywords(t *testing.T) {
	s := NewSTOPSearcher(nil)
	ctx := context.Background()

	matches, err := s.searchDocs(ctx, []string{})

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("expected empty result, got: %v", matches)
	}
}

func TestSTOPSearcher_searchHistory_NoStore(t *testing.T) {
	s := NewSTOPSearcher(nil) // nil store
	ctx := context.Background()

	hashResult := HashResult{
		FullHash:       "abc123",
		NormalizedHash: "abc123",
		Keywords:       []string{"test"},
	}

	matches, err := s.searchHistory(ctx, hashResult)

	if err != nil {
		t.Errorf("expected graceful fallback, got error: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("expected empty result, got: %v", matches)
	}
}

func TestSTOPSearcher_searchHistory_WithStore(t *testing.T) {
	store, err := learning.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Add a test pattern
	ctx := context.Background()
	pattern := &learning.SuccessfulPattern{
		TaskHash:           "abc12345678901234567890123456789012345678901234567890123456789012",
		PatternDescription: "Create user authentication service",
		SuccessCount:       5,
		LastAgent:          "golang-pro",
	}
	if err := store.AddPattern(ctx, pattern); err != nil {
		t.Fatalf("failed to add pattern: %v", err)
	}

	s := NewSTOPSearcher(store)

	// Search with similar hash prefix
	hashResult := HashResult{
		FullHash:       "abc12345678901234567890123456789012345678901234567890123456789012",
		NormalizedHash: "abc12345678901234567890123456789012345678901234567890123456789012",
		Keywords:       []string{"create", "user", "authentication", "service"},
	}

	matches, err := s.searchHistory(ctx, hashResult)

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	// Should find the pattern (high similarity since keywords match)
	if len(matches) == 0 {
		t.Error("expected to find matches")
	}

	// Verify match structure
	for _, match := range matches {
		if match.TaskHash == "" {
			t.Error("match task hash should not be empty")
		}
		if match.Similarity < 0 || match.Similarity > 1 {
			t.Errorf("similarity should be between 0 and 1, got: %v", match.Similarity)
		}
	}
}

func TestSearchResults_HasRelevantResults(t *testing.T) {
	tests := []struct {
		name    string
		results SearchResults
		want    bool
	}{
		{
			name:    "empty results",
			results: SearchResults{},
			want:    false,
		},
		{
			name: "with git matches",
			results: SearchResults{
				GitMatches: []GitCommit{{Hash: "abc123"}},
			},
			want: true,
		},
		{
			name: "with issue matches",
			results: SearchResults{
				IssueMatches: []GitHubIssue{{Number: 1}},
			},
			want: true,
		},
		{
			name: "with doc matches",
			results: SearchResults{
				DocMatches: []DocMatch{{FilePath: "docs/test.md"}},
			},
			want: true,
		},
		{
			name: "with history matches",
			results: SearchResults{
				HistoryMatches: []HistoryMatch{{TaskHash: "abc123"}},
			},
			want: true,
		},
		{
			name: "with errors but no results",
			results: SearchResults{
				Errors: []string{"some error"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.results.HasRelevantResults()
			if got != tt.want {
				t.Errorf("HasRelevantResults() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSearchResults_ToSearchResult(t *testing.T) {
	results := SearchResults{
		GitMatches: []GitCommit{
			{Hash: "abc123", Subject: "Add auth", Author: "dev"},
		},
		DocMatches: []DocMatch{
			{FilePath: "docs/auth.md", LineNumber: 10, LineText: "auth docs"},
			{FilePath: "docs/auth.md", LineNumber: 20, LineText: "more auth"}, // Same file
		},
		HistoryMatches: []HistoryMatch{
			{TaskHash: "xyz789", PatternDescription: "auth pattern", Similarity: 0.8},
		},
	}

	searchResult := results.ToSearchResult()

	// Should have pattern matches from git commits
	if len(searchResult.SimilarPatterns) != 1 {
		t.Errorf("expected 1 pattern match, got %d", len(searchResult.SimilarPatterns))
	}

	// Should have related files (deduplicated)
	if len(searchResult.RelatedFiles) != 1 {
		t.Errorf("expected 1 related file (deduplicated), got %d", len(searchResult.RelatedFiles))
	}

	// Should have existing implementations from history
	if len(searchResult.ExistingImplementations) != 1 {
		t.Errorf("expected 1 implementation, got %d", len(searchResult.ExistingImplementations))
	}

	// Should have search confidence
	if searchResult.SearchConfidence <= 0 {
		t.Error("search confidence should be > 0")
	}
	if searchResult.SearchConfidence > 1 {
		t.Error("search confidence should be <= 1")
	}
}

func TestSearchResults_ToSearchResult_Empty(t *testing.T) {
	results := SearchResults{}

	searchResult := results.ToSearchResult()

	if len(searchResult.SimilarPatterns) != 0 {
		t.Errorf("expected 0 pattern matches, got %d", len(searchResult.SimilarPatterns))
	}
	if len(searchResult.RelatedFiles) != 0 {
		t.Errorf("expected 0 related files, got %d", len(searchResult.RelatedFiles))
	}
	if len(searchResult.ExistingImplementations) != 0 {
		t.Errorf("expected 0 implementations, got %d", len(searchResult.ExistingImplementations))
	}
	if searchResult.SearchConfidence != 0 {
		t.Errorf("expected 0 confidence for empty results, got %v", searchResult.SearchConfidence)
	}
}

func TestParseGHIssuesOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   []GitHubIssue
	}{
		{
			name:   "empty output",
			output: "",
			want:   []GitHubIssue{},
		},
		{
			name:   "empty array",
			output: "[]",
			want:   []GitHubIssue{},
		},
		{
			name:   "single issue",
			output: `[{"number":123,"title":"Test issue","state":"open","url":"https://github.com/test/test/issues/123"}]`,
			want: []GitHubIssue{
				{Number: 123, Title: "Test issue", State: "open", URL: "https://github.com/test/test/issues/123"},
			},
		},
		{
			name:   "multiple issues",
			output: `[{"number":1,"title":"First","state":"open","url":"url1"},{"number":2,"title":"Second","state":"closed","url":"url2"}]`,
			want: []GitHubIssue{
				{Number: 1, Title: "First", State: "open", URL: "url1"},
				{Number: 2, Title: "Second", State: "closed", URL: "url2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGHIssuesOutput(tt.output)
			if len(got) != len(tt.want) {
				t.Errorf("parseGHIssuesOutput() returned %d issues, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i].Number != tt.want[i].Number {
					t.Errorf("issue[%d].Number = %d, want %d", i, got[i].Number, tt.want[i].Number)
				}
			}
		})
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{0, 10, 0},
		{-1, 1, -1},
	}

	for _, tt := range tests {
		got := min(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestGitCommit_Fields(t *testing.T) {
	commit := GitCommit{
		Hash:    "abc1234",
		Subject: "Add feature",
		Author:  "Developer",
		Date:    "2024-01-01",
	}

	if commit.Hash != "abc1234" {
		t.Errorf("Hash = %s, want abc1234", commit.Hash)
	}
	if commit.Subject != "Add feature" {
		t.Errorf("Subject = %s, want 'Add feature'", commit.Subject)
	}
	if commit.Author != "Developer" {
		t.Errorf("Author = %s, want Developer", commit.Author)
	}
	if commit.Date != "2024-01-01" {
		t.Errorf("Date = %s, want 2024-01-01", commit.Date)
	}
}

func TestGitHubIssue_Fields(t *testing.T) {
	issue := GitHubIssue{
		Number: 42,
		Title:  "Bug report",
		State:  "open",
		URL:    "https://github.com/test/test/issues/42",
	}

	if issue.Number != 42 {
		t.Errorf("Number = %d, want 42", issue.Number)
	}
	if issue.Title != "Bug report" {
		t.Errorf("Title = %s, want 'Bug report'", issue.Title)
	}
	if issue.State != "open" {
		t.Errorf("State = %s, want open", issue.State)
	}
}

func TestDocMatch_Fields(t *testing.T) {
	match := DocMatch{
		FilePath:   "docs/README.md",
		LineNumber: 10,
		LineText:   "This is a test",
	}

	if match.FilePath != "docs/README.md" {
		t.Errorf("FilePath = %s, want docs/README.md", match.FilePath)
	}
	if match.LineNumber != 10 {
		t.Errorf("LineNumber = %d, want 10", match.LineNumber)
	}
	if match.LineText != "This is a test" {
		t.Errorf("LineText = %s, want 'This is a test'", match.LineText)
	}
}

func TestHistoryMatch_Fields(t *testing.T) {
	match := HistoryMatch{
		TaskHash:           "abc123",
		PatternDescription: "Test pattern",
		SuccessCount:       5,
		LastAgent:          "golang-pro",
		LastUsed:           time.Now(),
		Similarity:         0.85,
	}

	if match.TaskHash != "abc123" {
		t.Errorf("TaskHash = %s, want abc123", match.TaskHash)
	}
	if match.Similarity != 0.85 {
		t.Errorf("Similarity = %f, want 0.85", match.Similarity)
	}
}

func TestSearchTimeout_Constant(t *testing.T) {
	if SearchTimeout != 5*time.Second {
		t.Errorf("SearchTimeout = %v, want 5s", SearchTimeout)
	}
}

// sliceContainsString checks if a slice contains a specific string.
func sliceContainsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkSTOPSearcher_Search(b *testing.B) {
	s := NewSTOPSearcher(nil)
	s.timeout = 100 * time.Millisecond // Short timeout for benchmark
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Search(ctx, "Create user authentication service", nil)
	}
}

func BenchmarkParseGHIssuesOutput(b *testing.B) {
	output := `[{"number":1,"title":"First","state":"open","url":"url1"},{"number":2,"title":"Second","state":"closed","url":"url2"}]`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseGHIssuesOutput(output)
	}
}

func BenchmarkSearchResults_ToSearchResult(b *testing.B) {
	results := SearchResults{
		GitMatches: []GitCommit{
			{Hash: "abc123", Subject: "Add auth", Author: "dev"},
		},
		DocMatches: []DocMatch{
			{FilePath: "docs/auth.md", LineNumber: 10, LineText: "auth docs"},
		},
		HistoryMatches: []HistoryMatch{
			{TaskHash: "xyz789", PatternDescription: "auth pattern", Similarity: 0.8},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results.ToSearchResult()
	}
}
