# E07 离线适配器验证结果

Task workflow-automation；wi-0012 / 2.2；实现来源 E07。用户正常终端执行一次，退出 0，unittest 输出 Ran 4 tests / OK：

- test_child_protocol_and_environment_only_credential
- test_frozen_target_and_protocol_reject_before_dispatch
- test_tls_redirect_and_attempt_only_response
- test_typed_errors_and_http_classification

实际解释器为 `C:\Python39\python.exe`，版本 3.9.13；命令 `python -I -B plugins/test_notify_managed.py -v`，cwd 为 worktree。开始 2026-09-25T04:28:27.206814Z，结束 04:28:27.477267Z。执行期间 changed_inputs 为空，stdout 为空；unittest 正常结果位于 [stderr.txt](stderr.txt)，不是运行错误。命令及哈希见 [run.json](run.json)。

[inputs.json](inputs.json) SHA256：`f3481e5357bd3a4a1e8970a1c5d9d04622ab30ddddbecb04f8d8b4cd306e8743`；冻结 [approved-plan.md](approved-plan.md) SHA256：`25aad2e20963d88251ecfb4fb927acc70b0f9103b4ac5bd769131c320d5b7fd4`。原始证据保留。

覆盖替身异常/HTTP 分类、TLS 上下文和重定向策略、冻结引用拒绝、子进程协议与凭据环境传递。socket、urlopen 及真实发送子进程均被阻断，没有发送通知；不证明真实 TLS 握手、消息送达或 Go 适配器端到端通过。

%% CONFIG_RUNTIME: 本组对 project_config 使用替身，未调用其 tomllib 配置解析。实际解释器为 Python 3.9.13，后续须核验受管入口所需解析器/解释器条件，不能从本组通过推断该运行环境可解析真实 aiw.toml；不自动安装依赖或切换解释器。

完整 AC28–AC30/AX04、2.2 与 Gate 保留，未生成产品 Runner grant。
