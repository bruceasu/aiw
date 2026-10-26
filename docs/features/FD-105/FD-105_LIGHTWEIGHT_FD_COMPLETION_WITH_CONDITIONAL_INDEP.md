# FD-105: Lightweight FD completion with conditional independent review

**Status:** Complete
**Revision:** 7
**Priority:** Medium
**Review policy:** Independent
**Evidence policy:** Markdown

## Problem

普通 FD 强制独立审查、逐项提交和重复报告，增加了小改动的执行成本。用户已确认默认自查后交付，在高风险、影响范围不清或明确要求 review 时安排独立审查。

## Options and decision

采用统一的轻量默认流程；保留需要时使用的独立审查和角色 runner 协议。只改提示词会与 CLI 的强制审查归档门槛冲突，因此同时修改共享合同、模板、CLI 和稳定规格。已有 FD 没有 Review policy 时继续要求独立审查，避免静默放宽历史约束。

## Solution

新 FD 使用 Self 审查策略和 Markdown 证据；默认由主会话负责计划、实现、自查和收尾，无需逐阶段事件。runner 或显式 handoff 仍使用真实事件和会话归属。Independent FD 仍须当前有效的独立通过证据。自查完成不生成 Reviewer 通过事件。

## Scope

修改 fd-workflow、共享工作合同、根指令中相关约定、FD 模板、aiw-fd.py 与 fd-workflow 稳定规格。保留既有修改、历史报告和旧 Tester 协议。用户明确授权在主工作区修改，不自动提交；不运行测试、网络、打包或发布。

## Work items

- [x] 1.1 精简技能、共享合同和模板，明确自查与条件独立审查、记录位置及按变更提交。
- [x] 1.2 调整 CLI 默认创建、事件路由、独立审查入口和归档门槛，保留旧 FD 与 runner 兼容性。
- [x] 1.3 同步稳定规格，完成 Python 内存编译和静态检查，记录实际结果。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- 新 FD 默认不创建逐阶段交接；显式 handoff 或 runner 保留原交接机制。
- Self FD 完成全部 Work Items 并记录 Verification 后，可以正常归档，不使用 force 或伪造独立审查。
- Independent、旧版无策略及 legacy Independent Tester FD 保留当前有效独立通过门槛；未完成事项、未决问题和运行中角色仍阻塞归档。
- 独立审查只在交付前按风险或用户要求触发，复审聚焦问题与修复影响；普通记录集中在 FD。

## TODO

- 独立静态审查及针对发现的限定复查已完成；按用户授权范围不自动提交。本次在主工作区保留修改，不合并或归档。

## Verification

- 本次修改跨共享合同和 CLI，使用独立静态审查；不额外增加设计审查轮次。
- 已运行两次 Python 内存 compile 检查，无 pyc 或可分发产物；两次均通过，第二次覆盖审查修复后的最终 Python 源码。命令：`python -c "from pathlib import Path; p=Path('src/plugins/aiw-fd.py'); compile(p.read_text(encoding='utf-8'), str(p), 'exec'); print('Python compile-only passed: src/plugins/aiw-fd.py (no artifact)')"`。
- 已运行 `git diff -- src/plugins/aiw-fd.py`，静态检查新建、事件路由、直接请求审查和归档策略分支；未把这些静态结论当作运行测试结果。
- 独立静态审查第 1 轮要求修改：Worker 交接刷新可绕过既有复审要求，稳定规格/恢复说明仍有无条件审查和双格式反馈规则。报告：`reviews/FD-105-review-r1.md`；来源 `FD-105-000004-implementation-ready`。修复后只复查发现和影响范围。
- 已修复：有效审查策略读取 FD 及全部已有 Reviewer 请求/结果；refresh-worker/recover-worker 不再清除独立审查约束。普通路由、归档和归档后审查入口统一使用该策略。
- 已修复：稳定规格中的阻塞、自动生命周期、恢复、重新审查和设计反馈段落，以及 delivery 恢复尾部，与 Self/条件独立审查规则对齐。
- 限定独立复查通过：`reviews/FD-105-review-r2.md`，来源 `FD-105-000006-implementation-ready`。仅复查首轮发现、修复及其影响，未新增完整审查范围；未声称运行命令链已测试。
- 未运行测试、构建、格式化、验证脚本和网络命令。

## Sources

- Issue: none

- 用户在当前会话确认轻量默认流程和条件独立审查，并允许主工作区修改且不自动提交。
- openspec/specs/fd-workflow/spec.md
