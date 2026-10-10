# FD-046 实现报告（第一轮）

<!-- aiw-data: FD-046-implementation-r1.json -->

- Worker 会话：`codex-main-20261009-fd046`。
- Source event：`FD-046-000003-design-ready`。

## 实现内容

- `aiw fd close <id> Complete --force` 增加 `-f` 别名；强制路径不要求调用前状态为 Complete，会在归档事务内设置 Complete 并增加 revision。
- 强制路径要求有效单行 `--reason`，记录本地操作者、原状态与事件、结果、跳过的状态和 Reviewer 检查，并写入 `review_verified: false`。不创建或修改事件收据。
- 保留最新 launching/dispatched 收据保护、路径安全、目标冲突拒绝、索引更新及归档失败回滚；不带 `--force` 的原归档行为保持不变。
- 更新 FD 工作流稳定规格、使用文档和 FD-046 计划。

## 实际证据

- 静态检查参数解析、forced 与正常 close 分支、force_context 原状态与事件读取、归档文件迁移、索引更新、待处理收据保留和错误回滚路径；核对规格与使用文档的门槛说明一致。
- 编译命令通过，未写出字节码：`python -c "from pathlib import Path; compile(Path('plugins/aiw-fd.py').read_text(encoding='utf-8'), 'plugins/aiw-fd.py', 'exec'); print('Python syntax compile passed; no bytecode written.')"`。
- 按用户先前请求，使用刚实现的主工作区插件执行 FD-044 强制归档。`python plugins/aiw-fd.py close FD-044 Complete --force --reason "User confirms Say plugin is complete and runtime validated; forced archive requested."` 成功归档 FD-044；输出明确 Reviewer verification was skipped and not recorded，并生成 operation audit。
- 直接通过 PATH 中的 `aiw fd close ... --force` 首次调用时，旧的外部插件副本报 `unrecognized arguments: --force`，没有执行归档；随后通过主工作区插件脚本完成归档。外部安装副本未更新。

## 未执行与剩余风险

未运行测试、文件系统故障注入、格式化、lint、最终构建或网络操作。强制归档不能证明 Reviewer 验证通过；最新 launching/dispatched 收据仍会阻止归档。PATH 中当前 `aiw` 使用的外部插件副本还不含新选项，需要单独更新安装副本后，`aiw fd close ... --force` 才可通过该入口使用。
