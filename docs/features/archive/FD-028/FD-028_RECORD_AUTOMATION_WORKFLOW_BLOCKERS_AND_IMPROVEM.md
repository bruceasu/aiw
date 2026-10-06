# FD-028：记录自动化流程阻塞与改进

**Status:** Complete
**Revision:** 9
**Priority:** Medium  
**Test policy:** Independent  
**Evidence policy:** Dual

## Problem

FD 自动化流程遇到无法继续的情况时，目前会在对话或 handoff 中描述 Gate，内容和粒度不一致，也没有稳定位置沉淀阻塞原因、采取过的恢复措施及最终解决方案。后续流程无法据此识别重复问题并改进提示、检查步骤或恢复路径。

## Options and decision

| 方案 | 取舍 |
| --- | --- |
| 只在最终回复中说明 | 成本低，但记录不持久，也难以在后续流程中检索。 |
| 每次阻塞都改 FD 正文 | 记录可见，但会改变 FD digest，可能干扰仍待处理的 handoff。 |
| 使用独立的阻塞反馈记录，并由 FD 引用 | 记录可持久归档，不改写 FD 正文或收据；可供后续流程检查和归纳。 |

选择独立反馈记录。每条记录至少关联 FD 与阶段/角色，描述阻塞表现、根因、已尝试的恢复、解决方案或未解决原因、是否需要人工决策，以及可复用的流程改进建议。没有确认的根因或方案必须标为未知/待解决，不得猜测。阻塞尚未解除时先记录已知事实和当前状态，解除后补记结果。

在自动化流程重新开始或完成归档前，检查该 FD 的反馈记录；若记录指出可复用的流程缺陷，提出针对技能、稳定规范或模板的具体修正，并按正常授权流程处理。记录本身不授权流程自行修改规则、跳过 Gate 或执行高风险操作。

## Solution

为阻塞反馈定义轻量、可归档的 Markdown 记录格式，并提供同名 JSON sidecar，满足 Dual evidence 约定。记录以 FD ID 和唯一后缀命名，保存在该 FD 的 `reports/` 下，并在 FD 自动流程停止点写入；FD 归档时随该 FD 的报告一起归档。流程规范说明创建时机、必填字段、未解决记录方式、恢复后补全方式，以及后续运行如何检查和识别可复用改进。

## Scope

- 定义自动化受阻反馈记录模板及结构化 JSON 字段。
- 更新 FD workflow 技能，规定阻塞、恢复和续跑时如何创建、补全、引用及复查反馈。
- 更新稳定 FD workflow 规范，明确记录的持久性、事实准确性和权限边界。
- 确保现有报告与归档流程能保留反馈记录，且不改变 handoff digest 或既有证据格式。
- 不实现自动根因推断、自动应用流程修改、独立的新反馈数据库或新的 CLI 命令。

## Work items

- [x] 1.1 定义反馈记录字段及 Markdown/JSON 模板。Size: S; difficulty: Low; dependencies: none. Complete when 一条记录可关联 FD、阶段、阻塞原因、尝试、解决/未解决状态、人工决策和改进建议，且 Markdown 与 JSON 字段语义一致。
- [x] 1.2 在 FD 自动流程技能中加入阻塞时记录、恢复后补全、续跑时复查的操作规则。Size: S; difficulty: Medium; dependencies: 1.1. Complete when 所有会停止自动流程的 Gate 都有明确记录步骤，并保留未解决状态及权限限制。
- [x] 1.3 更新稳定 FD workflow 规范及归档约定。Size: S; difficulty: Medium; dependencies: 1.1. Complete when 反馈记录可随 FD 归档、不会要求修改正在使用的 FD 收据，且已有 FD 记录兼容。
- [x] 1.4 静态审阅模板、技能与规范的一致性并更新 Verification。Size: S; difficulty: Low; dependencies: 1.1–1.3. Complete when 每项验收均有可审阅的静态证据，未授权的测试或运行检查保持未执行。
- [x] 1.5 汇总本次任务的反馈处理经验到 `docs/feedback.md`。Size: S; difficulty: Low; dependencies: 1.1–1.4. Complete when 文档记录本次实际阻塞、恢复与可复用改进建议。

## TODO

- [x] 模板、自动流程技能、稳定规范与共享约定已更新。
- [x] 本次失配交接的真实反馈及经验已记录。
- [x] 静态核对归档命名规则；待独立 Tester/Reviewer 按交接流程完成各自报告。

## Acceptance

- 自动化流程受阻时，能创建一条可追溯到 FD 和具体阶段/角色的独立反馈记录。
- 记录包括事实表现、根因（或未知）、尝试过的恢复、解决方案/未解决状态、人工决策和后续改进建议；恢复后可补充实际结果。
- 反馈记录不会要求在 handoff 仍有效时修改 FD 正文或 receipt digest。
- 后续自动流程能复查已有反馈并指出可复用的流程改进；是否修改规则仍遵循既有授权与 review 流程。
- Markdown 与 JSON sidecar 满足仓库 Dual evidence 约定，且反馈文件能与 FD 一起归档。
- 历史 FD、报告和现有 handoff 行为保持兼容。

## Verification

- 静态证据：`docs/features/BLOCKER_FEEDBACK_TEMPLATE.md` 与同名 JSON 模板均含 FD、事件、阶段/角色、事实、根因、尝试、状态、人工决策、解决和建议字段；本次实际记录使用同一结构。
- 静态证据：两份 `fd-workflow/SKILL.md` 的通用 Gate 段覆盖 preflight、设计、Worker、Tester、PM、Reviewer、交付、归档及清理，并要求恢复补记和续跑复查。
- 静态证据：`plugins/aiw-fd.py` 的 `evidence_moves` 遍历 `reports/` 中 `FD-028-*.md/.json`，现有归档路径可搬运反馈；稳定规范与共享约定明确不改有效收据。
- 未运行测试、最终产物构建、格式化、lint 或网络命令。此次仅改文档，编译检查不适用；实际归档结果待完成独立评审后核对。

## Sources

- 用户请求：自动化流程受阻时记录原因和解决方案，以便改进流程。
- `.agents/skills/fd-workflow/SKILL.md`：自动流程、Gate、续跑与归档约定。
- `openspec/specs/fd-workflow/spec.md`：FD handoff、证据与归档稳定要求。

**Completed:** 2026-10-06
