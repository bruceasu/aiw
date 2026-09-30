# aiw-cz Python runtime

`aiw cz` 使用 Python 插件入口。目标机器需要预装 Python；当前实现默认只使用
Python 标准库。若未来发布包包含第三方依赖，可在插件目录执行：

```text
pip install -r requirements.txt
```

LLM 回退顺序为：Codex CLI → Copilot CLI → OpenAI Responses HTTP → 交互式向导。
Copilot 仅在显式设置 `CZ_COPILOT_COMMAND` 为已安装 CLI 命令时调用；未配置时
直接跳过，不弹出安装提示。CLI 探测和调用均不读取交互式输入。
Codex 使用 `exec --sandbox read-only --json -`，提示通过 stdin 传入，并读取最终
消息；Copilot 使用 `--output-format text --prompt`。`--model` 会传给选中的 CLI。
交互式始终使用 TUI，不依赖图形界面。候选消息在 TUI 中预览，可输入
`e` 编辑、`n` 取消，或直接回车接受并自动执行本地 `git commit`；插件不会
执行 `git push`。未配置语言时使用操作系统语言，没有对应翻译时回退英语。
Windows 默认读取用户区域设置（`Get-Culture`）；显式 `--lang`、`CZ_LANGUAGE`
或 `[i18n].default_language` 优先。
