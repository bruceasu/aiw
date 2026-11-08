# FD-104 Worker 实施报告

<!-- aiw-data: FD-104-implementation-r1.json -->

## 结果

完成 FD-104 的 gateway schema 注解兼容、Python Agents 并行调用配置、配置用例、顺序调用验收脚本和稳定规格更新。Worker handoff：`FD-104-000006-design-ready`；会话：`fd104-worker-20261023-a`。

## 修改

- `src/programs/agent-gateway/schema.go` 仅接受 schema 各层节点上的字符串 `title`；其他 schema 校验保持不变。
- `openspec/specs/agent-gateway/spec.md` 记录兼容范围及拒绝边界。
- `workers/process-python/openai_api_runtime.py` 新增 `AIW_OPENAI_PARALLEL_TOOL_CALLS`，默认 `true`，接受大小写不敏感的 `true`/`false`，非法值 fail-fast；SDK 设置使用该值。
- `workers/process-python/README.md` 说明连接 agent-gateway 时设置为 `false`。
- `workers/process-python/test_parallel_tool_call_setting.py` 覆盖默认值、显式值和非法值。
- `workers/demo-python-with-openai-agents/fd104_test_tool_call.py` 检查 SDK schema 的根和参数 `title`，显式请求顺序调用，并检查工具调用一次及结果 `TOOL_CALL_OK:42`。
- FD Work Item 1.3 原定路径 `test_tool_call.py` 位于主 `aiw-de` 工作区 `.git/info/exclude` 指定的本地忽略目录，未纳入 feature worktree。为保留该用户本地文件，新增独立验收脚本且没有覆盖它。

## 验证

- 已运行 `python src/programs/agent-gateway/scripts/compile.py`：Go Windows 与 Linux compile-only 构建均完成，脚本退出码为 0。
- 已对三个变更 Python 文件使用 Python `compile()` 做语法编译检查，成功。
- 未运行 Python 配置用例、Go 测试、Agents SDK 集成/网络调用或 gateway 端到端检查。未验证实际 Agents SDK 与 Codex backend 的线上往返。
- 静态检查确认 gateway 仍在工具执行前拒绝 `parallel_tool_calls=true`；`title` 不进入 schemaNode 的验证语义。

## 剩余风险

用户日志没有 HTTP 响应体，无法证明原请求首先因并行标志还是 schema 注解被拒绝；修复后仍可能存在其他不支持字段。新增 Agents SDK 顺序往返脚本未运行，需要在明确配置 gateway 和凭据后单独执行。
