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

- **Wave-Based Execution** — Parallel tasks within waves, sequential between waves
- **Quality Control** — Automated reviews with GREEN/RED/YELLOW verdicts and retries
- **Adaptive Learning** — Learns from history, swaps agents on failures
- **Intelligent Selection** — Claude picks best agents for tasks and QC
- **Multi-File Plans** — Cross-file dependencies, split large plans
- **Runtime Enforcement** — Test commands, criterion verification
- **GUARD Protocol** — Pre-wave failure prediction blocks risky tasks
- **Budget & Rate Limits** — Intelligent auto-resume, state persistence
- **Voice Feedback** — Optional TTS via local Orpheus server

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
tts:
  enabled: false
  base_url: "http://localhost:5005"
```

## Documentation

- [Complete Reference](docs/conductor.md) — Full usage, formats, troubleshooting
- [CLAUDE.md](CLAUDE.md) — Developer guide

## Status

**Production-ready** — 86%+ test coverage, 465+ tests.

## License

[MIT](LICENSE)
