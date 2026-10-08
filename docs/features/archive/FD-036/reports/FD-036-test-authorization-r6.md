# FD-036 Planner 测试授权，第 6 轮

<!-- aiw-data: FD-036-test-authorization-r6.json -->

## 审核结论

**批准** Tester r4 `fd036-tester-20261008-r4-independent` 对实现交接 `FD-036-000023-implementation-ready` 执行一次以下精确命令。该交接绑定 FD revision 23，digest `ec7adbbb1e50c75d8e95834321a3f5d71f0765299d3934ddcee80694851f1c27`。

```powershell
$tag='aiw-say-fd036-r4-'+[guid]::NewGuid().ToString('N'); $testExe=Join-Path $env:TEMP ($tag+'.exe'); $env:GOCACHE=Join-Path $env:TEMP ($tag+'-cache'); $env:GOPROXY='off'; $env:GOSUMDB='off'; $env:PYTHONDONTWRITEBYTECODE='1'; go build -o $testExe ./cmd/aiw-say; if ($LASTEXITCODE -ne 0) { throw 'go build failed' }; $env:AIW_SAY_BIN=$testExe; python -B -m unittest -v tests.test_fd036_say_blackbox
```

工作目录：`C:\Users\svictor\workspace\tools\ai-tools\aiw\.wt\FD-036`。范围仅限 `./cmd/aiw-say` 临时编译和 `tests.test_fd036_say_blackbox` 黑盒行为场景；预计约 5 秒。

已检查测试入口：用例通过 `subprocess` 调用传入的二进制，使用 Python `TemporaryDirectory` 管理配置/profile，测试 HTTP 服务绑定 `127.0.0.1`；`PYTHONDONTWRITEBYTECODE=1` 与 `-B` 禁止写入字节码缓存。命令关闭 Go 代理，不下载依赖。命令会在 `%TEMP%` 新建随机命名 exe 和独立 Go cache；不会清理或覆盖既有路径。没有真实凭据、外网、系统配置或仓库生产文件副作用。

仅授权此命令执行一次，绑定上述实现事件、revision/digest 和 Tester session。Tester 不应在授权后改动用例；如需改动，应先停止并重新提交命令供 Planner 审核。
