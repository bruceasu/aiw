# E02 实施记录

Task/change：workflow-automation。Work Item：wi-0002。本轮只实现 E02，保留既有 worktree、E01 改动和设计。依据 design.md 的 R1、r2-authorization-design.md 及相关 capability delta。

## 实现与证据位置

- Store/durable：schema 10 条件提交、state_revision/commit_id、严格字段读取、事件确认和可证明的 PendingEvent 尾部恢复。保存原件，不重跑状态回调。Windows 使用临时文件 Sync/Close、MoveFileExW(REPLACE_EXISTING/WRITE_THROUGH)、目标 Sync/Close；事件追加检查写入/同步/关闭。替换未知不盲重试或删除旧文件降级。
- durable Task lock：稳定文件上的非阻塞 LockFileEx；释放系统锁不删除文件、不释放源码租约。迁移只能在旧写者已退出、没有旧锁/在途/待确认提交的受管维护边界进行；不按 PID/锁龄接管。真实 Task 未迁移。
- execution_protocol/stage_execution：Coder → 确定性报告校验 → Compile → Tester → Test → 接受。绑定身份、模型、输入、计划/grant/政策和 lease generation；同一 Task 串行派发。Coder 终态保留 Attempt；实现/测试缺陷分别回交，输入失效可只返回验证。
- stage_result/execution/stages：意图、认领、观察、终态与消费分开。认领后为 unknown，恢复只读对账；未证明终态或未派发时保留写权/预留。结果按原请求保存，消费中断后重放不重复计次。报告补交使用原生成请求一次额度；基础设施按 Work Item/阶段追加两次；不明缺陷保留一次只读诊断登记。
- execution_stop/workflow_supervisor：schema 10 的显式 Stop 独立于前台租约持久保存，查询展示阶段、在途请求、Stop 和恢复余额。重启/旧入口不清除 Stop；恢复需要决定引用，不重置预算。
- execution_acceptance/execution_reuse：E03 提供候选，Core 在同一版本下重核输入字节、报告、Compile/Test、计划/grant/政策及 N/A 依据；先保存不可变证据，再关闭 Work Item/Attempt。必需测试不接受 waived，依赖读取复核适用性。
- managed_delivery/execution/local_delivery/managed_cleanup：冻结计划、路径、消息与逐步动作。只暂存清单内路径；固定本地 commit/分支创建/merge。未知不重放，冲突不自动 abort/reset，已 merged 只续 cleanup。清理仅经 AIW 宿主，逐步核验精确 grant、目标、合并和来源封存；旧 localMerge/cleanup 拒绝 schema 10。

## E03/E04/E05 交接

默认 SchemaVersion 保持 9。E02 提供 Core 与宿主接缝；未注册生产提供方，未执行迁移或启用新链。测试替身不得作为启用证据。

E03 接入 ExecutionServices.Authorize/ValidateResult/ValidateAcceptance 和受控 StageExecutor。实际路径/链接/内容、宿主、grant/计划及终态/未派发证明由服务确定性核验，不能信任调用方给出的 status。Tester/Runner 编排和授权日志仍属 E03。

E04 实现 Budget/MigrateBudget 与 DeliveryServices.Account，在同次条件提交内更新账本，不在回调调用模型/外部执行器。保留预留、消费与未知余额；直接终态消费也必须计入真实派发，不依赖另一个启动观察先行成功。模型升级和六次修复政策不由 E02 替代。

VerifyActivation 必须核对 E01–E04 完整实现/迁移证据及 Store 实际根目录所在卷、目标平台持久化/恢复、授权与接受证据。锁迁移的维护核验须证明旧二进制已被隔离、旧写者退出。不能用空回调启用生产。旧原始状态按字节保存，未知字段拒绝迁移；旧 consumed 授权不修改，预算未知时不可派发。

交付须注册 DeliveryServices.Check/Verify/Account、HostCheck 和精确事实 Inspect。每次副作用前核对索引、提交、路径、hooks/过滤器/签名程序和宿主约束；commit 暂存后第二次检查验证计划内预期索引状态。Inspect 只读保存对账证据，不凭消息或缺失路径猜成功。E05 提供完整封存清单后才能连接 AIWCleanupHost。生产适配注册和联合启用由后续工作项完成。

## TODO

- [x] 实现 E02 Core、持久提交、系统锁/迁移、恢复、Stop、接受及受管交付接缝。
- [x] 更新 authored checklist 1.2 与本轮 TODO，不改 Core 生命周期或 Gate。
- [ ] supervisor 执行当前版本冻结 Compile Plan，记录实际结果。
- [ ] E03/E04 接生产提供方并完成联合启用；E05 完成来源封存。
- [ ] 获授权后运行平台故障恢复及 AC05–AC08/AC14/AC33。

## Verification

新增 protocol_windows_test.go，覆盖缺提供方/旧锁拒绝迁移、未知请求不重复认领、Stop 跨重开保留租约、结果重放不重复计次、Coder 不关闭整项、错 turn 拒绝、事件尾部恢复不重复推进。全部未运行。

静态核对 Store 提交/恢复和 schema 守卫、E01 工件/依赖读取、阶段绑定、对账、接受、CLI Stop、固定 Git argv 与 AIW 清理入口；这不证明真实 Windows 掉电恢复。

实际执行：本地 Get-Content（中文材料使用 -Encoding UTF8）、Get-ChildItem、Get-Command aiw、rg、aiw patch --help、文件补丁，以及使用用户指定命令局部 safe.directory/-C 前缀的 Git status/最终 diff。aiw patch 内部调用原始 git apply，无法注入本轮限定前缀，因此使用文件补丁。没有修改 Git 配置、索引、分支、提交或真实运行 JSON。

%% VALIDATION_PENDING：编译由 supervisor 执行冻结 Compile Plan，本执行者未运行编译、测试、最终制品构建、formatter、lint、vet、validator 或网络命令。历史通过不作为本轮结果。进程/存储中断、实际卷能力、链接竞态隔离、授权拒绝与 cleanup 中断矩阵仍待运行证据。

最终静态检查：使用指定 Git 前缀执行 diff --check、diff --stat、目标文件 diff、status --short --branch，并用 rg 核对清单。当前分支仍为 feature/workflow-automation；diff --check 未显示空白错误，Git 提示 LF/CRLF 转换。检查发现 Store 的恢复序号校验和严格解码被补丁置入错误函数，已通过局部读取修正到 RecoverPendingEvent 和 loadFromDir；未追加编译、测试或重复验证命令。当前静态修正后的实际编译结论仍由 supervisor 提供。
