---
name: setup-matt-pocock-skills
description: Configure AIW Task lifecycle, OpenSpec artifact management, triage labels, and domain-doc conventions for the engineering Skills.
disable-model-invocation: true
---

# Setup Engineering Skills

Read `skills/work-management.md`.

## Inspect

Inspect only:

- existing `AGENTS.md` or `CLAUDE.md`;
- AIW markers such as `task.toml`, `.wt/`, and available runtime help;
- `openspec/changes/` and `openspec/specs/`;
- `docs/agents/`, `CONTEXT.md`, `CONTEXT-MAP.md`, and ADR directories;
- whether the `triage` Skill is installed;
- clear monorepo markers.

Treat `.scratch` as legacy data.

## Configure

Record this ownership split in `docs/agents/work-management.md`:

- AIW owns Task lifecycle, branch, worktree, Session, and handoff state.
- OpenSpec owns proposal, design, capability specs, and `tasks.md`.
- External Issues are optional projections used only on explicit request.

Do not create a separate issue-tracker configuration for an AIW/OpenSpec
repository.

If `triage` is installed, ask once whether to keep the five default role labels:
`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, and
`wontfix`. Write `docs/agents/triage-labels.md` only when needed.

Use a root `CONTEXT.md` and `docs/adr/` by default. Offer a multi-context layout
only when clear monorepo boundaries exist.

## Update Agent Instructions

Update the existing `AGENTS.md` or `CLAUDE.md`; do not create the other file.
If neither exists, ask which one to create.

Add or update one `## Agent skills` block that points to:

- `docs/agents/work-management.md`;
- `docs/agents/domain.md`;
- `docs/agents/triage-labels.md` when triage is configured.

Show the proposed block and docs before writing. Preserve surrounding user
content and avoid duplicate sections.

## Finish

Report the files updated and the ownership split. Do not create a Task,
worktree, external Issue, or run tests during setup.
