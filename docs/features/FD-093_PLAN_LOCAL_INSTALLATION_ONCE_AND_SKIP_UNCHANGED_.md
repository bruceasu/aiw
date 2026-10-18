# FD-093: Plan local installation once and skip unchanged staging

**Status:** Complete
**Revision:** 7
**Priority:** Medium
**Evidence policy:** Dual

## Problem

build.py install all starts a new Operation for every root. Shared dependencies
and collection members are repeatedly planned, unzipped and verified. LocalPackages
also decodes an archive for its index and again for its file metadata. Manager
materializes a payload before discovering the same fingerprint is already installed.

## Decision and boundaries

Use one multi-root Operation for build.py installation with all selected roots.
Keep canonical root selection and manual/auto reasons. Deduplicate dependency
visits and require complete preflight before applying per-package transactions.
Retain the existing MAX_BYTES aggregate staging limit for this single operation;
large selections must fail clearly before mutation rather than exceed the budget.
This does not promise rollback of earlier successful package transactions.

Allow entry_for to use already validated prepared data when building local indexes.
Local fetch revalidates raw size/hash and safe paths against that original validated
index; identical archive bytes preserve identity/metadata validation without another
decompression. Retain full integrity validation when Operation prepares payloads.
Do not retain all decoded archives in LocalPackages or use a persistent cache.
Move Manager's same-fingerprint check under its existing lock before materializing
payloads, retaining platform/path/hook interpreter checks and graph preflight.
Keep changed-package staging and recovery behavior unchanged.

No CLI syntax, dependencies, persistent record schema, ownership policy or locking
model changes. Primary workspace explicitly selected in this conversation; no
branch/worktree/receipt. Independent design review checks multi-root compatibility,
aggregate limits, roots/reasons, byte identity and self-update pinning before coding.

## Work Items

- [x] 1.1 Multi-root graph selection in Operation and build.py.
  Size small, difficulty medium, dependencies none. Completion: all roots selected
  before expansion, deduplicated visits, manual reasons and final-state checks.
- [x] 1.2 Remove redundant local archive decoding and unchanged materialization.
  Size small, difficulty medium, depends on 1.1. Completion: raw fetch checks safe
  path/size/hash, prepared metadata reuse and locked unchanged fast path.
- [x] 1.3 Update docs/spec, compile-only and independent implementation review.
  Size small, difficulty low, depends on 1.2. Completion: truthful static evidence,
  one in-memory Python compile of changed files and independent managed pass.

## Acceptance

- install all plans each selected package once, including shared collection members.
- All requested roots keep manual semantics; dependency-only packages remain auto.
- Changed ZIP bytes or invalid paths fail before installation.
- Already-installed packages avoid temporary payload files and keep record semantics.
- Platform/ownership/version/cycle/size/conflict protections and self-update pinning remain.
- Earlier committed packages are not claimed to be collectively rolled back.

## TODO

- Independent design ready; implementation/docs and Python compile-only complete.
- Independent implementation review passed; host PM delivery/archive remains.

## Verification

Static review and one Python in-memory compile. No tests, benchmarks, final builds,
installation, publication or network. No measured speedup will be claimed.
Five changed Python files compiled in memory with exit 0; no packaging imports,
bytecode files or release artifacts. One implementation diff inspection completed.
See reports/FD-093-implementation.md and reviews/FD-093-design.md.
Independent static review passed for 5a81b9b..95e1b2c; see reviews/FD-093-review.md
and its JSON sidecar. No Reviewer runtime checks or benchmark were performed.

## Sources

- openspec/specs/aiw-packages/spec.md
- build.py install branch; scripts/local_packages.py
- src/plugins/aiw-pkg/{operations,manager,repository,archive}.py
- No linked Issue or OpenSpec change.
