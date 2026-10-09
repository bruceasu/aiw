# FD-045: Agent Gateway 迁移到 Codex App Server 后端

**Status:** Design
**Revision:** 1
**Priority:** Medium
**Evidence policy:** Dual

## Problem

Agent Gateway 面向其他程序提供 OpenAI Responses 文本与函数工具调用子集。
当前 `runner.go` 每请求创建临时目录并启动 `codex exec --json`，仅在
`item.completed` 时交付消息。长期服务需要请求级增量输出、取消和可复用的后端生命周期。
本次迁移目标是保持现有调用方协议，以受控常驻进程替换一次性执行。
不承诺未经测量的速度、吞吐或模型费用改善。预计总工作量 3–5 个开发人日，
为粗估；协议门禁结论可能改变范围与估计。

## Options and decision

1. 保留 exec：现有生命周期简单、故障范围小；进程启动重复，当前输出粒度粗。
2. 每请求 app-server：可以获得双向 RPC 和文本 delta，但仍重复初始化进程。
3. 有界 app-server 进程池：复用进程，每请求独立 thread；需要管理 RPC、取消和恢复。

采用方案 3。第一版每个进程同时只承载一个请求，避免单个进程故障影响多个活动请求。
池容量上限使用已有 `global_concurrency`，按需求惰性创建，不新增无界队列。
继续由 Gateway 执行主体认证、模型权限、RPM、每日额度和并发控制。
对外保持无状态请求：调用方重放历史，Gateway 不自动恢复或共享历史 thread。
此次用户只要求创建 FD；不实施、派发 Worker、执行 Git 写入或修改稳定规格。
此前对 build.bat 的主工作区例外仅适用于安装调整，不延伸到本 FD 实施。

## Solution

### 后端边界与协议

- 在独立 Go 模块 `program/agent-gateway` 内集中定义后端请求、事件、终态及用量边界。
  不把 JSON-RPC 细节扩散到 HTTP、存储和主体策略模块，不新增第三方依赖。
- 通过 stdin/stdout JSONL 连接 `codex app-server`，完成 initialize/initialized 握手。
  保留 codex_path 和可选 codex_script 的 Windows Node 启动方式。
- 显式关联 RPC id、threadId、turnId、itemId；错误 id、重复终态及旧进程事件
  不得归入新请求。进程代次用于隔离重启后的迟到事件。
- 每请求独立空 cwd 与新 thread；不使用 thread/resume，不接收外部 thread id。
  完成后释放 thread 资源并清理临时目录，确认安全后才复用进程。
- 保留服务端固定指令、环境白名单、read-only 和当前工具禁用范围。
  不因新后端开放本地 shell、MCP、apps、hooks、web 或子 agent。
  非预期工具/审批请求不能等待交互或自动批准，应失败并回收对应进程。

### 进程池与生命周期

- 每池槽一个进程、最多一个活动请求。现有 HTTP 容量拒绝行为保持，不新增排队策略。
- 请求 timeout_seconds 覆盖获取后端、启动、初始化、thread/turn 及结果等待。
  RPC 等待、缓冲和 stderr 排空均受限；4 MiB 原始输出预算按请求/握手区间计量，
  不能把常驻进程的终身累计输出作为单请求预算。文本保持 256 KiB。
- 客户端断连、超时先请求 turn/interrupt，并在现有清理等待预算内确认终态。
  无响应或协议失同步则终止该池槽进程树，禁止将它交给下一个请求。
- Linux 进程组和 Windows Job Object 继续用于进程级清理；正常 turn 完成不杀健康进程。
  不删除仍可能被活动 thread/后代使用的目录。清理失败可观测，且不能报告成功。
- 崩溃、EOF、协议损坏使对应请求失败；已开始或接受状态未知的请求不自动重试。
  移除坏进程，后续请求至多一次替换启动；失败直接返回，禁止无限重启循环。
- stop、Ctrl+C/SIGTERM 停止接受新执行，取消活动 turn，关闭池并等待 HTTP/清理退出，
  最后释放状态锁；清理未收敛沿用保留锁并要求运营者核对的规则。

### OpenAI 协议兼容

- 保留 `/v1/responses`、`/v1/models`、`/v1/usage` 现有字段校验、错误 envelope 和主体过滤。
  不新增 Chat Completions、附件、previous_response_id 或完整 OpenAI API 支持。
- 文本 SSE 将实际 agentMessage delta 映射为现有 Responses 事件，保持 response/item id、
  顺序和序号；最终 completed 消息只作汇总校验，不重复发送已交付文本。
  部分文本后失败必须使用失败终态，不能伪装 completed。
- 非流式文本汇总 delta；json_object 继续缓冲并在完整输出后验证。
  函数工具继续通过受约束输出生成标准 function_call，由调用方执行。
  tools+stream、并行工具及其他已有不支持组合仍在后端前拒绝。
- 用量按请求/turn 取值；若事件包含 thread 累计值，必须确认字段语义后转换。
  不能重复累加累计快照；缺失用量仍为 unknown/null，不能当成零。

### 额度、持久化与隐私

- 常驻进程启动不再消耗每日请求额度。先保留请求额度，确认 turn 已开始后
  将 reserved 转为 started 并扣一次；使用现有存储锁原子转换。
- 明确 turn/start 的接受响应与 turn/started 的顺序，统一“首次可信接受/开始证据”
  的处理，不因重复响应或事件重复扣费。开始后的失败、超时、取消不退额度。
- 明确拒绝且确认未开始才释放预留。发送后失联、接受状态未知继续保守占额度，
  不自动释放或重放；沿用跨日期保守预留与运营者核对原则。
- 保留现有 metadata、HTTP requests、manifest 和恢复规则；历史 started 记录
  仍代表旧 exec 启动，不重算、不清空历史额度。若必须新增持久字段，先说明兼容方案。
- app-server 可能额外保存 thread/rollout。必须明确其路径、禁持久能力或清理方式，
  覆盖现有正文 168 小时保留策略；仅 archive thread 不等于删除正文。
  不保存原始 RPC/stderr、隐藏推理或凭据，不将 thread 历史暴露给其他请求。

## Scope

包含后端适配、stdio RPC、池、文本增量、取消/关停/恢复、额度起点调整、
现有函数工具和结构化输出兼容、文档及受影响稳定规格。
不改认证来源、Key 契约、依赖、外部 API 范围或当前独立程序安装布局。
不实现多轮会话复用、工具执行、自动请求重试、远程 app-server 暴露、容器部署。
若协议门禁显示必须改变这些边界，先更新 FD 并取得对应设计决策。

## Work items

每项规模小、难度中，目标不超过半个开发日；未知协议先由 1.1 收敛，
仍超出目标则在开始实施前继续拆分。每项均需独立差异和静态证据。

- [ ] 1.1 固定目标 Codex 版本与 RPC 合约，解决下述协议/隐私门禁。依赖：无。
  完成：记录版本、初始化、thread/turn、output schema、工具策略、usage、清理字段及依据，
  所有实现所需未知已解决；执行性验证若必要先获明确授权。
- [ ] 1.2 提取后端请求/事件/终态边界。依赖：1.1。
  完成：HTTP 与存储无 RPC 类型依赖，所有当前输出模式有对应映射。
- [ ] 1.3 实现 stdio RPC 连接与初始化。依赖：1.2。
  完成：id 路由、服务端请求拒绝、有限读写、EOF 和握手失败均有确定退出路径。
- [ ] 1.4 实现新 thread/turn 执行和资源释放。依赖：1.3。
  完成：固定配置/禁工具、独立 cwd/thread、schema 输出及 thread 正文处理符合门禁结论。
- [ ] 1.5 实现有界池及坏进程替换。依赖：1.4。
  完成：单槽单请求、代次隔离、复用前清理和无界队列/循环禁止规则可静态追踪。
- [ ] 1.6 调整额度开始事件及不确定状态处理。依赖：1.4。
  完成：事件乱序/重复不重复扣费，未知接受保留预留，历史记录与跨日策略保持。
- [ ] 1.7 转换文本 SSE、函数调用、JSON 和 usage。依赖：1.4、1.6。
  完成：无重复 delta/终态，现有支持与拒绝矩阵保持，未知用量不记零。
- [ ] 1.8 接入断连、超时和池槽进程树回收。依赖：1.5、1.7。
  完成：interrupt 确认或有界强制回收，不把坏槽或在用目录交给下一个请求。
- [ ] 1.9 接入服务关停、崩溃恢复及数据保留。依赖：1.6、1.8。
  完成：取消/进程池/HTTP/锁释放顺序明确；无不确定请求自动重试或历史额度重算。
- [ ] 1.10 更新文档、agent-proxy 稳定规格及实现证据。依赖：1.9。
  完成：说明最低兼容版本、常驻后端、额度变化、运行限制及未验证风险；
  记录 compile-only 与静态检查，报告为中文 Markdown 和同名 JSON。

## Acceptance

1. 现有合法文本/JSON/函数工具请求保持支持；非法控制字段或无权限请求不启动 turn。
2. 进程池上限不超过 global_concurrency，每请求新 thread，无历史、模型或主体串用。
3. 普通文本实际 delta 可逐段交付，输出不重复；最终状态与 usage 与对应 turn 一致。
4. 失败、超时、取消及交付失败不会报告成功，不自动重试不确定的执行。
5. 进程启动不扣请求额度；确认开始仅扣一次，未知接受保守占额度，历史额度不变。
6. 坏进程不复用；请求取消、服务停止和恢复均遵循有界清理及状态锁规则。
7. 固定工具禁用、正文保留和脱敏日志在新后端有效，独立 cwd 不被描述为安全隔离。
8. Windows/Linux 编译支持、Node 启动配置和独立程序安装布局保持。

## TODO

- [ ] 解决协议版本、工具禁用和 thread 正文生命周期门禁。
- [ ] 门禁解决后更新 FD 为可实施状态，再按独立分支/worktree 规则启动实施。
- [ ] 完成 1.2–1.10 后提交真实证据并交独立 Reviewer；不能以创建 FD 代替审查通过。

## Verification

已完成：只读查看 runner.go、config.go、server.go、main.go、process_windows.go、
现有协议字段/输出路径，以及 openspec/specs/agent-proxy/spec.md。
已运行 FD CLI help、new 和 claim；本次未运行 Codex、模型请求、测试、构建或网络调用。

实施时默认：检查最终差异、RPC/终态/清理调用链和规格一致性；
执行一次 `python program/agent-gateway/scripts/compile.py`，离线编译到空设备。
不默认生成/执行 Go 测试，不进行最终构建、安装、性能基准或网络验证。
独立 Reviewer 依据当前 FD 和实现证据进行静态审查。
如果静态资料无法确定关键 RPC 行为，先说明精确验证命令、范围、时长和联网风险，
得到授权后才执行；测试不作为默认 FD 阶段。

%% NEEDS_INPUT: 1.1 技术门禁：确定本机/目标 Codex 版本，核对 stdio RPC、outputSchema、
usage 字段及 turn/start 响应和 started 事件顺序；先检查本地 schema/官方资料。
本次未读取本机 RPC schema，不将网站最新协议视为本机已支持。

%% NEEDS_INPUT: 1.1 安全/隐私门禁：证明工具禁用和 thread/rollout 的正文生命周期
在目标版本可实现。若无法保持现有正文期限及禁工具契约，需提出方案并获得用户决策，
不得通过 archive 或删除工作目录推断正文已删除。

%% 风险：未测量启动成本和吞吐收益；app-server 不自动提供完整 OpenAI API 兼容。
常驻运行新增事件、缓冲、子进程和本地历史资源管理责任。

## Sources

- 本会话：用户明确 Gateway 是 Codex 模拟 OpenAI API、供其他程序消费；
  认可迁移并请求创建 FD，尚未请求实施。
- `program/agent-gateway/runner.go`、`protocol.go`、`config.go`、`server.go`、`main.go`
- `program/agent-gateway/process_windows.go`、`process_linux.go`、`storage.go`、`shutdown.go`
- `openspec/specs/agent-proxy/spec.md`、`openspec/specs/cli-and-plugins/spec.md`
- 上一轮已查阅官方资料：https://learn.chatgpt.com/docs/app-server
  及 https://learn.chatgpt.com/docs/non-interactive-mode；版本适配以 1.1 结论为准。
- Planner session: `fd045-planner-20261009-b793ea`
- Source event: `FD-045-000002-design-requested`，本会话已 claim；未派发 Worker。
