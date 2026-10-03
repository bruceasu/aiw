# FD-017: Go Shared AI Gateway

**Status:** Complete
**Revision:** 22
**Priority:** Medium
**Test policy:** External
**Evidence policy:** Dual

## Problem

远程个人与应用需要通过网关自己的凭证使用 Codex AI 能力，不能获得宿主文件、项目、凭证或命令执行能力。已批准的 `REQ00006-agent-proxy-shared-gateway` 要求在现有插件目录将 TypeScript 代理替换为 Go 网关，并迁移 `ai` 客户端。

当前服务通过 `/v1/requests` 暴露 Provider 选择，远程 HTTP 无内置认证；Codex 复用 `STATE_DIR/codex-workspace` 并继承服务环境。这些行为不能满足新的共享安全边界。

## Options and decision

1. 保留 TypeScript 代理，新增另一个共享网关：可避免旧接口迁移，但不符合用户已确认的原目录替换决定，且增加两套服务维护成本。
2. 在 `plugins/aiw-agent-proxy/` 迁移为 Go 网关（选定）：符合批准 Issue；允许移除旧接口，将 `ai` 客户端同步迁移作为交付的一部分。
3. 先将本机进程直接共享，后补隔离：不符合用户确认的共享访问前必须结构性隔离的要求，不采用。

2026-10-03 用户进一步确认：接受 Proxy 与 Codex 共用一个 Docker 容器。宿主隔离边界调整为整个部署容器；开发与共享部署均复用 Process Runner，Proxy 不再管理每请求容器，也不需要 Docker socket。宿主直接运行只用于可信本机开发；共享访问必须使用符合部署安全约束的容器。

每请求仍创建独立临时目录、仅输入本请求数据，并在请求结束时清理。此机制避免主动复用前次工作区，但同一容器中的进程仍可能读取其他请求目录或容器内服务配置；用户接受不提供请求之间的容器级访问隔离。不得宣称独立 cwd 能阻止这类读取。此决定替代批准 Plan 中“不能继承其他请求工作区数据”的强隔离验收解释；宿主隔离要求仍需满足，历史批准 Plan 保留。

临时 cwd 和提示词限制不能作为宿主隔离证明。HTTP 层只接受 AI 请求，Runner 与部署配置由运营者控制。

2026-10-03 用户明确要求：“你无须考虑容器部署和测试，我会另外安排。”本 FD 的工作边界收缩为应用设计与实现；容器镜像、部署配置、认证材料的部署供给、隔离检查及测试均由用户另行安排，不作为本 FD 的设计阻塞项或交付条件。应用仍须实现下方业务与安全行为，但本工作不宣称部署隔离或运行测试已通过。

Planner 工程设计已完成，后续通过真实 CLI handoff 自动实施和独立审查。

## Solution

- Go 服务仍位于 `plugins/aiw-agent-proxy/`；入口、API、认证、用量与 Runner 分开组织，优先标准库，不将其并入根 Go 常驻服务。
- 用户已确认 Proxy 服务端支持 Linux 与 Windows，`ai` 客户端亦保留 Windows/Linux 使用。公共 HTTP、配置、存储与统计逻辑共用；Codex 可执行入口查找、进程树终止和退出信号处理按平台分别实现，不以 Go 可跨平台编译代替这些行为。目标平台不再是人类待确认项，具体策略由 Planner 决定。
- 初始公共接口为 `POST /v1/responses`、`GET /v1/models`、`GET /v1/usage`，采用逻辑模型名；外部请求不得透传 CLI 参数、cwd、挂载、宿主路径或隔离控制。设计需列明 schema、错误与大小限制。
- 用户已确认兼容目标：官方 OpenAI SDK 仅替换 `base_url` 与 API Key 即可调用 V1 明确支持的 Responses/Models 操作，包括 Responses 流式输出。公共请求、响应、错误和 SSE 遵循支持子集的 OpenAI 契约，不只模仿字段风格；不支持的参数明确报错，不能静默忽略。不实现全部 OpenAI endpoint，不将本地统计接口宣称为 OpenAI 标准接口。
- Planner 在 1.1、1.49、1.50 明确支持操作/参数及依据，不能用旧 `output_text` 简化响应或自定义 SSE 事件替代 SDK 所需契约。底层 Codex 能力、客户端 option 与标准 API 的可验证映射留待设计，不以 SDK 兼容要求自动承诺不存在的后端能力。
- 各主体使用独立网关 Key，服务端仅保存 hash/HMAC；所有公共 AI/模型/用量接口需认证，用量按主体隔离。用户已确认 V1 由运营者维护 Key 配置，新增、轮换与禁用在 Proxy 重启后生效，不增加在线 Key 管理接口。1.3 明确配置结构、哈希生成方式与主体关联；轮换 Key 不改变主体身份、不清空其历史统计或每日计数。
- Process Runner 支持上下文取消、超时、输出上限及可靠清理，在 Proxy 部署容器内启动 Codex 子进程。每请求拥有独立临时工作区，不主动复用其他请求数据；请求终止时删除该目录，不销毁 Proxy 容器。
- 可信宿主开发模式只监听本机并拒绝远程共享接入；共享模式显式配置监听地址、认证和限额。隔离环境与可用 Codex 登录状态由用户提供，应用不管理 Docker、挂载、网络或认证材料部署。
- 应用负责每主体并发与限额、模式和必要配置校验，不尝试证明自身处于安全容器。提供 Codex 可执行路径、必要运行环境与临时目录等应用配置说明，供用户接入其部署方案。
- SSE、每主体与全局限流、并发及可取得的用量在基础执行链路之后完成。2026-10-03 用户明确要求：“必须保存，我需要统计。”每日请求限额计数及用量统计必须持久化并在重启后恢复，不能以重启清零方案交付。RPM/并发仍可采用单实例内存机制，不引入 Redis 或队列。
- 应用提供可配置持久化存储位置与恢复逻辑；数据不放在请求结束即删除的临时目录。存储介质及容器数据保留由部署方安排，应用需说明持久化需求，不替用户编写部署配置。统计内容仅为请求、主体、模型、时间、状态及可取得的用量元数据，不保存完整提示词、结果或凭证。
- 2026-10-03 用户确认：Codex 成功启动后，该请求消耗一次每日额度，即使随后失败、超时或取消也不退回；成功、失败、超时、取消分别统计。认证/参数/限额校验拒绝及启动失败未进入 Codex 执行的请求不消耗每日额度。启动前额度预留、启动失败释放和持久化恢复的一致性由 1.44 设计，不因并发预留而重复计费；统计未知 Token/成本仍保持未知。
- 每日额度和每日统计使用同一可配置时区，默认 `UTC`，按该时区零点划分自然日。用户已明确选择 UTC，不使用宿主/容器隐式本地时区作为默认值。持久化时间保存明确时间基准；跨日请求归属及修改时区后的计数一致性由 Planner 记录策略，不把统计分组变化当作删除历史。
- `GET /v1/usage` 支持指定起止日期，按天和模型汇总当前认证主体的请求数、成功/失败/超时/取消数量及可取得的 Token 用量；日期采用配置的统计时区，默认 UTC。未知 Token/费用明确标记未知，不能把缺失字段当作零或推算金额。日期参数名、区间端点、默认区间与单次查询上限由 Planner 在 API 契约中明确。
- 2026-10-03 用户确认统计元数据默认长期保留、不自动删除，并要求“提示词和正文单独记录保存7天”。用量统计与最小审计仍只记录必要元数据；另设与请求 ID 关联的内容存储，保存提示词及响应正文，按 7 天期限自动清理。内容过期不删除统计历史。
- 正文记录不混入长期统计/审计，也不扩展为保存 API Key、完整环境、内部堆栈或隐藏推理。未知 Token/成本保持未知；客户端错误不暴露内部路径或凭证。用户已确认 V1 正文记录仅供运营者在服务端查看，不增加调用者历史正文查询 API；`GET /v1/usage` 只返回允许的统计元数据。服务端查看使用单独内容存储，不新增管理 UI 或管理接口。7 天起算点、异常/流式结果处理与清理频率由 1.47 明确，不能无限延长保留期。
- 旧 `/v1/requests`、`/v1/events` 与 ACK 结果暂存允许移除，旧客户端不再保证兼容。迁移当前 `plugins/aiw-ai/aiw-ai.mjs` 的 endpoint、网关认证与模型配置；保留单次输入、无自动付费重试和 stdout/诊断分离的可用行为。`--system`、`--json`、`--verbose` 必须在接口设计中定义支持或明确报错，不能静默丢弃。
- 已批准 Issue 写有历史客户端目录 `plugins/aiw-agent-proxy-client`，当前快照实际代码在 `plugins/aiw-ai/`；本 FD 采用实际路径完成同一客户端迁移，不修改批准来源或重建旧目录。

## Scope

本 FD 交付 Go 网关替换、基础 API 与独立认证、共用 Process Runner、SSE、限流与用量、当前 `ai` 客户端迁移及相应稳定规格和应用运行说明。容器部署与全部测试由用户另行安排：不编写镜像/部署/网络策略，不配置部署凭证，不设计、编写或运行测试，不自动派发 Tester。实现后的最窄 compile-only 与静态检查仍按仓库要求执行，不生成最终发布物。

附件属于后续阶段，会话仅在有明确需求时考虑：本 FD 不实现附件与会话，不以延期为由允许请求宿主路径。批准 Issue 中这些后续目标继续保留，尚未计入本 FD 已交付结果。

不实现 UI、SSO/OAuth、复杂计费、Redis、队列、Kubernetes、仓库检出、多区域或多 Agent 编排。不创建 OpenSpec change；未来契约实施变化时更新 `openspec/specs/agent-proxy/spec.md` 与 `agent-proxy-client/spec.md`。

用户已授权自动实施；使用主工作区，不下载依赖、不测试、不部署，不混入已有技能修改。

## Engineering decisions

用户已授权按建议完成工程设计及 `$implement $fd-workflow auto handle FD-017`。本节为 1.1–1.6、1.44、1.47、1.49–1.50 的实际设计证据。

### HTTP、SDK 与客户端（1.1、1.2、1.49、1.50）

全部接口使用 Bearer 网关 Key，响应带 x-request-id。只实现 POST /v1/responses 文本单轮 create、GET /v1/models list、本地扩展 GET /v1/usage；不实现历史正文查询、会话、附件、工具调用及 Responses retrieve/delete。

请求允许 model、input（非空字符串或 user/assistant 文本消息数组，content 支持字符串/input_text 数组）、instructions、stream、store:false、background:false、tools:[]、text.format.type=text/json_object。其余字段与不支持的非默认值报 400；JSON schema 本阶段明确不支持。body 上限 1 MiB，合并文本上限 64 KiB UTF-8；拒绝重复字段、尾随 JSON 和非法 UTF-8。instructions 映射 Codex developer_instructions，固定禁工具规则不允许请求改写。json_object 采用后端提示约束及完整结果解析校验，错误返回 backend_error，无自动重试。

标准 Response 包含 id（resp_随机值）、object=response、created_at、status、model、output（assistant message，content 的 output_text 含 text、annotations=[]、logprobs=[]）、error、incomplete_details、usage、instructions、tools、tool_choice、parallel_tool_calls、text、metadata、store、background、previous_response_id、temperature、top_p、max_output_tokens、reasoning、truncation。未知 usage=null，已知 input/output 时 total 为二者之和，未知 detail=null，不能假造零。Models 返回 object=list 和 data，每模型 id/object=model/created/owned_by=aiw，仅返回主体允许的逻辑模型。错误 envelope 为 error={message,type,param,code}，公共 message 固定脱敏。状态码：400 参数、401 认证、404 路由、405 方法、413 大小、429 RPM/每日额度、500 存储/清理、502 后端、503 并发、504 超时。

SSE 帧包含 event 和 JSON data，type 同事件名、sequence_number 单调递增。依次 response.created、response.in_progress；每条实际收到的 Codex item.completed/agent_message 对应 output_item.added、content_part.added、output_text.delta、output_text.done、content_part.done、output_item.done（事件名均以 response. 开头）。item/part 事件含 output_index，part/text 事件还含 item_id、content_index；delta 含 delta/logprobs=[]，done 含完整 text/part/item。最后 response.completed 或 response.failed 携带最终 response。无自定义 token/done 或 [DONE]。Codex exec JSONL 不保证逐 token delta，首条文本可能在消息完成后到达，不模拟字符分片。json_object 在完整解析成功后输出文本。流前错误用 HTTP；流中错误用 failed，断连/5 秒写入超时取消同一执行。

ai 的 --url 保留 origin 校验，追加 /v1/responses；默认 loopback:43127，AIW_AI_API_KEY 与 AIW_AI_MODEL 必填，不再 provider 路由。--system（含 @UTF-8 文件）映射 instructions；--json 映射 json_object，stdout 保证单个 object；--verbose 仅 stderr 输出真实用量及 unknown cost/unavailable summary。保留空输入不请求、完整 stdin、无重试、65 秒超时。

### 配置、身份与限额（1.3、1.4、1.6）

独立 Go 标准库模块，从 --config JSON 加载。mode 默认 development，监听默认 127.0.0.1:43127，仅 loopback 且检查 peer；shared 必须显式监听配置。state_dir/workspace_dir 为两个不同且不嵌套的绝对目录。codex_path 绝对且存在；Windows 支持 .exe，npm .cmd 需改指定真实 node.exe 与 codex_script 的 codex.js，不运行 shell。models 是逻辑模型映射；principals 包含稳定 id、enabled、key_hashes、allowed_models、rpm、daily_limit、concurrency。global_concurrency 默认 4，timeout_seconds 默认 60，输出文本上限 256 KiB、原始 JSONL 上限 4 MiB；限制均为有界正整数。timezone 默认 UTC，内嵌 tzdata 提供 IANA 时区。

网关 Key 至少 32 随机字节，配置仅保存 SHA-256 hex，以常量时间比较；禁止重复 hash/主体。运营者离线生成，新增/轮换/禁用重启生效，无在线管理。轮换在同一 principal 新增 hash，重启后删旧 hash 再重启；稳定 id 不变、不得重新分配给别人。

通过认证/参数后计滚动 60 秒 RPM（含限额拒绝），GET 不扣额度。并发即时拒绝，无队列。每日按 started_at 在配置时区的日期归属；成功启动扣一次，后续失败/超时/取消不退，spawn_failed 不扣；预留占额度防超售。UTC 时间持久化；时区变更仅停服后进行，历史重分组而不删除记录。Usage start_date/end_date 均 YYYY-MM-DD、双端包含，默认今天，最多 366 自然日，不用 24h 推进 DST 日期，不接受其他主体参数。返回 timezone、日期、daily_limit、today_used（实际启动）、today_reserved（尚未确认预留）、按 date/model 分组的已启动 requests/四类状态/in_progress、已知 token 累计、unknown_usage_requests、cost=null；无记录 groups=[]。

子进程环境白名单为 PATH、HOME、USERPROFILE、SystemRoot、WINDIR、APPDATA、LOCALAPPDATA、TEMP、TMP、CODEX_HOME、HTTP(S)_PROXY、NO_PROXY（兼容小写）。不复制 Key/完整环境、不探测登录。固定 Codex exec --json --sandbox read-only --skip-git-repo-check、approval never，禁 shell/hooks/apps/multi_agent/web_search。登录环境由用户预配置。目录 0700、文件 0600，Windows ACL 由部署方负责。

### 进程树（1.5）

Linux 每请求独立 process group：取消、超时、关停或根进程结束均向该组 SIGTERM，2 秒后 SIGKILL，额外最多等 3 秒。Windows 10/11 为每请求独立 Job Object、KILL_ON_JOB_CLOSE，CreateProcessW suspended 创建，AssignProcessToJobObject 后 ResumeThread；失败在恢复前销毁。限制继承为标准 pipe handles；取消及根进程结束 TerminateJobObject，最多等 5 秒。启动归 1.53，Job 管理归 1.52。只有树终止后删除目录；清理错误不返回成功。服务关停停止接入、取消活跃请求，最多 10 秒等待收尾。Linux 恶意进程主动脱离进程组属于残余风险，容器隔离由用户安排。

### 持久化与保留（1.44、1.47）

每请求一个 schema_version=1 元数据 JSON，标准库实现，同目录随机临时文件写入/Sync/Close/Rename 原子替换，单锁串行写。元数据为 id/principal/model、reserved_at/started_at/finished_at UTC、state（reserved/started/四类终端/spawn_failed）、nullable tokens、脱敏 error code，不存正文。state_dir 有 manifest；损坏、未知版本、已有非空无 manifest 目录拒绝启动。排他 lock 防多实例；异常遗留锁仅在运营者确认旧进程退出后离线移除。

执行顺序：验证、占槽、持久预留、保存输入、spawn、持久 started、消费输出、终止树、删目录、保存终端正文/元数据、响应。写入失败不继续执行或返回成功，并停止新 AI 请求。spawn 失败记录 spawn_failed 不扣额度。重启 started 转 failed/backend_interrupted，不更改 started_at、不重计。reserved 的崩溃窗口可能已经 spawn；保留待核实、占额度并拒绝该主体新执行。运营者核实后离线明确改为 spawn_failed 或 started/failed，再重启。不能假称 OS spawn 与磁盘事务原子。统计文件损坏 fail closed，不能清零。

content/<id>.json 独立保存 input/instructions/output 与公开状态、created_at、expires_at=reserved_at+168h。异常保存已得部分文本，不保存 stderr/JSONL/推理/Key。先保存输入再执行，终端写入失败可观测且停止新执行。启动与每 60 秒仅在 content 根清理严格 ID 的到期文件，重启不延期；更新时已过期则删而不重建。清理失败阻止新执行。长期统计不自动删，正文过期不影响计数。无正文管理接口；目录权限由部署安排。断电目录项耐久性依赖 OS/文件系统，不声称硬件级事务。

### 依据与顺序

协议依据：[Responses create](https://developers.openai.com/api/reference/python/resources/responses/methods/create)、[Models list](https://developers.openai.com/api/reference/python/resources/models/methods/list)、[SSE](https://developers.openai.com/api/reference/resources/responses/streaming-events)。后端依据：[Codex non-interactive](https://learn.chatgpt.com/docs/non-interactive-mode)、[Config reference](https://learn.chatgpt.com/docs/config-file/config-reference)。支持子集、存储、限额与平台策略是本项目设计，未执行 SDK/Codex 或平台运行验证。

依赖顺序为配置/契约、存储恢复、进程适配、执行/限额、客户端/SSE、退役/文档。各实现项保持单一成果和约半天目标，超出时继续拆分，不抹去跨平台风险。

## Work items

### 规模标准与原项评估

每项只交付一个可单独审查的结果，目标工作量约半天以内、难度低或中；这是基于当前设计的估计，不是工时承诺。每项包含必要的局部接线与说明，但不顺带修改其他能力。若落实设计后仍需多个机制、跨平台策略或未知外部契约，先继续拆分或记录 Gate，不将其自动视为中低难度。

| 原编号 | 评估 | 过大原因 | 拆分后编号 |
| --- | --- | --- | --- |
| 1.1 | 大 / 高不确定性 | API、客户端、Key、统计、进程与容器安全的多项设计 | 1.1–1.7 |
| 1.2 | 大 / 中高 | 工程入口、配置、身份认证、模型目录与 Responses 接线 | 1.8–1.13 |
| 1.3 | 大 / 高 | 工作区、环境、协议解析、进程树取消及失败清理 | 1.14–1.19 |
| 1.4 | 中 / 中 | 配置认证、请求迁移、输出与选项行为 | 1.20–1.22 |
| 1.5 | 中 / 中高 | SSE 编码与连接终止时的执行取消 | 1.23–1.24 |
| 1.6 | 大 / 中高 | 三类限额、并发槽位、用量与审计 | 1.25–1.30 |
| 1.7 | 大 / 高（原方案） | 每请求容器已取消，服务容器部署由用户安排；仅保留应用配置检查 | 1.37 保留；1.31–1.36、1.41 取消 |
| 1.8 | 中 / 中 | 服务退役、运维文档与两份稳定规格 | 1.38–1.40 |

上表保留最初拆分的范围对应关系。共容器修订及用户另行安排部署/测试取消了 1.7、1.31–1.36、1.41；新增 1.42–1.53。共 53 项，8 项取消，45 项在当前交付范围内；保留编号与取消原因，安全风险不会因任务变小而降低。

### Planner 设计项

- [x] 1.1 定义 SDK 兼容请求支持子集【小 / 中】。完成标准：明确 Responses/Models 的支持操作、请求参数、输入上限及不支持参数的拒绝规则，并记录 OpenAI 契约和 Codex 能力的映射依据；不混入响应/SSE 设计或 Runner 实现。本地统计查询参数在 1.4 单独设计。
- [x] 1.2 定义 `ai` 客户端参数映射【小 / 低；依赖 1.1】。完成标准：URL、逻辑模型、Key 来源及 system/json/verbose 的行为按确认的 OpenAI API 兼容契约记录；不为保留旧参数形状改变标准语义。
- [x] 1.3 定义 Key 生命周期【小 / 中】。完成标准：运营者维护配置且重启生效、不增加在线管理接口已确认；明确主体关联、hash/HMAC 存储与生成方式、配置校验及轮换/禁用步骤，不因换 Key 重建主体或丢失统计。
- [x] 1.4 定义计数与 Usage 语义【小 / 中】。完成标准：Codex 成功启动后消耗一次每日额度，后续失败/超时/取消不退回，各终止状态分开统计；每日额度/统计采用同一可配置时区、默认 UTC。起止日期查询及按天/模型汇总请求数、状态数量、实际 Token 已确定，明确跨日归属、区间端点、默认区间、查询上限及部分未知用量的表示。不重复询问已确认业务决定。
- [x] 1.5 定义进程取消与清理策略【小 / 中】。完成标准：Linux/Windows 均须支持已确认，分别记录可执行入口及进程树终止策略，统一超时/关停顺序、等待边界和清理失败处置；平台实现已拆为 1.51–1.53，不重复要求用户选择实现机制。
- [x] 1.6 定义 Process Runner 的应用配置边界【小 / 低】。完成标准：明确可执行路径、临时目录、子进程环境白名单和预配置 Codex 登录状态的应用约定；部署方负责提供可用环境，不设计容器内凭证供给或认证刷新。
- [-] 1.7 Proxy 部署容器策略设计取消：用户明确将容器部署另行安排，应用设计不等待基础镜像、挂载或网络策略。
- [x] 1.44 定义持久化记录与恢复契约【小 / 中；依赖 1.4】。完成标准：明确本地存储方案、必要元数据、请求唯一标识、写入时点、并发更新、重复记录处理、未完成请求恢复及写入失败处理；不增加外部数据库部署或迁移平台。
- [x] 1.47 定义内容存储与 7 天保留契约【小 / 中；依赖 1.4、1.44】。完成标准：正文仅供服务端运营者查看已确定；明确记录起止时点、过期起算点、流式/失败/取消结果处理、清理频率与启动清理规则。内容与长期统计分离，过期不影响历史计数，不增加历史正文查询 API、管理 UI 或管理接口。
- [x] 1.49 定义 SDK 兼容响应与错误契约【小 / 中；依赖 1.1】。完成标准：记录支持操作所需的完整响应对象、Models 列表、用量缺失表达及错误结构/HTTP 状态；以明确依据为准，不用简化自定义对象声称兼容。
- [x] 1.50 定义 SDK 兼容 SSE 契约【小 / 中；依赖 1.49】。完成标准：支持子集中的增量、结束及失败事件结构和顺序明确，给出内部事件映射；不混入连接取消实现，不凭记忆编造标准事件。

### 基础网关

- [x] 1.8 建立插件 Go 模块与启动入口【小 / 低；依赖 1.1–1.6、1.44、1.47、1.49–1.50 设计完成】。完成标准：独立启动路径与 compile-only 入口明确；不改根服务或移除旧实现。
- [x] 1.9 实现配置加载与启动校验【小 / 中；依赖 1.8】。完成标准：按确定 schema 读取配置，缺失/非法值在监听前失败；错误不泄露秘密。
- [x] 1.10 实现 Key 校验与主体解析【小 / 中；依赖 1.3、1.9】。完成标准：启动时加载运营者配置，有效 Key 得到稳定主体，无效或禁用 Key 在进入业务 handler 前被拒绝；不新增运行期间管理/热加载机制。
- [x] 1.11 实现逻辑模型目录与 Models 接口【小 / 低；依赖 1.10】。完成标准：只向认证主体返回配置的逻辑模型；未知模型不能进入 Runner。
- [x] 1.12 实现 Responses 请求解码与校验【小 / 中；依赖 1.1、1.10–1.11】。完成标准：大小、字段和类型按 schema 检查；危险控制字段及不支持参数在调用 Runner 前失败。
- [x] 1.13 定义内部 Runner/Event 契约并接线 JSON Responses【小 / 中；依赖 1.12、1.49】。完成标准：handler 通过内部契约获取结果，序列化支持子集的标准响应/错误对象，使 SDK 可解析；不直接拼接 Codex 命令。

### 共用 Process Runner

- [x] 1.14 实现请求工作区分配与删除【小 / 低；依赖 1.13】。完成标准：每请求独立目录，目录创建失败不启动执行，清理结果可返回调用方管理层。
- [x] 1.15 实现受控执行参数与环境供给【小 / 中；依赖 1.9、1.14】。完成标准：工作目录、参数及环境只来自服务端策略，不复制全部服务环境，不调用 shell 拼接命令；提供平台入口解析边界与 Linux 可执行路径解析，Windows 特殊入口由 1.53 处理。
- [x] 1.16 实现 Codex 输出到内部事件的解析【小 / 中；依赖 1.13、1.15】。完成标准：解析已确认协议中的文本、结束、错误及可用用量；未知成本保持未知，不处理容器生命周期。
- [x] 1.17 接线通用取消、超时与退出管理【小 / 中；依赖 1.5、1.51–1.52】。完成标准：取消/超时/关停调用统一平台终止边界，按设计等待退出并收尾；不在此项同时实现 Linux 与 Windows 进程控制。
- [x] 1.18 接线输出上限与最终清理【小 / 中；依赖 1.14、1.16–1.17】。完成标准：所有终止路径执行同一清理收尾；超限终止，不把清理失败隐藏为成功。
- [x] 1.19 区分宿主开发与容器共享运行模式【小 / 低；依赖 1.9、1.18】。完成标准：两者都使用 Process Runner，宿主开发模式拒绝远程绑定/接入；共享模式需显式配置，模式开关本身不证明宿主隔离。
- [x] 1.51 实现 Linux 进程树终止适配【小 / 中；依赖 1.5、1.15】。完成标准：按确定策略管理请求进程及其子进程，支持取消/超时/关停，返回退出或失败信息；不误终止其他请求或服务进程。
- [x] 1.52 实现 Windows 进程树终止适配【小 / 中；依赖 1.5、1.53】。完成标准：按确定 Windows 策略终止请求进程及其子进程，提供与 Linux 一致的取消/退出接口；不把仅杀父进程当作进程树清理完成。
- [x] 1.53 实现 Windows Codex 入口解析与启动适配【小 / 中；依赖 1.5、1.15】。完成标准：支持设计规定的可执行路径或既有 CLI 安装入口，正确处理路径、参数与后台进程启动；不拼接任意 shell 命令，不下载或安装 Codex。

### 客户端迁移

- [x] 1.20 迁移客户端 URL、模型与网关认证配置【小 / 低；依赖 1.2、1.10–1.11】。完成标准：使用网关 Key 与逻辑模型，不读取上游凭证，不将 Key 写入日志。
- [x] 1.21 迁移单次 Responses 请求与响应读取【小 / 中；依赖 1.13、1.20】。完成标准：调用新 endpoint，空输入不请求，错误有界且不重试，正常输出取自稳定响应字段。
- [x] 1.22 落实 system/json/verbose 选项及输出约定【小 / 中；依赖 1.2、1.21】。完成标准：按已决定映射执行或明确拒绝，stdout/诊断分离，不假造摘要或费用。

### Streaming

- [x] 1.23 实现 SSE 事件编码与发送【小 / 中；依赖 1.13、1.16、1.50】。完成标准：内部事件映射到 1.50 定义的 SDK 兼容增量与终止事件，顺序及流中失败表达符合契约，不用自定义事件替代，不输出内部敏感字段。
- [x] 1.24 接线断连取消与写入失败收尾【小 / 中；依赖 1.17–1.18、1.23】。完成标准：连接终止取消同一请求上下文，复用 Runner 清理，不重复实现进程终止或增加重试。

### 限额、用量与审计

- [x] 1.25 实现每主体 RPM 限制【小 / 低；依赖 1.4、1.10】。完成标准：按确定窗口统计，超限在启动 Runner 前拒绝，不引入共享存储。
- [x] 1.26 实现每主体及全局并发槽位【小 / 中；依赖 1.4、1.18、1.24】。完成标准：无可用槽位时拒绝，完成/失败/取消后释放；不增加任务队列。
- [x] 1.27 实现每日请求限额【小 / 中；依赖 1.4、1.10、1.42–1.43】。完成标准：从已恢复持久化计数判断额度，按 1.44 契约预留/确认；成功启动后扣一次，后续失败/超时/取消不退回，启动前拒绝或启动失败不消耗，重启不重获已消耗额度；按配置时区的自然日划分额度，默认 UTC。
- [x] 1.28 接线主体用量持久化【小 / 中；依赖 1.4、1.16、1.18、1.42】。完成标准：按请求标识记录终止状态及实际元数据，成功/失败/超时/取消分别可统计，恢复或重复处理不重复计入，未知 Token/成本保持未知；复用存储组件，不在此项另写存储实现。
- [x] 1.29 实现 Usage 查询接口【小 / 低；依赖 1.10、1.48】。完成标准：校验起止日期与查询边界，只查询当前认证主体的数据并序列化约定结果；不接受任意主体 ID 越权查询，响应不含正文。汇总算法复用 1.48，不在此项重复实现。
- [x] 1.30 实现最小审计输出【小 / 低；依赖 1.18、1.28】。完成标准：字段白名单及错误脱敏覆盖请求终止路径，默认无正文、Key 或完整环境。
- [x] 1.42 实现本地统计存储组件【小 / 中；依赖 1.9、1.44】。完成标准：按确定契约持久化必要记录，提供幂等更新与读取能力；错误不静默丢数据，不写入请求临时目录，不添加容器部署逻辑。
- [x] 1.43 实现启动恢复与统计重建【小 / 中；依赖 1.42】。完成标准：读取已有记录恢复每日额度与历史统计，按契约处理未完成请求及损坏记录；只读到空数据不能未经区分就当作新存储清零。
- [x] 1.45 实现独立提示词/响应正文记录【小 / 中；依赖 1.18、1.44、1.47】。完成标准：按请求 ID 单独写入内容存储，正常与异常结果按设计记录，写入失败可观测；不写入长期统计/审计，不保存凭证或隐藏推理。
- [x] 1.46 实现正文 7 天过期清理【小 / 中；依赖 1.45】。完成标准：按确定时点识别过期记录，启动与周期清理幂等，重启不重置保留期限；仅删除已过期内容，不删除统计元数据，清理失败可观测。
- [x] 1.48 实现日期区间与每日/模型统计汇总【小 / 中；依赖 1.4、1.28、1.43】。完成标准：从保留的当前主体元数据按配置时区和日期范围分组，汇总请求数与四类状态，累计可取得 Token 并明确缺失情况；费用未知时不生成估算金额，不扫描正文存储。

### 外部安排与应用配置检查

- [-] 1.31 共容器运行配置与挂载边界交付取消：容器部署由用户另行安排。
- [-] 1.32 容器内 Codex 认证材料供给取消：用户负责预配置可用 Codex 环境，应用仅落实 1.6 的调用约定。
- [-] 1.33 每请求容器创建、启动与事件接线取消：用户确认 Proxy/Codex 共容器，统一复用 1.13–1.18 的 Process Runner，无此独立交付。
- [-] 1.34 每请求容器取消与销毁取消：容器属于整个服务，单请求结束只终止其进程并清理工作区，服务关停由 1.5、1.17 与部署配置处理。
- [-] 1.35 部署容器 CPU/内存/进程限制配置取消：由用户另行安排，应用并发与限额仍由 1.25–1.27 交付。
- [-] 1.36 部署容器出站网络策略取消：由用户另行安排。
- [x] 1.37 实现共享模式应用前置检查【小 / 中；依赖 1.6、1.19、1.25–1.30】。完成标准：必要网关认证、模型、可执行路径及限额缺失时失败关闭；不自动回退为宿主开发模式，不通过执行登录探测或 Docker 命令检查部署环境。
- [-] 1.41 Proxy/Codex 共用镜像定义取消：镜像与部署由用户另行安排。

### 退役与文档

- [x] 1.38 移除旧服务入口、旧接口与 ACK 暂存路径【小 / 中；依赖 1.22、1.24、1.37、1.46】。完成标准：仅退役已被新能力替代的路径，检查入口/配置引用；不删除历史需求、证据或用户状态数据。
- [x] 1.39 更新服务与客户端运行、升级说明【小 / 低；依赖 1.38】。完成标准：启动、Key 配置、开发/共享边界、旧接口替换和客户端升级步骤明确，无未实现能力声明。
- [x] 1.40 更新两份稳定能力规格【小 / 低；依赖 1.38】。完成标准：规格与最终实现/验收边界一致，保留历史 FD，不生成 OpenSpec change。

1.1–1.6、1.44、1.47、1.49–1.50 是 Planner 的设计产物；全部完成且材料 Gate 已解决后才能 emit `design-ready`。其余未取消项是已授权 Worker 的交付项。按每项显式依赖执行，1.42–1.43 先于每日限额接线；不以编号数值作为唯一顺序。相同 Process Runner 收尾机制由 JSON、SSE 及开发/容器运行复用。

应用交付不依赖本 FD 提供容器部署或测试证据；外部共享开放及隔离验收由用户安排。Reviewer 仅审查应用 diff 与静态/compile-only 证据，不以未执行外部测试判定本 FD 未交付，也不宣称运行行为已验证。完成标准描述应提供的结果，不代表已执行运行验证。

## TODO

- 第三轮独立静态审查已闭环 R1/R1a/R2：Store.mu 内选择实际当前日，全部未确认预留跨日占容量，真实 StartedAt 归属保持；独立 watcher 贯穿标准管道消费期。当前无剩余阻塞发现；首两轮历史报告保留原结论。

- 当前 45 项范围内 Work Items 已提供设计/代码/文档，8 项取消；勾选代表编写完成，不代表测试通过。
- 实现证据：1.8–1.13 为 go.mod/main.go/config.go/protocol.go/server.go；1.14–1.19、1.51–1.53 为 runner.go/process_linux.go/process_windows.go；1.20–1.22 为 aiw-ai.mjs；1.23–1.24 为 eventStream 和 context 取消；1.25–1.30、1.42–1.48 为 Gateway/Store；1.37–1.40 为启动校验、旧服务退役、README 与稳定规格。
- 第三轮 Reviewer 双格式通过报告为 `docs/features/reviews/FD-017-review-r21.md/.json`；完成及归档以其真实 verification-passed receipt 和后续 CLI close 为准，不做 Git 写操作。
- 容器部署、测试、SDK/平台运行验收均由用户安排；附件/会话保留为后续目标。

## Acceptance

1. 有效 Key 与逻辑模型可取得稳定响应 ID、状态和结果；无效/禁用 Key 在启动 Runner 前被拒绝。
2. 旧接口允许移除；升级后的 `ai` 客户端调用新 Responses 接口，正常输出可管道使用，错误不泄露正文或凭证；空输入不请求，无自动重试。
3. 模型、Usage 仅对认证主体可见，Usage 不泄露其他主体数据；达到配置限额时额外调用被拒绝。
4. cwd、挂载、sandbox 禁用等请求控制不能改变服务器策略；未知或不支持选项按照明确 schema 报错，不静默降级安全配置。
5. 可信开发请求使用独立空临时目录，结束/失败/超时/取消后清理，开发模式不接受共享接入；不宣称本机进程完全隔离宿主。
6. 应用使用部署方提供的 Codex 环境与固定服务端执行配置，不操作 Docker 或接收请求级宿主路径/挂载参数。每请求独立工作区且不主动复用其他请求数据，不要求容器内请求之间的访问隔离；部署与宿主隔离验收归外部安排，不属于本 FD 的应用交付验收。
7. SSE 返回增量与终止事件，断连、超时和关停终止请求执行，清理不留下请求进程和临时数据；单请求结束不销毁 Proxy 容器。失败清理必须可观测。
8. 长期统计和最小审计不含完整提示词/结果，正文仅在单独内容存储保存 7 天；公共错误不泄露正文、凭证或宿主细节。未知 Token/成本明确为未知，不估造金额。
9. 使用同一已保留存储重启 Proxy 后，每日已消耗额度与历史用量仍在，不重复计数、不重获额度；Usage 可以查询保留的统计。该验收是待实现行为，本轮没有运行重启检查。
10. Codex 启动后即使失败、超时或取消，也只消耗一次每日额度，不退回；四类终止状态可分别统计。启动前被拒绝或无法启动 Codex 时，不消耗每日额度；实际 Token/成本不可得时保持未知。
11. 统计元数据默认不自动删除；提示词与响应正文记录在独立内容存储，按确定的 7 天到期策略清理。Proxy 重启不延长内容保留期限，内容删除不影响历史统计。
12. V1 不提供历史提示词/响应正文查询 API，Usage 响应与最小审计不包含正文；运营者通过服务端独立内容存储查看尚未过期记录，不新增管理 UI 或管理接口。
13. 未配置时区时，每日额度与每日统计均按 UTC 零点划分自然日；显式配置其他支持时区时，两者一致采用该时区，不受宿主/容器本地时区变化影响。
14. 当前主体可按有效起止日期查询按天、模型分组的请求数、四类终止状态数量与可取得 Token；服务重启后历史区间可查询。未知费用和缺失 Token 明确表示未知，不泄露其他主体统计或提示词/正文。
15. 运营者更新 Key 配置并重启后，新 Key 可用，已移除或禁用 Key 被拒绝；轮换到同一主体的新 Key 后，其原统计与当日额度计数保留。不提供在线 Key 管理接口。
16. 官方 OpenAI SDK 替换 `base_url` 与 API Key 后，可直接调用支持的 Responses/Models 操作及 Responses 流式输出，得到可解析的标准响应/错误/SSE。不支持的参数明确报错，不声明全 API 兼容。SDK 运行验证由用户另行安排，本轮没有运行 SDK。
17. Proxy 可在 Linux 与 Windows 使用已配置的 Codex 环境运行，公共接口/统计行为一致；平台差异限于入口解析与进程控制。取消、超时或服务关停后不遗留所属请求子进程。客户端继续支持 Windows/Linux；跨平台运行测试由用户安排。

## Verification

### 当前实现验证

- 第三轮 Reviewer `fd017-reviewer-20261003-a7d204` 已 claim `FD-017-000021-implementation-ready` 并独立静态审查：R1/R1a/R2 全部闭环，无剩余阻塞发现，结论 Verification Passed（应用静态交付）。双格式证据为 `docs/features/reviews/FD-017-review-r21.md/.json`；真实 verification-passed 事件结果以 CLI receipt 为准。
- 最终修复的离线 Windows/Linux amd64 compile-only exit 0 由第三轮 Worker 报告 `docs/features/reports/FD-017-implementation-r20.md/.json` 记录；Reviewer 未重新编译。测试、SDK/平台运行及部署仍未执行，不代表上线可用性通过。

### 历次实现与审查证据

- 第三轮实现来源 `FD-017-000020-changes-requested`，仅修正 Reserve 在锁内选择真实当前日，复用全部跨日预留占用规则；最窄离线 compile-only 结果记录在 `FD-017-implementation-r20.md/.json`，不运行午夜竞争测试。

- 第二轮独立 Reviewer 已 claim `FD-017-000019-implementation-ready`：R2 管道 watcher 修复静态完成；R1 仍残留 Reserve 锁内使用早前 ReservedAt 日期的遗漏，退回仅修复 R1a。证据为 `docs/features/reviews/FD-017-review-r19.md` 及同名 JSON；本自动周期已审查两次，没有运行测试或重新编译。
- 首轮审查真实 changes-requested 来源 `FD-017-000018-changes-requested`；Worker 以原 session claim 后修复 R1/R2。修复设计明确所有未确认预留跨日保守占额度，today_reserved 也含旧日；对实际启动的日期归属不变。独立 context.AfterFunc watcher 保持 stdout/stderr/stdin 消费可取消，即便根进程已退出，也不因脱组后代持有管道而永久阻塞。

- 首轮独立 Reviewer `fd017-reviewer-20261003-a7d204` 已 claim `FD-017-000017-implementation-ready`；静态审查发现跨日预留超售与根自然退出后超时不能解除 stdout 阻塞，结果为 Changes Requested。双格式证据：`docs/features/reviews/FD-017-review-r17.md` 及同名 JSON；通过真实 CLI 退回 Worker 修复 R1/R2，测试/容器仍由用户外部安排。
- 已执行最窄离线 compile-only：`python plugins/aiw-agent-proxy/scripts/compile.py`，Windows/Linux amd64 最终源码均编译成功，未保留发布产物。实现过程中另执行过一次相同命令，两次均 exit 0；第二次用于已完成源码，不是重跑失败命令。
- 已静态检查最终范围 diff、标准 Response/SSE 字段、调用和清理顺序、Key 配置、持久化恢复与正文到期处理；最终证据见 `docs/features/reports/FD-017-implementation-r16.md` 及同名 JSON。
- 未运行服务、Codex、SDK、测试、lint、formatter、vet、最终产物构建或容器命令；这些运行行为仍未验证。自动生命周期进入独立 Reviewer 后以其真实 receipt 为准。

### 历史设计修订记录

以下各条描述当时的设计活动及未运行项，不覆盖上方当前实现结果。

- 本次仅静态审查批准 Plan/决策、当前服务 Provider 调用、客户端输入配置/请求路径、相关稳定规格及 FD-009；未获得运行证据。
- 实现后按仓库预算执行一次最窄 compile-only 检查；不生成最终发布物。命令在实际 Go 布局确定后记录，不能使用完整 build/test 替代。
- 静态跟踪认证到 Runner 的调用顺序、参数过滤、工作区与环境供给、取消/清理、主体限流/用量及客户端响应解析；检查最终 diff 与升级文档。
- **Test policy: External**：按用户明确决定，测试由其另行安排。本工作不编写/运行测试，不创建测试授权、覆盖率或测试报告，不自动派发独立 Tester；依据当前 CLI，仅 Independent 标记启用 Tester 路由，External 保留直接 Reviewer 路由。
- Reviewer 检查应用设计与实现的静态一致性及实际 compile-only 结果。运行行为和隔离效果未测，不能宣称通过；缺少用户另行安排的测试/部署结果不阻塞本 FD 的应用交付，不等于已验证上线可用。
- 当前没有执行测试、编译、最终构建、lint、格式化、网络/Provider 请求或独立审查。
- 本次规模评估静态核对原 8 项与新 40 项的范围对应、依赖顺序、独立验收边界，以及未决设计对难度的影响；只修改本 FD 的计划、TODO 与 Verification。
- 共容器修订静态核对了用户最新决定与历史 Issue 的差异，更新 Process Runner、部署交付、取消项及验收；尚未执行 Docker 命令、构建镜像或取得容器运行证据。
- 本轮通过 `plugins/aiw-fd.py` 的 Test policy 解析静态确认 External 不启用 Independent Tester 路由；同步取消部署项、删除其前置依赖和部署设计 Gate，仅保留应用设计。
- 持久化修订静态核对用户决定、存储/恢复/业务接线的任务边界与依赖，新增重启保留验收；未进行持久化实现、重启测试或容器操作。
- 本轮静态核对每日额度计入规则与 1.4、1.27、1.28 及验收的一致性，删除已确认问题；没有执行测试、编译或 Provider 调用。
- 内容保留修订静态核对长期元数据/7 天内容的分离、存储与清理任务依赖及验收，替换“不记录任何正文”的旧规则；未创建实际内容记录，未执行清理或测试。
- 本轮静态核对正文仅服务端查看的决定与 Usage、内容存储设计及验收边界，移除查看范围问题；未执行测试或编译。
- 时区修订静态核对配置默认值、每日额度与统计日界线的一致性，移除时区待确认项；未执行时区或额度重置测试。
- 本轮静态核对统计查询范围、汇总/handler 分工、未知用量与主体权限的边界，新增 1.48 避免将聚合算法与接口接线合为大项；未执行统计查询测试。
- Key 管理修订静态核对重启生效、稳定主体关联、历史计数保留及无在线管理接口的范围；没有改动实际 Key 配置或执行认证测试。
- 本轮记录 OpenAI API 兼容优先的用户决定，保留兼容程度待确认项；仅静态核对受影响的 API/客户端设计条目，未查阅外部文档、运行 SDK 或声称已兼容。
- SDK 目标修订静态核对请求/响应/SSE 三项设计分工、实现依赖与验收；兼容程度已确认，具体标准契约待 Planner 依据文档完善。没有运行 SDK 或测试，没有发送网络请求。
- 跨平台修订静态核对统一进程管理与 Linux/Windows 适配的任务分工、依赖和验收，移除平台待确认问题；没有执行跨平台编译、Codex 启动或测试。

%% RISK: 共容器请求没有访问控制隔离，Codex 进程可能接触容器内其他请求目录、独立正文存储、网关配置或必要上游认证材料。独立 cwd 与目录清理不能证明其不可读取；用户接受共容器方案，不将此限制描述为已解决。

## Design Gates

10 项 Planner 设计已有 Engineering decisions 证据；业务待确认项关闭。启动崩溃窗口、文件系统耐久性和进程组限制记录为风险，不假造运行验证。

## Sources

- Issue: REQ00006-agent-proxy-shared-gateway
- Approved Plan: `docs/requirements/REQ00006-agent-proxy-shared-gateway/requirement-plan.md`，SHA256 `e4789111e9fa0bbaca5c63b07653aa1ef952c6e730ad28d503d3489d22228d17`。
- Approval: `docs/requirements/REQ00006-agent-proxy-shared-gateway/decision-log.md`、`requirement.toml`；revision 3，APPROVED by user。
- `docs/handoff.md` 及本会话三项用户范围决定，随后批准与 promote 请求。批准 Plan 中较早的“尚未批准”叙述作为历史阶段内容保留，当前批准以正式记录为准。
- 2026-10-03 本会话后续决定：“可以接受Proxy和Codex共用一个容器”；其前置说明明确每请求独立临时目录及清理、不要求请求间独立容器隔离。本 FD revision 4 记录当前隔离验收调整，原批准 Plan 不回写。
- 2026-10-03 用户决定：“你无须考虑容器部署和测试，我会另外安排。”revision 5 将对应交付与验证移出本 FD，并保留应用功能、静态检查和 compile-only 范围。
- 2026-10-03 grill-with-docs 访谈确认：“必须保存，我需要统计。”revision 6 记录每日额度与用量跨重启保存的决定，并新增独立持久化/恢复任务；否定内存清零方案。
- 2026-10-03 grill-with-docs 后续 `confirm`：接受 Codex 启动后的失败、超时、取消计入每日额度，并分别统计成功/失败/超时/取消；revision 7 记录该计数规则。
- 2026-10-03 用户确认：“同意建议。提示词和正文单独记录保存7天。”revision 8 记录元数据长期保留、提示词/响应正文单独保存 7 天的决定，替代早期不保存正文要求；保留原批准 Plan 的历史内容。
- 2026-10-03 grill-with-docs 后续 `confirm`：接受 V1 正文记录仅供服务端查看，不增加历史正文查询接口；revision 9 记录该查看边界。
- 2026-10-03 用户确认：“confirm. 默认为UTC”；revision 10 记录每日额度与每日统计采用同一可配置时区、默认 UTC、按零点划分自然日的决定。
- 2026-10-03 grill-with-docs 后续 `confirm`：接受按起止日期查询，并按天和模型汇总请求数、各终止状态数量与可取得的 Token，费用不可得时标记未知；revision 11 记录查询范围。
- 2026-10-03 grill-with-docs 后续 `confirm`：接受 V1 Key 由运营者维护配置，新增/轮换/禁用后重启生效，暂不增加在线管理接口；revision 12 记录 Key 管理边界。
- 2026-10-03 用户要求：“以兼容OPENAI API 为准。”revision 13 记录兼容优先原则，具体兼容程度待访谈确认。
- 2026-10-03 grill-with-docs 后续 `confirm`：接受官方 OpenAI SDK 只更换 base_url/API Key 即可调用 V1 明确支持的 Responses/Models 子集及流式输出，不支持参数明确报错；revision 14 记录兼容目标并拆分请求/响应/SSE 设计任务。
- 2026-10-03 用户 `confirm`：接受 Proxy 支持 Linux 与 Windows，必要的平台差异由实现分别处理，容器部署和测试仍另行安排；revision 15 记录平台边界，拆分 1.51–1.53。
- `openspec/specs/agent-proxy/spec.md`、`openspec/specs/agent-proxy-client/spec.md`。
- `plugins/aiw-agent-proxy/src/providers.ts`、`plugins/aiw-ai/aiw-ai.mjs`、`plugins/aiw-ai/README.md`。
- `docs/features/archive/FD-009/FD-009_REMOTE_AGENT_PROXY_URL.md`。
- Snapshot HEAD：`9ae91484d4835a50dad6d2aebbcad93c509583f9`。

Planner Session：`fd017-planner-20261003-9059c335`；已 claim 的来源事件：`FD-017-000002-design-requested`。后续移交以真实 CLI receipt 为准；Worker 来源事件为 FD-017-000016-design-ready。

### 当前 Worker 证据

Worker Session：fd017-worker-20261003-9059c335，来源事件 FD-017-000016-design-ready。
采用独立标准库模块和 Windows 原生 Job/进程启动包装，无依赖下载；旧状态不删除、不迁移，现有技能修改不属于本实现 diff。离线 compile-only 的实际结果和最终静态检查见双格式 Worker 报告。只查阅官方协议文档，未调用模型、启动 Codex/网关/SDK，未执行测试、lint、formatter、vet、最终产物构建或容器部署。编译不能证明进程清理、崩溃恢复、SDK 流式解析或宿主隔离已在运行中通过。

**Completed:** 2026-10-02
