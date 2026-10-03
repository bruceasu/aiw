# Issue Plan：Agent Proxy 共享 AI 网关更新

Issue：`REQ00006-agent-proxy-shared-gateway`。

状态：关键范围决策已确认；用户于 2026-10-03 回复 `confirm`，授权创建后续 Issue 并 capture 本 Plan。该确认不包含 Issue 批准或 FD 移交。

## 使用者、问题与目标

使用者是远程个人、应用或服务。运营者提供 Codex 支持的 AI 能力，调用者使用网关自己的凭证，无须自备 OpenAI 账号。调用者不得访问、修改宿主文件、项目、凭证或开发环境，也不得控制宿主命令执行。

用户本轮输入：`$issue-management update plugins\aiw-agent-proxy docs\handoff.md`。将根目录 `docs/handoff.md` 作为目标需求输入；不将本轮请求视为实现、部署、批准或覆盖历史需求的授权。

## 当前事实与来源冲突

- 已批准历史 Issue `REQ00005-ai-agent-proxy` 要求本机 TypeScript 代理，支持 Codex、Copilot、OpenAI，同步 HTTP 与 WebSocket ACK/一小时结果暂存；`client_id` 仅是自报路由标签。
- 当前稳定规格经后续 FD 扩展，允许显式配置远程 IPv4 HTTP；远程接口没有内置认证或 TLS，WebSocket 仍限本机。历史 Issue 的本机范围与当前规格的远程范围分别保留，不回写历史批准内容。
- 当前插件 README 与稳定规格承认 Codex 文件读取残余风险；`src/providers.ts` 中 Codex 使用 `STATE_DIR/codex-workspace`，子进程环境包含 `...process.env`。因此现有声明不能作为共享网关宿主隔离验收的证据。
- `docs/handoff.md` §2 要求 Go 网关，§6–8 要求逻辑模型与 `/v1/responses`，区别于现有 TypeScript、多 Provider、`/v1/requests` 契约。
- `docs/handoff.md` §1、4、31–32 要求宿主文件不可访问，但 §14、29、32 先采用本机进程和空临时目录、后加入 Docker。临时工作目录不能单独证明宿主不可访问；该冲突影响首阶段共享访问是否允许，必须明确决定。

## 目标范围

依据 handoff 记录以下目标；阶段划分仍受下方决策约束。

1. 使用 Go 构建小型独立网关，Codex CLI 为初始 AI 后端，HTTP 与 Runner 解耦。
2. 对外使用稳定 AI 接口与逻辑模型名称。V1 目标接口为 `POST /v1/responses`、`GET /v1/models`、`GET /v1/usage`；首个纵向里程碑聚焦 Responses、Models、认证、执行、超时及清理。
3. 每个个人/服务拥有独立网关 API Key；只存储 hash/HMAC，`client_id` 不替代身份认证。使用 principal 表达调用主体。
4. 共享运行环境仅含该请求明确提供的数据，不继承宿主项目或无关本地凭证。不挂载宿主 home、SSH/AWS/Git 凭证、项目目录、Docker socket 或宿主根目录。
5. 每个请求创建独立临时工作区；完成、超时、客户端断连、进程失败或网关取消时终止执行并清理。本机进程仅用于可信开发，不得用于共享访问；任何共享接入前必须完成 Docker 或等效结构性隔离并限制资源，不能仅以“尚非生产”为由绕过隔离。
6. 后续阶段增加 SSE、每主体 RPM/并发/每日请求限额、全局并发及可取得的用量统计。默认不记录完整提示词或结果；不编造未知 Token 或成本。
7. 附件在后续阶段仅允许显式上传，写入请求隔离区并清理；会话仅在明确需求后考虑。
8. 用户允许移除旧 `POST /v1/requests` 与 WebSocket `/v1/events`。`plugins/aiw-agent-proxy-client` 中 `ai` 客户端迁移属于本 Issue 交付范围：改用网关凭证、逻辑模型与新 Responses 接口；不得继续依赖自报 `client_id` 作为身份或旧 ACK 协议。不要求保留原始 Provider 选择及旧接口形状；客户端具体参数与结果展示在 FD 中设计。

## 非目标与约束

- 不开放 `/files`、`/shell`、`/workspaces`、`/git` 或宿主管理能力；请求不得指定宿主路径、挂载、CLI 参数、沙箱禁用或 hostAccess。
- 不引入 UI、OAuth/SSO、复杂计费、Redis、队列、Kubernetes、多区域、仓库检出或多 Agent 编排。
- 共享网关与可访问授权项目的开发 Worker 保持不同安全配置。
- 保留 `REQ00005-ai-agent-proxy` 的批准来源、决策、Session 与归档 FD。用户已确认在 `plugins/aiw-agent-proxy` 内迁移并替换现有插件，并允许移除旧接口；`ai` 客户端迁移须随本 Issue 一并交付。稳定规格在未来实现改变契约时更新，本轮不改。
- 本轮仅进行 Issue 整理；没有代码、依赖、公共 API、持久化、认证或部署实现变更。

## 验收示例

- 给定有效主体 Key 与已配置逻辑模型，提交文本请求后返回稳定响应 ID、状态与输出；底层 Codex 参数不暴露给调用者。
- 缺失、无效或禁用 Key 的请求在 Runner 启动前被拒绝；自报其他人的 `client_id` 不能获得其权限或用量。
- 请求含宿主路径、挂载或沙箱控制字段时，不能改变服务端隔离策略；具体未知字段处理留给 FD，但不能透传这些控制。
- 共享运行中的请求无法读取或修改宿主项目、home、SSH/AWS/Git 凭证，且不能继承其他请求的工作区数据。空临时目录本身不算达成此验收。
- 正常结束、超时、断连或 Runner 失败后不遗留请求进程/容器与工作区；清理失败的告警和处置在 FD 中设计。
- 达到已配置限额时拒绝额外调用；用量查询只返回调用主体获准访问的元数据，未知 Token/成本明确为未知。
- SSE 阶段返回增量输出及终止事件，断连取消执行；不向客户端泄露宿主路径、凭证、环境或内部堆栈。
- 迁移后的 `ai` 客户端通过网关 Key 和逻辑模型调用 `/v1/responses`，可取得一次文本结果或稳定错误；已移除旧接口的部署不再要求原版客户端可用。交付说明明确接口替换与客户端升级要求。

以上为待实现的验收要求，不是本轮已测结果。

## 待决事项与 Gates

已确认（本轮用户回复）：在 `plugins/aiw-agent-proxy` 内迁移并替换现有 TypeScript 插件。

已确认（本轮用户回复）：本机进程仅用于可信开发；共享访问前必须完成结构性隔离。因此开发里程碑只证明执行与清理链路，宿主不可访问的验收属于共享开放前的硬 Gate。

已确认（本轮用户回复）：允许移除旧接口；`ai` 客户端迁移作为本 Issue 的交付范围。

当前没有阻止记录本 Plan 的未决业务范围问题。共享隔离仍是未来开放接入前的硬 Gate；这不是已验证的能力。Issue 批准与 FD 移交尚未发生，必须分别按授权进行。

工程策略留待 FD：凭证配置与轮换、hash/HMAC 选择、隔离执行时上游认证的最小供给方式、网络限制、错误/SSE schema、输出截断、清理失败处置、限流计数时区与重启语义、Usage 的保留周期。handoff 未规定具体值，不在 Issue 中臆造默认值。

## 来源与修订证据

- 用户本轮请求及根目录 `docs/handoff.md`（当前未跟踪文件）；SHA256 `81555b548010dc657ced1723f2e743f5c6a082b1b283b5dc1cf5d7072d925105`。
- `docs/requirements/REQ00005-ai-agent-proxy/requirement-plan.md`、`decision-log.md`、`requirement.toml`：历史批准需求，revision 3，2026-10-01 用户批准。
- `openspec/specs/agent-proxy/spec.md`：当前稳定契约与隔离残余风险声明。
- `plugins/aiw-agent-proxy/README.md`：运行方式、远程无认证、Provider 限制。
- `plugins/aiw-agent-proxy/src/providers.ts`：工作区与环境继承的静态代码证据。
- 工作区 HEAD：`9ae91484d4835a50dad6d2aebbcad93c509583f9`。发现时 `git status --short` 仅列出 `?? docs/handoff.md`。

## 建议下一步与写入范围

三个范围决策已确认，用户已确认正式记录操作。`aiw issue new agent-proxy-shared-gateway "Agent Proxy 迁移为 Go 共享 AI 网关"` 已分配 `REQ00006-agent-proxy-shared-gateway`；通过 `aiw issue capture REQ00006-agent-proxy-shared-gateway requirement-plan --file .ai/requirements/drafts/agent-proxy-shared-gateway-update.md` 录入本 Plan。保留历史已批准 `REQ00005-ai-agent-proxy`，本次并非独立成果拆分，不建立 split parent 关系。

正式记录写入范围仅为新 Issue 及其 Plan。下一步由人决定是否批准 Issue；批准后再决定编号 FD 移交。没有创建 FD 或 OpenSpec change。

## 本轮证据检查

已执行文件静态读取、定向 `rg`、`Get-Command aiw`、`aiw issue --help`、`git status --short`、`git rev-parse HEAD` 和 handoff 的 `Get-FileHash`。首轮插件内 handoff 路径不存在，后续读取了用户指定的根目录路径；`internal/cli` 查询路径不存在，未据此作 CLI 实现推断。

未运行测试、编译、构建、格式化、lint、网络请求、Provider 调用或 Git 写操作。Issue 草稿没有代码变更，不适用实现后的编译检查。

2026-10-03 正式记录前重新读取草稿、执行 `aiw issue --help`、`aiw issue list`，并通过 `Get-FileHash docs/handoff.md -Algorithm SHA256` 确认来源内容未变化。随后执行上述 `aiw issue new` 创建新 Issue；capture 的实际结果以 CLI 返回和生成记录静态检查为准。
