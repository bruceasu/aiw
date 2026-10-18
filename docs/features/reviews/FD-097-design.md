# FD-097 有界独立设计反馈

<!-- aiw-data: FD-097-design.json -->

结论：ready。审阅对象为 FD-097 Revision 2 的设计，不构成实现审阅通过证据。

本次按 Planner 委派，独立静态检查启动时机、父环境优先级、Windows 键名、错误传播、插件继承及 LLM 兼容影响。未领取或产生流程事件，source_event 为 null。

## 已核对

- 在 main 的参数分支前加载，能够覆盖无参数帮助、版本、内建命令及插件发现。只定位 os.Executable 所在目录，避免随当前项目切换自动导入其他环境文件。
- 父进程变量包括空值优先，采用独立 defaults-only 模式，避免改动现有 Loader 的覆盖约定；文件默认值对后续引用可见。
- Windows 新模式统一键名，同时用于继承变量查找、默认值引用及递归检测，能够定义大小写不同的同名变量行为。
- 缺文件忽略，实际读取、解析或导出失败在 dispatch 前退出；循环展开改为 error，避免 panic。错误不得打印配置值。
- 既有 ExecPlugin 两条路径均从 os.Environ 继承，因此无需把默认值重新复制成独立插件配置。引擎保留的调用元数据仍覆盖默认值。
- LLM 的显式配置优先级不变；安装目录默认值进入进程环境后会先于 cwd dotenv fallback，这是明确记录的兼容变化。直接启动插件绕过主程序的范围限制也已明确。

## 实现关注点

这些是落实已决定语义的检查点，不是额外设计阻塞：父变量已存在时，应在展开文件值前跳过该赋值，避免被忽略的文件值仍影响错误结果；单引号和多行赋值也必须经过同一默认值规则。Windows 归一化需要贯穿查找和 visiting 集合，不能仅在最终 os.Setenv 时处理。

## 证据与限制

静态阅读 FD、src/internal/util/env/loader.go、src/cmd/aiw/main.go、src/internal/plugin/environment.go、src/internal/plugin/exec.go、src/internal/ai/llm.go 及 cli-and-plugins 稳定规格。未运行测试、运行诊断、编译、构建、安装或网络调用；未修改实现、流程状态、事件或 Git。未触碰现有无关 git-guide.py 修改。
