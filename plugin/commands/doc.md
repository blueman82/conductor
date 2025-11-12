---
allowed-tools: Read, Bash(ls*), Glob, Grep, Bash(git status*), Bash(git log*), Bash(fd*), Write
argument-hint: "feature description"
description: Generate comprehensive implementation plan with detailed tasks, testing strategy, and commit points
---

# Doc - Comprehensive Implementation Plan Generator

Create a detailed, step-by-step implementation plan for $ARGUMENTS. The plan should enable a skilled engineer with zero context about this codebase to successfully implement the feature.

## Phase 1: Codebase Context Analysis

**First, discover available agents:**

**Execute agent discovery first:**

Run the command:
```bash
fd '\.md$' ~/.claude/agents --type f
```

Extract agent names by removing the path and `.md` extension. For example:
- `/Users/harrison/.claude/agents/golang-pro.md` → `golang-pro`
- `/Users/harrison/.claude/agents/test-automator.md` → `test-automator`

Store these agent names for assignment to tasks. If no agents are found, use `general-purpose` as the default.

**Then, determine if this is an existing project or greenfield:**

Use `Glob` tool with pattern `"*"` to check files in the current directory. If empty or only contains minimal files (README, .git), this is a **greenfield project** - proceed to Phase 1B.

If an existing codebase exists, thoroughly analyze it:

1. **Project Structure Analysis**
   - Use `Bash` with `ls -la` to understand directory structure
   - Use `Glob` tool to find specific file patterns (e.g., `"**/*.test.*"`, `"src/**/*.js"`)
   - Identify key directories: src/, tests/, docs/, config/, etc.
   - Locate test directories and understand testing patterns

2. **Technology Stack Discovery**
   - Use `Read` tool to examine package.json, requirements.txt, Gemfile, go.mod, or similar
   - Identify frameworks and libraries in use
   - Check tsconfig.json, .eslintrc, pytest.ini, or other config files

3. **Architecture Pattern Recognition**
   - Use `Grep` tool to search for existing similar features
   - Identify architectural patterns (MVC, Clean Architecture, etc.)
   - Find service layers, repositories, controllers, models
   - Understand dependency injection patterns if present

4. **Testing Infrastructure**
   - Use `Glob` tool to locate test files and understand naming conventions
   - Use `Read` tool to examine test files and identify test frameworks (Jest, PyTest, RSpec, etc.)
   - Find test utilities, fixtures, factories, mocks
   - Review existing test examples for patterns

5. **Code Quality Standards**
   - Use `Glob` tool to check for linters, formatters (.eslintrc, .prettierrc, etc.)
   - Use `Read` and `Grep` tools to review existing code for naming conventions
   - Identify error handling patterns
   - Note logging/observability patterns

6. **Git History Context**
   - Run `git log --oneline -20` to see recent commit patterns
   - Note commit message style and size
   - Use `Glob` or `Read` to check if there's a CONTRIBUTING.md or similar

## Phase 1B: Greenfield Project Setup (if applicable)

If no existing codebase detected, the plan must include initial project setup:

1. **Technology Stack Decisions**
   - Document language/framework choice and rationale
   - List required dependencies and their purposes
   - Specify project structure to create
   - Define testing framework and approach

2. **Initial Setup Tasks**
   - Directory structure to create (src/, tests/, docs/, etc.)
   - Config files needed (package.json, tsconfig.json, pytest.ini, etc.)
   - Testing framework setup and configuration
   - Linting/formatting setup (.eslintrc, .prettierrc, etc.)
   - CI/CD configuration (if applicable)

3. **Foundation Commits**
   - First commit: Initial project scaffold with basic directory structure
   - Second commit: Configuration files and tooling setup
   - Third commit: README and documentation structure with project overview

4. **Development Environment**
   - Required tools and versions to install
   - Environment variables needed
   - Setup verification steps

## Phase 1C: Analyze Task Dependency Structure

After identifying all tasks for the feature, analyze the dependency relationships to determine optimal worktree grouping:

1. **Build Dependency Graph**
   - For each task, note any "Depends on" relationships
   - Create a directed graph where edges represent dependencies (Task A → Task B means B depends on A)

2. **Identify Connected Components (Dependency Chains)**
   - Find groups of tasks where dependencies create a chain or network
   - Example: Task 1 → Task 2 → Task 3 forms a single connected component
   - Example: Task 4 → Task 5 ← Task 6 (where Task 5 depends on both 4 and 6) forms another component
   - **Each connected component will share ONE worktree** to maintain dependency order

3. **Identify Independent Tasks**
   - Find tasks with no "Depends on" field AND no other tasks that depend on them
   - These tasks can be executed in parallel without conflict
   - **Each independent task gets its OWN worktree** to enable maximum parallelism

4. **Assign Worktree Groups**
   - Connected components: `chain-1`, `chain-2`, `chain-3`, etc.
   - Independent tasks: `independent-1`, `independent-2`, `independent-3`, etc.
   - Each group will have its own branch: `feature/<feature-slug>/[group-id]`

**Worktree Strategy Benefits**:
- **Parallelism**: Independent tasks execute simultaneously in separate worktrees
- **Dependency Safety**: Tasks in a chain execute sequentially in shared worktree, preventing dependency violations
- **Resource Efficiency**: Minimize worktree count by grouping dependent tasks
- **Clear Isolation**: Each worktree group has its own branch, preventing merge conflicts during parallel development

## Phase 2: Plan Generation

Create a comprehensive implementation plan in `docs/plans/<feature-slug>.md` where `<feature-slug>` is derived from the feature description (e.g., "user-authentication", "payment-integration").

The plan MUST include these sections:

## Phase 2A: Objective Plan Splitting

When generating task breakdowns, implement **metric-based plan splitting** to prevent files from becoming unwieldy while maintaining complete task details.

### Line Count Tracking

Continuously track the number of lines written to the current plan file as you generate task details. This is a precise, objective metric that requires no subjective judgment.

**Implementation:**
- Initialize `line_count = 0` when starting a new plan file
- Increment `line_count` for each line written to the plan
- Track `tasks_detailed` (number of tasks fully written) vs `total_tasks` (total tasks identified)

### Trigger Condition (Objective, Metric-Based)

Split the plan into a new file when ALL of these conditions are met:

```
IF (
  line_count > 2000
  AND tasks_detailed < total_tasks
  AND next_worktree_group_exists
):
  THEN split_plan_to_new_file(next_group)
```

**Critical rules:**
- **2000 lines is the hard limit** - no subjective "feels too long" judgment
- **Never split mid-task** - always complete the current task's full detail before splitting
- **Split at worktree group boundaries** - natural organizational boundaries for splitting
- **Only split if work remains** - don't create empty files

### Split Strategy

When the trigger condition is met, create numbered plan files organized by worktree group:

```
docs/plans/feature-name/
├── 1-chain-1.md          (2000 lines, tasks in chain-1 worktree group)
├── 2-chain-2.md          (1800 lines, tasks in chain-2 worktree group)
├── 3-independent.md      (1200 lines, independent tasks)
└── README.md             (index with cross-references to all plan files)
```

**File naming convention:**
- `N-[worktree-group-id].md` where N is the sequence number (1, 2, 3, etc.)
- The worktree group ID makes it clear which tasks are in which file
- Example: `1-chain-1.md`, `2-independent-tasks.md`, `3-chain-2.md`

### Metrics Tracked During Generation

Track these objective metrics throughout plan generation:

1. **line_count**: Total lines written to current plan file
2. **tasks_completed**: Number of tasks fully detailed in current file
3. **total_tasks**: Total number of tasks identified in Phase 1C
4. **current_worktree_group**: Which group is currently being documented
5. **worktree_group_boundaries**: List of group transitions (when to potentially split)

### Decision Logic (Pure Metrics-Based)

```
WHILE generating task breakdowns:
  - Write task details to current plan file
  - Increment line_count for each line written
  - When task is complete, increment tasks_completed

  IF line_count > 2000 AND tasks_completed < total_tasks:
    - Check if current task is last in current worktree group
    - IF yes (natural boundary exists):
      - STOP current plan file
      - Create new plan file for next worktree group
      - Reset line_count = 0 for new file
      - CONTINUE with remaining tasks in new file
    - IF no (mid-group):
      - CONTINUE in current file until group boundary reached
      - THEN split at next group boundary

  IF tasks_completed == total_tasks:
    - COMPLETE current plan file
    - Create README.md index if multiple files exist
    - BREAK while loop
```

**Key principle:** No subjective judgment - only objective line counts and task boundaries.

### Cross-Reference Index (README.md)

When plan splitting occurs, create a `docs/plans/<feature-name>/README.md` index file:

```markdown
# Implementation Plan: [Feature Name]

**Created**: [Date]
**Total Tasks**: [N]
**Total Plan Files**: [M]

This implementation plan is split across multiple files for maintainability.

## Plan Files (In Order)

1. **[1-chain-1.md](./1-chain-1.md)** (Tasks 1-8)
   - Worktree group: chain-1
   - Tasks: Core authentication flow
   - Line count: ~2000

2. **[2-chain-2.md](./2-chain-2.md)** (Tasks 9-14)
   - Worktree group: chain-2
   - Tasks: API integration layer
   - Line count: ~1800

3. **[3-independent.md](./3-independent.md)** (Tasks 15-20)
   - Worktree group: independent tasks
   - Tasks: Documentation, tests, cleanup
   - Line count: ~1200

## Getting Started

1. Start with [1-chain-1.md](./1-chain-1.md) - Task 1
2. Follow task dependencies as documented
3. Refer to each file's worktree group for parallel execution guidance

## Overview

[Brief 2-3 sentence summary of the entire feature and what the plan accomplishes]
```

### Output Confirmation

When plan generation completes, report to the user:

**Single file output:**
```
Plan created: docs/plans/<feature-slug>.md
- Total tasks: 12
- Total lines: 1,543
- Worktree groups: 3 (chain-1, independent-1, independent-2)
```

**Split file output:**
```
Plan created in: docs/plans/<feature-name>/
- Total tasks: 25
- Plan files created: 3
  - 1-chain-1.md (2,023 lines, tasks 1-10)
  - 2-chain-2.md (1,987 lines, tasks 11-18)
  - 3-independent.md (1,156 lines, tasks 19-25)
- Index: docs/plans/<feature-name>/README.md
- Worktree groups: 5 (chain-1, chain-2, independent-1, independent-2, independent-3)

Start with: docs/plans/<feature-name>/1-chain-1.md - Task 1
```

### Why This Approach Works

1. **Objective triggers**: No guessing - split happens at 2000 lines, period
2. **Complete tasks**: Never truncate task details across files
3. **Natural boundaries**: Worktree groups provide logical split points
4. **Maintainable**: Each file remains readable and navigable
5. **Discoverable**: README index makes multi-file plans easy to understand
6. **Consistent**: Metric-based approach produces repeatable results

### 1. Implementation Plan Header

```markdown
# Implementation Plan: [Feature Name]

**Created**: [Date]
**Target**: [Brief description of what we're building]
**Estimated Tasks**: [Number]

## Context for the Engineer

You are implementing this feature in a codebase that:
- Uses [framework/language]
- Follows [architecture pattern]
- Tests with [test framework]
- [Other critical context]

**You are expected to**:
- Write tests BEFORE implementation (TDD)
- Commit frequently (after each completed task)
- Follow existing code patterns
- Keep changes minimal (YAGNI - You Aren't Gonna Need It)
- Avoid duplication (DRY - Don't Repeat Yourself)
```

### 2. Worktree Groups Section

Add this section immediately after the header and before "Prerequisites Checklist":

```markdown
## Worktree Groups

This plan identifies task groupings for parallel execution using git worktrees. Each group operates in isolation with its own branch.

**Group chain-[N]**: Dependency Chain [N]
- **Tasks**: Task X, Task Y, Task Z (executed sequentially in dependency order)
- **Branch**: `feature/[feature-slug]/chain-[N]`
- **Execution**: Sequential (dependencies enforced)
- **Isolation**: Separate worktree from other groups
- **Rationale**: These tasks have dependencies that require sequential execution

**Group independent-[N]**: Independent Task [N]
- **Tasks**: Task N only
- **Branch**: `feature/[feature-slug]/independent-[N]`
- **Execution**: Parallel-safe (no dependencies)
- **Isolation**: Separate worktree from other groups
- **Rationale**: This task has no dependencies and can run in parallel with other work

[Repeat for each worktree group identified in Phase 1C]

### Worktree Management

**Creating worktrees**:
```bash
# For each group, create a worktree
git worktree add ../[repo-name]-[group-id] -b feature/[feature-slug]/[group-id]
```

**Switching between worktrees**:
```bash
cd ../[repo-name]-[group-id]
```

**Cleaning up after merge**:
```bash
git worktree remove ../[repo-name]-[group-id]
git branch -d feature/[feature-slug]/[group-id]
```
```

### 3. Prerequisites Checklist

List what to verify before starting:
- Required tools installed
- Development environment setup
- Access to necessary services/APIs
- Branch created from correct base

### 4. Detailed Task Breakdown

For EACH task, provide:

```markdown
## Task [N]: [Task Name]

**Agent**: `agent-name` (required - reference to agent from ~/.claude/agents/)
**File(s)**: `path/to/file.ext`
**Depends on**: [Task specification - if applicable]
  - Single dependency: `Task 1`
  - Multiple dependencies: `Task 1, Task 2, Task 3` (comma-separated, list each individually)
  - **IMPORTANT**: Range notation (e.g., `Tasks 1-3`) is NOT supported - always list task numbers individually
  - Omit this field entirely if no dependencies
**WorktreeGroup**: [group-id] (e.g., "chain-1", "independent-3")
**Estimated time**: [5m/15m/30m/1h]

**Agent Assignment Guidelines:**

Based on the task type and technology stack, assign appropriate agents:
- Go code: `golang-pro`
- Python code: `python-pro`
- JavaScript/TypeScript: `javascript-pro`, `typescript-pro`
- Testing tasks: `test-automator`
- Documentation: `technical-documentation-specialist`
- Database work: `database-optimizer` or `database-admin`
- Performance: `performance-engineer`
- General tasks: `general-purpose`

Always verify the agent exists in the discovered agent list. If uncertain, use `general-purpose`.

### What you're building
[2-3 sentences explaining WHAT and WHY]

### Test First (TDD)

**Test file**: `path/to/test_file.ext`

**Test structure**:
```[language]
describe/test block for [feature]
  - test case 1: [specific behavior]
  - test case 2: [specific edge case]
  - test case 3: [error condition]
```

**Test specifics**:
- Mock these dependencies: [list]
- Use these fixtures/factories: [list]
- Assert these outcomes: [list]
- Edge cases to cover: [list]

**Example test skeleton**:
```[language]
[Provide actual test code structure following project patterns]
```

### Implementation

**Approach**:
[Detailed explanation of how to implement]

**Code structure**:
```[language]
[Provide skeleton/pseudocode showing structure]
```

**Key points**:
- Follow pattern from: `path/to/similar/file.ext:line`
- Use utility: `utilityName` from `path/to/utility.ext`
- Error handling: [specific approach]
- Validation: [what to validate and how]

**Integration points**:
- Imports needed: [list]
- Services to inject: [list]
- Config values: [list]

### Verification

**Manual testing**:
1. [Step-by-step manual test]
2. [How to verify it works]

**Automated tests**:
```bash
[Exact command to run tests]
```

**Expected output**:
[What success looks like]

### Commit

**Commit message**:
```
[type]: [clear description]

[Optional body explaining why]
```

**Files to commit**:
- `path/to/implementation.ext`
- `path/to/test.ext`
```

### 5. Testing Strategy Section

Include a dedicated section:

```markdown
## Testing Strategy

### Unit Tests
- **Location**: [directory]
- **Naming**: [convention]
- **Run command**: `[command]`
- **Coverage target**: [X%]

### Integration Tests
- **Location**: [directory]
- **What to test**: [integration points]
- **Setup required**: [fixtures/test data]

### E2E Tests (if applicable)
- **Location**: [directory]
- **Critical flows**: [list]

### Test Design Principles for This Feature

**Use these patterns**:
1. [Pattern 1 with example]
2. [Pattern 2 with example]

**Avoid these anti-patterns**:
1. [Anti-pattern 1 and why]
2. [Anti-pattern 2 and why]

**Mocking guidelines**:
- Mock external services: [list]
- Don't mock: [list]
- Use project's mocking pattern: [reference to existing test]
```

### 6. Commit Strategy

```markdown
## Commit Strategy

Break this work into [N] commits following this sequence:

1. **[Commit type]**: [Description]
   - Files: [list]
   - Why separate: [reason]

2. **[Commit type]**: [Description]
   - Files: [list]
   - Why separate: [reason]

[Continue...]

**Commit message format**:
Follow the pattern seen in recent commits:
[Show example from git log]
```

### 7. Common Pitfalls

```markdown
## Common Pitfalls & How to Avoid Them

1. **[Pitfall 1]**
   - Why it happens: [explanation]
   - How to avoid: [solution]
   - Reference: [similar code that does it right]

2. **[Pitfall 2]**
   - Why it happens: [explanation]
   - How to avoid: [solution]
   - Reference: [similar code that does it right]
```

### 8. Resources Section

```markdown
## Resources & References

### Existing Code to Reference
- Similar feature: `path/to/similar.ext`
- Utility functions: `path/to/utils.ext`
- Test examples: `path/to/test_example.ext`

### Documentation
- Framework docs: [link if applicable]
- Internal docs: [path to relevant docs]
- API specs: [path if applicable]

### Validation
- [ ] All tests pass
- [ ] Linter passes
- [ ] Formatted correctly
- [ ] No console.log/print statements left
- [ ] Error handling in place
- [ ] Edge cases covered
```

## Phase 3: Plan Review

After generating the plan:

1. **Validate completeness**:
   - Every task has test-first approach
   - Every task has clear file paths
   - Every task has verification steps
   - Every task has worktree group assignment
   - Commit points are logical and frequent

2. **Ensure beginner-friendliness**:
   - No assumed knowledge about the codebase
   - Test design is explicitly explained
   - Patterns are referenced with file paths
   - Common mistakes are called out
   - Worktree grouping rationale is clear

3. **Verify DRY/YAGNI/TDD principles**:
   - Tests are written first for each task
   - No unnecessary abstractions suggested
   - Code reuse is explicitly called out
   - Frequent commits are mandated

4. **Validate worktree grouping**:
   - Dependency chains are correctly identified
   - Independent tasks are properly isolated
   - Group assignments enable maximum parallelism
   - Branch naming is consistent and clear

## Output Format

1. Write the complete plan to `docs/plans/<feature-slug>.md`
2. Confirm to the user:
   - Plan location
   - Number of tasks
   - Number of worktree groups (chains vs independent)
   - Parallelism opportunities identified
   - Estimated time
   - First task to start with

## Important Guidelines

- **Be specific**: Use actual file paths, not placeholders
- **Be concrete**: Provide code skeletons, not just descriptions
- **Be practical**: Reference existing code extensively
- **Be educational**: Explain the "why" behind decisions
- **Be thorough**: Assume zero codebase knowledge
- **Be test-focused**: TDD is mandatory, explain test design clearly
- **Be commit-focused**: Make commit strategy explicit
- **Be agent-aware**: Assign the most appropriate agent from the discovered list to each task based on the work type. Every task MUST have an agent assigned.
- **Be dependency-aware**: Analyze task dependencies and assign worktree groups correctly
- **Be parallelism-conscious**: Maximize concurrent execution while respecting dependencies

The goal is that an engineer can follow this plan mechanically and produce high-quality, well-tested code that fits the existing codebase perfectly, with the ability to work on independent tasks in parallel using separate worktrees.
