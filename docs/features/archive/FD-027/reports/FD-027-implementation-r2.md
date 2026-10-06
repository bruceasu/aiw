# FD-027 实施报告（第 2 轮）

<!-- aiw-data: FD-027-implementation-r2.json -->

## 交接

- FD：FD-027；Worker 收据：`FD-027-000007-changes-requested`。
- Worker 会话：`fd027-worker-20261006-d2b7e9`。
- 修复依据：独立审查 `FD-027-000006-test-accepted` 后的 R1 报告
  `docs/features/reviews/FD-027-review-r1.md`。

## R1 发现与修复

1. `wt add` 现在在创建 worktree 和 FD metadata 之前检查两个生成路径
   是否被 Git 忽略。缺少 `.wt/` 或 `.ai/` 忽略规则时，命令指出应添加并
   提交的规则，且不改动 Git 状态。此修复对应新增 Work Item 1.8。
2. README 已移除不可用的 `aiw-wf` 构建指引、空命令块和 `aiw wt ignore`
   章节；FD worktree 指南说明添加前所需的忽略规则。稳定 FD 规格与
   工作管理指南已同步。
3. FD Verification/TODO 现在记录真实 Worker、Tester、PM、Reviewer R1
   事件、首轮 7/9 的夹具失败、授权后 9/9 的结果，以及 14/19 场景和
   未测分支覆盖率。未运行场景仍标为未验证。

## 已执行检查与边界

- `git diff --check`：无空白差异错误；Git 给出换行符转换提示。
- `python -c "from pathlib import Path; p=Path('plugins/aiw-wt.py'); compile(p.read_text(encoding='utf-8'), str(p), 'exec')"`：退出 0，仅内存编译。
- 未为 R2 执行行为测试、Go 编译、最终构建、lint、格式化、网络或部署。
  R1 测试结果不冒充 R2 的运行证据；独立 Tester 须按新 FD 修订和精确
  授权重新验证新增忽略规则预检。

## 剩余风险

非内容冲突、abort/恢复失败等场景仍缺少运行时证据；分支覆盖率未测量。
主工作区另有 FD-028 未提交内容，后续合并须等待 parent 干净且不能
改动这些无关文件。
