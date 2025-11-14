# Installation Guide - Conductor Tools Plugin

## Prerequisites

- **Claude Code** v1.0 or later (installed and running)
- **Git** (optional, for manual installation)
- **Conductor** (to execute generated plans) - [Install here](https://github.com/blueman82/conductor#installation)

## Installation Methods

### Method 1: Via Claude Code Plugin Manager (Recommended)

**Easiest method - requires no terminal commands:**

1. **Open Claude Code**
2. **Run the plugin command**:
   ```
   /plugin
   ```
3. **Select "Browse Plugins"**
4. **Search for** `conductor-tools`
5. **Click "Install"**
6. **Restart Claude Code** (optional, may auto-load)

Done! Commands are now available: `/doc`, `/doc-yaml`, `/cook-man`

### Method 2: Manual Installation from Git

**For developers or offline installations:**

```bash
# Clone the conductor repository
git clone https://github.com/blueman82/conductor.git
cd conductor

# Navigate to plugin directory
cd plugin/conductor-tools

# Copy to Claude Code plugins directory
cp -r . ~/.claude/plugins/conductor-tools

# Restart Claude Code
```

Commands will be available in your next Claude Code session.

### Method 3: Local Development Installation

**For contributing to the plugin:**

```bash
# Clone conductor repo
git clone https://github.com/blueman82/conductor.git
cd conductor/plugin/conductor-tools

# In Claude Code, run:
/plugin marketplace add ./plugin/conductor-tools
/plugin install conductor-tools@dev-marketplace
```

This creates a symlink to your local plugin directory, allowing real-time editing.

## Verification

After installation, verify the commands are available:

**In Claude Code, run:**

```bash
/help
```

You should see:
- `/doc` - Generate Markdown implementation plan
- `/doc-yaml` - Generate YAML implementation plan
- `/cook-man` - Interactive plan design session

Or test directly:

```bash
/doc "test feature"
```

Should show the planning interface.

## Setup for Conductor Integration

To execute generated plans with Conductor, ensure it's installed:

```bash
# Check if Conductor is installed
conductor --version

# If not installed, see:
# https://github.com/blueman82/conductor#installation
```

## Troubleshooting

### Commands Not Appearing

**Problem**: Commands don't show up after installation

**Solutions**:
1. **Restart Claude Code**: Close and reopen Claude Code
2. **Verify installation**: `/plugin list` should show `conductor-tools`
3. **Check directory**: `~/.claude/plugins/conductor-tools/` should exist
4. **Reinstall**: Remove plugin and reinstall via Method 1

### Plugin Won't Install

**Problem**: "Plugin not found" error when browsing

**Solutions**:
1. **Check internet connection**: Plugin marketplace requires network access
2. **Update Claude Code**: Ensure you have the latest version
3. **Manual install**: Use Method 2 (manual Git installation)
4. **Clear cache**: Delete `~/.claude/plugins/.cache/` and reinstall

### Commands Execute But Generate Errors

**Problem**: Commands run but produce no plan output

**Solutions**:
1. **Provide better description**: Be specific about the feature
   - ❌ Bad: `/doc "new feature"`
   - ✅ Good: `/doc "Add OAuth2 authentication with JWT tokens"`

2. **Ensure codebase is available**: Commands analyze your project
   - Navigate to your project directory before running `/doc`
   - Commands need to see `README.md`, `package.json`, etc.

3. **Check Claude Code settings**:
   - Ensure you have enough token quota
   - Check that Claude Code is authenticated
   - Verify network connection

4. **Use interactive mode**: Try `/cook-man` if `/doc` isn't producing good results
   - Interactive design may yield better results for complex features

### Plans Generated But Format Is Wrong

**Problem**: Generated plan doesn't match Conductor format

**Solutions**:
1. **Use `/doc-yaml`**: YAML format is more strictly parsed
2. **Check task format**: See [Conductor Plan Format Guide](https://github.com/blueman82/conductor/blob/main/docs/conductor.md#plan-format)
3. **Provide more detail**: Add requirements to your feature description
   - Include tech stack, dependencies, constraints
4. **Use `/cook-man`**: Interactive mode helps validate format before generation

## Uninstallation

To remove the plugin:

```bash
# Via Claude Code
/plugin uninstall conductor-tools

# Or manually
rm -rf ~/.claude/plugins/conductor-tools/

# Restart Claude Code
```

## Updates

### Checking for Updates

```bash
/plugin info conductor-tools
```

Shows current version and available updates.

### Updating the Plugin

**Automatic** (recommended):
- Claude Code checks for updates automatically
- New versions are installed when available

**Manual**:
```bash
/plugin update conductor-tools
```

Or reinstall using one of the installation methods above.

## Getting Help

If installation fails or commands don't work:

1. **Check the logs**:
   ```bash
   /help  # See command documentation
   ```

2. **Review documentation**:
   - [Plugin README](./README.md)
   - [Conductor Main Docs](https://github.com/blueman82/conductor)
   - [Plan Format Guide](https://github.com/blueman82/conductor/blob/main/docs/conductor.md#plan-format)

3. **Report issues**:
   - [GitHub Issues](https://github.com/blueman82/conductor/issues)
   - [GitHub Discussions](https://github.com/blueman82/conductor/discussions)

4. **Questions about Conductor**:
   - See [Conductor Support](https://github.com/blueman82/conductor#support)

## Next Steps

After installation:

1. **Generate your first plan**:
   ```bash
   /doc "Your feature description"
   ```

2. **Review the generated plan** (markdown or YAML)

3. **Execute with Conductor**:
   ```bash
   conductor run generated-plan.md --verbose
   ```

4. **Monitor execution**:
   ```bash
   conductor learning stats generated-plan.md
   ```

Happy planning! 🚀
