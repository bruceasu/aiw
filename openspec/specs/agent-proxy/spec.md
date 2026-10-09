# Agent Proxy specification

## Purpose

提供基于 Codex CLI 的 Go 共享 AI 网关。FD-017 与 REQ00006 的后续确认替代旧 TypeScript 多 Provider/ACK 接口；历史 Issue 与归档 FD 保留。

明文网关 Key、调用主体跟踪及 HTTP 请求统计采用 [FD-020](../../../docs/features/FD-020_AGENT_GATEWAY_REQUEST_OBSERVABILITY.md) 中已确认的范围。此规格描述能力契约，实施进度和验证结果由 FD 及其证据记录。

## Requirements

### Requirement: Authenticated OpenAI text subset

网关 MUST 对 Responses/Models/Usage 验证 Bearer 网关 Key，并关联稳定主体。运营者维护 principals[].keys 明文 Key 配置，请求通过常量时间直接比较认证；Key 为高熵值，至少 43、最多 1024 字节，不含空格、CR、LF、TAB。配置加载和请求认证 MUST 不计算 Key 摘要，MUST 不接受 key_hashes 字段；Key 是不透明字符串，原摘要值作为 keys 中的 Key 时按原字符串直接匹配。MUST 拒绝重复 Key，不披露凭据。配置重启生效；轮换不清历史额度。Models MUST 只返回当前主体允许的逻辑模型。

POST /v1/responses MUST 支持 model、文本 input（字符串/user/assistant 消息数组）、instructions、stream、store:false、background:false、tools:[]、text.format.type=text/json_object。对于 `@openai/agents` v0.18.0 默认 HTTP 的非流式函数工具回合，还 MUST 接受空 include、最多 16 个 function tools、tool_choice auto/none/required、parallel_tool_calls:false，以及同一 input 中的 function_call 和匹配的字符串 function_call_output。Gateway MUST 只生成标准 function_call 项，不执行工具；调用方负责工具执行并在后续请求中重放调用及结果。非空 tools 与 stream:true、并行调用、previous_response_id 和结构化 text 输出组合 MUST 在 Runner 前拒绝。其它字段/非默认执行控制 MUST 在 Runner 前报错。body 上限 1 MiB，合并文本与工具定义上限 64 KiB UTF-8，未知/重复字段、尾随 JSON、非法 UTF-8 MUST 被拒绝。不实现会话、附件、托管工具、json_schema 或全 OpenAI API。

成功 MUST 返回标准 Response/assistant output_text 与 nullable usage；错误 MUST 使用标准 error envelope/脱敏 message。SDK 可仅改 base_url/API Key 调用支持子集，运行验证由外部安排。SSE MUST 使用标准 Responses 事件及序号，增量来自实际完成消息，不承诺逐 token 延迟。json_object MUST 校验完整结果，无自动重试。

#### Scenario: Unknown control

- WHEN 输入包含 cwd、sandbox 或不支持字段
- THEN 返回 400 且不启动 Codex

#### Scenario: Authenticate a plaintext gateway key

- WHEN 调用者传入合法 Bearer Key，且与一个 enabled 主体的 keys 配置值相同
- THEN 直接比较得到该主体，不计算请求摘要，后续模型权限、额度、日志及统计均使用稳定主体 id

#### Scenario: Use a former digest as an opaque key

- WHEN 运营者将原摘要字符串配置到一个 enabled 主体的 keys，调用者发送相同字符串作为 Bearer Key
- THEN 按该字符串直接认证，不再计算摘要；原摘要计算前的 Key 不因存在这种历史关系而获得认证

#### Scenario: Reject the removed key_hashes field

- WHEN 配置包含 key_hashes 字段，即使同时配置了 keys
- THEN 以脱敏配置错误拒绝加载，不转换、忽略或尝试兼容该字段

#### Scenario: Reject a disabled or invalid credential

- WHEN 主体 enabled=false，或 Bearer Key 缺失、格式非法、未匹配有效凭据
- THEN 返回 401，不启动 Codex；请求主体为空，日志和错误响应不披露 Key 或其摘要

#### Scenario: Reject an ambiguous credential configuration

- WHEN 同一主体或不同主体的配置含重复 Key
- THEN 拒绝加载配置并返回脱敏错误，不通过遍历顺序决定归属主体

#### Scenario: Rotate keys without changing caller identity

- WHEN 运营者为同一主体替换 Key 并重启网关
- THEN 新配置生效，已移除的 Key 不再认证；该主体的历史请求统计、执行用量和每日额度不因换 Key 清零

### Requirement: Controlled process lifecycle

每请求 MUST 使用独立空临时工作区和固定服务端配置/环境白名单。Linux MUST 使用独立进程组终止树，Windows MUST 使用独立 Job Object 并在加入 Job 后恢复 suspended 子进程。取消、超时、正常根退出和服务关停 MUST 收敛树后清理目录；清理失败 MUST 可观测且不能返回成功。输出文本 256 KiB、原始 stdout/stderr 各 4 MiB，默认请求 600 秒，`timeout_seconds` 保持 1–3600 秒配置范围且重启生效，流写入 5 秒 deadline。客户端期限独立于服务端执行上限。

development MUST 限 loopback 监听与 peer；shared MUST 显式配置认证/限额/监听。应用不操作 Docker、挂载或登录供给；Proxy/Codex 可共容器，部署/宿主隔离由用户负责。独立 cwd 不提供请求间访问隔离，Linux 恶意脱组及共容器跨读为残余风险。

### Requirement: Codex Chat Completions text subset

POST `/v1/chat/completions` MUST 复用 Responses 的 Gateway Key/主体认证、模型授权、RPM/并发/日额度、App Server 执行、超时/取消及清理逻辑；每个请求 MUST 仅执行和扣额一次。

请求 MUST 只接受 `model`、`messages`、`stream`、`n`。`messages` MUST 为非空数组，消息仅包含 `role` 和非空白字符串 `content`；支持开头的 system/developer 指令以及有序 user/assistant 文本对话，MUST 至少包含一条 user 消息，MUST 拒绝对话开始后的 system/developer。`stream` 仅允许省略或 false，`n` 仅允许省略或 1。未知/重复字段、null、尾随 JSON、非法 UTF-8、超过 1 MiB 主体、映射后超过 64 KiB UTF-8 的指令/对话及其他角色/内容类型 MUST 在启动后端前拒绝。函数工具、流式、content 数组和生成参数控制不属于此子集。

成功 MUST 返回 `object:chat.completion`、`chatcmpl_` ID、created、逻辑 model 和一个 index=0、assistant 字符串 content、finish_reason=stop 的 choice；文本来自成功执行结果。失败 MUST 返回现有脱敏错误 envelope/HTTP 状态，不伪称 stop。已知 usage MUST 映射为 prompt_tokens/completion_tokens/total_tokens，未知 usage MUST 保持 null。

HTTP 请求观测 MUST 识别 `/v1/chat/completions`，沿用现有元数据结构。新版本 MUST 能加载旧记录；旧版本可能不识别新增 route 值，直接降级的限制 MUST 写入文档。本能力不要求新增代理统计/计费组件。

Chat 请求的顶层和消息对象字段名 MUST 精确匹配上述白名单，包括大小写；MUST 拒绝 `MODEL`、`ROLE` 等别名以及别名与标准键同时出现的覆盖组合，不得仅依赖 Go struct 解码的大小写匹配。

#### Scenario: Accept the existing Say client

- WHEN Say 发送允许的 model、system/user 字符串消息和 stream:false
- THEN Gateway 使用既有 Codex 执行链，并在成功时返回一个完整文本 choice 和 stop

#### Scenario: Reject unsupported chat controls before execution

- WHEN 请求包含 stream:true、n 大于 1、tools、图片/音频内容或其他不支持字段
- THEN Gateway MUST 返回 400，且 MUST NOT 启动模型执行

### Requirement: Bounded Codex App Server backend

Codex execution MUST use the versioned App Server stdio JSON-RPC protocol and a process pool bounded by `global_concurrency`. A pool process MUST serve at most one HTTP request at a time; each request MUST start a new ephemeral thread with a separate temporary cwd and MUST NOT accept caller-supplied thread IDs. Healthy App Server processes MAY be reused after a terminal turn. A process with an unknown protocol or execution state MUST be discarded and MUST NOT be retried automatically.

The gateway MUST bound JSONL line length, total stdout, stderr, and notification buffering. It MUST negotiate experimental API support during `initialize`, send `initialized`, and verify effective configuration and MCP server status before every turn. Any enabled or unknown MCP server state MUST reject the turn. The child MUST reuse the configured `CODEX_HOME` authentication and MUST NOT copy credentials or edit user configuration. Local shell, unified exec, image viewing, sleep, apps, plugins, hooks, web search, browser/computer use, image generation, and multi-agent features MUST be explicitly disabled through process configuration; an unexpected server request MUST receive an error and invalidate the process.

Each turn MUST explicitly use the configured logical model, a request-specific cwd, `approvalPolicy=never`, and a read-only sandbox with network access disabled. The gateway MUST mark quota usage started only after a successful `turn/start` response; an ambiguous start MUST retain its conservative reservation and MUST NOT be retried. Usage MUST come from the current turn only and remain unknown when absent or invalid.

#### Scenario: Refuse an enabled or unknown MCP configuration

- WHEN effective configuration or `mcpServerStatus/list` reports an enabled MCP server, an unknown server state, or an unreadable safety configuration
- THEN the gateway MUST refuse the turn, discard that process slot, and MUST NOT start a model turn

#### Scenario: Discard a process after an ambiguous turn start

- WHEN `turn/start` may have been submitted but its successful response cannot be confirmed
- THEN the gateway MUST keep the reservation conservative, discard the process, return an error, and MUST NOT retry the request

#### Scenario: Reuse a healthy process with a fresh thread

- WHEN a turn reaches a known terminal state
- THEN the pool MAY reuse the process for another request, but MUST start a fresh ephemeral thread and cwd

### Requirement: Local graceful shutdown control

Gateway MUST 保留 Ctrl+C/SIGTERM 关停，并提供 `stop --config <file>` 和 `POST /internal/shutdown` 本地控制入口。该入口 MUST 要求 loopback peer 和现有 enabled 主体 Bearer Key，拒绝非 POST、非空 body 和 query；不得触发模型执行或新增远程管理权限。所有 enabled 主体 Key 可用于本机控制。审计存储不可写时仍可执行经过认证的本地关停并记录脱敏错误，不以此放开其他 AI 请求的存储门禁。

关停 MUST 复用全局取消和现有进程树清理，等待 HTTP 处理及定期内容清理退出后才释放状态锁。停止命令 MUST 有有限等待，无重定向、无自动重试、不暴露凭据或原始错误；监听结束且状态锁释放才能报告完成。失败不得强杀或自动删除锁。仅绑定具体非 loopback 地址的实例仍使用控制台信号。

#### Scenario: Stop a running local gateway

- WHEN 运营者运行配置匹配的停止脚本或 CLI，且本地认证通过
- THEN 接收 HTTP 202，取消活动执行，等待处理收敛与锁释放，停止命令报告完成

#### Scenario: Reject remote or unauthenticated shutdown

- WHEN 控制请求缺少有效 Key、来自非 loopback、方法或输入不符合约定
- THEN 拒绝请求且服务继续运行

### Requirement: Durable quotas and usage

每日额度 MUST 在 Codex 成功启动后扣一次，失败/超时/取消不退；启动前拒绝及 spawn_failed 不扣。所有尚未确认预留 MUST 跨日期保守占额度，转换到 actual started/failed 记录须在同一存储锁下提交，防止跨零点超售；today_reserved MUST 包含旧日仍占容量的预留，today_used 及统计分组仅累计实际启动。单实例 RPM 为滚动 60 秒，并发即时拒绝、无队列。每日与统计 MUST 采用同一可配置时区、默认 UTC，UTC 时间持久化，按 started_at 所属日归属。

文件存储 MUST 保留元数据、排他锁、版本标识、原子替换及损坏 fail-closed，不得重启清零。started 恢复为 interrupted failure；reserved 启动窗口 MUST 阻止该主体新执行，待运营者离线核实，不伪称 spawn/落盘原子。

GET /v1/usage MUST 只查询当前主体，支持 start_date/end_date 双端包含、最多 366 日、默认今天，按日期/模型返回请求数、四类终端数量、in_progress、已知 token 累计和 unknown_usage_requests，cost=null。不得查询其它主体或返回正文。

### Requirement: Separate seven day content retention

元数据 MUST 长期保留；提示词/instructions/已得响应文本 MUST 单独保存，自预留起 168 小时，启动及每 60 秒清理、重启不延期、过期更新不重建。执行元数据、正文及日志不能保存 Key、完整环境、stderr/JSONL 或隐藏推理；明文 Key 仅由运营者维护于服务端凭据配置。写入/清理失败 MUST 阻止新 AI 执行；正文仅服务端运营者查看，无查询 API/UI。停服期间到期文件由下次启动清理，离线副本管理归部署方。

### Requirement: HTTP request tracking and statistics

网关 MUST 为所有进入 HTTP Handler 的请求产生 request_started 日志，在请求正常收敛或可捕获异常时产生 request_finished 日志，以服务端生成的 x-request-id 关联稳定调用主体、接口、方法、逻辑模型、HTTP 状态、结果、错误码及毫秒耗时，覆盖认证失败、参数拒绝、限流、并发拒绝及 Models/Usage 请求。进程异常终止可能只留下开始记录，必须按下述中断恢复规则处理，不能伪造结束日志。认证失败主体 MUST 为空，未知接口/方法为 other，模型只记录配置中的逻辑模型，尚不能确定时为空，不记录用户任意输入或 URL 查询原文。日志及请求元数据 MUST 不含 Key、Key 摘要、Authorization、提示词或响应正文。SSE 的 HTTP 状态和执行结果 MUST 分开记录，传输失败可观测。

HTTP 元数据 MUST 单独保存于 requests/<request-id>.json，沿用现有排他锁、原子写入及长期元数据保留；不能将拒绝请求写入扣额 metadata。旧存储可新增 requests 目录及 requests-manifest.json 版本标记，不回填历史 HTTP 请求；已标记后目录缺失 MUST 拒绝启动，不静默清零。请求进入 Handler 时持久化 in_progress，终止时替换；异常重启将未完成记录标 interrupted，实际结束时间及耗时为 null，不伪造已知数据。损坏/未知版本 MUST 拒绝启动，写入失败 MUST 脱敏记录并阻止新执行。

GET /v1/usage MUST 保留原 groups、额度及 token 字段，增加 principal 与 http_groups，仍只查询当前主体。HTTP 分组按 received_at 在配置时区所属日、模型、接口、方法返回请求总数、成功、拒绝、失败、超时、取消、中断、处理中、实际执行数、传输失败数、HTTP 状态/错误码计数、已知耗时样本及总毫秒数。执行 groups 仍按 started_at 所属日统计。未认证请求仅运营者本地查看，查询自身可作为处理中请求计入 HTTP 分组。实际后端 Codex 登录帐户不属于调用主体身份。

http_groups 的分组键为 date、model、route、method；计数字段为 requests、succeeded、rejected、failed、timed_out、cancelled、interrupted、in_progress、executed、delivery_failed，HTTP 状态及错误码分别使用 http_statuses 和 error_codes 映射。duration_samples 仅包含已知耗时的终止请求，total_duration_ms 为其毫秒耗时总和；样本为零表示平均耗时未知，不能将中断请求补作零耗时。execution_started/executed 只表示已确认成功启动 Codex，不能由 HTTP 状态或是否存在预留推断。

HTTP 请求结果 MUST 区分 succeeded、rejected、failed、timed_out、cancelled、interrupted、in_progress。http_status=0 表示尚未发送或无法确认响应状态，不能解释为成功。请求元数据的 received_at/finished_at 使用 UTC，duration_ms 表示已知处理耗时；中断恢复的 finished_at/duration_ms 为 null。

#### Scenario: Authenticated request rejected before execution

- WHEN 已认证主体的 Responses 请求触发参数、模型、RPM 或并发拒绝
- THEN 请求保留主体及 request-id 并计入 HTTP 拒绝统计，不扣执行额度、不增加执行 groups

#### Scenario: SSE backend fails after headers

- WHEN 已开始输出 SSE 后后端失败或超时
- THEN HTTP 状态可为 200，但请求结果及错误码 MUST 体现失败或超时

#### Scenario: Upgrade existing store

- WHEN 已有执行存储尚未标记 HTTP 存储升级，也无 requests 目录
- THEN 创建独立 HTTP 元数据目录，保留历史执行、主体及额度，HTTP 历史统计从升级后开始

#### Scenario: Track an unauthenticated request

- WHEN 请求进入 Handler，但认证失败或不满足 development 的 loopback peer 限制
- THEN 开始及终止记录共用服务端 request-id，主体为空，记录拒绝结果及错误码；该记录不归入任意已认证主体的 http_groups

#### Scenario: Track model and usage queries

- WHEN 已认证主体调用 Models 或 Usage，无论成功还是被方法、查询参数校验拒绝
- THEN 保留该主体的 HTTP 请求记录并计入相应接口分组，不增加 Codex 执行数或 token 用量

#### Scenario: Query only the current caller's statistics

- WHEN 主体 A 使用自己的 Key 查询一个合法日期范围的 Usage
- THEN principal 为 A，groups 和 http_groups 仅包含 A 的记录，不包含主体 B 或未认证请求，且不能通过请求参数选择其他主体

#### Scenario: Separate received day from execution day

- WHEN 请求收到时间与实际 Codex 启动时间跨越配置时区的午夜
- THEN http_groups 按收到日归属，执行 groups 和每日实际用量按启动日归属，不为对齐两个分组而修改扣额规则

#### Scenario: Recover an unfinished HTTP request

- WHEN 服务重启时读到持久化的 in_progress HTTP 记录
- THEN 将其标为 interrupted 并保留原 request-id、主体和收到时间，错误码为 request_interrupted；结束时间和耗时保持 null，实际执行数只依据已确认启动的执行元数据

#### Scenario: Reject missing or corrupt upgraded HTTP storage

- WHEN requests-manifest.json 已存在但 requests 目录缺失，或 HTTP 存储标记/记录损坏、版本不受支持
- THEN 拒绝启动，不自动清空或重新初始化历史统计

#### Scenario: Report an HTTP audit write failure

- WHEN 请求元数据无法写入
- THEN 记录脱敏存储错误并阻止新 AI 执行；终态写入失败可留下原 in_progress 记录，不能宣称终态已持久化，后续重启按中断恢复规则处理

#### Scenario: Preserve execution outcome when response delivery fails

- WHEN 发送响应或刷新 SSE 失败
- THEN HTTP 跟踪记录传输失败，已启动执行的用量和已扣额度继续保留，不因传输失败退回额度；执行结果与客户端交付结果可分别辨别

### Requirement: Explicit legacy retirement

旧 /v1/requests、WebSocket /v1/events、ACK 暂存、TypeScript 入口与多 Provider 直连可以移除；MUST 同步迁移 ai 客户端。旧状态不自动迁移或删除；历史批准来源/证据保留。容器部署与测试不属于本应用 FD 的交付条件，不能声称未运行行为已经通过验证。
