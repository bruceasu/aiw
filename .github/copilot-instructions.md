# .github/copilot-instructions.md

Read `AGENTS.md` first.

## Default Behavior

- use static analysis and minimal edits by default
- plan before non-trivial changes
- inspect only nearby code, tests, config, and contracts
- stop broad discovery after three targeted batches unless blocked
- preserve package boundaries and explicit error handling

## Resource Guard

Do not automatically run tests, builds, formatters, linters, vet, verification
scripts, network calls, permission probes, privilege escalation,
`codex-auto-review`, or sub-agents.

After editing, use at most one static/read-only validation command by default.
Do not repeat equivalent commands. If runtime validation is authorized, run one
focused command and ask before widening.

## Final Summary

Include the change, static evidence, exact commands run, runtime checks not run,
risks, and optional checks requiring authorization.
