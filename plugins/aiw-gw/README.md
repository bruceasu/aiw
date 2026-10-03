# Agent Gateway 插件入口

将 `program/agent-gateway` 编译出的程序放到本目录：

- Windows：`agent-gateway.exe`
- Linux：`agent-gateway`（需要执行权限）

在仓库根目录运行 `build.bat gateway` 可生成两个平台的程序；
`build.bat plugins` 会构建并复制插件到安装目录。网关是独立 Go 模块，
构建目标必须是 `program/agent-gateway`，不可使用根目录的 `cmd/aiw-req`。

```sh
aiw gw start
# 指定其它配置：
aiw gw start --config /absolute/path/gateway.json
```

`aiw-gw.py` 需要 Python 3，定位自身目录中的程序，不通过 PATH
查找网关。未指定 `--config` 时，默认传入程序同目录的 `gateway.json`
绝对路径；显式的 `--config` / `-config`（含 `=路径` 写法）优先。
其它参数原样传递，继承调用者的工作目录、环境和标准输入输出。
Windows 返回子进程退出码；Linux 用网关程序替换入口进程，直接接收信号。
程序缺失或启动失败时，错误写入标准错误并返回非零退出码。
入口不会自动编译或下载程序。Windows 可直接运行 exe 管理服务关停。

默认配置由插件入口提供；直接运行网关程序时仍需按程序要求传入配置参数。

配置文件中的 `state_dir`、`workspace_dir`、`codex_path` 必须是当前操作系统
的绝对路径。Windows 示例：

```json
{
  "state_dir": "C:/green/aiw/plugins/aiw-agent-gateway/state",
  "workspace_dir": "C:/green/aiw/plugins/aiw-agent-gateway/work",
  "codex_path": "C:/green/nodejs/node.exe",
  "codex_script": "C:/green/nodejs/node_modules/@openai/codex/bin/codex.js"
}
```

这是路径字段片段，需要合入完整配置。按真实安装路径替换 Node/Codex 路径；
Windows npm 安装须使用 `node.exe` + `codex_script`，不能指定 `codex.cmd` 或
`codex.ps1`。模型映射和 Key 等示例占位值也必须替换后才能使用。

新配置使用 `principals[].keys` 保存运营者签发的明文网关 Key，请求直接匹配。
仅接受 `keys`，不支持 `key_hashes`，配置加载和认证均不计算摘要。原摘要字符串
可以直接作为 Key，客户端发送同一字符串即可。真实 Key 不应进入版本库或日志，
配置变更重启生效。

所有 HTTP 请求均记录请求 ID、网关调用主体、接口、结果和耗时；HTTP 元数据保存
于 `state_dir/requests`，与执行额度元数据分开。`GET /v1/usage` 的 `http_groups`
提供当前主体的请求及拒绝统计，原 `groups` 继续提供实际执行与 token 用量。
字段、重启恢复和限制见 [网关说明](../../program/agent-gateway/README.md)。
