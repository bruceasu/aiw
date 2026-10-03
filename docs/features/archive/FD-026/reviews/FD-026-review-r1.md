# FD-026 独立审查报告 r1

<!-- aiw-data: FD-026-review-r1.json -->

## 结论

**请求修改。** 核心短命令路径在静态代码中连通，但两个现有验收夹具仍硬编码旧安装目录；这与 FD 将当前测试夹具迁往短名目录的范围不符。未运行场景仍按 PM 例外记为未验证，不作为通过证据。

## 审查范围

- FD：`FD-026`，revision 7；来源事件：`FD-026-000007-test-accepted`。
- Reviewer session：`fd026-reviewer-20261004-6b9a2d`。
- Worker / Tester session：`fd026-worker-20261004-c19b` / `fd026-tester-20261004-a7e31c`。
- 审查基线：`c9b6d51cd2308e1ffbaf51755d697f6acda3ad83`，并检查当前工作区相关路径；工作区存在其他未提交改动，没有 FD 专属提交可供比较。
- PM 决策接受需求场景覆盖率 0%、分支覆盖率未测量的例外。4 个场景均未执行，compile-only 结果也未确认；本审查不将其描述为通过。

## 发现

### FD026-R1-001：验收夹具仍指向旧插件目录

`tests/fd021_gateway_acceptance.mjs:9` 将安装目录固定为 `C:/green/aiw/plugins/aiw-agent-gateway`，后续从该目录读取 `agent-gateway.exe`。新安装布局改为 `plugins/aiw-gw/`，因此此夹具无法针对新布局读取网关二进制。

### FD026-R1-002：Live 夹具仍从旧目录读取配置

`tests/fd021_gateway_live.mjs:46` 从 `D:/green/aiw/plugins/aiw-agent-gateway/gateway.json` 读取网关配置。迁移到短名安装布局后，该路径与新插件目录不一致，夹具不能验证新布局。

建议将这两个当前夹具的插件目录引用更新到 `aiw-gw`，并保留其既有状态目录字段；修复后再由 Worker 更新 FD 与实现证据并重新交审。

## 验收证据

- `aiw gw start|stop`：静态支持。插件发现按 `aiw-<name>` 匹配，入口文件为 `aiw-gw.py`；入口在缺省配置时将同目录 `gateway.json` 传给 start/stop。运行行为未验证。
- 新安装布局：静态支持。`build.bat` 的网关构建输出和安装复制目标均使用 `plugins\aiw-gw\`。脚本未运行。
- 旧命令入口和当前用法：静态支持。旧插件入口已删除，当前插件说明、`aiw-ai` 帮助及启动/停止脚本使用 `aiw gw`；发现中的旧目录引用是上述两个测试夹具，不是命令用法。
- 配置 schema、HTTP 契约及状态路径：本次入口/布局改名路径未显示对它们的修改；未运行配置加载或 HTTP 行为验证。旧安装配置不自动迁移仍是已记录风险。

## 命令与未执行检查

实际执行的命令：

- `aiw fd --help`
- `aiw fd claim --help`
- `aiw fd emit --help`
- `aiw fd show FD-026`
- `aiw fd claim FD-026 FD-026-000007-test-accepted --session fd026-reviewer-20261004-6b9a2d`
- `git rev-parse HEAD`
- `git diff -- build.bat plugins/aiw-agent-gateway plugins/aiw-gw plugins/aiw-ai/README.md plugins/aiw-ai/aiw-ai.mjs program/agent-gateway/README.md scripts/start-gateway.bat scripts/stop-gateway.bat tests/fd025_shutdown_blackbox.py`
- `rg -n --glob '!docs/features/archive/**' --glob '!docs/features/reviews/**' 'aiw agent-gateway|aiw-agent-gateway|plugins/aiw-agent-gateway' docs plugins program scripts tests build.bat cmd internal`
- `git diff --check -- docs/features/FD-026_GATEWAY_SHORT_COMMAND.md docs/features/reviews/FD-026-review-r1.md docs/features/reviews/FD-026-review-r1.json`
- 对 FD、实现/测试/PM 报告、插件入口、插件发现代码、构建脚本、网关入口及两个命中夹具进行只读查看。

未执行测试、覆盖率、网关进程、compile-only、最终产物构建、安装、部署或网络命令。剩余风险是启动/停止、安装行为、配置加载和旧命令不可用性尚无运行时证据。
