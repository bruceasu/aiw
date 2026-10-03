# Go Agent Proxy

FD-017 将原 TypeScript 多 Provider 服务替换为 Go 标准库网关。公开支持 OpenAI Responses/Models 的下列子集，后端为运营者预先配置登录状态的 Codex CLI。

## 启动

需要 Go 1.25.1 或以上。应用支持 Linux、Windows 10/11；编译证据覆盖 amd64，其它架构未验证。Go 服务是独立模块，不使用根模块依赖。

```sh
cd program/agent-gateway
go build -o bin/agent-gateway .
./bin/agent-gateway start --config /absolute/path/gateway.json
```

Windows 将输出改为 `bin/agent-gateway.exe`，直接运行该 exe。上述是运营者生成产物的命令，本次工作没有执行。AIW 插件入口为 `plugins/aiw-gw/aiw-gw.py`，调用同目录已编译程序，缺失时明确报错，无自动安装/构建。Windows 建议直接运行 Go exe 接收控制台关停信号；外部强制结束属于异常恢复路径。

配置示意（路径替换为真实绝对路径，Key 占位值不可用于启动）：

```json
{
  "mode": "development",
  "listen": "127.0.0.1:43127",
  "state_dir": "/srv/aiw-gateway/state",
  "workspace_dir": "/srv/aiw-gateway/work",
  "codex_path": "/usr/local/bin/codex",
  "timezone": "UTC",
  "timeout_seconds": 600,
  "global_concurrency": 4,
  "models": {"gateway-default": "YOUR_CODEX_MODEL"},
  "principals": [{
    "id": "team-a", "enabled": true,
    "keys": ["<server-issued-key>"],
    "allowed_models": ["gateway-default"],
    "rpm": 10, "daily_limit": 100, "concurrency": 2
  }]
}
```

`state_dir` 与 `workspace_dir` 必须不同、不嵌套，持久存储不可使用请求临时目录。Windows npm 安装不能指定 codex.cmd：配置 `codex_path` 为真实 node.exe，另配 `codex_script` 为真实 codex.js，参数通过原生 CreateProcessW 传入，不拼接 cmd.exe 命令。部署方提供可用 Codex CLI 与登录环境，网关不探测、安装或刷新登录。

## 友好关闭

在仓库根目录执行 `build.bat bin` 会安装 AIW、Gateway 二进制与插件入口，保留安装目录现有 `gateway.json`，构建不下载依赖。然后使用：

```bat
scripts\start-gateway.bat
scripts\stop-gateway.bat
```

停止命令在另一个终端执行；可向两份脚本传入 `--config "绝对路径\gateway.json"`，命令行 `aiw gw start|stop --config <file>` 也可使用。脚本沿用 PATH 上的 AIW，其默认配置位于安装的插件目录；请确保启停使用同一配置。

`stop` 从配置读取一个 enabled 主体的 Key，通过本地 `POST /internal/shutdown` 请求取消服务，不输出凭据。控制入口仅允许 loopback 来源、有效现有 Key、POST、空 body、无 query，不支持远程关闭；所有 enabled 主体的 Key 都可用于本机关闭。仅绑定具体非 loopback 网卡时使用服务控制台 Ctrl+C；通配监听时停止命令连接 loopback。

服务关停取消活动请求并复用现有后端进程树清理，等待 HTTP 处理和定期内容清理结束，再释放状态锁。客户端最多等待关停完成 15 秒，监听结束且状态锁释放才报告成功；没有服务且没有锁时报告已停止。审计不可写时仍尽力接受经过认证的本地停止操作并输出脱敏日志。失败不强杀、不自动删除遗留锁；按“异常恢复”步骤核实。旧二进制不支持此入口，需要先在原控制台退出旧版本，再运行新版本。

## 网关 Key

运营者离线生成至少 32 随机字节（base64url 为至少 43 字符），将明文保存到服务端私有配置的 `principals[].keys`，并分发给对应调用者。每个 Key 为 43–1024 字节，不含空格、CR、LF、TAB；示例中的短占位值必须替换。请求直接与配置 Key 做常量时间比较，配置加载和认证均不计算摘要；配置文件包含可用凭据，应限制读取权限，不把真实 Key 放入仓库、日志或命令行参数。本次未生成或安装 Key，没有新增在线签发 API。

仅支持 `keys`，配置中出现 `key_hashes` 会被拒绝。Key 是不透明字符串：可以将原摘要值直接放到 `keys`，客户端必须发送相同字符串，此时它就是 Key，不再转换或还原。加载配置时拒绝同一主体或不同主体的重复 Key。本次保留用户已调整的配置值。

Key 新增、轮换、禁用均在重启生效。同一主体可以配置多个 Key：先增加新 Key 重启，再移除旧 Key 重启；`enabled:false` 禁用整个主体。主体 id 永久稳定，不得重用给别人；换 Key 不清统计或每日额度。所有接口需要 `Authorization: Bearer <gateway-key>`。多人共用同一主体时无法再区分具体个人；不同调用用户应配置不同主体。

## API 支持范围

- `POST /v1/responses`：支持既有文本子集，以及 `@openai/agents` v0.18.0 默认 HTTP、非流式函数工具回合所需的 `include:[]`、函数型 `tools`、`tool_choice:auto/none/required`、`parallel_tool_calls:false`、`input` 中的 `function_call` 与字符串型 `function_call_output`。Gateway 仅生成 `function_call`；调用方执行工具后，在下一次请求中重放历史输入、该调用项和对应结果。函数工具最多 16 个，工具定义和正文合计最多 64 KiB；函数工具不支持 `stream:true`、`previous_response_id`、并行调用或结构化 `text` 输出。未知/不支持字段在 Runner 启动前报 400。
- `GET /v1/models`：仅返回主体允许的逻辑模型，标准 list 对象。
- `GET /v1/usage?start_date=2026-10-01&end_date=2026-10-03`：本地扩展，双端包含、最多 366 自然日、默认今天，仅当前主体，返回按天/模型分组的四类结果及真实 token 累计/缺失数量；不含正文，费用未知为 null。

SDK 使用 `base_url=http://host:43127/v1` 和网关 Key 调用支持子集。Response 的文本位于 `output[].content[].text`；使用 SDK 的 `response.output_text`。不支持 Responses retrieve/delete、Chat Completions、会话或完整 OpenAI API。SDK 运行兼容测试由用户安排，本次仅依据官方契约实现并静态审查。

Agents SDK v0.18.0 的默认 HTTP 函数工具流程也使用该 `baseURL` 和网关 Key。调用方须设置 `OPENAI_AGENTS_DISABLE_TRACING=1` 或 `tracingDisabled:true`，避免 SDK 默认把提示词及工具数据导出到 OpenAI tracing。Gateway 使用 Codex CLI 的 `--output-schema` 取得“文本回答或单个函数调用”决策，并验证函数名及 JSON 参数形状；它不执行函数，也不保证参数满足工具自身的完整 JSON Schema。此流程尚未经过安装版 Agents SDK 或真实 Codex 后端的运行验证。

SSE 使用 Responses 标准 created/in_progress、output item/content part/text delta/done、completed/failed 事件及 sequence_number。Codex exec 仅在完成的 agent_message 到达时输出真实文本片段，不保证逐 token 输出或首 token 延迟。json_object 经最终 JSON object 校验后发送；不自动付费重试。

body 至多 1 MiB，合并文本 64 KiB UTF-8，文本输出 256 KiB，原始 JSONL/stderr 各最多 4 MiB。请求默认 600 秒，`timeout_seconds` 可配置为 1–3600 秒，已有显式配置值仍优先于默认值，修改配置后重启网关生效；流写入 5 秒 deadline。客户端 `--timeout-seconds` 独立控制等待时间，不改变服务端执行上限。客户端超时取消请求后，不能恢复该次执行；超时/断连/关停复用同一进程树清理。

## 统计与内容保留

单实例文件存储：manifest.json、metadata/<request-id>.json 为长期执行元数据，requests/<request-id>.json 为长期 HTTP 请求元数据，content/<request-id>.json 为独立提示词/响应正文。文件采用同目录临时写入、Sync、Rename；持有排他 gateway.lock。已有存储缺损/损坏/未知 schema 时拒绝启动，不清空统计。升级旧存储时只新增 requests 目录，不回填历史请求。不能同时开两个实例共用存储。

所有进入 HTTP Handler 的请求输出 `request_started` / `request_finished` JSON 日志（Go 标准日志时间前缀），包括认证失败、参数错误、限流和 Models/Usage。`request_id` 与响应 `x-request-id`、执行元数据文件名一致；`principal` 是认证得到的网关主体，未认证为空，不表示后端 Codex 登录帐户。字段包含 `method`、`route`、`model`、UTC 接收/结束时间、`duration_ms`、`http_status`、`state`、`error_code`、`execution_started`、`delivery_failed`。未知路径/方法归 other；模型只记录已配置模型，未能确定时为空。日志不含凭据、正文、任意 URL 路径或查询原文。

SSE 已发送响应头后，HTTP 200 仍可能对应 failed/timed_out/cancelled，请按 `state` / `error_code` 判定执行结果。传输错误由 `delivery_failed` 表示。HTTP 记录先保存 in_progress，结束时保存终态；异常重启将未完成请求标 interrupted，结束时间和耗时保持 null，实际执行数从已有执行元数据确认。写入失败输出 `request_audit_failed` 并阻止新执行。

`GET /v1/usage` 保留原执行 `groups`，额外返回当前 `principal` 和 `http_groups`。HTTP 分组按接收时间在配置时区所属日、模型、接口、方法统计 `requests`、`succeeded`、`rejected`、`failed`、`timed_out`、`cancelled`、`interrupted`、`in_progress`、`executed`、`delivery_failed`、`http_statuses`、`error_codes`、`duration_samples`、`total_duration_ms`。平均已知耗时可用总毫秒数除以样本数；无样本表示未知。HTTP 分组只查询当前主体，查询自身可能计为 in_progress，未认证请求仅服务端文件/日志可查。跨零点的 HTTP 日归属与执行日归属可能不同。拒绝不进入执行 groups，不扣额度；token 仍只来自执行元数据。

HTTP 存储使用独立 `requests-manifest.json` 标记升级；已标记后 requests 目录缺失或版本损坏时拒绝启动，避免静默清空统计。升级中断在标记写入前可继续读取已有请求文件。请求元数据长期保留，文件数和启动读取成本随请求量增长；本轮不新增自动轮转或全局管理接口。统计不能补回升级前未记录的拒绝请求。

成功启动 Codex 消耗一次每日额度，后续成功/失败/超时/取消都不退；校验拒绝或启动失败不扣。尚未确认预留跨日期保守占额度，直到原子转换为实际启动记录或启动失败；新日请求同样计入全部待确认预留，防止跨零点超售。Usage 的 today_used 只计当日实际启动，today_reserved 表示所有仍占容量的待确认预留（含旧日），groups 仅汇总实际启动。RPM 是单实例滚动 60 秒，并发无队列。UTC 为默认统计时区，IANA timezone 可配置，历史时间保存 UTC，跨日按 started_at 归属。时区修改只能停服后进行，会改变历史分组；不要借此重置额度。

正文从请求预留时起保存 168 小时，启动及每 60 秒清理，重启不延期；更新已过期请求不重新建正文。异常保存已取得的部分文本；不保存 Key、完整环境、stderr/JSONL 或隐藏推理。运营者仅在服务端查看正文文件，V1 无历史正文 API/UI。到期清理的调度粒度为 60 秒，停服时由下次启动清理；离线服务器文件不是严格实时访问网关。部署方负责存储权限、备份及离线副本生命周期。

存储/正文写入或清理失败后停止新 AI 执行并输出脱敏错误，已获得用量仍尽量落盘。Usage 不把未知 token 当零，以 known 累计和 unknown_usage_requests 分开表示。

### 异常恢复

1. 确认旧 Proxy **及所属 Codex 进程树**均已退出；不得仅看到 Proxy PID 消失就删除锁。
2. 检查服务日志和 state 内容；确认没有活跃写入后移除遗留 gateway.lock，保存原记录备份。
3. started 记录恢复为 failed/backend_interrupted，原 started_at 仍扣额度；reserved 的启动窗口无法证明是否已执行，阻止该主体新执行并保留额度。
4. 对 reserved 记录离线核实：确未启动改 state=spawn_failed、started_at=null；已启动则补实际 started_at、finished_at、state=failed、error_code=backend_interrupted，再重启。不提供在线修改接口，不擅自核实或返还额度。

Sync/Rename 的断电耐久性依赖系统/文件系统，未承诺 OS spawn 与文件落盘原子。异常终止残留的临时工作区由运营者核实进程树已退出后处理，应用不自动删除未知旧工作区。

## 安全与迁移

development 仅 loopback 并检查 peer；shared 必须显式配置监听和完整认证/限额。共享前由用户提供符合要求的部署容器；Proxy/Codex 可共用容器。网关不操作 Docker、挂载、网络策略或部署凭证，不判断“当前在容器”即可证明安全。

每请求独立空 cwd，服务端固定禁工具/read-only 配置，环境只给运行必需白名单。cwd 和提示规则不提供访问隔离；共容器进程可能接触其它请求/配置/上游凭证。Linux 进程组无法约束恶意进程主动脱组；Windows Job Object 负责正常后端及其后代生命周期。部署与跨平台运行效果由用户验证。

旧 POST /v1/requests、WebSocket /v1/events、ACK 暂存、多 Provider 直连和 TypeScript 入口已移除。升级客户端至 plugins/aiw-ai/aiw-ai.mjs，设置逻辑模型与网关 Key，移除 AIW_AI_PROVIDER。旧临时状态不自动迁移、不删除，旧历史需求/报告保留。不再执行旧 npm build 或旧 smoke/check 脚本。

最窄离线 compile-only：`python program/agent-gateway/scripts/compile.py`，Windows/Linux amd64 输出 NUL，不保留产物，不运行应用/测试、不下载依赖。
