# Conductor Tools Plugin

**Implementation Plan Generator for Conductor**

Comprehensive slash commands that generate Conductor-ready implementation plans (Markdown or YAML) from natural language descriptions.

## What This Plugin Does

Adds three powerful plan generation commands to Claude Code:

- **`/doc`** - Generate detailed Markdown implementation plans
- **`/doc-yaml`** - Generate structured YAML implementation plans
- **`/cook-man`** - Interactive design session with plan generation

All generated plans are immediately executable with [Conductor](https://github.com/blueman82/conductor).

## Features

✅ **Conductor-Optimized**: Plans follow Conductor's task metadata format
✅ **Codebase-Aware**: Analyzes your project before generating tasks
✅ **Agent Assignment**: Automatically suggests appropriate agents
✅ **Dependency Detection**: Identifies task dependencies
✅ **Multiple Formats**: Markdown for humans, YAML for tooling
✅ **Interactive Design**: Optional Q&A-driven refinement before planning

## Quick Start

### Installation

```bash
# In Claude Code, run:
/plugin
```

Select "Browse Plugins" → Find "conductor-tools" → Install

Or install directly:
```bash
git clone https://github.com/blueman82/conductor.git
cd conductor/plugin/conductor-tools
# Then drag the folder to Claude Code's plugin manager
```

### Usage

```bash
# Generate Markdown plan
/doc "Add OAuth2 authentication to user service"

# Generate YAML plan
/doc-yaml "Implement payment processing with Stripe"

# Interactive design-first approach
/cook-man "Build notification system"
```

## Generated Plan Format

All plans include:
- Structured tasks with clear names
- File modifications tracked
- Task dependencies calculated
- Estimated execution time
- Appropriate agent assignments
- Testing strategy
- Commit boundaries

Example generated task:

```markdown
## Task 1: Setup Authentication Framework
**File(s)**: internal/auth/auth.go, internal/auth/auth_test.go
**Depends on**: None
**Estimated time**: 45m
**Agent**: backend-engineer

Implement JWT-based authentication with user login/signup endpoints.

**Requirements:**
- JWT token generation and validation
- Password hashing with bcrypt
- Token refresh mechanism
```

## Usage with Conductor

1. **Generate a plan** using any command:
   ```bash
   /doc "Build user dashboard"
   ```

2. **Execute immediately** with Conductor:
   ```bash
   conductor run generated-plan.md --verbose
   ```

3. **Monitor execution**:
   ```bash
   conductor learning stats generated-plan.md
   ```

## Command Details

### `/doc` - Markdown Plans

Best for:
- Human-readable documentation
- Quick iterations and team discussion
- Embedding in project README
- Git-friendly diffs

Generates `.md` files with narrative descriptions and code examples.

### `/doc-yaml` - YAML Plans

Best for:
- Automation and tooling integration
- Configuration management
- Strict formatting requirements
- Parsing and processing

Generates `.yaml` files with structured task definitions.

### `/cook-man` - Interactive Design

Best for:
- Complex features requiring validation
- Team alignment before implementation
- Requirement refinement
- Risk assessment

Guides you through a Q&A session, validates the design, then generates the plan.

## Requirements

- Claude Code v1.0+
- [Conductor](https://github.com/blueman82/conductor) installed (to execute generated plans)
- Git clone of your project (for codebase analysis)

## Examples

### Example 1: REST to GraphQL Migration

```bash
/doc-yaml "Migrate REST API to GraphQL with Apollo Server"
```

Generates:
- Schema design tasks
- Resolver implementation
- Testing strategy
- Deployment plan

### Example 2: Feature Implementation

```bash
/doc "Add real-time notifications with WebSockets"
```

Generates:
- Architecture design
- Backend implementation
- Frontend integration
- Testing and deployment

### Example 3: Design-First Approach

```bash
/cook-man "Implement multi-tenant SaaS architecture"
```

Generates:
- Interactive questions about requirements
- Design validation with user
- Comprehensive plan automatically

## Troubleshooting

**Command not found?**
- Ensure plugin is installed: `/plugin list`
- Restart Claude Code
- Check that you're in the correct directory

**Plan missing details?**
- Provide more specific feature description
- Use `/cook-man` for interactive refinement
- Check codebase is available in current directory

**Generated plan format incorrect?**
- Use `/doc-yaml` for strict YAML formatting
- Check task syntax in [Conductor documentation](https://github.com/blueman82/conductor/blob/main/docs/conductor.md)

## Documentation

- **[Conductor Main Docs](https://github.com/blueman82/conductor)** - Orchestrator documentation
- **[Plan Format Guide](https://github.com/blueman82/conductor/blob/main/docs/conductor.md#plan-format)** - Task metadata specifications
- **[Usage Examples](https://github.com/blueman82/conductor/blob/main/docs/examples/)** - Real plan examples

## Contributing

Found a bug or want to improve the commands?

- **Report issues**: [GitHub Issues](https://github.com/blueman82/conductor/issues)
- **Contribute**: See [Conductor Contributing Guide](https://github.com/blueman82/conductor#contributing)

## License

MIT - See [LICENSE](./LICENSE) for details

## Support

- 📖 [Conductor Documentation](https://github.com/blueman82/conductor)
- 💬 [GitHub Discussions](https://github.com/blueman82/conductor/discussions)
- 🐛 [Report Issues](https://github.com/blueman82/conductor/issues)
