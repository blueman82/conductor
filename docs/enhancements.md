# Conductor UI Enhancements

Visual interface improvements for the Conductor multi-agent orchestration CLI.

## Table of Contents

- [Vision](#vision)
- [Architecture Concepts](#architecture-concepts)
- [TUI Framework Analysis](#tui-framework-analysis)
- [Mockups](#mockups)
- [Recommendation](#recommendation)
- [Implementation Path](#implementation-path)

---

## Vision

Extend Conductor's powerful multi-agent orchestration capabilities with an interactive terminal UI that provides:

1. **Real-time visualization** of task execution across parallel waves
2. **Live dependency graph** showing task status and relationships
3. **Interactive controls** for pause/resume, agent override, manual retry
4. **Streaming logs** from executing agents
5. **QC feedback display** with verdict breakdown (GREEN/RED/YELLOW)
6. **Learning insights** showing patterns, success rates, agent recommendations
7. **Plan editor** with syntax highlighting for Markdown/YAML
8. **Agent history** and inter-retry swapping decisions

---

## Architecture Concepts

### AI-Corp Hybrid Model

Conductor's plugin system (`/cook-auto`, `/doc`, `conductor run`) already provides the intelligent multi-stage workflow similar to AI-Corp's daemon approach:

**AI-Corp (Always-On Daemon)**:
```
incoming/ folder → Orchestrator polls (60s)
  ↓
Complexity analysis
  ↙        ↘
SIMPLE   COMPLEX
  ↓        ↓
Direct → Review (awaits approval)
  ↓
Spawn agents sequentially:
  1. work_agent (implements)
  2. verification_agent (tests)
  3. documentation_agent (docs)
  ↓
processed/ (archived)
```

**Conductor (Session-Based)**:
```
User runs /cook-auto "feature description"
  ↓
Requirements gathering + design doc
  ↓
User runs /doc or /doc-yaml
  ↓
Implementation plan (structured)
  ↓
User runs conductor run plan.yaml
  ↓
Execution with QC + adaptive learning
```

**Key Difference**: Conductor is developer-driven and explicit; AI-Corp is autonomous and always-watching. Both achieve similar results through different paradigms.

### Desktop Integration Concepts

If building a **visual companion** to Conductor, consider these UI paradigms:

**Option A: TUI (Terminal UI)**
- Interactive within the terminal
- No separate window
- Integrates with shell workflow
- Real-time task monitoring
- Live log streaming

**Option B: Desktop Overlay**
- Draggable PNG status icon (like AI-Corp)
- Always-on-top window showing current status
- Right-click context menu (pin, hide, exit)
- Updates every 2 seconds from status file
- Non-technical user friendly

**Option C: Web Dashboard**
- HTTP server exposing execution state
- WebSocket for real-time updates
- Accessible from browser
- Works over network
- More resource-intensive

---

## TUI Framework Analysis

For a **terminal UI** integrated with Conductor, we evaluated 5 Go frameworks:

### 1. Bubble Tea (Charm/BubbleTea)

**Architecture**: Event-driven (Elm Model-View-Update pattern)

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ Conductor ▸ Multi-Agent Automation                                [ q: quit ]│
├──────────────────────┬───────────────────────────────────────────────────────┤
│ TASKS                │ DETAILS                                               │
│──────────────────────┼───────────────────────────────────────────────────────│
│ ▶  [T-001] Parse     │ Task: Parse spec & build graph                        │
│     implementation   │ Agent: Planner                                        │
│     plan             │ Status: Running                                       │
│                      │ Progress: [███████████──────] 72%                     │
│    [T-002] Generate  │                                                       │
│     task branches    │ Substeps:                                             │
│    [T-003] Implement │  • Load spec                          ✔               │
│     API handlers     │  • Normalize tasks                    ✔               │
│    [T-004] Tests     │  • Detect circular deps               ◉ Active        │
│    [T-005] Docs      │  • Write plan.json                    … Pending       │
├──────────────────────┴───────────────────────────────────────────────────────┤
│ AGENTS                                                                      │
│ Planner   [RUNNING]  → step 3/5                                              │
│ Coder     [IDLE]                                                            │
│ Reviewer  [WAITING]                                                         │
│ Executor  [QUEUED]                                                          │
├──────────────────────────────────────────────────────────────────────────────┤
│ LOGS (live)                                                                  │
│ 14:22:11 planner  Loaded 37 tasks                                            │
│ 14:22:14 coder    Validating graph…                                          │
│ 14:22:18 planner  Cycle detected between T-014 ↔ T-019                       │
│ 14:22:22 review   Requesting human approval…                                 │
└──────────────────────────────────────────────────────────────────────────────┘
```

**Strengths**:
- ✅ Message-passing maps naturally to Conductor's `resultsCh` channels
- ✅ No race conditions (serial message queue)
- ✅ Composable component architecture
- ✅ Smooth animations, professional polish
- ✅ Ecosystem support (Charm, Bubbles, Lip Gloss)
- ✅ Commercial backing (Charm)

**Tradeoffs**:
- ⚠️ Steeper learning curve (Elm/MVU pattern)
- ⚠️ DAG visualization requires custom ASCII rendering
- ⚠️ More dependencies (lipgloss, bubbles)

**Verdict**: **Best for long-term architecture** (0.88 confidence)

---

### 2. tview (Rivo's TUI Widgets)

**Architecture**: Widget-based (GTK-like toolkit)

```
┌──────────────────────┬──────────────────────────────────────────────────────┐
│ TASK TREE            │ TASK DETAIL                                          │
│──────────────────────┼──────────────────────────────────────────────────────│
│ ▶ IMPLEMENTATION     │ ID: T-001                                            │
│   ▶ Parsing          │ Name: Parse spec                                     │
│     • Load spec      │ Agent: Planner                                       │
│     • Normalize IDs  │ Status: Running                                      │
│   ▶ Generation       │                                                      │
│     • Generate...    │ Logs:                                                │
│   ▶ Execution        │  [14:22:18] Detected cycle T-014 <-> T-019          │
│   ▶ Review           │  [14:22:21] Awaiting human decision                  │
├──────────────────────┴──────────────────────────────────────────────────────┤
│ AGENTS                                                                      │
│ +------------------+------------------+------------------+---------------+   │
│ | Planner: RUN     | Coder: IDLE      | Reviewer: WAIT   | Executor: Q   |   │
│ +------------------+------------------+------------------+---------------+   │
└──────────────────────────────────────────────────────────────────────────────┘
```

**Strengths**:
- ✅ Built-in TreeView widget (perfect for DAG visualization)
- ✅ Rich widget toolkit (Table, TextView, Form, Flex)
- ✅ Fast prototyping (minimal code)
- ✅ Single dependency, minimal footprint
- ✅ `QueueUpdateDraw()` for goroutine→UI updates
- ✅ Widely used (htop, nmtui, proven stability)

**Tradeoffs**:
- ⚠️ Manual mutex management for concurrent updates
- ⚠️ Less polish (basic ASCII, no animations)
- ⚠️ Single maintainer risk
- ⚠️ Limited styling consistency

**Verdict**: **Best for rapid shipping** (0.83 confidence)

---

### 3. gocui (Low-Level)

```
╔══════════════════════════╗╔═════════════════════════════════════════════════╗
║     TASKS                ║║              DETAILS                            ║
║--------------------------║║-------------------------------------------------║
║ > Parse plan (RUN)       ║║ Task: Parse specification                       ║
║   Generate branches      ║║ Agent: Planner                                   ║
║   Implement handlers     ║║ Status: Running                                  ║
║   Write tests            ║║                                                 ║
║   Documentation          ║║ Logs:                                           ║
╚══════════════════════════╝║ 14:22:01 Loaded spec                             ║
╔══════════════════════════╗║ 14:22:04 Normalized tasks                        ║
║        AGENTS            ║║ 14:22:18 Cycle detected                          ║
║--------------------------║║                                                 ║
║ Planner: RUNNING         ║╚═════════════════════════════════════════════════╝
║ Coder: IDLE              ║
║ Reviewer: WAITING        ║
╚══════════════════════════╝
```

**Assessment**: Too low-level. Would require building custom widgets (tables, trees, forms) from scratch. Estimated 3-4 weeks of development for solo dev. **Not recommended**.

---

### 4. termdash (Dashboard/Monitoring)

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ CONDUCTOR EXECUTION DASHBOARD (termdash)                                     │
├──────────────────────────┬─────────────────────────┬─────────────────────────┤
│ TASK PROGRESS            │ AGENT LOAD              │ QUEUE STATS             │
│ [██████████────] 68%     │ Planner: ███████ 72%    │ Waiting: 4 tasks        │
│ Parsing                  │ Coder:   ██ 18%         │ In progress: 1          │
│ Generation               │ Review:  ████ 40%       │ Completed: 12           │
│ Execution                │ Executor: █  8%          │ Failed: 0               │
├──────────────────────────┴─────────────────────────┴─────────────────────────┤
│ RECENT LOGS                                                                   │
│ 14:22:11  Loaded 37 tasks                                                      │
│ 14:22:14  Planner started parsing                                             │
│ 14:22:18  Cycle detected in graph                                             │
│ 14:22:21  Reviewer triggered human-in-loop                                     │
└──────────────────────────────────────────────────────────────────────────────┘
```

**Assessment**: Read-only monitoring dashboard. Weak for interactive controls (agent override, pause/resume). **Not recommended**.

---

### 5. go-prompt (REPL/Command Palette)

**Assessment**: Not a TUI framework. Designed for interactive command-line prompts, not full-screen layouts. Could augment another framework for command input, but not standalone. **Not recommended**.

---

## Mockups

### Real-Time Execution View

Shows live task execution with wave-based parallelism:

```
WAVE 1 (In Progress)
├─ [✔ DONE] T-001: Parse specification
├─ [✔ DONE] T-002: Validate dependencies
└─ [◉ RUN ] T-003: Generate implementation plan
     └─ Agent: Planner | Progress: 67% [█████████──────]

WAVE 2 (Queued)
├─ [ ] T-004: Implement API handlers
│   └─ Depends on: T-003
├─ [ ] T-005: Write unit tests
│   └─ Depends on: T-004
└─ [ ] T-006: Update documentation
    └─ Depends on: T-003, T-004

AGENTS ACTIVE
Planner [RUN ] 14:22:18 - Parsing spec
Coder   [IDLE]
Review  [WAIT] (next task queued)
```

### QC Feedback Panel

```
QC VERDICT: 🔴 RED (RETRY NEEDED)

Agent: code-reviewer
Feedback: Implementation missing error handling in edge cases

Issues:
  [CRITICAL] Missing nil checks on line 47
  [WARNING]  Insufficient test coverage (68% vs 80% target)
  [WARNING]  Documentation incomplete

Recommendations:
  • Add nil pointer guards
  • Increase test coverage to 90%
  • Document API response formats

Suggested Agent: golang-pro
Retry Strategy: Inter-retry agent swap recommended
```

### Learning Insights Panel

```
LEARNING INSIGHTS (Last 10 runs)

Success Rate: 87% (9/10 tasks passed QC on first attempt)

Common Failure Patterns:
  • Type errors (3 occurrences) - golang-pro handles 92% better
  • Missing tests (2 occurrences) - test-automator recommended
  • API doc gaps (1 occurrence) - technical-writer recommended

Best Performing Agents:
  1. golang-pro (94% success rate, 12 runs)
  2. backend-developer (88% success rate, 8 runs)
  3. code-reviewer (85% success rate, all types)

Adaptation Suggestions:
  • Use golang-pro for Task 7 (Go implementation)
  • Use test-automator for Task 9 (test suite)
```

---

## Recommendation

### Framework Decision Matrix

| Criterion | Bubble Tea | tview | Winner |
|-----------|-----------|-------|--------|
| **Architectural fit** | Channel→Message (native) | Widget callbacks | 🍵 Bubble Tea |
| **Time to ship** | 1-2 weeks | 3-4 days | 📊 tview |
| **DAG visualization** | Custom ASCII | Built-in TreeView | 📊 tview |
| **Concurrent safety** | Serial message queue | Manual mutex | 🍵 Bubble Tea |
| **Polish/animations** | Professional | Basic | 🍵 Bubble Tea |
| **Maintenance burden** | Compositional | Widget mutation | 🍵 Bubble Tea |
| **Ecosystem longevity** | Charm (commercial) | Single maintainer | 🍵 Bubble Tea |
| **Single dev feasibility** | Medium (MVU learning) | High (widgets) | 📊 tview |

### Deliberation Result

**Counsel split 2-2** between Bubble Tea and tview (no clear consensus), with both offering legitimate tradeoffs:

- **Bubble Tea (0.88)**: Better long-term architecture, event-driven concurrency maps naturally to Conductor's `resultsCh` channels, professional polish
- **tview (0.83)**: Faster shipping, built-in widgets (TreeView, Table, TextView), proven stability, minimal maintenance

### Decision: Hybrid Path

**Phase 1 (Sprint 1-2): Prototype with tview**
- Ship working prototype fast
- Prove TreeView DAG visualization
- Real-time updates via `QueueUpdateDraw()`
- Gather user feedback

**Phase 2 (If Polish Matters): Migrate to Bubble Tea**
- Refactor channel messages to `tea.Msg` types
- Swap widgets for composable views
- Add smooth animations, modern styling (lipgloss)
- Only if users demand or performance requires

---

## Implementation Path

### Phase 1: tview Prototype

**Week 1: Core Layout**
```go
// Pseudo-code
app := tview.NewApplication()

// 3-panel layout
taskTree := tview.NewTreeView()
taskDetail := tview.NewTextView()
agentStatus := tview.NewTable()

flex := tview.NewFlex().
    AddItem(taskTree, 0, 1, false).
    AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
        AddItem(taskDetail, 0, 2, false).
        AddItem(agentStatus, 5, 1, false), 0, 2, false)

app.SetRoot(flex, true)
```

**Week 2: Real-time Updates**
```go
// Listen to Conductor's executor output
go func() {
    for result := range executor.resultsCh {
        app.QueueUpdateDraw(func() {
            updateTaskRow(taskTree, result)
            updateAgentStatus(agentStatus)
        })
    }
}()
```

### Phase 2: Bubble Tea Migration (Optional)

Convert channels to messages:
```go
// Message types
type TaskCompletedMsg struct{ result models.TaskResult }
type WaveStartedMsg struct{ wave models.Wave }
type QCVerdictMsg struct{ verdict string, feedback string }

// Event loop
for result := range executor.resultsCh {
    program.Send(TaskCompletedMsg{result})
}

// Handler
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case TaskCompletedMsg:
        m.tasks[msg.result.Task.Number].Status = msg.result.Status
        return m, nil
    }
}
```

### Integration Points

**Conductor→UI Communication**:
1. Expose executor's `resultsCh` to UI process
2. Write status JSON to `.conductor/status.json` (for overlay modes)
3. Stream logs to UI via named pipe or WebSocket

**UI→Conductor Communication**:
1. Pause/resume: Send signal to executor
2. Retry task: Write command to queue file
3. Agent override: Update task in plan file via file lock

---

## Future Considerations

### Desktop Overlay (AI-Corp Model)

If moving beyond TUI, consider desktop overlay:

```go
// Pseudo-code: Desktop status window
type StatusOverlay struct {
    Icon      string    // ● GREEN/YELLOW/ORANGE/RED
    Tasks     int       // Total tasks
    Incoming  int       // Files waiting
    Updated   time.Time // Last update
}

// Watch .conductor/status.json, update PNG icon every 2s
// Right-click menu: Pin, Hide, Exit
```

### Web Dashboard

Expose Conductor via HTTP:

```go
// REST API
GET  /api/execution/{id}/tasks
GET  /api/execution/{id}/waves
GET  /api/execution/{id}/logs
POST /api/execution/{id}/pause
POST /api/execution/{id}/resume

// WebSocket
WS /ws/execution/{id}/updates
```

---

## References

- **Bubble Tea**: https://github.com/charmbracelet/bubbletea
- **tview**: https://github.com/rivo/tview
- **Charm Ecosystem**: https://charm.sh
- **Conductor Docs**: docs/conductor.md
