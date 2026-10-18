# FD-085: Snapshot versions and latest build lookup

**Status:** In Progress
**Revision:** 2
**Priority:** Medium
**Evidence policy:** Dual

## Problem and decision

compile/build currently default to dev. The user requests date-time snapshot
versions and a way to query the latest build version.

Use --version first, explicit nonempty AIW_VERSION second, otherwise generate one
UTC snapshot-YYYYMMDD-HHMMSSffffffZ per invocation. Remove the active dev value
from .env.example so a copied example enables snapshots by default. Print the
chosen version before compile/build. Publish commands keep explicit versions.
Independent package scripts retain their existing required-version/env contract.

Add `python build.py latest [target]`: print only the latest successfully built
version, either globally or for a component/group/canonical alias. Read existing
dist/.manifests files; no compilation, packaging, publication or installation.
Missing manifests/packages or malformed manifests produce explicit errors.
Move new manifest writes after successful dist/repository staging. Add an optional
built_at_ns completion timestamp to manifests (same value for one build invocation).
Historical manifests remain readable using file mtime_ns. Compile-only operations
do not write manifests and do not change the latest build version. Chronological
build order applies, not semantic version ordering.

Alternative: a separate latest-version database. Rejected as redundant state.
Alternative: scan package ZIP names. Rejected because partial or older artifacts
do not represent the last build. Keep existing manifest version/packages fields.
No material compatibility/ownership decision needs another design review: this
adds a read-only CLI and optional metadata, preserving all explicit version syntax.

## Work Items

- [ ] 1.1 Implement snapshot fallback, version output and latest lookup.
  Size small, difficulty medium, no dependencies. Completion: aliases, historical
  manifests, failure paths and successful-build ordering covered statically.
- [ ] 1.2 Update example, usage and stable spec; compile and record evidence.
  Size small, difficulty low, depends on 1.1. Completion: compile-only success,
  consistent docs and final diff; independent Reviewer handoff.

## Acceptance

- compile/build without CLI/env version share one UTC snapshot per invocation.
- Explicit versions win; publish syntax remains unchanged.
- latest and latest aiw-main print the last build version without build side effects.
- Existing manifests work; invalid/missing records fail clearly.
- Failed compilation/packaging/staging does not replace successful manifests.

## TODO

- Implement in feature/FD-085 at .wt/FD-085, then independent static review.
- Deliver and archive when managed review and cleanup gates pass.

## Verification

Planning: inspected build.py compile/build dispatch, manifest ordering, package
scripts' env version behavior, .env.example, stable package spec and usage docs.
No tests, package builds, publication, installation or network commands authorized.

## Risks

%% Historical manifest mtime can change when copied; completion timestamps apply to new builds.
%% Runtime command behavior remains unverified under the repository budget.
