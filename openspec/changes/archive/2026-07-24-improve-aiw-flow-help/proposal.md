## Why

`aiw-flow --help` currently exposes command names and raw option syntax but does not explain what the commands do, how they fit together, or how to start a common workflow. Users must leave the terminal and search the README before they can confidently construct even basic commands.

## What Changes

- Add a plain-language overview, workflow guidance, quick-start examples, and command summaries to top-level HELP.
- Add purpose, argument descriptions, constraints, and copyable examples to every top-level command.
- Add descriptive HELP and examples to nested `memory`, `handoff`, and `daemon` commands.
- Use consistent metavariables and terminology so generated usage lines are easier to scan.
- Add regression tests for HELP content and parser compatibility.

## Capabilities

### New Capabilities

- `aiw-flow-cli-help`: Discoverable, task-oriented terminal help for the aiw-flow command hierarchy.

### Modified Capabilities

None.

## Impact

- Affected implementation: `plugins/aiw-flow/src/codex_flow/cli.py`.
- Affected tests: focused CLI HELP and parser-compatibility tests.
- No command names, option names, defaults, runtime behavior, dependencies, or stored Session data change.
