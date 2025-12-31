package pattern

import (
	"context"
	"testing"

	"github.com/harrison/conductor/internal/similarity"
)

func TestNewTaskHasher(t *testing.T) {
	h := NewTaskHasher()
	if h == nil {
		t.Fatal("NewTaskHasher returned nil")
	}
	if h.stopwords == nil {
		t.Fatal("stopwords map is nil")
	}
	if len(h.stopwords) == 0 {
		t.Fatal("stopwords map is empty")
	}

	// Verify some common stopwords are present
	expectedStopwords := []string{"the", "a", "an", "is", "are", "and", "or", "but"}
	for _, word := range expectedStopwords {
		if !h.stopwords[word] {
			t.Errorf("expected stopword %q not found", word)
		}
	}
}

func TestTaskHasher_Hash_Basic(t *testing.T) {
	h := NewTaskHasher()

	result := h.Hash("Create a user authentication service", nil)

	if result.FullHash == "" {
		t.Error("FullHash should not be empty")
	}
	if result.NormalizedHash == "" {
		t.Error("NormalizedHash should not be empty")
	}

	// Verify hash format (SHA256 produces 64 hex characters)
	if len(result.FullHash) != 64 {
		t.Errorf("FullHash should be 64 characters, got %d", len(result.FullHash))
	}
	if len(result.NormalizedHash) != 64 {
		t.Errorf("NormalizedHash should be 64 characters, got %d", len(result.NormalizedHash))
	}
}

func TestTaskHasher_Hash_WithFiles(t *testing.T) {
	h := NewTaskHasher()

	files := []string{"internal/auth/service.go", "internal/auth/handler.go"}
	result := h.Hash("Create authentication service", files)

	if result.FullHash == "" {
		t.Error("FullHash should not be empty")
	}

	// Hash with files should differ from hash without files
	resultNoFiles := h.Hash("Create authentication service", nil)
	if result.FullHash == resultNoFiles.FullHash {
		t.Error("Hash with files should differ from hash without files")
	}
}

func TestTaskHasher_Hash_FileOrder(t *testing.T) {
	h := NewTaskHasher()

	files1 := []string{"a.go", "b.go", "c.go"}
	files2 := []string{"c.go", "a.go", "b.go"}

	result1 := h.Hash("Test description", files1)
	result2 := h.Hash("Test description", files2)

	// File order should not affect hashes (files are sorted internally)
	if result1.FullHash != result2.FullHash {
		t.Error("File order should not affect FullHash")
	}
	if result1.NormalizedHash != result2.NormalizedHash {
		t.Error("File order should not affect NormalizedHash")
	}
}

func TestTaskHasher_Hash_Deterministic(t *testing.T) {
	h := NewTaskHasher()

	desc := "Implement user login functionality"
	files := []string{"login.go", "auth.go"}

	result1 := h.Hash(desc, files)
	result2 := h.Hash(desc, files)

	if result1.FullHash != result2.FullHash {
		t.Error("Same input should produce same FullHash")
	}
	if result1.NormalizedHash != result2.NormalizedHash {
		t.Error("Same input should produce same NormalizedHash")
	}
}

func TestTaskHasher_Hash_CaseSensitivity(t *testing.T) {
	h := NewTaskHasher()

	result1 := h.Hash("Create User Service", nil)
	result2 := h.Hash("create user service", nil)

	// FullHash should differ (case sensitive)
	if result1.FullHash == result2.FullHash {
		t.Error("FullHash should be case sensitive")
	}

	// NormalizedHash should be same (case insensitive after normalization)
	if result1.NormalizedHash != result2.NormalizedHash {
		t.Error("NormalizedHash should be case insensitive")
	}
}

func TestTaskHasher_Hash_PunctuationNormalization(t *testing.T) {
	h := NewTaskHasher()

	result1 := h.Hash("Create user service", nil)
	result2 := h.Hash("Create, user: service!", nil)

	// FullHash should differ (punctuation matters)
	if result1.FullHash == result2.FullHash {
		t.Error("FullHash should include punctuation")
	}

	// NormalizedHash should be same (punctuation removed)
	if result1.NormalizedHash != result2.NormalizedHash {
		t.Error("NormalizedHash should ignore punctuation")
	}
}

func TestTaskHasher_normalize(t *testing.T) {
	h := NewTaskHasher()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "basic normalization",
			input: "Create User Service",
			want:  "create service user",
		},
		{
			name:  "removes punctuation",
			input: "Hello, World!",
			want:  "hello world",
		},
		{
			name:  "removes stopwords",
			input: "The quick brown fox",
			want:  "brown fox quick",
		},
		{
			name:  "sorts words",
			input: "zebra apple banana",
			want:  "apple banana zebra",
		},
		{
			name:  "handles numbers",
			input: "version 2 release",
			want:  "2 release version", // Numbers are preserved
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name:  "only stopwords",
			input: "the a an is are",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := h.normalize(tt.input)
			if got != tt.want {
				t.Errorf("normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCompareTasks(t *testing.T) {
	h := NewTaskHasher()
	ctx := context.Background()

	t.Run("identical full hash returns 1.0", func(t *testing.T) {
		desc := "Create user service"
		files := []string{"user.go"}

		hash1 := h.Hash(desc, files)
		hash2 := h.Hash(desc, files)

		sim := CompareTasks(ctx, desc, desc, hash1, hash2, nil)
		if sim != 1.0 {
			t.Errorf("identical full hash should return 1.0, got %v", sim)
		}
	})

	t.Run("identical normalized hash returns 1.0", func(t *testing.T) {
		hash1 := h.Hash("Create User Service", nil)
		hash2 := h.Hash("create user service", nil)

		// Full hashes differ but normalized hashes match
		if hash1.FullHash == hash2.FullHash {
			t.Error("FullHash should differ for different case")
		}

		sim := CompareTasks(ctx, "Create User Service", "create user service", hash1, hash2, nil)
		if sim != 1.0 {
			t.Errorf("identical normalized hash should return 1.0, got %v", sim)
		}
	})

	t.Run("accepts ClaudeSimilarity parameter", func(t *testing.T) {
		desc1 := "Create user authentication service"
		desc2 := "Implement user login functionality"
		hash1 := h.Hash(desc1, nil)
		hash2 := h.Hash(desc2, nil)

		// Test with non-nil ClaudeSimilarity (won't make actual calls in unit tests)
		claudeSim := similarity.NewClaudeSimilarity(nil)
		// This would make a real Claude call, so just verify it compiles and runs
		// In real usage, Claude would be called for semantic comparison
		_ = CompareTasks(ctx, desc1, desc2, hash1, hash2, claudeSim)
	})

	t.Run("returns 0 when no similarity provided", func(t *testing.T) {
		desc1 := "Create user authentication"
		desc2 := "Delete database records"
		hash1 := h.Hash(desc1, nil)
		hash2 := h.Hash(desc2, nil)

		result := CompareTasks(ctx, desc1, desc2, hash1, hash2, nil)
		if result != 0.0 {
			t.Errorf("expected 0.0 without similarity, got %v", result)
		}
	})
}

func TestIsDuplicate(t *testing.T) {
	h := NewTaskHasher()
	ctx := context.Background()

	t.Run("identical is duplicate", func(t *testing.T) {
		desc := "Create user service"
		hash1 := h.Hash(desc, nil)
		hash2 := h.Hash(desc, nil)

		result := IsDuplicate(ctx, desc, desc, hash1, hash2, 0.9, nil)
		if !result {
			t.Error("identical tasks should be duplicates")
		}
	})

	t.Run("accepts ClaudeSimilarity parameter", func(t *testing.T) {
		desc1 := "Create user authentication"
		desc2 := "Implement user authentication"
		hash1 := h.Hash(desc1, nil)
		hash2 := h.Hash(desc2, nil)

		// Test that ClaudeSimilarity is accepted as parameter
		claudeSim := similarity.NewClaudeSimilarity(nil)
		// In real usage, Claude would provide semantic comparison
		_ = IsDuplicate(ctx, desc1, desc2, hash1, hash2, 0.9, claudeSim)
	})

	t.Run("without similarity returns false for different tasks", func(t *testing.T) {
		desc1 := "Create user service"
		desc2 := "Delete database records"
		hash1 := h.Hash(desc1, nil)
		hash2 := h.Hash(desc2, nil)

		result := IsDuplicate(ctx, desc1, desc2, hash1, hash2, 0.9, nil)
		if result {
			t.Error("expected false when no similarity available for different tasks")
		}
	})

	t.Run("normalized hash match is duplicate", func(t *testing.T) {
		desc1 := "Create User Service"
		desc2 := "create user service"
		hash1 := h.Hash(desc1, nil)
		hash2 := h.Hash(desc2, nil)

		// Normalized hashes match, so should be duplicate
		result := IsDuplicate(ctx, desc1, desc2, hash1, hash2, 0.9, nil)
		if !result {
			t.Error("expected true for normalized hash match")
		}
	})
}

func TestHashResult_Fields(t *testing.T) {
	h := NewTaskHasher()

	result := h.Hash("Create authentication service", []string{"auth.go", "service.go"})

	// Verify all fields are populated
	if result.FullHash == "" {
		t.Error("FullHash should not be empty")
	}
	if result.NormalizedHash == "" {
		t.Error("NormalizedHash should not be empty")
	}
}

func TestDefaultStopwords(t *testing.T) {
	stopwords := defaultStopwords()

	// Verify common stopwords are present
	expected := []string{
		"the", "a", "an", "is", "are", "was", "were",
		"and", "but", "or", "to", "of", "in", "for",
		"this", "that", "it", "i", "you", "he", "she",
	}

	for _, word := range expected {
		if !stopwords[word] {
			t.Errorf("expected stopword %q not found", word)
		}
	}

	// Verify non-stopwords are not present
	nonStopwords := []string{"create", "user", "service", "authentication"}
	for _, word := range nonStopwords {
		if stopwords[word] {
			t.Errorf("non-stopword %q should not be in stopwords", word)
		}
	}
}

func TestTaskHasher_Hash_EmptyInput(t *testing.T) {
	h := NewTaskHasher()

	result := h.Hash("", nil)

	// Should still produce valid hashes
	if result.FullHash == "" {
		t.Error("FullHash should not be empty even for empty input")
	}
	if len(result.FullHash) != 64 {
		t.Errorf("FullHash should be 64 characters, got %d", len(result.FullHash))
	}
}

func TestTaskHasher_Hash_SpecialCharacters(t *testing.T) {
	h := NewTaskHasher()

	// Test with various special characters
	descriptions := []string{
		"Create user-service",
		"Create user_service",
		"Create user.service",
		"Create user/service",
		"Create user@service",
		"Create user#service",
		"Create user$service",
		"Create user%service",
	}

	for _, desc := range descriptions {
		result := h.Hash(desc, nil)

		if result.FullHash == "" {
			t.Errorf("FullHash should not be empty for %q", desc)
		}
		if result.NormalizedHash == "" {
			t.Errorf("NormalizedHash should not be empty for %q", desc)
		}
	}
}

func TestTaskHasher_Hash_Unicode(t *testing.T) {
	h := NewTaskHasher()

	// Test with unicode characters
	result := h.Hash("Create user service avec des accents", nil)

	if result.FullHash == "" {
		t.Error("FullHash should not be empty for unicode input")
	}
	if result.NormalizedHash == "" {
		t.Error("NormalizedHash should not be empty for unicode input")
	}
}

func BenchmarkTaskHasher_Hash(b *testing.B) {
	h := NewTaskHasher()
	desc := "Create a comprehensive user authentication service with password hashing and session management"
	files := []string{"internal/auth/service.go", "internal/auth/handler.go", "internal/auth/middleware.go"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Hash(desc, files)
	}
}

func BenchmarkCompareTasks(b *testing.B) {
	h := NewTaskHasher()
	ctx := context.Background()
	desc1 := "Create user authentication service"
	desc2 := "Implement user login functionality"
	hash1 := h.Hash(desc1, []string{"auth.go"})
	hash2 := h.Hash(desc2, []string{"login.go"})
	// Using nil similarity for benchmark (hash comparison only, no Claude calls)
	var sim *similarity.ClaudeSimilarity = nil

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CompareTasks(ctx, desc1, desc2, hash1, hash2, sim)
	}
}
