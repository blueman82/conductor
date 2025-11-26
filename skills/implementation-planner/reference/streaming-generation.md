# Streaming Generation Strategy

Large plans (16+ tasks) require incremental generation to avoid token limits. This is ACTIVE MONITORING, not passive checking.

## Core Principle

**Write incrementally, don't accumulate.** Generate→check→write→repeat, not generate-all→write-once.

## Active Line Count Monitoring

### Initialize Tracking

```
current_file_line_count = 0
current_file_number = 1
tasks_written = 0
total_tasks = <from dependency analysis>
```

### During Generation Loop

```
WHILE generating plan:
  Write next section (header, task, support section)
  current_file_line_count += lines_just_written

  IF current_file_line_count > 1900 AND tasks_written < total_tasks:
    IF at worktree_group boundary:
      STOP current file
      COMPLETE YAML structure (close lists/objects)
      VALIDATE YAML is parseable
      WRITE file: plan-0N-<phase>.yaml
      CREATE next file: plan-0(N+1)-<next-phase>.yaml
      RESET current_file_line_count = 0
      CONTINUE with remaining content

  IF tasks_written == total_tasks:
    COMPLETE final YAML
    VALIDATE YAML
    WRITE final file
    BREAK
```

## TodoWrite Integration

Before generation, create todo list:

```
- [ ] Create docs/plans directory
- [ ] Write YAML header & metadata (~200-400 lines)
- [ ] Generate tasks for worktree group 1 (monitor actively)
- [ ] Check line count - split if >1900
- [ ] Generate tasks for worktree group 2
- [ ] Generate support sections
- [ ] Validate all YAML files
- [ ] Confirm completion
```

## Split Criteria

**Trigger:** `line_count > 1900 AND tasks_remaining > 0`

**Split at:**
- Worktree group boundaries (natural organization)
- After complete task (never mid-task)
- Valid YAML structure point

**File naming:**
```
docs/plans/<feature-slug>/
├── plan-01-foundation.yaml
├── plan-02-integration.yaml
└── plan-03-testing.yaml
```

## Example: 25-Task Generation

```
[plan-01-foundation.yaml]
Line 50: YAML header & metadata
Line 150: Context section
Line 300: Worktree groups
Line 400: Prerequisites
Line 570: Task 1 (current: 570)
Line 740: Task 2 (current: 740)
Line 910: Task 3 (current: 910)
Line 1080: Task 4 (current: 1080)
Line 1250: Task 5 (current: 1250)
Line 1420: Task 6 (current: 1420)
Line 1590: Task 7 (current: 1590)
Line 1760: Task 8 (current: 1760)
Line 1930: Task 9 (current: 1930) ⚠️ SPLIT NOW

COMPLETE YAML structure
VALIDATE plan-01-foundation.yaml
CREATE plan-02-integration.yaml

[plan-02-integration.yaml]
Line 30: Metadata (reset to 30)
Line 200: Task 10
...continues
```

## Validation Per File

Each file MUST:
1. Parse as valid YAML
2. Have complete structure (no truncated objects)
3. Include metadata header
4. Close all lists/objects properly

## Key Principles

1. **Active not passive** - Check after every section
2. **Write as you go** - Don't accumulate in memory
3. **Split at 1900** - Not 3000 when it's too late
4. **Task boundaries** - Never split mid-task
5. **Validate each** - Every file independently parseable
6. **Todo tracking** - Update progress as you generate

## Output Confirmation

```
YAML plan created in: docs/plans/<feature-name>/
- Total tasks: 25
- Plan files: 3 (Phase 2A format)
  - plan-01-foundation.yaml (2,023 lines, tasks 1-10)
  - plan-02-integration.yaml (1,987 lines, tasks 11-18)
  - plan-03-testing.yaml (1,156 lines, tasks 19-25)
- YAML validation: ALL FILES PASSED
```
