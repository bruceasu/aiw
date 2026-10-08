# FD-038: Improve FD report, review, and status inspection

**Status:** Complete
**Revision:** 9
**Priority:** Medium
**Test policy:** Waived
**Evidence policy:** Dual

## Problem

在 FD 流程中查找 report、review 和当前运行状态需要手动遍历文件与事件记录。`aiw fd show <FD-ID>` 当前显示 FD 正文及最后 handoff，但没有 worktree/branch 信息、最后事件详情或方便选择历史 evidence 的入口。FD 归档后，报告和 review 还会移动到 FD 专属归档目录；同一 FD 也可能分别存在于父分支和 worktree 分支，因此只查当前工作目录会漏掉用户需要的结果。

## Options and decision

| 方案 | 判断 |
| --- | --- |
| 扩展 `fd show` 并新增 `show-report`、`show-review` 子命令 | 采用。职责清楚，保留已有 `show` 用法，并为报告/审查列表提供独立交互入口。 |
| 只增强 `fd show`，把所有 evidence 塞入一个视图 | 不采用。报告和 review 的查找/打开属于独立操作，混合输出会使状态概览冗长。 |
| 通过 workspace 元数据与 Git 验证来源，不根据 branch 名推导目录 | 采用。`workspace.json` 是 worktree/branch/parent_branch 的记录契约；Git 注册信息与引用校验可验证路径归属。元数据缺失时搜索可验证来源，损坏时报告并跳过该来源。 |

Evidence 来源包括当前 checkout 的工作树与 HEAD、经 Git worktree list 验证的 FD worktree、workspace.json 记录且经 Git ref 验证的 FD branch 和 parent_branch，以及 `docs/features/archive/<FD-ID>/reports|reviews/`。branch-only 文件通过 Git tree 读取，不 checkout 分支；任何文件系统路径都不得由 branch 名拼接。仅枚举以当前 FD ID 为前缀的 reports/reviews Markdown。按 evidence kind 与内容 SHA-256 合并相同副本并保留全部来源路径；内容不同的同路径文件作为不同版本展示。默认列表按可观察的修改时间倒序，最多 20 项：checkout/worktree 文件用文件 mtime，branch-only 文件用 Git 最近改动该路径的提交时间；列表以 UTC 时间和来源路径标明来源。若 sidecar 存在，列表显示其路径，打开操作输出 Markdown 正文。

## Solution

新增 `aiw fd show-report <FD-ID> [--last]` 与 `aiw fd show-review <FD-ID> [--last]`，共享只读 evidence inventory 和选择器。无 `--last` 时列出最新 20 项并可用编号选择；空输入或 `q` 安全取消。仅在 stdin 与 stdout 都是终端时读取选择。非交互环境列出结果和可操作用法，不读 stdin；`--last` 直接显示最新 Markdown。无 evidence 时给出清楚的空结果。列表显示 UTC 时间、来源路径及存在时的 JSON sidecar 路径。

扩展 `aiw fd show <FD-ID>`，在原有 FD 正文和 receipt 输出后增加 status、经验证的 worktree 路径和 branch。当前 handoff 定义为最新处于 pending、launching 或 dispatched 的事件；没有此类事件时显示“无当前 handoff”。Latest event 始终指事件目录中编号最新的 receipt。两者分别显示时间与格式化 JSON；它们可指向同一事件。元数据未设置、无事件、或元数据不可验证时分别明确说明，不因可选字段缺失失败。

所有查询只读：不修改 FD、FEATURE_INDEX、workspace.json、receipts 或 evidence；不新增依赖，不 checkout/切换分支，也不执行网络 Git 操作。

## Scope

- 只增加 FD 查询/展示能力，不改 handoff 状态、receipt schema、报告内容或归档规则。
- 保持 `aiw fd show <FD-ID>` 现有 FD 正文展示兼容，并补充结构化状态信息。
- 覆盖活动和已归档 FD 的报告/review 查找；跨父分支与 worktree 的读取应为只读操作。
- 不新增依赖，不创建 OpenSpec change。
- PM 根据用户明确指示豁免本 FD 的 Tester 阶段。旧 Tester 收据与决策见 [FD-038-test-waiver-r1.md](reports/FD-038-test-waiver-r1.md)；当前状态进入独立 Reviewer 复核。

## Work items

拆分映射：旧 1.1 拆为新 1.1–1.4；旧 1.2 对应新 1.5；旧 1.3 对应新 1.6。以下 ID 在 implementation-ready 后保持稳定。

- [x] 1.1 实现经验证的 evidence 来源解析器。 Size: M; Difficulty: Medium; Dependencies: none. Completion: 解析当前 checkout/HEAD、workspace.json 中记录的 worktree、branch、parent_branch 及 FD archive；验证 branch/worktree 对应关系和 repo 内路径；branch-only 内容通过 Git tree 读取，不 checkout；外部 symlink、无效来源给出可见诊断并跳过，不按 branch 名拼接文件系统路径。
- [x] 1.2 实现共享 evidence inventory、时间和去重。 Size: M; Difficulty: Medium; Dependencies: 1.1. Completion: 仅汇集当前 FD ID 前缀下的 reports/reviews Markdown 与同名 JSON 路径；工作树使用文件 mtime、branch-only 文件使用最近改动提交时间，统一显示 UTC；按 evidence kind 与内容摘要合并相同副本并保留全部来源；内容不同的版本分别列出。
- [x] 1.3 实现只读 evidence 列表与选择器。 Size: M; Difficulty: Medium; Dependencies: 1.2. Completion: 最新优先且最多 20 项；`--last` 不读 stdin；只有 stdin/stdout 都是终端才读取编号选择；空输入或 `q` 安全取消；非交互输出列表及用法提示后退出；无匹配给出清楚空结果。
- [x] 1.4 增加 `show-report` 和 `show-review` 命令。 Size: S; Difficulty: Low; Dependencies: 1.3. Completion: argparse 路由、插件 META 命令清单和帮助一致；两命令共享 inventory/selector，列表含时间、来源及可用 sidecar 路径，选择后只输出对应 Markdown 正文。
- [x] 1.5 扩展 `fd show` 状态摘要。 Size: S; Difficulty: Low; Dependencies: none. Completion: 保留旧 FD 正文及 receipt 输出，增加 status、经验证的 worktree/branch、当前 handoff 和 latest event；无当前 handoff、无 event、元数据未设置/不可验证均有明确表示。
- [x] 1.6 更新 CLI 用法文档与验收场景索引。 Size: S; Difficulty: Low; Dependencies: 1.1–1.5. Completion: 用法说明三个命令、来源验证、时间排序、交互/非交互与只读行为；Verification 列出各可观察场景，供独立复核和后续经授权的验证参考。

## Acceptance

- `aiw fd show-report FD-XXX [--last]` 和 `aiw fd show-review FD-XXX [--last]` 都接受有效 FD ID；`--last` 显示可验证来源中时间最新的一项，不读取 stdin。
- evidence 来源覆盖当前 checkout/HEAD、经 `git worktree list --porcelain` 验证的 FD worktree、经 `git check-ref-format` 与本地 ref 验证的记录 branch/parent_branch，以及该 FD 的 archive reports/reviews。路径不由 branch 名拼接；无效元数据来源有诊断并被跳过。
- 列表按时间倒序最多显示 20 项，每项显示 UTC 时间、Markdown 来源路径及存在时的同名 JSON sidecar 路径；在交互终端中可按编号显示 Markdown 正文。
- 仅在 stdin 和 stdout 都是终端时读取选择。空输入或 `q` 取消并安全退出；其他非交互环境输出列表和可操作提示后退出，不阻塞等待 stdin。
- 无匹配 evidence 时给出清楚的空结果。相同 evidence kind 且内容相同的副本去重并合并来源路径；相同路径但内容不同的版本分别保留。
- `aiw fd show FD-XXX` 保持原有 FD 正文和 latest receipt 输出，并显示 status、已验证的 worktree 路径及 branch。当前 handoff 只指 pending/launching/dispatched receipt；没有时明确显示无当前 handoff。Latest event 始终显示最新 receipt；两者显示时间和格式化 JSON，允许指向同一事件。
- workspace metadata 未设置、没有 event、没有当前 handoff，或元数据无法验证时，都给出明确状态，不因可选字段缺失而失败。
- 三个查询命令只读；FD 文件、索引、workspace 元数据、receipts 和 evidence 均不被修改。

## Verification

- 静态检查 Python argparse/META/dispatch 路由，stdout/stderr 与错误路径，workspace.json 和 Git ref/worktree 验证，branch-only 只读读取，证据排序/去重及用法帮助一致性。
- Worker 在实现后运行一个仓库允许的 Python compile-only 命令；不运行最终构建、测试、formatter、linter 或依赖下载。
- Reviewer 独立核对各验收场景对应的实现路径和静态证据，并标明哪些行为因本次豁免而没有运行时证据：show 有效状态与元数据；缺失元数据/event；当前 handoff 与 latest event；两个命令各自的 `--last`；超过 20 项的排序、时间和来源；当前分支、FD worktree、parent branch 与 archive 发现；相同内容副本去重及不同内容版本保留；选择显示 Markdown 与 sidecar 路径；取消；非交互不阻塞；空结果；查询只读。
- 本 FD 的 Tester 阶段已按 [PM 豁免决策](reports/FD-038-test-waiver-r1.md)跳过。此前一次旧流程命令结果为 8 个用例中 7 个失败、1 个通过，且 PATH 使用了不含 worktree 新命令的旧 `aiw.exe`；该结果没有验证目标实现，也不是测试通过。本次不会重跑测试或生成 `test-accepted` 事件。
- `python -c "from pathlib import Path; compile(Path('plugins/aiw-fd.py').read_text(encoding='utf-8'), 'plugins/aiw-fd.py', 'exec')"` compile-only 已通过（首次因 `workspace_info` 缩进错误失败，修正后同命令重跑通过）；未运行测试。

- Worker 静态审阅已覆盖 Work Items 1.1–1.3 的来源校验、Git tree 只读枚举、去重、UTC 排序、最多 20 项、TTY 判断、取消及 `--last` 不读 stdin；尚未运行测试。
- Worker 静态审阅已核对 `show-report`/`show-review` 的 argparse、插件命令清单及 dispatch 均接入共享 selector；尚未运行测试。
- Worker 静态审阅已核对 `fd show` 保留原正文/receipt，并追加 status、已验证 workspace、当前 handoff 与最新 event；无 event、无活动 handoff、未设置/无效 workspace 均有显式状态。尚未运行测试。
- Worker 根据 [首轮 Reviewer 报告](reviews/FD-038-review-r1.md)修复了解析后 worktree 路径未限定在仓库根目录的问题，并更正 Sources 中关于稳定 spec 的描述。修复报告：[FD-038-implementation-r2.md](reports/FD-038-implementation-r2.md)。修复后 Python compile-only 已通过；测试按 PM 豁免未运行。

- 第二轮 Reviewer 已静态确认首轮 F-001 与 F-002 修复，并通过 Verification；详见 [第二轮独立审查报告](reviews/FD-038-review-r2.md)。本次测试豁免不代表未运行的行为已通过。
- F-001 的解析后仓库根包含性检查和 F-002 的 Sources 更正均已复核。CLI 路径拒绝分支、跨来源查询、终端交互和只读性仍无运行时证据。

## PM 决策

- 用户明确要求跳过测试并修正状态；本 FD 的 Tester 阶段豁免，策略设为 `Waived`。不创建 `test-accepted`，不把此前的失败运行描述为通过。
- PM 已授权一次性手工结束旧的已派发 Tester 收据，保留其事件与 session 历史，再请求独立 Reviewer。完整依据与剩余风险见 [FD-038-test-waiver-r1.md](reports/FD-038-test-waiver-r1.md)。

## TODO

- [x] 1.1 来源解析器：验证 workspace.json 与已注册 worktree、branch refs 和 repo 内路径；拒绝外部链接。Reviewer 第二轮已静态确认 F-001 修复；Worker compile-only 报告通过，未运行测试。
- [x] 1.2 inventory：按 FD ID 枚举 reports/reviews 和 archive，合并规范化内容相同的副本，保留来源与 sidecar 路径。静态审阅已完成；最终 Python compile-only 已通过，未运行测试。
- [x] 1.3 只读列表和交互/非交互选择器。
- [x] 1.4 show-report/show-review 命令和帮助。
- [x] 1.5 fd show 状态摘要。
- [x] 1.6 用法文档与场景索引已完成；两项 Reviewer findings 已修复并通过第二轮复核。PM 已豁免 Tester；运行时行为仍未验证。

## Sources

- Issue: none
- `plugins/aiw-fd.py`：当前 show/receipt 读取与 argparse 路由；evidence handoff 验证。
- `plugins/aiw-git/git-wt.py` 与 `tests/test_fd027_wt_blackbox.py`：workspace.json 字段、worktree/ref 验证契约（仅静态参考）。
- `docs/usage/aiw-fd.md`、`skills/work-management.md`、`docs/templates/`：CLI 使用说明、evidence 活动/归档路径、Dual evidence 格式。
- `openspec/specs/fd-workflow/spec.md` 与 `openspec/specs/command-help-consistency/spec.md`：FD 查询/归档和公开命令帮助要求；本次实现遵循这些既有稳定规范，未修改稳定 spec。
- `.ai/fd/<FD-ID>/workspace.json`：现有 worktree、branch、parent_branch 元数据契约。
