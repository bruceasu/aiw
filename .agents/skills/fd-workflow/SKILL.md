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

## Portable storage layout

Use repository-neutral files:

- `docs/features/FEATURE_INDEX.md`
- `docs/features/TEMPLATE.md`
- `docs/features/FD-XXX_TITLE.md`
- `docs/features/archive/`
- `AGENTS.md` for repository-wide FD conventions
- `CHANGELOG.md` when changelog support is enabled

Do not require `.claude/commands/`, `CLAUDE.md`, Claude tool names, or a pinned model.

For platform-specific installation and invocation details, read `references/platforms.md`.
For exact templates and lifecycle rules, read `references/templates.md`.

## Determine the requested operation

Map the user's request to one operation:

- **init**: initialize or repair the FD system.
- **new**: create a new FD and update the index.
- **explore**: summarize project architecture, FD history, and recent activity.
- **deep**: investigate a hard problem from four distinct angles, verify important claims, then synthesize.
- **status**: reconcile FD files/index/archive and print active status.
- **verify**: review implementation changes, fix issues, run appropriate verification, and report results.
- **close**: mark an FD complete/closed/deferred, archive it, update index/changelog, and optionally commit when authorized.

If the user names an FD number, normalize `1`, `001`, and `FD-001` to `FD-001`.

## Project context discovery

Before mutating FD files, inspect enough repository context to adapt the result:

1. Read `AGENTS.md` if present.
2. Read the primary project README and relevant build manifests (`pom.xml`, `build.gradle*`, `package.json`, `pyproject.toml`, `go.mod`, `Cargo.toml`, etc.).
3. Inspect recent git history for commit-message conventions.
4. Identify primary language, test/build commands, docs layout, and existing changelog conventions.
5. Preserve existing repository conventions unless they conflict with the user's explicit request.

Avoid exhaustive repository scans when a targeted read/search is sufficient.

## Operation: init

1. Check for `docs/features/FEATURE_INDEX.md`.
2. If it exists, inspect the current FD setup and repair only missing/inconsistent pieces unless the user asked for regeneration.
3. If it does not exist, create:
   - `docs/features/archive/`
   - `docs/features/FEATURE_INDEX.md`
   - `docs/features/TEMPLATE.md`
4. Enable changelog support by default when there is no existing project convention against it. If the user explicitly opts out, omit changelog behavior.
5. If `CHANGELOG.md` exists, preserve it and integrate with its existing format when practical. Otherwise create the minimal Keep a Changelog skeleton from `references/templates.md`.
6. Add the FD Management section from `references/templates.md` to `AGENTS.md`. If an equivalent section already exists, update it rather than duplicating it.
7. Customize project name, examples, and commit format based on repository context.
8. Report created, preserved, and modified files plus the next useful action.

Do not generate Claude-specific slash commands as part of the core setup.

## Operation: new

1. Read `docs/features/FEATURE_INDEX.md` and inspect FD files in `docs/features/` and `docs/features/archive/`.
2. Determine the highest FD number across active, completed, deferred/closed, backlog, and archived filenames.
3. Allocate the next zero-padded number.
4. Derive a concise title and `UPPER_SNAKE_CASE` slug from the user's request.
5. Create `docs/features/FD-XXX_SLUG.md` using the FD template.
6. Fill Problem, Solution, Files, and Verification sections when supported by the user's request and repository evidence. Do not invent implementation details that have not been established.
7. Add the FD to Active Features in the index.
8. Do not commit unless the user explicitly requested implementation/commit or repository instructions clearly authorize it.

## Operation: explore

Collect three independent views of the repository:

1. **Project overview**: purpose, architecture, stack, layout, build/test entry points, key constraints.
2. **FD history**: active FDs, archived/completed work, recent `FD-` commits.
3. **Recent activity**: recent commits, files in flux, current branch, working-tree status.

If the runtime supports subagents/parallel delegation, run these views concurrently. Otherwise perform them sequentially while keeping the analyses logically independent.

Synthesize into a compact briefing with Project Overview, FD Status, Recent Activity, and Quick Reference.

## Operation: deep

Use four genuinely different analytical lenses. Choose lenses based on the problem rather than using the same four labels mechanically.

Examples:

- Performance: algorithmic, data/structure, incremental/caching, environment/platform.
- Architecture: simplicity, scalability, precedent, contrarian alternative.
- Debugging: failure path, environment/config, challenged assumptions, similar patterns.

For each lens:

1. State the focused question.
2. Inspect relevant code/docs/config with read-only operations first.
3. Record concrete evidence using actual files, symbols, commands, or data paths.
4. State implications, recommendation, tradeoffs, assumptions, and biggest uncertainty.

Prefer parallel subagents when the runtime exposes them. Do not hard-code a vendor-specific agent tool, agent type, or model name. If no subagent capability is available, perform four sequential passes yourself.

Before synthesis:

1. Detect contradictory claims across the four analyses.
2. Verify the 3-5 factual claims that would most change the recommendation if wrong.
3. Correct unsupported paths, symbols, configuration claims, or complexity assertions.

Synthesize into Agreements, Tensions, Surprises, Corrections, Recommendation, Risks/Assumptions, and First Step. If an active FD is relevant, propose precise updates to its Solution section; modify it only when the user asked for the update or implementation work is already authorized.

## Operation: status

Choose one path:

- **Fast path** when this session just created/closed/reconciled FDs: read index + active FD files and report.
- **Full grooming** on a new/uncertain session or when requested.

Full grooming:

1. Compare each active FD file's `**Status:**` with the index. The FD file is source of truth.
2. Move Complete/Closed/Deferred FD files from `docs/features/` to `docs/features/archive/`.
3. Ensure each FD appears in the correct index section.
4. Detect index entries without files and FD files absent from the index. Report unresolved orphans rather than fabricating missing content.

Output the active FD table and counts by status.

## Operation: verify

1. Inspect `git status` and the implementation diff relevant to the current task/FD.
2. Review correctness, edge cases, security/escaping, consistency with repository patterns, completeness, and accidental scope creep.
3. Fix clear implementation defects when the user has authorized implementation work. Keep fixes in scope.
4. Run a smallest meaningful runtime check only with explicit authorization;
   otherwise perform static review and report runtime checks as skipped.
5. Include manual/integration checks when automation cannot verify the behavior.
6. Report what was reviewed, changes made, commands run, pass/fail results, and remaining uncertainties.

Do not force a commit-before-review sequence. Commit only when the user requested it or repository instructions authorize automatic commits.

## Operation: close

1. Resolve the target FD from the argument or clear conversation context. If multiple FDs are plausible, do not guess.
2. Read the FD and current index.
3. Set status to `Complete`, `Closed`, or `Deferred` and add the applicable date field.
4. Remove the FD from Active and insert it into the correct index section.
5. For Complete, update `CHANGELOG.md` under `[Unreleased]` when changelog support is enabled. Select Added/Changed/Fixed/Removed based on the implemented behavior.
6. Move the FD file to `docs/features/archive/`.
7. If committing is authorized, stage only files related to this FD and create one focused commit using the repository's commit convention (default `FD-XXX: title`).
8. Report final disposition, archive path, index/changelog changes, verification state, and commit hash if a commit was made.

## Inline annotations

Treat lines beginning with `%%` in FD/design documents as user annotations:

- Address every annotation.
- Apply requested changes when sufficiently clear and authorized.
- Remove an annotation after it has been addressed.
- If an annotation requires an unresolved product/design choice, preserve it and report the unresolved decision instead of silently guessing.

## Safety and repository hygiene

- Never discard unrelated working-tree changes.
- Never rewrite history, force-push, or perform destructive git operations unless explicitly requested.
- Keep FD metadata changes scoped to the relevant FD.
- Verify files and symbols before claiming they exist.
- Prefer repository-native build/test tools and existing scripts over introducing dependencies.
