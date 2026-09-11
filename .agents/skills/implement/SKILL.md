---
name: implement
description: "Implement one selected work item from an AIW Task and its matching OpenSpec change."
---

# Implement

Follow `skills/reviewed-skill-contract.md`; lifecycle ownership remains in
`skills/work-management.md`.

Read `skills/work-management.md` and
`docs/agents/work-management.md` when present.

## Resolve Work

1. Resolve one AIW Task.
2. Resolve its matching OpenSpec change and selected `tasks.md` item.
3. Read only the applicable `task.toml`, proposal, design, capability specs,
   checklist, and notes.
4. Stop and ask for the Task ID when several Tasks match.

## Design Readiness

Before editing, read `design.md` for `## Design Readiness` when it exists. Do
not implement when it is `BLOCKED`, or when an unresolved design-level `%%`
directly affects the selected item; report the Design Readiness Gate and route
back to `/to-spec` or `/to-tickets`. `implement` does not load `fd-workflow`.
If the selected item exposes an unresolved business term, entity relationship,
or domain boundary, stop and route to `domain-modeling` in engineering mode
and then back to `/to-spec` or `/to-tickets`. Do not infer a new domain model
while editing implementation code.

Older changes without this section remain valid when their existing design and
specs make the selected item actionable. If they do not, report the missing
design decision as a Gate rather than inferring one.

## Prepare The Workspace

Use the Task's declared workspace. Ordinary sequential work stays in the
primary workspace; verify its project root and branch before modifying files.
Use `aiw wt` only when isolation is explicitly authorized and beneficial.

Do not use raw `git worktree` when AIW is available. Do not silently implement
in another workspace. If the active agent cannot safely continue in the
resolved worktree, report its path and stop.

## Implement

- Apply the smallest complete change for the selected checklist item.
- Report the selected Work Item, produced Evidence, unresolved Gates, and
  recommended next Core transition. Do not directly write Task display status
  or manipulate a lease; managed adapters own those transitions.
- Use at most two bounded sub-agents under the shared contract.
- Use `aiw patch` as the preferred path for AI-generated patches when
  available. Use direct editing only when the patch command cannot represent
  the change or is unavailable, and report the fallback.
- Writing or updating tests is allowed, but do not run them automatically.
- After editing, compile the changed project. First look for a compile script
  under `scripts/` or in the repository root (`compile`, `compile.bat`,
  `compile.cmd`, or `compile.ps1`) and run that script. Do not run similarly
  named `build` scripts, test commands, linters, formatters, or verification
  workflows.
- If no compile script exists, use the narrowest language-level compile-only
  command available from the project manifest. For Go, use `go build` with
  package or module scope as appropriate; do not use `go test`. For other
  languages, choose the equivalent compile-only command and report it.
- If compilation fails, inspect the compiler output, fix the implementation,
  and compile again. Continue until compilation succeeds or the error is
  genuinely blocked by missing input or an environment dependency; report the
  command, output, and remaining blocker.
- Do not invoke `/tdd` or `/code-review` automatically.
- Do not run tests, broad build workflows, formatters, linters, vet, or
  verification scripts automatically.

## Complete Development

Update the selected checklist item, TODO, Verification, and remaining `%%`
risks or questions. Synchronize the coarse AIW Task status without overwriting
OpenSpec-owned content.

Perform one static review of the changed paths and mark Task completion
separately from Git delivery. Do not automatically commit, merge, push, remove
a worktree, delete a branch, synchronize, or archive. If the user explicitly
requests Git delivery, validate the recorded `parent_branch` and preserve all
resources on failure.

After development is complete, ask once whether the user wants one focused test
command run. Show the exact command, scope, and expected duration. Default to no
test when the user declines or does not respond. Ask again before broader tests
or builds.
