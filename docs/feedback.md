# 流程反馈

## FD-028：自动流程阻塞记录

- 本次任务把阻塞反馈放在 FD 的独立 `reports/` 文件中，以 Markdown 和同名 JSON 同时保存事实、未知项、恢复结果与改进建议；归档沿用现有 FD 前缀文件搬运规则。
- 原 `design-ready` 事件在 FD 增加 Work Item 后失配，Worker 领取被拒。使用 `refresh-worker` 创建新交接，提交更新后的 FD 计划，并对齐 worktree 后成功领取。详情见 `features/archive/FD-028/reports/FD-028-blocker-20261006T160143Z-6c94b2.md`。
- 后续改进建议：创建 worktree 前先核对交接摘要；失配时先刷新待领取交接并提交计划，再建立或对齐 worktree。此建议仍需按正常流程评审，反馈记录不授权规则变更。
- 交付阶段发现 PATH 中的旧版 `aiw.exe` 把 feature 分支快进到父分支。核对两端状态后恢复父分支，并用仓库当前 `src/plugins/aiw-wt.py` 生成含 `FD-Source` 的单父 squash 提交。详情见 `features/archive/FD-028/reports/FD-028-blocker-20261006T160833Z-delivery.md`。
- 后续改进建议：交付前核对 `aiw` 命令来源与仓库脚本版本，避免旧版工具改变交付形态；建议仍需独立评审。
- 结案结果：独立 Tester 将 8 项纯文档验收列为静态排除，PM 接受覆盖率不适用例外；独立 Reviewer 第 1 轮静态评审通过。反馈 Markdown 与 JSON 均已随 FD 归档。测试、编译和最终产物构建未运行。
- 归档后清理出现新的未解决问题：worktree 中的 `.ai` junction 清理报错；随后父工作区 `.ai` 目录被发现为空，原有收据缺失。Git 交付、FD 归档与报告仍在。未推断删除发生于哪一步，也未重建收据。详情见 `features/archive/FD-028/reports/FD-028-blocker-20261006T160953Z-cleanup.md`；需由维护者评估可信备份恢复和其他 FD 的运行时状态。
