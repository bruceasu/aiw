# FD-104 Worker 实施报告（r2）
<!-- aiw-data: FD-104-implementation-r2.json -->

## 结果

恢复已停止的 FD-104 Worker 交接，保留原归档快照，并将 Python 顺序工具调用配置恢复到 aiw-de `develop`。配置单测通过；端到端验收仍失败，具体轨迹和限制见下文。

Worker handoff：`FD-104-000011-work-requested`；会话：`fd104-worker-20261108-recovery`。

## 修改

- 保留旧归档于 `docs/features/archive/FD-104-pre-recovery-20261108/`，并恢复 FD 到活动目录。
- 在 aiw-de `develop` 提交 `cd5d632`：增加 `AIW_OPENAI_PARALLEL_TOOL_CALLS` 配置、README 说明、配置测试和脱敏的 Responses 请求轨迹。
- 为兼容 `ApiRuntimeConfig` 的旧位置参数调用，新字段追加在末尾并默认 `true`。
- 本次测试发现配置单测夹具缺少 `service_tier`；补齐后通过 4 项单测，并提交到 aiw-de `develop`（`857c280`）。

## 验证

- Python 内存 `compile()` 检查此前通过，覆盖 `openai_api_runtime.py`、`test_parallel_tool_call_setting.py` 和 `fd104_test_tool_call.py`。
- 按用户授权，在 `%TEMP%\aiw-fd104-20261108` 创建 Python 3.12 虚拟环境并安装 `workers/process-python/requirements.txt`，另装该验收脚本直接导入的 `httpx==0.28.1`。版本：`openai-agents==0.24.0`、`openai==3.46.0`。未修改项目依赖清单。
- `python -m unittest test_parallel_tool_call_setting` 首次运行暴露夹具缺字段；补齐后同一命令通过，4 项成功。
- 验收脚本首次因缺少 `httpx` 在导入阶段退出，没有发送请求。补齐依赖后，对安装版 gateway 的一次运行收到 HTTP 400 `invalid_request`，审计为 `execution_started=false`；该安装版早于 FD 源码。
- 使用当前源码启动 gateway（`GOTOOLCHAIN=local GOPROXY=off go run . start --config C:\green\aiw\gateway.json`）后重试一次。第 1 轮 HTTP 200，返回有效 `add_numbers(19, 23)` 调用，且请求明确为 `parallel_tool_calls=false`。SDK 提交结果 `42` 后，第 2、3 轮 backend 又返回新的 `function_call`；SDK 总共执行工具三次，最终以 `MaxTurnsExceeded` 退出（code 1），未得到 `TOOL_CALL_OK:42`。
- 静态归因：`decodeRequest` 校验并匹配 `function_call`/`function_call_output` 的 ID，再把两者写入下一轮 prompt；`appInstructions` 包含调用方指令和工具结果规则；`executeApp` 把 backend 下一次决策映射回新的 Responses `function_call`。轨迹显示这条路径返回 200、SDK 提交结果 `42`，随后 backend 再发工具调用。证据支持“模型/backend 响应”或“模型依赖型验收预期”的假设，但未确认 gateway schema/顺序配置缺陷，也未查明 backend 忽略指令的原因；端到端 Acceptance 未通过。
- 两个 gateway 实例均已优雅停止。未运行 Go 测试、宽泛 Python 测试套件或更多端到端重试。当前受影响验收的运行预算已耗尽，停止相关执行和修复。
- 原 Reviewer 报告 `docs/features/archive/FD-104-pre-recovery-20261108/reviews/FD-104-review-r1.md` 仍为 changes-requested。验收问题解决前不请求独立复审，也不归档 Complete。
- 最新人工决策交接为 `FD-104-000017-needs-decision`，源事件 `FD-104-000015-needs-decision`；目标为 human，CLI 未启动 agent。

## 下一步

静态判断重复调用属于产品缺陷、backend/model 行为还是模型依赖型验收预期不合适。当前预算内不要继续重试或修复；需要新的明确指示和测试周期后再继续。验收通过后再请求独立审查。
