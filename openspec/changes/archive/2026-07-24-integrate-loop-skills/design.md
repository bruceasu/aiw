## Context

aiw-flow keeps one logical Codex Thread while starting one `codex exec` process for each turn. The interactive loop currently recognizes a fixed set of local slash commands and sends every ordinary message through `_execute_turn`.

Codex already owns Skill activation and progressive loading. aiw also has an installer that places Skills in project or user `.codex/skills` directories, while current Codex conventions additionally discover `.agents/skills` from repository ancestry and the user home. The integration should make these Skills visible and convenient inside aiw-flow without introducing a second Skill runtime.

## Goals / Non-Goals

**Goals:**

- List valid project and user Skills from inside an interactive Session without consuming a turn.
- Invoke a named Skill through Codex's native `$skill-name` prompt syntax.
- Support current `.agents/skills` locations and aiw's existing `.codex/skills` locations without user configuration.
- Preserve the existing turn execution, Thread, persistence, timeout, and failure behavior.
- Give deterministic errors for malformed Skills, duplicate names, and invalid command syntax.

**Non-Goals:**

- Add configurable Skill search paths.
- Copy, snapshot, install, update, enable, disable, or delete Skills.
- Persist an active Skill set in Session status.
- Reimplement progressive disclosure, linked-resource loading, or Skill script execution.
- Change direct `$skill-name` messages or Codex's implicit Skill matching.
- Change the existing aiw Skill installer in this change.

## Decisions

### Discover from fixed project and user candidates

For a Session workspace, discovery will inspect:

1. `.agents/skills` under each directory from the workspace toward its Git repository root, nearest directory first.
2. `<project-root>/.codex/skills`, where the project root is the Git root or the workspace for a non-Git directory.
3. `~/.agents/skills`.
4. `<codex-home>/skills`, where the saved Session Codex home is used when present and `~/.codex` is the default.

Only immediate child directories containing `SKILL.md` are Skill candidates. A symlinked child directory is accepted when it resolves to a directory containing `SKILL.md`, matching Codex's supported local-authoring behavior.

Fixed candidates were chosen over `skill_paths` because they match established Codex and aiw locations, keep configuration small, and avoid making arbitrary filesystem paths part of the loop interface.

### Parse only the metadata required for discovery

A small discovery component will read `name` and `description` from each candidate's `SKILL.md` frontmatter. It will not interpret the Skill body, linked resources, scripts, or optional UI metadata.

`/skills` will display valid Skills grouped by scope and will report malformed candidates as warnings without hiding other valid Skills. Discovery will be repeated for each local Skill command so changes made between loop inputs are visible.

Reusing the full installer implementation was rejected because the loop needs read-only discovery, not archive extraction, copying, replacement, or backup behavior. Adding a YAML dependency was also rejected; the supported frontmatter subset can follow the repository's existing lightweight validation conventions.

### Treat duplicate names as ambiguous

If more than one discovered Skill declares the same `name`, `/skills` will show every source and mark the name ambiguous. `/skill <name> <message>` will refuse to execute that name and print the conflicting paths.

Silent precedence was rejected because aiw-flow cannot guarantee that an independently evolving Codex resolver would choose the same duplicate. Explicit failure is safer than invoking an unintended workflow.

### Translate Skill commands into ordinary turns

The interactive parser will recognize:

- `/skills`
- `/skill <name> <message>`

`/skills` is local and never calls Codex. A valid `/skill` invocation will be transformed into the ordinary prompt:

```text
$<name> <message>
```

The transformed prompt then uses `_execute_turn` exactly like a normal loop message. This preserves saved prompts, outputs, events, Session state, timeout behavior, and Thread resumption. Direct user messages beginning with `$skill-name` remain untouched, and the existing `//text` escape continues to send slash-prefixed ordinary text.

Passing the full `SKILL.md` in the composed prompt was rejected because it would duplicate Codex's loading rules, increase context use, and make linked resources less reliable.

### Keep invocation stateless in aiw-flow

The command invokes a Skill for the submitted turn but does not add an `active_skills` field to Session status. Codex Thread history already preserves the conversation, while aiw-flow cannot reliably remove previously seen instructions from that history.

A persistent `/skill use` or `/skill unload` model was rejected for the first version because its removal semantics would be misleading and it would introduce state not owned by Codex.

## Risks / Trade-offs

- [Codex versions may differ in legacy `.codex/skills` discovery] → Keep aiw-flow's role limited to validation and native `$name` invocation, document the supported aiw locations, and cover the installed Codex behavior with CLI smoke tests.
- [A Skill can change after `/skills` output] → Re-run discovery and validation immediately before every `/skill` turn.
- [Frontmatter parsing may reject advanced YAML forms] → Define and test the supported metadata forms, report the exact file as malformed, and do not add a dependency without approval.
- [Symlinks may point outside the candidate root] → Treat candidate roots as trusted local configuration, resolve the link for validation, and never execute scripts during discovery.
- [A name can be duplicated across scopes] → Report all paths and refuse shorthand invocation instead of guessing precedence.
- [A Skill may be listed by aiw-flow but unavailable to a different Codex version] → Preserve Codex stderr and normal failed-turn behavior; do not fall back to injecting the Skill body.

## Migration Plan

1. Add read-only Skill discovery and focused tests.
2. Extend loop input parsing and coordination behind the new slash commands.
3. Update source documentation and mirror runtime changes to the packaged plugin.
4. Verify against project and user Skills in both `.agents` and `.codex` locations.

Rollback removes the optional local commands and discovery component. Existing Sessions, Threads, prompts, and installed Skills require no migration.

## Open Questions

%% Confirmed during implementation: accept unquoted, single-quoted, and double-quoted single-line `name` and `description` values; reject YAML block descriptions instead of partially interpreting them.

%% Future change: consider qualified duplicate invocation such as `project:name` only if real usage shows that rejecting collisions is too restrictive.
