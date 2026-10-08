# FD-036 独立测试报告，第 2 轮

<!-- aiw-data: FD-036-test-report-r2.json -->

## 结论

在 `FD-036-000014-implementation-ready` 对应的 Revision 14 实现收据（digest `914ae9b1043fa69226b2c7e00fed2ff4c2efe7b7e9ffad7f84c56ae8f83ad537`）上，获 Planner 授权执行单个本地 mock 黑盒模块。修复测试夹具后，19 个测试全部通过；行为失败数为 0。15 个验收场景中 10 个有完整运行证据，需求覆盖率 66.7%；5 个场景仍未覆盖。Branch coverage 未测量。

首轮执行 19 个测试时，空 `--model` 子用例沿用了基础配置中的有效模型，导致测试自身断言失败；该轮没有证明产品缺陷。将测试改为配置空模型后，按同一授权命令复跑，19 个测试通过。测试代码另将 mock server 的 Windows 连接中止纳入预期关闭处理。

## 场景与证据

- **已覆盖（10）：** S01 位置文本和 source/user 消息边界；S02 UTF-8 stdin；S03 空输入及参数/stdin 冲突；S04 缺省配置回退及有效显式配置；S06 profile 选择、缺失与无效 profile；S07 profile 路径安全；S10 Unicode/换行/伪指令作为独立用户内容传递；S11 凭据、401、超时、无效响应及诊断不泄露原文；S12 限流重试次数有界、成功与耗尽无部分输出；S13 help/version、成功 stdout 和失败状态/空 stdout。
- **未覆盖（5）：** S05 未覆盖语法损坏 TOML；S08 未运行默认值→安装目录配置路径及其与 profile/CLI 的完整优先级矩阵；S09 未逐项检查全部 source/mode/style/politeness/profanity 枚举；S14 仅验证 `--clipboard`，其余未实现开关未逐个运行；S15 未跨平台执行，也未实际验证样例复制且不覆盖已有文件。profile 仅在当前 Windows `APPDATA` 路径执行，Linux/WSL 的 XDG 与 `~/.config` 路径未运行。
- S15 的手动复制/不覆盖是文档工作流，没有对应 CLI 写入命令；报告不将文件存在性断言视为复制流程通过。

## 命令与风险

- 实际执行 Planner 授权 r4 中的精确命令两次，工作目录 `.wt/FD-036`。第一次 `Ran 19 tests in 2.280s; FAILED`，失败定位为测试夹具未清空有效模型；修改夹具后复跑，`Ran 19 tests in 2.280s; OK`。
- 命令在 `%TEMP%` 留下随机命名 exe 和独立 Go build cache；Go module/checksum 网络访问关闭。mock API 仅监听 `127.0.0.1`，没有调用真实服务或读取真实凭据。
- 初次尝试带递归清理的授权命令被执行策略拦截，没有执行；随后批准并运行了不含删除操作的 r4 命令。

剩余风险：未覆盖的 5 个验收场景、Linux/WSL 配置路径、其他未实现选项及手动样例安装行为没有运行证据；真实 API 质量和模型可用性未验证。
