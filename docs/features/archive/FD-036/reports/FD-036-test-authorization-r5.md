# FD-036 Planner 测试授权，第 5 轮

<!-- aiw-data: FD-036-test-authorization-r5.json -->

## 审核结论

批准以下精确命令，绑定实现交接 `FD-036-000020-implementation-ready`、Revision 20、digest `757930093438ac7c76b37db7e8eac6cbd7adc4e356de2863fa287c8b860b6a0f` 和 Tester session `fd036-tester-20261008-r3-independent`。

```powershell
$tag='aiw-say-fd036-r3-'+[guid]::NewGuid().ToString('N'); $testExe=Join-Path $env:TEMP ($tag+'.exe'); $env:GOCACHE=Join-Path $env:TEMP ($tag+'-cache'); $env:GOPROXY='off'; $env:GOSUMDB='off'; $env:PYTHONDONTWRITEBYTECODE='1'; go build -o $testExe ./cmd/aiw-say; if ($LASTEXITCODE -ne 0) { throw 'go build failed' }; $env:AIW_SAY_BIN=$testExe; python -B -m unittest -v tests.test_fd036_say_blackbox
```

- Working directory：`.wt/FD-036`
- Scope：编译 `cmd/aiw-say` 到随机 TEMP 文件，然后仅运行 `tests.test_fd036_say_blackbox`，包含新 TOML 表头注释回归；mock 仅使用 loopback。
- Expected duration：约 10 秒；上一轮 19 项约 2.3 秒，另加编译。
- Side effects：唯一随机 TEMP exe 和独立 Go build cache 保留；Python 自管理临时目录清理，bytecode 禁用；关闭 Go module/checksum 网络访问。无仓库或用户配置写入、无真实 API、凭据、权限或部署动作。
- Risk review：已检查测试模块用标准库 HTTP mock、临时配置和隔离 subprocess；TOML 回归断言模型及配置值生效。命令离线、单模块、范围清晰。
- Human approval：not required

Planner：Codex workflow host  
Decision time：2026-10-08T08:38:47Z
