# FD-111: Separate portable agent guidance from AIW-specific rules

**Status:** Complete
**Revision:** 2
**Priority:** Medium
**Review policy:** Self
**Evidence policy:** Markdown

## Problem

The agent guidance template is installed into different projects, but its root
instructions and shared resource-budget prompt include AIW-specific workflow
details. The fixed three-batch discovery limit can also stop useful inspection
too early in projects with more complex layouts.

## Solution

Keep the installed root instructions and shared prompt portable. Leave AIW FD
lifecycle and AIW-only test workflow rules in the AIW repository's own guidance
and canonical work-management contract. Set four targeted discovery batches as
the ordinary default, with focused continuation when a concrete unresolved
question still needs evidence.

## Scope

Update `src/agent-templates/AGENTS.md` and
`src/agent-templates/prompts/core/resource-budget.md`. Preserve AIW-specific
rules in the repository's own `AGENTS.md` and `src/skills/work-management.md`.
Do not change installer behavior, unrelated validation budgets, or historical
FD evidence.

## Work items

- [x] 1.1 Confirm the installed template and shared prompt are the canonical
  sources and identify AIW-only guidance that should stay repository-local.
- [x] 1.2 Remove AIW-specific workflow instructions from the installed generic
  guidance while preserving the AIW repository's local contract.
- [x] 1.3 Update discovery guidance to a four-batch default with focused
  continuation for concrete unresolved questions.
- [x] 1.4 Review the edited templates for consistent routing and wording.

## Acceptance

Installed generic guidance contains no AIW FD lifecycle or `fd-test` workflow
requirements. AIW-specific workflow remains in AIW-local instructions. Both
portable files recommend four targeted discovery batches by default and allow
additional focused inspection only while a concrete question remains.

## Verification

Static review confirmed that the installed root template no longer specifies an
AIW FD workflow and the shared resource budget no longer contains `fd-test` or
legacy Tester instructions. AIW-specific lifecycle and runtime authorization
remain in `AGENTS.md` and `src/skills/work-management.md`. Both installed
templates recommend four discovery batches and focused continuation for a
concrete uncertainty. No code compilation applies to this documentation-only
change. Tests, builds, formatters and linters were not run.

## TODO

No remaining implementation items. Archive this FD after final diff review.

## Sources

- Issue: none

Link relevant specs and evidence used for decisions.

**Completed:** 2026-11-08
