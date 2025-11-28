---
name: implementation-planner
description: Generate comprehensive implementation plans for features. Use when user requests "help me implement X", "create a plan for X", "break down feature X", "how should I build X", or asks for detailed implementation guidance. Activates for planning requests, not exploratory design discussions.
allowed-tools: Read, Bash, Glob, Grep, Write, TodoWrite
---

# Implementation Planner Skill

**Version:** 2.6.0
**Last Updated:** 2025-11-27
**Purpose:** Generate conductor-compatible YAML implementation plans with full codebase analysis, data flow tracing, dependency graphing, package conflict detection, and quality control.

## When to Activate

Activate when user requests **concrete, actionable implementation plan**:

- "Help me implement [feature]"
- "Create a plan for [feature]"
- "Break down [feature] into tasks"
- "Generate implementation plan for [feature]"

**Do NOT activate for:**
- Exploratory discussions ("What do you think about...")
- Questions ("How does X work?")
- Debugging ("Why isn't X working?")
- Code reviews

## Instructions

### Phase 1: Agent Discovery & Codebase Analysis

**1. Discover Available Agents**

```bash
fd '\.md$' ~/.claude/agents --type f
```

Extract agent names (remove path/extension):
- `/Users/.../golang-pro.md` → `golang-pro`
- Store for task assignment
- Default to `general-purpose` if none found

**2. Determine Project Type**

Use `Glob` with pattern `"*"`:
- Empty/minimal → Greenfield project (Phase 1B)
- Existing code → Full analysis

**3. Analyze Existing Codebase**

See `reference/codebase-analysis.md` for complete patterns.

**Quick analysis:**
```bash
# Structure
ls -la
Glob: "src/**/*", "tests/**/*"

# Stack
Read: package.json, pyproject.toml, go.mod

# Architecture
Grep: "class.*Controller", "func New", patterns

# Tests
Glob: "**/*.test.*", "**/*_test.go"
Read: sample test files

# Quality
Glob: ".eslintrc*", ".prettierrc*"
```

**Document findings:**
- Framework & language
- Testing framework
- Architecture pattern
- Code quality tools
- Similar features (for patterns)
- **File organization patterns (CRITICAL - see Section 6 in codebase-analysis.md)**

**CRITICAL: Before specifying file paths, verify they match codebase patterns:**
```bash
# Example: Check if migrations use separate files or inline
ls internal/learning/migrations/ 2>/dev/null  # Does dir exist?
grep -r "CREATE TABLE" internal/learning/     # Where does SQL live?
```
Do NOT assume directory structures - verify them first!

### Phase 2: Dependency Graph, Package Conflicts & Worktree Groups

After breaking feature into tasks:

**1. Build dependency graph** from `depends_on` fields

**2. Detect package-level conflicts (Go projects):**
- Parse task files to extract Go package paths
- Group tasks by package (e.g., `internal/behavioral/`)
- For tasks modifying the same package without explicit dependencies:
  - **Default strategy:** Add `depends_on` to serialize execution
  - **Alternative:** Scope test commands to specific files/functions
  - Document resolution in task metadata or comments

**3. Identify connected components:**
- Dependency chains: Tasks X→Y→Z (sequential)
- Independent tasks: No deps, no dependents (parallel)

**4. Create worktree groups (organizational only):**
- Group related tasks for documentation
- Suggest git workflow with setup_commands
- **NOT used for execution control** (conductor uses `depends_on`)

See `reference/dependency-graphing.md` and `reference/package-conflicts.md` for details.

**Example grouping:**
```yaml
worktree_groups:
  - group_id: "auth-chain"
    description: "Tasks 1→2→4 auth flow"
    tasks: [1, 2, 4]
    rationale: "Related auth tasks"
    # Git workflow suggestion:
    setup_commands: |
      git worktree add ../wt-auth -b feature/auth
```

### Phase 2.5: Data Flow Analysis (CRITICAL)

**MANDATORY PHASE** - Prevents the most common planning failure: missing data dependencies.

**Problem:** Feature chain thinking produces WRONG dependencies:
```
WRONG: Task 16 (CLI) in "CLI chain" → depends_on: [13] (root command)
RIGHT: Task 16 calls ExtractMetrics() → Task 4 creates it → depends_on: [4, 5, 13, 15]
```

**Execute BEFORE generating task YAML:**

**1. Extract function/type references from descriptions:**
```
"Export metrics" → calls ExtractMetrics()
"Load session data" → calls LoadSession()
"Display BehavioralMetrics" → uses BehavioralMetrics struct
```

**2. Build producer registry:**
```yaml
producers:
  ExtractMetrics: {task: 4, file: "extractor.go"}
  LoadSession: {task: 5, file: "aggregator.go"}
  BehavioralMetrics: {task: 1, file: "models.go"}
```

**3. Analyze each task's consumption:**
```yaml
task_16_consumes:
  - ExtractMetrics → producer: Task 4
  - LoadSession → producer: Task 5
  - ApplyFilters → producer: Task 15
  structural: Task 13 (registers as subcommand)
```

**4. Validate depends_on includes ALL producers:**
```
Task 16 calls ExtractMetrics (Task 4) → depends_on includes 4? ✓
Task 16 calls LoadSession (Task 5) → depends_on includes 5? ✓
```

**Add Data Flow Registry header to plan (YAML comment):**
```yaml
# ═══════════════════════════════════════════════════════════════════════════
# DATA FLOW REGISTRY
# ═══════════════════════════════════════════════════════════════════════════
# PRODUCERS: Task 4 → ExtractMetrics, Task 5 → LoadSession
# CONSUMERS: Task 16 → [4, 5, 15], Task 18 → [4, 5]
# VALIDATION: All consumers depend_on their producers ✓
# ═══════════════════════════════════════════════════════════════════════════
```

See `reference/data-flow-analysis.md` for complete methodology.

### Phase 2.6: Prompt Template Generation

**For tasks WITH dependencies, generate structured prompts:**

```yaml
description: |
  # Task N: [Name]

  ## PHASE 0: DEPENDENCY VERIFICATION (EXECUTE FIRST)
  ```bash
  grep -q "func ExtractMetrics" internal/behavioral/extractor.go && \
    echo "✓ Task 4 ready" || echo "❌ STOP: Task 4 incomplete"
  ```
  If ANY check fails: Report "Dependency Task X incomplete" and STOP.

  ## IMPLEMENTATION REQUIREMENTS
  **MUST call:** extractor.ExtractMetrics() - from Task 4
  **PROHIBITED:** TODO comments, placeholder structs, unused variables

  ## TASK DESCRIPTION
  [Actual task description]
```

**Auto-append anti-pattern criteria to ALL tasks:**
```yaml
success_criteria:
  # Task-specific criteria
  - "Functional requirement"
  # Auto-appended
  - "No TODO comments in production code paths"
  - "No placeholder empty structs (e.g., Type{})"
  - "No unused variables (_ = x pattern)"
  - "All imports from dependency tasks resolve"
```

See `reference/prompt-templates.md` for complete templates.

### Phase 2.7: Success Criteria Classification (CRITICAL)

**MANDATORY PHASE** - Prevents cross-task scope leakage that causes QC failures.

**Problem:** Mixing CAPABILITY and INTEGRATION criteria in component tasks:
```yaml
# BAD - Task 14 (Cache Layer) with mixed criteria:
success_criteria:
  - "Caches responses with configurable TTL"      # ✅ CAPABILITY - testable with cache files
  - "Cache can be bypassed with --no-cache flag"  # ❌ INTEGRATION - requires CLI files!
```

**Classify each success criterion:**

| Type | Definition | Belongs In | Test Method |
|------|------------|------------|-------------|
| **CAPABILITY** | What component CAN do | Component task | Unit test with task's files only |
| **INTEGRATION** | How components WORK TOGETHER | Integration task | E2E test across components |

**CAPABILITY criteria (component-scoped):**
- Testable using ONLY the task's `files` list
- Describes internal behavior
- Examples:
  - ✅ "CacheManager accepts `enabled: boolean` option"
  - ✅ "When enabled=false, get() returns null"
  - ✅ "Automatically evicts expired entries"

**INTEGRATION criteria (cross-component):**
- Requires multiple components working together
- Describes end-to-end behavior
- Examples:
  - ❌ "Cache can be bypassed with --no-cache flag" (cache + CLI)
  - ❌ "Forecast shows loading spinner" (engine + UI)
  - ❌ "API errors display user-friendly message" (API + error handler + CLI)

**Indicator keywords for INTEGRATION:**
- CLI: "flag", "--", "command", "argument", "CLI"
- UI: "displays", "shows", "renders", "UI"
- Cross-component: "when X then Y", "triggers", "calls [other component]"

**Validation rule:**
```
For each task where type != integration:
  For each success_criterion:
    1. Check for integration indicator keywords
    2. Ask: "Can this be verified using ONLY files in this task?"
    3. If NO → criterion is INTEGRATION, must be moved or task restructured
```

**Resolution when INTEGRATION criterion found in component task:**

1. **Split the criterion:**
   ```yaml
   # Original (BAD):
   - "Cache can be bypassed with --no-cache flag"

   # Split into:
   # Task 14 (Cache - CAPABILITY):
   - "CacheManager constructor accepts { enabled: boolean } option"
   - "When enabled=false, get() always returns null, set() is no-op"

   # Task 10/11 (CLI - INTEGRATION):
   - "CLI --no-cache flag passes enabled=false to CacheManager"
   ```

2. **Or create integration task:**
   ```yaml
   - task_number: N
     type: integration
     name: "Wire Cache to CLI"
     files:
       - "packages/cli/src/program.ts"
       - "packages/core/src/cache/index.ts"  # read-only reference
     depends_on: [10, 14]
     integration_criteria:
       - "CLI --no-cache flag disables cache via CacheManager.enabled"
   ```

3. **Or expand task scope:**
   ```yaml
   # Add CLI file to cache task (if appropriate):
   files:
     - "packages/core/src/cache/index.ts"
     - "packages/core/src/cache/file-cache.ts"
     - "packages/cli/src/program.ts"  # Added for --no-cache flag
   depends_on: [1, 3, 10]  # Add CLI setup dependency
   ```

**BEHAVIORAL vs IMPLEMENTATION-SPECIFIC (Critical):**

Success criteria describe WHAT, not HOW. Implementation section describes HOW.

```yaml
# BAD - Implementation-specific:
- "RenderTier enum with (RICH=4, FULL=3, STANDARD=2)"  # Specific values
- "RGB interface with readonly number fields"          # Specific modifier
- "tempToColor returns chalk.rgb()"                    # Specific library call

# GOOD - Behavioral:
- "RenderTier enum has 5 distinct capability tiers"
- "RGB interface has r, g, b number properties"
- "tempToColor returns function that applies color styling"
```

**Test:** Can the agent achieve this with a different but valid implementation? If no, too specific.

**CRITERIA DERIVATION FROM KEY_POINTS (Critical):**

Success criteria MUST be derived directly from `implementation.key_points`. Never write criteria independently.

**Process:**
```
For each key_point in implementation.key_points:
  → Write criterion that verifies THIS specific point
  → Use SAME terminology as key_point
  → Criterion = testable assertion of key_point
```

**Example:**
```yaml
# key_points written first:
key_points:
  - point: "Weather-themed spinner set"
    details: "sunny, rainy, windy, snowy, stormy, default"

# criteria derived from key_points (same words):
success_criteria:
  - "LOADING_SPINNERS has weather-themed spinners: sunny, rainy, windy, snowy, stormy, default"
```

```yaml
# BAD - criteria invented independently:
key_points:
  - point: "Weather-themed spinner set"
    details: "sunny, rainy, windy, snowy, stormy"
success_criteria:
  - "LOADING_SPINNERS has spinners for: fetch, aggregate, analyze"  # ❌ Different terminology!

# GOOD - criteria derived from key_points:
key_points:
  - point: "Weather-themed spinner set"
    details: "sunny, rainy, windy, snowy, stormy"
success_criteria:
  - "LOADING_SPINNERS has weather-themed spinners: sunny, rainy, windy, snowy, stormy"  # ✓ Same words
```

**Validation:** After writing success_criteria, verify each criterion references terminology from a key_point above it.

**BEHAVIORAL FACT VERIFICATION (Critical):**

Before writing key_points that make claims about existing behavior, verify against codebase:

**Claims that require verification:**
- Default values ("default format is X")
- Existing options ("command accepts --flag")
- Current behavior ("function returns X")
- Feature availability ("compare has --format option")

**Process:**
```
For each behavioral claim in key_points:
  → grep/read codebase to verify
  → Include verification command as comment
```

**Example:**
```yaml
# BEFORE writing key_points, verify:
# grep -n "format ??" packages/cli/src/commands/forecast.ts
# Result: options.format ?? "narrative"

key_points:
  - point: "Backward compatibility"
    details: "Default format stays 'narrative'"  # ✓ Verified
```

```yaml
# BAD - unverified assumption:
key_points:
  - point: "Backward compatibility"
    details: "Default format stays 'table'"  # ❌ Never checked

# BAD - claim about feature that doesn't exist:
success_criteria:
  - "compare command accepts --format rich"  # ❌ Never verified compare has --format
```

**Verification commands to run:**
```bash
# Default values
grep -n "??" <file> | grep <option>

# Option existence
grep -n "option\|flag\|--" <file>

# Function behavior
grep -A5 "function <name>" <file>
```

**Add to plan header comment:**
```yaml
# ═══════════════════════════════════════════════════════════════════════════
# SUCCESS CRITERIA CLASSIFICATION
# ═══════════════════════════════════════════════════════════════════════════
# All component tasks have CAPABILITY-only criteria ✓
# Integration criteria isolated to integration tasks ✓
# Criteria are BEHAVIORAL not implementation-specific ✓
# Criteria DERIVED from key_points (same terminology) ✓
# Behavioral claims VERIFIED against codebase ✓
# ═══════════════════════════════════════════════════════════════════════════
```

See `reference/success-criteria-classification.md` for complete methodology.

### Phase 3a: Multi-File Planning with Cross-File Dependencies

**When to split into multiple files:**
- Plan exceeds ~2000 lines
- Related feature modules that can be worked independently
- Sequential execution dependencies across multiple sections

**File tracking during generation:**
1. Assign each task to a file during planning phase
2. Record file mapping as you generate tasks
3. Track which files each task modifies
4. When a task depends on a task in a different file, use cross-file notation

**Cross-file dependency notation:**
```yaml
# In plan-02-integration.yaml, Task 5 depends on Task 2 from another file:
depends_on:
  - 4                    # Same file - numeric reference
  - file: "plan-01-foundation.yaml"  # Different file
    task: 2
```

See `reference/cross-file-dependencies.md` for complete patterns and examples.

### Phase 3b: Integration Task Detection

**Generate integration tasks when:**
- Multi-component plan (3+ modules)
- Wiring multiple completed components
- Cross-cutting concerns (auth, logging)

**Integration task structure:**
```yaml
- task_number: N
  type: integration  # REQUIRED
  success_criteria: [...]  # Component-level
  integration_criteria: [...] # Cross-component
```

See `reference/integration-tasks.md` for patterns.

### Phase 3c: Documentation Task Analysis

**For any task that updates documentation:**

1. **Trace source location** - grep codebase for feature being documented
2. **Identify parent context** - which command/section owns it
3. **Distinguish similar features** - explicitly note what it's NOT
4. **Specify exact doc location** - section name, not just file
5. **Create location-aware tests** - grep with context, not just existence

**Common failure pattern:**
```yaml
# BAD - Ambiguous
- "Document `--daemon` flag"
- test: "grep -q 'daemon' docs/CLI.md"  # Passes if anywhere in file

# GOOD - Location-specific
- "`--daemon` documented in `run` command section (NOT `start` command)"
- test: "grep -B5 'run command' docs/CLI.md | grep -q 'daemon'"
```

**Required for documentation tasks:**
- DOCUMENTATION TARGET ANALYSIS section in description
- CRITICAL DISTINCTION section when similar names exist
- Location-aware test commands (grep with context)

See `reference/documentation-tasks.md` for complete guidelines.

### Phase 4: YAML Plan Generation with Streaming

**CRITICAL:** Use TodoWrite before starting generation:

```
Create todo list:
- [ ] Create docs/plans directory
- [ ] Write YAML header & metadata
- [ ] Generate tasks (monitor line count)
- [ ] Split at ~2000 lines if needed
- [ ] Validate YAML
- [ ] Confirm completion
```

**Active line count monitoring:**

```
current_line_count = 0
total_tasks = <from analysis>

WHILE generating:
  Write section (header, task, etc)
  current_line_count += lines_written

  IF current_line_count > 1900 AND tasks_remaining:
    COMPLETE YAML structure
    VALIDATE syntax
    WRITE file: plan-0N-<phase>.yaml
    CREATE next file
    RESET counter

  IF all_tasks_written:
    WRITE final file
    BREAK
```

**Split files at:**
- ~2000 lines
- Worktree group boundaries
- Complete task boundaries (never mid-task)

See `reference/streaming-generation.md` for complete flow.

### Phase 5: YAML Structure

**Root structure:**
```yaml
conductor:
  default_agent: general-purpose

  worktree_groups:
    - group_id: "chain-1"
      tasks: [1, 2, 4]
      # ... setup commands

plan:
  metadata:
    feature_name: "Feature Name"
    created: "YYYY-MM-DD"

  context:
    framework: "Framework name"
    architecture: "Pattern"

  tasks:
    - task_number: 1
      name: "Task name"
      agent: "agent-name"
      files: ["file1.ext", "file2.ext"]  # FLAT LIST
      depends_on: []
      estimated_time: "30m"

      # Task-level fields (conductor parses)
      success_criteria:
        - "Criterion 1"
      test_commands:
        - "test command"

      # Documentation sections
      test_first: {...}

      # Implementation section (REQUIRED)
      implementation:
        approach: "Implementation strategy and decisions"
        code_structure: "File/class structure (optional)"
        key_points:
          - point: "Critical decision"
            details: "Why this matters"
            reference: "Optional reference"
        integration:  # For tasks with depends_on
          imports: []
          services_to_inject: []
          config_values: []
          error_handling: []

      verification: {...}
      code_quality:  # REQUIRED
        python:
          full_quality_pipeline:
            command: |
              black . && mypy src/ && pytest
      commit: {...}
```

See `reference/yaml-structure.md` for complete spec.

**Implementation section guidance:**

Generate `implementation:` for every task with prescriptive guidance:

**Minimum (all tasks):**
- `approach`: Strategy, architectural decisions
- `key_points`: 1-3 critical steps with `point`, `details`, `reference`

**Add `code_structure` when:**
- Multiple files (3+)
- Complex architecture
- Class hierarchies not obvious

**Add `integration` when:**
- Task has `depends_on` (wiring components)
- Needs imports/services from other tasks
- Cross-cutting concerns (auth, logging)

**Reference:** `templates/implementation-section.yaml` for examples

### Phase 6: Validation

**Before finalizing:**

1. **YAML syntax** - Must parse correctly
2. **Completeness:**
   - Every task has `success_criteria` (task-level)
   - Every task has `test_commands` (task-level)
   - Every task has `implementation` section with approach + key_points
   - Every task has `code_quality` pipeline
   - Integration tasks have BOTH criteria types
   - Every task has agent assigned
   - Files are flat lists (not nested)
3. **Dependencies:**
   - All numeric deps exist (same file)
   - All cross-file references are valid:
     - File exists in split plan directory
     - Target task exists in referenced file
     - Task number is valid
   - No circular dependencies across files
   - Worktree groups match dependency analysis
4. **Cross-file references (multi-file plans):**
   - Validate all `file:` references point to actual files
   - Verify task IDs referenced actually exist in target files
   - Check for circular cross-file dependencies
   - Ensure file names are consistent (use relative paths)
5. **No reserved fields:**
   - Don't use: `status`, `completed_at`, `git_commit`

**Validate files:**
```bash
# Single file
conductor validate docs/plans/<slug>.yaml

# Multi-file plan (validates all + cross-file refs)
conductor validate docs/plans/<feature-name>/
```

See `reference/cross-file-dependencies.md` for validation checklist.

### Phase 7: Output Confirmation

**Single file:**
```
YAML plan: docs/plans/<slug>.yaml
- Total tasks: 12
- Total lines: 1,543
- Worktree groups: 3
- YAML validation: PASSED
```

**Split files:**
```
YAML plan: docs/plans/<feature-name>/
- Total tasks: 25
- Files: 3
  - plan-01-foundation.yaml (2,023 lines, tasks 1-10)
  - plan-02-integration.yaml (1,987 lines, tasks 11-18)
  - plan-03-testing.yaml (1,156 lines, tasks 19-25)
- YAML validation: ALL PASSED

Run: conductor run docs/plans/<feature-name>/
```

## Supporting Files Reference

- `reference/yaml-structure.md` - Complete YAML template & field specs
- `reference/codebase-analysis.md` - Discovery patterns
- `reference/dependency-graphing.md` - Kahn's algorithm, worktree groups
- `reference/data-flow-analysis.md` - **Producer/consumer mapping, dependency validation**
- `reference/prompt-templates.md` - **Dependency verification, anti-pattern rules**
- `reference/package-conflicts.md` - Package-level conflict detection & resolution strategies
- `reference/cross-file-dependencies.md` - Cross-file syntax, validation, patterns
- `reference/integration-tasks.md` - Integration vs component patterns
- `reference/documentation-tasks.md` - **Location-aware doc tasks, disambiguation**
- `reference/success-criteria-classification.md` - **CAPABILITY vs INTEGRATION criteria scoping**
- `reference/streaming-generation.md` - Active line monitoring, split logic
- `reference/quality-gates.md` - Language-specific pipelines, pitfalls

- `templates/task-component.yaml` - Component task template (with data flow docs)
- `templates/task-integration.yaml` - Integration task template (with dependency verification)
- `templates/data-flow-registry.yaml` - **Data flow registry header template**
- `templates/cross-file-dependency.yaml` - Cross-file dependency examples
- `templates/worktree-group.yaml` - Worktree group structure

## Key Principles

1. **Data flow first** - **Trace function calls to producers before setting depends_on**
2. **Capability vs Integration** - **Success criteria must match task scope (Phase 2.7)**
3. **Test-first** - TDD mandatory, explain test design
4. **Active monitoring** - Check line count after every section
5. **Write incrementally** - Don't accumulate in memory
6. **Split at boundaries** - Worktree groups, complete tasks
7. **Validate each file** - Every file must parse
8. **Use TodoWrite** - Track generation progress
9. **Quality gates** - Every task needs language pipeline
10. **Integration detection** - Generate when wiring components
11. **Anti-patterns blocked** - Every task prohibits TODO, placeholders, unused vars

## Success Criteria

Skill successful when:
- ✅ YAML plan generated with all required fields
- ✅ Valid syntax (parseable)
- ✅ **Data flow analysis complete** - all consumers depend_on their producers
- ✅ **Success criteria classified** - CAPABILITY vs INTEGRATION properly scoped
- ✅ **Criteria derived from key_points** - same terminology, direct derivation
- ✅ **Behavioral claims verified** - existing behavior checked against codebase
- ✅ **Dependency verification in prompts** - tasks check deps before implementing
- ✅ **Anti-pattern criteria included** - no TODO, placeholders, unused vars
- ✅ Dependency graph correct (no wave conflicts)
- ✅ Worktree groups optimize parallelism
- ✅ Integration tasks detected and structured
- ✅ Quality pipelines included
- ✅ Large plans split correctly (2000-line limit)
- ✅ User can execute with conductor immediately

## Version History

### v2.6.0 (2025-11-27)
- **CRITICAL: Added criteria derivation from key_points requirement**
  - Success criteria MUST be derived directly from `implementation.key_points`
  - Enforces same terminology between key_points and criteria
  - Process: for each key_point → write criterion using same words
  - Validation: verify each criterion references terminology from key_point above it
- Fixes Weather Oracle Task 9 failure pattern where criteria used "fetch, aggregate, analyze" but key_points specified "weather-themed spinners"
- Added to plan header comment: "Criteria DERIVED from key_points (same terminology) ✓"
- **CRITICAL: Added behavioral fact verification requirement**
  - Claims about existing behavior must be verified against codebase before writing key_points
  - Applies to: default values, existing options, current behavior, feature availability
  - Process: grep/read codebase to verify, include verification command as comment
- Fixes Weather Oracle Task 11 failure pattern where "default format is table" was assumed (actual: "narrative") and "compare accepts --format" was assumed (compare has no --format option)
- Added to plan header comment: "Behavioral claims VERIFIED against codebase ✓"

### v2.5.0 (2025-11-26)
- **CRITICAL: Added success criteria classification phase (Phase 2.7)**
  - Prevents cross-task scope leakage (like Task 14 Cache requiring CLI flag)
  - Distinguishes CAPABILITY criteria (component-scoped) from INTEGRATION criteria (cross-component)
  - Validation rule: component tasks must have ONLY capability criteria
  - Integration criteria must be in integration tasks or task scope must be expanded
- **Indicator keywords for detecting integration criteria:**
  - CLI: "flag", "--", "command", "argument"
  - UI: "displays", "shows", "renders"
  - Cross-component: "when X then Y", "triggers", "calls [other component]"
- **Three resolution strategies when integration criterion found in component task:**
  1. Split criterion into capability (component) + integration (CLI/UI task)
  2. Create dedicated integration task
  3. Expand task file scope with proper dependencies
- New reference guide: success-criteria-classification.md
- New plan header comment: SUCCESS CRITERIA CLASSIFICATION validation
- Fixes Weather Oracle Task 14 failure pattern where cache task had CLI integration criterion

### v2.4.0 (2025-11-25)
- **Added documentation task analysis phase (Phase 3c)**
  - Prevents ambiguous documentation tasks (like `--daemon` flag vs `start` command)
  - Location-aware success criteria (section-specific, not file-wide)
  - Context-aware test commands (grep with -B/-A for location verification)
- **Required for documentation tasks:**
  - DOCUMENTATION TARGET ANALYSIS section tracing source location
  - CRITICAL DISTINCTION section when similar names exist
  - Location-aware test commands instead of simple existence checks
- New reference guide: documentation-tasks.md
- Fixes Task 10 failure pattern where agent confused similar feature names

### v2.3.0 (2025-11-24)
- **CRITICAL: Added data flow analysis phase (Phase 2.5)**
  - Prevents missing data dependency failures (like Task 16 in Agent Watch)
  - Producer/consumer registry for function/type tracking
  - Automatic depends_on validation against data flow
- **Added prompt templates with dependency verification (Phase 2.6)**
  - Phase 0 verification commands in task prompts
  - Agents verify dependencies exist before implementing
- **Auto-appended anti-pattern criteria to ALL tasks**
  - No TODO comments in production code
  - No placeholder empty structs
  - No unused variables
  - All imports from dependency tasks resolve
- New reference guides: data-flow-analysis.md, prompt-templates.md
- New template: data-flow-registry.yaml
- Updated task templates with data flow documentation

### v2.2.0 (2025-11-24)
- Added package-level conflict detection for Go projects
- Automatic serialization of same-package tasks to prevent race conditions
- New Phase 2 step: Detect package conflicts and apply resolution strategies
- New reference guide: package-conflicts.md
- Prevents test contamination during parallel execution

### v2.1.0 (2025-11-23)
- Added cross-file dependency support for split plans
- New Phase 3a for multi-file planning with file tracking
- Cross-file notation: `{file: "...", task: N}` format
- Enhanced validation for cross-file references
- New reference guide: cross-file-dependencies.md
- New template: cross-file-dependency.yaml
- Supports both same-file numeric and cross-file references

### v2.0.0 (2025-11-22)
- Full migration from slash command delegation
- Integrated generation logic with supporting files
- Added streaming generation with active monitoring
- Progressive loading via reference files
- Template-based task generation

### v1.0.0 (2025-11-10)
- Initial delegation-based wrapper
- Format selection UX
- SlashCommand integration
