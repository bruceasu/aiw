# FD-028 自动流程阻塞反馈

<!-- aiw-data: FD-028-blocker-20261006T160833Z-delivery.json -->

## 定位

- FD：`FD-028`
- 阶段与角色：本地交付，PM
- 关联交接事件：`FD-028-000009-verification-passed`
- 记录时间：2026-10-06T16:08:33Z
- 记录者／会话：`fd028-pm-20261007-6c94b2`

## 阻塞事实

- 观察到的表现及停止位置：`aiw wt local-merge FD-028` 将 feature 分支快进到 `develop`，父分支继承了分项提交，未生成规定的单父 squash 提交；因此暂停归档。
- 已确认的根因：PATH 中的 `C:\green\aiw\aiw.exe` 执行了旧版合并行为；本仓库 `plugins/aiw-wt.py` 的同名操作使用 `git merge --squash` 并写入 `FD-Source`。
- 已尝试的恢复及结果：核对父分支原 HEAD、当前 HEAD、FD HEAD 与两工作区干净状态后，将父分支恢复至原 HEAD `7b16ff1`；运行 `python plugins/aiw-wt.py local-merge FD-028`，创建单父 squash 提交 `301b36e`，其 `FD-Source` 为 `91e3fbc`。
- 先前状态：未解决，不能归档或清理分支。
- 是否需要人工决策：否；误操作只移动了本地父分支，FD 分支及提交完整保留，可在核对状态后恢复。

## 结果与改进

- 当前状态：已解决。
- 实际解决方案：使用仓库当前版本的 worktree 脚本完成规定的 squash 交付。
- 解决时间：2026-10-06T16:08:33Z
- 剩余风险或下一步：归档后核对反馈 Markdown/JSON 一同搬运，并验证父分支 `FD-Source` 与当前 FD HEAD 相同。
- 可复用的流程改进建议：自动流程在交付前核对 PATH 中 `aiw` 的版本或命令来源与仓库当前脚本一致；发现不一致时使用已审阅的仓库脚本并记录原因。
- 建议处理状态：已记录，尚未作为独立规则变更评审。
