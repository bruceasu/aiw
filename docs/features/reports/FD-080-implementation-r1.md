# FD-080 本地包生命周期实现

<!-- aiw-data: FD-080-implementation-r1.json -->

- Source event: FD-080-000016-design-ready
- Worker session: fd080-host-20261018-a19d7f
- 基线：fafde4f；工作区 .wt/FD-080；父分支 develop。

## 变更与静态证据

build.py 分离 compile/build/install，Go 一次编译 Windows/Linux amd64，全部所需
compile 成功才组包；插件一包双平台，Skill 逐目录组包，support 安装在前。
发布目录符合约定，清单支持 install all 及子 target；安装不触发构建。
旧顶层入口及平台 target 不保留，默认安装根按平台解析，支持 --dest。

archive.py 统一 TOML info、双层 ZIP、平台映射和内容指纹，提供 workspace pack。
先检查完整性/路径/类型/文件集合/大小，再物化到专有临时目录。
manager.py 统一互斥、归属、配置保留、Python 脚本、事务恢复与单 previous 备份。
同版本同内容跳过，内容变化重装；remove 恢复后消费备份，purge 清除登记配置。
首次失败撤销非配置文件，配置仍登记供 purge；恢复失败保留 transaction 并阻止新修改。
previous 直到新版本成功才轮换；before 保存当前文件与记录，异常自动恢复。
aiw pkg 使用既有 manifest/Python 分派，无 Go API、依赖或既有 skills/plugin 命令改动。
包名获取明确报尚未实现，本地文件错误不回退仓库。文档及稳定规范已同步。

## 实际命令及限制

Python 语言级内存 compile 通过，聚合清单调整后同命令再运行一次通过：

```text
python -c "from pathlib import Path; paths = [Path('build.py'), *Path('src/plugins/aiw-pkg').glob('*.py')]; [compile(p.read_bytes(), str(p), 'exec') for p in paths]; print('Compile-only passed:', len(paths), 'Python files; no bytecode or binary written')"
```

`git diff --check fafde4f` 通过；另读取 diff stat/status。
本地流程实际执行 force-emit、claim、design-ready、Git 计划/实现提交及 wt add。
编辑工具出现过补丁结构/上下文错误，纠正补丁后应用，无产品代码执行。

未运行 build.py、pkg 安装、Go/Node 编译、发布构建、测试、formatter/linter/vet、
网络、依赖下载、发布或部署。跨平台行为、脚本故障、Windows 占用文件、
断电/强杀及 proxy 第三方依赖未经运行验证。
脚本为可信任本机代码，不是沙箱；外部副作用不能回滚。
既有未登记文件不会被静默接管；聚合安装按包处理，不是整批原子事务。
独立实现审查和交付尚待实际执行，本报告不宣称通过。
