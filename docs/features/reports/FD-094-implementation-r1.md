# FD-094 R1 修正报告

<!-- aiw-data: FD-094-implementation-r1.json -->

来源：FD-094-000007-changes-requested。Worker session：worker-FD094-junction-R1-20261019。
独立审阅指出，叶到根处理 symlink 的 ../ 会在 junction 祖先解析前错误消去路径分量。
原审阅和初始实现报告保留，不覆盖或声称原审阅通过。

已改为从卷根逐组件遍历，resolved 始终表示已解析的物理前缀。发现链接后把原始
目标分量插入队列，不调用 Join/Clean 消去未解析目标中的 ..。绝对目标重置卷根，
相对目标从已解析父目录继续；.. 只操作当前物理前缀。所有链接共享 255 次计数。

R1 示例静态追踪：根内 sub junction 到 E:/outside/sub，再遇 run.py -> ../payload.py，
结果是 E:/outside/payload.py，最终 Rel 正确判为根外。目标内部 link/../file 同理，
不会提前按词法消除。卷相对/空目标、未知 irregular、非目录中间组件均失败。
原入口相对路径、物理 containment 和末端 Stat/目录检查保留。

相关源码变更后重新执行一次相同离线 scripts/compile.py，仅输出 NUL，exit 0。
已进行一次修正 diff 静态检查。没有再次运行已消耗的路径诊断，没有测试、构建、
安装、联网或修正后运行验证。已安装程序不变。

提交新受管 implementation-ready 后，请 Reviewer 对修正及完整 resolver 做聚焦复审；
首次 handoff 因 Work Item 1.2 错把独立 pass 写成 Worker 前置条件而被 CLI 拒绝。
Planner 保留 item ID，把它的实现完成标准修正为文档和编译证据；独立通过仍是
整个 FD 完成的必需 Gate，没有跳过 review。修正后重新交接。
