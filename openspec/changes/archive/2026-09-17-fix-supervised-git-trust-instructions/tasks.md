## TODO

- [x] 1.1 分离 supervised 请求的通用 Git 查询指令与 scope-review 编辑限制，为所有通过当前预检的工作项注入限定目录指令；复用路径转义并保留各自实现范围。
- [x] 1.2 完善当前预检信任依据缺失或不匹配时的派发前失败处理，沿用既有状态清理和诊断，禁止空指令静默继续及扩大信任。
- [x] 1.3 在现有请求组装/provider 替身边界添加普通实现项、审查项、旧 handoff、预检缺失/失败、路径特殊字符的离线回归测试；不默认运行测试。
- [x] 1.4 更新监督 Git 查询与旧阻塞项恢复文档，完成静态调用链核对和一次适用的 compile-only 检查，记录 Verification 与剩余风险；不自动恢复其他任务。

## Verification

- 已读取普通实现项的 Session 故障日志及最终结果，确认 Git dubious ownership 的账户差异。
- 已静态核对预检环境、Session 环境传递、标题条件及通用提示组装调用点。
- 用户已确认沿用现有 Task Agent 请求组装和 Git 指令测试边界。
- 设计为 FD_NOT_REQUIRED；沿用共享 spec-driven 模板，未运行 OpenSpec validator。
- 清单落盘后执行 aiw task workflow sync fix-supervised-git-trust-instructions，以返回结果核对 Work Item 映射。
- 本轮仅创建规划工件，未实施、运行测试/编译/构建、修改 Git 配置或重启故障任务。
- 1.2 静态核对：预检环境现在必须精确包含单个 `safe.directory` 配置；缺失、重复或额外配置均在派发前进入既有 workspace-access Gate 与失败报告路径，未运行测试或编译。
- 1.3 静态核对：`internal/commands/task/supervised_git_test.go` 现覆盖普通中文实现项与 scope-review 请求的最终指令、旧 handoff 路径不复用、缺失/异常预检依据拒绝派发，以及 Windows/POSIX 特殊路径的命令前缀；未运行测试或编译。
- 1.4 静态核对：`docs/supervise.md` 现说明当前预检目录生成的命令级 `safe.directory` 与同目录 `-C` 只用于只读查询；恢复历史阻塞项需先确认修复版本和限定查询、解决 Gate、按需 reopen，再显式启动，且不会自动恢复其他任务。已核对 `scripts/compile.py` 会以工作树本地缓存执行 `go build` 且不保留二进制；按交接约束，该 compile-only 检查保留给 supervisor，未由本 Work Item 执行。

## Gates and Evidence

- 规划 Evidence：现有原始故障输出、工作区预检与指令组装源码、稳定 workflow-supervision spec、用户确认的测试边界。
- 映射 Gate：workflow sync 必须成功，不创建 Attempt、不 claim 租约、不推进执行。
- %% VERIFICATION: 原故障尚未通过修复后的同沙箱运行验证；本轮不声称临时方案或未来修复已经有效。
- %% VERIFICATION: 本 Work Item 未运行 compile-only；supervisor 负责执行冻结 Compile Plan 并决定是否接受。实际限定查询与旧阻塞项恢复仍待该受控验证。
