# Cross-File Dependencies Update Summary

**Version:** 2.1.0
**Updated:** 2025-11-23
**Status:** Complete

## Overview

The implementation-planner skill has been enhanced to support cross-file dependencies in split plans. This allows tasks in one plan file to explicitly depend on tasks in other files, enabling proper coordination of multi-file implementations.

## What Changed

### 1. SKILL.md - Main Skill Document

**Phase 3a: Multi-File Planning with Cross-File Dependencies**
- New phase added before integration task detection
- Explains when to split plans into multiple files
- Documents file tracking methodology
- Shows cross-file dependency notation with examples
- Links to reference guide

**Phase 6: Validation - Enhanced**
- Added cross-file reference validation checks
- New validation rules for multi-file plans
- Cross-file reference format validation
- File existence and task existence checks
- Circular dependency detection across files
- Updated conductor validate examples (both single and multi-file)

**Supporting Files Reference**
- Added `reference/cross-file-dependencies.md` to list
- Added `templates/cross-file-dependency.yaml` to list

**Version History**
- Added v2.1.0 entry documenting cross-file support
- Marked as major feature addition

### 2. New Reference: cross-file-dependencies.md

**Comprehensive guide covering:**

**Syntax Section**
- Same-file dependencies (numeric notation: `- 2`)
- Cross-file dependencies (object notation: `{file: "...", task: N}`)
- Mixed dependencies (both types in single task)
- File reference format rules

**Complete Examples**
- Two-file backend plan with execution flow
- Three-file microservices with multiple files
- Demonstrates wave calculation across files

**Validation Checklist**
- Syntax validation (format, paths, extensions, task numbers)
- Reference validation (file existence, task existence, typos, cycles)
- Logical validation (architecture, ordering, circularity)
- File organization validation

**Best Practices**
- When to use cross-file dependencies
- When NOT to use them
- DO patterns: Phase transitions, microservice coordination, infrastructure deps
- DON'T patterns: Same-file deps, weak relationships, circular patterns

**Conductor Validation**
- Single file validation command
- Multi-file validation command
- Specific checks performed for each

**File Organization Examples**
- Well-organized example (database → auth → api → deployment)
- Poorly-organized example (too many cross-file deps)

**Troubleshooting**
- Error: File not found (solution)
- Error: Task number does not exist (solution)
- Error: Circular dependency (solution)
- Warning: Forward reference (solution)

### 3. New Template: cross-file-dependency.yaml

**Complete working template showing:**

**5 Dependency Patterns**
1. Same-file dependency only (numeric notation)
2. Cross-file dependency only (object notation)
3. Mixed dependencies (both types)
4. Multiple cross-file dependencies (from multiple files)
5. Integration task with cross-file dependencies

**Each pattern includes:**
- Task definition with depends_on field
- Example values
- Explanation of the pattern
- Success criteria and test commands

**Reference Patterns**
- Pattern A: Foundation + Feature Chain
- Pattern B: Parallel + Sync
- Pattern C: Cascading Dependencies
- Diagram showing execution flow for each

**Validation Checklist**
- Syntax validation items
- Reference validation items
- Logical validation items
- File organization items

## Key Features

### Syntax Support

**Same-file (existing):**
```yaml
depends_on:
  - 2
  - 3
```

**Cross-file (new):**
```yaml
depends_on:
  - file: "plan-01-foundation.yaml"
    task: 2
```

**Mixed (new):**
```yaml
depends_on:
  - 4                                    # Same file
  - file: "plan-01-foundation.yaml"      # Different file
    task: 2
```

### File Tracking for Skill

When generating split plans, implementation-planner should:

1. **During planning phase:**
   - Assign each task to a file (foundation, integration, testing, etc.)
   - Record file mapping as you generate tasks
   - Track which files each task modifies

2. **During dependency resolution:**
   - Build complete dependency graph across all files
   - When a task depends on a task in a different file:
     - Use cross-file notation: `{file: "...", task: N}`
   - When a task depends on a task in the same file:
     - Use numeric notation: `- 2`

3. **During validation:**
   - Verify all files exist in split plan directory
   - Verify all referenced tasks exist in their files
   - Detect circular dependencies (including cross-file cycles)
   - Ensure no forward references

## Validation Enhancements

Conductor can now validate multi-file plans with cross-file references:

```bash
# Single file
conductor validate docs/plans/feature.yaml

# Multi-file
conductor validate docs/plans/feature-name/
```

Validation checks (new in multi-file mode):
- Each file parses as valid YAML
- All numeric references exist within files
- All cross-file references:
  - File exists
  - Target task exists in file
  - No forward references
  - No circular dependencies

## Usage Examples

### Two-File Plan with Cross-Deps

```
docs/plans/ecommerce/
├── plan-01-foundation.yaml  (Tasks 1-5: DB, cache, sessions)
└── plan-02-integration.yaml (Tasks 6-10: API, auth service)
```

**plan-02-integration.yaml:**
```yaml
- task_number: 6
  name: "Implement Auth Service"
  depends_on:
    - file: "plan-01-foundation.yaml"
      task: 3    # Session table from foundation
    - file: "plan-01-foundation.yaml"
      task: 5    # JWT setup from foundation
```

### Three-File Plan with Multiple Cross-Deps

```
docs/plans/microservices/
├── plan-01-auth-service.yaml
├── plan-02-api-service.yaml
└── plan-03-deployment.yaml
```

**plan-03-deployment.yaml:**
```yaml
- task_number: 9
  name: "Deploy Services"
  depends_on:
    - file: "plan-01-auth-service.yaml"
      task: 3    # Auth ready
    - file: "plan-02-api-service.yaml"
      task: 7    # API ready
    - 8          # Local task
```

## Benefits

1. **Clear Architecture** - File boundaries reflect system architecture
2. **Parallel Development** - Teams can work on different files independently
3. **Explicit Coordination** - Cross-file deps show integration points
4. **Validation Ready** - Conductor validates all references
5. **Documentation** - File structure documents how components integrate

## No Breaking Changes

- Single-file plans work exactly as before
- Numeric notation for same-file deps unchanged
- All existing templates and references still valid
- v2.1.0 is fully backward compatible

## Files Created/Modified

### Created:
1. `/Users/harrison/.claude/skills/implementation-planner/reference/cross-file-dependencies.md` (10.8 KB)
2. `/Users/harrison/.claude/skills/implementation-planner/templates/cross-file-dependency.yaml` (6.6 KB)

### Modified:
1. `/Users/harrison/.claude/skills/implementation-planner/SKILL.md`
   - Added Phase 3a: Multi-File Planning
   - Enhanced Phase 6: Validation
   - Updated Supporting Files Reference
   - Updated Version History

## Next Steps for Implementation-Planner Usage

When generating split plans:

1. **Analyze codebase** - Identify natural file boundaries
2. **Assign tasks to files** - Track which tasks go in which file
3. **Build dependency graph** - Across all files
4. **Generate cross-file notation** - Where dependencies cross files
5. **Validate completely** - All files + cross-file references
6. **Document organization** - Clear file naming and structure

## Testing Cross-File Dependencies

To test a generated split plan:

```bash
# Validate all files and cross-file refs
conductor validate docs/plans/feature-name/

# Dry-run to check execution plan
conductor run docs/plans/feature-name/ --dry-run

# Execute the full plan
conductor run docs/plans/feature-name/
```

Conductor will:
1. Load all files in directory
2. Merge plans together
3. Validate all dependencies (same-file and cross-file)
4. Calculate execution waves across all files
5. Execute with proper dependency ordering
