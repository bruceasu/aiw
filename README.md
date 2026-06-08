# aiw

A lightweight OpenSpec-inspired task workflow CLI implemented in Go.

## Feature Overview

* Initialize the OpenSpec-lite directory structure and default instruction files
* Automatically create or append `.wt/` entries to `.gitignore`
* Generate or merge AI prompt files from `docs/agent-templates/`
* Create, view, and update tasks
* Create dedicated Git worktrees for tasks
* Output task-specific context prompts
* Create long-lived specification documents
* Archive completed tasks
* Generate the task registry file `openspec/registry.json`

## Directory Structure

After running `aiw init`, the following structure will be created (if missing):

```text
repo/
├── openspec/
│   ├── changes/
│   ├── specs/
│   ├── archive/
│   └── registry.json
├── .wt/
├── AGENTS.md
└── .github/
    └── copilot-instructions.md
```

Notes:

* `AGENTS.md` and `.github/copilot-instructions.md` are created only if they do not already exist.
* `registry.json` is generated from all `openspec/changes/*/task.toml` files.

## Build and Installation

```bash
# TCC wrapper
aiw tcc hello.c -o hello.exe
aiw tcc dll hello.c -o hello.dll
aiw tcc x86_64 hello.c -o hello.exe
aiw tcc run hello.c

go build -o aiw .
```

Windows executable:

```powershell
.\aiw.exe init
```

## Commands

```text
aiw init [--prompts] [--merge] [--force] [--template <name>]
aiw new <task-id>
aiw list
aiw show <task-id>
aiw status <task-id> <status>
aiw done <task-id>
aiw archive <task-id> [--push] [--cleanup-wt] [--delete-branch] [--finalize]

aiw wt add <task-id> [base-branch]
aiw wt rm <task-id> [--delete-branch] [--force]
aiw wt list [--porcelain]
aiw wt prune [--dry-run]
aiw wt lock <task-id> [reason]
aiw wt unlock <task-id>
aiw wt repair
aiw wt ignore

aiw context <task-id>
aiw decision <task-id>
aiw spec <spec-id>
aiw registry

aiw prompts list
aiw prompts [template] [--merge] [--force]

aiw tcc [args...]       # TCC wrapper with automatic include/lib defaults
aiw git <subcommand>    # run: aiw git help
```

# Plugin System

`aiw` supports extending subcommands through external executable plugins.

When an unknown subcommand is invoked, `aiw` searches for an executable named `aiw-<plugin-name>` and executes it.

## Plugin Search Paths

Search order:

1. `plugins/` directory next to the `aiw` executable
2. `$HOME/.config/aiw/plugins`
3. System `PATH`

## Naming Convention

Plugin filenames must follow:

```text
aiw-<plugin-name>
```

Supported extensions:

```text
.exe
.py
.sh
.bat
.cmd
.ps1
.js
.jar
(no extension)
```

If subdirectories exist under `plugins/`, `aiw` recursively searches one level deeper.

## Execution Priority

When multiple matching plugins exist:

1. `.bat` / `.cmd` / `.sh`
2. `.py`
3. Extensionless scripts (shebang)
4. Native binaries (`.exe` / ELF)

## Interpreters and Shebang

* Extensionless scripts with a `#!` shebang are executed using the specified interpreter.
* For `.js` files, `bun` is preferred when available; otherwise `node` is used.

## Environment Variables

The following variables are injected into plugin processes:

| Variable                | Description                                |
| ----------------------- | ------------------------------------------ |
| `AIW_PLUGIN_NAME`       | Plugin name without the `aiw-` prefix      |
| `AIW_PLUGIN_PATH`       | Absolute path to the executed plugin       |
| `AIW_CMDLINE`           | Original command line after the subcommand |
| `AIW_HOME` / `AIW_ROOT` | AIW configuration or installation root     |

## Example

Place `plugins/aiw-hello.sh` in the repository's `plugins/` directory:

```bash
aiw hello arg1 arg2
```

The plugin should process arguments via standard `argv` and write output to stdout/stderr. Its exit code becomes the exit code of `aiw`.

## Security Notice

Plugins execute arbitrary external code and may pose security risks. Only install trusted plugins. Consider signature verification or allowlists in production environments.

# Command Behavior Details

## 1. `aiw init`

* Creates directories such as `openspec/` and `.wt/`
* Writes default template files only when missing
* Creates or appends `.wt/` to `.gitignore`
* Generates or refreshes `openspec/registry.json`
* Does not automatically merge templates from `docs/agent-templates/`

Options:

* `--prompts`
  Run prompt synchronization immediately after initialization.

* `--merge`
  Valid only with `--prompts`. Merge content into existing prompt files.

* `--force`
  Valid only with `--prompts`. Overwrite existing prompt files.

* `--template <name>`
  Valid only with `--prompts`. Explicitly specify the template directory (`go`, `java`, or `python`).

## 2. `aiw new <task-id>`

Creates:

```text
openspec/changes/<task-id>/
├── task.toml
├── task.md
└── notes.md
```

Default metadata:

```toml
type = "task"
status = "TODO"
branch = "feature/<task-id>"
worktree = ".wt/<task-id>"
```

## 3. `aiw decision <task-id>`

Creates `design.md` for the task if it does not already exist.

## 4. `aiw spec <spec-id>`

Creates:

```text
openspec/specs/<spec-id>/
├── spec.toml
└── spec.md
```

## 5. `aiw status <task-id> <status>`

Updates:

* `task.toml`
* `status` (converted to uppercase)
* `updated`

## 6. `aiw done <task-id>`

Equivalent to:

```bash
aiw status <task-id> DONE
```

Does not archive the task automatically.

## 7. `aiw archive <task-id>`

Moves:

```text
openspec/changes/<task-id>
```

to:

```text
openspec/archive/<YYYY-MM-DD>-<task-id>
```

Options:

* `--push`
  Execute:

  ```bash
  git push -u origin feature/<task-id>
  ```

* `--cleanup-wt`
  Remove the task worktree.

* `--delete-branch`
  Delete the local feature branch.

* `--finalize`
  Equivalent to:

  ```text
  --push --cleanup-wt --delete-branch
  ```

## 8. `aiw wt <subcommand>`

Worktree management commands (`aiw wt help` for details).

| Subcommand             | Description                                                                 |
| ---------------------- | --------------------------------------------------------------------------- |
| `add <task-id> [base]` | Create a worktree on branch `feature/<task-id>`, default base `origin/main` |
| `rm <task-id>`         | Remove a worktree                                                           |
| `list`                 | List all worktrees                                                          |
| `prune`                | Remove stale worktree metadata                                              |
| `lock`                 | Protect a worktree from accidental removal                                  |
| `unlock`               | Unlock a worktree                                                           |
| `repair`               | Repair worktree links after path relocation                                 |
| `ignore`               | Add `.wt/` to `.gitignore`                                                  |

`add` executes:

```bash
git fetch origin &&
git worktree add .wt/<task-id> -b feature/<task-id> <base>
```

and updates:

```text
branch
worktree
updated
```

in `task.toml`.

## 9. `aiw context <task-id>`

Prints recommended files to review and execution constraints for the task.

## 10. `aiw registry`

Regenerates:

```text
openspec/registry.json
```

## 11. `aiw prompts [template] [--merge] [--force]`

Features:

* `aiw prompts list` lists available templates under `docs/agent-templates/`
* Generates or merges repository-level AI prompt files

Auto-detected templates:

| Template | Detection                                        |
| -------- | ------------------------------------------------ |
| `go`     | `go.mod`                                         |
| `java`   | `pom.xml`, `build.gradle`, `build.gradle.kts`    |
| `python` | `pyproject.toml`, `requirements.txt`, `setup.py` |

Output files:

```text
AGENTS.md
.github/copilot-instructions.md
CODEX.md
```

Behavior:

* Default: create missing files only
* `--merge`: merge into AIW-managed sections
* `--force`: overwrite target files

Summary output reports:

```text
created
merged
wrote
skipped existing
```

## 12. `aiw wt ignore`

Creates `.gitignore` or appends:

```text
.wt/
```

If the rule already exists, no duplicate entry is added.

# Git Utilities

## `aiw git cz` (Conventional Commit Wizard)

### Default Behavior

* LLM disabled by default
* Interactive bilingual wizard for:

  * type
  * scope
  * subject
  * body
  * breaking changes
  * footer

### Long-Text Editing

For body/breaking/footer fields:

```text
/edit
Ctrl+E
```

launches an external editor.

### LLM Support

Enabled only with `--llm`.

Uses the OpenAI Chat Completions API directly.

```bash
set OPENAI_API_KEY=your_api_key

# optional
set OPENAI_MODEL=gpt-4o-mini
set OPENAI_BASE_URL=https://api.openai.com/v1
```

### Configuration Priority

```text
CLI
→ Project Root Configuration
→ Program Directory Configuration
```

Supported configuration files:

```text
aiw.toml
.aiw.toml
```

### OpenAI Configuration Priority

```text
[cz] section in config
→ environment variables
→ .env in current directory
→ .env in program directory
→ defaults
```

Supported options:

```toml
model
base_url
api_key
```

Mapping:

```text
OPENAI_MODEL
OPENAI_BASE_URL
OPENAI_API_KEY
```

Example:

```toml
[cz]
llm = false
candidates = 3
emoji = false
EDITOR = "code --wait"
model = "gpt-4o-mini"
base_url = "https://api.openai.com/v1"
api_key = ""

[[cz.types]]
value = "feat"
name = "feat:     New Feature | A new feature"

[[cz.types]]
value = "fix"
name = "fix:      Bug Fix | A bug fix"
```

### New Features

* Supports `--retry` (`-r`) to restore the most recent commit as a draft for amendment or resubmission.
* Interactive `issue-prefix` selection now supports both predefined options and custom input.

## task.toml Format

```toml
id = "payment-retry"
type = "task"
status = "TODO"
created = "2026-05-28"
updated = "2026-05-28"
branch = "feature/payment-retry"
worktree = ".wt/payment-retry"
```

## task-id / spec-id Rules

Allowed characters:

```text
a-z
A-Z
0-9
-
_
.
```

Any other character is considered invalid.

## Quick Start

```bash
# Initialize repository
aiw init
aiw init --prompts --merge
aiw init --prompts --template go

# Task workflow
aiw new payment-retry
aiw wt add payment-retry
aiw context payment-retry
aiw status payment-retry IN_PROGRESS
aiw done payment-retry
aiw archive payment-retry --finalize

# Worktrees
aiw wt list
aiw wt prune --dry-run
aiw wt lock payment-retry "in review"
aiw wt rm payment-retry --delete-branch
aiw wt ignore

# Git utilities
aiw git st
aiw git save "feat: add retry"
aiw git sync
aiw git update main
aiw git log
aiw git help

# TCC wrapper
aiw tcc hello.c -o hello.exe
aiw tcc dll hello.c -o hello.dll
aiw tcc x86_64 hello.c -o hello.exe
aiw tcc run hello.c

# Prompts
aiw prompts list
aiw prompts go --merge
```
