# FD-012: Consolidate agent guidance under .agents

**Status:** Complete
**Revision:** 7
**Priority:** Medium

## Problem

Agent guidance is split between the repository root, `prompts/`, `docs/agents/`,
and `docs/agent-templates/`, while current instructions already direct agents to
`.agents/prompts/`. This leaves the canonical prompt path missing and makes
`aiw init`, Skills, docs, and packaging depend on the old locations.

## Options and decision

Keep the existing directories and add compatibility references, or consolidate
the agent-owned content beneath `.agents/` and update consumers. Choose the
consolidation to match the existing instruction paths and make the directory
layout consistent. Preserve the existing directory names below `.agents/`:
`prompts/` -> `.agents/prompts/`, `docs/agents/` -> `.agents/agents/`, and
`docs/agent-templates/` -> `agent-templates/`.

## Solution

Move the three source trees without dropping their contents. Keep both prompt
trees for their existing roles: `.agents/prompts/` is the repository's active
guidance library, and `agent-templates/prompts/` remains the prompt
library packaged by `aiw init`. Update the root and template `monorepo.md`
files, all path references in Skills/docs/plugins, the CLI template lookup,
and local installation packaging so `aiw init` can find templates beside the
installed binary. Preserve the `aiw init` command surface and conflict behavior.

## Scope

Repository guidance, initialization templates, their path consumers, and
Windows installation packaging. No dependency, public API, CLI argument,
OpenSpec capability, or runtime behavior changes beyond resolving the same
templates and writing them to the same project targets. Do not touch unrelated
dirty files; update both canonical Skills and their checked-in `.agents/skills`
copies where they reference moved guidance.

## Work items

- [x] 1.1 Move `prompts/`, `docs/agents/`, and `docs/agent-templates/` under
  `.agents/`; update both `monorepo.md` files and nearby links.
- [x] 1.2 Update CLI path constants, prompt synchronization paths, installer
  packaging, plugin setup references, Skills, and README references.
- [x] 1.3 Statically search for stale paths, review the final diff, and run one
  compile-only check for the Go CLI.

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- The three source trees exist only at `.agents/prompts/`,
  `.agents/agents/`, and `agent-templates/`; the old locations are
  absent.
- Both monorepo guidance files direct shared prompts to `.agents/prompts/`.
- `aiw init` resolves base and language templates under
  `agent-templates/`, synchronizes its bundled prompt library, and
  preserves current conflict handling and project output paths.
- The Windows install script places `agent-templates/` and
  `.agents/prompts/` beside the installed executable.
- Skills, scripts, docs, and CLI help contain no stale references to the moved
  source paths.

## Verification

- Reviewer r2: verification passed for
  `FD-012-000006-implementation-ready`; see
  `docs/features/archive/FD-012/reviews/FD-012-review-r2.md`.
- Reviewer r1 requested replacing the missing `cp-mirror.bat` install calls;
  both install branches now use built-in recursive `xcopy`.
- Static path search found no remaining references to `docs/agents/`,
  `docs/agent-templates/`, or the old root prompt-library location outside
  this FD's migration record and archived evidence.
- Reviewed the scoped code, installer, README, plugin, and canonical/installed
  Skill diffs. Both `monorepo.md` copies now point to `.agents/prompts/`.
- Static review confirmed the Windows and Linux installer branches both copy
  `agent-templates/` and `.agents/prompts/` recursively using quoted
  `xcopy /E /I /Y` paths. The remaining install-time `docs/usage` copy also
  uses `xcopy`; these paths no longer depend on the missing helper.
- Ran `python scripts/compile.py` once; it returned no compiler diagnostics.
- Did not execute the Windows installer; copy behavior remains unverified at
  runtime.
- Tests, final-artifact builds, and runtime checks were not run.

## TODO

- [x] Resolve the install-copy helper finding in
  `docs/features/archive/FD-012/reviews/FD-012-review-r1.md`; request independent review.
- [x] Independent review r2 passed; a later close decision remains separate.

## Sources

- Issue: none
- Existing workspace changes accepted by the user as part of this task
- `skills/work-management.md`
- `openspec/specs/cli-and-plugins/spec.md` (reviewed; no capability change)

**Completed:** 2026-10-01
