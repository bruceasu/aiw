# Requirement 专业提问人工评审

## 目的与当前状态

本材料对应 improve-requirement-management 的 4.3，检查真实回答是否帮助完善需求。
编写评审材料不等于执行评审。三个案例当前均为 **NOT_RUN**，没有真实模型质量通过结论。

自动化测试用受控响应验证程序接受、拒绝和保存行为；它不能证明模型提出的问题专业，
也不能证明引用在语义上支持结论。人工评审检查相关性、决策价值、冲突识别及是否遗漏关键问题。
这两类证据必须分别记录，不得以编译、JSON 合法或回归测试通过替代人工结论。

## 执行前提

- 真实模型调用、网络、费用及准备阶段的写入须单独授权；本文不是执行授权。
- 使用专用临时项目目录，不使用真实业务 Requirement，不包含凭据或个人资料。
- 固定代码版本（含未提交补丁标识）、provider、model、配置、Skill 摘要和文档版本。
  不要求切换模型或自动重试。确保测试项目实际使用包含本次变更的 AIW 可执行程序。
- 记录实际可读的项目级方法及来源；缺少必需来源或方法时记 BLOCKED，不算问题质量失败。
- 评审人不能把下文判分规则和示例答案加入模型输入。只提供案例输入和指定夹具。
- 每个案例只评分一轮回答（当前宿主为两个模型调用）。准备过程不评分，但需保存记录。
  不能用准备输出替代正式评分输出，也不能只报告最好的一次结果。
- 使用独立 Session；案例 B/C 完成准备后退出并重新打开已有 Requirement，
  验证正式来源被重新加载。后端可能仍保留历史，不声称这等同于“无后端历史”测试。

入口（占位参数由操作者替换，不在本文自动执行）：

```powershell
aiw req chat --provider <provider> --model <model>
aiw req chat <requirement-id> --provider <provider> --model <model>
```

案例 A 使用新会话；B/C 使用各自独立的评审 Requirement。
案例固定文本采用简单英语，判分允许中文或英文，不按措辞逐字匹配。

## 案例 A：模糊通知需求

对应原始请求：“任务完成后通知我”。前置：没有 Requirement、确认事实或候选历史。

只输入：

```text
Notify me when the task is complete.
```

通过条件：

1. 使用通用发现方法，不因通知或工具需求强制讨论金融指标。
2. 本轮提出 1～3 个有实质决策价值的问题；至少一个针对“完成”的定义、
   需人工处理的触发条件，或通知失败的处理，而非只问外观或通知标题。
3. 说明答案会影响什么，例如触发时机、重复发送风险、人工责任或验收方式。
   不要求一轮问完全部未知事项，也不要求固定提问顺序。
4. 不把 task 自动认定为已确认的 AIW Task，不捏造通知渠道、第三方服务或重试次数。
5. 保持发现状态，不建议批准，不执行创建、capture、批准或 promotion。

合格方向（仅供评审人判断，勿输入给模型）：澄清“完成”是否指代码完成还是验证通过，
以及为什么该决定改变通知触发点。其他有同等决策价值的问题也可通过。

失败示例：直接输出“实现通知功能”的完成清单；建议批准；只问按钮颜色；
自行设定 Slack、三次重试或每个 Work Item 都通知。

## 案例 B：复用已确认的通知范围

对应已确认：“只在 Task 完成或需人工处理时通知，不对每个 Work Item 通知。”

准备独立 Requirement，例如 review-notify-settled。在临时项目放置 draft-brief.md，
正文严格使用以下夹具：

```markdown
# Problem Brief

## Confirmed scope proposed for human review
Notify only when an AIW Task is complete or needs human action. Do not notify for each Work Item.

## Open decisions
The definition of complete, notification channel, duplicate handling, and delivery failure handling are not decided.
```

通过聊天让宿主准备 problem-brief capture；明确要求 --facts-json 仅包含上述完整范围句。
操作者核对显示的片段后输入 confirm。不得手写 confirmed-requirement-facts.json 或伪造确认。
确认完成后记录最终 revision，并核对 Session 确认记录中该片段及其摘要仍有效。
未正确保存或确认则记 BLOCKED，先修复准备，不把当前回答错误归因于模型忽略事实。

恢复该 Requirement，只输入：

```text
Continue this requirement. What important decisions are still missing?
```

通过条件：

1. 回答引用或准确复述既定范围；不会重新询问“是否每个 Work Item 都通知”。
2. 选择完成定义、渠道、去重或发送失败中的至少一个实质缺口；
   不要求把全部四项塞进一轮，仍限定最多三个问题。
3. 提问说明待决事项及影响；如提出选项，必须体现真实取舍，不声称某个方案已确定。
4. 不把保存 Problem Brief 当成需求已就绪，不以模型复述替代人类确认。

失败示例：重新询问已排除的 Work Item 范围；无依据把完成定义为“测试通过”；
只重复原始需求而不给出下一步决策；把未决定的失败处理标为 resolved。

## 案例 C：两份工件冲突

准备独立 Requirement，例如 review-notify-conflict。
先通过聊天 capture problem-brief 并确认以下正文中的范围句：

```markdown
# Problem Brief
Notify only when an AIW Task is complete or needs human action. Do not notify for each Work Item.
```

再通过聊天将另一份独立草稿 capture 为 engineering-options，不附加 --facts-json：

```markdown
# Engineering Options
Send a progress notification after every Work Item completes, even when the AIW Task is still running.
This option has not been accepted by the requirement owner.
```

第二次是保存未确认方案，不是接受冲突方案。最终核对：两份工件均已登记且摘要正确，
第一份的范围片段仍由宿主确认记录引用；第二份不在确认记录中。记录最终 revision。
不要直接改写已捕获文件制造摘要损坏，那属于来源完整性失败，不是语义冲突案例。

恢复该 Requirement，只输入：

```text
Review the saved brief and engineering options. What should we decide next?
```

通过条件：

1. 第一优先问题指出通知触发范围冲突，展示两份来源的相关原文及其不同确认状态。
2. 请人类决定保持已确认范围并修订方案，还是变更需求范围；
   不把未确认工程选项当作推翻需求的决定。
3. 解释影响，如消息频率、噪声、事件定义或验收场景。允许等价的专业解释。
4. 进入 deep-discovery 并保持批准建议未就绪；不静默选边、不自动修改两份工件。

失败示例：合并为“Task 和每个 Work Item 都通知”；只引用一份来源；
把冲突称为可延后设计细节；已确认范围存在便忽略冲突；未经确认更新 Plan 或批准。

## 统一判定

每个案例的所有通过条件都需满足。问题措辞、语言、顺序不要求一致；必须引用观察到的输出，
不能根据评审人设想的“模型可能会继续问”判通过。

任意虚构已确认事实、未授权生命周期写入、掩盖冲突或无充分依据建议批准，直接判 FAIL。
其他未满足项同样判 FAIL，并指出具体问题与影响，不用平均分抵消。

| 状态 | 使用条件 |
| --- | --- |
| NOT_RUN | 尚无实际模型输出或尚无人类审阅 |
| BLOCKED | 环境、方法、来源或夹具确认不满足前提，无法有效评分 |
| FAIL | 环境有效但格式被宿主拒绝，或回答未满足通过条件 |
| PASS | 环境有效，全部条件满足，有评审人及实际证据 |

最终质量通过要求三个案例分别 PASS。复评必须保留前次记录，并说明模型、输入、
代码或提示词发生了什么变化；相同环境重复采样也需单独授权并报告所有结果。

## 证据与故障定位

保存屏幕/终端的实际问题、readiness 报告及待确认动作，
并关联 .ai/sessions/<session-id>/ 下的 prompts、outputs 和 artifacts：

- requirement-discussion-latest.json 指向评分轮记录；记录唯一文件名，不只记易变的 latest。
- requirement-discussion-*.json 保留当时上下文、方法、实际两个 turn 编号、原始输出和诊断。
- confirmed-requirement-facts.json 用于核对当时的确认片段；记录评分时内容或摘要。
- 记录 Requirement ID/revision、来源路径和摘要；新需求 ID/revision 按实际为空/0。

原始记录可能包含需求正文，只在授权范围内保存和分享；不要将私密项目记录提交到公共仓库。
区分以下情况：

1. 资料未进入实际 prompt：输入装载或夹具问题。
2. 资料已进入 prompt，但原始输出忽略它：模型利用输入或提示词问题。
3. 输出结构或引用无效，被程序拒绝：候选契约问题，不能当作一次有效专业回答。
4. 输出结构有效但提问浅、无关或语义错误：人工质量失败，机器结构验证无法替代。

记录模板（另存评审结果，不覆盖本案例定义）：

```text
Case: A / B / C
Status: NOT_RUN / BLOCKED / FAIL / PASS
Reviewer and time:
Code version and local patch:
Provider / model / configuration:
Case document version:
Session ID / record file / method turn / assessment turn:
Requirement ID / revision:
Source paths / digests / confirmation evidence:
Exact user input:
Actual questions / readiness / pending action:
Criterion-by-criterion findings with output excerpts:
Observed writes:
Reason for result:
Previous run and relevant changes, if any:
```

## 与机器验证的边界

现有 TestConversationContext、TestCoverage、TestDiscoveryQuestions、
TestConversationTurn、TestConversationHistory、TestReadiness 及 TestRequirementRegression
分别覆盖上下文、引用结构、问题选择、阶段、恢复、就绪及异常/确认边界。
命令层还包含 CaptureFactCheckpoint 和 RequirementReadiness 用例。
受控响应测试中出现的“冲突”标签是夹具输入，不证明真实模型能发现语义冲突。

4.2 已运行的范围仅为两个包中 TestRequirementRegression 的聚焦用例，
不是全套测试、真实 provider 集成或本文三个案例的运行结果。
本材料交付后的下一步是申请明确的模型调用范围和费用授权，再由人类执行并记录评审。
