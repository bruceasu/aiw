# FD-115：aiw ai Provider 选择与文件输入

**Status:** Complete
**Revision:** 8
**Priority:** Medium
**Review policy:** Independent
**Evidence policy:** Markdown

## 问题

`aiw ai` 目前只调用 Go Agent Gateway，要求启动网关并提供 Gateway Key。调用者无法通过同一命令使用 AIW `[ai]` 配置中的 Provider。提示词目前可从命令行或 UTF-8 stdin 传入，无法给请求正文指定文件编码；终端/stdin 转码可能让输入乱码。

## 方案

保留 Node 插件作为 `aiw ai` 的 CLI、文件解码及 Gateway HTTP 客户端。增加 `--backend aiw|gateway`，默认 `aiw`。为了兼容已有显式 `--url` 用法，未给 `--backend` 但提供 `--url` 时选择 Gateway；显式 `--backend aiw` 与 `--url` 同时出现时报参数错误。Gateway Key 和模型配置只在 Gateway 路由读取。所选路由失败时直接返回错误，不切换路由。

AIW 路由通过 AIW 安装目录中的 Go `aiw` 可执行文件调用一个隐藏内部子命令。Node 将有限的 JSON 请求写入子进程 stdin；Go 子命令使用 `internal/ai` 的共享配置和 Provider 实现，返回 JSON 响应。请求提示词不放在进程参数中，不重写或复制各 Provider 驱动。内部入口限定单次生成、严格输入字段、有限请求大小和超时；复用 AIW Provider 的 `ReadOnly` 调用路径，不写 Session。Node 正常启动插件时使用宿主提供的 `AIW_ROOT` 定位可执行文件；独立运行 npm `ai` 时从 PATH 查找 `aiw`，找不到则明确失败。

Node 插件增加单个 `--file PATH`，默认用严格 UTF-8 解码，支持 `--encoding LABEL` 指定 Node `TextDecoder` 接受的编码标签。文件最多读取 256 KiB，解码后与位置提示词合计最多 64 Ki 个字符。组合顺序为：位置提示词、空行、文件内容；没有位置提示词时直接使用文件内容。文件只在本机读取，Provider 只收到组合后的文本，不收到文件路径。`--encoding` 只用于 `--file`；`--system @FILE` 保持既有 UTF-8 规则。文件输入不能与 stdin 的 `-` 同时使用。

## 备选方案与取舍

- 在 Node 插件里重新实现 AIW 配置和所有 Provider：会复制配置优先级、凭据环境变量、CLI 调用和不同 HTTP API 行为；不采用。
- 将整个 `aiw ai` 迁移到 Go：可以原生复用 Provider，但需重写现有 Gateway HTTP 客户端、交互编辑、文件解码和输出处理，扩大兼容面；不采用。
- 由 Node 解析命令并通过受限 stdin/stdout 协议调用 Go 内部 Provider 命令：保留现有 Gateway 实现，并复用唯一的 Provider 层；只增加进程边界协议，最符合本次范围，采用。

该桥接是两个实际后端所需的适配器。调用失败时子命令返回非零并附诊断，Node 不重试另一路由；成功响应只返回文本和用于诊断的 Provider/model 元数据。`--json` 在 Provider 路由中要求 JSON object 并在 Node 侧校验；不合法响应报错，不输出伪成功 JSON。Provider 的具体 token 用量可能不可得，verbose 对缺失值如实显示 unknown/unavailable。Gateway 原有 Responses 校验和 verbose 字段保持。

## 范围

- `aiw ai` 的路由参数、默认值、路由特有参数校验以及内部 Go Provider 调用桥接。
- 请求文件的显式编码、UTF-8 默认值、文件/位置提示词组合、上限与失败行为。
- 同步插件帮助、README、manifest 描述、环境变量用法及 Gateway client / AI routing 稳定规格。
- 保留原 Gateway 请求语义、AIW Provider 配置优先级及单次请求边界。

## 非目标

- 不新增 Provider、修改其配置格式或依赖。
- 不修改 `aiw ask`、Session 持久化、Gateway 服务端或 Provider 的工具能力。
- 不增加自动重试、后台会话、流式响应或多文件输入。
- 不运行测试；本 FD 的允许验证为静态检查、Node 语法检查及仓库 compile-only 脚本。

## Work items

- [x] 1.1 增加 `aiw|gateway` 路由选择和有界 Go Provider 内部桥接；确保默认 AIW、明确错误、无跨路由回退，并保持单次输出、JSON 校验和超时行为。
- [x] 1.2 增加 UTF-8 默认/显式编码文件输入及与位置提示词组合；更新插件帮助、文档和相关稳定规格。

## 验收

- 无 `--backend`/`--url` 时，通过 AIW `[ai]` Provider 调用；`--backend gateway` 通过 Gateway 调用；仅有 `--url` 时保留显式 Gateway 选择。
- `--backend aiw --url URL`、未知/重复路由参数、缺失/重复文件或编码参数、编码标签不支持、文件不可读/超限、组合提示词超限，均在请求前报告参数或输入错误。
- 所选路由配置不完整或调用失败时返回该路由错误，绝不请求另一条路由。
- Gateway 所需的 `AIW_AI_API_KEY`、`AIW_AI_MODEL`、`AIW_AGENT_PROXY_PORT`、`--url` 只在 Gateway 路由生效；AIW 路由使用现有 `[ai]`/`AIW_LLM_*` 配置。
- `--file` 默认严格 UTF-8 解码；`--encoding` 改变文件解码编码。位置提示词在文件内容前，两者以一个空行分隔；文件路径不会发送给 Provider。
- `--encoding` 无 `--file` 时失败；`--file` 与 stdin `-` 不能同时出现；未指定文件时既有位置提示词、stdin 和交互输入行为保留。
- `--system` 继续传为系统指令；`--timeout-seconds` 对两路由施加一次调用的客户端期限。
- `--json` 两路由均只输出一个 JSON object；无效 JSON/object 报错；`--verbose` 仅在 stderr 输出可得的实际/配置元数据，不编造缺失用量。

## Verification

- 静态复核了参数解析、文件读取/解码、Gateway 请求、Go 内部桥接、共享 Provider 调用、只读选项及文档/稳定规格。`git diff --check` 通过。
- 修复两项审查发现后，`python scripts/compile.py` 和 `node --check src/plugins/aiw-ai/aiw-ai.mjs` 均退出码 0；编译脚本仅编译 `./cmd/aiw` 和 `./cmd/aiw-req`，未保留二进制产物。
- `git diff --check` 通过。
- 未运行测试、最终构建、真实 Provider/Gateway 请求、formatter、linter、vet 或验证脚本。
- Independent Reviewer 首轮提出的两项规格边界问题已修复。第二轮复审确认 4 MiB UTF-8 文本上限、25 MiB 序列化 envelope 总字节上限及 Node/Go Unicode 码点计数一致；未发现其他回归。详见 `docs/features/FD-115/reviews/FD-115-review-8f674a8.md`。本 Reviewer 未运行测试、构建、compile、Node check 或真实请求。

## TODO

- 两个 Work Item 和 Worker 自检完成；第二轮独立复审已通过。待 host PM 继续完成本地交付和归档。

## 来源

- 批准 Issue：`ISSUE-00013`。
- 稳定规格：`openspec/specs/agent-gateway-client/spec.md`、`openspec/specs/ai-routing/spec.md`。
- 用法文档：`docs/usage/environment-variables.md`。
- 实现依据：`src/plugins/aiw-ai/aiw-ai.mjs`、`src/internal/ai/provider.go`、`src/internal/ai/llm.go`、`src/cmd/aiw/main.go`。

**Completed:** 2026-11-10
