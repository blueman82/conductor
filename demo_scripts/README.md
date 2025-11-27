# Conductor Demo Scripts

This directory contains demo scripts that simulate Conductor's CLI output for presentations and demonstrations. These scripts replicate the exact output users would see when running actual Conductor commands.

## Quick Start

Run all demos in sequence:
```bash
./demo_all.sh
```

Or run individual demos:
```bash
./demo_validate.sh        # Plan validation
./demo_run_dry.sh         # Dry-run execution
./demo_run.sh             # Full execution with QC
./demo_run_failure.sh     # Execution with failures
./demo_learning.sh        # Learning system commands
./demo_observe_stats.sh   # Observe statistics
./demo_observe_project.sh # Observe project analysis
./demo_observe_session.sh # Observe session analysis
./demo_observe_tools.sh   # Observe tool analysis
./demo_observe_bash.sh    # Observe bash analysis
./demo_observe_files.sh   # Observe file analysis
./demo_observe_errors.sh  # Observe error analysis
./demo_observe_ingest.sh  # Observe ingestion daemon
./demo_observe_live.sh    # Observe live streaming
./demo_observe_transcript.sh  # Observe session transcript
./demo_observe_export.sh  # Observe data export
```

## Demo Scripts

### Core Commands

| Script | Command | Description |
|--------|---------|-------------|
| `demo_validate.sh` | `conductor validate` | Shows plan validation with success and error scenarios |
| `demo_run_dry.sh` | `conductor run --dry-run` | Simulates dry-run mode showing execution waves |
| `demo_run.sh` | `conductor run --verbose` | Full execution with QC reviews, retries, and summary |
| `demo_run_failure.sh` | `conductor run --verbose` | Execution with failures and blocked tasks |

### Learning System

| Script | Command | Description |
|--------|---------|-------------|
| `demo_learning.sh` | `conductor learning` | Shows stats, history, export, and clear commands |

### Agent Watch (Observe)

| Script | Command | Description |
|--------|---------|-------------|
| `demo_observe_stats.sh` | `conductor observe stats` | Summary statistics across all sessions |
| `demo_observe_project.sh` | `conductor observe project` | Project listing and analysis |
| `demo_observe_session.sh` | `conductor observe session` | Session listing and detailed analysis |
| `demo_observe_tools.sh` | `conductor observe tools` | Tool usage patterns |
| `demo_observe_bash.sh` | `conductor observe bash` | Bash command analysis |
| `demo_observe_files.sh` | `conductor observe files` | File operation patterns |
| `demo_observe_errors.sh` | `conductor observe errors` | Error pattern analysis |
| `demo_observe_ingest.sh` | `conductor observe ingest` | JSONL ingestion daemon |
| `demo_observe_live.sh` | `conductor observe live` | Real-time event streaming |
| `demo_observe_transcript.sh` | `conductor observe transcript` | Session transcript display |
| `demo_observe_export.sh` | `conductor observe export` | Export to JSON/Markdown/CSV |

## Features

- **Colored Output**: All scripts use ANSI colors matching Conductor's actual output
- **Timing Delays**: Short pauses simulate real execution timing
- **Realistic Data**: Sample data shows realistic task names, agents, and metrics
- **Multiple Scenarios**: Includes success, failure, and edge case scenarios

## Usage Tips

1. **Full Demo**: Run `./demo_all.sh` for a complete walkthrough with pauses between demos
2. **Specific Feature**: Run individual scripts to demo specific features
3. **Presentation Mode**: Terminal should support ANSI colors (most modern terminals do)
4. **Custom Speed**: Modify `sleep` values in scripts to adjust timing

## Notes

- These scripts simulate output only - they don't execute actual Conductor commands
- Output format matches Conductor v2.8.0
- Scripts work best in terminals with ANSI color support
