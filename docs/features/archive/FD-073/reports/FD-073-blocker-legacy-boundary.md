# FD-073 旧块迁移边界阻断

<!-- aiw-data: FD-073-blocker-legacy-boundary.json -->

来源FD-073-000005-changes-requested，Worker修复阶段。独立Reviewer静态确认R1：新source边界和legacy边界分别校验时，显式旧块迁移可能吞掉内部其它FD块。原因是没有跨两种标记统一检查嵌套与交叉，非运行测试结果。

Worker现已合并边界流检查并同步项目副本，源/项目内存编译通过；未运行测试、helper、模型或网络。原状态为awaiting-independent-review，原评审与实现未通过事实保留。不需要用户决定新的需求或授权远程操作。

2026-10-17 独立Reviewer r2在HEAD 96caf9e静态复核修复f01b62d：嵌套和交叉均在替换前拒绝，相邻独立块及单一旧块显式迁移仍可成立。R1解除，当前resolved；见reviews/FD-073-review-r2.md/.json。没有把静态推导表述为运行用例通过。

下一步Reviewer复核反例及正常相邻块。可复用改进：旧格式迁移必须检查与新格式的整体边界关系，不能只验证各自完整。本FD直接修复helper，不扩展无关全局规则。
