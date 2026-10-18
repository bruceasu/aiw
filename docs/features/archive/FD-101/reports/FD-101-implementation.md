# FD-101 实现报告

<!-- aiw-data: FD-101-implementation.json -->

来源事件 `FD-101-000004-design-ready`；Worker `fd101-host-20261019-58dca1`。
基线 `033da91`；分支 feature/FD-101，父分支 develop。

用户确认三个插件自包含并迁为目录；原修订 2 最小修复方案已由修订 3 取代。
file/patch/notify 的顶层脚本已移入各自同名目录，新增严格 schema-1 manifest，
保留命令名、入口逻辑、META 和协议；notify 唯一行为调整是从自身文件位置定位相邻
Teams 脚本，原隔离子进程、配置和 console 提前返回均不变。

源码目录内的 codec 开发 shim 用 importlib 加载顶层唯一实现，Teams shim 用 runpy
传递原 argv 和 __main__。PLUGIN_PRIVATE_FILES 同时控制 shim 排除和完整实现私有目标
写入，发行包不含源码树转交逻辑。三个包分别拥有私有辅助文件，不覆盖旧 runtime 根路径。

规划和包元数据的无条件 runtime 依赖均删除，新 pkg 和三插件不会因本身版本变化拉入
runtime。旧 runtime 目标及两个顶层 canonical 源码未改，旧发行 ZIP/状态不迁移；
安装器仍精确校验真实依赖，并通过既有 obsolete 清理移除本插件旧单文件。

entries 的目录分支只对目录内实际 Python 入口作 AST META 读取，保留行为/确认提示，
manifest 字段优先，不执行插件也不读取目录外 Python 元数据。其名称与路径采用现有
目录 manifest 规则。使用文档和稳定规范已同步源入口路径和新包/旧包边界。

## 实际检查

一次离线 compile-only 批次设置 GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local，
用 Python 内置 compile() 检查 scripts/package_definitions.py、entries.py、三个目录
入口及开发 shim 共 8 个文件，再 `runpy.run_path('scripts/compile.py', run_name='__main__')`。
退出码 0，输出 Python syntax compile: OK (8 files)；Python 不写 pyc，Go 两入口输出 NUL。
精确命令和路径清单见 JSON 侧车。

静态追踪目标发现/manifest 允许字段、tree_entries 排除与 c.one 私有目标、expand/
版本清单闭包、Manager 所有权/旧文件清理、codec 模块加载、notify argv/隔离和 entries
AST 合并。最后一次只读批次：git diff --find-renames 033da91 -- <打包/源码/文档/规范路径>、
git diff --check、git status --short，无空白错误或范围外改动。静态检查与独立实现审查不等同运行验收。

其他命令包括定向 Get-Content/rg、FD new/claim/emit、聚焦 Git 提交和 aiw git wt add。
若干早期猜测路径只读读取未命中，未重跑相同失败命令，设计后只使用确认路径。

未新增/运行测试、CLI/通知功能、规划模拟、最终包构建、格式化/lint/vet、验证脚本、
网络、push、发布或安装。目录实际运行/升级和 Teams 路径未运行验证；安装端尚未更新。
