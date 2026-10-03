# FD-017 实现报告（首轮）

<!-- aiw-data: FD-017-implementation-r16.json -->

## 来源与范围

Worker：`fd017-worker-20261003-9059c335`。已 claim 来源 `FD-017-000016-design-ready`。基线 HEAD：`9ae91484d4835a50dad6d2aebbcad93c509583f9`；审查当前工作区范围 diff 与新增源码。用户授权 `$implement $fd-workflow auto handle FD-017`，主工作区顺序实现。未创建工作树、提交或合并；原技能修改、handoff 输入和批准 Issue 属于已有会话内容，未经改写，不混入本实现。

## 实际结果

| Work Items | 实现证据 |
| --- | --- |
| 1.1–1.6、1.44、1.47、1.49–1.50 | FD Engineering decisions 明确 SDK 子集、平台进程树、存储/额度/内容生命周期；真实 design-ready receipt |
| 1.8–1.13 | 独立 go.mod、main.go、config.go、protocol.go、server.go：启动校验、Key/模型授权、严格输入、标准 Response/错误 |
| 1.14–1.19、1.51–1.53 | runner.go、process_linux.go、process_windows.go：独立 cwd/白名单、Codex JSONL、上下文/输出界限、进程组或 Job Object、统一收尾 |
| 1.20–1.22 | aiw-ai.mjs 与 package.json：网关 Key/逻辑模型、新 Responses、instructions/json_object、诚实 verbose |
| 1.23–1.24 | server.go eventStream：Responses SSE、标准字段/序号、5 秒写 deadline；断连取消同一执行 |
| 1.25–1.30、1.42–1.43、1.45–1.46、1.48 | Gateway/Store：滚动 RPM/并发、每日持久预留/扣除、重启恢复、统计分组、独立正文保存与启动/周期到期清理 |
| 1.37–1.40 | 配置 fail-closed；退役旧 TS/ACK/旧脚本和依赖；服务/客户端 README、两份稳定规格同步 |

45 项范围内任务标为编写完成，8 项取消；这不是运行测试通过声明。未知用量不伪造，Usage 分开 today_used（实际启动）与 today_reserved（占用但未确认），统计 groups 只累计实际启动。正文到期不删除长期元数据，异常预留不自动返还。原 TypeScript 状态不删、不迁移。

## 实际执行与静态证据

- 两次 `python plugins/aiw-agent-proxy/scripts/compile.py` 均 exit 0；第一次在核心代码编写后，第二次在所有最终源码完成后。脚本限定独立插件、Windows/Linux amd64，GOPROXY/GOSUMDB off、GOTOOLCHAIN local、CGO disabled，输出 NUL，没有最终产物或下载。
- 一批最终只读检查：git diff --stat、限定客户端/入口 diff、Windows handle 生命周期与 evidence schema 读取、git status --short；追踪 config→auth→quota→Runner→store→HTTP/SSE。个别注册路径 rg 使用了不存在路径并报错，没有运行权限探测或扩大验证；保留现有同名 JS 插件入口。
- 生命周期命令：真实 design-ready emit、claim Worker；implementation-ready 在本报告完成后执行，事件结果以 receipt 为准。
- 仅查询官方协议/CLI 文档，无模型或 Provider 请求。

## 未运行及残余风险

Test policy External：没有编写/运行测试或创建 Tester；没有启动网关/Codex/SDK，未做跨平台进程、断连、额度竞争、重启或到期运行检查；没有 lint/formatter/vet、最终发布构建或容器部署。静态审查与编译不能证明这些行为已经通过。

共享部署、宿主隔离、Windows ACL/登录供给由用户负责；共容器跨读和 Linux 恶意脱组仍存在。Windows 原生 API/Job 生命周期虽可编译，运行未验证。文件 Sync/Rename 不提供 OS spawn 与持久化原子事务或硬件断电保证；reserved 崩溃窗口需要运营者离线核实。异常旧工作区不自动删除，运营者确认进程树退出后处理。默认文本增量为完成消息粒度，非逐 token 输出。高流量历史查询/正文清理当前使用单实例文件扫描，尚未测性能。

## 下一阶段

通过 implementation-ready 移交独立 Reviewer；不在本 Worker 会话撰写 Reviewer 结论或宣称 FD Complete。
