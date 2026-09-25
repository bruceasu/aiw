# E06 知识生成、审阅与注入

2026-09-18；Task `workflow-automation`，Work Item `wi-0008`，清单 `1.6`，Attempt `attempt-1789725422735098500`。本轮仅实现 E06，保留现有 worktree、Task/Core/Gate 和默认 schema 9。实现完成不代表运行验收通过。

## 实现与证据

| 范围 | 实现证据 | 行为 |
| --- | --- | --- |
| SW20 / AC20 | `knowledge_generation.go`、`execution/auxiliary.go`、`execution/auxiliary_http.go` | 消费已提交接受事实，每个接受版本固定提取键；Task 完成后异步汇总。提取明确返回条目或有依据的无新增；结构错误在宿主记录前转为无效结果，复用 E05 的一次恢复账。失败不回滚接受。 |
| SW21 / AC21 | `knowledge_generation.go`、`auxiliary_queue.go` | 汇总冻结成功、运行中、失败、缺失覆盖与条目版本；补齐产生新输入/草稿。输出与条目投影在同一条件提交内保存，索引可从已保存结果恢复；未变化条目不重新生成、不继承新正文的旧确认。 |
| SW22 / AC22 | `knowledge.go`、`knowledge_review.go`、`execution/knowledge.go` | 五态、正文/来源/证据/范围/用途/失效条件内容版本；人工请求绑定所见版本和修订号。编辑新候选，确认/拒绝/废弃/替代只经交互式人工入口。拒绝指纹索引指向权威版本，跨 Task 重复输入不能换号绕过。 |
| SW23 / AC23 | `knowledge_review.go`、`knowledge_selection.go` | 宿主与读取时核对接受引用、不可变原始工件及路径内容摘要；未知或变化降为待复核。恢复相同内容仍须人工复核；无检查依据时明确说明，不能自动升级为主参考。 |
| SW24 / AC24 | `knowledge_import.go`、`execution/knowledge.go` | 优先显示已指定负责人，否则当前项目操作者；记录身份未验证。明确选中的旧人工文本保留完整原文、路径、摘要后导入候选；不扫描所有 Session。正式决定仍通过 E01 主来源引用读取。确认只记录同步建议，不写 CONTEXT/ADR/需求或代码。 |
| SW25/SW26 / AC25/AC26 | `knowledge_selection.go`、`agent_context.go`、`tester_context.go` | 按标签匹配、可信分区、相关性、优先级、稳定 ID/版本排序；少量通用规则；拒绝/废弃不注入。主来源优先，候选/待复核明确仅作次要参考；版本正文和跳过原因进入冻结输入。 |
| SW27 / AC27 / AX05 | `knowledge_history.go`、`knowledge_selection.go`、`knowledge_storage.go` | 仅查询已有有界资源索引及当前原始输入明确引用的旧 Task；先复用结果、再按原始接受键补录。消费方持久登记历史范围，来源方持有原恢复账；每批 16 来源/1 MiB，Task 累计 64 来源/4 MiB，不随 Session/重启重置。 |

相关 Go 文件位于 `internal/workflow/`；执行适配位于其 `execution/` 子目录；Coder 入口位于 `internal/commands/task/`。

可选注入保持 16 条、4,096 tokens、16 KiB 三重上限，并核对完整接收输入和输出预约。计数仅使用实际 provider/model/digest 匹配的本地审阅能力记录。辅助 HTTP 的能力证明不自动适用于 Coder/Tester：必须另外记录 `knowledge_protocol=receiving-fixed-turn-v1`、完整接收封装开销、输出预约及证据。未知时跳过可选知识，不估算 tokenizer、不探测网络。必需输入超过已证明容量时只停止受影响派发，不截断正文。

知识投影在 Task 内最多 512 KiB，人工原文导入最多 16 项，每个版本最多 64 条人工审阅记录；它们使用项目资源账额外保留的每 Task 4 MiB 峰值空间，仍计入既有 Task/项目存储上限，存储结算不重置此预约。达到上限保留历史并报告局部缺口，不自动清理。此为既有 R3 总额内的保守实现限额，不增加模型次数或恢复额度。

## 人工入口

- `aiw task workflow knowledge show <task>`：显示条目、历史、覆盖和审阅待办。
- `aiw task workflow knowledge review <task> <absolute-workspace> <request.json>`：请求含 `version`、`expected_revision`、`action`、`reason`，编辑附完整 `edit`，替代附 `replacement`。终端显示实际条目及请求，人工输入 `review` 后条件提交。
- `aiw task workflow knowledge import <task> <absolute-workspace> <request.json>`：请求含 `source_path` 和完整 `content`，原文保存为不可变工件；不会成为执行授权。
- 可选 `.ai/knowledge-review.json` 的 `owner` 仅选择优先审阅人，不授予权限，不把当前操作者冒认为该负责人。

本轮没有执行这些入口，没有创建实际配置或后台请求。

## Verification

已做一次编辑后静态检查：核对 CLI 差异、类型构造、条件提交入口、Tester 来源校验、队列发布及选择调用链；确认 Core 当前映射为 `wi-0008 → 1.6`。检查发现并修正了非条件 `UpdateWithEvent` 在 schema 10 被拒绝的问题，以及 Tester 对可选知识 unavailable 记录的错误拒绝；后续修正按补丁文本核对，没有再运行验证命令。

实际命令：`Get-Content`（正文使用 UTF-8）、`Get-ChildItem`、`rg`、`Select-String`、`Get-Command aiw`、`aiw patch --help`，以及用户指定前缀的 `git status --short --branch` 和针对 `internal/commands/task/workflow_commands.go` 的 `git diff`。首次未指定 UTF-8 的 Core JSON 读取解析失败；最终静态批次改用 UTF-8 成功确认映射。Git diff 只提示 LF/CRLF，不构成编译/测试结果。已有大量 E01–E05 未提交/未跟踪变更均予保留；新文件按补丁正文核对，不把 Git diff 当作包含全部新文件。

写入使用直接补丁工具。`aiw patch` 的实现通过 Git apply 写入，不符合本轮仅允许只读 Git 的约束，因此未执行它；补丁应用中三次工具校验拒绝（上下文/重复目标）均未生效，合并修正后应用成功。

未运行编译、测试、制品构建、格式化、lint/vet、验证脚本、模型、后台宿主、网络、Git 写操作或 Task 生命周期命令。编译由 supervisor 按冻结 Compile Plan 执行；所有 AC20–AC27、AX05 仍未执行。本记录不是运行成功证据。

%% E06_ACTIVATION_PENDING：接收模型计量、受管辅助授权、资源盘点及 schema 10 联合启用仍须实际证据；默认 schema 不变。下一步由 supervisor 编译并处理有界诊断，之后按既定依赖处理 E07/E08 与整体验收。不得凭本实现勾选解除 Gate 或宣布 Task 已完成。

%% E06_HISTORY_LIMIT：历史补录只处理资源索引中与当前输入关联的来源；没有可靠来源账/原文/计量的旧历史保留缺口，不扫描全项目或新建恢复余额。E08 的独立 Verifier 及通知先后顺序仍归对应工作项，本轮不伪造其报告或通知结果。
