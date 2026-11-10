# Issue Plan：aiw ai Provider 选择与文件输入

Issue ID：`ISSUE-00013`

状态：已捕获，等待 Issue 批准。捕获不授权实现。

## 用户问题与目标

用户希望 `aiw ai` 可选择通过 Go Agent Gateway 或 AIW 共享配置中的 Provider 提问；默认使用 AIW Provider。用户也希望能从文件读取请求输入，避免终端或标准输入传递文本时发生乱码。

目标：让调用者可选择 AI 后端，并通过指定编码的文件提供请求文本。

## 已确认事实与来源

- 当前 `aiw ai` 是调用 Go Agent Gateway Responses 子集的 Node 客户端，不直接调用 Provider。来源：`src/plugins/aiw-ai/README.md`、`openspec/specs/agent-gateway-client/spec.md`。
- AIW 已有共享 `[ai]` Provider 配置及解析/调用接口。来源：`src/internal/ai/config.go`、`src/internal/ai/provider.go`、`openspec/specs/ai-routing/spec.md`。
- 当前命令接受位置提示词、用 `-` 读取 stdin，并支持将本机 UTF-8 文件用于 `--system @FILE`；没有用于请求正文的文件输入选项。来源：`src/plugins/aiw-ai/README.md`、`src/plugins/aiw-ai/aiw-ai.mjs`。
- 用户确认默认路由为 AIW 共享配置的 Provider；可通过参数选择 Go Agent Gateway。
- 用户确认文件编码默认为 UTF-8，并可通过参数指定文件编码。
- 用户确认文件内容与位置提示词组合。

## 范围

- 支持通过命令参数选择 Go Agent Gateway；未指定时使用 AIW 共享配置的 Provider。
- 支持从文件读取请求输入，默认以 UTF-8 解码，并允许调用者指定文件编码；文件内容与位置提示词组合。
- 明确各路由的配置、失败行为和输出方式；显式选择的路由不可静默回退到另一条路由。

## 非目标与约束

- 选定路由不可用或配置错误时，不得静默改走另一条路由。
- 不改变无关 AIW 命令或共享 Provider 配置约定。
- Issue 捕获不授权实现、测试或 OpenSpec 变更。

## 验收示例

- 不提供路由参数时，请求使用 AIW 共享配置的 Provider；提供 Gateway 路由参数时，请求使用 Go Agent Gateway。
- 所选路由不可用或配置错误时，命令报告该路由的错误，不向另一条路由发送请求。
- 调用者可将文件作为请求输入；默认按 UTF-8 解码，也可指定编码。
- 同时提供文件和位置提示词时，请求包含两者内容。

## 假设与待决事项

- 每次调用只使用一条路由；命令仍为单次请求。
- 文件编码名称参数及运行时支持的编码集合由工程设计确定；不支持的编码必须清晰报错，不得静默按其他编码解码。

## 规模与拆分评估

这是一个中等规模的 CLI 改造：路由选择与文件输入都作用于同一次 `aiw ai` 请求，需要保持参数解析、请求行为、文档和错误处理一致。作为一个 Issue 保留，后续 FD 可拆分 Work Items。

## 建议下一步

Issue 业务范围和验收规则已清楚，可由用户决定是否批准 `ISSUE-00013`。批准后可晋级为 FD 进行工程设计。不得将捕获视为批准或实现授权。
