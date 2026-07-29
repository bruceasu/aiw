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
