<coding_guidelines>
# CLAUDE.md

## Project Overview

Conductor is a Go CLI for autonomous multi-agent orchestration. It parses plan files (Markdown/YAML), calculates task dependencies via graph algorithms, and executes tasks in parallel waves with QC reviews and adaptive learning.

**Current**: v2.26.0 - Production-ready with Pattern Intelligence, TTS feedback, budget tracking, and 86%+ test coverage.

## Quick Reference

```bash
# Build & Run
go build ./cmd/conductor && ./conductor run plan.md
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
timeout: 5m
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
  enhance_prompts: true
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
  llm_enhancement_enabled: true # Claude refines uncertain confidence (0.3-0.7)
  llm_timeout_seconds: 30
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
