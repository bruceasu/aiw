# AIW Say 发行目录

在仓库根目录执行 `build.bat say`，在本目录生成 Windows amd64 的
`aiw-say.exe`、Linux amd64 的 `aiw-say`、`aiw.toml.example` 和
`profiles/` 样例。源码位于 `cmd/aiw-say/` 与 `internal/say/`；
配置样例来源为 `program/aiw-say/`。

`build.bat say` 只构建。`build.bat plugins` 构建并安装插件到
`C:\green\aiw\plugins\aiw-say\`，`build.bat all` 也包含此流程。
安装不会复制或覆盖实际 `aiw.toml`，也不会修改用户 profile 目录。

首次使用时，将安装目录的 `aiw.toml.example` 复制为同目录的 `aiw.toml`，
设置 `[say.llm].model`，并在环境中设置 `OPENAI_API_KEY`。
默认配置从 Say 可执行文件所在目录读取；其他位置使用 `--config`。
profile 样例需要按需手动复制到用户配置目录。

详细选项和 profile 路径见仓库中的 `program/aiw-say/README.md`。
