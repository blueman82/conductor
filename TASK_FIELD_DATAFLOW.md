# Task Field Data Flow Diagram

## End-to-End Field Processing Flow

```
╔════════════════════════════════════════════════════════════════════════════╗
║                         TASK FIELD PROCESSING FLOW                         ║
╚════════════════════════════════════════════════════════════════════════════╝

STAGE 1: PARSING & VALIDATION
════════════════════════════════════════════════════════════════════════════

    Plan File (.md/.yaml)
            ↓
    ┌─────────────────────────────────────┐
    │ Auto-detect format                  │
    │ .md → MarkdownParser                │
    │ .yaml → YAMLParser                  │
    └─────────────────────────────────────┘
            ↓
    ┌─────────────────────────────────────┐
    │ YAML Parser extracts:               │
    │ • Number (int/float/string)         │
    │ • Name                              │
    │ • Files[]                           │
    │ • DependsOn[]                       │
    │ • Agent (optional)                  │
    │ • WorktreeGroup (optional)          │
    │ • Status (pending/completed/...)    │
    │ • SuccessCriteria[] ★               │
    │ • IntegrationCriteria[] ★           │
    │ • Type: "integration" ★             │
    │ • TestCommands[] ★                  │
    │ • EstimatedTime                     │
    │ • CompletedAt (timestamp)           │
    └─────────────────────────────────────┘
            ↓
    ┌─────────────────────────────────────┐
    │ Markdown Parser extracts:           │
    │ ✓ Number, Name, Files              │
    │ ✓ DependsOn, Agent, WorktreeGroup  │
    │ ✓ Status, EstimatedTime            │
    │ ✗ SuccessCriteria                  │
    │ ✗ IntegrationCriteria              │
    │ ✗ Type                             │
    │ ✗ TestCommands                     │
    └─────────────────────────────────────┘
            ↓
    ┌─────────────────────────────────────┐
    │ Validation (YAML only)              │
    │ • ValidateTaskType()                │
    │ • ValidateIntegrationTask()         │
    │   (Integration tasks must have      │
    │    integration_criteria)            │
    └─────────────────────────────────────┘
            ↓
    Task object created
    ├─ Number: string
    ├─ Name: string
    ├─ Prompt: string (built from YAML sections)
    ├─ Files: []string
    ├─ DependsOn: []string
    ├─ Agent: string
    ├─ SourceFile: string (for multi-file plans)
    ├─ Status: string
    ├─ SuccessCriteria: []string
    ├─ IntegrationCriteria: []string
    ├─ Type: string
    ├─ TestCommands: []string
    ├─ WorktreeGroup: string
    ├─ Metadata: map[string]interface{}
    └─ (all timestamps/duration fields)


STAGE 2: GRAPH BUILDING & WAVE CALCULATION
════════════════════════════════════════════════════════════════════════════

    Task collection
            ↓
    ┌─────────────────────────────────────┐
    │ Dependency Graph Builder            │
    │ • Uses: Number, DependsOn           │
    │ • Kahn's algorithm (topological)    │
    │ • Cycle detection: DFS              │
    │ • Returns: Waves (parallel groups)  │
    └─────────────────────────────────────┘
            ↓
    Waves created
    Wave 1: [Task with no dependencies]
    Wave 2: [Tasks whose dependencies are done]
    ...


STAGE 3: PRE-EXECUTION HOOKS
════════════════════════════════════════════════════════════════════════════

    For each task in wave:
            ↓
    ┌─────────────────────────────────────┐
    │ preTaskHook()                       │
    │ Input: task                         │
    │ • Query learning store              │
    │   AnalyzeFailures(PlanFile, Number)│
    │ • If failures detected:             │
    │   - Adapt: Agent = betterAgent      │
    │   - Enhance: Prompt += context      │
    └─────────────────────────────────────┘
            ↓
    ┌─────────────────────────────────────┐
    │ Integration Context Injection       │
    │ If: DependsOn not empty OR          │
    │     Type == "integration"           │
    │ Then:                               │
    │   Prompt = buildIntegrationPrompt()│
    │   • For each dependency:            │
    │     - Find Task by DependsOn number │
    │     - Extract: Name, Files[]        │
    │     - Build: "Read these files:"    │
    │     - Append to Prompt              │
    │   • Results in enhanced Prompt      │
    └─────────────────────────────────────┘
            ↓
    ┌─────────────────────────────────────┐
    │ Apply Default Agent                 │
    │ If: Agent is empty AND              │
    │     DefaultAgent configured         │
    │ Then: Agent = DefaultAgent          │
    └─────────────────────────────────────┘


STAGE 4: EXECUTION & INVOCATION
════════════════════════════════════════════════════════════════════════════

    ┌─────────────────────────────────────┐
    │ Per-File Locking                    │
    │ Lock: fileToLock (from SourceFile)  │
    │   (Ensures serial updates to        │
    │    same file in multi-file plans)   │
    └─────────────────────────────────────┘
            ↓
    ┌─────────────────────────────────────┐
    │ Mark Status = "in-progress"         │
    │ Update plan file                    │
    │ Fields used: Number, SourceFile     │
    └─────────────────────────────────────┘
            ↓
    ┌─────────────────────────────────────┐
    │ Invoke Agent                        │
    │ Pass to Claude CLI:                 │
    │   -p <Prompt>                       │
    │   agent: <Agent>                    │
    │ Task.Number used for identification │
    │ Agent gets modified Prompt (context │
    │  injection + learning context)      │
    └─────────────────────────────────────┘
            ↓
    Agent Output received
    └─ ExecutionAttempt record started
       ├─ Agent: <which agent executed>
       ├─ AgentOutput: <raw JSON>
       ├─ Duration: <elapsed time>
       └─ (awaiting QC verdict)


STAGE 5: QUALITY CONTROL REVIEW
════════════════════════════════════════════════════════════════════════════

    ┌─────────────────────────────────────┐
    │ Determine QC Mode                   │
    │ • Check: len(getCombinedCriteria()) │
    │   Criteria = SuccessCriteria[]      │
    │   If Type=="integration":           │
    │     Criteria += IntegrationCriteria[]
    │                                     │
    │ If Criteria present:                │
    │   → Structured QC mode              │
    │ Else:                               │
    │   → Legacy QC mode                  │
    └─────────────────────────────────────┘
            ↓
    ┌─────────────────────────────────────────────────┐
    │ STRUCTURED QC (if criteria present)             │
    ├─────────────────────────────────────────────────┤
    │ BuildStructuredReviewPrompt():                   │
    │ 1. Include task.Name                            │
    │ 2. Include task.Prompt                          │
    │ 3. Include SuccessCriteria as numbered list:    │
    │    0. [ ] Criterion 1                           │
    │    1. [ ] Criterion 2                           │
    │ 4. If Type=="integration":                       │
    │    Include IntegrationCriteria numbered:        │
    │    2. [ ] Integration criterion 1               │
    │    3. [ ] Integration criterion 2               │
    │ 5. Include TestCommands (if present)            │
    │ 6. Include expected Files to verify             │
    │ 7. Inject domain-specific checks from Files     │
    │    .go → Go checks                              │
    │    .sql → SQL checks                            │
    │    .ts/.tsx → TypeScript checks                 │
    │    .py → Python checks                          │
    │ 8. Include historical context (from learning DB)│
    │ 9. JSON format instructions                     │
    │    MUST return criteria_results[]               │
    └─────────────────────────────────────────────────┘
            ↓
    ┌─────────────────────────────────────────────────┐
    │ LEGACY QC (if NO criteria)                      │
    ├─────────────────────────────────────────────────┤
    │ BuildReviewPrompt():                            │
    │ • Simpler format                                │
    │ • No criteria_results array                     │
    │ • Agent verdict is final                        │
    └─────────────────────────────────────────────────┘
            ↓
    ┌─────────────────────────────────────────────────┐
    │ Multi-Agent QC Selection (if mode != empty)    │
    ├─────────────────────────────────────────────────┤
    │ SelectQCAgents():                               │
    │ • mode: "auto" → Select by file extension       │
    │ • mode: "explicit" → Use ExplicitList           │
    │ • mode: "mixed" → auto + AdditionalAgents       │
    │ • mode: "intelligent" → Claude selects agents   │
    │   (considers: task context, executing agent)    │
    │ • All: filtered by BlockedAgents                │
    │ • Result: agents[] list                         │
    └─────────────────────────────────────────────────┘
            ↓
    ┌─────────────────────────────────────────────────┐
    │ Invoke QC Agent(s)                              │
    ├─────────────────────────────────────────────────┤
    │ For each QC agent (single or parallel):         │
    │ • Execute agent with review prompt              │
    │ • Parse JSON response                           │
    │ • Extract: Verdict, CriteriaResults, Feedback   │
    │                                                  │
    │ If multi-agent:                                 │
    │ • Parallel invocation via goroutines            │
    │ • Collect all verdicts                          │
    └─────────────────────────────────────────────────┘
            ↓
    ┌─────────────────────────────────────────────────┐
    │ Aggregate QC Verdicts                           │
    ├─────────────────────────────────────────────────┤
    │ If criteria present:                            │
    │   → aggregateMultiAgentCriteria()               │
    │     Per-criterion unanimous consensus:          │
    │     • Criterion PASS only if ALL agents agree   │
    │     • Criterion FAIL if ANY agent disagrees     │
    │     • Final verdict: all PASS = GREEN           │
    │     • BUT: override to RED if any agent GREEN   │
    │       but said RED (caught issues outside spec) │
    │                                                  │
    │ If no criteria:                                 │
    │   → aggregateVerdicts()                         │
    │     Strictest wins: RED > YELLOW > GREEN        │
    └─────────────────────────────────────────────────┘
            ↓
    ReviewResult created
    ├─ Flag: "GREEN" | "RED" | "YELLOW"
    ├─ Feedback: concatenated from all agents
    ├─ AgentName: single or multi-agent list
    ├─ CriteriaResults: per-criterion verdicts
    └─ SuggestedAgent: from QC feedback


STAGE 6: VERDICT PROCESSING & RETRY LOGIC
════════════════════════════════════════════════════════════════════════════

    ┌─────────────────────────────────────┐
    │ updateFeedback()                    │
    │ Store in plan file:                 │
    │ • Number (task identification)      │
    │ • Attempt number                    │
    │ • Agent (which one executed)        │
    │ • AgentOutput (raw JSON)            │
    │ • QCFeedback (QC agent feedback)    │
    │ • Verdict (GREEN/RED/YELLOW)        │
    │ • Timestamp                         │
    └─────────────────────────────────────┘
            ↓
    ┌─────────────────────────────────────┐
    │ qcReviewHook()                      │
    │ Extract failure patterns (RED only):│
    │ • Keyword matching on feedback      │
    │ • Patterns: compilation, test, etc  │
    │ • Store in Metadata["failure_patterns"]
    │ • Store in Metadata["qc_verdict"]   │
    └─────────────────────────────────────┘
            ↓
    ┌─────────────────────────────────────┐
    │ Verdict Decision                    │
    └─────────────────────────────────────┘
            ↓
       ┌────────────────────────────────┐
       │ GREEN or YELLOW verdict?       │
       └────────────────────────────────┘
                ↓ YES
       ┌────────────────────────────────┐
       │ Task Complete!                 │
       │ Status = "completed"           │
       │ CompletedAt = now()            │
       │ → postTaskHook()               │
       └────────────────────────────────┘
                ↓
       ┌────────────────────────────────┐
       │ RED verdict?                   │
       │ Retries left?                  │
       └────────────────────────────────┘
            ↓ YES
       ┌────────────────────────────────────────┐
       │ Agent Swap Decision                    │
       │ If: SwapDuringRetries enabled AND      │
       │     red_count >= MinFailuresBeforeAdapt│
       │                                        │
       │ Call: SelectBetterAgent(               │
       │   currentAgent,                        │
       │   history,                             │
       │   review.SuggestedAgent)               │
       │                                        │
       │ If: returns different agent            │
       │   Agent = newAgent  ← FIELD MODIFIED   │
       │                                        │
       │ Retry execution with new agent        │
       └────────────────────────────────────────┘
            ↓
       ┌────────────────────────────────┐
       │ No retries left or not agent   │
       │ swap scenario                  │
       └────────────────────────────────┘
                ↓
       ┌────────────────────────────────┐
       │ Task Failed!                   │
       │ Status = "failed"              │
       │ → postTaskHook()               │
       └────────────────────────────────┘


STAGE 7: POST-EXECUTION HOOKS
════════════════════════════════════════════════════════════════════════════

    ┌─────────────────────────────────────┐
    │ postTaskHook()                      │
    │ Record to learning database:        │
    │ • PlanFile: (SourceFile)            │
    │ • TaskNumber: (Number)              │
    │ • TaskName: (Name)                  │
    │ • Agent: (Agent - may be swapped)   │
    │ • Prompt: (final prompt sent)       │
    │ • Output: (agent output)            │
    │ • Success: (GREEN/YELLOW or not)    │
    │ • ErrorMessage: (if failed)         │
    │ • Duration: (execution time)        │
    │ • QCVerdict: (final verdict)        │
    │ • QCFeedback: (from QC agent)       │
    │ • FailurePatterns: (from metadata)  │
    └─────────────────────────────────────┘
            ↓
    Learning database updated
    └─ Used by preTaskHook of future tasks
       (SelectBetterAgent for next runs,
        prompt enhancement for retries)
            ↓
    ┌─────────────────────────────────────┐
    │ updatePlanStatus()                  │
    │ Mark completion in plan file:       │
    │ • Number: (task identification)     │
    │ • Status: "completed" or "failed"   │
    │ • CompletedAt: timestamp            │
    │ • File: SourceFile (with locking)   │
    └─────────────────────────────────────┘


STAGE 8: MULTI-FILE PLAN COORDINATION
════════════════════════════════════════════════════════════════════════════

    Multiple plan files merged
            ↓
    ┌─────────────────────────────────────┐
    │ Per-File Locking Strategy           │
    ├─────────────────────────────────────┤
    │ task.SourceFile = which file task   │
    │                   came from         │
    │                                     │
    │ During Execute():                   │
    │ • Lock(task.SourceFile)             │
    │   [only one task updates this file  │
    │    at a time]                       │
    │ • Execute task                      │
    │ • updatePlanStatus() → same file    │
    │ • updateFeedback() → same file      │
    │ • Unlock()                          │
    │   [next task using this file can    │
    │    proceed]                         │
    │                                     │
    │ Result:                             │
    │ • Serial: updates to same file      │
    │ • Parallel: updates to diff files   │
    └─────────────────────────────────────┘


KEY FIELD MODIFICATIONS DURING EXECUTION
════════════════════════════════════════════════════════════════════════════

    Task.Prompt
    ├─ Original: [from parser]
    ├─ After preTaskHook: [+ learning context]
    ├─ After integration context: [+ dependency files]
    └─ Final: [sent to agent]

    Task.Agent
    ├─ Original: [from parser or default]
    ├─ After preTaskHook: [may change based on learning]
    ├─ After QC RED: [may change via SelectBetterAgent]
    └─ Final: [stored in execution record]

    Task.Metadata
    ├─ Original: [empty]
    ├─ After qcReviewHook: [+ failure_patterns]
    │                      [+ qc_verdict]
    └─ Final: [sent to postTaskHook]

    Task.Status
    ├─ Original: "pending"
    ├─ After pre-execute: "in-progress"
    ├─ After QC verdict: "completed" or "failed"
    └─ Final: [stored in plan file]

    Task.CompletedAt
    ├─ Original: nil
    ├─ After completion: timestamp
    └─ Final: [stored in plan file]
```

---

## Critical Decision Points

### 1. Integration Context Injection Decision

```
if task.DependsOn is not empty
   OR task.Type == "integration" {
    INJECT integration context into Prompt
}
```

**Impact**: Modifies Prompt BEFORE agent invocation
**Files affected**: Only that task's Prompt

### 2. Structured QC Mode Decision

```
combinedCriteria = task.SuccessCriteria
if task.Type == "integration" {
    combinedCriteria += task.IntegrationCriteria
}

if len(combinedCriteria) > 0 {
    USE structured review prompt
} else {
    USE legacy review prompt
}
```

**Impact**: Changes QC verification strategy
**Verdict logic**: Unanimous consensus vs. agent verdict

### 3. Domain-Specific QC Injection Decision

```
for each file in task.Files {
    extension = file.ext  (.go, .sql, .ts, .py)
    if extension in domainSpecificChecks {
        INJECT criteria into QC prompt
    }
}
```

**Impact**: Adds language-specific checks automatically
**No code change needed**: Just add more files with relevant extensions

### 4. Agent Swap Decision

```
if verdict == RED
   AND attempt >= MinFailuresBeforeAdapt
   AND SwapDuringRetries == true {

    newAgent = SelectBetterAgent(
        currentAgent,
        history,
        review.SuggestedAgent
    )

    if newAgent != currentAgent {
        task.Agent = newAgent
        RETRY with new agent
    }
}
```

**Impact**: Changes which agent executes next attempt
**Conditional**: Multiple conditions must be true

### 5. Per-File Locking Decision

```
if task.SourceFile != "" {
    fileToLock = task.SourceFile
} else if executor.SourceFile != "" {
    fileToLock = executor.SourceFile
} else {
    fileToLock = config.PlanPath
}

Lock(fileToLock)  ← Critical for multi-file
```

**Impact**: Determines locking granularity
**Priority order**: task.SourceFile > executor.SourceFile > config

---

## Field Dependency Graph

```
DependsOn ────────┐
Type              ├─→ Integration Context Injection ──→ Prompt (modified)
                  │
                  └─→ IntegrationCriteria included in QC

SuccessCriteria ──→ Structured QC Mode ──→ Per-criterion verdict
IntegrationCriteria (if Type="integration")
                  │
                  └─→ Unanimous consensus rules

Files ────────────→ Domain-specific QC injection
                  │
                  └─→ Expected file path verification

Agent ────────────→ Agent invocation
                  │
                  └─→ May be swapped on RED (via SelectBetterAgent)

SourceFile ───────→ Per-file locking
                  │
                  └─→ Plan file update routing

Number ───────────→ Task identification
                  │
                  └─→ Plan status updates

TestCommands ─────→ QC prompt information (not executed)

Status ───────────→ Resume logic (skip completed)

Metadata ─────────→ Stores QC verdict + failure patterns
                  │
                  └─→ postTaskHook retrieves patterns
```

---

## Execution State Machine

```
┌──────────────┐
│   PENDING    │
└──────┬───────┘
       │ Execute()
       ↓
┌──────────────────┐     qcReviewHook() captures:
│  IN-PROGRESS     │     • QC verdict
│ (agent working)  │     • Failure patterns (RED only)
└──────┬───────────┘
       │ Agent completes
       ↓
┌──────────────────┐
│  QC REVIEW       │
│  (structured or  │
│   legacy)        │
└──────┬───────────┘
       │
       ├─ GREEN/YELLOW ──┐
       │                  │
       └─ RED ────────────┤─ Agent swap? ──┐
                          │   (YES/NO)     │
                          ↓                │
                    ┌──────────────┐      │
                    │  COMPLETED   │      │
                    │              │      │
                    │ postTaskHook │      │
                    └──────────────┘      │
                                          │
                    ┌──────────────┐      │
                    │  RETRYING    │◄─────┘
                    │ (attempt++)  │
                    │ (agent swap) │
                    └──────┬───────┘
                           │ re-execute
                           ↓ (go to PENDING)

                    Eventually:
                    ┌──────────────┐
                    │   FAILED     │
                    │              │
                    │ postTaskHook │
                    └──────────────┘
```

