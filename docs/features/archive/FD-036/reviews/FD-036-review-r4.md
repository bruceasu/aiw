# FD-036 独立审查 r4

<!-- aiw-data: FD-036-review-r4.json -->

## 结论

**verification-passed**。本审查由 `fd036-reviewer-r4-20261008` 完成，认领 handoff `FD-036-000035-review-requested`，绑定 FD revision 35、digest `6abd1f6d233105ce0c1ddecc76a90b16cfca418a541c450a4916e79a7dd220ab`。审查当前 `feature/FD-036` 的 HEAD `ae3dc4b5270834ec7d2e52fae3e735c11f32a100`，以 `develop...HEAD` 为差异范围。

未发现未解决的 material issue。当前配置实现使用 BurntSushi TOML 解码完整文档，再提取支持的表和值；未知字段不会进入应用逻辑，已知字段先做类型和取值校验后才覆盖已有配置。损坏 TOML 在应用配置前返回解析错误。profile 通过同一解析和逐字段回退路径覆盖基础配置，CLI 最后由 `Validate` 严格检查。

## 双证据核对

- Worker r6 报告及提交 `2e795808908f7ad1af0d5972edc7c45e6e8d6807` 显示以 `github.com/BurntSushi/toml v1.6.0` 替换自定义 TOML 标量解析器；未知合法数组值可被完整解析后忽略，非法转义由 TOML 解析器拒绝。实现 diff 仅调整配置读取/应用、依赖、黑盒回归及 FD 记录；依赖增加有用户批准留档。
- Tester r5 报告记录绑定 revision 29 的完整套件 21 项中 20 项通过、1 项 S04 `ERROR`，且明确缺少 traceback、最终摘要和退出码。该历史失败仍为事实，没有被覆盖或声称通过。
- Planner r8 授权记录引用用户的一次批准，绑定实现事件 `FD-036-000032-implementation-ready`、revision 32、digest `2e7c2d42d2f522a19d9f6bfb5621947886142e31f28311d381f6f6edd7ae9b5d`、Tester session `fd036-tester-20261008-r6-s04-diagnostic` 和精确单用例命令。授权明确关闭 Go 模块代理及校验数据库、仅可执行一次、不得重跑。
- Tester r6 报告与授权命令一致，仅执行 S04 指定用例；报告及 `%TEMP%` 原始日志均显示 `ok`、`Ran 1 test in 0.511s`、`OK` 和 `UNITTEST_EXIT_CODE=0`。日志中的 PowerShell `NativeCommandError` 是 stderr 进度文本呈现，后续 unittest 摘要与退出码明确成功。
- A/B/C 三位独立评估者 session 与本 Reviewer 不同，均投 `accept-with-risk`。PM r6 decision 绑定事件 `FD-036-000033-test-report-ready`，按重大证据缺口升级后接受风险，只允许进入独立 Reviewer；明确说明 r6 不是完整套件重跑，r5 ERROR 根因未知。
- PM 接受的限制继续保留：r5 ERROR 未解释；没有在当前 revision 重跑完整套件；S08/S09/S14/S15 未覆盖；branch coverage 未测；真实 API、跨平台 profile 路径及翻译质量未验证。Reviewer 未将任何未运行检查算作通过，也未因这些已明确接受的限制重复要求测试。
- 对照稳定规范 `openspec/specs/cli-and-plugins/spec.md`，插件入口及 `aiw-<name>` 分派契约无冲突。FD 当前 Acceptance 的配置回退、忽略未知键、损坏 TOML 报错与实现的解析/逐字段应用路径一致。

## 发现与检查范围

- 未发现需要退回 Worker 的问题。
- 静态检查覆盖当前 FD 的 Acceptance、Work Items、Verification、TODO，相关配置实现与 profile 调用路径，Worker r6 实现报告与提交差异，Tester r5/r6 报告和授权，r6 A/B/C 评估、PM r6 决策，以及稳定 CLI/plugin spec。
- 未运行测试、构建、coverage、lint、格式化或网络命令。

## 实际命令

- `Get-Content` / `rg`：读取 FD、实现、Worker/Tester/Planner/PM/评估者报告及稳定规范。
- `git show --stat --oneline 2e795808` 与 `git show 2e795808 -- internal/say/config.go internal/say/profile.go go.mod go.sum`：核对实现提交。
- `Get-Content -Raw`：检查 r6 授权、报告和风险决策 JSON。
- 读取 `%TEMP%\aiw-say-fd036-r5-diagnose-dd291399e2db442ba0eec81172dd7914.log`：核对已存在的诊断输出；没有重新执行命令。
- `git diff --name-status develop...HEAD`、`git status --short`、`git rev-parse HEAD`：检查变更清单、工作区和审查提交。
- `aiw fd emit --help`：确认 handoff 命令用法。
- `aiw fd show FD-036`：因当前 PowerShell Python 默认 `cp932` 无法编码中文而失败；未更换配置或重试。FD revision/digest 来自精确 handoff 和当前 FD 文本。
- `git diff --check develop...HEAD`：报告了历史报告文件中的尾随空白/末尾空行；本 Reviewer 未改动这些文件，且不影响本次功能审查结论。

## 残余风险

S04 r5 ERROR 的原因仍未知，r6 只证明该单用例本次未复现。完整套件和上述未覆盖行为仍无当前修订的执行证据；这些限制已由 PM r6 明确接纳并保留，不代表通过。
