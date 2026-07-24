## Context

aiw-flow currently implements one physical process invocation per turn while persisting a logical multi-turn Codex Thread. This is appropriate for scripts and CI, but repetitive for human conversations because every answer requires a full `continue` command.

The existing `_execute_turn` path already owns prompt composition, Session state, event recording, backend creation, timeout handling, outputs, and Thread binding. Interactive mode must reuse it rather than create a second execution protocol.

## Goals / Non-Goals

**Goals:**

- Keep one aiw-flow CLI process open for repeated terminal input.
- Support interaction immediately after normal Session creation, immediately after Grill creation, or when resuming an existing Session.
- Preserve exactly the same persisted turn artifacts as one-shot execution.
- Provide a small set of local slash commands that do not consume Codex turns.
- Exit cleanly on explicit exit, EOF, or idle keyboard interruption.
- Preserve all existing one-shot command behavior.

**Non-Goals:**

- Keep one persistent `codex exec` subprocess alive.
- Add a daemon, socket, server, or multi-Session scheduler.
- Add a full-screen TUI, history editor, or multiline editor.
- Change Codex Thread semantics, Session schema, locking, or backend protocols.
- Intercept keyboard interruption while a Codex subprocess is actively executing.

## Decisions

### Add three entry paths

- `new ... --loop [--phase PHASE]` creates a normal Session and waits for its first user message. The first submitted message binds the Codex Thread.
- `grill ... --loop` runs the existing first Grill turn, then waits for answers.
- `loop SESSION_ID [--phase PHASE] [--timeout SECONDS]` resumes or starts interaction for an existing Session.

The standalone loop uses the explicit phase when supplied, otherwise the Session's current phase, otherwise `interactive`.

### Reuse one-shot turn execution

Every ordinary input line will call `_execute_turn` with the selected phase and timeout. A Session without a Thread uses first-turn semantics naturally; a Session with a Thread passes the saved Thread ID and resumes it.

The outer process remains alive, but each turn still starts and closes one `codex exec` subprocess. A persistent subprocess was rejected because `codex exec` is a one-turn interface and aiw-flow would lose reliable lifecycle boundaries.

### Keep parsing separate from coordination

`interactive_loop.py` will define the help text and parse a line into one of:

- message
- empty
- help
- status
- memory
- handoff
- done
- exit
- unknown command

`cli.py` will coordinate these actions using existing status, Memory, Handoff, and turn functions. This keeps the parser deterministic and independently testable.

### Define local command behavior

- `/help` prints available commands.
- `/status` prints current Session status.
- `/memory` prints current Session Memory.
- `/handoff` creates the deterministic handoff artifact.
- `/done` is valid in phase `grill`, sends `Grill Done` as the final Codex turn, then exits after the response.
- `/exit` exits without sending a turn or changing Session state.
- Unknown slash commands print a local error and do not call Codex.

### Treat EOF and idle Ctrl+C as normal exit

The loop catches `EOFError` and `KeyboardInterrupt` only while waiting for terminal input, prints a short exit message, and returns success. It does not catch interruption inside `_execute_turn`; subprocess cancellation behavior remains owned by the backend.

### Preserve one-shot compatibility

`new` and `grill` without `--loop` keep their current return points and output. `run` and `continue` are unchanged. New options only activate interaction when explicitly requested.

## Risks / Trade-offs

- [Each turn still pays subprocess startup cost] → Optimize user ergonomics first; a persistent backend requires a separate protocol design.
- [Single-line input is limited for large prompts] → Keep the first version predictable and record multiline/editor support as a follow-up.
- [Terminal input blocks the event loop while idle] → No concurrent async work exists between turns, so synchronous `input()` is simpler and preserves Ctrl+C behavior.
- [A Session may be completed, archived, or already running] → Reject non-interactive states with a clear error before entering the loop.
- [Slash-prefixed user content can be mistaken for a command] → Treat only the full first token as a known command and report how to send ordinary text in documentation.

## Migration Plan

1. Add the parser and loop coordinator behind new commands/options.
2. Add focused tests with mocked input and turn execution.
3. Mirror runtime changes to the plugin package.
4. Roll back by removing the optional command and flags; persisted Sessions and turns remain valid.

## Open Questions

%% Future change: add multiline input or external-editor integration after observing real usage.

%% Future change: evaluate a persistent MCP or daemon backend only if subprocess startup becomes a measured bottleneck.
