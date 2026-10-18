# FD-091: Resolve ordinary plugin paths without case enumeration

**Status:** Complete
**Revision:** 5
**Priority:** Medium
**Evidence policy:** Dual

## Problem and diagnosis

The installed aiw pkg reports resolve plugin directory: The system cannot find the
path specified. The user explicitly approved one offline help command, which
reproduced exit 1: & 'C:\green\aiw\aiw.exe' pkg --help from the primary checkout.
No retry occurred. The installed plugin directory, manifest and entrypoint exist;
PowerShell reports ordinary files/directories with no reparse point attributes.

Source calls filepath.EvalSymlinks for both the directory and explicit entrypoint.
Local Go 1.25.1 source shows EvalSymlinks first walks Lstat components, then performs
Windows case normalization using FindFirstFile, whose errors are returned without
a PathError. The observed bare Windows error supports a normalization/enumeration
failure, rather than proving files are missing. The exact failing segment was not
measured; do not claim that detail or a successful fixed-runtime reproduction.

## Decision

Add an internal filesystem path resolver. Make the path absolute; inspect every
non-volume-root component with os.Lstat. When all component types are exactly
regular files or ordinary directories, use the absolute path directly. Directory
case enumeration is unnecessary for containment because root and entrypoint share
the same source prefix, and filepath.Rel handles native path comparisons.

Any symlink, junction-like irregular reparse type or other nonordinary type MUST
retain filepath.EvalSymlinks, propagating failures. Never ignore a resolution error
and fall back to a lexical path. Both directory and entrypoint use this helper;
the existing relative-entrypoint guards, physical containment check, os.Stat,
directory rejection, manifest wrapping, legacy search and startup stay unchanged.
Use exact ModeType comparisons, not IsDir alone, to handle older Windows modes
that can combine directory and symlink bits. No new dependencies or public APIs.

This is a scoped internal correction preserving the existing boundary policy;
no new permission/compatibility decision needs a separate design review. An
independent implementation Reviewer must examine the no-link proof and fallback.
Do not change installed manifests, rebuild final binaries or install automatically.

## Work Items

- [x] 1.1 Add the ordinary-path resolver and preserve link/error containment behavior.
  Size small, difficulty medium, no dependencies. Completion: all ancestor checks,
  exact modes, fallback and both call sites statically traced.
- [x] 1.2 Document diagnosis/semantics, compile-only and independent review handoff.
  Size small, difficulty low, depends on 1.1. Completion: current offline Go compile,
  truthful runtime ledger and independent static review evidence.

## Acceptance

- Existing ordinary installed paths do not depend on Windows FindFirstFile normalization.
- Relative or absolute caller directory roots yield consistent absolute paths.
- Links/nonordinary ancestors retain physical resolution; escaping entrypoints still fail.
- Missing/unreadable paths and directory entrypoints fail clearly, with manifest context.
- Installed files and CLI/runtime policy remain unchanged by this implementation task.

## TODO

- Implementation and offline Go compile-only complete; independent static review passed.
- Local delivery and archive after current managed pass.

## Verification and authorization ledger

Consumed runtime evidence: one user-approved installed pkg --help command exited 1.
No additional runtime attempt or unchanged installed-executable retry authorized.
Default-authorized compile-only will use scripts/compile.py with GOPROXY/GOSUMDB off,
GOTOOLCHAIN local, GOWORK off, CGO disabled; outputs go to NUL, no final binary.
No tests, builds, installs, publication, network or permission probes authorized.
Read relevant manifest discovery source, stable CLI plugin contract, local Go
Windows symlink/mode implementation and installed filesystem metadata.
Implementation report reports/FD-091-implementation.md and JSON. The offline
scripts/compile.py command exited 0 for cmd/aiw and cmd/aiw-req, output NUL,
with worktree-local cache and no retained distributable. One scoped git diff
against b4aba07 traced no-link proof, exact ModeType checks, original physical
fallback, containment/error flow and documentation. No runtime success is claimed
for the unchanged installed launcher; approved help failure remains the only run.
Independent Reviewer: reviews/FD-091-review.md and JSON, session
fd091-independent-reviewer-plugin-path-review, source FD-091-000004-review-requested.
Static review of implementation 286faae against b4aba07 found no blocking issue.
No additional runtime, compile, tests, build or installation was performed by Reviewer.

## Risks

%% The correction targets the supported Windows normalization failure mechanism;
%% actual installed binary will remain old until the user rebuilds and installs.
%% Symlink/irregular path failures remain fail-closed; no promise to fix every reparse setup.
%% Single-writer filesystem assumptions and path replacement races remain unchanged.
