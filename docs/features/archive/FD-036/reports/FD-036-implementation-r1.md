# FD-036 Worker 实施报告

<!-- aiw-data: FD-036-implementation-r1.json -->

## 执行信息

- FD：`FD-036`，Revision 6
- Worker 会话：`fd036-worker-20261008-8f3c2a`
- 来源事件：`FD-036-000005-work-requested`
- 工作分支：`feature/FD-036`，工作树：`.wt/FD-036`

## 实施结果

新增 `cmd/aiw-say` 插件和 `internal/say` 核心包，实现文本参数/stdin 输入、安装目录 `aiw.toml`、`--config`、用户 profile、配置覆盖、参数校验、翻译提示词和 OpenAI Chat Completions 适配器。请求使用独立 system/user 消息，成功后才输出完整译文；失败诊断不包含原文、响应正文或密钥。API 凭据只读取 `OPENAI_API_KEY`；model 必须由配置或参数提供。

增加确定性 Python 黑盒用例和安装配置/profile 样例、Phase 1 README。Work Item `1.8` 原来混合 Worker 测试代码与独立 Tester 场景清单，已取消并拆为 `1.8.1`（测试代码，完成）和 `1.8.2`（独立 Tester 场景报告，待 Pending Test 阶段）。

OpenAI 请求形状按官方 Chat Completions API 参考实现，未在源码选择默认模型：[官方 API 参考](https://developers.openai.com/api/reference/resources/chat)。

## 命令与验证

- `aiw fd claim FD-036 FD-036-000005-work-requested --session fd036-worker-20261008-8f3c2a`：成功领取 Worker 事件。
- Worker 按 Work Item 创建本地提交：`c40a6024`, `50882019`, `e2f1e673`, `276faa99`, `ed3309bb`, `cec998b4`, `147584c3`, `a2627b62`, `3602586d`；随后提交类型修复 `ca2762cb`。
- `go build -o NUL ./cmd/aiw-say`：首次失败，报告 `Request.Polite` 类型不一致；修复后以相同命令重试，通过。未保留可执行产物。
- 静态核对插件分派、现有运行配置路径、配置/Profile 覆盖、提示词消息边界、stdout/stderr 路径及最终差异。

## 未执行的检查与风险

未运行 Python 黑盒用例、Go 测试、最终构建或真实 OpenAI 请求。仓库授权规则未允许执行这些运行时检查；测试用例需要独立 Tester 对具体命令和影响范围审查。模型可用性、真实 API 质量、Windows/Linux 安装布局和重试时序尚未通过运行验证。TOML 解析器仅处理本功能支持的标量字段；其他 AIW 表会被忽略，`[say]` 中不支持的字段会报错。
