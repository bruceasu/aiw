# FD-025 独立测试阻断报告 R3

<!-- aiw-data: FD-025-test-report-r3.json -->

**blocked，未执行任何测试。** 当前正式 Tester 事件已经 claim，PM 随后静态发现默认 stop 插件入口的参数插入位置错误，要求返工后重新交接。

来源：FD-025-000007-implementation-ready；Tester：fd025-tester-20261004-a173fa；FD Revision 7；回执摘要 c9e4889371818554e3f2826921c706c6b65faebe306b24dc4dd912b290dc3d47。

PM 的静态发现：Python 插件入口只对 start 识别配置插入位置，stop 未显式指定 --config 时，默认配置插到 stop 前，导致 CLI 参数位置失效。此判断来自 PM，不是本 Tester 本轮运行结果。本 Tester 没有读 shutdown 实现或通过新命令复现该问题。

当前只修改了准备脚本的 event/revision 与 R3 授权/原始结果路径绑定，行为范围未变。准备脚本摘要 38de8d51da9b1ec1d7e16c28c1673650b6c3da53c9c1bdd05e57cdfc81630dbe；读到的安装二进制摘要 97a6753035e944876d942ec48d7ec2de029f0a93e9d7f906f49cbdc49ef378a3 仅是准备时只读摘要，未测试该安装版本。

完整现有清单 31 场景，covered=0、executed=0、passed=0、failed=0、unrun=31，要求覆盖率 0%；分支覆盖率未测量，没有原始运行证据。逐项状态见同名 JSON data.scenarios。上轮 6 通过/1 失败保留为历史证据，不能计算为当前二进制运行结果。

本轮实际执行了只读 fd show、正式 fd claim 和只读 Get-FileHash，修改准备测试的版本绑定并写本阻断报告；没有执行测试命令、没有新测试授权记录，没有启动或停止服务、临时网络、模型、下载、构建、Git 操作，也没有修改 FD 或产品源码。

建议 PM test-rejected 返 Worker，修复默认配置位置并新生成 Tester 交接。后续默认配置测试可在自有 TemporaryDirectory 复制插件、exe 与合成 gateway.json，正常停止后运行该临时插件 stop 不带 --config，断言成功；该方案不接触真实配置或凭据，尚未添加或执行，必须等待新事件及版本绑定授权。
