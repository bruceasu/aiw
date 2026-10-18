# FD-087 独立实现审阅

<!-- aiw-data: FD-087-review.json -->

结论：通过静态审阅；没有阻塞发现。未执行运行验证。

- FD：FD-087，修订 4；审阅实现提交 `0b238d3`，比较基线 `fb67c60`。
- 来源事件：`FD-087-000004-review-requested`。
- 独立 Reviewer session：`codex-fd087-dual-version-review-20261018`。

已检查实际 build.py diff、FD 验收、实现报告、usage 与稳定规范，并追踪
scripts/package_definitions.py 和 archive.py 的相关包版本调用路径。
显式来源经 latest_compile 精确筛选版本和组件覆盖，再校验完整原生产物清单与
SHA-256；来源缺失、覆盖不足或产物变化均在组包前失败，不回退或重新编译。
复用路径跳过 compile_target，不改 bin 及编译记录。组包入口及独立脚本显式接收
包版本，ZIP 路径、元数据、依赖和集合成员使用包版本。组件、target、all 的
分组清单均写入编译来源 compile_version；latest 仍读取包版本。

显式 --compile-version 独立于 AIW_VERSION 的来源筛选；包版本依次采用 --version、
非空 AIW_VERSION、编译来源。未指定来源的复用/显式版本新编译、publish 新编译、
clean 及历史清单读取路径保持兼容。变更范围与 FD 一致，没有附带重构。

实际命令：Get-Content/rg 定向读取、git diff fb67c60 0b238d3（一次正文差异与
一次范围统计）、aiw fd --help、aiw fd show FD-087、claim/emit --help，以及精确
aiw fd claim。最初读取 .aiw、工作树 .ai 和旧审阅文件的路径不存在；随后通过
aiw fd show 获取真实最新 pending 事件，并成功 claim，没有伪造收据。
完成证据后执行 aiw fd emit FD-087 verification-passed --producer reviewer
--artifact docs/features/reviews/FD-087-review.md
--source-event FD-087-000004-review-requested。

Worker 报告的 Python 仅语法编译与当前实现文件一致；本 Reviewer 未重跑编译，
未运行测试、实际编译目标、构建、发布、clean、行为验证、网络或额外审阅轮次。

%% 运行行为未经验证；静态通过不表示包构建或发布已经成功执行。
%% 二进制内嵌版本保留 snapshot；当前资源可能与原编译源码时点不同。
%% 只支持仍有效的当前编译产物，不保留已覆盖的历史版本；继续遵循单写入者和
不可覆盖发布限制。上述边界均已在 FD 和使用文档中说明。
