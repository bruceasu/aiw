# E05 实施记录

Task/change：`workflow-automation`；Work Item：`wi-0007`；正式清单：`1.5`；Attempt：`attempt-1789720672532536600`。

当前状态：**E05 实现完成，等待 supervisor 编译与验收证据**。2026-09-18 的 attempt-1789723715632420700 补齐了受控 Worker、独立 helper、生产 Store/Session 接线及资源维护入口，详见 [E05 宿主接入](e05-host-integration.md)。以下原始组件记录及“尚缺”段落保留为前次 Attempt 的历史，不再代表当前代码缺口；实际能力配置、联合启用和运行验收仍未完成。

## 已写入的组件

- `internal/workflow/auxiliary_source.go`：在 schema 10 原有条件提交中捕获 Task 原始来源版本；内容变化生成来源事实，摘要/队列状态与展示时间不生成新来源。Session 投影区分 accepted 与未接受内容，逐一复核原始工件摘要，摘要过期/不可读时携带原文及降级说明，必需来源缺失则返回错误。保持人工 Session memory 原文。
- `auxiliary_queue.go`：操作类型、来源身份及完整输入摘要组成固定键；来源 Task 保存唯一执行和恢复账，其他 Task 只保存引用。来源登记与 memory 消费游标同一提交，项目预约先行，重启采用原意图。四类模型辅助操作最多一次额外恢复，模型重调、无效输出重生成和有效输出重存共享额度；未知观察不自动重试。结果按来源版本发布，迟到 memory 仅存历史输出。
- `auxiliary_resources.go`：项目系统锁、版本化政策引用、唯一赞助 Task、单项目执行槽、token/次数累计、队列和存储峰值预约。完整输入上限 32,768 tokens / 128 KiB，输出 4,096 tokens / 64 KiB，Task 累计 524,288 tokens / 64 次；Task/项目队列 128/1,024，存储 512 MiB/4 GiB，卷余量 256 MiB。缺能力、计量或历史盘点时保留局部缺口；不把缺账当零，不联网获取 tokenizer。未知 usage 保留预约，确认未派发才释放。控制记录和有限观察工件采用保守预留，可能早于硬上限停止新分配；不自动删历史。
- `auxiliary_validation.go`：验证来源摘要、固定键、游标覆盖、调用身份及共享恢复上限；来源提交后在释放 Task 锁后调用独立宿主启动接缝，启动失败不撤销开发事实。显式启动同样覆盖已完成 Task。
- `auxiliary_seal.go`：封存 Task 原始来源及辅助固定输入，并在 `managed_cleanup.go` 的受管删除前复核来源覆盖。封存失败保留工作树；没有执行任何实际 cleanup。
- `auxiliary_storage_windows.go` / `auxiliary_storage_other.go`：本地卷可用空间读取接缝；未验证平台返回不可用。
- `internal/workflow/execution/auxiliary.go`：真实 Store 转换驱动的宿主循环，宿主 10 分钟、调用 120 秒；启动先对账项目在途请求，遵守来源及赞助 Task 的 Stop；无可执行项退出，完成/交付本身不丢弃登记。只读 Worker 必须自己执行能力、输出、进程边界和终态日志契约。
- `auxiliary_process*.go`：可配置的 Windows 隐藏独立进程启动器及 `ConnectAuxiliary` 组装接缝；参数直接传递，不使用 shell，也不以 goroutine 代替独立宿主。启动器依赖实际可执行 helper，并不自行实现 helper 主程序。
- `agent_context.go` 和 `workflow_supervisor.go`：冻结 Session 输入时加入 Task 来源版本；受管启动调用辅助恢复接缝，并显示队列状态和恢复使用情况。

通知没有纳入四类模型操作的统一恢复额度。E07 必须在接入时共享项目槽和存储，并保留其独立重发政策；本轮没有实现通知、知识业务生成或 Verifier 审查。

## 尚缺的接入及限制

1. 仓库尚无本轮可注册的具体 `AuxiliaryWorker`。必须由真实宿主提供可靠本地 capability/计数器、完整 provider 封装计量、强制输出及整个执行进程范围的限制。`Start` 与 `Reconcile` 必须绑定同一请求和实际宿主，并可靠区分未派发、运行中、确定终止与未知；仅缺少日志、超时或 PID 消失不能作为重试证明。
2. `DetachedAuxiliaryLauncher` 仅实现启动边界。仍须提供真实 helper 入口，处理 `--auxiliary-root` / `--auxiliary-task`，重建受管 Store 与 Worker，调用 `AuxiliaryHost.Run`；当前普通 AIW 命令入口不解释这两个参数。不得把启动器返回成功视为模型派发或辅助完成。
3. `ConnectAuxiliary` 尚未注册到生产 Store 工厂。授权、模型能力、观察验证、历史盘点与可见缺口回调均为必须提供的边界，当前不会以宽松默认实现替代。schema 9 仍为默认；R1 平台、迁移和 E01–E04 联合启用条件仍保留。
4. `.ai/resource-policy.json`、项目辅助资源账及 Task 存储基线仍需受管提供。`InitializeAuxiliaryResources` 要求盘点证据和验证回调；未创建或初始化当前项目实际账本。提高已有政策必须有新版本和原因，仍不能超过批准的首期硬上限。
5. 资源预约采取保守保留；尚未实现经盘点证据结算物理共享工件/临时空间的管理入口。未知空间、达到控制账大小上限或缺失历史计量都会停止受影响辅助处理。需要继续核对其与 R3 完整生命周期的覆盖，不能宣称已通过资源验收。
6. 当前 Session 组装修改在既有 `prepareFrozenAgentContext` 接缝。生产阶段宿主与其他角色必须使用同一 Task 来源/封存边界；接通具体 Worker 时还须核对这些调用路径。

%% E05_INCOMPLETE：正式 1.5 保持未完成。缺少真实 Worker、helper 入口与生产组装时，当前实现不能满足 AC09 的实际后台执行和 AC18 的实际摘要生成。建议 supervisor 保留依赖缺口，继续本工作项；不跳到 E06/E07/E08，不处理 Gate，不改变 Task 生命周期。

## 验证与执行记录

- 首先读取主工作区 `artifacts/handoff.md`，随后读取 implement 技能、工作管理契约、Core 提示、Task 元数据、proposal、适用 design/R3 附件、tasks、task-memory 与 agent-session delta，以及 Store/接受/Session/cleanup 附近源码。新增文件没有另起 Task 或工作树。
- 实际执行的是本地 `Get-Content`、`Get-ChildItem`、`rg`，以及用户指定完整前缀的只读 Git：`status --short --branch`、`diff --check` 和受影响既有文件的 `diff -- ...`。一次默认编码的 `ConvertFrom-Json` 读取失败，没有据此改写 Core 或重试权限；后续正文读取显式使用 UTF-8。部分较大的发现输出被截断，未将被截断的未见内容当作证据。
- 仅一个编辑后静态命令批次；`git diff --check` 未报告空白错误，仅提示已有 LF/CRLF 转换。它只覆盖已跟踪差异，不能作为新增未跟踪 Go 文件已检查或已编译的证据。后续局部修正、封存接缝和文档按补丁内容检查，未再运行验证命令。
- 因 `aiw-patch.py` 使用 Git 写路径，而本轮 Git 仅获只读授权，使用直接补丁工具修改源码和 OpenSpec 文档；没有修改 Git 配置、索引、分支或提交。
- **编译未运行**：按当前监督指令由 supervisor 执行冻结 Compile Plan。**测试未运行**；未运行最终制品构建、formatter、lint、vet、验证脚本、网络调用、能力探测、模型生成、后台进程、通知、迁移或磁盘可用空间函数。
- AC09/AC18/AC19/AC24/AC25/AC27、AX02/AX04 等场景均未执行。没有把静态代码、清单进度或接口声明当作运行证据。

下一步是继续同一 wi-0007 的受控宿主接入，补齐后更新正式 1.5，再由 supervisor 编译。真实运行验证与一致启用另需相应授权和证据。

## 继续实现的交接（2026-09-18）

本次核对确认上述 Worker、helper 和生产注册缺口是 E05 自身尚未完成的实现，不是等待用户审批的外部依赖。R3 已固定能力与计量契约及缺失时的局部等待规则；无需再次批准这些政策。恢复同一 wi-0007，不迁移历史 Attempt、不勾选 1.5，也不解除 E06–E08 对 E05 的依赖。

继续顺序：先提供真实独立 helper 的命令解析和受管 Store/宿主组装；再实现具备固定请求日志、计量、输出/进程限制及 Start/Reconcile 对账的具体 Worker，并接入生产工厂；最后补齐政策与历史存储盘点的受管入口及 Session 调用路径。不得以空 Worker、恒成功验证回调或仅实现接口代替产品接入。缺少本地模型能力记录、历史盘点或启用证据时，具体调用保持 unavailable/waiting，仍可继续实现其配置解析和受控执行路径，不联网探测、不编造模型参数、不自动启用 schema 10。

本恢复将临时 supervised-dependency-wi-0007 按“内部实现待续”处理；不是宣称该缺口已实现或豁免验收。只有遇到无法由已批准设计和本地代码确定的外部决策，才建立包含确切缺失输入的新 Gate，避免再次只因实现未完成而要求用户重复授权。具体恢复命令结果记入 tasks.md。
