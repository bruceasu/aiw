# FD-085 独立实现审阅

<!-- aiw-data: FD-085-review.json -->

结论：通过静态审阅，无阻断发现。未声称运行行为已经验证。

- FD：FD-085；来源事件：FD-085-000004-review-requested。
- 独立 Reviewer session：snapshot-review-20261018-root-snapshot-review。
- 实现提交：b144e33；比较基线：460fa7e。审阅时 FD revision 为 4。

已读取 FD、Worker 双文件报告、实际 build.py 调用路径、相关包路径/标识符校验、
usage、.env.example 与 aiw-packages 稳定规范。此 FD 无关联 Issue 或 OpenSpec change。

静态证据支持全部验收：compile/build 在分派点调用一次 build_version，显式参数优先于
非空环境值，缺省使用一次 UTC 微秒时间；同一版本传入全部组件与独立组包脚本。
publish 的显式版本参数不变。latest 的可选目标保留 None，全包名先归一化；全局读取
清单并按 built_at_ns 排序，旧清单回退 mtime_ns；选中记录检查版本、非空包列表与安全路径。
缺失文件、无效 JSON、时间戳或版本均进入明确错误路径。latest 无编译、组包、发布或安装调用。
清单写入位于所有组包和 dist/repository 暂存成功之后；compile 分支不写清单。
文档与示例的版本优先级、只读查询、旧记录兼容及完成时间契约一致。

实际命令：定向 Get-Content/rg；一批 `git diff 460fa7e b144e33 -- build.py .env.example docs openspec`；
`aiw fd --help`、`aiw fd show FD-085`、`aiw fd claim --help`、`aiw fd emit --help`，以及
`aiw fd claim FD-085 FD-085-000004-review-requested --session snapshot-review-20261018-root-snapshot-review`。
部分发现命令对不存在的本地 .ai 路径或无匹配 rg 返回失败；通过真实 aiw fd show/claim 获取交接状态，未伪造 receipt。
随后使用 apply_patch 写入本审阅证据并更新 FD；以真实 aiw fd emit 记录审阅结果。

Worker 的 Python compile-only 成功仅引用其报告，本 Reviewer 未重复执行。
未运行测试、编译、build.py CLI 行为验证、包构建、发布、安装、网络、lint 或验证脚本。

%% 运行行为仍未验证；本结论来自有界静态审阅。
%% 历史清单复制会影响 mtime 排序；新清单使用完成时间。
%% 清单逐个原子写入，不提供多清单事务；manifest 写入本身失败可能留下部分新记录。
