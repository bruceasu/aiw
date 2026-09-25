# E06 提取结果、部分草稿与版本验证

Task workflow-automation；wi-0012 / 2.2；实现来源 E06。状态：测试及脚本已准备，尚未运行。

依据 project-knowledge/spec.md 的 SW20/SW21/SW23，新增 knowledge_generation_test.go，直接调用真实语义校验、覆盖计算和发布转换。使用内存状态和固定引用，不读写生产 Store，不调用模型/网络；不证明后台登记、宿主调度或真实来源工件完整性。

| 用例 | 覆盖 |
| --- | --- |
| TestKnowledgeExtractionRequiresExplicitNoNewEvidence | 六个子场景区分有依据的显式空结果与空响应、缺字段、null、无理由、未知字段；合法无新增记录发布但不生成规则 |
| TestKnowledgePartialDraftPreservesCoverageAndHistory | success/running/failed/missing 全部保留并确定排序；部分草稿带缺口，覆盖补齐产生新草稿，旧草稿不可变，重放不重复发布 |
| TestKnowledgeGeneratedVersionsPreservePriorReview | 未变已接受内容保留原确认；正文或接受引用改变生成新候选，不继承人工确认，不改旧审阅历史 |

正常终端一次执行，预计 1–3 分钟：

```powershell
python C:\Users\svictor\workspace\tools\aiw\.wt\workflow-automation\openspec\changes\workflow-automation\run-e06-generation-terminal.py
```

固定命令：

```powershell
go test ./internal/workflow -run '^(TestKnowledgeExtractionRequiresExplicitNoNewEvidence|TestKnowledgePartialDraftPreservesCoverageAndHistory|TestKnowledgeGeneratedVersionsPreservePriorReview)$' -count=1 -timeout=60s -vet=off -json
```

GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local，保存独立计划/输入/环境/工具链/输出证据，不自动重试。本组需单独显式执行；不生成产品 grant 或解除 Gate。

## TODO 与 Verification

2026-09-25 离线执行 `python scripts/compile.py`，退出 0，仅编译生产程序并清理临时产物，不覆盖新增测试源码。测试和取证脚本仅静态核对，未执行本组三项、网络、模型、最终制品构建或 Git 写入。

- [x] 三项测试及单次取证脚本已准备。
- [ ] 取得本组运行证据。

%% REMAINING: 接受来源的异步登记、重启登记去重、真实 Store 草稿持久发布、宿主派发、实际模型和完整 AC20/AC21/AX05 仍未由本组覆盖。人工确认在版本测试中作为已有状态构造，真实审阅入口由此前 E06 审阅测试覆盖。
