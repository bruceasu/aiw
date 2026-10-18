# FD-098 独立实现审查

<!-- aiw-data: FD-098-review.json -->

结论：**通过静态审查**，无阻断发现。来源事件：`FD-098-000004-implementation-ready`；独立 Reviewer 会话：`fd098-review-20261019-7d2c`。比较基线 `3ea5e78`，实现提交 `9ba924bc0a1959ba9adeca66635383b53d1f9e2f`；当前 FD 为第 4 版。审查覆盖实际源码 diff、FD 全部验收条件、实现报告及对应文档和稳定规范；未将静态结论当作运行验证。

## 发现与依据

没有需要修改的正确性、规范或范围问题。代码改动集中于 `src/plugins/aiw-pkg/operations.py` 两处遍历逻辑，未改变安装记录格式、公共 CLI、事务或安装端。

- 独立升级：`preflight.check`（约第 162 行）仍遍历完整 final 图，包含未选中的旧集合；dependencies 检查存在和精确版本，members 只检查存在并递归。保留 aiw-all/aiw-core 的旧成员配方不会再拒绝依赖兼容的成员升级；旧集合不在 selected 中，因此不会被重写。
- 默认更新：根选择先为全部已安装包选择 latest。`visit`（约第 105 行）只对默认无 scope 更新的成员根省略 wanted，采用已经选择的成员版本；嵌套集合执行相同规则。既有成员交集裁剪保留，新成员不因集合更新而补装。
- 精确契约：dependencies 始终传入精确 wanted；显式集合安装和 --core/--all 的成员边仍传入配方版本。已选根冲突、反向运行依赖冲突、缺失依赖仍失败，不以成员关系豁免。`repository.select` 的版本选择和歧义拒绝未改变。
- 图完整性：最终图对缺失成员报错，dependencies/members 均使用 active/visited 递归检测循环；visit 同样保留规划阶段循环检测。全部 preflight 完成后才开始逐包写入。
- 状态兼容：`Manager.record` 继续解析旧 name/version 引用；`Manager.remove` 按成员名称保护被引用包，`orphans` 从 manual 根按名称遍历两类边。安装原因保留、选定直接安装根 manual 的既有规则不变；单成员升级不改变其他成员原因。
- 受限行为：锁下旧记录快照复核、平台/目标检查、逐包失败报告和事务逻辑未改变；dry-run 在 preflight 后跳过 save_record/install，不写安装文件或记录。临时规划目录不等于安装端写入。
- 文档和规格准确区分历史配方版本与当前成员记录，并说明真实共享资源依赖仍可能阻挡本机升级。没有声称已完成发布或安装。

## 命令与证据限制

实际执行：定向 `Get-Content -Encoding utf8`、`rg`、`Select-Object`、`Get-ChildItem` 读取；`git diff 3ea5e78 9ba924b --stat` 及限定源码/文档 diff；`git log -1`；`git status --short` 及 FD/索引工作区 diff；`aiw fd claim --help`、`aiw fd emit --help`；以本会话领取确切 implementation-ready 事件。两次定向 rg 中的猜测路径不存在，已明确记录且未重跑相同失败命令；可用源码读取未受影响。工作区 FD/索引状态差异来自已有实现就绪事件，源码无后续漂移。

Worker 报告记录的一次离线 compile-only 退出码为 0；检查了其命令和 `scripts/compile.py`：Python 内存 compile，不写 pyc；两个 Go 入口输出到 NUL。此结果为 Worker 提供的证据，本 Reviewer 未重复执行，也未独立观察该次命令输出。

本 Reviewer 未运行测试、编译、构建、验证脚本、格式化、lint、安装或网络；未新增测试代码，未修改实现，未作 Git 提交/合并/归档。写入本报告及 JSON，并更新 FD TODO/Verification 后，发出 reviewer verification-passed 事件。

## 剩余风险

安装、嵌套更新和失败路径仅作静态推演，未执行运行场景。真实 dependencies 继续精确匹配，本机 24 个旧技能仍可能阻止共享资源升级；本 FD 完成不代表现有 aiw-main 已升级或已解决这些依赖冲突。安装端尚未使用此源码，需另行授权打包发布及安装。
