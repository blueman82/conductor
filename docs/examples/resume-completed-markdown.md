# Resume: Feature Implementation with Completed Tasks

This example demonstrates resuming a partially-completed plan using the skip-completed feature.

**Run with**: `conductor run resume-completed-markdown.md --skip-completed`

---

## Task 1: Database Schema

**File(s)**: migrations/001_users.sql
**Estimated time**: 10 minutes
**Status**: completed

Create users table with authentication fields:
- email (unique, indexed)
- password_hash
- created_at
- updated_at

This task was already completed in the previous run and will be skipped.

---

## Task 2: User Model

**File(s)**: models/user.go
**Depends on**: Task 1
**Estimated time**: 15 minutes
**Status**: completed

Implement the User struct with database methods:
- User struct with fields (ID, Email, PasswordHash, CreatedAt)
- Create() method for new users
- FindByEmail() method for lookup
- ValidatePassword() method for auth

This task was also completed previously and will be skipped.

---

## Task 3: Authentication Service

**File(s)**: services/auth.go
**Depends on**: Task 2
**Estimated time**: 20 minutes

Implement core authentication business logic:
- Login service (email + password verification)
- Register service (validation, hashing, storage)
- Token generation using JWT
- Token refresh functionality
- Password reset workflow

This is a pending task. It will execute normally on resume.

---

## Task 4: API Endpoints

**File(s)**: handlers/auth_handler.go
**Depends on**: Task 3
**Estimated time**: 15 minutes

Implement RESTful endpoints for authentication:
- POST /auth/login
- POST /auth/register
- POST /auth/refresh
- POST /auth/logout
- POST /auth/password-reset

Add proper error handling and validation.

---

## Task 5: Integration Tests

**File(s)**: tests/auth_integration_test.go
**Depends on**: Task 4
**Estimated time**: 20 minutes

Write comprehensive end-to-end tests:
- Test user registration flow
- Test login with valid/invalid credentials
- Test token refresh
- Test logout
- Test password reset
- Test error cases

---

## Usage Examples

### Resume with skip-completed:
```bash
conductor run resume-completed-markdown.md --skip-completed
```

Expected output:
```
Wave 1 (1 task):
  [1/1] Task 1: Database Schema... ⊘ SKIPPED (completed)

Wave 2 (1 task):
  [1/1] Task 2: User Model... ⊘ SKIPPED (completed)

Wave 3 (1 task):
  [1/1] Task 3: Authentication Service... ✓ GREEN

Wave 4 (1 task):
  [1/1] Task 4: API Endpoints... ✓ GREEN

Wave 5 (1 task):
  [1/1] Task 5: Integration Tests... ✓ GREEN
```

### Preview what will execute (dry run):
```bash
conductor run resume-completed-markdown.md --skip-completed --dry-run
```

### Resume with verbose logging:
```bash
conductor run resume-completed-markdown.md --skip-completed --verbose
```

Check logs:
```bash
tail -f .conductor/logs/conductor-latest.log
```

---

## Modifying Status

To re-execute a completed task, remove or change its status:

```markdown
## Task 1: Database Schema

**File(s)**: migrations/001_users.sql
**Estimated time**: 10 minutes
# Removed: **Status**: completed

Then run without --skip-completed to re-execute.
```

---

## Status Values

Supported status values:
- `completed` - Task successfully completed (will be skipped)
- `failed` - Task failed previously (skipped by default, re-executed with `--retry-failed`)
- `in-progress` - Task is currently running (re-executed to verify)
- (empty) - Task not yet executed (will execute)

---

## Related Documentation

- [Skip Completed Tasks Usage Guide](../usage.md#skip-completed-tasks)
- [Plan Format Guide - Status Field](../plan-format.md#status--status)
- [Skip Examples and Scenarios](./skip-completed-examples.md)
