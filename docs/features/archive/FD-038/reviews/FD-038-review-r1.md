<!-- aiw-data: FD-038-review-r1.json -->

# FD-038 独立审查报告

## 评审结论

**结论：changes-requested。** 本次评审对应 `FD-038-000006-review-requested`，Reviewer session 为 `fd038-reviewer-20261009-codex-review-01`。实现覆盖了主要查询与展示路径，但来源验证仍有一处不满足 FD 约束的问题，需要修正后再复核。

## 发现

### F-001（中）：验证 worktree 时没有限制解析后的路径仍位于仓库内

- 文件：`plugins/aiw-fd.py:307-320`，尤其是 312-315 行。
- `workspace_info` 对 `primary/.wt/<FD-ID>` 调用 `resolve()`，但只检查 metadata 路径解析结果与它相等，并检查 Git worktree 注册和 branch。它没有检查解析后的 `expected_worktree` 是否仍位于 `primary` 下。
- 因此，若 `.wt/<FD-ID>` 通过外部目录重定向，且 metadata 与 Git worktree 注册指向同一解析路径，`workspace_info` 仍会返回该路径，`fd show` 会标为 `Verified worktree`。后续 evidence scanner 会拒绝仓库外路径，但这不能纠正 `fd show` 对该 workspace 的错误信任状态。
- 这未满足 Work Item 1.1 和验收条件中“验证 repo 内路径、无效来源给出诊断并跳过”的要求。
- 修正：在接受 workspace 元数据前，对解析后的 worktree 路径执行仓库根目录包含性检查；对越界路径返回不可用状态并给出诊断。保留现有 Git worktree/ref 校验。

### F-002（低）：FD Sources 对稳定 spec 改动的表述不准确

- 文件：`docs/features/FD-038_IMPROVE_FD_REPORT_REVIEW_AND_STATUS_INSPECTION.md:96`。
- 本次 `develop...HEAD` diff 没有修改 `openspec/specs/fd-workflow/spec.md`；对应只读状态/evidence 要求已存在于 `develop`。
- 修正：删除“本 FD 更新了稳定 FD workflow 规范”的表述，或明确记录确有需要的 spec 变更。

## 验收覆盖与证据

- Work Item 1.1：部分满足。metadata、注册 worktree 和本地 branch ref 有静态校验；F-001 所述的解析后路径边界缺失。
- Work Items 1.2–1.3：静态路径显示工作树与 Git tree 来源共用 inventory；Markdown 按 kind 和规范化内容摘要去重；按时间倒序并截取 20 项；TTY 双重判断、非交互提示、取消和 `--last` 路径均在代码中实现。未运行时验证这些行为。
- Work Item 1.4：静态核对了 argparse 路由、插件 `META` 命令清单和共享 selector 的 dispatch。
- Work Item 1.5：静态核对了原 FD 正文和 latest receipt 输出后的状态摘要、workspace 状态、当前 handoff 与 latest event。F-001 影响 workspace 是否能被正确标为已验证。
- Work Item 1.6：使用文档列出了三个命令、来源、排序、TTY 行为和只读约束。
- Worker 报告记载 Python compile-only 命令在修正缩进后通过。Reviewer 未重跑 compile-only，也未运行测试或构建。PM 已明确豁免本次 Tester 阶段；没有 `test-accepted`，也没有把此前旧命令的失败结果当作通过证据。
- 此次 diff 没有修改 `openspec/specs/fd-workflow/spec.md`；该规格中的只读状态/evidence 要求已存在于 `develop`。FD Sources 中“本 FD 更新了稳定 FD workflow 规范”的说法与本次 diff 不符，应一并更正。

## 实际执行与未执行

- 领取评审：`python plugins/aiw-fd.py claim FD-038 FD-038-000006-review-requested --session fd038-reviewer-20261009-codex-review-01`。
- 发送结论：`python plugins/aiw-fd.py emit FD-038 changes-requested --producer reviewer --artifact docs/features/reviews/FD-038-review-r1.md --source-event FD-038-000006-review-requested`。首次调用因 sidecar 重复 JSON key 被拒；删除重复字段后，用相同命令重试成功，创建 `FD-038-000007-changes-requested`。
- 静态检查变更范围：`git diff --name-status develop...HEAD`、`git diff --unified=15 develop...HEAD -- plugins/aiw-fd.py`、`git diff --unified=8 develop...HEAD -- docs/usage/aiw-fd.md`。
- 静态检查 spec 变更：`git diff --unified=3 develop...HEAD -- openspec/specs/fd-workflow/spec.md`，无 diff；`develop` 已含相关只读状态/evidence 要求。
- 静态空白检查：`git diff --check develop...HEAD`，无输出。更新评审记录后，`git diff --check` exit 0，仅提示 FD 与索引文件的 LF/CRLF 转换。
- 未运行测试、compile-only、最终构建、formatter 或 linter。

## 剩余风险

由于用户决定跳过测试，CLI 运行时行为仍未验证，包括跨来源读取、排序与去重、交互选择、非交互不阻塞以及查询只读性。当前发现来自静态路径追踪，不应描述为运行时复现。
