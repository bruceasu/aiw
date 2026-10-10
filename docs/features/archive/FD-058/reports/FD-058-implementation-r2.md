# FD-058 Worker 实现报告 r2

<!-- aiw-data: FD-058-implementation-r2.json -->

**FD：** FD-058  
**来源事件：** `FD-058-000007-design-ready`  
**范围：** 七个插件子目录；根目录平铺插件继续使用旧规则。

## 变更

- 保留 `aiw-ai`、`aiw-cz`、`aiw-git`、`aiw-github`、`aiw-req`、`aiw-say`、`aiw-skills` 七个插件子目录的 `plugin.toml`，清单包含名称、说明和多行帮助，并省略入口和启动字段。
- 删除 `src/plugins/plugin.toml`。根目录平铺单文件插件继续按原文件名和扩展名规则发现。
- 回退了本轮临时加入的搜索根清单发现/列表逻辑与 OpenSpec 规格改动；最终没有 Go 源码或稳定规格改动。

## 静态检查与编译

- 静态检查了七个清单的插件名、目录位置、说明和帮助；确认根目录清单已删除，发现器与稳定规格的临时改动已回退。
- compile-only 命令 `$env:GOPROXY='off'; $env:GOTOOLCHAIN='local'; python scripts/compile.py` 在收敛范围前已运行，退出码为 `1`。Go 编译子进程因系统分页文件/内存不足未能启动或分配内存，没有源码诊断。本次收敛后不重跑该命令；最终范围没有 Go 源码改动。
- 未运行测试、TOML 运行时解析、最终构建、格式化、lint 或验证脚本。

## 剩余风险

清单解析和插件实际调用没有运行时验证。编译命令因环境资源问题未能确认结果。
