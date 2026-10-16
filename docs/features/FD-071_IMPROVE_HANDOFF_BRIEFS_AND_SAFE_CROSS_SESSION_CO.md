# FD-071: Improve handoff briefs and safe cross-session continuation

**Status:** Open
**Revision:** 2
**Priority:** Medium
**Evidence policy:** Dual

## Problem

handoff 缺少明确版本锚点、跨会话授权与预算、接手核查和未知结果恢复规则；辅助会话与多课程教学也缺少身份说明。接手者可能在错误工作区重复执行已完成操作或重置预算。

## Options and decision

- 保持现状：依赖作者自由选择，难以可靠继续，不采用。
- 新增状态机、强制 JSON 或 CLI：超出轻量文档需求，不采用。
- 强化现有独立 Skill：采用，默认简短，跨 Agent/环境时按需补充，引用权威资料和已有恢复机制。

## Solution

只更新 handoff 的 SKILL.md、README.md 及项目副本，保持显式触发/无自动派发。最小结构包含目标/决定、状态/下一步、授权/预算、阻断/证据。适用且可获得时记录 cwd、branch、HEAD、dirty scope 和采集时间。分开已完成、进行中、结果未知，未知先核查。

工程交接条件包含 FD、当前事件/领取者/Work Items；普通讨论写作学习无需 FD。辅助交接引用模式、Session/课程身份、目标/摘要、产物、进度和 pending 状态；teach 补教学位置、反馈与下一课。未知标明，不自行放大权限、抢锁或重放模型。

默认约一页；用户请求跨 Agent/环境时补必要工具、路径映射及未迁移资料，不复制秘密或自动安装。交接是快照，接手先读当前规则和原授权证据，核查身份、权威状态与未知结果。写出交接文件是有限输出权限，不改变业务或工作流状态。不新增脚本、模板或机器状态。

## Scope

包含源 Skill/README、项目副本、双格式证据和独立实现评审。不改产品命令、FD 状态或 Session 持久化，故无需 stable spec 变动。不涉及自动派发、运行模型、全局安装、CLI/API/依赖/并发或其它 Skill 重构。

## Work items

- [ ] 1.1 优化交接与接手指令。Size: S; Difficulty: Low; Dependencies: none. Completion: 两种深度、版本、授权预算、结果核查、条件FD/辅助/课程字段明确，旧触发边界保持。
- [ ] 1.2 更新说明并同步项目副本。Size: S; Difficulty: Low; Dependencies: 1.1. Completion: README 用途/示例/边界一致，两组文件字节相同，无全局安装。
- [ ] 1.3 静态证据、独立评审及交付归档。Size: S; Difficulty: Low; Dependencies: 1.2. Completion: 有界静态情景核查与报告、独立评审，过门槛后本地合并归档。

## Acceptance

- 默认简报约一页，链接已有材料，Suggested skills 仅列下一步需要的方法；无FD可用。
- 适用身份包含 cwd/branch/HEAD/未提交范围、事件/领取者；不可获得值标unknown，不虚构或将未提交内容当成commit。
- 授权来源、限制、消耗预算/诊断状态可继续，接手不重置次数或放大权限。
- 接手核查规则/版本/事件/所有者，过期资料与当前事实协调；模型/提交/合并/派发未知结果先查而非重放。
- 辅助交接不强加Issue/FD；课程可定位、记录反馈/下一步而不宣称掌握，不混课，pending使用现有恢复机制。
- 唯一文件路径明确、排除秘密、扩展交接不自动迁移安装，原显式触发及无自动派发不变。

## Verification

- 仅Markdown，不编译无关二进制，不运行测试/模型/网络/格式化/lint/发行构建。
- 一批只读检查两组副本、相对链接及diff；静态情景：无FD、FD领取中、未知模型结果、课程中断、过期摘要、预算耗尽。
- 独立Reviewer核查真实实现。文档改动无公开接口/持久化设计，不额外设计评审。

## TODO

- [ ] 完成1.1–1.3及独立评审。
- [ ] 本地交付与归档。

## Sources

- Issue: none
- 用户确认本会话五项优化及辅助课程/两种深度建议。
- src/skills/handoff/{SKILL.md,README.md,agents/openai.yaml}。
- src/skills/work-management.md、reviewed-skill-contract.md；FD-070 辅助Session与使用指南。
