# FD-092: Unique ownership for package usage documents

**Status:** Open
**Revision:** 3
**Priority:** Medium
**Evidence policy:** Dual

## Problem

`install all` stops because aiw-file includes docs/usage/aiw-file-convert.md,
already owned by aiw-file-convert 1.0.0. Static inspection confirmed both the
installed record and incoming aiw-file ZIP metadata. Broad prefix matching
in scripts/package_definitions.py assigns the same document to both packages.

## Decision and scope

Choose the longest matching document prefix across current plugin components
and the main component. Match a complete stem or a prefix followed by a hyphen.
Preserve aiw-cz/cz and aiw-req/aiw-issue aliases and main's explicit documents.
This fixes nested names without a one-file exclusion that would miss future
overlaps. Keep the existing aiw-docs docs/reference namespace unchanged.

This is an internal packaging correction with no new public interface,
dependency, installation ownership policy or migration. No separate design
review is required; independent implementation review remains required.
The user explicitly chose the primary workspace: create no branch,
worktree or workspace receipt. Do not rebuild ZIPs, install, change installed
records, publish, or bypass ownership checks. Existing ZIPs require an explicit
later build. No linked Issue or OpenSpec change exists.

## Work Items

- [ ] 1.1 Resolve usage document ownership by most specific package prefix.
  Size small, difficulty low, dependencies none. Completion: file/file-convert,
  exact matches, aliases and existing explicit main documents statically traced.
- [ ] 1.2 Document the rule and compile-only evidence; independent static review.
  Size small, difficulty low, depends on 1.1. Completion: stable spec and nearby
  docs agree, Python source compiles without importing or building, review recorded.

## Acceptance

- aiw-file-convert.md and its suffixed guides belong only to aiw-file-convert.
- aiw-file.md and unrelated aiw-file-* guides remain with aiw-file.
- Existing exact matches, cz and issue aliases remain supported.
- docs resource paths and main's explicit document selection remain unchanged.
- No installed files or release ZIPs are changed by this task.

## TODO

- Implement 1.1, then record compile-only evidence and request independent review.

## Verification

Static review plus one Python in-memory compile of scripts/package_definitions.py.
The root scripts/compile.py targets unrelated Go launchers; use the narrow Python
language compiler without importing packaging code or producing distributables.
No tests, runtime validation, final builds, install, publication or network.

## Sources

- openspec/specs/aiw-packages/spec.md
- scripts/package_definitions.py
- D:/green/aiw/.aiw/packages/aiw-file-convert/current.json (read only)
- dist/plugins/1.0.0/aiw-file-1.0.0.zip info (read only)
