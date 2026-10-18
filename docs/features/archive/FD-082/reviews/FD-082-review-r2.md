# FD-082 输入包保护补充审查 r2

<!-- aiw-data: FD-082-review-r2.json -->

结论：changes-requested。来源 FD-082-000007-review-requested 已由独立
session fd082-reviewer-20261018-6b2e94 认领；审查提交仍为
fc26fe61246626c7991ffb059233616c6f5eeb8a，FD revision 7。

## R1 / P1：自动接管可覆盖本地输入 ZIP

沿用已确认的 reviews/FD-082-review-r1-supplement.md 证据：Manager.install 先
read_package，再对本次目标开启 adopt_existing，未检查 target 与输入包路径重合。
输入为 DEST/demo-1.zip、载荷 target 为 demo-1.zip 时，通过预检后覆盖输入包；
随后 package.read_bytes() 保存覆盖后的字节到 current.zip。
违反本地输入包应保留的现有契约，属于本次自动接管新增的写入路径。

修复要求：发布前比较当前平台实际选中目标与输入包的真实路径，重合则拒绝，
不得写输入包或运行安装脚本；处理安装根别名，并保留内部链接限制。
未执行运行验证，该缺陷由直接控制流确认。

原 000005 verification-passed 作为历史保留；000007 已明确 supersede 它，
本次发出真实 changes-requested 返回 Worker，不用原通过结论支持交付。
本轮仅读取新事件、FD 头和 HEAD，并执行 claim；未重复审查无关功能，未修改实现。
未执行测试、编译、构建、安装或网络。修复后须基于新事件复查，不重置审查历史。
