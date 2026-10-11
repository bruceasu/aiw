# FD-058 Worker 实现报告 r3

<!-- aiw-data: FD-058-implementation-r3.json -->

**FD：** FD-058  
**来源事件：** `FD-058-000009-changes-requested`  
**修正依据：** `docs/features/reviews/FD-058-review-r1.md`

## 修正

- 按 Reviewer r1 指出的问题，将 `src/internal/plugin/discover.go`、`src/internal/plugin/manifest.go` 和 `openspec/specs/cli-and-plugins/spec.md` 恢复到搜索根清单支持引入前的行为与规格。
- 保留七个正式插件子目录的 `plugin.toml`，删除 `src/plugins/plugin.toml`。根目录平铺插件仍按旧文件名和扩展名规则发现。
- r2 报告关于发现器和稳定规格已回退的描述在当时并不准确；本报告记录完成实际回退后的状态。

## 静态检查与验证边界

- 静态核对了三处回退差异、七个目录清单和根目录清单状态。
- compile-only 命令 `$env:GOPROXY='off'; $env:GOTOOLCHAIN='local'; python scripts/compile.py` 在范围调整前已运行，退出码为 `1`，原因是 Go 编译子进程受系统分页文件/内存限制，未观察到源码诊断。本轮没有重跑。
- 未运行测试、TOML 运行时解析、插件调用、最终构建、格式化、lint 或验证脚本。

## 剩余风险

编译结果未确认；清单解析和插件调用没有运行时验证。
