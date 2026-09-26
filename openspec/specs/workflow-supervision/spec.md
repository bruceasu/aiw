# Workflow 监督执行

## Purpose

固定当前 Work Item、Attempt、Session outcome 与前台 supervise 的执行规则。此规格描述目前的 Coder 与编译修复链路，明确请求归属、清单同步、失败重试及完成状态之间的关系。
## Requirements
### Requirement: 清单映射与稳定工作项

Workflow Core MUST 将 OpenSpec 清单编号映射为稳定 WorkItemID，保留既有 Attempt、Gate 与证据。同步 MUST 校验依赖存在且不是自身；已映射清单项消失时 MUST 记录可诊断的协调问题，不能静默删除历史工作项。

#### Scenario: 清单内容重新同步

- **WHEN** 同一个清单编号仍然存在
- **THEN** 系统复用其 WorkItemID，并更新标题与依赖映射。

### Requirement: 取消项结清 Task 执行

Workflow Core MUST 将用户明确取消的 Work Item 视为已结清项。Task 的执行状态 MUST 推导为 completed，当且仅当至少有一个 Work Item 且所有 Work Item 均为 completed 或 cancelled，并且不存在更高优先级的活动 Attempt、租用项、阻塞项或阻塞 Gate。取消项不得单独阻止 Task 达到 DONE；验证状态仍按独立验证规则决定最终显示状态。

#### Scenario: 所有范围内工作完成或取消

- **WHEN** 每个 Work Item 均为 completed 或用户明确取消，且没有活动执行、阻塞项或阻塞 Gate
- **THEN** Task execution 推导为 completed；所需验证完成后 Task 状态为 DONE

#### Scenario: 仍有未结清工作

- **WHEN** 至少一个 Work Item 仍为 planned、ready、leased、running 或 blocked
- **THEN** Task 不得仅因其他 Work Item 已取消而推导为 DONE

### Requirement: 请求与执行归属

监督请求 MUST 保留 Task、Work Item、Attempt、Session、预期 turn 和 workspace 绑定，追加阶段、角色、模型、允许路径及输入版本快照。既有阻塞、未修复投影和归属冲突优先于准备新工作。Coder/Tester 串行持有写入权；只有已保存执行终态或对账证明可安全恢复时才转交。

#### Scenario: 恢复未知阶段
- **WHEN** 前台退出而在途执行结果未知
- **THEN** 核对同一请求的派发、Session/turn 和执行器事实，不因超时释放写入权或重复派发。

### Requirement: 监督结果必须来自已派发 Session

监督层 MUST 要求请求已有 dispatch 标记、Session 结果为 completed 且存在最终输出文件。Session 的 Attempt 绑定 MUST 与请求一致；有预期 turn 时，实际 last_turn MUST 不小于该值。

#### Scenario: 读取旧 Session 结果

- **WHEN** Session 仍指向其他 Attempt，或 turn 小于预期值
- **THEN** 拒绝消费该结果，保留供诊断的错误，不关闭当前 Attempt 为完成。

#### Scenario: 请求尚未派发

- **WHEN** prepared request 没有派发标记
- **THEN** 系统不能把 Session 中已有输出作为本次结果。

### Requirement: 结构化 outcome 与失败分类

系统 MUST 保留 completed、blocked、no-progress 及实际输出引用，按实现缺陷、测试缺陷、环境、需求未决和报告缺失归因。无效/缺失报告采用 SW02 的一次仅报告补交，不直接套用普通源码修复。workspace-access 仍由 Core 预检判定；排除明确环境/需求问题后无法归因的失败按 SW07 有界诊断。

#### Scenario: 无效完成报告
- **WHEN** 执行已终止但报告格式无效
- **THEN** 关联原执行补交一次报告，仍无效转人工，不重新派发源码修改。

### Requirement: 编译完成后再同步清单

系统 MUST 将原编译后同步规则扩展为完整接受链：有效报告、当前适用编译和必需单元测试证据满足后才关闭 Work Item、同步完成及释放下游。Coder completed 只结束实现阶段，不关闭整体 Attempt；Session completed、清单勾选和旧 waived 均不得绕过。

#### Scenario: 提前勾选清单
- **WHEN** 编译通过但必需测试尚未完成或失效
- **THEN** 即使清单已勾选也不接受、不释放依赖，继续显示阶段和证据缺口。

### Requirement: 有界 no-progress 重试

系统 MUST 按真实原因使用专门预算，不为 no-progress 额外叠加额度。缺陷使用 SW16/SW17，报告使用 SW02，诊断和基础设施使用 SW07；恢复、普通 resume 或配置变化不得重置计数。先消费有效成功结果，再检查下一次派发是否有预算。

#### Scenario: 已归因缺陷遇旧阈值
- **WHEN** 当前缺陷仍在新模型/修复预算内
- **THEN** 不按旧三次连续编译或普通重试上限提前停止。

### Requirement: 前台监督与可观察报告

supervise MUST 保留顺序主流程和持久归属，并支持有界辅助宿主继续已登记工作。report MUST 展示阶段、在途身份、证据、阻塞、预算及已保存失败报告。退出前台不等于 Stop；显式 Stop 持久阻止新派发并触发在途对账。

#### Scenario: 辅助工作晚于完成
- **WHEN** Task 开发已完成但已登记辅助队列未结束
- **THEN** 有界处理队列，不回撤开发完成；退出和重启仍可恢复其事实。

### Requirement: 首次请求准备识别 Session 真实缺失

系统 MUST 识别 Session 存储的明确缺失错误及其包装形式。对于满足现有执行前置条件且 Session ID 等于 Task ID 的新任务，系统 MUST 在创建 Attempt 和 prepared request 前创建缺失 Session 并重新读取。系统 MUST 保留已有工作区绑定。

#### Scenario: 首次启动同名 Session 不存在
- **WHEN** Task 已有合法工作树、可执行 Work Item、同名 Session 绑定且存储确认 Session 不存在
- **THEN** 系统创建并读取 Session，再准备绑定该 Session 的请求，不因错误包装跳过创建。

#### Scenario: 重复准备已有请求
- **WHEN** 合法 Session 和匹配的 prepared request 已存在
- **THEN** 系统沿用现有归属，不创建重复 Session 或 Attempt。

#### Scenario: Session 创建失败
- **WHEN** Session 创建或重新读取失败
- **THEN** 系统报告错误，不据此创建新的 Attempt 或 prepared request，不删除已有工作树。

### Requirement: Session 非缺失错误不得触发自动重建

系统 MUST 区分真正缺失、损坏、不可读、身份冲突与归档记录。系统 MUST NOT 将后四者视为可自动创建的新 Session，MUST 保留记录与原始诊断。异名 Session 绑定缺失 MUST NOT 触发同名 Session 创建。

#### Scenario: Session 目录存在但状态文件损坏或缺少
- **WHEN** Session 已有目录但状态文件缺失或无法解析
- **THEN** 系统返回记录损坏或读取错误，不覆盖目录或创建替代 Session。

#### Scenario: Session 不可读或身份冲突
- **WHEN** 存储返回权限错误、重复记录或身份不匹配
- **THEN** 请求准备失败并保留原记录，不把错误转换为缺失。

#### Scenario: Session 已归档
- **WHEN** 绑定 Session 位于合法归档位置
- **THEN** 系统保持归档只读和不可运行约束，不在活动目录重建同 ID Session。

#### Scenario: 异名绑定缺失
- **WHEN** Task 显式绑定其他 Session ID 且该记录不存在
- **THEN** 系统返回缺失诊断，不猜测绑定或创建同名 Session。

### Requirement: 缺失 Session 的恢复保留 Attempt 审计

既有 Workflow repair MUST 使用相同的 Session 缺失分类，并仅在原恢复资格满足时调用已有缺失 Session Attempt 恢复操作。系统 MUST 保留 Attempt 历史，MUST NOT 为损坏、不可读或归档记录执行缺失恢复。

#### Scenario: 满足原恢复资格的缺失 Session Attempt
- **WHEN** 写租约存在、无 prepared request 且对应 Attempt 的 Session 确认不存在
- **THEN** repair 执行原有恢复转换并保留 Attempt 审计，不因包装错误漏掉恢复。

#### Scenario: 恢复读取到非缺失错误
- **WHEN** 对应 Session 存储返回损坏或读取错误
- **THEN** repair 报告错误，不执行缺失 Session 的 Attempt 转换。

### Requirement: 通用监督 Git 查询指令

通过当前工作区预检的每个 supervised Work Item 请求 MUST 携带限定于该工作区的只读 Git 查询指令，不依赖标题、语言或嵌套工具是否继承环境变量。指令 MUST 使用命令级 safe.directory 和相同目录的 -C，并保持 shell 字面路径正确。

#### Scenario: 普通实现项

- **WHEN** 中文标题的普通实现项通过预检并准备派发
- **THEN** 最终 provider 请求包含限定目录 Git 查询指令
- **AND** 不要求标题包含 no unrelated changes，也不把实现范围缩减为只修改检查框

#### Scenario: 范围审查项

- **WHEN** 现有规则识别出 scope-review 项
- **THEN** 请求包含通用查询指令及独立的审查编辑限制
- **AND** 原有只能修改所选检查框的限制仍然生效

#### Scenario: 路径包含特殊字符

- **WHEN** 规范工作树路径包含空格、单引号或美元符号
- **THEN** 对应 shell 中的命令参数保持完整字面路径，safe.directory 与 -C 指向同一已验证目录

### Requirement: 信任依据及失败边界

监督查询指令 MUST 仅来自当前请求对应的成功工作区预检。预检失败、信任目录缺失或不匹配时 MUST 明确停止派发。系统 MUST NOT 修改持久 Git 配置、信任任意目录或因此扩大 Agent 写入权限。

#### Scenario: 缺少当前预检依据

- **WHEN** 当前请求没有有效且匹配的信任目录
- **THEN** 在 provider 调用前返回明确诊断，并沿用准备失败的状态清理规则
- **AND** 不以空提示、全局 Git 配置或通配信任继续执行

#### Scenario: 旧 handoff 引用其他工作树

- **WHEN** 旧 handoff 中存在不同路径或此前的 Git 阻塞说明
- **THEN** 新请求仅使用当前预检核验的目录，明确限定新的查询授权
- **AND** 不改变其他执行、文件编辑或 Git 写操作限制

#### Scenario: 限定查询仍失败

- **WHEN** Agent 使用限定目录查询仍遇到访问错误
- **THEN** 保留具体错误并报告阻塞，不扩大目录信任或自动循环重试
- **AND** 保持现有 Core 与 Agent outcome 分类职责

### Requirement: 历史阻塞项显式恢复

修复生效 MUST NOT 自动关闭已有 Gate、重开 Work Item 或重新启动 supervise。恢复文档 MUST 说明确认修复版本与实际查询结果后，先解决 Gate，再 reopen，最后显式启动监督执行，并保留原失败证据。

#### Scenario: 存在旧的访问阻塞 Gate

- **WHEN** 修复交付但历史 Work Item 仍为 blocked
- **THEN** 该项保持阻塞直到显式恢复，不因代码更新或清单同步自动完成

### Requirement: SW01 每轮报告

系统 MUST 满足以下规则：每次实现/修复结果必须保存可追溯报告，绑定实际 Task、Work Item、Attempt、角色和 turn；包含需求覆盖、变更位置/版本、可观察输入输出及副作用、测试入口、决定/限制/风险、验证引用和历史关系。确实为空、未知与未验证必须区分；不为报告强造公共接口，不覆盖旧报告。

追踪：SW01 / AC01。

#### Scenario: AC01 验收行为

- **WHEN** 同项第二轮修复提交报告
- **THEN** 两轮身份、内容、引用可区分；未知理由不编造，空列表不掩盖缺字段

### Requirement: SW02 报告校验与补交

系统 MUST 满足以下规则：Supervisor 确定性核对报告内容、执行绑定和引用。缺失或无效时最多追加一次仅报告补交，关联原执行及补交身份，不重新派发源码修改；仍失败转人工。不得以模型升级、重启或新 Attempt 刷新补交额度。

追踪：SW02 / AC02。

#### Scenario: AC02 验收行为

- **WHEN** 报告缺失
- **THEN** 只补报告一次；补交仍无效及随后重启 → 等人工，不再写源码或补第二次

### Requirement: SW05 阶段及写入权

系统 MUST 满足以下规则：每阶段明确进入条件、执行者、输入版本、输出接受条件和失败去向；请求冻结身份、角色、模型、范围及输入。Coder、Tester 串行持有写入权，只有 Supervisor 在持久终态或安全恢复后转交；Coder completed 仅结束实现阶段，不结束整个 Attempt/Work Item。查询须显示阶段、在途身份、阻塞原因、证据和剩余预算。

追踪：SW05 / AC05。

#### Scenario: AC05 验收行为

- **WHEN** Coder completed/提前勾清单
- **THEN** 尚不关闭整项；Tester 仅在安全转交后获写权；查询可定位每个阶段

### Requirement: SW06 未知结果与 Stop

系统 MUST 满足以下规则：结果未知先核实；仅能证明未派发或可安全重发才重发。缺报告、超时或前台退出不能证明执行结束，不得重复双写或擅自释放租约。明确 Stop 阻止新派发，重启不解除；在途执行需对账，不能伪称已停止、已撤回或安全清理。

追踪：SW06 / AC06。

#### Scenario: AC06 验收行为

- **WHEN** 写入结果未知/明确 Stop 后重启
- **THEN** 先对账且不新派发；前台退出不冒充 Stop/执行终态

### Requirement: SW07 故障分类与恢复

系统 MUST 满足以下规则：规则先识别实现/测试缺陷、基础设施、需求未决及缺报告；缺陷归属不清时，排除明确环境/需求问题后最多一次独立只读诊断，无效或仍不清转人工。基础设施故障同 Work Item 同阶段最多追加 2 次恢复，须无执行器在途、可安全重跑且条件满足；缺条件等待。无进展按实际原因归类，不另加预算；耗尽只暂停受影响执行。

追踪：SW07 / AC07。

#### Scenario: AC07 验收行为

- **WHEN** 无法归责
- **THEN** 一次只读诊断，仍不明转人工；基础设施无在途且条件就绪 → 最多追加两次，缺条件等待，不能套到专门预算

### Requirement: SW08 预算恢复

系统 MUST 满足以下规则：失败、升级、修复、恢复和已用授权事实必须可恢复，重启、换 Session、配置/环境恢复或普通 resume 不刷新。旧计数先自动重建；证明余额足够可继续，不能证明时不归零或伪记耗尽，按 Task 集中请人工决定后续额度，有效旧结果仍可复用。

追踪：SW08 / AC08。

#### Scenario: AC08 验收行为

- **WHEN** 恢复旧 Task 的计数或继续运行
- **THEN** 旧记录可恢复余额则继续；不可恢复则一次汇总缺口、不归零；resume/改配置后余额保持

### Requirement: R1 持久提交与恢复一致性

系统 MUST 将阶段、写入权、Stop、预算变化、结果引用及相关消费标记作为可恢复的 Core 提交处理。被引用工件和精确授权条目 MUST 先可靠保存，再提交引用，最后才允许依赖它们的外部副作用。状态替换和事件确认 MUST 使用经过目标平台核定的持久化协议；保存结果未知时先对账，不能按未发生重做。恢复 MUST 只补齐原提交，不再次执行原副作用或消费额度。投影更新失败不得改变已提交事实。

此要求细化 SW01/SW02/SW05/SW06/SW08，追踪 AC01/AC02/AC05/AC06/AC08、AX02/AX03；不增加新的运行授权。

方案 A 的首期可靠性验证范围为 Windows 本机磁盘。系统 MUST 将设计就绪与协议启用分开：设计 Gate 解除可以开始实现，不能代替真实持久化和恢复证据；未验证的网络盘、NAS、同步目录或平台不得宣称具有相同恢复保证。

#### Scenario: 事件已保存但确认中断

- **WHEN** 同一提交的事件已保存，但确认状态保存中断
- **THEN** 恢复核对原事件身份和内容后确认一次，不再次计次、派发或递增逻辑状态版本；冲突保留证据并暂停相关推进

#### Scenario: 派发意图后结果未知

- **WHEN** 意图和预留额度已保存，但不能证明执行器是否启动
- **THEN** 保持请求身份、预留和写入权，先对账；不把未知当失败、未派发或可释放租约，重启不绕过 Stop

#### Scenario: 结果保存后消费中断

- **WHEN** 同一请求的结果已可靠保存，但 Core 未完成消费
- **THEN** 核对精确请求和输入版本后一次提交结果引用、归因、对应预算变化及消费标记；重复恢复不重跑执行器或重复计次

#### Scenario: 授权记录或持久化能力不足

- **WHEN** 精确授权条目未可靠保存、内容损坏、版本无法解释，或目标存储不能满足持久化协议
- **THEN** 不允许依赖它的派发，保留旧记录及具体缺口；索引、临时文件和仅存在于内存的同意不代替有效授权

#### Scenario: 残留锁与在途执行

- **WHEN** 恢复发现旧文件锁或源码写租约，而原执行器终态尚未证明
- **THEN** 不依据超时、PID 或前台退出单独解锁和转交写入权；文件锁经安全恢复也不自动释放仍在途的源码租约

### Requirement: R1 旧运行记录无破坏兼容

系统 MUST 保留迁移来源与不可解释事实，按明确版本规则恢复请求、Stop、授权、结果及预算；迁移不能创建新权限或重置已用额度。未知未来版本 MUST 拒绝相关写入和派发。旧结果仅在仍适用时复用，新格式回退不得重新启用缺证据的旧完成路径。

此要求细化 SW08，关联 SW04，追踪 AC04/AC08、AX02/AX03。

#### Scenario: 旧授权已用且预算无法重建

- **WHEN** 迁移发现 consumed 授权及无法证明余额的旧预算
- **THEN** 保留已用授权和未知预算，集中记录 Task 缺口；不将其转为有效授权、零计次或伪造耗尽，适用的旧结果仍可独立复用

#### Scenario: 迁移中断或未来格式

- **WHEN** 迁移在保存来源后中断，或读取到不支持的未来记录版本
- **THEN** 前者按同一迁移身份恢复，后者保留原件并禁止相关写入；均不得通过丢弃未知字段或恢复旧快照继续派发

## 实现依据

- [清单同步](../../../internal/workflow/commands.go)、[调度选择](../../../internal/workflow/runner.go)、[工作项选择](../../../internal/workflow/selection.go)。
- [Attempt 与 outcome](../../../internal/workflow/attempts.go)、[失败报告](../../../internal/workflow/failure_report.go)。
- [请求准备与派发](../../../internal/workflow/cli/command.go)、[前台循环](../../../internal/workflow/execution/supervisor.go)。
- [Session 绑定校验](../../../internal/workflow/execution/session.go)、[outcome 解析](../../../internal/workflow/execution/outcome.go)。
- [清单回归材料](../../../internal/workflow/commands_test.go)、[outcome 回归材料](../../../internal/workflow/execution/outcome_test.go)；本次未执行。

%% 当前完成仍依赖 Agent 的结构化声明、实际编译结果及清单同步，不证明每条需求都已实现。Verifier 自动调用与需求缺项返工尚不在本链路中。
