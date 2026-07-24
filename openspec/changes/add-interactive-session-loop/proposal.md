## Why

aiw-flow preserves Session and Codex Thread state, but users must still invoke a full `continue` command for every conversational turn. An optional interactive loop will make Grill and other human-driven workflows comfortable while preserving one-shot commands for scripts and CI.

## What Changes

- Add `aiw-flow loop SESSION_ID` for repeated terminal input within one aiw-flow process.
- Add `--loop` and `--phase` to `new` so a normal Session can enter interaction before its first Codex turn.
- Route every submitted message through the existing turn execution path so prompts, outputs, events, status, timeouts, and Thread IDs remain authoritative.
- Add local commands for help, status, memory, handoff creation, Grill completion, and exit.
- Add `--loop` to `grill` so a new Grill session can remain interactive after its first response.
- Keep `new`, `run`, `continue`, and non-loop `grill` behavior unchanged.
- Handle empty input, EOF, and idle `Ctrl+C` without marking the Session failed.

## Capabilities

### New Capabilities

- `interactive-session-loop`: Optional terminal interaction for sending multiple turns to one aiw-flow Session.

### Modified Capabilities

- `grill-workflow`: Allow a newly started Grill interview to enter the interactive Session loop.

## Impact

- Affects the aiw-flow CLI coordinator, a new interactive loop module, tests, and documentation.
- Mirrors runtime changes into `plugins/aiw-flow`.
- Adds no dependency, daemon, persistent Codex subprocess, or status schema change.
- Maintains backward compatibility for all existing single-shot invocations.
