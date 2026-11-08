# FD-104 Worker 后续验收报告（r3）
<!-- aiw-data: FD-104-implementation-r3.json -->

## 结果

按用户指示改用 `.env` 中的 `DEEPSEEK_MODEL`，DeepSeek Responses 端到端验收通过。`.env` 由临时 launcher 在进程内加载；没有展示、输出或保存其中任何变量值。

来源事件：`FD-104-000017-needs-decision`。测试脚本提交：aiw-de `82c2988`；gateway 源码基线：AIW `8e73956d3fe1ed12e97906d4959aad11be8b1aee`。

## 执行

- 在 `%TEMP%\aiw-fd104-20261108` 的临时 Python 3.12 环境中安装 `python-dotenv==1.2.4`。其余版本：`openai-agents==0.24.0`、`openai==3.46.0`、`httpx==0.28.1`。
- 精确命令：

  ```powershell
  & (Join-Path $env:TEMP 'aiw-fd104-20261108\Scripts\python.exe') (Join-Path $env:TEMP 'aiw-fd104-20261108\fd104_deepseek_runner.py')
  ```

- 工作目录：AIW 仓库根目录。launcher 从 `C:\green\aiw\.env` 加载变量，为 gateway 创建临时 `openai_proxy` 配置，启动当前源码 gateway 并运行 `fd104_test_tool_call.py`。最长 7 分钟，最多三轮 API 请求；本次实际两轮。
- 验收脚本 dirty diff SHA-256：`9525fdd8183d9c0f3525fb48fbcd345844dffb3a612eb390e4bcab1ef85b98d0`；临时 launcher SHA-256：`38f88a1941726af79256f35819f7c38ab9d3e0c70132f0e464130e84bf56fadf`。

## 结果证据

- 第 1 轮请求含 `parallel_tool_calls=false` 和 SDK 生成的 strict function schema；gateway/DeepSeek 返回 HTTP 200 和唯一调用 `add_numbers(19, 23)`。`title_annotations=true`。
- SDK 提交工具输出 `42` 后，第 2 轮返回 HTTP 200 及 `TOOL_CALL_OK:42`。
- 终端结果：`model_source=DEEPSEEK_MODEL`、`tool_calls=[[19, 23]]`、`output=TOOL_CALL_OK:42`、`status=ok`、exit code 0。
- launcher 将上游 `DEEPSEEK_API_KEY` 只传给 gateway；测试客户端只使用临时生成的本地 gateway key。没有记录 `.env` 值。Gateway 优雅停止，临时代理配置自动清理。
- DeepSeek 官方文档说明 `parallel_tool_calls` 参数会被忽略；这次证明 schema 接受和单次函数调用往返，不证明 DeepSeek 服务端遵守了并行开关。旧 gpt-6-luna 重复调用失败证据仍保留。
- 配置单测 4 项通过，夹具修正已在 aiw-de `857c280` 提交。未运行 Go 测试或宽泛 Python 测试套件。

## 下一步

针对当前实现和上述证据执行独立 Reviewer；保持历史 changes-requested finding 可见，并按新评审结论处理。
