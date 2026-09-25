## Why

AIW 的命令帮助、Shell 补全和 README 已经落后于实际命令分派代码，尤其是 workflow 相关命令存在遗漏或参数说明不完整。用户因此无法仅依靠帮助或 README 发现可用操作，且不同入口可能给出相互矛盾的命令表面。

## What Changes

- 对照命令分派、参数校验和实际支持的选项，补齐并校正顶层、Task 和 workflow 帮助文本。
- 补齐 workflow 的命令补全集合及其参数提示所依赖的命令名。
- 更新 README 中的命令摘要、示例和 workflow 说明，使其与源码支持的命令和选项一致。
- 增加静态/行为检查，防止新增公开命令再次只更新源码而遗漏帮助或 README。
- 不改变任何命令的执行语义、参数兼容性或持久化行为。

## Capabilities

### New Capabilities

- `command-help-consistency`: 保证公开 CLI 命令、帮助、补全和 README 的命令表面一致。

### Modified Capabilities

无。现有 CLI 运行时能力不变，本 change 只补齐其用户可见说明和一致性检查。

## Impact

影响 CLI 帮助渲染、Task/workflow 命令补全、README 命令文档及其相关测试。不会引入依赖、改变公共命令行为或修改稳定运行时数据结构。
