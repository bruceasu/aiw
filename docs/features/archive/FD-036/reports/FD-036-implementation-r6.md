# FD-036 Worker 实现报告，第 6 轮

<!-- aiw-data: FD-036-implementation-r6.json -->

## 结果

根据 Reviewer r3 的 changes-requested，完成 Work Item 1.13。用户授权新增 TOML 解析依赖并获取；本轮将 `github.com/BurntSushi/toml` 锁定为 v1.6.0。`readConfig` 现在先解析完整 TOML 文档，再提取已知配置表；解析器负责拒绝格式损坏，未知键的数组等值不会触发应用层标量解析错误。应用配置时按字段要求检查 TOML 类型，类型错误、枚举错误或字段值无效均不覆盖较低优先级值。

新增黑盒回归：合法数组型未知键被忽略；TOML 不允许的 Go `\a` 字符串转义报错退出。原有配置回退、未知标量键和无效 CLI 行为用例保留。

## 检查

- `go get github.com/BurntSushi/toml@v1.6.0`：成功，依赖已记录在 `go.mod`/`go.sum`。
- PowerShell 设置 `GOPROXY=off`、`GOSUMDB=off` 后执行 `go build -o NUL ./cmd/aiw-say`：通过，未保留可分发产物。
- 静态检查解析、表提取和字段类型转换路径，以及新增黑盒断言。
- 黑盒测试未运行；新增与既有行为需由独立 Tester 基于本轮实现提出精确命令并取得 Planner 授权。

## 剩余风险

- 本轮新增 TOML 边界用例尚无运行时证据，等待 Tester。
- Tester r4 已接受但未完整覆盖的 S08/S09/S14/S15、branch coverage、真实 API/模型行为、跨平台 profile 路径和翻译质量风险仍然适用。
