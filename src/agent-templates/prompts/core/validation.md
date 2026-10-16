# Validation

## Default

Static review is the default and normally the only validation:

- inspect the final diff;
- trace changed types, config, contracts, and call paths;
- check docs and prompt consistency.

After implementation, run one compile-only check. Prefer a `compile*` script
under `scripts/` or at the repository root; otherwise use the narrowest
language-level compiler command. Do not run `build*` scripts, tests,
final-artifact builds, formatters, linters, type checkers, verification
scripts, or smoke commands after an edit.
If that compile-only check finds a source error, correct it and rerun the same
compile-only command once. A second failure must be reported without another
automatic retry.

## Authorization

Executable validation other than compile-only is allowed only when:

- the user explicitly requests it;
- the task is specifically to create or repair tests; or
- static analysis cannot answer a decisive question; or
- the user explicitly requests execution through the standalone `fd-test`
  Skill, or a legacy CLI Tester has the Planner approval record required for
  that exact command and implementation revision.

Explicit FD test-support wording can select the optional orchestration without
separate Skill calls. Execution still follows exact scope/revision authority and
local language restrictions. Cases-only never authorizes execution. Planned TDD
Red/Green phases need authorization; target Red is not an unexpected retry.

For the decisive-runtime-evidence case, ask first. State the exact command, purpose, expected duration,
scope, and any network or permission risk.
For a legacy Planner approval, inspect invoked test code and side effects.
Approve a focused offline command confined to assigned or temporary paths;
escalate destructive, external, privileged, secret-bearing, or unclear effects
to the human. Record the decision before execution.

## Runtime Budget

When authorized:

1. Run one focused command covering the changed path.
2. Rerun once only after relevant code or environment changed.
3. Ask before widening to package, module, repository, integration, or full
   build scope.
4. Never loop equivalent commands to obtain a passing result.

When unexpected failures exhaust this budget, or no safe repair/cause is known,
stop affected test repair/execution, perform bounded static diagnosis-only and
continue independent authorized work. Retain the ledger across resume/review;
restart needs explicit renewed direction and authority. Confirmed acceptance
defects/material unknowns still block delivery; failure never becomes a pass.

## Evidence

- Report exact commands run and their result.
- Report tests, builds, or checks intentionally not run.
- Say what remains unverified and why.
- Never imply runtime success from static review alone.
