---
name: fd-review
description: Independently review a numbered FD implementation and hand findings back to Worker or pass Verification.
---

# FD Review

Read `skills/work-management.md` and the repository instructions. Use this
Skill when an FD is Pending Verification or the user explicitly requests an
independent review. The Reviewer must use a separate session from the Worker
that wrote the code. Do not implement new scope during review.

Read the FD, Issue evidence when present, relevant stable specs, the Worker's
implementation report, and the exact diff or commits under review. Check
correctness, acceptance coverage, compatibility, unexpected files, and the
truth of each reported verification result. Run no test or final build unless
the repository authorization rules permit it. An unrun check is not passed.
When taking a pending Reviewer handoff in a host session, first claim its
exact event with `aiw fd claim <id> <event-id> --session <host-session-id>`.
Do not take over an event owned by another session.

Write a concise review report to a project-relative file under
`docs/features/reviews/`. Include FD ID, source event ID, reviewed commit or
diff base, findings with file paths, commands actually run, skipped checks,
and residual risk. Keep the report separate from the Worker's report.

If a material issue remains, emit `changes-requested` with `--producer
reviewer --artifact <report>` and `--source-event <id>` when the original
handoff was dispatched. This returns work to Worker. If all stated acceptance
conditions are supported by real evidence, emit `verification-passed` with the
same identity fields. Update the FD Verification section with the report path
and outcome. A passed review permits a later close decision; it does not
push, merge, release, deploy, or archive by itself.
