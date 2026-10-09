# Agent Gateway 独立程序

`agent-gateway` 与 `aiw.exe` 同级安装，直接调用：

- Windows：`agent-gateway.exe`
- Linux：`agent-gateway`（需要执行权限）

在仓库根目录运行 `build.bat gateway` 可在 `bin/` 生成两个平台的程序，
并安装到 `%INSTALL_DIR%`（默认 `C:\green\aiw`）。`build.bat bin` 和
`build.bat all` 也会安装 Gateway；`build.bat plugins` 不再安装 Gateway。
网关是独立 Go 模块，
构建目标必须是 `program/agent-gateway`，不可使用根目录的 `cmd/aiw-req`。

```sh
agent-gateway start --config /absolute/path/gateway.json
agent-gateway stop --config /absolute/path/gateway.json
```

Windows 直接运行 `agent-gateway.exe`。启动和停止均须显式传入 `--config`，
并使用同一配置文件。安装时将配置样例写入程序同目录的 `gateway-new.json`，
保留已有 `gateway.json`；首次使用应按实际环境修改样例并另存为 `gateway.json`。
安装不再复制 `aiw-gw.py`。旧安装目录中的插件和配置不会自动删除或迁移。

配置文件中的 `state_dir`、`workspace_dir`、`codex_path` 必须是当前操作系统
的绝对路径。Windows 示例：

```json
{
  "state_dir": "C:/green/aiw/gateway-state",
  "workspace_dir": "C:/green/aiw/gateway-work",
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
旧配置可以通过 `--config` 显式指定；移动配置时应保留或正确调整这些绝对路径。
