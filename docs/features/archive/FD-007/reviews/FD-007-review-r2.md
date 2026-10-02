# FD-007 independent review, pass 2

- Source event: `FD-007-000006-implementation-ready` (claimed by `reviewer-r2-20261001-1`).
- Reviewed base: `HEAD c01fe2c` to the current uncommitted FD-007 diff, including the Worker's repair after `FD-007-000005-changes-requested`.
- Outcome: `changes-requested`.

## Finding

1. **The resume entry can still claim a stopped Worker handoff.** `skills/fd-workflow/SKILL.md:68-80` routes every existing `In Progress` FD to Worker and says to claim its pending event. The new check at lines 107-114 runs only after this invocation's Reviewer subagent returns. If auto stops after a third `changes-requested` (or a review with no actionable repair), a later `$fd-workflow auto FD-XXX` invocation enters at step 1 and can claim the deliberately pending Worker event without checking the prior review count or report. This violates FD-007 Acceptance 2-3 and the stable spec's three-review scenario. `plugins/aiw-fd.py:531-557` permits that claim because the CLI does not enforce the auto review limit. Add the same count/actionability gate to preflight before any resumed Worker claim; when the gate fails, leave the event pending and report the human-directed recovery state. Keep the step 5 gate for uninterrupted execution.

## Evidence and limits

- Confirmed that source and installed Skills expose `auto`, require a separate `fd-review` subagent, and place the immediate post-review check before Worker claim. The stable spec and usage guide state the intended three-review stop. Manual FD commands remain described separately.
- Read the FD, first Reviewer report, Worker report, source and installed Skills, portable references, stable spec, usage guide, and `plugins/aiw-fd.py` claim/emit/close paths. Inspected the relevant diff against `c01fe2c`.
- Commands run: `go run cmd/aiw/main.go fd claim FD-007 FD-007-000006-implementation-ready --session reviewer-r2-20261001-1`; `go run cmd/aiw/main.go fd show FD-007`; targeted `Get-Content`, `rg`, `git status`, `git log`, `git diff`, and `git diff --check`. The diff check passed with line-ending warnings only.
- No tests, builds, network calls, Git writes, or end-to-end auto run. The finding follows the documented resume path and the CLI's claim behavior; runtime validation was not performed.
