## MODIFIED Requirements

### Requirement: 推广与可恢复关联

promote MUST 要求有效的 APPROVED 决定，并把一个 Requirement 关联到一个 Task。已有关联只允许继续同一 Task 的推广。推广 MUST 生成 handoff，并通过共享需求工件生成流程准备 proposal、design、capability specs 和 tasks。系统 MUST 在工件通过来源、结构、需求覆盖、保护式写入及清单同步检查后才完成新的推广并记录 SPEC_DRAFTED；MUST NOT 把通用模板、失败调用或等待会话 Agent 的状态报告为完成。

系统 MUST 报告路由建议结果或其不可用诊断；无模型时可使用确定性配置或明确未配置状态，MUST NOT 因可选模型建议不可用而使已经通过交接验证的工件永远无法完成推广。后续执行的路由前置条件不因推广完成而被豁免。批准和推广仍各自要求原有正式授权，已授权范围内的工件补齐不要求重复批准。

#### Scenario: 推广批准需求

- **WHEN** 已批准需求通过创建预检并完成候选内容检查
- **THEN** 系统复用或创建目标 Task，写入表达实际需求的工件，同步清单并记录 SPEC_DRAFTED。
- **AND** 人工内容、清单身份、批准记录与来源关联保持完整。

#### Scenario: 无模型时等待接续

- **WHEN** 没有明确可用的生成模型或配置候选全部失败
- **THEN** 系统保留 Task 关联并输出交接文件实际路径与恢复入口，不把该次推广记录为 SPEC_DRAFTED。

#### Scenario: 恢复中断的推广

- **WHEN** 需求已关联 Task 且生成、写入或同步中断
- **THEN** 继续同一请求的可恢复步骤，拒绝不同 Task 关联，不重复创建 Task 或清单映射。

#### Scenario: 会话 Agent 完成交接

- **WHEN** Agent 提交绑定当前请求的候选且通过统一检查和同步
- **THEN** 系统完成同一推广，不要求为已批准范围再次批准，不启动实现。

#### Scenario: 历史完成记录缺少内容证据

- **WHEN** 显式恢复旧 SPEC_DRAFTED 需求且缺少生成接受证据
- **THEN** 系统保留历史状态与审批审计，报告需要复核，不把旧状态等同于本次内容检查通过。
