# FD-088 独立设计审阅

结论：ready。当前设计覆盖用户明确提出的固定 snapshot 编译槽和 build 自动复用要求；没有阻断实施的问题。本报告仅评估设计，不代表实现验证通过。

来源为当前用户请求与 Planner 的有界直接委派，没有关联的工作流事件，source_event 为 null。审阅范围为 FD-088 revision 1、build.py 的编译记录、库存校验、编译和组包调用链，以及 openspec/specs/aiw-packages/spec.md 的现有版本与复用契约。

## 设计判断

- compile 固定使用 snapshot，移除 compile --version 和 build --compile-version，直接消除了编译身份与包版本混用的问题。参数兼容性变化已由用户的新模型授权，FD 明确记录了替代关系。
- 包版本独立使用显式 --version、非空 AIW_VERSION、一次调用生成的 UTC 时间戳回退；AIW_VERSION 不应影响编译身份或是否允许复用。保留 latest 与 latest-compile 的不同含义合理。
- build_packages 应统一负责 snapshot 前置条件；build 和 publish 都经过该边界，避免 publish 再次按包版本编译。成功 manifest 保留 compile_version=snapshot，可继续解释历史记录。
- 仅新固定槽的 snapshot 记录可复用。旧的时间戳或发布版本编译记录不构成新槽的有效证明，应重新编译；无需迁移历史发布清单。
- 现有写入前重叠记录失效、单写者、完整原生产物路径与哈希校验和 clean 边界均保留，范围适当。源码新鲜度不属于本次复用保证，FD 已明确风险。

## 实施时需落实的非阻断边界

1. 将“普通产物确实缺失”与“路径非法、链接、权限错误、记录错误、文件内容修改或新增”分开表达。不要捕获所有 PackageError 或 OSError 后自动编译；仅可靠判定的普通缺失允许回退。
2. 同一覆盖记录中，若既有缺失文件又有修改文件、额外文件或非法路径，必须先完成其他可检查输出的检查，再决定是否自动编译，防止缺失分支掩盖应报错状态。proxy 的完整目录库存也应遵循此边界。
3. latest-compile 是只读查询；无有效 snapshot 时应清晰失败，不触发编译。build/publish 的自动补编译规则不可被查询复用为副作用。
4. 同步替换 stable spec 中旧的显式编译版本选择、AIW_VERSION 过滤复用及 publish 总是重编译条款，并更新对应场景。独立包脚本的版本契约保持原有范围。

这些点是 FD 已有错误边界与验收条件的实现提示，不增加设计决策或新的人工确认门。

## 静态证据及限制

实际仅使用 PowerShell Get-Content 与 rg 读取 FD、源码、稳定规格和 planning 规则，并写入本设计审阅及 JSON。一次历史设计报告读取因文件不存在失败；一次 rg 搜索未命中。没有重复失败命令。

未运行测试、目标编译、组包、发布、clean、网络或 Git 写入。未改变工作流事件和 FD 状态。后续实现仍需语法仅编译检查及独立实现审阅。
