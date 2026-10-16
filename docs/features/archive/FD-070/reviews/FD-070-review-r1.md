# FD-070 独立实现评审，第 1 轮

<!-- aiw-data: FD-070-review-r1.json -->

结论：**changes-requested**。有两项实现与当前验收不一致；不修改实现，不交付或归档。

- Reviewer：`fd070-reviewer-20261017-71c3`，独立于 Worker。
- 来源：`FD-070-000004-implementation-ready`。
- 比较基线：`8f4d6bb`；评审 HEAD：`e7acefd`；Worker 实现修订：`bd1aa3b`。

## 发现

1. **R1，High，Copilot 丢失所选 Skill 和阶段协议。** `src/internal/ai/cli.go` 的 `args` 在 Copilot 分支最终传入 `--prompt r.Prompt`；只有 Codex 的 stdin 使用 `cliPrompt(request)` 合并系统指令。`assist/model.go:step` 把 Skill、模式适配和唯一的 JSON 输出协议放在 `SystemPrompt`／`Instructions`，而冻结 `Prompt` 仅含阶段、资料、记忆和用户输入；`ExecuteFrozenTurn` 不再自动拼接 Instructions。因此四个命令选择 `--provider copilot` 时，模型实际收不到 Skill 和协议，无法按约定完成澄清、teach 或结构化产物。当前 FD 要求沿用提供商路由并冻结技能方法。最小修正：只对辅助阶段将完整系统指令交给 Copilot，保留旧阶段兼容性；若后端不支持该能力，则在模型执行前明确拒绝，不能默默降级。

2. **R2，Medium，EOF 不会取消正在进行的模型请求。** `src/internal/commands/assist/command.go:inputStream` 在 EOF 时只关闭 `lines`，`Dispatch` 的 context 仅注册 `os.Interrupt`。`step` 同步等待 provider 时不会读取关闭的 channel，因而 EOF 无法传播取消；它只会在模型结束后退出下一次读取。当前 FD Acceptance 明确包括输入或模型期间 EOF／Ctrl+C 可取消，使用指南也宣称 EOF 取消当前模型请求。应使活动模型请求收到明确 EOF 取消通知，并保留结果登记和无重放边界；或由 Planner 明确协调并记录 EOF 与 Ctrl+C 的不同语义，修订验收和文档后再评审。当前实现不能宣称两者等价。

## 已检查的证据

检查了 FD、Worker 双格式报告、辅助能力 stable spec、使用指南、四个 assist 源文件、Session 冻结执行及结果提交调用路径、CLI／OpenAI／Gemini／兼容接口的系统指令接线、主命令／帮助／补全实际 diff，以及相关 Skill 方法契约。

静态上可支持：任务与课程分开存储；新讨论先撤销 ready；显式 `/run`；资料快照和预算；安全单文件名及整组校验；独立外层锁；pending 与 Session 轮次不一致时阻止模型重放；恢复仅使用保存结果，部分文件相同时补写；失败历史复用；有限窗口与显式 recall。上述是代码路径判断，不是运行结果。未发现无关依赖或测试改动。

Worker 报告的两次离线 compile-only 成功作为已有证据引用；Reviewer 没有重跑，也没有独立证明模型、终端信号或 Windows 文件系统的真实行为。

## 实际命令及限制

执行 `Get-Content -Encoding UTF8`、`rg`、`git diff 8f4d6bb e7acefd` 的定向静态读取；`aiw fd show` 与精确事件 claim。若干探查文件名不存在，已从实际调用路径定位，未执行相关程序。随后仅写评审／阻断证据、更新 FD 并提交和发出受管 Reviewer 事件。

未运行测试、编译、模型、联网、安装、最终构建、formatter、lint、vet 或验证脚本。未修改产品实现。CLI 只读能力仍依赖后端；命令行长度、取消和恢复的实际平台行为仍未运行验证，不是本轮额外测试门槛。

阻断记录：`../reports/FD-070-blocker-review-r1.md`，由 Worker 修复 R1、协调 R2 后提交新实现证据并重新独立评审。
