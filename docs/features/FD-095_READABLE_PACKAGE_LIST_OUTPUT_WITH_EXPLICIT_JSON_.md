# FD-095: Readable package list output with explicit JSON mode

**Status:** Pending Verification
**Revision:** 5
**Priority:** Medium
**Evidence policy:** Dual

## Problem and intent

User reports aiw pkg list is not useful as a raw JSON dump. Both repository and
installed listing paths currently print full metadata including internal fields.
User requests a human-readable default; explicit JSON must remain available.

## Decision and compatibility

Default list output is a plain aligned table with NAME, VERSION and KIND.
Installed listings add REASON (manual/auto, legacy default manual). Sort displayed
rows by name then version for stable output; do not mutate or sort the JSON data.
Print a clear empty-list message. Add list --json to retain the original JSON array
and all existing fields for both repository and --installed listings.

Keep --all latest-versus-all selection, --kind filtering, repository configuration,
--dest selection and exit codes unchanged. search/info/show/entries/skill commands
keep existing output. No new dependencies, terminal detection, ANSI colors,
network calls, persistent state changes or installation required by this task.

Changing the default output is the user's requested behavior. Existing consumers
of JSON must explicitly use --json; docs must call this out. Independent bounded
design feedback checks the output/compatibility scope before implementation.
Continue in the user-selected primary workspace; no new branch/worktree/receipt.
No linked Issue or OpenSpec change.

## Work Items

- [x] 1.1 Render package list tables and add explicit JSON mode in both list paths.
  Size small, difficulty low, no dependencies. Completion: filters and array data
  preserved, installed reason defaults and empty output handled, other commands unchanged.
- [x] 1.2 Document output contract and prepare narrow compile/static evidence.
  Size small, difficulty low, depends on 1.1. Completion: docs/spec and Python
  in-memory compile result recorded. Independent review is the lifecycle gate.

## Acceptance

- list and list --installed show readable tables by default, without file metadata.
- --json preserves complete original JSON arrays and filtering for both sources.
- --all/--kind and latest selection remain intact; empty lists are readable.
- show/info/search/entries/skill output and installation records are unchanged.

## TODO

- Independent design ready; implementation/docs and Python compile-only complete.
- Independent implementation review remains required before completion.

## Verification

One in-memory Python compile of aiw-pkg.py plus one static diff inspection.
No tests, runtime list invocation, network, final build or installation.
Python in-memory compile exited 0. One implementation diff inspection traced both
list branches, display-only sorting, JSON preservation, empty selections and
short-circuit handling for other subcommands. See implementation report.

## Sources

- src/plugins/aiw-pkg/aiw-pkg.py
- docs/usage/aiw-pkg.md
- openspec/specs/aiw-packages/spec.md
