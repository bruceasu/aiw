# aiw

AIW is a workflow-first CLI for organizing work, preserving task state, and exposing reusable capabilities through a small core, Skills, AI support, and plugins.

## Feature Overview

* Initialize the OpenSpec-backed directory structure and default instruction files
* Automatically create or append `.wt/` entries to `.gitignore`
* Generate or merge AI prompt files from `docs/agent-templates/`
* Create, view, and update tasks
* Create dedicated Git worktrees for tasks
* Output task-specific context prompts
* Create and maintain long-lived specification documents
* Archive completed tasks
* Generate the task registry file `openspec/registry.json`

## Directory Structure

After running `aiw init`, the following structure will be created (if missing):

```text
repo/
鈹溾攢鈹€ openspec/
鈹?  鈹溾攢鈹€ changes/
鈹?  鈹溾攢鈹€ specs/
鈹?  鈹溾攢鈹€ archive/
鈹?  鈹斺攢鈹€ registry.json
鈹溾攢鈹€ .wt/
鈹溾攢鈹€ AGENTS.md
鈹斺攢鈹€ .github/
    鈹斺攢鈹€ copilot-instructions.md
```

Notes:

* `AGENTS.md` and `.github/copilot-instructions.md` are created only if they do not already exist.
* `registry.json` is generated from all `openspec/changes/*` task folders.
  `openspec/changes/archive/` is reserved for archived changes.
  Metadata discovery prefers `task.toml` and falls back to legacy `tasks.toml`.

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
aiw --help
aiw help [command|topic]

aiw init [--prompts] [--merge] [--force] [--template <name>]
aiw new <task-id> [--backend auto|openspec|native]
aiw list
aiw show <task-id>
aiw status <task-id> <status>
aiw done <task-id>
aiw archive <task-id> [--push] [--cleanup-wt] [--delete-branch] [--finalize] [--backend auto|openspec|native]

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
aiw task agent next <task-id>
aiw registry

aiw prompts list
aiw prompts [template] [--merge] [--force]

aiw tcc [args...]       # TCC wrapper with automatic include/lib defaults
aiw git <subcommand>    # run: aiw git help
```

Every built-in command accepts `-h`/`--help` where detailed command help is
available. Use `aiw help <command>` for the same overview from the top-level
help router. Unknown commands are resolved as plugins and use the same
`aiw <plugin> --help` convention.

## Workflow backend selection

Task workflow commands use `auto` by default. When a verified OpenSpec CLI is
available, `new` and `archive` delegate to it; otherwise AIW uses its native
implementation. `decision` and `spec` currently have no direct OpenSpec CLI
mapping and use native fallback in `auto` mode.

Use `--backend native` to force the built-in behavior, or
`--backend openspec` to require OpenSpec and fail if delegation is unavailable.
Configure a specific executable with `AIW_OPENSPEC_BIN`.

## Sequential agent handoff

`aiw task agent next <task-id>` is an auxiliary orchestration command for a
task. The task metadata must bind an AIW-managed session and worktree:

```toml
session = "TASK-123"
worktree = ".wt/TASK-123"
```

The command acquires a per-task lease, creates `artifacts/handoff.md`, and
starts a new Codex Thread in the same worktree. Lineage is recorded in
`openspec/changes/<task-id>/agent-lineage.json`.
Use `aiw task agent status <task-id>` to inspect the recorded parent/child
Thread transition.

This workflow is sequential. Use `aiw wt` to give parallel agents separate
worktrees and sessions. Existing `aiw flow` and `aiw cxs` entry points are
auxiliary execution and session-support surfaces, not the core workflow.
`task.toml` is canonical; `tasks.toml` is a legacy fallback.

# Plugin System

`aiw` supports extending subcommands through external executable plugins.

## Managed Skills

Use `aiw skills` to list and safely install canonical Portable Skills or
path-based Skill bundles into the current project's `.agents/skills`
directory:

```text
aiw skills list
aiw skills install tdd --dry-run
aiw skills install ./bundle.zip
aiw skills install tdd
```

The installer protects unmanaged same-name directories and records verified
AIW-managed copies in `.agents/skills/.aiw-skills.json`. Path-based installs
use the same managed pipeline as canonical Skill installs. Run
`aiw skills --help` for constraints and JSON automation options.

Canonical Skill packages are maintained in the repository-root `skills/`
directory. Release layouts keep `skills/` beside `program/` and `plugins/`.
`aiw-install-skill` is deprecated; use `aiw skills install` for both canonical
names and local bundle sources.

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

### Python Interpreter Configuration

Python plugins use the first available interpreter in this order:

1. The absolute path in `AIW_PYTHON`
2. `[runtime].python` in the user `aiw.toml`
3. `[runtime].python` in `aiw.toml` beside the AIW executable
4. `python/python.exe` on Windows, or `python/python` on other platforms, beside
   the AIW executable
5. `python`, then `python3`, from `PATH`

The program-directory configuration provides defaults. User configuration
overrides those defaults, and `AIW_PYTHON` provides a temporary environment
override.

```toml
[runtime]
python = "C:/Python312/python.exe"
```

Configured interpreter paths must be absolute paths to existing files. An
invalid explicit path produces an error instead of silently selecting another
Python runtime. An empty value is treated as unset.

The canonical user configuration locations are:

* Windows: `%APPDATA%\aiw\aiw.toml`
* Linux and other XDG platforms:
  `$XDG_CONFIG_HOME/aiw/aiw.toml`, or `$HOME/.config/aiw/aiw.toml` when
  `XDG_CONFIG_HOME` is unset
* macOS: `$HOME/Library/Application Support/aiw/aiw.toml`

If the canonical file does not exist, AIW also checks
`$HOME/.config/aiw/aiw.toml` as a compatibility fallback. AIW reads only the
first existing user configuration file and does not merge user files. It does
not read project-root configuration for interpreter selection, and it does not
create a missing user configuration file or directory.

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
鈹溾攢鈹€ task.toml
鈹溾攢鈹€ tasks.md
鈹斺攢鈹€ notes.md
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
鈹溾攢鈹€ spec.toml
鈹斺攢鈹€ spec.md
```

## 5. `aiw status <task-id> <status>`

Updates:

* `task.toml` (or legacy `tasks.toml` if present)
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

## 13. `aiw wt ignore`

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
鈫?Project Root Configuration
鈫?Program Directory Configuration
```

Supported configuration files:

```text
aiw.toml
.aiw.toml
```

### OpenAI Configuration Priority

```text
[cz] section in config
鈫?environment variables
鈫?.env in current directory
鈫?.env in program directory
鈫?defaults
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

Compatibility note:

* Existing repositories that still use `tasks.toml` are supported.

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

# Codex sessions (plugin)
aiw cxs list -n 20
aiw cxs exec "summarize current diff"
aiw cxs exec --session payment-retry "continue implementation"

# Help
aiw --help
aiw help git
aiw flow --help
aiw github --help
aiw skills --help
```

For full `aiw cxs` usage, see `docs/usage/aiw-cxs.md`.

## Available Plugins

Plugins are discovered beside the `aiw` executable. The repository currently
ships these common entry points as external plugins:

| Command | Purpose | Detailed help |
| --- | --- | --- |
| `aiw wt` | Create and maintain task worktrees | `aiw wt --help` |
| `aiw git` | Git helpers and discoverable Git subcommands | `aiw git help` |
| `aiw flow` | Auxiliary automation for Codex session workflows | `aiw flow --help` |
| `aiw cxs` | Auxiliary inspection and continuation for Codex CLI sessions | `aiw cxs --help` |
| `aiw skills` | List and install canonical Skills | `aiw skills --help` |
| `aiw github` | Read and publish GitHub Issues and PRs | `aiw github --help` |
| `aiw cz` | Run the Conventional Commit wizard | `aiw cz --help` |
| `aiw tcc` | Compile or run Tiny C Compiler programs | `aiw tcc --help` |

Plugin HELP is intentionally generated by each plugin so its examples stay
next to the parser. If a plugin is not installed, the top-level help reports
the missing discovery location instead of showing stale commands.





