# E07 配置快照与请求边界验证

Task workflow-automation / wi-0012 / 2.2。三项全部 passed，用户终端退出 0，见 [结果](verification-results/e07-preflight-terminal-20260925T044622374590Z/report.md)。

- TestNotificationOriginalSnapshotBinding：真实 Python preflight 接受原始配置；只追加注释，语义相同但原始字节改变，必须拒绝旧快照。
- TestNotificationDisabledProcessConfiguration：缺文件和明确禁用均通过真实进程返回 disabled/not-ready。
- TestNotificationRequestRejectedBeforeProcess：超 64 KiB JSON 和不可序列化请求必须在进程启动前拒绝；预取消 context 用于区分被绕过的请求校验。

使用 Python 3.11+ 执行 run-e07-preflight-terminal.py。固定三项 Go 正则、60 秒测试上限，关闭依赖下载，临时前置解释器目录，不改变系统 PATH。只创建临时 console 配置，不发送 Teams 通知，不自动重试。保存精确命令、输入、计划和日志。

## TODO 与 Verification

- [x] 编写三项测试及单次取证入口。
- [x] 离线生产 Go 编译 `python scripts/compile.py` 退出 0；该命令不编译测试源码，不代表测试通过。
- [x] 用户终端三项通过，347 项输入和原始证据哈希一致。

%% REMAINING: 运行中超时、真实 TLS/Teams、生产 PATH、完整 Store/宿主及 AC/AX 仍待证据。
