# FD-081 Say 重复资源打包修复

<!-- aiw-data: FD-081-implementation-r1.json -->

Source event: FD-081-000003-design-ready
Worker session: fd081-host-20261018-b73a4c
基线：3baed64；父分支 develop；工作区 .wt/FD-081。

## 证据与原因

用户实际执行 build all --version 1.0.0，日志在 Say 包处报告
Duplicate source: common/plugins/aiw-say/aiw.toml.example。用户之后执行 install all，
报告缺少 all 清单。本会话未重跑这两条命令。
静态查看主工作区可见旧的 ignored 示例配置、profiles 和 Go 二进制；
package_component 先扫描插件目录，再显式从 programs 目录添加同名资源，
归档层正确拒绝重复条目。缺少 all 清单是前一失败的连带结果。

## 修复与检查

tree_entries 增加可选精确相对路径排除；Say 排除示例配置、profiles 和两个热键脚本，
统一从 src/programs/aiw-say 收集。目录在进入前剪枝，文件在加入前跳过。
其他调用不传 exclude，保持行为；重复条目检查不变，未删除用户的旧资源。
文档说明资源归属。无接口、包格式、安装状态、依赖或权限变更。

执行一次 Python 内存 compile(build.py)，通过且未写字节码。另执行一次静态 diff 检查。
未创建测试，未运行测试、build.py、完整构建、安装、网络、依赖下载或 linters。
运行构建仍待用户按原命令验证；本报告不称打包成功或安装成功。
