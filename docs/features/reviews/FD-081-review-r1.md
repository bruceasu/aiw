# FD-081 独立实现审查

<!-- aiw-data: FD-081-review-r1.json -->

结论：verification-passed。本次静态审查无阻塞发现；不表示完整 build all 已运行通过。

- 来源事件：FD-081-000004-implementation-ready，已认领。
- Reviewer session：fd081-reviewer-20261018-6b2e94，与 Worker 独立。
- 审查提交：2ac50d2457dd930dbda95798bcead1cbec667c2b；基线：3baed64；FD revision：4。

## 验收证据

已检查 build.py 实际差异及 tree_entries → package_component → main 调用路径。
SAY_PROGRAM_ASSETS 明确列出 aiw.toml.example、profiles 和两个热键脚本。
目录排除使用 source 相对路径，在 os.walk 进入 profiles 前修改 directories；
文件排除在生成条目前完成，因此旧生成资源存在时不会重复进入 Say 包。
无旧资源时该排除为空操作；canonical programs/aiw-say 的显式文件收集与 profiles
独立扫描保持原样。插件 manifest、入口和其他文件不在排除集合中，双平台程序添加不变。

其他插件及 tree_entries 调用未传 exclude，原默认遍历规则保持不变。
archive.write_package 的真实重复 source 检查仍在，未改成覆盖或隐式去重。
main 仍先完成全部 package_component，再写各发布清单；此前 install all 缺清单
与用户先前 build all 失败的因果关系吻合，未引入绕过失败构建的安装路径。
使用说明与实际 canonical 来源一致。改动局限于本 FD，不重新评审 FD-080 全部功能。

## 命令与限制

实际执行 Get-Content（事件、FD、Worker 报告及 JSON、相关源码和稳定规范）、
git rev-parse HEAD、git diff 3baed64..2ac50d2 -- build.py、
git diff 3baed64..2ac50d2 -- docs/usage/aiw-pkg.md、git diff --stat 3baed64..2ac50d2、
rg（排除集合/调用点及重复来源保护），以及 aiw fd claim。
随后写入本报告/JSON、同步 FD TODO/Verification 并发出 verification-passed。

未修改实现；未执行测试、编译、完整构建、安装、网络、下载或 lint/formatter/vet。
Worker 报告中的单文件内存 compile 通过为其真实证据，本 Reviewer 未重跑。
用户原 build all/install all 命令尚未由本会话重新执行，实际打包及安装成功仍待运行验证。
