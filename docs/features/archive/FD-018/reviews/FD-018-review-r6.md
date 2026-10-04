# FD-018 独立静态审查（修订 6）

<!-- aiw-data: FD-018-review-r6.json -->

结论：静态审查通过，未发现阻塞问题。此结论不表示网关启动、Windows/Linux 信号行为或安装运行已验证。

- FD：`FD-018`，修订 6，Test policy 为 External，Evidence policy 为 Dual。
- 来源事件：`FD-018-000006-review-requested`。
- Reviewer session：`codex-reviewer-FD018-20261004-r6`；本会话未实施入口代码。
- 审查版本：`c9b6d51cd2308e1ffbaf51755d697f6acda3ad83`，范围为该提交相对父提交的四个文件及当前相同的工作区实现：`plugins/aiw-agent-gateway/aiw-agent-gateway.py`、同目录 `README.md`、`.gitignore`、根目录 `build.bat`。上述文件工作区相对 HEAD 无差异。HEAD 中其它网关改动不属于此 FD 的重新审查范围。
- 恢复依据：FD Verification 记录用户一次性授权恢复状态，CLI 已取消过期修订 2 Planner 请求并创建修订 6 Reviewer 请求。没有当前 Worker 报告或完成事件；本报告直接检查实际提交和源码，不将恢复记录冒充 Worker 交接或运行结果。

## 工作项及验收依据

| 工作项 | 静态结论与依据 |
| --- | --- |
| 1.1 跨平台入口 | `Path(__file__).resolve().parent` 定位自身目录；Windows 选择 `agent-gateway.exe`，其余平台选择 `agent-gateway`。命令为参数列表，Windows `subprocess.call` 未启用 shell，Linux `os.execv` 替换进程。未覆盖标准流、环境或工作目录。缺文件及 `OSError` 向 stderr 输出并返回 1；Windows 返回子进程退出码。 |
| 1.2 说明与忽略规则 | README 说明 Python 3、两个文件名、Linux 执行权限、构建来源、配置路径与 Windows 服务关停限制；`.gitignore` 忽略两种二进制和 Python 字节码目录。 |
| 1.3 默认配置 | 未见 `--config`、`-config` 或各自等号形式时插入同目录 `gateway.json` 的绝对路径；可选首参数 `start` 保持首位。显式选项原样传递。`--` 后参数不误作配置选项；无效位置参数仍由网关拒绝。`program/agent-gateway/main.go` 移除可选 `start` 后调用 Go flag 解析，接受这些配置形式。 |
| 1.4 构建目标 | `:build_gateway` pushd 到独立模块 `program/agent-gateway`，以 `go build ... .` 构建 Windows/Linux，使用根目录绝对输出路径。两条编译失败分支及成功路径均 popd；pushd 失败立即返回。`:build_plugins` 先调用网关构建，再复制插件目录。 |

`internal/plugin/discover.go` 查找一层子目录的 `aiw-<name>.py`；`internal/plugin/exec.go` 解析 Python 解释器、以参数列表执行并传递 stdin/stdout/stderr 和环境，因此入口可由既有插件链发现及执行。网关二进制名不具有 `aiw-` 前缀，不会覆盖该 Python 入口。入口只使用 Python 标准库，没有编译、下载或依赖安装路径。

已对照 `openspec/specs/cli-and-plugins/spec.md` 与 `openspec/specs/agent-proxy/spec.md`；README 中现行 `principals[].keys` 描述与稳定规格一致。Issue 为 none；本 FD 未链接 OpenSpec change。未改公共 API、网关认证或存储实现。

## 实际命令及证据边界

- 静态发现读取：定向 `Get-Content` / `rg`，读取 FD、规则、报告模板、来源事件、四个目标文件及上述调用链和规格；首次 PowerShell 默认编码输出不正确，随后以 `-Encoding utf8` 读取必要中文证据。
- CLI 帮助：`aiw fd --help`、`aiw fd claim --help`、`aiw fd emit --help`。
- 提交定位：`git rev-parse HEAD`、`git log -3 --format='%h %s' -- plugins/aiw-agent-gateway/aiw-agent-gateway.py build.bat`、`git show --format=fuller --stat HEAD -- plugins/aiw-agent-gateway/aiw-agent-gateway.py plugins/aiw-agent-gateway/README.md plugins/aiw-agent-gateway/.gitignore build.bat`；范围内变更为 135 行新增、1 行删除。
- 工作区对照：`git diff -- plugins/aiw-agent-gateway build.bat`，目标范围无未提交代码差异。
- 领取：`aiw fd claim FD-018 FD-018-000006-review-requested --session codex-reviewer-FD018-20261004-r6`，成功。
- 报告写入后唯一静态校验：`git diff HEAD^ HEAD -- plugins/aiw-agent-gateway/aiw-agent-gateway.py plugins/aiw-agent-gateway/README.md plugins/aiw-agent-gateway/.gitignore build.bat`，检查实际提交差异。
- 结果交接：`aiw fd emit FD-018 verification-passed --producer reviewer --artifact docs/features/reviews/FD-018-review-r6.md --source-event FD-018-000006-review-requested`；交接结果以 CLI 收据为准。

未运行测试、编译、最终构建、格式化、lint、网络或部署。FD 中旧编译成功仅是历史记录，本 Reviewer 不复述为本轮执行结果。未作 Git 写操作。

## 残余风险

实际插件发现/解释器供应、跨平台启动、标准流及退出码、Windows 控制信号、构建安装脚本尚无本轮运行证据，按 External 策略由外部安排。FD 历史记录指出已有二进制曾错误编译为需求 CLI；本轮仅确认源构建目标正确，未重建或替换已有产物，现有部署需运营者另行更新。静态通过允许后续状态决策，不授权构建、发布或部署。
