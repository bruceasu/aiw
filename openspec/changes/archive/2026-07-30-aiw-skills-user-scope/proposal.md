## Problem Statement

`aiw skills install` only writes to the current project's `.agents/skills`.
Users who work across multiple projects need a shared user-level Skill catalog
that is usable by both Codex and GitHub Copilot CLI.

## Solution

Add an explicit `--scope project|user` option. Project scope remains the
default; user scope writes to the current user's `.agents/skills` directory.
The implementation does not add agent-specific `~/.codex/skills` or
`~/.copilot/skills` targets.

## User Stories

1. As a project maintainer, I want existing commands to remain project-scoped by default, so that current workflows remain compatible.
2. As a developer using multiple projects, I want to install a Skill into user-level `.agents/skills`, so that compatible Agents share one copy.
3. As an automation author, I want scope and destination in command results, so that scripts can verify where a Skill was installed.
4. As a developer, I want discovery, adoption, and synchronization to use the selected scope, so that management operations affect the intended catalog.

## Implementation Decisions

- Use one scope-aware destination resolver for install, discover, adopt, and sync.
- Resolve project scope from the current directory and user scope from the platform standard home directory.
- Keep existing validation, staging, digest, manifest, idempotency, and unmanaged-destination protection rules for both scopes.

## Testing Decisions

- Extend the existing subprocess CLI tests with isolated user-home cases for install and discover, plus result and manifest assertions.
- Do not run tests automatically during implementation.

## Out of Scope

- Codex-specific or Copilot-specific installation directories.
- Arbitrary custom target paths, removal, or linking.

## Further Notes

%% The test harness uses isolated HOME and USERPROFILE values; no new runtime public override is introduced.

