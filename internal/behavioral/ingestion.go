package behavioral

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/harrison/conductor/internal/learning"
)

// IngestionConfig holds configuration for the IngestionEngine
type IngestionConfig struct {
	RootDir      string        // Root directory to watch for JSONL files
	Pattern      string        // File pattern to match (e.g., "*.jsonl")
	BatchSize    int           // Max events before flush (default: 50)
	BatchTimeout time.Duration // Max time before flush (default: 500ms)
	LockFile     string        // Path to lock file for single-writer guard
}

// DefaultIngestionConfig returns sensible defaults
func DefaultIngestionConfig() IngestionConfig {
	return IngestionConfig{
		Pattern:      "*.jsonl",
		BatchSize:    50,
		BatchTimeout: 500 * time.Millisecond,
		LockFile:     ".conductor/ingestion.lock",
	}
}

// IngestionStats holds runtime statistics for the engine
type IngestionStats struct {
	FilesTracked    int           // Number of files being tracked
	EventsProcessed int64         // Total events processed
	EventsPending   int           // Events waiting to be flushed
	SessionsCreated int64         // Total sessions created
	LastFlushTime   time.Time     // Last successful flush
	Errors          int64         // Total errors encountered
	Uptime          time.Duration // Time since engine started
}

// sessionTimeData tracks timestamp and model info for duration calculation
type sessionTimeData struct {
	firstTimestamp time.Time
	lastTimestamp  time.Time
	model          string
}

// IngestionEngine coordinates real-time JSONL ingestion
type IngestionEngine struct {
	watcher *FileWatcher
	store   *learning.Store
	config  IngestionConfig

	// Internal state
	pendingEvents    []eventWithMeta
	sessionCache     map[string]int64           // externalID -> internal sessionID
	sessionTimeCache map[int64]*sessionTimeData // sessionID -> timestamp tracking
	mu               sync.Mutex

	// Lock file handling
	lockFile *os.File
	lockPath string

	// Statistics
	stats struct {
		filesTracked    int64
		eventsProcessed int64
		sessionsCreated int64
		errors          int64
		startTime       time.Time
		lastFlush       time.Time
	}

	// Control channels
	done   chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
}

// eventWithMeta wraps an event with metadata for batch processing
type eventWithMeta struct {
	event     Event
	sessionID string // external session ID
	filePath  string
}

// NewIngestionEngine creates a new IngestionEngine
func NewIngestionEngine(store *learning.Store, cfg IngestionConfig) (*IngestionEngine, error) {
	if store == nil {
		return nil, fmt.Errorf("store is required")
	}

	// Apply defaults
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = 500 * time.Millisecond
	}
	if cfg.Pattern == "" {
		cfg.Pattern = "*.jsonl"
	}
	if cfg.LockFile == "" {
		cfg.LockFile = ".conductor/ingestion.lock"
	}

	// Create watcher but don't start yet
	watcher, err := NewFileWatcher(cfg.RootDir, cfg.Pattern)
	if err != nil {
		return nil, fmt.Errorf("create file watcher: %w", err)
	}

	e := &IngestionEngine{
		watcher:          watcher,
		store:            store,
		config:           cfg,
		pendingEvents:    make([]eventWithMeta, 0, cfg.BatchSize),
		sessionCache:     make(map[string]int64),
		sessionTimeCache: make(map[int64]*sessionTimeData),
		done:             make(chan struct{}),
		lockPath:         cfg.LockFile,
	}

	return e, nil
}

// Start begins the ingestion engine
// Acquires exclusive lock and starts watching for file events
func (e *IngestionEngine) Start(ctx context.Context) error {
	// Acquire lock file
	if err := e.acquireLock(); err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}

	e.ctx, e.cancel = context.WithCancel(ctx)
	e.stats.startTime = time.Now()

	// Scan existing files first (for one-time import and initial catch-up)
	if err := e.scanExistingFiles(); err != nil {
		return fmt.Errorf("scan existing files: %w", err)
	}

	// Start processing goroutines for new file events
	go e.processFileEvents()
	go e.flushTimer()

	return nil
}

// scanExistingFiles finds and processes all existing JSONL files
func (e *IngestionEngine) scanExistingFiles() error {
	var files []string

	err := filepath.Walk(e.config.RootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) || os.IsPermission(err) {
				return nil // Skip inaccessible paths
			}
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Match pattern
		if matched, _ := filepath.Match(e.config.Pattern, filepath.Base(path)); matched {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("walk directory: %w", err)
	}

	// Track file count
	e.mu.Lock()
	e.stats.filesTracked = int64(len(files))
	e.mu.Unlock()

	// Process each file
	for _, filePath := range files {
		select {
		case <-e.ctx.Done():
			return e.ctx.Err()
		default:
		}

		if err := e.readNewLines(filePath); err != nil {
			e.mu.Lock()
			e.stats.errors++
			e.mu.Unlock()
			// Continue processing other files
			continue
		}
	}

	// Flush any remaining events
	e.mu.Lock()
	if len(e.pendingEvents) > 0 {
		e.flushBatchLocked()
	}
	e.mu.Unlock()

	return nil
}

// Stop gracefully shuts down the engine
func (e *IngestionEngine) Stop() error {
	// Signal shutdown
	if e.cancel != nil {
		e.cancel()
	}
	close(e.done)

	// Flush pending events
	e.mu.Lock()
	if len(e.pendingEvents) > 0 {
		e.flushBatchLocked()
	}
	e.mu.Unlock()

	// Close watcher
	if e.watcher != nil {
		e.watcher.Close()
	}

	// Release lock
	e.releaseLock()

	return nil
}

// Stats returns current ingestion statistics
func (e *IngestionEngine) Stats() IngestionStats {
	e.mu.Lock()
	defer e.mu.Unlock()

	uptime := time.Duration(0)
	if !e.stats.startTime.IsZero() {
		uptime = time.Since(e.stats.startTime)
	}

	return IngestionStats{
		FilesTracked:    int(e.stats.filesTracked),
		EventsProcessed: e.stats.eventsProcessed,
		EventsPending:   len(e.pendingEvents),
		SessionsCreated: e.stats.sessionsCreated,
		LastFlushTime:   e.stats.lastFlush,
		Errors:          e.stats.errors,
		Uptime:          uptime,
	}
}

// acquireLock acquires exclusive lock on the lock file
func (e *IngestionEngine) acquireLock() error {
	// Ensure lock directory exists
	lockDir := filepath.Dir(e.lockPath)
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		return fmt.Errorf("create lock directory: %w", err)
	}

	// Open/create lock file
	f, err := os.OpenFile(e.lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}

	// Try to acquire exclusive lock (non-blocking)
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return fmt.Errorf("another ingestion engine is running (lock held)")
	}

	e.lockFile = f
	return nil
}

// releaseLock releases the exclusive lock
func (e *IngestionEngine) releaseLock() {
	if e.lockFile != nil {
		syscall.Flock(int(e.lockFile.Fd()), syscall.LOCK_UN)
		e.lockFile.Close()
		e.lockFile = nil
	}
}

// processFileEvents handles file watcher events
func (e *IngestionEngine) processFileEvents() {
	for {
		select {
		case <-e.done:
			return
		case <-e.ctx.Done():
			return
		case event, ok := <-e.watcher.Events():
			if !ok {
				return
			}
			e.handleFileEvent(event)
		case err, ok := <-e.watcher.Errors():
			if !ok {
				return
			}
			e.mu.Lock()
			e.stats.errors++
			e.mu.Unlock()
			fmt.Fprintf(os.Stderr, "watcher error: %v\n", err)
		}
	}
}

// handleFileEvent processes a single file event
func (e *IngestionEngine) handleFileEvent(event FileEvent) {
	switch event.Op {
	case FileWritten, FileCreated:
		// Read new lines from the file
		if err := e.readNewLines(event.Path); err != nil {
			e.mu.Lock()
			e.stats.errors++
			e.mu.Unlock()
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", event.Path, err)
		}
	case FileRemoved:
		// Clean up tracking for removed file
		e.mu.Lock()
		delete(e.sessionCache, event.Path)
		e.mu.Unlock()
	}
}

// readNewLines reads new lines from a file starting at tracked offset
func (e *IngestionEngine) readNewLines(filePath string) error {
	ctx := e.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	// Get current offset
	offset, err := e.store.GetFileOffset(ctx, filePath)
	if err != nil {
		return fmt.Errorf("get offset: %w", err)
	}

	var byteOffset int64 = 0
	var lastLineHash string
	if offset != nil {
		byteOffset = offset.ByteOffset
		lastLineHash = offset.LastLineHash
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	// Get file info for inode
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	// Check if file was truncated/rotated (size < offset)
	if info.Size() < byteOffset {
		byteOffset = 0
		lastLineHash = ""
	}

	// Seek to offset
	if byteOffset > 0 {
		if _, err := file.Seek(byteOffset, io.SeekStart); err != nil {
			return fmt.Errorf("seek: %w", err)
		}
	}

	// Read new lines
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // 10MB max line size

	var newLastLineHash string
	var newOffset int64 = byteOffset
	linesRead := 0
	skipFirst := lastLineHash != "" // Skip first line if we have a hash (might be partial from last read)

	for scanner.Scan() {
		line := scanner.Text()
		lineLen := int64(len(scanner.Bytes())) + 1 // +1 for newline
		newOffset += lineLen

		if line == "" {
			continue
		}

		// Compute hash of this line
		hash := hashLine(line)

		// Skip if this is the line we already processed
		if skipFirst && hash == lastLineHash {
			skipFirst = false
			continue
		}
		skipFirst = false

		// Parse and queue events from this line
		events, err := parseEventLine(line)
		if err != nil {
			// Log and continue - graceful degradation
			continue
		}

		// Extract session ID from line
		sessionID := extractSessionIDFromLine(line)

		// Queue events
		e.mu.Lock()
		for _, evt := range events {
			e.pendingEvents = append(e.pendingEvents, eventWithMeta{
				event:     evt,
				sessionID: sessionID,
				filePath:  filePath,
			})
		}

		// Check if we should flush
		if len(e.pendingEvents) >= e.config.BatchSize {
			e.flushBatchLocked()
		}
		e.mu.Unlock()

		newLastLineHash = hash
		linesRead++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan error: %w", err)
	}

	// Update offset if we read anything
	if linesRead > 0 || newOffset > byteOffset {
		// Get inode from stat
		var inode uint64
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			inode = stat.Ino
		}

		newOffsetRecord := &learning.IngestOffset{
			FilePath:     filePath,
			ByteOffset:   newOffset,
			Inode:        inode,
			LastLineHash: newLastLineHash,
		}
		if err := e.store.SetFileOffset(ctx, newOffsetRecord); err != nil {
			return fmt.Errorf("update offset: %w", err)
		}
	}

	return nil
}

// flushTimer periodically flushes pending events
func (e *IngestionEngine) flushTimer() {
	ticker := time.NewTicker(e.config.BatchTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-e.done:
			return
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.mu.Lock()
			if len(e.pendingEvents) > 0 {
				e.flushBatchLocked()
			}
			e.mu.Unlock()
		}
	}
}

// flushBatchLocked writes pending events to the database
// Must be called with e.mu held
func (e *IngestionEngine) flushBatchLocked() {
	if len(e.pendingEvents) == 0 {
		return
	}

	ctx := e.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	// Process each event
	for _, em := range e.pendingEvents {
		if err := e.processEvent(ctx, em); err != nil {
			e.stats.errors++
			fmt.Fprintf(os.Stderr, "error processing event: %v\n", err)
			continue
		}
		e.stats.eventsProcessed++
	}

	// Flush session timestamps to database
	e.flushSessionTimestamps(ctx)

	// Clear pending events
	e.pendingEvents = e.pendingEvents[:0]
	e.stats.lastFlush = time.Now()
}

// flushSessionTimestamps updates all tracked sessions with their calculated durations
func (e *IngestionEngine) flushSessionTimestamps(ctx context.Context) {
	for sessionID, data := range e.sessionTimeCache {
		// Calculate duration in seconds
		duration := int64(0)
		if !data.firstTimestamp.IsZero() && !data.lastTimestamp.IsZero() {
			duration = int64(data.lastTimestamp.Sub(data.firstTimestamp).Seconds())
		}

		// Update session with timestamps and duration
		if err := e.store.UpdateSessionTimestamps(ctx, sessionID, data.firstTimestamp, data.lastTimestamp, duration, data.model); err != nil {
			fmt.Fprintf(os.Stderr, "error updating session timestamps: %v\n", err)
		}
	}
	// Clear the cache after flushing
	e.sessionTimeCache = make(map[int64]*sessionTimeData)
}

// processEvent writes a single event to the database
func (e *IngestionEngine) processEvent(ctx context.Context, em eventWithMeta) error {
	// Get or create session
	sessionID, err := e.getOrCreateSession(ctx, em.sessionID, em.filePath)
	if err != nil {
		return fmt.Errorf("get/create session: %w", err)
	}

	// Track timestamp for duration calculation
	if baseEvent, ok := em.event.(interface{ GetTimestamp() time.Time }); ok {
		ts := baseEvent.GetTimestamp()
		if !ts.IsZero() {
			e.updateSessionTimestamp(sessionID, ts, "")
		}
	}

	// Write event based on type
	switch evt := em.event.(type) {
	case *ToolCallEvent:
		return e.store.AppendToolExecution(ctx, sessionID, learning.ToolExecutionData{
			SessionID:    sessionID,
			ToolName:     evt.ToolName,
			Parameters:   mapToJSON(evt.Parameters),
			DurationMs:   evt.Duration,
			Success:      evt.Success,
			ErrorMessage: evt.Error,
		})

	case *BashCommandEvent:
		return e.store.AppendBashCommand(ctx, sessionID, learning.BashCommandData{
			SessionID:    sessionID,
			Command:      evt.Command,
			DurationMs:   evt.Duration,
			ExitCode:     evt.ExitCode,
			StdoutLength: evt.OutputLength,
			Success:      evt.Success,
		})

	case *FileOperationEvent:
		return e.store.AppendFileOperation(ctx, sessionID, learning.FileOperationData{
			SessionID:     sessionID,
			OperationType: evt.Operation,
			FilePath:      evt.Path,
			DurationMs:    evt.Duration,
			BytesAffected: evt.SizeBytes,
			Success:       evt.Success,
			ErrorMessage:  evt.Error,
		})

	case *TokenUsageEvent:
		// Track model name for cost calculation
		if evt.ModelName != "" {
			e.updateSessionTimestamp(sessionID, evt.Timestamp, evt.ModelName)
		}
		tokens := evt.InputTokens + evt.OutputTokens
		return e.store.UpdateSessionAggregates(ctx, sessionID, tokens, 0)

	default:
		// Unknown event type - skip
		return nil
	}
}

// updateSessionTimestamp updates the first/last timestamp for a session
func (e *IngestionEngine) updateSessionTimestamp(sessionID int64, ts time.Time, model string) {
	data, exists := e.sessionTimeCache[sessionID]
	if !exists {
		data = &sessionTimeData{
			firstTimestamp: ts,
			lastTimestamp:  ts,
		}
		e.sessionTimeCache[sessionID] = data
	}

	// Update first timestamp if this is earlier
	if ts.Before(data.firstTimestamp) {
		data.firstTimestamp = ts
	}
	// Update last timestamp if this is later
	if ts.After(data.lastTimestamp) {
		data.lastTimestamp = ts
	}
	// Update model if provided
	if model != "" {
		data.model = model
	}
}

// getOrCreateSession returns internal session ID, creating if needed
func (e *IngestionEngine) getOrCreateSession(ctx context.Context, externalID, filePath string) (int64, error) {
	// For agent files (agent-XXXXX.jsonl), use filename as unique identifier
	// because agent subprocesses share their parent's sessionId
	baseName := filepath.Base(filePath)
	if strings.HasPrefix(baseName, "agent-") && strings.HasSuffix(baseName, ".jsonl") {
		// Extract agent ID from filename: agent-e3672e13.jsonl -> e3672e13
		agentID := strings.TrimSuffix(strings.TrimPrefix(baseName, "agent-"), ".jsonl")
		if agentID != "" {
			// Use parent sessionId + agent ID for uniqueness
			if externalID != "" {
				externalID = externalID + "-agent-" + agentID
			} else {
				externalID = "agent-" + agentID
			}
		}
	}

	if externalID == "" {
		// Use file path as fallback session ID
		externalID = filePath
	}

	// Check cache first
	cacheKey := externalID
	if id, ok := e.sessionCache[cacheKey]; ok {
		return id, nil
	}

	// Extract project from file path
	project := extractProjectFromPath(filePath)

	// Create or get session in database
	sessionID, isNew, err := e.store.UpsertClaudeSession(ctx, externalID, filePath, "")
	if err != nil {
		return 0, err
	}

	// Update cache
	e.sessionCache[cacheKey] = sessionID

	if isNew {
		e.stats.sessionsCreated++
	}

	// Ensure project is set - ignore this for now as project comes from path
	_ = project

	return sessionID, nil
}

// hashLine computes a hash of a line for deduplication
func hashLine(line string) string {
	h := sha256.Sum256([]byte(line))
	return hex.EncodeToString(h[:8]) // Use first 8 bytes for brevity
}

// extractSessionIDFromLine extracts session ID from a JSONL line
func extractSessionIDFromLine(line string) string {
	// Quick extraction without full JSON parse
	// Look for "sessionId":"<value>" pattern
	patterns := []string{`"sessionId":"`, `"session_id":"`, `"id":"`}
	for _, pattern := range patterns {
		if idx := findPattern(line, pattern); idx >= 0 {
			start := idx + len(pattern)
			end := start
			for end < len(line) && line[end] != '"' {
				end++
			}
			if end > start {
				return line[start:end]
			}
		}
	}
	return ""
}

// findPattern finds pattern in string, returns start index or -1
func findPattern(s, pattern string) int {
	for i := 0; i <= len(s)-len(pattern); i++ {
		if s[i:i+len(pattern)] == pattern {
			return i
		}
	}
	return -1
}

// mapToJSON converts a map to JSON string
func mapToJSON(m map[string]interface{}) string {
	if m == nil {
		return "{}"
	}
	// Simple conversion - for complex cases use encoding/json
	result := "{"
	first := true
	for k, v := range m {
		if !first {
			result += ","
		}
		result += fmt.Sprintf(`"%s":"%v"`, k, v)
		first = false
	}
	result += "}"
	return result
}
