# FD-085: Snapshot versions and latest build lookup

**Status:** Complete
**Revision:** 5
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

- [x] 1.1 Implement snapshot fallback, version output and latest lookup.
  Size small, difficulty medium, no dependencies. Completion: aliases, historical
  manifests, failure paths and successful-build ordering covered statically.
- [x] 1.2 Update example, usage and stable spec; compile and record evidence.
  Size small, difficulty low, depends on 1.1. Completion: compile-only success,
  consistent docs and final diff; independent Reviewer handoff.

## Acceptance

- compile/build without CLI/env version share one UTC snapshot per invocation.
- Explicit versions win; publish syntax remains unchanged.
- latest and latest aiw-main print the last build version without build side effects.
- Existing manifests work; invalid/missing records fail clearly.
- Failed compilation/packaging/staging does not replace successful manifests.

## TODO

- Implementation and compile-only complete; independent static review passed.
- Deliver and archive when managed review and cleanup gates pass.

## Verification

Planning: inspected build.py compile/build dispatch, manifest ordering, package
scripts' env version behavior, .env.example, stable package spec and usage docs.
No tests, package builds, publication, installation or network commands authorized.

Implementation evidence: reports/FD-085-implementation.md and same-name JSON.
Python compile-only passed for build.py with compile(read_bytes(), path, 'exec'),
without imports or retained artifacts. The Go-specific scripts/compile.py is
unrelated to this Python-only change. One final scoped git diff against 460fa7e
reviewed version precedence, optional latest target/aliases, timestamp ordering,
historical fallback, missing-package errors and successful-stage manifest writes.
Tests, runtime CLI execution, builds, publishing and installation were not run.

Independent Reviewer: reviews/FD-085-review.md and same-name JSON; reviewed
460fa7e..b144e33 under source event FD-085-000004-review-requested in independent
session snapshot-review-20261018-root-snapshot-review. Static review passed with
no blocking findings. Reviewer did not run tests, compile or runtime validation.

## Risks

%% Historical manifest mtime can change when copied; completion timestamps apply to new builds.
%% Runtime command behavior remains unverified under the repository budget.
