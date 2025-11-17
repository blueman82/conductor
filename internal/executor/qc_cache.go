package executor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// QCSelectionCache caches intelligent agent selection results to avoid redundant API calls
type QCSelectionCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
}

type cacheEntry struct {
	agents    []string
	rationale string
	expiresAt time.Time
}

// IntelligentSelectionResult contains the result of an intelligent agent selection
type IntelligentSelectionResult struct {
	Agents    []string `json:"agents"`
	Rationale string   `json:"rationale"`
}

// NewQCSelectionCache creates a new cache with the specified TTL
func NewQCSelectionCache(ttlSeconds int) *QCSelectionCache {
	if ttlSeconds <= 0 {
		ttlSeconds = 3600 // Default 1 hour
	}
	return &QCSelectionCache{
		entries: make(map[string]*cacheEntry),
		ttl:     time.Duration(ttlSeconds) * time.Second,
	}
}

// Get retrieves a cached selection result if available and not expired
func (c *QCSelectionCache) Get(key string) (*IntelligentSelectionResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		// Entry expired - will be cleaned up later
		return nil, false
	}

	return &IntelligentSelectionResult{
		Agents:    entry.agents,
		Rationale: entry.rationale,
	}, true
}

// Set stores a selection result in the cache
func (c *QCSelectionCache) Set(key string, result *IntelligentSelectionResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &cacheEntry{
		agents:    result.Agents,
		rationale: result.Rationale,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// GenerateCacheKey creates a unique key for a task + executing agent combination
func GenerateCacheKey(task models.Task, executingAgent string) string {
	// Sort files for consistent hashing
	sortedFiles := make([]string, len(task.Files))
	copy(sortedFiles, task.Files)
	sort.Strings(sortedFiles)

	// Combine task identifiers
	data := fmt.Sprintf(
		"task:%s|files:%s|agent:%s|name:%s",
		task.Number,
		strings.Join(sortedFiles, ","),
		executingAgent,
		task.Name,
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16]) // Use first 16 bytes for shorter key
}

// Cleanup removes expired entries from the cache
func (c *QCSelectionCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, key)
		}
	}
}

// Size returns the number of entries in the cache
func (c *QCSelectionCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Clear removes all entries from the cache
func (c *QCSelectionCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry)
}
