# Proposal: AIW Capability Discovery

## Problem Statement

AIW is now organized as a workflow-first core with three supporting layers: capabilities, AI support, and plugins. Codex and AIW templates still need a stable way to discover those layers without relying on stale command lists or absolute filesystem paths.

## Solution

Provide a stable discovery contract for the capability, AI support, and plugin layers. Human-readable catalogs and runtime help/discovery commands point to the right layer, while plugin metadata remains owned by each plugin and runtime output exposes normalized capability records.

## User Stories

1. As an AI assistant, I want to discover available AIW capabilities at runtime, so that I do not rely on stale command lists.
2. As an AI assistant, I want metadata paths and side-effect declarations, so that I can choose tools safely.
3. As a developer, I want Codex templates to contain stable discovery instructions, so that worktrees and installed builds use the same guidance.
4. As a maintainer, I want plugin metadata to remain local to each plugin, so that capabilities can evolve independently.

## Implementation Decisions

- Add a human-readable capability catalog under docs/agents.
- Add machine-readable help/discovery output for commands and plugins.
- Normalize metadata fields for purpose, invocation, metadata source, read/write behavior, confirmation, and output format.
- Use repository-relative or runtime-resolved paths; never require absolute paths in templates.
- Keep dynamic capability records authoritative over static documentation.

## Testing Decisions

- Test JSON output shape and plugin discovery.
- Test missing, malformed, and legacy plugin metadata.
- Test that Codex templates mention discovery without embedding absolute paths.
- Test read-only and mutating capability declarations.

## Out of Scope

- Replacing plugin execution.
- Automatically installing plugins.
- Publishing capability catalogs to external services.
- Loading every plugin implementation into the prompt.

## Further Notes

%% Existing plugins use different META shapes; normalization should be additive and backward compatible.

