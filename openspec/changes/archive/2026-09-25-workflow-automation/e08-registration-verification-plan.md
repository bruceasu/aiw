# E08 接受来源登记验证

Task workflow-automation / wi-0012 / 2.2；修复测试比较基线后，原两项全部 passed，见 [当前结果](verification-results/e08-registration-terminal-20260925T045038918158Z/report.md)；保留 [首跑失败与修复](verification-results/e08-registration-terminal-20260925T044930011542Z/report.md)。

- TestVerifierRegistrationReusesFrozenSourceAfterRestart：真实临时 Store 首次登记、重启和后续来源变化复用固定 job/snapshot，不重开工作或 Gate。
- TestVerifierHistoricalAcceptanceDoesNotInventSnapshot：历史接受缺快照时报告明确缺口，整个状态不变，不用当前代码补造快照。

fixture 构造接受事实与不可变快照，不调用真实接受流程或 Git diff 捕获；摘要内容为测试数据，不作为真实文件证据。复用已有测试资源配置，没有模型请求、网络、Git 写入或生产迁移。

run-e08-registration-terminal.py 固定上述两项、60 秒上限、离线 Go 依赖设置；保留计划、输入、实际输出，不自动重试。

## TODO 与 Verification

- [x] 新增两项及单次取证入口。
- [x] 离线生产 Go 编译 `python scripts/compile.py` 退出 0；不编译测试源码，测试结果待取证。
- [x] 用户终端重跑原两项全部通过，输入和日志哈希已核对。

%% REMAINING: 真实接受时需求/diff/receipt 封存、完整宿主与联合启用仍待证据，不据此完成 2.2/3.1/3.2。
