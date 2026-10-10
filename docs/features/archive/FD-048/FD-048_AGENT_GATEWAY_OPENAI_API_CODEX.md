# FD-048: Agent Gateway 统一 OpenAI API 出口与 Codex 兼容适配

**Status:** Complete
**Revision:** 7
**Priority:** Medium
**Evidence policy:** Dual

## Problem

`aiw say` 当前调用 `/v1/chat/completions`，FD-045 的 Agent Gateway 只提供 Codex Responses 兼容子集，二者不能直接接入。调用方还需要真实 OpenAI／兼容服务的统一出口，集中管理 Key 和转发流量，并为后续统计计费保留扩展位置。

用户于 2026-10-10 确认：代理范围包含全部上游 API、文件/资源操作和 Realtime WebSocket；真实上游模式负责流量转发、Key 使用及统计计费。真实上游代理和 Codex CLI 模拟是两种模式，基本不会同时开启。本 FD 按每实例互斥模式设计，不延续已完成的 FD-045 实施周期。

同日用户进一步明确：代理模式的统计与计费目前只预留，不要求完整实现；缺失统计/计费功能不影响转发，也不阻塞本 FD 交付。以下范围以此补充决定为准。

## Options and decision

1. 只让 Say 改用 Responses：解决当前翻译接入，但没有统一出口能力。
2. 每个实例按模型混合路由真实上游和 Codex：增加无 model 资源请求的路由及资源归属复杂度，与用户确认的模式边界不符。
3. 每个实例选择互斥 `openai_proxy` 或 `codex` 后端模式：采用此方案。共用入口认证；传输和请求校验按模式分开，代理统计/计费仅预留，Codex 既有额度保持兼容。需要同时使用两种模式时运行不同实例，失败不自动切换后端。

用户提出使用 Go OpenAI SDK。代理模式采用 Go 标准库 `net/http/httputil.ReverseProxy` 透明转发，不重新编码完整 API，也不新增 SDK 依赖；Codex 模式继续使用已有 App Server/Responses 实现。

## Solution

### 模式与兼容性

- 新增后端模式字段；不复用现有用于部署策略的 `mode` 字段。旧配置默认 `codex`，保留现有 Codex 配置、额度和 Responses 兼容行为。
- `codex` 模式补充 `/v1/chat/completions` 的明确子集，使 Say 无须更换接口。共享现有 App Server 执行、取消、额度及 usage；不把 Codex 描述为全部 OpenAI API 实现。
- `openai_proxy` 模式不要求 Codex 程序、模型映射或工作目录。使用固定上游 base URL 和服务端凭据，转发其支持的全部 API；不应用 Codex 严格字段解码、文本大小上限或工具禁用规则。

### 真实上游流量代理

- Gateway 将 `/v1` 下的全部 HTTP 方法、路径和查询转发到固定配置的 OpenAI-compatible API base URL；`/internal/*` 保留为 Gateway 控制面。`/v1/models`、`/v1/usage` 等在代理模式中也转发上游。
- 使用 Go 标准库反向代理保留未知接口、字段、请求/响应 body、状态码、响应头和错误语义；不解析或重建 JSON，不缓存整个请求或响应。请求 body 流式传输并由 HTTP 背压自然限速；断开客户端时取消上游请求。SSE 事件原样转发并及时 flush。
- 通过同一 Go HTTP 反向代理处理 WebSocket upgrade 和双向数据；上游目标始终由服务端固定配置。入站 gateway key 只用于认证，出站认证由服务端环境变量解析的上游 key 替换。剥除调用方的 `Authorization`、`api-key`、组织/项目身份及逐跳 header，不把 gateway key 或调用方身份转发给上游。
- `proxy_base_url` 是固定的 HTTP(S) API base URL，允许配置 API 路径前缀，不允许 userinfo、query 或 fragment。Gateway 只把本地 `/v1` 路径尾部拼接到该 base URL；客户端不能提交绝对 URL、任意目标或可改变 host 的重定向。反向代理不自动跟随上游 3xx，不重试请求。
- 上游 key 通过进程环境变量名配置；每个 principal 可引用专属变量，未配置时回退到实例默认变量。启动时解析并验证所有启用 principal 的有效凭据；凭据值不进入配置、响应、日志或持久化。共用上游 key 的 principal 共享其上游权限；模型与资源授权由该凭据和上游决定。
- Gateway 入口 key、principal、RPM、全局/主体并发限制继续生效；代理请求不做 Codex 模型白名单映射、日额度扣减或 token usage 提取。Codex 模式的旧额度和恢复规则保持不变。
- 代理模式不依赖 Codex 可执行文件、工作目录、额度存储或统计存储。请求 ID 和 principal 可用于不含正文/凭据的日志；上游 usage/费用字段原样透传。本期不新增费用、统计、预算或账本机制，可选记录失败不得阻断转发。
- WebSocket 占用受全局及 principal 并发上限约束；客户端断开、进程取消及服务关停会关闭活动升级连接并在有限关停预算内收敛。HTTP/SSE/WebSocket 请求均不自动重试，避免重复创建资源或产生重复费用。

## Scope

**优先交付决定（2026-10-10）：** 用户要求先实现 `/v1/chat/completions`。Say 所需 Codex 文本子集拆为 `FD-049_AGENT_GATEWAY_CHAT_COMPLETIONS.md` 独立交付；本 FD 保留完整代理模式设计，待 FD-049 交付后再继续，不要求先实现模式配置才能提供该接口。

范围：两种互斥后端模式、配置的 OpenAI-compatible API base URL 下全部 `/v1` HTTP 与 Realtime WebSocket 流量、入口/上游 key 分离、有限资源收敛、统计/计费扩展位置，以及复用 FD-049 的 Codex Chat Completions 接口。完整统计/计费不在本期范围；未完成全部工作项前不宣称全 API 支持。

实现约定：新增独立 `backend_mode`，缺省 `codex` 以兼容旧配置；既有 `mode` 继续只表示 `development`/`shared` 部署策略。代理模式使用配置的 API base URL 与环境变量凭据，不引入 SDK/第三方 WebSocket 依赖；无自动 fallback。代理入口保留 gateway key、RPM 与并发限制，跳过 Codex 模型映射、Codex 日额度和 usage 存储。网络集成、依赖下载、运行时请求和测试不在默认验证授权内。

## Work items

- [x] 1.1 实现独立 backend mode 和兼容旧配置的条件校验。验收：旧配置缺省为 Codex；部署 `mode` 语义不变；proxy 模式不要求 Codex 路径/模型；模式专属冲突配置拒绝。大小：小；难度：中；依赖：无。
- [x] 1.2 实现固定上游 URL 与环境变量凭据解析。验收：只允许 HTTP(S) base URL；拒绝 userinfo/query/fragment；principal 可指定凭据变量或继承实例默认；启用主体凭据缺失时启动失败；凭据值不落盘。大小：小；难度：中；依赖：1.1。
- [x] 1.3 实现按 backend mode 分流与本地/上游路由边界。验收：Codex 既有路由原样工作；proxy 的 `/v1/*`（含 models/usage）交上游；`/internal/*` 保留本地；失败不自动切换后端。大小：小；难度：中；依赖：1.1–1.2。
- [x] 1.4 实现透明 HTTP 反向代理及固定路径拼接。验收：保留 method、查询、body、状态码、响应头和未知 API；客户端不能控制目标 host；上游 3xx 原样返回且不跟随；无重试。大小：中；难度：中；依赖：1.2–1.3。
- [x] 1.5 实现凭据与身份 header 替换。验收：认证 gateway key 后关联 principal；出站替换 Authorization/api-key；移除调用方组织/项目身份与逐跳 header；上游 key 不回传、不写日志；共享凭据权限限制有文档说明。大小：小；难度：中；依赖：1.2–1.4。
- [x] 1.6 实现请求/响应 body 流式与 multipart/二进制传输。验收：不缓存完整 body；背压和客户端取消传递至上游；不记录文件或消息正文。大小：小；难度：中；依赖：1.4–1.5。
- [x] 1.7 实现 SSE 原样转发。验收：事件字节/顺序不重建，响应及时 flush；客户端断开取消上游；不解析 usage。大小：小；难度：中；依赖：1.4、1.6。
- [x] 1.8 实现 WebSocket upgrade 与双向转发。验收：走固定上游；不泄露 caller key/身份；双向传输有背压，连接错误不重试。大小：中；难度：中；依赖：1.3、1.5。
- [x] 1.9 实现代理并发与运行关闭收敛。验收：沿用全局/principal 并发及 RPM 上限；关闭时取消 HTTP/SSE、关闭活动 WebSocket 并在有限预算内返回；Codex App Server 生命周期保持原样。大小：中；难度：中；依赖：1.6–1.8。
- [x] 1.10 解耦代理模式与 Codex 存储/额度。验收：代理可在无 Codex 程序、工作目录、额度数据或统计存储时启动/转发；不扣 Codex 日额度、不解析 usage；可选审计写入失败不阻断响应。大小：小；难度：中；依赖：1.1、1.3、1.9。
- [x] 1.11 复用 FD-049 的 Codex Chat Completions 行为。验收：FD-049 实现与审查证据仍适用于 Codex mode；proxy mode 则透明代理相同路径；不重复实现。大小：小；难度：低；依赖：FD-049、1.3。
- [x] 1.12 更新稳定 spec、配置样例、Gateway/Say 说明与 Dual 实施报告。验收：模式、路径、凭据/权限、支持范围、统计预留和未验证限制一致。大小：小；难度：低；依赖：1.1–1.11。

## Acceptance

1. 每个实例只运行一个 `backend_mode`；旧配置缺省继续运行 Codex，不自动 fallback 到真实上游。
2. Say 通过 Gateway Key 和允许的 Codex 模型，以现有 Chat Completions 请求形状获得翻译响应。
3. 代理模式下全部 `/v1/*` HTTP/SSE/multipart/二进制及 Realtime WebSocket 请求均到固定上游，保持上游语义；本地控制路由不代理。
4. 目标、凭据及上游身份由服务端配置；gateway key、上游 key、调用主体 header 与正文不泄露；共享上游凭据权限边界明确。
5. 代理模式不依赖 Codex 组件或额度/统计存储，可选审计失败不阻断转发；不解析 usage/费用、不改变上游响应。Codex 模式既有额度/usage 行为保持兼容。
6. 全局/principal 并发和 RPM 有界；大文件、SSE、WebSocket、断线及 shutdown 有限收敛，不自动重试。

## Verification

- 设计依据：`program/agent-gateway/config.go` 将部署 mode 与 Codex 模型/文件/额度配置耦合；`server.go` 统一认证、RPM/并发及管理路由；`main.go` 的 HTTP shutdown 不处理升级 WebSocket；`observation.go` 只持久化请求元数据；当前独立 Go 模块无 SDK/第三方依赖。
- 兼容证据：旧配置新增字段缺省为 `codex`；FD-049 已完成并通过独立静态审查，Codex chat 接口不得重复实现。
- Worker 执行 `python -B scripts/compile.py`（工作目录 `program/agent-gateway`），编译检查通过；静态检查 `git diff --check` 通过。未运行测试、完整构建、真实上游/Realtime 请求或下载依赖。实现证据见 `docs/features/reports/FD-048-implementation-20261010T103000+0900.md` 及同名 JSON。
- 静态调用链检查确认：认证后按 backend mode 分流；ReverseProxy 固定目标并替换凭据；HTTP/SSE 请求使用标准流式传输，WebSocket Hijack 连接纳入关闭清单；代理启动不打开 Store/App Server。编译与静态检查不证明所有上游 API、SSE、WebSocket 和部署代理环境的运行兼容性；本期不声称运行时全 API 已验证。
- 独立 Reviewer 对实现提交 `1f9aa98`、`9ef62e3`、`72a63d2` 及 Say 文档提交 `4f77697` 静态审查通过；报告见 `docs/features/reports/FD-048-review-r1.md` 及同名 JSON。Reviewer 未运行测试、编译或运行时请求；完整 API、SSE、Realtime WebSocket 和 shutdown 行为仍未运行时验证。

## TODO

- [x] 记录全部 API 代理与 Codex 模拟两种互斥模式的用户决定。
- [x] 记录统计/计费仅预留的用户决定；移除持久化/费用/预算决策 Gate，取消原 1.11–1.12。
- [x] 固定 SDK 定位（透明代理不新增 SDK）、模式兼容、认证/凭据和传输/shutdown 契约；按半天内小项拆分。
- [x] 记录用户 2026-10-10 对取消旧 Planner 交接并重派的指示；旧 Agent 未被停止，风险见 blocker 记录。
- [x] 实现并按 Work Items 提交，更新 TODO/Verification 与 Dual 证据。
- [x] 独立 Reviewer 通过后按 Auto 流程交付并归档。

## Sources

- Issue: none

- 用户在当前会话对全部 API 流量代理、Key 使用/统计计费及互斥模式的确认。
- `internal/say/openai.go`、`program/aiw-say/README.md`
- `internal/ai/openai.go`、根 `go.mod`
- `program/agent-gateway/config.go`、`server.go`、`README.md`、`go.mod`
- `openspec/specs/agent-proxy/spec.md`
- `docs/features/archive/FD-045/FD-045_AGENT_GATEWAY_CODEX_APP_SERVER.md`
- `docs/features/archive/FD-049/FD-049_AGENT_GATEWAY_CHAT_COMPLETIONS.md`：已交付的 Codex Chat Completions 子集与审查证据

**Completed:** 2026-10-10
