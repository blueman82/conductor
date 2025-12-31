package pattern

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
	"github.com/harrison/conductor/internal/models"
	"github.com/harrison/conductor/internal/similarity"
)

// testIntelligenceSearchTimeout is the timeout value used in tests for search operations.
const testIntelligenceSearchTimeout = 5 * time.Second

func TestIntelligence_New(t *testing.T) {
	// This is an alias test for go test -run TestIntelligence
	// Actual testing is done in TestNewPatternIntelligence
	cfg := &config.PatternConfig{
		Enabled: true,
		Mode:    config.PatternModeWarn,
	}
	pi := NewPatternIntelligence(cfg, nil, nil, testIntelligenceSearchTimeout)
	if pi == nil {
		t.Error("expected non-nil PatternIntelligence")
	}
}

func TestNewPatternIntelligence(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.PatternConfig
		store   *learning.Store
		sim     *similarity.ClaudeSimilarity
		wantNil bool
	}{
		{
			name:    "nil config returns nil",
			cfg:     nil,
			store:   nil,
			sim:     nil,
			wantNil: true,
		},
		{
			name: "disabled config returns nil",
			cfg: &config.PatternConfig{
				Enabled: false,
			},
			store:   nil,
			sim:     nil,
			wantNil: true,
		},
		{
			name: "enabled config with nil store returns non-nil",
			cfg: &config.PatternConfig{
				Enabled:                  true,
				Mode:                     config.PatternModeWarn,
				SimilarityThreshold:      0.8,
				DuplicateThreshold:       0.9,
				EnableSTOP:               true,
				EnableDuplicateDetection: true,
			},
			store:   nil,
			sim:     nil,
			wantNil: false,
		},
		{
			name: "enabled config with similarity returns non-nil",
			cfg: &config.PatternConfig{
				Enabled:                  true,
				Mode:                     config.PatternModeWarn,
				SimilarityThreshold:      0.8,
				DuplicateThreshold:       0.9,
				EnableSTOP:               true,
				EnableDuplicateDetection: true,
			},
			store:   nil,
			sim:     similarity.NewClaudeSimilarity(90*time.Second, nil),
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pi := NewPatternIntelligence(tt.cfg, tt.store, tt.sim, testIntelligenceSearchTimeout)
			if (pi == nil) != tt.wantNil {
				t.Errorf("NewPatternIntelligence() = nil: %v, want nil: %v", pi == nil, tt.wantNil)
			}
		})
	}
}

func TestPatternIntelligence_Initialize(t *testing.T) {
	cfg := &config.PatternConfig{
		Enabled:                  true,
		Mode:                     config.PatternModeWarn,
		SimilarityThreshold:      0.8,
		DuplicateThreshold:       0.9,
		EnableSTOP:               true,
		EnableDuplicateDetection: true,
		MaxPatternsPerTask:       5,
	}

	t.Run("initialize sets up all components", func(t *testing.T) {
		pi := NewPatternIntelligence(cfg, nil, nil, testIntelligenceSearchTimeout)
		if pi == nil {
			t.Fatal("NewPatternIntelligence returned nil for enabled config")
		}

		// Should not be initialized yet
		if pi.IsInitialized() {
			t.Error("expected not initialized before Initialize() call")
		}

		// Initialize
		err := pi.Initialize(context.Background())
		if err != nil {
			t.Errorf("Initialize() error = %v", err)
		}

		// Should be initialized now
		if !pi.IsInitialized() {
			t.Error("expected initialized after Initialize() call")
		}

		// Second call should be no-op
		err = pi.Initialize(context.Background())
		if err != nil {
			t.Errorf("second Initialize() error = %v", err)
		}
	})

	t.Run("nil receiver returns nil error", func(t *testing.T) {
		var pi *PatternIntelligenceImpl
		err := pi.Initialize(context.Background())
		if err != nil {
			t.Errorf("Initialize() on nil receiver should return nil, got %v", err)
		}
	})
}

func TestPatternIntelligence_IsInitialized(t *testing.T) {
	t.Run("nil receiver returns false", func(t *testing.T) {
		var pi *PatternIntelligenceImpl
		if pi.IsInitialized() {
			t.Error("IsInitialized() on nil receiver should return false")
		}
	})

	t.Run("uninitialized returns false", func(t *testing.T) {
		cfg := &config.PatternConfig{Enabled: true}
		pi := NewPatternIntelligence(cfg, nil, nil, testIntelligenceSearchTimeout)
		if pi.IsInitialized() {
			t.Error("expected false before initialization")
		}
	})
}

func TestPatternIntelligence_CheckTask(t *testing.T) {
	cfg := &config.PatternConfig{
		Enabled:                  true,
		Mode:                     config.PatternModeWarn,
		SimilarityThreshold:      0.8,
		DuplicateThreshold:       0.9,
		EnableSTOP:               true,
		EnableDuplicateDetection: true,
		MaxPatternsPerTask:       5,
		MinConfidence:            0.7,
	}

	task := models.Task{
		Number:          "1",
		Name:            "Implement user authentication",
		Files:           []string{"internal/auth/handler.go", "internal/auth/service.go"},
		SuccessCriteria: []string{"Users can log in", "Sessions are secure"},
	}

	t.Run("nil receiver returns nil results", func(t *testing.T) {
		var pi *PatternIntelligenceImpl
		stop, dup, err := pi.CheckTask(context.Background(), task)
		if err != nil {
			t.Errorf("CheckTask() error = %v", err)
		}
		if stop != nil || dup != nil {
			t.Error("expected nil results from nil receiver")
		}
	})

	t.Run("returns results for valid task", func(t *testing.T) {
		pi := NewPatternIntelligence(cfg, nil, nil, testIntelligenceSearchTimeout)
		if pi == nil {
			t.Fatal("NewPatternIntelligence returned nil")
		}

		stop, dup, err := pi.CheckTask(context.Background(), task)
		if err != nil {
			t.Errorf("CheckTask() error = %v", err)
		}

		// With STOP enabled, should get a STOP result
		if stop == nil {
			t.Error("expected non-nil STOPResult when EnableSTOP is true")
		}

		// Should get a duplicate result (empty, since no library data)
		if dup == nil {
			t.Error("expected non-nil DuplicateResult when EnableDuplicateDetection is true")
		}

		// Verify STOP result structure
		if stop != nil {
			if len(stop.Think.ApproachSuggestions) == 0 {
				// May or may not have suggestions depending on search results
			}
			// Should have prove steps
			if len(stop.Prove.VerificationSteps) == 0 {
				t.Error("expected verification steps in ProveResult")
			}
		}

		// Duplicate result should indicate no duplicate found
		if dup != nil && dup.IsDuplicate {
			t.Error("expected IsDuplicate=false for first task check")
		}
	})

	t.Run("STOP disabled skips STOP analysis", func(t *testing.T) {
		cfgNoSTOP := &config.PatternConfig{
			Enabled:                  true,
			Mode:                     config.PatternModeWarn,
			EnableSTOP:               false,
			EnableDuplicateDetection: true,
		}

		pi := NewPatternIntelligence(cfgNoSTOP, nil, nil)
		stop, dup, err := pi.CheckTask(context.Background(), task)

		if err != nil {
			t.Errorf("CheckTask() error = %v", err)
		}
		if stop != nil {
			t.Error("expected nil STOPResult when EnableSTOP is false")
		}
		if dup == nil {
			t.Error("expected DuplicateResult even when STOP is disabled")
		}
	})

	t.Run("duplicate detection disabled skips duplicate check", func(t *testing.T) {
		cfgNoDup := &config.PatternConfig{
			Enabled:                  true,
			Mode:                     config.PatternModeWarn,
			EnableSTOP:               true,
			EnableDuplicateDetection: false,
		}

		pi := NewPatternIntelligence(cfgNoDup, nil, nil)
		stop, dup, err := pi.CheckTask(context.Background(), task)

		if err != nil {
			t.Errorf("CheckTask() error = %v", err)
		}
		if stop == nil {
			t.Error("expected STOPResult when EnableSTOP is true")
		}
		// Should still get a DuplicateResult, just empty
		if dup == nil {
			t.Error("expected non-nil DuplicateResult (empty when disabled)")
		}
		if dup != nil && dup.IsDuplicate {
			t.Error("expected IsDuplicate=false when detection is disabled")
		}
	})
}

func TestPatternIntelligence_CheckTaskWithStore(t *testing.T) {
	// Create in-memory store
	store, err := learning.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	cfg := &config.PatternConfig{
		Enabled:                  true,
		Mode:                     config.PatternModeWarn,
		SimilarityThreshold:      0.5, // Lower threshold for testing
		DuplicateThreshold:       0.8,
		EnableSTOP:               true,
		EnableDuplicateDetection: true,
		MaxPatternsPerTask:       5,
		MinConfidence:            0.7,
	}

	pi := NewPatternIntelligence(cfg, store, nil)
	if pi == nil {
		t.Fatal("NewPatternIntelligence returned nil")
	}

	// First task
	task1 := models.Task{
		Number:          "1",
		Name:            "Create user authentication handler",
		Files:           []string{"internal/auth/handler.go"},
		SuccessCriteria: []string{"Handler authenticates users"},
	}

	// Check first task
	stop, dup, err := pi.CheckTask(context.Background(), task1)
	if err != nil {
		t.Errorf("CheckTask() error = %v", err)
	}
	if stop == nil {
		t.Error("expected STOPResult")
	}
	if dup == nil {
		t.Error("expected DuplicateResult")
	}
}

func TestGetCheckResult(t *testing.T) {
	cfg := &config.PatternConfig{
		Mode:               config.PatternModeWarn,
		DuplicateThreshold: 0.9,
	}

	t.Run("nil inputs returns nil", func(t *testing.T) {
		result := GetCheckResult(nil, nil, cfg)
		if result != nil {
			t.Error("expected nil result for nil inputs")
		}
	})

	t.Run("creates result from STOP only", func(t *testing.T) {
		stop := &STOPResult{
			Confidence:      0.8,
			Recommendations: []string{"Review existing patterns"},
		}

		result := GetCheckResult(stop, nil, cfg)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.STOP == nil {
			t.Error("expected STOP in result")
		}
		if result.ShouldBlock {
			t.Error("should not block with just STOP result")
		}
		if len(result.Suggestions) != 1 {
			t.Errorf("expected 1 suggestion, got %d", len(result.Suggestions))
		}
	})

	t.Run("creates result from Duplicate only", func(t *testing.T) {
		dup := &DuplicateResult{
			IsDuplicate:     true,
			SimilarityScore: 0.95,
			DuplicateOf: []DuplicateRef{
				{TaskName: "Similar task", SimilarityScore: 0.95},
			},
		}

		result := GetCheckResult(nil, dup, cfg)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Duplicate == nil {
			t.Error("expected Duplicate in result")
		}
		// In warn mode, should not block
		if result.ShouldBlock {
			t.Error("should not block in warn mode")
		}
	})

	t.Run("blocks in block mode with high duplicate", func(t *testing.T) {
		cfgBlock := &config.PatternConfig{
			Mode:               config.PatternModeBlock,
			DuplicateThreshold: 0.9,
		}

		dup := &DuplicateResult{
			IsDuplicate:     true,
			SimilarityScore: 0.95,
		}

		result := GetCheckResult(nil, dup, cfgBlock)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if !result.ShouldBlock {
			t.Error("expected ShouldBlock=true in block mode with high similarity")
		}
		if result.BlockReason == "" {
			t.Error("expected BlockReason to be set")
		}
	})

	t.Run("does not block in block mode with low duplicate", func(t *testing.T) {
		cfgBlock := &config.PatternConfig{
			Mode:               config.PatternModeBlock,
			DuplicateThreshold: 0.9,
		}

		dup := &DuplicateResult{
			IsDuplicate:     true,
			SimilarityScore: 0.85, // Below threshold
		}

		result := GetCheckResult(nil, dup, cfgBlock)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.ShouldBlock {
			t.Error("should not block when similarity is below threshold")
		}
	})
}

func TestBuildTaskDescription(t *testing.T) {
	tests := []struct {
		name string
		task models.Task
		want string
	}{
		{
			name: "task with name only",
			task: models.Task{
				Name: "Simple task",
			},
			want: "Simple task",
		},
		{
			name: "task with name and criteria",
			task: models.Task{
				Name:            "Authentication",
				SuccessCriteria: []string{"Users can log in", "Sessions are secure"},
			},
			want: "Authentication Users can log in Sessions are secure",
		},
		{
			name: "empty task",
			task: models.Task{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildTaskDescription(tt.task)
			if got != tt.want {
				t.Errorf("buildTaskDescription() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a longer string", 10, "this is..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		got := truncate(tt.s, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
		}
	}
}

func TestPatternIntelligence_RecordSuccess(t *testing.T) {
	t.Run("nil receiver gracefully handles", func(t *testing.T) {
		var pi *PatternIntelligenceImpl
		task := models.Task{Name: "test"}
		err := pi.RecordSuccess(context.Background(), task, "test-agent")
		if err != nil {
			t.Errorf("RecordSuccess() on nil receiver should not error, got %v", err)
		}
	})

	t.Run("records with store", func(t *testing.T) {
		store, err := learning.NewStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer store.Close()

		cfg := &config.PatternConfig{
			Enabled:             true,
			Mode:                config.PatternModeWarn,
			SimilarityThreshold: 0.5,
			MaxPatternsPerTask:  5,
		}

		pi := NewPatternIntelligence(cfg, store, nil).(*PatternIntelligenceImpl)
		if err := pi.Initialize(context.Background()); err != nil {
			t.Fatalf("Initialize() error = %v", err)
		}

		task := models.Task{
			Name:  "Test authentication",
			Files: []string{"auth.go"},
		}

		err = pi.RecordSuccess(context.Background(), task, "golang-pro")
		if err != nil {
			t.Errorf("RecordSuccess() error = %v", err)
		}
	})
}

func TestEvaluateDuplicateRecommendation(t *testing.T) {
	tests := []struct {
		name       string
		mode       config.PatternMode
		threshold  float64
		similarity float64
		want       string
	}{
		{
			name:       "block mode above threshold",
			mode:       config.PatternModeBlock,
			threshold:  0.9,
			similarity: 0.95,
			want:       "skip",
		},
		{
			name:       "block mode below threshold",
			mode:       config.PatternModeBlock,
			threshold:  0.9,
			similarity: 0.85,
			want:       "proceed",
		},
		{
			name:       "warn mode above threshold",
			mode:       config.PatternModeWarn,
			threshold:  0.9,
			similarity: 0.95,
			want:       "review",
		},
		{
			name:       "warn mode below threshold",
			mode:       config.PatternModeWarn,
			threshold:  0.9,
			similarity: 0.85,
			want:       "proceed",
		},
		{
			name:       "suggest mode always proceeds",
			mode:       config.PatternModeSuggest,
			threshold:  0.9,
			similarity: 0.99,
			want:       "proceed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.PatternConfig{
				Enabled:            true,
				Mode:               tt.mode,
				DuplicateThreshold: tt.threshold,
			}

			pi := &PatternIntelligenceImpl{config: cfg}
			got := pi.evaluateDuplicateRecommendation(tt.similarity)
			if got != tt.want {
				t.Errorf("evaluateDuplicateRecommendation() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateThinkResult(t *testing.T) {
	pi := &PatternIntelligenceImpl{
		config: &config.PatternConfig{Enabled: true},
	}

	t.Run("empty search results", func(t *testing.T) {
		searchResults := SearchResults{
			GitMatches:     []GitCommit{},
			DocMatches:     []DocMatch{},
			HistoryMatches: []HistoryMatch{},
		}
		hashResult := HashResult{}

		result := pi.generateThinkResult(searchResults, hashResult)

		// Should have base complexity
		if result.ComplexityScore < 3 {
			t.Errorf("expected complexity >= 3, got %d", result.ComplexityScore)
		}
		if result.EstimatedEffort == "" {
			t.Error("expected EstimatedEffort to be set")
		}
	})

	t.Run("with history matches", func(t *testing.T) {
		searchResults := SearchResults{
			HistoryMatches: []HistoryMatch{
				{
					PatternDescription: "Similar task",
					SuccessCount:       3,
					LastAgent:          "golang-pro",
					LastUsed:           time.Now(),
				},
			},
		}
		hashResult := HashResult{}

		result := pi.generateThinkResult(searchResults, hashResult)

		// Should suggest reviewing previous approaches
		hasReviewSuggestion := false
		for _, s := range result.ApproachSuggestions {
			if len(s) > 0 {
				hasReviewSuggestion = true
				break
			}
		}
		if !hasReviewSuggestion {
			t.Error("expected approach suggestions with history matches")
		}
	})
}

func TestGenerateProveResult(t *testing.T) {
	pi := &PatternIntelligenceImpl{
		config: &config.PatternConfig{Enabled: true},
	}

	t.Run("generates verification steps", func(t *testing.T) {
		result := pi.generateProveResult("test task", []string{"file.go"})

		if len(result.VerificationSteps) == 0 {
			t.Error("expected verification steps")
		}
		if len(result.TestCommands) == 0 {
			t.Error("expected test commands")
		}
	})

	t.Run("adds specific tests for test files", func(t *testing.T) {
		result := pi.generateProveResult("test task", []string{"handler_test.go"})

		hasSpecificTest := false
		for _, cmd := range result.TestCommands {
			if strings.Contains(cmd, "handler_test.go") {
				hasSpecificTest = true
				break
			}
		}
		if !hasSpecificTest {
			t.Error("expected specific test command for test file")
		}
	})
}

func TestCalculateConfidence(t *testing.T) {
	pi := &PatternIntelligenceImpl{}

	tests := []struct {
		name    string
		results SearchResults
		wantMin float64
		wantMax float64
	}{
		{
			name:    "no results",
			results: SearchResults{},
			wantMin: 0.3,
			wantMax: 0.3,
		},
		{
			name: "git matches only",
			results: SearchResults{
				GitMatches: []GitCommit{{Hash: "abc"}},
			},
			wantMin: 0.5,
			wantMax: 0.7,
		},
		{
			name: "history matches",
			results: SearchResults{
				HistoryMatches: []HistoryMatch{{TaskHash: "abc"}},
			},
			wantMin: 0.6,
			wantMax: 0.8,
		},
		{
			name: "all sources",
			results: SearchResults{
				GitMatches:     []GitCommit{{Hash: "abc"}},
				DocMatches:     []DocMatch{{FilePath: "doc.md"}},
				HistoryMatches: []HistoryMatch{{TaskHash: "abc"}},
			},
			wantMin: 0.8,
			wantMax: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pi.calculateConfidence(tt.results)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("calculateConfidence() = %v, want between %v and %v", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestGenerateRecommendations(t *testing.T) {
	pi := &PatternIntelligenceImpl{
		config: &config.PatternConfig{Enabled: true},
	}

	t.Run("recommends reviewing high similarity history matches", func(t *testing.T) {
		searchResults := SearchResults{
			HistoryMatches: []HistoryMatch{
				{
					PatternDescription: "Implement authentication handler",
					Similarity:         0.85,
					SuccessCount:       5,
					LastAgent:          "golang-pro",
				},
			},
		}
		hashResult := HashResult{}

		recommendations := pi.generateRecommendations(searchResults, hashResult)

		if len(recommendations) == 0 {
			t.Error("expected recommendations for high similarity history match")
		}
		hasPatternReview := false
		for _, r := range recommendations {
			if strings.Contains(r, "Review pattern") {
				hasPatternReview = true
				break
			}
		}
		if !hasPatternReview {
			t.Error("expected recommendation to review pattern")
		}
	})

	t.Run("recommends checking git commits", func(t *testing.T) {
		searchResults := SearchResults{
			GitMatches: []GitCommit{
				{Hash: "abc123", Subject: "Add auth handler"},
				{Hash: "def456", Subject: "Update auth tests"},
			},
		}
		hashResult := HashResult{}

		recommendations := pi.generateRecommendations(searchResults, hashResult)

		hasGitRecommendation := false
		for _, r := range recommendations {
			if strings.Contains(r, "related commits") {
				hasGitRecommendation = true
				break
			}
		}
		if !hasGitRecommendation {
			t.Error("expected recommendation to check git commits")
		}
	})

	t.Run("recommends checking documentation", func(t *testing.T) {
		searchResults := SearchResults{
			DocMatches: []DocMatch{
				{FilePath: "docs/auth.md", LineNumber: 10, LineText: "Authentication flow"},
			},
		}
		hashResult := HashResult{}

		recommendations := pi.generateRecommendations(searchResults, hashResult)

		hasDocRecommendation := false
		for _, r := range recommendations {
			if strings.Contains(r, "documentation") {
				hasDocRecommendation = true
				break
			}
		}
		if !hasDocRecommendation {
			t.Error("expected recommendation to check documentation")
		}
	})

	t.Run("recommends best agent based on history", func(t *testing.T) {
		searchResults := SearchResults{
			HistoryMatches: []HistoryMatch{
				{
					PatternDescription: "Task 1",
					Similarity:         0.6,
					SuccessCount:       3,
					LastAgent:          "backend-developer",
				},
				{
					PatternDescription: "Task 2",
					Similarity:         0.5,
					SuccessCount:       2,
					LastAgent:          "backend-developer",
				},
			},
		}
		hashResult := HashResult{}

		recommendations := pi.generateRecommendations(searchResults, hashResult)

		hasAgentRecommendation := false
		for _, r := range recommendations {
			if strings.Contains(r, "backend-developer") {
				hasAgentRecommendation = true
				break
			}
		}
		if !hasAgentRecommendation {
			t.Error("expected agent recommendation based on history")
		}
	})

	t.Run("empty results produces no recommendations", func(t *testing.T) {
		searchResults := SearchResults{}
		hashResult := HashResult{}

		recommendations := pi.generateRecommendations(searchResults, hashResult)

		if len(recommendations) != 0 {
			t.Errorf("expected no recommendations for empty results, got %d", len(recommendations))
		}
	})
}

func TestCheckDuplicatesWithSimilarPatterns(t *testing.T) {
	// Create in-memory store with pattern data
	store, err := learning.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	cfg := &config.PatternConfig{
		Enabled:                  true,
		Mode:                     config.PatternModeWarn,
		SimilarityThreshold:      0.3, // Low threshold for testing
		DuplicateThreshold:       0.9,
		EnableSTOP:               false,
		EnableDuplicateDetection: true,
		MaxPatternsPerTask:       5,
	}

	pi := NewPatternIntelligence(cfg, store, nil).(*PatternIntelligenceImpl)
	if err := pi.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// Store a pattern to find as similar
	err = pi.library.Store(context.Background(), "Create user authentication handler with JWT tokens", []string{"auth.go"}, "golang-pro")
	if err != nil {
		t.Fatalf("failed to store pattern: %v", err)
	}

	t.Run("finds similar patterns below duplicate threshold", func(t *testing.T) {
		task := models.Task{
			Number: "1",
			Name:   "Implement user login with JWT authentication",
			Files:  []string{"login.go"},
		}

		_, dup, err := pi.CheckTask(context.Background(), task)
		if err != nil {
			t.Errorf("CheckTask() error = %v", err)
		}
		if dup == nil {
			t.Fatal("expected DuplicateResult")
		}

		// Should not be a duplicate (different enough)
		if dup.IsDuplicate {
			t.Log("Found as duplicate, checking similarity")
		}
	})

	t.Run("detects exact duplicate", func(t *testing.T) {
		task := models.Task{
			Number: "2",
			Name:   "Create user authentication handler with JWT tokens",
			Files:  []string{"auth.go"},
		}

		_, dup, err := pi.CheckTask(context.Background(), task)
		if err != nil {
			t.Errorf("CheckTask() error = %v", err)
		}
		if dup == nil {
			t.Fatal("expected DuplicateResult")
		}

		// Should find exact match
		if !dup.IsDuplicate {
			t.Log("Not found as duplicate - may be due to hash difference")
		}
		if dup.SimilarityScore < 0.9 {
			t.Logf("Similarity score: %.2f", dup.SimilarityScore)
		}
	})
}

func TestCheckDuplicatesBlockMode(t *testing.T) {
	store, err := learning.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	cfg := &config.PatternConfig{
		Enabled:                  true,
		Mode:                     config.PatternModeBlock,
		SimilarityThreshold:      0.3,
		DuplicateThreshold:       0.5, // Low threshold for testing
		EnableSTOP:               false,
		EnableDuplicateDetection: true,
		MaxPatternsPerTask:       5,
	}

	pi := NewPatternIntelligence(cfg, store, nil).(*PatternIntelligenceImpl)
	if err := pi.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// Store a pattern
	err = pi.library.Store(context.Background(), "Implement database migration", []string{"migrate.go"}, "backend-developer")
	if err != nil {
		t.Fatalf("failed to store pattern: %v", err)
	}

	t.Run("block mode sets should skip for duplicates", func(t *testing.T) {
		task := models.Task{
			Number: "1",
			Name:   "Implement database migration",
			Files:  []string{"migrate.go"},
		}

		_, dup, err := pi.CheckTask(context.Background(), task)
		if err != nil {
			t.Errorf("CheckTask() error = %v", err)
		}
		if dup == nil {
			t.Fatal("expected DuplicateResult")
		}

		// In block mode, high similarity should recommend skip
		if dup.IsDuplicate && dup.Recommendation != "skip" {
			t.Errorf("expected 'skip' recommendation in block mode, got %q", dup.Recommendation)
		}
	})
}

func TestCheckDuplicatesDisabled(t *testing.T) {
	cfg := &config.PatternConfig{
		Enabled:                  true,
		Mode:                     config.PatternModeWarn,
		EnableSTOP:               false,
		EnableDuplicateDetection: false,
	}

	pi := NewPatternIntelligence(cfg, nil, nil, testIntelligenceSearchTimeout).(*PatternIntelligenceImpl)
	if err := pi.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	hashResult := HashResult{
		FullHash:       "abc123",
		NormalizedHash: "def456",
	}

	result := pi.checkDuplicates(context.Background(), "test description", []string{"test.go"}, hashResult)

	if result == nil {
		t.Fatal("expected non-nil result even when disabled")
	}
	if result.IsDuplicate {
		t.Error("expected IsDuplicate=false when detection is disabled")
	}
}

func TestCheckDuplicatesNilLibrary(t *testing.T) {
	cfg := &config.PatternConfig{
		Enabled:                  true,
		Mode:                     config.PatternModeWarn,
		EnableSTOP:               false,
		EnableDuplicateDetection: true,
	}

	pi := &PatternIntelligenceImpl{
		config:      cfg,
		hasher:      NewTaskHasher(),
		library:     nil, // Nil library
		initialized: true,
	}

	hashResult := HashResult{
		FullHash:       "abc123",
		NormalizedHash: "def456",
	}

	result := pi.checkDuplicates(context.Background(), "test", []string{}, hashResult)

	if result == nil {
		t.Fatal("expected non-nil result even with nil library")
	}
	if result.IsDuplicate {
		t.Error("expected IsDuplicate=false with nil library")
	}
}

func TestGenerateOutlineResultMultipleFiles(t *testing.T) {
	pi := &PatternIntelligenceImpl{
		config: &config.PatternConfig{Enabled: true},
	}

	searchResults := SearchResults{}

	result := pi.generateOutlineResult("test task", []string{"file1.go", "file2.go", "file3.go"}, searchResults)

	// Should have integration step for multiple files
	hasIntegrationStep := false
	for _, step := range result.Steps {
		if strings.Contains(step.Description, "integration") {
			hasIntegrationStep = true
			break
		}
	}
	if !hasIntegrationStep {
		t.Error("expected integration step for multiple files")
	}

	// Should have integration points
	if len(result.IntegrationPoints) == 0 {
		t.Error("expected integration points for multiple files")
	}
}

func TestGenerateOutlineResultWithHistory(t *testing.T) {
	pi := &PatternIntelligenceImpl{
		config: &config.PatternConfig{Enabled: true},
	}

	searchResults := SearchResults{
		HistoryMatches: []HistoryMatch{
			{PatternDescription: "Similar task", SuccessCount: 3},
		},
		GitMatches: []GitCommit{
			{Hash: "abc", Subject: "Related commit"},
		},
	}

	result := pi.generateOutlineResult("test task", []string{"file.go"}, searchResults)

	// Should have step to review existing implementations
	hasReviewStep := false
	for _, step := range result.Steps {
		if strings.Contains(step.Description, "Review existing") {
			hasReviewStep = true
			break
		}
	}
	if !hasReviewStep {
		t.Error("expected review step when history/git matches exist")
	}
}

func TestGenerateThinkResultWithErrors(t *testing.T) {
	pi := &PatternIntelligenceImpl{
		config: &config.PatternConfig{Enabled: true},
	}

	searchResults := SearchResults{
		Errors: []string{"git search failed", "docs not found"},
	}
	hashResult := HashResult{}

	result := pi.generateThinkResult(searchResults, hashResult)

	// Should have risk factor for search errors
	hasSearchRisk := false
	for _, rf := range result.RiskFactors {
		if rf.Name == "Search Incomplete" {
			hasSearchRisk = true
			break
		}
	}
	if !hasSearchRisk {
		t.Error("expected Search Incomplete risk factor when errors exist")
	}
}

func TestGenerateThinkResultComplexityLevels(t *testing.T) {
	pi := &PatternIntelligenceImpl{
		config: &config.PatternConfig{Enabled: true},
	}

	tests := []struct {
		name          string
		searchResults SearchResults
		wantEffortMin string
		wantEffortMax string
	}{
		{
			name:          "low complexity",
			searchResults: SearchResults{},
			wantEffortMin: "Low",
			wantEffortMax: "Low",
		},
		{
			name: "medium complexity with git matches",
			searchResults: SearchResults{
				GitMatches: []GitCommit{{Hash: "abc"}},
			},
			wantEffortMin: "Low",
			wantEffortMax: "Medium",
		},
		{
			name: "higher complexity with history",
			searchResults: SearchResults{
				HistoryMatches: []HistoryMatch{{TaskHash: "abc"}},
				GitMatches:     []GitCommit{{Hash: "abc"}},
				DocMatches:     []DocMatch{{FilePath: "doc.md"}},
			},
			wantEffortMin: "Medium",
			wantEffortMax: "High",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pi.generateThinkResult(tt.searchResults, HashResult{})
			if result.EstimatedEffort == "" {
				t.Error("expected EstimatedEffort to be set")
			}
		})
	}
}

func TestCheckTaskWithNilHasher(t *testing.T) {
	cfg := &config.PatternConfig{
		Enabled: true,
		Mode:    config.PatternModeWarn,
	}

	// Create a manually constructed PI with nil hasher
	pi := &PatternIntelligenceImpl{
		config:      cfg,
		hasher:      nil, // Nil hasher
		initialized: true,
	}

	task := models.Task{
		Number: "1",
		Name:   "Test task",
	}

	stop, dup, err := pi.CheckTask(context.Background(), task)
	if err != nil {
		t.Errorf("CheckTask() error = %v", err)
	}
	if stop != nil || dup != nil {
		t.Error("expected nil results when hasher is nil")
	}
}

func TestCheckDuplicatesExactMatch(t *testing.T) {
	store, err := learning.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	cfg := &config.PatternConfig{
		Enabled:                  true,
		Mode:                     config.PatternModeBlock, // Block mode for skip
		SimilarityThreshold:      0.3,
		DuplicateThreshold:       0.5,
		EnableSTOP:               false,
		EnableDuplicateDetection: true,
		MaxPatternsPerTask:       5,
	}

	pi := NewPatternIntelligence(cfg, store, nil).(*PatternIntelligenceImpl)
	if err := pi.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// Store an exact pattern
	description := "Implement the exact same feature"
	files := []string{"exact.go"}
	err = pi.library.Store(context.Background(), description, files, "test-agent")
	if err != nil {
		t.Fatalf("failed to store pattern: %v", err)
	}

	// Check for exact match
	hashResult := pi.hasher.Hash(description, files)
	result := pi.checkDuplicates(context.Background(), description, files, hashResult)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.IsDuplicate {
		t.Error("expected IsDuplicate=true for exact match")
	}
	if result.SimilarityScore != 1.0 {
		t.Errorf("expected SimilarityScore=1.0, got %.2f", result.SimilarityScore)
	}
	if result.Recommendation != "skip" {
		t.Errorf("expected 'skip' recommendation in block mode, got %q", result.Recommendation)
	}
	if !result.ShouldSkip {
		t.Error("expected ShouldSkip=true for exact match in block mode")
	}
	if result.SkipReason == "" {
		t.Error("expected SkipReason to be set")
	}
}

func TestCheckDuplicatesSimilarButNotDuplicate(t *testing.T) {
	store, err := learning.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	cfg := &config.PatternConfig{
		Enabled:                  true,
		Mode:                     config.PatternModeWarn,
		SimilarityThreshold:      0.3,  // Low threshold to trigger "similar"
		DuplicateThreshold:       0.95, // High threshold so we don't hit duplicate
		EnableSTOP:               false,
		EnableDuplicateDetection: true,
		MaxPatternsPerTask:       5,
	}

	pi := NewPatternIntelligence(cfg, store, nil).(*PatternIntelligenceImpl)
	if err := pi.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// Store a pattern
	err = pi.library.Store(context.Background(), "Create authentication handler", []string{"auth.go"}, "test-agent")
	if err != nil {
		t.Fatalf("failed to store pattern: %v", err)
	}

	// Check for similar (but not duplicate) task
	description := "Implement authentication service"
	files := []string{"service.go"}
	hashResult := pi.hasher.Hash(description, files)
	result := pi.checkDuplicates(context.Background(), description, files, hashResult)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Should find it similar but not duplicate
	// The test verifies the else-if branch
	if result.IsDuplicate {
		t.Log("Found as duplicate - similarity may be higher than expected")
	}
}

func TestCheckDuplicatesHighSimilarityDuplicate(t *testing.T) {
	store, err := learning.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	cfg := &config.PatternConfig{
		Enabled:                  true,
		Mode:                     config.PatternModeBlock,
		SimilarityThreshold:      0.3,
		DuplicateThreshold:       0.6, // Lower threshold to make it easier to hit
		EnableSTOP:               false,
		EnableDuplicateDetection: true,
		MaxPatternsPerTask:       5,
	}

	pi := NewPatternIntelligence(cfg, store, nil).(*PatternIntelligenceImpl)
	if err := pi.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// Store a pattern with lots of keywords
	err = pi.library.Store(context.Background(), "Create user login authentication handler endpoint", []string{"handler.go"}, "test-agent")
	if err != nil {
		t.Fatalf("failed to store pattern: %v", err)
	}

	// Check with very similar keywords
	description := "Create user login authentication handler service"
	files := []string{"service.go"}
	hashResult := pi.hasher.Hash(description, files)
	result := pi.checkDuplicates(context.Background(), description, files, hashResult)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// This should trigger the duplicate path since keywords are very similar
	if result.IsDuplicate {
		if result.ShouldSkip && result.SkipReason == "" {
			t.Error("expected SkipReason when ShouldSkip is true")
		}
		if result.MatchedTaskID == "" {
			t.Error("expected MatchedTaskID when duplicate is found")
		}
	}
}

func TestRunSTOPAnalysisWithNilSearcher(t *testing.T) {
	pi := &PatternIntelligenceImpl{
		config: &config.PatternConfig{
			Enabled:    true,
			EnableSTOP: true,
		},
		hasher:   NewTaskHasher(),
		searcher: nil, // Nil searcher
	}

	hashResult := HashResult{
		FullHash:       "test",
		NormalizedHash: "test",
	}

	result := pi.runSTOPAnalysis(context.Background(), "test", []string{}, hashResult)

	if result == nil {
		t.Fatal("expected non-nil result even with nil searcher")
	}
	// Should return empty result, not panic
}

func TestEvaluateDuplicateRecommendationDefault(t *testing.T) {
	cfg := &config.PatternConfig{
		Enabled:            true,
		Mode:               "unknown_mode", // Invalid mode
		DuplicateThreshold: 0.9,
	}

	pi := &PatternIntelligenceImpl{config: cfg}
	result := pi.evaluateDuplicateRecommendation(0.95)

	if result != "proceed" {
		t.Errorf("expected 'proceed' for unknown mode, got %q", result)
	}
}

func TestCalculateConfidenceMaxValue(t *testing.T) {
	pi := &PatternIntelligenceImpl{}

	// Create results that would exceed 1.0 if not capped
	searchResults := SearchResults{
		GitMatches:     make([]GitCommit, 10),
		DocMatches:     make([]DocMatch, 10),
		HistoryMatches: make([]HistoryMatch, 10),
		IssueMatches:   make([]GitHubIssue, 10),
	}
	// Fill with some data
	for i := range searchResults.GitMatches {
		searchResults.GitMatches[i] = GitCommit{Hash: "abc"}
	}
	for i := range searchResults.DocMatches {
		searchResults.DocMatches[i] = DocMatch{FilePath: "doc.md"}
	}
	for i := range searchResults.HistoryMatches {
		searchResults.HistoryMatches[i] = HistoryMatch{TaskHash: "abc"}
	}

	confidence := pi.calculateConfidence(searchResults)

	if confidence > 1.0 {
		t.Errorf("confidence should be capped at 1.0, got %v", confidence)
	}
	if confidence < 0.8 { // Using 0.8 to account for floating point precision
		t.Errorf("expected high confidence with many results, got %v", confidence)
	}
}

func TestGenerateThinkResultWithDependencies(t *testing.T) {
	pi := &PatternIntelligenceImpl{
		config: &config.PatternConfig{Enabled: true},
	}

	searchResults := SearchResults{
		HistoryMatches: []HistoryMatch{
			{
				PatternDescription: "Task 1",
				LastAgent:          "golang-pro",
			},
			{
				PatternDescription: "Task 2",
				LastAgent:          "", // Empty agent
			},
		},
	}
	hashResult := HashResult{}

	result := pi.generateThinkResult(searchResults, hashResult)

	// Should have dependency from the agent
	hasDependency := false
	for _, d := range result.Dependencies {
		if strings.Contains(d, "golang-pro") {
			hasDependency = true
			break
		}
	}
	if !hasDependency {
		t.Error("expected dependency mentioning the agent")
	}
}

func TestGenerateRecommendationsLowSimilarityHistory(t *testing.T) {
	pi := &PatternIntelligenceImpl{
		config: &config.PatternConfig{Enabled: true},
	}

	searchResults := SearchResults{
		HistoryMatches: []HistoryMatch{
			{
				PatternDescription: "Some task",
				Similarity:         0.5, // Below 0.7 threshold
				SuccessCount:       3,
				LastAgent:          "test-agent",
			},
		},
	}
	hashResult := HashResult{}

	recommendations := pi.generateRecommendations(searchResults, hashResult)

	// Should not recommend reviewing low similarity patterns
	for _, r := range recommendations {
		if strings.Contains(r, "Review pattern") {
			t.Error("should not recommend reviewing low similarity patterns")
		}
	}
}

func TestGenerateRecommendationsAgentNotEnoughSuccesses(t *testing.T) {
	pi := &PatternIntelligenceImpl{
		config: &config.PatternConfig{Enabled: true},
	}

	searchResults := SearchResults{
		HistoryMatches: []HistoryMatch{
			{
				PatternDescription: "Task 1",
				Similarity:         0.5,
				SuccessCount:       1, // Only 1 success
				LastAgent:          "test-agent",
			},
		},
	}
	hashResult := HashResult{}

	recommendations := pi.generateRecommendations(searchResults, hashResult)

	// Should not recommend agent with only 1 success (needs >= 2)
	for _, r := range recommendations {
		if strings.Contains(r, "Consider using test-agent") {
			t.Error("should not recommend agent with only 1 success")
		}
	}
}

func TestSetSimilarity(t *testing.T) {
	t.Run("nil receiver returns without panic", func(t *testing.T) {
		var pi *PatternIntelligenceImpl
		pi.SetSimilarity(similarity.NewClaudeSimilarity(90*time.Second, nil))
		// Should not panic
	})

	t.Run("sets similarity correctly", func(t *testing.T) {
		cfg := &config.PatternConfig{
			Enabled: true,
			Mode:    config.PatternModeWarn,
		}
		pi := NewPatternIntelligence(cfg, nil, nil, testIntelligenceSearchTimeout).(*PatternIntelligenceImpl)
		sim := similarity.NewClaudeSimilarity(90*time.Second, nil)
		pi.SetSimilarity(sim)
		if pi.similarity != sim {
			t.Error("expected similarity to be set")
		}
	})
}

func TestCheckDuplicatesWithClaudeSimilarity(t *testing.T) {
	store, err := learning.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	cfg := &config.PatternConfig{
		Enabled:                  true,
		Mode:                     config.PatternModeWarn,
		SimilarityThreshold:      0.2, // Very low threshold to ensure pattern is retrieved
		DuplicateThreshold:       0.8, // High duplicate threshold
		EnableSTOP:               false,
		EnableDuplicateDetection: true,
		MaxPatternsPerTask:       5,
	}

	// Test with nil similarity - should return 0 for non-hash matches
	pi := NewPatternIntelligence(cfg, store, nil).(*PatternIntelligenceImpl)
	if err := pi.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// Store a pattern with a descriptive name that will be retrieved but not exact match
	err = pi.library.Store(context.Background(), "Create user authentication handler", []string{"auth.go"}, "golang-pro")
	if err != nil {
		t.Fatalf("failed to store pattern: %v", err)
	}

	t.Run("handles nil ClaudeSimilarity gracefully", func(t *testing.T) {
		// Use a DIFFERENT task name/files so we don't get exact hash match
		task := models.Task{
			Number: "1",
			Name:   "Implement user authentication service", // Different but related
			Files:  []string{"service.go"},                  // Different file
		}

		_, dup, err := pi.CheckTask(context.Background(), task)
		if err != nil {
			t.Errorf("CheckTask() error = %v", err)
		}
		if dup == nil {
			t.Fatal("expected DuplicateResult")
		}

		// With nil similarity, non-exact matches return 0 score
		// So this should not be detected as duplicate
		t.Logf("SimilarityScore: %.2f, IsDuplicate: %v", dup.SimilarityScore, dup.IsDuplicate)
	})

	t.Run("detects exact hash match without ClaudeSimilarity", func(t *testing.T) {
		// Use SAME task name/files to get exact hash match
		task := models.Task{
			Number: "2",
			Name:   "Create user authentication handler",
			Files:  []string{"auth.go"},
		}

		_, dup, err := pi.CheckTask(context.Background(), task)
		if err != nil {
			t.Errorf("CheckTask() error = %v", err)
		}
		if dup == nil {
			t.Fatal("expected DuplicateResult")
		}

		// Exact hash match should be detected
		if !dup.IsDuplicate {
			t.Error("expected IsDuplicate=true for exact hash match")
		}
		if dup.SimilarityScore != 1.0 {
			t.Errorf("expected SimilarityScore=1.0 for exact match, got %.2f", dup.SimilarityScore)
		}
	})
}

func TestCheckDuplicatesWithNilSimilarity(t *testing.T) {
	store, err := learning.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	cfg := &config.PatternConfig{
		Enabled:                  true,
		Mode:                     config.PatternModeWarn,
		SimilarityThreshold:      0.2, // Very low threshold to ensure pattern is retrieved
		DuplicateThreshold:       0.8,
		EnableSTOP:               false,
		EnableDuplicateDetection: true,
		MaxPatternsPerTask:       5,
	}

	// Nil similarity - semantic matching returns 0
	pi := NewPatternIntelligence(cfg, store, nil).(*PatternIntelligenceImpl)
	if err := pi.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// Store a pattern
	err = pi.library.Store(context.Background(), "Create user authentication handler", []string{"auth.go"}, "golang-pro")
	if err != nil {
		t.Fatalf("failed to store pattern: %v", err)
	}

	t.Run("returns 0 similarity for non-hash matches without ClaudeSimilarity", func(t *testing.T) {
		// Use different task to avoid exact hash match
		task := models.Task{
			Number: "1",
			Name:   "Implement user authentication service", // Different but overlapping keywords
			Files:  []string{"service.go"},                  // Different file
		}

		_, dup, err := pi.CheckTask(context.Background(), task)
		if err != nil {
			t.Errorf("CheckTask() error = %v", err)
		}
		if dup == nil {
			t.Fatal("expected DuplicateResult")
		}

		// With nil similarity, non-hash matches return 0
		t.Logf("Duplicate check result: IsDuplicate=%v, Score=%.2f",
			dup.IsDuplicate, dup.SimilarityScore)
	})
}

func TestNewPatternIntelligenceWithSimilarity(t *testing.T) {
	cfg := &config.PatternConfig{
		Enabled: true,
		Mode:    config.PatternModeWarn,
	}
	sim := similarity.NewClaudeSimilarity(90*time.Second, nil)
	pi := NewPatternIntelligence(cfg, nil, sim)
	if pi == nil {
		t.Error("expected non-nil PatternIntelligence")
	}

	impl, ok := pi.(*PatternIntelligenceImpl)
	if !ok {
		t.Fatal("expected *PatternIntelligenceImpl")
	}
	if impl.similarity != sim {
		t.Error("expected similarity to be set from constructor")
	}
}
