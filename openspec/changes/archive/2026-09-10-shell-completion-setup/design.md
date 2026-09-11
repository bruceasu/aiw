## Context

AIW 已能用 `aiw completion <shell>` 生成 PowerShell、Bash、Zsh 和 Fish 的补全脚本，但 `setup-project` 目前只配置项目文档。CLI 子进程不能修改父 Shell 的当前会话。

## Goals / Non-Goals

**Goals:** 检测并幂等安装持久补全配置；缺失时提示当前会话加载命令。

**Non-Goals:** 不直接修改父 Shell 会话；不覆盖用户既有 profile/rc 内容；不重复追加配置块。

## Decisions

- 使用成对的 AIW 注释标记包围持久配置，并仅在完整标记存在时视为已安装。
- 根据当前环境检测 PowerShell、Bash、Zsh 或 Fish；无法识别时报告跳过原因而不猜测写入位置。
- 用户配置写入后输出对应即时加载命令；该命令由用户在父 Shell 中执行。

## Risks / Trade-offs

- [Shell 无法可靠识别] → 不写入任何配置并清晰提示。
- [profile 不可写] → 报告路径与原始错误，不回退到其他 Shell 配置。
