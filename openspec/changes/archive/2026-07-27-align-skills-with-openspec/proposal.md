## Why

Several bundled engineering Skills treat an issue tracker, including a separate
`.scratch/` Markdown layout, as the canonical place for specifications, tickets,
and execution state. AIW already provides an OpenSpec-compatible local task,
branch, worktree, status, and archive workflow, so the duplicate model creates
unnecessary setup choices and allows local state to drift.

## What Changes

- Make OpenSpec the canonical local source of truth for engineering Skills in an
  AIW/OpenSpec repository.
- Replace the separate Local Markdown issue layout with OpenSpec change
  artifacts: proposal, design, capability specs, tasks, notes, and AIW task
  metadata where available.
- Define how Skills resolve the active change, workspace, branch, and work item
  before reading or modifying task state.
- Map ordinary tracer-bullet tickets to checklist items in `tasks.md`; use a
  separate OpenSpec change only when work needs an independent lifecycle or
  worktree.
- Treat GitHub and GitLab as optional external projections invoked explicitly,
  rather than asking users to choose a tracker during normal local setup.
- Add an explicit OpenSpec-to-GitHub Issue publishing workflow that renders a
  bounded Issue projection and delegates GitHub transport to `aiw-github`.
- Extend `aiw-github` with the transport primitives required for reliable,
  repeatable publication, including body-file input and Issue updates.
- **BREAKING**: bundled engineering Skills will no longer create canonical
  specifications or tickets under `.scratch/` in an AIW/OpenSpec repository.

## Capabilities

### New Capabilities

- `skill-work-management`: Defines OpenSpec-canonical local work management,
  active work-context resolution, artifact mapping, task granularity, lifecycle
  updates, and optional external Issue projection for bundled Skills.

### Modified Capabilities

- `plugins`: Adds reliable `aiw-github` Issue publication primitives and aligns
  its documented command surface with its actual parser.

## Impact

- Affects engineering Skill definitions under `skills`, especially
  setup, routing, specification, ticketing, implementation, review, and handoff
  Skills.
- Replaces the repository guidance in `docs/agents/issue-tracker.md` with
  OpenSpec-canonical work-management guidance.
- Affects `plugins/aiw-github`, its README, command behavior, and tests.
- Adds OpenSpec-to-GitHub projection guidance and a place under each change to
  record external publication references.
- Does not introduce a GitHub- or GitLab-first workflow, automatic publication,
  or bidirectional synchronization.
