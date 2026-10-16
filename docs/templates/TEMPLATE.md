# {{FD_ID}}: {{TITLE}}

**Status:** Planned  
**Revision:** 1  
**Priority:** Medium
**Evidence policy:** Dual

## Problem

%% NEEDS_INPUT: Describe the problem and affected people.

## Options and decision

%% NEEDS_INPUT: Compare options and record the chosen approach.

## Solution

%% NEEDS_INPUT: Describe boundaries, compatibility, data and permission effects, and the chosen approach.

## Scope

%% NEEDS_INPUT: State scope and exclusions.

## Work items

- [ ] 1.1 Define the first independently reviewable outcome and its acceptance evidence.

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

%% NEEDS_INPUT: List observable success, failure and recovery behavior.

## Verification

- Compile-only check and static review by default; record unrun tests explicitly.
- For explicitly requested test support (any wording/language), record suitable
  TDD scope, independent fd-test scope, command/revision authorization and finite
  budget here. Otherwise no testing stage is added. On exhausted unexpected-
  failure budget, stop affected test repair/execution, diagnose statically and
  continue independent work; unresolved acceptance defects still block delivery.

## Sources

{{ISSUE}}

Link relevant specs and evidence used for decisions.
