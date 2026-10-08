# AIW 代码行为基线

本目录是根据当前工作区源码重建的 OpenSpec 稳定规格。基线日期：2026-09-16。
源码定位基点：`7161d0f6525361686fd50be84309348b82e03dee`；取证以本次工作区实际文件为准，不假定其他未提交文件已进入该提交。
它描述当前可追踪到的行为，不把设计文档、TODO、角色类型或清单勾选视为功能已接通的证明。
这里的 MUST / MUST NOT 用于固定现有行为；不代表相关行为已经通过运行验证，也不代表它是未来不能修改的产品决定。

## 能力索引

| 能力 | 内容 |
|---|---|
| [CLI 与扩展入口](cli-and-plugins/spec.md) | 内置命令、初始化、插件发现与执行、ask 和 cz |
| [Task 生命周期](task-lifecycle/spec.md) | 工件归属、创建预检、元数据兼容、状态与归档 |
| [Requirement 管理与发现](requirement/spec.md) | Issue 生命周期、人工决策、推广、来源校验、发现上下文、覆盖评估、提问、恢复与就绪 |
| [Agent Session](agent-session/spec.md) | 持久化会话、prompt 构造、后端调用与 turn 证据 |
| [AI 路由](ai-routing/spec.md) | provider、profile、路由计划与监督请求快照 |
| [Workflow 监督执行](workflow-supervision/spec.md) | Work Item、Attempt、结构化结果、重试与完成语义 |
| [验证与报告](verification/spec.md) | 编译修复、受控测试、Verifier 当前边界 |
| [工作区与本地交付](workspace-delivery/spec.md) | 隔离工作区、Git 预检、本地合并与清理 |

## 基线范围与维护

- 证据以生产入口及其调用链为主；每个 spec 附实现位置，测试文件仅作为可查阅的回归材料，不声称本次运行过。
- `improve-requirement-management` 已于 2026-09-16 同步并归档至 [归档清单](../changes/archive/2026-09-16-improve-requirement-management/tasks.md)。编号与存储聚焦测试已有通过证据；人工质量评审及其他未验证项随归档保留，不把清单完成当作完整运行验收。
- `docs/tmp-issue.md`、`docs/tmp-workflow-requirements/` 和 `docs/multi-actor-turn-handoff.md` 中尚未接通的目标不纳入已实现契约。
- 插件只固定公共发现/执行边界及工作区、Git export 等已检查的关键行为；不枚举每个第三方插件或每个 Git 包装命令。
- 修改行为时，应通过对应 change 描述差异，再更新相关稳定 spec；新增能力应补充独立 spec 与代码依据。

## 已知能力边界

- FD Workflow 默认在 Coder 完成编译与静态检查后直接交由 Verifier；独立 `fd-test` Skill 可按需生成黑盒测试报告，但不参与验收。
- 当前 Verifier 只提供请求/结果契约和 report-only 持久化辅助能力，没有监督执行中的 Agent 派发或需求缺项返工。
- 编译通过、勾选 tasks.md、Task DONE 与需求质量验收是不同事实；本基线不把它们等同。
- 通知 outbox、测试编写器等基础能力存在，不据此推断所有阶段已自动调用它们。

## TODO

- [x] 识别现有稳定 spec 缺失与主要生产入口。
- [x] 完成九个能力的 Requirement / Scenario 及源码依据。
- [x] 静态核对目录、相对链接、规格结构与变更范围。

%% 运行行为、真实模型质量、外部 CLI 兼容性及未逐项检查的插件功能仍需各自验证；本次不新增这些执行授权。

## Verification

首次重建仅新增本目录下的 Markdown 规格，共九个能力、52 条 Requirement、85 个 Scenario。后续实现按对应 change 更新能力规格；Requirement 自动编号由 improve-requirement-management 的 5.1 追加，其验证记录位于该 change 的 tasks.md。

- 使用 `rg`、`Get-Content -Encoding UTF8` 读取入口与关键调用分支，使用 `git rev-parse HEAD` 记录定位基点。
- 一次 PowerShell 只读逐文件检查覆盖新文件的相对链接、Purpose / Requirements 标题、规范用语、每条 Requirement 的 Scenario、WHEN / THEN 及尾随空白；发现问题为 0。
- 同批执行 `git diff --check -- openspec/specs` 和 `git status --short`；新文件尚未跟踪，因此新文件内容以逐文件检查为依据，不把空的 tracked diff 当作新文件验证证据。
- 未运行测试、编译、OpenSpec CLI 校验、模型调用或网络请求；本次没有生产代码变更。

%% 该静态检查不替代 OpenSpec CLI 的版本相关校验，也不证明实现行为通过运行验收。

