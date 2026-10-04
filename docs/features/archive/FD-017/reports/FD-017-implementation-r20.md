# FD-017 修复报告（第三轮实现）

<!-- aiw-data: FD-017-implementation-r20.json -->

## 来源与变更

Worker session `fd017-worker-20261003-9059c335` 已真实 claim `FD-017-000020-changes-requested`。本轮仅修复第二轮审查 R1a，保留前两轮历史报告。

Store.Reserve 在取得 Store.mu 后，以 `time.Now().In(zone)` 选择额度校验的实际当前日，不再使用请求创建时的 ReservedAt。全部 pending reserved 跨日保守占容量、同锁提交转 started、按真实 started_at 的消耗归属保持；ReservedAt 仅保留审计及正文 168 小时起算。不会因锁等待/调度跨日漏掉新日已经启动的请求。R2 的独立管道 watcher 保持不变。

## 实际证据

- 本轮一次 `python plugins/aiw-agent-proxy/scripts/compile.py`：exit 0，Windows/Linux amd64 离线 compile-only、NUL 输出，无下载/最终产物。
- 本轮一批只读 storage.go excerpt，核对锁获取在日期选择之前、全部未确认预留占用与 quota 检查在同一锁内。
- 更新 FD TODO/Verification，不改业务范围、API 或测试安排；45 项 authored、8 项 cancelled 不变。
- 未运行测试、模型、Codex、网关、SDK、formatter/lint/vet、发布构建或容器部署；没有 Git 写操作。

## 移交与限制

第三轮独立审查是本自动周期最后一轮；只有真实 verification-passed 才可完成，若仍退回则保留 pending Worker 给人类恢复，不自行继续第四轮。编译与静态推导不代表跨日竞争/管道超时的运行验证，原平台、隔离、耐久性及人工崩溃核实风险保留。
