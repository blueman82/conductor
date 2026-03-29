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

## Validation Before Building

Run 5-10 complex plans, track cases where a task fails or produces suboptimal output because it lacked information that a sibling task had. If that pattern is common, the blackboard is justified. If failures are mostly about agent capability or task scoping, the communication layer isn't the bottleneck.

## Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| Agent acts on rolled-back observation | Only publish after GREEN verdict |
| Prompt bloat from too many observations | Cap at N observations, prioritize by relevance |
| Ordering dependency on shared state | Empty blackboard = today's behavior (safe default) |
| Debugging complexity | Log all blackboard reads/writes with task attribution |
