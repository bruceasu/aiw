# FD-082 安装根别名与旧安装接管

<!-- aiw-data: FD-082-implementation-r1.json -->

Source event: FD-082-000003-design-ready
Worker session: fd082-host-20261018-c821be
基线：3e31b68；父分支 develop；工作区 .wt/FD-082。

用户真实日志：C:\green\aiw 因祖先链接拒绝，D:\green\aiw 因旧 aiw.exe 无归属拒绝。
只读 Get-Item 确认 C:\green 为 D:\green 的 junction。未复现运行安装。
用户明确选择普通 install 自动备份并接管，此决定替代 FD-080 对无归属普通文件的拒绝。

Manager 初始化先 resolve 显式安装根，safe_path 仍检查 canonical 根和根内路径，
两种别名共享同一锁和状态。preflight 的 adopt_existing 只由 install 开启；
其他包归属、类型、根内链接和已受管文件被修改仍拒绝。remove 不自动接管无归属冲突。

before 始终保存真实原 current 和受影响非配置文件。没有 installed 记录而有旧文件时，
previous_record 根据实际快照哈希生成 synthetic legacy 记录；仅在安装成功后的 previous
副本上修改 record，不篡改事务 before，失败按真实原记录恢复。配置保留登记。
legacy 无原包 ZIP；remove 使用快照恢复字节/mode，后续安装/再次 remove 不依赖旧 ZIP。
已有正常 installed 时继续保存原当前版为唯一备份，并快照保护新增接管文件。

命令：一次 manager.py 内存 compile() 通过，无字节码；一次 git diff --check 及差异静态检查。
已同步文档、plugin 帮助和稳定规范。未执行测试、构建、安装、网络、下载或 ACL/链接更改。
未宣称 Windows 文件占用、回滚或真实安装运行成功；mode 不等于 ACL 恢复。
安装后的 pkg 插件来自已发布 ZIP；用户需重新构建包含新 manager 的发行包再安装。
