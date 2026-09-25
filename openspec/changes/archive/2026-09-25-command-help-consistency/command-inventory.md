# CLI Dispatch Inventory

This inventory is the implementation reference for command help, shell completion, and README alignment. It records source-dispatched forms; it does not change command behavior. Sources reviewed: `main.go`, `internal/commands/task/command.go`, `internal/commands/task/workflow_commands.go` (dispatch and validation), and `internal/commands/completion/completion.go`.

## Top-level commands

`main.go` routes `-h`/`--help` and `help`; `init`, `new`, `list`, `show`, `status`, `done`, `archive`, `context`, `decision`, `spec`, `requirement`, `prompts`, `completion`, `ask`, `session`, and `cz`; `turn`/`chat`; `workflow`/`workspace`; and `task`. `wt` is handled by the external `aiw-wt.py` plugin fallback, not a built-in dispatch case. Unknown top-level names also use plugin fallback.

## Task commands

`aiw task <command> [args...]` dispatches `workflow`, `turn`/`chat`, `requirement`, `init`, `new`, `list`, `show`, `status`, `done`, `archive`, `context`, `decision`, `spec`, `prompts`, and `workspace`. The `workspace` task command binds a Task workspace; `workflow` has both top-level (`aiw workflow`) and Task (`aiw task workflow`) entry points. Task operations accept their respective positional forms: `new <task-id> [--allow-unrelated-dirty]`; `list [--all]`; `show <task-id>`; `status <task-id> <status>`; `done <task-id>`; `archive <task-id> [--push] [--cleanup-wt] [--delete-branch] [--force]`; `context <task-id>`; `decision <task-id>`; `spec <spec-id>`; `prompts [options]`; and `workspace bind <task-id> --primary`.

## Workflow commands

Both workflow entry points accept the operations below. `repair-metadata` is a special form and takes no Task ID; the remaining operations are dispatched with a Task ID as shown.

| Operation | Accepted form / arguments |
| --- | --- |
| `plan`, `sync` | `<task-id>` |
| `advance` | `<task-id>` |
| `run` | `<task-id> [--execute] [--primary] [--provider NAME] [--model MODEL]` |
| `supervise` | `<task-id> <start|status|stop>`; its parser also accepts provider/model overrides and execution/primary controls where applicable |
| `recommend-routing` | `<task-id>` |
| `attempt` | `<task-id> start <work-item-id> <attempt-id>` or `<task-id> checkpoint <attempt-id> <running|paused>` |
| `evidence` | `<task-id> <evidence-id> <work-item-id> <kind> <state> [reference]` |
| `gate` | `<task-id> <gate-id> <resolved|waived>` |
| `skip-focused-test` | `<task-id> <reason>` |
| `complete` | `<task-id> <work-item-id>` |
| `retry-policy` | `<task-id> <work-item-id> <1-5>` |
| `reopen` | `<task-id> <work-item-id> <reason>` |
| `force-close` | `<task-id> <merged|discarded> <reason>` |
| `focused-test` | `<task-id> <attempt-id>` |
| `delivery` | `<task-id> <merged|discarded>` |
| `local-merge` | `<task-id> <commit-message>` |
| `delivery-failed` | `<task-id> <stage> <detail>` |
| `report`, `diagnose`, `recover`, `repair` | `<task-id>` |
| `repair-metadata` | `[task-id] [--dry-run]` |

`run` and supervised execution support provider/model overrides; `run` accepts `--execute` and `--primary`. Metadata repair accepts `--dry-run`. Exact constraints and additional supervisor options remain defined by the parsers in `workflow_commands.go` and should be preserved when help is updated.

## Callable operations intended for internal workflow use

The dispatch code deliberately makes low-level workflow state transitions callable: `attempt`, `evidence`, `gate`, `skip-focused-test`, `complete`, `retry-policy`, `reopen`, and `force-close`. They are lifecycle adapters used by managed workflow orchestration; their presence in dispatch does not make them ordinary user discovery examples. The `knowledge` and `auxiliary` prefixes are also callable through workflow dispatch and route to execution maintenance handlers before normal workflow parsing; they are internal maintenance entry points. Keep them in the source inventory, and explicitly decide their help/completion exposure against the intended public inventory rather than silently treating them as stale.

`repair-metadata`, `report`, `diagnose`, `recover`, `repair`, `recommend-routing`, delivery operations, and focused-test are separately dispatched callable operations and are part of workflow discovery/help alignment as applicable.
