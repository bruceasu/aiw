# FD-005 Review r2

- FD: `FD-005`
- Source event: none; this review was requested directly. The previous review event `FD-005-000005-review-requested` already resulted in `changes-requested` and is not reused.
- Reviewed range: `61acf474abf222e87e6278ee21226692b309428..879a072` (`HEAD`); current uncommitted changes in the FD, implementation report, and usage guide were also reviewed.
- Outcome: `CHANGES_REQUESTED`.

## Findings

1. **The FD's verification gate is still open.** The FD states it must not reach Verification Passed until actual Codex/Copilot CLI compatibility and OpenAI HTTP response evidence are resolved. The implementation report confirms those checks were not run at the user's direction, and documents the resulting version/response compatibility risk. The code statically enforces the expected CLI flags and handles provider errors, but cannot establish compatibility with the installed CLI versions or actual Responses API behavior. Keep the FD in progress until the gate is explicitly revised or the required evidence is supplied. Relevant records: `docs/features/FD-005_AIW_CZ_TYPESCRIPT_PYTHON.md`, `docs/features/archive/FD-005/reports/FD-005-implementation.md`.

2. **1.5 has no recorded focused evidence.** The implementation report says fallback orchestration was statically inspected, but the FD's Verification section only says the providers are connected and does not trace or record evidence for invalid-candidate fallback and explicit-provider failure behavior. The implementation in `plugins/aiw-cz/cz_llm.py` appears to sequence providers and catch provider/validation failures, but this is not recorded against both acceptance cases.

## Evidence reviewed

- FD-005, linked REQ00004 requirement artifacts, implementation report, prior review report, and current usage documentation.
- Stable specs: `openspec/specs/cz-configuration-priority/spec.md` and `openspec/specs/cz-python-runtime/spec.md`.
- Provider/config/TUI code: `plugins/aiw-cz/cz_providers.py`, `cz_llm.py`, `cz_openai.py`, `cz_config.py`, and `cz_ui.py`.
- Plugin discovery/execution and installation exclusion: `internal/plugin/discover.go`, `internal/plugin/exec.go`, and `build.bat`.
- Current diff from the prior review base and current uncommitted documentation/FD changes.

The TUI acceptance and Python installation evidence is user-provided and recorded in the implementation report. The install command excludes `cz.toml` and `.cz.toml` from overwrite. Static code review also supports local commit after preview and no push.

## Commands run

- `git status --short`
- `git diff --stat`, `git diff --name-status`, and focused `git diff` reads
- `git diff 61acf474abf222e87e6278ee21226692b309428..HEAD --stat` and focused code diff
- `git log --oneline --decorate -8`
- `python plugins/aiw-fd.py show FD-005` (read-only)
- `rg` searches and `Get-Content` reads for the cited files

One initial PowerShell command had a brace-expansion syntax error; it was corrected once. No other command failures affected the review.

## Skipped checks and residual risk

- No tests, build, compile, CLI invocation, HTTP request, TUI interaction, or installation was run in this review. CLI and HTTP runtime checks were explicitly declined by the user; TUI and installation evidence were supplied by the user.
- Runtime compatibility of installed Codex/Copilot versions and the configured OpenAI endpoint remains unknown.
- No current FD-005 event receipt directory or new review-requested event was present, so no claim or lifecycle event was issued. The FD Verification section records this review outcome; a later dispatched handoff should carry its own source event.
