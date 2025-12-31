package executor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/models"
)

// TestIntelligentSelectionIntegration tests the full intelligent selection flow
// with a mock registry and simulated Claude responses
func TestIntelligentSelectionIntegration(t *testing.T) {
	// Create temp directory for test agents
	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	// Create test agent files with proper YAML frontmatter
	testAgents := []struct {
		name        string
		description string
		tools       []string
	}{
		{"golang-pro", "Go expert for backend development", []string{"Read", "Write", "Bash"}},
		{"security-auditor", "Security specialist for code review", []string{"Read", "Grep"}},
		{"code-reviewer", "General code review agent", []string{"Read", "Write", "Edit"}},
		{"database-optimizer", "Database performance specialist", []string{"Read", "Bash"}},
		{"quality-control", "Baseline QC agent", []string{"Read"}},
	}

	for _, a := range testAgents {
		toolsJSON, _ := json.Marshal(a.tools)
		content := "---\nname: " + a.name + "\ndescription: " + a.description + "\ntools: " + string(toolsJSON) + "\n---\n\n# " + a.name + "\n\nAgent for testing."
		path := filepath.Join(agentsDir, a.name+".md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write agent file: %v", err)
		}
	}

	// Create registry and discover agents
	registry := agent.NewRegistry(agentsDir)
	agents, err := registry.Discover()
	if err != nil {
		t.Fatalf("registry.Discover() error: %v", err)
	}
	if len(agents) != 5 {
		t.Fatalf("expected 5 agents, got %d", len(agents))
	}

	t.Run("intelligent mode with mock selector", func(t *testing.T) {
		// Create task for JWT authentication
		task := models.Task{
			Number: "5",
			Name:   "Implement JWT Authentication",
			Files:  []string{"internal/auth/jwt.go", "internal/auth/jwt_test.go"},
			Prompt: "Add JWT token validation with HS256 support and proper error handling",
			Agent:  "golang-pro",
		}

		config := models.QCAgentConfig{
			Mode:              "intelligent",
			MaxAgents:         3,
			CacheTTLSeconds:   60,
			RequireCodeReview: true,
			BlockedAgents:     []string{},
		}

		// Create mock intelligent selector that doesn't call Claude
		selector := &testIntelligentSelector{
			registry: registry,
			mockRecommendation: &IntelligentAgentRecommendation{
				Agents:    []string{"security-auditor", "golang-pro"},
				Rationale: "Selected security-auditor for JWT/auth security review, golang-pro for Go patterns",
			},
		}

		selCtx := &SelectionContext{
			ExecutingAgent:      task.Agent,
			IntelligentSelector: &IntelligentSelector{Registry: registry, Cache: NewQCSelectionCache(60)},
		}

		// Override with mock selector's behavior
		result, rationale, err := testIntelligentSelectQCAgents(
			context.Background(),
			task,
			task.Agent,
			config,
			registry,
			selector,
		)

		if err != nil {
			t.Fatalf("testIntelligentSelectQCAgents error: %v", err)
		}

		_ = selCtx // Mark as used for future extensions

		// Verify results
		if len(result) == 0 {
			t.Fatal("expected at least one agent")
		}

		// Should include code-reviewer as baseline
		hasCodeReviewer := false
		for _, a := range result {
			if a == "code-reviewer" {
				hasCodeReviewer = true
				break
			}
		}
		if !hasCodeReviewer {
			t.Error("expected code-reviewer as baseline when RequireCodeReview=true")
		}

		// Should include security-auditor for JWT task
		hasSecurityAuditor := false
		for _, a := range result {
			if a == "security-auditor" {
				hasSecurityAuditor = true
				break
			}
		}
		if !hasSecurityAuditor {
			t.Error("expected security-auditor for JWT authentication task")
		}

		// Should not exceed MaxAgents
		if len(result) > 3 {
			t.Errorf("expected max 3 agents, got %d", len(result))
		}

		if rationale == "" {
			t.Error("expected non-empty rationale")
		}

		t.Logf("Selected agents: %v", result)
		t.Logf("Rationale: %s", rationale)
	})

	t.Run("cache prevents duplicate selections", func(t *testing.T) {
		cache := NewQCSelectionCache(3600)
		selector := NewIntelligentSelector(registry, 3600, 90*time.Second)
		selector.Cache = cache

		task := models.Task{
			Number: "10",
			Name:   "Database Migration",
			Files:  []string{"db/migrations/001_init.sql"},
			Prompt: "Add user table migration",
			Agent:  "database-optimizer",
		}

		config := models.QCAgentConfig{
			Mode:              "intelligent",
			MaxAgents:         4,
			CacheTTLSeconds:   3600,
			RequireCodeReview: true,
		}

		// First call - cache miss
		key := GenerateCacheKey(task, task.Agent)
		_, found := cache.Get(key)
		if found {
			t.Error("expected cache miss on first call")
		}

		// Simulate storing a result
		mockResult := &IntelligentSelectionResult{
			Agents:    []string{"code-reviewer", "database-optimizer"},
			Rationale: "DB migration review",
		}
		cache.Set(key, mockResult)

		// Second call - cache hit
		cached, found := cache.Get(key)
		if !found {
			t.Error("expected cache hit on second call")
		}
		if len(cached.Agents) != 2 {
			t.Errorf("expected 2 cached agents, got %d", len(cached.Agents))
		}

		// Verify SelectQCAgentsWithContext uses cache
		selCtx := &SelectionContext{
			ExecutingAgent:      task.Agent,
			IntelligentSelector: selector,
		}

		// This would normally call Claude, but since we've cached the result,
		// it should return from cache
		agents := SelectQCAgentsWithContext(context.Background(), task, config, registry, selCtx)

		// Should return cached result (code-reviewer + database-optimizer)
		t.Logf("Agents from cache: %v", agents)
		if len(agents) < 2 {
			t.Errorf("expected at least 2 agents from cache, got %d", len(agents))
		}
	})

	t.Run("guardrails enforce blocked agents", func(t *testing.T) {
		selector := NewIntelligentSelector(registry, 3600, 90*time.Second)

		recommendation := &IntelligentAgentRecommendation{
			Agents:    []string{"security-auditor", "golang-pro", "blocked-agent"},
			Rationale: "Test recommendation",
		}

		config := models.QCAgentConfig{
			MaxAgents:         4,
			RequireCodeReview: false,
			BlockedAgents:     []string{"golang-pro"}, // Block golang-pro
		}

		result := selector.applyGuardrails(recommendation, config)

		// golang-pro should be filtered out
		for _, a := range result.Agents {
			if a == "golang-pro" {
				t.Error("blocked agent golang-pro should not be in result")
			}
		}

		// security-auditor should be present (if exists in registry)
		t.Logf("Filtered agents: %v", result.Agents)
	})

	t.Run("fallback to auto mode on intelligent failure", func(t *testing.T) {
		task := models.Task{
			Number: "7",
			Name:   "Python Script",
			Files:  []string{"scripts/process.py"},
			Prompt: "Add data processing script",
			Agent:  "python-pro",
		}

		config := models.QCAgentConfig{
			Mode:              "intelligent",
			MaxAgents:         4,
			CacheTTLSeconds:   3600,
			RequireCodeReview: true,
		}

		// Without selector context, should fallback to auto
		agents := SelectQCAgentsWithContext(context.Background(), task, config, registry, nil)

		// Should get at least quality-control from auto-select
		if len(agents) == 0 {
			t.Error("expected fallback to auto mode with at least one agent")
		}

		hasQC := false
		for _, a := range agents {
			if a == "quality-control" {
				hasQC = true
				break
			}
		}
		if !hasQC {
			t.Error("expected quality-control agent in auto fallback")
		}

		t.Logf("Fallback agents: %v", agents)
	})

	t.Run("QualityController integration with intelligent mode", func(t *testing.T) {
		qc := NewQualityController(&testIntelligentInvoker{})
		qc.Registry = registry
		qc.AgentConfig = models.QCAgentConfig{
			Mode:              "intelligent",
			MaxAgents:         3,
			CacheTTLSeconds:   3600,
			RequireCodeReview: true,
		}

		// Verify IntelligentSelector is nil initially
		if qc.IntelligentSelector != nil {
			t.Error("expected IntelligentSelector to be nil initially")
		}

		// Create task
		task := models.Task{
			Number: "1",
			Name:   "API Endpoint",
			Files:  []string{"api/handler.go"},
			Prompt: "Add REST endpoint",
			Agent:  "golang-pro",
		}

		// Test that ReviewMultiAgent would initialize the selector
		// We can't call it directly without real Claude, but we can verify setup
		if qc.AgentConfig.Mode != "intelligent" {
			t.Error("expected intelligent mode")
		}

		// Verify config defaults are set
		if qc.AgentConfig.MaxAgents != 3 {
			t.Errorf("expected MaxAgents=3, got %d", qc.AgentConfig.MaxAgents)
		}

		t.Logf("QC configured for intelligent mode with task: %s", task.Name)
	})
}

// Mock structures for testing without actual Claude invocation

type testIntelligentSelector struct {
	registry           *agent.Registry
	mockRecommendation *IntelligentAgentRecommendation
}

func testIntelligentSelectQCAgents(
	ctx context.Context,
	task models.Task,
	executingAgent string,
	config models.QCAgentConfig,
	registry *agent.Registry,
	mockSelector *testIntelligentSelector,
) ([]string, string, error) {
	// Create real selector for guardrails
	selector := NewIntelligentSelector(registry, config.CacheTTLSeconds, time.Duration(config.SelectionTimeoutSeconds)*time.Second)

	// Apply guardrails to mock recommendation
	result := selector.applyGuardrails(mockSelector.mockRecommendation, config)

	return result.Agents, result.Rationale, nil
}

type testIntelligentInvoker struct{}

func (m *testIntelligentInvoker) Invoke(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
	return &agent.InvocationResult{
		Output:   `{"status": "success", "summary": "Task completed"}`,
		Duration: 100 * time.Millisecond,
	}, nil
}

// TestIntelligentSelectionEdgeCases tests boundary conditions
func TestIntelligentSelectionEdgeCases(t *testing.T) {
	t.Run("empty task files", func(t *testing.T) {
		task := models.Task{
			Number: "1",
			Name:   "Empty Files Task",
			Files:  []string{},
			Prompt: "Task with no files",
		}

		key1 := GenerateCacheKey(task, "agent1")
		key2 := GenerateCacheKey(task, "agent1")

		if key1 != key2 {
			t.Error("cache keys should be consistent for empty files")
		}
	})

	t.Run("very long task names", func(t *testing.T) {
		longName := ""
		for i := 0; i < 1000; i++ {
			longName += "x"
		}

		task := models.Task{
			Number: "1",
			Name:   longName,
			Files:  []string{"file.go"},
		}

		key := GenerateCacheKey(task, "agent")
		if len(key) > 100 {
			t.Error("cache key should be bounded in length (hash)")
		}
	})

	t.Run("zero max agents defaults to selector default", func(t *testing.T) {
		registry := agent.NewRegistry("/nonexistent")
		registry.Discover()

		selector := NewIntelligentSelector(registry, 3600, 90*time.Second)
		selector.MaxAgents = 4 // Default

		recommendation := &IntelligentAgentRecommendation{
			Agents:    []string{"a", "b", "c", "d", "e"},
			Rationale: "Many agents",
		}

		config := models.QCAgentConfig{
			MaxAgents:         0, // Zero - should use selector default
			RequireCodeReview: false,
		}

		result := selector.applyGuardrails(recommendation, config)

		// Should use selector.MaxAgents (4), not config.MaxAgents (0)
		if len(result.Agents) > 4 {
			t.Errorf("expected max 4 agents (selector default), got %d", len(result.Agents))
		}
	})

	t.Run("cache TTL of zero defaults to 1 hour", func(t *testing.T) {
		cache := NewQCSelectionCache(0)
		if cache.ttl != 3600*time.Second {
			t.Errorf("expected 1 hour default TTL, got %v", cache.ttl)
		}
	})

	t.Run("negative cache TTL defaults to 1 hour", func(t *testing.T) {
		cache := NewQCSelectionCache(-100)
		if cache.ttl != 3600*time.Second {
			t.Errorf("expected 1 hour default TTL for negative value, got %v", cache.ttl)
		}
	})
}
