# FD-019 审查问题修复（修订 6）

<!-- aiw-data: FD-019-implementation-r6.json -->

已修复独立审查报告 `docs/features/reviews/FD-019-review-r5.md` 的 R1，等待独立复审。

Worker session：`codex-worker-FD019-20261004-r6-a71c`。
来源交接：`FD-019-000006-changes-requested`。

帮助和 README 明确 `key_hashes` 已废除，服务端 `principals[].keys` 与客户端
`AIW_AI_API_KEY` 使用同一个字符串，认证不计算摘要。移除 README 中当前配置步骤形式的
哈希 BAT 调用，工具仅保留为历史文件。旧摘要作为不透明 Key 使用时，两端必须发送相同摘要
字符串。客户端稳定规格补充同一配置指引要求，网关行为和认证实现没有改变。

本轮仅改 `plugins/aiw-ai/aiw-ai.mjs` 的帮助文本、同目录 README、
`openspec/specs/agent-proxy-client/spec.md` 及 FD TODO/Verification 和本报告。
八项历史工作编号和完成记录保留。本机安装副本未同步，后续由运营者更新。

静态证据：定向读取现行客户端和网关规格、旧 R1 报告及帮助/README；扫描现行文档，
其他网关说明已使用 keys，旧哈希核对保留为历史证据。最终定向 diff 由独立 Reviewer 核对。

实际执行：领取上述 Worker 交接；`node --check plugins/aiw-ai/aiw-ai.mjs` 退出码 0，
仅语法编译，不执行模块或保留产物。随后只调整帮助中的“已废除”文字，不改变 JavaScript 结构，
未重复编译。使用 CLI implementation-ready 交给独立 Reviewer，结果以收据为准。

未运行测试、客户端请求、哈希 BAT、编辑器、最终构建、网络请求或 Git 写操作。
External 策略保持；真实认证、模型调用、跨平台编辑器与信号行为仍待外部验证。
此前编译和本机同步属于历史证据，不能作为本轮已验证结果。
