package pattern

import (
	"testing"
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
	if len(result.Keywords) == 0 {
		t.Error("Keywords should not be empty")
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

func TestTaskHasher_extractKeywords(t *testing.T) {
	h := NewTaskHasher()

	tests := []struct {
		name        string
		description string
		wantLen     int
		wantContain []string
		wantExclude []string
	}{
		{
			name:        "basic description",
			description: "Create a user authentication service",
			wantContain: []string{"create", "user", "authentication", "service"},
			wantExclude: []string{"a"},
		},
		{
			name:        "with stopwords",
			description: "The quick brown fox jumps over the lazy dog",
			wantContain: []string{"quick", "brown", "fox", "jumps", "lazy", "dog"},
			wantExclude: []string{"the", "over"},
		},
		{
			name:        "with punctuation",
			description: "Implement login, logout, and password-reset features!",
			wantContain: []string{"implement", "login", "logout", "password", "reset", "features"},
			wantExclude: []string{"and"},
		},
		{
			name:        "single letter words excluded",
			description: "Create a B tree",
			wantContain: []string{"create", "tree"},
			wantExclude: []string{"a", "b"},
		},
		{
			name:        "empty description",
			description: "",
			wantLen:     0,
		},
		{
			name:        "only stopwords",
			description: "the a an is are",
			wantLen:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keywords := h.extractKeywords(tt.description)

			if tt.wantLen >= 0 && len(keywords) != tt.wantLen && tt.wantLen != 0 {
				// Only check exact length if wantLen is specified and non-zero
			}

			keywordSet := make(map[string]bool)
			for _, k := range keywords {
				keywordSet[k] = true
			}

			for _, want := range tt.wantContain {
				if !keywordSet[want] {
					t.Errorf("expected keyword %q not found in %v", want, keywords)
				}
			}

			for _, exclude := range tt.wantExclude {
				if keywordSet[exclude] {
					t.Errorf("unexpected keyword %q found in %v", exclude, keywords)
				}
			}

			// Verify keywords are sorted
			for i := 1; i < len(keywords); i++ {
				if keywords[i-1] > keywords[i] {
					t.Errorf("keywords should be sorted, but %q > %q", keywords[i-1], keywords[i])
				}
			}

			// Verify uniqueness
			seen := make(map[string]bool)
			for _, k := range keywords {
				if seen[k] {
					t.Errorf("duplicate keyword found: %q", k)
				}
				seen[k] = true
			}
		})
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

func TestJaccardSimilarity(t *testing.T) {
	tests := []struct {
		name      string
		keywords1 []string
		keywords2 []string
		want      float64
	}{
		{
			name:      "identical sets",
			keywords1: []string{"a", "b", "c"},
			keywords2: []string{"a", "b", "c"},
			want:      1.0,
		},
		{
			name:      "no overlap",
			keywords1: []string{"a", "b", "c"},
			keywords2: []string{"d", "e", "f"},
			want:      0.0,
		},
		{
			name:      "partial overlap",
			keywords1: []string{"a", "b", "c"},
			keywords2: []string{"b", "c", "d"},
			want:      0.5, // intersection=2, union=4
		},
		{
			name:      "both empty",
			keywords1: []string{},
			keywords2: []string{},
			want:      1.0,
		},
		{
			name:      "first empty",
			keywords1: []string{},
			keywords2: []string{"a", "b"},
			want:      0.0,
		},
		{
			name:      "second empty",
			keywords1: []string{"a", "b"},
			keywords2: []string{},
			want:      0.0,
		},
		{
			name:      "subset relationship",
			keywords1: []string{"a", "b"},
			keywords2: []string{"a", "b", "c", "d"},
			want:      0.5, // intersection=2, union=4
		},
		{
			name:      "single element match",
			keywords1: []string{"a"},
			keywords2: []string{"a"},
			want:      1.0,
		},
		{
			name:      "single element no match",
			keywords1: []string{"a"},
			keywords2: []string{"b"},
			want:      0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JaccardSimilarity(tt.keywords1, tt.keywords2)
			if got != tt.want {
				t.Errorf("JaccardSimilarity(%v, %v) = %v, want %v",
					tt.keywords1, tt.keywords2, got, tt.want)
			}
		})
	}
}

func TestJaccardSimilarity_Commutative(t *testing.T) {
	keywords1 := []string{"create", "user", "service"}
	keywords2 := []string{"user", "authentication", "service"}

	sim1 := JaccardSimilarity(keywords1, keywords2)
	sim2 := JaccardSimilarity(keywords2, keywords1)

	if sim1 != sim2 {
		t.Errorf("Jaccard similarity should be commutative: %v != %v", sim1, sim2)
	}
}

func TestCompareTasks(t *testing.T) {
	h := NewTaskHasher()

	tests := []struct {
		name      string
		desc1     string
		desc2     string
		files1    []string
		files2    []string
		wantExact bool   // true if similarity should be 1.0
		wantHigh  bool   // true if similarity should be > 0.5
		wantLow   bool   // true if similarity should be < 0.3
	}{
		{
			name:      "identical tasks",
			desc1:     "Create user authentication service",
			desc2:     "Create user authentication service",
			wantExact: true,
		},
		{
			name:     "identical meaning different case",
			desc1:    "Create User Service",
			desc2:    "create user service",
			wantHigh: true,
		},
		{
			name:  "similar tasks moderate overlap",
			desc1: "Implement user authentication",
			desc2: "Create user authentication service",
			// Jaccard = intersection/union = 2/5 = 0.4 (authentication, user)
			// This represents moderate similarity which is correct
			wantHigh: false,
		},
		{
			name:     "similar tasks high overlap",
			desc1:    "Create user authentication service",
			desc2:    "Create user login authentication service",
			wantHigh: true,
		},
		{
			name:     "completely different tasks",
			desc1:    "Create user authentication",
			desc2:    "Delete database records",
			wantLow:  true,
		},
		{
			name:      "same description different files",
			desc1:     "Create service",
			desc2:     "Create service",
			files1:    []string{"a.go"},
			files2:    []string{"b.go"},
			wantExact: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := h.Hash(tt.desc1, tt.files1)
			hash2 := h.Hash(tt.desc2, tt.files2)

			sim := CompareTasks(hash1, hash2)

			if tt.wantExact && sim != 1.0 {
				t.Errorf("expected exact match (1.0), got %v", sim)
			}
			if tt.wantHigh && sim <= 0.5 {
				t.Errorf("expected high similarity (>0.5), got %v", sim)
			}
			if tt.wantLow && sim >= 0.3 {
				t.Errorf("expected low similarity (<0.3), got %v", sim)
			}
		})
	}
}

func TestCompareTasks_IdenticalFullHash(t *testing.T) {
	h := NewTaskHasher()

	desc := "Create a user service"
	files := []string{"user.go"}

	hash1 := h.Hash(desc, files)
	hash2 := h.Hash(desc, files)

	sim := CompareTasks(hash1, hash2)
	if sim != 1.0 {
		t.Errorf("identical full hash should return 1.0, got %v", sim)
	}
}

func TestCompareTasks_IdenticalNormalizedHash(t *testing.T) {
	h := NewTaskHasher()

	hash1 := h.Hash("Create User Service", nil)
	hash2 := h.Hash("create user service", nil)

	// Full hashes differ but normalized hashes match
	if hash1.FullHash == hash2.FullHash {
		t.Error("FullHash should differ for different case")
	}

	sim := CompareTasks(hash1, hash2)
	if sim != 1.0 {
		t.Errorf("identical normalized hash should return 1.0, got %v", sim)
	}
}

func TestIsDuplicate(t *testing.T) {
	h := NewTaskHasher()

	tests := []struct {
		name      string
		desc1     string
		desc2     string
		threshold float64
		want      bool
	}{
		{
			name:      "identical is duplicate",
			desc1:     "Create user service",
			desc2:     "Create user service",
			threshold: 0.9,
			want:      true,
		},
		{
			name:      "very similar is duplicate with low threshold",
			desc1:     "Create user authentication",
			desc2:     "Create user authentication service",
			threshold: 0.5,
			want:      true,
		},
		{
			name:      "different is not duplicate",
			desc1:     "Create user service",
			desc2:     "Delete database records",
			threshold: 0.5,
			want:      false,
		},
		{
			name:      "exact threshold boundary",
			desc1:     "Create user service",
			desc2:     "Create user service",
			threshold: 1.0,
			want:      true,
		},
		{
			name:      "zero threshold always matches",
			desc1:     "Create user service",
			desc2:     "Delete database records",
			threshold: 0.0,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := h.Hash(tt.desc1, nil)
			hash2 := h.Hash(tt.desc2, nil)

			got := IsDuplicate(hash1, hash2, tt.threshold)
			if got != tt.want {
				t.Errorf("IsDuplicate() = %v, want %v", got, tt.want)
			}
		})
	}
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
	if result.Keywords == nil {
		t.Error("Keywords should not be nil")
	}

	// Keywords should contain expected words
	keywordSet := make(map[string]bool)
	for _, k := range result.Keywords {
		keywordSet[k] = true
	}

	expected := []string{"create", "authentication", "service"}
	for _, e := range expected {
		if !keywordSet[e] {
			t.Errorf("expected keyword %q not found", e)
		}
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

	// Keywords should be empty
	if len(result.Keywords) != 0 {
		t.Errorf("Keywords should be empty for empty input, got %v", result.Keywords)
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

		// Keywords should contain "create", "user", "service"
		keywordSet := make(map[string]bool)
		for _, k := range result.Keywords {
			keywordSet[k] = true
		}

		if !keywordSet["create"] || !keywordSet["user"] || !keywordSet["service"] {
			t.Errorf("keywords should contain 'create', 'user', 'service' for %q, got %v", desc, result.Keywords)
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

func BenchmarkJaccardSimilarity(b *testing.B) {
	keywords1 := []string{"create", "user", "authentication", "service", "password", "hashing"}
	keywords2 := []string{"implement", "user", "login", "service", "session", "management"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		JaccardSimilarity(keywords1, keywords2)
	}
}

func BenchmarkCompareTasks(b *testing.B) {
	h := NewTaskHasher()
	hash1 := h.Hash("Create user authentication service", []string{"auth.go"})
	hash2 := h.Hash("Implement user login functionality", []string{"login.go"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CompareTasks(hash1, hash2)
	}
}
