# FD-031 Worker 实现报告（第 4 轮）

<!-- aiw-data: FD-031-implementation-r4.json -->

## 来源与修复

- Source event：`FD-031-000012-changes-requested`；Worker session：`fd031-worker-repair-20261008-4e367cb0`。认领前事件为 pending、FD revision 为 12，摘要与当前 FD 相符。
- Reviewer r1 指出：`update_index` 遇到其他 FD 文件的无效 UTF-8 时抛出 `UnicodeDecodeError`，先前只捕获 `OSError` 与 `FDError`，旧 FD、旧回执与新事件会停在不一致状态。
- 在 `recover_worker` 写入前保存索引原始字节。写入阶段的普通异常均进入回滚；回滚按新事件、FD、旧回执、索引逐项执行，索引直接恢复原始字节或原先的不存在状态，不再重读 FD 文件。回滚自身失败时，错误列出失败步骤和异常类型，并保留原始异常为原因。
- `atomic_text` 复用新增的原子字节写入函数。UTF-8 编码与原有文本写入一致，索引恢复可保留原始字节。

## 静态证据

- 核对 `recover_worker` 的事件 ID、旧 session、角色、状态、FD revision/digest 条件，以及正常成功路径。`UnicodeDecodeError` 是 `Exception`，会从 `update_index` 进入修复后的回滚块；索引恢复使用写入前快照，不依赖可能再次失败的 `write_index`。
- 核对回滚中每一步独立执行，即使前一步失败也继续尝试其余步骤；异常收集带步骤名与类型。系统级中断不由此普通异常回滚块接管。
- `python -c "from pathlib import Path; compile(Path('plugins/aiw-fd.py').read_bytes(), 'plugins/aiw-fd.py', 'exec')"` 通过，编译对象只在内存中。
- `git diff --check` 和 r4 JSON 静态解析通过；Git 仅提示本地换行符转换。

## 未执行检查与风险

- 本轮未运行 Tester 黑箱测试、故障注入、构建产物、格式化、lint、vet 或网络命令。此前修订的 12/12 黑箱通过结果不能证明本轮回滚修复；S21 仍待验证。
- 独立 Tester 须对新的 `implementation-ready` 事件和 FD 修订重新取得精确授权，再验证索引解码失败时 FD、旧回执、新事件与索引的恢复情况。PM 需重新决定测试报告。
