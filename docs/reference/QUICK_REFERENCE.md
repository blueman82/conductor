# Cross-File Dependencies - Quick Reference

## In 60 Seconds

Conductor now supports tasks depending on tasks in other plan files.

### YAML Example
```yaml
# plan-01.yaml
tasks:
  - task_number: 1
    name: Setup Database
    depends_on: []

# plan-02.yaml
tasks:
  - task_number: 2
    name: Implement API
    depends_on:
      - file: "plan-01.yaml"  # Reference to other file
        task: 1               # Depends on task 1 in plan-01
```

### Markdown Example
```markdown
# plan-01.md
## Task 1: Setup Database
**Depends on**: None

# plan-02.md
## Task 2: Implement API
**Depends on**: file:plan-01.md:task:1
```

### Run Multi-File Plans
```bash
conductor run plan-01.yaml plan-02.yaml
```

## All Formats

### YAML
```yaml
# Numeric (local)
depends_on: [1, 2]

# Cross-file
depends_on:
  - file: "plan.yaml"
    task: 2

# Mixed
depends_on:
  - 1
  - file: "plan.yaml"
    task: 2
```

### Markdown
```markdown
# Numeric
**Depends on**: 1, 2

# Cross-file (slash)
**Depends on**: file:plan.yaml/task:2

# Cross-file (colon)
**Depends on**: file:plan.yaml:task:2

# Mixed
**Depends on**: 1, file:plan.yaml/task:2
```

## Normalized Format
All formats normalize internally to:
```
file:{filename}:task:{task-id}
```

## Key Points

✓ Works with YAML and Markdown
✓ Supports any task ID (1, 2.5, "integration-1", etc.)
✓ Supports mixed local and cross-file dependencies
✓ Fully backward compatible
✓ Handles whitespace robustly
✓ Clear error messages

## Common Patterns

### Sequential (A → B → C)
```yaml
# phase-1.yaml
tasks:
  - task_number: 1
    depends_on: []

# phase-2.yaml
tasks:
  - task_number: 1
    depends_on:
      - file: "phase-1.yaml"
        task: 1

# phase-3.yaml
tasks:
  - task_number: 1
    depends_on:
      - file: "phase-2.yaml"
        task: 1
```

### Parallel Converge (A,B → C)
```yaml
# service-a.yaml
tasks:
  - task_number: 1
    depends_on: []

# service-b.yaml
tasks:
  - task_number: 1
    depends_on: []

# gateway.yaml
tasks:
  - task_number: 1
    depends_on:
      - file: "service-a.yaml"
        task: 1
      - file: "service-b.yaml"
        task: 1
```

## Validation

Conductor validates:
- File existence
- Task numbers exist
- No circular dependencies
- Proper type for task IDs

## Files to Read

1. **CROSS_FILE_DEPS_IMPLEMENTATION.md** - Technical details
2. **CROSS_FILE_USAGE_EXAMPLES.md** - Practical examples
3. **PARSER_CHANGES_SUMMARY.md** - Code changes
4. **IMPLEMENTATION_COMPLETE.md** - Final status

## Testing

```bash
# Run tests
go test ./internal/parser -v

# Check coverage
go test ./internal/parser -cover

# Run specific cross-file tests
go test ./internal/parser -run "CrossFile" -v
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| File not found | Verify file path and existence |
| Task not found | Check task number exists in file |
| Circular dependency | Review and break the cycle |
| Format error | Use `file:name.yaml:task:N` or YAML map |

## Example Execution

```bash
# Validate first
conductor validate setup.yaml features.yaml

# Dry run
conductor run setup.yaml features.yaml --dry-run

# Execute
conductor run setup.yaml features.yaml
```

## One More Thing

All existing plans still work exactly as before. This is purely additive.

---

**Questions?** See the full documentation files listed above.
