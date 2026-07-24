## Context

The `aiw-git` dispatcher imports every `git-*.py` file while discovering
subcommands. `git-guide.py` declares two internal data models with
`@dataclass(slots=True)`. Python 3.9 accepts the module's other syntax but its
`dataclass` function does not accept `slots`, so one optional subcommand stops
the entire dispatcher from starting.

## Goals / Non-Goals

**Goals:**

- Keep `git-guide.py` importable on Python 3.9.
- Preserve the fields, defaults, and behavior of its internal data models.
- Cover real dispatcher discovery in the regression tests.

**Non-Goals:**

- Supporting Python versions older than 3.9.
- Changing dispatcher failure isolation or command semantics.
- Adding dependencies or redesigning the guide feature.

## Decisions

- Remove `slots=True` from the two internal dataclass decorators. Plain
  dataclasses are supported by Python 3.9 and preserve construction, equality,
  representation, and field defaults used by the module.
- Register dynamically loaded Python subcommands in `sys.modules` before
  executing them, matching normal import behavior. Python 3.9 dataclasses need
  this registration to resolve postponed annotations. Restore any previous
  entry if execution fails so a partially initialized module is not retained.
- Keep the existing dispatcher discovery test as the regression boundary and
  assert that the real `guide` command is present. This directly exercises the
  import path that failed.
- Do not add a compatibility shim that conditionally rewrites dataclass
  arguments. The models do not require slot semantics, so a shim would add
  complexity without preserving observable CLI behavior.

## Risks / Trade-offs

- [Plain dataclass instances have a `__dict__` and slightly higher memory use]
  → These are short-lived internal records, so the impact is negligible.
- [Other Python 3.10-only features could remain] → Compile and execute the
  dispatcher and tests with the available Python 3.9.13 runtime.
- [A generated module name could collide with an existing module] → Preserve
  and restore the prior `sys.modules` entry when dynamic execution fails.
