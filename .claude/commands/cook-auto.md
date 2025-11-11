---
allowed-tools: Read, Bash(git status*), Bash(git log*), Bash(ls*), Glob, Grep, mcp__ai-counsel__deliberate, SlashCommand, AskUserQuestion
argument-hint: "feature description for autonomous multi-model design"
description: Autonomous AI-powered design session - uses multi-model deliberation to design features collaboratively
---

# Cook Auto - Autonomous Multi-Model Design Session

Help the user turn a rough idea into a fully formed design by using AI Counsel to deliberate on each design question autonomously.

## Process Overview

This command uses a meta-pattern where multiple AI models collaborate to answer design questions:
1. Identify a design question
2. Use AI Counsel deliberation (quick mode) to get multi-model perspectives
3. Summarize the consensus/recommendations
4. Move to the next design question
5. Repeat until all aspects are covered
6. Present the complete design

## Phase 1: Project Context Analysis

First, analyze the current state of the project to understand the starting point:

1. **Check git status** to understand what branch we're on and current changes
2. **Review project structure** using ls/glob to understand the codebase organization
3. **Read key files** like README.md, package.json, or other configuration files to understand the tech stack and architecture
4. **Identify related code** using grep if the user mentions specific features or components

## Phase 2: Autonomous Design Through Multi-Model Deliberation

For each critical design aspect, **AUTONOMOUSLY** deliberate using AI Counsel:

**Process for each design question:**
1. **State the design question** clearly
2. **Immediately call mcp__ai-counsel__deliberate** in quick mode with:
   - The design question
   - Relevant context from the codebase
   - At least 2-3 diverse AI models (e.g., Claude Sonnet, Claude Opus, GPT-5)
3. **Summarize the deliberation results** - what did the models recommend?
4. **Move to the next design question**

**Focus on these critical aspects:**
- Purpose and goals (What problem does this solve? Who is it for?)
- Scope boundaries (What's in scope vs out of scope?)
- Technical approach (How should this integrate with existing code?)
- User experience (How will users interact with this?)
- Data models and state management (if applicable)
- API design and interfaces (if applicable)
- Success criteria (How will we know this is working?)
- Dependencies and constraints (What are we working with/around?)
- Testing strategy

Continue deliberating on questions until you have a clear understanding of:
- The problem being solved
- The proposed solution approach
- Key technical decisions
- Integration points with existing code
- Success metrics

## Phase 3: Design Presentation

Once all deliberations are complete, present the comprehensive design synthesized from the multi-model input:

1. **Present the complete design** incorporating insights from all deliberation rounds
2. **Organize by the sections you deliberated on**
3. **Attribute key insights** to the AI models when relevant (e.g., "Claude Opus suggested...", "GPT-5 recommended...")
4. **Highlight areas of consensus vs. debate** among the models
5. **Include trade-offs and alternatives** that were discussed

Typical sections to cover (based on deliberations):
- **Overview & Objectives** (what we're building and why)
- **Architecture & Technical Approach** (how it fits into the existing system)
- **User Experience & Interface** (how users will interact with it)
- **Data Model & State Management** (if applicable)
- **API Design & Interfaces** (if applicable)
- **Implementation Considerations** (key technical decisions, trade-offs)
- **Testing Strategy** (how we'll validate it works)
- **Success Metrics** (how we'll measure success)

## Phase 4: User Approval Checkpoint

After presenting the complete design, use `AskUserQuestion` to get approval before proceeding:

**Question**: "How would you like to proceed with this design?"
- **Option 1**: "Approve and generate implementation plan" → Proceed to Phase 5
- **Option 2**: "Revise specific section" → Ask which section needs changes, re-deliberate
- **Option 3**: "Re-deliberate on different aspect" → Ask which aspect, run new deliberation
- **Option 4**: "Start over with different approach" → Begin fresh from Phase 2

## Phase 5: Generate Implementation Plan

After the user approves the design, **AUTOMATICALLY** invoke the `/doc` command to generate a comprehensive implementation plan:

1. **Summarize the feature** from the deliberations into a concise description
2. **Call `/doc [feature description]`** using the SlashCommand tool
3. **Let the doc command run** - it will analyze the codebase and generate the detailed plan

This creates a complete workflow: Design → Deliberate → Approve → Plan → Implement

## Execution Guidelines

- **Be autonomous during deliberation** - don't wait for the user to answer design questions; use AI Counsel to get answers
- **Get human approval at checkpoints** - use `AskUserQuestion` after presenting the complete design (Phase 4)
- **Use diverse models** - include at least 2-3 different AI models in each deliberation (Claude Sonnet, Claude Haiku, GPT-5, Gemini-2.5-Pro, etc.)
- **Quick mode is preferred** - use mode="quick" for faster single-round deliberations
- **Provide rich context** - include relevant code snippets, requirements, and constraints in each deliberation
- **Synthesize, don't just repeat** - summarize what the models agreed on and where they differed
- **Show your work** - let the user see how the multi-model collaboration shaped the design
- **Move efficiently** - don't over-deliberate; aim for 5-8 key design questions
- **Handle disagreements** - if models strongly disagree on a critical decision, use `AskUserQuestion` to get user input

Remember: This is a **meta-design pattern** - you're using AI Counsel itself to collaboratively design features. Show the power of multi-model collaboration!
