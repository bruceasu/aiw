# FD-004 implementation report

Implemented `aiw fd request-review <FD-ID> --reason "..."` for archived,
Complete FDs. The command requires a one-line reason and the latest acknowledged
Reviewer `verification-passed` receipt. It increments the FD revision, changes
the status to `Pending Verification`, preserves the prior completion date,
moves the FD back to the active feature directory, records a
`review-requested` receipt containing the reason and current content digest,
and rebuilds the index. The standard Reviewer claim and source-event protocol
handles the new handoff. Work Item checkboxes are unchanged.

The CLI help metadata, user guide, and stable FD workflow specification now
document the command and its lifecycle.

## Verification

- Passed: `python -c "from pathlib import Path; p=Path('plugins/aiw-fd.py'); compile(p.read_text(encoding='utf-8'), str(p), 'exec')"`
- Not run: tests and `scripts/fd_smoke.py` (not authorized under the default
  repository validation budget).
- Independent Reviewer assessment is pending.
