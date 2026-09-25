## 1. 实现

- [x] 1.1 在 list 调用链实现元数据驱动的候选发现、旧路径兼容、去重与稳定排序，排除无元数据目录。
- [x] 1.2 区分不存在、文件检查/读取错误和无效 Task 身份，输出带路径的诊断并继续其他记录，保留正常行和 RUNTIME_ERROR 语义。

## 2. 回归与说明

- [x] 2.1 添加 list 命令入口的离线回归用例，覆盖用户所列内部目录、任意普通目录、正常状态、空列表、兼容路径及重复身份。
- [x] 2.2 添加错误与混合记录用例，覆盖缺失/错误 ID、非文件元数据、读取故障和 Workflow 错误，检查输出通道、错误返回及未启动执行。
- [x] 2.3 更新列表使用说明和本 Change 的验证记录，明确元数据错误导致非零结果且有效 Task 仍会显示。

## TODO

上述五项为唯一实现清单，编写均已完成；本次仅完成 wi-0005（2.3）的列表使用说明与验证记录。勾选不代表编译或回归通过，也不代表 Core 已接受或整个 Task 已完成。下一步由 supervisor 按冻结 Compile Plan 编译并评估 Evidence；聚焦回归仍待后续授权。测试采用 list 命令入口作为主边界；不更改或清理用户现有运行目录。

## Verification

- wi-0005 文档证据：`README.md` 的 List Tasks 说明元数据发现、规范目录/旧目录及文件名优先级、去重排序、不递归扫描、无元数据目录与空列表、OpenSpec 工件缺失不隐藏 Task；明确元数据错误在 stderr 带路径和原因、有效 Task 继续输出到 stdout、最终非零及错误计数，并提醒脚本检查退出码。单独说明 Workflow 摘要失败的 RUNTIME_ERROR 行与原有成功返回语义，以及列表不启动执行或清理运行目录。
- wi-0005 静态核对：对照本 Change 的 proposal、design、task-lifecycle 增量规格及稳定规格中的兼容约束，追踪 `workflow.go` 的 `listTaskIDs`、`listTaskMetaPath`、`listTasks`，核对文档的排序、路径选择、错误通道与返回分支；未改变需求或设计决策，因此未同步修改稳定规格或 design。
- wi-0005 命令与边界：使用 `Get-Content`、`Get-ChildItem`、`Select-Object`、`rg` 读取交接、Task/worktree/HEAD 绑定、技能规则、规格、实现及 README。首次默认编码显示异常后改用 UTF8；通配路径及预估文档/源文件路径读取失败后改读实际路径。沿用历史 Git ownership 限制下的直接 `apply_patch` 回退，未重试 Git 或修改配置；修改后一次静态读取核对两个编辑区域。未运行测试、编译、构建、格式化、lint、vet、验证脚本、网络、子代理或 Git 写操作，未修改 Core 状态或解决 Gate。
- %% wi-0005 剩余验证：本轮仅完成文档和清单更新，没有新增运行结果。编译及有限修复循环由 supervisor 负责；须由其提供 Compile Plan 结果并确认是否覆盖 `_test.go`，不能以生产包编译替代测试验证。既有回归用例的执行、跨平台错误分类与 OpenSpec 结构验证仍未确认；后续获授权可运行 `go test ./internal/commands/task -run '^TestListCommand' -count=1`，本轮明确不运行。
- wi-0004 静态证据：`list_test.go` 新增命令入口混合用例，包含空文件、缺失/空白 ID、非法 ID、点路径 ID、目录身份不符、非普通文件及真实 Scanner 读取失败；逐字断言带绝对路径的 stderr、错误总数及包含前后有效记录和 RUNTIME_ERROR 的 stdout，确保损坏规范文件不会被旧目录或兼容文件名掩盖。独立 Workflow 错误用例断言原有成功返回和空 stderr 语义。
- wi-0004 只读证据：临时运行根在调用前后比较完整目录与文件内容快照，覆盖可执行 Work Item 的状态、事件日志和无 Workflow 状态的记录，防止列表创建 Attempt、写入执行状态或改变完成状态；沿用空 PATH 隔离外部工具。
- wi-0004 命令与边界：使用 `Get-Content`、`Get-ChildItem`、`Select-Object`、`rg` 读取交接、Task/worktree 绑定、技能规则、规格及实现；为确认错误输出和 Workflow 加载语义补充一次定向读取。沿用历史 Git ownership 限制的直接 `apply_patch` 回退。修改后一次静态读取复核；未运行测试、编译、构建、格式化、验证脚本、网络、子代理或 Git 写操作，未修改 Core 状态或解决 Gate。
- %% wi-0004 剩余验证：新增用例尚未执行；读取故障通过真实 Scanner 超长行错误稳定触发，未模拟操作系统权限/I/O 故障。编译由 supervisor 按冻结 Compile Plan 执行，需确认 `_test.go` 是否纳入。勾选仅表示 2.2 编写完成，不表示回归通过或整个 Task 完成；后续获授权可运行 `go test ./internal/commands/task -run '^TestListCommand' -count=1`。
- wi-0003 静态证据：新增 `internal/commands/task/list_test.go`，经 `DispatchTopLevel("list", nil)` 调用实际发现、解析和输出链；临时目录及空 PATH 隔离外部工具，分别捕获 stdout、stderr 和返回错误。完整行断言覆盖七类内部目录、任意目录、普通文件、空根、仅内部目录、嵌套元数据不递归、缺少 OpenSpec 工件、排序及无重复记录；内部名称带有效元数据仍列出，防止黑名单实现。
- wi-0003 状态与兼容证据：构造与元数据状态不同的 Workflow DRAFT / DONE，断言派生状态及既有列格式；覆盖旧目录下两种元数据文件名、规范目录的兼容文件名、规范/旧目录重复身份和同目录文件名优先级。
- wi-0003 命令与边界：使用 `Get-Content`、`Get-ChildItem`、`Select-Object`、`rg` 读取交接、绑定、规则、规格、入口及邻近测试；额外定向读取用于确认输出捕获、根目录隔离和 Workflow fixture 接口，部分预估路径/PowerShell 通配路径读取失败，未进行权限重试。沿用已记录 Git ownership 限制下的直接 `apply_patch` 回退；修改后仅一次静态读取复核。未运行测试、编译、构建、格式化、验证脚本、网络、子代理或 Git 写操作，未变更 Core 状态或解决 Gate。
- %% wi-0003 剩余验证：用例尚未执行；编译由 supervisor 按冻结 Compile Plan 执行，需留意其是否覆盖 `_test.go`。清单勾选仅表示 2.1 编写完成，不表示回归通过或整个 Task 完成。建议后续获授权运行 `go test ./internal/commands/task -run '^TestListCommand' -count=1`；本轮遵守禁止测试指令，不执行。
- wi-0002 静态证据：`listTaskMetaPath` 按规范目录、旧目录及 task.toml、tasks.toml 的既有优先级检查路径，仅在不存在时回退；检查失败和非普通文件保留路径诊断。`listTasks` 对读取失败、缺失/非法 ID、目录身份不符写 stderr 并继续；最后返回元数据错误计数。有效记录仍调用原有 Workflow 摘要及正常/RUNTIME_ERROR 行输出。
- wi-0002 命令：使用 `Get-Content`、`Get-ChildItem`、`Test-Path`、`rg`、`Select-Object`、`Select-String` 读取交接、Task 元数据、Git HEAD、技能/规则、规格及调用链；首次读取中文因默认编码显示异常，改用 UTF8；两个预估源文件路径不存在，随后定位实际文件。使用直接文件补丁，沿用前轮 Git ownership 阻止后的回退，不重试 Git 或修改配置。
- wi-0002 验证范围：仅静态检查修改区域、清单及错误/输出调用链；未运行测试、编译、构建、格式化、检查脚本、网络或 Git 写操作。编译由 supervisor 按冻结 Compile Plan 执行，未修改 Core 状态或解决 Gate。
- %% wi-0002 剩余验证：编译待 supervisor；错误与混合记录的运行结果待 2.1–2.2 回归验证。保留现有宽松元数据解析器，不声明完整 TOML 语法校验或整个 Task 完成。
- wi-0001 静态证据：`listTaskIDs` 合并规范根直接目录及旧根一层目录，按候选 ID 去重排序；`listTasks` 沿用 `ResolveTaskMetaPath` 的目录和文件优先级，跳过不存在的元数据，不依赖 OpenSpec 工件存在。正常状态及 RUNTIME_ERROR 输出调用保持原样。
- 本轮命令：使用 `Get-Content`、`rg`、`Get-Command` 读取交接、规格、代码及工具位置，读取 Git HEAD 核对当前分支；`aiw patch --help` 成功。`git status --short` 被 dubious ownership 检查阻止，未重试或修改 Git 配置；本轮回退至直接文件补丁。
- 本轮未运行测试、编译、格式化、检查脚本、网络或 Git 写操作；编译由 supervisor 按冻结 Compile Plan 执行，清单勾选仅表示所选实现已完成，不表示编译或回归通过。
- wi-0001 历史限制：当轮保留读取错误 UNKNOWN 行行为；此项已由 wi-0002 实现替代，回归验证仍待完成。
- 静态证据：用户提供的错误列表、listTasks 调用链、Task 元数据解析及 task-lifecycle 稳定规格。
- AIW 创建：`aiw new fix-task-list-discovery --backend auto --allow-unrelated-dirty`，主工作区创建，无实现工作树。
- 规划收尾：`aiw task workflow sync fix-task-list-discovery` 成功同步五项 Work Item，全部 authored=open / core=ready；Workflow 为 DRAFT / queued，未启动执行。
- 未运行：测试、编译、模型调用、Git 写操作。OpenSpec PowerShell 入口已有执行策略阻止记录，本次不重复失败命令或绕过策略；尚未完成 CLI instructions / validate。

%% 验证待办：在允许的环境执行 OpenSpec 结构验证；实现后经授权运行聚焦 list 回归，不能将规划文档视为 Bug 已修复。
