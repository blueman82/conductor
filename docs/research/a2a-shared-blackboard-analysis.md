# A2A & Shared Blackboard Research Analysis

**Date**: 2026-03-29
**Branch**: `claude/conductor-research-ZypA3`

## Context

Evaluated whether Conductor should adopt agent-to-agent (A2A) communication to replace or supplement its existing learning/context mechanisms: QC context, warm-up injection, and failure analysis.

## Current Architecture

Agents are **one-shot CLI subprocesses** (`internal/agent/invoker.go:379`):
```go
cmd := exec.CommandContext(ctx, inv.ClaudePath, args...)
```

No agent persists after execution. All coordination is orchestrator-mediated.

### Existing Context Mechanisms

| Mechanism | Location | What It Does |
|-----------|----------|-------------|
| Failure analysis | `task.go:302-315` | Queries SQLite for prior failures, appends patterns to prompt |
| Warm-up injection | `warmup_hook.go:38-76` | Finds similar successful tasks, prepends XML context to prompt |
| QC context | `qc.go:1011-1042, 186-188` | Loads execution history for same task into QC prompt |
| Integration prompts | `integration_prompt.go:24-40` | Tells downstream tasks which files to read from upstream tasks |
| Shared filesystem | All agents share `WorkDir` (`task.go:180`) | Git repo is implicit communication channel |

### The Gap

**No intra-wave knowledge sharing.** Tasks in the same wave execute as isolated goroutines (`wave.go:244-299`). Task 3 and Task 4 running in parallel cannot benefit from each other's discoveries. Cross-run learning exists (SQLite). Cross-wave handoff exists (integration prompts). Same-task retry learning exists (failure analysis). Intra-wave sharing does not exist.

## Approaches Evaluated

### Approach A: Richer Result Schema + QC-as-Mediator

Extend `AgentResponse` (`models/response.go:6-14`) with fields like `ApproachTaken`, `Tradeoffs`, `KnownRisks`, `Confidence`. Have QC synthesize cross-task insights.

**Pros**: Builds on existing code. Low risk. Ships fast.
**Cons**: Only improves sequential wave-to-wave handoff. Doesn't solve intra-wave blindness. Risks prompt bloat — context is already layered from warm-up, failure analysis, integration prompts, and QC history. Adding more meta-context could degrade agent focus.

**Verdict**: Useful but insufficient. Doesn't address the primary gap.

### Approach B: Shared Blackboard (Recommended)

A `sync.RWMutex`-protected `map[string][]Observation` on `WaveExecutor`. Agents read at startup, orchestrator writes after task completion.

**Pros**:
- Fills the actual gap (intra-wave knowledge sharing)
- Compatible with subprocess model — read/write at execution boundaries
- Integration points already exist: `preTaskHook` (read), `postTaskHook` (write)
- `PackageGuard` (`package_guard.go:126`) demonstrates the concurrent shared-state pattern
- Worst case (empty blackboard) is identical to today — no regression

**Cons**:
- Shared mutable state in a system designed around isolation
- Risk of acting on observations from tasks that later get rolled back
- Mitigation: only publish observations after QC passes (GREEN verdict)

**Integration points**:
- **Write**: `postTaskHook` at `task.go:519` — already records to SQLite, add parallel blackboard write
- **Read**: `preTaskHook` at `task.go:290` — already queries learning store, add blackboard read
- **Concurrency model**: Follow `PackageGuard` pattern — mutex-protected store with channel notification

**Observation structure** (minimal):
```go
type Observation struct {
    TaskNumber    string
    FilesModified []string
    Approach      string
    Discoveries   []string
    Verified      bool // true after GREEN QC verdict
}
```

**Verdict**: Best value-to-effort ratio. Fills the real gap without architectural change.

### Approach C: Persistent Agent Sessions

Replace one-shot invocations with long-lived Claude sessions that exchange messages.

**Pros**: Most capable — enables genuine agent collaboration.
**Cons**: Requires rewriting `invoker.go` and `wave.go`. Debugging complexity explodes. Cost model becomes unpredictable. Rollback system (`rollback_hook.go`) wasn't designed for reverting cross-agent advice. Not compatible with current Claude CLI usage.

**Verdict**: Right answer long-term if Conductor moves to Agent SDK. Wrong answer for current architecture.

## Key Files for Implementation

| File | Role | Relevant Lines |
|------|------|---------------|
| `internal/executor/wave.go` | Wave execution, goroutine dispatch | 244-299 (task goroutines), 226 (semaphore) |
| `internal/executor/task.go` | Pre/post hooks, execution pipeline | 290-335 (preTaskHook), 519-549 (postTaskHook) |
| `internal/executor/package_guard.go` | Concurrent shared-state pattern to follow | 126-143 (struct), 283-292 (notification) |
| `internal/agent/invoker.go` | Agent invocation (subprocess model) | 368-460 (Invoke) |
| `internal/executor/integration_prompt.go` | Existing cross-task context pattern | 11-48 |
| `internal/executor/warmup_hook.go` | Existing prompt injection pattern | 38-76, 126-194 |
| `internal/models/response.go` | AgentResponse schema | 6-14 |

## Validation Before Building (Action Required)

**The recommendation for Approach B is based on architectural reasoning, not empirical evidence.** Before implementing, the next session should answer these questions using actual data:

### 1. Query the learning database for intra-wave failure patterns

The SQLite learning store (`internal/learning/store.go`) records every task execution with plan file, task number, QC verdict, failure patterns, and QC feedback. The schema is at `internal/learning/schema.sql`.

**What to look for:**
- Tasks in the same wave that both received RED verdicts — do their failure patterns suggest they were working at cross purposes (e.g., conflicting file modifications)?
- Tasks that failed with patterns like `compilation_error` or `dependency_missing` where a sibling task in the same wave modified the same package — this suggests PackageGuard prevented file conflicts but not logical conflicts
- RED verdicts where QC feedback references work done by another task ("doesn't match the API from task X", "inconsistent with the approach in task Y")

**How to query:**
```sql
-- Find tasks that ran in the same wave and both failed
-- Cross-reference with plan file wave assignments
SELECT te1.task_number, te1.task_name, te1.qc_verdict, te1.qc_feedback,
       te2.task_number, te2.task_name, te2.qc_verdict, te2.qc_feedback
FROM task_executions te1
JOIN task_executions te2 ON te1.plan_file = te2.plan_file
  AND te1.run_number = te2.run_number
  AND te1.task_number != te2.task_number
WHERE te1.qc_verdict = 'RED' AND te2.qc_verdict = 'RED';
```

### 2. Check warm-up injection effectiveness

- How often does warm-up context actually get injected? (confidence >= 0.3 threshold at `warmup_hook.go:59`)
- When it fires, do subsequent attempts succeed at a higher rate than without it?
- Is the 0.3 confidence threshold appropriate or is useful context being filtered out?

### 3. Check failure analysis effectiveness

- How often does `enhancePromptWithLearning` fire? (requires `analysis.FailedAttempts > 0` at `task.go:314`)
- When failure patterns are injected, do retries succeed more often?
- Are the keyword-matched patterns (`task.go:370-386`) capturing the right categories?

### 4. Measure prompt accumulation

- For tasks that receive warm-up + failure analysis + integration context, how large does the injected context get relative to the actual task prompt?
- Is there a correlation between context size and task success/failure?

### What the answers mean

- **If intra-wave conflicts are common** → Blackboard is justified, proceed with implementation
- **If failures are mostly single-task capability issues** → Communication isn't the bottleneck, invest in agent tooling or task decomposition instead
- **If existing learning mechanisms have low effectiveness** → Fix those first before adding another context layer
- **If prompt bloat correlates with failure** → Any new context injection (including blackboard) could make things worse

### Where to find data

- SQLite database location: check `internal/learning/store.go` for the database path (typically `.conductor/learning.db`)
- Plan files with wave assignments: parsed by `internal/parser/` from Markdown/YAML
- Execution logs: check `.conductor/` directory for any log files

## Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| Agent acts on rolled-back observation | Only publish after GREEN verdict |
| Prompt bloat from too many observations | Cap at N observations, prioritize by relevance |
| Ordering dependency on shared state | Empty blackboard = today's behavior (safe default) |
| Debugging complexity | Log all blackboard reads/writes with task attribution |

## Agent SDK Migration Path

The subprocess constraint (`exec.CommandContext` in `invoker.go:379`) is the architectural reason Approach C (persistent sessions) was rejected. If Conductor migrates to the Claude Agent SDK:

- Agents become in-process, long-lived, and capable of receiving messages mid-execution
- The blackboard could evolve into real-time pub/sub between agents
- Approach C becomes viable without a full rewrite
- The blackboard (Approach B) would still be useful as the persistence layer behind live messaging

This analysis did NOT evaluate the Agent SDK migration path in depth. A separate investigation should assess: migration effort, cost model changes (streaming vs one-shot), impact on the existing budget/rate-limit system, and whether the blackboard should be designed with SDK migration in mind.

## Open Questions

1. Is the subprocess model the right long-term choice, or is Agent SDK migration already on the roadmap?
2. What is the actual failure rate for intra-wave tasks, and how much is attributable to isolation?
3. Are the existing learning mechanisms (warm-up, failure analysis) measurably effective?
4. Would a blackboard create a false sense of coordination while masking deeper issues in task decomposition?
