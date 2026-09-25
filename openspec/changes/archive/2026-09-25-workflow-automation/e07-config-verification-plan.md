# E07 真实配置解析验证

Task workflow-automation；wi-0012 / 2.2；实现来源 E07。状态：四项测试已准备，用户已提供 D:\green\python3.12，已确认其中 python.exe 文件存在；等待本组运行证据，实际版本由 runner 登记。

E07 实施记录已明确要求 Python 3.11+ 的标准库 tomllib，没有 Python 3.9 回退解析器。本机只读命令定位确认默认 python 为 C:\Python39\python.exe，python3 为 WindowsApps 别名，所查 PythonCore 注册位置未找到其他安装；这不能证明全盘没有其他解释器。上一组替身测试通过不满足真实配置入口的运行条件。

新增 plugins/test_notify_config.py：在独立临时目录读写实际 aiw.toml，不替换 project_config 或 tomllib；阻断网络/发送子进程，使用虚构环境值，不读取用户 .env。

| 用例 | 覆盖 |
| --- | --- |
| test_missing_and_disabled_configuration_never_ready | 缺文件、无通知段、显式禁用保持 disabled/not-ready |
| test_real_config_identity_credentials_and_ca | 缺凭据不 ready、忽略 .env、凭据轮换不改目标摘要；目标及 CA 内容变化改变冻结身份，CA 路径规范化 |
| test_invalid_real_toml_configuration_is_rejected | 未知字段、非法类型/超限 timeout、HTTP/userinfo/query、版本、重复 TOML 键、非法环境名与缺 CA 拒绝 |
| test_console_dispatch_rechecks_actual_configuration | 实际 console 配置可作本地尝试，之后目标改变则发送前拒绝且不显示旧消息 |

使用已确认的 Python 3.11+ 可执行文件运行本目录 run-e07-config-terminal.py。runner 沿用同一解释器，固定 `-I -B plugins/test_notify_config.py -v`，60 秒上限，仅四项。旧解释器提前退出 2，明确没有运行测试；不自动安装、切换 PATH 或寻找其他解释器重试。

该组需显式执行授权；无模型/网络/真实通知，不生成产品 grant。正式 Go 适配器仍通过 PATH 的 python 启动，单独指定新解释器运行测试不能证明 Go 运行环境已就绪。

## TODO 与 Verification

- [x] 核对实施记录、受管入口及解释器定位；不修改已批准的版本要求。
- [x] 四项真实配置测试及单次取证入口已准备。
- [x] 用户提供解释器目录，确认 python.exe 文件存在。
- [ ] 使用该解释器执行本组并记录实际版本和结果。

执行命令（PowerShell）：
```powershell
& 'D:\green\python3.12\python.exe' 'C:\Users\svictor\workspace\tools\aiw\.wt\workflow-automation\openspec\changes\workflow-automation\run-e07-config-terminal.py'
```

本轮离线 `python scripts/compile.py` 退出 0，仅证明生产 Go 编译，不证明 Python 测试通过。本组测试尚未运行。

%% ENVIRONMENT: 已取得用户提供的解释器位置；不自动修改 PATH，生产适配器默认 python 的版本要求仍需单独满足。
%% REMAINING: 真实 TLS/CA 验证、Go 适配器进程协议、真实宿主与服务仍待验证；本组 CA 字节仅验证配置身份，不证明证书可用。
