## Context

当前 `listTaskIDs` 枚举 `.ai` 直接子目录并兼容 `.ai/tasks`；`listTasks` 从活动位置读取运行状态并输出活动 change 路径。原生 `archiveTask` 仅移动 change；委托后端直接调用 OpenSpec archive，也没有配对移动运行数据。

实现依据：`internal/commands/task/workflow.go`、`backend.go`、`command.go`、`list_test.go`、`command_test.go` 和 `internal/taskx/meta.go`。遵循主工作区默认 ADR；本任务不需要创建工作树。

## Design Readiness

FD_APPLIED。已收敛配对路径、受控深度扫描、失败恢复、后端一致性、历史兼容和记录缺失容错策略。用户已确认沿用现有 CLI 层测试边界，并要求运行记录缺失时自动补建；这构成列表只读规则的唯一例外。本设计只规划未来实现，不在规格阶段迁移数据或修改 CLI。

## Goals / Non-Goals

**Goals:** 配对归档、可靠识别历史归档、容忍规格缺失、自动补建缺失运行记录、明确失败信息、保留 Workflow Core 的状态语义。

**Non-Goals:** 任意深度搜索、全局 Session 迁移、新的 Task 状态枚举、自动覆盖损坏或冲突记录、恢复执行归档任务、并行归档能力。

## Decisions

### 1. 配对存储与身份

活动路径保留 `.ai/<id>/` 和 `openspec/changes/<id>/`。归档路径为 `.ai/archive/<date>-<id>/` 与 `openspec/changes/archive/<date>-<id>/`，同一次操作只生成一个日期。整个任务运行目录移动，保留元数据、状态、证据和任务本地记录；Task ID 与 Session 引用不变。实际绑定的 Session 纳入同一次协调归档。

归档是存储生命周期信息，不将 Workflow 状态改为 ARCHIVED。读取归档记录必须使用发现的真实运行目录，不能回落到活动目录或通过兼容初始化重新创建活动任务。

### 2. 归档协调和失败边界

沿用现有完成、交付、工作区、规格同步和显式 Git 选项的资格检查。在移动前验证 Task 与 Session 身份、所有本次移动目标均不存在，以及任务和会话均无活动写入；未完成执行或未释放租约不得归档。

将配对移动的协调放在 AIW 层，覆盖 native、auto 和显式 openspec。OpenSpec 后端继续负责其规格操作，AIW 必须确定并核验它实际生成的归档路径。只有规格、任务运行目录及存在的关联 Session 均完成归档且身份一致才报告成功；Session 缺失采用下述明确提示的容错规则。

任意后续移动失败时按相反顺序恢复本次已经移动的目录，包括 Session；恢复失败则返回非零并报告所有实际路径和下一步恢复方式，不删除数据、不覆盖冲突目标。移动恢复不承诺撤销此前已经完成的规格同步或显式 Git 操作。重复调用已完成的配对归档应无操作成功，不产生第二份档案；此前已经归档的部分不属于本次回滚范围。

### 2.1 Task 绑定 Session 的归档

根据 `task.toml.session` 查找 `.ai/sessions/<session-id>/`，不假设 Session ID 必须等于 Task ID。当前默认二者同名，`validateTaskBindings` 拒绝活动 Task 重复绑定，但没有全局唯一性保证。归档前检查支持的活动、旧版和归档 Task 记录；重复绑定或 Session 内 TaskID 与本 Task 冲突时拒绝移动，不将共享 Session 作为正常业务。

关联 Session 的新归档位置为 `.ai/sessions/archive/<date>-<task-id>/<session-id>/`，与 Task 和 change 使用同一归档名称。整个会话目录移动，保留 status.json、memory、instructions、turn 和输出；Session ID、后端线程 ID、Task/Attempt 引用及历史结果不变。若路径字段引用旧绝对目录，读取时按核验的归档映射解析，不改写历史内容。

Task 满足既有终止资格且 Session 无运行中状态、进程或写入占用时，按存储位置归档并保留原 Session 状态；不把 failed/paused 等状态伪造为 completed。所有执行与写入入口必须拒绝归档位置的 Session。running 或仍有进程/写入占用时，在任何移动前拒绝归档。

当前 `session.Store.Archive` 将 Session 写到 `.ai/archive/<session-id>/` 且先修改状态，`Load` 仅查活动位置。Task 协调归档不得直接串联该方法造成提前改状态或遗漏补偿；应复用下层存储能力，提供归档位置的只读解析。保持独立 session archive 命令既有契约，不批量迁移无关会话。旧位置通过 status.json 的 Session ID 与 Task 关联核验后兼容读取，不能误识别为 Task 归档。

Task 关联的 session status/get 及其他既有只读读取应能由归档 Task 元数据定位新旧归档 Session；Session list 必须排除 archive 容器。无绑定则无需会话移动；有绑定但所有合法位置均不存在时提示“会话记录缺失”并继续归档，不创建虚假 Session 或线程。损坏、权限错误、重复候选不是普通缺失，不跳过。

历史 Task 已归档但 Session 留在活动位置时，再次显式 archive 补齐会话归档，沿用原 Task 归档日期；已在新位置的会话不重复移动，已在旧独立归档位置的会话核验后可在本次显式修复中移动到关联归档位置。list 不搬动会话。最小运行记录补建不包含 Session 历史重建或根据同名目录猜测绑定。

### 3. 列表的受控发现与输出

扫描固定根下的一层任务目录：活动 `.ai`、旧版 `.ai/tasks`、归档 `.ai/archive`，并合并活动与支持的归档 change 根中的规格候选；排除 archive 容器本身。以元数据文件或合法 change 目录身份校验候选，兼容 `tasks.toml`，保留规范目录优先规则。不得递归进入 session、证据等子目录。

默认列表排除已归档任务并保留 ID、Workflow 状态、真实 change 路径三个字段及顺序。`--all` 增加归档标识（ACTIVE/ARCHIVED），按 Task ID 稳定排序。除下述运行记录缺失补建外不修改记录；不得为了列表创建 Attempt、租约或启动执行。规格缺失不是错误；损坏、读取权限失败与身份冲突仍须如实诊断。

### 3.1 规格缺失和运行记录补建

“规格记录”指该 Task 对应的 change 记录，不以任一单独 Markdown 缺失作为整个规格已删除的依据。优先检查活动和归档位置：移动到 archive 不等于删除。仍有运行记录但找不到匹配 change 时，保留该行，在 PATH 单元格显示“规格已删除”；不输出无效规格路径、不仅因此返回非零、不自动重建规格。归档分类沿用运行记录的实际位置；孤立活动运行记录仍在默认列表可见。

“运行记录缺失”覆盖整个运行目录、task.toml 或核心 state.json 缺失。先按既有兼容优先级检查旧目录及 tasks.toml，再判断是否确实不存在。对于本次列表范围内、身份唯一且有规格或剩余有效运行记录作为依据的任务，通过 AIW/Workflow Core 存储接口仅创建缺失部分。先完成归档过滤再补建：普通 list 不为被隐藏的归档任务写入；--all 可以在对应归档运行目录补建，沿用规格的归档名称，不复活到活动根。

保留已有合法元数据和状态中的可用信息；没有依据的 branch、parent_branch、Session、交付与历史结果不猜测。全量丢失时建立最小 DRAFT 记录，工作区使用既有 unknown/unassigned 语义，不自动绑定当前分支、不从 checklist 或归档位置推断 DONE/CANCELLED。不生成 Attempt、证据、写租约或执行结果。表格之外的补建提示写入 stderr，明确说明这是最小重建、历史状态可能丢失；stdout 保持既定列结构。

补建必须幂等，只创建缺失文件，不覆盖并发出现的文件、不更改已有清单完成状态。重复 list 不重复写入。运行记录正常时保持只读。内容损坏、权限错误、多个候选以及目录身份不明不视为缺失，不自动覆盖；补建确实失败时给出明确诊断并继续其他任务，不谎报修复成功。规格和运行依据均消失时无法发现 Task，不凭空创建。

### 3.2 动态对齐与颜色

先收集可显示的行，再按每列完整内容的可见宽度计算最大值，列间至少两个空格；不能继续使用固定 24 字符最小宽度，也不能把 ANSI 转义序列计入宽度。任务 ID 和状态不截断，路径放在最后且保留完整值。较窄终端允许自然换行，不引入本轮之外的自适应卡片布局。

交互终端默认显示 `TASK / STATUS / PATH` 表头；`--all` 显示 `TASK / STATUS / ARCHIVE / PATH`。不使用边框或逐行分隔线。管道、文件重定向和无法确认交互能力的输出保持无表头的数据行，保留字段顺序；列间空格可随内容变化。空结果不输出悬空表头，保持既有空列表行为。

优先复用 `internal/ui/terminal.go` 的终端与禁色策略，不引入依赖。仅在支持 ANSI 颜色的交互终端着色；非空 `NO_COLOR`、`TERM=dumb`、管道和文件输出禁色。终端检测不能仅凭字符设备判断就假定支持 ANSI；无法确认时回退纯文本。

颜色只作辅助，始终保留完整状态文本：DONE 为绿色，DRAFT 为灰色，READY/执行中为青色，等待/待验证为黄色，失败/阻塞/RUNTIME_ERROR 为红色，CANCELLED 和 ARCHIVED 为灰色。其他状态保留默认文字颜色。ACTIVE 保持默认颜色，路径保持默认颜色；不得因整行淡化降低错误可读性。填充空格不着色，单元格结束后复位。

纯文本布局示例（交互表头）：

```text
TASK                            STATUS  PATH
fix-task-list-discovery          DONE    openspec/changes/fix-task-list-discovery
improve-requirement-management   DONE    openspec/changes/improve-requirement-management
requirement-artifact-generation  DRAFT   openspec/changes/requirement-artifact-generation
workflow-automation             DRAFT   openspec/changes/workflow-automation
```

### 4. 历史半归档数据

若运行数据仍在活动路径，但活动 change 不存在，且在 `openspec/changes/archive` 或旧版 `openspec/archive` 找到唯一匹配的归档 change，则视为历史已归档任务。匹配必须核验日期前缀与完整 Task ID，不做模糊后缀匹配。默认列表隐藏它，`--all` 显示其真实 change 路径，不移动任何文件。

再次显式调用 `aiw archive <id>` 时，补齐缺少的运行目录及关联 Session 归档；保留原归档日期，不重做规格同步和 Git 交付。该路径仍需身份、终止资格和无活动写入检查。

多个匹配、活动与归档 change 并存或重复运行记录必须输出明确诊断，不猜测、不覆盖。不完整档案按记录缺失规则显示或补建。普通孤立运行记录没有唯一归档证据时不能被静默判为已归档。读取旧归档根用于兼容，不在列表期间迁移它。

### 5. 现有测试边界

优先沿用 `internal/commands/task` 的临时仓库和 CLI 分派测试。观察目录、输出、退出结果及文件内容保持情况；沿用现有文件操作测试能力，必要时只对配对移动增加最小故障注入边界。委托后端使用本地替身，不联网、不依赖外部 OpenSpec 安装。测试代码属于实现工作，运行需要单独授权。

在相同边界补充用户给出的长 ID、长 Workflow 状态、三列/四列布局、交互表头、空结果、颜色复位及 NO_COLOR/TERM=dumb/重定向场景；验证去除 ANSI 后各列起点一致，纯文本输出不含转义序列。

补充规格缺失正常返回、仅规格任务发现、运行目录/元数据/状态分别缺失、最小补建幂等、旧记录优先、归档位置补建、默认过滤不写归档记录，以及损坏/权限错误不被当作缺失的场景。仍使用现有 CLI 层，不额外运行测试。

## Risks / Trade-offs

- 三处源目录无法一次原子移动：使用预检和补偿恢复；进程中断可能留下待恢复记录，重试必须识别而非覆盖。
- Session 与 Task 原归档共用 `.ai/archive` 根且读取仅查活动位置：按文件身份区分旧记录，补充新旧位置的只读解析及归档写入保护。
- OpenSpec 版本可能改变目标命名：必须核验实际目标，无法唯一确认时失败并报告路径。
- 历史多重匹配不得自动选取：输出诊断，要求人工处理冲突。
- `--all` 的归档状态读取必须避免调用会写入活动根的兼容路径。

## Migration Plan

先实现双侧候选定位、历史分类和最小补建，再接入配对归档与显式修复，最后增加列表选项与文档。不执行批量迁移；旧半归档记录可以直接被 `--all` 查阅，之后逐个显式修复。缺失运行记录只在列表选中范围内按需补建。

## Open Questions

%% RISK: 进程中断与补偿恢复失败只能通过故障场景获得运行证据；本轮不运行测试，实现完成后需单独授权聚焦测试。
%% RISK: 自动补建无法恢复已经丢失的执行历史和证据；必须保守展示并提示，不将新记录等同于原记录完整恢复。
%% VERIFICATION: OpenSpec 指令读取受到本机 PowerShell 执行策略限制；本轮使用仓库 internal/openspecgen/generator.go 的共享模板结构，未运行 OpenSpec validator。
