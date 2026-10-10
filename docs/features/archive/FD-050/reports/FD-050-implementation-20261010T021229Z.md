# FD-050 实施报告

<!-- aiw-data: FD-050-implementation-20261010T021229Z.json -->

## 交付内容

- 将主 Go 模块迁入 `src/`，保持 `module aiw` 和现有 Go 导入路径；编译与测试辅助脚本从新模块根运行。
- 将独立项目、插件、技能和 Agent 模板分别迁入 `src/programs/`、`src/plugins/`、`src/skills/` 和 `src/agent-templates/`。
- 新增根级 `build.py`，实现原有构建动作和组合顺序；主程序输出到 `bin/`，插件暂存输出到 `dist/plugins/`，安装来源使用新的源码位置。删除 `build.bat`。
- 插件发现保留可执行文件旁和当前工作区插件优先，并从当前 Git checkout 的 `src/plugins/` 查找开发插件。
- 更新仓库指引、使用文档、构建夹具、稳定规格源码链接与 `.gitignore`。历史归档报告未改写。

## 验证

- `python scripts/compile.py`：通过，按脚本配置使用 NUL 输出，没有保留最终二进制。
- `git diff --check develop...HEAD`：通过。
- 静态核对确认根级 `cmd/`、`internal/`、`program/`、`plugins/`、`skills/`、`agent-templates/`、`go.mod` 和 `go.sum` 均已移除；构建动作的子进程目录、离线 Go 环境、安装来源及配置排除规则已审阅。
- 未运行测试、`build.py` 构建或安装动作、完整构建、lint、格式化和验证脚本。

## 剩余风险

- Python 构建器的 Windows 交叉构建和文件复制/配置保留行为经过静态审阅，未在本次运行；如需运行构建或安装动作，应另行授权。
- 技能 CLI 的路径引用已按新源码位置调整，但测试夹具未运行。

Worker handoff：`FD-050-000006-work-requested`（Worker session `fd050-worker-20261010-0d6e71`）。
