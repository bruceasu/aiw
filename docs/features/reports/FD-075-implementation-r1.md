# FD-075 实现报告

<!-- aiw-data: FD-075-implementation-r1.json -->

Worker fd075-worker-20261017-2e49；来源FD-075-000003-design-ready；基线f1415a8，主要实现5863954，证据提交另含最终规格标题修正。

ask-asu保留一个入口，默认读取现有状态、推荐下一步，不执行建议、不写文件或派发角色。可选咨询仅针对具体第二意见问题且已有实际授权，选择一次只读宿主Agent或固定provider的单轮aiw ask，不叠加渠道、不递归、不自动重试/切换模型。

新增consultation引用，明确问题/约束/必要证据/版本/授权/未知的输入包，渠道能力和费用/传输/历史副作用，建议与事实/人类决定/独立fd-review之间的区别。现有预算和diagnosis-only不因咨询重置；已有诊断引用原分析，不通过第二意见重复受限工作。

代码静态依据：ask command的parse仅支持json/system-prompt/system-prompt-file/allow-path与chat/resume，单轮使用cfg provider/model；llm.go在auto下可逐后端尝试，故本模式要求已知固定provider。persist尝试保存到~/.aiw/ask，失败可不影响答案返回，不能断言历史已保存。CLI目录限制依后端，非普遍allowlist；Copilot ask当前路径不使用独立SystemPrompt，已记录能力限制，不以flag冒充隔离、不在本FD改后端。

同步源/项目入口、README、UI简述、workflow参考与新consultation引用；补齐声明的reviewed-skill-contract项目物化副本；目录和stable FD规格更新。不改aiw ask CLI/API、配置或运行行为。

一批只读文件/链接/Git diff检查退出0：13个指令/metadata文件，6组源/物化副本一致，34本地链接有效，原角色交接Requirement只保留一处，未涉及可执行代码。静态追踪仅导航、具体授权的单渠道咨询、auto范围不明拒绝调用、材料/能力缺失、咨询失败结束及预算保留情景；未执行情景或真实咨询。

实际执行：定向Get-Content/rg、受管FD与专属worktree操作、范围内编辑/复制/Git提交及一批静态检查。未运行aiw ask、咨询子Agent、测试、编译（纯文档）、模型业务请求、网络、安装、最终构建、格式化/lint/vet或验证脚本。正常FD独立Reviewer由宿主另外派发，不属于本Skill的咨询运行。

剩余限制：真实模型遵从/后端调用范围尚未运行验证；供应商SDK内部重试次数不能由一次CLI调用保证；全局外部Skill副本未同步。下一步独立实现评审，符合门槛后正常本地合并归档。
