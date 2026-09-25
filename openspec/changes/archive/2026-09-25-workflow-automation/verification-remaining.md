# 剩余验证工作（2026-09-25）

本记录服务于现有 wi-0012 / authored 2.2，不创建新 Work Item，不改变 Core/Gate。E01–E08 均已有局部测试通过证据；每轮结果绑定各自输入清单，不能把历次通过合并宣称当前整个 worktree 已完成验收。最新为 [E08 修复后结果](verification-results/e08-verifier-terminal-20260925T031341466094Z/report.md)。

## 当前收尾决定

用户已决定跳过 E08 真实接受封存集成验证，保留未验证风险。停止扩展可选测试，其他缺口是否延期见 [收尾范围提案](closeout-decision.md)，尚未作全局豁免。

## 历史继续顺序

E08 登记重启复用及历史缺快照两项已通过，见 [结果](verification-results/e08-registration-terminal-20260925T045038918158Z/report.md)；下一缺口为真实接受时需求/diff/receipt 封存，不能用本组构造快照替代。

当前 E07 配置、Go 协议和真实 console 子进程均已有通过证据：

E07 config：四项全部 OK；Python 3.12.13，见 [结果](verification-results/e07-config-terminal-20260925T043543064108Z/report.md)。

E07 protocol：四项全部 passed；Go 1.25.1，见 [结果](verification-results/e07-protocol-terminal-20260925T043800322332Z/report.md)。

E07 process：三项和两个子场景全部 passed；Go 1.25.1，见 [结果](verification-results/e07-process-terminal-20260925T044205271177Z/report.md)。

原始配置快照、禁用配置和进程请求限额三项也已通过，见 [结果](verification-results/e07-preflight-terminal-20260925T044622374590Z/report.md)。下方早期计划按历史记录保留，以这里的当前结果为准。

E07 配置子集已准备 [真实配置四项计划](e07-config-verification-plan.md)，须使用实施记录要求的 Python 3.11+。用户提供 D:\green\python3.12，已确认 python.exe 存在，实际版本由运行记录确认；测试尚未执行，生产 Go 入口的 PATH 兼容性仍需单独核验。

顺序 3 的 [E07 适配器离线四项](verification-results/e07-adapter-terminal-20260925T042827204819Z/report.md) 全部通过；传输/进程均为替身，没有发送通知。下一步优先核验 Python 3.9.13 下未覆盖的真实配置解析器条件，再补 Go 适配器协议；真实 TLS/服务和端到端仍待验证。

E06 登记子集已通过：[接受来源登记与持久草稿两项结果](verification-results/e06-registration-terminal-20260925T042234944157Z/report.md)，使用真实临时 Store 与构造的接受事实；完整接受链、真实宿主和登记中断窗口仍待验证。下一步可推进顺序 3 的 E07 适配器协议及错误分类本地测试。

顺序 2 的语义子集已通过：[E06 三项及六个子场景](verification-results/e06-generation-terminal-20260925T041741918570Z/report.md)，覆盖无新增契约、部分覆盖和版本保留；登记与临时 Store 发布的后续结果见上。真实宿主/模型仍待独立验证。

顺序 1 的 memory 保存恢复子集已通过：[三项结果](verification-results/e05-save-recovery-terminal-20260925T041224728311Z/report.md)。采用持久中断状态重建，不包含真实磁盘故障/进程 kill，其余 E05 缺口保留。下一项可独立推进顺序 2 的 E06 异步提取/部分草稿验证，不将 E05 标为完整验收。

| 顺序 | 工作与缺口 | 完成所需证据 |
| --- | --- | --- |
| 1 | E05 模型重调与保存失败共用一次恢复；赞助者 Stop、未知预约及持久提交中断窗口 | 临时 Store/故障注入的同输入恢复账，证明重启或跨消费者不刷新；保留失败输出及所有副作用前后状态 |
| 2 | E06 异步提取、部分草稿、无新增与失败区分、生成新版本不继承确认 | 从已接受来源登记至发布的精确版本/覆盖轨迹；保留未变条目审阅状态和缺口 |
| 3 | E07 适配器协议、TLS/HTTP/权限错误分类与冻结目标；E08 接受时真实需求/diff/receipt 封存及首次登记去重 | 先准备本地可控替身测试；真实网络/模型另有明确范围及配置，不以状态机/手工快照替代实际适配器证据 |
| 4 | E01–E04 受管阶段、Tester/Runner 授权、当前输入接受边界以及 AX01–AX05 跨组序列 | 有界联合计划和实际记录，绑定 Task/Attempt/阶段/计划/grant/清单/结果；平台崩溃、路径边界及副作用恢复单独明确场景 |
| 5 | schema 10 联合启用与真实宿主条件 | 平台持久性、精确模型能力/完整输入计量、授权、资源盘点及配置证据；证据不足保持当前未启用状态 |
| 6 | wi-0003 / 3.1 与 wi-0004 / 3.2 | 在 2.2 依赖满足后确认批准范围，再针对明确基线核对整个 diff 的相关性；各轮局部静态检查不替代整体验收 |

上述是工作排序，不自动授权新的测试、网络、模型、Git 写操作或生产迁移。测试编写和静态核对可按已授权实施范围推进；执行时沿用精确计划和单次取证模式，相关修复后仅同范围重跑。

## 证据与 Core 对账边界

用户终端记录是实际命令证据，保留在本 change 的 verification-results；它不自行生成产品 Runner grant、Attempt 或接受记录。后续通过 AIW 受管接口登记适用计划/授权和证据，保留现有 Task、worktree、Attempt/lease 及全部失败历史。不能因补了本地报告就直接改 Core JSON、抹去验证缺口或豁免整个 Gate。

%% REMAINING: 完整 AC01–AC34 与 AX01–AX05 尚未取得逐场景完整证据。部分路径已有通过，不能继续笼统称为“所有测试未执行”；同样不能把局部通过当作全表验收。当前执行阻塞应按具体未覆盖场景和受管授权/证据缺口处理。

## TODO 与 Verification

- [x] 汇总 E01–E08 已有局部证据与剩余工作排序。
- [ ] 按上述顺序继续 wi-0012，不提前勾选 2.2/3.1/3.2。

本记录仅整理已有规格和本轮结果，未新增需求或实施政策，未运行新的测试、编译、构建、模型、网络或 Git 写操作。
