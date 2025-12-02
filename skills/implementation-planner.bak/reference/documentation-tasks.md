# Documentation Task Guidelines

## Problem

Documentation tasks frequently fail because:
1. **Ambiguous location** - "Document X" doesn't specify WHERE in docs
2. **Feature vs flag confusion** - Similar names cause wrong interpretation
3. **Weak test commands** - `grep -q 'term'` passes if term exists ANYWHERE

## Example Failure

```yaml
# BAD - Ambiguous
description: |
  - Document `--daemon` flag

success_criteria:
  - "--daemon flag documented"

test_commands:
  - "grep -q 'daemon' docs/CLI.md"
```

**Result:** Agent documented `start` command's daemon mode, thought `--daemon` flag on `run` was covered.

## Solution: Location-Aware Documentation

### Phase 1: Trace Documentation Target

Before generating documentation tasks, trace WHERE the item belongs:

```bash
# Find where --daemon is defined
grep -n "daemon" internal/cmd/*.go
# Result: run.go:85 - flag on NewRunCmd()
```

**Mapping:**
- Flag on `run` command → Document in "Run Command" section
- Flag on `start` command → Document in "Start Command" section
- New command → Document in new section + add to command list

### Phase 2: Generate Explicit Task Description

```yaml
# GOOD - Location-specific
description: |
  ## Documentation Target Analysis
  - `--daemon` is a FLAG on the `run` COMMAND (run.go:85)
  - NOT related to `start` command's daemon mode (different feature)
  - Must be documented in the RUN section of CLI.md

  ## Required Updates
  1. Add `--daemon` to run command's flags table
  2. Add usage example: `myapp run --daemon`
  3. Explain: "Runs process in background daemon mode"
```

### Phase 3: Context-Aware Success Criteria

```yaml
success_criteria:
  # Specific location requirement
  - "`--daemon` documented in run command section (NOT start command)"

  # Usage example present
  - "Example shows: myapp run --daemon"

  # Explanation present
  - "Documents that flag runs process in background"

test_commands:
  # Location-aware grep (flag near "run" context)
  - "grep -B5 -A5 'run command' docs/CLI.md | grep -q 'daemon'"

  # Or section-specific
  - "sed -n '/## Run/,/## /p' docs/CLI.md | grep -q 'daemon'"
```

## Documentation Task Template

```yaml
- task_number: N
  name: "Document [FEATURE] in [LOCATION]"
  agent: "technical-writer"
  files:
    - "docs/TARGET.md"
  description: |
    # Task N: Documentation for [FEATURE]

    ## DOCUMENTATION TARGET ANALYSIS
    Source location: `path/to/file.go:LINE`
    Feature type: [command | flag | config | API]
    Parent context: [which command/section it belongs to]

    ## CRITICAL DISTINCTION
    - [FEATURE] is a [FLAG on COMMAND X], NOT [related but different thing]
    - Must go in [SECTION Y], not [SECTION Z]

    ## REQUIRED UPDATES
    1. Add to [specific section/table]
    2. Include usage example showing exact syntax
    3. Explain what it does and when to use it

  success_criteria:
    - "[FEATURE] documented in [SPECIFIC SECTION] of [FILE]"
    - "Usage example: [exact command syntax]"
    - "Explanation covers: [key points]"
    - "NOT confused with [similar feature]"

  test_commands:
    # Location-aware test
    - "sed -n '/## [SECTION]/,/## /p' docs/FILE.md | grep -q '[FEATURE]'"
    # Or context grep
    - "grep -B3 -A3 '[PARENT CONTEXT]' docs/FILE.md | grep -q '[FEATURE]'"
```

## Checklist for Documentation Tasks

Before generating a documentation task:

- [ ] **Traced source location** - grep for feature in codebase
- [ ] **Identified parent context** - which command/section owns it
- [ ] **Distinguished from similar features** - explicitly note what it's NOT
- [ ] **Specified exact doc location** - section name, not just file
- [ ] **Created location-aware test** - grep with context, not just existence
- [ ] **Added usage example requirement** - exact syntax to document

## Common Confusions to Prevent

| Feature A | Feature B | How to Distinguish |
|-----------|-----------|-------------------|
| `--daemon` flag on run | daemon mode on start | Different commands, different behaviors |
| `--watch` on build | `--watch` on serve | Check which commands support which |
| `--verbose` | `--debug` | Check which commands support which |
| `export` command | `--export` flag | Command vs flag on another command |

Always include a "CRITICAL DISTINCTION" section when similar names exist.
