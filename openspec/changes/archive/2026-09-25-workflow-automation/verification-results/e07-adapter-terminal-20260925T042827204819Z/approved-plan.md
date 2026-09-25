# E07 受管通知适配器离线验证

Task workflow-automation；wi-0012 / 2.2；实现来源 E07。状态：四项 Python 测试及取证入口已准备，尚未运行。

依据 R4 与 SW29/SW30，新增 plugins/test_notify_managed.py，导入受管入口，不执行旧 CLI。每项阻断 socket、urlopen 和真实 subprocess.run；HTTPS opener/子进程结果均为替身，环境只保留虚构测试凭据，不读真实项目配置或 .env，不发送通知。

| 用例 | 覆盖 |
| --- | --- |
| test_typed_errors_and_http_classification | TLS/权限/有类型网络异常/未知阶段超时/普通异常及 HTTP 302/401/403/429/503 分类；每次只调用一次，无内部重试；异常和远端正文不进结果 |
| test_tls_redirect_and_attempt_only_response | 使用证书及主机名验证默认上下文；忽略代理、禁止重定向；不读响应正文；200 只表示 attempted；请求体使用虚构凭据 |
| test_frozen_target_and_protocol_reject_before_dispatch | 目标/配置/正文摘要不匹配时发送前拒绝；重复字段、多 JSON、非法 UTF-8 拒绝 |
| test_child_protocol_and_environment_only_credential | 固定受管脚本 argv、stdin 只传凭据引用，值只在环境；不转发无关变量；结果 attempt 不匹配变 unknown，原 content_digest 保留 |

正常终端执行一次，预计一分钟内完成：

```powershell
python C:\Users\svictor\workspace\tools\aiw\.wt\workflow-automation\openspec\changes\workflow-automation\run-e07-adapter-terminal.py
```

脚本使用当前 Python：`python -I -B plugins/test_notify_managed.py -v`，cwd 为 worktree，60 秒上限，仅该文件四项；不执行 Go 全包测试，不自动重试。保存两个生产插件、测试文件、runner 和计划的摘要及 Python 版本、argv、时间、stdout/stderr。本组显式授权不等于真实通知授权或产品 grant。

## TODO 与 Verification

2026-09-25 按仓库流程离线执行 `python scripts/compile.py`，退出 0，清理临时产物；该入口只编译生产 Go 程序，不覆盖 Python 测试。新增 Python 文件仅 AST 静态核对，未执行测试、网络、真实通知或 Git 写入。

- [x] 四项离线测试及单次取证脚本已准备。
- [ ] 取得并核对本组实际结果。

%% REMAINING: 真实 TLS 握手、HTTP 服务、子进程隔离执行、显式 CA/配置解析全边界及 Go 适配器端到端协议仍待覆盖；不证明消息送达。此前 Core 状态机测试的 60 秒一次网络重发规则不在本组重复计时。
