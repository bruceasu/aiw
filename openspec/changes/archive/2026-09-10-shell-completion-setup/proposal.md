## Why

用户在首次设置项目后仍需手动配置 AIW Shell 补全。`setup-project` 应默认且幂等地安装持久补全配置，并说明如何在当前 Shell 会话中立即启用。

## What Changes

- 在 `aiw setup-project` 中检测当前 Shell 的 AIW 补全配置标记。
- 缺失时追加持久补全初始化；已存在时不改写配置。
- 输出当前 Shell 可复制执行的即时加载命令。

## Capabilities

### New Capabilities

- `shell-completion-setup`: 由 `setup-project` 幂等安装并提示 AIW Shell 补全。

### Modified Capabilities

无。

## Impact

- 已安装的 `aiw-setup-project.py` 插件。
- 用户的 Shell 启动配置；不改变 AIW 命令参数或补全生成器。
