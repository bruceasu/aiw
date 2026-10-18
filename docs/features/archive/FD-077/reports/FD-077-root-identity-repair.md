# FD-077 根身份边界修复报告

<!-- aiw-data: FD-077-root-identity-repair.json -->

- Worker：fd077-host-20261018-53b7ac。
- 来源事件：FD-077-000005-changes-requested。
- 对应发现：reviews/FD-077-implementation-review.md 的 R1/P1。

原实现丢弃草稿 family/root 的 Lstat 身份，OpenRoot 后才建立快照，可能把被替换的根当成合法来源。修复保留预检目录身份，并增加 checkStorage：打开后和清理前均以 Lstat 核对两层目录非链接、身份不变，且固定根句柄与原始根 SameFile。任何不匹配降级为保留源及警告，不重建可信快照。其他清理顺序、隔离恢复和输出契约不变。

实际验证：定向静态检查修复 diff；源修改后，以完全相同的离线环境 GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local 再执行 python scripts/compile.py，退出码 0。此前完整实现的 compile-only 亦为 0；本次重编译由实际源修复触发，不是重复未变命令。

未运行测试、删除/恢复/链接竞争实验、格式化、vet、最终构建或网络；不宣称实际竞争窗口已运行验证。原实现报告继续作为全范围证据，本报告补充修复及当前编译结果。等待独立 Reviewer 聚焦复核 R1 与相关 diff，不能沿用第一次 changes-requested 作为通过。
