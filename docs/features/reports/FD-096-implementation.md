# FD-096 基于 index 提前跳过安装和更新

<!-- aiw-data: FD-096-implementation.json -->

用户明确优先安装/更新速度，接受本地修改，不需要旧文件内容校验来阻止更新。
本次在当前主工作区完成，无分支/worktree/Worker handoff。

Operation 先用 index 与安装记录的 name/version/fingerprint 选择保留节点。
相同节点不 fetch、不解包、不检查载荷路径或计算旧文件哈希；仍使用 index 的完整
依赖/成员规划，再执行原 collection trim、最终 exact-version/cycle 检查。
锁内重新核对 metadata 快照和未完成事务，保留 reason/member 更新且仅变化时写记录。

变化节点使用 Repo.prepare 或 LocalPackages.prepare 返回单次验证的 prepared tuple，
在首次自更新写入前固定载荷。Repo 对新字节只计算一次 raw SHA-256，单次解包校验
内部完整性与 index metadata；fetch 保留原有验证后返回路径的接口。显式 ZIP 的 CLI
已验证 tuple 直接交给 Operation，避免二次解包。build.py 用 exact root_requests
从内存 index 选择根，不再额外 fetch/decode 根 ZIP；本地最初 catalog 和输入防覆盖
preflight 仍保留，因此其初始 ZIP 扫描不属于零获取承诺。

Manager metadata-only 读取仍校验相对目标语法及 .aiw 保留区，不访问载荷路径。
install/update preflight 不再调用 unchanged(old)；普通托管文件修改可覆盖，旧文件
可按包声明移除，配置文件保留。真正写入前的路径/链接/类型/归属、快照、恢复和
hooks 仍在。remove/purge 的 unchanged 检查及项目/用户 Skill 保护未改动。
512 MiB 总量只计算实际获取的 ZIP；保留节点不占用下载/暂存配额。
unchanged dry-run 不获取包；changed dry-run 仍完整准备预检，不改安装记录或文件。

实际执行一次六文件内存 compile（exit 0），及一批最终 diff/关键调用点静态检查：

```powershell
python -c "from pathlib import Path; paths = ['build.py', 'scripts/local_packages.py', 'src/plugins/aiw-pkg/repository.py', 'src/plugins/aiw-pkg/operations.py', 'src/plugins/aiw-pkg/manager.py', 'src/plugins/aiw-pkg/aiw-pkg.py']; [compile(Path(p).read_text(encoding='utf-8-sig'), p, 'exec') for p in paths]; print('Python compile-only: OK')"
```

未运行测试、安装/更新、benchmark、网络、下载、构建或发布。未声称实测速度或已安装
程序已经生效。source_event=null，因为沿用用户选择的 primary workspace，使用受管
状态/review 路由。独立实现审阅待完成。

已修改/删除的本地文件在相同版本/指纹下不会自动修复；这是此次采用的行为。