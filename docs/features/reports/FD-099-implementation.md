# FD-099 实现报告

<!-- aiw-data: FD-099-implementation.json -->

来源事件：`FD-099-000003-design-ready`；Worker：`fd099-host-20261019-c142f0`。
设计基线：`01606f6`，分支 `feature/FD-099`，父分支 `develop`。

`git-merge-to.py` 增加可选 `--push`。参数解析先拒绝重复开关，再移除该开关并沿用
原有模式和分支个数约束。普通模式及 --new 模式均支持开关在分支参数前后出现，
不改变默认源选择或原 --new 位置要求。没有该开关时不推送。

推送位于全部源合并、worktree remove 和临时父目录清理成功之后。先刷新输出本地
合并成功信息，再执行一次 `git push --no-follow-tags --recurse-submodules=no -- origin
refs/heads/<target>:refs/heads/<target>`，包括 --new 单源创建。显式目标 refspec 不依赖
调用方分支或 push.default，关闭 follow-tags/子模块推送，不强推、不设置 upstream。

任何合并或清理失败保持原返回/恢复路径，不进入 push。push 非零则返回其原退出码，
并说明本地合并保留；OSError/中断继续通过原异常处理返回失败，本地成功信息已输出。
没有重试、回滚或实际远程访问。META、CLI 文档及稳定规范已同步。

## 验证与命令

执行一次离线 compile-only 批次，环境为 GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local。
命令：`python -c "from pathlib import Path; import runpy; p = Path('src/plugins/aiw-git/git-merge-to.py'); compile(p.read_text(encoding='utf-8'), str(p), 'exec'); print('Python syntax compile: OK'); runpy.run_path('scripts/compile.py', run_name='__main__')"`。
退出码 0，输出 `Python syntax compile: OK`；Python 内存语法编译无 pyc，Go 两入口输出 NUL。

静态追踪 parse_args/preflight/main/git、全部提前失败返回和异常恢复路径。
最终一次只读 diff/status 批次：`git diff 01606f6 -- <实现/文档/FD/设计报告路径>`、
`git diff --check`、`git status --short`，无空白错误及范围外改动。随后仅澄清 META
帮助字符串：合并失败保留 worktree，push 失败发生在清理之后；执行逻辑未再变更。
该字符串澄清作静态复核，未重复编译；独立审查覆盖最后源码版本。
独立设计报告为 ready，实现仍须独立 Reviewer 审查；静态结论不等同真实远程验证。

其他实际命令：定向 Get-Content/rg；FD new/claim/emit；聚焦 Git 计划/实现提交及
aiw git wt add。发现一次猜测脚本路径不存在及一次 PowerShell glob 搜索路径错误，
均未重跑原失败命令，改为对实际目录的定向静态读取。

## 限制与风险

未新增或执行测试、merge-to 功能验证、真实 push、fetch、网络、构建、发布安装、
格式化、lint、vet 或验证脚本。远程认证、origin 可用性和非快进拒绝未运行验证，
由 Git 返回失败；成功的本地合并不因此回滚。安装端插件尚未更新。
