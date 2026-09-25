---
name: fd-workflow
description: Deepen an AIW/OpenSpec Task design when material decisions remain, or manage explicitly requested standalone Feature Design storage.
---

# Feature Design Workflow

Use the current working directory as the repository root unless the user explicitly specifies another repository.

The user's explicit request takes precedence over defaults in this skill. Complete reversible/read-only work without unnecessary approval pauses. Ask only when a choice would materially change the outcome and cannot be inferred safely.

## Managed AIW/OpenSpec Mode

When a resolved AIW Task and matching OpenSpec change are in scope, this mode
takes precedence over the portable storage workflow below. `fd-workflow` is a
planning adapter: it may analyze the request and, when authorized, update the
OpenSpec proposal, design, or checklist owned by that change. It reports
proposed Work Items, Evidence, Gates, and next actions to Workflow Core.

In managed mode, do not create or update `docs/features/FEATURE_INDEX.md`,
allocate FD numbers, maintain a second status, archive an FD, update a
changelog, commit, or run verification. Do not claim/release a lease or change
Task/Attempt state. Record unresolved design input as `%% NEEDS_INPUT` in the
appropriate OpenSpec artifact.

Use portable mode only when the user explicitly requests standalone FD storage
or no managed AIW/OpenSpec context exists. Portable mode has its own files and
must not be reconciled into a managed Task automatically.

`to-spec` is the normal managed-mode caller. Before this design pass, use
`domain-modeling` in engineering mode when terminology, entity relationships,
or domain boundaries are not stable. `to-tickets` may call this Skill only
when ticket splitting reveals a material missing design decision.
`implement` consumes the resulting OpenSpec design and must not call this
Skill. A managed design pass records `FD_APPLIED`, `FD_NOT_REQUIRED`, or
`BLOCKED` in the change's `design.md`; only `BLOCKED` prevents implementation.

For a managed design pass, read only the relevant proposal, design, capability
specs, and checklist. Resolve material decisions in the existing `design.md`:
state the requirement or constraint, evidence, chosen approach and rationale,
compatibility or migration effect, verification intent, and remaining risk.
Keep settled requirements from the approved handoff; ask only about a new
material conflict or missing decision. Report Design Readiness from those
artifacts, and leave unresolved choices as `%% NEEDS_INPUT`.

## Standalone FD mode

Use this mode only when the user explicitly requests standalone FD storage or no managed AIW/OpenSpec context exists. Read `references/portable-operations.md` only for standalone FD work; it contains the storage layout, operation steps, annotations, and safety rules. Do not load it for a managed design pass.
