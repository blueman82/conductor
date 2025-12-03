# Markdown Format Specification

## Overview

This document defines the markdown syntax for parsing Conductor plan files. It specifies the format for both existing and new fields, including syntax patterns, regex extraction rules, and comprehensive examples.

---

## Section 1: Syntax Definition

### Core Task Structure

Each task is defined under a level-2 heading (`## Task N: Task Name`) followed by metadata fields and sections.

#### Basic Fields (Existing)

| Field | Markdown Syntax | Example |
|-------|-----------------|---------|
| Task Number & Name | `## Task N: Task Name` | `## Task 1: Initialize project` |
| Estimated Time | `**Estimated Time**: 30m` | `**Estimated Time**: 1h 30m` |
| Agent | `**Agent**: agent-name` | `**Agent**: code-reviewer` |
| Status | `**Status**: pending\|in_progress\|completed` | `**Status**: pending` |
| Completed At | `**Completed At**: YYYY-MM-DD HH:MM:SS` | `**Completed At**: 2024-01-15 14:30:00` |

#### New Fields

##### 1. Type Field

**Syntax**:
```markdown
**Type**: component|integration
```

**Examples**:
```markdown
**Type**: component
**Type**: integration
```

**Notes**:
- Valid values: `component` (default, single module) or `integration` (multi-module wiring)
- Inline metadata field, consistent with existing pattern
- If omitted, defaults to `component`

---

##### 2. Success Criteria

**Syntax**:
```markdown
**Success Criteria**:
- Criterion 1 description
- Criterion 2 description with multiple lines
  can continue on next line indented
- Criterion 3
```

**Format Rules**:
- Bullet list under `**Success Criteria**:` heading
- Each criterion is a `-` bullet point
- Multi-line criteria use indentation (2 spaces) for continuation
- Empty list allowed (no bullets means no criteria)
- List ends when next section heading or field appears

**Examples**:
```markdown
**Success Criteria**:
- Function returns correct value
- Error handling works for edge cases
  and handles all null pointer conditions
- Performance meets requirements
```

**Edge Cases**:
```markdown
**Success Criteria**:
(empty - no criteria defined)

**Success Criteria**:
- Single criterion
```

---

##### 3. Test Commands

**Syntax** (Format A - Code Block):
```markdown
**Test Commands**:
```bash
go test ./...
go test -race ./...
```
```

**Syntax** (Format B - Bullet List):
```markdown
**Test Commands**:
- go test ./...
- go test -race ./...
```

**Format Rules**:
- Can use either triple-backtick code block OR bullet list
- Code block: ` ```bash ` ... ` ``` `
- Bullet list: `-` prefixed commands
- Each command on separate line
- Empty list allowed

**Examples**:

Format A (Code Block):
```markdown
**Test Commands**:
```bash
npm test
npm run build
npm run lint
```
```

Format B (Bullet List):
```markdown
**Test Commands**:
- npm test
- npm run build
- npm run lint
```

---

##### 4. Integration Criteria (Integration Tasks Only)

**Syntax**:
```markdown
**Integration Criteria**:
- Criterion 1 for cross-component interaction
- Criterion 2 describing data flow
- Criterion 3 describing control flow
```

**Format Rules**:
- Bullet list, identical format to Success Criteria
- Only applicable to `type: integration` tasks
- Each criterion describes cross-component requirements
- Multi-line supported with indentation

**Examples**:
```markdown
**Integration Criteria**:
- Auth middleware executes before handlers
- Database transaction commits atomically
- Error propagates end-to-end to client
```

---

##### 5. Runtime Metadata

**Syntax**:
```markdown
### Dependency Checks:
- Check 1 description
- Check 2 description

### Documentation Targets:
- Target 1 description
- Target 2 description

### Prompt Blocks:
- Block 1 context
- Block 2 context
```

**Format Rules**:
- Use level-3 headings (`###`) for categories
- Each category has bullet list of items
- Categories under this section: Dependency Checks, Documentation Targets, Prompt Blocks
- Optional section (entire section can be omitted)
- Items are descriptive strings

**Examples**:
```markdown
### Dependency Checks:
- Verify database migrations completed
- Confirm environment variables set

### Documentation Targets:
- API reference in docs/api.md
- Architecture diagram in docs/architecture.md

### Prompt Blocks:
- Core business logic requirements
- Security considerations for auth system
```

---

##### 6. Structured Criteria

**Syntax**:
```markdown
**Structured Criteria**:
- Criterion description here
  - Verification Command: `command to run`
  - Expected: expected output or result
  - Description: detailed explanation

- Another criterion
  - Verification Command: `another command`
  - Expected: expected behavior
  - Description: what this checks
```

**Format Rules**:
- Parent list with criterion descriptions
- Indented sub-bullets (2 spaces) for verification details
- Three required sub-fields:
  - `Verification Command:` - command to validate
  - `Expected:` - expected result/output
  - `Description:` - explanation of what's being verified
- Multiple criteria supported
- Sub-fields can span multiple lines with additional indentation

**Examples**:
```markdown
**Structured Criteria**:
- API response includes auth token
  - Verification Command: `curl -X GET /api/user -H "Authorization: Bearer $TOKEN"`
  - Expected: HTTP 200 with user object containing token field
  - Description: Ensures token is included in response for client session management

- Database transaction is atomic
  - Verification Command: `go test -run TestTransactionRollback`
  - Expected: Test passes, rollback executed on error
  - Description: Verifies transaction isolation and rollback behavior
```

---

##### 7. Key Points

**Syntax**:
```markdown
**Key Points**:
1. First key point with optional detail
   - Reference: docs/architecture.md
   - Impact: High priority feature

2. Second key point
   - Note: Important implementation detail

3. Third key point
```

**Format Rules**:
- Numbered list (1., 2., 3., etc.) under `**Key Points**:` heading
- Each point can have nested details (indented 2 spaces)
- Optional nested items:
  - `Reference:` - links to related docs/code
  - `Impact:` - priority/importance level
  - `Note:` - additional context
- Multi-line points supported with indentation
- Empty list allowed

**Examples**:
```markdown
**Key Points**:
1. Implement according to RFC 7231 HTTP spec
   - Reference: https://tools.ietf.org/html/rfc7231
   - Impact: Required for compliance

2. Consider backward compatibility
   - Note: Maintain v1 API endpoints during transition

3. Performance critical path
   - Impact: High - affects user experience
   - Reference: docs/performance-targets.md
```

---

## Section 2: Parsing Patterns

### Regex Patterns for Extraction

#### Type Field
```regex
^\*\*Type\*\*:\s*(component|integration)\s*$
```
- Matches inline field on single line
- Captures: `(component|integration)`
- Case-sensitive

#### Success Criteria
```regex
^\*\*Success Criteria\*\*:\s*$
(?:^[\s]*-\s+.+$(?:\n(?:^[\s]{2,}.*$))*)*
```
- Matches heading line
- Captures all following `-` bullet lines and indented continuations
- Terminates at next field or heading (non-whitespace at column 0)

#### Test Commands (Code Block)
```regex
^\*\*Test Commands\*\*:\s*$
^```bash\s*$
(.*?)
^```\s*$
```
- Matches heading, opening fence, content, closing fence
- Captures content between fences
- Splits on newline to get individual commands

#### Test Commands (Bullet List)
```regex
^\*\*Test Commands\*\*:\s*$
(?:^[\s]*-\s+.+$(?:\n(?:^[\s]{2,}.*$))*)*
```
- Alternative pattern for bullet list format
- Same as Success Criteria bullet extraction

#### Integration Criteria
```regex
^\*\*Integration Criteria\*\*:\s*$
(?:^[\s]*-\s+.+$(?:\n(?:^[\s]{2,}.*$))*)*
```
- Identical to Success Criteria pattern
- Only present in `type: integration` tasks

#### Runtime Metadata (Section Heading)
```regex
^###\s+(Dependency Checks|Documentation Targets|Prompt Blocks):\s*$
(?:^[\s]*-\s+.+$(?:\n(?:^[\s]{2,}.*$))*)*
```
- Matches level-3 heading with colon
- Captures category name
- Captures following bullet list

#### Structured Criteria
```regex
^\*\*Structured Criteria\*\*:\s*$
(?:^[\s]*-\s+(.+)$(?:\n(?:^[\s]{2,}-\s+(\w+):\s+(.*)$)*)*)*
```
- Top-level bullets: criterion descriptions
- Sub-bullets: key-value pairs (Key: value format)
- Required sub-fields: Verification Command, Expected, Description

#### Key Points
```regex
^\*\*Key Points\*\*:\s*$
(?:^\d+\.\s+(.+)$(?:\n(?:^[\s]{2,}-\s+(\w+):\s+(.*)$)*)*)*
```
- Numbered list items
- Optional nested details with key: value format
- Reference, Impact, Note as common nested keys

---

## Section 3: Complete Example

```markdown
# Service Migration Plan

## Task 1: Set up development environment

**Type**: component
**Estimated Time**: 30m
**Agent**: code-reviewer
**Status**: pending

**Success Criteria**:
- Node.js 18+ installed and verified
- Development dependencies installed successfully
- Test suite runs without configuration errors

**Test Commands**:
```bash
node --version
npm install
npm test -- --passWithNoTests
```

**Key Points**:
1. Use Node 18 LTS for compatibility
   - Reference: .nvmrc configuration
   - Impact: High - baseline requirement

2. Install all dev dependencies
   - Note: Includes testing and linting tools

---

## Task 2: Implement API router

**Type**: component
**Estimated Time**: 1h 30m
**Agent**: typescript-pro
**Status**: pending

**Success Criteria**:
- Express router created with GET /api/users endpoint
- Request validation middleware integrated
  and handles malformed JSON gracefully
- Router returns JSON with correct HTTP status codes
- Error handling catches and logs all exceptions

**Test Commands**:
- npm run test:api
- npm run test:coverage
- npm run lint

**Structured Criteria**:
- API endpoint accepts valid requests
  - Verification Command: `curl -X GET http://localhost:3000/api/users`
  - Expected: HTTP 200 with JSON array of users
  - Description: Basic endpoint functionality

- Invalid requests return 400
  - Verification Command: `curl -X POST http://localhost:3000/api/users -d '{invalid}'`
  - Expected: HTTP 400 with error message
  - Description: Validation middleware rejects malformed input

**Key Points**:
1. Follow REST conventions
   - Reference: src/middleware/validation.ts
   - Impact: Standard API design

2. Error responses include request ID for tracing
   - Note: Aids debugging in production

---

## Task 3: Implement database persistence

**Type**: component
**Estimated Time**: 2h
**Agent**: database-optimizer
**Status**: pending
**Depends on**: Task 1

**Success Criteria**:
- SQLite database initialized with schema
- User model created with CRUD operations
- Connection pooling configured
- Migration system implemented for future schema changes

**Test Commands**:
```bash
go test ./internal/database -v
go test ./internal/database -race
```

**Structured Criteria**:
- Database connection establishes successfully
  - Verification Command: `./conductor run db-test.md`
  - Expected: All database tests pass
  - Description: Validates connectivity and initialization

### Dependency Checks:
- Verify schema migration files are in db/migrations/
- Confirm database file is gitignored

### Documentation Targets:
- docs/database-schema.md for schema reference
- docs/migration-guide.md for migration procedures

### Prompt Blocks:
- Database normalization requirements
- Transaction isolation level specifications

---

## Task 4: Connect API to database

**Type**: integration
**Estimated Time**: 1h
**Agent**: backend-developer
**Status**: pending
**Depends on**: Task 2, Task 3

**Success Criteria**:
- API handlers call database layer correctly
- Database queries return expected data structure
- Connection lifecycle properly managed

**Integration Criteria**:
- API requests trigger correct database queries
- Response data includes all required fields from database
- Error from database propagates to API error handler
- Transaction boundaries maintained across API-DB calls

**Test Commands**:
- npm run test:integration
- npm run test:e2e

**Structured Criteria**:
- GET /api/users returns data from database
  - Verification Command: `npm run test:integration -- --grep "GET /api/users"`
  - Expected: Test passes, response matches database schema
  - Description: Integration point between API and persistence layer

- Database errors handled gracefully in API responses
  - Verification Command: `npm run test:integration -- --grep "database error"`
  - Expected: API returns 500 with error message, transaction rolled back
  - Description: Error handling across service boundaries

### Dependency Checks:
- Verify Task 2 API router implementation complete
- Confirm Task 3 database layer functional

### Documentation Targets:
- docs/api-integration.md for API-database contract
- docs/error-handling.md for error propagation rules

### Prompt Blocks:
- Database connection management patterns
- Transaction handling for multi-step operations
- Error recovery strategies

**Key Points**:
1. Wire components together without coupling
   - Reference: internal/services/user-service.ts
   - Impact: High - core integration task

2. Test all failure modes
   - Note: Database unavailable, connection timeout, transaction failure scenarios

3. Monitor database query performance
   - Impact: Medium - affects response times
   - Reference: docs/performance-monitoring.md

---

## Task 5: Documentation and validation

**Type**: component
**Estimated Time**: 45m
**Agent**: technical-writer
**Status**: pending
**Depends on**: Task 4

**Success Criteria**:
- API documentation complete with examples
- Database schema documented
- Architecture diagram updated
- Setup instructions tested and verified

**Test Commands**:
```bash
npm run validate:docs
npm run build:docs
```

**Key Points**:
1. Keep documentation in sync with code
   - Reference: docs/ directory
   - Impact: Medium - improves developer experience

2. Include example requests and responses
   - Note: Helps developers onboard quickly

3. Document breaking changes prominently
```

---

## Section 4: Edge Cases and Optional Patterns

### Empty Criteria Lists

When a task has no success criteria:

```markdown
## Task N: Task Name

**Success Criteria**:
(continue immediately to next section or field)
```

When optional sections are not needed:

```markdown
## Task N: Task Name

**Type**: component
**Estimated Time**: 30m
**Agent**: code-reviewer
**Status**: pending
(no Runtime Metadata, Structured Criteria, or Key Points sections)
```

### Optional Fields Summary

| Field | Required | Default | Notes |
|-------|----------|---------|-------|
| Type | No | component | Defaults to component if omitted |
| Success Criteria | No | (empty) | No criteria if section omitted |
| Test Commands | No | (empty) | Can be omitted for pure documentation tasks |
| Integration Criteria | No | N/A | Only valid for `type: integration` |
| Runtime Metadata | No | (empty) | Entirely optional section |
| Structured Criteria | No | (empty) | Advanced field, optional |
| Key Points | No | (empty) | Optional implementation notes |

### Multi-Line Field Continuation

Fields that span multiple lines use indentation:

```markdown
**Success Criteria**:
- Long criterion that spans
  multiple lines with proper indentation
  and continues naturally
- Another criterion
```

### Whitespace Handling

- Leading/trailing spaces trimmed from field values
- Blank lines between sections are ignored
- Indentation (2 spaces) used consistently for multi-line content

---

## Section 5: Migration Guide

### Converting Existing Plans

Existing plans using only core fields require no changes. To add new features:

1. **Add Type**: Insert `**Type**: component` or `**Type**: integration`
2. **Add Criteria**: Insert `**Success Criteria**:` section with bullets
3. **Add Tests**: Insert `**Test Commands**:` section (code block or list)
4. **For Integration Tasks**: Add `**Integration Criteria**:` and `### Dependency Checks:` sections
5. **Add Key Points**: Optional `**Key Points**:` for implementation guidance

### Example Conversion

**Before**:
```markdown
## Task 1: Implement feature

**Estimated Time**: 1h
**Agent**: code-reviewer
**Status**: pending
**Depends on**: Task 0
```

**After**:
```markdown
## Task 1: Implement feature

**Type**: component
**Estimated Time**: 1h
**Agent**: code-reviewer
**Status**: pending
**Depends on**: Task 0

**Success Criteria**:
- Feature works as specified
- Tests pass

**Test Commands**:
- npm test

**Key Points**:
1. Follow existing patterns
   - Reference: src/features/
```

---

## Appendix A: Field Reference

| Field | Type | Format | Example |
|-------|------|--------|---------|
| Type | Enum | `component\|integration` | `**Type**: integration` |
| Success Criteria | List | Bullets | `- Criterion text` |
| Test Commands | Code/List | ```bash``` or `-` | `npm test` |
| Integration Criteria | List | Bullets | `- Cross-component check` |
| Runtime Metadata | Section | `### Category:` | `### Dependency Checks:` |
| Structured Criteria | List | Bullets + sub-fields | `- Text` + sub-bullets |
| Key Points | List | Numbered | `1. Point text` |

---

## Appendix B: Parser Implementation Notes

### Parsing Strategy

1. **Split on `## Task N:` headings** to identify task boundaries
2. **Extract inline fields** (Type, Status, Agent, etc.) line by line
3. **Identify section headings** (`**Field Name**:` or `### Category:`)
4. **Collect list content** until next heading or field
5. **Handle indentation** for multi-line items (2+ space indent = continuation)
6. **Validate cross-references** (dependency links, file references)

### Error Handling

- Missing required fields → default values or skip task
- Malformed lists → trim whitespace, log warning
- Invalid Type values → default to `component`
- Empty lists → treat as zero items (valid)
- Circular dependencies → detected by graph algorithm

### Performance Considerations

- Lazy parse criteria (don't split until needed)
- Use regex to identify section boundaries first
- Cache compiled patterns for repeated use
- Limit backtracking in multi-line patterns

