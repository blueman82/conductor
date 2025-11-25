package behavioral

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewAggregator(t *testing.T) {
	tests := []struct {
		name         string
		maxCacheSize int
		wantSize     int
	}{
		{"default size", 0, DefaultMaxCacheSize},
		{"negative size", -5, DefaultMaxCacheSize},
		{"custom size", 100, 100},
		{"small size", 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := NewAggregator(tt.maxCacheSize)
			if agg.maxCacheSize != tt.wantSize {
				t.Errorf("NewAggregator(%d).maxCacheSize = %d, want %d", tt.maxCacheSize, agg.maxCacheSize, tt.wantSize)
			}
			if agg.cache == nil {
				t.Error("cache map should be initialized")
			}
			if agg.lruList == nil {
				t.Error("LRU list should be initialized")
			}
		})
	}
}

func TestNewAggregatorWithBaseDir(t *testing.T) {
	baseDir := "/custom/base/dir"
	agg := NewAggregatorWithBaseDir(25, baseDir)

	if agg.maxCacheSize != 25 {
		t.Errorf("maxCacheSize = %d, want 25", agg.maxCacheSize)
	}
	if agg.baseDir != baseDir {
		t.Errorf("baseDir = %s, want %s", agg.baseDir, baseDir)
	}
}

func TestAggregator_LoadSession(t *testing.T) {
	// Create temp directory with test data
	tmpDir := t.TempDir()

	// Create a valid session file
	sessionFile := filepath.Join(tmpDir, "test-session.jsonl")
	sessionContent := `{"type":"session_start","session_id":"test-123","project":"test-project","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","tool_name":"Read","timestamp":"2024-01-15T10:01:00Z","success":true,"duration":100}
{"type":"tool_call","tool_name":"Write","timestamp":"2024-01-15T10:02:00Z","success":true,"duration":200}
{"type":"token_usage","timestamp":"2024-01-15T10:03:00Z","input_tokens":1000,"output_tokens":500,"cost_usd":0.01}
`
	if err := os.WriteFile(sessionFile, []byte(sessionContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	agg := NewAggregator(10)

	// First load - should parse
	metrics, err := agg.LoadSession(sessionFile)
	if err != nil {
		t.Fatalf("LoadSession failed: %v", err)
	}

	if metrics.TotalSessions != 1 {
		t.Errorf("TotalSessions = %d, want 1", metrics.TotalSessions)
	}
	if len(metrics.ToolExecutions) != 2 {
		t.Errorf("ToolExecutions count = %d, want 2", len(metrics.ToolExecutions))
	}

	// Second load - should hit cache
	metrics2, err := agg.LoadSession(sessionFile)
	if err != nil {
		t.Fatalf("second LoadSession failed: %v", err)
	}
	if metrics2 != metrics {
		t.Error("second load should return cached metrics pointer")
	}

	// Verify cache size
	if agg.CacheSize() != 1 {
		t.Errorf("CacheSize = %d, want 1", agg.CacheSize())
	}
}

func TestAggregator_CacheInvalidation(t *testing.T) {
	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, "test-session.jsonl")
	sessionContent := `{"type":"session_start","session_id":"test-123","project":"test-project","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","tool_name":"Read","timestamp":"2024-01-15T10:01:00Z","success":true,"duration":100}
`
	if err := os.WriteFile(sessionFile, []byte(sessionContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	agg := NewAggregator(10)

	// Load to cache
	_, err := agg.LoadSession(sessionFile)
	if err != nil {
		t.Fatalf("LoadSession failed: %v", err)
	}

	if !agg.IsCached(sessionFile) {
		t.Error("file should be cached after load")
	}

	// Invalidate cache
	agg.InvalidateCache(sessionFile)

	if agg.IsCached(sessionFile) {
		t.Error("file should not be cached after invalidation")
	}

	if agg.CacheSize() != 0 {
		t.Errorf("CacheSize = %d, want 0 after invalidation", agg.CacheSize())
	}
}

func TestAggregator_MtimeTracking(t *testing.T) {
	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, "test-session.jsonl")
	sessionContent := `{"type":"session_start","session_id":"test-123","project":"test-project","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","tool_name":"Read","timestamp":"2024-01-15T10:01:00Z","success":true,"duration":100}
`
	if err := os.WriteFile(sessionFile, []byte(sessionContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	agg := NewAggregator(10)

	// Load to cache
	metrics1, err := agg.LoadSession(sessionFile)
	if err != nil {
		t.Fatalf("LoadSession failed: %v", err)
	}

	cachedMtime := agg.GetCachedMtime(sessionFile)
	if cachedMtime.IsZero() {
		t.Error("cached mtime should not be zero")
	}

	// Wait and modify file
	time.Sleep(10 * time.Millisecond)
	newContent := `{"type":"session_start","session_id":"test-123","project":"test-project","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","tool_name":"Read","timestamp":"2024-01-15T10:01:00Z","success":true,"duration":100}
{"type":"tool_call","tool_name":"Write","timestamp":"2024-01-15T10:02:00Z","success":true,"duration":200}
`
	if err := os.WriteFile(sessionFile, []byte(newContent), 0644); err != nil {
		t.Fatalf("failed to update test file: %v", err)
	}

	// Load again - should re-parse due to mtime change
	metrics2, err := agg.LoadSession(sessionFile)
	if err != nil {
		t.Fatalf("second LoadSession failed: %v", err)
	}

	// Should be different metrics (re-parsed)
	if len(metrics1.ToolExecutions) == len(metrics2.ToolExecutions) {
		t.Error("metrics should differ after file modification")
	}
}

func TestAggregator_LRUEviction(t *testing.T) {
	tmpDir := t.TempDir()
	maxSize := 3
	agg := NewAggregator(maxSize)

	// Create more files than cache size
	files := make([]string, 5)
	for i := 0; i < 5; i++ {
		sessionFile := filepath.Join(tmpDir, "test-session-"+string(rune('a'+i))+".jsonl")
		sessionContent := `{"type":"session_start","session_id":"test-` + string(rune('a'+i)) + `","project":"test-project","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","tool_name":"Read","timestamp":"2024-01-15T10:01:00Z","success":true,"duration":100}
`
		if err := os.WriteFile(sessionFile, []byte(sessionContent), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		files[i] = sessionFile
	}

	// Load all files
	for _, f := range files {
		if _, err := agg.LoadSession(f); err != nil {
			t.Fatalf("LoadSession failed for %s: %v", f, err)
		}
	}

	// Cache size should be limited to maxSize
	if agg.CacheSize() != maxSize {
		t.Errorf("CacheSize = %d, want %d", agg.CacheSize(), maxSize)
	}

	// First files should be evicted (LRU)
	if agg.IsCached(files[0]) {
		t.Error("first file should have been evicted")
	}
	if agg.IsCached(files[1]) {
		t.Error("second file should have been evicted")
	}

	// Last files should still be cached
	if !agg.IsCached(files[4]) {
		t.Error("last file should still be cached")
	}
	if !agg.IsCached(files[3]) {
		t.Error("second-to-last file should still be cached")
	}
}

func TestAggregator_ClearCache(t *testing.T) {
	tmpDir := t.TempDir()
	agg := NewAggregator(10)

	// Create and load multiple files
	for i := 0; i < 3; i++ {
		sessionFile := filepath.Join(tmpDir, "test-session-"+string(rune('a'+i))+".jsonl")
		sessionContent := `{"type":"session_start","session_id":"test-` + string(rune('a'+i)) + `","project":"test-project","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
`
		if err := os.WriteFile(sessionFile, []byte(sessionContent), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		if _, err := agg.LoadSession(sessionFile); err != nil {
			t.Fatalf("LoadSession failed: %v", err)
		}
	}

	if agg.CacheSize() != 3 {
		t.Errorf("CacheSize = %d, want 3", agg.CacheSize())
	}

	agg.ClearCache()

	if agg.CacheSize() != 0 {
		t.Errorf("CacheSize = %d after clear, want 0", agg.CacheSize())
	}
}

func TestAggregator_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	agg := NewAggregator(10)

	// Create test file
	sessionFile := filepath.Join(tmpDir, "test-session.jsonl")
	sessionContent := `{"type":"session_start","session_id":"test-123","project":"test-project","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","tool_name":"Read","timestamp":"2024-01-15T10:01:00Z","success":true,"duration":100}
`
	if err := os.WriteFile(sessionFile, []byte(sessionContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Concurrent access
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := agg.LoadSession(sessionFile)
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent LoadSession error: %v", err)
	}

	// Should still have exactly 1 cached entry
	if agg.CacheSize() != 1 {
		t.Errorf("CacheSize = %d after concurrent access, want 1", agg.CacheSize())
	}
}

func TestAggregator_LoadSessionError(t *testing.T) {
	agg := NewAggregator(10)

	// Non-existent file
	_, err := agg.LoadSession("/nonexistent/path/file.jsonl")
	if err == nil {
		t.Error("LoadSession should fail for non-existent file")
	}

	// Invalid content
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.jsonl")
	if err := os.WriteFile(invalidFile, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err = agg.LoadSession(invalidFile)
	if err == nil {
		t.Error("LoadSession should fail for invalid JSONL")
	}
}

func TestAggregator_GetProjectMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create multiple session files with valid hex filenames
	sessions := []string{
		"agent-11111111.jsonl",
		"agent-22222222.jsonl",
	}

	for i, filename := range sessions {
		sessionFile := filepath.Join(projectDir, filename)
		success := "true"
		if i == 1 {
			success = "false"
		}
		sessionContent := `{"type":"session_start","session_id":"` + filename[6:14] + `","project":"test-project","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":` + success + `}
{"type":"tool_call","tool_name":"Read","timestamp":"2024-01-15T10:01:00Z","success":true,"duration":100}
{"type":"token_usage","timestamp":"2024-01-15T10:02:00Z","input_tokens":1000,"output_tokens":500,"cost_usd":0.01}
`
		if err := os.WriteFile(sessionFile, []byte(sessionContent), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
	}

	agg := NewAggregatorWithBaseDir(10, tmpDir)

	metrics, err := agg.GetProjectMetrics("test-project")
	if err != nil {
		t.Fatalf("GetProjectMetrics failed: %v", err)
	}

	if metrics.Project != "test-project" {
		t.Errorf("Project = %s, want test-project", metrics.Project)
	}
	if metrics.TotalSessions != 2 {
		t.Errorf("TotalSessions = %d, want 2", metrics.TotalSessions)
	}
	if metrics.SuccessRate != 0.5 {
		t.Errorf("SuccessRate = %f, want 0.5", metrics.SuccessRate)
	}
	if metrics.TotalCost == 0 {
		t.Error("TotalCost should be non-zero")
	}
}

func TestAggregator_GetSessionMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	sessionID := "11111111"
	sessionFile := filepath.Join(projectDir, "agent-"+sessionID+".jsonl")
	sessionContent := `{"type":"session_start","session_id":"` + sessionID + `","project":"test-project","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","tool_name":"Read","timestamp":"2024-01-15T10:01:00Z","success":true,"duration":100}
`
	if err := os.WriteFile(sessionFile, []byte(sessionContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	agg := NewAggregatorWithBaseDir(10, tmpDir)

	metrics, err := agg.GetSessionMetrics("test-project", sessionID)
	if err != nil {
		t.Fatalf("GetSessionMetrics failed: %v", err)
	}

	if metrics.TotalSessions != 1 {
		t.Errorf("TotalSessions = %d, want 1", metrics.TotalSessions)
	}
}

func TestAggregator_ListSessions(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create session files with valid hex filenames
	sessions := []string{
		"agent-11111111.jsonl",
		"agent-22222222.jsonl",
	}

	for _, filename := range sessions {
		sessionFile := filepath.Join(projectDir, filename)
		if err := os.WriteFile(sessionFile, []byte(`{"type":"session_start","session_id":"test","project":"test","timestamp":"2024-01-15T10:00:00Z"}`), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
	}

	agg := NewAggregatorWithBaseDir(10, tmpDir)

	sessionInfos, err := agg.ListSessions("test-project")
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if len(sessionInfos) != 2 {
		t.Errorf("ListSessions returned %d sessions, want 2", len(sessionInfos))
	}
}

func TestAggregator_IsCached(t *testing.T) {
	agg := NewAggregator(10)

	// Not cached initially
	if agg.IsCached("/some/path") {
		t.Error("path should not be cached initially")
	}

	// Zero mtime for uncached
	mtime := agg.GetCachedMtime("/some/path")
	if !mtime.IsZero() {
		t.Error("mtime should be zero for uncached path")
	}
}

func TestAggregator_ListSessionsFiltered(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create session files with different timestamps
	now := time.Now()
	sessions := []struct {
		filename  string
		timestamp time.Time
	}{
		{"agent-11111111.jsonl", now.Add(-48 * time.Hour)}, // 2 days ago
		{"agent-22222222.jsonl", now.Add(-24 * time.Hour)}, // 1 day ago
		{"agent-33333333.jsonl", now.Add(-1 * time.Hour)},  // 1 hour ago
	}

	for _, s := range sessions {
		sessionFile := filepath.Join(projectDir, s.filename)
		ts := s.timestamp.Format(time.RFC3339)
		sessionContent := `{"type":"session_start","session_id":"` + s.filename[6:14] + `","project":"test-project","timestamp":"` + ts + `","status":"completed","success":true}
`
		if err := os.WriteFile(sessionFile, []byte(sessionContent), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		// Set file mtime to match
		os.Chtimes(sessionFile, s.timestamp, s.timestamp)
	}

	agg := NewAggregatorWithBaseDir(10, tmpDir)

	// Test without filter - should get all 3
	all, err := agg.ListSessions("test-project")
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("ListSessions returned %d sessions, want 3", len(all))
	}

	// Test with time range filter - last 2 hours
	criteria := FilterCriteria{
		Since: now.Add(-2 * time.Hour),
	}
	filtered, err := agg.ListSessionsFiltered("test-project", criteria)
	if err != nil {
		t.Fatalf("ListSessionsFiltered failed: %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("ListSessionsFiltered returned %d sessions, want 1", len(filtered))
	}

	// Test with time range filter - last 36 hours
	criteria = FilterCriteria{
		Since: now.Add(-36 * time.Hour),
	}
	filtered, err = agg.ListSessionsFiltered("test-project", criteria)
	if err != nil {
		t.Fatalf("ListSessionsFiltered failed: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("ListSessionsFiltered returned %d sessions, want 2", len(filtered))
	}
}

func TestAggregator_GetProjectMetricsFiltered(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create session files with different agents
	now := time.Now()
	sessions := []struct {
		filename  string
		agentName string
		timestamp time.Time
	}{
		{"agent-11111111.jsonl", "code-reviewer", now.Add(-48 * time.Hour)},
		{"agent-22222222.jsonl", "test-automator", now.Add(-24 * time.Hour)},
		{"agent-33333333.jsonl", "code-reviewer", now.Add(-1 * time.Hour)},
	}

	for _, s := range sessions {
		sessionFile := filepath.Join(projectDir, s.filename)
		ts := s.timestamp.Format(time.RFC3339)
		sessionContent := `{"type":"session_start","session_id":"` + s.filename[6:14] + `","project":"test-project","timestamp":"` + ts + `","status":"completed","success":true,"agent_name":"` + s.agentName + `"}
{"type":"token_usage","timestamp":"` + ts + `","input_tokens":1000,"output_tokens":500,"cost_usd":0.01}
`
		if err := os.WriteFile(sessionFile, []byte(sessionContent), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		os.Chtimes(sessionFile, s.timestamp, s.timestamp)
	}

	agg := NewAggregatorWithBaseDir(10, tmpDir)

	// Test filtering by time range - last 36 hours
	criteria := FilterCriteria{
		Since: now.Add(-36 * time.Hour),
	}
	metrics, err := agg.GetProjectMetricsFiltered("test-project", criteria)
	if err != nil {
		t.Fatalf("GetProjectMetricsFiltered failed: %v", err)
	}
	// Should include sessions from last 36 hours (2 sessions)
	if metrics.TotalSessions != 2 {
		t.Errorf("TotalSessions = %d, want 2", metrics.TotalSessions)
	}

	// Test filtering by agent name
	criteria = FilterCriteria{
		Search: "code-reviewer",
	}
	metrics, err = agg.GetProjectMetricsFiltered("test-project", criteria)
	if err != nil {
		t.Fatalf("GetProjectMetricsFiltered failed: %v", err)
	}
	// Should match sessions with code-reviewer agent (2 sessions have code-reviewer)
	if len(metrics.AgentPerformance) == 0 {
		t.Error("AgentPerformance should not be empty for agent filter")
	}
	if metrics.AgentPerformance["code-reviewer"] != 2 {
		t.Errorf("AgentPerformance[code-reviewer] = %d, want 2", metrics.AgentPerformance["code-reviewer"])
	}
}
