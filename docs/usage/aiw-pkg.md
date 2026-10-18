# 构建阶段与本地软件包

需要 Python 3.11+；Go 产物支持 Windows/Linux amd64。以下是用户显式执行的
构建与安装命令，普通 FD 自动流程不执行发布构建或安装。

```text
python build.py compile aiw
python build.py build plugins --version 1.0.0
python build.py build skills --version 1.0.0
python build.py build all --version 1.0.0
python build.py install --dest C:\green\aiw all
python build.py install --dest ~/apps/aiw all
```

compile 只编译到 bin/<windows|linux>-amd64/，每个 Go 组件一次生成双平台文件。
build 自动先调用所需 compile；任一编译失败不开始发布组装。纯资源无需编译。
它使用已有本地依赖，不下载 Go/Node 包。proxy 的编译结果位于 bin/http-openai-proxy/。
install 只使用 dist/.manifests/ 的已发布清单，不编译、不下载、不扫描猜测最新版本。
无参数显示帮助；旧的 build.py windows/linux/req 等顶层动作不再支持。

直接组件为 aiw、req、say、gateway、packer、file-convert、http-openai-proxy、cz、docs；
聚合 target 为 bin、plugins、skills、all。bin 包括 aiw/Gateway/proxy；
plugins 按 manifest 发现插件，skills 按 SKILL.md 发现 Skills。
all 是此工程的组件集合，不是远程仓库所有包；逐包安装，失败即停止，已成功的包保留。

## 发布目录

| 内容 | 位置 |
| --- | --- |
| 主程序及 Gateway 的原始双平台程序 | dist/bin/<platform>-amd64/ |
| 插件包 | dist/plugins/<version>/<package>-<version>.zip |
| Skill 包及共享资源包 | dist/skills/<version>/<package>-<version>.zip |
| 程序安装包、proxy、文档等 | dist/mist/<package>-<version>.zip |

每个 Go 插件只有一个 ZIP，内含双平台程序和公共文件；安装只发布当前平台文件。
Say 的示例配置、profiles 与热键脚本统一从 src/programs/aiw-say 收集；
源码插件目录内旧流程留下的同名生成资源不参与打包，无需先删除这些文件。
Skills 保持 canonical skills/ 布局，共享资源 aiw-skills-support 先于各 Skill 安装。
独立安装 Skill ZIP 不隐式安装其他依赖包；现有 aiw skills 的项目/用户级功能不变。

## 本地包命令

pkg 插件安装后：

```text
aiw pkg install --dest C:\green\aiw .\aiw-say-1.0.0.zip
aiw pkg remove --dest C:\green\aiw aiw-say
aiw pkg purge --dest C:\green\aiw aiw-say
aiw pkg pack ./my-package --output ./my-package-1.0.0.zip
```

--dest 优先于 AIW_INSTALL_DIR；否则 Windows 默认 C:\green\aiw，Linux 默认 ~/apps/aiw。
pkg 与 build.py install 共享同一本地安装引擎。包文件缺失、损坏或不合法时本地报错。
包名输入预留为仓库获取入口，当前明确报告尚未实现，不联网、不修改安装状态。
HTTP/S3、index 和 publish 留待后续 FD；仓库下载包使用后清理的约定届时适用。
本地输入包始终保留；解压/预检使用每次操作专有的系统临时目录。

## 包格式与工作区

工作区包含 info、data/、可选 preinst/postinst。外层包名必须与 info 一致：

```text
my-package-1.0.0.zip
  info
  data.zip
  preinst     (optional Python)
  postinst    (optional Python)
```

info 是 UTF-8 TOML。示例工作区的 data/common/hello.txt 对应：

```toml
format_version = 1
name = "my-package"
version = "1.0.0"
kind = "resource"

[[files]]
source = "common/hello.txt"
target = "share/hello.txt"
platform = "all"
config = false
executable = false
```

pack 自动计算每文件 sha256 和 data_sha256，不需要手工填校验值。
发布包中 kind 可为 program/plugin/skill/resource，每条 files 必须包含六个字段
source/target/platform/sha256/config/executable；根含 format_version/name/version/kind/data_sha256/files。
platform 可为 all/windows/linux；source 相对 data/，target 相对安装根。
平台程序可放 data/windows-amd64/ 和 data/linux-amd64/；映射到当前平台目标路径。
限制为 512 MiB 总载荷、64 MiB 单文件、20,000 ZIP 条目、4 MiB 元信息/单脚本。
禁止链接、特殊文件、路径越界、重复/冲突目标、Windows 保留名称及 .aiw/ 载荷。
SHA-256 用于完整性检查，不代替可信发布者认证。脚本是用户信任的本机代码，不是沙箱。

preinst/postinst 通过当前 Python 显式执行，不依赖 shebang 或文件扩展名。
位置参数为 name、version、action（install/upgrade）、old_version、dest；
环境提供 AIW_PKG_STAGE（临时 data/）、AIW_INSTALL_DIR、AIW_PKG_ACTION。
标准输出/错误写入事务日志；返回非零即安装失败。remove 恢复不执行这些脚本。

## 备份、配置与恢复

状态位于 DEST/.aiw/packages/，不同安装根独立。每包仅有一个 previous 备份。
成功升级才用原当前版替换备份；升级失败恢复原文件/记录并保留原备份。
同版本同内容跳过；同版本不同内容正常重装并备份原内容，包含 dev 版本。
指纹覆盖规范化元信息、文件内容和脚本，不受 ZIP 时间戳影响。

remove 恢复 previous 后删除备份，不反向备份被移除版本；没有备份时卸载程序。
remove 保留配置及其登记；purge 删除当前受管文件、登记配置及备份，
不删除无归属的用户文件，不递归删除整个安装目录。
既有配置不会被安装、升级或恢复覆盖；被用户修改的非配置受管文件阻止替换/删除，
应先自行保存和处理。未登记的旧式安装不会被静默接管；冲突文件会明确报错。

首次安装失败撤销本次新增受管非配置文件，保留配置及诊断；升级失败自动恢复。
恢复本身失败时保留 transaction 并阻止后续安装修改，报告材料位置；不自动接管失效锁。
成功后清理临时数据；失败日志/恢复快照保存在 failed-* 诊断目录，不是额外可恢复版本。
断电/强杀不保证原子恢复；脚本写外部系统的副作用不能自动撤销。
Windows 正在运行的程序若无法替换，明确失败，不启动后台自更新进程。
