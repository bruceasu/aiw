# FD-038: Improve FD report, review, and status inspection

**Status:** Planned  
**Revision:** 2  
**Priority:** Medium
**Test policy:** Independent
**Evidence policy:** Dual

## Problem

在 FD 流程中查找 report、review 和当前运行状态需要手动遍历文件与事件记录。`aiw fd show <FD-ID>` 当前显示 FD 正文及最后 handoff，但没有 worktree/branch 信息、最后事件详情或方便选择历史 evidence 的入口。FD 归档后，报告和 review 还会移动到 FD 专属归档目录；同一 FD 也可能分别存在于父分支和 worktree 分支，因此只查当前工作目录会漏掉用户需要的结果。

## Options and decision

| 方案 | 判断 |
| --- | --- |
| 扩展 `fd show` 并新增 `show-report`、`show-review` 子命令 | 采用。职责清楚，保留已有 `show` 用法，并为报告/审查列表提供独立交互入口。 |
| 只增强 `fd show`，把所有 evidence 塞入一个视图 | 不采用。报告和 review 的查找/打开属于独立操作，混合输出会使状态概览冗长。 |

Evidence 搜索应覆盖当前分支、FD 记录的 worktree 分支以及父分支可见的活动/归档目录；实现前确认现有 Git/worktree 元数据和路径解析方式，避免仅凭分支名拼接路径。默认交互列表按文件时间倒序，最多 20 项；`--last` 直接打开最新项。

## Solution

新增 `aiw fd show-report <FD-ID> [--last]` 与 `aiw fd show-review <FD-ID> [--last]`。无 `--last` 时展示按时间倒序、最多 20 条的可选择列表；选择后输出对应 Markdown 内容（若有同名 JSON sidecar，可一并提示/展示路径）。无 evidence 时给出清楚提示。`--last` 直接展示最新项，不弹出列表。

扩展 `aiw fd show <FD-ID>` 的概览，包含 FD 状态、worktree 路径、branch、最后 handoff 时间及格式化 JSON、最后 event 时间及格式化 JSON。Worktree/branch 未记录时明确显示未设置；没有 handoff/event 时明确显示无记录。跨分支搜索以 FD 元数据中的 `worktree`、`branch`、`parent_branch` 为边界，并包含活动 evidence 目录与 `docs/features/archive/<FD-ID>/reports|reviews/`。同一证据在多个可见位置应去重；列表明确展示时间和来源路径。

CLI 应继续支持非交互终端：`--last` 可直接使用；无 `--last` 且 stdin/stdout 非终端时，输出可读列表与选择用法提示，不阻塞等待输入。选择取消安全退出，不修改 FD、receipts 或 evidence。

## Scope

- 只增加 FD 查询/展示能力，不改 handoff 状态、receipt schema、报告内容或归档规则。
- 保持 `aiw fd show <FD-ID>` 现有 FD 正文展示兼容，并补充结构化状态信息。
- 覆盖活动和已归档 FD 的报告/review 查找；跨父分支与 worktree 的读取应为只读操作。
- 不新增依赖，不创建 OpenSpec change。

## Work items

- [ ] 1.1 定义并实现 report/review evidence 的跨当前目录、worktree 和父分支发现及时间倒序选择行为。 Size: M; Difficulty: Medium; Dependencies: none. Completion: 两个新命令支持 `--last`、最多 20 条的交互列表、非交互提示、空列表和归档路径，并不修改 evidence。
- [ ] 1.2 扩展 `fd show` 的状态概览与格式化 handoff/event 输出。 Size: S; Difficulty: Low; Dependencies: none. Completion: 输出状态、worktree、branch、最后 handoff 时间与 JSON、最后 event 时间与 JSON；缺少可选数据时有明确表示且旧正文仍可读。
- [ ] 1.3 更新 CLI 用法文档和聚焦验证证据。 Size: S; Difficulty: Low; Dependencies: 1.1, 1.2. Completion: 帮助/使用文档说明三个命令、跨分支来源、交互与非交互规则；独立 Tester 按仓库授权规则报告可观察场景。

## Acceptance

- `aiw fd show-report FD-XXX --last` 与 `aiw fd show-review FD-XXX --last` 分别展示可见 evidence 中最新的一项。
- 不带 `--last` 时按时间倒序列出最多 20 项，用户能选择其中一项查看；列表列明时间及来源路径。
- 搜索能够找到 FD 当前分支、记录的 worktree/父分支及该 FD 归档目录中的 evidence，并对重复路径去重。
- 无匹配 evidence 时给出清楚的空结果；非交互环境不等待 stdin，提供可操作的提示。
- `aiw fd show FD-XXX` 显示 status、worktree、branch、最后 handoff 和最后 event 的时间及格式化 JSON；可选字段缺失时不报错。
- 查询命令只读，不变更 FD 文件、索引、workspace 元数据、receipts 或 evidence。

## Verification

- 静态检查 Go/Python 命令路由、输出与错误处理、worktree 元数据读取、证据路径搜索/去重及文档帮助的一致性。
- 编译仅使用仓库规定的 compile-only 命令；不运行测试，除非 Tester 按仓库规则获得精确命令授权。
- 独立 Tester 覆盖有/无 evidence、`--last`、选择列表上限/排序、取消、非交互、归档、worktree/父分支发现、重复项、缺少 handoff/event 和只读性场景。
- 当前未运行验证。

## TODO

- [ ] 实现两个 evidence 浏览子命令及确定安全的跨 worktree/父分支发现方式。
- [ ] 扩展 `fd show` 概览并更新用法文档。
- [ ] 按仓库验证策略完成静态审查、compile-only 检查和授权后的独立测试。

## Sources

- Issue: none
- `plugins/aiw-fd.py`：当前 `fd show` 仅展示 FD 正文及最后 handoff；CLI 参数由 argparse 子命令定义。
- `skills/work-management.md` 与 `docs/features/TEMPLATE.md`：FD/evidence 活动与归档布局、Dual evidence 约定。
- `.ai/fd/<FD-ID>/workspace.json`：现有 worktree、branch、parent_branch 元数据契约。
