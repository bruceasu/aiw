## ADDED Requirements

### Requirement: Task 配对归档

系统 MUST 将同一 Task 的 OpenSpec change、任务运行目录及存在的绑定 Session 作为一组归档，以相同日期和完整 Task ID 关联各归档位置。系统 MUST 保留原任务身份、Workflow 状态、证据及 Session 引用，并保持既有归档资格和 Git 授权要求。原生与委托后端 MUST 满足同一契约。

#### Scenario: 成功归档

- **WHEN** 满足归档条件的 Task 执行 archive
- **THEN** 两处活动目录分别移动到对应的归档根，并具有相同归档名称
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

`aiw list` MUST 只显示未归档任务。`aiw list --all` MUST 同时显示未归档与已归档任务，分别展示 Workflow 状态、归档标识和实际 change 路径或规格缺失标识。发现过程 MUST 合并规格与运行记录候选，限定在支持的固定目录层级，兼容既有元数据路径和文件名。除缺失运行记录的最小补建外，列表 MUST NOT 修改运行数据。

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

Task archive MUST 按真实 Session 绑定归档会话到 `.ai/sessions/archive/<date>-<task-id>/<session-id>/`，保留 Session 身份、原状态和历史内容。系统 MUST 检查活动、旧版及归档 Task 的重复绑定，并核对 Session 的 Task 关联。归档 Session MUST 能通过 Task 关联的既有只读接口查询，MUST NOT 被继续执行或写入。无关独立会话 MUST NOT 被移动。

#### Scenario: Session ID 与 Task ID 不同

- **WHEN** Task 绑定一个独占、无运行进程或写入占用的 Session，且归档资格通过
- **THEN** 移动实际绑定的会话目录，保留原 Session ID 和 Task 引用
- **AND** 不按 Task 同名目录猜测绑定，不把失败或暂停状态伪造为 completed

#### Scenario: Session 正在运行或绑定冲突

- **WHEN** Session 为 running、仍有执行/写入占用，或存在重复绑定及身份冲突
- **THEN** 在任何归档移动前拒绝操作并报告原因

#### Scenario: Session 缺失

- **WHEN** 有绑定但活动及合法归档位置均找不到对应 Session
- **THEN** 提示“会话记录缺失”，继续处理规格及运行目录归档，不伪造会话历史
- **AND** 无绑定时无需会话移动；权限错误及损坏记录不得被当作缺失跳过

#### Scenario: 归档后只读查询

- **WHEN** 通过 Task 查询其已归档 Session 的 status、get 或其他既有只读内容
- **THEN** 由归档 Task 绑定定位真实会话位置，返回保留的内容
- **AND** 兼容核验过身份的旧 `.ai/archive/<session-id>/` 会话，拒绝归档会话的执行与写入

#### Scenario: 历史会话补归档与重试

- **WHEN** Task 已归档而关联 Session 仍在活动或旧独立归档位置，用户再次显式 archive
- **THEN** 沿用原 Task 归档日期补齐会话归档，已完成的新位置不重复移动
- **AND** 任一步骤失败仅补偿本次移动，list 不搬动或重建 Session 历史

### Requirement: 规格缺失时正常显示

运行记录存在但对应 change 记录在支持的位置均不存在时，列表 MUST 保留该任务行并在 PATH 单元格显示“规格已删除”。系统 MUST NOT 仅因规格缺失使列表失败，也 MUST NOT 自动重建规格。

#### Scenario: 规格记录已经删除

- **WHEN** 列表范围内的 Task 仍有运行记录，但其 change 记录不存在
- **THEN** 显示任务状态和“规格已删除”，继续显示其他任务
- **AND** 没有其他错误时命令正常返回，不显示无效的规格路径

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
