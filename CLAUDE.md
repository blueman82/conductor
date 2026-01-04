<coding_guidelines>
# CLAUDE.md
Last updated: 2026-01-03


## Project Overview

Conductor is a Go CLI for autonomous multi-agent orchestration. It parses plan files (Markdown/YAML), calculates task dependencies via graph algorithms, and executes tasks in parallel waves with QC reviews and adaptive learning.

**Current**: v3.2.0 - Git Rollback: Task-level checkpoints with automatic rollback on QC failure. Includes BranchGuard for plan-level branch isolation and CheckpointCleanupHook for stale branch cleanup. See v3.1.0 for SetupIntrospector.

## Quick Reference

**Note:** `conductor` is in PATH. Use `conductor` directly, not `./conductor`.

```bash
# Build & Run
go build ./cmd/conductor && conductor run plan.md
make build-patch  # Auto-increment version

# Testing
go test ./... -cover
go test ./internal/pattern/... -v

# Version
cat VERSION
```

## Architecture

```
Plan File(s) → Parser → Graph Builder → Orchestrator → Wave Executor → Task Executor → QC → Learning
```

### Key Packages
| Package | Purpose |
|---------|---------|
| `internal/parser/` | Markdown/YAML plan parsing |
| `internal/executor/` | Wave/task execution, QC, pattern hooks |
| `internal/agent/` | Claude CLI invocation, agent discovery |
| `internal/learning/` | Execution history, SQLite storage |
| `internal/pattern/` | STOP protocol, duplicate detection, LLM enhancement |
| `internal/budget/` | Rate limit handling, state persistence |
| `internal/tts/` | Optional voice feedback |
| `internal/config/` | Configuration loading |

### Claude CLI Invocation Pattern
```go
args := []string{
    "-p", prompt,
    "--json-schema", schema,          // Enforces response structure
    "--output-format", "json",
    "--settings", `{"disableAllHooks": true}`,
}
cmd := exec.CommandContext(ctx, "claude", args...)
```

## Configuration (.conductor/config.yaml)

### Core Settings
```yaml
max_concurrency: 4
skip_completed: true
retry_failed: true

quality_control:
  enabled: true
  agents:
    mode: intelligent  # auto | explicit | mixed | intelligent
    max_agents: 4
  retry_on_red: 2

learning:
  enabled: true
  swap_during_retries: true
  warmup_enabled: true
  keep_executions_days: 90
```

### Pattern Intelligence
```yaml
pattern:
  enabled: true
  mode: warn                    # block | warn | suggest
  similarity_threshold: 0.8
  duplicate_threshold: 0.9
  enable_stop: true             # STOP protocol (Search/Think/Outline/Prove)
  require_justification: true   # QC asks for justification when prior art found
  llm_enhancement_enabled: true # Claude refines similarity via ClaudeSimilarity
```

### Budget & Rate Limits
```yaml
budget:
  enabled: true
  auto_resume: true
  max_wait_duration: 6h
  announce_interval: 15m
```

### TTS (Optional)
```yaml
tts:
  enabled: true
  base_url: "http://localhost:5005"
  voice: "tara"
```

### Setup (Optional)
```yaml
setup:
  enabled: true  # Enable pre-wave setup phase (default: false)
```
When enabled, Claude introspects the project before wave execution to discover and run setup commands (e.g., dependency installation, build prerequisites).

### Rollback (v3.2+)
```yaml
rollback:
  enabled: true               # Enable task-level git checkpoints
  checkpoint_prefix: "conductor-checkpoint-"  # Branch name prefix
  keep_checkpoint_days: 7     # Auto-cleanup stale branches on startup
```
Creates checkpoint branches before each task. On RED verdict after exhausting retries, rolls back to checkpoint. BranchGuard provides plan-level branch isolation.

### Timeouts (v2.33+)
```yaml
timeouts:
  task: 12h    # Main agent task execution
  llm: 90s     # Claude CLI calls (enhancer, assessor, similarity, selectors)
  http: 30s    # External HTTP (TTS, webhooks)
  search: 30s  # Local CLI operations (git, grep)
```

### Mandatory Commit Verification (v2.30+)
Tasks can specify required commits. Agents are instructed to commit, conductor verifies:
```yaml
# In plan YAML task:
commit:
  type: "feat"
  message: "add user authentication"
  files:
    - "internal/auth/*.go"
```
- Agent receives MANDATORY COMMIT instructions in prompt
- After agent completion, `git log --grep` verifies commit exists
- QC is informed of commit status (missing = RED factor)
- Results persisted to plan `execution_history` and learning DB

## QC Response Schema
```go
type QCResponse struct {
    Verdict         string   `json:"verdict"`          // GREEN, RED, YELLOW
    Feedback        string   `json:"feedback"`
    ShouldRetry     bool     `json:"should_retry"`
    SuggestedAgent  string   `json:"suggested_agent"`
    CriteriaResults []CriterionResult `json:"criteria_results"`
}
```

## Task Dependencies

**Kahn's algorithm** for topological sort. Cycle detection via DFS color marking.

```yaml
# Cross-file dependencies
depends_on:
  - 1                           # Local task
  - file: foundation.yaml       # Cross-file
    task: 3
```

## RFC 2119 Mapping
| Level | Field | Behavior |
|-------|-------|----------|
| MUST | `test_commands` | Hard gate - blocks task |
| MUST | `dependency_checks` | Preflight - blocks start |
| SHOULD | `success_criteria` | QC reviews |
| MAY | `documentation_targets` | Informational |

## Integration Tasks
```yaml
- id: 4
  type: integration
  success_criteria: [...]       # Component-level
  integration_criteria: [...]   # Cross-component flow
```

## Agent Watch Commands
```bash
conductor observe stats         # Summary statistics
conductor observe stream        # Real-time activity
conductor observe ingest --watch # Daemon mode
conductor observe export --format json
```

## Key Files
- `internal/executor/task.go` - Task execution pipeline
- `internal/executor/qc.go` - Quality control, STOP justification
- `internal/executor/commit_verifier.go` - Git commit verification
- `internal/executor/setup_introspector.go` - Pre-wave setup phase
- `internal/pattern/intelligence.go` - Pattern orchestrator
- `internal/pattern/claude_enhancement.go` - LLM confidence refinement
- `internal/budget/waiter.go` - Rate limit waiting with TTS

## Test-Driven Development

**Strict TDD**: Red → Green → Refactor → Commit

Coverage targets: Critical paths 90%+, overall 70%+

## Dependencies
- `github.com/spf13/cobra` - CLI
- `github.com/yuin/goldmark` - Markdown
- `gopkg.in/yaml.v3` - YAML

No networking, no ML frameworks. Local CLI orchestration only.
</coding_guidelines>
