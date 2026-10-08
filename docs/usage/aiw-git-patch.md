# `aiw git patch` 使用说明

```text
aiw git patch create changes.patch
aiw git patch create staged.patch --staged
aiw git patch create unstaged.patch --worktree
aiw git patch create transfer.patch --from main --to feature/topic
aiw git patch create transfer.patch --from abc1234 --to def5678
aiw git patch apply changes.patch
```

`create` 默认把已跟踪文件相对 `HEAD` 的暂存和未暂存改动写入 `.patch`，同时生成同名 `.patch.md`，其中列出补丁的改动统计、应用方法和失败建议。`--staged` 只取暂存区；`--worktree` 只取未暂存区。补丁采用 `git diff --binary`，可包含已跟踪二进制文件的改动。未跟踪文件不会自动纳入；先 `git add` 后使用 `--staged` 才能包含它们。空差异或输出文件已存在时，命令报错并避免覆盖。

跨设备传递时，可同时指定 `--from A --to B`；两端可以是 commit ID 或分支名，含义是从 A 的文件树变到 B 的文件树，与 `git diff A B` 相同，不自动取共同祖先。说明文件会记录输入的 ref 和解析后的完整提交 ID。目标设备应用时应有与 A 对应的文件版本，不必有相同的分支名。该模式不能与 `--staged` 或 `--worktree` 混用；单独给一个 ref 或无效 ref 会报错且不生成文件。

`apply` 先执行只读的 `git apply --check`，通过后才执行 `git apply`。成功后改动位于工作区，不会自动暂存、提交或改写历史。应用前可用 `git status --short` 查看目标仓库状态，应用后用 `git diff` 查看结果。

如果检查失败，命令会显示 Git 错误并尝试反向检查：反向检查通过意味着补丁可能已经应用；否则应检查现有改动、目标文件版本与补丁来源。预检失败时不会执行应用。实际应用失败时需先检查工作区是否有部分改动，再决定如何处理。不要在未知状态下执行 `git reset --hard`。命令不会自动使用 `--3way` 或 `--reject`，以免留下未经确认的冲突或部分应用。

此命令处理文件差异补丁，不处理 `git format-patch` 生成的邮件式提交补丁；后者通常使用 `git am`。
