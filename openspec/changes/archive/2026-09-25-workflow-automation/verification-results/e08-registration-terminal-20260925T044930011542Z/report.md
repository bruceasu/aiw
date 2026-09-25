# E08 登记首跑失败与修复

用户终端退出 1：TestVerifierHistoricalAcceptanceDoesNotInventSnapshot 通过；TestVerifierRegistrationReusesFrozenSourceAfterRestart 在最终开发状态比较失败。原始日志、计划及输入哈希已核对且一致，changed_inputs 为空；见 [run.json](run.json)、[原始事件](stdout.jsonl)、[输入](inputs.json)。

根因：测试辅助函数主动修改 WorkItems[0].Title 后，断言仍比较修改前的 WorkItems，误把测试自身的标题变化归因于登记。此前重启 revision 不变及 job 去重断言已执行至通过。

修复只涉及测试：在标题修改后、再次登记前加载 beforeRegistration；最后比较完整状态，确保登记没有新增变更。保留原始 job/snapshot 和重启去重断言，不修改生产逻辑。失败证据不覆盖、不删除。

修复后原两项尚未重跑；不得把此轮标记为通过。完整接受/Git 快照封存和宿主仍待证据。

修复后离线生产 Go 编译 `python scripts/compile.py` 退出 0；该命令不编译测试源码，不能替代重跑结果。

后续：用户已在修复后重跑原两项，全部通过，见 [当前结果](../e08-registration-terminal-20260925T045038918158Z/report.md)。上文待重跑说明保留为当时状态。
