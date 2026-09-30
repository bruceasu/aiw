# FD-005 独立评审报告

- FD：FD-005
- 来源事件：未发现有效的 Worker `implementation-ready` 移交；当前可见的 `FD-005-000002-design-requested` 是旧 Planner 请求，不能作为本次 Reviewer 移交。
- 评审基线：当前工作区相对 `HEAD` 的变更
- 评审方式：独立 Reviewer 会话进行静态检查
- 实施报告：未找到 `docs/features/reports/FD-005-implementation.md`

## 结论

`CHANGES_REQUESTED`。FD 保持 Pending Verification。未发出生命周期事件：没有有效的 Worker 移交事件可供引用；不伪造来源或越过角色流程。

## Findings

1. **高：OpenAI 请求失败不会按 FD 要求回退。** `plugins/aiw-cz/cz_openai.py` 抛出 `OpenAIUnavailable`；`plugins/aiw-cz/cz_llm.py` 的回退捕获范围不包含该异常，网络/API 错误会中断 provider 链。
2. **高：Codex/Copilot 的只读安全要求没有得到执行层保障。** `plugins/aiw-cz/cz_providers.py` 通过 `subprocess.run` 在当前仓库目录执行 CLI，未设置或强制只读权限语义。FD 要求检测和调用不得修改仓库；仅靠提示文本不能保证此项。
3. **中：Python 配置兼容不完整。** `plugins/aiw-cz/cz_config.py` 使用简化 TOML 解析，`#` 会截断引号字符串；未读取现有配置使用的 `[i18n].default_language` 和 `cz.llm_provider`。TypeScript 参考实现 `plugins/aiw-cz/src/config.ts` 读取这些字段，故迁移可能改变既有配置行为。
4. **高：验收勾选缺乏所需证据。** FD 将 1.1–1.8 全部标为完成，但自身 Verification 明确记载未运行发布/安装、CLI、GUI/TUI 及提交流程，且仍保留目标环境 CLI 与安装行为的 `NEEDS_INPUT`。没有独立实施报告或运行证据能证明这些验收项。

## 命令与检查

- 阅读 FD-005、Issue 计划、相关 CZ 稳定规格、Python/TypeScript 实现和工作区变更。
- `aiw fd show FD-005`
- `git status --short`（限定查看 FD-005 审查相关路径）
- `rg` 搜索 FD-005 事件、报告和索引引用
- 未运行测试、构建、CLI/provider、GUI/TUI、OpenAI 网络请求、安装或 Git 写操作。

## 剩余风险

插件发现优先级可找到 Python 实现，但没有安装/升级运行证据证明旧 JS 文件清理和用户配置保留。应由 Worker 修复上述问题并补齐证据，再由有效的独立 Reviewer 移交重新评审。
