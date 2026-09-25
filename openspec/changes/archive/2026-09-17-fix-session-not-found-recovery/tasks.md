## 1. TODO：修复错误分类

- [x] 1.1 在 Workflow 请求准备与既有 repair 入口使用 Session 领域缺失判断，保留同名自动创建及原恢复资格；定向核对相关 Load 消费点，保留非缺失错误和归档约束。

## 2. TODO：回归与恢复说明

- [x] 2.1 在现有 Workflow 命令/请求准备边界编写首次缺失创建、重复准备复用、创建失败、异名缺失、损坏/不可读/冲突/归档不误建，以及原 Attempt 恢复保留审计的聚焦测试；不默认执行。
- [x] 2.2 静态核对调用链、更新恢复说明和 Verification，实现后按预算执行 compile-only；说明使用修复版程序 diagnose/status 后显式 start，保留原 Task/worktree，不自动重启。

## Verification

- 规划依据：用户提供 supervisor 启动失败日志；此前静态核对 Session sentinel、Store.Load/Resolve、请求准备与 repair 分支，以及 Go 本地 os.IsNotExist 实现。
- 用户已确认主要测试边界为现有 Workflow 命令入口，覆盖首次 Session 自动创建及损坏/归档记录不误建；本轮不运行测试。
- AIW 创建命令：`aiw new fix-session-not-found-recovery --backend auto --allow-unrelated-dirty`。Task 与 change 同 ID，primary 工作区，不创建实现分支或工作树。
- 实际只读命令：Get-Content、rg、Test-Path；已读取 to-spec、工作管理契约、上下文及相关稳定规格。通过 openspec.cmd instructions 获取四类工件模板，apply_patch 写入本 change。
- 规划收尾：`aiw task workflow sync fix-session-not-found-recovery` 成功同步三项 Work Item，全部 authored=open/core=ready，状态 DRAFT/queued；`openspec.cmd validate fix-session-not-found-recovery --strict` 返回 valid。该校验由用户显式调用的 to-spec 技能要求，不是代码测试。已静态确认四类工件存在且 task.toml.id 与 change 目录一致，Design Readiness 为 FD_NOT_REQUIRED，没有未决设计 Gate。
- 未实现生产代码、未执行测试/编译/最终构建/网络/Git 写操作，未启动或恢复任何 supervisor，未修改受阻任务记录。
- %% 2.2 的恢复操作说明与最终静态核对仍待完成；不将未执行的新回归或未完成的清单项视为整个 change 完成。
- 1.1 已实现：`advanceWorkflow` 与 `repairWorkflowState` 仅以 `errors.Is(err, session.ErrSessionNotFound)` 识别 Session 记录缺失；同名绑定仍沿用既有创建后重读，repair 仍要求既有写租约、无 prepared request 和 Attempt 匹配。损坏、不可读、冲突或归档错误不会进入自动创建/恢复分支。
- 1.1 静态核对：`session.Store.Load` 委托 `Resolve`，而 `Resolve` 只在未找到合法位置的记录时返回 `ErrSessionNotFound`；该 sentinel 包装 `os.ErrNotExist`，因此保留包装错误识别，同时不把其他普通文件缺失扩大为 Session 缺失。
- 1.1 未运行测试；2.1 的聚焦回归夹具仍待实现。已执行 `python scripts\\compile.py`，退出码 0；脚本仅在临时目录中编译 `main.go`，未保留发行物。
- 经用户授权执行 `go test ./internal/commands/task -run '^TestWorkflow'`；首次在测试包 setup 前因 Go 默认缓存 `C:\\Users\\svictor\\AppData\\Local\\go-build\\...` 被 Windows 拒绝访问而失败。经用户批准以受控权限重试同一命令后通过（0.749 秒）；未更换缓存路径，未启动 supervisor 或访问网络。
- 2.1 已新增 `workflow_session_recovery_test.go`：覆盖同名缺失创建和重复 prepared request 复用、创建前目录故障不产生 Attempt、异名缺失/缺少状态/身份冲突不重建、归档记录不重建活动 Session，以及 repair 对缺失记录保留 Attempt 审计、对损坏记录不改变 Attempt 的分支。不可读路径继续采用既有权限错误边界，避免依赖 Windows ACL。
- 2.1 已执行 `python scripts\\compile.py`，退出码 0；该 compile-only 不会编译或运行 `_test.go` 文件。经用户授权执行 `go test ./internal/commands/task -run '^(TestAdvanceWorkflow|TestRepairWorkflowState)'`：首次因夹具使用相对临时工作区、触发真实创建路径的项目根解析失败而失败；将夹具改为临时绝对工作区后，同一命令重跑通过（2.464 秒）。
