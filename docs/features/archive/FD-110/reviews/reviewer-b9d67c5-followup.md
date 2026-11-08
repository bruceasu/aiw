# FD-110 独立评审补充：DeepSeek CLI provider 选择

- **FD：** FD-110
- **来源事件：** `FD-110-000004-review-requested`
- **Reviewer session：** `reviewer-20261108-fd110-static`
- **评审提交：** `b9d67c5`（此前评审基线 `12b3920`）
- **结论：** Verification passed

## 上次发现复核

上次发现位于 `src/plugins/aiw-cz/aiw-cz.py:22`：CLI 的 `--provider` choices 缺少 `deepseek`。提交 `b9d67c5` 已将 `deepseek` 加入 choices，`aiw-cz --provider deepseek` 可通过 argparse 解析。

该入口随后将 `config.provider` 设为 CLI 参数值；现有 `--model` 逻辑按 `config.provider` 选择对应配置槽，因此 `--provider deepseek --model <id>` 写入 DeepSeek 的模型槽。入口调用 `candidates(..., config.provider)`，而 `cz_llm.candidates` 对显式 provider 使用该选择并路由到 `generate_deepseek`。上次唯一发现已解决，本次聚焦范围未发现其他问题。

## 命令与限制

- 已运行：`aiw fd show FD-110`、`aiw fd claim FD-110 FD-110-000004-review-requested --session reviewer-20261108-fd110-static`、`git show --stat --oneline b9d67c5`、`git diff 12b3920..b9d67c5 -- src/plugins/aiw-cz/aiw-cz.py docs/features/FD-110/FD-110_ADD_DEEPSEEK_PROVIDER_SUPPORT_ACROSS_AIW_LLM_CAL.md`。
- 未运行测试、构建、Python 执行或联网请求；本次只验证命令行 provider 选择与对应模型配置槽的静态调用路径。
- 剩余风险：无针对本次修正的运行时证据；遵循 FD 验证预算，本次未执行 CLI。
