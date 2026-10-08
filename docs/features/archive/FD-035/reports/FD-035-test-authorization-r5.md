# FD-035 Planner 覆盖率命令授权 r5

<!-- aiw-data: FD-035-test-authorization-r5.json -->

## 授权范围

用户已明确要求测量 FD-035 的 CLI 子进程分支覆盖率，并允许一次性将 coverage.py 安装到系统临时目录。本授权允许 Tester `fd035-tester-20261008-a7d92e` 在 `C:\Users\svictor\workspace\tools\ai-tools\aiw\.wt\FD-035` 执行一次：

```powershell
$env:FD035_COVERAGE_PACKAGE_DIR='C:\Users\svictor\AppData\Local\Temp\aiw-fd035-coverage-py'; python -B -m tests.fd035_coverage
```

本次绑定实现事件 `FD-035-000011-implementation-ready`、FD revision 11、digest `359ffa2cb9c8e6e4307a4189549da31a66bb69d6f9aca12bcb736fd6492b7017` 和 Tester session。已检查 runner：它会运行获批的 FD-035 unittest 模块，并为由黑盒测试启动的 CLI 子进程收集 `plugins/aiw-fd.py` 分支数据；测试仓库、bootstrap、配置和 coverage 数据均写入自动清理的临时目录。依赖仅位于系统临时目录；不改仓库依赖、不访问外部服务、不生成发布产物。预计少于 2 分钟。仅此一次，不授权重跑或扩大范围。
