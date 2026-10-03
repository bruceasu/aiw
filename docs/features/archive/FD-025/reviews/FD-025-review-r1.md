# FD-025 独立评审 R1

<!-- aiw-data: FD-025-review-r1.json -->

结论：**验证通过，未发现需返工的实质问题**。此结论依据当前差异、12 项真实运行结果与 PM 明确接受的覆盖例外；20 项未运行仍未运行，不能称全部行为已实测。

- Reviewer：`fd025-reviewer-20261004-fd021_reviewer`，区别于 Worker `fd025-host-20261004-729bf1` 与 Tester `fd025-tester-20261004-a173fa`。
- source_event：`FD-025-000012-test-accepted`，已先读取 pending 事件并独立 claim；来源 Revision 12、摘要 `912f379dc75320bc0d0a6f31b928b3a39fe39cc3871184b7bd10739b5b05b812`。
- 差异基线：`c9b6d51cd2308e1ffbaf51755d697f6acda3ad83` 至当前工作区。仅评审 FD-025 路径；其他 FD 历史变更不属于本次结论。
- 范围包含未跟踪的 shutdown.go、shutdown_windows.go、shutdown_linux.go、stop-gateway.bat，及 main/server、start 脚本、build.bat、插件入口、README、稳定规范。
- 发现清单：无。

## 行为静态核实

| 要求 | 当前依据 |
| --- | --- |
| 认证且仅 loopback 控制 | ServeHTTP 先调用现有 authenticate，仍要求 enabled 主体；requestShutdown 单独解析 RemoteAddr 的 loopback，拒绝非 POST、query、非空 body；不信任转发头。所有 enabled Key 复用既有认证。 |
| 审计失败仍可安全 stop | PutHTTPRequest 错误仅对 /internal/shutdown 路径允许继续，认证与本地输入门禁仍执行；固定日志不含 Key/正文。其他 AI 路径仍返回 storage_error。 |
| 202 后取消且不执行模型 | requestShutdown 先写 accepted JSON，再调用全局 cancel；该分支不调用 acquire/execute。并发 Handler 的审计 defer 由 HTTP Shutdown 等待。 |
| 活动请求复用既有清理 | main 的同一 signal context 仍接受 Ctrl+C/SIGTERM；Responses 通过 context.AfterFunc(g.ctx,cancel) 取消请求；runner 的 runCtx 关闭管道、stopTree、等待根进程和记录终态/清理。Windows Job Object 包含子孙，Linux 使用独立进程组及既有 TERM/KILL 收敛。没有更改此清理协议。 |
| HTTP/周期清理完成后释放锁 | main 等待 server.Shutdown 成功，再等待 cleanupDone；之后才由 Store.Close 释放 gateway.lock。10 秒服务截止失败时 closeStore=false，保留锁并返回错误。 |
| CLI 有界、无强杀/删锁 | stop 只读取配置，不打开状态存储、不要求 backend 存在；HTTP client 5 秒、Dial 3 秒，禁 redirect；接受严格 202/shutting_down 后最多约 15 秒等待监听拒绝与锁消失。失败返回固定错误，无 kill 或 Remove；无自动 HTTP 重试，轮询只进行 TCP 停止确认。 |
| 本地监听及旧版边界 | loopback 直接使用、通配地址转换为对应 loopback、具体远程网卡配置明确拒绝；404 提示先退出旧版本并重建。Windows 按 Winsock 10061 包装错误识别，Linux 保留 ECONNREFUSED。 |
| 脚本和默认 stop | start/stop bat 通过 PATH 调用 aiw agent-gateway，转发参数和退出码；Python 默认配置插入 start/stop 动作之后，显式 --config 保留。不会依赖调用者工作目录查找插件默认配置。 |
| 安装配置保护 | build.bat bin 仅新增 Gateway Windows/Linux 构建及安装二进制和 Python 入口；install_gateway 明确未复制 gateway.json。Go 网络下载关闭。用户已明确授权这一最终安装例外。 |
| 原协议兼容 | 默认 start 参数解析保留，Responses/Models/Usage 路由及数据格式未改；新增内部路由和全局取消链接，不扩展 OpenAI 请求子集。 |

服务端对已有后端进程树使用既有终止流程，不等同 stop CLI 强杀 Gateway。关停异常保留锁供运营者核实；报告不承诺故障下绝无后端逃逸，该风险继承既有跨平台实现且未实测。

## 运行证据与授权核对

读取 Worker R1/R2/R3、Tester R4 Markdown/JSON/raw、Planner R4 授权与 PM R3 决策。测试来源为 `FD-025-000010-implementation-ready`，Revision 10、FD digest `ac7ac370bc33e29843e5b220e62943b8784cc204a1b6e9e8fe326cef5631741e`；后续 Revision 11 为 PM 决策，Revision 12 为 Reviewer 派发。

Planner 在源码检查后批准精确命令 `python tests/fd025_shutdown_blackbox.py`，绑定 Tester、测试、已安装 exe 和入口摘要。测试代码在启动任何临时进程/网络前验证该绑定。raw 三份摘要与授权/报告一致；Reviewer 另以 Get-FileHash 静态读取当前三文件摘要。测试不使用 43127、真实 Key/状态或 /v1/responses，临时合成配置，正常退出且锁消失后才清理自有临时目录；失败保留目录，未强杀或删除活跃锁。

Tester 单次执行退出 0，约 1.6672538 秒，12 passed、0 failed：任意工作目录启动、未认证/无效 Key/GET/query/body 拒绝且服务继续、正常 stop、启动链退出、监听和锁释放、重复 stop、零模型/正文记录、隔离默认配置 stop。raw 记录 temporary_workspace_removed=true。历史 R1/R2 失败和 R3 未运行仍保留，不能合计为当前通过数。

完整清单为 32 个独立行为，12/32 覆盖 **37.5%**，20 unrun；分支覆盖未测量。PM `FD-025-test-decision-r3.md/json` 明确接受低于 70% 和分支未测量例外，要求 Reviewer 静态追踪剩余行为。本报告尊重该决定，不重提旧阈值要求，不免除任何行为要求，也不把静态核实计为运行通过。

Worker 记录 Windows/Linux 无产物编译、Python 内存 compile 和用户授权 build.bat bin 安装均退出 0；Reviewer 未重新运行这些命令。

## 剩余风险与实际操作

20 项仍缺运行证据，尤其活动模型树取消、Linux 真正运行、共享非 loopback、审计/清理失败、旧版404、15秒期限及故障保锁、产品脱敏和原 API。静态链路支持要求；没有把这些声明为实测通过。

Reviewer 实际执行定向 Get-Content/rg、scoped git diff、git rev-parse HEAD、Get-FileHash，读取命令帮助，并 claim 本次精确 Reviewer 事件；未运行测试、编译、构建、网络、真实模型、formatter/lint/vet 或 Git 写操作。仅写本报告/JSON 与 FD TODO/Verification；依本报告正式 emit verification-passed。此通过允许后续 PM 完成/关闭决策，不授权 push、merge、发布、部署或归档。
