# Work Management Contract

Use this contract for engineering Skills running in an AIW/OpenSpec
repository. OpenSpec is the canonical local source of truth; GitHub and GitLab
are optional external projections only when the user explicitly asks for them.

## Resolve the active change

Resolve context in this order:

1. A change identifier explicitly supplied by the user.
2. A change already established by the active session or conversation.
3. The AIW task associated with the current worktree or branch.
4. The only active OpenSpec change, when exactly one exists.

If no unique change can be resolved, stop before writing and ask for the change
identifier. Do not infer a target from `.scratch` and do not create a parallel
issue hierarchy.

## Artifact roles

- `proposal.md`: motivation, scope, and impact.
- `issue.md`: optional source context; never the normative specification.
- `design.md`: architecture and implementation decisions.
- `specs/<capability>/spec.md`: normative requirements and scenarios.
- `tasks.md`: ordered implementation slices, TODOs, verification, and `%%`
  notes.
- `notes.md`: temporary investigation material.
- `task.toml`: AIW status, branch, worktree, and other supported task metadata.

## Task granularity

Use a numbered `tasks.md` item when work shares one goal, branch, lifecycle,
worktree, and delivery boundary. Create a separate OpenSpec change only when a
slice needs an independent lifecycle or worktree.

## Mutation rules

Before implementation changes, verify that the current branch and worktree
match the resolved AIW task metadata. After implementation, update the selected
task item, TODO, Verification, and remaining `%%` risks or questions.

## External publication

Normal planning and implementation remain local. An explicit GitHub publication
request uses the `publish-github-issue` Skill and `aiw-github`; it records a
mapping under `openspec/changes/<change-id>/external/github.json`. OpenSpec
remains authoritative, and remote edits do not silently update local state.

## Examples

- Existing change: `/to-spec add context to configure-python-interpreter` reads
  and updates the resolved OpenSpec change.
- New change: when no unique change exists, `/to-spec` asks for the change ID or
  creates one through the OpenSpec workflow before writing.
- Ambiguous context: `/implement` stops when several active changes exist and
  no change, session, branch, or worktree identifies one uniquely.
- Independent worktree: `/to-tickets` proposes a separate OpenSpec change when
  a slice needs its own branch, worktree, status, or archive lifecycle.
