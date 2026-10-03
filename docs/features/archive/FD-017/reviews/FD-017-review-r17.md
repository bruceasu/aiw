# FD-017 独立静态审查（首轮）

<!-- aiw-data: FD-017-review-r17.json -->

## 结论

**Changes Requested**。发现两项影响明确行为要求的缺陷；测试和容器部署仍按用户决定交由外部安排，不要求补跑测试。

Reviewer：`fd017-reviewer-20261003-a7d204`，真实 claim 来源事件 `FD-017-000017-implementation-ready`。审查 FD revision 17，基线 HEAD `9ae91484d4835a50dad6d2aebbcad93c509583f9`，对象为当前未提交工作区限定 diff 和新增 Go 源文件。Worker 报告为 `docs/features/reports/FD-017-implementation-r16.md` 及同名 JSON。原技能修改、批准 Issue、原 handoff 输入不属于本实现审查范围。

## 阻塞发现

### R1 — 高：跨零点预留转启动可超售新日额度

- 位置：`plugins/aiw-agent-proxy/storage.go:121`、`:129`；`plugins/aiw-agent-proxy/runner.go:82`、`:85`。
- 要求：FD Engineering decisions 的每日归属为 `started_at`，预留必须防并发超售；Work Item 1.27 和持久额度规格均要求每日限额。
- 静态证据：Reserve 只统计与 `reserved_at` 日期相同的记录；未启动预留的 recordDate 使用 ReservedAt。成功 spawn 后以新时间设置 StartedAt 并直接 Put，没有对新日期的额度重新约束。
- 可行时序：每日额度 1、并发至少 2；A 在午夜前预留但尚未启动，B 在午夜后校验时看不到旧日 A 的预留。B 与 A 随后都在新日启动，两个启动均归新日且不退额度，实际计数变成 2。进程启动串行锁不覆盖 Reserve，因此不能排除此时序。
- 修改建议：让每次额度校验保守计入该主体所有尚未确认的预留，并以同一存储锁保证预留/转启动占用的一致性；或采用其他可证明不超售的跨日转移机制。不能在成功启动后简单退款，也不能把已启动归属改回预留日来规避已确认语义。同步 Usage 预留展示说明，避免旧日预留仍占当前容量却显示为零。

### R2 — 高：根进程自然退出后，后续超时不能打断悬挂的 stdout 读取

- 位置：`plugins/aiw-agent-proxy/runner.go:96`、`:102`、`:103`、`:110`。
- 要求：FD 的请求超时与断连必须终止执行并进行有界收尾；Work Items 1.17–1.18。Linux 恶意脱组是记录的进程隔离残余风险，不表示可以取消请求超时保证。
- 静态证据：termination goroutine 在根进程退出后调用 stopTree；如果此时上下文尚未取消且组已经消失，不关闭管道，发送 termination 后立即返回。Scanner 仍同步等待 stdout EOF，之后的 ctx 超时没有存活 watcher 去关闭 outputRead。
- 可行时序：Linux 后代脱离原进程组并保留 stdout 写端；根进程自然退出，原组消失使 stopTree 返回 nil。Scanner 一直等待读取；60 秒 timeout 虽触发，但不会解除阻塞，execute 无法进入状态记录、释放并发槽或目录收尾。客户端断连同样无法恢复该已退出的 watcher。
- 修改建议：将上下文取消与标准管道关闭绑定到独立、贯穿整个消费期的 watcher，或等效的可取消读取；根进程自然退出不能提前解除这个保证。收尾时取消 watcher，stdout/stderr/stdin 都要有明确关闭所有权。即便脱组后代不受进程组控制，也必须保证本请求在超时后退出读取和收尾，不误称残余后代已经被终止。

## 已审查与实际命令

- 读取 fd-review Skill、工作管理契约、AGENTS、核心预算/验证/沟通规则及 Go service 规则；读取 FD、批准 Issue、两份稳定规格与 Worker 双格式报告。
- 通过 `aiw fd --help`、`aiw fd emit --help` 核对真实命令；执行 `aiw fd claim FD-017 FD-017-000017-implementation-ready --session fd017-reviewer-20261003-a7d204` 成功。
- 静态读取全部新增 Go 实现、客户端与入口；使用 `git diff --stat`、定向 `rg` 和文件 excerpts 跟踪认证、额度、存储/恢复、SSE、进程树与关闭顺序；检查 compile.py 是离线 Windows/Linux amd64 compile-only，输出系统空设备。没有由 Reviewer 重新编译；Worker 两次 exit 0 是其报告的证据，不是本 Reviewer 执行结果。
- 两次定向 rg 使用了不适合 Windows 的 glob 路径或不存在的目录，报路径错误；后续使用已知目录和 `-g '*.go'` 获得行号。这不是权限探测，没有扩大验证。
- 未发现必须取消用户最新测试豁免的理由。没有运行测试、网关、Codex、SDK、网络请求、formatter、lint、vet、发布构建或容器命令。

## 残余风险与移交

Windows 原生 API/Job Object、SDK 实际解析、并发与重启/清理均未运行验证；共容器跨读、Linux 主动脱组、文件系统断电耐久性及离线预留核实仍为已记录风险。上述两项是静态可定位的代码缺陷，不能由这些豁免替代。

将本报告通过真实 `changes-requested` 事件退回 Worker。仅要求修复 R1/R2、更新证据并按预算编译；没有进行实现修改，也没有声明 FD Complete。
