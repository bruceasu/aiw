# FD-082 独立实现审查

<!-- aiw-data: FD-082-review-r1.json -->

结论：verification-passed，无阻塞发现；结论为静态审查，不表示实际安装成功。

- 来源事件：FD-082-000004-implementation-ready，已认领。
- Reviewer session：fd082-reviewer-20261018-6b2e94，与 Worker 及设计 Reviewer 独立。
- 审查提交：fc26fe61246626c7991ffb059233616c6f5eeb8a；基线：3e31b68；FD revision：4。

## 验收与失败路径

检查 manager.py 实际差异及完整调用路径。Manager 在边界 resolve 安装根，
两个目录别名指向同一真实根时使用同一 .aiw/packages 与锁。后续仍调用严格 safe_path，
未放宽根内部目标、状态区和恢复文件的链接检查；不更改用户链接或 ACL。

adopt_existing 仅 install 显式开启。其他包归属、非普通文件、路径父目录冲突、
已有受管非配置文件内容变化及配置分类变化仍由原预检拒绝；remove 保持不接管冲突。
无归属普通文件在脚本和发布前进入 before 快照，包含原字节摘要和 mode。

previous_record 对无 installed 记录且确有旧非配置文件的情况构造 legacy 记录，
使用实际快照摘要；不存在旧文件或仅有保留配置时返回 None，不虚构旧程序备份。
已有 installed 时直接沿用真实当前记录；removed 配置记录通过配置合并继续保留。
事务 before 保留真实原记录，synthetic legacy 只写入 previous 的独立副本。
快照/记录生成失败发生在发布前；发布、脚本或备份副本改写失败沿 failed 恢复真实
before，原 previous 仍受轮换恢复规则保护，不把失败接管记为成功。

remove 通过 previous 快照恢复旧字节/mode，并删除原先不存在的新增文件；
恢复后 current.zip 可缺失，原 snapshot/restore 对旧 ZIP 使用可选分支，
不调用包读取器。恢复出的 legacy 记录保存实际旧文件摘要，支持后续升级、
再次 remove 或 purge；配置继续保留或按 purge 规则删除。单备份消费规则未改变。

使用说明、插件帮助和稳定规范与用户确认的自动接管规则一致；不承诺 ACL 或崩溃原子恢复。
Worker 报告的内存 compile 与未执行项目表述准确；发布包需要重新构建才能携带新引擎。

## 命令与限制

实际执行 Get-Content（事件、FD、完整 manager、设计报告、Worker 报告及 JSON）、
git rev-parse HEAD、git diff 3e31b68..fc26fe6 -- src/plugins/aiw-pkg/manager.py、
git diff 3e31b68..fc26fe6 -- openspec/specs/aiw-packages/spec.md docs/usage/aiw-pkg.md src/plugins/aiw-pkg/plugin.toml，
以及 aiw fd claim。随后写报告/JSON、同步 FD TODO/Verification 并发出 verification-passed。

未修改实现、提交、交付；未执行测试、编译、构建、安装、网络、下载、lint 或权限/链接操作。
本 Reviewer 未重跑 Worker 的 compile。Windows junction 运行、文件占用、接管/恢复故障和
真实安装未经本轮运行验证，mode 恢复不代表 ACL 恢复。
