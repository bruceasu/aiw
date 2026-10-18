# FD-082 输入包路径重合修复

<!-- aiw-data: FD-082-implementation-r2.json -->

Source event: FD-082-000008-changes-requested
Worker session: fd082-host-20261018-c821be

首轮通过后补查发现自动接管可能覆盖位于安装根内的输入 ZIP。原通过结论保留为历史，
PM 通过 managed set-status/request-review 获得新独立 changes-requested 交接后修复。
一次 PM 状态操作误用于主工作区计划副本，静态确认仅改自身 FD 状态/revision 和索引后
恢复为提交前内容；实际恢复操作随后在 FD worktree 完成，审计均保留，未覆盖用户修改。

Manager.install 在暂存、锁、状态写入或脚本调用前，对当前平台选中的每个目标与
输入 ZIP 做真实路径比较，重合即报错；包含配置目标和经不同 junction/符号链接别名
到达同一位置的路径。root 内链接限制不放宽，正常旧文件接管仍走原事务。
文档和稳定规范同步该边界。本次不修改包文件或安装目录。

改后同一 manager.py 内存 compile() 再执行一次通过，无字节码。
命令后附加读取一个猜测的 blocker 文件名失败，不影响前面的明确编译通过输出；
未重复运行编译，也不把该读取错误作为产品运行结果。
未执行测试、构建、安装、网络或故障注入；待独立有界复查。
