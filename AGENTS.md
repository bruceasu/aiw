# AGENTS.md

Always respond in Chinese.
Write prompts in Easy English when asked to draft prompts.

## OpenSpec Workflow

Before coding, read only the relevant files under:

- `openspec/changes/<task>/`
- `openspec/specs/`

Prioritize `tasks.md`, `design.md` when present, and the relevant spec.

- Work on one task at a time.
- Keep changes scoped and reviewable.
- Do not refactor unrelated modules.
- Preserve backward compatibility unless explicitly required.
- Update TODO and Verification before finishing.
- Record unresolved risks or questions with `%%` notes instead of guessing.
- Update stable specs or design notes only when their requirements or decisions
  changed.

When a dedicated branch or worktree is needed:

- branch: `feature/<task-id>`
- worktree: `.wt/<task-id>`

## Resource Budget

Token and command cost are hard constraints.

Default budget for an ordinary implementation request:

- tests: `0`
- builds, linters, formatters, vet, and verification scripts: `0`
- network calls and dependency downloads: `0`
- permission probes or privilege escalation requests: `0`
- `codex-auto-review`, sub-agents, and repeated review passes: `0`
- post-edit validation commands: at most `1`, and static/read-only

Implementation does not imply authorization to test or build.

Before the first edit, use no more than three targeted discovery batches unless
the task is genuinely blocked. Batch related reads and searches. Read relevant
symbols or excerpts instead of dumping large files, logs, generated output, or
lockfiles.

Do not rerun the same or an equivalent failed command. One cheap corrected retry
is allowed only for a command spelling, shell entrypoint, or path mistake. A
permission failure is not a reason to try alternate shells, escalation, or
broader commands.

## Runtime Authorization

Run a test or other executable validation only when:

- the user explicitly asks for it;
- the task is specifically to create or repair tests; or
- runtime evidence is decisive and static analysis cannot answer the question.

For the third case, pause first and state the exact command, why it is needed,
expected duration, scope, and any network or permission risk. Wait for approval.

Even when runtime validation is authorized:

- run one focused command first;
- allow one rerun only after a relevant code or environment change;
- ask before widening to a package, module, repository, integration, or full
  build scope.

Network access is off by default. Do not probe permissions by intentionally
running a command expected to fail. Request permission only when it is essential
to the user's requested outcome.

## Working Rules

- For managed engineering work, read `skills/work-management.md`. Use AIW for
  Task lifecycle and explicit workspace isolation, and OpenSpec for requirement
  and checklist artifacts. Work in the primary workspace by default.
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

Do not run `scripts/verify.sh`, tests, builds, formatters, linters, or vet by
default.

Report:

- what changed and why;
- static evidence reviewed;
- commands actually run;
- tests, builds, or checks intentionally not run;
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
2. Prefer the nearest local `AGENTS.md` or `CODEX.md`.
3. Load `prompts/core/resource-budget.md`,
   `prompts/core/universal-principles.md`,
   `prompts/core/validation.md`, and `prompts/core/communication.md`.
4. Add at most one repo-type prompt, one domain prompt, and one task-mode prompt.
5. Use `project-local > domain > language > repo-type > root` precedence.

Do not load the whole prompt library.

## Resource Guard

- Static analysis and editing are the default.
- Tests, builds, formatters, linters, type checks, verification scripts,
  network calls, permission probes, privilege escalation, `codex-auto-review`,
  and sub-agents have a default budget of zero.
- Implementation does not imply authorization to run them.
- Use no more than three targeted discovery batches before editing unless a
  concrete blocker remains.
- After editing, use at most one static/read-only validation command by default.
- Do not repeat equivalent commands.

Follow `prompts/core/resource-budget.md` and `prompts/core/validation.md` for
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

- Mixed-language repository: `prompts/repo-types/monorepo.md`
- Python: `python/AGENTS.md` or `python/CODEX.md`
- Java: `java/AGENTS.md` or `java/CODEX.md`
- Go: `go/AGENTS.md` or `go/CODEX.md`
- Prompt changes: `prompts/domains/prompt-authoring.md`
- Task mode: one matching file under `prompts/task-modes/`

## Final Report

Include the change, reason, static evidence, commands actually run, checks not
run, residual risks, and optional focused checks that require authorization.

---

# AGENTS.md

## Purpose
This is the default local rule file for Go subtrees.
Use it when the task is mostly Go or the current directory contains Go markers.

## Inspect First
- `go.mod` and module layout
- `cmd/` entrypoints, handlers, services, and `internal/` packages
- config packages and tests near the touched code

## Detect The Go Domain
- Service markers:
  `cmd/server`, `internal/`, handler packages, config packages
  - also load `../prompts/domains/go-service.md`
- CLI markers:
  `cobra`, `urfave/cli`, command trees under `cmd/`, single-binary tools
  - also load `../prompts/domains/go-cli.md`

Use one domain prompt by default.
Load both only when the task truly spans both service and CLI code.

## Keep Stable
- package boundaries and `internal/` ownership
- exported APIs
- `context.Context` flow
- concurrency, retry, and shutdown behavior

## High-Risk Go Areas
- dependency changes
- exported API changes
- context or concurrency changes
- auth, schema, or deployment changes

## Validation Options
Use static review by default. Do not automatically run `scripts/verify.sh`,
tests, vet, or builds.

When the shared resource budget authorizes runtime validation, choose one
smallest relevant command:
- `go test ./path/to/package`
- `go vet ./path/to/package`
- `go build ./path/to/package`

Ask before repository-wide commands. Rerun only after a relevant change.

## Escalation
If a deeper subtree has its own `AGENTS.md` or `CODEX.md`, prefer that local file.
<!-- aiw-prompts:go:agents end -->
