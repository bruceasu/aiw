# FD-027 独立审查报告（第 1 轮）

<!-- aiw-data: FD-027-review-r1.json -->

## 审查范围与结论

- Reviewer 会话：`fd027-reviewer-20261006-bd79a1`；认领收据：`FD-027-000006-test-accepted`。
- 对比基准：`develop...feature/FD-027`；实现分支 HEAD：`86b4b38`。
- 结论：**changes-requested**。当前 `wt add` 可把原本干净的 parent 留在无法交付的脏状态，README 也仍发布不存在的命令。

## 发现

1. **需修复，`wt add` 会破坏后续交付的干净前提。** `plugins/aiw-wt.py` 的 `worktree_add` 只检查 parent 在创建前干净，随后创建 `.wt/<id>` 与 `.ai/fd/<id>/workspace.json`；代码未要求这些路径被 Git 忽略，也未保证创建后 parent 仍干净。若一个原本干净的仓库未忽略 `.wt/`、`.ai/`，`add` 返回成功，但 `local-merge` 的 `clean_worktree(parent)` 必然拒绝。Tester 首轮真实运行的两项失败正显示 `?? .ai/`、`?? .wt/`；修订的夹具把它们加入 `.gitignore` 后 9/9 通过，因此最终运行证据只覆盖已有忽略规则的仓库。FD 验收要求 `add` 和 `local-merge` 可连续用于 FD worktree、且脏状态在 Git 改动前被可操作地拒绝。请在 `add` 改动 Git 前确保元数据与 worktree 目录不会令 parent 变脏，或明确拒绝缺少必要忽略规则的仓库并说明修复方法；增加相应的黑盒场景。
2. **需修复，命令文档与实际接口冲突。** [README.md](../../../README.md) 第 793 行仍有独立章节 `aiw wt ignore`，声称该命令会修改 `.gitignore`；`plugins/aiw-wt.py` 的 dispatch 与帮助只支持 `add/status/commit/local-merge/list`，补全也已删除 `ignore`。README 第 63–73 行还指导构建已移除的 `aiw-wf` 插件，并留下空的命令代码块。FD-027 的 Acceptance 和 `openspec/specs/command-help-consistency/spec.md` 明确要求 README 仅展示实际可用的 FD worktree 命令，并移除旧 `wf` 路径。请删除或改写这些当前指引。
3. **交接记录待同步。** 主工作区 FD 当前 `Verification` 仍写“测试未运行”，`TODO` 仍将 Worker 完成、Tester 报告和 PM 决策列为未完成；实际收据与报告已记录这些阶段。请在修复后、重新交接前更新 FD 文档中的真实证据与 TODO，保留未覆盖场景和未测分支覆盖率为未验证项。勿把它们写成通过。

## 证据核对

- `FD-027-000004-implementation-ready` 为 Worker 真实收据，绑定 FD revision 4、digest `5ac7cadd…`；两份 Planner 授权均绑定该收据、同一 Tester 会话及同一精确命令。当前测试文件 SHA-256 为 `594DB3EAFC9E223286597516D34CD7F8362494D1E68DF71F0AB7FACD1A4933FC`，与第 2 轮授权一致。
- Tester 报告保留首轮 7/9 和夹具失败的原始输出；修订夹具后的唯一重跑报告 9/9 通过，19 个独立场景中 14 个有通过证据，需求场景覆盖率 73.68%，业务分支覆盖率未测。PM 对未测覆盖率和 S15–S19 的例外仅准许进入审查，并未把未执行项判为通过。
- 静态检查 `plugins/aiw-wt.py` 的 metadata 校验、双工作树干净预检、parent 冲突的 `MERGE_HEAD`/未合并索引判定、abort 后干净检查、反向合并和显式重试路径；检查 `plugins/aiw-fd.py`、帮助和补全对旧 `fd worktree` 的移除。S17–S19 的异常保留路径仍缺少运行证据，当前仅有静态支持。
- 主工作区 FD revision 6 的文件 SHA-256 与 `FD-027-000006-test-accepted` 收据 digest 一致。主工作区另有 FD-028 未提交改动，交付前必须由其所有者处理，审查未触碰这些文件。

## 实际命令与边界

执行了只读的 `Get-Content`、`rg`、`git diff`、`git log`、`git status` 和 `Get-FileHash`；从主工作区执行了 `aiw fd claim FD-027 FD-027-000006-test-accepted --session fd027-reviewer-20261006-bd79a1`。本轮未运行测试、覆盖率、Go 编译或最终构建，也未编辑实现源码。

剩余风险：S15–S19 尚无运行时通过证据；主工作区 FD-028 改动当前阻止安全交付。修复后应重新提交 Worker 报告与新的独立 Tester/PM/Reviewer 收据链，依据当时的 FD 修订重新判定授权。
