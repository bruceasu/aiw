# FD-005 实施补充报告

- 本轮范围：修复 `--retry` 的 LLM 误调用、CLI 参数绕过只读/输出契约、同层配置文件误合并，以及 FD Reviewer 事件因 CRLF 被拒绝认领的问题。
- 代码位置：`plugins/aiw-cz/aiw-cz.py`、`cz_providers.py`、`cz_config.py`、`plugins/aiw-fd.py`。
- 规格与记录：同步 Python CZ runtime、配置优先级和 FD workflow 稳定规格；FD-005 Work Items 现均已勾选，免测决定和残余风险另行记录。
- 静态证据：`--retry` 分支位于候选生成分支之前；Codex/Copilot 调用不再接受任意 `CZ_*_ARGS` 覆盖；每一配置层仅选首个存在的文件；FD 事件生成和核对使用相同的 CRLF/LF 规范化摘要。
- 此前未验证：真实 Codex/Copilot CLI 参数与输出、OpenAI HTTP、TUI 提交、Python 发布安装及配置保留。当时未运行测试、完整构建或目标环境交互流程；TUI 提交证据现已由用户补充，见下文。
- 后续门槛：独立 Reviewer 需对当前 FD、实现和验收决定重新评审；旧报告的运行测试要求不得替代用户后续明确的免测决定。未经 Reviewer `verification-passed` 不得将 FD 关闭。
- 1.6 补充证据（用户提供）：`go run cmd\aiw\main.go cz --no-llm` 进入 TUI，预览 `fix(cz): fix cz`，默认接受后生成本地提交 `879a072`。`cz_ui.py` 静态显示 `e` 编辑、`n` 取消以及仅调用 `git commit -F`；本会话未运行该交互命令。
- 1.7 补充证据（用户提供）：`build cz` 准备 Python 发布目录；`build plugins` 两段 ROBOCOPY 均无失败。只读核对 `c:\green\aiw\plugins\aiw-cz` 已有 Python 入口、模块、`locales/` 和 `requirements.txt`；插件发现优先选 `.py`，执行器调用 Python。旧 JavaScript 文件仍在安装目录但不是当前入口；重复 `mkdir` 产生“目录已存在”提示，不影响复制。目标目录没有 `cz.toml`/`.cz.toml` 样本，配置保留只有下述静态证据。本会话未运行构建或安装。
- 1.3/1.4 验证取舍：用户明确要求不测试。静态核对 CLI 命令发现、版本/help、只读调用、超时、输出解析和回退接线，以及 OpenAI 标准库 HTTP 请求、JSON Schema、超时和错误处理；未运行真实 CLI 或 HTTP 请求。两项保留已完成状态，实际版本和响应兼容性列为风险，不再列为待补测试。
- 1.5 逐项静态证据：`cz_llm.candidates` 默认 provider 列表为 `codex`、`copilot`、`openai`，显式选择时列表只含所选项；`run_cli`、`openai_generate` 和 `parse_candidates` 的 provider/校验异常由同一循环捕获并继续，耗尽后返回 `None`，入口据此进入 `wizard`。未运行真实 provider。
- 用户配置保留：`build.bat` 的 CZ 安装使用非镜像 ROBOCOPY `/E` 并排除 `cz.toml`、`.cz.toml`，静态支持保留已存在的配置。目标安装目录没有配置样本，因此没有重新安装的运行证据；此项列为残余风险，不再作为缺失的人类决定。
- r3 评审修复：`cz_openai._text` 现在拒绝非对象 JSON 顶层及非数组 `output`，跳过非数组 `content`，最终以 `OpenAIUnavailable` 表示无可用文本。`cz_llm.candidates` 已捕获该异常，返回 `None` 后入口进入 TUI。此项只有静态调用路径证据；按用户决定未运行 1.4 HTTP 测试。
