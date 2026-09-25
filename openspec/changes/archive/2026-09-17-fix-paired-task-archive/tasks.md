## TODO

- [x] 1.1 实现规格与运行目录的双侧候选发现及活动/归档/历史半归档定位，保留旧路径兼容并诊断冲突；规格缺失显示“规格已删除”，运行记录缺失通过 AIW/Workflow Core 幂等补建最小记录，保留有效数据且不启动执行。
- [x] 1.2 实现规格、任务运行目录及绑定 Session 的协调归档，覆盖双后端、绑定/占用预检、逐步失败补偿、缺失提示、幂等重试及历史补归档；增加新旧归档 Session 只读定位和归档写入保护，保留既有 Task 资格与 Git 授权规则。
- [x] 1.3 增加 aiw list --all 参数与归档列，默认列表排除归档任务；实现动态列宽、交互表头和状态颜色，复用现有终端能力并支持禁色及纯文本重定向，保留默认三个字段及顺序；更新帮助、文档与补全。
- [x] 1.4 在现有 CLI 层添加三处归档与逐步失败恢复、后端一致性、历史兼容及冲突测试；覆盖 Session 异名绑定/缺失/运行占用/重复绑定/归档查询与拒绝写入，规格缺失及运行记录幂等补建，以及长列对齐、表头、颜色、空列表和重定向；仅编写测试，不默认运行。
- [x] 1.5 完成聚焦静态检查，核对命令入口、后端、路径及 Workflow 状态读取，更新 TODO 和 Verification，并按仓库预算执行一次适用的 compile-only 检查（按本次交接，compile-only 由 supervisor 执行并验收）。

## Verification

- wi-0005 / 1.5：完成本项静态核对及清单更新；复核 proposal、design、变更 spec 与稳定 task-lifecycle spec。只更新本文件的人写清单和验证记录，不修改生产代码、既有 Task/worktree、Workflow Core 状态、Gate 或租约。勾选表示本项交付完成，不表示 supervisor 编译已经通过或整个 Task 已验收。
- 命令与后端证据：`main.go` 的顶层与 task 子命令均分派到 `DispatchTopLevel`；list 参数进入 `listTasks`，native 与 OpenSpec archive 分别经 `archiveTask` / `runOpenSpec` 汇合到 `archiveWithBackend`。归档链路先预检与资格核验，再刷新计划；逐步移动、核验委托实际目录、最终重新定位，失败时仅逆序补偿本次移动。历史归档沿用原名称并跳过规格同步和 Git 交付；显式 push/清理参数与终止资格仍在 `prepareArchiveEligibility` 内检查。
- 路径与状态证据：`DiscoverTaskLocations` 固定扫描活动、旧版及两个规格归档根，核验完整日期/ID 与重复候选；`collectTaskListRows` 在 `EnsureRuntime` 前过滤归档任务，使用真实 `RuntimeDir` 的 `LoadFromDirectory` 与 `DeriveSummary`，不进入会迁移活动目录的 Store.EnsureCompatible 路径。缺失状态通过 `EnsureCompatibleInDirectory` 排他创建，已存在文件重新读取；规格缺失显示“规格已删除”，归档标识与 Workflow 状态分列。
- Session 与展示证据：Task 查询由真实运行目录读取绑定，`Store.Resolve` 核验活动、新配对及旧独立归档位置；`Load`/`ReadText` 保留历史状态并映射旧绝对路径。Save、锁定更新、工件、提示词、事件、内存及删除路径受 requireWritable 保护，执行入口调用 RequireRunnable。列表在着色前计算完整列宽，填充位于复位之后；交互表头、空输出与 NO_COLOR/TERM=dumb 策略有对应分支。核对主帮助、README、补全入口及既有 CLI 测试场景名称；未把测试源码存在当作运行成功。
- 本轮实际使用 Get-Content、Get-ChildItem、rg 进行定向只读核对，apply_patch 更新本文件。PowerShell 花括号路径语法错误作一次限定修正；部分猜测文件路径不存在，依据实际目录定位到 main.go、commands/help、commands/completion 和 session/store.go。静态核对跨文件调用链所需源码未包含在初始材料中，因此追加局部读取。沿用已记录的 Git dubious ownership 限制，不重试 Git、不修改信任设置，直接 apply_patch 编辑。
- 未运行测试、编译、最终构建、格式化、lint、vet、validator、网络、实际列表或归档、Git 写操作。已静态读取 `scripts/compile.py`：其使用工作树缓存，将 main.go 编译到临时目录并清理二进制；本轮遵照 handoff 与用户指令，把冻结 Compile Plan 的执行及有限修复循环交给 supervisor。建议下一步由 supervisor 执行冻结编译计划、记录结果并决定本项验收，不由本轮解除 Gate。
- %% VERIFICATION: supervisor 的 compile-only 结果待回填；`scripts/compile.py` 不编译 `_test.go`，新增测试仍未获编译或运行验证。外部 OpenSpec 实际版本、真实终端/Windows ACL、进程中断及三处非原子移动的补偿失败仍保留原风险，不由静态核对宣称通过。

- wi-0004 / 1.4：新增 `internal/commands/task/archive_test.go`，在临时目录通过 CLI 分派覆盖 native、auto、openspec 及 auto 回退；OpenSpec 使用当前测试可执行文件的受环境变量与参数约束的本地替身，不依赖安装、shell 或网络。核对三处目录完整内容、异名 Session、无关同名 Session 不变、重复归档、各步移动失败逆序补偿、最终核验失败恢复 Session、补偿失败路径诊断及历史归档不纳入本次回滚。
- 归档测试补充当前/旧规格归档根、旧独立 Session 归档及原日期修复；Session 缺失、损坏、身份/重复绑定、运行状态、执行时间、Task/Session 锁、目标冲突、Attempt/Work Item/租约占用，以及既有终止/工作区/finalize 限制。通过 Session CLI 检查 status/get/memory/handoff 只读查询和写入拒绝，并检查旧绝对输出路径映射与归档 Session 不可运行。
- 新增 `list_paired_test.go`，覆盖整个运行目录、元数据、状态及旧目录残留的最小补建，保留已有文件与清单、无 Attempt/租约/虚构证据、重复读取的内容及时间戳不变；覆盖归档过滤先于补建、归档位置补建、DONE/CANCELLED 与 ARCHIVED 分离、双根历史兼容、冲突/损坏/不可读及非目录路径诊断。POSIX 权限场景在 Windows 或可绕过权限的用户下明确跳过，不伪称 ACL 覆盖。
- 列表展示测试覆盖长 ID/长状态三列与四列对齐、交互表头、颜色移除后的布局一致、单元格复位与未着色填充、状态色、空列表、文件/管道/未知 writer、NO_COLOR、TERM=dumb 和参数错误。更新原 `list_test.go` 的固定列宽、规格缺失、补建与错误语义断言；更新 `command_test.go` 的规格同步夹具为显式 primary 工作区和已完成的编号清单。
- 静态证据：核对 CLI 分派、archiveRename 故障边界、Session Store/CLI 签名、Workflow 类型与引用，以及补建/过滤/渲染调用路径；补齐占用夹具的 Attempt/Work Item 引用，并断言具体占用错误。执行一次聚焦只读核对，路径/PowerShell 引号错误作一次限定修正。实际命令为 Get-Content、Get-ChildItem、rg 的定向读取；编辑使用 apply_patch。初始材料输出受编码/截断影响，使用 UTF-8 补读；接口未确定时追加了局部源码读取。
- 沿用交接的 Git dubious ownership 限制，以 task.toml、.git 指针和 worktree HEAD 核对 Task/工作区/分支，直接 apply_patch 编辑，未执行 aiw patch、Git 写操作或修改信任设置。未改生产代码、Workflow Core 状态、Gate 或租约；只勾选 1.4，保留 1.5。
- 本轮未运行测试、编译、最终构建、格式化、lint、validator、网络或实际归档；compile-only 由 supervisor 按冻结计划执行。建议接收本项 Evidence 后继续 1.5，不把测试编写完成视为测试通过。
- %% RISK: 新增测试尚未编译或运行；supervisor 的普通生产包 compile-only 未必编译 `_test.go`。真实交互终端/Windows ACL、外部 OpenSpec 实际版本行为及进程中断仍无运行证据，后续聚焦测试需要另行授权。

- wi-0003 / 1.3：list 与 task list 接入 --all，拒绝多余/未知参数，支持 --help/-h；复用 collectTaskListRows，默认过滤后补建，--all 输出独立 ACTIVE/ARCHIVED 列及真实路径。
- 展示层按未着色的完整 ID、状态和归档标签计算列宽，列间两个空格，路径置后；交互表头、空列表无输出、状态颜色逐单元格复位且填充不着色。复用 internal/ui，使用已有 readline 依赖确认交互能力，Windows 检查现有 VT 模式，其他平台仅接受已知 TERM 家族；NO_COLOR、TERM=dumb 和重定向禁色。
- 更新主帮助、Task 帮助、命令帮助、README 和 PowerShell/Bash/Zsh/Fish 补全。直接 apply_patch/Python 编辑沿用交接的 Git dubious ownership 限制，不重试 Git 或修改信任设置；Task/worktree/分支绑定由 task.toml、.git 指针及 worktree HEAD 静态核对。
- 本轮执行一次编辑后只读静态核对，检查参数分派、过滤/补建调用、表格布局、终端能力及帮助/补全链路。实际使用 Get-Content/Get-ChildItem/rg 读取、apply_patch 和 Python 编辑；编码与截断导致关键片段补读。静态核对发现 PowerShell 管道将 Python 脚本中的中文转为问号，已用 apply_patch 修复 README 和本段 Verification；补齐 NEEDS_DECISION 等待状态颜色。
- 未运行测试、编译、最终构建、格式化、lint、validator、网络、Git 写操作或实际列表/归档。compile-only 由 supervisor 执行冻结计划；建议接收本项 Evidence 后继续 1.4，不直接推进 Gate 或修改 Workflow Core 状态。
- %% RISK: 终端渲染及 shell 补全尚无运行验证；现有列表测试仍按固定列宽与旧缺失记录语义断言，须由 1.4 更新。本项仅完成 1.3，不代表整个 Task 完成。

- wi-0002 / 1.2：native、auto、openspec 统一进入 AIW 三处归档协调；沿用终止资格、工作区和显式 Git 选项检查。预检真实 Task 身份、活动/旧版/归档 Task 的 Session 独占绑定、运行中的 Work Item、未结束 Attempt、未释放租约、Task/Session 写锁、Session 状态和执行时间戳，以及所有移动目标。
- 规格、完整运行目录、真实绑定 Session 依次移动；任何后续失败逆序恢复本次已移动目录，不覆盖已有目标。OpenSpec 返回后核验实际归档目录，命令失败但已移动时也尝试恢复；无法唯一确认或补偿失败时报告路径和人工恢复方式。规格同步与已授权 Git 交付不属于目录补偿范围。
- 历史补归档使用原日期，跳过规格同步和 Git 交付；已完成的部分不重复移动、不纳入回滚。规格缺失提示“规格已删除”；绑定 Session 确实不存在时提示“会话记录缺失”，不重建 Session。运行元数据缺失无法证明终止资格时明确拒绝归档。
- Session 支持活动、新配对归档及旧独立归档的固定深度只读定位、重复候选及身份核验；不改写历史 JSON 和状态。ReadText 将旧 Session 绝对路径映射到核验后的目录；status/get/memory/handoff 查询从真实 Task 运行目录解析绑定。list 排除 archive 容器，执行、保存、更新、提示词、事件、工件、内存及删除入口拒绝归档 Session。
- 静态证据：核对 command/backend → archiveWithBackend → prepareTaskArchive → eligibility → move/rollback → 实际路径复核，以及 Store.Resolve → Load/ReadText/RequireRunnable/写入保护；保留单个 archiveRename 故障注入边界供 1.4 使用。
- 本轮实际执行 Get-Content、Get-ChildItem、rg 进行交接、规格、源码和静态调用链读取；apply_patch 与 Python 用于编辑。一个 Python 编辑命令因默认 cp932 解码失败，在未写入时改为显式 UTF-8 后重试；只读 rg 路径错误作了限定修正。沿用交接中 Git dubious ownership 已阻断的事实，不重试 Git 路径、不修改信任配置，使用直接文件编辑。
- 本轮未运行测试、编译、最终构建、格式化、lint、validator、网络、Git 写操作或实际归档；未修改 Workflow Core 工件、Gate、租约或 Task 生命周期。compile-only 由 supervisor 按冻结计划执行；后续建议验证本 Work Item 后继续 1.3，CLI 测试由 1.4 编写。
- %% RISK: 三处移动及恢复仍非原子，进程中断、外部并发写入、OpenSpec 非预期命名和补偿失败只有静态分析证据；不承诺并行归档能力。运行故障场景待后续明确授权，编译结论待 supervisor。

- wi-0001 / 1.1：增加固定深度的双侧发现与日期/完整 ID 归档定位；运行目录和规格重复候选、身份冲突及读取错误阻止补建。兼容旧根及 tasks.toml，规范文件名优先；重复运行记录作为冲突诊断。
- 列表使用发现的真实运行目录读取 Workflow 状态，缺失规格显示“规格已删除”；先过滤归档任务再补建。提供 includeArchived 收集入口供 1.3 接线，未实现 --all 参数及表格样式。
- 最小补建通过 taskx / Workflow Core 存储接口排他创建缺失文件，保留已有状态和元数据；不迁移目录、不创建 Session、Attempt、证据或租约，不从清单和归档位置推断完成历史。
- 静态核对范围：发现 → 过滤 → 元数据/状态核验 → 最小补建 → Workflow 摘要输出。既有 EnsureCompatible 的迁移和事件日志写入路径不用于列表。
- 本轮实际命令：Get-Content、Get-ChildItem、rg、Get-Command 用于只读定位；aiw patch --help 读取用法；Python 与 apply_patch 用于编辑。git status --short --branch 被 dubious ownership 拒绝，未重试或修改 Git 配置；因此使用直接文件编辑。
- 本轮不运行测试、编译、构建、格式化或 validator；compile-only 由 supervisor 按冻结的 scripts/compile.py 计划执行。未修改 Workflow Core 运行工件、Gate、租约或 Task 生命周期状态。
- %% RISK: 新的最小补建与冲突路径只有静态证据，编译待 supervisor；对应 CLI 测试由 1.4 编写。文件写入中断可能留下不完整文件，后续读取应报错而非覆盖修复。

- 规划依据：已静态读取任务列表、归档、后端委托、元数据定位及已有 CLI 测试；确认当前归档仅移动 change。
- 用户已确认沿用现有 CLI 层测试边界。
- 已补充用户提供的长任务名错位场景；静态读取现有 internal/ui/terminal.go，确认可复用终端输出与禁色策略，颜色和列宽尚未实现或运行验证。
- 已纳入用户故事 8、9：规格缺失正常显示，运行记录缺失自动最小补建；同步调整列表只读约束，参考现有 WorkflowRuntimeFromMeta 的兼容映射，不推断执行历史。
- 已纳入用户要求的关联 Session 归档；静态读取绑定检查、Session Archive 与 Load，确认当前旧归档位置和活动路径读取限制，新增只读定位、写入保护与补偿场景。尚未执行实际归档或测试。
- 工件采用仓库共享 spec-driven 模板结构；运行数据与 OpenSpec 工件使用同一 Task ID。
- 规划轮仅创建任务和规格；本实现轮完成 1.1，未执行测试、编译、构建或 Git 写操作。
- OpenSpec 指令读取受到 PowerShell 执行策略限制；未运行其 validator。结构通过静态核对，运行验证未完成。
- Workflow Core 映射由 aiw task workflow sync fix-paired-task-archive 在清单落盘后生成；以命令结果和运行记录为准。

## Gates and Evidence

- 规划 Evidence：源码行为、稳定 task-lifecycle spec、主工作区 ADR、用户确认的 CLI 测试边界。
- 映射 Gate：workflow sync 必须成功，且不得创建 Attempt、租约或推进执行。
- 验证 Gate：运行测试需要另行授权；规划完成不代表实现或运行验证完成。
- %% RISK: 故障恢复、进程中断和委托后端行为尚无本任务的运行证据，须在实现后记录验证结果。
