# FD-082 首轮审查补充更正

<!-- aiw-data: FD-082-review-r1-supplement.json -->

首轮 verification-passed（FD-082-000005-verification-passed）发出后，host 提供了
输入包位于安装根内且与载荷目标重合的边界。本 Reviewer 静态核对确认是真实缺陷，
首轮通过结论不足以支持交付，已立即通知 host 暂停，并通过受支持交接处理修复。
本补充不伪造撤回事件，也不改写原报告/事件历史。

## R1 / P1：自动接管可覆盖本地输入包

审查提交 fc26fe61246626c7991ffb059233616c6f5eeb8a，来源仍为
FD-082-000004-implementation-ready，session=fd082-reviewer-20261018-6b2e94。

`src/plugins/aiw-pkg/manager.py` 的 install 首先 read_package(package)，随后对
选中 target 开启 adopt_existing=True，却未排除输入包本身。若输入是
`DEST/demo-1.zip` 且包载荷 target 也为 `demo-1.zip`，该普通文件通过预检并被
atomic_bytes 覆盖。本次新增的自动接管将原来的拒绝路径变成允许写入。
尾部 package.read_bytes() 还会将已经覆盖的内容存入 current.zip，失去原始包副本。
这直接违反本地输入包保留的契约，不能因有历史快照就视为满足。

最小修复：发布前比较输入包真实路径与本次选中的安装目标，若重合则明确拒绝且不改输入。
需处理安装根别名，不放宽内部链接规则；纯文件名字符串比较不足。
本问题无需运行验证即可由控制流确认。未新增测试、构建、安装或网络命令。
