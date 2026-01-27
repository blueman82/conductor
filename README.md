<p align="center">
  <img src="assets/logo.png" alt="Conductor Logo" width="400">
</p>

<h1 align="center">Conductor</h1>

<p align="center">
  <strong>Autonomous Multi-Agent Orchestration for Claude Code</strong>
</p>

<p align="center">
  <img src="assets/conductor-demo.gif" alt="Conductor Demo" width="700">
</p>

Conductor orchestrates multiple Claude Code agents to execute implementation plans in parallel waves. It parses Markdown/YAML plans, calculates dependencies, spawns agents, and manages quality control reviews with automatic retries.

## Features

[![Run in Smithery](https://smithery.ai/badge/skills/blueman82)](https://smithery.ai/skills?ns=blueman82&utm_source=github&utm_medium=badge)


- **Wave-Based Execution** — Parallel tasks within waves, sequential between waves
- **Quality Control** — Automated reviews with GREEN/RED/YELLOW verdicts and retries
- **Adaptive Learning** — Learns from history, swaps agents on failures
- **Intelligent Selection** — Claude picks best agents for tasks and QC
- **Multi-File Plans** — Cross-file dependencies, split large plans
- **Runtime Enforcement** — Test commands, criterion verification
- **Pattern Intelligence** — STOP protocol for prior art detection, duplicate prevention
- **Mandatory Commits** — Agents instructed to commit, conductor verifies via git log
- **Budget & Rate Limits** — Intelligent auto-resume, session resume, state persistence
- **Voice Feedback** — Optional TTS via local Orpheus server
- **Setup Introspection** — Pre-wave setup phase via Claude introspection for dependency installation
- **Git Rollback** — Task-level checkpoints with automatic rollback on QC failure
- **LOC Tracking** — Track lines added/deleted per task, aggregate in wave summary
- **Human Time Estimation** — Estimate human developer time, calculate speedup ratio

## Quick Start

```bash
# Install (macOS Apple Silicon)
curl -L https://github.com/blueman82/conductor/releases/latest/download/conductor-darwin-arm64 -o conductor
chmod +x conductor && sudo mv conductor /usr/local/bin/

# Validate and run a plan
conductor validate plan.md
conductor run plan.md --verbose
```

See [releases](https://github.com/blueman82/conductor/releases) for other platforms.

## Usage

```bash
conductor validate plan.md          # Check syntax and dependencies
conductor run plan.md               # Execute the plan
conductor run plan.md --dry-run     # Preview without executing
conductor run plan.md --task 3      # Run only task 3
conductor run plan.md --skip-completed --retry-failed  # Resume
```

Key flags: `--max-concurrency N`, `--timeout 10h`, `--verbose`, `--log-dir PATH`

## Plan Format

**Markdown:**
```markdown
## Task 1: Setup
**File(s)**: setup.go
**Depends on**: None

Create project structure.

## Task 2: Implementation
**File(s)**: main.go
**Depends on**: Task 1
**Success Criteria**:
- Compiles without errors
- Tests pass

Implement main logic.
```

**YAML:**
```yaml
plan:
  tasks:
    - id: 1
      name: Setup
      files: [setup.go]
      description: Create project structure.
    - id: 2
      name: Implementation
      files: [main.go]
      depends_on: [1]
      success_criteria: ["Compiles", "Tests pass"]
      description: Implement main logic.
```

## Configuration

Create `.conductor/config.yaml`:

```yaml
max_concurrency: 3
timeout: 10h
quality_control:
  enabled: true
  retry_on_red: 2
  agents:
    mode: intelligent
learning:
  enabled: true
  swap_during_retries: true
guard:
  enabled: true
  mode: warn  # block | warn | adaptive
  probability_threshold: 0.7
pattern:
  enabled: true
  mode: warn  # block | warn | suggest
  enable_stop: true
  require_justification: true
tts:
  enabled: false
  base_url: "http://localhost:5005"
budget:
  enabled: true
  auto_resume: true           # Wait for rate limit reset
  max_wait_duration: 6h       # Save state if wait exceeds this
setup:
  enabled: false              # Pre-wave setup via Claude introspection
rollback:
  enabled: false              # Task-level git checkpoints with auto-rollback
  keep_checkpoint_days: 7     # Auto-cleanup stale checkpoint branches
```

## Development

### Local CI

Run the CI pipeline locally before pushing:

```bash
# Create symlink (one-time setup)
ln -sf scripts/ci-local.sh check

# Run full CI (format, vet, tests with race detection)
./check

# Quick check (no tests)
./check --quick

# Auto-fix formatting issues
./check --fix

# Verbose output with custom parallelism
./check -v -p 8
```

Or use `make ci` directly.

### Worktrees for Feature Development

Use `worktree-add` to create isolated worktrees for parallel feature development:

```bash
# Create a worktree for a new feature branch
./worktree-add ../conductor-my-feature feature/my-feature

# Work in the new worktree
cd ../conductor-my-feature
conductor run docs/plans/my-feature.yaml

# When done, remove the worktree
git worktree remove ../conductor-my-feature
```

This allows running Conductor plans in isolation without affecting your main working directory.

## Claude Code Integration

Conductor includes a slash command for Claude Code that generates conductor-compatible YAML plans:

```bash
# In Claude Code, use the slash command:
/conductor implement user authentication with JWT tokens
```

The command invokes the `implementation-planner` skill to create a structured YAML plan ready for execution. Install by copying `.claude/commands/conductor.md` to your `~/.claude/commands/` directory.

## Documentation

- [Complete Reference](docs/conductor.md) — Full usage, formats, troubleshooting
- [CLAUDE.md](CLAUDE.md) — Developer guide

## Status

**Production-ready** — 86%+ test coverage, 500+ tests.

## Acknowledgments

Inspired by [CLAWED](https://github.com/badvision/clawed) (Claude Agent Workflow Execution Delegation) by [@badvision](https://github.com/badvision) — a production-tested multi-agent orchestration framework that shares many of the same quality control and agent coordination patterns.

## License

[MIT](LICENSE)
