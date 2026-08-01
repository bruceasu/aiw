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

Git convention:

- managed implementation branch: `feature/<task-id>`
- managed implementation worktree: `.wt/<task-id>`

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

- For managed engineering work, read `skills/work-management.md`. Prefer AIW
  for Task lifecycle and worktrees, and OpenSpec for requirement and checklist
  artifacts. If AIW is unavailable, use standalone OpenSpec or ordinary Git
  fallback whenever safe; do not invent AIW state.
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
## Go Addendum

For Go work, preserve package ownership, exported APIs, `context.Context`
propagation, explicit error handling, and concurrency shutdown behavior.

Treat `go.mod`, `go.sum`, public APIs, persistence, auth, deployment, and
concurrency changes as high-risk. Ask before adding or upgrading dependencies.

Go validation commands are options, not defaults. If runtime validation is
authorized, choose one smallest relevant command such as:

- `go test ./path/to/package`
- `go vet ./path/to/package`
- `go build ./path/to/package`

Do not start with `go test ./...`, `go vet ./...`, or `go build ./...`.
<!-- aiw-prompts:go:agents end -->
