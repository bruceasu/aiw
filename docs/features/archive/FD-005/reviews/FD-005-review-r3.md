# FD-005 独立评审 r3

- FD：`FD-005`
- 来源事件：`FD-005-000007-implementation-ready`，由 `codex-review-fd005-r3` 认领
- 评审范围：`61acf474abf222e87e6278ee21226692b309428..879a072`，以及当前未提交的 FD、实施报告和 CZ 使用文档差异
- 结论：`CHANGES_REQUESTED`

## 发现

1. **无效 OpenAI 响应可能中断回退。** `plugins/aiw-cz/cz_openai.py` 的 `generate` 对 HTTP 200 的 JSON 响应直接调用 `_text(body)`；`_text` 假定顶层是对象并调用 `response.get`。若响应体是合法 JSON `[]` 或 `null`，分别抛出 `AttributeError`，而 `plugins/aiw-cz/cz_llm.py` 的 `candidates` 仅捕获 `OpenAIUnavailable`、`ProviderUnavailable`、`ValueError` 和 `KeyError`。因此会直接退出，未进入 `aiw-cz.py` 的 TUI `wizard`。对象中的 `output: null` 也可能导致未捕获的 `TypeError`。这不满足 FD Acceptance 和 `openspec/specs/cz-python-runtime/spec.md` 中“无效候选时回退交互流程”的行为要求。请让非预期响应形状转成 `OpenAIUnavailable`，使现有回退分支处理；不要求运行 1.4 的 HTTP 测试。

## 已核对的证据

- 当前 FD 已明确记录用户免除 1.3/1.4 运行测试，旧评审报告对此提出的运行门槛不再适用；真实 CLI 版本与 HTTP 响应兼容性仍是未测风险。
- `cz_llm.candidates` 的默认顺序、显式 provider 的单项列表、校验异常捕获和耗尽后返回 `None`，与 1.5 的静态证据相符；入口收到 `None` 后进入 `wizard`。上述未捕获异常是这一链路的例外。
- 用户提供的 TUI 输出证明预览和本地提交 `879a072`；编辑、取消及无 push 路径由 `cz_ui.py` 静态支持。
- 用户提供的 `build cz`、`build plugins` 输出和实施报告记录了 Python 发布安装。`build.bat` 的 `/E` 与 `/XF cz.toml .cz.toml` 静态支持保留已有配置，但目标目录无配置样本，重新安装保留行为仍未经运行验证。
- 评审阅读了 FD、REQ00004 Issue 计划、实施报告、两份 CZ 稳定规格、相关 provider/TUI/入口/安装代码及当前差异。未发现其他必须阻止本次评审的缺口。

## 命令和限制

- 已运行：`python plugins/aiw-fd.py --help`、`claim --help`、`claim FD-005 FD-005-000007-implementation-ready --session codex-review-fd005-r3`、`emit --help`、`git status --short`、`git diff --stat`、定向 `git diff`、`git log --oneline -8`、`rg` 和 `Get-Content` 静态读取。
- `python plugins/aiw-fd.py show FD-005` 因 Windows CP932 输出编码失败；该命令未提供评审证据，事件仍已按指定 ID 成功认领。
- 未运行测试、编译、构建、CLI 调用、HTTP 请求、TUI、安装或网络访问。用户已免除 1.3/1.4 运行测试；其余运行风险如上记录。
