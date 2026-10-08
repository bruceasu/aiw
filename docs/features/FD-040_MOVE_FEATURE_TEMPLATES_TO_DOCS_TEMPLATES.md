# FD-040: Move feature templates to docs/templates

**Status:** Open
**Revision:** 3
**Priority:** Medium
**Evidence policy:** Dual

## Problem

`docs/features/` 目前同时存放 FD 设计、状态索引、活动证据和各种 Markdown/JSON 模板，目录职责混杂。模板路径也写在 FD 创建程序、技能说明、用户文档和 smoke fixture 中；单纯移动文件会让新 FD 创建和指导文档继续读取旧位置。

## Options and decision

| 方案 | 决定 | 理由 |
| --- | --- | --- |
| 只移动模板文件 | 不采用 | `aiw fd new`、技能和文档仍引用旧路径，迁移后会产生失效入口。 |
| 移动所有模板文件并同步活跃读取者和说明 | 采用 | `docs/features/` 保留 FD 与证据，`docs/templates/` 成为模板唯一规范位置；改动集中且不触及历史报告。 |
| 保留旧位置副本或读取回退 | 不采用 | 会留下双份模板并使后续维护位置不明确；`aiw fd new` 缺少模板时已有内置默认内容可生成新位置模板。 |

## Solution

将 `docs/features/` 顶层所有模板文档移至 `docs/templates/`，包括 FD、反馈、可选测试报告/授权/决策/风险评估模板及 JSON sidecar。更新 `aiw fd new` 的模板读取与默认模板生成位置、FD/测试技能、工作管理约定、用户用法文档、可移植布局说明和 smoke fixture 路径。FD 记录、FEATURE_INDEX、活动/归档 evidence 的目录和历史报告内容不变；不改 CLI 参数、依赖、公开接口或稳定工作流行为。

## Scope

范围仅限 `docs/features/` 顶层模板文件与引用其规范路径的活跃程序、技能和文档。排除历史 FD/报告/审查中的旧路径记录、测试执行、格式化、依赖与其他 FD 功能。

## Work items

- [ ] 1.1 将全部模板文件迁移至 `docs/templates/`，并更新 `aiw fd new` 的模板读取/生成位置。Size: S; Difficulty: Low; Dependencies: none. Completion: 原模板文件在新目录完整可用；创建逻辑只把默认模板写入新目录并从新位置读取。
- [ ] 1.2 更新技能、使用文档与可移植布局说明中的模板路径。Size: S; Difficulty: Low; Dependencies: 1.1. Completion: 所有当前指导入口将模板定位到 `docs/templates/`，FD 与 evidence 路径仍指向 `docs/features/`。
- [ ] 1.3 更新 smoke fixture 的模板源路径。Size: S; Difficulty: Low; Dependencies: 1.1. Completion: fixture 在临时仓库中复制新位置模板；本任务不运行 smoke 或测试。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- `docs/features/` 顶层不再包含模板文档，所有原模板均在 `docs/templates/`。
- `aiw fd new` 从 `docs/templates/TEMPLATE.md` 读取；模板缺失时将内置默认模板写入 `docs/templates/TEMPLATE.md`，不在 `docs/features/` 创建模板。
- 活跃技能、程序说明、用户文档及 smoke fixture 无失效的规范模板读取路径；历史证据保持不变。

## Verification

- 静态检查：核对迁移清单、模板读取/生成调用点及所有活跃引用；检查最终 diff 和文档路径一致性。
- 按仓库要求执行 Python 内存 compile-only 检查；不运行测试、smoke、最终构建、lint 或格式化。

## TODO

- [ ] 1.1 迁移模板并调整 `aiw fd new` 模板路径。
- [ ] 1.2 更新技能和使用文档中的模板路径。
- [ ] 1.3 更新 smoke fixture 模板路径。

## Sources

- Issue: none

相关规范：`openspec/specs/fd-workflow/spec.md`。依据：`plugins/aiw-fd.py` 的模板加载逻辑、`skills/fd-workflow/`、`skills/fd-test/`、`skills/work-management.md`、`docs/usage/aiw-fd.md` 与 `scripts/fd_smoke.py` 中现有模板路径。
