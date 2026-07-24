## Why

Codex can discover reusable Skills, but aiw-flow's interactive loop does not expose a convenient way to inspect or explicitly invoke them. Users currently need to know the exact `$skill-name` syntax and cannot see which project or user Skills are available from inside the loop.

## What Changes

- Add `/skills` to the interactive loop to list discoverable project and user Skills without consuming a Codex turn.
- Add `/skill <name> <message>` to validate a Skill name and execute the message through the normal turn path using Codex's native `$<name>` invocation syntax.
- Discover Skills from project and user `.agents/skills` locations while retaining compatibility with aiw's existing `.codex/skills` locations.
- Report malformed Skills, unsupported command syntax, and duplicate names clearly without calling Codex.
- Keep direct `$skill-name` messages and all existing loop commands unchanged.
- Do not add configurable Skill paths, copy or snapshot Skills into Session state, or reimplement Codex's Skill instruction loading.

## Capabilities

### New Capabilities

- `session-skill-invocation`: Discover, list, validate, and explicitly invoke Codex Skills from an aiw-flow interactive Session.

### Modified Capabilities

None.

## Impact

- Affects aiw-flow interactive input parsing and CLI coordination, with a small Skill discovery component.
- Reads Skill metadata from the Session workspace, its repository ancestry, and user-level Codex locations.
- Routes Skill turns through the existing prompt, output, event, status, timeout, and Thread persistence path.
- Mirrors runtime and documentation changes from `program/aiw-flow` into `plugins/aiw-flow`.
- Adds no dependency, Session schema change, configurable path, daemon, or persistent subprocess.
