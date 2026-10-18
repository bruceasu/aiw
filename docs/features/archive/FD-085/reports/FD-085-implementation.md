# FD-085 实现报告

<!-- aiw-data: FD-085-implementation.json -->

已实现两个 Work Items：compile/build 优先使用 --version，其次非空 AIW_VERSION，
否则一次生成 UTC snapshot-YYYYMMDD-HHMMSSffffffZ，并在开始前打印。
.env.example 不再默认设置 dev；独立组包脚本、publish 的版本契约不变。

新增 latest [target]，只打印最近构建版本，支持完整包名与分组。读取既有构建清单，
以新增 built_at_ns 判断完成顺序，历史清单回退 mtime_ns。校验所选版本及包路径，
缺失/损坏时明确失败。compile-only 不更新记录；成功组包及 dist/repository 暂存
之后才写清单，避免编译/组包/暂存失败覆盖先前成功记录。

静态证据：一批 git diff 460fa7e，限定 build.py、.env.example、usage 和 stable spec。
检查显式/env/自动版本优先级、每次调用一次生成、latest 无参数 None 路径及别名、
全局/目标清单筛选、历史时间回退、包缺失错误和清单写入顺序。

实际 compile-only 命令：

```text
python -c "from pathlib import Path; compile(Path('build.py').read_bytes(), 'build.py', 'exec'); print('Python compile-only passed; no imports or output artifacts')"
```

退出码 0，无模块执行、字节码或最终产物。现有 scripts/compile.py 仅编译无关 Go
入口，因此使用 Python 编译器。另执行定向 Get-Content/rg、Git 状态/提交及 aiw fd
状态、worktree 创建等本地流程命令。用户直接提出实现，无 Worker 派发，source_event
为 null；通过 set-status/request-review 获取真实独立审阅。

未运行测试、CLI 运行验证、包构建、安装、发布、网络、lint 或验证脚本。
%% 历史清单复制可能改变 mtime；新清单使用完成时间。
%% 清单逐个原子写入，未新增多文件事务；运行行为尚未验证。
