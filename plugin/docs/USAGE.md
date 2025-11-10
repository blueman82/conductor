# Usage Guide - Conductor Tools Plugin

Complete guide to using the conductor-tools plugin for feature development workflows.

## Quick Start

The conductor-tools plugin provides a complete workflow:

1. **Design** → Create feature design (cook-auto or cook-man)
2. **Plan** → Generate implementation plan (doc or doc-yaml)
3. **Execute** → Run conductor to implement tasks
4. **Review** → Quality control and completion

## Commands

### /doc - Generate Markdown Plan

Generate a comprehensive implementation plan in Markdown format.

**When to use:**
- You have a clear feature idea
- Want human-readable documentation
- Plan will be read/edited manually

**Usage:**
```bash
claude "/doc Add user authentication with JWT tokens"
```

**Output:**
- Creates `docs/plans/user-authentication.md`
- Detailed tasks with TDD approach
- Worktree groups for parallel execution
- Testing strategy
- Commit strategy

**Example:**
```bash
# Navigate to your project
cd /path/to/my-project

# Generate plan
claude "/doc Add payment processing with Stripe integration"

# Result:
# docs/plans/payment-processing.md created
# - 18 tasks defined
# - 5 worktree groups
# - Est. time: 10 hours
```

### /doc-yaml - Generate YAML Plan

Generate a comprehensive implementation plan in YAML format.

**When to use:**
- Planning to use conductor for execution
- Want machine-readable format
- Need programmatic plan manipulation

**Usage:**
```bash
claude "/doc-yaml Add user authentication with JWT tokens"
```

**Output:**
- Creates `docs/plans/user-authentication.yaml`
- Structured YAML with all task details
- Compatible with conductor CLI
- Includes agent assignments

**Example:**
```bash
# Generate YAML plan
claude "/doc-yaml Implement GraphQL API with Apollo Server"

# Validate the plan
conductor validate docs/plans/graphql-api.yaml

# Execute with conductor
conductor run docs/plans/graphql-api.yaml --max-concurrency 3
```

### /cook-auto - Autonomous Design

AI-powered design session using multi-model deliberation.

**When to use:**
- Have a rough feature idea
- Want fast, comprehensive design
- Trust AI to make design decisions
- Time is limited

**Usage:**
```bash
claude "/cook-auto Build a notification system with email and push notifications"
```

**Process:**
1. Analyzes your codebase
2. Uses multiple AI models to deliberate on design questions
3. Generates comprehensive design document
4. Asks for approval
5. Automatically generates implementation plan

**Example:**
```bash
claude "/cook-auto Add real-time chat feature with WebSockets"

# AI models deliberate:
# - Claude Opus: Suggests WebSocket architecture
# - GPT-5: Recommends message persistence strategy
# - Gemini: Proposes scaling approach

# Design generated → User approves → Plan generated automatically
```

### /cook-man - Interactive Design

Interactive design session with user-guided refinement.

**When to use:**
- Have a vague idea to explore
- Want control over design decisions
- Prefer step-by-step refinement
- Learning the domain

**Usage:**
```bash
claude "/cook-man Create a dashboard for analytics"
```

**Process:**
1. Analyzes your codebase
2. Asks questions one at a time
3. Presents design sections incrementally
4. Gets approval on each section
5. Generates implementation plan after approval

**Example:**
```bash
claude "/cook-man Build a search feature"

# Claude asks:
# Q1: "What type of search? Full-text, filters, or both?"
# You: "Full-text with filters"

# Q2: "Which search engine? Elasticsearch, PostgreSQL full-text, or built-in?"
# You: "PostgreSQL full-text"

# ... more questions ...

# Claude presents design section by section
# You approve each section
# Final plan is generated
```

## Skills

### plan-executor - Execute Plans

Execute conductor plans with monitoring and reporting.

**When to use:**
- You have a validated YAML plan
- Ready to execute tasks
- Want progress monitoring and reporting

**Activation:**
```bash
# In Claude Code session
"Use the plan-executor skill to execute docs/plans/user-auth.yaml"
```

**What it does:**
1. Validates the plan
2. Runs dry-run preview
3. Asks for approval
4. Executes with conductor
5. Monitors progress
6. Reports results

**Example:**
```
User: "Execute the authentication plan"

plan-executor:
- Reads docs/plans/user-authentication.yaml
- Validates (✓ 12 tasks, no cycles)
- Previews execution (Wave 1: Tasks 1,3,7)
- Asks approval
- Executes: conductor run --max-concurrency 3
- Reports: 12/12 complete, 11 GREEN, 1 YELLOW
```

### task-implementer - Implement Tasks

Implement individual tasks following TDD principles.

**When to use:**
- Implementing specific task from plan
- Want guided TDD workflow
- Following plan specification

**Activation:**
```bash
# In Claude Code session
"Use the task-implementer skill to implement Task 5 from the auth plan"
```

**What it does:**
1. Reads task specification
2. Writes tests first (RED)
3. Implements to pass tests (GREEN)
4. Refactors for quality
5. Verifies all checks pass
6. Creates commit

**Example:**
```
User: "Implement Task 3: Add validation middleware"

task-implementer:
1. Reads task spec from plan
2. Creates test/middleware/validation_test.go
   - Writes failing tests
   - Runs tests (all fail - RED ✓)
3. Implements middleware/validation.go
   - Runs tests (all pass - GREEN ✓)
4. Refactors and adds docs
5. Runs: go test, go vet, go fmt
6. Commits: "feat: add input validation middleware"
```

### quality-reviewer - Review Quality

Review task implementation and provide quality verdict.

**When to use:**
- After task implementation
- Want comprehensive quality check
- Need GREEN/RED/YELLOW verdict

**Activation:**
```bash
# In Claude Code session
"Use the quality-reviewer skill to review Task 5"
```

**What it does:**
1. Reads task specification
2. Reviews code changes (git diff)
3. Runs tests and checks
4. Verifies requirements met
5. Provides structured verdict
6. Offers actionable feedback

**Verdicts:**
- **GREEN**: All good, proceed
- **RED**: Critical issues, must fix
- **YELLOW**: Minor issues, consider fixing

**Example:**
```
User: "Review the authentication middleware implementation"

quality-reviewer:
- Reads Task 5 spec
- Reviews commit abc123
- Runs tests: ✓ 15/15 pass
- Runs go vet: ✓ no issues
- Checks coverage: ✓ 92%
- Verifies requirements: ✓ all met
- Verdict: GREEN
  "Excellent implementation with comprehensive tests.
   Follows existing patterns. Ready to proceed."
```

## Agents

### conductor-orchestrator - Orchestrate Workflows

Complete workflow orchestration from design to deployment.

**When to use:**
- Want full end-to-end workflow
- Need guidance through process
- First time using conductor

**Usage:**
```bash
# Start Claude Code with this agent
claude -p "I'm the conductor-orchestrator agent" "Help me build user authentication"
```

**What it does:**
1. Understands your request
2. Guides you through design (cook-auto/cook-man)
3. Generates plan (doc-yaml)
4. Validates plan
5. Executes with conductor
6. Monitors progress
7. Reports completion

**Example:**
```
User: "Build a payment integration"

conductor-orchestrator:
"I'll guide you through building a payment integration.

Step 1: Design
Would you like autonomous design (cook-auto) or interactive (cook-man)?
[User chooses auto]

[Runs /cook-auto, gets approval]

Step 2: Generate Plan
[Runs /doc-yaml]
Plan created: docs/plans/payment-integration.yaml
- 20 tasks
- 6 worktree groups

Step 3: Validate
[Runs conductor validate]
✓ Plan is valid

Step 4: Execute
Ready to execute. Options:
1. Dry run first (recommended)
2. Full execution
[User chooses dry run, then full execution]

[Executes with monitoring]

Step 5: Complete
✓ 20/20 tasks complete
✓ All GREEN verdicts
✓ 20 commits created
Feature complete!"
```

### plan-validator - Validate Plans

Comprehensive plan validation before execution.

**When to use:**
- Before executing a plan
- After manually editing a plan
- Checking plan quality

**Usage:**
```bash
# Start Claude Code with this agent
claude -p "I'm the plan-validator agent" "Validate docs/plans/user-auth.yaml"
```

**What it does:**
1. Syntax validation (YAML/Markdown)
2. Dependency analysis (no cycles)
3. Agent assignment checks
4. Worktree group validation
5. Completeness checks
6. Best practices verification

**Example:**
```
User: "Validate the authentication plan"

plan-validator:
# Plan Validation Report

Plan: docs/plans/user-authentication.yaml
Status: ✗ FAIL - 3 issues found

## Issues

1. Task 7: Missing estimated_time field
   Fix: Add estimated_time: "30m"

2. Task 12: Agent "test-master" doesn't exist
   Fix: Change to "test-automator"

3. Task 8: Worktree group mismatch
   Task 8 in "chain-2" depends on Task 3 in "independent-1"
   Fix: Move Task 8 to "independent-1" or Task 3 to "chain-2"

## Recommendations

After fixing, re-run:
conductor validate docs/plans/user-authentication.yaml
```

## Complete Workflow Examples

### Example 1: New Feature from Scratch

```bash
# 1. Design the feature
claude "/cook-auto Add user profile page with avatar upload"

# AI models design the feature
# Design document created: docs/designs/user-profile-page.md
# User reviews and approves

# 2. Generate implementation plan (automatically after design approval)
# Plan created: docs/plans/user-profile-page.yaml
# - 15 tasks defined
# - 4 worktree groups

# 3. Validate the plan
conductor validate docs/plans/user-profile-page.yaml
# ✓ Plan is valid

# 4. Execute the plan
conductor run docs/plans/user-profile-page.yaml \
  --max-concurrency 3 \
  --verbose

# 5. Monitor execution
# Wave 1: Tasks 1, 4, 7 (parallel) - 100% complete
# Wave 2: Tasks 2, 5, 8 (parallel) - 100% complete
# ...
# All 15 tasks complete

# 6. Review results
# 15/15 complete
# 14 GREEN, 1 YELLOW
# Feature ready!
```

### Example 2: Execute Existing Plan

```bash
# You have a plan file already
ls docs/plans/
# payment-integration.yaml

# Validate before execution
conductor validate docs/plans/payment-integration.yaml
# ✓ Valid

# Preview with dry-run
conductor run docs/plans/payment-integration.yaml --dry-run
# Wave 1: Tasks 1, 3
# Wave 2: Tasks 2, 4, 5
# ...

# Execute for real
conductor run docs/plans/payment-integration.yaml \
  --max-concurrency 2 \
  --timeout 1h \
  --verbose

# Results:
# 18/18 tasks complete
# All GREEN
```

### Example 3: Manual Task-by-Task Implementation

```bash
# Start Claude Code session
claude

# Use task-implementer for each task
User: "Use task-implementer skill to implement Task 1 from docs/plans/auth.yaml"

# task-implementer:
# - Reads Task 1 spec
# - Writes tests
# - Implements code
# - Verifies and commits

User: "Now implement Task 2"

# Continue for all tasks...

# Or use orchestrator
User: "Use conductor-orchestrator to guide me through implementing the auth plan"

# orchestrator manages the workflow for you
```

### Example 4: Review and Fix Failed Task

```bash
# Execution failed on Task 5
conductor run docs/plans/plan.yaml
# ...
# Task 5: RED verdict - 3 tests failing

# Review what failed
claude "Use quality-reviewer skill to analyze Task 5 failure"

# quality-reviewer:
# Issues:
# 1. Missing error handling at line 23
# 2. Test expects 401, getting 500
# 3. No validation for nil input
#
# Recommendations:
# [specific fixes]

# Fix the issues
claude "Use task-implementer to fix Task 5 issues"

# Re-run conductor (it will skip completed tasks)
conductor run docs/plans/plan.yaml
# Task 5: GREEN ✓
```

### Example 5: Interactive Design Workflow

```bash
# Start interactive design
claude "/cook-man Build a recommendation engine"

# Claude asks questions one by one:
Q: "What type of recommendations? Content-based, collaborative filtering, or hybrid?"
A: "Collaborative filtering"

Q: "What data will you use? User interactions, ratings, purchases?"
A: "User ratings and purchases"

Q: "Where should recommendations display? Homepage, product pages, or both?"
A: "Both"

# ... more questions ...

# Claude presents design sections:
# Section 1: Architecture & Data Model
# [Shows design]
# Approve? Yes

# Section 2: Algorithm Approach
# [Shows design]
# Approve? Yes

# ... more sections ...

# Design complete!
# Generate plan? Yes

# Plan generated: docs/plans/recommendation-engine.yaml
```

## Tips and Best Practices

### Choosing Between Markdown and YAML Plans

**Use Markdown (`/doc`) when:**
- Plans are primarily read by humans
- You'll edit plans manually
- Rich documentation is important
- Not using conductor for execution

**Use YAML (`/doc-yaml`) when:**
- Planning to use conductor for execution
- Want programmatic plan manipulation
- Need structured, parseable format
- Building automation around plans

### Choosing Between Autonomous and Interactive Design

**Use Autonomous (`/cook-auto`) when:**
- Feature is well-understood
- Time is limited
- Comfortable with AI decisions
- Want comprehensive design quickly

**Use Interactive (`/cook-man`) when:**
- Feature is exploratory
- Want to guide design decisions
- Learning the problem space
- Prefer step-by-step approach

### Optimizing Conductor Execution

**Concurrency:**
```bash
# More parallelism (faster but uses more resources)
conductor run plan.yaml --max-concurrency 5

# Less parallelism (slower but safer)
conductor run plan.yaml --max-concurrency 1

# Recommended: 2-3 for most projects
conductor run plan.yaml --max-concurrency 3
```

**Timeouts:**
```bash
# Short timeout for quick tasks
conductor run plan.yaml --timeout 30m

# Long timeout for complex tasks
conductor run plan.yaml --timeout 2h

# Per-task timeout in plan (recommended)
# Set estimated_time in YAML for each task
```

**Dry Run First:**
```bash
# Always preview before executing
conductor run plan.yaml --dry-run

# Review wave groupings
# Check parallelization
# Then execute for real
conductor run plan.yaml
```

### Handling Failures

**RED Verdicts (Critical):**
```bash
# Review the failure
claude "Review Task 5 RED verdict"

# Fix the issues
claude "Fix Task 5 issues"

# Re-run (skips completed tasks)
conductor run plan.yaml
```

**YELLOW Verdicts (Warnings):**
```bash
# Decide: fix now or later?
# Option 1: Fix now
claude "Address YELLOW warnings in Task 7"

# Option 2: Continue, fix later
# Mark as tech debt
echo "Task 7: YELLOW - test coverage at 75%" >> TODO.md
```

### Resume Partial Execution

```bash
# Execution interrupted at Task 8?
# Conductor tracks completed tasks in plan file

# Check status
grep "status: completed" docs/plans/plan.yaml

# Resume (skips completed tasks)
conductor run docs/plans/plan.yaml

# Continues from Task 8
```

## Troubleshooting

See [SETUP.md](./SETUP.md#troubleshooting) for installation issues.

### Common Usage Issues

**Issue: Command not recognized**
```bash
# Solution: Check plugin is enabled
claude commands | grep doc

# If missing, activate plugin
# Add to ~/.claude/config.yaml:
plugins:
  enabled:
    - conductor-tools
```

**Issue: Conductor not found**
```bash
# Solution: Install conductor binary
cd /path/to/conductor
go build ./cmd/conductor
sudo mv conductor /usr/local/bin/
```

**Issue: Agent not found in plan**
```bash
# Solution: List available agents
fd '\.md$' ~/.claude/agents --type f

# Update plan to use available agents
# Or install missing agents
```

## Next Steps

- **Read Architecture**: [ARCHITECTURE.md](./ARCHITECTURE.md)
- **View Examples**: Check `templates/` directory
- **Try a Workflow**: Start with `/cook-auto`
- **Read Conductor Docs**: [Conductor README](https://github.com/harrison/conductor)

## Summary

The conductor-tools plugin provides:
- **4 commands** for design and planning
- **3 skills** for execution and quality
- **2 agents** for orchestration and validation
- **Complete workflows** from idea to implementation

Start with `/cook-auto` or `/doc-yaml` and let conductor handle the rest!
