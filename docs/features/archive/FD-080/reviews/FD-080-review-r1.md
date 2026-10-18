# FD-080 独立实现审查 r1

<!-- aiw-data: FD-080-review-r1.json -->

结论：changes-requested，以下两项为验收缺陷，须修复后重新交接。

- 来源事件：FD-080-000017-implementation-ready（本 Reviewer 已认领）。
- Reviewer session：fd080-reviewer-20261018-6b2e94。
- 审查提交：76b563d41c2a4daa1e2afc1d98c284729b12d584；基线：fafde4f；FD revision：17。

## 必须修复

1. **R1 / P1：备份轮换的首个 rename 失败时会删除原唯一备份。**
   `src/plugins/aiw-pkg/manager.py:300` 先创建 rotating 标记，随后第 304 行
   `os.replace(previous, transaction / "old-previous")` 可能失败。例如 Windows 上
   previous 内文件被占用时，旧目录仍在 previous，old-previous 尚未产生。
   `failed` 第 214–218 行仅凭 rotating 标记就删除 previous，再尝试从不存在的
   old-previous 恢复；如果此时目录可删除，旧备份永久丢失，甚至记录 recovered=true。
   这违反“升级失败保留原有唯一备份”。应记录明确的轮换阶段和此前是否存在备份，
   仅删除本次创建的 previous；移动旧备份失败时必须保留原目录。

2. **R2 / P2：公共资源会掩盖包缺少当前平台程序的情况。**
   `src/plugins/aiw-pkg/manager.py:246` 选择 all 或当前平台条目，只用第 247 行
   `if not files` 判断平台兼容。合法的 program/plugin 包若声明 Windows 程序和
   all 公共资源，Linux 会选中公共资源并执行脚本、登记安装成功，实际没有匹配程序。
   这违反“缺少匹配平台产物明确报错”和平台不匹配在发布前失败的验收。
   应在包含平台限定文件时校验当前平台确有对应文件，或明确声明并验证支持平台；
   纯 all 资源包仍允许安装。校验必须在脚本和安装状态写入前进行。

## 已检查范围及证据

阅读实际 build.py 差异、archive.py、manager.py、pkg CLI/manifest、Worker 报告、
FD、aiw-packages 稳定规范，以及 packer/file-convert/req 的相关部署入口。
静态追踪双平台编译与组包、发布清单、共享资源顺序、配置登记、首次失败撤销、
before/previous、同版本指纹和 remove/purge。未发现需要扩大本 FD 范围的理由。
指纹排除 data.zip 容器摘要，保留各文件摘要及脚本字节摘要，静态支持忽略 ZIP 时间戳。

实际命令为 Get-Content、Get-ChildItem、rg、git diff/stat、git status、git rev-parse，
以及 aiw fd show 和认领命令。曾读取不存在的 receipt/模板路径，及一次 Select-Object
参数形式错误，均为静态读取错误，未运行产品代码；正确事件文件已读取并成功认领。
本轮写入报告、JSON 及 FD Verification，然后发出 changes-requested；未修改实现。

未执行测试、编译、构建、安装演练、网络、依赖下载、格式化、lint 或 vet。
Worker 的四文件内存 compile 为其报告证据，本 Reviewer 未重跑，也不将其当作运行验收。
跨平台行为、Windows 文件占用、故障恢复、崩溃恢复及 proxy 依赖仍未经运行验证；
本轮缺陷由明确控制流证实，不以增加未授权测试作为修复门槛。
