## 1. Inventory and contract

- [x] 1.1 Build a complete inventory from the top-level, Task, workflow, validation, and completion dispatch code, including aliases, argument forms, and options.
- [x] 1.2 Record the inventory as the implementation reference and identify any intentionally internal-but-callable operations, with no command behavior changes.

## 2. Help and completion alignment

- [x] 2.1 Update top-level and Task help summaries/examples so their command names and workflow entry points match the dispatcher.
- [x] 2.2 Update workflow help to include every callable operation and accurate argument/option descriptions, including delivery-failed, provider/model overrides, focused-test, and repair-metadata.
- [x] 2.3 Update workflow shell completion definitions to cover the complete public operation inventory and remove stale omissions.

## 3. README alignment

- [x] 3.1 Update README command summaries, workflow examples, and option descriptions to match the source inventory and help output.
- [x] 3.2 Review nearby workflow documentation for stale command names or unsupported invocation forms and correct only the affected references.

## 4. Verification

- [x] 4.1 Add or update focused checks that detect drift between the source command inventory, help output, completion list, and README command references.
- [x] 4.2 Perform static review of the final diff and confirm no runtime command behavior, API, dependency, or persistence changes were introduced.

## TODO

- None.

## Verification

- Work Item 1.1: statically traced top-level routing in `main.go`, Task dispatch in `internal/commands/task/command.go`, workflow dispatch and validation in `internal/commands/task/workflow_commands.go`, and shell completion definitions in `internal/commands/completion/completion.go`.
- Work Item 1.2: recorded the reviewed command and argument inventory in `command-inventory.md`, including aliases, workflow options, and callable internal workflow maintenance/state operations. No command behavior was changed.
- Work Item 2.1: updated top-level and Task help summaries/examples to use dispatcher-supported commands, including workflow aliases/entry points and Task lifecycle, agent, and workspace forms. Removed stale `task agent` and `cxs` examples from built-in help. Static review only; no runtime behavior changed.
- Work Item 2.2: expanded workflow help descriptions for dispatched progress, delivery, recovery, routing, and metadata-repair operations; documented `delivery-failed`, `run` provider/model overrides and `--primary` constraint, and `supervise` provider/model overrides. Confirmed `focused-test` and `repair-metadata` descriptions against their parsers. Static source review only; no runtime behavior changed.
- Work Item 2.3: aligned the shell completion workflow operation list with the public operations documented in workflow help and the source inventory, adding routing, focused-test, delivery suboperations, reporting, and metadata repair. Internal lifecycle adapters (`attempt`, `evidence`, `gate`, `complete`, retry/reopen/force-close) remain callable but are excluded from user discovery completion. No command behavior changed.
- Work Item 3.1: aligned README workflow command summaries with the inventory, documented the full dispatched operation set and argument/option constraints, and added valid examples for delivery failure, focused-test, and metadata repair.
- Work Item 3.2: reviewed the adjacent workflow walkthrough in `docs/plugins-skills-development-review.md`; added supported focused-test, delivery-failed, and repair-metadata forms plus run/supervise option constraints. Existing workflow examples use the supported Task-ID-first argument order.
- Work Item 3.1/3.2: static review of README and adjacent documentation changes found no command behavior, API, dependency, or persistence changes.
- Work Item 4.1: added `internal/commands/task/workflow_surface_test.go`. Its focused consistency check compares the documented public workflow inventory against dispatch source, rendered workflow help, shell completion names, README, and the recorded command inventory. The test was not run as instructed.
- Work Item 4.2: reviewed the accumulated diff for README/help/completion/documentation and consistency-check additions. Changed production Go code is limited to help text and the shell completion operation list; no dispatch, execution, API, dependency, or persistence code changed. The new test is not run. No tests or runtime validation were run; compile-only validation is owned by the supervisor.
