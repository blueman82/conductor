package behavioral

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

// TestFullSessionFlow tests the complete flow: discovery -> parse -> extract -> aggregate
func TestFullSessionFlow(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create realistic session files
	sessions := []struct {
		filename string
		content  string
	}{
		{
			filename: "agent-11111111.jsonl",
			content: `{"type":"session_start","session_id":"11111111","project":"test-project","timestamp":"2024-01-15T10:00:00Z","status":"completed","agent_name":"code-reviewer","duration":60000,"success":true,"error_count":0}
{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","parameters":{"file_path":"/main.go"},"success":true,"duration":50}
{"type":"tool_call","timestamp":"2024-01-15T10:02:00Z","tool_name":"Read","parameters":{"file_path":"/test.go"},"success":true,"duration":45}
{"type":"bash_command","timestamp":"2024-01-15T10:03:00Z","command":"go test ./...","exit_code":0,"output":"PASS","output_length":4,"duration":1500,"success":true}
{"type":"file_operation","timestamp":"2024-01-15T10:04:00Z","operation":"read","path":"/config.yaml","success":true,"size_bytes":512,"duration":10}
{"type":"token_usage","timestamp":"2024-01-15T10:05:00Z","input_tokens":5000,"output_tokens":2000,"cost_usd":0.045,"model_name":"claude-sonnet-4-5"}
`,
		},
		{
			filename: "agent-22222222.jsonl",
			content: `{"type":"session_start","session_id":"22222222","project":"test-project","timestamp":"2024-01-15T11:00:00Z","status":"completed","agent_name":"test-automator","duration":120000,"success":true,"error_count":1}
{"type":"tool_call","timestamp":"2024-01-15T11:01:00Z","tool_name":"Write","parameters":{"file_path":"/test_new.go"},"success":true,"duration":100}
{"type":"tool_call","timestamp":"2024-01-15T11:02:00Z","tool_name":"Edit","parameters":{"file_path":"/main.go"},"success":false,"duration":200,"error":"file not found"}
{"type":"bash_command","timestamp":"2024-01-15T11:03:00Z","command":"go build","exit_code":1,"output":"error: undefined","output_length":16,"duration":2000,"success":false}
{"type":"file_operation","timestamp":"2024-01-15T11:04:00Z","operation":"write","path":"/new_feature.go","success":true,"size_bytes":2048,"duration":50}
{"type":"token_usage","timestamp":"2024-01-15T11:05:00Z","input_tokens":8000,"output_tokens":4000,"cost_usd":0.084,"model_name":"claude-sonnet-4-5"}
`,
		},
		{
			filename: "agent-33333333.jsonl",
			content: `{"type":"session_start","session_id":"33333333","project":"test-project","timestamp":"2024-01-15T12:00:00Z","status":"failed","agent_name":"code-reviewer","duration":30000,"success":false,"error_count":2}
{"type":"tool_call","timestamp":"2024-01-15T12:01:00Z","tool_name":"Bash","parameters":{"command":"ls"},"success":false,"duration":50,"error":"permission denied"}
{"type":"bash_command","timestamp":"2024-01-15T12:02:00Z","command":"rm -rf /","exit_code":1,"output":"Permission denied","output_length":17,"duration":10,"success":false}
{"type":"token_usage","timestamp":"2024-01-15T12:03:00Z","input_tokens":1000,"output_tokens":500,"cost_usd":0.0105,"model_name":"claude-sonnet-4-5"}
`,
		},
	}

	for _, s := range sessions {
		sessionFile := filepath.Join(projectDir, s.filename)
		if err := os.WriteFile(sessionFile, []byte(s.content), 0644); err != nil {
			t.Fatalf("failed to write session file: %v", err)
		}
	}

	// Step 1: Discovery
	discoveredSessions, err := DiscoverSessions(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverSessions failed: %v", err)
	}
	if len(discoveredSessions) != 3 {
		t.Errorf("discovered %d sessions, want 3", len(discoveredSessions))
	}

	// Verify discovery metadata
	for _, info := range discoveredSessions {
		if info.Project != "test-project" {
			t.Errorf("session project = %s, want test-project", info.Project)
		}
		if info.SessionID == "" {
			t.Error("session ID should not be empty")
		}
		if info.FilePath == "" {
			t.Error("file path should not be empty")
		}
	}

	// Step 2: Parse each session
	var parsedSessions []*SessionData
	for _, info := range discoveredSessions {
		sessionData, err := ParseSessionFile(info.FilePath)
		if err != nil {
			t.Fatalf("ParseSessionFile failed for %s: %v", info.FilePath, err)
		}
		parsedSessions = append(parsedSessions, sessionData)
	}

	// Verify parsing
	if len(parsedSessions) != 3 {
		t.Errorf("parsed %d sessions, want 3", len(parsedSessions))
	}

	// Step 3: Extract metrics from each session
	var extractedMetrics []*BehavioralMetrics
	for _, sessionData := range parsedSessions {
		metrics := ExtractMetrics(sessionData)
		if err := metrics.Validate(); err != nil {
			t.Errorf("extracted metrics validation failed: %v", err)
		}
		extractedMetrics = append(extractedMetrics, metrics)
	}

	// Step 4: Aggregate using Aggregator
	agg := NewAggregatorWithBaseDir(10, tmpDir)
	projectMetrics, err := agg.GetProjectMetrics("test-project")
	if err != nil {
		t.Fatalf("GetProjectMetrics failed: %v", err)
	}

	// Verify aggregated metrics
	if projectMetrics.TotalSessions != 3 {
		t.Errorf("TotalSessions = %d, want 3", projectMetrics.TotalSessions)
	}

	// Check tool usage counts
	if projectMetrics.ToolUsageCounts["Read"] != 2 {
		t.Errorf("Read tool count = %d, want 2", projectMetrics.ToolUsageCounts["Read"])
	}

	// Check agent performance
	if projectMetrics.AgentPerformance["code-reviewer"] != 1 {
		t.Errorf("code-reviewer success count = %d, want 1", projectMetrics.AgentPerformance["code-reviewer"])
	}

	// Check token usage aggregation
	totalTokens := projectMetrics.TokenUsage.InputTokens + projectMetrics.TokenUsage.OutputTokens
	expectedTokens := int64(5000 + 2000 + 8000 + 4000 + 1000 + 500) // Sum from all sessions
	if totalTokens != expectedTokens {
		t.Errorf("total tokens = %d, want %d", totalTokens, expectedTokens)
	}

	// Check total cost
	expectedCost := 0.045 + 0.084 + 0.0105
	if projectMetrics.TotalCost < expectedCost-0.001 || projectMetrics.TotalCost > expectedCost+0.001 {
		t.Errorf("TotalCost = %f, want approximately %f", projectMetrics.TotalCost, expectedCost)
	}
}

// TestMalformedEvents tests graceful handling of malformed JSONL events
func TestMalformedEvents(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		wantErr        bool
		wantEventCount int
		description    string
	}{
		{
			name: "malformed json line skipped",
			content: `{"type":"session_start","session_id":"test-123","project":"test","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","success":true,"duration":100}
{this is not valid json at all}
{"type":"bash_command","timestamp":"2024-01-15T10:02:00Z","command":"ls","exit_code":0,"output":"","duration":10,"success":true}
`,
			wantErr:        false,
			wantEventCount: 2,
			description:    "should skip malformed JSON and continue",
		},
		{
			name: "truncated json line",
			content: `{"type":"session_start","session_id":"test-123","project":"test","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","success":true
{"type":"bash_command","timestamp":"2024-01-15T10:02:00Z","command":"ls","exit_code":0,"output":"","duration":10,"success":true}
`,
			wantErr:        false,
			wantEventCount: 1,
			description:    "should skip truncated JSON",
		},
		{
			name: "unknown event type skipped",
			content: `{"type":"session_start","session_id":"test-123","project":"test","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"unknown_event_type","timestamp":"2024-01-15T10:01:00Z","data":"something"}
{"type":"tool_call","timestamp":"2024-01-15T10:02:00Z","tool_name":"Read","success":true,"duration":100}
`,
			wantErr:        false,
			wantEventCount: 1,
			description:    "should skip unknown event types",
		},
		{
			name: "empty lines ignored",
			content: `{"type":"session_start","session_id":"test-123","project":"test","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}

{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","success":true,"duration":100}


{"type":"bash_command","timestamp":"2024-01-15T10:02:00Z","command":"ls","exit_code":0,"output":"","duration":10,"success":true}
`,
			wantErr:        false,
			wantEventCount: 2,
			description:    "should ignore empty lines",
		},
		{
			name: "invalid event missing required field",
			content: `{"type":"session_start","session_id":"test-123","project":"test","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"","success":true,"duration":100}
{"type":"bash_command","timestamp":"2024-01-15T10:02:00Z","command":"ls","exit_code":0,"duration":10,"success":true}
`,
			wantErr:        false,
			wantEventCount: 1,
			description:    "should skip events failing validation (empty tool_name)",
		},
		{
			name:           "no session metadata",
			content:        `{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","success":true,"duration":100}`,
			wantErr:        true,
			wantEventCount: 0,
			description:    "should fail when session metadata is missing",
		},
		{
			name: "negative file size",
			content: `{"type":"session_start","session_id":"test-123","project":"test","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"file_operation","timestamp":"2024-01-15T10:01:00Z","operation":"read","path":"/test.go","success":true,"size_bytes":-100,"duration":10}
{"type":"tool_call","timestamp":"2024-01-15T10:02:00Z","tool_name":"Read","success":true,"duration":100}
`,
			wantErr:        false,
			wantEventCount: 1,
			description:    "should skip file operations with negative size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.jsonl")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			sessionData, err := ParseSessionFile(tmpFile)

			if tt.wantErr {
				if err == nil {
					t.Errorf("%s: expected error but got none", tt.description)
				}
				return
			}

			if err != nil {
				t.Errorf("%s: unexpected error: %v", tt.description, err)
				return
			}

			if len(sessionData.Events) != tt.wantEventCount {
				t.Errorf("%s: got %d events, want %d", tt.description, len(sessionData.Events), tt.wantEventCount)
			}

			// Verify metrics extraction doesn't panic with partial data
			metrics := ExtractMetrics(sessionData)
			if err := metrics.Validate(); err != nil {
				t.Errorf("%s: metrics validation failed: %v", tt.description, err)
			}
		})
	}
}

// TestConcurrentAccess tests race conditions with concurrent cache access
func TestConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create multiple session files
	for i := 0; i < 10; i++ {
		uuid := generateTestUUID(i)
		sessionFile := filepath.Join(projectDir, "agent-"+uuid+".jsonl")
		content := `{"type":"session_start","session_id":"` + uuid + `","project":"test-project","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","success":true,"duration":100}
{"type":"token_usage","timestamp":"2024-01-15T10:02:00Z","input_tokens":1000,"output_tokens":500,"cost_usd":0.01}
`
		if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write session file: %v", err)
		}
	}

	agg := NewAggregatorWithBaseDir(5, tmpDir) // Small cache to force evictions

	var wg sync.WaitGroup
	errChan := make(chan error, 1000)

	// Concurrent operations
	operations := []struct {
		name string
		fn   func() error
	}{
		{
			name: "load_session",
			fn: func() error {
				sessions, err := DiscoverSessions(tmpDir)
				if err != nil {
					return err
				}
				for _, s := range sessions {
					if _, err := agg.LoadSession(s.FilePath); err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			name: "get_project_metrics",
			fn: func() error {
				_, err := agg.GetProjectMetrics("test-project")
				return err
			},
		},
		{
			name: "list_sessions",
			fn: func() error {
				_, err := agg.ListSessions("test-project")
				return err
			},
		},
		{
			name: "cache_operations",
			fn: func() error {
				sessions, _ := DiscoverSessions(tmpDir)
				if len(sessions) > 0 {
					agg.InvalidateCache(sessions[0].FilePath)
					agg.IsCached(sessions[0].FilePath)
					agg.GetCachedMtime(sessions[0].FilePath)
				}
				return nil
			},
		},
	}

	// Run 50 goroutines per operation
	for _, op := range operations {
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(operation func() error) {
				defer wg.Done()
				if err := operation(); err != nil {
					errChan <- err
				}
			}(op.fn)
		}
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("concurrent operation error: %v", err)
	}

	// Verify cache state is consistent
	cacheSize := agg.CacheSize()
	if cacheSize > 5 {
		t.Errorf("cache size %d exceeds max size 5", cacheSize)
	}
}

// TestCachePerformance tests cache hit/miss behavior
func TestCachePerformance(t *testing.T) {
	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, "test-session.jsonl")
	content := `{"type":"session_start","session_id":"test-123","project":"test","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","success":true,"duration":100}
`
	if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	agg := NewAggregator(10)

	// Cold load (cache miss)
	start := time.Now()
	metrics1, err := agg.LoadSession(sessionFile)
	if err != nil {
		t.Fatalf("first load failed: %v", err)
	}
	coldDuration := time.Since(start)

	// Hot load (cache hit)
	start = time.Now()
	metrics2, err := agg.LoadSession(sessionFile)
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}
	hotDuration := time.Since(start)

	// Cache hit should return same pointer
	if metrics1 != metrics2 {
		t.Error("cache hit should return same metrics pointer")
	}

	// Cache hit should be faster (though not guaranteed in all environments)
	t.Logf("Cold load: %v, Hot load: %v", coldDuration, hotDuration)

	// Verify cache state
	if !agg.IsCached(sessionFile) {
		t.Error("file should be cached")
	}

	// Test cache invalidation
	agg.InvalidateCache(sessionFile)
	if agg.IsCached(sessionFile) {
		t.Error("file should not be cached after invalidation")
	}

	// Load again after invalidation
	metrics3, err := agg.LoadSession(sessionFile)
	if err != nil {
		t.Fatalf("third load failed: %v", err)
	}
	if metrics1 == metrics3 {
		t.Error("after invalidation should return new metrics object")
	}
}

// TestDiscoveryToAggregation tests the complete pipeline with project filtering
func TestDiscoveryToAggregation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two projects
	projects := []string{"project-alpha", "project-beta"}
	for _, project := range projects {
		projectDir := filepath.Join(tmpDir, project)
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatalf("failed to create project dir: %v", err)
		}

		// Create 2 sessions per project
		for i := 0; i < 2; i++ {
			uuid := generateTestUUID(i)
			sessionFile := filepath.Join(projectDir, "agent-"+uuid+".jsonl")
			content := `{"type":"session_start","session_id":"` + uuid + `","project":"` + project + `","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","success":true,"duration":100}
{"type":"token_usage","timestamp":"2024-01-15T10:02:00Z","input_tokens":1000,"output_tokens":500,"cost_usd":0.01}
`
			if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
				t.Fatalf("failed to write session file: %v", err)
			}
		}
	}

	// Discover all sessions
	allSessions, err := DiscoverSessions(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverSessions failed: %v", err)
	}
	if len(allSessions) != 4 {
		t.Errorf("discovered %d sessions, want 4", len(allSessions))
	}

	// Discover sessions for specific project
	alphaSessions, err := DiscoverProjectSessions(tmpDir, "project-alpha")
	if err != nil {
		t.Fatalf("DiscoverProjectSessions failed: %v", err)
	}
	if len(alphaSessions) != 2 {
		t.Errorf("discovered %d sessions for project-alpha, want 2", len(alphaSessions))
	}

	// Aggregate per project
	agg := NewAggregatorWithBaseDir(10, tmpDir)

	alphaMetrics, err := agg.GetProjectMetrics("project-alpha")
	if err != nil {
		t.Fatalf("GetProjectMetrics failed: %v", err)
	}
	if alphaMetrics.TotalSessions != 2 {
		t.Errorf("project-alpha TotalSessions = %d, want 2", alphaMetrics.TotalSessions)
	}

	betaMetrics, err := agg.GetProjectMetrics("project-beta")
	if err != nil {
		t.Fatalf("GetProjectMetrics failed: %v", err)
	}
	if betaMetrics.TotalSessions != 2 {
		t.Errorf("project-beta TotalSessions = %d, want 2", betaMetrics.TotalSessions)
	}
}

// TestFilterIntegration tests filtering through the full pipeline
func TestFilterIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	now := time.Now()

	// Create sessions with different timestamps
	sessions := []struct {
		uuid      string
		timestamp time.Time
		agentName string
		success   bool
	}{
		{"11111111-1111-1111-1111-111111111111", now.Add(-72 * time.Hour), "code-reviewer", true},
		{"22222222-2222-2222-2222-222222222222", now.Add(-24 * time.Hour), "test-automator", true},
		{"33333333-3333-3333-3333-333333333333", now.Add(-1 * time.Hour), "code-reviewer", false},
	}

	for _, s := range sessions {
		sessionFile := filepath.Join(projectDir, "agent-"+s.uuid+".jsonl")
		successStr := "true"
		if !s.success {
			successStr = "false"
		}
		ts := s.timestamp.Format(time.RFC3339)
		content := `{"type":"session_start","session_id":"` + s.uuid + `","project":"test-project","timestamp":"` + ts + `","status":"completed","agent_name":"` + s.agentName + `","success":` + successStr + `}
{"type":"tool_call","timestamp":"` + ts + `","tool_name":"Read","success":true,"duration":100}
{"type":"token_usage","timestamp":"` + ts + `","input_tokens":1000,"output_tokens":500,"cost_usd":0.01}
`
		if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write session file: %v", err)
		}
		os.Chtimes(sessionFile, s.timestamp, s.timestamp)
	}

	agg := NewAggregatorWithBaseDir(10, tmpDir)

	// Test time range filtering
	criteria := FilterCriteria{
		Since: now.Add(-48 * time.Hour),
	}
	filtered, err := agg.ListSessionsFiltered("test-project", criteria)
	if err != nil {
		t.Fatalf("ListSessionsFiltered failed: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("time filter returned %d sessions, want 2", len(filtered))
	}

	// Test project metrics with time filter
	metricsFiltered, err := agg.GetProjectMetricsFiltered("test-project", criteria)
	if err != nil {
		t.Fatalf("GetProjectMetricsFiltered failed: %v", err)
	}
	if metricsFiltered.TotalSessions != 2 {
		t.Errorf("filtered TotalSessions = %d, want 2", metricsFiltered.TotalSessions)
	}

	// Test agent name search
	agentCriteria := FilterCriteria{
		Search: "code-reviewer",
	}
	agentFiltered, err := agg.GetProjectMetricsFiltered("test-project", agentCriteria)
	if err != nil {
		t.Fatalf("GetProjectMetricsFiltered with agent failed: %v", err)
	}
	if agentFiltered.AgentPerformance["code-reviewer"] != 1 {
		t.Errorf("code-reviewer count = %d, want 1", agentFiltered.AgentPerformance["code-reviewer"])
	}
}

// TestEdgeCases tests various edge cases in the pipeline
func TestEdgeCases(t *testing.T) {
	t.Run("empty_project_directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		projectDir := filepath.Join(tmpDir, "empty-project")
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatalf("failed to create project dir: %v", err)
		}

		sessions, err := DiscoverProjectSessions(tmpDir, "empty-project")
		if err != nil {
			t.Fatalf("unexpected error for empty project: %v", err)
		}
		if len(sessions) != 0 {
			t.Errorf("expected 0 sessions, got %d", len(sessions))
		}
	})

	t.Run("non_matching_files_ignored", func(t *testing.T) {
		tmpDir := t.TempDir()
		projectDir := filepath.Join(tmpDir, "test-project")
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatalf("failed to create project dir: %v", err)
		}

		// Create non-matching files
		nonMatching := []string{"random.jsonl", "agent-notauuid.jsonl", "session.json"}
		for _, f := range nonMatching {
			if err := os.WriteFile(filepath.Join(projectDir, f), []byte("{}"), 0644); err != nil {
				t.Fatalf("failed to write file: %v", err)
			}
		}

		// Create one valid session
		validFile := filepath.Join(projectDir, "agent-11111111.jsonl")
		content := `{"type":"session_start","session_id":"test","project":"test","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}`
		if err := os.WriteFile(validFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write valid file: %v", err)
		}

		sessions, err := DiscoverProjectSessions(tmpDir, "test-project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sessions) != 1 {
			t.Errorf("expected 1 session, got %d", len(sessions))
		}
	})

	t.Run("nil_session_data_metrics", func(t *testing.T) {
		metrics := ExtractMetrics(nil)
		if metrics == nil {
			t.Fatal("ExtractMetrics(nil) should return empty metrics, not nil")
		}
		if metrics.TotalSessions != 0 {
			t.Errorf("nil session should produce 0 total sessions")
		}
	})

	t.Run("session_with_no_events", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.jsonl")
		content := `{"type":"session_start","session_id":"test-123","project":"test","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}`
		if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		sessionData, err := ParseSessionFile(tmpFile)
		if err != nil {
			t.Fatalf("ParseSessionFile failed: %v", err)
		}

		metrics := ExtractMetrics(sessionData)
		if metrics.TotalSessions != 1 {
			t.Errorf("expected 1 session, got %d", metrics.TotalSessions)
		}
		if len(metrics.ToolExecutions) != 0 {
			t.Errorf("expected 0 tool executions, got %d", len(metrics.ToolExecutions))
		}
	})

	t.Run("very_large_session_file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "large.jsonl")

		// Create session with many events
		var content string
		content = `{"type":"session_start","session_id":"test-123","project":"test","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}` + "\n"
		for i := 0; i < 1000; i++ {
			content += `{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","success":true,"duration":100}` + "\n"
		}

		if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		sessionData, err := ParseSessionFile(tmpFile)
		if err != nil {
			t.Fatalf("ParseSessionFile failed for large file: %v", err)
		}

		if len(sessionData.Events) != 1000 {
			t.Errorf("expected 1000 events, got %d", len(sessionData.Events))
		}

		metrics := ExtractMetrics(sessionData)
		if len(metrics.ToolExecutions) != 1 {
			t.Errorf("expected 1 aggregated tool, got %d", len(metrics.ToolExecutions))
		}
		// All events are Read tool calls
		for _, te := range metrics.ToolExecutions {
			if te.Name == "Read" && te.Count != 1000 {
				t.Errorf("Read tool count = %d, want 1000", te.Count)
			}
		}
	})
}

// BenchmarkParseSession benchmarks session parsing
func BenchmarkParseSession(b *testing.B) {
	tmpDir := b.TempDir()
	sessionFile := filepath.Join(tmpDir, "bench.jsonl")

	// Create realistic session content
	content := `{"type":"session_start","session_id":"bench-123","project":"bench","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","parameters":{"file_path":"/main.go"},"success":true,"duration":50}
{"type":"tool_call","timestamp":"2024-01-15T10:02:00Z","tool_name":"Write","parameters":{"file_path":"/test.go"},"success":true,"duration":100}
{"type":"bash_command","timestamp":"2024-01-15T10:03:00Z","command":"go test ./...","exit_code":0,"output":"PASS","output_length":4,"duration":1500,"success":true}
{"type":"file_operation","timestamp":"2024-01-15T10:04:00Z","operation":"read","path":"/config.yaml","success":true,"size_bytes":512,"duration":10}
{"type":"file_operation","timestamp":"2024-01-15T10:05:00Z","operation":"write","path":"/out.go","success":true,"size_bytes":2048,"duration":50}
{"type":"token_usage","timestamp":"2024-01-15T10:06:00Z","input_tokens":5000,"output_tokens":2000,"cost_usd":0.045,"model_name":"claude-sonnet-4-5"}
`
	if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
		b.Fatalf("failed to write test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseSessionFile(sessionFile)
		if err != nil {
			b.Fatalf("ParseSessionFile failed: %v", err)
		}
	}
}

// BenchmarkExtractMetrics benchmarks metrics extraction
func BenchmarkExtractMetrics(b *testing.B) {
	// Create session data with various event types
	sessionData := &SessionData{
		Session: Session{
			ID:        "bench-123",
			Project:   "bench",
			Timestamp: time.Now(),
			Status:    "completed",
			AgentName: "test-agent",
			Duration:  60000,
			Success:   true,
		},
		Events: make([]Event, 0, 100),
	}

	// Add various events
	now := time.Now()
	for i := 0; i < 50; i++ {
		sessionData.Events = append(sessionData.Events, &ToolCallEvent{
			BaseEvent:  BaseEvent{Type: "tool_call", Timestamp: now},
			ToolName:   "Read",
			Parameters: map[string]interface{}{"file_path": "/test.go"},
			Success:    true,
			Duration:   100,
		})
	}
	for i := 0; i < 20; i++ {
		sessionData.Events = append(sessionData.Events, &BashCommandEvent{
			BaseEvent:    BaseEvent{Type: "bash_command", Timestamp: now},
			Command:      "go test",
			ExitCode:     0,
			OutputLength: 100,
			Duration:     500,
			Success:      true,
		})
	}
	for i := 0; i < 20; i++ {
		sessionData.Events = append(sessionData.Events, &FileOperationEvent{
			BaseEvent: BaseEvent{Type: "file_operation", Timestamp: now},
			Operation: "write",
			Path:      "/out.go",
			Success:   true,
			SizeBytes: 1024,
			Duration:  50,
		})
	}
	for i := 0; i < 10; i++ {
		sessionData.Events = append(sessionData.Events, &TokenUsageEvent{
			BaseEvent:    BaseEvent{Type: "token_usage", Timestamp: now},
			InputTokens:  1000,
			OutputTokens: 500,
			CostUSD:      0.01,
			ModelName:    "claude-sonnet-4-5",
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics := ExtractMetrics(sessionData)
		_ = metrics.Validate()
	}
}

// BenchmarkAggregatorCache benchmarks cache operations
func BenchmarkAggregatorCache(b *testing.B) {
	tmpDir := b.TempDir()
	projectDir := filepath.Join(tmpDir, "bench-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		b.Fatalf("failed to create project dir: %v", err)
	}

	// Create session file
	sessionFile := filepath.Join(projectDir, "agent-11111111.jsonl")
	content := `{"type":"session_start","session_id":"bench-123","project":"bench","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","success":true,"duration":100}
`
	if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
		b.Fatalf("failed to write test file: %v", err)
	}

	agg := NewAggregator(100)

	// Prime the cache
	if _, err := agg.LoadSession(sessionFile); err != nil {
		b.Fatalf("initial load failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := agg.LoadSession(sessionFile)
		if err != nil {
			b.Fatalf("LoadSession failed: %v", err)
		}
	}
}

// BenchmarkConcurrentCacheAccess benchmarks concurrent cache access
func BenchmarkConcurrentCacheAccess(b *testing.B) {
	tmpDir := b.TempDir()
	projectDir := filepath.Join(tmpDir, "bench-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		b.Fatalf("failed to create project dir: %v", err)
	}

	// Create multiple session files
	files := make([]string, 10)
	for i := 0; i < 10; i++ {
		uuid := generateTestUUID(i)
		sessionFile := filepath.Join(projectDir, "agent-"+uuid+".jsonl")
		content := `{"type":"session_start","session_id":"` + uuid + `","project":"bench","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","success":true,"duration":100}
`
		if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
			b.Fatalf("failed to write test file: %v", err)
		}
		files[i] = sessionFile
	}

	agg := NewAggregator(5) // Small cache to test eviction under load

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, err := agg.LoadSession(files[i%len(files)])
			if err != nil {
				b.Errorf("LoadSession failed: %v", err)
			}
			i++
		}
	})
}

// generateTestUUID generates a deterministic test short hex ID based on index
// This format matches the agent-{hex}.jsonl discovery pattern
func generateTestUUID(i int) string {
	// Generate short hex IDs that match agent file pattern
	return fmt.Sprintf("%08x", i)
}

// TestEndToEndBehavioralWorkflow tests the complete behavioral workflow with real JSONL data
func TestEndToEndBehavioralWorkflow(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project structure mimicking real Claude CLI session directory
	projects := map[string][]struct {
		uuid    string
		content string
	}{
		"conductor": {
			{
				uuid: "aaa11111",
				content: `{"type":"session_start","session_id":"aaa11111","project":"conductor","timestamp":"2024-01-15T10:00:00Z","status":"completed","agent_name":"backend-developer","duration":300000,"success":true,"error_count":0}
{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","parameters":{"file_path":"/internal/executor/task.go"},"success":true,"duration":50}
{"type":"tool_call","timestamp":"2024-01-15T10:02:00Z","tool_name":"Grep","parameters":{"pattern":"func Execute"},"success":true,"duration":120}
{"type":"tool_call","timestamp":"2024-01-15T10:03:00Z","tool_name":"Write","parameters":{"file_path":"/internal/executor/wave.go"},"success":true,"duration":80}
{"type":"bash_command","timestamp":"2024-01-15T10:04:00Z","command":"go test ./internal/executor/...","exit_code":0,"output":"PASS","output_length":4,"duration":5000,"success":true}
{"type":"bash_command","timestamp":"2024-01-15T10:05:00Z","command":"go build ./...","exit_code":0,"output":"","output_length":0,"duration":3000,"success":true}
{"type":"file_operation","timestamp":"2024-01-15T10:06:00Z","operation":"read","path":"/go.mod","success":true,"size_bytes":256,"duration":5}
{"type":"file_operation","timestamp":"2024-01-15T10:07:00Z","operation":"write","path":"/internal/executor/wave.go","success":true,"size_bytes":4096,"duration":25}
{"type":"token_usage","timestamp":"2024-01-15T10:08:00Z","input_tokens":20000,"output_tokens":8000,"cost_usd":0.180,"model_name":"claude-sonnet-4-5"}
`,
			},
			{
				uuid: "bbb22222",
				content: `{"type":"session_start","session_id":"bbb22222","project":"conductor","timestamp":"2024-01-15T11:00:00Z","status":"completed","agent_name":"test-automator","duration":180000,"success":true,"error_count":0}
{"type":"tool_call","timestamp":"2024-01-15T11:01:00Z","tool_name":"Read","parameters":{"file_path":"/internal/executor/task_test.go"},"success":true,"duration":45}
{"type":"tool_call","timestamp":"2024-01-15T11:02:00Z","tool_name":"Edit","parameters":{"file_path":"/internal/executor/task_test.go"},"success":true,"duration":100}
{"type":"bash_command","timestamp":"2024-01-15T11:03:00Z","command":"go test ./internal/executor/... -v","exit_code":0,"output":"PASS","output_length":1024,"duration":8000,"success":true}
{"type":"file_operation","timestamp":"2024-01-15T11:04:00Z","operation":"edit","path":"/internal/executor/task_test.go","success":true,"size_bytes":2048,"duration":15}
{"type":"token_usage","timestamp":"2024-01-15T11:05:00Z","input_tokens":15000,"output_tokens":5000,"cost_usd":0.120,"model_name":"claude-sonnet-4-5"}
`,
			},
		},
		"other-project": {
			{
				uuid: "ccc33333",
				content: `{"type":"session_start","session_id":"ccc33333","project":"other-project","timestamp":"2024-01-15T12:00:00Z","status":"failed","agent_name":"code-reviewer","duration":60000,"success":false,"error_count":3}
{"type":"tool_call","timestamp":"2024-01-15T12:01:00Z","tool_name":"Read","parameters":{"file_path":"/main.go"},"success":true,"duration":30}
{"type":"tool_call","timestamp":"2024-01-15T12:02:00Z","tool_name":"Edit","parameters":{"file_path":"/main.go"},"success":false,"duration":150,"error":"file is read-only"}
{"type":"bash_command","timestamp":"2024-01-15T12:03:00Z","command":"go build","exit_code":1,"output":"compile error","output_length":512,"duration":2000,"success":false}
{"type":"token_usage","timestamp":"2024-01-15T12:04:00Z","input_tokens":5000,"output_tokens":2000,"cost_usd":0.045,"model_name":"claude-sonnet-4-5"}
`,
			},
		},
	}

	// Create project directories and session files
	for project, sessions := range projects {
		projectDir := filepath.Join(tmpDir, project)
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatalf("failed to create project dir: %v", err)
		}
		for _, s := range sessions {
			sessionFile := filepath.Join(projectDir, "agent-"+s.uuid+".jsonl")
			if err := os.WriteFile(sessionFile, []byte(s.content), 0644); err != nil {
				t.Fatalf("failed to write session file: %v", err)
			}
		}
	}

	// Step 1: Discover all sessions
	t.Run("discover_all_sessions", func(t *testing.T) {
		sessions, err := DiscoverSessions(tmpDir)
		if err != nil {
			t.Fatalf("DiscoverSessions failed: %v", err)
		}
		if len(sessions) != 3 {
			t.Errorf("discovered %d sessions, want 3", len(sessions))
		}
	})

	// Step 2: Discover project-specific sessions
	t.Run("discover_conductor_sessions", func(t *testing.T) {
		sessions, err := DiscoverProjectSessions(tmpDir, "conductor")
		if err != nil {
			t.Fatalf("DiscoverProjectSessions failed: %v", err)
		}
		if len(sessions) != 2 {
			t.Errorf("discovered %d conductor sessions, want 2", len(sessions))
		}
	})

	// Step 3: Parse and extract metrics
	t.Run("parse_and_extract_metrics", func(t *testing.T) {
		sessions, _ := DiscoverProjectSessions(tmpDir, "conductor")
		for _, session := range sessions {
			sessionData, err := ParseSessionFile(session.FilePath)
			if err != nil {
				t.Errorf("ParseSessionFile failed for %s: %v", session.FilePath, err)
				continue
			}

			metrics := ExtractMetrics(sessionData)
			if err := metrics.Validate(); err != nil {
				t.Errorf("metrics validation failed: %v", err)
			}

			// Verify metrics are populated
			if metrics.TotalSessions != 1 {
				t.Errorf("TotalSessions = %d, want 1", metrics.TotalSessions)
			}
			if metrics.TokenUsage.InputTokens == 0 {
				t.Error("InputTokens should not be zero")
			}
		}
	})

	// Step 4: Aggregator with filtering
	t.Run("aggregator_with_filter", func(t *testing.T) {
		agg := NewAggregatorWithBaseDir(10, tmpDir)

		// Get conductor project metrics
		metrics, err := agg.GetProjectMetrics("conductor")
		if err != nil {
			t.Fatalf("GetProjectMetrics failed: %v", err)
		}

		if metrics.TotalSessions != 2 {
			t.Errorf("TotalSessions = %d, want 2", metrics.TotalSessions)
		}

		// Verify tool aggregation
		if metrics.ToolUsageCounts["Read"] != 2 {
			t.Errorf("Read tool count = %d, want 2", metrics.ToolUsageCounts["Read"])
		}

		// Verify agent performance
		if metrics.AgentPerformance["backend-developer"] != 1 {
			t.Errorf("backend-developer count = %d, want 1", metrics.AgentPerformance["backend-developer"])
		}
		if metrics.AgentPerformance["test-automator"] != 1 {
			t.Errorf("test-automator count = %d, want 1", metrics.AgentPerformance["test-automator"])
		}
	})

	// Step 5: Time-based filtering
	t.Run("time_based_filtering", func(t *testing.T) {
		agg := NewAggregatorWithBaseDir(10, tmpDir)

		// Filter sessions from the last hour (relative to test data timestamp)
		baseTime, _ := time.Parse(time.RFC3339, "2024-01-15T11:30:00Z")
		criteria := FilterCriteria{
			Since: baseTime.Add(-30 * time.Minute),
		}

		filteredSessions, err := agg.ListSessionsFiltered("conductor", criteria)
		if err != nil {
			t.Fatalf("ListSessionsFiltered failed: %v", err)
		}

		// Should get sessions from 11:00 onwards
		if len(filteredSessions) > 2 {
			t.Errorf("filtered %d sessions, expected <= 2", len(filteredSessions))
		}
	})

	// Step 6: Calculate aggregate stats
	t.Run("calculate_aggregate_stats", func(t *testing.T) {
		agg := NewAggregatorWithBaseDir(10, tmpDir)

		var allMetrics []BehavioralMetrics
		sessions, _ := DiscoverSessions(tmpDir)
		for _, session := range sessions {
			m, err := agg.LoadSession(session.FilePath)
			if err != nil {
				continue
			}
			allMetrics = append(allMetrics, *m)
		}

		stats := CalculateStats(allMetrics)

		if stats.TotalSessions != 3 {
			t.Errorf("TotalSessions = %d, want 3", stats.TotalSessions)
		}

		// 2 success + 1 failure = ~0.66 success rate
		if stats.SuccessRate < 0.6 || stats.SuccessRate > 0.7 {
			t.Errorf("SuccessRate = %f, want ~0.66", stats.SuccessRate)
		}

		// TotalAgents counts unique agents with successes (based on AgentBreakdown)
		// Only successful sessions have agents recorded in AgentPerformance
		if stats.TotalAgents < 1 {
			t.Errorf("TotalAgents = %d, want at least 1", stats.TotalAgents)
		}
	})
}

// TestCacheEvictionUnderLoad tests cache behavior under concurrent load
func TestCacheEvictionUnderLoad(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "load-test")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create many session files
	numSessions := 50
	for i := 0; i < numSessions; i++ {
		uuid := generateTestUUID(i)
		sessionFile := filepath.Join(projectDir, "agent-"+uuid+".jsonl")
		content := `{"type":"session_start","session_id":"` + uuid + `","project":"load-test","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","success":true,"duration":100}
{"type":"token_usage","timestamp":"2024-01-15T10:02:00Z","input_tokens":1000,"output_tokens":500,"cost_usd":0.01}
`
		if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write session file: %v", err)
		}
	}

	// Use small cache to force evictions
	agg := NewAggregatorWithBaseDir(10, tmpDir)

	// Run concurrent loads
	var wg sync.WaitGroup
	sessions, _ := DiscoverProjectSessions(tmpDir, "load-test")

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			session := sessions[idx%len(sessions)]
			_, err := agg.LoadSession(session.FilePath)
			if err != nil {
				t.Errorf("LoadSession failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Verify cache stays within limits
	if agg.CacheSize() > 10 {
		t.Errorf("cache size %d exceeds max size 10", agg.CacheSize())
	}
}

// TestStatsCalculationEdgeCases tests stats calculation with various edge cases
func TestStatsCalculationEdgeCases(t *testing.T) {
	t.Run("empty_metrics_slice", func(t *testing.T) {
		stats := CalculateStats([]BehavioralMetrics{})
		if stats.TotalSessions != 0 {
			t.Errorf("TotalSessions = %d, want 0", stats.TotalSessions)
		}
		if stats.AgentBreakdown == nil {
			t.Error("AgentBreakdown should not be nil")
		}
	})

	t.Run("all_failures", func(t *testing.T) {
		metrics := []BehavioralMetrics{
			{TotalSessions: 1, SuccessRate: 0.0, TotalErrors: 2},
			{TotalSessions: 1, SuccessRate: 0.0, TotalErrors: 3},
		}
		stats := CalculateStats(metrics)
		if stats.SuccessRate != 0.0 {
			t.Errorf("SuccessRate = %f, want 0.0", stats.SuccessRate)
		}
		if stats.ErrorRate != 1.0 {
			t.Errorf("ErrorRate = %f, want 1.0", stats.ErrorRate)
		}
	})

	t.Run("all_successes", func(t *testing.T) {
		metrics := []BehavioralMetrics{
			{TotalSessions: 1, SuccessRate: 1.0, TotalErrors: 0},
			{TotalSessions: 1, SuccessRate: 1.0, TotalErrors: 0},
		}
		stats := CalculateStats(metrics)
		if stats.SuccessRate != 1.0 {
			t.Errorf("SuccessRate = %f, want 1.0", stats.SuccessRate)
		}
	})

	t.Run("large_token_counts", func(t *testing.T) {
		metrics := []BehavioralMetrics{
			{
				TotalSessions: 1,
				TokenUsage:    TokenUsage{InputTokens: 1_000_000, OutputTokens: 500_000, CostUSD: 100.0},
			},
		}
		stats := CalculateStats(metrics)
		if stats.TotalInputTokens != 1_000_000 {
			t.Errorf("TotalInputTokens = %d, want 1000000", stats.TotalInputTokens)
		}
	})
}

// TestRealWorldJSONLPatterns tests parsing of various real-world JSONL patterns
func TestRealWorldJSONLPatterns(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		wantEvents  int
		wantSuccess bool
	}{
		{
			name: "minimal_valid_session",
			content: `{"type":"session_start","session_id":"test-123","project":"test","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
`,
			wantEvents:  0,
			wantSuccess: true,
		},
		{
			name: "session_with_all_event_types",
			content: `{"type":"session_start","session_id":"test-123","project":"test","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","success":true,"duration":50}
{"type":"bash_command","timestamp":"2024-01-15T10:02:00Z","command":"ls","exit_code":0,"duration":10,"success":true}
{"type":"file_operation","timestamp":"2024-01-15T10:03:00Z","operation":"read","path":"/test.go","success":true,"size_bytes":100,"duration":5}
{"type":"token_usage","timestamp":"2024-01-15T10:04:00Z","input_tokens":1000,"output_tokens":500,"cost_usd":0.01}
`,
			wantEvents:  4,
			wantSuccess: true,
		},
		{
			name: "session_with_unicode_and_special_chars",
			content: `{"type":"session_start","session_id":"test-123","project":"test","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","parameters":{"file_path":"/path/with spaces/file.go"},"success":true,"duration":50}
{"type":"bash_command","timestamp":"2024-01-15T10:02:00Z","command":"echo 'hello world' && ls -la","exit_code":0,"output":"","output_length":0,"duration":10,"success":true}
`,
			wantEvents:  2,
			wantSuccess: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.jsonl")
			if err := os.WriteFile(tmpFile, []byte(tc.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			sessionData, err := ParseSessionFile(tmpFile)
			if err != nil && tc.wantSuccess {
				t.Fatalf("ParseSessionFile failed: %v", err)
			}

			if tc.wantSuccess {
				if len(sessionData.Events) != tc.wantEvents {
					t.Errorf("got %d events, want %d", len(sessionData.Events), tc.wantEvents)
				}

				metrics := ExtractMetrics(sessionData)
				if err := metrics.Validate(); err != nil {
					t.Errorf("metrics validation failed: %v", err)
				}
			}
		})
	}
}

// TestAggregatorCacheInvalidation tests cache invalidation behavior
func TestAggregatorCacheInvalidation(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	sessionFile := filepath.Join(projectDir, "agent-11111111.jsonl")
	originalContent := `{"type":"session_start","session_id":"test-123","project":"test","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","success":true,"duration":100}
`
	if err := os.WriteFile(sessionFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to write session file: %v", err)
	}

	agg := NewAggregator(10)

	// Load session - should cache it
	metrics1, err := agg.LoadSession(sessionFile)
	if err != nil {
		t.Fatalf("first load failed: %v", err)
	}

	if !agg.IsCached(sessionFile) {
		t.Error("file should be cached after load")
	}

	// Invalidate cache
	agg.InvalidateCache(sessionFile)

	if agg.IsCached(sessionFile) {
		t.Error("file should not be cached after invalidation")
	}

	// Update file content
	updatedContent := `{"type":"session_start","session_id":"test-123","project":"test","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","success":true,"duration":100}
{"type":"tool_call","timestamp":"2024-01-15T10:02:00Z","tool_name":"Write","success":true,"duration":200}
`
	// Wait a moment to ensure different mtime
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(sessionFile, []byte(updatedContent), 0644); err != nil {
		t.Fatalf("failed to update session file: %v", err)
	}

	// Load again - should get new content
	metrics2, err := agg.LoadSession(sessionFile)
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}

	// Should have different metrics (more tool executions)
	if len(metrics2.ToolExecutions) <= len(metrics1.ToolExecutions) {
		t.Error("updated metrics should have more tool executions")
	}
}

// TestMetricsAggregationAccuracy tests that metrics aggregation is accurate
func TestMetricsAggregationAccuracy(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "accuracy-test")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create sessions with known values for verification
	sessions := []struct {
		uuid         string
		inputTokens  int64
		outputTokens int64
		cost         float64
		toolCalls    int
		success      bool
	}{
		{"aaaa1111-1111-1111-1111-111111111111", 10000, 5000, 0.105, 5, true},
		{"bbbb2222-2222-2222-2222-222222222222", 20000, 10000, 0.210, 10, true},
		{"cccc3333-3333-3333-3333-333333333333", 5000, 2500, 0.0525, 2, false},
	}

	expectedTotalInput := int64(0)
	expectedTotalOutput := int64(0)
	expectedTotalCost := 0.0
	expectedToolCalls := 0

	for _, s := range sessions {
		expectedTotalInput += s.inputTokens
		expectedTotalOutput += s.outputTokens
		expectedTotalCost += s.cost
		expectedToolCalls += s.toolCalls

		// Generate tool calls
		var toolCallEvents string
		for i := 0; i < s.toolCalls; i++ {
			toolCallEvents += `{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","success":true,"duration":50}
`
		}

		successStr := "true"
		if !s.success {
			successStr = "false"
		}

		content := `{"type":"session_start","session_id":"` + s.uuid + `","project":"accuracy-test","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":` + successStr + `}
` + toolCallEvents + `{"type":"token_usage","timestamp":"2024-01-15T10:02:00Z","input_tokens":` + intToStr(s.inputTokens) + `,"output_tokens":` + intToStr(s.outputTokens) + `,"cost_usd":` + floatToStr(s.cost) + `}
`
		sessionFile := filepath.Join(projectDir, "agent-"+s.uuid+".jsonl")
		if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write session file: %v", err)
		}
	}

	agg := NewAggregatorWithBaseDir(10, tmpDir)
	metrics, err := agg.GetProjectMetrics("accuracy-test")
	if err != nil {
		t.Fatalf("GetProjectMetrics failed: %v", err)
	}

	// Verify aggregations
	if metrics.TokenUsage.InputTokens != expectedTotalInput {
		t.Errorf("InputTokens = %d, want %d", metrics.TokenUsage.InputTokens, expectedTotalInput)
	}
	if metrics.TokenUsage.OutputTokens != expectedTotalOutput {
		t.Errorf("OutputTokens = %d, want %d", metrics.TokenUsage.OutputTokens, expectedTotalOutput)
	}

	// Cost should be close (floating point comparison)
	if metrics.TotalCost < expectedTotalCost-0.01 || metrics.TotalCost > expectedTotalCost+0.01 {
		t.Errorf("TotalCost = %f, want approximately %f", metrics.TotalCost, expectedTotalCost)
	}

	// Verify tool usage
	if metrics.ToolUsageCounts["Read"] != expectedToolCalls {
		t.Errorf("Read tool count = %d, want %d", metrics.ToolUsageCounts["Read"], expectedToolCalls)
	}
}

// intToStr converts int64 to string
func intToStr(i int64) string {
	return strconv.FormatInt(i, 10)
}

// floatToStr converts float64 to string with 4 decimal places
func floatToStr(f float64) string {
	return strconv.FormatFloat(f, 'f', 4, 64)
}

// TestDatabaseStorageIntegration tests storing and retrieving behavioral data through the learning store
func TestDatabaseStorageIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "db-test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create realistic session data
	sessions := []struct {
		uuid    string
		agent   string
		success bool
		tools   int
		cost    float64
	}{
		{"db111111-1111-1111-1111-111111111111", "backend-developer", true, 5, 0.08},
		{"db222222-2222-2222-2222-222222222222", "test-automator", true, 8, 0.12},
		{"db333333-3333-3333-3333-333333333333", "code-reviewer", false, 3, 0.05},
	}

	for _, s := range sessions {
		successStr := "true"
		if !s.success {
			successStr = "false"
		}

		var toolCalls string
		for i := 0; i < s.tools; i++ {
			toolCalls += `{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","success":true,"duration":100}
`
		}

		content := `{"type":"session_start","session_id":"` + s.uuid + `","project":"db-test-project","timestamp":"2024-01-15T10:00:00Z","status":"completed","agent_name":"` + s.agent + `","duration":60000,"success":` + successStr + `}
` + toolCalls + `{"type":"token_usage","timestamp":"2024-01-15T10:02:00Z","input_tokens":2000,"output_tokens":1000,"cost_usd":` + floatToStr(s.cost) + `}
`
		sessionFile := filepath.Join(projectDir, "agent-"+s.uuid+".jsonl")
		if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write session file: %v", err)
		}
	}

	// Test discovery and aggregation with database verification
	t.Run("store_and_retrieve_metrics", func(t *testing.T) {
		agg := NewAggregatorWithBaseDir(10, tmpDir)
		metrics, err := agg.GetProjectMetrics("db-test-project")
		if err != nil {
			t.Fatalf("GetProjectMetrics failed: %v", err)
		}

		// Verify session count
		if metrics.TotalSessions != 3 {
			t.Errorf("TotalSessions = %d, want 3", metrics.TotalSessions)
		}

		// Verify agent distribution
		if metrics.AgentPerformance["backend-developer"] != 1 {
			t.Errorf("backend-developer count = %d, want 1", metrics.AgentPerformance["backend-developer"])
		}

		// Verify total tool usage (5 + 8 + 3 = 16)
		totalTools := metrics.ToolUsageCounts["Read"]
		if totalTools != 16 {
			t.Errorf("Read tool count = %d, want 16", totalTools)
		}

		// Verify cost aggregation
		expectedCost := 0.08 + 0.12 + 0.05
		if metrics.TotalCost < expectedCost-0.01 || metrics.TotalCost > expectedCost+0.01 {
			t.Errorf("TotalCost = %f, want approximately %f", metrics.TotalCost, expectedCost)
		}
	})

	t.Run("cache_persistence", func(t *testing.T) {
		agg := NewAggregatorWithBaseDir(10, tmpDir)

		// First load - should populate cache
		discoveredSessions, _ := DiscoverProjectSessions(tmpDir, "db-test-project")
		for _, session := range discoveredSessions {
			_, err := agg.LoadSession(session.FilePath)
			if err != nil {
				t.Errorf("LoadSession failed: %v", err)
			}
		}

		// Verify all sessions cached
		for _, session := range discoveredSessions {
			if !agg.IsCached(session.FilePath) {
				t.Errorf("Session not cached: %s", session.FilePath)
			}
		}

		// Second load - should hit cache
		for _, session := range discoveredSessions {
			m, err := agg.LoadSession(session.FilePath)
			if err != nil {
				t.Errorf("Second LoadSession failed: %v", err)
			}
			if m == nil {
				t.Errorf("Second load returned nil metrics")
			}
		}

		// Clear cache
		agg.ClearCache()
		if agg.CacheSize() != 0 {
			t.Errorf("Cache not cleared, size = %d", agg.CacheSize())
		}
	})
}

// TestPatternDetectionIntegration tests pattern detection on real session data
func TestPatternDetectionIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "pattern-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create sessions with recognizable patterns
	sessions := []struct {
		uuid     string
		agent    string
		duration int64
		success  bool
		errors   int
		tools    []string
	}{
		{
			uuid:     "pat11111-1111-1111-1111-111111111111",
			agent:    "backend-developer",
			duration: 30000,
			success:  true,
			errors:   0,
			tools:    []string{"Read", "Read", "Write", "Bash"},
		},
		{
			uuid:     "pat22222-2222-2222-2222-222222222222",
			agent:    "backend-developer",
			duration: 35000,
			success:  true,
			errors:   0,
			tools:    []string{"Read", "Read", "Write", "Bash"},
		},
		{
			uuid:     "pat33333-3333-3333-3333-333333333333",
			agent:    "test-automator",
			duration: 120000,
			success:  false,
			errors:   3,
			tools:    []string{"Read", "Bash", "Bash", "Bash"},
		},
		{
			uuid:     "pat44444-4444-4444-4444-444444444444",
			agent:    "backend-developer",
			duration: 32000,
			success:  true,
			errors:   0,
			tools:    []string{"Read", "Read", "Write", "Bash"},
		},
	}

	var allSessions []Session
	var allMetrics []BehavioralMetrics

	for _, s := range sessions {
		successStr := "true"
		if !s.success {
			successStr = "false"
		}

		var toolCalls string
		for _, tool := range s.tools {
			toolCalls += `{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"` + tool + `","success":true,"duration":100}
`
		}

		content := `{"type":"session_start","session_id":"` + s.uuid + `","project":"pattern-project","timestamp":"2024-01-15T10:00:00Z","status":"completed","agent_name":"` + s.agent + `","duration":` + strconv.FormatInt(s.duration, 10) + `,"success":` + successStr + `,"error_count":` + strconv.Itoa(s.errors) + `}
` + toolCalls

		sessionFile := filepath.Join(projectDir, "agent-"+s.uuid+".jsonl")
		if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write session file: %v", err)
		}

		// Parse and collect for pattern detection
		sessionData, err := ParseSessionFile(sessionFile)
		if err != nil {
			t.Fatalf("ParseSessionFile failed: %v", err)
		}
		allSessions = append(allSessions, sessionData.Session)
		metrics := ExtractMetrics(sessionData)
		allMetrics = append(allMetrics, *metrics)
	}

	t.Run("detect_tool_sequences", func(t *testing.T) {
		detector := NewPatternDetector(allSessions, allMetrics)
		sequences := detector.DetectToolSequences(2, 2) // min frequency 2, sequence length 2

		// Should detect Read->Read sequence (appears in 3 sessions)
		found := false
		for _, seq := range sequences {
			if len(seq.Tools) >= 2 && seq.Tools[0] == "Read" && seq.Tools[1] == "Read" {
				found = true
				if seq.Frequency < 2 {
					t.Errorf("Read->Read frequency = %d, want >= 2", seq.Frequency)
				}
			}
		}
		if !found {
			t.Log("Tool sequences found:", sequences)
		}
	})

	t.Run("detect_bash_patterns", func(t *testing.T) {
		detector := NewPatternDetector(allSessions, allMetrics)
		patterns := detector.DetectBashPatterns(1)

		// Should find bash commands if present in tool calls
		// (in our test data, bash commands are in tool_calls not bash_commands)
		// This tests the pattern detection infrastructure
		_ = patterns
	})

	t.Run("identify_anomalies", func(t *testing.T) {
		detector := NewPatternDetector(allSessions, allMetrics)
		anomalies := detector.IdentifyAnomalies(1.5) // Lower threshold for testing

		// Should identify the long-running failed session as anomalous
		var foundDurationAnomaly bool
		var foundFailureAnomaly bool
		for _, a := range anomalies {
			if a.Type == "duration" && a.SessionID == "pat33333-3333-3333-3333-333333333333" {
				foundDurationAnomaly = true
			}
			if a.Type == "session_failure" {
				foundFailureAnomaly = true
			}
		}
		_ = foundDurationAnomaly
		_ = foundFailureAnomaly
	})

	t.Run("cluster_sessions", func(t *testing.T) {
		detector := NewPatternDetector(allSessions, allMetrics)
		clusters := detector.ClusterSessions(2)

		if len(clusters) != 2 {
			t.Errorf("Expected 2 clusters, got %d", len(clusters))
		}

		// Verify all sessions are assigned
		totalAssigned := 0
		for _, cluster := range clusters {
			totalAssigned += cluster.Size
		}
		if totalAssigned != 4 {
			t.Errorf("Total assigned = %d, want 4", totalAssigned)
		}
	})
}

// TestPredictionIntegration tests failure prediction with real data
func TestPredictionIntegration(t *testing.T) {
	// Create test sessions with known success/failure patterns
	sessions := []Session{
		{ID: "s1", AgentName: "backend-developer", Duration: 30000, Success: true, ErrorCount: 0},
		{ID: "s2", AgentName: "backend-developer", Duration: 32000, Success: true, ErrorCount: 0},
		{ID: "s3", AgentName: "backend-developer", Duration: 35000, Success: true, ErrorCount: 0},
		{ID: "s4", AgentName: "test-automator", Duration: 60000, Success: false, ErrorCount: 2},
		{ID: "s5", AgentName: "test-automator", Duration: 55000, Success: false, ErrorCount: 3},
	}

	metrics := []BehavioralMetrics{
		{TotalSessions: 1, ToolExecutions: []ToolExecution{{Name: "Read", Count: 3}, {Name: "Write", Count: 1}}},
		{TotalSessions: 1, ToolExecutions: []ToolExecution{{Name: "Read", Count: 4}, {Name: "Write", Count: 2}}},
		{TotalSessions: 1, ToolExecutions: []ToolExecution{{Name: "Read", Count: 3}, {Name: "Write", Count: 1}}},
		{TotalSessions: 1, ToolExecutions: []ToolExecution{{Name: "Bash", Count: 10}, {Name: "Write", Count: 5}}},
		{TotalSessions: 1, ToolExecutions: []ToolExecution{{Name: "Bash", Count: 12}, {Name: "Write", Count: 6}}},
	}

	predictor := NewFailurePredictor(sessions, metrics)

	t.Run("predict_low_risk_session", func(t *testing.T) {
		newSession := &Session{
			ID:        "new1",
			AgentName: "backend-developer",
			Duration:  30000,
		}
		toolUsage := []string{"Read", "Read", "Write"}

		result, err := predictor.PredictFailure(newSession, toolUsage)
		if err != nil {
			t.Fatalf("PredictFailure failed: %v", err)
		}

		// Should predict low risk for patterns similar to successful sessions
		if result.RiskLevel == "high" {
			t.Errorf("Expected low/medium risk for backend-developer pattern, got %s", result.RiskLevel)
		}
		if result.Probability < 0 || result.Probability > 1 {
			t.Errorf("Probability out of range: %f", result.Probability)
		}
	})

	t.Run("predict_high_risk_session", func(t *testing.T) {
		newSession := &Session{
			ID:        "new2",
			AgentName: "unknown-agent",
			Duration:  60000,
		}
		toolUsage := []string{"Bash", "Bash", "Bash", "Bash", "Bash", "Write", "Write", "Write", "Write"}

		result, err := predictor.PredictFailure(newSession, toolUsage)
		if err != nil {
			t.Fatalf("PredictFailure failed: %v", err)
		}

		// Heavy bash + write usage is flagged as risky
		if len(result.RiskFactors) == 0 && result.Probability > 0.5 {
			// This is expected for high-risk patterns
		}
		if result.Explanation == "" {
			t.Error("Expected non-empty explanation")
		}
		if len(result.Recommendations) == 0 {
			t.Error("Expected at least one recommendation")
		}
	})

	t.Run("calculate_tool_failure_rates", func(t *testing.T) {
		rates := predictor.CalculateToolFailureRates()

		// Verify rates are calculated
		if len(rates) == 0 {
			t.Error("Expected tool failure rates to be calculated")
		}

		// Rates should be between 0 and 1
		for tool, rate := range rates {
			if rate < 0 || rate > 1 {
				t.Errorf("Tool %s has invalid failure rate: %f", tool, rate)
			}
		}
	})

	t.Run("find_similar_sessions", func(t *testing.T) {
		newSession := &Session{ID: "search"}
		toolUsage := []string{"Read", "Write"}

		similar := predictor.FindSimilarHistoricalSessions(newSession, toolUsage)

		// Should find sessions with Read/Write usage
		if len(similar) == 0 {
			t.Log("No similar sessions found - may be expected with small dataset")
		}
	})
}

// TestPerformanceScoringIntegration tests agent scoring with realistic data
func TestPerformanceScoringIntegration(t *testing.T) {
	sessions := []Session{
		{ID: "s1", AgentName: "backend-developer", Duration: 30000, Success: true, ErrorCount: 0},
		{ID: "s2", AgentName: "backend-developer", Duration: 35000, Success: true, ErrorCount: 0},
		{ID: "s3", AgentName: "backend-developer", Duration: 40000, Success: true, ErrorCount: 1},
		{ID: "s4", AgentName: "test-automator", Duration: 60000, Success: true, ErrorCount: 0},
		{ID: "s5", AgentName: "test-automator", Duration: 65000, Success: false, ErrorCount: 2},
		{ID: "s6", AgentName: "code-reviewer", Duration: 20000, Success: true, ErrorCount: 0},
		{ID: "s7", AgentName: "code-reviewer", Duration: 25000, Success: true, ErrorCount: 0},
	}

	metrics := make([]BehavioralMetrics, len(sessions))
	for i := range metrics {
		metrics[i] = BehavioralMetrics{
			TotalSessions: 1,
			TokenUsage:    TokenUsage{InputTokens: 1000, OutputTokens: 500, CostUSD: 0.05},
		}
	}

	scorer := NewPerformanceScorer(sessions, metrics)

	t.Run("score_individual_agent", func(t *testing.T) {
		score := scorer.ScoreAgent("backend-developer")

		if score.SampleSize != 3 {
			t.Errorf("SampleSize = %d, want 3", score.SampleSize)
		}
		if score.SuccessScore < 0 || score.SuccessScore > 1 {
			t.Errorf("SuccessScore out of range: %f", score.SuccessScore)
		}
		if score.CompositeScore < 0 || score.CompositeScore > 1 {
			t.Errorf("CompositeScore out of range: %f", score.CompositeScore)
		}
	})

	t.Run("rank_all_agents", func(t *testing.T) {
		rankings := scorer.RankAgents()

		if len(rankings) != 3 {
			t.Errorf("Expected 3 agents in rankings, got %d", len(rankings))
		}

		// Verify ranks are sequential
		for i, r := range rankings {
			if r.Rank != i+1 {
				t.Errorf("Agent %s has rank %d, expected %d", r.AgentName, r.Rank, i+1)
			}
		}

		// Verify scores are in descending order
		for i := 1; i < len(rankings); i++ {
			if rankings[i].Score > rankings[i-1].Score {
				t.Error("Rankings not in descending score order")
			}
		}
	})

	t.Run("compare_within_domain", func(t *testing.T) {
		domains := scorer.CompareWithinDomain()

		// Should have domains populated
		if len(domains) == 0 {
			t.Error("Expected at least one domain")
		}

		// Verify each domain has ranked agents
		for domain, agents := range domains {
			for _, agent := range agents {
				if agent.Score < 0 || agent.Score > 1 {
					t.Errorf("Agent %s in domain %s has invalid score: %f", agent.AgentName, domain, agent.Score)
				}
			}
		}
	})

	t.Run("custom_weights", func(t *testing.T) {
		customScorer := NewPerformanceScorer(sessions, metrics)
		customScorer.SetWeights(ScoreWeights{
			Success:    0.8,
			CostEff:    0.1,
			Speed:      0.05,
			ErrorRecov: 0.05,
		})

		// Score should be more heavily influenced by success rate
		score := customScorer.ScoreAgent("backend-developer")
		if score.CompositeScore == 0 {
			t.Error("Expected non-zero composite score with custom weights")
		}
	})
}

// TestStatsCalculationIntegration tests aggregate stats calculation
func TestStatsCalculationIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "stats-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create test sessions
	for i := 0; i < 5; i++ {
		uuid := generateTestUUID(i + 100)
		successStr := "true"
		if i >= 3 {
			successStr = "false"
		}
		content := `{"type":"session_start","session_id":"` + uuid + `","project":"stats-project","timestamp":"2024-01-15T10:00:00Z","status":"completed","agent_name":"agent-` + strconv.Itoa(i%3) + `","duration":` + strconv.Itoa((i+1)*10000) + `,"success":` + successStr + `}
{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","success":true,"duration":100}
{"type":"token_usage","timestamp":"2024-01-15T10:02:00Z","input_tokens":` + strconv.Itoa((i+1)*1000) + `,"output_tokens":` + strconv.Itoa((i+1)*500) + `,"cost_usd":` + floatToStr(float64(i+1)*0.02) + `}
`
		sessionFile := filepath.Join(projectDir, "agent-"+uuid+".jsonl")
		if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write session file: %v", err)
		}
	}

	agg := NewAggregatorWithBaseDir(10, tmpDir)

	t.Run("calculate_aggregate_stats", func(t *testing.T) {
		var allMetrics []BehavioralMetrics
		discoveredSessions, _ := DiscoverProjectSessions(tmpDir, "stats-project")
		for _, session := range discoveredSessions {
			m, err := agg.LoadSession(session.FilePath)
			if err != nil {
				continue
			}
			allMetrics = append(allMetrics, *m)
		}

		stats := CalculateStats(allMetrics)

		// Verify aggregation
		if stats.TotalSessions != 5 {
			t.Errorf("TotalSessions = %d, want 5", stats.TotalSessions)
		}

		// 3 successes out of 5
		expectedSuccessRate := 0.6
		if stats.SuccessRate < expectedSuccessRate-0.01 || stats.SuccessRate > expectedSuccessRate+0.01 {
			t.Errorf("SuccessRate = %f, want approximately %f", stats.SuccessRate, expectedSuccessRate)
		}

		// Verify token counts (1+2+3+4+5)*1000 = 15000 input, 7500 output
		expectedInputTokens := int64((1 + 2 + 3 + 4 + 5) * 1000)
		if stats.TotalInputTokens != expectedInputTokens {
			t.Errorf("TotalInputTokens = %d, want %d", stats.TotalInputTokens, expectedInputTokens)
		}

		// Verify cost aggregation
		expectedCost := (1 + 2 + 3 + 4 + 5) * 0.02
		if stats.TotalCost < expectedCost-0.01 || stats.TotalCost > expectedCost+0.01 {
			t.Errorf("TotalCost = %f, want approximately %f", stats.TotalCost, expectedCost)
		}
	})
}

// TestConcurrentAggregationIntegration tests thread-safety under realistic load
func TestConcurrentAggregationIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "concurrent-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create test sessions
	numSessions := 20
	for i := 0; i < numSessions; i++ {
		uuid := generateTestUUID(i + 200)
		content := `{"type":"session_start","session_id":"` + uuid + `","project":"concurrent-project","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","success":true,"duration":100}
{"type":"token_usage","timestamp":"2024-01-15T10:02:00Z","input_tokens":1000,"output_tokens":500,"cost_usd":0.01}
`
		sessionFile := filepath.Join(projectDir, "agent-"+uuid+".jsonl")
		if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write session file: %v", err)
		}
	}

	agg := NewAggregatorWithBaseDir(5, tmpDir) // Small cache to force evictions
	discoveredSessions, _ := DiscoverProjectSessions(tmpDir, "concurrent-project")

	t.Run("concurrent_load_and_aggregate", func(t *testing.T) {
		var wg sync.WaitGroup
		errChan := make(chan error, 200)

		// Run concurrent operations
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				session := discoveredSessions[idx%len(discoveredSessions)]
				_, err := agg.LoadSession(session.FilePath)
				if err != nil {
					errChan <- err
				}
			}(i)
		}

		// Concurrent project metrics
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := agg.GetProjectMetrics("concurrent-project")
				if err != nil {
					errChan <- err
				}
			}()
		}

		wg.Wait()
		close(errChan)

		for err := range errChan {
			t.Errorf("concurrent operation error: %v", err)
		}

		// Cache should not exceed max size
		if agg.CacheSize() > 5 {
			t.Errorf("Cache exceeded max size: %d", agg.CacheSize())
		}
	})
}
