# FD-036 Planner 测试授权，第 7 轮

<!-- aiw-data: FD-036-test-authorization-r7.json -->

## 审核结论

**批准** Tester r5 `fd036-tester-20261008-r5-toml` 对实现交接 `FD-036-000029-implementation-ready` 执行一次以下精确命令。交接绑定 FD revision 29、digest `64a3b0ec060f0d1d492f6ac3e8acf7d4c3ad20184449064c0b6962596051f522`。

```powershell
$tag='aiw-say-fd036-r5-'+[guid]::NewGuid().ToString('N'); $testExe=Join-Path $env:TEMP ($tag+'.exe'); $env:GOCACHE=Join-Path $env:TEMP ($tag+'-cache'); $env:GOPROXY='off'; $env:GOSUMDB='off'; $env:PYTHONDONTWRITEBYTECODE='1'; go build -o $testExe ./cmd/aiw-say; if ($LASTEXITCODE -ne 0) { throw 'go build failed' }; $env:AIW_SAY_BIN=$testExe; python -B -m unittest -v tests.test_fd036_say_blackbox
```

工作目录：`C:\Users\svictor\workspace\tools\ai-tools\aiw\.wt\FD-036`。范围仅限 `./cmd/aiw-say` 临时编译及 `tests.test_fd036_say_blackbox` 这一组黑盒场景；预计约 5 秒。用例重点观察合法数组型未知键被忽略、已知字段 TOML 类型错误保留低优先级值、TOML 非法 `\a` 转义报错、无效字段回退及无效 CLI 仍失败。

已检查测试入口及新增断言：测试通过 `subprocess` 调用 `$AIW_SAY_BIN`；配置/profile 使用 Python `TemporaryDirectory`；模拟 API 仅绑定 `127.0.0.1`。随机 exe 与独立 Go 编译缓存写入 `%TEMP%`，`GOPROXY=off` 和 `GOSUMDB=off` 禁止外网依赖获取，`PYTHONDONTWRITEBYTECODE=1`/`-B` 禁止仓库字节码写入。命令不访问真实 API 或系统配置，不修改仓库生产数据。

本授权只允许对上述 event/revision/digest 和 Tester session 绑定的精确命令执行一次。Tester 不得在授权后改动用例；若需要更改，须先停止并重新提交命令供 Planner 审核。
