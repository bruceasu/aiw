# FD-036 Planner 测试授权，第 2 轮

<!-- aiw-data: FD-036-test-authorization-r2.json -->

## 审核结论

本授权替代 r1；r1 命令尚未运行。静态顾问指出 r1 会写共享 Go build cache、可能生成仓库 `__pycache__` 且使用固定 TEMP 路径。r2 将临时目录、可执行文件、Go build cache 和 Python 临时文件限定在唯一随机 TEMP 沙箱中，禁用 Go module/checksum 网络访问并禁止 Python bytecode 文件。

批准以下一条精确命令，绑定 implementation-ready `FD-036-000014`、Revision 14、digest `914ae9b1043fa69226b2c7e00fed2ff4c2efe7b7e9ffad7f84c56ae8f83ad537`，Tester session `fd036-host-20261008-27cf10`：

```powershell
$baseTemp=[IO.Path]::GetFullPath($env:TEMP); $sandbox=Join-Path $baseTemp ('aiw-say-fd036-'+[guid]::NewGuid().ToString('N')); $oldTemp=$env:TEMP; $oldTmp=$env:TMP; $oldTmpdir=$env:TMPDIR; $oldGoCache=$env:GOCACHE; $oldGoProxy=$env:GOPROXY; $oldGoSum=$env:GOSUMDB; $oldPy=$env:PYTHONDONTWRITEBYTECODE; try { New-Item -ItemType Directory -Path $sandbox | Out-Null; $env:TEMP=$sandbox; $env:TMP=$sandbox; $env:TMPDIR=$sandbox; $env:GOCACHE=Join-Path $sandbox 'gocache'; $env:GOPROXY='off'; $env:GOSUMDB='off'; $env:PYTHONDONTWRITEBYTECODE='1'; $testExe=Join-Path $sandbox 'aiw-say.exe'; go build -o $testExe ./cmd/aiw-say; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; $env:AIW_SAY_BIN=$testExe; python -B -m unittest -v tests.test_fd036_say_blackbox; exit $LASTEXITCODE } finally { if ($null -eq $oldTemp) { Remove-Item Env:TEMP -ErrorAction SilentlyContinue } else { $env:TEMP=$oldTemp }; if ($null -eq $oldTmp) { Remove-Item Env:TMP -ErrorAction SilentlyContinue } else { $env:TMP=$oldTmp }; if ($null -eq $oldTmpdir) { Remove-Item Env:TMPDIR -ErrorAction SilentlyContinue } else { $env:TMPDIR=$oldTmpdir }; if ($null -eq $oldGoCache) { Remove-Item Env:GOCACHE -ErrorAction SilentlyContinue } else { $env:GOCACHE=$oldGoCache }; if ($null -eq $oldGoProxy) { Remove-Item Env:GOPROXY -ErrorAction SilentlyContinue } else { $env:GOPROXY=$oldGoProxy }; if ($null -eq $oldGoSum) { Remove-Item Env:GOSUMDB -ErrorAction SilentlyContinue } else { $env:GOSUMDB=$oldGoSum }; if ($null -eq $oldPy) { Remove-Item Env:PYTHONDONTWRITEBYTECODE -ErrorAction SilentlyContinue } else { $env:PYTHONDONTWRITEBYTECODE=$oldPy }; Remove-Item Env:AIW_SAY_BIN -ErrorAction SilentlyContinue; $resolvedSandbox=[IO.Path]::GetFullPath($sandbox); if ($resolvedSandbox.StartsWith($baseTemp+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase) -and (Split-Path $resolvedSandbox -Leaf).StartsWith('aiw-say-fd036-')) { Remove-Item -LiteralPath $resolvedSandbox -Recurse -Force -ErrorAction SilentlyContinue } }
```

- Working directory：`.wt/FD-036`
- Scope：构建 `cmd/aiw-say` 到唯一沙箱，然后运行根目录单个 `tests.test_fd036_say_blackbox` 模块；仅访问本地 loopback mock API。
- Expected duration：少于 2 分钟。
- Side effects：唯一 TEMP 沙箱中的 exe、Go build cache 和 Python 临时目录，均在 `finally` 清理；Python bytecode 禁用；环境变量在 `finally` 恢复/删除；Go module/checksum 网络访问禁用。仓库内仅读取源码/测试文件，不写入 `__pycache__` 或用户配置。
- Risk review：已静态检查所有测试方法，确认使用 Python 标准库、本地 mock、临时配置、子进程 timeout 和临时目录；最终删除前验证沙箱是系统 TEMP 下带 FD-036 唯一前缀的目录。无依赖下载、外部服务、真实凭据、权限提升或发布产物。
- Human approval：not required

Planner：Codex workflow host  
Decision time：2026-10-08T08:27:00Z
