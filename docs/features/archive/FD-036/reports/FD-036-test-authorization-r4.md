# FD-036 Planner 测试授权，第 4 轮

<!-- aiw-data: FD-036-test-authorization-r4.json -->

## 审核结论

r3 的文件清理命令被执行策略拦截，未运行。r4 不含任何删除操作：使用唯一随机 `%TEMP%` exe 和独立 `%TEMP%` Go cache，运行后保留这两项临时文件；PowerShell 子进程结束后临时环境变量自然消失。`GOPROXY=off` 与 `GOSUMDB=off` 禁止网络下载/校验，Python bytecode 禁用。

批准以下一条精确命令，绑定 implementation-ready `FD-036-000014`、Revision 14、digest `914ae9b1043fa69226b2c7e00fed2ff4c2efe7b7e9ffad7f84c56ae8f83ad537`，Tester session `fd036-host-20261008-27cf10`：

```powershell
$tag='aiw-say-fd036-'+[guid]::NewGuid().ToString('N'); $testExe=Join-Path $env:TEMP ($tag+'.exe'); $env:GOCACHE=Join-Path $env:TEMP ($tag+'-cache'); $env:GOPROXY='off'; $env:GOSUMDB='off'; $env:PYTHONDONTWRITEBYTECODE='1'; go build -o $testExe ./cmd/aiw-say; if ($LASTEXITCODE -ne 0) { throw 'go build failed' }; $env:AIW_SAY_BIN=$testExe; python -B -m unittest -v tests.test_fd036_say_blackbox
```

- Working directory：`.wt/FD-036`
- Scope：编译 `cmd/aiw-say` 到随机 TEMP 文件，再运行根目录单个 `tests.test_fd036_say_blackbox` 模块；mock 服务仅监听 loopback。
- Expected duration：少于 2 分钟。
- Side effects：唯一随机 TEMP exe 与独立 Go cache 保留；测试的 Python 临时目录自动清理；子进程退出后环境变量自然消失；禁用下载与 Python bytecode。无仓库/用户配置/外部 API 写入或权限变化。
- Risk review：静态检查标准库 mock、临时配置、subprocess timeout 和断言。命令离线、聚焦、可检查，残留仅为系统 TEMP 下带唯一随机前缀的临时项。
- Human approval：not required

Planner：Codex workflow host  
Decision time：2026-10-08T08:29:00Z
