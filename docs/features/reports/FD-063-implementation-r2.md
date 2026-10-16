# FD-063 阶段派发边界修复

<!-- aiw-data: FD-063-implementation-r2.json -->

- Worker：fd063-worker-20261017-f6a4
- 来源事件：FD-063-000005-changes-requested
- 修复依据：reviews/FD-063-review-r1.md/.json，R1 及主工作区 blocker 提交建议。

已在 implement 与 fd-workflow 的事件发送、runner 调用和 managed review 之前增加明确的阶段/范围判断：implementation-only、no-review、未授权审查的部分工作项请求只保存进度与证据，不派发 Reviewer；no-merge 可以允许审查，但不能交付。

进一步明确 implementation-ready 必须在整个 FD 实现范围完成后发送。即便用户单独授权部分范围的审查，也应保持部分审查边界，不能把未完成 FD 推入整体验收或宣称 ready。同步 Worker 角色、canonical 共享契约、31 个生成副本和稳定规格。

主工作区 blocker 反馈提交段改为继承共享契约已授权范围和明确用户限制，不再要求无条件额外授权。源与对应项目副本同步修改。未改 CLI/runner 实现、权限机制或其他模块。

本轮以精确补丁和既有静态路径证据修正两个文档问题，最终一致性由新的独立审查检查。纯 Markdown/YAML，无编译目标；未重复运行编译、测试、验证脚本、网络或构建。原报告的真实多会话/失败路径未运行、用户级安装未更新等风险仍保留。
