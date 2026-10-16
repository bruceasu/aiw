# FD-070 独立设计反馈 r1

<!-- aiw-data: FD-070-design-r1.json -->

- 结论：**changes-needed**。整体方案可实施；下列两项持久化与预算决策补明后可进入设计就绪。
- 来源：`FD-070-000002-design-requested`；审阅 FD Revision 2。
- 本报告仅为设计反馈，不是实现审查或 verification-passed。

## 已确认的设计价值

四种辅助命令共用 Session、模式专属 Skill 和受限 host 成果写入，能满足独立任务与多课程教学，不建立 Issue/FD 的第二条运行时流程。显式 `/run`、就绪摘要失效、独立成果目录和不覆盖原稿的选择清楚。research 首版仅利用显式本地资料、prototype 不自动运行，与成本边界一致，也明确缩减了原 Skill 中后台调查与工程分支的默认动作。

现有 `ExecuteFrozenTurn` 支持冻结输入、指定下一轮、强制新 provider thread、只读标记、原始输出与 usage 保存。`Store.WriteArtifact` 可原子替换单个 sidecar。因此不需要为了这次能力修改公共 Session JSON 或引入新依赖。

## 需要解决的发现

### D1：明确 Session、辅助历史与成果提交之间的中断一致性

**影响：高；设计就绪前解决。**

依据：`src/internal/session/execute.go` 中 executeTurn 先写请求，再置 running，provider 返回后写原始输出并更新 LastTurn。`src/internal/session/store.go` 的 WriteArtifact 只对单文件原子替换，不把 Session 状态、辅助历史和成果目录组成一次事务。

当前 FD 说明“原子替换”“可恢复”，但未定义这几个写入之间失败时的可见状态。例如：provider 已完成且 LastTurn 已推进，auxiliary.json 写入失败；或部分成果已落盘，最终结果登记失败。如果简单恢复旧 sidecar 再次调用，将重复执行并产生第二份成果；若直接沿用旧 ready，用户也可能运行未对应最新输入的摘要。硬杀遗留 running 的门禁只覆盖其中一部分情况。

建议在辅助包内部定义版本化 sidecar 与提交协议，不修改公共 Session：

1. 派发前持久化 pending 记录，包含下一 Session turn、阶段、输入/brief digest；用户输入先落盘并使旧 ready 失效，持久化失败则不调用模型。
2. 模型结果、解析结果、成果文件与最终提交状态使用同一 turn 关联；成果先完整校验，再写独占目录，成功登记后才宣布成功。部分目录明确为未提交成果，不能作为课程已完成内容。
3. 恢复比较 pending、Session LastTurn/状态及已有原始输出。身份一致时可纯本地解析/补登记；不具备安全恢复证据时拒绝继续并说明现场。两种方式均不得自动重新调用模型、覆盖成果或静默回退历史。
4. 文档明确实际采用的恢复方式；至少覆盖“Session已完成但sidecar未提交”和“部分成果写入”两个静态追踪案例。

这是需要确定的恢复合同；不要求增加后台服务、自动重试或通用事务框架。

### D2：为长期课程历史和冻结 Skill 指定有限上下文策略

**影响：中；设计就绪前解决。**

依据：FD 已限制 --source 与生成文件，但没有对完整请求中的课程历史、累计课件、Skill 引用、用户单次输入和模型响应设定范围。每次强制新 provider thread，意味着恢复的教学上下文必须由 host 明确组成，不能依赖 provider 的线程记忆。

如果每次发送所有历史，多课程中的任意长期课程最终都会超过预算；如果仅保留最近聊天，已确认目标、用户偏好和学习误区又可能消失，与“无需重新交代”冲突。

建议明确：

- 完整记录保留在本地；模型上下文与磁盘历史分开管理。
- 每轮固定包含目标、来源与 Skill 快照、当前 brief 和明确保存的学习进展，再选择有界近期对话；旧课件不全部注入。
- 对用户输入、Skill/引用总量、历史选取和总请求设有限上限；不额外调用模型做隐藏压缩。
- 无法装入必需上下文时不派发，明确提示缩小任务或开启后续课程；截取历史时说明范围，避免把未看到的历史声称为已复习。
- 记录实际注入的内容范围或 digest，使恢复轮次可追溯。数字阈值可以由实现采用集中常量，但上述超限行为应先在 FD 确定。

## 非阻塞实现提醒

- 安全单文件名需覆盖 Windows：除分隔符和控制字符，还应拒绝冒号/备用数据流、设备名、尾部点或空格，以及大小写折叠后的重复名。仅 filepath.Base 检查不足。
- 辅助进程锁不能占用 Session Update 使用的 `.ai/locks/<id>.lock`；使用独立名称，避免自身死锁。泛用 Session 命令并不遵守新增外层锁，运行前仍需核对 Session 状态与期望 turn。
- Codex ReadOnly 分支在无 InvocationObserver 时才应用只读 sandbox；本功能不要附加会使 provider 走 workspace-write 分支的 observer。Copilot ReadOnly 只是取消 allow-all-tools，不能宣称完整工具或网络隔离。FD 已承认这一限制，帮助和实现应保留。
- 只读取已声明 source 是 host 上下文规则，不等于现有 CLI provider 在项目目录下不能自行读取其他文件。文档不得把来源选择范围描述为全面文件系统隔离。
- `/run` 前须确实展示本次 ready brief；`/brief` 不应触发模型。teach 问答作为教学轮次保存，不能仅用通用“需求澄清”模板持续追问而没有教学。

## 静态证据与范围

读取 FD、planning 方法、Session execute/store/location/lifecycle/backend、AI CLI 参数映射、四 Skill、ai-routing 与 command-help-consistency 相关定义。执行三批本地只读文件读取和搜索；未执行应用、测试、编译、安装、网络或 provider 调用。一个批次中的 PowerShell rg 表达式转义失败，未重跑；同批文件读取结果仍用于本次反馈。未修改 FD 或角色事件，未实现代码。

## 建议交接

Planner 在 FD 补明 D1、D2 的有限实现选择，并将静态追踪场景映射到 1.2、1.3、1.4 和 1.6。无需扩大范围或引入额外运行时验证；本报告不能替代后续独立实现评审。

host 后续提出为 FrozenTurn 增加可选内部 SystemPrompt 字段，并仅由辅助调用赋值。该调整与现有 AI Request 的 SystemPrompt 能力相容，可作为本 FD 的最小适配：默认空值必须保持原有调用行为，Session JSON 不变；冻结记录包含实际阶段指令，source 原文仍仅处于数据区。executeTurn 当前 frozen 分支直接采用 frozen.Prompt，不能假定 Instructions 会自动拼入最终模型提示。此项为设计建议检查，未审阅新增代码。
