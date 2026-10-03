# FD-025 独立黑盒测试报告 R4

<!-- aiw-data: FD-025-test-report-r4.json -->

**本轮授权的 12 项行为全部通过**，单次命令 exit 0。正常和重复 stop 都返回 0，启动链正常退出，临时监听关闭、锁释放；新增临时插件默认配置 stop 也返回 0。完整验收清单仍有 20 项未运行，不能描述为全部 FD 运行验收已通过。

## 正式来源与授权

来源事件并已 claim：FD-025-000010-implementation-ready；Tester：fd025-tester-20261004-a173fa；Worker：fd025-host-20261004-729bf1。

- FD Revision 10，回执及执行摘要：ac7ac370bc33e29843e5b220e62943b8784cc204a1b6e9e8fe326cef5631741e。
- 测试脚本摘要：30fe778a4a1986aba6024e804751764211c52c2aee8ddef33141dc95149ebe77。
- 已安装 Windows Gateway 二进制摘要：97a6753035e944876d942ec48d7ec2de029f0a93e9d7f906f49cbdc49ef378a3。
- 已安装插件入口摘要：e9924962f30f39d5112a0eed1743fb6c6223c22127aeddb87e1f761ae44113c5。

Planner 的版本绑定授权为 docs/features/reports/FD-025-test-authorization-r4.md 及同名 JSON。脚本在任何临时进程/网络前检查 decision、event、FD revision/digest、Tester、exact command 以及上述三项文件摘要。该授权用于实际产品修复后的单次同范围验证；R3 是未运行的阻断报告，不虚构第三次失败后重试。

## 实际运行与结果

仓库根仅执行一次 `python tests/fd025_shutdown_blackbox.py`，工具 exit 0，约 1.67 秒。原始安全结果为 docs/features/reports/FD-025-test-results-r4.json，独占创建；无重试。

12 项实际行为为：任意工作目录启动脚本及就绪；未认证 Key/无效合成 Key 401；有效 Key 的 GET 405、query/body 400 且服务继续；正常 stop 0；启动链退出 0；监听关闭及 gateway.lock 消失；重复 stop 0；执行和正文记录为 0；临时复制插件入口/exe/合成 gateway.json 后，不带 --config 的 stop 为 0。

测试只用临时 loopback 端口，明确不用 43127；配置仅从安装配置复制非敏感 codex_path/codex_script/models/timezone，所有 Key、listen、state/work 均替换为本测试合成或临时值。默认配置场景只读自有临时 gateway.json，没有读用真实默认凭据。无 /v1/responses、模型执行、外部网络或用户真实实例启停。

stdout 丢弃；stderr 仅写自有临时文件并只输出固定分类，未输出凭据、配置、正文或原始日志。正常关停后，自有启动链已退出且锁消失，才在验证后的 TemporaryDirectory 边界清理；temporary_workspace_removed=true。没有强杀或删除活跃锁。

## 完整验收清单与覆盖

沿用 R2/R3 的 31 项不同可观察需求，新增默认插件配置 stop 一项，总计 **32**。覆盖通过 12/32（**37.5%**），执行行为测试 12、通过 12、失败 0、未运行 20。每项行为、状态、运行测试标识和安全结果见同名 JSON data.scenarios；Covered scenarios 只计 passed，未执行项均为 uncovered。

未运行项包括 Ctrl+C/SIGTERM、活动模型进程树取消、shared 非loopback控制和具体远程监听配置、旧版404提示、安装配置保护/跨平台入口、审计不可写、活动周期清理、15秒期限与停机失败不强杀/不删锁、重定向/自动重试、产品输出脱敏、多 enabled Key、Responses/Models/Usage 原协议。它们仍是完整清单的一部分，不因本轮范围窄而被删除。

分支覆盖率未测量，没有覆盖仪器；低于70%的要求覆盖和分支缺口需 PM 用静态证据、明确例外和残余风险决策。本报告的 pass 仅指已授权聚焦测试没有失败，不替 PM 或 Reviewer 宣告完整 FD 验收。

## 历史与未运行检查

R1 是测试入口引用错误导致 startup 失败；R2 在入口纠正后 6 通过、正常 stop 返回1；R3 未执行，因 PM 静态发现默认插件配置顺序需要返工。本轮运行当前安装版本，解决后的 stop 和默认配置入口均得到真实成功证据；历史记录不覆盖、不改写、不合计为当前通过数。

本 Tester 没有读取 main/server/shutdown 实现设计黑盒，没有修改产品、FD 或其他 FD。没有 Go 测试、下载、构建、lint/formatter/vet、验证脚本、Git 写操作；编译/安装属于 Worker 自己的已记录命令，不能冒充 Tester 运行。未新增模型或网络预算。

## 建议

建议 **pass** 本轮测试报告并正式 test-report-ready 交 PM；PM 需对完整清单覆盖和分支缺口作显式决策，再交独立 Reviewer。本轮不再运行任何测试或真实模型。
