# CODEX.md

Always respond in Chinese.
Follow `AGENTS.md` first.

## Default Mode

- Use static analysis and code editing by default.
- Keep the active context and prompt set small.
- For non-trivial work, state understanding, affected files, assumptions, plan,
  risks, and the proposed validation level before editing.
- Do not treat implementation as permission to test, build, use the network,
  probe permissions, escalate privileges, or start automated reviews.

## Execution Sequence

1. Read the nearest relevant instructions and OpenSpec artifacts.
2. Inspect only the affected code, config, docs, and nearby tests.
3. Apply the smallest correct change.
4. Review the final diff statically.
5. Run no executable validation unless `AGENTS.md` authorizes it.
6. Report what was and was not verified.

Stop broad exploration after three targeted discovery batches unless a concrete
unknown blocks the task. Batch related reads and searches, limit command output,
and do not repeat equivalent commands.

## Go Guidance

- Preserve package boundaries and exported behavior.
- Prefer concrete types unless an existing seam requires an interface.
- Preserve context cancellation, timeouts, retries, shutdown, and error flow.
- Treat concurrency, dependencies, public APIs, auth, persistence, migrations,
  deployment, and CI as high-risk.

## Validation

Static review is sufficient by default. Tests, builds, formatting, linting, vet,
verification scripts, networking, and `codex-auto-review` have a default budget
of zero.

If runtime validation is authorized, run one narrow command. A second run is
allowed only after a relevant change. Ask before widening scope.

## Completion

A task is complete when the requested change is implemented and the final
report clearly distinguishes:

- static evidence;
- commands actually run;
- checks intentionally not run;
- residual risks.

Do not claim runtime correctness without runtime evidence.
