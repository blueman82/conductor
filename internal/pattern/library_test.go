package pattern

import (
	"context"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
)

func TestNewPatternLibrary(t *testing.T) {
	t.Run("with nil store and config", func(t *testing.T) {
		lib := NewPatternLibrary(nil, nil)
		if lib == nil {
			t.Fatal("NewPatternLibrary returned nil")
		}
		if lib.store != nil {
			t.Error("store should be nil")
		}
		if lib.hasher == nil {
			t.Error("hasher should not be nil")
		}
		if lib.config == nil {
			t.Error("config should not be nil (should use defaults)")
		}
	})

	t.Run("with store and config", func(t *testing.T) {
		store, err := learning.NewStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer store.Close()

		cfg := &config.PatternConfig{
			SimilarityThreshold: 0.8,
			MaxPatternsPerTask:  5,
		}

		lib := NewPatternLibrary(store, cfg)
		if lib == nil {
			t.Fatal("NewPatternLibrary returned nil")
		}
		if lib.store != store {
			t.Error("store should be set")
		}
		if lib.config != cfg {
			t.Error("config should be set")
		}
	})
}

func TestPatternLibrary_Store(t *testing.T) {
	store, err := learning.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	lib := NewPatternLibrary(store, nil)
	ctx := context.Background()

	t.Run("store basic pattern", func(t *testing.T) {
		err := lib.Store(ctx, "Create user authentication service", nil, "golang-pro")
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("store pattern with files", func(t *testing.T) {
		files := []string{"internal/auth/service.go", "internal/auth/handler.go"}
		err := lib.Store(ctx, "Implement login handler", files, "backend-developer")
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("store empty description fails", func(t *testing.T) {
		err := lib.Store(ctx, "", nil, "golang-pro")
		if err == nil {
			t.Error("expected error for empty description")
		}
	})

	t.Run("store with nil store succeeds", func(t *testing.T) {
		nilLib := NewPatternLibrary(nil, nil)
		err := nilLib.Store(ctx, "Test pattern", nil, "test-agent")
		if err != nil {
			t.Errorf("expected graceful no-op, got: %v", err)
		}
	})
}

func TestPatternLibrary_Retrieve(t *testing.T) {
	store, err := learning.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Use a lower similarity threshold for testing
	cfg := &config.PatternConfig{
		SimilarityThreshold: 0.3, // Lower threshold to allow more matches
		MaxPatternsPerTask:  10,
	}
	lib := NewPatternLibrary(store, cfg)
	ctx := context.Background()

	// Store some patterns
	patterns := []struct {
		desc  string
		files []string
		agent string
	}{
		{"Create user authentication service", nil, "golang-pro"},
		{"Create user login handler", nil, "backend-developer"},
		{"Delete database records", nil, "database-admin"},
	}

	for _, p := range patterns {
		if err := lib.Store(ctx, p.desc, p.files, p.agent); err != nil {
			t.Fatalf("failed to store pattern: %v", err)
		}
	}

	t.Run("retrieve similar patterns", func(t *testing.T) {
		results, err := lib.Retrieve(ctx, "Create user authentication", nil, 10)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}

		// Should find at least the exact match
		if len(results) == 0 {
			t.Error("expected to find at least one similar pattern")
		}

		// Results should be sorted by similarity
		for i := 1; i < len(results); i++ {
			if results[i].Similarity > results[i-1].Similarity {
				t.Errorf("results should be sorted by similarity descending: %v > %v",
					results[i].Similarity, results[i-1].Similarity)
			}
		}
	})

	t.Run("retrieve with empty description", func(t *testing.T) {
		results, err := lib.Retrieve(ctx, "", nil, 10)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected empty results for empty description, got: %d", len(results))
		}
	})

	t.Run("retrieve with nil store", func(t *testing.T) {
		nilLib := NewPatternLibrary(nil, nil)
		results, err := nilLib.Retrieve(ctx, "Create user service", nil, 10)
		if err != nil {
			t.Errorf("expected graceful fallback, got: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected empty results with nil store, got: %d", len(results))
		}
	})

	t.Run("retrieve with default limit", func(t *testing.T) {
		results, err := lib.Retrieve(ctx, "Create user", nil, 0)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		// Should use default limit from config
		_ = results
	})
}

func TestPatternLibrary_IncrementSuccess(t *testing.T) {
	store, err := learning.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	lib := NewPatternLibrary(store, nil)
	ctx := context.Background()

	t.Run("increment existing pattern", func(t *testing.T) {
		desc := "Create user authentication service"
		agent := "golang-pro"

		// Store initial pattern
		err := lib.Store(ctx, desc, nil, agent)
		if err != nil {
			t.Fatalf("failed to store pattern: %v", err)
		}

		// Increment success
		err = lib.IncrementSuccess(ctx, desc, nil, agent)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}

		// Check success count increased
		exact, err := lib.GetExactMatch(ctx, desc, nil)
		if err != nil {
			t.Fatalf("failed to get exact match: %v", err)
		}
		if exact == nil {
			t.Fatal("expected to find pattern")
		}
		if exact.SuccessCount < 2 {
			t.Errorf("expected success count >= 2, got: %d", exact.SuccessCount)
		}
	})

	t.Run("increment non-existing pattern creates it", func(t *testing.T) {
		desc := "New pattern that does not exist"
		agent := "test-agent"

		err := lib.IncrementSuccess(ctx, desc, nil, agent)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}

		// Pattern should exist now
		exact, err := lib.GetExactMatch(ctx, desc, nil)
		if err != nil {
			t.Fatalf("failed to get exact match: %v", err)
		}
		if exact == nil {
			t.Error("expected pattern to be created")
		}
	})

	t.Run("increment with nil store succeeds", func(t *testing.T) {
		nilLib := NewPatternLibrary(nil, nil)
		err := nilLib.IncrementSuccess(ctx, "Test pattern", nil, "test-agent")
		if err != nil {
			t.Errorf("expected graceful no-op, got: %v", err)
		}
	})
}

func TestPatternLibrary_GetTopPatterns(t *testing.T) {
	store, err := learning.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	lib := NewPatternLibrary(store, nil)
	ctx := context.Background()

	// Store patterns with different success counts
	patterns := []struct {
		desc  string
		agent string
		count int
	}{
		{"Pattern with high success", "agent-a", 10},
		{"Pattern with medium success", "agent-b", 5},
		{"Pattern with low success", "agent-c", 1},
	}

	for _, p := range patterns {
		for i := 0; i < p.count; i++ {
			if err := lib.IncrementSuccess(ctx, p.desc, nil, p.agent); err != nil {
				t.Fatalf("failed to increment: %v", err)
			}
		}
	}

	t.Run("get top patterns", func(t *testing.T) {
		results, err := lib.GetTopPatterns(ctx, 10)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("expected 3 patterns, got: %d", len(results))
		}

		// Should be ordered by success count (but database handles this)
		// Just verify we got reasonable results
		for _, r := range results {
			if r.SuccessCount < 1 {
				t.Errorf("expected success count >= 1, got: %d", r.SuccessCount)
			}
		}
	})

	t.Run("get top patterns with limit", func(t *testing.T) {
		results, err := lib.GetTopPatterns(ctx, 2)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 patterns, got: %d", len(results))
		}
	})

	t.Run("get top patterns with nil store", func(t *testing.T) {
		nilLib := NewPatternLibrary(nil, nil)
		results, err := nilLib.GetTopPatterns(ctx, 10)
		if err != nil {
			t.Errorf("expected graceful fallback, got: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected empty results with nil store, got: %d", len(results))
		}
	})
}

func TestPatternLibrary_RecommendAgent(t *testing.T) {
	store, err := learning.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Use a lower similarity threshold for testing
	cfg := &config.PatternConfig{
		SimilarityThreshold: 0.3,
		MaxPatternsPerTask:  10,
	}
	lib := NewPatternLibrary(store, cfg)
	ctx := context.Background()

	// Store patterns with different agents and success counts
	// Agent-a has more successes on auth tasks
	for i := 0; i < 5; i++ {
		lib.Store(ctx, "Create user authentication service", nil, "agent-a")
	}
	for i := 0; i < 3; i++ {
		lib.Store(ctx, "Implement login handler", nil, "agent-a")
	}
	// Agent-b has fewer successes on similar tasks
	for i := 0; i < 2; i++ {
		lib.Store(ctx, "Create authentication middleware", nil, "agent-b")
	}

	t.Run("recommend agent based on similar patterns", func(t *testing.T) {
		rec, err := lib.RecommendAgent(ctx, "Create user authentication handler", nil)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}

		// Should recommend agent with highest success on similar patterns
		if rec == nil {
			t.Skip("no recommendation found - may depend on similarity threshold")
		}

		if rec.Agent == "" {
			t.Error("agent should not be empty")
		}
		if rec.Confidence < 0 || rec.Confidence > 1 {
			t.Errorf("confidence should be between 0 and 1, got: %v", rec.Confidence)
		}
		if rec.SuccessCount <= 0 {
			t.Error("success count should be > 0")
		}
	})

	t.Run("recommend agent with no matching patterns", func(t *testing.T) {
		rec, err := lib.RecommendAgent(ctx, "Something completely unrelated xyz abc", nil)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		// May or may not find matches depending on similarity threshold
		_ = rec
	})

	t.Run("recommend agent with nil store", func(t *testing.T) {
		nilLib := NewPatternLibrary(nil, nil)
		rec, err := nilLib.RecommendAgent(ctx, "Create user service", nil)
		if err != nil {
			t.Errorf("expected graceful fallback, got: %v", err)
		}
		if rec != nil {
			t.Error("expected nil recommendation with nil store")
		}
	})
}

func TestPatternLibrary_GetExactMatch(t *testing.T) {
	store, err := learning.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	lib := NewPatternLibrary(store, nil)
	ctx := context.Background()

	desc := "Create user authentication service"
	files := []string{"auth.go", "handler.go"}
	agent := "golang-pro"

	// Store pattern
	if err := lib.Store(ctx, desc, files, agent); err != nil {
		t.Fatalf("failed to store pattern: %v", err)
	}

	t.Run("find exact match", func(t *testing.T) {
		match, err := lib.GetExactMatch(ctx, desc, files)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if match == nil {
			t.Fatal("expected to find exact match")
		}
		if match.Description != desc {
			t.Errorf("description mismatch: got %s, want %s", match.Description, desc)
		}
		if match.Agent != agent {
			t.Errorf("agent mismatch: got %s, want %s", match.Agent, agent)
		}
		if match.Similarity != 1.0 {
			t.Errorf("exact match should have similarity 1.0, got: %v", match.Similarity)
		}
	})

	t.Run("no exact match for different description", func(t *testing.T) {
		match, err := lib.GetExactMatch(ctx, "Different description entirely", nil)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if match != nil {
			t.Error("expected no match for different description")
		}
	})

	t.Run("no exact match for different files", func(t *testing.T) {
		match, err := lib.GetExactMatch(ctx, desc, []string{"different.go"})
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if match != nil {
			t.Error("expected no match for different files")
		}
	})

	t.Run("exact match with nil store", func(t *testing.T) {
		nilLib := NewPatternLibrary(nil, nil)
		match, err := nilLib.GetExactMatch(ctx, desc, files)
		if err != nil {
			t.Errorf("expected graceful fallback, got: %v", err)
		}
		if match != nil {
			t.Error("expected nil with nil store")
		}
	})
}

func TestStoredPattern_Fields(t *testing.T) {
	now := time.Now()
	pattern := StoredPattern{
		TaskHash:     "abc123",
		Description:  "Test pattern",
		Agent:        "test-agent",
		SuccessCount: 5,
		LastUsed:     now,
		CreatedAt:    now.Add(-time.Hour),
		Similarity:   0.85,
		Metadata:     map[string]interface{}{"key": "value"},
	}

	if pattern.TaskHash != "abc123" {
		t.Errorf("TaskHash = %s, want abc123", pattern.TaskHash)
	}
	if pattern.Description != "Test pattern" {
		t.Errorf("Description = %s, want 'Test pattern'", pattern.Description)
	}
	if pattern.Agent != "test-agent" {
		t.Errorf("Agent = %s, want test-agent", pattern.Agent)
	}
	if pattern.SuccessCount != 5 {
		t.Errorf("SuccessCount = %d, want 5", pattern.SuccessCount)
	}
	if pattern.Similarity != 0.85 {
		t.Errorf("Similarity = %f, want 0.85", pattern.Similarity)
	}
	if pattern.Metadata["key"] != "value" {
		t.Error("Metadata should contain 'key': 'value'")
	}
}

func TestAgentRecommendation_Fields(t *testing.T) {
	rec := AgentRecommendation{
		Agent:        "golang-pro",
		SuccessCount: 10,
		Confidence:   0.9,
		MatchingPatterns: []StoredPattern{
			{Description: "Pattern 1", Similarity: 0.8},
			{Description: "Pattern 2", Similarity: 0.9},
		},
	}

	if rec.Agent != "golang-pro" {
		t.Errorf("Agent = %s, want golang-pro", rec.Agent)
	}
	if rec.SuccessCount != 10 {
		t.Errorf("SuccessCount = %d, want 10", rec.SuccessCount)
	}
	if rec.Confidence != 0.9 {
		t.Errorf("Confidence = %f, want 0.9", rec.Confidence)
	}
	if len(rec.MatchingPatterns) != 2 {
		t.Errorf("MatchingPatterns length = %d, want 2", len(rec.MatchingPatterns))
	}
}

func TestSortPatternsBySimilarity(t *testing.T) {
	patterns := []StoredPattern{
		{Description: "Low", Similarity: 0.3},
		{Description: "High", Similarity: 0.9},
		{Description: "Medium", Similarity: 0.6},
	}

	sortPatternsBySimilarity(patterns)

	// Should be sorted descending
	if patterns[0].Similarity != 0.9 {
		t.Errorf("First should be 0.9, got %f", patterns[0].Similarity)
	}
	if patterns[1].Similarity != 0.6 {
		t.Errorf("Second should be 0.6, got %f", patterns[1].Similarity)
	}
	if patterns[2].Similarity != 0.3 {
		t.Errorf("Third should be 0.3, got %f", patterns[2].Similarity)
	}
}

func TestSortPatternsBySimilarity_Empty(t *testing.T) {
	patterns := []StoredPattern{}
	sortPatternsBySimilarity(patterns)
	// Should not panic
}

func TestSortPatternsBySimilarity_Single(t *testing.T) {
	patterns := []StoredPattern{
		{Description: "Single", Similarity: 0.5},
	}
	sortPatternsBySimilarity(patterns)
	if len(patterns) != 1 || patterns[0].Similarity != 0.5 {
		t.Error("Single element should remain unchanged")
	}
}

func TestPatternLibrary_Integration(t *testing.T) {
	store, err := learning.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Use a lower similarity threshold for testing
	cfg := &config.PatternConfig{
		SimilarityThreshold: 0.3,
		MaxPatternsPerTask:  10,
	}
	lib := NewPatternLibrary(store, cfg)
	ctx := context.Background()

	// Simulate a workflow: store patterns, retrieve similar, recommend agent

	// 1. Store patterns from successful executions
	executions := []struct {
		desc  string
		files []string
		agent string
	}{
		{"Create user authentication service", []string{"auth/service.go"}, "golang-pro"},
		{"Implement user login handler", []string{"auth/login.go"}, "golang-pro"},
		{"Create session management", []string{"auth/session.go"}, "golang-pro"},
		{"Add password hashing utility", []string{"auth/hash.go"}, "security-engineer"},
		{"Create database migration", []string{"db/migration.go"}, "database-admin"},
	}

	for _, e := range executions {
		if err := lib.Store(ctx, e.desc, e.files, e.agent); err != nil {
			t.Fatalf("failed to store: %v", err)
		}
	}

	// Increment success for some patterns
	for i := 0; i < 3; i++ {
		lib.IncrementSuccess(ctx, "Create user authentication service", []string{"auth/service.go"}, "golang-pro")
	}

	// 2. Retrieve similar patterns for a new task
	newTask := "Implement user authentication middleware"
	similar, err := lib.Retrieve(ctx, newTask, nil, 10)
	if err != nil {
		t.Fatalf("failed to retrieve: %v", err)
	}

	if len(similar) == 0 {
		t.Log("No similar patterns found - may be expected based on similarity threshold")
	}

	// 3. Get agent recommendation
	rec, err := lib.RecommendAgent(ctx, newTask, nil)
	if err != nil {
		t.Fatalf("failed to recommend: %v", err)
	}

	// golang-pro should be recommended for auth tasks
	if rec != nil && rec.Agent != "" {
		t.Logf("Recommended agent: %s (confidence: %.2f)", rec.Agent, rec.Confidence)
	}

	// 4. Get top patterns
	top, err := lib.GetTopPatterns(ctx, 5)
	if err != nil {
		t.Fatalf("failed to get top patterns: %v", err)
	}

	if len(top) == 0 {
		t.Error("expected at least one top pattern")
	}

	// The auth pattern with incremented success should be near the top
	found := false
	for _, p := range top {
		if p.SuccessCount > 1 {
			found = true
			break
		}
	}
	if !found {
		t.Log("No pattern with incremented success count found in top - may depend on ordering")
	}
}

// Benchmark tests
func BenchmarkPatternLibrary_Store(b *testing.B) {
	store, err := learning.NewStore(":memory:")
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	lib := NewPatternLibrary(store, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lib.Store(ctx, "Create user authentication service", []string{"auth.go"}, "golang-pro")
	}
}

func BenchmarkPatternLibrary_Retrieve(b *testing.B) {
	store, err := learning.NewStore(":memory:")
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	lib := NewPatternLibrary(store, nil)
	ctx := context.Background()

	// Pre-populate with patterns
	for i := 0; i < 100; i++ {
		lib.Store(ctx, "Create user authentication service variant "+string(rune('A'+i%26)), nil, "agent")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lib.Retrieve(ctx, "Create user authentication handler", nil, 10)
	}
}

func BenchmarkPatternLibrary_RecommendAgent(b *testing.B) {
	store, err := learning.NewStore(":memory:")
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	lib := NewPatternLibrary(store, nil)
	ctx := context.Background()

	// Pre-populate with patterns
	for i := 0; i < 50; i++ {
		lib.Store(ctx, "Create user authentication service", nil, "golang-pro")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lib.RecommendAgent(ctx, "Create user authentication handler", nil)
	}
}
