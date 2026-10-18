# FD-078 审查修复交接

<!-- aiw-data: FD-078-review-repair.json -->

- Worker fd078-host-20261018-a83f92；来源 FD-078-000005-changes-requested；原审查 FD-078-implementation-review。
- 使用 diagnosing-bugs 的静态诊断方法，未做运行复现；三项均为源码可证明的产品缺陷，按原 FD 授权修复。
- R1：AST 正文原来直接写 source。普通文本现复用 Goldmark DefaultWriter 的单次 Markdown 转义/实体处理，再单次 HTML text decode；raw/code 保持字面值，code span 换行转空格。避免双展开 `&amp;#65;`，也保护转义的 ampersand。未声称运行样例已通过。
- R2：原摘要包含相对 target，执行依赖 cwd/home。现要求绝对路径，不做 expanduser，resolve 后写回同一 plan；预览、规范 JSON 摘要和实际目录来自同一快照。解析结果改变则摘要改变。说明文档同步。
- R3：FontFaceSet.ready 只代表流程结束。现 await 后检查每个字体的 error 状态，拒绝打印；Mermaid 单一 Promise 和图片 decode 顺序保留。
- 非阻塞体积建议：移除本地复制的 Goldmark `_tools`、`_benchmark` 和 `testutil` 六份非运行源码。保留渲染依赖和许可证。
- 修复后实际执行同一命令 `python src/programs/md-converter/scripts/compile.py`，退出0；Python 只编译源文本，Go 离线输出 NUL，没有留存可执行文件。
- 测试、真实转换/PDF、下载/安装、网络、formatter/lint/vet、最终构建均未执行。原报告记录的运行风险保持。
- 交给独立 Reviewer 做这三项和相关文档/代码差异的有界复查，不请求新的完整审查或测试。
