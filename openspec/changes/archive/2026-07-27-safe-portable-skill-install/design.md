## Context

AIW already discovers and executes external plugins named `aiw-<command>`, and
the repository keeps canonical Skill sources under the root `skills`. The
existing `aiw-install-skill` plugin accepts arbitrary folders and archives, but
it defaults to Codex-specific locations, copies directly into the destination,
and uses recursive deletion for force replacement.

Ticket 01 needs a narrow new public seam: `aiw skills`. It must list canonical
Skills and safely install one Portable Skill into the current project. Later
tickets will add scope/target selection, bundles, lifecycle operations, links,
and capability integration.

Constraints include Python 3.9 compatibility, standard-library-only
implementation, cross-platform paths, no `shell=True`, and behavior tests
through the public CLI.

## Goals / Non-Goals

**Goals:**

- Provide `aiw skills list` and `aiw skills install <name>`.
- Default installation to the current project's `.agents/skills`.
- Validate `name` and `description` frontmatter before writes.
- Stage and hash content before making it visible.
- Record managed ownership and integrity in a versioned manifest.
- Make identical reinstallations idempotent.
- Refuse unmanaged same-name destinations.
- Support dry-run, human-readable output, JSON output, and stable exit codes.

**Non-Goals:**

- User scope or Codex-specific targets.
- Bundle, archive, update, remove, doctor, or link operations.
- Duplicate resolution across multiple discovery roots.
- AIW capability manifests or `aiw-flow` integration.
- Refactoring or changing the behavior of `aiw-install-skill`.

## Decisions

### Add a separate `aiw-skills` plugin

The feature will be a new external plugin instead of extending the existing
generic archive installer. This keeps the managed canonical-source contract
separate from the older arbitrary-source behavior and avoids changing existing
scripts.

Alternative considered: modify `aiw-install-skill` in place. Rejected because
its source model, targets, and destructive overwrite flags do not match the
safe managed-install contract.

### Test through the CLI seam

Behavior tests will execute the plugin entry point with arguments and an
isolated working directory, home, and canonical source fixture. Tests will
assert output, exit status, installed content, and the manifest rather than
private helper calls.

An environment override for the canonical source root will support isolated
tests. Normal execution will resolve the repository canonical source from the
AIW/plugin installation layout.

### Keep canonical Skills beside program and plugins

The repository and release layout will use a root `skills` directory. The
plugin at `plugins/aiw-skills/aiw-skills.py` resolves two levels upward to the
installation root and then selects `skills`. Release scripts copy this
directory without changing its name or nesting.

Alternative considered: keep `program/skills`. Rejected because Skills are
agent-consumed workflow packages, not executable program code, and the name
would make the public release layout misleading.

### Use a deterministic directory digest

The content digest will hash every regular file in stable relative-path order,
including each normalized relative path and its bytes. Symlinks and
non-regular filesystem entries in a canonical Skill will be rejected for this
first copy-only slice.

Alternative considered: hash only `SKILL.md`. Rejected because references,
scripts, and agent metadata are part of the installed Skill.

### Stage one new Skill before publication

The installer will copy the selected Skill to a unique staging directory on the
same destination filesystem, verify the staged digest, then atomically rename
it to its final name. The destination must not already exist unless it is an
identical AIW-managed installation.

This ticket does not replace existing content, so rollback is narrow: if the
manifest write fails after publication, remove only the newly published
managed directory.

### Store a versioned manifest at the target root

The JSON manifest will record schema version and a map of managed Skills. Each
entry records source identity, source revision when discoverable, install mode,
and SHA-256 digest. Manifest writes use a temporary sibling plus atomic replace.

The manifest is ownership evidence. A directory without a matching managed
entry is unmanaged even when its contents happen to equal the source.

### Keep command output deterministic

Human-readable output is the default. `--json` returns one JSON object for both
success and operational failure. Successful no-op reinstalls report an
idempotent state distinct from a new install. Argument parser errors retain the
parser's standard usage exit behavior.

## Risks / Trade-offs

- [Canonical source cannot be found in a repackaged installation] → Report the
  searched root clearly; bundle packaging is handled by a later ticket.
- [Directory rename semantics differ on Windows] → Only rename a new staged
  directory into a nonexistent destination and keep staging on the same
  filesystem.
- [Manifest failure occurs after publishing the Skill] → Remove only the new
  directory and preserve the prior manifest.
- [Source changes during copying] → Hash both source and staged content and
  abort when they differ.
- [Simple frontmatter parsing accepts less YAML than a full parser] → Support
  the same single-line `name` and `description` contract already used by
  `aiw-flow`; reject unsupported forms explicitly.
- [A hostile process can race filesystem validation] → Preserve symlinks while
  staging and validate both source and staged trees again before publication.
  Fully eliminating cross-platform TOCTOU requires directory-handle traversal
  outside this ticket's standard-library portability boundary.

## Migration Plan

1. Move canonical content from `program/skills` to root `skills`.
2. Add the new plugin without changing existing commands.
3. Copy root `skills` into the release installation root.
4. Document the new single-Skill flow.
5. Verify direct plugin execution and normal AIW plugin dispatch.
6. Rollback consists of removing the new plugin and its documentation; existing
   installers remain unchanged.

## Open Questions

%% None for Ticket 01. Scope selection, bundles, replacement policy, and links
are deliberately deferred to their approved tickets.
