# FD-103 独立实现审阅 R1

<!-- aiw-data: FD-103-review-r1.json -->

结论：changes-requested。Reviewer session：fd103-reviewer-20261021-r1。
来源事件：FD-103-000004-implementation-ready。比较基线：08f8fe5；审阅 HEAD：c3766b6；产品实现提交：3e0d89c。

## 必须修复

### R1 / P2：历史 receipt 的旧设计路径仍被直接使用

位置：src/plugins/aiw-fd.py:1230、857、930。

要求：FD Acceptance 和稳定规格要求迁移后保持已签发记录及新旧布局操作兼容。safe_artifact 已提供旧路径映射，且 receipt 不应为了迁移被改写。

实际：dispatch 先以 resolve_fd 找到新位置并验证摘要，但启动配置的 AIW_FD_ROLE_RUNNER 时仍传入 event["fd_path"]。旧路径的设计被移入 FD 目录后，即使 claim/resume 的设计摘要检查通过，runner 接到的仍是不存在的旧路径。此外，legacy Independent 路径的 validate_test_authorization 和 validate_test_report 仍直接读取 base / previous["fd_path"]，遇到相同迁移记录会 FileNotFoundError，阻塞合法测试证据交接。这不是要求执行测试，而是静态可确认的路径解引用错误。

最小修复：保持 receipt 原文、摘要和溯源不变；runner 参数使用已经解析并验证的设计路径，两个读取点通过既有兼容解析或按 FD ID 解析当前设计。检查同类读取点，确保不会绕过路径安全、歧义和摘要门禁。

## 其余静态证据

- active_files、archived_files、resolve_fd、git-wt 的 workspace/add 以及四种 shell 补全覆盖目录和旧平铺发现。
- update_index 仅写所属 FD 的 active/archive 局部导航；状态不进入索引内容。FEATURE_INDEX 仅在新建且缺失时初始化，常规状态变化无共享受跟踪状态表写入。
- evidence_moves 在移动前拒绝新旧证据目标重名；close、reopen、归档 request-review 快照并恢复两处索引。空证据目录不生成残留活动导航；历史重复设计导航保留多个链接。
- Git 合并/冲突恢复逻辑未被更改，没有 merge driver 或自动选择一侧。静态支持消除共享索引带来的必然冲突，真实源码冲突仍由原 Git 路径处理。
- 当前使用文档和稳定规格说明状态表格式变化、物理路径及安装版本边界。102 个历史归档目录仅新增导航；历史正文不作为本次改写对象。

## 命令与限制

实际执行：Get-Content -Encoding utf8、Select-String、rg 定向读取代码/契约/报告；git diff --stat 08f8fe5..c3766b6 和 git diff 08f8fe5..c3766b6 -- <上述变更文件>；Get-ChildItem 定位说明及 receipt；python src/plugins/aiw-fd.py claim FD-103 FD-103-000004-implementation-ready --session fd103-reviewer-20261021-r1（成功）。首次读取误用根 reports 路径及 worktree 本地 runtime 路径，之后改读实际 FD 目录与父工作区共享 runtime；首次未指定 UTF-8 的中文显示异常已用 UTF-8 重新读取。

Worker 报告记载 Python compile() 和离线 Go completion 包编译通过；本 Reviewer 未重复编译，未运行测试、最终构建、并发运行、失败注入、交互 shell、网络或安装。声明遵循 skills/testing-methods.md；没有新增测试证据需评估。

剩余风险：并发执行、归档失败恢复和 shell 交互的结论限于静态分析及 Worker 编译记录；R1 是已确认兼容缺陷，修复前不能通过。
