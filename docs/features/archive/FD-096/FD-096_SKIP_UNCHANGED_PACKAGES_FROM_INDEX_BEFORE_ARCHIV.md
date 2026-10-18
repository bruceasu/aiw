# FD-096: Skip unchanged packages from index before archive acquisition

**Status:** Complete
**Revision:** 7
**Priority:** High
**Evidence policy:** Dual

## Problem and authorized policy

User prioritizes fast installation/update and explicitly accepts local file edits.
Current Operation downloads/decodes every selected archive before comparing its
fingerprint, repeatedly decodes repository ZIPs and scans installed file hashes.
Even reading installation records inspects every stored target path on disk.

## Decision

Plan named repository install/update from one validated index and installation
metadata. A matching installed name/version/fingerprint is retained without fetch,
unzip, local file stat/hash or temporary payload. Continue exact dependency/member
resolution, final-state validation, collection update trim and manual/auto reasons.
A skipped package is not repaired if its local files were changed or removed.

Changed packages use adapter prepare(entry, folder), returning a path plus the
single fully validated prepared payload, pinned before self-update writes. Keep
fetch's existing path-returning API as a wrapper for other callers. Share index
entry generation from prepared info so raw ZIP SHA-256 is computed once on newly
acquired bytes; verify internal data/file integrity once during decode.
LocalPackages prepare still returns original ZIP paths. build.py passes exact
root name/version requests from its in-memory index, avoiding repeat root fetch.
Its initial local catalog still reads ZIPs because old build manifests store only
paths, not a package index. Its existing before-Manager destination/input-ZIP
preflight remains; the zero per-target inspection promise applies to repository
named operations, not build-local catalog/preflight. Explicit ZIP install also
still decodes the input once; the CLI passes its validated prepared tuple through
to Operation instead of discarding it and decoding again.
No new on-disk cache, manifest/record schema, dependency or CLI syntax.

Metadata-only record reads validate record structure, relative target spelling and
reserved-state exclusion but do not inspect installed target files. Recheck the
metadata snapshot under the existing lock; keep incomplete-transaction blocking.
Retained metadata/reason/member updates happen under lock and only save if changed.
No-op operations, including unchanged dry-runs, do not fetch or write payloads.
Changed-package dry-runs retain full preparation/preflight but never mutate installed
files/records or execute hooks. No dry-run behavior is weakened into metadata-only preview.

For install/update, stop enforcing unchanged installed content hashes. Changed
ordinary managed files may be overwritten (or obsolete files removed) after the
existing before-snapshot; existing configuration files stay preserved. Keep safe
paths, link/type/ownership checks before writes, archive integrity, exact-version,
cycle/platform/size guards, hooks, recovery and per-package transactions.
Removal/purge and project/user Skill management keep their current protections.

## Compatibility and workspace

This is the user's requested validation policy change. It does not disable archive
integrity or permit unsafe/foreign-owned targets. Index/installed fingerprints are
the authority for no-op selection; skipped packages do not prove local health.
The 512 MiB aggregate limit applies to archives actually acquired, not skipped
metadata. Existing explicit ZIP semantics stay intact. No network or install runs.
Current primary workspace remains selected; no branch/worktree/receipt.
Independent design feedback must check graph/reason/collection behavior, trusted
metadata versus changed payload, API compatibility, self-update pinning and limits.
No linked Issue or OpenSpec change.

## Work Items

- [x] 1.1 Add index-first retained nodes to install/update planning.
  Size small, difficulty medium, no dependencies. Completion: exact roots and graph,
  skipped nodes avoid acquisition/file preflight, locked metadata/reason updates.
- [x] 1.2 Decode changed archives once through shared prepared adapters.
  Size small, difficulty medium, depends on 1.1. Completion: raw integrity and index
  metadata binding retained, fetch API preserved, local original paths stay pinned.
- [x] 1.3 Remove local-content rejection from install/update only.
  Size small, difficulty low, depends on 1.1. Completion: metadata-only record reads,
  no hash scan for install/update, changed-file snapshots/config preservation intact.
- [x] 1.4 Document policy and prepare compile/static evidence.
  Size small, difficulty low, depends on 1.2/1.3. Completion: docs/spec and one
  in-memory Python compile of changed files. Independent review is a completion gate.

## Acceptance

- Repository named install/update with all fingerprints unchanged performs zero archive fetch,
  zero unzip, zero installed-content hashing and no per-target filesystem inspection.
- Same-name changed version/fingerprint still acquires/verifies actual ZIP bytes.
- Missing/edited local files do not block install/update or trigger repair of skips.
- Final graph/version/cycle rules and collection/full-versus-retained member semantics hold.
- Existing manual roots stay manual; dependencies do not unexpectedly become manual.
- New payload writes remain path/type/ownership safe and recoverable; configs preserved.
- Explicit local ZIP install validates input; other fetch callers retain full validation.
- Removal/purge/Skill rules and record schema remain unchanged.

## TODO

- Independent design ready; implementation/docs and six-file Python compile-only complete.
- Independent implementation review passed; installed plugin unchanged. PM archive pending.

## Verification

Static review plus one in-memory Python compile of changed files. No tests, runtime
install/update, benchmarks, network, downloads, final builds or installation.
Report unmeasured performance and installed-plugin update requirement honestly.
Six changed Python files compiled in memory with exit 0. One static command batch
reviewed final source diff and the unchanged removal hash-check call; no runtime or
benchmark execution. See reports/FD-096-implementation.md.
Independent static implementation review passed at `1cddc45` against `c7c5ebb`;
see reviews/FD-096-review.md and its JSON sidecar. Reviewer ran no runtime checks.

## Sources

- src/plugins/aiw-pkg/{operations,manager,repository,archive}.py
- scripts/local_packages.py and build.py install branch
- openspec/specs/aiw-packages/spec.md; docs/usage/aiw-pkg.md

**Completed:** 2026-10-18
