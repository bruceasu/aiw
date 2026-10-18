# FD-086 独立实现审阅

<!-- aiw-data: FD-086-review.json -->

结论：通过本轮独立静态审阅，未发现阻塞验收的问题。

审阅会话：`codex-review-FD086-20261018-1324-reuse`。
来源事件：`FD-086-000004-review-requested`。
审阅提交：`329de34`；比较基线：`23caf71`。FD Revision 4。

审阅实际代码与文档的结论：

- `build.py` 的默认 build 查找最新覆盖组件的成功 compile 记录；环境版本限制候选，显式版本及 publish 保持重新编译。复用同时沿用版本及原生产物，资源组包仍使用当前文件。
- 编译开始前移除同目标及原生组件重叠记录，整次编译及扫描成功后才登记。失败不会重新授权旧二进制；资源目标允许空原生清单。
- Go 清单覆盖 Windows/Linux 两套精确路径；TypeScript 遍历全部输出文件。复用重新计算完整字典，要求路径与 SHA-256 全等，拒绝增加、删除、修改及链接。覆盖较大目标的记录也验证整个记录，而非仅检查当前子集。
- `latest-compile` 验证被选择的最新覆盖记录；原有 latest 保留成功组包查询语义。无匹配记录才编译，选中记录产物异常会直接报错。
- `clean_outputs` 在任何删除前检查 bin/dist 两个绝对目标的归属及全部后代；`archive.safe_path/reject_link` 覆盖 Windows reparse point、祖先及叶节点。缺失目录可忽略，异常非目录或链接会拒绝；删除目标固定为 checkout 两树，不读取外部仓库配置作为删除目标。
- `docs/usage/aiw-pkg.md` 与稳定 spec 的新行为一致；实现未引入额外依赖或无关模块修改。规划阶段 D1 已落实到完整清单相等比较，无需重新打开设计审阅。

实际命令：定向 `Get-Content` / `rg`；`git diff 23caf71 329de34 -- build.py scripts/packages/native.py scripts/packages/common.py`；`git diff --stat 23caf71 329de34`；限定 usage/spec 的 `git diff`；`aiw fd --help`、`claim --help`、`emit --help`、`show FD-086`，以及上述精确事件的 claim。初次 `aiw fd status FD-086` 因不存在该子命令失败，随后改用 show；对不存在的 dispatch.json、common.py 的定向读取失败，未改实现或扩大执行范围。

Worker 报告中的 Python compile-only 成功为其记录证据，审阅者未重复运行。未运行测试、编译、实际 build/clean/publish、安装、网络或产品 CLI 行为验证。

%% 运行行为尚未实测；静态通过不代表运行验证通过。
%% 复用不检测源码新鲜度；需要更新原生二进制时必须重新 compile。
%% 现有单写入者假设不提供并发保证；预检后系统删除失败可能造成部分清理。

后续：PM 提交审阅证据，按现有交付和清理门禁完成本地交付与归档。
