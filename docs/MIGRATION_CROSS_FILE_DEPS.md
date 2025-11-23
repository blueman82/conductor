# Migration Guide: Cross-File Dependencies (v2.5)

This guide helps you migrate existing multi-file plans to use explicit cross-file dependency notation introduced in Conductor v2.5.

## Table of Contents

- [Why Migrate?](#why-migrate)
- [Migration Levels](#migration-levels)
- [Step-by-Step Migration](#step-by-step-migration)
- [Before and After Examples](#before-and-after-examples)
- [Common Patterns](#common-patterns)
- [Testing Your Migration](#testing-your-migration)
- [Troubleshooting](#troubleshooting)
- [Rollback Plan](#rollback-plan)

---

## Why Migrate?

### Current State (v2.4)

Multi-file plans rely on **implicit ordering**:

```bash
# Order matters - foundation must come before services
conductor run foundation.yaml services.yaml integration.yaml
```

**Issues with implicit ordering:**
- Dependencies are not explicit (hidden in file order)
- Breaks if files are passed in different order
- No self-documenting dependency relationships
- Resume operations (--skip-completed) may miss cross-file dependencies
- Hard to understand which file depends on which

### Migrated State (v2.5+)

Explicit cross-file dependencies:

```yaml
# services.yaml explicitly declares its dependencies
tasks:
  - id: 3
    name: Auth Service
    depends_on:
      - file: foundation.yaml
        task: 1   # Crystal clear: needs foundation task 1
```

**Benefits of migration:**
- Self-documenting code
- Works in any file order
- Better error messages when dependencies fail
- Resumable execution tracks correctly
- Easier to understand complex plans

---

## Migration Levels

Choose the level that matches your needs:

### Level 1: No Migration Required (Quickest)

**Best for:** Simple plans, small teams, no complexity

Keep existing multi-file setup unchanged. Conductor v2.5+ is fully backward compatible.

```bash
# Keep doing this - still works perfectly
conductor run file1.yaml file2.yaml file3.yaml
```

**Time Required:** 0 minutes

**When to Choose:**
- Your plans work fine with implicit ordering
- You have only 2-3 files with obvious ordering
- You're not hitting resume/completion tracking issues

---

### Level 2: Partial Migration (Recommended)

**Best for:** Growing teams, moderate complexity, future-proofing

Add explicit cross-file dependencies to new and modified files only. Leave stable files unchanged.

**Time Required:** 5-15 minutes per file (depending on complexity)

**When to Choose:**
- You're actively developing plans
- You want to start documenting dependencies
- You want better error messages
- You plan to grow to many files

**How to Implement:**
1. When editing a file, add cross-file dependencies
2. Keep existing files that don't need changes
3. Both formats work together seamlessly

---

### Level 3: Complete Migration (Recommended for Large Plans)

**Best for:** Large implementations, complex microservices, enterprise

Migrate all files to use explicit cross-file dependencies.

**Time Required:** 30-60 minutes for comprehensive multi-file plan

**When to Choose:**
- You have 5+ plan files
- Multiple teams working on plans
- Complex inter-service dependencies
- Plans are frequently modified

**How to Implement:**
Follow the step-by-step guide below.

---

## Step-by-Step Migration

### Phase 1: Preparation (5 minutes)

**Step 1.1: Backup Existing Plans**
```bash
# Create a backup directory
mkdir -p backups
cp *.yaml backups/
cp *.md backups/

# Or use git (recommended)
git add *.yaml *.md
git commit -m "backup: save plans before migration"
```

**Step 1.2: Document Current File Order**
```bash
# List files in execution order
# Create a text file documenting why this order exists

cat > FILE_ORDER.txt << 'EOF'
Execution order for multi-file plans:

1. foundation.yaml
   - Initializes core infrastructure (database, cache)
   - No dependencies

2. services.yaml
   - Implements services that use infrastructure from foundation.yaml
   - Depends on: foundation.yaml (tasks 1-2)

3. integration.yaml
   - Wires services together
   - Depends on: services.yaml (all tasks)
EOF
```

**Step 1.3: Validate Current Setup**
```bash
# Ensure current setup works
conductor validate *.yaml --verbose

# If there are errors, fix them before migrating
```

### Phase 2: Migration (15-30 minutes)

**Step 2.1: Start with Leaf Files**

Start with files that have no dependencies on other files (usually the first file).

```bash
# Example: foundation.yaml has no cross-file dependencies
# It's safe to process first
```

**Step 2.2: Add Explicit Cross-File Dependencies**

For each file, add cross-file dependency notation:

**Before (foundation.yaml - no changes needed):**
```yaml
plan:
  name: Foundation Tasks
  tasks:
    - id: 1
      name: Setup Database
      depends_on: []  # No cross-file deps

    - id: 2
      name: Setup Cache
      depends_on: [1]  # Local dependency only
```

**After (foundation.yaml - no changes):**
```yaml
# No changes needed! This file has no cross-file dependencies
```

**Before (services.yaml - implicit dependency):**
```yaml
plan:
  name: Service Implementation
  tasks:
    - id: 3
      name: Auth Service
      depends_on: []  # Missing: implicit dependency on foundation.yaml!

    - id: 4
      name: API Service
      depends_on: [3]  # Local dependency
```

**After (services.yaml - explicit dependency):**
```yaml
plan:
  name: Service Implementation
  tasks:
    - id: 3
      name: Auth Service
      depends_on:
        - file: foundation.yaml   # ADDED: explicit reference
          task: 1                 # ADDED: which task in foundation.yaml

    - id: 4
      name: API Service
      depends_on:
        - file: foundation.yaml   # ADDED: explicit reference
          task: 2                 # ADDED: which task in foundation.yaml
        - 3                       # Existing: local dependency
```

**Step 2.3: Validate After Each File**

```bash
# After modifying a file, validate immediately
conductor validate *.yaml --verbose

# Look for these in output:
# - All tasks found: ✓
# - All dependencies valid: ✓
# - No circular dependencies: ✓
```

**Step 2.4: Process Files in Order**

Process files in the order they're executed:
1. foundation.yaml (usually no changes)
2. services.yaml (add cross-file deps)
3. integration.yaml (add cross-file deps)

### Phase 3: Verification (10 minutes)

**Step 3.1: Validate All Files Together**
```bash
conductor validate *.yaml --verbose
```

Look for output like:
```
✓ Loaded 3 files with 15 total tasks
✓ All dependencies validated (6 cross-file)
✓ No circular dependencies detected
✓ Ready to execute
```

**Step 3.2: Dry Run**
```bash
# Simulate execution without running tasks
conductor run *.yaml --dry-run --verbose
```

Verify execution order looks correct.

**Step 3.3: Single-Task Test**
```bash
# Run just the first wave to test
# If first tasks complete successfully, plan structure is correct
conductor run *.yaml --max-concurrency 1 --verbose
```

**Step 3.4: Full Execution**
```bash
# If all validation passes, run normally
conductor run *.yaml --max-concurrency 3
```

### Phase 4: Cleanup (5 minutes)

**Step 4.1: Verify Completion**

All tasks should complete with their statuses marked:
```bash
# Check plan files for completion status
grep "Status:" foundation.yaml services.yaml integration.yaml
```

**Step 4.2: Commit Migration**
```bash
git add *.yaml
git commit -m "feat: add explicit cross-file dependencies for v2.5

- Added file: references in services.yaml
- Added file: references in integration.yaml
- All cross-file dependencies now explicit
- Improved plan maintainability and error messages
- Fully backward compatible with v2.4"
```

**Step 4.3: Update Documentation**
```bash
# Update your project's README or docs with new dependency structure
# Example: Add diagram showing cross-file dependencies
```

---

## Before and After Examples

### Example 1: Simple Two-File Plan

**Before (v2.4):**

foundation.yaml:
```yaml
plan:
  name: Foundation
  tasks:
    - id: 1
      name: Database Setup
      depends_on: []
```

services.yaml:
```yaml
plan:
  name: Services
  tasks:
    - id: 2
      name: Auth Service
      depends_on: []  # PROBLEM: No explicit dependency!
```

Execution:
```bash
conductor run foundation.yaml services.yaml  # Order matters!
```

**After (v2.5):**

foundation.yaml:
```yaml
# No changes - no cross-file deps
```

services.yaml:
```yaml
plan:
  name: Services
  tasks:
    - id: 2
      name: Auth Service
      depends_on:
        - file: foundation.yaml
          task: 1  # FIXED: Explicit dependency!
```

Execution:
```bash
conductor run *.yaml  # Order doesn't matter anymore!
```

### Example 2: Three-File Microservices Plan

**Before (v2.4):**

```bash
# Must execute in specific order (implicit)
conductor run shared.yaml services.yaml deployment.yaml
```

**After (v2.5):**

shared.yaml:
```yaml
tasks:
  - id: 1
    name: Shared Infrastructure
    depends_on: []
```

services.yaml:
```yaml
tasks:
  - id: 2
    name: Auth Service
    depends_on:
      - file: shared.yaml
        task: 1

  - id: 3
    name: API Service
    depends_on:
      - file: shared.yaml
        task: 1
```

deployment.yaml:
```yaml
tasks:
  - id: 4
    name: Deploy All
    depends_on:
      - file: services.yaml
        task: 2
      - file: services.yaml
        task: 3
```

Execution:
```bash
# Can execute in any order now - dependencies are explicit!
conductor run shared.yaml services.yaml deployment.yaml
conductor run deployment.yaml shared.yaml services.yaml  # Same result!
```

### Example 3: Complex Microservices

**Before (v2.4):**

```bash
# Fragile: breaks if file order changes
conductor run \
  infrastructure.yaml \
  auth-service.yaml \
  api-service.yaml \
  data-service.yaml \
  integration.yaml
```

**After (v2.5):**

infrastructure.yaml:
```yaml
tasks:
  - id: 1
    depends_on: []
```

auth-service.yaml:
```yaml
tasks:
  - id: 2
    depends_on:
      - file: infrastructure.yaml
        task: 1
```

api-service.yaml:
```yaml
tasks:
  - id: 3
    depends_on:
      - file: infrastructure.yaml
        task: 1
      - file: auth-service.yaml
        task: 2
```

data-service.yaml:
```yaml
tasks:
  - id: 4
    depends_on:
      - file: infrastructure.yaml
        task: 1
```

integration.yaml:
```yaml
tasks:
  - id: 5
    depends_on:
      - file: auth-service.yaml
        task: 2
      - file: api-service.yaml
        task: 3
      - file: data-service.yaml
        task: 4
```

**Benefits:**
- Execution order is now explicit
- Robust: Works in any file order
- Self-documenting: Why each file depends on which other files
- Better error messages if dependencies break

---

## Common Patterns

### Pattern 1: Linear Dependency Chain

**Situation:** File A → File B → File C

**Migration:**
```yaml
# file_a.yaml
tasks:
  - id: 1
    depends_on: []

# file_b.yaml
tasks:
  - id: 2
    depends_on:
      - file: file_a.yaml
        task: 1

# file_c.yaml
tasks:
  - id: 3
    depends_on:
      - file: file_b.yaml
        task: 2
```

### Pattern 2: Star Dependency (Multiple Files Depend on One)

**Situation:** Foundation file is used by multiple service files

**Migration:**
```yaml
# foundation.yaml
tasks:
  - id: 1
    depends_on: []

# service1.yaml
tasks:
  - id: 2
    depends_on:
      - file: foundation.yaml
        task: 1

# service2.yaml
tasks:
  - id: 3
    depends_on:
      - file: foundation.yaml
        task: 1

# integration.yaml
tasks:
  - id: 4
    depends_on:
      - file: service1.yaml
        task: 2
      - file: service2.yaml
        task: 3
```

### Pattern 3: Complex Web of Dependencies

**Situation:** Multiple files with multiple cross-file dependencies

**Migration Strategy:**
1. Start with foundation files (lowest dependencies)
2. Work outward to dependent files
3. Validate after each file
4. Document why each cross-file dependency exists

```yaml
# Add comments to complex dependencies
tasks:
  - id: 5
    name: Complex Integration
    # Rationale: Needs DB (foundation), Auth (service1), and API (service2)
    depends_on:
      - file: foundation.yaml
        task: 1      # Database
      - file: service1.yaml
        task: 2      # Auth service
      - file: service2.yaml
        task: 3      # API service
```

---

## Testing Your Migration

### Test 1: Syntax Validation

```bash
# Step 1: Validate all files
conductor validate *.yaml --verbose

# Expected output:
# ✓ All files loaded successfully
# ✓ All dependencies found
# ✓ No circular dependencies
# ✓ Ready to execute
```

### Test 2: Execution Order

```bash
# Step 2: Verify execution order with dry-run
conductor run *.yaml --dry-run --verbose

# Review output to confirm:
# Wave 1, Wave 2, etc. are in expected order
```

### Test 3: Partial Execution

```bash
# Step 3: Run first wave only
conductor run *.yaml --max-concurrency 1 --verbose

# Verify first tasks complete successfully
```

### Test 4: Resume Execution

```bash
# Step 4: Resume from completed state
conductor run *.yaml --skip-completed --verbose

# Verify:
# - Completed tasks are skipped
# - Remaining tasks execute correctly
# - Cross-file dependencies still work
```

### Test 5: Different File Order

```bash
# Step 5: Run files in different order (should produce same result)
conductor validate service.yaml foundation.yaml integration.yaml

# With explicit dependencies, order doesn't matter!
```

---

## Troubleshooting

### Problem: Validation Fails with "Task X not found"

**Cause:** Task number in cross-file reference doesn't exist

**Solution:**
```bash
# 1. Check which file and task is referenced
grep -n "file:" services.yaml

# 2. Verify task exists in target file
grep -n "id: 2" foundation.yaml

# 3. Update reference to correct task number if needed
# - file: foundation.yaml
#   task: 2  # Correct task number
```

### Problem: Different Files Passed, Resume Fails

**Cause:** Resume uses different file set than original execution

**Solution:**
```bash
# Always use same file set for resume
PLAN_FILES="foundation.yaml services.yaml integration.yaml"

# Initial run
conductor run $PLAN_FILES

# Resume later with SAME files in SAME order
conductor run $PLAN_FILES --skip-completed
```

### Problem: Execution Order Doesn't Match Expected

**Cause:** Cross-file dependencies create unexpected wave order

**Solution:**
```bash
# Debug with verbose output
conductor run *.yaml --dry-run --verbose

# Check wave assignments in output
# Add missing explicit dependencies if needed

# Example: If service2 runs before service1 but shouldn't:
# Add explicit dependency:
depends_on:
  - file: service1.yaml
    task: 1  # Force service1 to complete first
```

### Problem: File Path Not Found

**Cause:** File path in `file:` reference is incorrect

**Solution:**
```bash
# 1. Verify file exists and is readable
ls -la foundation.yaml

# 2. Check file path in reference (relative or absolute)
grep "file:" services.yaml

# 3. Use relative path from current directory
# - file: foundation.yaml    # Correct
# - file: ./foundation.yaml  # Also works
# - file: ../plans/foundation.yaml  # Subdirectory
```

---

## Rollback Plan

If migration causes issues, you can easily rollback:

### Quick Rollback

```bash
# Restore from backup
cp backups/*.yaml .

# Verify old setup works
conductor validate *.yaml
conductor run *.yaml --dry-run
```

### Git Rollback

```bash
# If using git, revert migration commit
git revert HEAD  # Undo last commit

# Or reset to backup commit
git reset --hard <backup-commit-sha>

# Verify working state
conductor validate *.yaml
```

### Gradual Rollback

If you migrated partially, you can keep changes that work and revert others:

```bash
# Keep successful migrations
git add services.yaml integration.yaml

# Revert problematic file
git checkout foundation.yaml

# Verify mixed state works
conductor validate *.yaml
```

---

## Summary

### What Changed

| Aspect | Before (v2.4) | After (v2.5) |
|--------|--------------|--------------|
| Dependency Notation | Implicit file order | Explicit file: references |
| Self-Documenting | No - order is hidden | Yes - dependencies explicit |
| Order-Independent | No - must pass in specific order | Yes - works in any order |
| Error Messages | Vague ("task not found") | Clear ("task 2 in foundation.yaml not found") |
| Resumable | Partial - may miss cross-file | Robust - tracks correctly |

### No Breaking Changes

- All v2.4 plans work unchanged
- No migration required
- Gradual adoption possible
- Fully backward compatible

### Benefits of Migration

1. **Self-documenting** - Dependencies are explicit
2. **Robust** - Works regardless of file order
3. **Better errors** - Know exactly which file is missing
4. **Maintainable** - Easy to understand and modify
5. **Future-proof** - Scales to complex multi-service systems

---

## Next Steps

1. **Backup** your plans
2. **Choose migration level** (1, 2, or 3)
3. **Follow step-by-step guide** for your level
4. **Validate** after each change
5. **Test** with dry-run and partial execution
6. **Commit** migration with clear messages
7. **Document** your plan structure

For detailed examples and patterns, see [Cross-File Dependencies Reference](./CROSS_FILE_DEPENDENCIES.md).
