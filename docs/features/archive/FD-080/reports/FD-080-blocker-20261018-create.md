# FD-080 创建流程阻塞

<!-- aiw-data: FD-080-blocker-20261018-create.json -->

阶段：Planner / Design。记录日期：2026-10-18（Asia/Tokyo）。
会话：fd080-planner-20261018-create。关联事件：未读取确认，故记为无；不声明已经 claim。

执行 `bin/aiw-windows-amd64.exe fd new "Separate compile build install and unify package management"`
返回 1：替换 docs/features/FEATURE_INDEX.md 时 WinError 5（拒绝访问）。
已生成 FD-080 设计文件，保留一个 FEATURE_INDEX.md.tmp-17764-8c7de04136be4ef989209df8dcfada2e。
只静态查看了生成文件与目录，未重试命令、未提权、未删除现场、未更换执行方式。
索引替换拒绝访问的根因未知，不能据此认定为文件锁或 ACL 问题。

原始停止状态：未解决。用户后续已确认统一 Python 脚本、remove 恢复上一版并保留
配置、不保留旧入口、安装状态位于各安装根 .aiw/packages/。
设计文件已写出供审阅，但不进入实现、Git 提交或 worktree 创建。
下一步是解决索引替换问题后，通过受支持 CLI 检查实际交接状态并恢复，
避免再次分配 FD 或伪造 receipt。未执行测试、构建、安装和网络请求。

可复用改进建议：FD 创建应尽可能具备原子性，并在索引失败时明确报告已生成文件
及恢复入口；本记录仅提出建议，未实现相关工具修改。

## 后续恢复

2026-10-18，用户明确授权 PM 交接恢复；force-emit design-requested 成功更新 FD、
索引及审计，当前 Planner 事件已成功认领，设计文件与事件未重复创建。
原索引写入阻塞在本次实际写入成功后解除；最初 WinError 5 根因仍未知。
失效 Planner 交接的恢复详情见 FD-080-blocker-20261018-auto-preflight.md。
