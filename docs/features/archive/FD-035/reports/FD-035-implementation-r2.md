# FD-035 Worker 实施报告：补测场景与覆盖率 runner

<!-- aiw-data: FD-035-implementation-r2.json -->

## 交付

Worker session `fd035-worker-20261008-3f7c9a` 领取 `FD-035-000010-changes-requested`，按 revision 10 的新增范围完成 Work Items 1.5 和 1.6 的测试素材。

- 在 `tests/test_fd032_risk_decision_blackbox.py` 新增 S20–S25 六个独立黑盒场景。拒绝路径核对命令失败且没有产生新 PM 事件；S25 用 76% 场景覆盖率和零失败报告验证单份评估仍能路由 Reviewer。
- 在 `tests/fd035_coverage.py` 增加临时 coverage.py runner，并为 CLI 子进程添加仅在 runner 环境启用的 `sitecustomize` 启动钩子。运行完成后收集 CLI 源文件分支数和命中数，临时配置及原始数据随临时目录清理。
- 更新 FD-035 的新增 Work Item、Acceptance、TODO 和 Verification，保留 Tester 尚未执行的事实。

## 检查与限制

执行的 compile-only 命令：

```text
python -B -c "from pathlib import Path; files = ('tests/test_fd032_risk_decision_blackbox.py', 'tests/fd035_coverage.py'); [compile(Path(p).read_bytes(), p, 'exec') for p in files]; print('compile-only OK')"
```

结果为 `compile-only OK`。已静态检查测试夹具的临时仓库写入路径和覆盖率 runner 的 OS 临时目录配置。未运行黑盒测试或覆盖率命令。当前 Python 环境没有 coverage.py；未安装依赖，也未访问网络。执行测试与分支覆盖率仍需独立 Tester 及各自精确授权。
