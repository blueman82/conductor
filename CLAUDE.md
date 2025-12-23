# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Conductor is an autonomous multi-agent orchestration CLI built in Go that executes implementation plans by spawning and managing multiple Claude Code CLI agents in coordinated waves. It parses plan files (Markdown or YAML), calculates task dependencies using graph algorithms, and orchestrates parallel execution with quality control reviews and adaptive learning.

**Current Status**: Production-ready v2.17.0 with comprehensive multi-agent orchestration, multi-file plan support with cross-file dependencies (v2.6+), quality control reviews, adaptive learning system, inter-retry agent swapping, structured success criteria with per-criterion verification, intelligent QC agent selection, domain-specific review criteria, integration tasks with dual criteria validation, GUARD Protocol for pre-wave failure prediction (v2.17+), optional TTS voice feedback (v2.14+), and auto-incrementing version management.

## Codebase Search

**Use docvec first** for semantic search, then fallback to Glob/Grep:
```
mcp__docvec__search("how does QC validation work")  # Semantic search
mcp__docvec__search_with_filters("parser", {"source_file": "internal/parser"})
```
Codebase is indexed (145 files, 5106 chunks). Docvec finds conceptually related code even without exact keyword matches.

## Development Commands

### Build & Run
```bash
go build ./cmd/conductor          # Build binary
go run ./cmd/conductor            # Run from source
./conductor --version             # Show version
./conductor --help                # Show help
./conductor learning stats        # View learning statistics
./conductor learning export       # Export learning data
./conductor learning clear        # Clear learning history
```

### Testing
```bash
go test ./...                              # Run all tests
go test ./... -cover                       # Run with coverage
go test ./internal/parser/ -v              # Test specific package
go test ./internal/parser/ -run TestMarkdownParser -v  # Run specific test
go test -race ./...                        # Run with race detection
go test ./... -coverprofile=coverage.out   # Generate coverage report
go tool cover -html=coverage.out           # View coverage HTML
```

### Version Management
```bash
cat VERSION              # Current version (stored in VERSION file)
make build               # Build with current version
make build-patch         # Auto-increment patch (1.1.0 → 1.1.1) and build
make build-minor         # Auto-increment minor (1.1.0 → 1.2.0) and build
make build-major         # Auto-increment major (1.1.0 → 2.0.0) and build
```

**Versioning**: Conductor uses semantic versioning with auto-increment targets. The VERSION file is the single source of truth, automatically injected into the binary at build time via LDFLAGS.

## Architecture Overview

### Execution Pipeline

```
Plan File(s) → Parser → Graph Builder → Orchestrator → Wave Executor → GUARD Gate → Task Executor → QC → Learning System
```

**Multi-File Plans** (v2.6+): Parser handles multiple files → Plan Merger validates cross-file deps → Rest of pipeline

### Key Components

**Parser (`internal/parser/`)**: Auto-detects format (.md/.yaml), parses into Task structs. Markdown extracts from `## Task N:` headings, YAML handles structured format. Both support YAML frontmatter for config.

**Plan Merger**: Loads/merges multiple plan files, validates cross-file dependencies, tracks file-to-task mapping for resume operations.

**Models (`internal/models/`)**: Task, Plan, Wave, TaskResult. Task has Status/CompletedAt for resumable execution, DependsOn for dependency graph. Plan has WorktreeGroups (organizational only) and FileToTaskMap.

**Executor (`internal/executor/`)**:
- `graph.go`: Kahn's algorithm for wave calculation, DFS cycle detection
- `qc.go`: Quality control with JSON parsing, GREEN/RED/YELLOW verdicts, retry logic (max 2)
- `task.go`: Invoke → review → retry pipeline, inter-retry agent swapping, single updateFeedback() call after QC
- `wave.go`: Sequential wave execution, bounded concurrency, filters completed tasks

**Agent (`internal/agent/`)**:
- `discovery.go`: Scans `~/.claude/agents/`, directory whitelisting, file filtering
- `invoker.go`: Executes `claude -p` with JSON output, disables hooks, prefixes prompts with agent name

**Learning (`internal/learning/`)**: Stores execution history (store.go), analyzes patterns (analyzer.go), integrates via hooks (hooks.go). Data in `.conductor/learning/`.

**CLI (`internal/cmd/`)**: root.go, run.go, validate.go, learning.go. Flags: --dry-run, --max-concurrency, --timeout, --verbose, --skip-completed, --retry-failed, --log-dir.

**Config (`internal/config/`)**: .conductor/config.yaml. Priority: defaults → config → CLI flags. Fields: max_concurrency, timeout, dry_run, skip_completed, retry_failed, log_dir, log_level, learning settings.

### Dependency Graph Algorithm

**Kahn's algorithm** for topological sort:
1. Calculate in-degree (number of dependencies)
2. Wave 1: tasks with in-degree 0
3. Process wave, decrease dependent in-degrees
4. Repeat until all processed

**Cycle detection**: DFS with color marking (White=0, Gray=1, Black=2). Back edge = cycle.

### Claude CLI Invocation

```go
// v2.8+: JSON schema enforcement guarantees response structure
args := []string{
    "--agents", agentJSON,                    // Agent definition
    "--json-schema", models.AgentResponseSchema(), // Enforces response structure
    "-p", prompt,
    "--permission-mode", "bypassPermissions",
    "--settings", `{"disableAllHooks": true}`,
    "--output-format", "json",                // CLI wrapper format
}
cmd := exec.CommandContext(ctx, "claude", args...)
```

**Schema Enforcement (v2.8+)**: `--json-schema` flag guarantees Claude's response matches expected structure at generation time, eliminating parse failures and retry logic.

### Quality Control

**JSON Response Schema:**
```go
type QCResponse struct {
    Verdict         string   `json:"verdict"`          // GREEN, RED, YELLOW
    Feedback        string   `json:"feedback"`
    Issues          []Issue  `json:"issues"`
    Recommendations []string `json:"recommendations"`
    ShouldRetry     bool     `json:"should_retry"`
    SuggestedAgent  string   `json:"suggested_agent"`  // For inter-retry swapping
    CriteriaResults []CriterionResult `json:"criteria_results"`  // Per-criterion (v2.3+)
    IntegrationResults []CriterionResult `json:"integration_results"`  // Integration (v2.5+)
}
```

**Parsing (v2.8+)**: `parseQCJSON()` extracts content from Claude CLI envelope. Schema enforcement guarantees valid JSON - no fence stripping or retry needed.

**Inter-Retry Agent Swapping (v2.1)**: On RED with `SuggestedAgent`, `learning.SelectBetterAgent()` validates and swaps for retry. Controlled by `learning.swap_during_retries` config.

**Structured Criteria (v2.3+)**: Tasks define `success_criteria` list. QC returns per-criterion PASS/FAIL. Multi-agent unanimous consensus required.

**Intelligent Selection (v2.4+)**: Claude-based QC agent recommendations analyzing task context + executing agent. Modes: auto, explicit, mixed, intelligent. Guardrails: code-reviewer baseline, max agents cap, registry validation, TTL caching.

### RFC 2119 Requirement Levels

Conductor's existing structure maps to [RFC 2119](https://datatracker.ietf.org/doc/html/rfc2119) requirement levels:

| RFC 2119 Level | Plan Field | Behavior | QC Verdict on Failure |
|----------------|------------|----------|----------------------|
| **MUST** | `test_commands` | Hard gate - blocks task | RED (task fails) |
| **MUST** | `dependency_checks` | Preflight gate - blocks start | Task blocked |
| **SHOULD** | `success_criteria` | Soft signal - QC reviews | RED or YELLOW |
| **SHOULD** | `integration_criteria` | Soft signal - QC reviews | RED or YELLOW |
| **MAY** | `documentation_targets` | Soft signal - informational | YELLOW max |

**Usage in Plans:**
- Put absolute requirements in `test_commands` (must pass or task fails)
- Put reviewable criteria in `success_criteria` (QC evaluates)
- Put optional enhancements in `documentation_targets` (checked but non-blocking)

**Embedding Levels in Criteria Text:**
```yaml
success_criteria:
  - "MUST: Function validates all input before processing"
  - "SHOULD: Error messages include file paths for debugging"
  - "MAY: Supports custom exclusion patterns via config"
```

QC agents interpret RFC 2119 keywords in criterion text to determine severity.

### Adaptive Learning

**Flow**: Pre-Task Hook (load history) → Task Execution → QC Review → Inter-Retry Swap → Dual Storage (plan file + DB) → Post-Task Hook (record outcome)

**Storage**: SQLite `.conductor/learning/executions.db`. Tracks: task_name, timestamp, outcome, duration, files_modified, error_message, retry_count.

**Config** (.conductor/config.yaml):
```yaml
learning:
  enabled: true                    # Master switch
  auto_adapt_agent: false         # Auto-switch on failures
  swap_during_retries: true       # v2.1: within-run swapping
  enhance_prompts: true           # Add context to prompts
  qc_reads_plan_context: true     # Load from plan file
  qc_reads_db_context: true       # Load from database
  max_context_entries: 10         # Limit context
  min_failures_before_adapt: 2    # Threshold
```

### GUARD Protocol (v2.17+)

**Pre-wave failure prediction** using behavioral analytics to gate task execution.

**Modes:**
- `block`: Hard gate - fails tasks with high failure probability
- `warn`: Soft signal - logs warning but allows execution
- `adaptive`: Smart gate - blocks only when both probability AND confidence exceed thresholds

**Config** (.conductor/config.yaml):
```yaml
guard:
  enabled: true                    # Master switch
  mode: warn                       # block | warn | adaptive
  probability_threshold: 0.7       # Minimum failure probability to trigger
  confidence_threshold: 0.7        # For adaptive mode
  min_history_sessions: 5          # Minimum data points needed
```

**CLI Override:** `--no-guard` disables GUARD at runtime regardless of config.

**Graceful Degradation:** GUARD errors never block execution - failures are logged and tasks proceed.

**Key Files:**
- `guard.go`: Core GuardProtocol, CheckWave(), evaluateBlockDecision()
- `wave.go`: Gate insertion after task filtering, before semaphore
- `guard_test.go`: 19 unit tests covering modes and thresholds

### Budget & Rate Limits (v2.20+)

**Intelligent rate limit auto-resume** with state persistence for long-running plans.

**Features:**
- Parses actual reset time from Claude CLI output (not hardcoded delays)
- Waits with countdown announcements when within max wait duration
- Saves execution state and exits cleanly for long waits (>6h)
- Resume paused executions via CLI commands

**Config** (.conductor/config.yaml):
```yaml
budget:
  enabled: true
  auto_resume: true           # Enable intelligent wait/exit
  max_wait_duration: 6h       # Save-and-exit if wait exceeds this
  announce_interval: 15m      # Countdown announcement frequency
  safety_buffer: 60s          # Extra wait after reset
```

**CLI Commands:**
```bash
conductor budget list-paused    # Show paused executions
conductor budget resume         # Resume ready executions
conductor budget resume <id>    # Resume specific session
```

**Key Files:**
- `internal/budget/ratelimit.go`: Parse reset time from CLI patterns
- `internal/budget/waiter.go`: Smart wait with countdown
- `internal/budget/state.go`: Execution state persistence
- `internal/executor/task.go`: Rate limit recovery wrapper

## Test-Driven Development

**Strict TDD**: Red → Green → Refactor → Commit

**Test Organization**: `foo.go` → `foo_test.go`, fixtures in `testdata/`, table-driven tests with `t.Run()`, interfaces for mocking.

**Coverage Targets**: Critical paths 90%+, overall 70%+ (currently 86.4%).

## Task Metadata Parsing

**Core fields** (always parsed):
- `Number`, `Name`, `Files`, `DependsOn`, `EstimatedTime`, `Agent`, `Status`, `CompletedAt`, `SuccessCriteria`, `TestCommands`

**Multi-file fields**:
- `WorktreeGroup`: Organizational only, NOT for execution control
- `SourceFile`: Internal, tracks file origin for multi-file plans

**Cross-File Dependencies (v2.6+)**:
- Local: Integer task number
- Cross-file: `{file: "path.yaml", task: 5}`

**YAML Format:**
```yaml
depends_on:
  - 1                              # Local
  - file: foundation.yaml          # Cross-file
    task: 3
```

**Markdown Format:**
```markdown
**Depends on**: Task 1, foundation.yaml#3, infrastructure/db.yaml#5
```

## Integration Tasks (v2.5+)

**Component tasks** (`type: "component"`): Single module, `success_criteria` only.

**Integration tasks** (`type: "integration"`): Wire components, BOTH `success_criteria` AND `integration_criteria`.

**Dual Criteria Example:**
```yaml
tasks:
  - id: 4
    type: integration
    success_criteria:          # Component-level
      - "Router accepts requests"
      - "Auth function correct signature"
    integration_criteria:      # Cross-component
      - "Auth middleware executes before handlers"
      - "Database transaction commits atomically"
      - "Error propagates end-to-end"
```

**QC Validation**: All criteria must pass (unanimous consensus). Detailed per-criterion feedback.

**Context Injection**: Tasks with dependencies get integration context explaining what to read and why.

**Best Practices**:
- Use integration tasks for 2+ component wiring
- Define criteria focusing on data/control flow between components
- Read dependency files before implementing
- Verify sequencing constraints ("X before Y")

## Production Status

**v2.20.0**: 86%+ test coverage (500+ tests). Complete pipeline with runtime enforcement, GUARD Protocol, budget tracking, and optional TTS.

**Major Features**:
- Multi-agent orchestration with dependency resolution
- QC reviews with structured JSON responses
- JSON schema enforcement (v2.8+) - guaranteed response structure
- Runtime enforcement (v2.9+) - hard gates and soft signals
- Adaptive learning from execution history
- Multi-file plans with cross-file dependencies (v2.6+)
- Resumable execution with state tracking
- Dual feedback storage (plan files + database)
- Integration tasks with dual criteria
- Agent Watch behavioral analytics (v2.7+)
- GUARD Protocol with behavioral prediction (v2.17+)
- Optional TTS voice feedback (v2.14+)
- Comprehensive logging and error handling

**Recent Enhancements**:
- v2.1: QC JSON parsing, agent swapping, dual feedback
- v2.3: Structured success criteria, per-criterion verification
- v2.4: Intelligent QC selection, domain-specific criteria, RED verdict fix
- v2.5: Integration tasks, dependency context injection
- v2.6+: Explicit cross-file dependencies, enhanced validation
- v2.7: Agent Watch integration, behavioral analytics
- v2.8: JSON schema enforcement via --json-schema flag
- v2.9: Runtime enforcement with test commands, criterion verification, dynamic terminal width
- v2.14: Optional Orpheus TTS voice feedback for hands-free monitoring
- v2.17: GUARD Protocol for pre-wave failure prediction

## Module Path

`github.com/harrison/conductor`

```go
import (
    "github.com/harrison/conductor/internal/parser"
    "github.com/harrison/conductor/internal/executor"
    "github.com/harrison/conductor/internal/agent"
    "github.com/harrison/conductor/internal/models"
    "github.com/harrison/conductor/internal/learning"
)
```

## Dependencies

Minimal external dependencies:
- `github.com/spf13/cobra` - CLI framework
- `github.com/yuin/goldmark` - Markdown parsing
- `gopkg.in/yaml.v3` - YAML parsing

No networking, no ML frameworks, no web frameworks. Just local CLI orchestration.

## Best Practices

### Single-File Plans
- Keep tasks focused and well-scoped
- Use clear dependency declarations
- Test with --dry-run first
- Leverage --skip-completed for resume

### Multi-File Plans
- 1 file per feature/module/service (5-20 tasks per file)
- Clear, descriptive filenames
- Keep intra-file dependencies tight, minimize cross-file
- Use explicit cross-file notation (v2.6+) for clarity
- Validate before execution: `conductor validate *.md`
- Worktree groups for organization (NOT execution control)

## Agent Watch (v2.7+)

### Overview

Agent Watch provides behavioral observability for Claude Code agents. It extracts and analyzes data from session JSONL files in `~/.claude/projects/`.

### Commands

```bash
conductor observe              # Interactive mode with project selection
conductor observe stats        # Display summary statistics
conductor observe stream       # Real-time activity streaming
conductor observe export       # Export to JSON/Markdown/CSV
conductor observe tools        # Tool usage analysis
conductor observe bash         # Bash command patterns
conductor observe files        # File operation analysis
conductor observe errors       # Error pattern analysis
conductor observe project      # Project-level metrics
conductor observe session      # Session-specific analysis
conductor observe ingest       # Real-time JSONL ingestion daemon
conductor observe live         # Real-time watching of new JSONL events
conductor observe transcript   # Formatted transcript of session events
```

### Real-Time Ingestion

```bash
conductor observe ingest                      # One-time import of all files
conductor observe ingest --watch              # Run as daemon, watch for changes
conductor observe ingest --watch --verbose    # Verbose daemon mode
conductor observe ingest --batch-size 100     # Custom batch size
conductor observe ingest --batch-timeout 1s   # Custom flush interval
```

The ingestion daemon:
- Watches `~/.claude/projects/` for JSONL files
- Tracks byte offsets for incremental reads (never re-reads data)
- Uses batched writes for efficient database operations
- Handles graceful shutdown (flushes pending events)
- Uses WAL mode for concurrent read/write access

### Filtering

```bash
--project myapp          # Filter by project name
--session abc123         # Filter by session ID or search
--filter-type tool       # Event type: tool, bash, file
--errors-only            # Show only errors
--time-range 24h         # Time range: 1h, 7d, 30d, today, yesterday
```

### Export Formats

```bash
conductor observe export --format json --output metrics.json
conductor observe export --format markdown --output report.md
conductor observe export --format csv --output data.csv
```

### Analytics Features

**Pattern Detection**: Identifies common tool sequences and bash command patterns.

**Failure Prediction**: Predicts task failure probability based on:
- Tool failure rates
- Similar session outcomes
- High-risk tool combinations

**Performance Scoring**: Multi-dimensional agent scoring:
- Success (40%): Task completion rate
- Cost Efficiency (25%): Token usage vs baseline
- Speed (20%): Duration vs baseline
- Error Recovery (15%): Recovery from errors

**Anomaly Detection**: Statistical outliers using standard deviation thresholds.

**Behavior Clustering**: K-means clustering of similar sessions.

### Key Packages

- `internal/behavioral/` - Core models, filtering, export, analytics
- `internal/cmd/observe*.go` - CLI commands
- `internal/learning/` - Database integration

### Data Flow

```
~/.claude/projects/ → JSONL Parser → SQLite DB → observe commands → Analysis/Export
```

### Documentation

- [Full Documentation](docs/AGENT_WATCH.md)
- [Usage Examples](docs/examples/agent-watch-usage.md)
- [Filtering Guide](docs/examples/agent-watch-filtering.md)
- [Analytics Features](docs/examples/agent-watch-analytics.md)

## Text-to-Speech (TTS) Voice Feedback (v2.14+)

### Overview

Conductor supports optional voice announcements via a local Orpheus TTS server. When enabled, Conductor announces execution events audibly, providing hands-free monitoring of long-running plans.

### What Gets Announced

| Event | Example Announcement |
|-------|---------------------|
| Wave start | "Starting Wave 1 with 3 tasks" |
| Agent deployment | "Deploying agent golang-pro" |
| QC agent selection | "Deploying QC agents code-reviewer and qa-expert" |
| QC selection rationale | Full intelligent selection reasoning |
| Individual QC verdicts | "code-reviewer says GREEN" |
| Aggregated QC result | "QC passed" / "QC failed" / "QC passed with warnings" |
| Wave complete | "Wave 1 completed, all tasks passed" |
| Run complete | "Run completed. All 5 tasks passed" |

### Prerequisites

| Component | Purpose | Installation |
|-----------|---------|--------------|
| **Ollama** | Local LLM runtime | Download from [ollama.ai](https://ollama.ai) |
| **Orpheus Model** | TTS model | `ollama pull legraphista/Orpheus` (~2.4GB) |
| **Orpheus-FastAPI** | OpenAI-compatible API server | Clone from GitHub, run on port 5005 |
| **afplay** (macOS) | Audio playback | Built-in |
| **aplay** (Linux) | Audio playback | Install `alsa-utils` package |

### Server Setup

```bash
# 1. Install Ollama
# Download from https://ollama.ai

# 2. Pull Orpheus model
ollama pull legraphista/Orpheus

# 3. Clone and run Orpheus-FastAPI
git clone https://github.com/Lex-au/Orpheus-FastAPI.git
cd Orpheus-FastAPI
pip install -r requirements.txt
python main.py  # Runs on http://localhost:5005

# 4. Verify server is running
curl http://localhost:5005/
```

### Configuration

Add to `.conductor/config.yaml`:

```yaml
tts:
  enabled: true                        # Enable TTS (default: false)
  base_url: "http://localhost:5005"    # Orpheus server URL
  model: "orpheus"                     # TTS model
  voice: "tara"                        # Voice: tara, leah, jess, leo, dan, mia, zac, zoe
  timeout: 30s                         # Request timeout (Orpheus takes ~2-3s per phrase)
```

### Key Packages

- `internal/tts/client.go` - HTTP client with serialized speech queue
- `internal/tts/announcer.go` - Human-readable message generation
- `internal/tts/logger.go` - executor.Logger implementation for TTS
- `internal/config/config.go` - TTSConfig struct and defaults

### Architecture

```
Execution Event → Logger Interface → TTSLogger → Announcer → Client → HTTP POST → Orpheus
                                                                    → WAV bytes → afplay/aplay
```

**Key Design Decisions:**
- **Fire-and-forget**: TTS errors are silently ignored (never blocks execution)
- **Lazy health check**: Server availability checked once on first announcement
- **Serialized queue**: 100-message buffer prevents overlapping audio
- **Graceful degradation**: Disabled TTS = zero behavior change

### Graceful Degradation

| Condition | Behavior |
|-----------|----------|
| `tts.enabled: false` | No TTS API calls |
| Server unavailable | Silently ignored |
| Queue full (100 msgs) | New announcements dropped |
| Unsupported OS | Silently ignored |
| HTTP timeout | Silent failure, continues |
