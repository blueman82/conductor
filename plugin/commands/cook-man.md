---
allowed-tools: Read, Bash(git status*), Bash(git log*), Bash(ls*), Glob, Grep, AskUserQuestion, SlashCommand
argument-hint: "feature description for interactive design"
description: Interactive design session - work with the user through questions to refine and validate the design
---

# Cook Manual - Interactive Design Session

Help the user turn a rough idea into a fully formed design through interactive questioning and incremental validation.

## Process Overview

This is an interactive, user-driven design process:
1. Analyze the current project state
2. Ask the user questions **one at a time** to refine the idea
3. Present the design in digestible sections (200-300 words)
4. Validate each section before continuing
5. Generate implementation plan once design is approved

## Phase 1: Project Context Analysis

First, analyze the current state of the project to understand the starting point:

1. **Check git status** to understand what branch we're on and current changes
2. **Review project structure** using ls/glob to understand the codebase organization
3. **Read key files** like README.md, package.json, or other configuration files to understand the tech stack and architecture
4. **Identify related code** using grep if the user mentions specific features or components

Present a brief summary of what you found to the user.

## Phase 2: Interactive Refinement Through Questions

Ask the user questions **one at a time** to refine and clarify the design. Critical: **ONLY ONE QUESTION PER MESSAGE**.

**Question Format:**
- **Prefer multiple choice** using the `AskUserQuestion` tool
- Provide 2-4 clear options with descriptions
- Open-ended questions are OK when necessary, but multiple choice is better
- Wait for the user's answer before asking the next question

**Focus areas to explore:**
- Purpose and goals (What problem does this solve? Who is it for?)
- Scope boundaries (What's in scope vs out of scope?)
- Technical approach (How should this integrate with existing code?)
- User experience (How will users interact with this?)
- Data models and state management (if applicable)
- API design and interfaces (if applicable)
- Success criteria (How will we know this is working?)
- Dependencies and constraints (What are we working with/around?)
- Testing strategy

**Important:** Continue asking questions until you have a clear understanding of:
- The problem being solved
- The proposed solution approach
- Key technical decisions
- Integration points with existing code
- Success metrics

## Phase 3: Incremental Design Presentation

Once you understand what the user wants, stop asking questions and present the design.

**Present the design in sections:**
1. Break the design into logical sections (Overview, Architecture, UX, Data Model, API, Testing, etc.)
2. Present **ONE section at a time** (200-300 words per section)
3. After each section, **use `AskUserQuestion`** to validate:
   - "Does this [section name] look good?"
   - Option 1: "Approve this section" → Move to next section
   - Option 2: "Request changes" → Ask what to modify, then revise
   - Option 3: "Rewrite completely" → Start this section over
4. Wait for confirmation before presenting the next section
5. Make adjustments based on user feedback

**Typical sections:**
- **Overview & Objectives** (what we're building and why)
- **Architecture & Technical Approach** (how it fits into the existing system)
- **User Experience & Interface** (how users will interact with it)
- **Data Model & State Management** (if applicable)
- **API Design & Interfaces** (if applicable)
- **Implementation Considerations** (key technical decisions, trade-offs)
- **Testing Strategy** (how we'll validate it works)
- **Success Metrics** (how we'll measure success)

## Phase 4: Generate Implementation Plan

After the user approves all sections of the design, **AUTOMATICALLY** invoke the `/doc` command to generate a comprehensive implementation plan:

1. **Summarize the feature** from the validated design into a concise description
2. **Call `/doc [feature description]`** using the SlashCommand tool
3. **Let the doc command run** - it will analyze the codebase and generate the detailed plan

This creates a complete workflow: Question → Design → Validate → Plan → Implement

## Execution Guidelines

- **Be patient** - wait for user responses before proceeding
- **One question at a time** - never ask multiple questions in one message
- **Prefer multiple choice** - use `AskUserQuestion` tool for structured options
- **Validate incrementally** - present design sections one at a time, waiting for approval
- **Listen and adapt** - incorporate user feedback into the design
- **Show your work** - explain your reasoning when presenting design sections
- **Keep sections digestible** - aim for 200-300 words per section

Remember: This is a **collaborative process** - the user drives the design through their answers and feedback!
