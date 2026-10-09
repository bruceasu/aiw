# FD-first development workflow

Use `aiw fd` for new engineering work. A numbered FD contains the design,
stable Work Item IDs, progress, acceptance, and Verification. The FD file is
the source of truth; `FEATURE_INDEX.md` is a lookup index. An approved Issue
can be linked with `--issue`; a Task is optional.

```text
aiw fd new "Improve report export" --issue ISSUE-001
aiw fd list
aiw fd show FD-002
aiw fd claim FD-002 FD-002-000002-design-requested --session <host-session-id>
aiw fd emit FD-002 design-ready --producer planner --artifact docs/features/FD-002_IMPROVE_REPORT_EXPORT.md
aiw fd resume FD-002
aiw fd emit FD-002 implementation-ready --producer worker --artifact docs/features/reports/FD-002-implementation.md --source-event FD-002-000003-design-ready
aiw fd emit FD-002 verification-passed --producer reviewer --artifact docs/features/reviews/FD-002-review.md --source-event FD-002-000004-implementation-ready
aiw fd request-review FD-001 --reason "文档在验收后更新"
aiw fd reopen FD-004 --reason "继续未完成的验证"
```

`--issue` uses `aiw issue show <id> --json` to resolve the canonical source ID,
validate captured artifacts, and check approval. Both ISSUE and existing REQ
records are supported, including unique REQ number abbreviations. Existing FD
source lines are preserved; a new FD records the canonical full ID. Use matching
versions of the FD plugin and Issue CLI so the structured interface is available.

### FD list output

`aiw fd list` builds a plain-text table from active and archived FD documents,
grouped by status. Each row shows the FD ID, status, priority, and title.
Status and priority use ANSI colors only when stdout is a TTY, `NO_COLOR` is
unset, `TERM` is not `dumb`, and at least one recognized capability signal is present:

- `TERM` starts with `alacritty`, `ansi`, `cygwin`, `eterm`, `foot`, `iterm`,
  `kitty`, `konsole`, `linux`, `msys`, `putty`, `rxvt`, `screen`, `st-`,
  `tmux`, `wezterm`, or `xterm`.
- `TERM_PROGRAM` is `alacritty`, `apple_terminal`, `hyper`, `iterm.app`,
  `kitty`, `tabby`, `vscode`, or `wezterm`.
- `WT_SESSION` or `ANSICON` is nonempty, or `ConEmuANSI` is `on`.

These signal comparisons are case-insensitive. Redirected output, disabled
colors, and terminals without a recognized signal receive plain text with all
fields preserved. Invisible format and control characters are removed from displayed
fields; characters unavailable in the stdout encoding are escaped so listing
does not fail.

The creation event routes to Planner. Planner writes options, decision,
acceptance, and numbered Work Items before `design-ready`. Worker implements
all ready items, performs the repository-authorized compile-only check and
static review, and records a report before emitting `implementation-ready`.
New FDs route directly to an independent Reviewer. New FDs use
**Evidence policy: Dual**: reports use Chinese Markdown for people and a
same-basename JSON sidecar for CLI/AI. Put one
`<!-- aiw-data: FD-XXX-report.json -->` comment in Markdown, use Markdown for
`--artifact`, and write JSON with schema `aiw.fd.evidence.v1`, FD ID, kind,
source event, Markdown filename, and `data`. Use the JSON templates under
`docs/templates/`. Reviewer reports use kind `reviewer-report` and live under
`docs/features/reviews/`. The CLI checks the JSON before handoff, and close
archives both files. Older Markdown-only evidence is preserved. Relative
filename references remain valid after archive. An independent Reviewer emits
`changes-requested` with findings or
`verification-passed` with a review report. A role must claim or receive its
handoff before completing it, then pass `--source-event <event-id>`.
`aiw fd show` displays the last ID.

#### Read-only status and evidence inspection

```text
aiw fd show FD-038
aiw fd show-report FD-038
aiw fd show-report FD-038 --last
aiw fd show-review FD-038
aiw fd show-review FD-038 --last
```

`show` keeps the FD Markdown and existing last-handoff receipt, then adds the
FD status, verified worktree/branch/parent branch, active handoff (when the
latest receipt is pending, launching, or dispatched), and latest event. Event
summaries include their creation time and JSON. Missing workspace metadata,
unverifiable metadata, no active handoff, and no event are reported explicitly.

`show-report` and `show-review` inspect Markdown evidence named
`FD-XXX-*.md` in the current checkout, `HEAD`, the verified FD worktree and
its recorded local/parent branches, and the FD archive. The branch and
worktree metadata must match registered Git refs/worktrees; invalid sources
are skipped with a warning. Branch-only files are read from Git trees without
checking out a branch. Results are sorted newest first, with UTC timestamps,
Markdown source paths, and matching JSON sidecar paths when present. Worktree
timestamps use file modification time; branch-only timestamps use the last
commit that touched the file. Identical Markdown content is listed once per
evidence kind with all source and sidecar paths retained.

In a terminal with both stdin and stdout attached to a terminal, the commands
show a numbered list and accept a selection. Blank input or `q` cancels.
Outside an interactive terminal, they print the list and a usage hint without
waiting for stdin. `--last` prints the newest Markdown body directly and does
not read stdin. No matches produce a clear empty result. These three commands
are read-only.

Invoke `$fd-test` when you want black-box scenarios, test execution, or a
factual test report. It is a standalone Skill: it does not emit FD events or
change status, and its report is not an acceptance gate. Reviewer and other
review agents do not inspect or assess that optional report. The Skill follows
the repository's test authorization rules; asking for case design alone does
not authorize execution. New repository test code belongs under the root
`tests/` directory.

#### Legacy Tester CLI compatibility

FDs that already declare `**Test policy:** Independent` retain their existing
Tester events, PM decision events, and `aiw fd refresh-tester` behavior. The
CLI continues validating legacy report fields and authorization records for
those receipts. This path is not used for new FDs; do not dispatch it as part
of the default workflow. Historical report templates remain available under
`docs/templates/TEST_*_TEMPLATE.md`.

## One-operation host workflow

Ask the host agent to run `$fd-workflow auto` with a feature request or a
single FD ID. This is a Skill operation, not an `aiw fd auto` CLI command. It
creates or resumes one numbered FD, splits ordered Work Items, designs and
implements them, performs compile-only and static checks, and delegates each
review to a separate `fd-review` subagent. It repairs concrete findings and
closes with `aiw fd close <id> Complete` after a current Reviewer pass. The
host counts at most three Reviewer outcomes for
the active implementation cycle across interrupted/resumed auto runs. A third
failed review leaves the FD active with its findings. The resulting Worker
handoff stays pending for a later human-directed recovery.
On a later `$fd-workflow auto` invocation, the host checks the earlier review
count and latest findings before claiming that pending Worker handoff; it does
not reset the three-round limit.

Auto uses the normal claim/source-event receipts. It stops if a role is
already in flight, another Session owns a handoff, a material choice needs
the human, or no separate Reviewer subagent is available. The request does
not authorize tests, final builds, network access, permission escalation, push,
or deployment. Under the shared work-management contract, a whole-lifecycle
or isolation request authorizes focused local commits, squash delivery to the
recorded parent after a passed review, verified worktree cleanup, and archive
after delivery. Follow narrower repository rules.

The event receipt includes FD ID, revision, type, producer, target role, and
artifact path. `AIW_FD_ROLE_RUNNER` may name an executable that receives
`<role> <fd-path> <event-json-path>`. AIW starts it when a new event is ready.
The runner should read those files, work in the named role, and emit the next
event with the original event ID. Its stdout and stderr go to the event log.
If no runner is configured, the event stays pending. Before a host Agent starts
writing, bind that exact event with `aiw fd claim <fd-id> <event-id> --session
<host-session-id>`. The claim is atomic and idempotent for the same session;
another session cannot take it. When that role emits its result, pass the
claimed event ID through `--source-event`. If a runner is configured, AIW
claims the event as it starts that runner. A pending event may also be
dispatched later by `aiw fd resume`.

`resume` does not launch a second writer for a launching or dispatched event.
Inspect its original session or event log and reconcile the result first.
Changing the FD manually does not itself emit an event. Stage operations
increase `**Revision:**`; a pending role event must be claimed before its role
can complete it, and a changed FD cannot be claimed until reconciled. A
`Complete` archive also requires the FD content to match the Reviewer's
`verification-passed` receipt. Reconcile changed content with a new review.
FD claim and dispatch compare content after normalizing CRLF to LF, so a
Windows line-ending conversion alone does not invalidate a handoff.
For an active `Open` or `In Progress` FD whose latest pending Worker event is
stale after PM edits, run `aiw fd refresh-worker <fd-id> --reason "..."`.
This cancels the old event and creates a `work-requested` Worker handoff for
the current FD revision and digest. Claim the new event and cite it on
`implementation-ready`. A current or in-flight Worker event cannot be
replaced; use `request-review` only for the separate `Pending Verification`
review-recovery case.
If PM has confirmed that a dispatched Worker session has stopped, use
`aiw fd recover-worker <fd-id> --expected-event <event-id>
--expected-session <old-session> --reason "..."`. This records the reason,
cancels the old receipt, and creates a new pending Worker event. The new
Worker claims the event under its own session and cites it on
`implementation-ready`. A mismatched event or session is rejected.
If a process crashes while holding `.ai/fd/<id>/.mutation-lock`, inspect the
original process and event receipt before removing the stale lock. Never
remove it while the role may still be writing.

When implementation uses isolation or `wt`, commit the ready FD plan and
ensure `.wt/` and `.ai/` are ignored by Git, then create its worktree with
`aiw git wt add FD-002`. Add checks these paths before changing Git state and
reports missing ignore rules. It writes the FD ID, parent
branch, feature branch, and worktree path to
`.ai/fd/FD-002/workspace.json`; read and verify that record immediately.
Inspect both worktrees with `aiw git wt status FD-002`. After a passed review,
deliver one squash commit with `aiw git wt local-merge FD-002`. If the parent-side
squash has content conflicts, the command resets it and merges the parent into the FD worktree.
Resolve and commit there, then rerun `local-merge` explicitly. Successful
delivery verifies the squash source, then removes the FD worktree and branch
while preserving `.ai/fd/<FD-ID>` receipts. Archive only after delivery
succeeds. This authorizes local commits and the requested squash/archive; it
does not authorize push, release, or deployment.

After a passed review, run `aiw fd close FD-002 Complete` once any requested
isolated merge has succeeded. Use `--reason "..."` for `Deferred` and `Closed`
outcomes. Manual close requests still require their own explicit decision.
This archives the FD file and rebuilds the index; it does not change Git
delivery state.

Native archived FDs live at
`docs/features/archive/<FD-ID>/<FD-ID>_SLUG.md`. Their reports and reviews
live in that FD directory's `reports/` and `reviews/` subdirectories. Existing
flat native archives are migrated into this layout with their evidence.

To continue an archived `Closed` or `Deferred` FD, run `aiw fd reopen <fd-id>
--reason "..."`. This preserves the earlier close record and Work Items,
returns the FD to the active index as `In Progress`, and creates a Worker
handoff. Claim that exact handoff before editing, then use the normal
`implementation-ready` and independent review flow. Earlier evidence stays
archived; write new reports and reviews in the active evidence directories.
Give new evidence distinct filenames, such as `FD-004-implementation-r2.md`,
because a later close will reject an archive filename collision.
If the reason was entered incorrectly, use `aiw fd reopen <fd-id> --reason
"corrected text" --correct-reason` before the Worker claims the new event.
This keeps the event ID, updates its FD digest, and records the old reason in
the receipt's correction history. A claimed handoff cannot be corrected here.
Use `request-review` for an archived `Complete` FD.

If an archived Complete FD is later updated, request a fresh review with
`aiw fd request-review FD-001 --reason "..."`. The command records the current
content and reason, increments the revision, returns the FD to the active
index as `Pending Verification`, and preserves its previous completion date.
The new Reviewer handoff uses the normal `claim` and `--source-event` flow.
After `verification-passed`, close it as Complete again. The request does not
change Work Item checkboxes; explicitly revise the FD if review findings add
work.

An active `Pending Verification` FD with a stale unclaimed handoff can also use
`aiw fd request-review FD-005 --reason "恢复独立评审"`. This creates a new Reviewer
event for the current FD digest and cancels the old pending receipt. It does
not invent a Worker `implementation-ready` event. Claim the new event in an
independent Reviewer session. Outstanding `%% NEEDS_INPUT` notes still block a
Reviewer pass; the Reviewer can issue `changes-requested` to return work to
Worker. An in-flight event must be reconciled before requesting review.

Legacy Task records remain readable, but `aiw wf` has been removed. New
engineering work uses numbered FDs and the `aiw fd` / `aiw git wt` commands.
Stable specs still live in `openspec/specs/`; create an OpenSpec change only
when explicitly requested.

## Legacy migration

An existing Task can keep its current FD, Core records, Session IDs, evidence,
and Git lineage. Read it through the legacy commands; a new numbered FD does
not require conversion. To move unfinished design work deliberately, create a
new numbered FD, cite the old Task and FD in `Sources`, and copy only decisions
that still apply. Keep old Work Item IDs and completed evidence in the legacy
record. Give the new FD its own Work Item IDs and record what remains to be
verified. Never synthesize FD events or Reviewer approval from historical Core
records. Archive or delete the old Task only through a separate explicit
decision after reconciling its outstanding work.

## 人工强制恢复（FD-042）

### 选择恢复命令

先用 `aiw fd show <fd-id>` 查看当前状态、最新事件与会话，再根据实际情况选择命令。

| 情况 | 命令 | 前提或结果 |
| --- | --- | --- |
| 活动 Open / In Progress FD 的 pending Worker 交接已过期 | `refresh-worker` | 替换旧交接，新 Worker 领取新事件 |
| 已确认 dispatched Worker 会话停止 | `recover-worker` | 必须匹配旧事件和会话，创建 pending Worker 交接 |
| Pending Verification 的未领取交接已过期 | `request-review` | 创建当前 FD 内容对应的 Reviewer 交接 |
| 继续归档的 Closed / Deferred FD | `reopen` | 返回活动 In Progress，保留旧证据并创建 Worker 交接 |
| 重新评审归档的 Complete FD | `request-review` | 需要正常 Reviewer 通过证据，返回 Pending Verification |
| 人工取消最新 dispatched 收据 | `cancel-event` | 只取消收据，不停止原 Agent，不创建后继交接 |
| 人工覆盖活动 FD 状态 | `set-status` | 增加 revision，不归档文件，不生成审查证据 |
| 强制归档为 Complete | `close <id> Complete --force --reason "..."` | 允许任一活动状态；设为 Complete 并归档，记录跳过的 Reviewer 门槛 |
| 人工跳过流程门槛创建交接 | `force-emit` | 新事件始终 pending，后续显式 claim / resume |

强制命令的审计记录不能替代独立 Reviewer 结果。取消或替换收据之前，操作者需要
处理原会话可能继续写入的问题；这些命令不提供进程停止功能。

以下人工强制命令仅操作活动 FD。`cancel-event`、`set-status` 和
`force-emit` 要求 `--reason`（非空单行，最多 500 字）与 `--operator`
（非空单行，最多 200 字）；`close --force` 要求非空单行 `--reason`，并从本地
用户信息记录操作者。`--operator` 是声明身份，不是身份认证；审计同时记录本地用户。
不带 `--force` 的普通 `close` 校验保持不变。

```text
aiw fd cancel-event FD-001 --expected-event FD-001-000004-implementation-ready --reason "Operator recovery" --operator maintainer
aiw fd set-status FD-001 "In Progress" --reason "Resume implementation" --operator maintainer
aiw fd force-emit FD-001 design-ready --producer planner --artifact docs/features/FD-001_EXAMPLE.md --reason "Restart Worker handoff" --operator maintainer
aiw fd close FD-044 Complete --force --reason "Operator accepts completion without Reviewer evidence"
```

- `cancel-event` 只取消精确指定的最新 dispatched 收据，保留会话、pid 和历史字段，
  不修改 FD 状态或 revision。取消收据不会停止 Agent；原会话仍可能继续写入。
- `set-status` 可设为 Planned、Design、Open、In Progress、Pending Test、
  Pending Test Acceptance、Pending Verification、Complete、Deferred 或 Closed。
  它跳过状态转换和终态证据校验，增加 revision，更新索引，保留 Work Items、
  收据和文件位置。原收据可能失效；应显式 force-emit 或按正常恢复路径继续。
- `force-emit` 支持 emit 帮助中的全部已定义事件，跳过当前阶段、Work Items、
  NEEDS_INPUT、证据和前序 claim 校验；仍要求仓库内存在的 artifact 和有效角色配对。
  design-requested 的 producer 为 pm，decision-recorded 为 human，其余结果事件
  沿用正常 producer；needs-decision 可用 pm/planner/worker/tester/reviewer/human。
  新事件沿用定义的目标角色和结果状态；decision-recorded 按当前状态恢复角色，
  终态目标为 pm；Independent FD 的 implementation-ready 仍指向 tester。
  它增加 revision，将旧未结束收据取消并保留后继关联，新收据标记 `forced: true`，
  始终停在 pending，配置了 runner 也不会自动启动。确认原会话已处理后可显式 claim/resume。

`set-status Complete` 或强制 verification-passed 不代表审查通过，不会满足普通
`close Complete` 或 archived `request-review` 对真实 Reviewer 验收的要求。
`close Complete --force` 是显式例外：它会同时把活动 FD 设为 Complete 并归档，
记录跳过状态与 Reviewer 检查的审计，但不会创建 Reviewer 通过证据。强制归档仍拒绝
有 launching/dispatched 收据、目标冲突或不安全路径的 FD。归档 FD 使用现有
reopen/request-review。强制归档会在同一事务中取消最新的非 `verification-passed`
pending 收据；已有 `verification-passed` 收据保持不变。其他强制命令不会自动搬移历史文件。

每次成功操作在共享 `.ai/fd/<FD-ID>/operations/<uuid>.json` 留下审计，包含操作者、
本地用户、原因、UTC 时间、原状态/事件、结果和跳过的检查；取消/强制事件保存审计引用。
审计与 FD/收据/索引写入受现有 FD 锁保护，失败时恢复原内容；回滚不完整会明确报错。
该锁不能阻止原 Agent 直接写文件，操作者仍负责处理并发写入风险。
