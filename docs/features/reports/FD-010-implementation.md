# FD-010 Worker Implementation Report

- FD: `FD-010`; Worker handoffs `FD-010-000003-changes-requested` and
  `FD-010-000005-changes-requested`, claimed by `codex-worker-fd010-r1` and
  `codex-worker-fd010-r2`.
- Source review report: `docs/features/reviews/FD-010-review-r1.md`.

## Changes

- Split prompt-library source and destination paths in
  `internal/commands/initcmd/prompts.go`.
- Read the checked-in template library from
  `docs/agent-templates/prompts/` and install its relative file tree under
  `.agents/prompts/`. Preserve the existing overwrite/skip behavior.
- Correct the template auto-detection error example to use the restored
  `aiw init --prompts --template go --merge` interface.
- Record the actual compile-only result and leave runtime checks explicitly
  unverified.

## Verification

- PowerShell:
  `$env:GOPROXY = 'off'`
  `go build -o NUL ./cmd/aiw`
  Result: exit code 0 after each of the source/destination path fix and the
  stale command example fix.
- Tests, `aiw init`, final builds, formatters, linters, and verification scripts
  were not run under the repository resource budget. Prompt-file writes remain
  statically reviewed but not runtime exercised.

## Review response

- Reviewer pass 1's two findings and pass 2's stale command example were
  actionable and repaired.

## Human-directed review recovery

- The user requested correction of the stale pass-3 receipt on 2026-10-01.
  Preserve the current FD's help Work Item 1.3 and its Verification records.
  Request an independent supplemental review of the complete current scope.
- The FD records that `scripts/compile.py` compiled `./cmd/aiw`, then failed
  because `cmd/aiw-wf` no longer exists. That overall command is not passed.
  No compile, test, or runtime check was rerun during this document recovery.

## Supplemental review repair

- Claimed `FD-010-000009-changes-requested` as
  `codex-worker-fd010-recovery` and addressed the single finding in
  `docs/features/reviews/FD-010-review-recovery.md`.
- `internal/commands/help/super-help.go` now uses the existing dispatcher
  discovery for plugin-help existence and lists `.py`, `.exe`, and
  extensionless plugin files, excluding nested directories. The existing
  `wf` compatibility plugin can therefore be listed and routed to help.
- Compile-only command: `go build -o NUL ./cmd/aiw` with `GOPROXY=off` and
  `GOTOOLCHAIN=local`; exit code 0. Used the narrow target because the root
  compile script still references removed `cmd/aiw-wf`. No tests, runtime
  help checks, network calls, or final-artifact builds were run.

- Claimed `FD-010-000012-changes-requested` as
  `codex-worker-fd010-recovery-win`. Corrected `internal/plugin/discover.go`
  so Windows ranks `.exe` before an extensionless platform binary. Script
  priority and non-Windows ranking are preserved.
- After this source fix, `go build -o NUL ./cmd/aiw` with `GOPROXY=off` and
  `GOTOOLCHAIN=local` passed (exit 0). Runtime plugin execution remains
  unverified; no tests or network calls were run.
