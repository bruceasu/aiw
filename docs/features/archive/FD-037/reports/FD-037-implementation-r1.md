# FD-037 Worker 实现报告 r1

<!-- aiw-data: FD-037-implementation-r1.json -->

## 结果

完成 FD-037 的 1.1–1.9 实现项：`aiw say` 增加互斥的参数、stdin、剪贴板和 Zenity 输入源，只有翻译成功后才按 `--copy` 或图形模式回写剪贴板；新增 Windows Unicode API、Linux Wayland/X11、WSL Windows PowerShell 剪贴板适配器；接入 Zenity 输入/结果窗口和 Windows 原生 Zenity 查找；提供 AHK v2 J/E/B/T 快捷键及 UTF-8 PowerShell 辅助脚本，并更新使用说明。

本次 Worker 会话：`fd037-worker-20261010-21c74a`；来源事件：`FD-037-000003-design-ready`。

## 静态证据

- `cmd/aiw-say/run.go` 先校验输入源互斥和文本，再调用 FD-036 的 `say.Provider`；只有提供方返回成功后才执行剪贴板写入。普通 CLI 默认不写剪贴板，GUI 成功路径复制后显示结果。
- `internal/say/clipboard_windows.go` 以 Windows Unicode 文本格式读写系统剪贴板；`clipboard_unix.go` 在 Wayland/X11 选择成对读写工具，WSL 优先选择 Windows PowerShell 通道，并通过固定命令参数和 UTF-8 标准流传递文本。
- `internal/say/dialog.go` 使用固定 Zenity 参数和 stdin/stdout，不拼接用户文本到命令行；输入取消以错误结束，先于 provider 调用。无显示会话和缺失可执行文件有明确错误。
- `program/aiw-say/aiw-say-hotkeys.ahk` 和 `.ps1` 使用 base64 编码的 stdin 中转任意 UTF-8 文本，检查 CLI 退出码；回写前精确核对剪贴板是否仍为原文，避免覆盖已更换的内容。
- 文档列出 Wayland/X11 工具、WSL Windows interop/PowerShell、Windows/Linux/WSL Zenity、AHK v2 的前置条件与命令示例。

## 编译与未执行检查

- 执行 PowerShell 命令 `$env:GOPROXY='off'; $env:GOSUMDB='off'; go build -o NUL ./cmd/aiw-say`。首次因公共占位方法与 Windows 平台方法重复而失败；移除占位方法后按规则重跑同一命令，通过。通过后又静态修正 WSL PowerShell 的 `-ErrorAction Stop` 和 Zenity 退出码分类；未再次编译，编译结果早于这两项小型静态修正。
- 未运行测试、最终构建、格式化器或平台程序；未调用真实翻译 API。
- 当前 Windows 编译不包含 `!windows` 的 Linux/WSL 源文件，且编译结果早于最后两项源码修正。Windows/Linux/WSL 剪贴板 Unicode 往返、Zenity 取消和结果窗口、AHK/PowerShell 行为仍未由运行证据验证。

## 后续风险

FD-037 保留 `Test policy: Independent`。Tester 仍需按授权流程核验平台场景；本报告不把未运行场景记为通过。独立 Reviewer 的审查尚未开始。
