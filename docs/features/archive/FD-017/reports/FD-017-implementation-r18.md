# FD-017 修复报告（第二轮实现）

<!-- aiw-data: FD-017-implementation-r18.json -->

## 来源

Worker session `fd017-worker-20261003-9059c335` 已真实 claim `FD-017-000018-changes-requested`。保留首轮实现和审查报告；本次仅修复 `docs/features/reviews/FD-017-review-r17.md` 的 R1/R2，没有扩大范围或更改用户外部测试/部署安排。

## 修复与静态证据

- **R1**：Store.countLocked 对当前主体所有 reserved 保守占当日容量，不按旧 reserved_at 日期移出占用。Reserve 和 started/终端 Put 使用同一 Store.mu；转换后按真实 started_at 日期计数，新日请求进入校验时已计入跨日预留。Usage today_reserved 显示所有尚未确认预留，包括旧日，today_used/groups 仍仅实际启动。README、稳定规格及 FD 当前验证同步。
- **R2**：execute 在创建 runCtx 后注册独立 context.AfterFunc，关闭 stdout/stderr/stdin 父端，直到消费收尾才解除。根进程自然退出不撤销 watcher；若脱组后代持有 stdout，后续超时/断连仍可打断 Scanner，不等永远不会到达的 EOF。没有把脱组后代声明为已受进程组控制。
- 通过一批只读检查查看两处源码及对应文档 diff，未改其它实现。已有 45 项 authored/8 项 cancelled 保持；勾选不代表运行通过。

## 实际命令

`python plugins/aiw-agent-proxy/scripts/compile.py`：本修复轮一次，exit 0，Windows/Linux amd64 离线 compile-only、输出 NUL、无最终产物/下载。另执行 scoped git diff 与 storage/runner excerpts 的只读检查。首轮两次编译保留原报告，本轮不是重跑未改变的失败命令。

未运行测试、网关、Codex、SDK、formatter/lint/vet、最终发布构建、容器部署或模型网络请求；未提交、合并、推送。当前修复为静态与编译证据，不宣称午夜竞争或管道超时已运行验证。

## 风险与移交

保留首轮已记载共容器跨读、Linux 脱组、Windows 原生 API 未运行验证、文件系统耐久性和离线预留核实风险。下一步由独立 Reviewer 审查修复及当前限定 diff，通过真实 receipt 决定是否完成；Worker 不撰写通过结论。
