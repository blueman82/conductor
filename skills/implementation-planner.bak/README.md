# Implementation Planner Skill

> Generate conductor-compatible YAML implementation plans with data flow analysis, dependency verification, and quality control

## What It Does

The Implementation Planner Skill generates comprehensive YAML plans for the [Conductor](https://github.com/blueman82/conductor) multi-agent orchestration system. It automatically:

1. **Discovers available agents** from `~/.claude/agents/`
2. **Analyzes your codebase** - stack, architecture, patterns
3. **Traces data flow** - maps function/type producers and consumers
4. **Generates dependency graph** - prevents missing dependency failures
5. **Creates structured prompts** - agents verify dependencies before coding
6. **Outputs conductor-compatible YAML** - ready to execute

## Quick Start

```
You: Help me implement user authentication

Claude: [Analyzes codebase, discovers agents, generates plan]

Result: docs/plans/user-authentication.yaml
        Ready to run: conductor run docs/plans/user-authentication.yaml
```

## When It Activates

**Keywords that trigger this skill:**
- "Help me implement [feature]"
- "Create a plan for [feature]"
- "Break down [feature] into tasks"
- "Generate implementation plan for [feature]"

**Does NOT activate for:**
- Exploratory discussions ("What do you think about...")
- Questions ("How does X work?")
- Debugging ("Why isn't X working?")
- Code reviews

## Key Features (v2.4.0)

### Data Flow Analysis (Prevents Task Failures)

The skill traces function calls to ensure correct dependencies:

```
BEFORE: Task 16 "Export" in "CLI chain" → depends_on: [13]
        Result: RED - ExtractMetrics() doesn't exist yet

AFTER:  Task 16 calls ExtractMetrics() → produced by Task 4
        depends_on: [4, 5, 13, 15]
        Result: GREEN - all dependencies available
```

### Dependency Verification in Prompts

Generated task prompts include verification commands:

```yaml
description: |
  ## PHASE 0: DEPENDENCY VERIFICATION (EXECUTE FIRST)

  ```bash
  grep -q "func ExtractMetrics" internal/behavioral/extractor.go && \
    echo "✓ Task 4 ready" || echo "❌ STOP: Task 4 incomplete"
  ```

  If ANY check fails: Report "Dependency incomplete" and STOP.
```

### Anti-Pattern Detection

Every task includes criteria that QC validates:

```yaml
success_criteria:
  - "Functional requirement here"
  # Auto-appended to ALL tasks:
  - "No TODO comments in production code paths"
  - "No placeholder empty structs (e.g., Type{})"
  - "No unused variables (_ = x pattern)"
  - "All imports from dependency tasks resolve"
```

### Documentation Task Analysis (v2.4.0)

Prevents ambiguous documentation tasks with location-aware criteria:

```yaml
# BAD - Agent confused --with-ingest flag with observe ingest command
- "Document --with-ingest flag"
- test: "grep -q 'with-ingest' docs/FILE.md"

# GOOD - Location-specific, unambiguous
- "--with-ingest documented in observe stream section (NOT observe ingest)"
- test: "grep -B5 'observe stream' docs/FILE.md | grep -q 'with-ingest'"
```

### Multi-File Plans

Large plans auto-split at ~2000 lines with cross-file dependency support:

```yaml
depends_on:
  - 4                              # Same file
  - file: "plan-01-foundation.yaml"  # Cross-file
    task: 2
```

### Integration Task Detection

Automatically generates integration tasks when wiring components:

```yaml
- task_number: 8
  type: integration
  success_criteria: [...]      # Component-level
  integration_criteria: [...]  # Cross-component validation
```

## Output Structure

```
docs/plans/
├── feature-name.yaml           # Single file plan
└── feature-name/               # Multi-file plan
    ├── plan-01-foundation.yaml
    ├── plan-02-integration.yaml
    └── plan-03-testing.yaml
```

## Execution

```bash
# Validate plan
conductor validate docs/plans/feature.yaml

# Execute plan
conductor run docs/plans/feature.yaml --verbose

# Resume interrupted plan
conductor run docs/plans/feature.yaml --skip-completed
```

## File Structure

```
~/.claude/skills/implementation-planner/
├── SKILL.md                    # Main skill configuration
├── README.md                   # This file
├── reference/
│   ├── data-flow-analysis.md   # Producer/consumer methodology
│   ├── prompt-templates.md     # Dependency verification templates
│   ├── yaml-structure.md       # Complete YAML spec
│   ├── dependency-graphing.md  # Kahn's algorithm, waves
│   ├── integration-tasks.md    # Component vs integration
│   ├── cross-file-dependencies.md
│   ├── package-conflicts.md
│   ├── codebase-analysis.md
│   ├── streaming-generation.md
│   └── quality-gates.md
├── templates/
│   ├── task-component.yaml     # Component task template
│   ├── task-integration.yaml   # Integration task template
│   ├── data-flow-registry.yaml # Registry header template
│   ├── cross-file-dependency.yaml
│   ├── worktree-group.yaml
│   └── implementation-section.yaml
└── examples/
    └── [example plans]
```

## Planning Phases

1. **Phase 1**: Agent discovery & codebase analysis
2. **Phase 2**: Dependency graph & package conflict detection
3. **Phase 2.5**: **Data flow analysis** (producer/consumer mapping)
4. **Phase 2.6**: **Prompt template generation** (dependency verification)
5. **Phase 3a**: Multi-file planning (if needed)
6. **Phase 3b**: Integration task detection
7. **Phase 4**: YAML generation with streaming
8. **Phase 5**: Structure validation
9. **Phase 6**: Output confirmation

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 2.3.0 | 2025-11-24 | Data flow analysis, prompt templates, anti-pattern criteria |
| 2.2.0 | 2025-11-24 | Package conflict detection for Go projects |
| 2.1.0 | 2025-11-23 | Cross-file dependency support |
| 2.0.0 | 2025-11-22 | Full implementation (replaced delegation model) |
| 1.0.0 | 2025-11-10 | Initial delegation-based wrapper |

## Compatibility

- **Claude Code**: 0.5+
- **Conductor**: v2.3+ (for integration tasks), v2.5+ (for dual criteria)
- **Output**: Standard YAML, conductor-compatible

## Related Tools

- `conductor` - Executes generated plans
- `/doc-yaml` - Alias (delegates to this skill)
- `/cook-man` - Interactive design before planning
- `/cook-auto` - AI deliberation before planning
