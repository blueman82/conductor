# Quick Reference: Task Field Usage in Conductor Executor

## Critical Fields (Execution Must-Haves)

```
Number      → Task identifier, enables plan updates
Name        → Appears in QC prompts, logging
Prompt      → Sent to Claude agent (MODIFIED by integration context!)
Agent       → Selects executing Claude agent (or defaults to config)
```

## Integration Context Injection (Pre-Execution)

**When**: `DependsOn` is not empty OR `Type == "integration"`
**What happens**:
1. Files listed in DependsOn tasks injected into Prompt
2. Explains why those files must be read
3. Happens BEFORE agent invocation (line 517-519 of task.go)

```go
if te.Plan != nil && (task.Type == "integration" || len(task.DependsOn) > 0) {
    task.Prompt = buildIntegrationPrompt(task, te.Plan)  // Modifies prompt
}
```

## Structured QC vs. Legacy QC

**Structured QC triggers on**: `SuccessCriteria` is not empty

```
✗ Legacy: Single verdict (GREEN/RED/YELLOW) on entire output
✓ Structured: Per-criterion verdicts with unanimous consensus
  - Any criterion FAILED = RED verdict
  - ALL criteria PASSED = GREEN verdict
```

**Criteria merging**:
- success_criteria + integration_criteria (if Type="integration")
- Formatted as numbered checklist in QC prompt (0, 1, 2, ...)
- QC returns `criteria_results` array with per-criterion verdicts

## Domain-Specific QC Injection

**Based on file extensions** (auto-injected):
- `.go` → Go-specific checks (error handling, nil pointers, context)
- `.sql` → SQL checks (injection, indexes, FK constraints)
- `.ts/.tsx` → TypeScript/React checks (no 'any', hooks rules, async)
- `.py` → Python checks (type hints, specific exceptions, context managers)

## Multi-File Plan Coordination

**SourceFile field**:
- Set by parser for multi-file plans
- Determines which file to lock and update
- File-level locking: Only one task updates a file at a time
- Priority: `task.SourceFile` > `executor.SourceFile` > `config.PlanPath`

## Inter-Retry Agent Swapping

**When RED verdict received** (if swap enabled):
```
1. Extract execution history for this task
2. Call SelectBetterAgent(currentAgent, history, qcSuggestedAgent)
3. If returns different agent: task.Agent = newAgent
4. Retry with new agent
```

**Conditions**:
- SwapDuringRetries must be enabled (default: true)
- Must reach MinFailuresBeforeAdapt threshold (default: 2)
- QC may suggest agent via `SuggestedAgent` field

## Learning System Integration

**Three hooks**:

1. **preTaskHook()** (before invocation):
   - Analyze past failures for this task
   - Adapt Agent if pattern suggests better choice
   - Enhance Prompt with failure context

2. **qcReviewHook()** (after QC verdict):
   - Extract failure patterns from RED verdicts
   - Store in task.Metadata["failure_patterns"]
   - Track QC verdict for consensus

3. **postTaskHook()** (after completion):
   - Record entire execution to learning database
   - Stores: Agent, Verdict, Duration, Output, Errors, Patterns

## Field Access Timeline

```
┌──────────────────────────────────────┐
│ 1. Parse Task from Plan File         │
│    → Extract all fields              │
└──────────────────────────────────────┘
                ↓
┌──────────────────────────────────────┐
│ 2. Pre-Task Hook                     │
│    → Adapt Agent from learning       │
│    → Enhance Prompt from history     │
└──────────────────────────────────────┘
                ↓
┌──────────────────────────────────────┐
│ 3. Integration Context Injection     │
│    → If DependsOn or Type="integration"
│    → Modify Prompt with file context │
└──────────────────────────────────────┘
                ↓
┌──────────────────────────────────────┐
│ 4. Apply Default Agent               │
│    → If Agent empty & config has default
└──────────────────────────────────────┘
                ↓
┌──────────────────────────────────────┐
│ 5. Mark as In-Progress               │
│    → Update Status in plan file      │
└──────────────────────────────────────┘
                ↓
┌──────────────────────────────────────┐
│ 6. Invoke Agent                      │
│    → Pass: Number, Name, Prompt, Agent
└──────────────────────────────────────┘
                ↓
┌──────────────────────────────────────┐
│ 7. Quality Control Review            │
│    → If SuccessCriteria: Structured  │
│    → Else: Legacy blob review        │
│    → If Type="integration": include  │
│      IntegrationCriteria in verdict  │
└──────────────────────────────────────┘
                ↓
┌──────────────────────────────────────┐
│ 8. Store Feedback & Patterns         │
│    → Update plan file with verdict   │
│    → Extract failure patterns        │
│    → Store in task.Metadata          │
└──────────────────────────────────────┘
                ↓
┌──────────────────────────────────────┐
│ 9. QC Review Hook                    │
│    → Extract failure patterns        │
│    → Store in Metadata               │
└──────────────────────────────────────┘
                ↓
   ┌─────────────────────┐
   │ GREEN or YELLOW?    │
   │ YES → Complete      │
   └─────────────────────┘
           ↓
   ┌─────────────────────┐
   │ RED & Retries Left? │
   │ YES → Agent Swap?   │
   │       Retry         │
   │ NO → Fail           │
   └─────────────────────┘
                ↓
┌──────────────────────────────────────┐
│ 10. Post-Task Hook                   │
│     → Record execution to database   │
│     → Include all metadata           │
└──────────────────────────────────────┘
                ↓
┌──────────────────────────────────────┐
│ 11. Update Plan Status               │
│     → Set Status = completed/failed  │
│     → Set CompletedAt timestamp      │
└──────────────────────────────────────┘
```

## Unified Field Behaviors

### Behaviors Triggered by Field Presence

| Behavior | Trigger | Field |
|----------|---------|-------|
| Integration context injection | Non-empty | DependsOn |
| Structured QC mode | Non-empty | SuccessCriteria |
| Integration task validation | equals | Type="integration" |
| Domain-specific QC criteria | File extensions | Files |
| Per-file locking | Non-empty | SourceFile |
| Agent adaptation | Learning match | Agent + learning history |
| Integration criteria in verdict | equals | Type="integration" |

### Fields Modified During Execution

| Field | When | Who | Why |
|-------|------|-----|-----|
| Prompt | Before invocation | buildIntegrationPrompt() | Add dependency context |
| Prompt | Before invocation | enhancePromptWithLearning() | Add failure context |
| Agent | Pre-invocation | preTaskHook() | Learning adaptation |
| Agent | After RED verdict | SelectBetterAgent() | Inter-retry swap |
| Metadata | After QC review | qcReviewHook() | Store verdict, patterns |
| Status | Execution flow | updatePlanStatus() | Track completion |

## QC Verdict Rules

### Single Agent QC
```
If SuccessCriteria present:
  → Structured prompt with criteria
  → Any failed criterion = RED verdict
  → (Unanimous consensus with self = criterion verdict)
Else:
  → Legacy blob review
  → Agent verdict is final
```

### Multi-Agent QC (v2.2+)
```
If SuccessCriteria present:
  → Unanimous consensus required
  → Criterion passes ONLY if ALL agents agree PASS
  → Even one agent RED + criteria fail = RED verdict
Else:
  → Strictest-wins: RED > YELLOW > GREEN
```

### Critical Rule (v2.4)
```
Even if all criteria PASS:
  → If ANY agent returned RED verdict
  → Final verdict = RED
  → (Agents may catch issues outside explicit criteria)
```

## Fields NOT Used During Execution

These are parsed and stored but don't affect executor behavior:

- `EstimatedTime` - Metadata only, never consulted
- `StartedAt` - Set but not read (redundant with ExecutionStartTime)
- `WorktreeGroup` - Organizational metadata, preserved but not used
- `ExecutionStartTime`, `ExecutionEndTime`, `ExecutionDuration` - Display only
- `ExecutedBy` - Display only
- `FilesModified`, `FilesCreated`, `FilesDeleted` - Display only

## Prompt Modification Chain

```
Original Prompt (from parser)
           ↓
[Maybe] enhancePromptWithLearning()
           ↓
[Maybe] buildIntegrationPrompt()
        (if DependsOn or Type="integration")
           ↓
[Final Prompt sent to Agent]
```

**Key**: Both modifications happen BEFORE agent invocation in Execute() method.

## Success Criteria Examples

### Component Task (no integration)
```yaml
success_criteria:
  - "JWT validation function implemented"
  - "Supports HS256 algorithm"
  - "Unit tests achieve 90% coverage"
```

QC verifies each criterion independently → ALL pass = GREEN.

### Integration Task (wiring components)
```yaml
type: integration
success_criteria:
  - "Router accepts GET /api/users request"
  - "Response returns user data as JSON"
integration_criteria:
  - "Auth middleware executes before route handlers"
  - "Unauthenticated requests receive 401"
  - "Database connection from Task 2 is used"
```

QC verifies BOTH types → ALL pass = GREEN.

## File-Level Locking in Multi-File Plans

```
Task 1 (from file1.md)         Task 2 (from file1.md)       Task 3 (from file2.md)
     ↓                               ↓                             ↓
Lock(file1.md) ──────────────→ Wait ──────────────→ Lock(file2.md)
     ↓                                                    ↓
Update file1.md           (file1.md locked)         Update file2.md
Unlock ────────────────────→ Lock(file1.md) ◄────────────────────
                                ↓
                           Update file1.md
                           Unlock
```

Results:
- Serial updates to same file (prevents corruption)
- Parallel updates to different files (faster)

---

## Common Patterns

### Check if structured QC should be used:
```go
allCriteria := getCombinedCriteria(task)  // Success + Integration
if len(allCriteria) > 0 {
    // Use BuildStructuredReviewPrompt
} else {
    // Use BuildReviewPrompt (legacy)
}
```

### Check if integration context should be injected:
```go
if te.Plan != nil && (task.Type == "integration" || len(task.DependsOn) > 0) {
    task.Prompt = buildIntegrationPrompt(task, te.Plan)
}
```

### Check if task can be skipped:
```go
if task.CanSkip() {  // Status == "completed" or "skipped"
    // Don't execute, create synthetic GREEN result
}
```

### Extract failure patterns:
```go
patterns := extractFailurePatterns(verdict, feedback, output)
// Only returns patterns for RED verdicts
// Empty array for GREEN/YELLOW
```
