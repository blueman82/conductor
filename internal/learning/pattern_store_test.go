package learning

import (
	"context"
	"testing"
)

func TestPatternStoreMethods(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	t.Run("AddPattern and GetPattern", func(t *testing.T) {
		pattern := &SuccessfulPattern{
			TaskHash:           "abc123def456",
			PatternDescription: "Test pattern for validation",
			LastAgent:          "golang-pro",
		}
		if err := store.AddPattern(ctx, pattern); err != nil {
			t.Fatalf("Failed to add pattern: %v", err)
		}

		retrieved, err := store.GetPattern(ctx, "abc123def456")
		if err != nil {
			t.Fatalf("Failed to get pattern: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Pattern not found")
		}
		if retrieved.SuccessCount != 1 {
			t.Errorf("Expected success_count=1, got %d", retrieved.SuccessCount)
		}
		if retrieved.PatternDescription != "Test pattern for validation" {
			t.Errorf("Pattern description mismatch")
		}
	})

	t.Run("AddPattern increments success_count", func(t *testing.T) {
		pattern := &SuccessfulPattern{
			TaskHash:           "increment123",
			PatternDescription: "Increment test",
			LastAgent:          "test-agent",
		}
		store.AddPattern(ctx, pattern)
		store.AddPattern(ctx, pattern)
		store.AddPattern(ctx, pattern)

		retrieved, _ := store.GetPattern(ctx, "increment123")
		if retrieved.SuccessCount != 3 {
			t.Errorf("Expected success_count=3, got %d", retrieved.SuccessCount)
		}
	})

	t.Run("GetSimilarPatterns", func(t *testing.T) {
		// Add patterns with similar hash prefixes
		store.AddPattern(ctx, &SuccessfulPattern{TaskHash: "similar_abc1", PatternDescription: "Similar 1"})
		store.AddPattern(ctx, &SuccessfulPattern{TaskHash: "similar_abc2", PatternDescription: "Similar 2"})
		store.AddPattern(ctx, &SuccessfulPattern{TaskHash: "different_xyz", PatternDescription: "Different"})

		similar, err := store.GetSimilarPatterns(ctx, "similar", 10)
		if err != nil {
			t.Fatalf("Failed to get similar patterns: %v", err)
		}
		if len(similar) != 2 {
			t.Errorf("Expected 2 similar patterns, got %d", len(similar))
		}
	})

	t.Run("GetTopPatterns", func(t *testing.T) {
		patterns, err := store.GetTopPatterns(ctx, 5)
		if err != nil {
			t.Fatalf("Failed to get top patterns: %v", err)
		}
		if len(patterns) == 0 {
			t.Error("Expected some patterns")
		}
		// Verify ordered by success_count desc
		for i := 1; i < len(patterns); i++ {
			if patterns[i].SuccessCount > patterns[i-1].SuccessCount {
				t.Error("Patterns not ordered by success_count desc")
			}
		}
	})

	t.Run("GetPattern returns nil for nonexistent", func(t *testing.T) {
		retrieved, err := store.GetPattern(ctx, "nonexistent_hash")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if retrieved != nil {
			t.Error("Expected nil for nonexistent pattern")
		}
	})
}

func TestDuplicateDetectionMethods(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	t.Run("RecordDuplicateDetection and GetDuplicateDetections", func(t *testing.T) {
		detection := &DuplicateDetection{
			SourceHash:  "source123",
			MatchedHash: "matched456",
			Similarity:  0.95,
			Action:      "warned",
			TaskName:    "Test Task",
		}
		if err := store.RecordDuplicateDetection(ctx, detection); err != nil {
			t.Fatalf("Failed to record duplicate detection: %v", err)
		}
		if detection.ID == 0 {
			t.Error("Expected ID to be set")
		}

		detections, err := store.GetDuplicateDetections(ctx, "source123", 10)
		if err != nil {
			t.Fatalf("Failed to get duplicate detections: %v", err)
		}
		if len(detections) != 1 {
			t.Fatalf("Expected 1 detection, got %d", len(detections))
		}
		if detections[0].Similarity != 0.95 {
			t.Errorf("Similarity mismatch: expected 0.95, got %f", detections[0].Similarity)
		}
		if detections[0].Action != "warned" {
			t.Errorf("Action mismatch: expected 'warned', got '%s'", detections[0].Action)
		}
	})

	t.Run("Multiple detections for same source", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			store.RecordDuplicateDetection(ctx, &DuplicateDetection{
				SourceHash:  "multi_source",
				MatchedHash: "matched" + string(rune('A'+i)),
				Similarity:  float64(i) * 0.1,
				Action:      "blocked",
			})
		}

		detections, _ := store.GetDuplicateDetections(ctx, "multi_source", 10)
		if len(detections) != 3 {
			t.Errorf("Expected 3 detections, got %d", len(detections))
		}
	})
}

func TestSTOPAnalysisMethods(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	t.Run("SaveSTOPAnalysis and GetSTOPAnalysis", func(t *testing.T) {
		analysis := &STOPAnalysis{
			TaskHash:           "stop_task_123",
			TaskName:           "Test STOP Task",
			SearchResults:      `{"found": true}`,
			ThinkAnalysis:      `{"reasoning": "test"}`,
			OutlinePlan:        `{"steps": ["a", "b"]}`,
			ProveJustification: `{"proof": "validated"}`,
			FinalDecision:      "proceed",
			Confidence:         0.85,
		}
		if err := store.SaveSTOPAnalysis(ctx, analysis); err != nil {
			t.Fatalf("Failed to save STOP analysis: %v", err)
		}
		if analysis.ID == 0 {
			t.Error("Expected ID to be set")
		}

		retrieved, err := store.GetSTOPAnalysis(ctx, "stop_task_123")
		if err != nil {
			t.Fatalf("Failed to get STOP analysis: %v", err)
		}
		if retrieved == nil {
			t.Fatal("STOP analysis not found")
		}
		if retrieved.FinalDecision != "proceed" {
			t.Errorf("FinalDecision mismatch: expected 'proceed', got '%s'", retrieved.FinalDecision)
		}
		if retrieved.Confidence != 0.85 {
			t.Errorf("Confidence mismatch: expected 0.85, got %f", retrieved.Confidence)
		}
		if retrieved.SearchResults != `{"found": true}` {
			t.Errorf("SearchResults mismatch")
		}
	})

	t.Run("GetSTOPAnalysis returns most recent by ID", func(t *testing.T) {
		// Add multiple analyses for same task
		analysis1 := &STOPAnalysis{
			TaskHash:      "multi_stop",
			FinalDecision: "skip",
			Confidence:    0.5,
		}
		store.SaveSTOPAnalysis(ctx, analysis1)

		analysis2 := &STOPAnalysis{
			TaskHash:      "multi_stop",
			FinalDecision: "proceed",
			Confidence:    0.9,
		}
		store.SaveSTOPAnalysis(ctx, analysis2)

		// Second one should have higher ID
		if analysis2.ID <= analysis1.ID {
			t.Fatalf("Expected second analysis to have higher ID: %d vs %d", analysis2.ID, analysis1.ID)
		}

		retrieved, _ := store.GetSTOPAnalysis(ctx, "multi_stop")
		// Most recent by ID should be the second one
		if retrieved.ID != analysis2.ID {
			t.Errorf("Expected most recent by ID %d, got ID %d", analysis2.ID, retrieved.ID)
		}
	})

	t.Run("GetRecentSTOPAnalyses", func(t *testing.T) {
		analyses, err := store.GetRecentSTOPAnalyses(ctx, 5)
		if err != nil {
			t.Fatalf("Failed to get recent STOP analyses: %v", err)
		}
		if len(analyses) == 0 {
			t.Error("Expected some analyses")
		}
		// Should be ordered by id DESC (most recent first)
		for i := 1; i < len(analyses); i++ {
			if analyses[i].ID > analyses[i-1].ID {
				t.Error("Analyses not ordered by id desc")
			}
		}
	})

	t.Run("GetSTOPAnalysis returns nil for nonexistent", func(t *testing.T) {
		retrieved, err := store.GetSTOPAnalysis(ctx, "nonexistent_task")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if retrieved != nil {
			t.Error("Expected nil for nonexistent analysis")
		}
	})
}

func TestMigration8_PatternTables(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Verify migration 8 was applied
	applied, err := store.IsMigrationApplied(8)
	if err != nil {
		t.Fatalf("Failed to check migration: %v", err)
	}
	if !applied {
		t.Error("Migration 8 should be applied")
	}

	// Verify tables exist
	tables := []string{"successful_patterns", "duplicate_detections", "stop_analyses"}
	for _, table := range tables {
		exists, err := store.tableExists(table)
		if err != nil {
			t.Fatalf("Failed to check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("Table %s should exist", table)
		}
	}

	// Verify required indexes exist with exact names as specified
	requiredIndexes := []string{
		"idx_patterns_hash",     // For patterns hash lookup
		"idx_duplicates_source", // For duplicate source hash lookup
	}
	for _, idx := range requiredIndexes {
		exists, err := store.indexExists(idx)
		if err != nil {
			t.Fatalf("Failed to check index %s: %v", idx, err)
		}
		if !exists {
			t.Errorf("Index %s should exist", idx)
		}
	}
}
