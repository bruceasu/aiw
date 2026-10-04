# FD-017 独立静态审查（第三轮）

<!-- aiw-data: FD-017-review-r21.json -->

## 结论与范围

**Verification Passed（应用静态交付）**。本轮没有剩余阻塞发现；前两轮 R1、R1a、R2 已由当前源码和修复证据静态闭环。用户明确安排的 External 测试/容器部署仍未执行，不宣称运行或上线可用性通过。

Reviewer：`fd017-reviewer-20261003-a7d204`，与 Worker session 独立。已真实 claim 来源 `FD-017-000021-implementation-ready`，审查 FD revision 21、当前未提交工作区，基线 HEAD `9ae91484d4835a50dad6d2aebbcad93c509583f9`。实现范围为 Go proxy、ai 客户端、对应两份稳定规格、FD 及证据；已有技能变更、历史批准 Issue 和原 handoff 输入不属于实现。首轮完整范围审查和第二轮局部复核保留原报告，本轮核对唯一新增源码修正及其与前轮修复的一致性。

## 发现闭环

| 发现 | 当前静态证据 | 结果 |
| --- | --- | --- |
| R1/R1a：跨日额度超售 | storage.go:130–137 在取得 Store.mu 后读取实际当前日，countLocked 保守计入全部未确认 reserved；Put 也在同一锁内转 started。延迟的旧日请求不会再忽略新日已提交启动记录。ReservedAt 保留内容起算，已启动归属仍取真实 StartedAt。 | 已解决 |
| R1：Usage 预留表达 | Aggregate.today_reserved 包含跨日未确认预留；today_used 和分组仍只统计实际启动，元数据长期保留及 unknown token 表达不变。 | 已解决 |
| R2：自然根退出后读取不再可取消 | runner.go:87–90 的独立 runCtx AfterFunc 贯穿读取期，取消关闭 stdout/stderr/stdin；根终止 goroutine 退出不会撤销该 watcher，消费退出后才解除。 | 已解决 |

当前 45 项范围内 Work Items 提供设计/代码/文档证据，8 项取消；原有测试与部署任务不重新纳入交付。首轮静态检查的认证/参数、Response/SSE、存储恢复及内容清理、平台进程控制、客户端迁移与旧服务退役范围没有被本轮一处日期修正扩大。已批准历史来源与首两轮问题报告保留。

## 实际命令与证据限制

- 执行 `aiw fd claim FD-017 FD-017-000021-implementation-ready --session fd017-reviewer-20261003-a7d204` 成功。
- 使用定向 UTF-8 文件读取和限定 Go 文件 rg 核对 receipt、第三轮 Worker 双格式报告、锁内日期与占用转换、Usage、watcher、FD 当前状态及 compile.py 内容；没有执行网关、Codex 或 SDK。
- 第三轮 Worker 报告 `docs/features/reports/FD-017-implementation-r20.md/.json` 记录本轮一次 `python plugins/aiw-agent-proxy/scripts/compile.py` exit 0，Windows/Linux amd64 离线编译、输出 NUL、无最终产物。脚本静态内容与此范围一致；Reviewer 没有重跑编译。首轮两次及第二轮一次编译仅作为各轮报告历史，不冒称为本 Reviewer 执行。
- 未运行测试、并发/午夜/崩溃/管道场景、SDK 解析、网络、formatter、lint、vet、发布构建或容器部署。不存在测试通过或覆盖率结论。

## 保留风险与下一步

共容器跨读、Linux 主动脱组、Windows 原生 API 与 SDK 运行未验证、文件系统断电耐久性、异常预留离线核实及文件扫描性能均保留；独立 cwd 不等于访问隔离，compile-only 不等于运行行为证明。

本轮通过真实 `verification-passed` 移交完成决定。Reviewer 仅更新 FD TODO/Verification 与独立证据，不提交、合并、推送、部署或归档；归档须由主会话执行真实 CLI。前两轮报告维持当时 Changes Requested 结论，不改写历史。
