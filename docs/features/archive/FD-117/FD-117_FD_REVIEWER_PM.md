# FD-117: 明确 FD Reviewer 通过后的 PM 继续步骤

**Status:** Complete
**Revision:** 4
**Priority:** Medium
**Review policy:** Independent
**Evidence policy:** Markdown

## Problem

在 Reviewer 通过后，AIW 会生成目标角色为 PM 的待处理回执。Agent 不能 claim 该回执，但当前错误容易让助手把角色边界误解为需要用户接收；`close Complete` 的通用错误也没有指出 Reviewer 结果缺失或已因 FD revision/content digest 变化而失效。

## Solution

保留 PM/human 事件不能由 Agent claim 的限制，并在 PM claim 错误中说明由宿主 PM 继续处理的命令和用户介入边界。细化 Complete 归档失败信息，区分缺少 Reviewer 通过、Reviewer revision 过期和同 revision 的 FD digest 不匹配，并给出重新审查或继续归档的下一步。

## Scope

只调整 `aiw fd claim` 与 `aiw fd close <id> Complete` 的错误说明和所需的静态/黑盒验收依据。保留角色权限、Reviewer 状态/revision/digest 门槛、事件路由、force 行为及 FD 生命周期不变；不增加 Plane 或人工接收步骤。

## Work items

- [x] 1.1 更新 PM claim 拒绝信息，说明宿主 PM 的正常继续命令，且不允许 Agent claim。
- [x] 1.2 细化 Complete 归档的 Reviewer 缺失、revision 不匹配和 digest 不匹配错误及对应下一步。
- [x] 1.3 静态检查变更路径、既有门槛和 Issue 验收覆盖；记录编译结果及未执行的检查。

## Acceptance

1. Agent claim PM 回执仍被拒绝，错误说明这是宿主 PM 工作流，并指向 `aiw fd close <id> Complete`；不会提示用户接收该回执。
2. 无当前 Reviewer 通过结果时，Complete 失败说明缺少的门槛及需要的 Reviewer 动作。
3. Reviewer event 的 revision 与当前 FD 不同，或 revision 相同但 FD digest 不同，Complete 分别说明真实失效原因并要求当前 FD 重新审查；错误文案不弱化校验或暗示使用 force。
4. 当前 Reviewer 通过结果有效时，原有 Complete 归档与 PM 回执确认行为保持不变。

## Verification

- 静态检查 `claim()`、`close_locked()`、事件字段、当前 diff 和 `src/skills/work-management.md`；确认角色拒绝、Reviewer 状态/revision/digest 门槛未放宽。
- `python -c "from pathlib import Path; p=Path('src/plugins/aiw-fd.py'); compile(p.read_text(encoding='utf-8'), str(p), 'exec'); print('Python syntax compile passed')"` — 通过。
- 未运行测试、构建、lint 或验证脚本；本次没有测试执行授权。
- Review policy 为 Independent：改动属于公开 CLI 错误行为；交付前需要当前实现 revision 的独立 Reviewer 通过。
- 风险：调用方若匹配错误文本，文案变化可能影响自动化；权限与完成门槛本身保持不变。
- Independent review：PASS；报告见 `docs/features/FD-117/reviews/FD-117-review-20261115.md`，来源事件 `FD-117-000003-review-requested`。

## TODO

实现和自检已完成；待当前 revision 的独立 Reviewer 审查通过后归档、提交并执行 FD-117 worktree 的本地 squash 交付与验证清理。

## Sources

- Issue: ISSUE-00014

无受影响的稳定 OpenSpec spec；Issue 未请求创建 OpenSpec change。

**Completed:** 2026-11-15
