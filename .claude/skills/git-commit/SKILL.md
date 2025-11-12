---
name: git-commit
description: Create clean git commits following conventional commit format without emojis or AI attribution. Use when committing code changes, creating git commits, or managing version control.
allowed-tools: Bash, Read, Grep
---

# Git Commit Skill

**Purpose**: Create professional git commits without emojis or AI/Claude attribution to avoid triggering GitHub bot detection.

## Commit Message Format

Follow conventional commits specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, semicolons, etc.)
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `test`: Adding or updating tests
- `build`: Build system changes
- `ci`: CI/CD changes
- `chore`: Other changes (version bumps, dependencies, etc.)

### Rules

**MUST NOT include:**
- ❌ Emojis (🤖, ✅, ⚡, etc.)
- ❌ "Generated with Claude Code"
- ❌ "Co-Authored-By: Claude"
- ❌ Any AI attribution
- ❌ Bot indicators
- ❌ Badges in documentation (shields.io, img.shields.io, etc.)

**MUST include:**
- ✓ Clear, concise subject line (50 chars or less)
- ✓ Imperative mood ("add" not "added" or "adds")
- ✓ Body explaining WHAT and WHY (if needed)
- ✓ Reference issue numbers if applicable

## Instructions

### Step 1: Review Changes
```bash
git status
git diff --staged
```

### Step 2: Analyze Changes
- Determine commit type (feat, fix, docs, etc.)
- Identify scope (optional: parser, executor, cli, etc.)
- Summarize what changed and why

### Step 3: Create Commit Message

**Format:**
```
type(scope): subject line

Optional body paragraph explaining what changed
and why it was necessary.

Fixes #123
```

**Example - Good:**
```
feat(parser): add multi-file plan loading support

Implement parser.LoadMultipleFiles to merge plans from
multiple markdown or yaml files with cross-file dependency
validation.

Relates to #45
```

**Example - Bad:**
```
🤖 feat: add parser stuff

Added some new parsing features.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

### Step 4: Execute Commit

Use simple commit command:

```bash
git add <files>
git commit -m "type(scope): subject" -m "body text here"
```

**For multi-line commits**, use this format:

```bash
git commit -m "type(scope): subject line" -m "
Body paragraph explaining the change
and why it was needed.

Fixes #123"
```

## Examples

### Example 1: Feature Addition
```bash
git add internal/parser/parser.go internal/parser/parser_test.go
git commit -m "feat(parser): add YAML frontmatter support" -m "
Parse optional YAML frontmatter from markdown plan files
to extract conductor configuration settings.

Enables per-plan configuration overrides."
```

### Example 2: Bug Fix
```bash
git add internal/executor/task.go
git commit -m "fix(executor): handle timeout errors correctly" -m "
Previously timeout errors were not properly wrapped,
causing confusing error messages.

Fixes #67"
```

### Example 3: Documentation
```bash
git add README.md
git commit -m "docs: update installation instructions"
```

### Example 4: Refactoring
```bash
git add internal/models/plan.go internal/models/task.go
git commit -m "refactor(models): extract FileToTaskMap to separate type" -m "
Improves code organization and makes the mapping
logic more testable."
```

### Example 5: Version Bump
```bash
git add VERSION Makefile
git commit -m "chore: bump version to 1.1.0"
```

## Quality Checks

Before committing, verify:
1. ✓ Subject line is clear and concise
2. ✓ No emojis anywhere in the message
3. ✓ No AI/Claude attribution
4. ✓ Uses conventional commit format
5. ✓ Body explains WHY if change is non-obvious
6. ✓ All staged files are intentional

## Anti-Patterns to Avoid

❌ **Don't:**
```bash
# Too vague
git commit -m "fix stuff"

# Has emoji
git commit -m "feat: add parser ✨"

# Has AI attribution
git commit -m "feat: add feature

🤖 Generated with Claude Code"

# Wrong tense
git commit -m "feat: added new feature"
```

✓ **Do:**
```bash
# Clear and specific
git commit -m "fix(parser): handle empty task lists correctly"

# Proper format
git commit -m "feat(cli): add --version flag" -m "
Display conductor version and build information
when --version flag is provided."

# Good scope
git commit -m "docs(readme): add version management section"
```

## Version History

### v1.0.0 (2025-11-12)
- Initial skill creation
- Conventional commit format
- GitHub bot detection avoidance
