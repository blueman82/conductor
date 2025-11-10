# Architecture - Conductor Tools Plugin

Understanding how the conductor-tools plugin works and integrates with Claude Code and Conductor.

## Overview

The conductor-tools plugin creates a complete workflow pipeline:

```
Design → Plan → Execute → Quality Check → Deploy
```

Each stage has dedicated tools (commands, skills, agents) that work together.

## Component Architecture

### High-Level Components

```
┌─────────────────────────────────────────────────────────┐
│              Claude Code CLI                            │
│  (Processes commands, invokes skills/agents)            │
└─────────┬───────────────────────────────────────────────┘
          │
          ↓
┌─────────────────────────────────────────────────────────┐
│         Conductor Tools Plugin                          │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Commands Layer                                   │  │
│  │  - /doc (Markdown plans)                          │  │
│  │  - /doc-yaml (YAML plans)                         │  │
│  │  - /cook-auto (Autonomous design)                 │  │
│  │  - /cook-man (Interactive design)                 │  │
│  └───────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Skills Layer                                     │  │
│  │  - plan-executor (Execute plans)                  │  │
│  │  - task-implementer (Implement tasks)             │  │
│  │  - quality-reviewer (QA checks)                   │  │
│  └───────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Agents Layer                                     │  │
│  │  - conductor-orchestrator (Full workflow)         │  │
│  │  - plan-validator (Plan validation)               │  │
│  └───────────────────────────────────────────────────┘  │
└─────────┬───────────────────────────────────────────────┘
          │
          ↓
┌─────────────────────────────────────────────────────────┐
│           Conductor CLI Binary                          │
│  - Parses plans (YAML/Markdown)                         │
│  - Calculates dependency graph                          │
│  - Executes tasks in waves                              │
│  - Quality control reviews                              │
│  - Updates plan files                                   │
└─────────────────────────────────────────────────────────┘
```

## Component Descriptions

### Commands

Commands are slash commands that extend Claude Code's functionality.

**Location**: `commands/*.md`

**Format**:
```markdown
---
allowed-tools: [list of tools command can use]
argument-hint: "description"
description: "what the command does"
---

# Command implementation
[Markdown content with instructions for Claude]
```

**How they work**:
1. User types `/doc feature description`
2. Claude Code reads `commands/doc.md`
3. Executes instructions in the file
4. Generates output (plan file)

**Commands in this plugin**:
- `doc.md` - Generate Markdown plan
- `doc-yaml.md` - Generate YAML plan
- `cook-auto.md` - Autonomous design with AI deliberation
- `cook-man.md` - Interactive design with user questions

### Skills

Skills are reusable capabilities that can be invoked during execution.

**Location**: `skills/[skill-name]/SKILL.md`

**Format**:
```markdown
---
name: skill-name
description: "what the skill does"
version: 1.0.0
allowed-tools: [list of tools]
---

# Skill implementation
[Detailed instructions for Claude]
```

**How they work**:
1. Skill is invoked by name
2. Claude Code loads skill instructions
3. Executes skill workflow
4. Returns result

**Skills in this plugin**:
- `plan-executor/` - Execute conductor plans
- `task-implementer/` - Implement individual tasks
- `quality-reviewer/` - Review and provide verdicts

### Agents

Agents are specialized personas with specific expertise and workflows.

**Location**: `agents/*.md`

**Format**:
```markdown
---
name: agent-name
description: "agent role"
version: 1.0.0
tools: [list of tools]
---

# Agent definition
[Role description, workflows, decision points]
```

**How they work**:
1. Agent is selected (via flag or reference in plan)
2. Claude Code adopts agent persona
3. Follows agent's workflows and decision logic
4. Acts according to agent expertise

**Agents in this plugin**:
- `conductor-orchestrator.md` - End-to-end workflow orchestration
- `plan-validator.md` - Plan validation specialist

## Integration Points

### Claude Code Integration

The plugin integrates with Claude Code through:

1. **PLUGIN.yaml manifest**
   - Declares all commands, skills, agents
   - Specifies version and dependencies
   - Defines configuration options

2. **Command registration**
   - Commands appear in `/` slash command menu
   - Auto-completion support
   - Help text from description field

3. **Skill registry**
   - Skills available via "use the [skill] skill" invocation
   - Can be referenced in plans for task execution

4. **Agent directory**
   - Agents available for selection
   - Can be assigned to tasks in plans

### Conductor Integration

The plugin generates plans that conductor consumes:

1. **Plan format**
   ```yaml
   plan:
     metadata:
       feature_name: "..."
     tasks:
       - task_number: 1
         name: "..."
         agent: "golang-pro"
         depends_on: []
         # ... task details
   ```

2. **Conductor reads plan**
   - Parses YAML or Markdown
   - Extracts tasks and dependencies
   - Builds dependency graph

3. **Conductor executes**
   - Groups tasks into waves
   - Spawns Claude Code agents for each task
   - Monitors execution
   - Updates plan file with results

4. **Plan updates**
   ```yaml
   - task_number: 1
     name: "..."
     status: "completed"          # Added by conductor
     completed_at: "2025-01-09"   # Added by conductor
     git_commit: "abc123..."      # Added by conductor
   ```

## Workflow Pipelines

### Design → Plan → Execute Pipeline

```
1. User Request
   └─→ "/cook-auto Add user authentication"
        │
        ↓
2. Design Phase (cook-auto command)
   ├─→ Analyze codebase
   ├─→ Multi-model deliberation
   ├─→ Generate design document
   └─→ Get user approval
        │
        ↓
3. Planning Phase (auto-invokes /doc-yaml)
   ├─→ Read design document
   ├─→ Discover available agents
   ├─→ Analyze dependencies
   ├─→ Generate YAML plan
   └─→ Validate plan
        │
        ↓
4. Execution Phase (conductor run)
   ├─→ Parse plan file
   ├─→ Build dependency graph
   ├─→ Calculate waves
   ├─→ Execute tasks in parallel
   │   └─→ For each task:
   │        ├─→ Spawn agent
   │        ├─→ Implement task
   │        ├─→ Run tests
   │        ├─→ Create commit
   │        └─→ Quality review
   └─→ Update plan file
        │
        ↓
5. Completion
   └─→ All tasks complete
        └─→ Feature ready!
```

### Task Implementation Pipeline

```
1. Task Assigned (from conductor)
   └─→ task-implementer skill activated
        │
        ↓
2. Read Task Spec
   ├─→ Parse task from plan YAML
   ├─→ Extract requirements
   ├─→ Identify files to change
   └─→ Note test requirements
        │
        ↓
3. TDD Red Phase
   ├─→ Create/update test file
   ├─→ Write failing tests
   └─→ Run tests (verify they fail)
        │
        ↓
4. TDD Green Phase
   ├─→ Implement code
   ├─→ Run tests iteratively
   └─→ All tests pass
        │
        ↓
5. TDD Refactor Phase
   ├─→ Improve code quality
   ├─→ Remove duplication
   ├─→ Add documentation
   └─→ Keep tests passing
        │
        ↓
6. Verification
   ├─→ Run full test suite
   ├─→ Run linter
   ├─→ Run formatter
   └─→ Check requirements met
        │
        ↓
7. Commit
   └─→ Create atomic commit
        │
        ↓
8. Quality Review (quality-reviewer skill)
   ├─→ Review changes
   ├─→ Check test coverage
   ├─→ Verify requirements
   └─→ Return verdict (GREEN/RED/YELLOW)
        │
        ↓
9. Result
   ├─→ If GREEN: Task complete
   ├─→ If YELLOW: Task complete with warnings
   └─→ If RED: Retry or fix
```

## Data Flow

### Plan File Lifecycle

```
Initial State (after /doc-yaml):
┌─────────────────────────────────────┐
│ plan.yaml                           │
│                                     │
│ tasks:                              │
│   - task_number: 1                  │
│     name: "Create user model"       │
│     agent: "golang-pro"             │
│     depends_on: []                  │
│     estimated_time: "30m"           │
│     # No status field yet           │
│     commit:                         │
│       type: "feat"                  │
│       message: "add user model"     │
└─────────────────────────────────────┘
        │
        ↓ (conductor run)
        │
During Execution:
┌─────────────────────────────────────┐
│ plan.yaml (updated by conductor)    │
│                                     │
│ tasks:                              │
│   - task_number: 1                  │
│     name: "Create user model"       │
│     agent: "golang-pro"             │
│     status: "in_progress"   ← Added │
│     # ... rest unchanged             │
└─────────────────────────────────────┘
        │
        ↓ (task completes)
        │
After Completion:
┌─────────────────────────────────────┐
│ plan.yaml (final state)             │
│                                     │
│ tasks:                              │
│   - task_number: 1                  │
│     name: "Create user model"       │
│     agent: "golang-pro"             │
│     status: "completed"     ← Updated│
│     completed_at: "2025-01-09" ← Added│
│     git_commit: "abc123..." ← Added │
│     commit:                         │
│       type: "feat"                  │
│       message: "add user model"     │
└─────────────────────────────────────┘
```

### Quality Control Flow

```
Task Implementation Complete
        │
        ↓
┌─────────────────────────────────────┐
│ Quality Review Triggered            │
│ (quality-reviewer skill)            │
└─────────┬───────────────────────────┘
          │
          ↓
┌─────────────────────────────────────┐
│ Review Process                      │
│ 1. Read task specification          │
│ 2. Review git commit                │
│ 3. Run tests                        │
│ 4. Run linter/formatter             │
│ 5. Check requirements               │
└─────────┬───────────────────────────┘
          │
          ↓
     Decision Tree
          │
    ┌─────┴─────┐
    │           │
Tests pass?    NO → RED verdict → Retry task
    │
   YES
    │
    ↓
Requirements met?
    │
    ├─ NO → RED verdict → Retry task
    │
   YES
    │
    ↓
Coverage adequate?
    │
    ├─ NO → YELLOW verdict → Warning, continue
    │
   YES
    │
    ↓
Minor issues?
    │
    ├─ YES → YELLOW verdict → Warning, continue
    │
    NO
    │
    ↓
GREEN verdict → Task complete
```

## Extension Points

### Adding New Commands

1. Create `commands/my-command.md`:
```markdown
---
allowed-tools: Read, Write, Bash
argument-hint: "description"
description: "What my command does"
---

# My Command

Instructions for Claude...
```

2. Register in `PLUGIN.yaml`:
```yaml
commands:
  - name: my-command
    file: commands/my-command.md
    description: "What my command does"
```

3. Use it:
```bash
claude "/my-command arguments"
```

### Adding New Skills

1. Create `skills/my-skill/SKILL.md`:
```markdown
---
name: my-skill
description: "What my skill does"
version: 1.0.0
allowed-tools: [Read, Write]
---

# My Skill

Skill instructions...
```

2. Register in `PLUGIN.yaml`:
```yaml
skills:
  - name: my-skill
    directory: skills/my-skill
    description: "What my skill does"
```

3. Use it:
```bash
claude "Use the my-skill skill to ..."
```

### Adding New Agents

1. Create `agents/my-agent.md`:
```markdown
---
name: my-agent
description: "Agent role"
version: 1.0.0
tools: [Read, Write, Bash]
---

# My Agent

Agent definition...
```

2. Register in `PLUGIN.yaml`:
```yaml
agents:
  - name: my-agent
    file: agents/my-agent.md
    description: "Agent role"
```

3. Use it:
```bash
# Reference in plans
agent: "my-agent"

# Or invoke directly
claude -p "I'm the my-agent agent" "Task description"
```

## Design Principles

### 1. Separation of Concerns

Each component has a single responsibility:
- **Commands**: User-facing workflows (design, plan)
- **Skills**: Reusable execution capabilities
- **Agents**: Specialized personas and orchestration

### 2. Composability

Components work independently and together:
- Commands can invoke other commands
- Skills can be chained
- Agents can use any skill or command

### 3. Automation with Approval

Automate the repetitive, seek approval for decisions:
- Automatic: codebase analysis, test execution
- Approval: design decisions, plan execution

### 4. TDD by Default

Test-driven development is mandatory:
- All tasks specify tests first
- Implementation follows tests
- Quality checks verify tests exist and pass

### 5. Quality Control

Every task goes through quality review:
- GREEN: Proceed
- YELLOW: Warning, but continue
- RED: Must fix

## Performance Considerations

### Parallel Execution

Conductor executes tasks in waves based on dependencies:

```
Wave 1: Tasks 1, 4, 7 (no dependencies)
  → Execute in parallel (3 concurrent agents)

Wave 2: Tasks 2, 5, 8 (depend on Wave 1)
  → Wait for Wave 1 to complete
  → Execute in parallel

Wave 3: Task 3 (depends on Tasks 2 and 5)
  → Wait for dependencies
  → Execute
```

### Resource Management

- **Max concurrency**: Limit parallel tasks (default: 1, recommended: 2-3)
- **Timeout**: Per-task timeout to prevent hangs
- **Memory**: Each agent runs in separate Claude Code process

### Caching

- **Plan parsing**: Parsed once, cached for execution
- **Agent discovery**: Scanned once at startup
- **Test results**: Not cached (always re-run for accuracy)

## Security Considerations

### Safe Operations

Commands and skills use restricted tool sets:
- Read-only operations for analysis
- Write operations require approval
- No network access (local only)

### Secrets Management

- Never log passwords or tokens
- Never commit secrets to plan files
- Use environment variables for sensitive config

### Code Execution

- All code execution is local
- No remote code execution
- User controls when code runs

## Troubleshooting

### Debug Mode

Enable verbose logging:
```bash
# Conductor with verbose output
conductor run plan.yaml --verbose

# Claude Code with debug output
CLAUDE_DEBUG=1 claude "/doc feature"
```

### Common Issues

**Plan parsing fails**:
- Check YAML syntax
- Validate with: `conductor validate plan.yaml`

**Task execution hangs**:
- Check timeout settings
- Increase with: `--timeout 2h`

**Quality review fails**:
- Check test coverage
- Review git diff for changes
- Verify requirements in task spec

## Summary

The conductor-tools plugin provides:
- **Modular architecture** with commands, skills, and agents
- **Complete workflow** from design to deployment
- **Integration** with Claude Code and Conductor
- **Extensible** design for adding new components
- **Quality-focused** with mandatory TDD and reviews

The architecture enables autonomous feature development while maintaining quality and providing visibility at every step.
