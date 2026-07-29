## Why

AIW currently has native task scaffolding while OpenSpec provides the richer
spec-driven workflow. Users with OpenSpec installed should get its canonical
behavior, while minimal installations must continue working without it.

## What Changes

- Add backend selection for AIW task workflow commands with `auto`,
  `openspec`, and `native` modes.
- In `auto`, detect a usable OpenSpec CLI and delegate supported operations;
  otherwise use the existing AIW implementation.
- Keep `native` behavior explicit and stable for scripts and offline installs.
- Report which backend was selected and provide actionable errors when
  `openspec` is explicitly requested but unavailable.
- Keep OpenSpec Skills (`to-spec`, `implement`, and archive workflows) as the
  user-facing orchestration layer.

## Capabilities

### New Capabilities

- `workflow-backend-routing`: Select and diagnose the OpenSpec or native task
  workflow backend.

### Modified Capabilities

- None. Existing native task commands remain backward-compatible; routing is
  introduced as an implementation seam and explicit option.

## Impact

- Affected Go task command parsing and subprocess execution.
- Affected CLI help and README documentation.
- No new third-party dependencies.
- OpenSpec CLI remains an optional external dependency.
