# aiw-cz Python runtime

`aiw cz` 使用 Python 插件入口。目标机器需要预装 Python；当前实现默认只使用
Python 标准库。若未来发布包包含第三方依赖，可在插件目录执行：

```text
pip install -r requirements.txt
```

LLM 回退顺序为：Codex CLI → Copilot CLI → OpenAI Responses HTTP → 交互式向导。
交互式且检测到可用图形界面时使用 GUI，否则使用 TUI。未配置语言时使用操作
系统语言，没有对应翻译时回退英语。
