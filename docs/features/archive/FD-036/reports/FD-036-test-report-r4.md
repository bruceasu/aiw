# FD-036 独立测试报告 r4

<!-- aiw-data: FD-036-test-report-r4.json -->

## 结论

针对 `FD-036-000023-implementation-ready`（FD revision 23，digest `ec7adbbb1e50c75d8e95834321a3f5d71f0765299d3934ddcee80694851f1c27`），执行了经 Planner r6 授权的离线黑盒命令。构建成功，21 项测试全部通过，0 项失败。18 个适用场景中，14 个完整覆盖，需求场景覆盖率为 77.78%；分支覆盖未测量。

## 场景结果

| 场景 | 结果 | 测试与观察 |
| --- | --- | --- |
| S01 参数翻译并将源文本作为 user 内容 | 通过 | `test_argument_translation_keeps_source_in_user_message` |
| S02 UTF-8 stdin 翻译 | 通过 | `test_stdin_translation_preserves_utf8` |
| S03 空输入和参数/stdin 冲突失败且无部分输出 | 通过 | `test_empty_input_fails_without_request_or_partial_stdout`、`test_argument_and_stdin_are_mutually_exclusive` |
| S04 缺少默认配置使用默认行为；有效显式配置生效 | 通过 | `test_missing_default_config_uses_defaults_and_requires_explicit_model`、`test_valid_explicit_config_replaces_base_settings` |
| S05 缺失或语法损坏的显式配置报错且 stdout 为空 | 通过 | `test_invalid_explicit_config_fails_without_partial_output` 检查缺失文件及损坏 TOML；两种情形均非零退出、无 stdout、没有 provider 请求 |
| S06 profile overlay、缺失 profile 和不安全名称 | 通过 | `test_profile_overlays_base_and_cli_overlays_profile`、`test_missing_profile_and_unsafe_profile_name_fail_before_request` |
| S07 profile 路径穿越/分隔符防护 | 通过 | `test_missing_profile_and_unsafe_profile_name_fail_before_request` |
| S08 内置默认、安装配置、profile、CLI 的完整优先级 | 覆盖不足 | `test_profile_overlays_base_and_cli_overlays_profile` 证明安装配置→profile→CLI；内置默认→安装配置的完整链未覆盖 |
| S09 provider/model/timeout/target 及所有枚举校验 | 覆盖不足 | `test_invalid_cli_options_and_missing_model_fail_before_request` 检查部分 CLI 错误；没有覆盖全部枚举。配置中的无效值回退见 S17 |
| S10 Unicode、换行、引号及伪指令作为普通源文本 | 通过 | `test_argument_translation_keeps_source_in_user_message`、`test_stdin_translation_preserves_utf8` |
| S11 凭据、认证、限流、超时和无效响应错误不泄露内容且无部分输出 | 通过 | `test_missing_credentials_fails_without_leaking_source`、`test_auth_failure_has_no_partial_stdout`、`test_rate_limit_exhaustion_fails_without_partial_output`、`test_request_timeout_fails_without_partial_stdout`、`test_invalid_api_response_fails_without_partial_stdout_or_body_leak` |
| S12 重试有界且耗尽后失败 | 通过 | `test_rate_limit_retries_are_bounded`、`test_rate_limit_exhaustion_fails_without_partial_output` |
| S13 help/version 与输出契约 | 通过 | `test_help_and_version_do_not_call_provider` |
| S14 所有未实现选项均明确报错 | 覆盖不足 | `test_unimplemented_option_fails_before_request` 只覆盖 `--clipboard` |
| S15 headless 和 profile 示例复制且不覆盖现有文件 | 覆盖不足 | `test_profile_examples_exist_for_copy_based_user_installation` 只检查示例文件存在；未验证复制、不覆盖及跨平台行为 |
| S16 合法 TOML 表头行尾注释 | 通过 | `test_valid_explicit_config_replaces_base_settings` 验证 `[say] # ...` 与 `[say.llm] # ...` 后配置生效 |
| S17 配置值无效时逐项回退；profile 无效值保留低优先级值 | 通过 | `test_invalid_config_values_fall_back_and_unknown_keys_are_ignored` 验证无效 target/style 回到内置值且请求成功；`test_invalid_profile_values_preserve_lower_precedence_settings` 验证无效 profile target/model/timeout 保留较低优先级值 |
| S18 未知配置键被忽略 | 通过 | `test_invalid_config_values_fall_back_and_unknown_keys_are_ignored` 与 `test_invalid_profile_values_preserve_lower_precedence_settings` 在配置/profile 中分别含未知键，仍成功处理 |

## 执行范围与限制

测试使用本地 loopback mock provider，不调用真实 API、不访问外网。成功结果证明上述模拟场景；不证明真实模型翻译质量、跨平台行为或分支覆盖。S08、S09、S14、S15 仍有未覆盖部分。配置语法损坏按验收要求报错；合法 TOML 中无效配置项回退并忽略未知键。

## 授权与执行

- Tester session：`fd036-tester-20261008-r4-independent`
- 来源 handoff：`FD-036-000023-implementation-ready`
- Planner 授权：`docs/features/reports/FD-036-test-authorization-r6.md`，绑定 revision 23 与上述 digest
- 工作目录：`.wt/FD-036`
- 实际命令及完整原始结果记录在同名 JSON sidecar。