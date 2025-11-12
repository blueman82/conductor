-- Schema version 1 for adaptive learning system

-- Schema version tracking table
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert initial version (idempotent)
INSERT OR IGNORE INTO schema_version (version) VALUES (1);

-- Task execution history table
-- Stores every task execution attempt with success/failure data
CREATE TABLE IF NOT EXISTS task_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_number TEXT NOT NULL,
    task_name TEXT NOT NULL,
    agent TEXT,
    prompt TEXT NOT NULL,
    success BOOLEAN NOT NULL,
    output TEXT,
    error_message TEXT,
    duration_seconds INTEGER,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    context TEXT -- JSON blob for additional context
);

-- Indexes for fast lookups
CREATE INDEX IF NOT EXISTS idx_task_executions_task ON task_executions(task_number);
CREATE INDEX IF NOT EXISTS idx_task_executions_timestamp ON task_executions(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_task_executions_success ON task_executions(success);

-- Approach history table
-- Tracks different approaches tried for recurring task patterns
CREATE TABLE IF NOT EXISTS approach_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_pattern TEXT NOT NULL, -- Pattern identifier (e.g., "test-fix", "build-error")
    approach_description TEXT NOT NULL,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    last_used TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT -- JSON blob for additional data
);

-- Indexes for approach lookups
CREATE INDEX IF NOT EXISTS idx_approach_history_task ON approach_history(task_pattern);
CREATE INDEX IF NOT EXISTS idx_approach_history_success ON approach_history(success_count DESC);
