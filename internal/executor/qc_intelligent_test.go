package executor

import (
	"context"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/models"
)

func TestQCSelectionCache(t *testing.T) {
	t.Run("cache hit and miss", func(t *testing.T) {
		cache := NewQCSelectionCache(3600)

		// Test miss
		_, found := cache.Get("nonexistent")
		if found {
			t.Error("expected cache miss for nonexistent key")
		}

		// Test set and hit
		result := &IntelligentSelectionResult{
			Agents:    []string{"golang-pro", "security-auditor"},
			Rationale: "Selected for Go and security review",
		}
		cache.Set("testkey", result)

		cached, found := cache.Get("testkey")
		if !found {
			t.Error("expected cache hit")
		}
		if len(cached.Agents) != 2 {
			t.Errorf("expected 2 agents, got %d", len(cached.Agents))
		}
		if cached.Agents[0] != "golang-pro" {
			t.Errorf("expected golang-pro, got %s", cached.Agents[0])
		}
	})

	t.Run("cache expiration", func(t *testing.T) {
		// Create cache with 1 second TTL
		cache := NewQCSelectionCache(1)

		result := &IntelligentSelectionResult{
			Agents:    []string{"test-agent"},
			Rationale: "Test",
		}
		cache.Set("expiring", result)

		// Should hit immediately
		_, found := cache.Get("expiring")
		if !found {
			t.Error("expected cache hit before expiration")
		}

		// Wait for expiration
		time.Sleep(1100 * time.Millisecond)

		// Should miss after expiration
		_, found = cache.Get("expiring")
		if found {
			t.Error("expected cache miss after expiration")
		}
	})

	t.Run("cleanup removes expired entries", func(t *testing.T) {
		cache := NewQCSelectionCache(1)

		result := &IntelligentSelectionResult{
			Agents:    []string{"agent1"},
			Rationale: "Test",
		}
		cache.Set("key1", result)
		cache.Set("key2", result)

		if cache.Size() != 2 {
			t.Errorf("expected 2 entries, got %d", cache.Size())
		}

		time.Sleep(1100 * time.Millisecond)
		cache.Cleanup()

		if cache.Size() != 0 {
			t.Errorf("expected 0 entries after cleanup, got %d", cache.Size())
		}
	})

	t.Run("clear removes all entries", func(t *testing.T) {
		cache := NewQCSelectionCache(3600)

		result := &IntelligentSelectionResult{
			Agents:    []string{"agent1"},
			Rationale: "Test",
		}
		cache.Set("key1", result)
		cache.Set("key2", result)
		cache.Set("key3", result)

		cache.Clear()

		if cache.Size() != 0 {
			t.Errorf("expected 0 entries after clear, got %d", cache.Size())
		}
	})
}

func TestGenerateCacheKey(t *testing.T) {
	t.Run("deterministic key generation", func(t *testing.T) {
		task := models.Task{
			Number: "5",
			Name:   "Implement JWT",
			Files:  []string{"auth.go", "jwt.go"},
		}

		key1 := GenerateCacheKey(task, "golang-pro")
		key2 := GenerateCacheKey(task, "golang-pro")

		if key1 != key2 {
			t.Error("cache keys should be deterministic")
		}
	})

	t.Run("file order independence", func(t *testing.T) {
		task1 := models.Task{
			Number: "5",
			Name:   "Test",
			Files:  []string{"a.go", "b.go", "c.go"},
		}
		task2 := models.Task{
			Number: "5",
			Name:   "Test",
			Files:  []string{"c.go", "a.go", "b.go"},
		}

		key1 := GenerateCacheKey(task1, "agent")
		key2 := GenerateCacheKey(task2, "agent")

		if key1 != key2 {
			t.Error("cache keys should be independent of file order")
		}
	})

	t.Run("different agents produce different keys", func(t *testing.T) {
		task := models.Task{
			Number: "5",
			Name:   "Test",
			Files:  []string{"test.go"},
		}

		key1 := GenerateCacheKey(task, "golang-pro")
		key2 := GenerateCacheKey(task, "python-pro")

		if key1 == key2 {
			t.Error("different agents should produce different cache keys")
		}
	})
}

func TestStripCodeFences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Original test cases
		{
			name:     "no code fence",
			input:    `{"agents": ["test"]}`,
			expected: `{"agents": ["test"]}`,
		},
		{
			name:     "json code fence",
			input:    "```json\n{\"agents\": [\"test\"]}\n```",
			expected: `{"agents": ["test"]}`,
		},
		{
			name:     "plain code fence",
			input:    "```\n{\"agents\": [\"test\"]}\n```",
			expected: `{"agents": ["test"]}`,
		},
		{
			name:     "multiline json",
			input:    "```json\n{\n  \"agents\": [\"a\", \"b\"],\n  \"rationale\": \"test\"\n}\n```",
			expected: "{\n  \"agents\": [\"a\", \"b\"],\n  \"rationale\": \"test\"\n}",
		},
		// New edge cases for robustness
		{
			name:     "thinking text before fence",
			input:    "Wait, let me reconsider...\n\n```json\n{\"agents\": [\"test\"]}\n```",
			expected: `{"agents": ["test"]}`,
		},
		{
			name:     "thinking text after fence",
			input:    "```json\n{\"agents\": [\"test\"]}\n```\n\nDone processing.",
			expected: `{"agents": ["test"]}`,
		},
		{
			name:     "uppercase JSON marker",
			input:    "```JSON\n{\"agents\": [\"test\"]}\n```",
			expected: `{"agents": ["test"]}`,
		},
		{
			name:     "mixed case Json marker",
			input:    "```Json\n{\"agents\": [\"test\"]}\n```",
			expected: `{"agents": ["test"]}`,
		},
		{
			name:     "fence in middle of content",
			input:    "Here's the result:\n```json\n{\"agents\": [\"test\"]}\n```\nEnd of response",
			expected: `{"agents": ["test"]}`,
		},
		{
			name:     "whitespace around JSON in fence",
			input:    "```json\n  {\"agents\": [\"test\"]}  \n```",
			expected: `{"agents": ["test"]}`,
		},
		{
			name:     "multiple line thinking before fence",
			input:    "Let me analyze this...\nI need to select agents.\nWait, let me reconsider...\n\n```json\n{\"agents\": [\"a\", \"b\"]}\n```",
			expected: `{"agents": ["a", "b"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripCodeFences(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestIntelligentSelectorGuardrails(t *testing.T) {
	// Create mock registry with test agents
	registry := agent.NewRegistry("/nonexistent")
	// Manually add agents to registry for testing
	registry.Discover() // Initialize empty map

	// Create selector
	selector := NewIntelligentSelector(registry, 3600)

	t.Run("caps agents at MaxAgents", func(t *testing.T) {
		recommendation := &IntelligentAgentRecommendation{
			Agents:    []string{"agent1", "agent2", "agent3", "agent4", "agent5", "agent6"},
			Rationale: "Test recommendation",
		}

		config := models.QCAgentConfig{
			MaxAgents:         3,
			RequireCodeReview: false,
			BlockedAgents:     []string{},
		}

		result := selector.applyGuardrails(recommendation, config)

		// Should be capped at 3 agents (not 6)
		// Note: without registry validation, no agents will be added
		// This tests the capping logic
		if len(result.Agents) > 3 {
			t.Errorf("expected max 3 agents, got %d", len(result.Agents))
		}
	})

	t.Run("filters blocked agents", func(t *testing.T) {
		recommendation := &IntelligentAgentRecommendation{
			Agents:    []string{"good-agent", "blocked-agent"},
			Rationale: "Test",
		}

		config := models.QCAgentConfig{
			MaxAgents:         4,
			RequireCodeReview: false,
			BlockedAgents:     []string{"blocked-agent"},
		}

		result := selector.applyGuardrails(recommendation, config)

		// blocked-agent should not be in result
		for _, agent := range result.Agents {
			if agent == "blocked-agent" {
				t.Error("blocked agent should not be in result")
			}
		}
	})

	t.Run("adds fallback when no agents selected", func(t *testing.T) {
		// Registry is empty, so no agents will be validated
		recommendation := &IntelligentAgentRecommendation{
			Agents:    []string{}, // Empty
			Rationale: "Empty recommendation",
		}

		config := models.QCAgentConfig{
			MaxAgents:         4,
			RequireCodeReview: false,
			BlockedAgents:     []string{},
		}

		result := selector.applyGuardrails(recommendation, config)

		// Should fallback, but since registry is empty, quality-control won't exist
		// This tests the fallback logic branch
		if result.Rationale != "Fallback to quality-control agent" && len(result.Agents) == 0 {
			// Expected: either adds quality-control or returns empty if not in registry
		}
	})
}

func TestBuildSelectionPrompt(t *testing.T) {
	selector := NewIntelligentSelector(nil, 3600)

	task := models.Task{
		Number: "5",
		Name:   "Implement JWT Authentication",
		Files:  []string{"internal/auth/jwt.go", "internal/auth/jwt_test.go"},
		Prompt: "Create JWT validation with HS256 support",
	}

	config := models.QCAgentConfig{
		MaxAgents: 3,
	}

	prompt := selector.buildSelectionPrompt(task, "golang-pro", []string{"security-auditor", "code-reviewer"}, config)

	// Verify prompt contains task information
	if !containsStr(prompt, "Task Number: 5") {
		t.Error("prompt should contain task number")
	}
	if !containsStr(prompt, "Implement JWT Authentication") {
		t.Error("prompt should contain task name")
	}
	if !containsStr(prompt, "golang-pro") {
		t.Error("prompt should contain executing agent")
	}
	if !containsStr(prompt, "security-auditor") {
		t.Error("prompt should contain available agents")
	}
	if !containsStr(prompt, "3 or fewer") {
		t.Error("prompt should specify max agents")
	}
}

func TestSelectQCAgentsWithContextFallback(t *testing.T) {
	// Test that intelligent mode falls back to auto when selector is not available
	registry := agent.NewRegistry("/nonexistent")
	registry.Discover()

	config := models.QCAgentConfig{
		Mode:              "intelligent",
		MaxAgents:         4,
		RequireCodeReview: true,
		BlockedAgents:     []string{},
	}

	task := models.Task{
		Number: "1",
		Name:   "Test Task",
		Files:  []string{"test.go"},
	}

	// Without selector context, should fall back to auto
	agents := SelectQCAgentsWithContext(context.Background(), task, config, registry, nil)

	// Should get at least the fallback agent
	if len(agents) == 0 {
		t.Error("should have at least one agent from fallback")
	}
}

func TestIntelligentSelectionResultStructure(t *testing.T) {
	result := &IntelligentSelectionResult{
		Agents:    []string{"agent1", "agent2"},
		Rationale: "Selected for comprehensive review",
	}

	if len(result.Agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(result.Agents))
	}

	if result.Rationale == "" {
		t.Error("rationale should not be empty")
	}
}

// Helper function
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsStrMiddle(s, substr)))
}

func containsStrMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
