# Issue Plan: 本机 TypeScript Agent Proxy

## Actor and problem

Agent 客户端需要调用 Codex CLI、Copilot CLI 或 OpenAI API。当前可参考的 TypeScript 代码位于 `.ai/tmp/ts-agent/`，提供 Provider 调用逻辑，但没有统一的本机 HTTP/WebSocket 接口，也没有异步结果确认与暂存能力。

## Goal

提供一个仅供本机 Agent 客户端使用的 TypeScript Agent Proxy，统一发起单次 LLM 请求，支持同步和异步调用，并在不保存完整请求/结果审计日志的前提下记录可取得的审计与成本信息。

## Scope

- 支持 Codex CLI、Copilot CLI、OpenAI API。
- 客户端请求可指定 Provider、模型、效能参数和输出类型 JSON 或 Markdown。
- 提供同步 HTTP 单次请求。
- 提供 WebSocket 长连接上的异步单次请求，以结果 Event 返回并由客户端 ACK。
- 发送失败的结果写入临时文件；客户端重连后重发；ACK 后立即删除；一小时未 ACK 则过期删除。
- 服务仅监听本机地址。客户端自报 ID 用于关联请求和结果，不作为身份认证。
- 记录审计所需的元数据及可获得的调用用量和费用；不把完整请求正文或完整结果写入审计日志。

## Rules and exceptions

- Codex CLI 和 Copilot CLI 调用不得执行 Shell、修改文件或读取敏感文件。
- 若 CLI Provider 无法可靠强制上述限制，应拒绝该调用（fail closed）。
- 失败结果临时文件属于结果交付队列，不属于审计日志；ACK 或超过一小时后必须清除。
- Token、费用等 Provider 用量数据按实际可得情况记录，不要求不可用的 Provider 强行提供。
- 本机绑定限制网络来源；自报 ID 仍不能证明调用方进程身份。

## Acceptance

1. 三种 Provider 均可通过统一请求接口调用，并可按请求选择模型、效能参数及 JSON/Markdown 输出类型。
2. HTTP 请求同步返回一次结果。
3. WebSocket 请求以带关联信息的结果 Event 返回；客户端 ACK 后结果立即删除。
4. 发送失败时结果写入临时文件；客户端重连后可重发；一小时未 ACK 的结果过期删除。
5. 服务仅监听本机地址，并使用客户端自报 ID 关联请求与结果。
6. Codex CLI 与 Copilot CLI 调用无法执行 Shell、修改文件或读取敏感文件；不能保证这些限制时调用失败关闭。
7. 审计日志不含完整请求正文或完整结果，并尽可能包含调用时间、Provider、模型、状态、Token 和费用。

## Sources

- User request: “参考 .ai\tmp\ts-agent 代码，使用一个 Agent Proxy” and the listed providers, output types, HTTP/WebSocket behavior, one-hour result retention, and request/result logging.
- User decision: “身份认证：暂时由客户端自己声称自己的ID”。
- User decision: “结果何时算‘成功接收’：客户端ACK 发送的结果Event”。
- User decision: “日志是否保存完整请求和结果：只保留审计必须的内容，以及使用的成本，例如：时间，token, 费用等，能收集多少算多少”。
- User rule: “当使用codex cli/copilot cli时，不允许执行shell, 修改文件，不能读取有敏感信息的文件。”
- User decision: “发送失败的结果，保存到临时文件。”
- User decision: “由客户端自己指定”（效能参数）。
- User decision: “访问范围：仅限本机”。

## Deferred engineering decisions

- HTTP/WebSocket request and response schemas, including event IDs and duplicate handling if an ACK is lost.
- Exact mapping and validation of provider-specific model/effort/output parameters.
- How CLI tool use and sensitive-file access are technically disabled and verified for each SDK version.
- Temporary spool location, permissions, restart recovery, and cleanup mechanism.
- Provider-specific sources for Token and cost data.
- Service packaging, startup, shutdown, and local port configuration.

These are engineering design items for the FD. No OpenSpec change was requested.
