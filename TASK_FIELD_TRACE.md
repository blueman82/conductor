# Task Field Execution Trace - Conductor v2.5.2

## Executive Summary

This document traces how Conductor's executor component accesses and uses Task struct fields during execution. The analysis covers which fields are actively used versus stored but unused, and how each field triggers specific executor behaviors.

---

## Task Struct Definition

**Location**: `/Users/harrison/Github/conductor/internal/models/task.go`

```go
type Task struct {
    // Core identification
    Number        string                 // Task number/identifier
    Name          string                 // Task name/title

    // Execution context
    Files         []string               // Files to be modified/created
    DependsOn     []string               // Task dependencies
    EstimatedTime time.Duration          // Estimated time to complete
    Agent         string                 // Agent to use
    Prompt        string                 // Full task description/prompt
    WorktreeGroup string                 // Worktree group assignment

    // Status tracking
    Status        string                 // pending, in_progress, completed, skipped
    StartedAt     *time.Time             // Execution start timestamp
    CompletedAt   *time.Time             // Execution completion timestamp
    SourceFile    string                 // Source plan file (multi-file plans)
    Metadata      map[string]interface{} // Custom metadata

    // Structured QC (v2.3+)
    SuccessCriteria     []string          // Success criteria for QC
    TestCommands        []string          // Commands to run for verification
    Type                string            // Task type: regular or integration
    IntegrationCriteria []string          // Cross-component validation criteria

    // Enhanced console output
    ExecutionStartTime time.Time          // When execution started
    ExecutionEndTime   time.Time          // When execution ended
    ExecutionDuration  time.Duration      // Total duration
    ExecutedBy         string             // Agent name that executed task
    FilesModified      int                // Count of modified files
    FilesCreated       int                // Count of created files
    FilesDeleted       int                // Count of deleted files
}
```

---

## Field Usage During Execution

### 1. CRITICAL FIELDS (Must be present for execution)

#### `Number` (string)
- **Required**: YES - Validation error if empty
- **Accessed In**:
  - `task.go`: Line 482 - Execute() method entry
  - `task.go`: Line 530 - updatePlanStatus() to track which task
  - `task.go`: Line 644 - ReviewResult.Task reference
  - `qc.go`: Line 361 - QC review task identification
  - `learning.go`: Pre/post-task hooks for learning store queries
- **Behavior Trigger**: Identifies task in waves, enables plan updates
- **Stored in**: Task.Number (string)

#### `Name` (string)
- **Required**: YES - Validation error if empty
- **Accessed In**:
  - `task.go`: Line 482 - Execute() result
  - `qc.go`: Line 125-130 - BuildReviewPrompt() includes task name
  - `qc.go`: Line 163 - BuildStructuredReviewPrompt() heading
  - `qc.go`: Line 529 - reviewSingleAgent() prompt building
  - `integration_prompt.go`: Line 31 - Dependency context
- **Behavior Trigger**: Appears in QC prompts and user-facing output
- **Used For**: Logging, feedback, context

#### `Prompt` (string)
- **Required**: YES - Validation error if empty
- **Accessed In**:
  - `task.go`: Line 517-519 - buildIntegrationPrompt() injects context BEFORE invocation
  - `task.go`: Line 564 - Passed to invoker.Invoke()
  - `qc.go`: Line 125-130 - BuildReviewPrompt() includes full prompt
  - `qc.go`: Line 165 - BuildStructuredReviewPrompt() includes in section
  - `agent.go`: Invoker creates Claude CLI command with this prompt
- **Behavior Trigger**: Directly sent to Claude agent, contains task implementation details
- **Critical**: Integration context injected here BEFORE agent invocation

### 2. ACTIVELY USED FIELDS (Direct behavior impact)

#### `Agent` (string) - Optional but highly impactful
- **Default**: Overridden by config.DefaultAgent if empty (line 522-524)
- **Accessed In**:
  - `task.go`: Line 522-524 - Apply default agent if empty
  - `task.go`: Line 564 - Passed to invoker.Invoke(agent specified)
  - `task.go`: Line 721 - SelectBetterAgent() for inter-retry agent swapping
  - `task.go`: Line 725-726 - Agent swap on RED verdict
  - `task.go`: Line 464 - Stored in ExecutionAttempt
  - `qc.go`: Line 446-449 - SelectionContext.ExecutingAgent for intelligent QC
  - `learning.go`: Hooks track which agent executed task
- **Behavior Trigger**:
  - Determines which Claude agent executes the task
  - Can be swapped between retries based on QC feedback
  - Influences intelligent QC agent selection
  - Stored in execution history for learning
- **Inter-Retry Swapping**: When RED verdict received + swap enabled:
  ```go
  if newAgent, reason := learning.SelectBetterAgent(task.Agent, history, review.SuggestedAgent);
     newAgent != "" && newAgent != task.Agent {
      task.Agent = newAgent  // Swap for next retry
  }
  ```

#### `SourceFile` (string) - Multi-file plans
- **Priority**: High (checked before SourceFile fallback)
- **Accessed In**:
  - `task.go`: Line 490-491 - Lock file for concurrent updates
  - `task.go`: Line 456 - Determine which file to update with feedback
  - `task.go`: Line 714-716 - Learning store query file path
  - `task.go`: Line 776 - updatePlanStatus() file selection
- **Behavior Trigger**:
  - Determines which plan file to update (critical for multi-file plans)
  - File-level locking to prevent concurrent modifications
  - Priority order: task.SourceFile > executor.SourceFile > config.PlanPath
- **File Locking**: `FileLockManager.Lock(fileToLock)` ensures only one task updates file at a time

#### `DependsOn` ([]string)
- **Parser**: Graph builder (graph.go) validates dependencies exist
- **Accessed In**:
  - `task.go`: Line 517 - Check if len(task.DependsOn) > 0 to inject integration context
  - `integration_prompt.go`: Line 13-15 - Used to build dependency context
  - `integration_prompt.go`: Line 24-39 - Loop through each dependency for context
  - `graph.go`: Kahn's algorithm for wave calculation
  - `models.go`: HasCyclicDependencies() DFS with color marking
- **Behavior Trigger**:
  - Enables integration context injection (see integration_prompt.go)
  - Drives wave calculation via topological sort
  - Cycle detection validates acyclic graph
- **Integration Context Injection**:
  ```go
  if te.Plan != nil && (task.Type == "integration" || len(task.DependsOn) > 0) {
      task.Prompt = buildIntegrationPrompt(task, te.Plan)  // Injects before invocation
  }
  ```

#### `Files` ([]string)
- **Accessed In**:
  - `qc.go`: Line 198-205 - BuildStructuredReviewPrompt() includes expected file paths
  - `qc.go`: Line 208 - inferDomainChecks() auto-injects domain-specific criteria
  - `integration_prompt.go`: Line 33-34 - Lists files to read in dependency context
  - `executor/learning`: File change detection for pattern analysis
- **Behavior Trigger**:
  - Generates domain-specific QC criteria based on file extensions
  - QC prompt verifies agent modified expected files
  - Dependency context shows which files to read
- **Domain-Specific Checks** (auto-injected):
  - `.go` files → Go-specific review criteria (error handling, nil pointer safety, etc.)
  - `.sql` files → SQL-specific criteria (SQL injection, indexes, ON DELETE behavior)
  - `.ts/.tsx` files → TypeScript/React criteria (no 'any' types, hooks rules, etc.)
  - `.py` files → Python-specific criteria (type hints, exception handling, etc.)

### 3. QUALITY CONTROL IMPACT FIELDS

#### `SuccessCriteria` ([]string)
- **Parser**: YAML parser reads directly (yaml.go line 170, 43)
- **Accessed In**:
  - `qc.go`: Line 149-155 - getCombinedCriteria() merges with IntegrationCriteria
  - `qc.go`: Line 348 - Check hasSuccessCriteria for prompt selection
  - `qc.go`: Line 383-390 - Criteria aggregation with unanimous consensus
  - `qc.go`: Line 518 - reviewSingleAgent() includes in structured prompt
  - `qc.go`: Line 168-176 - BuildStructuredReviewPrompt() formats as numbered list
- **Behavior Trigger**:
  - Switches QC from legacy blob review to structured criteria verification
  - When present: QC returns CriteriaResults array with per-criterion PASS/FAIL
  - Aggregation: Any failed criterion = RED verdict (unanimous consensus required)
  - Prompt includes numbered list: `0. [ ] Criterion 1`, `1. [ ] Criterion 2`, etc.
- **QC Response Schema**:
  ```json
  {
    "verdict": "RED",  // Will be RED if ANY criterion failed
    "criteria_results": [
      {"index": 0, "criterion": "...", "passed": true},
      {"index": 1, "criterion": "...", "passed": false}
    ]
  }
  ```

#### `IntegrationCriteria` ([]string)
- **Markdown**: Extracted as `### Integration Criteria` section
- **YAML**: Parsed as `integration_criteria` field
- **Accessed In**:
  - `qc.go`: Line 149-155 - getCombinedCriteria() merges into success_criteria
  - `qc.go`: Line 152 - Only appended if task.Type == "integration"
  - `qc.go`: Line 178-185 - BuildStructuredReviewPrompt() displays as separate section
  - `qc.go`: Line 493-495 - Multi-agent criteria aggregation (unanimous consensus)
- **Behavior Trigger**:
  - Only used for integration tasks (type: "integration")
  - Combined with success_criteria into single verification array
  - QC verifies BOTH types before GREEN verdict
  - Enables cross-component validation
- **Dual Criteria System**:
  - success_criteria: Component-level checks
  - integration_criteria: Cross-component interaction checks
  - ALL criteria must pass for GREEN (unanimous consensus)

#### `TestCommands` ([]string)
- **Parser**: YAML parser reads (yaml.go line 171, 44)
- **Accessed In**:
  - `qc.go`: Line 189-194 - BuildStructuredReviewPrompt() includes test commands section
  - QC receives list of commands to suggest running for verification
- **Behavior Trigger**:
  - Listed in QC prompt as "## TEST COMMANDS"
  - QC agent can suggest running these to verify task completion
  - Informational - not automatically executed by conductor
- **Prompt Injection**:
  ```markdown
  ## TEST COMMANDS
  - `go test ./internal/auth/ -v`
  - `curl -X GET http://localhost:8080/api/users`
  ```

#### `Type` (string)
- **Values**: "integration", empty (defaults to regular)
- **Accessed In**:
  - `qc.go`: Line 152 - If task.Type == "integration", append IntegrationCriteria
  - `task.go`: Line 517 - If Type == "integration" OR has dependencies, build integration prompt
  - `models.go`: Line 101-103 - IsIntegration() helper returns Type == "integration"
  - `parser/yaml_validation.go`: Validates integration tasks have integration_criteria
- **Behavior Trigger**:
  - Marks task for integration context injection
  - Enables IntegrationCriteria usage in QC prompts
  - Validation: Integration tasks must have integration_criteria (yaml_validation.go)
- **Note**: Tasks WITHOUT explicit type but WITH dependencies still get integration context

### 4. EXECUTION TRACKING FIELDS

#### `Status` (string)
- **Values**: "pending", "in-progress", "completed", "failed", "skipped"
- **Accessed In**:
  - `executor/wave.go`: Line ~X - CanSkip() check if Status == "completed" or "skipped"
  - `executor/wave.go`: ~Filters completed tasks before execution
  - `task.go`: Line 530, 634, 682, 693, 742 - updatePlanStatus() sets status
  - Parser reads from plan files for resume capability
- **Behavior Trigger**:
  - Tasks with Status == "completed" or "skipped" are filtered in wave execution
  - SkipCompleted flag: Re-read plan to skip already-completed tasks
  - Enables resumable execution
- **Lifecycle**: pending → in-progress → completed/failed

#### `CompletedAt` (*time.Time)
- **Accessed In**:
  - `task.go`: Line 765-768 - Set when task completes (updatePlanStatus)
  - `updater.UpdateTaskStatus()` writes to plan file
  - Enables completion timestamp tracking
- **Used For**: Resume capability, execution history

### 5. INTEGRATION CONTEXT FIELDS

#### `WorktreeGroup` (string)
- **Parser**: Markdown reads `**WorktreeGroup**: value` (markdown.go line 352-355)
- **Parser**: YAML reads `worktree_group: value` (yaml.go line 38, 168)
- **Accessed In**:
  - `orchestrator.go`: Line 307-311 - Deduplicates worktree groups during multi-file merge
  - Wave.GroupInfo: map[string][]string (groups → task numbers)
- **Behavior Trigger**:
  - Organizational metadata only - NOT used for execution control
  - Preserved during multi-file plan merging
  - Stored in Wave.GroupInfo for logging/analysis
- **Note**: Currently stored but not actively used for execution isolation or ordering

#### `Metadata` (map[string]interface{})
- **Accessed In**:
  - `task.go`: Line 362-368 - qcReviewHook() stores QC verdict and failure patterns
  - `task.go`: Line 385-388 - postTaskHook() retrieves failure patterns for learning
  - Extensible for custom data
- **Behavior Trigger**:
  - Stores: `qc_verdict`, `failure_patterns`
  - Failure patterns extracted via keyword matching
  - Used to enhance prompt with learning context

### 6. EXECUTION METADATA FIELDS (Display/Logging Only)

#### `ExecutionStartTime`, `ExecutionEndTime`, `ExecutionDuration`
- **Set By**: Task executor during/after execution
- **Used For**: Enhanced console output, duration display
- **Behavior Trigger**: None during execution, used for display

#### `ExecutedBy` (string)
- **Set By**: Task executor (stores Agent name)
- **Used For**: Execution history, display
- **Behavior Trigger**: None during execution

#### `FilesModified`, `FilesCreated`, `FilesDeleted`
- **Set By**: Console logger during execution tracking
- **Used For**: Display and metrics
- **Behavior Trigger**: None during execution

---

## Field Access by Executor Component

### `DefaultTaskExecutor.Execute()` Method

**Location**: `/Users/harrison/Github/conductor/internal/executor/task.go` line 482

**Execution Flow**:

```
1. Line 490-494: Determine fileToLock
   → Uses: SourceFile > executor.SourceFile > config.PlanPath
   → File-level locking via FileLockManager

2. Line 509: preTaskHook(ctx, &task)
   → Learning disabled: No-op
   → Else: AnalyzeFailures(), adapt agent if recommended

3. Line 517-519: Inject integration context
   → If: Plan != nil AND (Type == "integration" OR len(DependsOn) > 0)
   → buildIntegrationPrompt() adds dependency file context

4. Line 522-524: Apply default agent
   → If: Agent == "" AND config.DefaultAgent != ""
   → Sets: task.Agent = config.DefaultAgent

5. Line 530: updatePlanStatus(task, "in-progress", false)
   → Updates: Status = "in-progress"
   → File: task.SourceFile (with per-file locking)

6. Line 564: invoker.Invoke(ctx, task)
   → Passes: task.Agent, task.Prompt, task.Number
   → Claude CLI receives prompt with integrated context

7. Line 644: te.reviewer.Review(ctx, task, output)
   → QC review with structured prompt (if SuccessCriteria present)
   → Checks: success_criteria + integration_criteria (if Type="integration")

8. Line 662: updateFeedback(task, attempt+1, agentOutput, qcFeedback, verdict)
   → Stores: Agent, Verdict, AgentOutput, QCFeedback, Timestamp
   → File: task.SourceFile (with per-file locking)

9. Line 667: qcReviewHook(ctx, &task, verdict, feedback, output)
   → Extract failure patterns from RED verdicts
   → Store in task.Metadata["failure_patterns"]
   → Store in task.Metadata["qc_verdict"]

10. Line 721-727: Agent swap (if RED and swap enabled)
    → Call: learning.SelectBetterAgent(task.Agent, history, review.SuggestedAgent)
    → If: newAgent != "" AND newAgent != task.Agent
    → Swap: task.Agent = newAgent (for next retry)

11. Line 743: postTaskHook(ctx, &task, &result, verdict)
    → Record execution to learning store
    → Stores: Agent, Verdict, Duration, Output, ErrorMessage, FailurePatterns
```

### Quality Control Integration (`qc.go`)

**Key Methods**:

1. **Review()** (line 339)
   - Checks: `shouldUseMultiAgent()` to route to single vs. multi-agent
   - Single agent: Calls `reviewSingleAgent()`
   - Multi-agent: Calls `ReviewMultiAgent()`

2. **BuildStructuredReviewPrompt()** (line 160)
   - Uses: task.Name, task.Prompt, task.Files
   - Uses: SuccessCriteria + IntegrationCriteria (if Type="integration")
   - Uses: TestCommands
   - Injects: Domain-specific criteria based on file extensions
   - Includes: Expected file paths verification section

3. **getCombinedCriteria()** (line 149)
   - Merges: SuccessCriteria + IntegrationCriteria
   - Logic:
     ```go
     criteria := SuccessCriteria...
     if Type == "integration" {
         criteria += IntegrationCriteria...
     }
     ```

4. **aggregateMultiAgentCriteria()** (line 826)
   - Uses: SuccessCriteria + IntegrationCriteria count
   - Per-criterion consensus: ALL agents must agree for PASS
   - Falls back to `aggregateVerdicts()` if no criteria
   - Critical: Line 905-911 - Respects agent RED verdicts even if criteria pass

### Learning Integration (`task.go` hooks)

**preTaskHook()** (line 213):
- Queries: learning.Store.AnalyzeFailures(planFile, task.Number, minThreshold)
- Adapts: task.Agent if ShouldTryDifferentAgent
- Enhances: task.Prompt with learning context

**qcReviewHook()** (line 349):
- Extracts: Failure patterns via keyword matching
- Stores: In task.Metadata["failure_patterns"]
- Only RED verdicts: Returns empty array for GREEN/YELLOW

**postTaskHook()** (line 377):
- Records: TaskExecution to learning.Store
- Includes: Agent, Verdict, Duration, Output, ErrorMessage, FailurePatterns

---

## Integration Context Injection

**Location**: `/Users/harrison/Github/conductor/internal/executor/integration_prompt.go`

**Triggered When**:
```go
if te.Plan != nil && (task.Type == "integration" || len(task.DependsOn) > 0)
```

**What Gets Injected**:
1. Header: "# INTEGRATION TASK CONTEXT"
2. For each dependency task:
   - Heading: "## Dependency: Task X - Name"
   - Files to read from dependency
   - Justification for reading those files
   - Integration interface explanation
3. Separator: "---\n\n# YOUR INTEGRATION TASK"
4. Original task.Prompt

**Result**: task.Prompt is modified BEFORE agent invocation

---

## Field Dependencies and Interactions

```
┌─────────────────────────────────────────┐
│          Task Fields & Behaviors        │
└─────────────────────────────────────────┘

Number → Identifies task in waves, enables updates
Name → User-facing identification
Prompt → Core implementation details
    ↓ (modified by)
DependsOn → Triggers integration context injection
Type="integration" → Includes IntegrationCriteria in QC

Agent → Determines executing agent
    ↓ (can be swapped on)
RED verdict + swap enabled → SelectBetterAgent()

SuccessCriteria → Enables structured QC
IntegrationCriteria → Only if Type="integration"
    ↓
BuildStructuredReviewPrompt() includes both
    ↓
Aggregation: unanimous consensus (any fail = RED)

Files → Injects domain-specific QC criteria
    ↓
.go, .sql, .ts, .py → Language-specific checks

SourceFile → Determines which file to lock/update
    ↓
FileLockManager.Lock() → Per-file concurrent safety

Status → Enables skip-completed resume logic
WorktreeGroup → Stored but not actively used for execution
Metadata → Stores QC verdict and failure patterns
```

---

## Fields NOT Used During Execution

| Field | Status | Why | Potential Use |
|-------|--------|-----|---|
| `EstimatedTime` | Parsed, not used | Metadata only | Timeout calculation, capacity planning |
| `StartedAt` | Set but not read | Redundant with ExecutionStartTime | Could replace ExecutionStartTime |
| `WorktreeGroup` | Stored in Plan | Organizational only | Could drive execution isolation/ordering |
| `ExecutionStartTime` | Set for display | Display-only | Duration tracking |
| `ExecutionEndTime` | Set for display | Display-only | Duration tracking |
| `ExecutionDuration` | Set for display | Display-only | Duration tracking |
| `ExecutedBy` | Set for display | Display-only | Execution history |
| `FilesModified` | Tracking only | Console display | Metrics aggregation |
| `FilesCreated` | Tracking only | Console display | Metrics aggregation |
| `FilesDeleted` | Tracking only | Console display | Metrics aggregation |

---

## Critical Execution Behaviors Triggered by Fields

### 1. Integration Context Injection
**Trigger**: `DependsOn` not empty OR `Type == "integration"`
**Effect**: Modifies `Prompt` to include dependency file context BEFORE agent invocation
**Location**: `task.go` line 517-519

### 2. Structured QC vs. Legacy QC
**Trigger**: `SuccessCriteria` not empty
**Effect**: Switches to BuildStructuredReviewPrompt() with numbered criteria
**Location**: `qc.go` line 348-356

### 3. Domain-Specific Review Criteria
**Trigger**: `Files` array with .go/.sql/.ts/.py extensions
**Effect**: Auto-injects language-specific QC criteria to prompt
**Location**: `qc.go` line 208, domain checks at line 18-49

### 4. Integration Criteria Validation
**Trigger**: `Type == "integration"`
**Effect**: Includes IntegrationCriteria in QC prompt and verdict calculation
**Location**: `qc.go` line 152-154, 178-185

### 5. Per-File Locking
**Trigger**: `SourceFile` set (multi-file plans)
**Effect**: Serializes updates to same file, allows parallel updates to different files
**Location**: `task.go` line 490-498

### 6. Agent Swapping on Retry
**Trigger**: RED verdict + `SwapDuringRetries` enabled + red count >= `MinFailuresBeforeAdapt`
**Effect**: Calls `SelectBetterAgent()` to potentially change `Agent` for next retry
**Location**: `task.go` line 708-728

### 7. Learning Adaptation
**Trigger**: `LearningStore` configured + `AutoAdaptAgent` enabled
**Effect**: Adapts `Agent` based on historical failure patterns
**Location**: `task.go` line 233-246

---

## Parsing: Where Task Fields Come From

### Markdown Parser (`internal/parser/markdown.go`)

**What it extracts**:
- Number: From heading `## Task N:` (regex match)
- Name: From heading text after `Task N: `
- Files: From `**Files**: ...` or `**File(s)**: ...` metadata
- DependsOn: From `**Depends on**: ...` metadata (comma/space separated)
- EstimatedTime: From `**Estimated time**: ...` metadata
- Agent: From `**Agent**: ...` metadata
- WorktreeGroup: From `**WorktreeGroup**: ...` metadata
- Status: From `**Status**: ...` or checkbox `[x]` notation
- Prompt: Entire task section content (lines from heading to next ## heading)
- SuccessCriteria: NOT EXTRACTED in Markdown (legacy markdown format)
- IntegrationCriteria: NOT EXTRACTED in Markdown
- Type: NOT EXTRACTED in Markdown
- TestCommands: NOT EXTRACTED in Markdown

**Note**: Markdown format doesn't support SuccessCriteria/IntegrationCriteria/Type/TestCommands

### YAML Parser (`internal/parser/yaml.go`)

**What it extracts** (lines 162-174):
```go
task := models.Task{
    Number:              taskNum,                  // From task_number field
    Name:                yt.Name,                  // From name field
    Files:               yt.Files,                 // From files array
    DependsOn:           dependsOn,                // From depends_on array
    Agent:               yt.Agent,                 // From agent field
    WorktreeGroup:       yt.WorktreeGroup,         // From worktree_group field
    Status:              yt.Status,                // From status field
    SuccessCriteria:     yt.SuccessCriteria,       // ✓ From success_criteria array
    TestCommands:        yt.TestCommands,          // ✓ From test_commands array
    Type:                yt.Type,                  // ✓ From type field
    IntegrationCriteria: yt.IntegrationCriteria,   // ✓ From integration_criteria array
}
```

**Validations** (lines 195-202):
- ValidateTaskType(): Checks task type is valid ("integration" or empty)
- ValidateIntegrationTask(): Enforces that integration tasks have integration_criteria

---

## Summary Table: Field Usage

| Field | Parser | Active Use | Behavior Trigger | Multi-Exec Impact |
|-------|--------|-----------|-----------------|-------------------|
| Number | Both | High | Task identification, plan updates | Critical |
| Name | Both | High | QC prompts, logging | Important |
| Prompt | Both | High | Agent invocation, modified by integration context | Critical |
| Files | Both | High | Domain-specific QC injection, integration context | Important |
| DependsOn | Both | High | Integration context injection trigger | Critical |
| Agent | Both | High | Agent selection, inter-retry swapping | Critical |
| SourceFile | Parser | High | Per-file locking, multi-file coordination | Critical |
| SuccessCriteria | YAML only | High | Structured QC mode trigger | Important |
| IntegrationCriteria | YAML only | High (if Type="integration") | Dual-criteria QC | Important |
| Type | YAML only | High | Integration context, criteria merging | Important |
| TestCommands | YAML only | Medium | QC prompt information | Informational |
| Status | Both | Medium | Skip-completed resume logic | Important |
| WorktreeGroup | Both | Low | Stored in Wave.GroupInfo, not used | Organizational |
| EstimatedTime | Both | None | Parsed but unused | Unused |
| Metadata | Code only | Medium | Stores QC verdict, failure patterns | Execution tracking |
| CompletedAt | Both | Low | Timestamp tracking | Informational |
| All Execution* | Code only | Low | Display and metrics only | Display only |

---

## Conclusions

### Must-Have Fields for Execution
1. **Number** - Task identification
2. **Name** - Human identification
3. **Prompt** - Implementation details
4. **Agent** - (or DefaultAgent fallback)

### High-Impact Optional Fields
5. **DependsOn** - Triggers integration context injection
6. **SourceFile** - Required for multi-file plans
7. **Files** - Enables domain-specific QC
8. **SuccessCriteria** - Enables structured QC mode
9. **Type** - Enables integration task features
10. **IntegrationCriteria** - For integration tasks only

### Stored But Not Active
- EstimatedTime
- WorktreeGroup (organizational metadata)
- All Execution* fields (display only)

### Field Modification During Execution
- **Prompt**: Modified by buildIntegrationPrompt() BEFORE invocation (critical)
- **Agent**: Modified by preTaskHook() (learning), qcReviewHook() (inter-retry swap)
- **Metadata**: Modified by qcReviewHook() (stores verdict, patterns)
