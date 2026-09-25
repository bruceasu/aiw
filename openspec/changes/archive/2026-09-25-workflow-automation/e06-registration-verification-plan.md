# E06 接受来源登记与持久草稿验证

Task workflow-automation；wi-0012 / 2.2；实现来源 E06。状态：用户正常终端两项全部 passed、退出 0，见 [结果](verification-results/e06-registration-terminal-20260925T042234944157Z/report.md)。下方未运行说明为准备阶段历史。

依据 SW20/SW21，复用 E05 fixture，保存明确构造的接受事实及不可变工件，再使用真实 RegisterTaskKnowledge、Claim/Observe/Publish 入口。模型、能力、盘点及授权仍是测试替身，不执行真实宿主。

| 用例 | 断言 |
| --- | --- |
| TestKnowledgeAcceptedRegistrationDeduplicatesAfterRestart | 未接受进展不生成知识作业；接受来源登记提取及汇总；重开 Store 再登记不重复、不移动已消费游标；后续进展仍复用同一接受的提取；登记不直接派发模型 |
| TestKnowledgePartialDraftPublishesDurablyThenAdvances | 提取运行中也可保存部分草稿；有依据无新增提取完成后登记新汇总、持久保存新版本；重开 Store 后旧草稿保留、无虚构规则，现有接受/Gate/交付不变 |

正常终端执行一次，预计 1–3 分钟：

```powershell
python C:\Users\svictor\workspace\tools\aiw\.wt\workflow-automation\openspec\changes\workflow-automation\run-e06-registration-terminal.py
```

固定命令：

```powershell
go test ./internal/workflow -run '^(TestKnowledgeAcceptedRegistrationDeduplicatesAfterRestart|TestKnowledgePartialDraftPublishesDurablyThenAdvances)$' -count=1 -timeout=60s -vet=off -json
```

GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local，不自动重试。仅临时 Store，不改实际项目账本，不联网、调用模型或执行 Git 写入。脚本保存计划/输入/环境/工具链/原始输出。本组单独授权，不生成产品 Runner grant。

## TODO 与 Verification

2026-09-25 离线执行 `python scripts/compile.py`，退出 0；清理临时产物，只覆盖生产代码，不覆盖新增测试源码。本组未运行，未执行 formatter/lint/vet、网络、模型或 Git 写入。

- [x] 两项测试和取证脚本已准备。
- [x] 取得本组运行证据：e06-registration-terminal-20260925T042234944157Z，两项通过、退出 0。

%% REMAINING: 接受事实由 fixture 构造，不证明完整 Coder/Tester/Runner 接受链；未启动后台宿主或验证真实模型，未注入登记提交中断/磁盘故障，不宣称完整 AC20/AC21/AX05。
