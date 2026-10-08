# FD-036 独立测试报告，第 1 轮

<!-- aiw-data: FD-036-test-report-r1.json -->

## 结论

本轮已从 Acceptance 拆解 15 个独立可观察场景，并准备以指定黑盒测试文件执行。所有 15 个场景均未运行；因此需求场景覆盖率为 0/15（0%），已执行、通过、失败行为测试均为 0。业务代码分支覆盖率未测量。当前报告不对任何运行时行为作通过判断。

- FD revision/digest：9 / `ac307da464bad4e9a6e9188c9374e5edd8f4547237092c5a2c3c18cb8ab6c151`
- Tester session：`fd036-tester-20261008-4c91e7`
- 来源实现事件：`FD-036-000009-implementation-ready`
- 测试授权：尚无绑定本次实现事件、FD revision/digest 和 Tester session 的 Planner 授权，故未执行命令。

## 场景与证据

下列场景均为 `prepared_unrun`，其 JSON `status` 使用 `uncovered` 表示没有执行证据。计划场景来自 FD Acceptance；Worker 报告说明本轮仅完成交接协调，实际实现与先前编译证据在 r1 报告，且没有本轮运行时证据。

| ID | 可观察行为 | 状态 | 未运行原因 |
| --- | --- | --- | --- |
| S01 | 插件布局下，位置文本输入可获得译文，stdout 只有译文 | 未运行 | 等待命令授权 |
| S02 | 从 stdin 输入可获得译文 | 未运行 | 等待命令授权 |
| S03 | stdin 与位置文本冲突、或输入为空时返回非零、stderr 有诊断且 stdout 为空 | 未运行 | 等待命令授权 |
| S04 | 缺省安装配置文件时采用内置默认值；有效 `--config` 可替换基础配置 | 未运行 | 等待命令授权 |
| S05 | 指定的配置文件不存在或无效时失败且无部分 stdout | 未运行 | 等待命令授权 |
| S06 | profile 可从用户配置路径选择；缺失或无效 profile 时失败 | 未运行 | 等待命令授权 |
| S07 | profile 名拒绝路径分隔符及目录穿越输入 | 未运行 | 等待命令授权 |
| S08 | 默认值、安装配置、profile、显式 CLI 参数按规定优先级合并 | 未运行 | 等待命令授权 |
| S09 | 语言/模式等枚举、未知 provider 与 timeout/model 参数非法时失败 | 未运行 | 等待命令授权 |
| S10 | Unicode、换行、引号及命令/伪指令源文本按数据提交并保留约定内容 | 未运行 | 等待命令授权 |
| S11 | 缺少 API 凭据、认证失败、限流、超时、无效响应均返回非零且不泄露敏感信息 | 未运行 | 等待命令授权 |
| S12 | 瞬时错误重试有界；失败时不输出部分译文 | 未运行 | 等待命令授权 |
| S13 | `--help`、`--version` 输出可用，成功/失败的 stdout、stderr、退出码符合约定 | 未运行 | 等待命令授权 |
| S14 | 未实现的 clipboard/file/dialog 等选项明确报错 | 未运行 | 等待命令授权 |
| S15 | 无图形环境可运行 CLI；四个 profile 示例复制使用且不覆盖已有用户文件 | 未运行 | 等待命令授权 |

### 静态排除项

- 真实 API 翻译质量、模型可用性及外部服务响应：本轮不请求真实 API；需另行明确授权，并可能产生网络访问或费用。
- 默认不记录原文、译文和密钥、提示词信任边界及错误脱敏：不能仅凭当前未执行的黑盒用例宣称满足；需结合静态代码审查或授权的 mock 断言。
- 示例文件字段与解析器一致、发行物安装布局及跨操作系统配置目录：未通过构建/打包或各平台运行验证。
- GUI、剪贴板、文件翻译、对话框、术语表、第二提供方：FD 明确不在本范围，不作为本轮适用场景。

## 命令与风险

**提议命令（未执行）**：`$tmpExe = Join-Path $env:TEMP ('aiw-say-fd036-' + [guid]::NewGuid().ToString('N') + '.exe'); try { go build -o $tmpExe ./cmd/aiw-say; if ($LASTEXITCODE -ne 0) { throw 'go build failed' }; $env:AIW_SAY_BIN = $tmpExe; python -m unittest tests.test_fd036_say_blackbox -v; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE } } finally { Remove-Item -LiteralPath $tmpExe -Force -ErrorAction SilentlyContinue; Remove-Item Env:AIW_SAY_BIN -ErrorAction SilentlyContinue }`

- 工作目录：`C:\Users\svictor\workspace\tools\ai-tools\aiw\.wt\FD-036`
- 预计时长：2 分钟内
- 预期范围与副作用：该 PowerShell 命令先将 `./cmd/aiw-say` 编译到唯一随机 `%TEMP%` exe，设置 `AIW_SAY_BIN`，运行指定 unittest，并在 finally 中删除该 exe、清理环境变量。测试会使用临时目录/进程并启动 127.0.0.1 loopback mock server。预计会写入 Go 临时编译缓存；不请求真实 API、不下载依赖、不需权限提升。执行前需 Planner 检查并生成绑定当前实现事件、revision/digest 与 Tester session 的授权。
- 实际命令：无。未执行测试、覆盖率、编译或程序运行命令。
- 原始运行证据：无（未运行）。
- 测试文件：`tests/test_fd036_say_blackbox.py`（Worker 报告称已存在；本轮未读取或修改）。PM 静态检查确认该用例使用 `unittest`、由 `AIW_SAY_BIN` 提供可执行文件，并将外部 API 配置覆盖到本地 mock。

未运行带来的剩余风险是所有运行时验收行为仍无独立测试证据。`recommendation: blocked` 仅反映测试执行受授权门槛阻塞，不是交付风险接纳结论；交付决定留给 PM 流程。
