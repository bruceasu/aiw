# FD-036 独立审查 r3

<!-- aiw-data: FD-036-review-r3.json -->

## 结论

**changes-requested**。本审查由 `fd036-reviewer-20261008-r1-independent` 完成，绑定 handoff `FD-036-000026-review-requested`、FD revision 26、digest `7e4977b143e589f74c2e18fd7547e4a5ed3aadab866283f782c77477bdc34232`。审查基于 `develop...HEAD`，基线 `5b04879e9cbfbdccca13bed13feeeeccb31e609c`，实现提交 `e14172a30f889747c48c21c1f0b9bf94ceec1616`。

## 发现

- **中：配置读取器把合法 TOML 子集之外的值当作损坏文件，同时也会接受非 TOML 转义。** [internal/say/config.go](../../../internal/say/config.go#L61) 只支持自定义标量语法；未知配置键在 `applyConfigValue` 中忽略之前，已由 `parseTOMLScalar` 解析。因此 `[say]` 下的合法 TOML `future_option = [1, 2]` 会报错，而不是忽略未知键。反过来，基本字符串 `model = "m\aname"` 中的 `\a` 是 Go `strconv.Unquote` 接受但 TOML 不支持的转义，读取器会接受该损坏 TOML。FD 的 Acceptance 明确要求未知键忽略且损坏 TOML 报错；用户确认了这两项行为。建议使用符合 TOML 语法的解析器，再对已知键逐字段校验：未知键跳过，已知字段值无效时保留较低优先级值，语法损坏时报错；增加上述两类边界用例。

## 证据核对

- Worker r5 报告所述 compile-only 命令与 FD Verification 记录一致；本 Reviewer 未运行构建或测试。
- Tester r4 Markdown/JSON 和事件24一致：针对实现事件23、revision 23、digest `ec7adbbb1e50c75d8e95834321a3f5d71f0765299d3934ddcee80694851f1c27`；21 个行为测试通过、0 失败，18 个场景中14个完整覆盖（77.78%），branch coverage 未测量。配置回退、profile 保留低优先级值、未知字符串键忽略、损坏配置报错和无效 CLI 参数仍报错均有对应通过用例。
- Planner r6 授权的命令与 Tester r4 报告一致，绑定实现 revision/digest、Tester session 和指定 worktree；声明禁用外部依赖下载、只用 loopback mock。PM r4 decision、三份风险评估和事件25一致：3票 `accept-with-risk`，明确保留 S08/S09/S14/S15 及 branch coverage、真实 API/模型、跨平台和翻译质量风险。本审查未因这些已接受的缺口要求重测。
- 检查了 `internal/say/config.go`、`internal/say/profile.go`、FD acceptance/work items/Verification、Tester/PM/Planner 双格式报告及 `develop...HEAD` 文件清单。实现的普通无效已知字段会通过候选副本验证后才覆盖合并值，CLI 参数仍由最终 `Validate` 严格校验；本次发现集中在 TOML 语法解析层。

## 命令与未执行检查

- 已运行：读取 `fd-review/SKILL.md`、`skills/work-management.md` 和仓库 `AGENTS.md`；`aiw fd --help`、`aiw fd claim --help`、`aiw fd emit --help`、`aiw fd show FD-036`、`aiw fd claim FD-036 FD-036-000026-review-requested --session fd036-reviewer-20261008-r1-independent`；`git status`、`git diff --stat/name-status develop...HEAD`、`git log` 和针对报告/源码的 `Get-Content`、`rg`。
- 未运行：测试、构建、coverage、格式化、lint、网络请求。

## 剩余风险

除上述未披露的 TOML 语法契约问题外，保留 PM r4 已接受的 S08/S09/S14/S15 场景缺口、branch coverage 未测、跨平台 profile 路径与真实 API/模型行为和翻译质量未验证。
