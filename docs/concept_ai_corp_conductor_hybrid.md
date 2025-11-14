# AI-Corp Architecture Visualization

ASCII art diagrams showing AI-Corp's architecture and how it compares to Conductor's hybrid approach.

## Desktop View

```
┌─────────────────────────────────────────────────────────────────────┐
│  Windows Desktop                                                     │
│                                                                      │
│   [Chrome]  [VS Code]  [File Explorer]                              │
│                                                                      │
│                                              ┌──────────────────┐   │
│                                              │  AI-Corp Status  │×  │
│   ┌─────────────────────┐                   ├──────────────────┤   │
│   │  File: incoming/    │                   │   ●  IDLE        │   │
│   │  feature-req.md     │                   │                  │   │
│   │                     │                   │  Tasks: 3        │   │
│   │  [Dragging...]      │                   │  Incoming: 1     │   │
│   └─────────────────────┘                   │                  │   │
│                                              │  Updated: 14:32  │   │
│                      Code/                   └──────────────────┘   │
│                      └─ my-app/                  ↑                  │
│                         └─ src/              Draggable PNG          │
│                                               status overlay         │
└─────────────────────────────────────────────────────────────────────┘
```

## Status Icon States

```
Status Overlay Icons (PNG):

    ●  GREEN      = IDLE (watching for work)
    ●  YELLOW     = PROCESSING (intake analysis)
    ●  ORANGE     = DEVELOPING (agent working)
    ●  BLUE       = VALIDATING (checking work)
    ●  PURPLE     = DOCUMENTING (updating docs)
    ●  RED        = ERROR (system failure)
    ●  MAGENTA    = FEEDBACK NEEDED (awaiting review)
```

## Folder Structure & Workflow

```
OneDrive/ai-corp/
│
├─── incoming/ ◄──────────── [User drops files here]
│    │                             │
│    └─ feature-request.md         │
│                                  │
│    [Orchestrator polls every 60s] ◄───┐
│                 │                      │
│                 ▼                      │
│    ┌──────────────────────┐          │
│    │  Complexity Analysis │          │
│    └──────────────────────┘          │
│            │         │                │
│         SIMPLE    COMPLEX             │
│            │         │                │
│            ▼         ▼                │
├─── pending/      review/              │
│    │             │                    │
│    └─tasks.md    └─requirements.md   │
│                  └─design.md          │
│                       │                │
│                       ▼                │
│              [User reviews & approves]│
│                       │                │
│                       ▼                │
│    ┌──────────────────────────────┐   │
│    │   Spawn Work Agent           │   │
│    │   (Claude CLI session)       │   │
│    └──────────────────────────────┘   │
│                 │                      │
│                 ▼                      │
│    ┌──────────────────────────────┐   │
│    │   Spawn Verification Agent   │   │
│    │   (Check completed work)     │   │
│    └──────────────────────────────┘   │
│                 │                      │
│                 ▼                      │
│    ┌──────────────────────────────┐   │
│    │   Spawn Documentation Agent  │   │
│    │   (Update project docs)      │   │
│    └──────────────────────────────┘   │
│                 │                      │
│                 ▼                      │
├─── processed/                         │
│    └─ feature-request.md ─────────────┘
│         (archived)
│
├─── logs/
│    ├─ orchestrator_log.txt
│    ├─ task_log.txt
│    └─ orchestrator_status.json ◄──── [OSD reads this]
│
└─── Documents/code/my-app/ ◄──── [Agents work here]
     ├─ src/
     ├─ tests/
     └─ docs/
```

## Orchestrator State Machine

```
┌─────────────────────────────────────────────────────────────────┐
│                    AI-Corp Orchestrator                          │
│                      (Python Daemon)                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
              ┌───────────────────────────┐
              │    MONITORING STATE       │
              │  ● GREEN - Watching...    │
              └───────────────────────────┘
                      │           ▲
            File detected?        │
                      │           │
                      ▼           │
              ┌───────────────────────────┐
              │    PROCESSING STATE       │
              │  ● YELLOW - Analyzing...  │
              │  - Read incoming file     │
              │  - Assess complexity      │
              │  - Route to queue         │
              └───────────────────────────┘
                      │
            ┌─────────┴──────────┐
            ▼                    ▼
    ┌───────────────┐    ┌────────────────────┐
    │  SIMPLE TASK  │    │  COMPLEX FEATURE   │
    │  ● ORANGE     │    │  ● MAGENTA         │
    │  Direct to    │    │  Requirements      │
    │  pending/     │    │  gathering in      │
    │  tasks.md     │    │  review/           │
    └───────────────┘    └────────────────────┘
            │                    │
            │            [Awaits approval]
            │                    │
            └────────┬───────────┘
                     ▼
         ┌───────────────────────────┐
         │   DEVELOPING STATE        │
         │ ● ORANGE - Implementing...|
         │ spawn: work_agent         │
         └───────────────────────────┘
                     │
                     ▼
         ┌───────────────────────────┐
         │   VALIDATING STATE        │
         │ ● BLUE - Checking...      │
         │ spawn: verification_agent │
         └───────────────────────────┘
                     │
                     ▼
         ┌───────────────────────────┐
         │  DOCUMENTING STATE        │
         │ ● PURPLE - Updating docs..|
         │ spawn: documentation_agent│
         └───────────────────────────┘
                     │
                     ▼
         ┌───────────────────────────┐
         │    COMPLETED STATE        │
         │  Move to processed/       │
         │  Update task_log.txt      │
         └───────────────────────────┘
                     │
                     └──► Back to MONITORING
```

## Parallel View: OSD + Terminal

```
┌──────────────────────┐         ┌─────────────────────────────────┐
│   Desktop Overlay    │         │     Console Output              │
│   (PyQt5 Window)     │         │  (orchestrator terminal)        │
├──────────────────────┤         ├─────────────────────────────────┤
│                      │         │                                 │
│    ●  DEVELOPING     │         │ [2025-01-12 14:32:15]          │
│                      │         │ Detected: incoming/feature.md   │
│   Tasks: 3           │         │                                 │
│   Incoming: 0        │         │ [2025-01-12 14:32:16]          │
│                      │         │ Complexity: HIGH                │
│   Last: 14:32        │         │ → Creating requirements doc     │
│                      │         │                                 │
│  [Right-click menu]  │         │ [2025-01-12 14:33:42]          │
│   • Pin on top       │         │ Spawning work_agent...          │
│   • Hide 10s         │         │ Session ID: abc123              │
│   • Exit             │         │                                 │
│                      │         │ [2025-01-12 14:45:10]          │
└──────────────────────┘         │ ✓ Work complete                │
         ↕                        │ Spawning verification_agent...  │
    Updates every                │                                 │
    2 seconds from               │ [2025-01-12 14:48:33]          │
    status.json                  │ ✓ Verification passed           │
                                 │                                 │
                                 │ [2025-01-12 14:48:34]          │
                                 │ Returning to monitoring...      │
                                 │                                 │
                                 └─────────────────────────────────┘
```

## Agent Spawn Pattern

```
                 Orchestrator
                      │
    ┌─────────────────┼─────────────────┐
    │                 │                 │
    ▼                 ▼                 ▼
┌─────────┐      ┌─────────┐      ┌─────────┐
│ Work    │      │ Verify  │      │ Doc     │
│ Agent   │──┐   │ Agent   │──┐   │ Agent   │
└─────────┘  │   └─────────┘  │   └─────────┘
             │                │
             ▼                ▼
        claude -p         claude -p
        "implement..."    "verify..."
             │                │
             ▼                ▼
     Documents/code/    Checks output
     [makes changes]    [validates]
             │                │
             ▼                ▼
     task_log.txt       task_log.txt
     + completed        + validation
```

## Conductor's Hybrid Architecture

Conductor achieves similar workflow through its plugin system:

```
User Request
    ↓
/cook-man (interactive) OR /cook-auto (autonomous)
    ↓
Requirements Gathering + Design Document
    ↓
/doc or /doc-yaml
    ↓
Implementation Plan (structured)
    ↓
conductor run plan.yaml
    ↓
Execution with QC + Adaptive Learning
```

## Key Architectural Differences

### AI-Corp: Event-Driven Daemon
- **Always-on**: Polls every 60 seconds
- **File-based**: Drop files → system reacts
- **Visual feedback**: Desktop overlay with PNG icons
- **Sequential**: One agent at a time
- **Consumer-friendly**: No CLI knowledge needed

### Conductor: On-Demand Executor
- **Invoked**: Explicit command execution
- **Plan-driven**: Structured execution plans
- **CLI feedback**: Console output with logs
- **Parallel**: Wave-based concurrent execution
- **Developer-focused**: Git-friendly, version controlled

## The Real Distinction

The workflow is similar (intake → design → plan → execute), but the **UX paradigm** differs:

**Conductor** = Developer-driven, session-based workflow
- You're in a Claude Code session
- You explicitly run `/cook-auto feature-request.md`
- You review the design
- You run `/doc design.md`
- You execute `conductor run plan.yaml`
- CLI-native experience

**AI-Corp** = Autonomous, always-on daemon
- System runs in background 24/7
- Stakeholders drop files without CLI knowledge
- System automatically processes without human trigger
- Visual feedback without opening terminal
- Consumer-friendly experience

---

**Key Insight**: Conductor's plugin system already provides the intelligent intake layer that AI-Corp implements. The `/cook-auto` command is conceptually doing what AI-Corp's requirements agent does—just invoked differently (on-demand vs. continuous monitoring).
