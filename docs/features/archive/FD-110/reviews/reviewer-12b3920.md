# FD-110 独立评审

- **FD：** FD-110
- **来源事件：** `FD-110-000002-review-requested`
- **Reviewer session：** `reviewer-20261108-fd110-static`
- **评审提交：** `12b3920`（基线 `12b3920^`）
- **结论：** Changes requested

## 发现

### [High] `aiw-cz --provider deepseek` 会在参数解析阶段被拒绝

- **位置：** `src/plugins/aiw-cz/aiw-cz.py:22`
- **要求：** FD-110 Acceptance 要求 `aiw-cz` 可显式选择 DeepSeek；现有 CLI 的 `--provider` 是显式 provider 选择入口。
- **问题：** 新增 DeepSeek provider 实现和配置后，argparse 的 `--provider` choices 仍仅包含 `codex`、`copilot`、`openai`。因此用户传入 `--provider deepseek` 时会直接收到参数错误，不能通过 CLI 显式选择该 provider。
- **影响：** TOML `provider = "deepseek"` 路径有实现，但 CLI provider 选择契约对 DeepSeek 不完整；常见显式调用方式不可用。
- **最小修正：** 将 `deepseek` 加入 `--provider` choices，并确认 `--model` 覆盖使用所选 DeepSeek provider 的模型槽。

## 已审查范围与证据

- 检查了 Go provider/config/auto-detection 路径、Python `aiw-cz` 配置与 fallback 路径、Direct/Gateway/proxy 文档和 Agent Gateway `openai_proxy` 稳定 spec。
- Go DeepSeek provider 使用独立 `DEEPSEEK_*` 环境变量；自动检测只以 `DEEPSEEK_API_KEY` 选择 DeepSeek，且不会借用 OpenAI key。Python fallback 只在 DeepSeek key 已配置时加入 DeepSeek，也不会回退到 `OPENAI_API_KEY`。
- Go Responses 请求仅在 OpenAI provider 下设置 `store` 和 `previous_response_id`；DeepSeek 不设置二者，也不把返回 ID 作为 `ThreadID` 暴露。
- Agent Gateway 的现有 `openai_proxy` spec 支持固定 upstream URL 和环境变量凭据替换，所提供的 DeepSeek 配置示例与该契约一致。
- 本 Reviewer 未运行编译或测试。FD 记录的 Go compile-only 检查结果并非本次 Reviewer 执行；未运行测试、最终构建、实时 API 请求或网络检查。

## 命令与限制

- 已运行：`aiw fd show FD-110`、`aiw fd claim FD-110 FD-110-000002-review-requested --session reviewer-20261108-fd110-static`、`aiw fd --help`、`aiw fd emit --help`、`git show --stat --oneline 12b3920`、`git diff 12b3920^ 12b3920 -- ...`、`rg -n "openai_proxy|proxy_base_url|proxy_api_key_env" openspec/specs src/programs/agent-gateway`、`Select-String ... aiw-cz.py`。
- 跳过：测试、构建、Python 执行和 DeepSeek/Gateway 实时调用，遵循本 FD 与仓库执行预算。
- **剩余风险：** DeepSeek 与真实上游的 wire-level 兼容性没有运行时验证；当前发现可通过修复 CLI provider 列表解决，其他审查结论基于静态实现与现有接口契约。
