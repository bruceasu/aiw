# E08 独立 Verifier 实施记录

2026-09-18；Task/change `workflow-automation`；Work Item `wi-0010`；Attempt `attempt-1789727915907036500`；authored 清单 `1.8`。

## 实现范围

- `verifier_snapshot.go`：接受前固定需求/验收正文、分项覆盖清单、实现报告、接受候选、受控编译/测试 receipt 与输出证据。需求文件若在接受清单内，使用该清单绑定的版本；否则复核原始输入版本。快照后复核内容清单，接受路径再按原规则复核。
- diff 明确以接受时解析出的 Git HEAD 为基线，保存限定输入 scope 的 staged/unstaged 二进制 diff，以及未跟踪新增文件的完整正文与摘要。该基线可能包含本 Work Item 开始前已有的改动，报告须按原始需求判断，不能称为“仅此轮 diff”。不执行 Git 写入、测试或修复。
- `prepareVerifierSnapshot` 复用 E05 项目资源预约，并按项目锁→Task 锁顺序保存 Task-owned、内容寻址的快照。快照引用与接受事实在同一 Core 提交绑定，后续辅助来源包含引用；cleanup 的现有封存路径覆盖它。
- 缺来源、无法读取精确证据、Git 不可用、非 UTF-8 未跟踪文件、输入超限或无存储预约时，在接受记录保存 `verifier_gap`。接受逻辑仍按 E01–E04 的原条件执行，Verifier 不产生 Gate 或接受失败。完整输入超过 128 KiB 不截断后派发。
- `RegisterTaskVerifier` 从已接受来源登记任务，固定首次包含该接受引用的来源。重复启动复用快照、job key 和源账；已完成 Task 也能通过现有 helper 恢复。缺少历史快照时显式记录局部缺口，不用当前代码补造旧 diff，不重开实现或执行测试。
- `AuxiliaryHost` 与生产 helper 接入 `verifier` HTTP Worker，调用使用独立、固定快照正文，不带 Coder 会话或后续源码。沿用能力证明、显式授权、无工具响应、输入/输出限额、单槽、Stop、120 秒调用与 10 分钟宿主边界，不新增线程、宿主或计次器。
- v2 报告绑定 Task/Work Item/Attempt/snapshot，逐项保存适用性、结论、依据、证据摘要和缺口。每个固定 criterion 必须恰好覆盖一次，证据摘要必须属于封存输入；有据失败优先 failed，无失败但有缺口或无适用项为 inconclusive，适用项全部有据通过且无缺口才 passed。
- 合法 failed/inconclusive 不算模型失败；空报告、空覆盖、漏项、未知证据、结论矛盾是无效输出，沿用 E05 单次共享恢复。报告正文及结构化投影与辅助输出一起持久提交；保存成功只表示报告记录成功。
- 保留原 schema 1 报告接口兼容；生产 E08 只接受 v2 及完整覆盖。没有改变默认 schema 9、Task 生命周期、接受状态、交付或源代码修复责任。

## TODO

- [x] 接受输入封存及局部不可用事实。
- [x] E05 来源登记、独立宿主和生产 Worker 接线。
- [x] 覆盖校验、三态结论及迟到报告版本隔离。
- [x] 更新 authored `1.8`、TODO、Verification。
- [ ] supervisor 执行冻结 Compile Plan，并提供编译结果。
- [ ] 授权后执行 AC19/AC27/AC31/AC32：重启复用、Stop/迟到结果、空覆盖、失败优先、证据不足、保存失败、资源超限和完成后交付隔离。

## Verification

静态核查范围：`AcceptExecution`→快照/资源预约→来源 outbox→`RegisterTaskVerifier`→`AuxiliaryHost`→HTTP Worker→语义校验→辅助输出发布；同时核对 `ReadExecutionArtifact`、receipt、来源封存和存量 schema 兼容边界。一次最终静态命令读取本轮差异和新文件，不代表运行验证。

该静态命令退出 0，确认 authored `1.8` 已勾选、宿主/HTTP 校验/发布接线存在，Go manifest 为 1.25.1；Git 仅提示 LF/CRLF 转换。核查后按补丁文本修正 Git 输出缓冲器，避免嵌入 `bytes.Buffer` 的 `ReadFrom` 绕过写入上限，并限定管道收尾等待；未再执行检查命令。未跟踪新文件通过 `Get-Content` 查看，未声称被 `git diff` 覆盖。

实际执行的命令类别：`Get-Content`、`Get-ChildItem`、`Get-Command aiw`、`rg`、`Select-Object`、`ConvertFrom-Json`；`aiw patch --help`；用户指定命令前缀的 `git status --short --branch` 与最终 `git diff`。初次 JSON 读取因 Windows PowerShell 默认编码失败，使用明确 UTF-8 后读取成功；两次带文件 glob 的 rg 定位未成功，未重试等价命令。实际编辑使用直接补丁工具：本地 `aiw patch` 的实现最终调用 `git apply`，不能满足本轮仅允许指定只读 Git 的约束。

未运行编译（由 supervisor 负责）、测试、最终制品构建、formatter、lint、vet、验证脚本、网络、模型、真实辅助宿主、资源维护、Git 写操作或 Core/Gate 生命周期操作。保留已有其他工作项改动；原始 AC/AX 场景不标记通过。

%% E08_ACTIVATION_PENDING：生产运行仍需 E01–E05 的 schema 10 联合启用证据、辅助能力/授权配置及资源盘点。具体模型容量未知时等待，不联网探测。当前没有运行证据，不宣称编译通过、实际 Verifier 已调用或 Task 整体验收完成。

%% E08_SNAPSHOT_LIMIT：超过完整输入上限、不可完整表示的文件或旧接受缺快照会显示 unavailable，不自动拆分、截断、换模型或改用当前版本。可运行快照的存储/队列/调用恢复均使用原 E05 账；无法预约时保留接受事实和原因。报告依据是否充分仍由独立模型审查，静态结构校验只保证覆盖完整、引用在界内和三态汇总一致。
