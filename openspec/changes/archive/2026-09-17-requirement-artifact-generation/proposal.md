# 基于已批准需求生成 OpenSpec 工件

## Why

当前推广已批准 Requirement 时，proposal、design 和 spec 的正文仍描述 AIW 的工件生成机制，capability 被固定为 requirement-management。Task 创建还可能预先写入通用 tasks.md，使后续 WriteMissing 跳过真正的需求清单。文件存在和结构合法不能证明实际需求已进入工程规划。

本 Change 来源于当前会话及 `.ai/issue/2026-09-17-aiw-promotion-generic-openspec-artifacts.md`。它独立于 workflow-automation 的业务实现，不重新推广或改写该 Task。requirement-management Skill 的事后修复是交互式补救，尚未构成 CLI 可恢复的生成流程。

## What Changes

- 将批准的 Plan、捕获正文、已确认会话结论和修订组装为有来源与版本的生成输入；不把未保存聊天或未确认建议当作批准需求。
- 在 promotion 和 prepare-spec 的共享路径生成实际功能的 proposal、design、delta specs 和 tasks，支持多个新增或修改的 capability。
- 支持 provider 无关的生成契约；备用选择使用完整 provider/model/endpoint 配置，记录每次调用结果。
- 无配置、无可用模型或调用失败时输出可恢复交接文件和路径，由当前会话 Agent 接续；无人接续时保持未完成。
- 对模型输出和会话 Agent 输出使用同一套来源、结构、覆盖、路径与冲突检查，再写入正式工件。
- 保留人工内容和清单身份；仅替换有充分证据的生成占位内容。中断恢复复用 Task 和生成请求。
- 分别报告结构检查、需求覆盖、Design Readiness 与实现验收；不以模板或仅有合法 JSON 标记 SPEC_DRAFTED。

## Capabilities

### New Capabilities

- `requirement-artifact-generation`: 需求上下文快照、模型生成与交接 fallback、候选验证、安全写入和恢复。

### Modified Capabilities

- `requirement-management`: 推广完成以实际需求工件被接受为前提，保留批准及同一 Task 恢复规则。

## Impact

影响 Requirement 推广与 prepare-spec、共享 OpenSpec 生成器、生成请求记录、模型配置解析和受管清单同步。AIW CLI 与 requirement-management / to-spec Skill 的接续说明需要同步。新增运行记录放在已有 Task 的运行目录，正式工件仍归 OpenSpec 所有。

## User Stories

1. 作为需求负责人，我希望工件表达批准的业务功能，避免标题正确但正文跑题。
2. 作为审阅者，我希望每项业务规则能追溯到批准来源及其修订。
3. 作为工程师，我希望一个需求能修改多个实际 capability。
4. 作为不同模型的使用者，我希望生成契约保持一致，且知道实际使用了哪个 provider/model。
5. 作为没有配置 AIW 模型的用户，我希望当前会话 Agent 能接手续写，无须再次批准同一范围。
6. 作为纯 CLI 用户，我希望无法生成时得到明确的未完成结果及恢复入口。
7. 作为已有工件的维护者，我希望人工内容、清单编号和完成标记不会被重试覆盖。
8. 作为操作人员，我希望中断恢复不重复创建 Task，也不把失败记为生成成功。
9. 作为后续实现者，我希望区分初始工件完整与工程设计就绪。

## Out of Scope

- 自动读取其他应用中的聊天记录，或把未确认讨论升级为批准事实。
- 自动安装模型、CLI 或依赖；借用未知凭据或无限重试。
- 取消 approve / promote 的原有正式授权。
- 修改通用 RunLLM 的全局 fallback 行为、接通 Coder/Tester 工作流或执行实现。
- 自动 Git 操作、发布、归档及批量迁移所有历史 Task。
