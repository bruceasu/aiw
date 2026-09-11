# Implementation Contract

Status: COMPLETE

## CLI

- `aiw ask "<prompt>"` performs one structured LLM turn.
- `--chat` and `--resume` enter lightweight promptui chat; `/send` submits
  accumulated lines and `/exit` ends the session.
- `--system-prompt`, `--system-prompt-file`, and repeatable `--allow-path` are
  accepted by the command surface.

## Response

The LLM is requested to return schema version `1.0`; invalid JSON or missing
required fields is reported as an error without retry.

## Persistence

Turns are appended under `$HOME/.aiw/ask/<YYYY-MM-DD>/` using mode `0600` and a
Windows-safe UTC timestamp plus SHA-256 prompt filename. Failed turns contain
status/error only and no answer payload.

%% NEEDS_INPUT: Provider-specific `--add-dir` forwarding and strict Codex
read-scope enforcement remain deferred to the provider boundary task.
