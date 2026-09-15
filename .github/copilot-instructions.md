# .github/copilot-instructions.md

Read `AGENTS.md` first.

## Default Behavior

- use static analysis and minimal edits by default
- plan before non-trivial changes
- inspect only nearby code, tests, config, and contracts
- stop broad discovery after three targeted batches unless blocked
- preserve package boundaries and explicit error handling

## Resource Guard

After implementation, run one compile-only check: prefer `scripts/compile*` or
root `compile*`; otherwise use the narrowest language-level compiler command.
Do not automatically run tests, `build*` scripts, final-artifact builds,
formatters, linters, vet, verification scripts, network calls, permission
probes, privilege escalation,
`codex-auto-review`, or sub-agents.

After editing, use at most one static/read-only validation command by default.
Do not repeat equivalent commands. If runtime validation is authorized, run one
focused command and ask before widening.

## Final Summary

Include the change, static evidence, exact commands run, runtime checks not run,
risks, and optional checks requiring authorization.

<!-- aiw-prompts:go:copilot begin -->
# .github/copilot-instructions-for-go.md

Read `AGENTS.md` first, then `go/AGENTS.md`.

## Go Routing
- service markers:
  `cmd/server`, `internal/`, handler packages, config packages
  - also load `prompts/domains/go-service.md`
- CLI markers:
  `cobra`, `urfave/cli`, command trees under `cmd/`
  - also load `prompts/domains/go-cli.md`
- bugfix, feature, review, debugging, test, docs, or risky work:
  also load the matching file in `prompts/task-modes/`

## Defaults
- inspect local tests, config, and public contracts first
- respect package boundaries
- keep exported APIs and context flow stable unless the task requires change
- use static review by default
- after implementation, run one compile-only check using a `compile*` script or
  the narrowest language-level compiler command; do not retain a final artifact
- do not run tests, vet, `build*` scripts, final-artifact builds, verification
  scripts, network calls, or permission probes without authorization
- when authorized, run one package-focused command and ask before widening
<!-- aiw-prompts:go:copilot end -->
