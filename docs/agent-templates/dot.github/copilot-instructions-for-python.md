# .github/copilot-instructions-for-python.md

Read `AGENTS.md` first, then `python/AGENTS.md`.

## Python Routing
- service or API markers:
  `fastapi`, `django`, `flask`, `pydantic`, `app/`, `tests/`
  - also load `prompts/domains/python-service.md`
- CLI or tool markers:
  `typer`, `click`, `argparse`, `__main__.py`
  - also load `prompts/domains/python-cli.md`
- bugfix, feature, review, debugging, test, docs, or risky work:
  also load the matching file in `prompts/task-modes/`

## Defaults
- inspect local tests, schemas, and config first
- keep typing and config patterns consistent
- keep framework glue thin
- use static review by default
- after implementation, run one compile-only check using a `compile*` script or
  the narrowest language-level compiler command; do not retain a final artifact
- do not run formatters, linters, type checks, tests, `build*` scripts,
  final-artifact builds, verification scripts, network calls, or permission
  probes without authorization
- when authorized, run one path-focused command and ask before widening
