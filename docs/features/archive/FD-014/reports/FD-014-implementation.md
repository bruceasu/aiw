# FD-014 Worker implementation report

- Source handoff: `FD-014-000009-work-requested`, claimed by session
  `01a0f7b5-5fc4-70b0-ab29-69d25ce53239`.
- Predecessor: FD-015's reviewed `refresh-worker` operation cancelled the
  stale FD-014 revision-6 Worker receipt and created the current handoff.

The FD CLI now routes `implementation-ready` to Tester for FDs with
`**Test policy:** Independent`; old FDs without the marker retain direct
Reviewer routing. A Tester in a separate session reports scenario/test counts,
both coverage measures, raw evidence or unavailable reasons, commands, and
risks. PM accepts or rejects a versioned decision tied to the Tester report
and FD revision/digest; failed executed behavior tests cannot be accepted.
Reviewer uses a third session and cannot emit an independent-policy review
outcome without a current PM acceptance. Reimplementation restarts the Tester
round. New FD and evidence templates, source Skills, usage guidance, and the
stable workflow spec describe the path.

The optional parallel preparation path is a PM-assigned separate test
workspace; it does not advance the canonical handoff or authorize execution.
Role independence is enforced by session references in receipts. The CLI
checks labelled test report and decision fields, while the independent
Reviewer checks the truth and quality of the underlying evidence.

`python scripts/compile.py` passed with `GOTOOLCHAIN=local` and
`GOPROXY=off` (exit code 0). It compiles Go without retaining a binary and
does not exercise the Python lifecycle. Python source parsed successfully
through `ast.parse` with no bytecode artifact. `git diff --check` found no
whitespace error in the tracked changed paths. Static review inspected the
FD event transitions, claim/emit gates, current PM acceptance requirement,
and legacy direct-review route. No tests, coverage runs, final builds,
network calls, or external service calls were run.

`python plugins/aiw-skills/aiw-skills.py sync fd-workflow` reported the project
Skill already installed. The source and installed `SKILL.md` files have the
same SHA-256 digest,
`296017CDF6D92A0AED8B8B872C7C306FFAB498CDAC598467C97949A1B622254B`.
Work Item 1.3 is complete. This report is implementation evidence, not an
independent Tester or Reviewer result.
