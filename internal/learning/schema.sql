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
    plan_file TEXT,
    run_number INTEGER DEFAULT 1,
    task_number TEXT NOT NULL,
    task_name TEXT NOT NULL,
    agent TEXT,
    prompt TEXT NOT NULL,
    success BOOLEAN NOT NULL,
    output TEXT,
    error_message TEXT,
    duration_seconds INTEGER,
    qc_verdict TEXT, -- Quality control verdict: GREEN, RED, YELLOW
    qc_feedback TEXT, -- Detailed feedback from QC review
    failure_patterns TEXT, -- JSON array of identified failure patterns
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    context TEXT -- JSON blob for additional context
);

-- Indexes for fast lookups
CREATE INDEX IF NOT EXISTS idx_task_executions_task ON task_executions(task_number);
CREATE INDEX IF NOT EXISTS idx_task_executions_timestamp ON task_executions(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_task_executions_success ON task_executions(success);
CREATE INDEX IF NOT EXISTS idx_task_executions_plan_file ON task_executions(plan_file);
CREATE INDEX IF NOT EXISTS idx_task_executions_run_number ON task_executions(run_number);

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

-- Behavioral sessions table
-- Tracks high-level session metadata for each task execution
CREATE TABLE IF NOT EXISTS behavioral_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_execution_id INTEGER NOT NULL,
    session_start TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    session_end TIMESTAMP,
    total_duration_seconds INTEGER,
    total_tool_calls INTEGER DEFAULT 0,
    total_bash_commands INTEGER DEFAULT 0,
    total_file_operations INTEGER DEFAULT 0,
    total_tokens_used INTEGER DEFAULT 0,
    context_window_used INTEGER DEFAULT 0,
    FOREIGN KEY (task_execution_id) REFERENCES task_executions(id) ON DELETE CASCADE
);

-- Indexes for session lookups
CREATE INDEX IF NOT EXISTS idx_behavioral_sessions_task_id ON behavioral_sessions(task_execution_id);
CREATE INDEX IF NOT EXISTS idx_behavioral_sessions_start ON behavioral_sessions(session_start DESC);

-- Tool executions table
-- Tracks every tool invocation within a session
CREATE TABLE IF NOT EXISTS tool_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL,
    tool_name TEXT NOT NULL,
    parameters TEXT, -- JSON blob of tool parameters
    execution_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    duration_ms INTEGER,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    FOREIGN KEY (session_id) REFERENCES behavioral_sessions(id) ON DELETE CASCADE
);

-- Indexes for tool execution lookups
CREATE INDEX IF NOT EXISTS idx_tool_executions_session_id ON tool_executions(session_id);
CREATE INDEX IF NOT EXISTS idx_tool_executions_tool_name ON tool_executions(tool_name);
CREATE INDEX IF NOT EXISTS idx_tool_executions_success ON tool_executions(success);

-- Bash commands table
-- Tracks bash command executions and their outcomes
CREATE TABLE IF NOT EXISTS bash_commands (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL,
    command TEXT NOT NULL,
    execution_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    duration_ms INTEGER,
    exit_code INTEGER,
    stdout_length INTEGER, -- Length of stdout in bytes
    stderr_length INTEGER, -- Length of stderr in bytes
    success BOOLEAN NOT NULL,
    FOREIGN KEY (session_id) REFERENCES behavioral_sessions(id) ON DELETE CASCADE
);

-- Indexes for bash command lookups
CREATE INDEX IF NOT EXISTS idx_bash_commands_session_id ON bash_commands(session_id);
CREATE INDEX IF NOT EXISTS idx_bash_commands_success ON bash_commands(success);
CREATE INDEX IF NOT EXISTS idx_bash_commands_exit_code ON bash_commands(exit_code);

-- File operations table
-- Tracks file read/write/edit operations
CREATE TABLE IF NOT EXISTS file_operations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL,
    operation_type TEXT NOT NULL, -- 'read', 'write', 'edit'
    file_path TEXT NOT NULL,
    execution_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    duration_ms INTEGER,
    bytes_affected INTEGER, -- Bytes read/written/modified
    success BOOLEAN NOT NULL,
    error_message TEXT,
    FOREIGN KEY (session_id) REFERENCES behavioral_sessions(id) ON DELETE CASCADE
);

-- Indexes for file operation lookups
CREATE INDEX IF NOT EXISTS idx_file_operations_session_id ON file_operations(session_id);
CREATE INDEX IF NOT EXISTS idx_file_operations_type ON file_operations(operation_type);
CREATE INDEX IF NOT EXISTS idx_file_operations_path ON file_operations(file_path);

-- Token usage table
-- Tracks token consumption per session
CREATE TABLE IF NOT EXISTS token_usage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL,
    measurement_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    input_tokens INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    context_window_size INTEGER,
    FOREIGN KEY (session_id) REFERENCES behavioral_sessions(id) ON DELETE CASCADE
);

-- Indexes for token usage lookups
CREATE INDEX IF NOT EXISTS idx_token_usage_session_id ON token_usage(session_id);
CREATE INDEX IF NOT EXISTS idx_token_usage_time ON token_usage(measurement_time DESC);
