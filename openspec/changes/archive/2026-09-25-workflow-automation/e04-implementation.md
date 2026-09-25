# E04 实施记录

Task/change：workflow-automation。Work Item：wi-0006，清单 1.4，Attempt：attempt-1789718088692918200。保持 feature/workflow-automation、现有 worktree 和前序 E01–E03 改动；不更改真实 Task/Core 状态、Gate、租约或 Git 索引。

## 实现

- `internal/workflow/generation_budget.go`：schema 10 内的版本化请求账，与 E02 阶段和租约同一条件提交。按 Work Item、Actor、实际 provider/model、生成请求保存有效失败；编译/测试及延迟诊断使用派发前冻结的产物来源，不把 Compiler/Runner 当作模型 Actor。重复缺陷只计同一生成请求一次，成功、环境失败、其他 Actor、Stop 和重启不清零。
- 每档两次有效失败后，下一次合法生成要求下一不同模型；每 Actor 最多升级两次，无更高档或达到最终失败阈值时返回等待原因。准备阶段只预留；确认派发或保存终态后消费，unknown 保留预留，有证据的 not-dispatched 释放。未派发的升级不消费次数。
- 测试失败归因为实现/测试缺陷后，后续 Coder/Tester 生成共用六次修复派发，包括 Coder 成功后的测试重写；中间成功不打开无计次通道。额度检查仅限制新修复，结果消费与编译/测试不受第六次上限拦截。报告补交不得借基础设施恢复扩额；诊断仍沿用 E03 原失败的一次只读账。
- `generation_routing.go`：固定配置推荐模型只调用一次，接收 Work Item、Actor、冻结输入、需求正文及历史有效失败。失败/无效推荐保存默认理由；配置显式给出升级顺序，实际模型别名去重，不从名称推断能力。请求保存模型快照、路由理由和上下文摘要；已有账户从持久档位继续，不随配置或重启重置。
- `execution/generation_routing.go`：复用现有 AI 配置、Profile 和调用接口，提供配置快照/推荐适配；单次 provider/model 覆盖优先于初始推荐。Compiler/Test Runner 不接入模型路由。Verifier 可取得选择，但其实际后台消费属于 E08。
- `StageRequest` 保存路由及原生成请求引用；`PrepareStage` 在 Core 内核对来源，冻结 Actor 输入的模型必须与请求相同。`ConnectBudgetedVerification` 组装 E03 受控验证、E04 预算和外部平台启用权威，未提供真实宿主/启用权威时拒绝注册；注册本身不迁移或启用 Task。
- 旧账缺少生成请求/模型身份时，集中保留 unknown Work Item 列表，不将旧编译失败计数转换为新零余额。未来账本版本和未知字段拒绝消费。`GenerationBudgetSnapshot` 提供只读诊断数据。

## TODO

- [x] 实现 E04 路由、请求去重、双 Actor 升级与共享修复账。
- [x] 补充聚焦测试代码：延迟归因/重复观察、重启序列化、预留释放、两次升级、独立 Actor、六次修复边界、无效推荐/模型别名、旧预算未知和未来版本拒绝。
- [ ] 取得 E01–E04 联合启用的宿主隔离、平台持久性、真实恢复和接受验证证据；再由受管入口执行迁移和生产配置注册。
- [ ] 获授权后执行 AC08/AC15–AC17 及相关恢复/接受场景。

## Verification

实现依据：handoff、Task 绑定、proposal、design 的 R1/预算/一致启用规则、ai-routing delta 与稳定规范、E02/E03 实施记录及阶段、授权、归因、接受调用链。由于初次中文读取编码错误及大段输出截断，追加了针对性 UTF-8/源码片段读取，未据不完整输出猜测接缝。

实际操作包含 Get-Content/Get-ChildItem/Get-Command、rg、git status --short、aiw patch --help 和文件补丁。`aiw patch` 是 Git apply 包装；本轮没有 Git 写授权，使用直接文件补丁，保留原索引与工作区。未运行测试、最终制品构建、formatter、lint、vet、验证脚本、网络、依赖下载或子 Agent。

编译命令：`$env:GOPROXY = 'off'; $env:GOSUMDB = 'off'; $env:GOTOOLCHAIN = 'local'; python scripts/compile.py`。首次调用的返回值只保留输出，未保存退出状态，不作为通过证据。补齐单次覆盖后的一次编译退出 0；随后静态分析收紧“中间成功后的生成仍计六次额度”并修改代码，再次编译最终源码退出 0（约 2.1 秒）。没有重试已知失败的编译。脚本生成临时二进制后删除，不保留最终制品，也不编译/运行 `_test.go`。

一次最终静态批次：`git diff --check -- openspec/changes/workflow-automation/tasks.md`、该文件的 `git diff --numstat`、三个新路由/预算源码的 UTF-8 读取及阶段/清单的 Select-String。空白检查无错误，Git 提示 LF/CRLF 转换；统计包含前序未提交改动，不能全归为本轮。新 Go 文件和前序未跟踪阶段文件通过补丁及源码读取核对，未声称 diff 检查覆盖了它们。此后仅修正注释缩进/导入顺序并补全报告；未重复静态检查。

依 implement 技能提供一次可选聚焦测试：`go test ./internal/workflow -run '^TestGeneration(Budget|Routing)' -count=1`，预计 10–60 秒，禁用依赖/工具链下载；未获明确授权时不运行。测试不会自动扩展到 execution 包、完整仓库或 AC/AX。

建议下一 Core 动作：由原 managed adapter 消费本 Attempt 的报告与编译证据，完成 wi-0006，并按已有依赖继续 E05 / 1.5；不手写状态、不释放租约、不解除 Gate，3.1/3.2 继续等待整体验收。

%% ACTIVATION_PENDING：默认 schema 9 及旧执行行为保持不变；新账与新接受通过 schema 10 同时生效。生产宿主隔离/观察权威、平台验证、配置的显式档位顺序和旧账对账仍是启用前置条件，本轮不以空回调、静态检查或编译代替证据，不迁移真实 Task、不解除 Gate。未知旧账继续阻止派发。

%% VALIDATION_PENDING：测试代码未运行，所有 AC/AX 场景仍未执行；本工作项勾选只表示实现完成，整体接受仍由剩余工作项与当前证据决定。
