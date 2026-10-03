# FD-017 独立静态审查（第二轮）

<!-- aiw-data: FD-017-review-r19.json -->

## 结论与来源

**Changes Requested**。首轮 R2 的独立管道 watcher 修复可由静态证据支持；R1 已改善跨日预留占用，但仍使用过早捕获的请求日期校验额度，需要一处收尾修正。

Reviewer session：`fd017-reviewer-20261003-a7d204`，不同于 Worker。已真实 claim `FD-017-000019-implementation-ready`，审查 FD revision 19；基线 `9ae91484d4835a50dad6d2aebbcad93c509583f9` 与限定未提交实现范围不变。读取第二轮 Worker 报告 `docs/features/reports/FD-017-implementation-r18.md/.json`；保留首轮证据。

## 唯一阻塞发现

### R1a — 高：额度校验仍取请求创建时的旧日，而非锁内实际当前日

- 位置：`plugins/aiw-agent-proxy/storage.go:134`；日期来源是 `plugins/aiw-agent-proxy/server.go:111`。
- 要求：每日额度按成功启动所属自然日归属，预留不得造成超售。首轮修复使全部未确认 reserved 占当前容量，但已经 started 的记录仍按传入 date 过滤。
- 静态时序：A 在午夜前捕获 created/ReservedAt，因锁等待或 goroutine 调度直到新日才进入 Reserve；期间 B 已在新日完成预留、启动和 Put。A 进入锁后仍用旧日 ReservedAt 查询，B 已是 started，且新日日期与旧日不符，因此不计入 A 的限额校验。每日额度 1 时 A 仍可获准并在新日启动，计数超额。
- 修改要求：在获得 Store.mu 后，以当时 `time.Now().In(zone)` 的当前自然日进行 countLocked 校验，不能依赖调用前的 created/ReservedAt。ReservedAt 可继续用于原审计和七天内容起算；所有 pending reserved 跨日占容量的修复保留，started 历史仍按真实 StartedAt 分组。不改变已确认的扣费语义。

## 首轮修复核对

- R1 部分完成：countLocked 将所有 reserved 纳入占用，Aggregate.today_reserved 同样展示所有待确认预留；转 started 使用同一 Store.mu，真实 started_at 的统计归属保留。上方 R1a 是剩余遗漏。
- R2 静态完成：runCtx 的 context.AfterFunc 独立于根终止 goroutine，整个输出消费期间仍能关闭 outputRead/errorRead/inputWrite；自然根退出不再提前撤销取消保证，消费退出时才解除 watcher。未运行脱组或超时验证，也未宣称 Linux 脱组后代必被终止。

## 命令、豁免与移交

- 实际执行真实 claim，定向 UTF-8 文件读取、限定 Go 文件 rg 行号查询与 FD/spec excerpts；未重新编译。Worker 报告本修复轮一次 `python plugins/aiw-agent-proxy/scripts/compile.py` exit 0，脚本范围沿用已审查的离线 Windows/Linux amd64 compile-only。这是 Worker 证据，不是本 Reviewer 执行结果。
- 未运行测试、网关、Codex、SDK、网络、formatter、lint、vet、发布构建或容器命令。External 测试和用户部署安排没有改变。
- Windows 原生生命周期、SDK 实际解析、竞争/重启等运行风险与首轮报告相同；本轮只退回可由静态分析确认的剩余额度缺陷，不要求新增测试或扩展范围。
- 通过真实 changes-requested 退回 Worker，仅修复 R1a 并保存新证据；这是第二次审查退回，下一轮为本自动周期第三次审查。
