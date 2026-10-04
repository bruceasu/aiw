# FD-020 实施报告

<!-- aiw-data: FD-020-implementation-r1.json -->

## 结果

新增明文网关 Key 的直接认证、全部 HTTP Handler 请求的开始/结束日志，以及独立持久化的请求统计。跟踪对象为网关 principal，来源于已验证的 Bearer Key；不识别后端 Codex 登录帐户。源码实施完成，尚未独立测试或审查通过。

## 变更及理由

- config.go 新增 principals[].keys，明文常量时间匹配成功不计算请求摘要；旧 key_hashes 保持兼容，只有未匹配且存在旧摘要时才走 SHA-256。启动时校验重复凭据及明文/摘要交叉重复，禁用主体不能认证。既有真实配置保持不变，摘要不可逆，明文须由运营者从原安全来源填写。
- server.go 在统一入口关联服务端请求 ID，覆盖认证失败、参数拒绝、RPM/并发拒绝及 Models/Usage。observation.go 包装写入器，以 Unwrap/FlushError 保留 SSE 的 ResponseController deadline/flush；记录 HTTP 状态、执行状态与传输失败，避免 SSE HTTP 200 被误当作执行成功。仅记录配置中的逻辑模型和已知路由/方法，未知值归空/other。
- storage.go 加载独立 requests 元数据，observation.go 复用现有排他锁、互斥锁和 atomicJSON。新增 requests-manifest.json 防止升级后目录丢失被静默重建；读取损坏或未知版本拒绝启动。请求从 in_progress 转终态，异常重启标 interrupted，未知结束时间及耗时为 null。
- runner.go 返回实际成功启动标记。HTTP 拒绝记录与扣额 metadata 分开，Usage 新增 principal/http_groups，仅查询当前主体；原 groups、token 累计、今日 used/reserved 和扣额规则保持原逻辑。
- 更新网关/插件说明及稳定 agent-proxy 规格；未改其它现有工作区变更，未写 Git。

## 静态证据

检查配置校验和认证分支、HTTP 提前返回、SSE flush/deadline 路径、执行启动标记、存储升级及中断恢复、主体隔离和分组日期口径。HTTP 分组按 received_at 所属日，执行 groups 按 started_at 所属日；拒绝不会增加执行统计。未认证主体为空，其记录只供运营者本地查看。已知耗时与未知耗时分开，查询自身可以作为 in_progress 计入 HTTP 分组。

## 实际命令

- 使用 Get-Content、rg 和一次 git status --short 阅读相关源码、规格、工作区状态；实现后执行一次只读文件/符号检查。
- `aiw fd --help`、`aiw fd new "Agent Gateway request observability"`、`aiw fd claim FD-020 FD-020-000002-design-requested --session fd020-planner-20261003-a7c92d`：前一轮完成设计创建和 Planner 认领。
- `aiw fd show FD-020`：本轮确认已认领的 Planner 事件。
- `aiw fd emit FD-020 design-ready --producer planner --artifact docs/features/FD-020_AGENT_GATEWAY_REQUEST_OBSERVABILITY.md --source-event FD-020-000002-design-requested`。
- `aiw fd claim FD-020 FD-020-000004-design-ready --session fd020-worker-20261003-bf319e`。
- `python program/agent-gateway/scripts/compile.py`：退出码 0，Windows/Linux amd64 离线编译通过，输出 os.devnull，不保留最终产物。已检查脚本无测试、应用启动或下载操作。

## 未执行及残余风险

未运行测试、服务、运行时复现、最终产物构建、格式化、lint、vet、网络请求、发布或 Git 写操作。独立 Tester/Reviewer 尚未提供结论，不将编译通过描述成运行行为通过。

现有运行程序未替换，真实配置仍可使用旧摘要；需要运营者提供原始 Key 才能切换配置到 keys。明文配置含可用凭据，须使用私有配置和既有部署权限管理。请求文件长期保留，文件数、磁盘写入和启动扫描成本随请求量增加，本轮未做高流量运行评估或轮转。历史未记录的 HTTP 请求不能补回；磁盘写入失败时最终持久化记录可能停在 in_progress，日志报告 request_audit_failed，后续重启恢复为 interrupted。

## 交接

FD-020 工作项已完成源码及文档实施；按 Independent 策略交接 Tester，测试执行仍须遵守精确命令授权规则。Worker Session：fd020-worker-20261003-bf319e；source event：FD-020-000004-design-ready。
