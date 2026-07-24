## 1. HELP Structure

- [x] 1.1 Add shared argparse HELP formatting and command-construction metadata.
- [x] 1.2 Add a task-oriented top-level overview, workflow, command summaries, and quick starts.

## 2. Command Guidance

- [x] 2.1 Add descriptions, argument HELP, constraints, and examples to every top-level command.
- [x] 2.2 Add parent and action-level HELP for `memory`, `handoff`, and `daemon`.

## 3. Tests

- [x] 3.1 Add focused tests for top-level, command, and nested HELP output.
- [x] 3.2 Add parser compatibility tests for representative existing invocations and required arguments.

## 4. Verification

- [x] 4.1 Run compilation, focused HELP tests, and the complete existing aiw-flow regression suite.
- [x] 4.2 Run CLI HELP smoke tests, strict OpenSpec validation, and final diff review.

## Notes

%% HELP examples use Easy English and generic paths so they remain readable on Windows, Linux, and macOS.
%% Verification: compileall passed; 6 focused HELP tests passed; 51 existing regression tests passed; all 23 HELP paths passed; plugin entrypoint smoke test passed; strict OpenSpec validation and git diff checks passed.
