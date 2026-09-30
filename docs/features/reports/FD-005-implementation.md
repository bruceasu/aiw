# FD-005 实施补充报告

- 本轮范围：修复 `--retry` 的 LLM 误调用、CLI 参数绕过只读/输出契约、同层配置文件误合并，以及 FD Reviewer 事件因 CRLF 被拒绝认领的问题。
- 代码位置：`plugins/aiw-cz/aiw-cz.py`、`cz_providers.py`、`cz_config.py`、`plugins/aiw-fd.py`。
- 规格与记录：同步 Python CZ runtime、配置优先级和 FD workflow 稳定规格；FD-005 中缺少目标环境证据的 Work Items 保持未完成。
- 静态证据：`--retry` 分支位于候选生成分支之前；Codex/Copilot 调用不再接受任意 `CZ_*_ARGS` 覆盖；每一配置层仅选首个存在的文件；FD 事件生成和核对使用相同的 CRLF/LF 规范化摘要。
- 未验证：真实 Codex/Copilot CLI 参数与输出、OpenAI HTTP、TUI 提交、Python 发布安装及配置保留。未运行测试、完整构建或目标环境交互流程。
- 后续门槛：独立 Reviewer 需认领当前有效的 review-requested 事件并复核修复；目标环境运行证据补齐前，不得发出 `verification-passed` 或将 FD 关闭。
