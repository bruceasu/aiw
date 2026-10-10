# AIW Say 发行目录

在仓库根目录执行 `python build.py say`，会在 `dist/plugins/aiw-say/`
生成 Windows amd64 的 `aiw-say.exe`、Linux amd64 的 `aiw-say`、
`aiw.toml.example` 和 `profiles/` 样例。源码位于 `src/cmd/aiw-say/`
与 `src/internal/say/`；
配置样例来源为 `src/programs/aiw-say/`。

`python build.py say` 只构建。`python build.py plugins` 构建并安装插件到
`C:\green\aiw\plugins\aiw-say\`，`python build.py all` 也包含此流程。
安装不会复制或覆盖实际 `aiw.toml`，也不会修改用户 profile 目录。

首次使用时，将安装目录的 `aiw.toml.example` 复制为同目录的 `aiw.toml`，
设置 `[say.llm].model`，并在环境中设置 `OPENAI_API_KEY`。
默认配置从 Say 可执行文件所在目录读取；其他位置使用 `--config`。
profile 样例需要按需手动复制到用户配置目录。

详细选项和 profile 路径见仓库中的 `src/programs/aiw-say/README.md`。

剪贴板与 GUI 是可选功能：Windows 使用系统剪贴板；Wayland 使用
`wl-paste`/`wl-copy`；X11 使用 `xclip` 或 `xsel`。WSL 需启用 Windows
interop 并能找到 `powershell.exe` 或 `pwsh.exe`。只有运行相应功能时才需要
这些工具。`--dialog zenity` 需要安装 Zenity 和可用的图形会话；AHK v2
快捷键脚本与 PowerShell UTF-8 辅助脚本见 `src/programs/aiw-say/`。
