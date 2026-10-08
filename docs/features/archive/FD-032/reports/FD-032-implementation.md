# FD-032 Worker 实现报告

<!-- aiw-data: FD-032-implementation.json -->

## 改动与原因

本轮把 Tester 报告的接纳规则改为三位独立评估者的多数决策。Tester 继续如实报告失败、未运行与覆盖缺口；三份评估分别说明严重性、影响范围、修复时间、交付影响和风险。PM 的 `test-accepted` 或 `test-rejected` 事件必须引用三份有效报告并与票数相符。Reviewer 使用不同 session，核查证据和未评估缺陷，不重复否决多数票明确接纳的同一已知风险。

新增双证据评估模板，更新 PM 模板、FD 稳定规格、Auto/PM/Tester/Reviewer 指引和用法。历史归档报告未改写；没有增加新的 FD 事件类型。

## 静态证据与命令

- 已检查 `test-report-ready` → 三份评估 → PM 决策 → Reviewer claim 的类型、事件、FD 修订/摘要和 session 传递。
- `rg` 检查相关规则文件不再保留 70% 自动接纳门槛或失败测试必拒文字；规格中的 “coverage threshold” 仅用于描述不因数值向用户请求解除 Gate。
- `python -c "from pathlib import Path; p=Path('plugins/aiw-fd.py'); compile(p.read_bytes(), str(p), 'exec')"` 通过，无产物。
- 未运行测试、覆盖率、最终构建、lint、格式化、vet 或网络命令。

## 剩余风险

多数接纳失败测试、少数接纳退回、缺失或过时评估、Reviewer session 冲突等路径尚无当前修订的独立运行时证据。Tester 需按精确授权验证；Reviewer 仍需复核评估证据真实性和未评估缺陷。Worker session：`fd032-host-20261008-31a7c2`；来源事件：`FD-032-000003-design-ready`。
