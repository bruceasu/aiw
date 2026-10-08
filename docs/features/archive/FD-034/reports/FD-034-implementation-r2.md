# FD-034 实现报告 r2

<!-- aiw-data: FD-034-implementation-r2.json -->

Worker 会话：`fd034-host-20261008-5c8b71`。原 Worker 来源事件：`FD-034-000005-design-ready`。本报告补充用户在 `FD-034-000009-implementation-ready` 待认领期间提出的跨设备比较需求；此前报告 `FD-034-implementation-r1.md` 保留。

## 新增范围

- `aiw git patch create FILE.patch --from A --to B` 接受两个提交 ID 或分支，先分别解析为固定提交 ID，再导出直接的 A 到 B 文件差异。
- 两个参数必须成对出现，不能与 `--staged` 或 `--worktree` 混用；无效 ref 或空差异不写文件。
- 补丁说明记录输入 ref、解析后的完整提交 ID、改动统计及目标设备需要相应基础版本的提示。帮助、使用文档、稳定规格和 FD 接受条件同步更新。

## 证据和限制

静态检查了参数互斥、ref 解析顺序、固定 ID 的 Git 调用、无效 ref/空差异的写入前退出路径以及文档一致性。`python -m py_compile plugins/aiw-git/git-patch.py` 通过。

未运行真实 `git diff`/`git apply` 场景、测试、构建、格式化或 lint。跨设备补丁应用的结果取决于目标工作区与起点文件版本的匹配程度，仍需独立 Tester 记录覆盖情况。此前待认领的 Tester handoff 绑定旧 FD 修订，应由 PM 刷新，不能把旧收据解释为本次修订的验证。
