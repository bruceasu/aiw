# FD-098: Allow independent upgrades of collection members

**Status:** Complete
**Revision:** 5
**Priority:** Medium
**Evidence policy:** Dual

## Problem

已安装的 aiw-all / aiw-core 1.0.0 通过 members 精确要求 aiw-main 1.0.0，
导致用户无法独立安装主程序 1.0.1。默认 update 同时选择成员最新版本和集合旧版清单，
也会产生精确版本冲突。用户明确要求：集成 package 不应阻止成员独立升级。
共享资源等 dependencies 是另一类真实依赖；本次不将其一并放宽。

## Options and decision

1. 移除集合记录：破坏集合管理和自动清理可达性，不采用。
2. 全面忽略 members 和 dependencies 版本：可能安装不兼容运行依赖，不采用。
3. 区分安装选取与持续约束：采用。集合清单保留精确成员版本作为显式整套安装配方，
   安装后成员关系只约束成员存在和清理归属，不锁定版本。dependencies 继续精确匹配。
   默认 update 优先使用其已选中的成员最新版本，不被集合配方降回旧版。

## Solution

- Operation.visit 在默认 update（无 --core/--all）中遍历集合成员时，若成员已经是
  更新根，按已选中的版本访问；真实 dependencies 始终传递精确 wanted 版本。
- preflight 最终状态校验拆分两种边：dependencies 要求存在且精确版本一致；
  members 要求存在，递归参与循环检查，但版本不再是持续约束。
- 显式 install 集合、update --core/--all 仍使用发行清单的精确选取，选择冲突仍失败。
  不自动扩大单组件更新，不隐式降级，不默默替换不兼容的运行依赖。
- 既有 members 记录不迁移、不改写；version 表示该集合当时选取的版本，
  当前成员版本通过其自身安装记录读取。members 名称仍决定 remove 保护和 autoremove 可达性。
- 保留 manual/auto、无载荷集合、事务/锁、平台/路径保护、dry-run 和部分失败语义。
- 这是用户明确授权的集合约束行为变更；不改归档格式、包格式、公共 CLI 或依赖语法。
- 本机 aiw-main 1.0.1 的共享资源 dependencies 仍可能被旧技能阻挡，不能将集合修复
  描述为已解决全部兼容问题或已安装成功。发行包构建和本机安装不属于本 FD 生命周期。

## Scope

范围：包图规划及最终校验、包管理使用文档和稳定规范。
排除：真实依赖范围/兼容版本机制、资源拆包、重新打包 24 个技能、安装端变更、
发布/安装、网络、测试执行和新增测试代码。无关联 Issue 或 OpenSpec change。

## Work items

- [x] 1.1 分离集合成员选择和持续约束，允许独立升级及默认 update；规模 small、难度 medium、无前置。完成证据：visit 区分两类边且仅默认更新成员根不传 wanted；final check 保留真实依赖精确匹配，成员只验存在和循环；Manager.remove/orphans 按名称处理关系，无需状态迁移。
- [x] 1.2 同步使用文档和稳定规范，解释历史成员版本与运行依赖边界；规模 small、难度 low、依赖 1.1。完成证据：使用文档和稳定规范明确集合配方与运行依赖差异，规范补充独立升级/默认更新/真实依赖失败场景；双格式实现报告记录编译及未执行检查。

## TODO

- 无剩余实现项。`FD-098-000005-verification-passed` 后已交付至 develop：`2a56bbe`，来源 `870c2f7`；CLI 验证交付并移除专用 worktree/分支，`aiw fd close FD-098 Complete` 已归档。下列阶段记录保留为历史。

- 独立设计审查 ready，见 `docs/features/reviews/FD-098-design.md`；Worker 已领取 FD-098-000003-design-ready。
- 文档、规范、compile-only 和一次最终只读 diff/status 检查已完成；独立实现静态审查通过，无阻断发现，见 `docs/features/reviews/FD-098-review.md`；待 PM 本地交付、清理并归档。

## Acceptance

1. aiw-all 和 aiw-core 1.0.0 已安装、成员 M 为 1.0.0 时，独立安装依赖兼容的
   M 1.0.1 成功，不更新/移除集合，不改其他成员和安装原因。
2. 默认 update 选择 M 1.0.1 而仓库集合仍为 1.0.0 时，members 不造成版本冲突；
   集合新增成员仍不补装，嵌套集合遵循同一规则。
3. 显式整套安装继续按清单精确选择；互相冲突的明确选取失败，不猜测兼容性。
4. 真正 dependencies 要求旧 M、依赖缺失或依赖循环仍在修改安装端前失败。
5. 集合保留一个缺失成员仍失败；remove 保留成员保护，autoremove 仍按成员名称可达性工作。
6. 不迁移旧安装状态，不伪造成员当前版本；dry-run 不写安装文件或记录。

## Verification

- 本地交付完成：`aiw git wt local-merge FD-098` 生成 develop 单父 squash `2a56bbe`，校验来源 `870c2f7` 后完成 worktree/分支清理；`aiw fd close FD-098 Complete` 成功。未发布、安装或推送。

- 计划：一次 `python scripts/compile.py`，设置 GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local，禁止网络和保留发行二进制；Python 改动另以同一 compile-only 命令批次调用内置 compile() 作内存语法编译，无 pyc 产物。
- 静态检查：图规划 visit/preflight、Manager.install/remove/orphans、保留记录、文档和规范，覆盖上述验收场景；只运行一次实现后只读 diff 检查。
- 未请求测试支持：测试、新增测试、最终包构建、格式化、lint、验证脚本和安装操作均不运行。
- 独立设计审查：由于改变安装状态约束兼容性，需要一轮有边界的静态设计审查；报告在 reviews 下，中英文字段 JSON 侧车按仓库约定。
- 实现审查由独立 Reviewer 完成；未运行的检查不得记录为通过。
- 实际 compile-only：`GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local python -c "from pathlib import Path; import runpy; p = Path('src/plugins/aiw-pkg/operations.py'); compile(p.read_text(encoding='utf-8'), str(p), 'exec'); print('Python syntax compile: OK'); runpy.run_path('scripts/compile.py', run_name='__main__')"`，退出码 0；Python 内存语法编译通过，脚本编译两个 Go 入口到 NUL，无最终二进制。
- 独立设计审查：`docs/features/reviews/FD-098-design.md`，ready，无阻断发现；实现报告：`docs/features/reports/FD-098-implementation.md`。
- 最终静态检查：`git diff 3ea5e78 -- <四个实现/文档/FD路径>`、`git diff --check`、`git status --short` 同一批次完成；无空白错误，无范围外改动。
- 独立实现审查：Reviewer `fd098-review-20261019-7d2c` 领取 `FD-098-000004-implementation-ready`，对比 `3ea5e78..9ba924b` 并追踪实际图规划、Manager 和文档/规范，全部验收获静态证据支持，无阻断发现。双格式报告：`docs/features/reviews/FD-098-review.md` / `FD-098-review.json`。Reviewer 未运行测试、编译、构建、安装或网络；运行场景仍未验证，真实共享资源依赖限制仍保留。

## Sources

- 用户本次命令记录及“实现组件独立升级机制。集成package不应该阻止成员升级。”
- `openspec/specs/aiw-packages/spec.md`
- `src/plugins/aiw-pkg/operations.py`、`manager.py`
- `scripts/package_definitions.py`、`docs/usage/aiw-pkg.md`

**Completed:** 2026-10-18
