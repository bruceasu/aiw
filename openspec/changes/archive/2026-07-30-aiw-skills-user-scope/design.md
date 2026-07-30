## Context

The installer currently derives its destination only from the current working
directory. `.agents/skills` is the shared convention needed at both project
and user scope.

## Design

Use one resolver shared by all Skill catalog commands:

| Scope | Destination root | Default |
|---|---|---|
| `project` | `<current project>/.agents/skills` | Yes |
| `user` | `<current user home>/.agents/skills` | No |

The resolver returns an absolute normalized path. User home resolution uses
the runtime's standard home API, not shell-specific `$HOME` or `%USERPROFILE%`
syntax. Each destination root owns its own `.aiw-skills.json` manifest.

The selected scope is accepted consistently by install, discover, adopt, and
sync. Human-readable, dry-run, and JSON results expose the selected scope and
resolved destination. All existing safe-publish protections apply unchanged.

