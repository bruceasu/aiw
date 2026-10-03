# FD-026 独立审查报告 r2

<!-- aiw-data: FD-026-review-r2.json -->

## 结论

**审查通过。** R1 指出的两个夹具路径已改为短名插件目录；静态追踪支持 FD 的命令发现、入口转发、安装布局和当前用法要求。PM 允许在零测试预算下继续静态审查，但 13 个场景仍全部未运行，本结论不把它们记作行为测试通过。

## 范围与证据

- FD：`FD-026`，当前 revision 11；来源事件：`FD-026-000011-test-accepted`，FD digest：`585af32ebe27f2378f16ec9baa19329046c9957377a0d8da10a1ddedff35033e`。
- Reviewer session：`fd026-reviewer-r2-20261004-5d72`；Worker / Tester session：`fd026-worker-r2-20261004-9f4e` / `fd026-tester-r2-20261004-a71c`。
- 审查基线：`c9b6d51cd2308e1ffbaf51755d697f6acda3ad83`；该提交之后的共享工作区有大量其他未提交改动，没有 FD 专属提交。本审查只检查 FD 相关路径，不提交或归档。
- PM r2 接受 Tester 报告作为准确记录并允许继续静态审查：13/13 场景未运行，需求覆盖率 0%，分支覆盖率未测量。未运行项目不构成通过证据。

## 发现

无新增问题。R1 的两项发现均已修复：

- `tests/fd021_gateway_acceptance.mjs` 的安装目录为 `C:/green/aiw/plugins/aiw-gw`。
- `tests/fd021_gateway_live.mjs` 的配置路径为 `D:/green/aiw/plugins/aiw-gw/gateway.json`。

## 验收判断与残余风险

- `aiw gw start|stop`：`cmd/aiw` 将未内置的子命令交给插件发现；`internal/plugin` 按 `aiw-<name>` 查找，入口名为 `aiw-gw.py`。入口把缺省配置指向同目录 `gateway.json`，并转发 start/stop 参数。静态链路完整，未实际调用。
- 新安装布局：`build.bat` 的网关二进制输出、安装复制目标和 `plugins` 复制路径均指向 `aiw-gw`。构建和安装脚本未运行。
- 旧命令与当前用法：旧插件入口已删除；当前网关说明、客户端帮助及启停脚本使用 `aiw gw`。旧路径字符串仍出现在 FD 历史/本 FD 描述及新 README 中保留的绝对状态目录示例；它们不是旧命令入口或命令用法。该状态路径示例维持 FD 的“既有绝对状态路径不变”决策。
- 配置 schema、HTTP 契约、绝对状态路径及旧配置不迁移：本轮短命令改动未显示对实现契约的修改，但没有运行时证据验证配置加载或 HTTP 行为。旧配置仍需由操作者按 FD 方案处理。
- 本轮未运行测试、覆盖率、compile-only、网关进程、构建、安装、部署或网络命令。Worker 记录的 `go build -o NUL ./cmd/aiw` 未能确认退出码，不能当作通过。
- 工作区存在多项范围外改动，且新插件目录和测试夹具在当前 Git 基线上为未跟踪路径；无 FD 专属提交可供独立提交级比较。这限制了对共享工作区来源和整合状态的确认。

## 实际命令

- `Get-Content -Raw 'C:\Users\suk\.agents\skills\fd-review\SKILL.md'`
- `Get-Content -Raw 'skills/work-management.md'`
- `rg -n -C 4 'FD-026-000011|Pending Verification|revision|digest|Test policy|Evidence policy|Acceptance|Verification|Work Items' docs/features/FD-026_GATEWAY_SHORT_COMMAND.md`
- `aiw fd --help`
- `aiw fd show FD-026`
- `aiw fd claim FD-026 FD-026-000011-test-accepted --session fd026-reviewer-r2-20261004-5d72`
- `git status --short`
- `git rev-parse HEAD`
- `git diff -- build.bat plugins/aiw-agent-gateway plugins/aiw-gw plugins/aiw-ai/README.md plugins/aiw-ai/aiw-ai.mjs program/agent-gateway/README.md scripts/start-gateway.bat scripts/stop-gateway.bat tests/fd025_shutdown_blackbox.py tests/fd021_gateway_acceptance.mjs tests/fd021_gateway_live.mjs`
- `rg -n --glob '!docs/features/archive/**' --glob '!docs/features/reviews/**' 'aiw agent-gateway|aiw-agent-gateway|plugins/aiw-agent-gateway' docs plugins program scripts build.bat`
- `rg -n -C 3 'build_plugins|build_gateway|install_gateway|plugin' build.bat`
- `rg -n -C 2 'DiscoverPlugin|aiw-' internal/plugin/discover.go cmd/aiw/main.go`
- `rg -n 'plugins/aiw-gw|aiw gw|agent-gateway' scripts/start-gateway.bat scripts/stop-gateway.bat plugins/aiw-ai/README.md plugins/aiw-ai/aiw-ai.mjs program/agent-gateway/README.md`
- `rg -n -C 2 'aiw-gw|aiw-agent-gateway|gateway.json' tests/fd021_gateway_live.mjs`
- 只读查看 FD、Worker r2、Tester r2、PM r2、Reviewer r1 报告及插件入口、README、构建脚本、两个夹具相关内容。

未执行测试或覆盖率命令、网关进程、compile-only、最终构建、安装、部署或网络访问。
