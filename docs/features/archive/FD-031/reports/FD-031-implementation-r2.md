# FD-031 Worker 修复报告（第 2 轮）

<!-- aiw-data: FD-031-implementation-r2.json -->

## 身份与交接

- 本轮新 Worker session：`fd031-worker-repair-20261008-4e367cb0`。
- 历史 `FD-031-000006-test-rejected` 已派发并绑定旧 session `fd031-worker-repair-20261008-01`。本轮未认领、替代或完成该事件，也未修改 receipt。正式 handoff 需要 PM 先通过受支持的恢复方式处理该旧事件；当前修复提交本身不构成 `implementation-ready`。

## 修复

- `internal/issue/store.go`：读取以 `%q` 写入的元数据字符串时使用 `strconv.Unquote`，还原标题中的引号与其他合法转义；对旧式或不合法引号值保留原有剥引号行为。
- `tests/test_fd031_promote_blackbox.py`：在临时项目中让 `TEMPLATE.md` 路径成为目录，使 FD 创建器的原子写入确实失败；检查错误指向模板、没有创建 FD 或 handoff、Issue 元数据与 Task 均未改变。
- FD TODO 和 Verification 记录历史失败、新版静态证据与独立 Tester 待办。

## 验证与剩余风险

- 静态追踪 `%q` 写入、元数据读取、promote 参数及 FD 创建器模板错误路径。旧测试报告中的 S06 和 S12 原因均有对应修复。
- `git diff --check` 通过；Git 仅提示工作区换行符将按本地设置转换。
- 本轮未执行黑箱测试；旧测试结果为 6/8，不能代表修复后的版本。独立 Tester 需要针对当前修订重新取得精确授权。
- `go build -o NUL ./cmd/aiw-req` 通过，未保留构建产物。正式 handoff 仍受旧 dispatched 事件占用。
