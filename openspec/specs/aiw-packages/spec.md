# 本地包与构建阶段

## Purpose

FD-080 定义源码构建阶段及不依赖 Debian 工具的本地 ZIP 安装生命周期。
远程仓库、发布、索引及现有 skills/plugin 管理命令整合不属于本能力。

## Requirements

### Requirement: 显式构建阶段

build.py MUST 提供 compile/build/install，target 为组件，不接受 windows/linux target
或旧顶层动作。无参数 MUST 显示帮助且不安装。
compile MUST 只写 bin/，Go 组件 MUST 一次生成 Windows/Linux amd64 两套产物。
build MUST 先完成所选组件的 compile，再组装 dist；编译失败 MUST 不进入组装。
install MUST 只消费显式发布清单，不隐式编译、组包或下载。
Go 原始程序 MUST 发布到 dist/bin，plugin/skill 包 MUST 位于 dist/plugins 或
dist/skills 的版本子目录，其他发布文件 MUST 位于 dist/mist。
一个 Go 插件包 MUST 包含双平台产物，安装 MUST 自动选当前平台。

### Requirement: 可移植包契约

包 MUST 命名为 name-version.zip，内含 info、data.zip 及可选 Python preinst/postinst。
info MUST 为 format_version=1 的 UTF-8 TOML，声明文件来源、目标、平台、SHA-256、
配置及可执行标记。解包 MUST 在操作专有临时目录完成；载荷与声明 MUST 完全对应。
安装前 MUST 完成完整性、平台、路径、类型及目标冲突预检；MUST 拒绝链接、
越界路径、重复目标、特殊文件以及覆盖 .aiw 状态区。
包文件错误 MUST 不回退为仓库查询。本地包输入 MUST 不因使用完成而删除。
安装选中的任一目标与本地输入 ZIP 真实路径相同时 MUST 在写状态或执行脚本前拒绝。
未来仓库下载的临时包 MUST 用后清理；正式备份及失败恢复材料不属下载缓存。

### Requirement: 共同安装状态与单备份

aiw pkg install/remove/purge 与 build.py install MUST 使用相同本地引擎。
--dest MUST 优先于 AIW_INSTALL_DIR 和平台默认值（Windows C:\green\aiw；Linux ~/apps/aiw）。
状态 MUST 位于 DEST/.aiw/packages/，不同根独立；MUST 阻止互相冲突的写入及未完成恢复后的新修改。
普通 install MUST 在备份后自动接管当前包声明的无归属普通文件；
MUST 保留既有配置、无关文件，并拒绝其他包归属冲突。
没有已安装记录时，成功接管 MUST 生成可恢复旧文件的唯一 legacy 备份，
不能假定旧版 ZIP 存在或篡改事务 before 的真实记录。
remove 恢复 MUST 不自动覆盖新出现的无归属冲突文件。
用户指定的安装根 MUST 先解析为真实绝对路径，允许根及其祖先的链接别名；
根内部的路径及状态目录 MUST 继续拒绝链接/重解析点。
每包 MUST 只保留一个上一版备份，升级成功才替换；失败 MUST 保留原备份。
同版本同内容 MUST 跳过，不执行脚本或修改文件/记录/备份；同版本不同内容 MUST 正常重装。
内容指纹 MUST 忽略 ZIP 容器时间戳，覆盖实际元信息、载荷及脚本。

#### Scenario: 消费唯一备份

- WHEN v1 → v2 → v3 均安装成功
- THEN 当前为 v3，唯一备份为 v2
- WHEN 执行 remove
- THEN 恢复 v2 并删除备份，保留配置，不备份 v3
- WHEN 再次 remove
- THEN 卸载 v2，保留配置及其登记供 purge 使用

### Requirement: 失败恢复与清理

Python 脚本 MUST 显式调用解释器并记录退出码/日志，不依赖 Bash 或 Debian。
升级 postinst 失败 MUST 自动恢复旧受管文件与记录，保留配置和诊断并返回非零。
首次安装失败 MUST 撤销新增受管非配置文件，不产生成功记录。
恢复失败 MUST 保留现场并报告未完成，不宣称已恢复或安装成功。
脚本外部副作用及断电/强杀 MUST 不被宣称可原子回滚。
purge MUST 删除受管文件、登记配置和备份，MUST 保留无归属用户文件。
临时清理失败 MUST 报告实际状态，不把已完成安装伪称为未安装。
聚合安装 MUST 逐包处理并在失败时停止，不承诺整批原子性。
