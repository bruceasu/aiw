# E06 知识审阅、注入与历史额度聚焦验证

Task workflow-automation；wi-0012 / 2.2；实现来源 E06 / wi-0008。状态：已新增三项测试，尚未运行。

依据 project-knowledge/spec.md 的 SW22–SW24、SW26/SW27 及 R3。新增 knowledge_durable_windows_test.go，复用 E05 临时 Store fixture。能力、容量、盘点和授权是测试替身；注入计数器使用确定性字节长度替身，只验证选择/预算分支，不证明真实模型 token 计量。

| 用例 | 覆盖 |
| --- | --- |
| TestKnowledgeVersionReviewAndExplicitRecheck | 人工原文完整保存且默认候选、未验证身份标记；旧 revision 拒绝；编辑新候选不继承确认；关联文件变化标待复核，恢复原文仍须人工复核；拒绝不可改来源绕过；无替代也能废弃 |
| TestKnowledgeInjectionDeterministicAndPreservesRequiredInput | 同输入同顺序；已确认优于高优先级候选；候选次要参考，拒绝/废弃不注入；可选超限留原因、保全必需输入；必需自身超限返回错误，未知计量降级 |
| TestKnowledgeHistoryLimitsSurviveRestart | 两个子场景分别达到 64 来源或 4 MiB；重开 Store、拆批均不刷新累计；重复来源复用，超额提交不改变原账 |

执行一次，预计 1–3 分钟：

```powershell
python C:\Users\svictor\workspace\tools\aiw\.wt\workflow-automation\openspec\changes\workflow-automation\run-e06-knowledge-terminal.py
```

固定命令：

```powershell
go test ./internal/workflow -run '^(TestKnowledgeVersionReviewAndExplicitRecheck|TestKnowledgeInjectionDeterministicAndPreservesRequiredInput|TestKnowledgeHistoryLimitsSurviveRestart)$' -count=1 -timeout=60s -vet=off -json
```

GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local；仅运行本组三项，不自动重试。保留计划/输入/环境/工具链/原始结果。需要本组显式确认或用户执行，不复用 E05 测试授权；不生成产品 grant，不运行模型/网络/后台宿主，不迁移实际项目。

## TODO 与 Verification

2026-09-25 离线执行 `python scripts/compile.py`，退出 0；仅编译生产程序，临时产物已清理，不覆盖新测试源码。未执行 E06 测试、模型、网络或 Git 写入；沿用此前 aiw patch 内部 Git apply 不可用时的直接补丁回退。

- [x] 编写三项契约测试与单次取证脚本。
- [ ] 取得并核对本组运行证据。

%% REMAINING: 异步提取、部分覆盖草稿、条目生成版本、真实模型计量、跨 Task 选择、完整一般规则和注入数量/字节边界仍待其他验证；不宣称完整 E06 或 AC20–AC27 / AX05 已通过。
