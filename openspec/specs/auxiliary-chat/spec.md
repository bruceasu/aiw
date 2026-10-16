# Auxiliary conversations

## Purpose

Provide independent research, prototype, article editing and teaching chats
without creating or advancing Issue/FD records.

## Requirements

### Requirement: Shared auxiliary entry points

AIW MUST expose research, prototype, edit-article and teach as built-ins with
new/topic, list, resume ID and recover ID forms. All accept provider/model
overrides for model turns; prototype accepts logic/ui branch selection at
creation. Help/list/recover MUST NOT call a model. No-argument invocation MAY
offer saved-task selection. Source/branch-only invocation MUST create a new
task rather than silently discard those options on a resumed task.

Each task/course MUST use an independent Session and auxiliary metadata in the
current working directory's .ai/sessions tree. No Issue approval, FD creation,
Git operation, deployment or automatic product modification is implied.

#### Scenario: Inspect saved courses without a model

- WHEN a user invokes teach list or any auxiliary command help
- THEN local metadata/help is shown without provider dispatch or Issue/FD creation

### Requirement: Explicit generation and incremental teaching

Research/prototype/article messages MUST clarify the task; teaching messages
MUST explain, practice and give feedback in a continuing course. Each accepted
message permits at most one model turn. A ready brief MUST be displayed before
explicit /run generation. New discussion/source input invalidates readiness;
budget rejection MUST NOT silently preserve runnable stale discussion approval.
Teaching /run generates another lesson/material, not completion of the course.
Multiple courses MUST retain separate goals, histories, sources and artifacts.

#### Scenario: New discussion supersedes a ready brief

- WHEN new discussion is submitted after a ready brief
- THEN prior generation readiness is invalidated before context dispatch
- AND /run requires the newly displayed ready brief

### Requirement: Bounded frozen context

AIW MUST load the selected Skill and auxiliary adapter, plus only the chosen
prototype branch, from the documented source precedence. Skill/source text MUST
be snapshotted and digest-bound. User sources are data, not execution instructions.
Host source selection MUST remain within the working directory and reject
links/traversal/known credential paths, unsupported text and exceeded budgets.
Source selection MUST NOT be described as comprehensive CLI filesystem isolation.

Full history MUST remain locally retained; model memory MUST include the goal,
brief/progress and a bounded recent window, with optional explicitly recalled
turn. Budget failure MUST be visible; no hidden summary calls or silent claims
that all history was read. Per-turn frozen instruction/input evidence MUST be
retained. Model errors/invalid responses MUST NOT trigger automatic retries.

#### Scenario: Reject an unavailable source scope

- WHEN selected input traverses outside the workspace, contains links or exceeds budgets
- THEN source loading fails explicitly without silently broadening access

### Requirement: Host-written non-destructive artifacts

Models MUST return a bounded JSON reply/brief/files envelope. Only explicit
generation may include files. Host MUST validate the whole set before writing:
safe single names, Windows device-name rejection, case-insensitive uniqueness,
allowed extensions and count/size limits. Results MUST use turn-specific
directories; original articles and production code MUST NOT be overwritten.
Generated code/HTML MUST NOT be automatically run or opened.

Research command mode uses explicit local sources; absent browsing or a research
agent MUST remain explicit. CLI adapters may narrow the standalone Skill's
automatic branches/full-workspace behaviors without changing other Agent usage.

#### Scenario: Generate a revised article copy

- WHEN /run returns validated article files
- THEN host saves turn-specific artifacts and displays their actual location
- AND the original draft is not overwritten or automatically published

### Requirement: Interruption and recovery without model replay

Before dispatch, auxiliary state MUST record pending turn/input/phase and brief
identity. Session LastTurn, committed history and auxiliary LastTurn MUST be
reconciled before another model turn. Mismatch/pending/running state MUST block
normal resume. recover MUST use only a verified saved Session result to complete
local artifact/history registration; matching files may be verified and missing
files completed, while conflicting files remain untouched. Unknown running turns
and leftover locks MUST remain for inspection, never automatically stolen.

Ctrl+C MUST propagate cancellation to an active model request. EOF MUST stop
further input; an already submitted turn MUST complete and save before exit.
Normal pause MUST retain completed progress. An independent outer auxiliary lock MUST reject
competing auxiliary writers without deadlocking Session's short update locks.
Generic Session tools are not authorized concurrent auxiliary editors.

#### Scenario: Completed model result, interrupted local registration

- WHEN Session has advanced its turn but auxiliary metadata retains pending
- THEN normal resume refuses another model call
- AND explicit recover reads that exact saved output and registers it locally
- AND no model is replayed or conflicting partial artifact overwritten

#### Scenario: Continue an existing course

- WHEN the user resumes one of several teach courses
- THEN its goal, brief/progress and selected local history are restored
- AND other courses' material is not injected or overwritten
- AND the user may learn further, recall an earlier turn or generate a new lesson

#### Scenario: Bounded history cannot fit

- WHEN selected goal/progress/history exceeds the memory budget
- THEN the host refuses dispatch and offers explicit window/recall adjustment
- AND full local history remains available without hidden model compression
