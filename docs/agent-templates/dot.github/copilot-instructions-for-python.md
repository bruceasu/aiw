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
- do not run formatters, linters, type checks, tests, builds, verification
  scripts, network calls, or permission probes without resource-budget
  authorization
- when authorized, run one path-focused command and ask before widening
