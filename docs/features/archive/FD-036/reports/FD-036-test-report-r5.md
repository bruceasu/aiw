# FD-036 独立测试报告 r5

<!-- aiw-data: FD-036-test-report-r5.json -->

## 结论

本轮依据 Planner r7 授权，对实现交接 `FD-036-000029-implementation-ready`（FD revision 29，digest `64a3b0ec060f0d1d492f6ac3e8acf7d4c3ad20184449064c0b6962596051f522`）执行一次黑盒命令。可见输出显示 20 项通过，`test_missing_default_config_uses_defaults_and_requires_explicit_model` 显示 `ERROR`。工具返回的输出未包含 traceback、unittest 最终摘要或可用退出码；不能据此归因于产品实现或测试夹具。本轮结果为失败且诊断证据不完整。

需求场景覆盖率按已完整通过的场景计为 13/18（72.22%）。S04 的一条测试报错，因此标为阻塞；S08、S09、S14、S15 继续保留既有覆盖不足。分支覆盖未测量。

## 场景与证据

| 场景 | 结果 | 测试与观察 |
| --- | --- | --- |
| S01 参数文本翻译及 user 消息边界 | 通过 | `test_argument_translation_keeps_source_in_user_message` |
| S02 UTF-8 stdin 翻译 | 通过 | `test_stdin_translation_preserves_utf8` |
| S03 空输入及参数/stdin 冲突 | 通过 | `test_empty_input_fails_without_request_or_partial_stdout`、`test_argument_and_stdin_are_mutually_exclusive` |
| S04 默认配置缺失行为与显式配置 | 阻塞 | `test_valid_explicit_config_replaces_base_settings` 显示通过；`test_missing_default_config_uses_defaults_and_requires_explicit_model` 显示 `ERROR`，但无 traceback 与最终汇总，无法判断原因 |
| S05 显式配置缺失及 TOML 语法错误 | 通过 | `test_invalid_explicit_config_fails_without_partial_output`，包括非法表头及 TOML 不支持的 `\a` 转义；可见结果为 `ok` |
| S06 profile overlay、缺失及无效 profile | 通过 | `test_profile_overlays_base_and_cli_overlays_profile`、`test_missing_profile_and_unsafe_profile_name_fail_before_request` |
| S07 profile 路径安全 | 通过 | `test_missing_profile_and_unsafe_profile_name_fail_before_request` |
| S08 默认→安装配置→profile→CLI 完整优先级 | 覆盖不足 | `test_profile_overlays_base_and_cli_overlays_profile` 未覆盖默认值到安装配置的完整链 |
| S09 provider/model/timeout/target 全部校验 | 覆盖不足 | `test_invalid_cli_options_and_missing_model_fail_before_request` 覆盖部分字段和枚举；未穷尽 |
| S10 Unicode、换行、引号及伪指令按源文本传递 | 通过 | `test_argument_translation_keeps_source_in_user_message`、`test_stdin_translation_preserves_utf8` |
| S11 凭据、认证、限流、超时及响应错误 | 通过 | `test_missing_credentials_fails_without_leaking_source`、`test_auth_failure_has_no_partial_stdout`、`test_rate_limit_exhaustion_fails_without_partial_output`、`test_request_timeout_fails_without_partial_stdout`、`test_invalid_api_response_fails_without_partial_stdout_or_body_leak` |
| S12 重试有界及耗尽行为 | 通过 | `test_rate_limit_retries_are_bounded`、`test_rate_limit_exhaustion_fails_without_partial_output` |
| S13 help/version 与流契约 | 通过 | `test_help_and_version_do_not_call_provider` |
| S14 所有未实现选项均报错 | 覆盖不足 | `test_unimplemented_option_fails_before_request` 仅检查 `--clipboard` |
| S15 headless、示例复制不覆盖及跨平台行为 | 覆盖不足 | `test_profile_examples_exist_for_copy_based_user_installation` 仅检查示例文件存在 |
| S16 TOML 表头行尾注释 | 通过 | `test_valid_explicit_config_replaces_base_settings` |
| S17 无效值及字段类型错误回退 | 通过 | `test_invalid_config_values_fall_back_and_unknown_keys_are_ignored`；`test_invalid_profile_values_preserve_lower_precedence_settings` 将 `model` 设为合法 TOML 数组并断言请求仍使用 `base-model` |
| S18 未知 TOML 键（含合法数组值）忽略 | 通过 | `test_invalid_config_values_fall_back_and_unknown_keys_are_ignored` 验证未知标量和数组；profile 未知键由 `test_invalid_profile_values_preserve_lower_precedence_settings` 覆盖 |

## 命令与风险

- Tester session：`fd036-tester-20261008-r5-toml`
- 来源交接：`FD-036-000029-implementation-ready`
- Planner 授权：`docs/features/reports/FD-036-test-authorization-r7.md`
- 精确执行命令：

```powershell
$tag='aiw-say-fd036-r5-'+[guid]::NewGuid().ToString('N'); $testExe=Join-Path $env:TEMP ($tag+'.exe'); $env:GOCACHE=Join-Path $env:TEMP ($tag+'-cache'); $env:GOPROXY='off'; $env:GOSUMDB='off'; $env:PYTHONDONTWRITEBYTECODE='1'; go build -o $testExe ./cmd/aiw-say; if ($LASTEXITCODE -ne 0) { throw 'go build failed' }; $env:AIW_SAY_BIN=$testExe; python -B -m unittest -v tests.test_fd036_say_blackbox
```

- 工作目录：`.wt/FD-036`
- 原始可见结果：build 后 unittest 输出 20 行 `... ok`；`test_missing_default_config_uses_defaults_and_requires_explicit_model ... ERROR`。工具结果没有提供 traceback、最终测试摘要或退出码。随后只读检查未发现存活的 Python 或 Go 进程。未重跑命令。
- 运行范围使用本地 `127.0.0.1` mock API，不访问真实 API/外网；Go 代理和校验网络关闭。执行可能在 `%TEMP%` 创建临时 exe 与独立 Go cache，测试配置使用 TemporaryDirectory。
- 本轮未执行 coverage。完整 TOML 解析行为以新增黑盒案例为依据；S04 错误尚无诊断，因此此次执行不能证明或否定该场景的产品行为。

## 剩余风险

当前可见 `ERROR` 的根因未知，且缺少 traceback/最终汇总；不得把它当作产品缺陷或测试夹具缺陷。S08、S09、S14、S15 未完整覆盖；真实 API、跨平台行为、实际翻译质量和分支覆盖不在本轮证据中。
