# FD-098 实现报告

<!-- aiw-data: FD-098-implementation.json -->

来源事件：`FD-098-000003-design-ready`；Worker 会话：`fd098-host-20261019-b83a1c`。
设计基线：`3ea5e78`；工作分支：`feature/FD-098`，父分支 `develop`。

集合成员现在是安装配方和保留关系，不再构成已安装成员的持续版本锁。
`Operation.visit` 只在默认无 scope 更新的 member 根边使用已选版本，真实
dependencies 和显式发行配方选取保持精确。最终图检查对 members 检查存在性并递归
循环检测，对 dependencies 仍检查精确版本，保留旧安装记录，无须迁移。

使用文档和稳定规范已同步，明确集合版本不代表当前成员版本。单成员更新不选择
旧集合，因此不会重写其配方。默认 update 原有成员交集裁剪仍保留，未安装的新成员
不会补装；remove 和 autoremove 的名称可达性与安装原因、事务、锁均未改变。

## 验证与命令

执行一次离线 compile-only 命令批次：设置 `GOPROXY=off`、`GOSUMDB=off`、
`GOTOOLCHAIN=local`，调用 Python 内置 `compile()` 检查 operations.py，随后在同一
Python 进程中 `runpy.run_path('scripts/compile.py', run_name='__main__')`。
退出码 0；输出 `Python syntax compile: OK`。Go 脚本将两个入口输出到 NUL，
Python 语法编译不生成 pyc，未保留发行二进制。

静态追踪：单成员安装保留旧集合、默认更新已选根、嵌套集合、显式发行配方冲突、
反向真实依赖冲突、缺失成员和循环检测；同时读取 Manager.remove/orphans 的按名称
引用路径及锁下快照/dry-run 写入边界。最终执行一次只读批次：
`git diff 3ea5e78 -- <四个实现/文档/FD路径>`、`git diff --check`、`git status --short`，
检查无空白错误、无范围外改动；独立 Reviewer 将对当前提交进一步审查。

运行命令包括定向 Get-Content/rg、FD new/claim/emit、计划和实现的聚焦 Git 提交、
aiw git wt add、一次 compile-only 批次和一次最终只读 diff/status 批次。
独立设计报告为 ready；本报告不代替独立实现审查。

## 限制

未新增或执行测试；未运行格式化、lint、vet、验证脚本、运行时安装验证、最终包构建、
发布、安装或网络访问。集合修复不放宽共享资源真实 dependencies，本机 24 个旧技能
仍可能阻止共享资源换成 1.0.1，不能据此声称主程序已升级。实际安装目录保持原状。
