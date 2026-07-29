﻿# AIW Tools

AIW is a workflow-first CLI. Use this catalog to understand which capabilities belong to the core workflow, which belong to the capability layer, which belong to AI support, and which are external plugins.

## Discovery First

Before unfamiliar or mutating operations, inspect runtime discovery:

- `aiw help --json`
- `aiw help <command>`
- Plugin-specific `--help`

Runtime discovery is authoritative for the current installation. Prefer it over static lists when deciding whether a command is core, auxiliary, or plugin-provided.

## Layer Map

### Core Workflow

Use AIW core commands as the authority for project structure, Task lifecycle,
branch, worktree, Session, and handoff state. Use OpenSpec for requirement and
implementation artifacts.

### Capability Layer

Use Skills when you need reusable methods or domain-specific behavior that can strengthen Codex or AIW workflows.

### AI Support Layer

Use AI support commands when you need AI-assisted execution or session management.

### Plugin Layer

Use plugins for external commands and specialized extensions.

## File Operations

Use the shared file-operation tools for project text work:

- `aiw file read` for reading project text
- `aiw file info` for inspecting file metadata
- `aiw file write` for text writes
- `aiw patch` for AI-generated code patches

Prefer these tools over ad hoc shell redirection or handwritten file-edit scripts when the task is about repository content.

For repository search, use `rg`. For repository inspection, use Git and native commands. For tests and builds, use the repository's standard commands.

## Safety Notes

- Do not hard-code absolute plugin paths.
- Inspect runtime metadata before unfamiliar or mutating plugin operations.
- Treat legacy META fields as backward-compatible input, not as the source of truth.
- When a capability can modify files, state the side effects and confirmation requirements before using it.

## Practical Order

When working in AIW, the usual order is:

1. Resolve or create the AIW Task.
2. Read its OpenSpec artifacts.
3. Check runtime discovery once before unfamiliar mutations.
4. Create or resolve its worktree with `aiw wt` before implementation.
5. Use Skills for method support.
6. Use AI support or plugins only when the task needs them.
