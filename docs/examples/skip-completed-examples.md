# Skip Completed Tasks: Example Plans

This guide shows practical examples of using Conductor's skip-completed feature to resume interrupted plans.

## Table of Contents

- [Use Case](#use-case)
- [Example 1: Resume After Interruption](#example-1-resume-after-interruption)
- [Example 2: Skip and Retry Workflow](#example-2-skip-and-retry-workflow)
- [Example 3: Parallel Tasks with Partial Completion](#example-3-parallel-tasks-with-partial-completion)

## Use Case

The skip-completed feature is useful for:

1. **Long-running plans** that get interrupted mid-execution
2. **Iterative development** where you fix issues and retry
3. **Partial failures** that need selective re-execution
4. **Testing** of specific workflow portions

## Example 1: Resume After Interruption

**Scenario**: A 5-task implementation plan was running, but got interrupted after Task 2 completed.

**Original plan** (`implementation-plan.md`):

```markdown
# Feature: User Authentication

## Task 1: Database Schema
**File(s)**: migrations/001_users.sql
**Estimated time**: 10 minutes

Create users table with auth fields (email, password_hash, created_at).

## Task 2: User Model
**File(s)**: models/user.go
**Depends on**: Task 1
**Estimated time**: 15 minutes

Implement User struct and database methods.

## Task 3: Authentication Service
**File(s)**: services/auth.go
**Depends on**: Task 2
**Estimated time**: 20 minutes

Implement login/register services.

## Task 4: API Endpoints
**File(s)**: handlers/auth.go
**Depends on**: Task 3
**Estimated time**: 15 minutes

Implement /login and /register endpoints.

## Task 5: Tests
**File(s)**: tests/auth_test.go
**Depends on**: Task 4
**Estimated time**: 20 minutes

Write comprehensive tests for auth flow.
```

**After interruption**, mark completed tasks (manually or via previous conductor run):

```markdown
# Feature: User Authentication

## Task 1: Database Schema
**File(s)**: migrations/001_users.sql
**Estimated time**: 10 minutes
**Status**: completed

Create users table with auth fields (email, password_hash, created_at).

## Task 2: User Model
**File(s)**: models/user.go
**Depends on**: Task 1
**Estimated time**: 15 minutes
**Status**: completed

Implement User struct and database methods.

## Task 3: Authentication Service
**File(s)**: services/auth.go
**Depends on**: Task 2
**Estimated time**: 20 minutes

Implement login/register services.

## Task 4: API Endpoints
**File(s)**: handlers/auth.go
**Depends on**: Task 3
**Estimated time**: 15 minutes

Implement /login and /register endpoints.

## Task 5: Tests
**File(s)**: tests/auth_test.go
**Depends on**: Task 4
**Estimated time**: 20 minutes

Write comprehensive tests for auth flow.
```

**Resume execution**:

```bash
# Skip Tasks 1 & 2 (already completed), run Tasks 3, 4, 5
conductor run implementation-plan.md --skip-completed

# Output:
# Loading plan from implementation-plan.md
# Validating plan...
# Found 5 tasks in 5 waves
#
# Wave 1 (1 task):
#   [1/1] Task 1: Database Schema... ⊘ SKIPPED (completed)
#
# Wave 2 (1 task):
#   [1/1] Task 2: User Model... ⊘ SKIPPED (completed)
#
# Wave 3 (1 task):
#   [1/1] Task 3: Authentication Service... ✓ GREEN (18m30s)
#
# Wave 4 (1 task):
#   [1/1] Task 4: API Endpoints... ✓ GREEN (14m15s)
#
# Wave 5 (1 task):
#   [1/1] Task 5: Tests... ✓ GREEN (19m45s)
#
# Execution completed successfully
# Total time: 52m30s (skipped 2 tasks, executed 3)
```

## Example 2: Skip and Retry Workflow

**Scenario**: Previous run partially failed. Now you want to skip completed tasks and retry the failed one.

**Plan state after previous failed run**:

```markdown
# Payment System Integration

## Task 1: Stripe API Setup
**Estimated time**: 10 minutes
**Status**: completed

Set up Stripe API credentials and client.

## Task 2: Payment Models
**File(s)**: models/payment.go
**Depends on**: Task 1
**Estimated time**: 15 minutes
**Status**: completed

Create Payment and Transaction models.

## Task 3: Payment Service
**File(s)**: services/payment.go
**Depends on**: Task 2
**Estimated time**: 20 minutes
**Status**: failed

Implement payment processing (FAILED - needs fixing).

## Task 4: Payment API Endpoints
**File(s)**: handlers/payment.go
**Depends on**: Task 3
**Estimated time**: 15 minutes

Implement /pay and /refund endpoints.

## Task 5: Integration Tests
**File(s)**: tests/payment_test.go
**Depends on**: Task 4
**Estimated time**: 20 minutes

Write end-to-end payment tests.
```

**Retry the failed task**:

```bash
# Skip completed, retry the failed Task 3
conductor run payment-plan.md --skip-completed --retry-failed

# Output shows:
# Wave 1-2: Skipped (already completed)
# Wave 3: Task 3 RE-EXECUTED (was failed, now retry)
# Waves 4-5: Execute normally if Task 3 now succeeds
```

Alternatively, without `--retry-failed`, failed tasks are also skipped:

```bash
# Skip completed AND failed, only run pending
conductor run payment-plan.md --skip-completed

# Skips Tasks 1-3, runs Tasks 4-5 if dependencies allow
```

## Example 3: Parallel Tasks with Partial Completion

**Scenario**: A complex plan with parallel waves where some tasks are done.

**YAML format** (`feature-plan.yaml`):

```yaml
plan:
  name: Data Migration & Reporting
  description: Multi-wave data migration with parallel reporting setup
  tasks:
    - id: 1
      name: Database Backup
      files: [migrations/backup.sql]
      estimated_time: 10 minutes
      status: completed
      description: Full database backup before migration.

    - id: 2
      name: Migrate User Data
      files: [migrations/users.go]
      depends_on: [Task 1]
      estimated_time: 20 minutes
      description: Extract and transform user data.

    - id: 3
      name: Migrate Transaction Data
      files: [migrations/transactions.go]
      depends_on: [Task 1]
      estimated_time: 25 minutes
      status: completed
      description: Extract and transform transaction data.

    - id: 4
      name: Reporting Module
      files: [reporting/dashboard.go]
      depends_on: [Task 2, Task 3]
      estimated_time: 30 minutes
      description: Build reporting dashboard.

    - id: 5
      name: Data Validation
      files: [scripts/validate.go]
      depends_on: [Task 4]
      estimated_time: 15 minutes
      description: Validate migrated data integrity.
```

**Execution flow**:

```bash
# First run (everything)
conductor run feature-plan.yaml --verbose

# Later: Resume with skip
conductor run feature-plan.yaml --skip-completed --verbose

# Output:
# Wave 1 (1 task):
#   Task 1: Database Backup... ⊘ SKIPPED (completed)
#
# Wave 2 (2 parallel tasks):
#   Task 2: Migrate User Data... ✓ GREEN
#   Task 3: Migrate Transaction Data... ⊘ SKIPPED (completed)
#
# Wave 3 (1 task):
#   Task 4: Reporting Module... ✓ GREEN
#
# Wave 4 (1 task):
#   Task 5: Data Validation... ✓ GREEN
```

## Configuration Example

Set defaults in `.conductor/config.yaml`:

```yaml
# Enable skip-completed by default for this project
skip_completed: true
retry_failed: false

# But allow CLI override
# conductor run plan.md --no-skip-completed  (disables skip)
# conductor run plan.md --retry-failed       (enables retry)
```

## Best Practices

1. **Always mark status explicitly** if manually editing plans
2. **Use `--skip-completed` for long plans** that are likely to be interrupted
3. **Combine with `--dry-run`** to preview what will execute:
   ```bash
   conductor run plan.md --skip-completed --dry-run
   ```

4. **Check logs** to verify skip decisions:
   ```bash
   tail -f .conductor/logs/conductor-latest.log
   ```

5. **Use config file** for team defaults:
   ```yaml
   # .conductor/config.yaml
   skip_completed: true  # Default for all team members
   ```

## Troubleshooting

**Problem**: Completed tasks still executed

```bash
# Ensure flag is correct
conductor run plan.md --skip-completed  # Correct

# Not:
conductor run plan.md --skip  # This won't work
```

**Problem**: Want to re-execute a completed task

```bash
# Remove or change status to empty
# Edit the plan file: remove **Status**: completed line
# Then run without --skip-completed
conductor run plan.md
```

**Problem**: Mixed status not working as expected

```bash
# Check the plan for correct status values
grep "Status:" plan.md

# Valid values: completed, failed, in-progress (or empty)
# Invalid: done, success, error, etc.
```

## See Also

- [Usage Guide: Skip Completed Tasks](../usage.md#skip-completed-tasks)
- [Plan Format Guide: Status Field](../plan-format.md#status--status)
- [Main README](../../README.md#resume-interrupted-plans)
