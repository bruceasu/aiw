# FD-025 独立黑盒测试报告 R2

<!-- aiw-data: FD-025-test-report-r2.json -->

**本轮不能通过：正常 stop 脚本返回 1。** 当前授权唯一纠正重试共运行 7 项：6 通过、1 失败；计划后续 4 项被该失败阻断。未执行第三次命令，不将脚本失败直接推断为某个产品实现原因。

## 来源与版本

- 正式 Tester 交接并已 claim：FD-025-000004-implementation-ready。
- Tester 会话：fd025-tester-20261004-a173fa；Worker 会话：fd025-host-20261004-729bf1。
- FD Revision 4，回执及测试文件摘要：48434b4de48b862943635a4e936d0d6d8bb802e8eb4f4308d4632237feb77fb6。
- 当前测试摘要：075986116c3a6f55ddfd46b0fa06826928947d9b4235997e15f3a4e7141345fa；安装二进制摘要：94990c8de93f72f35c4858ea7cc6f8cfd93b15b1ab75324f9455ff78b3e33cdc。
- R1/R2 Planner 授权分别为 docs/features/reports/FD-025-test-authorization-r1.json 与 FD-025-test-authorization-r2.json。执行前脚本校验 exact command、event、FD revision/digest、Tester、脚本摘要及二进制摘要。不得把本次正规交接与旧 FD-021 的直接测试混同。

## 两轮结果

首次命令 `python tests/fd025_shutdown_blackbox.py`，工具 exit 1，约 0.62 秒。启动就绪场景失败，后续没有执行。父会话静态确认测试器 Windows cmd 引用构造错误；这是测试入口修正，不是产品修复。首次脚本摘要 d586cec0cc151e6472cf051218b24dc3619325af80d0228e9abd3debfa4b5881，原始结果 docs/features/reports/FD-025-test-results-r1.json 保留。

只修 cmd 入口引用、安全 stderr 分类与 R2 证据路径，获新的版本绑定授权后，同一命令进行唯一廉价纠正重试。启动及认证/输入保护均通过：未认证/无效合成 Key 为 401、有效 Key 的 GET 为 405、query 和非空 body 为 400，拒绝后服务仍运行。随后正常 stop 脚本返回 1，测试失败并终止后续断言。

重试工具首段等待约 10.01 秒返回 session，最终 poll exit 1；执行工具未提供涵盖两段之间等待的完整总耗时，本报告不伪造总时长。重试原始结果 docs/features/reports/FD-025-test-results-r2.json 保留。

finally 仅观察到启动进程已退出且锁已消失，于是自有临时目录安全清理；这只能证明清理条件成立，**不能替代未执行的正常退出码、监听关闭/锁释放正式断言、重复 stop 和零执行记录断言**。stdout 被丢弃，stderr 仅临时捕获并返回固定分类，未保留原始正文；失败原因需要 Worker/Reviewer 静态诊断。

## 完整验收清单与覆盖

完整清单按 FD、README 与稳定规范的不同可观察要求分项；不靠合并或漏项提高比例。本报告列 31 项，实际进入目标行为 7 项，通过 6、失败 1、未运行 24。CLI 的覆盖定义只计通过场景，因此 Covered scenarios 为 6/31（19.35%）；失败 stop 标记 uncovered 并保留 execution_result=failed，不能计为覆盖通过。其中 11 项属于本条授权的聚焦运行计划，余项明确保留为未运行或静态检查范围；详细逐项状态见同名 JSON data.scenarios。

本条计划未跑的 4 项：正常退出码、监听/锁完成断言、重复 stop、无执行/正文记录。其他未运行要求包括 Ctrl+C/SIGTERM、活动后端取消、shared 非loopback来源/具体远程监听配置、旧版404、安装保护与跨平台入口、审计写失败、周期清理收敛、15秒期限、失败不强杀/不删锁、重定向/重试、凭据脱敏、其他enabled Key以及Responses/Models/Usage兼容。

分支覆盖率未测量，无覆盖仪器。低覆盖和静态排除需 PM 明确风险处理；存在实际失败，不能以覆盖例外把行为失败验收为通过。

## 副作用与隔离

只在独立 TemporaryDirectory 创建合成 Key、新配置、state/work 和 stderr 临时文件；配置仅从已安装文件复制 codex_path/codex_script/models/timezone，未复制真实 Key 或状态。隐藏后台运行启停 bat，使用临时 loopback 端口，明确不用 43127；不改或启停用户真实实例。

没有调用 /v1/responses、真实模型、外部网络、构建、下载、Go 测试、Git 写操作或生产源码修改。运行时只测试 shutdown 控制入口。失败不强杀、不删除活跃锁；两轮均在自有启动进程退出且锁消失后清理自有临时目录。独占写 R1/R2 raw，保留历史。

## 结论与下一步

建议 **fail**：正常 stop 返回非零这一失败未解决，暂不能推进测试接受。保留 claimed Tester 的正式来源，完成报告后发出 test-report-ready 交由 PM 决策。停止进一步运行，后续需 Worker/Reviewer 静态定位，并在新的适当交接和版本绑定授权下处理验证。
