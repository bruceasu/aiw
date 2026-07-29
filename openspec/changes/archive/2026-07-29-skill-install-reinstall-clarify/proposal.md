# Proposal: Clarify AIW Skill install, adopt, and sync semantics

AIW's current skill installer treats a managed destination as immutable once it
has been published. That is safe for protecting user-owned content, but it
blocks normal reinstall and upgrade workflows. A managed Skill can no longer be
reinstalled from the canonical source after source changes, and the installer
cannot express "publish the next version of this already managed Skill".

This change separates the user-facing operations by ownership intent:

- `install` publishes a canonical Skill into an empty destination and may
  replace an already managed destination with the selected canonical source.
- `adopt` records an existing valid installed Skill as AIW-managed without
  changing the directory content.
- `discover` reports installed Skills and their managed status.
- `sync` republishes the canonical source into an already managed destination
  and refreshes the manifest digest.

The result is a safer contract for unmanaged user content and a practical
upgrade path for managed Skills.

## Goals

- Preserve protection for unmanaged same-name destinations.
- Allow reinstall and upgrade of managed Skills.
- Make the ownership model discoverable and explicit.
- Keep the manifest authoritative for managed ownership.

## Non-goals

- Installing arbitrary non-Skill directories.
- Automatically forcing adoption of unmanaged destinations.
- Changing canonical source discovery rules.
- Introducing a general-purpose package manager for non-Skill assets.

## User Impact

- Users can reinstall a managed Skill after source changes without first
  deleting the destination.
- Users can inspect installed Skills before deciding whether to adopt or
  replace them.
- Existing unmanaged destinations remain protected unless the user explicitly
  adopts or removes them.
