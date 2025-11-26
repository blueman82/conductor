package behavioral

import (
	"container/list"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DefaultMaxCacheSize is the default maximum number of sessions to cache
const DefaultMaxCacheSize = 50

// cacheEntry holds cached metrics with file modification tracking
type cacheEntry struct {
	metrics  *BehavioralMetrics
	mtime    time.Time
	filePath string
}

// Aggregator manages session loading with LRU caching
type Aggregator struct {
	mu           sync.RWMutex
	cache        map[string]*list.Element // filepath -> list element
	lruList      *list.List               // LRU ordered list of cache keys
	maxCacheSize int
	baseDir      string // base directory for session discovery
}

// NewAggregator creates a new Aggregator with the specified max cache size
func NewAggregator(maxCacheSize int) *Aggregator {
	if maxCacheSize <= 0 {
		maxCacheSize = DefaultMaxCacheSize
	}
	return &Aggregator{
		cache:        make(map[string]*list.Element),
		lruList:      list.New(),
		maxCacheSize: maxCacheSize,
		baseDir:      "~/.claude/projects",
	}
}

// NewAggregatorWithBaseDir creates a new Aggregator with custom base directory
func NewAggregatorWithBaseDir(maxCacheSize int, baseDir string) *Aggregator {
	agg := NewAggregator(maxCacheSize)
	agg.baseDir = baseDir
	return agg
}

// LoadSession loads and caches metrics for a session file
// Returns cached metrics if file hasn't changed, otherwise re-parses
func (a *Aggregator) LoadSession(filePath string) (*BehavioralMetrics, error) {
	// Get file info for mtime check
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat session file: %w", err)
	}
	mtime := info.ModTime()

	// Check cache with read lock first
	a.mu.RLock()
	if elem, ok := a.cache[filePath]; ok {
		entry := elem.Value.(*cacheEntry)
		if entry.mtime.Equal(mtime) {
			// Cache hit - move to front and return
			a.mu.RUnlock()
			a.mu.Lock()
			a.lruList.MoveToFront(elem)
			metrics := entry.metrics
			a.mu.Unlock()
			return metrics, nil
		}
	}
	a.mu.RUnlock()

	// Cache miss or stale - parse the file
	sessionData, err := ParseSessionFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse session file: %w", err)
	}

	// Validate that we got usable session data
	if sessionData == nil || (sessionData.Session.ID == "" && len(sessionData.Events) == 0) {
		return nil, fmt.Errorf("no valid session data found in file")
	}

	metrics := ExtractMetrics(sessionData)

	// Store in cache with write lock
	a.mu.Lock()
	defer a.mu.Unlock()

	entry := &cacheEntry{
		metrics:  metrics,
		mtime:    mtime,
		filePath: filePath,
	}

	// Check if already in cache (might have been added by another goroutine)
	if elem, ok := a.cache[filePath]; ok {
		// Update existing entry
		elem.Value = entry
		a.lruList.MoveToFront(elem)
	} else {
		// Add new entry
		elem := a.lruList.PushFront(entry)
		a.cache[filePath] = elem

		// Evict oldest if over capacity
		a.evictOldestLocked()
	}

	return metrics, nil
}

// GetSessionMetrics retrieves metrics for a specific session by project and session ID
func (a *Aggregator) GetSessionMetrics(project, sessionID string) (*BehavioralMetrics, error) {
	// Build expected file path
	expandedDir, err := expandHomeDir(a.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to expand base directory: %w", err)
	}

	// Try both file formats: agent-{id}.jsonl and {uuid}.jsonl
	filenames := []string{
		fmt.Sprintf("agent-%s.jsonl", sessionID),
		fmt.Sprintf("%s.jsonl", sessionID),
	}

	for _, filename := range filenames {
		filePath := filepath.Join(expandedDir, project, filename)
		if _, err := os.Stat(filePath); err == nil {
			return a.LoadSession(filePath)
		}
	}

	return nil, fmt.Errorf("session file not found for ID: %s", sessionID)
}

// ListSessions returns all discovered sessions for a project
func (a *Aggregator) ListSessions(project string) ([]SessionInfo, error) {
	return DiscoverProjectSessions(a.baseDir, project)
}

// ListSessionsFiltered returns sessions filtered by criteria (time range, agent name)
func (a *Aggregator) ListSessionsFiltered(project string, criteria FilterCriteria) ([]SessionInfo, error) {
	sessions, err := a.ListSessions(project)
	if err != nil {
		return nil, err
	}

	var filtered []SessionInfo
	for _, session := range sessions {
		// Apply time range filter using CreatedAt
		if !criteria.Since.IsZero() && session.CreatedAt.Before(criteria.Since) {
			continue
		}
		if !criteria.Until.IsZero() && session.CreatedAt.After(criteria.Until) {
			continue
		}
		filtered = append(filtered, session)
	}
	return filtered, nil
}

// AggregateProjectMetrics aggregates metrics across all sessions in a project
type AggregateProjectMetrics struct {
	Project          string         `json:"project"`
	TotalSessions    int            `json:"total_sessions"`
	SuccessRate      float64        `json:"success_rate"`
	AverageDuration  time.Duration  `json:"average_duration"`
	TotalCost        float64        `json:"total_cost"`
	TotalErrors      int            `json:"total_errors"`
	ErrorRate        float64        `json:"error_rate"`
	ToolUsageCounts  map[string]int `json:"tool_usage_counts"`
	AgentPerformance map[string]int `json:"agent_performance"`
	TokenUsage       TokenUsage     `json:"token_usage"`
	Sessions         []SessionInfo  `json:"sessions"`
}

// GetProjectMetrics aggregates metrics across all sessions in a project
func (a *Aggregator) GetProjectMetrics(project string) (*AggregateProjectMetrics, error) {
	return a.GetProjectMetricsFiltered(project, FilterCriteria{})
}

// GetProjectMetricsFiltered aggregates metrics with optional filtering by time range and agent
func (a *Aggregator) GetProjectMetricsFiltered(project string, criteria FilterCriteria) (*AggregateProjectMetrics, error) {
	sessions, err := a.ListSessions(project)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	// Filter sessions by time range if specified
	var filteredSessions []SessionInfo
	for _, session := range sessions {
		// Apply time range filter using CreatedAt
		if !criteria.Since.IsZero() && session.CreatedAt.Before(criteria.Since) {
			continue
		}
		if !criteria.Until.IsZero() && session.CreatedAt.After(criteria.Until) {
			continue
		}
		filteredSessions = append(filteredSessions, session)
	}

	result := &AggregateProjectMetrics{
		Project:          project,
		TotalSessions:    len(filteredSessions),
		ToolUsageCounts:  make(map[string]int),
		AgentPerformance: make(map[string]int),
		Sessions:         filteredSessions,
	}

	if len(filteredSessions) == 0 {
		return result, nil
	}

	// Aggregate metrics from filtered sessions
	var totalDuration time.Duration
	successCount := 0
	var allMetrics []*BehavioralMetrics

	for _, session := range filteredSessions {
		metrics, err := a.LoadSession(session.FilePath)
		if err != nil {
			// Log warning but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to load session %s: %v\n", session.SessionID, err)
			continue
		}

		// Apply agent name filter if specified in Search
		if criteria.Search != "" {
			// Check if metrics contains this agent
			found := false
			for agent := range metrics.AgentPerformance {
				if containsIgnoreCase(agent, criteria.Search) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		allMetrics = append(allMetrics, metrics)

		// Aggregate basic metrics
		totalDuration += metrics.AverageDuration
		result.TotalCost += metrics.TotalCost
		result.TotalErrors += metrics.TotalErrors

		if metrics.SuccessRate >= 0.5 {
			successCount++
		}

		// Aggregate tool usage
		for _, tool := range metrics.ToolExecutions {
			result.ToolUsageCounts[tool.Name] += tool.Count
		}

		// Aggregate agent performance
		for agent, count := range metrics.AgentPerformance {
			result.AgentPerformance[agent] += count
		}

		// Aggregate token usage
		result.TokenUsage.InputTokens += metrics.TokenUsage.InputTokens
		result.TokenUsage.OutputTokens += metrics.TokenUsage.OutputTokens
		result.TokenUsage.CostUSD += metrics.TokenUsage.CostUSD
	}

	// Calculate averages
	if len(allMetrics) > 0 {
		result.SuccessRate = float64(successCount) / float64(len(allMetrics))
		result.AverageDuration = totalDuration / time.Duration(len(allMetrics))
		result.ErrorRate = float64(result.TotalErrors) / float64(len(allMetrics))
	}

	return result, nil
}

// InvalidateCache removes a specific file from the cache
func (a *Aggregator) InvalidateCache(filePath string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if elem, ok := a.cache[filePath]; ok {
		a.lruList.Remove(elem)
		delete(a.cache, filePath)
	}
}

// ClearCache removes all entries from the cache
func (a *Aggregator) ClearCache() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.cache = make(map[string]*list.Element)
	a.lruList = list.New()
}

// CacheSize returns the current number of cached sessions
func (a *Aggregator) CacheSize() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.cache)
}

// evictOldestLocked removes the oldest cache entry if over capacity
// Must be called with write lock held
func (a *Aggregator) evictOldestLocked() {
	for len(a.cache) > a.maxCacheSize {
		oldest := a.lruList.Back()
		if oldest == nil {
			break
		}
		entry := oldest.Value.(*cacheEntry)
		delete(a.cache, entry.filePath)
		a.lruList.Remove(oldest)
	}
}

// containsIgnoreCase checks if s contains substr (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(substr) == 0 ||
			(len(s) > 0 && containsLower(toLower(s), toLower(substr))))
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// IsCached checks if a file is currently cached
func (a *Aggregator) IsCached(filePath string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	_, ok := a.cache[filePath]
	return ok
}

// GetCachedMtime returns the cached mtime for a file, or zero if not cached
func (a *Aggregator) GetCachedMtime(filePath string) time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if elem, ok := a.cache[filePath]; ok {
		entry := elem.Value.(*cacheEntry)
		return entry.mtime
	}
	return time.Time{}
}
