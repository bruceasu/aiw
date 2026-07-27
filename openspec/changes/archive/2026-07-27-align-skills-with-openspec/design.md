## Context

AIW stores local work under `openspec/changes/<change-id>/`, tracks stable
requirements under `openspec/specs/`, and associates a task with a branch and
worktree through `task.toml`. Several bundled engineering Skills instead assume
that specifications and tickets belong to a configured issue tracker. Their
Local Markdown profile creates a second hierarchy under `.scratch/`, while
`implement` and related Skills operate on the current directory or branch
without resolving the corresponding AIW task.

The repository also contains `aiw-github`, which already creates, reads,
comments on, labels, and closes GitHub Issues. It is a suitable transport
boundary, but it does not understand OpenSpec artifacts, update an existing
Issue body, accept a body file, or persist a publication mapping.

The design must minimize user choices, keep OpenSpec authoritative locally,
preserve the ability to publish externally when explicitly requested, and avoid
inventing unsupported Skill frontmatter.

## Goals / Non-Goals

**Goals:**

- Give bundled engineering Skills one default local work-management model in an
  AIW/OpenSpec repository.
- Make local issue-like work compatible with the existing OpenSpec change
  structure instead of creating `.scratch` artifacts.
- Resolve task, workspace, branch, and artifact context consistently before a
  Skill performs work.
- Keep ordinary implementation slices lightweight while preserving independent
  AIW tasks for independently managed work.
- Publish an OpenSpec change to GitHub Issues explicitly and repeatably through
  the existing `aiw-github` plugin.
- Keep local OpenSpec artifacts authoritative after publication.

**Non-Goals:**

- Automatically publishing every OpenSpec change.
- Bidirectional synchronization of GitHub comments, labels, or edits.
- Introducing a GitHub- or GitLab-first development workflow.
- Implementing GitLab publication in this change.
- Deleting or automatically migrating existing `.scratch` content.
- Adding dependency-graph fields to `task.toml`.
- Changing AIW Skill discovery or invocation semantics.

## Decisions

### Use OpenSpec as the single local work manager

In an AIW/OpenSpec repository, engineering Skills will treat
`openspec/changes/<change-id>/` as the canonical local work container and
`openspec/specs/` as the canonical stable-requirement store. The default setup
will not ask the user to select GitHub, GitLab, or Local Markdown.

The artifact roles are:

- `proposal.md`: motivation, scope, and impact.
- `issue.md`: optional source material and user-oriented context; never the
  normative specification.
- `design.md`: architecture and implementation decisions.
- `specs/<capability>/spec.md`: change requirements and scenarios.
- `tasks.md`: implementation slices, progress, verification, and `%%` notes.
- `notes.md`: temporary investigation material.
- `task.toml`: AIW lifecycle, branch, and worktree metadata where present.

Keeping `.scratch` as another supported local canonical store was considered,
but rejected because it preserves the ambiguity and drift this change is meant
to remove.

### Put work-manager rules in shared Skill guidance

A shared `work-management` reference will describe how Skills resolve and
update OpenSpec work. A concise repository document under `docs/agents/` will
declare OpenSpec as canonical and external trackers as optional projections.
Affected Skills will reference that contract instead of embedding backend
choices and `.scratch` paths independently.

Custom Skill frontmatter was considered, but rejected because the current Skill
loader only requires standard metadata and does not define work-manager fields.

### Resolve work context deterministically

Before modifying work state, an affected Skill will resolve context in this
order:

1. A change explicitly named by the user.
2. A change already established by the active session or conversation.
3. The AIW task associated with the current worktree or branch.
4. The only active OpenSpec change when exactly one exists.

If no unique change can be resolved, the Skill will stop before writing and ask
for the missing change identifier. It will not infer a target from `.scratch`
or create a parallel work hierarchy.

Implementation Skills will verify that the resolved branch and worktree agree
with `task.toml` before changing code. Read-only Skills may inspect a resolved
change without entering its worktree.

### Map tracer bullets to tasks before creating more changes

A tracer-bullet ticket that shares one goal, branch, lifecycle, and delivery
unit with its siblings will become a numbered checklist item in `tasks.md`. A
separate OpenSpec change will be created only when the slice needs an
independent worktree, status, archive lifecycle, or delivery boundary.

This preserves the existing AIW convention of one task per branch and worktree
without requiring parent or blocking metadata in `task.toml`. Dependency order
inside one change will be represented by task numbering and explicit wording
where necessary.

### Keep external Issue publication explicit and one-way

A new publishing Skill will read a resolved OpenSpec change, render a bounded
GitHub Issue projection, and call `aiw-github`. Normal specification,
ticketing, and implementation Skills will not publish externally.

The generated body will contain managed markers, a scope summary, key
requirements, task progress, and the local change identifier. Detailed design
notes and temporary findings will remain local unless the user explicitly asks
to include them.

OpenSpec remains authoritative. Closing or editing the GitHub Issue will not
silently update local status.

### Persist GitHub mappings outside task.toml

Publication state will be stored at:

`openspec/changes/<change-id>/external/github.json`

The record will include a format version, repository, Issue number, URL, and
the last published content hash or timestamp. A later publish will update the
mapped Issue rather than create a duplicate. If the mapped Issue cannot be
read, the workflow will fail visibly instead of silently creating another one.

Adding arbitrary keys to `task.toml` was rejected because the current AIW
reader/writer owns a fixed field set and can discard unknown content when it
rewrites task metadata.

### Keep OpenSpec rendering separate from GitHub transport

The publishing Skill owns OpenSpec selection and Markdown projection.
`aiw-github` remains a general GitHub transport and gains:

- Issue body input from a file or standard input.
- Update operations for Issue title and body.
- Stable JSON output containing the repository, Issue number, URL, and state.
- Documentation that uses the parser's actual command names.

This separation keeps GitHub concerns out of the general engineering Skills and
allows a future GitLab adapter to reuse the projection contract without
changing local work management.

## Risks / Trade-offs

- [Existing users may still have useful `.scratch` issues] → Leave existing
  files untouched, document them as legacy input, and require an explicit
  migration request before importing them.
- [A checklist item has less independent metadata than an Issue] → Promote only
  independently managed work to a separate OpenSpec change.
- [Generated GitHub content may overwrite human edits] → Update only a managed
  marker block or clearly declare the generated body authoritative; preserve
  content outside the markers.
- [Local and remote status can diverge] → Declare OpenSpec authoritative and
  make publish/close actions explicit.
- [Long Markdown bodies are fragile as command arguments] → Require body-file or
  standard-input support in the transport.
- [Repository and Issue mappings can become stale] → Validate the mapped Issue
  before updating and fail without creating a duplicate.

## Migration Plan

1. Add shared OpenSpec work-management guidance and declare it in
   `docs/agents/`.
2. Update setup, routing, specification, ticketing, implementation, review, and
   handoff Skills to use the shared contract.
3. Remove new `.scratch` publication behavior while leaving existing `.scratch`
   files untouched.
4. Add the missing `aiw-github` Issue transport operations and align its
   documentation and tests.
5. Add the explicit OpenSpec-to-GitHub publishing Skill and mapping record.
6. Validate affected Skills, plugin command behavior, OpenSpec artifacts, and
   scoped repository diffs.

Rollback consists of restoring the previous Skill definitions and plugin
commands. Existing OpenSpec changes and external mapping files remain readable
and require no data transformation.

## Open Questions

None. GitLab publication and dependency metadata are intentionally deferred to
separate changes when there is a concrete workflow that needs them.
