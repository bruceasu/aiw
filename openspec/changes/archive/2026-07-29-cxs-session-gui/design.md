## Context

`plugins/aiw-cxs.py` currently combines session JSONL discovery, metadata
caching, alias persistence, readable rendering, argument parsing, and Codex
process execution in one Python module. Session files are global under
`~/.codex/sessions`, while aliases are already scoped by the selected workspace
through `.ai/sessions/index.json`.

The current session model does not retain the session's original working
directory. Listing therefore includes sessions from every workspace, and resume
uses `codex exec resume` unless the caller manually supplies a working directory.
The requested GUI must default to the current workspace and hand off to
interactive `codex resume`.

## Goals / Non-Goals

**Goals:**

- Preserve the current CLI and alias-index behavior.
- Add original-working-directory metadata with backward-compatible caching.
- Provide workspace-scoped listing as a reusable core operation.
- Add a standard-library desktop GUI for listing, previewing, alias editing,
  refreshing, and interactive resume.
- Add a concise `resume-ext` skill that reuses machine-readable `aiw cxs`
  output and presents numbered session options.
- Keep help text and skill prompts in Easy English.

**Non-Goals:**

- Replacing Codex's native session storage or modifying session JSONL files.
- Embedding an interactive terminal emulator in the GUI.
- Resuming a different thread inside the currently running Codex process.
- Adding synchronization, remote session discovery, or multi-user aliases.
- Adding an external GUI framework dependency.

## Decisions

### Use the session working directory as workspace identity

The session scanner will extract an optional `original_cwd` from known Codex
metadata shapes and store it in `SessionMeta` and the metadata cache. A session
belongs to the current workspace when its normalized original directory equals
the normalized workspace directory or is a descendant of it. Path comparison
will follow platform case-sensitivity rules.

Sessions without recoverable working-directory metadata will be excluded from
the default workspace view and included in the explicit all-workspaces view with
an "unknown workspace" marker.

This keeps the default conservative and avoids showing unrelated global
sessions. Title-based or path-name inference was rejected because it can assign
a session to the wrong project.

### Keep alias scope and session scope separate

The existing `--workspace` value remains the root for `.ai/sessions/index.json`.
The same normalized root is the default GUI and skill session filter, but
filtering is a separate operation over `SessionMeta`. The existing `list`
command remains global by default for backward compatibility and gains an
explicit workspace-only option. Alias storage remains schema-compatible.

Alias rename will be represented as an atomic index update: validate the new
name, reject conflicts unless replacement is explicitly confirmed, add the new
entry, remove the old entry, and save once.

### Evolve the cache without migrating session data

The metadata cache schema will be incremented or otherwise invalidated so
unchanged legacy cache records are rescanned for `original_cwd`. Loading remains
tolerant of old cache files. The alias index schema and existing alias entries
will not be rewritten merely by opening the GUI.

### Build the GUI as a thin adapter over reusable operations

Tkinter/ttk will provide the desktop UI because it is in the Python standard
library and is sufficient for a session table, bounded text preview, form
dialogs, and action buttons.

Core operations will return values or raise `AiwCxsError`; CLI handlers and GUI
callbacks will adapt those results instead of parsing each other's printed
output. The GUI will use:

- a workspace-scoped session query;
- bounded user/assistant preview rendering;
- alias create, rename, and remove operations;
- a command builder for interactive resume.

The default view will show alias, title, updated time, turn count, and original
directory. "Show all workspaces" is explicit and reversible. Preview will omit
raw system/tool events by default and will never write to session files.

### Hand interactive resume back to the invoking terminal

The resume action will construct `codex resume <session-id>` and run it with the
selected session's original directory as the child process working directory.
The GUI will close before the interactive process starts so Codex can own the
invoking terminal. If no usable original directory or interactive terminal is
available, the GUI will not launch Codex and will show the exact command and a
clear recovery message.

Launching `codex exec resume` was rejected because the user explicitly selected
interactive continuation. Embedding or spawning a nested terminal was rejected
for the first version because it adds platform-specific process and quoting
behavior.

### Make `resume-ext` a thin selection workflow

The skill will be named `resume-ext` and use the supported skill invocation
surface, such as `$resume-ext`. It will request workspace-scoped, machine-readable
session data, render a compact numbered list, accept a number or alias, and
produce the exact interactive resume command and original directory.

The skill will not launch a nested Codex process from an active Codex session.
Where the host cannot perform a native thread handoff, it will provide a
copyable command. An exact `/resume-ext` slash alias is an optional host/plugin
surface, not a guarantee made by the skill itself.

No script will be bundled inside the skill; deterministic session logic remains
owned by `aiw cxs`.

## Risks / Trade-offs

- [Codex JSONL metadata shapes can vary by version] → Use tolerant extraction
  over known metadata keys, preserve unknown sessions, and mark missing
  workspaces instead of guessing.
- [A session may have started in a subdirectory] → Treat descendants as members
  of the selected workspace and display the exact original directory.
- [Tkinter can be unavailable in some Python distributions] → Fail with an
  actionable message while leaving all CLI commands operational.
- [Interactive resume requires a terminal] → Detect the condition before
  launch and provide a copyable command when handoff is unavailable.
- [Conversation logs may contain sensitive content] → Keep all processing local,
  bound preview size, omit system/tool events by default, and never modify JSONL.
- [Skill syntax differs from slash-command syntax] → Document `$resume-ext` as
  canonical and treat `/resume-ext` as a separate optional integration.

## Migration Plan

1. Add tolerant `original_cwd` extraction and invalidate only the metadata
   cache records that lack the new field.
2. Add reusable filtering, alias mutation, preview, and interactive command
   operations while preserving existing CLI handlers.
3. Add the GUI command and documentation.
4. Add the `resume-ext` skill against the machine-readable command surface.
5. Preserve rollback by allowing the GUI/skill additions to be removed without
   changing session JSONL or the alias index.

## Open Questions

None. Implementation inspection confirmed `session_meta.payload.cwd` in a
representative local JSONL record and confirmed the installed CLI accepts
`codex resume [SESSION_ID]`.

## Remaining Risks

- %% Codex can change its JSONL metadata shape in a future release. Unknown
  shapes remain unscoped instead of falling back to ordinary event payloads.
- %% The GUI controller is intentionally kept in the existing single-file
  plugin for this change. Extract a separate UI module if additional GUI
  workflows make the controller materially larger.
