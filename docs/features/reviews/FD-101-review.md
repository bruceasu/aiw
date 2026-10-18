# FD-101 独立实现审查

<!-- aiw-data: FD-101-review.json -->

结论：通过。未发现阻断验收的正确性、标准、规范或证据问题。

Reviewer：`fd101-review-independent-20261019`；来源事件
`FD-101-000005-implementation-ready`，已独立领取。
审查基线 `033da91`，实现提交 `5889483`，当前 FD 修订 5。
采用修订 3 目录方案及其独立设计报告；修订 2 报告仅作历史记录。

## 静态证据

- `scripts/package_definitions.py` 同时删除规划依赖和包元数据的统一 runtime
  绑定；`build.py` 单目标和集合闭包不再因插件拉入 runtime。显式兼容分组仍可选择
  runtime，旧目标和两个 canonical 根实现保持不变。
- 三个旧顶层入口确实迁入目录，没有重复入口；manifest 只含 Go schema-1 接受的字段，
  命令名、显式入口和 Python startup 可由现有发现器解析。file/patch 入口内容未改，
  所需 codec 符号与各自开发 shim 导出相符，canonical codec 仅依赖 Python 内置能力。
- `PLUGIN_PRIVATE_FILES` 同时控制 `tree_entries` 的相对路径排除和 `one` 私有目标；
  `archive.write_package` 从 canonical 源读取真实字节，发行载荷不会携带开发 shim。
  三个私有目标互异且不包含旧顶层入口或 runtime 根路径。
- `Manager.owners/preflight/install` 仍校验跨包路径归属，按当前包记录移除 obsolete
  非配置文件；旧 `plugins/aiw-file.py` 等路径与新目录不是父子关系，迁移可移除自身
  旧入口且不触碰 runtime 的根辅助文件。安装器的真实精确依赖校验未修改。
- notify 只改变发送器定位为自身相邻文件；console 分支仍提前返回。`-I`、环境白名单、
  配置读取、超时、协议与错误处理保持原样。开发 shim 的 `runpy(..., run_name='__main__')`
  保留原 argv 中的 `--managed-json`，canonical 脚本的 managed 分支不会载入旧 `.env`
  或采用旧凭证默认值；发行包直接包含该完整脚本。
- `entries.py` 对目录内实际 Python 入口作 AST literal META 读取，解析前限制解析后路径
  属于当前目录；manifest 覆盖名称/入口/startup 字段，行为及确认字段保留，不执行入口。
  使用文档和稳定规范说明目录路径、新旧依赖及发布边界，与实现一致。

## 实际命令与未执行项

审查命令仅为定向 `Get-Content -Encoding utf8`、`rg -n`、`Get-ChildItem`、
`git diff --stat 033da91 5889483`、`git diff --find-renames 033da91 5889483 -- ...`、
文档/规范 diff，以及 `aiw fd --help`、`aiw fd emit --help` 和上述来源事件的
`aiw fd claim ... --session fd101-review-independent-20261019`。
报告完成后通过 `aiw fd emit FD-101 verification-passed --producer reviewer
--artifact docs/features/reviews/FD-101-review.md
--source-event FD-101-000005-implementation-ready` 记录结论。

读取 Worker 双格式报告：其一次离线 compile-only 已记录 Python 内存编译 8 文件、
两个 Go 入口输出 NUL，退出码 0；Reviewer 未重复执行，也不将其视为功能验收。
没有新增或运行测试、编译、CLI 功能/规划模拟、最终包构建、lint/format/vet、验证脚本、
网络、Teams 发送、发布、安装或 Git 写操作。

剩余限制：真实 ZIP 和安装迁移尚未运行；旧已发布 ZIP 保留原依赖，需另行授权构建新版本
才改变安装行为。本审查依据静态路径和现有事务代码支持验收，不声称安装或通知已成功。
