# AIW Skills Guide

This directory contains reusable Skills for AIW/Codex. A Skill is a workflow description for an Agent, not a standalone CLI command. The CLI discovers, installs, and manages Skills; the Agent decides when to invoke a Skill from its frontmatter and instructions.

## Actual Skills in This Directory

### Routing and Exploration

- `ask-asu`: identify the current AIW/OpenSpec stage and recommend the next step.
- `wayfinder`: turn uncertain work into a path of decisions, research, and implementation.
- `triage`: organize reports and work items around evidence and next actions.
- `grilling`: pressure-test a plan or decision one question at a time.
- `grill-me`: keep questioning a plan or idea until its key assumptions are exposed.
- `grill-with-docs`: record ADRs, terminology, and design notes during deep discovery.
- `prototype`: build a throwaway prototype to test a state model, logic, or UI direction.
- `research`: investigate high-trust primary sources and save findings as Markdown.
- `diagnosing-bugs`: diagnose failures, regressions, exceptions, and performance problems from evidence.

### Requirements, Domain, and Design

- `requirement-management`: manage requirement discussion, decisions, approval, and Task handoff.
- `finance-requirement-intake`: clarify finance, operations, reporting, and analytics requests.
- `finance-value-assessment`: assess value, cost, risk, and scope.
- `finance-metric-brief`: define metric formulas, sources, time dimensions, precision, refresh, and ownership.
- `finance-engineering-options`: review boundaries, permissions, auditability, failure handling, observability, and testing.
- `finance-requirement-synthesis`: consolidate finance discovery into a Requirement Plan.
- `domain-modeling`: establish shared terminology, concept relationships, and boundary decisions.
- `codebase-design`: improve module interfaces, ownership, encapsulation, and test seams.
- `improve-codebase-architecture`: find architecture improvement opportunities and deep-dive into one.
- `fd-workflow`: deepen Feature Design when material design choices remain.

### OpenSpec and Implementation

- `to-spec`: turn an agreed problem and solution into an OpenSpec change.
- `to-tickets`: split a spec or plan into ordered checklist items with acceptance criteria.
- `implement`: implement one selected AIW Work Item; it does not run tests or publish automatically.
- `tdd`: use a red-green-refactor loop around one agreed test seam.
- `code-review`: review changes on separate Standards and Spec axes.
- `finance-release-gate`: check migration, permissions, audit, rollback, and operational readiness for a finance change.

### Sessions, Collaboration, and Maintenance

- `handoff`: create or consume handoff information for a later Agent.
- `resume-ext`: find and resume an existing Codex session.
- `resolving-merge-conflicts`: handle an in-progress Git merge or rebase conflict.
- `setup-project`: prepare AIW/OpenSpec project documents, templates, and conventions.
- `edit-article`: edit technical documentation or other articles.
- `teach`: explain a technical topic or code step by step.
- `writing-great-skills`: reference guidance for authoring and improving Skills.
- `github-issues-management`: organize and manage GitHub Issues.
- `publish-github-issue`: publish a result as a GitHub Issue only when explicitly requested.

## Invoking a Skill

Invoke a Skill by its Skill name, for example:

```text
$ask-asu I have a requirement and some code. What should I do next?
$diagnosing-bugs This request sometimes times out. Diagnose it from evidence.
$to-spec Turn the agreed solution into an OpenSpec change.
$implement Implement the selected checklist item for the current Task.
```

The exact syntax depends on the host Agent. `$name` means “invoke the Skill”; it is not a PowerShell command and not an `aiw name` CLI command.

Some Skills use `disable-model-invocation: true`. Those Skills require explicit invocation by the user or by a clearly defined workflow; do not assume that every Skill is automatically selected.

## Managing Skills with the CLI

These commands manage the canonical Skills in this repository and copy them into the project's `.agents/skills/` directory:

```powershell
aiw skills list
aiw skills list --json
aiw skills install implement --dry-run
aiw skills install implement
aiw skills install --all --dry-run
aiw skills install --all
aiw skills discover
aiw skills discover --json
aiw skills adopt
aiw skills sync implement
```

The default scope is the current project. Install into the user-wide catalog with:

```powershell
aiw skills install tdd --scope user
aiw skills discover --scope user
```

You can also install a directory, ZIP file, or bundle containing `SKILL.md`:

```powershell
aiw skills install .\skills\my-skill
aiw skills install .\skill-bundle.zip --dry-run
```

Installation rules:

- Every Skill directory must contain valid YAML frontmatter with `name` and `description`.
- The directory name must match the frontmatter `name`.
- An unmanaged same-name Skill is not replaced by default.
- `adopt` records an existing directory; it does not make that directory a canonical source.
- `sync` refreshes a destination that is already recorded as managed.
- `--dry-run` shows the plan without writing the destination.

The current implementation rejects Skill symlinks and non-regular files. Inspect the source before installing an untrusted ZIP; ZIP path validation remains a high-priority security improvement.

## Recommended Workflows

### When the Requirement Is Unclear

```text
requirement-management
  -> grilling / grill-with-docs
  -> domain-modeling
  -> finance-value-assessment (when the request is finance or operations)
  -> finance-metric-brief (when metrics are involved)
  -> finance-engineering-options
```

### When the Requirement Is Clear

```text
to-spec -> to-tickets -> implement
                       \-> tdd (when the user requests test-first work)
```

After implementation, choose `code-review`, `finance-release-gate`, or a publishing workflow as authorized. `implement` does not automatically invoke `tdd` or `code-review`.

### When Something Is Broken

```text
triage -> diagnosing-bugs -> implement
```

Use `codebase-design` before implementation when the module boundary is unclear.

## Shared Skill Boundaries

All reviewed Skills should follow:

- `skills/reviewed-skill-contract.md` and `skills/work-management.md`.
- Record missing facts as `%% NEEDS_INPUT: ...`; use `BLOCKED` or `INCOMPLETE` when work cannot proceed.
- Report produced artifacts, Evidence, Gates, and the recommended next action.
- Do not fabricate Task status, Attempt results, leases, or publication results.
- Do not run tests, builds, formatters, linters, network operations, or publication without authorization.
- Distinguish completed static work from runtime checks that were not run.

A new Skill should state its trigger and non-trigger conditions, inputs, outputs, completion criteria, unresolved-input behavior, and verification boundary.

## Related Documentation

- [Reviewed Skill Contract](reviewed-skill-contract.md)
- [Work Management](work-management.md)
- [AIW Work Management](../docs/agents/work-management.md)
- [AIW Guide](../README.md)
