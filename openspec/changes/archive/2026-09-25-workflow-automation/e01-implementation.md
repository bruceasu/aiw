# E01 实施与验证记录

Work Item：wi-0001；清单：1.1。保留 `workflow-automation` Task 和 `feature/workflow-automation` 工作区，不修改 Core 运行 JSON、Gate、租约或 Git 状态。

## 实现边界

E01 在现有 workflow Store、Task adapter、Session 和 Supervisor 接缝内实现以下契约；没有切换 schema 9、锁协议或接受规则。`prepareFrozenAgentContext` 仅在 schema 10 迁移后使用新输入路径，旧请求仍沿用既有行为。该边界执行 design R1 的 E01–E04 一致启用要求，不是新增开关或用户审批。

| 来源 | 代码与静态依据 |
| --- | --- |
| SW01 / AC01 | `execution_report.go` 保存精确请求、原始输出、输入引用和补交关系；`implementation_report.go` 校验需求覆盖、内容版本、接口/副作用、测试入口、决定、限制、风险、验证和历史。字段区分 known、empty、unknown、unverified，按派发前内容基线检查报告未遗漏实际变更。 |
| SW02 / AC02 | `PrepareReportSupplement` 通过 Store 保存原请求的一次补交预约；`ReportOrigin` 固定原执行身份，预算不随新模型选择重置。Supervisor 不把报告错误当成源码修复；补交使用 `ExecuteFrozenTurn` 和 provider 的 ReadOnly 参数，再失败进入人工处理 Gate。 |
| SW03 / AC03 | `CaptureValidationInputs` 对排序后的实际文件内容、目录发现、计划与工具链身份取摘要，包含未跟踪新增/删除和同文件再次修改。根目录 `.ai`、`.wt` 与 Git 元数据不进入代码清单；工作区内其他内容保守纳入。编译请求保存不可变清单，旧通过只有仍适用才可消费。 |
| SW04 / AC04 | `AssessExecutionReuse` 区分 reconcile、wait、authorization-required、validate、report-only、reuse；未知旧来源不补造通过/授权/预算。`ReadAcceptedExecution` 核验 Core 接受引用、当前内容和报告；未接受成果不能作为下游输入。 |
| SW25 / AC25 | Task adapter 加载 proposal/design/tasks/spec 及其本地 Markdown 主来源正文，检查精确摘要引用，保存读取状态与可选摘要降级原因；handoff 不提升新需求。最终组合 prompt、instructions、memory、角色与允许路径保存在 Task，不从 Session latest 改写历史。必需来源或依赖缺失、越界、版本冲突、输入超限均在实际模型调用前失败。 |

来源展开限定为 128 个主来源文件、4 MiB 原始正文；最终组合 prompt 另受 4 MiB 字节上限约束，超限阻止派发，不静默截断。该上限是同步输入的保守文件大小保护，不是模型 token 容量声明，也不代替 E04 的模型预算或 E05/E06 的辅助预算。

工件使用现有 Task 锁、同目录临时文件、文件 Sync 和不覆盖已有内容的发布流程。schema 10 的 Windows 持久替换/锁迁移与真实文件系统可靠性仍由 E02 实施验证；本记录不宣称掉电安全已经成立。

## 验证

- 已新增 `execution_inputs_test.go`：实际内容与新增/删除发现、运行目录更新不失效、报告缺字段/空/未知区分、报告 turn 和内容绑定、旧证据不生成授权及下游接受边界。测试未执行。
- 实际执行：PowerShell `Get-Content` / `Get-ChildItem` / `Get-Command`、`rg` 定位与读取；指定前缀的 `git ... status --short --branch`；`aiw patch --help`。初次 PowerShell 默认编码不适合中文，后续读取显式使用 UTF8；路径/通配符错误没有被计为有效检查。
- 编辑使用 `apply_patch`。安装的 `aiw patch` 帮助未提供本次要求的 Git 命令前缀接口，因此直接修改工作区文件，避免包装器执行未核定的 Git 操作。
- 本轮最终验证仅进行一次指定前缀的只读 `git diff` 静态检查；新增文件按补丁内容核对。检查输入引用、请求归属、补交预约与只读调用、内容失效、旧 schema 路径及清单更新。
- 未运行 compile、tests、formatter、linter、vet、OpenSpec validator、故障注入、最终制品构建或网络访问。supervisor 拥有本轮冻结 Compile Plan 与后续有界修复；历史 compile 成功不是本轮结果。

## 后续接缝与未验证事项

%% ACTIVATION_PENDING：E02 迁移到 schema 10 时须保留补交历史和既有请求引用，不能将未知预算补零；E02/E03 负责生成 `WorkItem.AcceptedReference` 所指的真实接受事实，E04 负责请求归因及预算，不能仅因 E01 已实现而提前启用新接受链。

%% VALIDATION_PENDING：AC01–AC04/AC25、真实故障恢复、Windows 目录持久性、只读 provider 限制及跨重启补交去重尚未运行。无来源证明的旧结果保持 reconcile；不靠清单或 Agent 声明补造控制执行记录。

%% TOOLCHAIN_BOUNDARY：当前指纹读取注册工具的可执行文件与相关选择环境，不运行版本探测。通用 script 环境显式标记 incomplete，适用性检查拒绝将其作为可复用通过。脚本间接调用的额外工具、解释器环境与外部依赖必须由 E03 对应执行适配器提供完整身份后才能放行；不能将基础指纹理解为任意脚本的环境封存。此限制不改变 schema 9 的冻结编译入口。

推荐下一步：supervisor 对当前差异执行冻结编译计划，记录真实命令与结果；随后继续 E02。保留其他清单项未完成，不进行 Git 交付或 Task 关闭。
