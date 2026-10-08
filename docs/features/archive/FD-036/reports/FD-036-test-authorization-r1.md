# FD-036 Planner 测试授权，第 1 轮

<!-- aiw-data: FD-036-test-authorization-r1.json -->

## 审核结论

批准以下一条精确命令，绑定 implementation-ready `FD-036-000014`、Revision 14、digest `914ae9b1043fa69226b2c7e00fed2ff4c2efe7b7e9ffad7f84c56ae8f83ad537`，并绑定 Tester session `fd036-host-20261008-27cf10`：

```powershell
$testExe=Join-Path $env:TEMP ('aiw-say-fd036-'+[guid]::NewGuid().ToString('N')+'.exe'); $oldGoProxy=$env:GOPROXY; try { $env:GOPROXY='off'; go build -o $testExe ./cmd/aiw-say; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; $env:AIW_SAY_BIN=$testExe; python -m unittest -v tests.test_fd036_say_blackbox; exit $LASTEXITCODE } finally { Remove-Item -LiteralPath $testExe -Force -ErrorAction SilentlyContinue; Remove-Item Env:AIW_SAY_BIN -ErrorAction SilentlyContinue; if ($null -eq $oldGoProxy) { Remove-Item Env:GOPROXY -ErrorAction SilentlyContinue } else { $env:GOPROXY=$oldGoProxy } }
```

- Working directory：`.wt/FD-036`
- Scope：构建 `cmd/aiw-say` 到唯一随机 TEMP 路径，然后只运行根目录 `tests.test_fd036_say_blackbox`；测试使用本地 loopback mock，不调用真实服务。
- Expected duration：少于 2 分钟。
- Side effects：Go 临时编译缓存；TEMP 下临时 exe 在 `finally` 中删除；环境变量 `AIW_SAY_BIN` 和临时设定的 `GOPROXY` 在 `finally` 中恢复/删除；Python 临时目录和本地 mock server。命令禁用 Go module 下载，不触碰用户配置或仓库外业务数据。
- Risk review：静态检查了测试模块的 mock HTTP handler、固定临时配置、子进程超时和临时目录清理；模块只使用 Python 标准库。前次 compile-only 构建已通过。范围离线、聚焦且可检查，按 Planner 低风险批准。
- Human approval：not required

Planner：Codex workflow host  
Tester session：`fd036-host-20261008-27cf10`  
Decision time：2026-10-08T08:25:00Z
