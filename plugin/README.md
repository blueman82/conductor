# Conductor Tools Plugin

Complete design → plan → execute workflow tools for Conductor and Claude Code.

## What is This?

A Claude Code plugin that provides everything you need for the full feature development lifecycle:

**Design** → Use AI to design features (autonomous or interactive)
**Plan** → Generate detailed implementation plans (Markdown or YAML)
**Execute** → Run conductor to implement tasks automatically
**Quality** → Built-in quality control with GREEN/RED/YELLOW verdicts

## Quick Start

### Install

```bash
cd ~/.claude/plugins/
git clone https://github.com/harrison/conductor-plugin.git conductor-tools
```

### Enable

Add to `~/.claude/config.yaml`:
```yaml
plugins:
  enabled:
    - conductor-tools
```

### Use

```bash
# Design a feature with AI
claude "/cook-auto Add user authentication with JWT tokens"

# Or generate a plan directly
claude "/doc-yaml Implement payment processing with Stripe"

# Execute the plan
conductor run docs/plans/payment-processing.yaml --verbose
```

## What's Included

### Commands (4)

- **`/doc`** - Generate comprehensive Markdown implementation plan
- **`/doc-yaml`** - Generate YAML plan (for conductor execution)
- **`/cook-auto`** - Autonomous AI-powered design session
- **`/cook-man`** - Interactive design with user guidance

### Skills (3)

- **`plan-executor`** - Execute conductor plans with monitoring
- **`task-implementer`** - Implement tasks using TDD
- **`quality-reviewer`** - Review code and provide quality verdicts

### Agents (2)

- **`conductor-orchestrator`** - Orchestrate full workflows end-to-end
- **`plan-validator`** - Validate plans before execution

### Templates (2)

- **`example-plan.yaml`** - Complete example YAML plan
- **`example-design.md`** - Example design document

### Documentation (3)

- **`SETUP.md`** - Installation and configuration guide
- **`USAGE.md`** - How to use each command and skill
- **ARCHITECTURE.md`** - How the plugin works internally

## Example Workflow

```bash
# 1. Design a feature
claude "/cook-auto Add real-time notifications system"

# AI models collaborate on design
# Design document generated
# User reviews and approves

# 2. Plan is generated automatically
# docs/plans/notifications-system.yaml created
# - 22 tasks defined
# - 6 worktree groups
# - Agents assigned
# - Tests specified

# 3. Validate the plan
conductor validate docs/plans/notifications-system.yaml
# ✓ Valid - no issues found

# 4. Execute with conductor
conductor run docs/plans/notifications-system.yaml \
  --max-concurrency 3 \
  --verbose

# Conductor:
# - Spawns agents for each task
# - Executes in parallel waves
# - Runs tests (TDD)
# - Quality control reviews
# - Creates commits
# - Updates plan file

# 5. Review results
# 22/22 tasks complete
# 20 GREEN, 2 YELLOW
# 22 commits created
# Feature complete!
```

## Features

### Design Phase

- **Autonomous design** with multi-model AI deliberation
- **Interactive design** with guided questions
- **Codebase analysis** to match existing patterns
- **Architecture recommendations** from AI experts

### Planning Phase

- **Test-driven approach** - tests specified before implementation
- **Dependency management** - automatic dependency graph analysis
- **Worktree grouping** - parallel execution with git worktrees
- **Agent assignment** - right agent for each task
- **Commit strategy** - atomic commits planned upfront

### Execution Phase

- **Parallel execution** - tasks run concurrently when safe
- **Quality control** - every task reviewed (GREEN/RED/YELLOW)
- **Automatic retries** - failed tasks retry up to 2 times
- **Progress tracking** - real-time status updates
- **Plan updates** - completion status tracked in YAML

### Quality Phase

- **Comprehensive reviews** - tests, code quality, requirements
- **Clear verdicts** - GREEN (pass), RED (fail), YELLOW (warning)
- **Actionable feedback** - specific fixes with examples
- **Coverage checks** - test coverage validation

## Prerequisites

- **Claude Code** - CLI installed and working
- **Conductor** - Binary built and in PATH
- **Git 2.5+** - For worktree support
- **Go 1.21+** - If working with Go projects

## Documentation

- **[Setup Guide](docs/SETUP.md)** - Installation and configuration
- **[Usage Guide](docs/USAGE.md)** - How to use commands and skills
- **[Architecture](docs/ARCHITECTURE.md)** - How the plugin works
- **[Examples](templates/)** - Example plans and designs

## Key Concepts

### TDD (Test-Driven Development)

Every task follows TDD:
1. **Red** - Write failing tests
2. **Green** - Implement to pass tests
3. **Refactor** - Improve code quality

### Quality Control

Every task gets a verdict:
- **GREEN** ✓ - All good, proceed
- **RED** ✗ - Critical issues, must fix
- **YELLOW** ⚠ - Minor issues, consider fixing

### Worktree Groups

Tasks are grouped for parallel execution:
- **Dependency chains** - Sequential in same worktree
- **Independent tasks** - Parallel in separate worktrees
- **Maximum parallelism** - Safe concurrent execution

### Agent Assignment

Each task assigned to appropriate agent:
- **golang-pro** - Go implementation
- **python-pro** - Python implementation
- **test-automator** - Test writing
- **technical-documentation-specialist** - Documentation

## Use Cases

### New Feature Development

```bash
# Full workflow: design → plan → execute
claude "/cook-auto Add OAuth2 authentication"
conductor run docs/plans/oauth2-authentication.yaml
```

### Refactoring

```bash
# Generate plan for refactoring
claude "/doc-yaml Refactor user service to use repository pattern"
conductor run docs/plans/refactor-user-service.yaml
```

### Bug Fixes

```bash
# Plan and execute bug fix
claude "/doc-yaml Fix race condition in payment processor"
conductor run docs/plans/fix-race-condition.yaml
```

### Documentation

```bash
# Generate plan for documentation work
claude "/doc Add API documentation with OpenAPI specs"
# Execute manually or with conductor
```

## Requirements

### For This Plugin

- Claude Code CLI installed
- Conductor binary in PATH
- Git 2.5+ for worktree support

### For Generated Plans

- Language-specific tools (go, npm, python, etc.)
- Test frameworks (as used in your project)
- Linters and formatters (as configured in project)

## Compatibility

- **Claude Code**: v1.0+
- **Conductor**: v1.0+
- **Operating Systems**: macOS, Linux, Windows (WSL)
- **Languages**: Any (Go, TypeScript, Python, etc.)

## Architecture

```
┌─────────────────────────┐
│    Claude Code CLI      │
└───────────┬─────────────┘
            │
            ↓
┌─────────────────────────┐
│  Conductor Tools Plugin │
│  - Commands             │
│  - Skills               │
│  - Agents               │
└───────────┬─────────────┘
            │
            ↓
┌─────────────────────────┐
│    Conductor Binary     │
│  - Parse plans          │
│  - Execute tasks        │
│  - Quality control      │
└─────────────────────────┘
```

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

MIT License - see LICENSE file for details

## Support

- **Issues**: [GitHub Issues](https://github.com/harrison/conductor-plugin/issues)
- **Discussions**: [GitHub Discussions](https://github.com/harrison/conductor-plugin/discussions)
- **Conductor Issues**: [Conductor Repo](https://github.com/harrison/conductor/issues)

## Links

- **Conductor Repository**: https://github.com/harrison/conductor
- **Claude Code**: https://claude.ai/code
- **Documentation**: [docs/](docs/)
- **Examples**: [templates/](templates/)

## Version

**Current Version**: 1.0.0

See [CHANGELOG.md](CHANGELOG.md) for version history.

## Credits

Created for use with [Conductor](https://github.com/harrison/conductor), an autonomous multi-agent orchestration CLI.

## Summary

This plugin provides everything you need for feature development:

- Design features with AI assistance
- Generate comprehensive implementation plans
- Execute plans automatically with conductor
- Built-in quality control at every step
- TDD by default
- Parallel execution where safe

**Start developing**: `claude "/cook-auto Your feature idea here"`
