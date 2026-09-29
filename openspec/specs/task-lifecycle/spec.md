# Task 生命周期

## Purpose

固定 Task、OpenSpec 工件及 Workflow Core 的职责与当前生命周期行为，覆盖创建、元数据兼容、状态派生及归档边界，作为后续变更比较现有行为的依据。
## Requirements
### Requirement: Task 活动存储与旧位置诊断

系统 MUST 将活动 Task 元数据和运行信息归于 `.ai/tasks/<id>/`。新原生 Task 的计划与工作项归于 `docs/features/<id>.md`，OpenSpec change 是可选关联。旧 Task 的 `openspec/changes/<id>/tasks.md` 仍可作为计划来源。系统 MUST 保留旧元数据文件名 `tasks.toml` 的读取兼容性。旧活动目录 `.ai/<id>/` MUST NOT 被正常读取、写入、自动迁移或用于同 ID 的新建/补建；发现旧目录或迁移标记时 MUST 返回可操作的手工迁移诊断。Task 归档位置 MUST 保持 `.ai/archive/<date>-<id>/`。

#### Scenario: 创建和读取活动 Task

- **WHEN** Task 位于 `.ai/tasks/<id>/`
- **THEN** 元数据、Workflow 状态及运行记录均从该目录解析。

#### Scenario: 发现旧活动目录

- **WHEN** Task 只存在于 `.ai/<id>/` 或新位置含有迁移标记
- **THEN** 正常运行停止并说明旧目录及目标路径，等待操作者在所有 AIW 写入者停止后调整完整目录。
- **AND** 系统不自动搬动、合并、覆盖或重建该 Task。

#### Scenario: 归档位置不变

- **WHEN** Task 被归档
- **THEN** 运行数据仍存放于 `.ai/archive/<date>-<id>/`。

### Requirement: 原生创建预检与初始状态

原生 new MUST 从主工作区创建，校验 ID 和目标不存在，执行脏路径预检。与目标重叠的脏路径 MUST 拒绝；无关脏路径只有显式 `--allow-unrelated-dirty` 才可放行。

#### Scenario: 创建普通 Task

- **WHEN** 原生 new 的预检通过
- **THEN** 系统创建 FD、Task 元数据和工作项映射，不要求 OpenSpec change。
- **AND** 初始 Task 为 TODO，工作区为 primary / `.`，branch 与 parent_branch 为当前分支，delivery 为 unmanaged。
- **AND** 创建清单映射不派发 Agent、不启动实现。

#### Scenario: 脏路径触及目标

- **WHEN** 目标 Task 或变更目录存在未提交的重叠路径
- **THEN** 创建被拒绝，即使传入允许无关脏路径的选项也不能覆盖它们。

### Requirement: OpenSpec 后端选择

适用命令的 backend MUST 支持 auto、native、openspec。native MUST 直接走本地实现。auto 在 OpenSpec 不可用或该操作没有直接映射时 MUST 明确回退；显式 openspec 对不可用或不支持的情况 MUST 报错。

省略 `--backend` 时 MUST 默认 native。只有显式 auto 或 openspec 才可委托 OpenSpec。

#### Scenario: 未安装 OpenSpec

- **WHEN** 使用 auto 创建 Task，而无法找到可用 OpenSpec
- **THEN** 系统提示 native fallback 并执行原生路径。

#### Scenario: 强制 OpenSpec

- **WHEN** 用户显式选择 openspec 但该后端不可用
- **THEN** 系统返回错误，不静默改变所选后端。

### Requirement: 状态由 Workflow Core 映射与派生

status/done MUST 由 Core 事实派生并分开 planning、execution、validation、delivery、workspace。Work Item 接受服从 SW14；缺必需验证时不能以 checkbox/旧 metadata/waived 派生完成。开发完成事实保存后立即登记通知和汇总，Git、知识、Verifier 的状态单独展示。

#### Scenario: 辅助结果未结束
- **WHEN** 所有实现项已按当前证据接受而辅助队列仍有缺口
- **THEN** 开发完成保持成立，辅助失败不撤销它；交付仍可 pending。

### Requirement: 归档前置条件与显式清理

archive MUST 检查 DONE / CANCELLED 的终止资格、工作区绑定和交付状态。主工作区 Task MUST 拒绝受管理工作树/分支的 finalize 选项；隔离 Task 的归档 MUST 核验交付和清理结果。已交付且资源已清理的 Task MUST 支持使用核验后的主工作区工件归档，不再读取失效的隔离路径，也不要求重复删除资源。push MUST 由显式选项触发。

#### Scenario: 工作区归属未知
- **WHEN** archive 无法确认 Task 的工作区绑定或历史交付及清理依据
- **THEN** 系统要求先修复绑定或补齐证据，不继续归档。

#### Scenario: 主工作区归档要求删除分支
- **WHEN** primary Task 使用 cleanup-wt、delete-branch 或 finalize
- **THEN** 系统拒绝，因为该 Task 没有对应受管理工作树或分支可清理。

#### Scenario: 完成并交付后工作树已删除
- **WHEN** Core 确认 DONE、交付为 merged、历史隔离资源均已清理，且主工作区 FD 或旧 change 清单可用
- **THEN** native 和 openspec 归档优先从 FD 同步工作项；没有 FD 时才读取旧 change 清单，不访问已删除工作树。

### Requirement: Task 配对归档

系统 MUST 将同一 Task 的受管 FD、任务运行目录、存在的 OpenSpec change 及绑定 Session 作为一组归档，以相同日期和完整 Task ID 关联各归档位置。FD 归于 `docs/features/archive/<date>-<id>.md`；没有 OpenSpec change 不阻止归档。系统 MUST 保留原任务身份、Workflow 状态、证据及 Session 引用，并保持既有归档资格和 Git 授权要求。原生与委托后端 MUST 满足同一契约。

#### Scenario: 成功归档

- **WHEN** 满足归档条件的 Task 执行 archive
- **THEN** Task 运行目录、存在的 FD 和 OpenSpec change 分别移动到对应的归档根，并具有相同日期与 Task ID
- **AND** 存在的绑定 Session 一并归入同一日期与 Task ID 对应的 Session 归档目录
- **AND** 原有运行数据被保留，活动位置不再保留该任务目录

#### Scenario: 委托后端

- **WHEN** archive 使用可用的 OpenSpec 后端
- **THEN** AIW 核验 OpenSpec 的实际归档目标并完成运行目录及存在的绑定 Session 归档
- **AND** 不允许只完成 change 移动就报告成功

#### Scenario: 目标冲突或任务仍在写入

- **WHEN** 目标已存在、身份冲突或任务仍有活动写入
- **THEN** 归档被拒绝且不覆盖任何记录

#### Scenario: 移动失败

- **WHEN** 规格、运行目录或 Session 中任意后续移动失败
- **THEN** 系统按相反顺序尝试恢复本次已经移动的目录并返回失败
- **AND** 恢复失败时明确报告实际位置，不删除记录或虚报配对成功

#### Scenario: 重复归档

- **WHEN** 相同任务已经完整配对归档且没有身份冲突
- **THEN** archive 无操作成功，不创建第二份归档

### Requirement: 列表发现与归档可见性

`aiw list` MUST 只显示未归档任务。`aiw list --all` MUST 同时显示未归档与已归档任务，分别展示 Workflow 状态、归档标识和现存 change、FD 或 Task 运行记录路径。发现过程 MUST 合并可选 change 与运行记录候选，限定在支持的固定目录层级，兼容既有元数据路径和文件名。除缺失运行记录的最小补建外，列表 MUST NOT 修改运行数据。

#### Scenario: 默认列表

- **WHEN** 活动与归档任务同时存在且用户运行 list
- **THEN** 仅显示未归档任务，并保留原未归档行的三个字段及其顺序，列宽按内容调整

#### Scenario: 显示全部

- **WHEN** 用户运行 list --all
- **THEN** 按 Task ID 稳定排序，输出 ID、Workflow 状态、ACTIVE/ARCHIVED 标识及真实 change 路径
- **AND** 归档任务的原完成或取消状态不被归档标识替代

#### Scenario: 深层元数据

- **WHEN** session 或其他不受支持的深层目录含有 task.toml
- **THEN** 该记录不被作为任务枚举

#### Scenario: 读取归档运行状态

- **WHEN** 列表读取已经移动的任务运行记录
- **THEN** 使用真实归档运行位置，不创建活动目录、Attempt 或租约
- **AND** 完整记录保持只读，缺失记录仅按最小补建规则写入归档运行位置

### Requirement: 列表对齐与终端配色

list 和 list --all MUST 按本次输出内容的可见宽度对齐列，列间至少两个空格，且 MUST 保留完整 Task ID、状态文本和路径。支持颜色的交互终端 SHOULD 使用状态配色，颜色 MUST NOT 替代文字。交互输出 MUST 显示表头；非交互输出 MUST 保持无表头数据行且不包含 ANSI 控制序列。非空 NO_COLOR、TERM=dumb 或无法确认颜色能力时 MUST 禁用颜色。

#### Scenario: 长任务名与状态

- **WHEN** 任务 ID 超过 24 个字符或存在较长 Workflow 状态
- **THEN** 所有数据行的状态列、归档列和路径列分别对齐，长内容不截断
- **AND** 开启颜色不会改变可见列宽和列起点

#### Scenario: 交互终端

- **WHEN** 用户在支持颜色的交互终端运行列表且结果非空
- **THEN** 默认表头为 TASK、STATUS、PATH，--all 增加 ARCHIVE 表头
- **AND** DONE 为绿色、DRAFT 为灰色、等待状态为黄色、失败或阻塞为红色，未知状态保留默认颜色和完整文字
- **AND** 每个着色单元格结束后复位颜色

#### Scenario: 禁色和非交互输出

- **WHEN** 输出重定向至文件或管道，或设置非空 NO_COLOR、TERM=dumb，或终端颜色能力不确定
- **THEN** 输出不含 ANSI 转义序列，字段仍然对齐
- **AND** 管道、重定向或无法确认交互能力时不增加表头，保持既有字段顺序

#### Scenario: 空列表

- **WHEN** 没有符合过滤条件的任务
- **THEN** 不输出悬空表头并保留既有空列表行为

### Requirement: 历史半归档兼容与显式修复

系统 MUST 识别运行目录仍在活动根、活动 change 缺失且存在唯一匹配归档 change 的历史任务。list MUST NOT 移动已有记录，完整记录 MUST 保持只读；再次显式 archive MUST 能补齐运行目录配对，并沿用原归档日期。冲突和歧义 MUST 被报告而非猜测处理。

#### Scenario: 列出历史任务

- **WHEN** 唯一匹配 change 位于当前或旧版归档根而运行目录仍在活动位置
- **THEN** 默认列表隐藏该任务，--all 显示 ARCHIVED 和真实归档路径
- **AND** 文件位置及内容不变

#### Scenario: 修复历史配对

- **WHEN** 用户显式归档唯一匹配的历史半归档任务且终止资格与安全检查通过
- **THEN** 补齐缺少的运行目录或绑定 Session 归档，保留原日期、身份及数据
- **AND** 不重做规格同步或 Git 交付

#### Scenario: 多重匹配与孤立记录

- **WHEN** 存在多个归档匹配、重复运行记录或活动与归档 change 冲突
- **THEN** 系统报告诊断且不选择其中一份进行修复
- **AND** 没有唯一归档证据的孤立运行记录不被静默标为已归档

### Requirement: Task 关联 Session 归档与查询

Task archive MUST 将存在且可定位的真实 Session 绑定归档到 `.ai/sessions/archive/<date>-<task-id>/<session-id>/`，保留 Session 身份、原状态和历史内容。Session 是可选 Task 附件；未绑定或无法定位时，系统 MUST NOT 为归档创建占位 Session，也不得因此阻塞 Task 归档。存在 Session 时，系统 MUST 检查活动、旧版及归档 Task 的重复绑定，并核对 Session 的 Task 关联。归档 Session MUST 能通过 Task 关联的既有只读接口查询，MUST NOT 被继续执行或写入。无关独立会话 MUST NOT 被移动。

#### Scenario: Session ID 与 Task ID 不同

- **WHEN** Task 绑定一个独占、无运行进程或写入占用的 Session，且归档资格通过
- **THEN** 移动实际绑定的会话目录，保留原 Session ID 和 Task 引用
- **AND** 不按 Task 同名目录猜测绑定，不把失败或暂停状态伪造为 completed

#### Scenario: Session 正在运行或绑定冲突

- **WHEN** Session 为 running、仍有执行/写入占用，或存在重复绑定及身份冲突
- **THEN** 在任何归档移动前拒绝操作并报告原因

#### Scenario: Session 可选且缺失

- **WHEN** Task 未绑定 Session，或绑定的 Session 在活动及合法归档位置均找不到
- **THEN** 不创建占位 Session、不输出缺失错误，并继续处理规格及运行目录归档
- **AND** 存在且可定位的 Session 才执行绑定校验和会话归档；权限错误及损坏记录不得被当作缺失跳过

#### Scenario: 归档后只读查询

- **WHEN** 通过 Task 查询其已归档 Session 的 status、get 或其他既有只读内容
- **THEN** 由归档 Task 绑定定位真实会话位置，返回保留的内容
- **AND** 兼容核验过身份的旧 `.ai/archive/<session-id>/` 会话，拒绝归档会话的执行与写入

#### Scenario: 历史会话补归档与重试

- **WHEN** Task 已归档而关联 Session 仍在活动或旧独立归档位置，用户再次显式 archive
- **THEN** 沿用原 Task 归档日期补齐会话归档，已完成的新位置不重复移动
- **AND** 任一步骤失败仅补偿本次移动，list 不搬动或重建 Session 历史

### Requirement: 可选规格缺失时正常显示

运行记录存在但没有对应 change 时，列表 MUST 保留该任务行并优先显示现存 FD 路径；FD 也不存在时显示 Task 运行记录路径。系统 MUST NOT 仅因 change 缺失使列表失败，也 MUST NOT 自动重建规格。

#### Scenario: 规格记录已经删除

- **WHEN** 列表范围内的 Task 仍有运行记录，但其 change 记录不存在
- **THEN** 显示任务状态及现存 FD 或运行记录路径，继续显示其他任务
- **AND** 没有其他错误时命令正常返回，不显示不存在的 change 路径

#### Scenario: 规格只是归档或部分文档缺失

- **WHEN** change 存在于支持的归档根，或 change 记录存在但某个 Markdown 文件缺失
- **THEN** 不将整个规格标记为已删除，仍按归档过滤与真实位置显示

### Requirement: 自动补建缺失运行记录

列表 MUST 从规格和有效运行记录中发现身份唯一的 Task，在本次列表范围内通过 AIW/Workflow Core 存储接口自动补建缺失的运行目录、元数据或核心状态。补建 MUST 保留已有有效数据、兼容旧位置，幂等且不覆盖已有文件。系统 MUST NOT 推断丢失的完成结果、分支绑定或执行历史，MUST NOT 因补建启动任务或创建 Attempt、证据和租约。

#### Scenario: 只有规格记录

- **WHEN** 合法且唯一的 change 存在，但运行目录完全缺失
- **THEN** 自动创建最小 DRAFT 运行记录并正常显示该任务
- **AND** 未知工作区和历史信息保持未确定，通过 stderr 提示最小重建及历史信息可能丢失

#### Scenario: 运行记录部分缺失

- **WHEN** task.toml 或 state.json 缺失且有唯一身份依据
- **THEN** 仅补建缺失部分，保留其他有效运行数据，不因该缺失报错
- **AND** 优先使用受支持的旧目录或 tasks.toml，不创建重复任务

#### Scenario: 归档任务补建

- **WHEN** --all 发现归档规格存在而其运行记录缺失
- **THEN** 在配对的归档运行位置补建，保留原归档名称和 ARCHIVED 标识
- **AND** 不因规格已归档就推断原执行状态为 DONE，也不创建活动任务目录

#### Scenario: 重复读取与默认过滤

- **WHEN** 再次列出已成功补建的任务，或默认 list 排除了归档任务
- **THEN** 不重复写入完整记录，也不为被过滤掉的归档任务补建

#### Scenario: 损坏、歧义或写入失败

- **WHEN** 记录损坏、身份存在歧义、读取无权限或补建写入失败
- **THEN** 如实报告具体诊断并继续处理其他任务，不将其伪装为普通缺失或修复成功
- **AND** 不覆盖既有数据；无规格及运行依据的任务不被凭空创建

### Requirement: 清理结果同步当前绑定

系统 MUST 在成功移除隔离工作树后先将当前绑定设为 unassigned、worktree 置空，并保留历史分支、父分支、Session 和交付证据。若 Task 未完成且分支清理成功，系统 MUST 将其恢复绑定到已验证的 primary 工作区继续推进；已完成 Task 保持 unassigned。清理失败 MUST 保留符合实际结果的状态并允许恢复。

#### Scenario: 分支清理失败
- **WHEN** 合并和工作树移除成功，但分支删除失败
- **THEN** delivery 保持 merged，当前工作树绑定解除，历史分支保留，并可重试未完成步骤。

### Requirement: 历史终止任务的幂等修复

显式修复 MUST 依据 Core 派生状态及实际资源检查处理活动和归档 Task，保留未知字段、原位置及修复前快照。无法确定身份、终止状态、占用或清理结果时 MUST 跳过并诊断。修复 MUST NOT 改写 Core 执行事实、凭清单推断完成或自动执行 Git 写操作。

#### Scenario: 已完成任务残留旧元数据
- **WHEN** Core 为 DONE 且 merged，工作树目录、注册和临时分支均不存在，但 task.toml 为 TODO/pending/isolated
- **THEN** 修复为 DONE/merged/unassigned，清空 worktree 并保留历史字段；再次修复不写入文件。

#### Scenario: 已归档主工作区任务
- **WHEN** 归档目录中的 Core 确认 DONE，元数据仍为 TODO，交付为 unmanaged
- **THEN** 就地同步完成快照，保留 primary/unmanaged，不复活活动目录。

#### Scenario: 尚未完成或证据不足
- **WHEN** 存在待验证、待授权、阻塞、活动执行或缺失损坏的 Core
- **THEN** 不将 Task 修正为 DONE，不解除绑定，报告跳过原因。

### Requirement: Task 列表按元数据发现身份

`aiw list` MUST 仅将受支持位置中具有 Task 元数据的记录识别为 Task。系统 MUST 忽略没有 task.toml 或兼容 tasks.toml 的普通目录，MUST NOT 通过硬编码内部目录名黑名单实现识别，MUST NOT 为这些目录输出 UNKNOWN 行或推导的 Change 路径。

#### Scenario: Task 与内部运行目录混合

- **WHEN** `.ai` 同时包含有效 Task 和 compile-cache、issue、locks、requirements、sessions、tasks、tmp 等无 Task 元数据的目录
- **THEN** 正常列表只包含有效 Task，普通目录不会被展示为 Task。

#### Scenario: 未来新增运行目录

- **WHEN** 任意未预先列入规则的新目录不包含 Task 元数据
- **THEN** 系统同样忽略它，无需更新名称排除表。

#### Scenario: 没有 Task

- **WHEN** 可读运行根只含内部目录或普通文件
- **THEN** 列表为空且成功，不生成 UNKNOWN 记录。

### Requirement: 列表保留元数据兼容与唯一性

Task 列表 MUST 支持现有规范路径、旧 `.ai/tasks/<id>/` 位置及 tasks.toml 文件名，遵循既有规范位置和文件优先级，按身份去重并稳定排序。系统 MUST NOT 递归发现任意运行子目录，也 MUST NOT 因 OpenSpec 工件缺失而隐藏具有有效元数据的 Task。

#### Scenario: 仅有旧记录

- **WHEN** Task 仅有旧位置或兼容文件名的有效元数据
- **THEN** 系统列出该 Task，而不把旧容器目录自身列为 Task。

#### Scenario: 同 ID 存在两份记录

- **WHEN** 规范与旧位置均存在同 ID Task
- **THEN** 仅按规范路径优先级处理一次，不重复列出，不修改任一记录。

### Requirement: 列表区分无记录与元数据错误

不存在元数据 MUST 被作为非 Task 跳过；元数据不可读、路径类型错误、缺失身份、非法身份或 ID 与候选目录不符 MUST 被报告为错误，包含实际路径及原因，MUST NOT 作为正常 UNKNOWN Task 输出。系统 MUST 继续展示其他有效 Task，并在存在元数据错误时返回非零结果。根枚举失败 MUST 报错。

#### Scenario: 有效 Task 与损坏记录混合

- **WHEN** 一项元数据缺 ID，而另一项有效
- **THEN** 有效 Task 正常输出，错误记录在 stderr 被诊断，最终命令返回错误。

#### Scenario: 读取权限或 I/O 错误

- **WHEN** 候选元数据检查或读取遇到非不存在错误
- **THEN** 系统报告原因，不静默跳过或回退到旧副本。

### Requirement: 正常列表保留 Workflow 摘要

有效 Task 行 MUST 保留现有身份、Workflow 派生状态和 PATH 列的展示格式；Workflow 摘要失败 MUST 保留现有 RUNTIME_ERROR 行语义。列表修复 MUST NOT 启动 Attempt、派发 Agent 或修改任务完成状态。

#### Scenario: 有效 DRAFT 与 DONE Task

- **WHEN** 两项有效 Task 的 Workflow 分别派生 DRAFT 和 DONE
- **THEN** 列表仍分别显示对应状态，不以元数据静态状态替代。

#### Scenario: Workflow 摘要读取失败

- **WHEN** Task 身份有效但 Workflow 摘要无法读取
- **THEN** 保留该 Task 的 RUNTIME_ERROR 行，不误分类为普通目录。

### Requirement: SW34 生命周期及设计闭合

系统 MUST 满足以下规则：本批维持 REQ00001-workflow-automation → workflow-automation 的一个 Requirement、一个 Task/change 生命周期，按 E01–E08 分组。推广后的 proposal、design、specs 和 tasks 须基于已批准目标、验收和授权政策形成实质规划，不能以通用模板冒充语义完成。工程在对应能力实施前闭合状态机、存储、迁移、恢复等设计，不逐字段或为补全工件再次请求用户确认；只有改变已确认行为、权限或成本边界的决定才重新提问。需求批准、实现、验证及交付各有其适用授权，不以产品中的 AI 计划审批取代 Requirement 的用户决定。

追踪：SW34 / AC34。

#### Scenario: AC34 验收行为

- **WHEN** 推广已批准的本批需求
- **THEN** 一个 Task/change 含有需求驱动的 proposal/design/specs/tasks 与 E01–E08 追踪；普通工件补全不新设用户确认，设计未知留给对应工程项

## 实现依据

- [命令入口](../../../internal/commands/task/command.go)、[生命周期](../../../internal/commands/task/workflow.go)、[创建预检](../../../internal/commands/task/creation_preflight.go)。
- [后端选择](../../../internal/commands/task/backend.go)、[元数据路径](../../../internal/task/meta.go)、[兼容映射](../../../internal/task/workflow/workflow.go)。
- [摘要派生](../../../internal/workflow/state.go)、[运行兼容](../../../internal/workflow/compat.go)。
