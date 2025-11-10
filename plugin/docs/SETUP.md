# Setup Guide - Conductor Tools Plugin

Complete guide to installing and configuring the conductor-tools plugin for Claude Code.

## Prerequisites

Before installing this plugin, ensure you have:

1. **Claude Code CLI installed**
   ```bash
   # Check if installed
   claude --version
   ```

2. **Conductor binary installed**
   ```bash
   # Check if installed
   conductor --version

   # If not installed, build from source
   cd /path/to/conductor
   go build ./cmd/conductor
   sudo mv conductor /usr/local/bin/
   ```

3. **Git 2.5+** (for worktree support)
   ```bash
   git --version
   ```

4. **File descriptor tool (optional but recommended)**
   ```bash
   # macOS
   brew install fd

   # Linux
   apt install fd-find  # or: dnf install fd-find
   ```

## Installation Methods

### Method 1: Copy from Conductor Repository (Recommended)

The plugin is now included in the [Conductor monorepo](https://github.com/blueman82/conductor).

```bash
# Clone conductor repository (if you haven't already)
git clone https://github.com/blueman82/conductor.git
cd conductor

# Copy plugin to Claude Code plugins directory
cp -r plugin ~/.claude/plugins/conductor-tools

# Verify installation
ls ~/.claude/plugins/conductor-tools/
```

### Method 2: Direct Copy from Existing Conductor Clone

If you already have Conductor cloned:

```bash
# From any directory, copy the plugin
cp -r /path/to/conductor/plugin ~/.claude/plugins/conductor-tools

# Verify installation
ls ~/.claude/plugins/conductor-tools/
```

## Directory Structure

After installation, you should have:

```
~/.claude/plugins/conductor-tools/
├── PLUGIN.yaml                       # Plugin manifest
├── README.md                         # Overview
├── commands/
│   ├── doc.md                       # Generate Markdown plans
│   ├── doc-yaml.md                  # Generate YAML plans
│   ├── cook-auto.md                 # Autonomous design
│   └── cook-man.md                  # Interactive design
├── skills/
│   ├── plan-executor/
│   │   └── SKILL.md                # Execute plans
│   ├── task-implementer/
│   │   └── SKILL.md                # Implement tasks
│   └── quality-reviewer/
│       └── SKILL.md                # Review quality
├── agents/
│   ├── conductor-orchestrator.md    # Orchestrate workflows
│   └── plan-validator.md            # Validate plans
├── templates/
│   ├── example-plan.yaml            # Example YAML plan
│   └── example-design.md            # Example design doc
└── docs/
    ├── SETUP.md                     # This file
    ├── USAGE.md                     # Usage guide
    └── ARCHITECTURE.md              # Architecture overview
```

## Activating the Plugin

### Option 1: Activate for All Sessions

Add to your `.claude/config.yaml`:

```yaml
plugins:
  enabled:
    - conductor-tools
```

### Option 2: Activate for Current Project

Add to `.claude/config.yaml` in your project root:

```yaml
plugins:
  enabled:
    - conductor-tools
```

### Option 3: Activate Ad-Hoc

```bash
# Use --plugin flag
claude --plugin conductor-tools "Generate a plan for user authentication"

# Or in interactive session
claude
> /use-plugin conductor-tools
```

## Verifying Installation

### Check Commands Are Available

```bash
# List available commands (should include doc, doc-yaml, cook-auto, cook-man)
claude commands

# Test a command
claude "/doc --help"
```

### Check Skills Are Available

```bash
# List skills
claude skills

# Should show:
# - plan-executor
# - task-implementer
# - quality-reviewer
```

### Check Agents Are Available

```bash
# List agents
ls ~/.claude/agents/

# Should include:
# - conductor-orchestrator.md
# - plan-validator.md
```

### Run Example

```bash
# Navigate to a test project
cd /path/to/test/project

# Generate a simple plan
claude "/doc-yaml Add a hello world endpoint"

# Should create: docs/plans/hello-world-endpoint.yaml

# Validate the plan
conductor validate docs/plans/hello-world-endpoint.yaml

# Success! Plugin is working.
```

## Configuration

### Custom Agent Directory

If your agents are in a different location:

```yaml
# .claude/config.yaml
plugins:
  conductor-tools:
    agent_directory: "/custom/path/to/agents"
```

### Default Concurrency

Set default concurrency for conductor execution:

```yaml
# .claude/config.yaml
plugins:
  conductor-tools:
    default_concurrency: 3
    default_timeout: "1h"
```

### Auto-Validation

Enable automatic plan validation after generation:

```yaml
# .claude/config.yaml
plugins:
  conductor-tools:
    auto_validate: true
```

## Troubleshooting

### Problem: Commands not found

**Symptom:**
```
Error: Unknown command '/doc'
```

**Solution:**
```bash
# Check plugin is installed
ls ~/.claude/plugins/conductor-tools/

# Check plugin is enabled
cat ~/.claude/config.yaml | grep conductor-tools

# Restart Claude Code
# Commands should now be available
```

### Problem: Conductor binary not found

**Symptom:**
```
Error: conductor: command not found
```

**Solution:**
```bash
# Build and install conductor
cd /path/to/conductor
go build ./cmd/conductor
sudo mv conductor /usr/local/bin/

# Verify
conductor --version
```

### Problem: Agent files not found

**Symptom:**
```
Warning: Agent 'golang-pro' not found
```

**Solution:**
```bash
# Check if agents exist
ls ~/.claude/agents/

# If missing, install required agents
# Or update plan to use available agents

# List available agents
fd '\.md$' ~/.claude/agents --type f
```

### Problem: Skills not loading

**Symptom:**
```
Error: Skill 'plan-executor' not found
```

**Solution:**
```bash
# Check skill files exist
ls ~/.claude/plugins/conductor-tools/skills/*/SKILL.md

# Check SKILL.md files have proper YAML frontmatter
head -5 ~/.claude/plugins/conductor-tools/skills/plan-executor/SKILL.md

# Should see:
# ---
# name: plan-executor
# description: ...
# ---
```

### Problem: Permission denied errors

**Symptom:**
```
Error: Permission denied: ~/.claude/plugins/conductor-tools/
```

**Solution:**
```bash
# Fix permissions
chmod -R u+rw ~/.claude/plugins/conductor-tools/

# If installed by root, change ownership
sudo chown -R $USER:$USER ~/.claude/plugins/conductor-tools/
```

## Updating the Plugin

### Update from Conductor Repository

Since the plugin is now part of the Conductor monorepo:

```bash
# Update conductor repository
cd /path/to/conductor
git pull origin main

# Copy updated plugin
cp -r plugin ~/.claude/plugins/conductor-tools

# Verify update
ls ~/.claude/plugins/conductor-tools/
```

### Check Version

```bash
# View version in PLUGIN.yaml
cat ~/.claude/plugins/conductor-tools/PLUGIN.yaml | grep version
```

## Uninstalling

### Remove Plugin

```bash
# Remove plugin directory
rm -rf ~/.claude/plugins/conductor-tools/

# Remove from config
# Edit ~/.claude/config.yaml and remove conductor-tools from enabled plugins
```

### Remove Associated Files

```bash
# Optional: Remove generated plans and designs
# (Only if you want to start fresh)
rm -rf docs/plans/
rm -rf docs/designs/
```

## Next Steps

Now that you have the plugin installed:

1. **Read the Usage Guide**: [USAGE.md](./USAGE.md)
2. **Review Architecture**: [ARCHITECTURE.md](./ARCHITECTURE.md)
3. **Try the Examples**: See templates/ directory
4. **Start a Workflow**: Try `/cook-auto` or `/doc-yaml`

## Support

### Documentation

- [Usage Guide](./USAGE.md) - How to use each command and skill
- [Architecture](./ARCHITECTURE.md) - How the plugin works
- [Conductor README](https://github.com/blueman82/conductor/blob/main/README.md) - Conductor documentation

### Getting Help

- **Issues**: https://github.com/blueman82/conductor/issues
- **Documentation**: See plugin docs in the conductor repository
- **Conductor Documentation**: https://github.com/blueman82/conductor/blob/main/README.md

### Contributing

Contributions welcome! See CONTRIBUTING.md for guidelines.

## Summary

You've completed the setup! The conductor-tools plugin is now installed and ready to use for:
- Designing features (cook-auto, cook-man)
- Generating implementation plans (doc, doc-yaml)
- Executing plans (plan-executor skill)
- Implementing tasks (task-implementer skill)
- Quality control (quality-reviewer skill)

Happy conducting!
