# FD-036 Worker 实现报告，第 4 轮

<!-- aiw-data: FD-036-implementation-r4.json -->

## 结果

处理 Reviewer r1 的 `changes-requested`：在解析 TOML 表头前调用现有注释剥离逻辑，支持标准合法的表头行内注释；回归用例使用 `[say] # translation settings` 与 `[say.llm] # provider settings` 并验证显式设置生效。Work Item 1.11 已完成。FD Verification 记录了 Reviewer r1 报告和结果，随后以支持的 `refresh-worker` 生成与新 revision/digest 一致的 Worker 交接。

## 检查

- 执行 `go build -o NUL ./cmd/aiw-say`：通过，未保留最终可分发产物。
- 静态核对 `stripTOMLComment` 对表头的处理、现有 key/value 注释解析、回归用例与 Reviewer finding 对应关系。
- 本轮 Python 黑盒测试未运行；必须由新实现版本的独立 Tester 提出命令，并由 Planner 绑定当前事件授权。没有网络请求。

## 剩余风险

新回归用例和 Reviewer 修复尚无行为运行证据。先前 PM 明确接受的 S05/S08/S09/S14/S15、Linux/WSL profile 路径及 branch coverage 风险保持原样，需由 Reviewer r2 复核风险披露。
