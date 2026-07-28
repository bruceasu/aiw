﻿# AIW Design Overview

## Goal

AIW is a workflow-first CLI. Its job is to organize work, preserve task state, and provide a stable project structure for AI-assisted execution.

The project now uses four layers:

1. Core workflow
   - task lifecycle
   - OpenSpec-lite change and spec folders
   - worktree management
   - context, decision, registry, and archive operations

2. Capability layer
   - Skills that encode reusable methods and domain knowledge
   - Skill installation and discovery for Codex and AIW workflows

3. AI support layer
   - `aiw-flow` for automation-oriented AI process execution
   - `aiw cxs` for Codex session inspection and continuation support
   - interactive session behavior, handoff artifacts, and task-agent lineage

4. Plugin layer
   - external executable commands such as `git`, `github`, `cz`, and `tcc`
   - runtime discovery and help metadata for commands outside the core

## Boundary Rules

- AIW core owns workflow structure and persistence.
- Skills own reusable capability content.
- AI support commands can use skills and workflow context, but they do not redefine the core product.
- Plugins extend the CLI without becoming workflow authority.
- OpenSpec documents describe expected behavior; they do not replace the workflow model.

## Spec Mapping

### Core workflow package
- `workflow`

### Capability package
- `capabilities`

### AI support package
- `ai-support`

### Plugin package
- `plugins`

## Maintenance Guidance

- If a change affects the workflow model, update this design overview first.
- If a change affects a stable behavior contract, update the matching package under `openspec/specs/`.
- If a change affects discovery, help text, or runtime metadata, keep the capability and plugin layers separate.
- If a change introduces a new AI execution path, classify it first as workflow, capability, AI support, or plugin before adding docs or code.
