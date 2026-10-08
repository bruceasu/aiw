# FD-036 独立评审，第 1 轮

<!-- aiw-data: FD-036-review-r1.json -->

## 结论

**changes-requested**。Reviewer session `fd036-reviewer-r1-20261008-0915` 认领了 `FD-036-000017-review-requested`。审查的 FD 为 revision 17，digest `f0350f03e9220f96da7a4b2d4381110f13134d437f20b31e2cd8910189f99f2d`；比较基线为 `develop` (`5b04879e9cbfbdccca13bed13feeeeccb31e609c`) 至 `HEAD` (`1e37356bec0401b8e8ff023a4e6857dd3f02a8fd`)。Reviewer session 与 Worker、Tester/PM 及三位风险评估者均不同。

## 发现

- **中：拒绝合法 TOML 表头注释。** [internal/say/config.go](../../../internal/say/config.go#L82) 在第 82–86 行将表头限制为 `]` 必须是整行最后一个字符。因此标准合法配置 `[say] # 翻译设置` 会被当成 `invalid table header` 而无法加载。FD Work Item 1.3 要求解析 TOML 配置，Acceptance 要求安装配置可设定翻译选项；该输入是合法 TOML，不属于 PM 接纳的缺失测试场景。建议解析表头前正确剥离行内注释，并增加覆盖该有效配置形式的验证。未修改实现。

## 证据核对

- `FD-036-test-report-r2.json` 与 Markdown 对齐：测试针对 revision 14 / digest `914ae9b1043fa69226b2c7e00fed2ff4c2efe7b7e9ffad7f84c56ae8f83ad537`；报告 handoff 是随后 revision 15 / digest `8a27b890461b8899f79a646f6814a4031afe6fb3407b2892169b579deb5d981e`。PM decision-r2 明确区分 tested snapshot 与报告 handoff，事件语义一致。
- 测试 JSON 记录 19/19 通过、0 行为失败、15 场景中 10 项完整覆盖，覆盖率 66.7%，branch coverage 未测量。它也准确保留首轮测试夹具失败（空 `--model` 继承有效基础模型）及修正夹具后的同命令成功结果；缺失的 S05/S08/S09/S14/S15 未写成通过。
- Planner authorization-r4 将命令、Tester session、FD revision/digest、`.wt/FD-036` 工作目录和范围绑定一致。命令只编译到随机 `%TEMP%`、使用独立 Go cache、关闭 Go 下载/校验网络，并运行指定 unittest 模块及 loopback mock。两次运行均有记录；首轮失败后修正测试夹具再重跑符合相关变更后一次重跑的边界。报告披露临时 exe/cache 保留，未发现用户配置或仓库外部数据写入，也没有真实 API 请求。
- 三份风险评估与 PM decision-r2 在 2 票接纳、1 票修复、风险例外及未测内容方面一致。S05/S08/S09/S14/S15、Linux/WSL profile 路径和 branch coverage 均按 PM 明确接受的例外处理，不作为本次拒绝理由。
- Worker implementation-r1/r2/r3 与 `develop...HEAD` 差异中的实现、文档、样例及测试文件相符；静态阅读确认 CLI 到配置、profile、校验、提示词、OpenAI 请求及输出的主要调用路径。另注意 FD Verification 当前仍以 `000016` 作为待审事件；请在 repair 阶段更新至本次 report/outcome，并按支持的 refresh-worker 流程生成新 handoff。

## 命令与剩余风险

本 Reviewer 执行了只读命令：`aiw help --json`、`aiw fd --help`、`aiw fd show --help`、`aiw fd claim --help`、`aiw fd emit --help`、`aiw fd show FD-036`、`git status --short`、`git branch --show-current`、`git rev-parse HEAD`、`git rev-parse develop`、`git diff --stat develop...HEAD`、`git diff --name-status develop...HEAD`、`git log`、`git show --stat`、定向 `rg --files`/`rg -n`，以及定向 `Get-Content`/`Get-ChildItem`。`aiw fd show FD-036 --json` 因该命令不支持 `--json` 未成功；首次 `aiw fd show FD-036` 遇到 Windows cp932 输出编码错误，设置 `PYTHONIOENCODING=utf-8` 后读取成功。没有运行测试、构建、覆盖率、lint、格式化或网络命令。

剩余验证风险仍包括 PM 接受的五个场景缺口、未测 branch coverage、Linux/WSL 路径、手动复制流程及真实 API 行为。本次新增发现是有效 TOML 输入被实现拒绝的静态证据；修复后的运行证据留待后续获准验证。
