# Split Plan Examples

This directory contains example split plans demonstrating Phase 2A multi-file plan organization.

## Examples

### Backend Implementation (3 parts)

**Files:**
- `split-plan-backend-1-setup.md` - Database and server foundation
- `split-plan-backend-2-features.md` - API endpoints and business logic
- `split-plan-backend-3-testing.md` - Testing and deployment

**Total Tasks:** 10
**Structure:** Part 1 (3) → Part 2 (4) → Part 3 (3)
**Dependencies:** Sequential parts (Part 1 → 2 → 3), some parallel tasks within parts

**Key Concepts:**
- Objective splitting by feature/phase
- Cross-file dependencies
- Worktree groups for execution control
- Clear file organization

**Execution:**
```bash
# Validate all parts
conductor validate split-plan-backend-*.md

# Run all parts
conductor run split-plan-backend-*.md --verbose

# Run with max concurrency
conductor run split-plan-backend-*.md --max-concurrency 4

# Resume interrupted execution
conductor run split-plan-backend-*.md --skip-completed
```

---

## How to Use These Examples

1. **Study the Structure**: Examine how tasks are organized across files
2. **Validate**: Run `conductor validate` to understand dependencies
3. **Dry Run**: Use `--dry-run` to see execution order without running tasks
4. **Execute**: Run with `--verbose` to see detailed progress
5. **Resume**: Use `--skip-completed` if execution is interrupted

---

## Learning Outcomes

After working through these examples, you'll understand:

✅ How to split monolithic plans into focused files
✅ How to manage dependencies across files
✅ How to use worktree groups for execution control
✅ How to organize large projects logically
✅ Best practices for team collaboration on split plans

---

## Patterns Demonstrated

### Pattern 1: Sequential Phases
- Each file represents a development phase
- Clear dependencies between phases
- Allows team members to work in parallel on different phases (when ready)

### Pattern 2: Feature-Based Organization
- Each file focuses on a specific feature or component
- Related tasks stay together
- Clear responsibility boundaries

### Pattern 3: Execution Groups
- Worktree groups enable controlled parallelization
- Some tasks must run sequentially (database migrations)
- Other tasks can run in parallel (independent APIs)

---

## Extending These Examples

Feel free to:
- Add more tasks to each phase
- Create additional split plan examples
- Combine examples with your own workflows
- Share your own split plan patterns!

---

## See Also

- [Phase 2A Guide](../phase-2a-guide.md) - Comprehensive Phase 2A documentation
- [Plan Format Guide](../plan-format.md) - Detailed format specifications
- [Usage Guide](../usage.md) - CLI command reference
