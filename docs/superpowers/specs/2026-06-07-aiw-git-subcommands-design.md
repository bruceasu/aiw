# AIW Git Subcommands Design

## Summary

This change restructures the git plugin family from many top-level plugin entrypoints like
`plugins/aiw-git-show.py` into a single plugin entrypoint:

- `aiw git`

The new entrypoint remains `plugins/aiw-git/aiw-git.py`, and all git subcommand scripts move into
the same directory as implementation files. The root-level `plugins/aiw-git-*.py` files are removed
without compatibility aliases.

## Goals

- Make `aiw git` the single public entrypoint for git-related plugin commands.
- Move all git subcommand scripts into `plugins/aiw-git/` and rename them to `git-*`.
- Let `aiw-git.py` discover, dispatch, and document git subcommands.
- Provide two help levels:
  - concise overview for `aiw git`, `aiw git -h`, and `aiw git help`
  - detailed per-command help for `aiw git help <subcommand>` and `aiw git <subcommand> -h`
- Keep diffs localized to plugin files and related usage documentation.

## Non-Goals

- Preserve old command names such as `aiw git-show` or `aiw git-st`.
- Refactor unrelated plugin discovery or global command routing behavior.
- Redesign the underlying git command implementations beyond what is needed for the new structure.

## Command Model

### Public CLI shape

Supported command forms after migration:

- `aiw git`
- `aiw git -h`
- `aiw git --help`
- `aiw git help`
- `aiw git help <subcommand>`
- `aiw git <subcommand> [args...]`
- `aiw git <subcommand> -h`
- `aiw git <subcommand> --help`

Removed command forms:

- `aiw git-show ...`
- `aiw git-add-mirror ...`
- all other `aiw-git-*` top-level plugin entrypoints

### Help behavior

Overview help:

- show command summary, one-line description, and short example per subcommand
- sorted consistently
- optimized for scanning rather than completeness

Detailed subcommand help:

- show command name
- short summary
- long description when available
- usage
- flags/arguments
- examples

## File Layout

### Before

- `plugins/aiw-git/aiw-git.py`
- `plugins/aiw-git-show.py`
- `plugins/aiw-git-add-mirror.py`
- `plugins/aiw-git-add-remote.py`
- many more `plugins/aiw-git-*.py`

### After

- `plugins/aiw-git/aiw-git.py`
- `plugins/aiw-git/aiw-git-core.py`
- `plugins/aiw-git/git-show.py`
- `plugins/aiw-git/git-add-mirror.py`
- `plugins/aiw-git/git-add-remote.py`
- all remaining git subcommand scripts in the same directory

No root-level `plugins/aiw-git-*.py` files remain.

## Discovery and Dispatch

`plugins/aiw-git/aiw-git.py` becomes the coordinator.

Responsibilities:

- scan its own directory for files matching:
  - `git-*.py`
  - `git-*.bat`
  - `git-*.sh`
  - `git-*.ps1`
  - `git-*` without extension when executable/script rules allow it
- exclude implementation helpers such as `aiw-git.py` and `aiw-git-core.py`
- convert filenames to subcommand names
  - `git-show.py` -> `show`
  - `git-add-mirror.py` -> `add-mirror`
- dynamically load the target module
- call its exported `main(argv)` function
- read command metadata for help rendering

The global Go plugin discovery does not need structural changes because it already supports
`plugins/<dir>/aiw-<name>.py`, which matches `plugins/aiw-git/aiw-git.py`.

## Subcommand Contract

Each Python git subcommand script should follow one consistent contract:

- define `META`
- define `main(argv)`

For non-Python script forms such as `.bat`, `.sh`, `.ps1`, or extensionless executables, the
coordinator can still list and dispatch them, but detailed metadata may need to come from a simpler
convention if they are introduced later. For the current migration, the expected command
implementations remain Python scripts renamed to `git-*.py`.

Expected `META` shape:

```python
META = {
    "name": "show",
    "short": "Inspect repository state and history views.",
    "long": "Longer explanation shown in detailed help.",
    "usage": "show <mode> [options]",
    "args": [
        {"flag": "--diff", "description": "Show unresolved hunks."},
    ],
    "examples": [
        "show status",
        "show log -n 20",
    ],
}
```

Rules:

- `name` must match the filename-derived subcommand name
- `short` is required for overview help
- `usage` and `examples` should exist for all public commands
- `long` and `args` may be empty but should be present where useful

## Help Rendering Rules

### Overview output

Shown by:

- `aiw git`
- `aiw git -h`
- `aiw git --help`
- `aiw git help`

Content:

- top-level usage line
- short explanation of the command family
- list of subcommands with aligned short descriptions
- one short example line per subcommand if present
- hint to use `aiw git help <subcommand>` for details

### Detailed output

Shown by:

- `aiw git help <subcommand>`
- `aiw git <subcommand> -h`
- `aiw git <subcommand> --help`

Content:

- usage
- short summary
- long description
- option list
- multiple examples

If a subcommand is unknown, return non-zero and show a short error plus the overview hint.

## Error Handling

- unknown subcommand: exit `1`
- subcommand help request on missing command: exit `1`
- module load failure: exit `1` with actionable filename context
- subcommand execution returns its own exit code when available

The coordinator should avoid swallowing subprocess or module errors.

## Testing Strategy

This change should be implemented test-first where practical.

Priority checks:

- `aiw-git.py` can discover moved subcommands from its own directory using the `git-*` naming rule
- overview help renders all discovered commands in concise form
- `help <subcommand>` renders detailed metadata
- `aiw git <subcommand> ...` dispatches to the correct module
- root-level `plugins/aiw-git-*.py` files are no longer required

Suggested verification commands:

```bash
python plugins/aiw-git/aiw-git.py -h
python plugins/aiw-git/aiw-git.py help show
go test ./internal/plugin ./internal/commands/help
```

Additional targeted checks may be added if repository-standard Python tests are introduced.

## Documentation Impact

The existing usage docs under `docs/usage/aiw-git-*.md` may still be valuable, but they no longer
describe top-level plugin entrypoints accurately after this change.

Minimum required documentation updates:

- add or update documentation describing `aiw git` as the entrypoint
- remove wording that implies `aiw git-show` style usage remains supported

## Risks

- Some existing git subcommand scripts may not expose a uniform `META` structure, so small metadata
  normalization edits may be required during migration.
- The new scan rule accepts non-Python script extensions, but detailed help generation is simplest
  for Python-based commands with in-module metadata. If non-Python git subcommands are added later,
  they may need an auxiliary metadata convention.
- The current `help` command in Go scans plugin source for `short`; after files move, it should still
  find `plugins/aiw-git/aiw-git.py`, but it will only describe the top-level `git` plugin unless
  additional future work teaches it about plugin-internal subcommands.
- Usage docs may lag behind if only code is migrated.

## Implementation Scope

In scope:

- git plugin file moves
- `aiw-git.py` dispatcher/help redesign
- metadata normalization needed for git subcommands
- git plugin documentation updates directly affected by the new entrypoint

Out of scope:

- global nested command support beyond `aiw git`
- non-git plugin restructuring
- backward-compatibility wrappers for removed root-level git plugin files
