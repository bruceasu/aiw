# AGENTS.md
Always respond in Chinese.
Write document in Easy English if occure encoding problem with Chinese.

## FD Workflow

Before coding, read the selected numbered FD and its Work Items, the approved
Issue when present, and relevant stable specs under `openspec/specs/`. For a
legacy AIW Task, also read its Task record and linked FD.
Read a linked OpenSpec change only when one exists and affects the work:

- `openspec/changes/<id>/`

The numbered FD is the primary plan for new work. A new FD does not require
An OpenSpec change is optional.

- Work on one FD at a time.
- Keep changes scoped and reviewable.
- Do not refactor unrelated modules.
- Preserve backward compatibility unless explicitly required.
- Update TODO and Verification before finishing.
- Record unresolved risks or questions with `%%` notes instead of guessing.
- Update stable specs or design notes only when their requirements or decisions
  changed.

Before implementing any numbered FD, use its dedicated branch and worktree:

- branch: `feature/<fd-id>`
- worktree: `.wt/<fd-id>`

Commit the ready FD plan on the parent branch and ensure the parent workspace
is clean before using `aiw git wt add <fd-id>`. Keep the parent clean
while the FD is in flight when possible; a dirty parent blocks merge until it
is clean again. Do not mix unrelated changes into the FD plan commit. Legacy
Tasks keep their existing `feature/<task-id>` and `.wt/<task-id>` conventions.

## Resource Budget

Token and command cost are hard constraints.

Default budget for an ordinary implementation request:

- tests: `0`
- final-artifact builds, linters, formatters, vet, and verification scripts: `0`
- compile-only checks: allowed after implementation; prefer `scripts/compile*`
  or root `compile*`, otherwise use the narrowest language-level compile
  command. The command must not retain a final distributable artifact.
- network calls and dependency downloads: `0`
- permission probes or privilege escalation requests: `0`
- `codex-auto-review` and repeated review passes: `0`
- post-edit validation commands: at most `1`, and static/read-only

Implementation does not imply authorization to test or create final build
artifacts.
For an independent FD Tester, a Planner may approve one focused low-risk test
command after inspecting its invoked code and recording the exact scope and
revision-bound decision. This is an explicit exception to the zero-test
default, not permission for broader validation.
Place repository test code under the root `tests/` directory.
For new FD reports, write human-readable Markdown in Chinese and a same-name
JSON sidecar for machine/AI fields. Keep historical reports unchanged.

Before the first edit, use no more than three targeted discovery batches unless
the task is genuinely blocked. Batch related reads and searches. Read relevant
symbols or excerpts instead of dumping large files, logs, generated output, or
lockfiles.

Do not rerun an unchanged failed command. After fixing the source of a
compile-only failure, rerun the same compile-only command once to verify the
fix; if it fails again, stop and report the remaining error. One cheap
corrected retry is also allowed for a command spelling, shell entrypoint, or
path mistake. A permission failure is not a reason to try alternate shells,
escalation, or broader commands.

## Runtime Authorization

Run a test or executable validation other than compile-only only when:

- the user explicitly asks for it;
- the task is specifically to create or repair tests; or
- runtime evidence is decisive and static analysis cannot answer the question; or
- the FD Planner has recorded approval for that exact Tester command and
  implementation revision.

For the third case, pause first and state the exact command, why it is needed,
expected duration, scope, and any network or permission risk. Wait for approval.
For the Planner case, inspect the invoked test code and side effects first.
Approve without human review only when the command is focused, offline,
inspectable, and confined to its assigned workspace or temporary files.
Escalate commands involving unrelated writes/deletes, secrets, network or
external services, downloads, privilege changes, release artifacts, or unknown
effects to the human before recording approval. A Tester handoff alone does
not authorize execution.

Even when runtime validation is authorized:

- run one focused command first;
- allow one rerun only after a relevant code or environment change;
- ask before widening to a package, module, repository, integration, or full
  build scope.

Network access is off by default. Do not probe permissions by intentionally
running a command expected to fail. Request permission only when it is essential
to the user's requested outcome.

## Working Rules

- For engineering work, read `src/skills/work-management.md`. Use numbered FDs for
  new work. Use OpenSpec for stable
  capability specs and explicit change requests. Work in the primary workspace
  by default.
- Plan first for non-trivial work.
- Inspect the nearest code, tests, config, and docs.
- Expand only when current evidence is insufficient.
- Prefer static analysis and minimal edits.
- Keep interfaces and package boundaries stable.
- Pause before dependency, public API, concurrency, persistence, auth,
  migration, deployment, or CI changes.
- Do not perform Git write operations unless the user asks.

## Validation And Reporting

Static review is the default validation:

- inspect the final diff;
- trace changed types, config, and call paths;
- check instruction and documentation consistency.

After implementation, run one compile-only check. Prefer `scripts/compile*` or
root `compile*`; otherwise use the narrowest language-level compiler command.
Do not run `build*` scripts, `scripts/verify.sh`, tests, final-artifact builds,
formatters, linters, or vet by default.

Report:

- what changed and why;
- static evidence reviewed;
- commands actually run;
- tests, final-artifact builds, or checks intentionally not run;
- remaining risks or optional focused commands the user may authorize.

Never claim a runtime result for a command that was not run.

<!-- aiw-prompts:go:agents begin -->
Always respond in Chinese.
Write prompts in Easy English when asked to draft prompts.

# AGENTS.md

## Purpose

This is the root rule file for mixed-language repositories.
Keep the active instruction set and execution budget small.

## Load Order

1. Apply this file.
2. Prefer the nearest local `AGENTS.md`.
3. Load `.agents/prompts/core/resource-budget.md`,
   `.agents/prompts/core/universal-principles.md`,
   `.agents/prompts/core/validation.md`, and `.agents/prompts/core/communication.md`.
4. Add at most one repo-type prompt, one domain prompt, and one task-mode prompt.
5. Use `project-local > domain > language > repo-type > root` precedence.

Do not load the whole prompt library.

## Resource Guard

- Static analysis and editing are the default.
- Tests, final-artifact builds, formatters, linters, type checks, verification
  scripts, network calls, permission probes, privilege escalation,
  `codex-auto-review` has a default budget of zero.
- An independent FD Tester may execute one exact, revision-bound command after
  recorded Planner approval under the Runtime Authorization rule above.
- After implementation, run one compile-only check: prefer `scripts/compile*`
  or root `compile*`, otherwise use the narrowest language-level compiler
  command. The command must not retain a final distributable artifact.
- Implementation does not imply authorization to run the restricted commands.
- Use no more than three targeted discovery batches before editing unless a
  concrete blocker remains.
- After editing, use at most one static/read-only validation command by default.
- Do not repeat unchanged failed commands; a compile-only check may be retried
  once after a relevant source fix.

Follow `.agents/prompts/core/resource-budget.md` and `.agents/prompts/core/validation.md` for
authorization and retry rules.

## Working Rules

- Plan first for non-trivial work.
- Inspect the nearest code, tests, config, and docs.
- Expand only when current evidence is insufficient.
- Change only what the task needs.
- Preserve current contracts unless explicitly changed.
- Keep diffs small, local, and reviewable.
- Update nearby docs when behavior or workflow changes.

## Stop And Ask

Pause when:

- local instructions conflict;
- external behavior is unclear;
- designs have materially different blast radius;
- migration, rollout, or compatibility policy is missing;
- runtime evidence is decisive but not already authorized.

## High-Risk Areas

Call out risk before changing dependencies, auth, permissions, billing, schema,
migrations, persistence, public APIs, events, CLI contracts, CI/CD, deployment,
concurrency, retries, timeouts, or shutdown behavior.

## Prompt Routing

- Mixed-language repository: `.agents/prompts/repo-types/monorepo.md`
- Python: `python/AGENTS.md`
- Java: `java/AGENTS.md`
- Go: `go/AGENTS.md`
- Prompt changes: `.agents/prompts/domains/prompt-authoring.md`
- Task mode: one matching file under `.agents/prompts/task-modes/`

## Final Report

Include the change, reason, static evidence, commands actually run, checks not
run, residual risks, and optional focused checks that require authorization.

---

# AGENTS.md

## Purpose
This is the default local rule file for Go subtrees.
Use it when the task is mostly Go or the current directory contains Go markers.

## Inspect First
- `src/go.mod` and module layout
- `src/cmd/` entrypoints, handlers, services, and `src/internal/` packages
- config packages and tests near the touched code

## Detect The Go Domain
- Service markers:
  `src/cmd/server`, `src/internal/`, handler packages, config packages
  - also load `.agents/prompts/domains/go-service.md`
- CLI markers:
  `cobra`, `urfave/cli`, command trees under `src/cmd/`, single-binary tools
  - also load `.agents/prompts/domains/go-cli.md`

Use one domain prompt by default.
Load both only when the task truly spans both service and CLI code.

## Keep Stable
- package boundaries and `src/internal/` ownership
- exported APIs
- `context.Context` flow
- concurrency, retry, and shutdown behavior

## High-Risk Go Areas
- dependency changes
- exported API changes
- context or concurrency changes
- auth, schema, or deployment changes

## Validation Options
Use static review by default. After implementation, run one compile-only check:
prefer `scripts/compile*` or root `compile*`, otherwise run the narrowest
applicable `go build` command without retaining a final artifact. Do not run
`scripts/verify.sh`, tests, vet, `build*` scripts, or final-artifact builds.

For Go work, use the current compile result and an independent static review.
Do not create or run Go tests. Runtime validation still requires the shared
resource budget's explicit authorization.

Ask before repository-wide commands. Rerun only after a relevant change.

## Escalation
If a deeper subtree has its own `AGENTS.md`, prefer that local file.
<!-- aiw-prompts:go:agents end -->
