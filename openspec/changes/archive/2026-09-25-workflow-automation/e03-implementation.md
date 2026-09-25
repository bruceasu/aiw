# E03 实施记录

Task/change：workflow-automation。Work Item：wi-0003，authored checklist 1.3。保留现有 feature/workflow-automation 和 worktree，不修改真实 Task 元数据、Core 状态、Gate 或 Git 索引。依据 design.md、R2 授权设计及 verification delta；E04–E08 不在本轮实现范围。

> 2026-09-18 映射更正：上文保留历史执行者原始记录。当前 Core 的正式映射为 E03 / 1.3 → wi-0005，wi-0003 → 3.1（整体验收）；历史 E03 执行登记在 wi-0003 的 attempt-1789697370140599800 下，属于执行归属与实际实现范围不一致。本注不迁移或重新签发该 Attempt 的证据，也不将 E03 局部实现视为 3.1 已通过。后续按正式映射调度，E04 对应 1.4 / wi-0006。

## 实现与证据

- `grant_log.go`：严格 UTF-8 Markdown/JSON 日志，拒绝重复区块/键、未知或缺失字段、null、尾随值、未知版本、跨 Task、非规范集合和摘要链损坏。完整日志与 Core revision/尾部锚点一致才可授权。条件提交内先持久保存文件再提交锚点；孤立追加只允许原决定精确对账，相同已提交决定幂等。后续拒绝使旧批准失效，新批准须显式替代冲突历史。
- `tester_context.go`：保存独立需求/验收、报告、接口、源码、测试/fixture 清单、政策及允许路径正文；使用新的 Tester Session 和冻结 prompt，不注入 Coder 对话或 Session 最新 memory。Tester 提交 inline 具体计划，由宿主保存 Task 工件；AI/user 决策证据与身份、申请阶段、计划和政策版本逐字段核验。
- `controlled_test_plan.go`：schema 1 固定计划与 schema 2 范围计划分支隔离。旧 FocusedTestAuthorization/consumed 入口未改，新链不制造追溯批准。schema 2 发现结果进入本次清单，授权绑定范围/规则和约束；范围内新增文件刷新证据。核对 Task/workspace、路径及链接目标、具体 argv、程序字节、环境、网络和文件/字节/输出/内存/进程/超时约束。首批受控工具为 Go test 与 Python pytest/unittest；未注册工具返回 host-unavailable。
- `verification_service.go`：连接 E02 的 Authorize/ValidateResult/ValidateAcceptance，不代替 E04 的预算或启用验证。Tester 只接受当前编译和政策允许的测试写路径；Coder 修改测试/fixture 不获接受。结果必须对应真实宿主记录，绑定原请求、租约、输入、计划、grant、政策、逐项退出码/超时/原始输出。旧 waived 和 Agent passed 声明不能满足必需测试。
- `execution/verification.go`、`execution/verification_host.go`、`verification_journal.go`：接入 E02 RunStage 的同一意图/认领/Stop/观察/消费路径。实际调用独立 ExecuteFrozenTurn 和逐项 exec.CommandContext；执行前要求宿主强制隔离、稳定目标访问、间接工具身份和完整进程树约束。没有普通 shell 或网络豁免后备。输出有界保存为原始字节；终态按原请求持久记录，重启只读对账、不重放未知调用。
- `test_failure.go`、`execution/test_diagnosis.go`：普通测试失败先保存 unattributed，沿用 E02 原失败的一次只读诊断额度；精确诊断引用提交后，implementation/test 分别回交 Coder/Tester，infrastructure/requirements/仍不明保持恢复或人工等待。原终态不改写。E04 通过同一条件提交的 attribute:<class> 事件去重计次，不能再次消费生成请求。
- E02 接缝局部补齐：Tester 成功后回到 Compile，再进入 Test，防止使用测试编写前的编译证据；保存已验证报告引用；接受必须引用该当前报告。N/A 可以在编译后进入确定性接受检查，但只能覆盖政策逐内容批准的说明文档，不能依据扩展名豁免执行型 prompt/配置/脚本。

## E04 与联合启用交接

默认 schema 保持 9，本轮没有注册生产提供方或迁移真实 Task。E04 组装既有 Coder/Compiler 执行宿主、E03 VerificationService/JournaledVerificationHost 与其预算服务后才能进入统一启用核验；所有 generation/diagnosis/repair 仍须由 Core 原请求账授权。

JournaledVerificationHost 要求真实 VerificationBoundary，其 Within 返回的终态证明必须包含全部子进程停止；静态路径检查不等于消除了链接竞态。隔离提供方还须证明 network=deny、仅允许测试路径写入、测试运行只允许隔离临时输出、资源限制及间接工具环境。生产边界未安装时返回 host-unavailable，不能接空回调冒充平台证据。Coder/Compiler 原宿主使用同一执行身份与可核对的终态记录。

GrantDecisionAuthority 校验实际 AI 请求/输出或用户来源，不是由 Agent 自填 approved 即产生权限。模型决策调度及账本归 E04 组装；SaveDecision 只消费已保存的精确决定，不在 Store 锁内调用模型。Tester 修复的断言审阅权威核验原需求与修改依据，不能只信任 Agent 提供的引用。恢复/未派发证明使用独立的 VerificationObservationHost，不把缺文件、超时或前台退出当作未启动。

## TODO

- [x] 完成本轮 E03 服务、执行适配及 E02 局部衔接。
- [x] 更新 tasks.md 的 1.3、TODO、Verification 和启用风险。
- [ ] supervisor 执行当前版本冻结 Compile Plan，并登记实际结果。
- [ ] E04 接预算、迁移、生产注册及联合启用核验。
- [ ] 获授权后执行 AC10–AC14、AX01/AX03、授权日志崩溃/并发和真实宿主约束场景。

## Verification

新增 grant_log_test.go：歧义/缺失字段拒绝、跨 Task 拒绝、批准后拒绝、显式完整替代及实际范围变化失效。仅编写，未执行。

静态证据为上述函数的输入/输出、真实宿主调用、Core 状态转换和失败回交路径。最终静态检查使用用户指定的 Git safe.directory/-C 前缀，以及针对改动位置的文本读取；不将静态检查解释为编译或场景通过。

实际命令：本地 Get-Content（中文使用 -Encoding UTF8）、Get-ChildItem、Get-Command、rg、aiw patch --help、指定前缀的 Git status/diff，以及直接文件补丁。初次中文读取编码不匹配后改用 UTF-8；两次 PowerShell 通配路径 rg 出现路径错误，后续使用已定位的具体文件，没有权限重试。aiw patch 内部使用原始 git apply，无法符合本轮 Git 边界，因此使用文件补丁。文档补丁首次因上下文不匹配未应用，随后使用精确标题上下文修正。未执行 Git 写操作、网络、编译、测试、最终制品构建、格式化、lint、vet 或 validator。

%% VALIDATION_PENDING：本工作项实现勾选不等于编译通过、平台隔离成立或 AC/AX 已执行。编译和有界修复由 supervisor 负责。Windows 持久性、日志中断恢复、真实链接竞态、进程树退出和授权撤销场景仍需运行证据。

最终静态检查：一个批次执行指定前缀的 git diff --check、git diff --stat、git status --short --branch，配合 rg 核对新增类型/调用与 1.3 勾选，并读取宿主执行、接受路径片段。未显示 diff 空白错误；Git 仅提示已有工作副本的 LF/CRLF 转换。分支仍为 feature/workflow-automation，既有改动保留。diff --check 只覆盖已跟踪文件；新增文件的证据为本轮补丁与针对性源码读取，不能将该命令理解为新增 Go 文件已编译。没有追加检查或测试。
