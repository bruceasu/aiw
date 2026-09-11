## ADDED Requirements

### Requirement: 幂等持久补全安装
`aiw setup-project` SHALL 检测当前 Shell 的持久化启动配置中是否已存在 AIW 补全标记。标记不存在时，它 SHALL 追加一个生成 AIW 补全并加载的标记块；标记存在时，它 SHALL 不改写该配置。

#### Scenario: 首次配置当前 Shell
- **WHEN** 用户运行 `aiw setup-project` 且当前 Shell 的配置缺少 AIW 补全标记
- **THEN** 系统 SHALL 追加该标记块并报告已更新的配置路径

#### Scenario: 已配置当前 Shell
- **WHEN** 用户运行 `aiw setup-project` 且当前 Shell 的配置已有 AIW 补全标记
- **THEN** 系统 SHALL 保持该文件不变并报告补全已配置

### Requirement: 当前会话启用提示
`aiw setup-project` SHALL 在持久补全已配置或已安装后，输出与当前 Shell 对应的即时加载命令，并说明用户需要在当前 Shell 中执行该命令。

#### Scenario: PowerShell 会话
- **WHEN** 当前 Shell 是 PowerShell
- **THEN** 输出 SHALL 包含 `aiw completion powershell | Out-String | Invoke-Expression`

#### Scenario: 无法识别 Shell
- **WHEN** 系统无法可靠识别当前 Shell
- **THEN** 系统 SHALL 跳过持久化写入并说明原因
