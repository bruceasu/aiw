# FD-063: 默认自动交付并归档符合条件的 FD 实现

**Status:** Open
**Revision:** 3
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户确认符合条件时应自动执行 local-merge 和归档，不应在实现审查通过后要求再次输入 auto。当前 implement 结束于 Reviewer 交接，共享契约和 fd-workflow 又将完整交付授权限定于显式 auto/全生命周期请求，导致重复操作。

## Options and decision

选择将编号 FD 实现请求默认视为该 FD 的本地完整生命周期授权，由宿主 PM 在独立审查通过及交付条件满足后继续。只加一句自动合并不能解除原有授权冲突；修改 CLI 或自动绕过 Gate 超出需求，均不采用。显式阶段限制与更窄的用户指令优先，auto 保留为兼容入口。

## Solution

统一 root AGENTS、共享契约、implement、fd-workflow、相关角色及 fd-review 的宿主交付约定，更新 README、基础 Agent 模板和稳定规格，同步项目副本与所有已跟踪共享契约生成副本。默认授权仅覆盖当前已确认 FD 的准备计划、本地聚焦提交、独立审查、记录父分支上的 squash 交付、校验清理及成功后的归档提交，不覆盖 push、release、部署、测试、构建、下载或提权。

交付前检查当前通过事件、FD revision/digest、审查后实现未漂移、全部范围完成、无未解决 blocker、双方工作区干净及登记坐标一致。任何实质阻塞或合并冲突均保留现场并报告，不自动扩大范围、豁免检查或重复失败操作。CLI 自身冲突恢复行为不变，宿主自动流程在冲突后暂停，用户另行明确处理后方可恢复。

独立 Reviewer 只产生报告和事件，Worker 只处理其角色；宿主 PM 继续收尾，不接管已派发角色。用户明确只规划/只实现/只审查/暂不合并或只做部分工作项时，按其范围结束；未完成整个 FD 不得交付归档。显式主工作区实现同样遵从已授权 FD 范围、独立审查及聚焦提交，跳过不适用的 worktree 合并。

## Scope

仅修改工作流 Markdown/YAML、Agent 规则、稳定规格和项目副本，不改 CLI/调度实现、状态/receipt/schema、依赖、CI、部署或历史归档。显式确认也授权本 FD 在满足条件后自动交付并归档。共享契约保持 canonical 单一所有者，已跟踪生成副本统一刷新。

## Work items

- [ ] 1.1 统一共享授权和 root/template Agent 规则。规模小、难度中，无依赖；明确默认 FD 授权、阶段例外、执行权限和交付 Gate。
- [ ] 1.2 衔接实现、独立审查、PM 交付和归档流程，更新说明/规格/项目副本。规模小、难度中，依赖 1.1；无默认提前结束或相互冲突的旧授权，角色隔离与冲突停止保留。
- [ ] 1.3 一次静态一致性检查，双格式证据并独立审查，按本轮授权交付归档。规模小、难度低，依赖 1.2；如实记录静态证据及未执行检查。

## Acceptance

- 一般编号 FD 实现请求不再需要第二次 auto；通过独立审查且 Gate 满足后宿主自动 local-merge、close 和归档提交。
- 显式阶段限制、未完成 FD、过期/不匹配审查、脏工作区、未知在途角色、冲突或未解决 blocker 都阻止受影响的自动收尾。
- 不新增测试/构建/网络/push/部署权限；既有独立审查、真实证据和交付校验不被省略。
- 主工作区模式不伪造 worktree 或 Worker receipt；隔离模式只交付登记父分支，由 CLI 校验清理。
- canonical/项目指引及共享副本一致；稳定规格反映默认授权和实际先 local-merge 清理再归档的顺序。

## TODO

- 完成三个 Work Items、独立审查和符合条件的自动本地交付/归档。

## Verification

- 计划一次静态 diff/授权/角色及 Gate 路径一致性检查，独立 Reviewer 检查最终版本。
- 纯 Markdown/YAML，无适用编译目标；不运行编译、测试、效果评估、验证脚本、构建、网络或安装同步。
- 尚未实施或验证。

## Sources

- Issue: none

- 本轮用户明确确认符合条件时应自动操作，包括当前 FD。
- AGENTS.md、src/agent-templates/AGENTS.md、src/skills/work-management.md。
- src/skills/implement、fd-workflow、fd-review 与相关角色。
- openspec/specs/fd-workflow/spec.md。
