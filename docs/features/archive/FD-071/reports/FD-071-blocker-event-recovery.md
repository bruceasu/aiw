# FD-071 事件恢复记录

<!-- aiw-data: FD-071-blocker-event-recovery.json -->

PM 在重新填写计划时错误地把 aiw fd new 产生的 Revision 2 写回 1，导致 design-ready 和 design-requested 同为序号 000002。latest_event 按文件名排序，错误选择已经 acknowledged 的 design-requested；Worker claim 被拒绝。现有实现 87652be 在 claim 失败后仍被写入并提交，不能记为已正常领取后的工作；这是本次执行失误，不是 handoff Skill 行为。

已检查两张实际回执和 CLI latest_event/refresh-worker/force-emit 实现。旧 Worker 事件为 pending，无运行 Agent；refresh-worker 要求 latest 是 pending Worker，但当前 latest 为 acknowledged Planner，故不盲目执行。没有编辑或删除回执，也没有重复失败命令。

PM 按 work-management 的明确恢复权限使用受管 force-emit design-ready，生成高于所有旧回执的新 Worker 事件；随后正常领取，核对已有改动并继续独立评审。此次只越过损坏的前置事件/claim链，不代表实现、审查或验证通过，也不豁免后续独立评审。旧回执及 override audit 保留。

当前状态：恢复待执行。下一步记录实际结果后解除本阻断。可复用改进：填写计划须保留 aiw 生成的 Revision，命令失败必须阻止依赖的写入；本次记录，不扩大为产品修复。

实际结果：force-emit 成功生成 FD-071-000003-design-ready，Worker 正常领取成功。audit 为 .ai/fd/FD-071/operations/4268e3a877be49a0b1c80b17caed9555.json。此阻断状态 resolved；旧事件保留、无运行 Agent 被接管，独立实现评审仍是交付门槛。
