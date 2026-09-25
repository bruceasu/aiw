# E05 来源与固定输入验证结果

Task workflow-automation；wi-0012 / authored 2.2；实现来源 E05 / wi-0007。

用户在本会话 confirm 后，代理于既有 worktree 执行 `python openspec/changes/workflow-automation/run-e05-source-terminal.py` 一次。退出码 0；三个测试及四个子场景全部 passed，无重试。

| 测试 | 结果 |
| --- | --- |
| TestAuxiliarySourceTracksMaterialFactsOnly | passed |
| TestAuxiliaryJobIdentitySharedAcrossSponsors | passed |
| TestAuxiliaryOversizeInputIsUnavailableWithoutTruncation | passed；memory、knowledge-extraction、knowledge-summary、verifier 四个子场景全部 passed |

实际命令：

```text
go test ./internal/workflow -run ^(TestAuxiliarySourceTracksMaterialFactsOnly|TestAuxiliaryJobIdentitySharedAcrossSponsors|TestAuxiliaryOversizeInputIsUnavailableWithoutTruncation)$ -count=1 -timeout=60s -vet=off -json
```

开始 2026-09-25T00:57:20.981583Z；结束 00:57:25.632549Z。Go 1.25.1 windows/amd64；GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local。执行期间 changed_inputs 为空，stderr 为空。准确 argv、cwd、环境及输出哈希见 [run.json](run.json)，原始事件见 [stdout.jsonl](stdout.jsonl)。

冻结输入清单 [inputs.json](inputs.json) SHA256：`e4108f66f58de97d0083d291a92adb622f520ba81c43d20ed1b41d023e7843ce`。执行前计划快照 [approved-plan.md](approved-plan.md) SHA256：`45ada446d607f0a8d781ae0385255a8b9b0cc516a994801bda86b3f235f7e793`。计划中“尚未运行”为冻结时状态，本文记录后续结果，不改写该快照。

本次授权来自用户会话确认；脚本通用 invocation.json 的 operator invocation 不代表用户亲自在终端输入，实际由代理执行。没有生成产品 Runner grant，没有绕过 Core Gate。

覆盖来源实质变化/旧快照保全、跨消费者的确定性标识、完整输入字节上限。未验证持久队列、恢复额度共享、资源累计、Stop/迟到结果、memory 发布、真实宿主或模型；未调用网络或发送通知，不代表完整 E05 或 AC/AX 验收。2.2 保持未完成。
