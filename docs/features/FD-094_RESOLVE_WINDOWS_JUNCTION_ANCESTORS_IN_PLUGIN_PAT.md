# FD-094: Resolve Windows junction ancestors in plugin paths

**Status:** In Progress
**Revision:** 7
**Priority:** High
**Evidence policy:** Dual

## Problem and evidence

User reports aiw pkg still fails to resolve the plugin directory after reinstalling.
Static byte inspection confirms installed C:/green/aiw/aiw.exe and the snapshot
binary have identical SHA-256 0085f32189290394695a36360602d31f2bd7b734ce655a6d13e3637473afd1ff
and contain the prior resolver symbol. This is not merely a stale-binary hypothesis.

The user explicitly approved exactly one offline Go path diagnostic. It exited 0:
plugin directory and intermediate directories Lstat as ordinary directories;
C:/green Lstat is ModeIrregular; EvalSymlinks of full descendants fails, while
EvalSymlinks(C:/green) returns the original C:/green. PowerShell confirms C:/green
is a junction to D:/green. Local Go source marks mount-point reparse tags irregular,
while filepath walking follows only ModeSymlink and rejects a non-directory ancestor.
The previous FD-091 no-link fast path therefore still enters the failing fallback.
The one diagnostic authorization is consumed; no rerun or installed mutation implied.

## Decision

Inspect components from the volume root and explicitly follow both ModeSymlink and
ModeIrregular using os.Readlink, which the Windows standard library implements for
symlink and mount-point tags. Queue raw link target components before the remaining
suffix, bounded to 255 traversals across the entire path. Resolve `..` only against
the already resolved physical parent; never Join/Clean an unresolved link target.
Absolute targets reset the volume root; reject volume-relative targets.
Unknown irregular types with unreadable link targets fail closed with path context.
Other unusual filesystem types retain existing physical resolution behavior.

Keep ordinary-path fast behavior, absolute paths for directory and entrypoint,
relative-entrypoint guards, physical containment via Rel, final Stat and directory
rejection. Do not silently use lexical paths after a link-resolution failure.
This is a scoped correction of the existing physical-containment contract with
no dependency, CLI, permission or persistence policy change; no separate design
review is required. Independent implementation review must inspect link cycles,
relative/absolute targets, old mixed mode bits and junction containment.

User-selected primary workspace remains in force. Do not create worktree/branch.
Do not change junctions, installed files, final binaries, ZIPs or package records.
No linked Issue/OpenSpec change. Existing builds must be recompiled explicitly.

## Work Items

- [x] 1.1 Follow link/junction ancestors with bounded traversal and error context.
  Size small, difficulty medium, dependencies none. Completion: static trace covers
  C:/green to D:/green, links in suffix/target, cycles, containment and unknown types.
- [ ] 1.2 Update docs/spec, offline Go compile-only and independent static review.
  Size small, difficulty low, depends on 1.1. Completion: truthful diagnosis ledger,
  root compile-only to NUL and independent managed pass.

## Acceptance

- C:/green junction resolves to its physical target for plugin directory and entrypoint.
- Links inside an entrypoint that escape the physical plugin root remain rejected.
- Missing or unsupported link targets fail clearly; cycles cannot recurse forever.
- Complete ordinary ancestor paths retain no case-enumeration dependency.
- No actual installation/runtime success is claimed without separate authorization.

## TODO

- Independent review R1 corrected with root-to-leaf raw-component processing;
  fresh focused review required before completing 1.2.
- Installed executable remains unchanged; do not claim runtime recovery.

## Verification

The approved one-shot diagnostic ran: GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local
`go run ./.ai/diagnostics/plugin-path.go`; only path inspections and Go temp/cache
writes, no install or network. Do not repeat it. After implementation run one
root scripts/compile.py with offline Go settings, then one static diff inspection.
No tests, final builds, installation, publication, network or fixed-runtime check.
Offline scripts/compile.py returned exit 0, compiling both Go entrypoints to NUL.
One implementation diff inspection traced target substitution, recursion bounds,
mixed mode bits and unchanged final entrypoint guards. See implementation report.
Independent review requests changes: docs/features/reviews/FD-094-review.md and JSON.
The leaf-first relative-link expansion can select the wrong physical target when
an ancestor is a junction. Work Item 1.2 remains incomplete; no runtime check rerun.
R1 repair now uses root-to-leaf traversal and raw target queues. A fresh offline
compile-only after this source change returned exit 0; one focused diff inspection
completed. Fresh independent review is required; the earlier finding remains history.

## Sources

- openspec/specs/cli-and-plugins/spec.md
- src/internal/plugin/manifest.go and archived FD-091 diagnosis
- Local Go src/os/types_windows.go and src/path/filepath/symlink.go
- Local Go src/os/file_windows.go readlink implementation
