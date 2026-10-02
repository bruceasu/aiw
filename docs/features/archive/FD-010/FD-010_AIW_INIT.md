# FD-010: AIW Project Initialization

**Status:** Complete
**Revision:** 14
**Priority:** Medium

## Problem

`aiw init` was removed along with the legacy Task command package, but it also
provides useful project setup and prompt-template synchronization. The current
help still advertises the command, while invoking it falls through to plugin
discovery and fails because no `aiw-init` plugin exists.

## Options and decision

1. **Restore project initialization as a standalone built-in command (chosen).**
   Preserve the useful base-instruction and prompt-template setup without
   reintroducing the deleted Task lifecycle package.
2. Add an `aiw-init` plugin. This creates a second implementation boundary for
   a command that already has templates and Go-side prompt merge behavior.
3. Remove `init` from help. This leaves the useful project initialization
   unavailable and contradicts the user's intent.

## Solution

Add `internal/commands/initcmd` and dispatch `aiw init` to it. The command
creates generic agent instruction files only when absent; `--prompts` applies
the selected language template and prompt library with the existing skip,
merge, and force semantics. Keep the `--template` detection and supported
template behavior. Base and language prompt files remain at the repository
root, while the reusable prompt library is installed under
`.agents/prompts/`. `--no-setup` skips generic instruction-file setup. When
`--merge` and `--force` are combined, merge the marked instruction sections and
refresh the managed prompt library. Do not create or modify OpenSpec
directories/artifacts, Task runtime records, or worktree directories. Do not
invoke `aiw-setup-project`, whose setup output adds project workflow documents
outside this command's prompt-template contract.

## Scope

In scope: restore the built-in `init` entry point, prompt template sync,
generic instruction-file scaffolding, relocate the installed prompt library
under `.agents/prompts/`, update path references in the copied instruction
templates, and accurate CLI help. Out of scope: Task lifecycle commands,
OpenSpec/worktree scaffolding or mutation, and changing prompt contents.

## Work items

- [x] 1.1 Restore `aiw init` without OpenSpec or Task runtime setup and preserve
  Java prompt sync options.
- [x] 1.2 Align the user-facing `init` help and prompt paths with the restored
  behavior.

- [x] 1.3 Replace stale Task and worktree help with the current FD commands,
  examples, and built-in command inventory. Remove the wf summary from
  top-level help and plugin listings; keep explicit plugin help routing available.

## Acceptance

- `aiw init --prompts --merge --force --template java` reaches the built-in
  command instead of plugin discovery.
- The reusable prompt library is written under `.agents/prompts/`; generated
  instruction templates point there.
- `--merge` and `--force` can be combined to merge instruction sections while
  refreshing the managed prompt library; prompt-related flags still require
  `--prompts`.
- Initialization does not create or modify OpenSpec artifacts or Task runtime
  records.
- Project instruction files are created only when absent unless prompt sync
  options explicitly select merge or overwrite behavior.

## Verification

- Recovery Worker: claimed `FD-010-000012-changes-requested` and corrected
  Windows plugin binary selection. `scoreExt` prefers `.exe` over an
  extensionless binary on Windows, preserving script priority and other
  platforms' existing order. `go build -o NUL ./cmd/aiw` passed (exit 0)
  after this source fix with `GOPROXY=off` and `GOTOOLCHAIN=local`; tests and
  runtime plugin execution were not run.
- Supplemental Reviewer pass for `FD-010-000013-implementation-ready`:
  `VERIFICATION_PASSED`. The reviewed Windows score ordering selects `.exe`
  ahead of extensionless candidates, while preserving script priority and
  non-Windows ordering. See
  `docs/features/archive/FD-010/reviews/FD-010-review-recovery-r3.md`. Runtime plugin launch
  remains unverified.

- Recovery Worker: claimed `FD-010-000009-changes-requested` and corrected
  help discovery for the existing `wf` binary plugin. Plugin help existence
  now uses the dispatcher's `DiscoverPlugin`; plugin listings accept Python,
  extensionless, and `.exe` entries and skip nested directories.
- Supplemental Reviewer pass for `FD-010-000011-review-requested`:
  `CHANGES_REQUESTED`. On Windows, shared plugin resolution can choose an
  extensionless Linux binary ahead of the `.exe`, preventing explicit `wf`
  help from starting. See
  `docs/features/archive/FD-010/reviews/FD-010-review-recovery-r2.md`.
- Recovery compile-only check: `go build -o NUL ./cmd/aiw` passed (exit 0),
  with `GOPROXY=off` and `GOTOOLCHAIN=local`. The narrow command avoids the
  known removed target in `scripts/compile.py`; that script's earlier failure
  remains recorded below. Tests and runtime help checks were not run.

- Human-directed recovery (2026-10-01): the pass-3 receipt no longer matches
  the current FD content. Its result remains historical evidence; the active
  FD requires a fresh claimed independent review, including Work Item 1.3
  and its recorded compile-script failure, before archive. This recovery does
  not reset the three outcomes from the original Auto cycle.
- Supplemental Reviewer pass for `FD-010-000008-review-requested`:
  `CHANGES_REQUESTED`. `help wf` cannot discover the binary compatibility
  plugin. See `docs/features/archive/FD-010/reviews/FD-010-review-recovery.md`.

- Revision 6: statically traced top-level help against `cmd/aiw/main.go` and
  the FD plugin parser. Removed unsupported Task, spec, session, and wt entries
  and internal directory-name inference from the built-in help inventory.
- Revision 6 compile-only command: `python -c 'import os, runpy;
  os.environ["GOPROXY"] = "off"; os.environ["GOTOOLCHAIN"] = "local";
  runpy.run_path("scripts/compile.py", run_name="__main__")'`. The script
  compiled `./cmd/aiw` successfully without retaining a binary, then failed
  because `cmd/aiw-wf` no longer exists. No retry or script changes were made.
- Follow-up: removed wf from the top-level summary and plugin listings
  (including JSON), while preserving explicit `aiw help wf` routing.
  Offline compile-only `go build -o NUL ./cmd/aiw` passed with
  `GOPROXY=off`, `GOTOOLCHAIN=local`, and the repository-local Go cache.
- Runtime help checks and tests were not run for revision 6.
- %% README and shell completion still contain legacy Task commands; their
  broader migration is outside this focused help correction.

- Post-feedback compile-only check passed: `go build -o NUL ./cmd/aiw` with
  `GOPROXY=off`.
- Worker command context and result are recorded in
  `docs/features/archive/FD-010/reports/FD-010-implementation.md`.
- Static trace now reads the prompt library from `docs/agent-templates/prompts/`
  and writes it under `.agents/prompts/`.
- The template auto-detection error now suggests
  `aiw init --prompts --template go --merge`.
- Reviewer passes 1 and 2 both returned `CHANGES_REQUESTED`; all reported
  findings are addressed in the current Worker handoff. Reports:
  `docs/features/archive/FD-010/reviews/FD-010-review-r1.md` and
  `docs/features/archive/FD-010/reviews/FD-010-review-r2.md`.
- Reviewer pass 3: `VERIFICATION_PASSED`. See
  `docs/features/archive/FD-010/reviews/FD-010-review-r3.md` for static acceptance evidence
  and the unverified runtime-write risk.
- Runtime command checks were not run; installation and file-write behavior
  remain unverified at runtime.

## Sources

- User request and scope clarification in this conversation (2026-10-01).
- `599b11d` removed `internal/commands/task/init.go` and the `init` dispatch
  case; current `plugins/` has no `aiw-init` entry point.
- Existing templates under `docs/agent-templates/`.

- Worker implementation report: `docs/features/archive/FD-010/reports/FD-010-implementation.md`.
- Reviewer pass 1 (Revision 2): `CHANGES_REQUESTED`; both findings are
  addressed. See `docs/features/archive/FD-010/reviews/FD-010-review-r1.md`.

**Completed:** 2026-10-01
