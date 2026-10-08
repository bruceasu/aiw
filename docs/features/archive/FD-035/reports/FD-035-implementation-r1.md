# FD-035 Worker 实现报告，第 1 轮

<!-- aiw-data: FD-035-implementation-r1.json -->

Worker session：`fd035-worker-20261008-01`；来源事件：`FD-035-000003-design-ready`。

## 结果

FD 测试报告的 PM 决策新增 `adaptive-v1`：常规无失败报告使用一份独立评估并遵循该票；存在失败测试、PM 判定证据缺口重大或 PM 与首份意见分歧时使用三份评估，以至少两票接纳作为 Reviewer 交接条件。三份模式指定验收/用户影响、工程证据/修复、交付/运行风险三个侧重点；每位评估者仍填写完整风险判断。CLI 校验来源、修订、摘要、session、侧重点、升级原因和票数。未带新策略字段的三份式决策仍可按旧规则处理，归档证据未改动。

## 改动与静态证据

- 更新风险评估、PM 决策的 Markdown/JSON 模板，以及 `openspec/specs/fd-workflow/spec.md`。
- 更新 `plugins/aiw-fd.py` 的单份/三份 PM 决策校验；原有事件回执携带的评估者 session 列表继续供 Reviewer claim 检查。
- 更新共享工作契约、两份 FD workflow Skill、PM/Tester/Reviewer 角色提示、两份 fd-review Skill 与使用文档。
- 扩展 `tests/test_fd032_risk_decision_blackbox.py` 的黑盒夹具，准备单份接纳/退回、失败测试强制三份、侧重点、重大缺口、PM 分歧和旧三份兼容场景。
- 静态追踪了 Tester 报告 → PM 校验 → `test-accepted`/`test-rejected` → Reviewer claim；`git diff --check develop...HEAD` 无差异错误。

## 实际命令与限制

- `git diff --check develop...HEAD`：通过，无输出。
- `python -B -c "from pathlib import Path; files = ('plugins/aiw-fd.py', 'tests/test_fd032_risk_decision_blackbox.py'); [compile(Path(p).read_bytes(), p, 'exec') for p in files]; print('compile-only OK')"`：通过，输出 `compile-only OK`，未保留编译产物。
- 未运行测试、覆盖率、最终构建、格式化、lint、网络命令或发布操作。测试场景的运行结果尚待独立 Tester；不能将已编写用例视作通过。

## 剩余风险

PM 对证据缺口是否重大的判断是人工风险决定；CLI 能核对记录与模式，却不能证明该判断内容正确。Reviewer 仍需核查事实依据。旧三份式决策被保留用于进行中的旧交接，新流程文档要求使用 `adaptive-v1`。
