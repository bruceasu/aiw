# AGENTS.md

## Purpose
This is the default local rule file for Python subtrees.
Use it when the task is mostly Python or the current directory contains Python markers.

## Inspect First
- `pyproject.toml`, `requirements*.txt`, or environment files
- the nearest package or app entrypoints
- tests near the touched code
- config or settings modules

## Detect The Python Domain
- Service or API markers:
  `fastapi`, `django`, `flask`, `pydantic`, `app/`, `src/`, `tests/`
  - also load `../prompts/domains/python-service.md`
- CLI or tool markers:
  `typer`, `click`, `argparse`, `__main__.py`, `scripts/`
  - also load `../prompts/domains/python-cli.md`

Use one domain prompt by default.
Load both only when the task truly spans both service and CLI code.

## Keep Stable
- existing typing style
- request, response, and validation behavior
- config and environment loading patterns
- packaging and entrypoint conventions

## High-Risk Python Areas
- dependency changes
- async, multiprocessing, or task-queue behavior
- auth, schema, or migration changes
- runtime or deployment config

## Validation Options
Use static review by default. Do not automatically run `scripts/verify.sh`,
formatters, linters, type checks, tests, or builds.

When the shared resource budget authorizes runtime validation, choose one
smallest relevant command:
- `ruff check path/to/module.py`
- `ruff format --check path/to/module.py`
- `mypy path/to/package`
- `pytest tests/test_name.py`

Ask before repository-wide commands. Rerun only after a relevant change.

## Escalation
If a deeper subtree has its own `AGENTS.md` or `CODEX.md`, prefer that local file.
