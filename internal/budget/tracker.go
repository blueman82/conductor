package budget

import (
	"sync"
	"time"
)

// UsageBlock represents a 5-hour billing window for Claude Code usage.
// Each block starts at an hour boundary and tracks all token usage within
// that 5-hour period.
type UsageBlock struct {
	ID            string       // ISO timestamp of block start (e.g., "2025-12-22T16:00:00Z")
	StartTime     time.Time    // When the block started
	EndTime       time.Time    // StartTime + 5 hours
	ActualEndTime time.Time    // Timestamp of last activity in this block
	TotalTokens   int64        // Total tokens (input + output)
	InputTokens   int64        // Input tokens only
	OutputTokens  int64        // Output tokens only
	CostUSD       float64      // Cumulative cost in USD
	Entries       []UsageEntry // Individual usage entries
	Models        []string     // Unique models used in this block
}

// UsageEntry represents a single API call's token usage.
// This corresponds to one Claude API request with detailed token breakdown.
type UsageEntry struct {
	Timestamp                time.Time
	InputTokens              int64
	OutputTokens             int64
	CacheCreationInputTokens int64
	CacheReadInputTokens     int64
	CostUSD                  float64
	Model                    string
}

// BurnRate tracks consumption velocity within a usage block.
// Calculated from actual usage history to predict future consumption.
type BurnRate struct {
	TokensPerMinute float64 // Average token consumption rate
	CostPerHour     float64 // Average cost burn rate in USD
}

// Projection estimates end-of-block usage based on current burn rate.
// Used to predict whether the 5-hour block will exceed budget thresholds.
type Projection struct {
	TotalTokens      int64   // Projected total tokens at block end
	TotalCost        float64 // Projected total cost at block end
	RemainingMinutes int     // Minutes remaining in block
}

// BlockStatus provides a comprehensive summary of block state.
// Includes current usage, elapsed time, burn rate, and projections.
type BlockStatus struct {
	Block            *UsageBlock
	ElapsedMinutes   int
	RemainingMinutes int
	PercentElapsed   float64
	BurnRate         *BurnRate
	Projection       *Projection
	IsActive         bool
}

// UsageTracker monitors Claude Code usage across sessions.
// Thread-safe tracking of 5-hour billing windows with burn rate analysis.
type UsageTracker struct {
	mu          sync.RWMutex
	baseDir     string                  // ~/.claude/projects
	costModel   map[string]ModelPricing // Model pricing configuration
	blocks      []UsageBlock            // Historical blocks
	activeBlock *UsageBlock             // Currently active block
}

// ModelPricing defines cost per million tokens for a specific model.
// Used to calculate USD cost from token usage.
type ModelPricing struct {
	InputPer1M  float64 // Cost per 1M input tokens
	OutputPer1M float64 // Cost per 1M output tokens
}

// floorToHour floors a timestamp to the nearest hour boundary.
// Used for determining 5-hour block boundaries.
//
// Example: 2025-12-22T16:37:42Z -> 2025-12-22T16:00:00Z
func floorToHour(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
}

// NewUsageTracker creates a new usage tracker with the specified base directory
// and cost model. The base directory should point to ~/.claude/projects.
func NewUsageTracker(baseDir string, costModel map[string]ModelPricing) *UsageTracker {
	return &UsageTracker{
		baseDir:   baseDir,
		costModel: costModel,
		blocks:    make([]UsageBlock, 0),
	}
}

// IsActive returns true if the block is still active.
// A block is active if:
// 1. Current time is before the block's end time (within 5-hour window)
// 2. Last activity was less than 5 hours ago (prevents stale blocks)
func (b *UsageBlock) IsActive() bool {
	now := time.Now()
	if now.After(b.EndTime) {
		return false
	}

	// If there's no activity yet, check if we're within the time window
	if b.ActualEndTime.IsZero() {
		return true
	}

	// Block is active if last activity was less than 5 hours ago
	return now.Sub(b.ActualEndTime) < 5*time.Hour
}

// CalculateBurnRate computes the current consumption velocity.
// Returns nil if there are fewer than 2 entries (insufficient data).
// Burn rate is calculated from first to last entry timestamp.
func (b *UsageBlock) CalculateBurnRate() *BurnRate {
	if len(b.Entries) < 2 {
		return nil
	}

	firstEntry := b.Entries[0]
	lastEntry := b.Entries[len(b.Entries)-1]

	duration := lastEntry.Timestamp.Sub(firstEntry.Timestamp)
	minutes := duration.Minutes()

	if minutes <= 0 {
		return nil
	}

	return &BurnRate{
		TokensPerMinute: float64(b.TotalTokens) / minutes,
		CostPerHour:     (b.CostUSD / minutes) * 60,
	}
}

// Project uses burn rate to project end-of-block totals.
// Returns nil if burn rate cannot be calculated.
// Projection assumes burn rate remains constant for remaining duration.
func (b *UsageBlock) Project() *Projection {
	burnRate := b.CalculateBurnRate()
	if burnRate == nil {
		return nil
	}

	remainingMinutes := time.Until(b.EndTime).Minutes()
	if remainingMinutes < 0 {
		remainingMinutes = 0
	}

	return &Projection{
		TotalTokens:      b.TotalTokens + int64(burnRate.TokensPerMinute*remainingMinutes),
		TotalCost:        b.CostUSD + (burnRate.CostPerHour/60)*remainingMinutes,
		RemainingMinutes: int(remainingMinutes),
	}
}

// AddEntry adds a usage entry to the block and updates all totals.
// Tracks unique models and updates the actual end time.
func (b *UsageBlock) AddEntry(entry UsageEntry) {
	b.Entries = append(b.Entries, entry)
	b.TotalTokens += entry.InputTokens + entry.OutputTokens
	b.InputTokens += entry.InputTokens
	b.OutputTokens += entry.OutputTokens
	b.CostUSD += entry.CostUSD
	b.ActualEndTime = entry.Timestamp

	// Track unique models
	modelExists := false
	for _, model := range b.Models {
		if model == entry.Model {
			modelExists = true
			break
		}
	}
	if !modelExists {
		b.Models = append(b.Models, entry.Model)
	}
}

// GetActiveBlock returns the currently active block.
// Thread-safe for concurrent access.
func (t *UsageTracker) GetActiveBlock() *UsageBlock {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.activeBlock
}

// GetBlocks returns a copy of all blocks.
// Thread-safe for concurrent access.
func (t *UsageTracker) GetBlocks() []UsageBlock {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Return a copy to prevent external modification
	blocksCopy := make([]UsageBlock, len(t.blocks))
	copy(blocksCopy, t.blocks)
	return blocksCopy
}

// GetStatus returns comprehensive status of the active block.
// Includes elapsed time, burn rate, and projections.
// Returns nil if no active block exists.
func (t *UsageTracker) GetStatus() *BlockStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.activeBlock == nil {
		return nil
	}

	now := time.Now()
	elapsed := now.Sub(t.activeBlock.StartTime)
	remaining := t.activeBlock.EndTime.Sub(now)

	if remaining < 0 {
		remaining = 0
	}

	elapsedMinutes := int(elapsed.Minutes())
	remainingMinutes := int(remaining.Minutes())
	percentElapsed := (elapsed.Minutes() / (5 * 60)) * 100

	return &BlockStatus{
		Block:            t.activeBlock,
		ElapsedMinutes:   elapsedMinutes,
		RemainingMinutes: remainingMinutes,
		PercentElapsed:   percentElapsed,
		BurnRate:         t.activeBlock.CalculateBurnRate(),
		Projection:       t.activeBlock.Project(),
		IsActive:         t.activeBlock.IsActive(),
	}
}

// DefaultCostModel returns the default pricing model for Claude models.
// Prices are in USD per million tokens.
// Matches pricing from internal/behavioral/analytics.go ModelCosts.
func DefaultCostModel() map[string]ModelPricing {
	return map[string]ModelPricing{
		"claude-opus-4-5-20251101": {
			InputPer1M:  15.00,
			OutputPer1M: 75.00,
		},
		"claude-sonnet-4-5-20250929": {
			InputPer1M:  3.00,
			OutputPer1M: 15.00,
		},
		"claude-sonnet-3-7-20250219": {
			InputPer1M:  3.00,
			OutputPer1M: 15.00,
		},
		"claude-3-5-sonnet-20241022": {
			InputPer1M:  3.00,
			OutputPer1M: 15.00,
		},
		"claude-3-5-sonnet-20240620": {
			InputPer1M:  3.00,
			OutputPer1M: 15.00,
		},
		"claude-3-5-haiku-20241022": {
			InputPer1M:  1.00,
			OutputPer1M: 5.00,
		},
		"claude-3-opus-20240229": {
			InputPer1M:  15.00,
			OutputPer1M: 75.00,
		},
		"claude-3-sonnet-20240229": {
			InputPer1M:  3.00,
			OutputPer1M: 15.00,
		},
		"claude-3-haiku-20240307": {
			InputPer1M:  0.25,
			OutputPer1M: 1.25,
		},
	}
}
