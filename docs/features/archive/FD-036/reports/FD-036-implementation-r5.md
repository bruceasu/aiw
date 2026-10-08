# FD-036 Worker 实现报告，第 5 轮

<!-- aiw-data: FD-036-implementation-r5.json -->

## 结果

根据 PM r3 拒绝事件 `FD-036-000022-test-rejected` 和用户确认，完成 Work Item 1.12。合法 TOML 中已识别字段的无效值不覆盖当前低优先级值；当没有较低层配置时保留内置默认值。未知键忽略。TOML 语法错误仍在读取阶段报错，CLI 参数仍在最终校验阶段报错。配置按键名排序后逐项应用，避免 map 遍历顺序改变结果。

黑盒用例新增基础配置无效值回退到内置默认值、profile 无效值回退到基础配置、未知键忽略，以及损坏表头报错；现有 CLI 无效参数用例保留。README 和 FD Acceptance 已同步。

## 检查

- `go build -o NUL ./cmd/aiw-say`：通过，未保留可分发产物。
- 静态检查配置字段校验、优先级应用顺序、profile 合并、CLI 最终校验和新增黑盒断言。
- Python 黑盒测试未运行；新实现需由独立 Tester 按当前 revision 提出精确命令并获得 Planner 授权。

## 剩余风险

- 新增行为尚无运行时测试证据。
- 解析器仍实现应用所需的 TOML 表格与标量子集；本轮没有验证 TOML 标准的全部语法。
- S08、S14、S15 中剩余场景，真实 API/模型行为、Linux/WSL profile 路径及 branch coverage 仍未验证。
