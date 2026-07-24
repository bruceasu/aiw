## Why

`aiw git` cannot start under the repository's current Python 3.9 runtime because
subcommand discovery imports `git-guide.py`, which uses the Python 3.10-only
`dataclass(slots=True)` option. The dispatcher must remain usable on Python 3.9.

## What Changes

- Make the `git-guide` data models importable on Python 3.9.
- Add regression coverage proving the dispatcher discovers the real `guide`
  subcommand without an import-time exception.
- Preserve existing `aiw git guide` behavior and command metadata.

## Capabilities

### New Capabilities

- `aiw-git-python-compatibility`: Defines Python 3.9 compatibility for loading
  and discovering `aiw-git` subcommands.

### Modified Capabilities

None.

## Impact

The change is limited to `plugins/aiw-git/git-guide.py`, its dispatcher tests,
and OpenSpec artifacts. It changes no public CLI, dependency, or persistent
data contract.
