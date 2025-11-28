# Task Field Analysis - Executive Summary

## Overview

This analysis traces how Conductor's executor component uses Task struct fields during execution. Three comprehensive documents provide different levels of detail:

1. **TASK_FIELD_TRACE.md** - Detailed field-by-field analysis with code locations
2. **TASK_FIELD_REFERENCE.md** - Quick reference guide for common patterns
3. **TASK_FIELD_DATAFLOW.md** - Complete data flow diagrams and decision trees

---

## Key Findings

### 1. CRITICAL FIELDS (Execution Must-Haves)

These four fields are required for execution:

```
Number   → Task identification (enables plan updates)
Name     → Human identification (appears in QC prompts)
Prompt   → Implementation details (sent to Claude, MODIFIED before sending)
Agent    → Agent selection (or falls back to DefaultAgent)
```

**Note**: Prompt is the ONLY core field modified during execution. It receives integration context and learning context BEFORE being sent to the agent.

### 2. INTEGRATION CONTEXT INJECTION (Critical Behavior)

**Trigger**: `DependsOn` is not empty OR `Type == "integration"`

**What happens**:
- Integration context is injected into `Prompt` field
- Lists each dependency task's files
- Explains why those files must be read
- Happens BEFORE agent invocation (critical!)
- Works for ANY task with dependencies, not just integration-typed tasks

```go
// In DefaultTaskExecutor.Execute() line 517-519:
if te.Plan != nil && (task.Type == "integration" || len(task.DependsOn) > 0) {
    task.Prompt = buildIntegrationPrompt(task, te.Plan)
}
```

### 3. STRUCTURED QC VS. LEGACY QC (Major Behavior Shift)

**Decision point**: Does `SuccessCriteria` exist?

| Aspect | Structured QC | Legacy QC |
|--------|---|---|
| Trigger | SuccessCriteria not empty | SuccessCriteria empty |
| Criteria | Per-criterion PASS/FAIL | Single verdict |
| Consensus | Unanimous (all agents) | Strictest wins (RED > YELLOW > GREEN) |
| QC response | criteria_results[] array | No array |
| Failure rule | ANY failed criterion = RED | Agent verdict is final |

**Impact**: This fundamentally changes how quality control works.

### 4. INTEGRATION CRITERIA (Dual-Level Validation)

**Trigger**: `Type == "integration"`

When enabled:
- `SuccessCriteria`: Component-level checks
- `IntegrationCriteria`: Cross-component interaction checks
- BOTH must pass for GREEN verdict
- Criteria are merged: `success_criteria + integration_criteria`
- Unanimous consensus: Any agent must agree on each criterion

```go
// In qc.go getCombinedCriteria():
criteria := SuccessCriteria...
if Type == "integration" {
    criteria += IntegrationCriteria...
}
```

### 5. DOMAIN-SPECIFIC QC INJECTION (Auto-Enhancement)

**Based on**: File extensions in `Files` field

```
.go   → Go-specific checks (error handling, nil pointers, context)
.sql  → SQL checks (injection, indexes, FK constraints)
.ts   → TypeScript checks (no 'any' types, async handling)
.tsx  → React checks (hooks rules, prop typing)
.py   → Python checks (type hints, specific exceptions)
```

**How**: Auto-injected into QC prompt via `inferDomainChecks()` (qc.go line 208)

**No configuration needed**: Just add files with the right extensions.

### 6. MULTI-FILE PLAN COORDINATION (Per-File Locking)

**Critical for**: Multi-file plan execution safety

**Field**: `SourceFile` (set by parser for multi-file plans)

**Behavior**:
- Each file gets its own lock
- Only one task updates a file at a time
- Different files can be updated in parallel
- File-level locking via FileLockManager

**Priority order**:
1. `task.SourceFile` (if set)
2. `executor.SourceFile` (legacy)
3. `config.PlanPath` (fallback)

### 7. INTER-RETRY AGENT SWAPPING (Learning Integration)

**Trigger**: RED verdict + conditions met

```go
if verdict == RED
   AND red_count >= MinFailuresBeforeAdapt  // default: 2
   AND SwapDuringRetries == true             // default: true
{
    newAgent = SelectBetterAgent(
        task.Agent,
        history,
        review.SuggestedAgent
    )
    if newAgent != task.Agent {
        task.Agent = newAgent  // FIELD MODIFIED
        // Retry with new agent
    }
}
```

**Impact**: Can improve success rates by switching agents mid-execution

### 8. LEARNING SYSTEM INTEGRATION (Three Hooks)

**preTaskHook()** - Before invocation:
- Analyze failures for this task
- Adapt Agent if pattern suggests better choice
- Enhance Prompt with failure context

**qcReviewHook()** - After QC verdict:
- Extract failure patterns (RED verdicts only)
- Store in `task.Metadata["failure_patterns"]`
- Store QC verdict in metadata

**postTaskHook()** - After completion:
- Record entire execution to learning database
- Stores: Agent (may be swapped), Verdict, Duration, Output, Errors, Patterns

### 9. FIELDS MODIFIED DURING EXECUTION

Only these fields are changed after parsing:

```
Prompt          → Enhanced with learning context + integration context
Agent           → May be adapted by preTaskHook or swapped on RED
Metadata        → Stores QC verdict + failure patterns
Status          → Updated during execution flow
CompletedAt     → Set when task completes
ExecutionStart* → Set for display purposes
ExecutionEnd*   → Set for display purposes
ExecutedBy      → Set to Agent name
```

**All other fields**: Read-only after parsing

### 10. FIELDS NOT USED DURING EXECUTION

These fields are parsed and stored but don't affect executor behavior:

```
EstimatedTime           → Never consulted during execution
StartedAt              → Set but not read (redundant)
WorktreeGroup          → Organizational metadata only
ExecutionStartTime     → Display only
ExecutionEndTime       → Display only
ExecutionDuration      → Display only
FilesModified          → Display/metrics only
FilesCreated           → Display/metrics only
FilesDeleted           → Display/metrics only
```

---

## Field Usage by Executor Stage

### Parsing Stage
- **Markdown**: Extracts basic fields (Number, Name, Files, DependsOn, etc.)
- **YAML**: Extracts all fields including SuccessCriteria, IntegrationCriteria, Type, TestCommands
- **Validation**: YAML only - validates integration task requirements

### Pre-Execution
- **Number**: Task identification
- **Prompt**: Enhanced with learning + integration context
- **Agent**: May be adapted based on history
- **DependsOn**: Triggers integration context injection
- **Type**: Affects integration criteria inclusion

### QC Decision
- **SuccessCriteria**: Switches to structured mode
- **IntegrationCriteria**: Included if Type="integration"
- **TestCommands**: Listed in QC prompt
- **Files**: Triggers domain-specific criteria injection

### Execution
- **Number**: Task identification in logging
- **Agent**: Used for invocation
- **SourceFile**: Determines which file to lock/update
- **Metadata**: Stores QC verdict and patterns

### Post-Execution
- **Number**: Identifies task for plan updates
- **Agent**: Recorded in learning database
- **Metadata**: Retrieved for failure patterns

---

## Critical Design Decisions

### 1. Prompt is Modified BEFORE Invocation
- Integration context injected (if dependencies exist)
- Learning context enhanced (if failures detected)
- Agent sees enriched prompt, not just original spec

### 2. Both SuccessCriteria AND IntegrationCriteria Merged
- Single criteria array passed to QC
- Unanimous consensus on ALL criteria
- No distinction in QC logic (just numbering)

### 3. Agent Can Be Swapped Mid-Execution
- Between retry attempts
- Based on learning history + QC feedback
- Only if conditions met (threshold, flag enabled)

### 4. Per-File Locking in Multi-File Plans
- Prevents file corruption from concurrent updates
- Allows parallel execution across different files
- Critical for large multi-file plans

### 5. Domain-Specific Checks Auto-Injected
- Based on file extensions alone
- No configuration needed
- Applied consistently across all tasks with matching files

---

## Most Important Code Locations

| Concept | File | Lines | Purpose |
|---------|------|-------|---------|
| Task struct | `internal/models/task.go` | 9-38 | Field definitions |
| Integration context | `internal/executor/integration_prompt.go` | 11-46 | Builds enhanced prompt |
| Pre-execution hook | `internal/executor/task.go` | 213-255 | Learning adaptation |
| Integration injection | `internal/executor/task.go` | 517-519 | Modifies prompt |
| QC structured prompt | `internal/executor/qc.go` | 160-227 | Builds criteria prompt |
| Criteria combination | `internal/executor/qc.go` | 149-155 | Merges criteria arrays |
| Domain checks | `internal/executor/qc.go` | 18-66 | Language-specific injection |
| Multi-agent QC | `internal/executor/qc.go` | 433-512 | Parallel review |
| Criteria aggregation | `internal/executor/qc.go` | 826-934 | Unanimous consensus |
| Agent swapping | `internal/executor/task.go` | 721-727 | Inter-retry swap |
| Per-file locking | `internal/executor/task.go` | 490-498 | Multi-file safety |
| Learning hooks | `internal/executor/task.go` | 213-440 | Integration with learning |
| YAML parser | `internal/parser/yaml.go` | 162-174 | Field extraction |
| Field validation | `internal/parser/yaml_validation.go` | - | Integration task checks |

---

## Execution Flow Summary

```
1. PARSE
   └─ Extract all fields from plan file

2. GRAPH BUILDING
   └─ Use DependsOn to build execution waves

3. PRE-EXECUTION
   ├─ preTaskHook(): Adapt Agent, enhance Prompt
   ├─ Integration context injection: Modify Prompt if DependsOn/Type
   └─ Apply DefaultAgent if Agent empty

4. INVOCATION
   ├─ Lock file (SourceFile)
   ├─ Mark status = "in-progress"
   └─ Invoke agent with modified Prompt + Agent selection

5. QC REVIEW
   ├─ Build prompt: SuccessCriteria + IntegrationCriteria (if applicable)
   ├─ Inject domain-specific checks based on Files
   ├─ Invoke QC agent(s) - single or parallel
   └─ Aggregate verdicts: unanimous consensus if criteria exist

6. RETRY DECISION
   ├─ If GREEN/YELLOW: Complete
   ├─ If RED + retries left:
   │  ├─ Check for agent swap opportunity
   │  └─ Retry with possibly different Agent
   └─ If RED + no retries: Fail

7. POST-EXECUTION
   ├─ qcReviewHook(): Extract failure patterns, store metadata
   ├─ postTaskHook(): Record to learning database
   └─ updatePlanStatus(): Mark completed/failed, set CompletedAt

8. PERSISTENCE
   └─ Plan file updated with execution attempt history
```

---

## Recommendations for Future Development

### 1. Leverage EstimatedTime
Currently parsed but unused. Could be used for:
- Timeout calculation per task
- Capacity planning and scheduling
- SLA enforcement

### 2. Activate WorktreeGroup
Currently organizational metadata. Could enable:
- Execution isolation based on groups
- Ordered execution within groups
- Resource allocation by group

### 3. Structured Logging for SourceFile
Multi-file execution could benefit from:
- Per-file execution logs
- Cross-file dependency tracking
- File-level resume capability

### 4. Metadata Extensibility
Task.Metadata already enables custom data. Could formalize:
- Plugin-specific annotations
- Custom execution hints
- Task-specific learning parameters

### 5. Failure Pattern Analysis
extractFailurePatterns() uses keyword matching. Could enhance with:
- ML-based pattern classification
- Historical pattern correlation
- Proactive remediation suggestions

---

## Testing Considerations

### Critical Paths to Test
1. Integration context injection with various dependency structures
2. Structured QC with unanimous consensus logic
3. Agent swapping on RED verdict with learning history
4. Per-file locking with concurrent multi-file execution
5. Domain-specific criteria injection for each file type
6. Metadata propagation through hooks

### Edge Cases
1. Empty DependsOn but Type="integration"
2. SuccessCriteria with multi-agent QC (unanimous consensus)
3. Agent swap when history has no better agent
4. SourceFile missing in multi-file plan
5. EstimatedTime parsing edge cases
6. Concurrent tasks updating same file (locking)

---

## Conclusion

Conductor's Task field usage is highly intentional and tightly integrated:

1. **Core fields** (Number, Name, Prompt, Agent) drive execution
2. **Structural fields** (DependsOn, Type, Files) trigger behavior changes
3. **QC fields** (SuccessCriteria, IntegrationCriteria, TestCommands) enable verification
4. **Coordination fields** (SourceFile, WorktreeGroup) manage multi-file execution
5. **Learning fields** (Agent, Metadata) enable adaptive behavior

The most important insight: **Prompt is the only core field modified during execution**, receiving both integration context and learning context BEFORE being sent to the agent. This design ensures agents have the full context needed for successful task completion.

Key execution patterns:
- Integration context injected (if dependencies exist)
- Structured QC enabled (if criteria exist)
- Domain-specific checks auto-injected (based on files)
- Agent can be swapped (if RED + conditions met)
- Per-file locking (for safe multi-file execution)
- Learning integration (via three hooks)

All field access is documented in the detailed trace documents for reference during development and debugging.
