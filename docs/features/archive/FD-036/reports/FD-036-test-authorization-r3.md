# FD-036 Planner 测试授权，第 3 轮

<!-- aiw-data: FD-036-test-authorization-r3.json -->

## 审核结论

r2 的递归沙箱清理命令被执行策略拦截，未运行。r3 改用 TEMP 中随机命名的可执行文件与独立 Go build cache；仅以 `Remove-Item -LiteralPath` 删除单个已知 exe，不递归删除目录。唯一残留是 `%TEMP%` 中随机命名的专用 Go 编译缓存。

批准以下一条精确命令，绑定 implementation-ready `FD-036-000014`、Revision 14、digest `914ae9b1043fa69226b2c7e00fed2ff4c2efe7b7e9ffad7f84c56ae8f83ad537`，Tester session `fd036-host-20261008-27cf10`：

```powershell
$tag='aiw-say-fd036-'+[guid]::NewGuid().ToString('N'); $testExe=Join-Path $env:TEMP ($tag+'.exe'); $oldGoCache=$env:GOCACHE; $oldGoProxy=$env:GOPROXY; $oldGoSum=$env:GOSUMDB; $oldPy=$env:PYTHONDONTWRITEBYTECODE; try { $env:GOCACHE=Join-Path $env:TEMP ($tag+'-cache'); $env:GOPROXY='off'; $env:GOSUMDB='off'; $env:PYTHONDONTWRITEBYTECODE='1'; go build -o $testExe ./cmd/aiw-say; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; $env:AIW_SAY_BIN=$testExe; python -B -m unittest -v tests.test_fd036_say_blackbox; exit $LASTEXITCODE } finally { if ($null -eq $oldGoCache) { Remove-Item Env:GOCACHE -ErrorAction SilentlyContinue } else { $env:GOCACHE=$oldGoCache }; if ($null -eq $oldGoProxy) { Remove-Item Env:GOPROXY -ErrorAction SilentlyContinue } else { $env:GOPROXY=$oldGoProxy }; if ($null -eq $oldGoSum) { Remove-Item Env:GOSUMDB -ErrorAction SilentlyContinue } else { $env:GOSUMDB=$oldGoSum }; if ($null -eq $oldPy) { Remove-Item Env:PYTHONDONTWRITEBYTECODE -ErrorAction SilentlyContinue } else { $env:PYTHONDONTWRITEBYTECODE=$oldPy }; Remove-Item Env:AIW_SAY_BIN -ErrorAction SilentlyContinue; if (Test-Path -LiteralPath $testExe -PathType Leaf) { Remove-Item -LiteralPath $testExe -Force -ErrorAction SilentlyContinue } }
```

- Working directory：`.wt/FD-036`
- Scope：编译 `cmd/aiw-say` 到随机 TEMP 文件，再运行单个 `tests.test_fd036_say_blackbox` 模块；mock 服务仅监听 loopback。
- Expected duration：少于 2 分钟。
- Side effects：单个 exe 在 `finally` 中删除；唯一随机 TEMP Go build cache 保留；测试创建的 Python 临时目录自动清理；变量在 `finally` 恢复/删除；禁用 Go module 与 checksum 下载并禁用 Python bytecode。无外网/API/凭据/用户配置/仓库写入。
- Risk review：静态检查测试模块的标准库 mock、临时文件、subprocess timeout、请求断言和临时目录清理。精确 scope 离线、聚焦、可检查。
- Human approval：not required

Planner：Codex workflow host  
Decision time：2026-10-08T08:28:00Z
