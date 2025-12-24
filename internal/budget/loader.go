package budget

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LoadUsage loads all Claude Code session data from JSONL files and builds usage blocks.
// This function:
// 1. Discovers all JSONL session files in baseDir (~/.claude/projects)
// 2. Parses each file to extract usage entries
// 3. Groups entries into 5-hour blocks using identifyBlocks()
// 4. Sets activeBlock to the currently active window (if any)
//
// Thread-safe: Acquires write lock on UsageTracker.
func LoadUsage(t *UsageTracker) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Discover all JSONL files
	sessionFiles, err := discoverSessionFiles(t.baseDir)
	if err != nil {
		return fmt.Errorf("failed to discover session files: %w", err)
	}

	if len(sessionFiles) == 0 {
		// No session files found - this is valid for new installations
		return nil
	}

	// Parse all session files to extract usage entries
	var allEntries []UsageEntry
	for _, filePath := range sessionFiles {
		entries, err := parseSessionFile(filePath, t.costModel)
		if err != nil {
			// Log error but continue processing other files
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", filePath, err)
			continue
		}
		allEntries = append(allEntries, entries...)
	}

	if len(allEntries) == 0 {
		// No usage data found
		return nil
	}

	// Group entries into 5-hour blocks
	blocks := identifyBlocks(allEntries)
	t.blocks = blocks

	// Set active block to the last block if it's still active
	if len(blocks) > 0 {
		lastBlock := &blocks[len(blocks)-1]
		if lastBlock.IsActive() {
			t.activeBlock = lastBlock
		}
	}

	return nil
}

// discoverSessionFiles recursively finds all *.jsonl files in baseDir.
// Skips hidden directories (starting with '.').
// Expands ~ to home directory if present in baseDir.
func discoverSessionFiles(baseDir string) ([]string, error) {
	// Expand ~ to home directory
	if strings.HasPrefix(baseDir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		baseDir = filepath.Join(home, baseDir[2:])
	}

	// Check if base directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("base directory does not exist: %s", baseDir)
	}

	var sessionFiles []string

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories (but not the base directory itself)
		if info.IsDir() && info.Name() != filepath.Base(baseDir) && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// Collect JSONL files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".jsonl") {
			sessionFiles = append(sessionFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return sessionFiles, nil
}

// parseSessionFile parses a single JSONL file and extracts usage entries.
// Each line is a JSON object that may contain usage data under message.usage.
//
// Expected structure:
//
//	{
//	  "type": "assistant",
//	  "timestamp": "2025-12-22T16:37:42Z",
//	  "message": {
//	    "usage": {
//	      "input_tokens": 123,
//	      "output_tokens": 456,
//	      "cache_creation_input_tokens": 789,  // optional
//	      "cache_read_input_tokens": 101       // optional
//	    },
//	    "model": "claude-sonnet-4-5-20250929"
//	  }
//	}
func parseSessionFile(path string, costModel map[string]ModelPricing) ([]UsageEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var entries []UsageEntry
	scanner := bufio.NewScanner(file)

	// Increase buffer size for large JSONL lines (some can be >1MB)
	const maxScanTokenSize = 10 * 1024 * 1024 // 10MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Parse JSON line
		var record map[string]interface{}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			// Skip malformed lines but continue processing
			continue
		}

		// Extract usage data if present
		entry, ok := extractUsageEntry(record, costModel)
		if ok {
			entries = append(entries, entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file at line %d: %w", lineNum, err)
	}

	return entries, nil
}

// extractUsageEntry extracts a UsageEntry from a JSONL record.
// Returns (entry, true) if usage data found, (empty, false) otherwise.
func extractUsageEntry(record map[string]interface{}, costModel map[string]ModelPricing) (UsageEntry, bool) {
	// Extract timestamp
	timestampStr, ok := record["timestamp"].(string)
	if !ok {
		return UsageEntry{}, false
	}

	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return UsageEntry{}, false
	}

	// Extract message.usage and message.model
	message, ok := record["message"].(map[string]interface{})
	if !ok {
		return UsageEntry{}, false
	}

	usage, ok := message["usage"].(map[string]interface{})
	if !ok {
		return UsageEntry{}, false
	}

	// Extract model
	model, _ := message["model"].(string)
	if model == "" {
		model = "unknown"
	}

	// Extract token counts (with fallbacks for missing fields)
	inputTokens := extractInt64(usage, "input_tokens")
	outputTokens := extractInt64(usage, "output_tokens")
	cacheCreationTokens := extractInt64(usage, "cache_creation_input_tokens")
	cacheReadTokens := extractInt64(usage, "cache_read_input_tokens")

	// Calculate cost
	cost := calculateCost(inputTokens, outputTokens, model, costModel)

	return UsageEntry{
		Timestamp:                timestamp,
		InputTokens:              inputTokens,
		OutputTokens:             outputTokens,
		CacheCreationInputTokens: cacheCreationTokens,
		CacheReadInputTokens:     cacheReadTokens,
		CostUSD:                  cost,
		Model:                    model,
	}, true
}

// extractInt64 safely extracts an int64 value from a map.
// Returns 0 if key not found or type assertion fails.
func extractInt64(m map[string]interface{}, key string) int64 {
	val, ok := m[key]
	if !ok {
		return 0
	}

	// Handle both float64 (JSON numbers) and int
	switch v := val.(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	default:
		return 0
	}
}

// calculateCost computes USD cost from token counts and model.
// Falls back to Sonnet pricing if model not found in costModel.
func calculateCost(inputTokens, outputTokens int64, model string, costModel map[string]ModelPricing) float64 {
	pricing, ok := costModel[model]
	if !ok {
		// Default to Sonnet pricing for unknown models
		pricing = ModelPricing{InputPer1M: 3.0, OutputPer1M: 15.0}
	}

	inputCost := float64(inputTokens) / 1_000_000 * pricing.InputPer1M
	outputCost := float64(outputTokens) / 1_000_000 * pricing.OutputPer1M

	return inputCost + outputCost
}

// identifyBlocks groups usage entries into 5-hour billing blocks.
// Algorithm:
// 1. Sort entries by timestamp
// 2. For each entry, determine appropriate block (floor to hour boundary)
// 3. Detect gaps: if entry timestamp > 5 hours from last activity, start new block
// 4. Block ends at start + 5 hours
//
// A new block is created when:
// - No current block exists
// - Entry timestamp is after current block's end time
// - Gap > 5 hours from last activity (session ended)
func identifyBlocks(entries []UsageEntry) []UsageBlock {
	if len(entries) == 0 {
		return nil
	}

	// Sort entries by timestamp
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})

	var blocks []UsageBlock
	var currentBlock *UsageBlock

	for _, entry := range entries {
		// Determine which block this entry belongs to
		blockStart := floorToHour(entry.Timestamp)
		blockEnd := blockStart.Add(5 * time.Hour)
		blockID := blockStart.Format(time.RFC3339)

		// Check if we need a new block:
		// 1. No current block
		// 2. Entry is outside current block's time window
		// 3. Gap > 5 hours from last activity (session ended)
		needNewBlock := currentBlock == nil ||
			entry.Timestamp.After(currentBlock.EndTime) ||
			(!currentBlock.ActualEndTime.IsZero() &&
				entry.Timestamp.Sub(currentBlock.ActualEndTime) > 5*time.Hour)

		if needNewBlock {
			// Save current block if exists
			if currentBlock != nil {
				blocks = append(blocks, *currentBlock)
			}

			// Start new block
			currentBlock = &UsageBlock{
				ID:        blockID,
				StartTime: blockStart,
				EndTime:   blockEnd,
				Entries:   make([]UsageEntry, 0),
				Models:    make([]string, 0),
			}
		}

		currentBlock.AddEntry(entry)
	}

	// Don't forget the last block
	if currentBlock != nil {
		blocks = append(blocks, *currentBlock)
	}

	return blocks
}
