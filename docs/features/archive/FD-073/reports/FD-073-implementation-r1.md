# FD-073 实现报告

<!-- aiw-data: FD-073-implementation-r1.json -->

Worker fd073-worker-20261017-c7a2；来源 FD-073-000003-design-ready；基线91aaec3，实现c569141。

publish-github-issue 以实际本地Issue/REQ或FD为来源，不要求OpenSpec change。关联FD默认复用Issue所有者映射，多来源独立受管块；无Issue或明确分开跟踪的FD可独立。新映射约定位于docs/external/github/issue或fd，跨归档稳定；旧version1/OpenSpec映射和helper接口继续可用，迁移必须明确。

新增publication引用，记录映射所有者与投影来源区别、独立策略和冲突阻断；远程写入前独占/原子保存pending资料，响应先入凭据、再保存映射、最后清理自己的凭据。未知结果不因无映射就重复create。该流程由Skill执行者依现有本地/远程工具实施，helper不是自动发布引擎，也不管理pending/锁或发网络请求。

纯helper新增source_markers、mapping_path、render_source_projection；replace_managed_block增加可选来源参数和明确旧块迁移。新路径检查重复/残缺/逆序/嵌套/不匹配markers，保留块外文本及其它来源，限制64KiB；旧两参数路径保留历史空白行为。load/save严格校验version/repository/正整数Issue号/URL/可选所有者与策略，拒绝静默改目标，临时文件加原子replace写映射。

源/项目入口、README、UI metadata、引用和helper同步；目录说明及稳定github-issue-projection规格同步。设计D1独立策略、D2未知create的pending资料均经r2独立设计ready；不将设计意见当成实现通过。

## 实际验证

- 使用Python compile(source, path, 'exec')在内存编译两份projection.py，退出0，无helper导入/执行，无pyc或发行产物；没有运行无关Go构建。
- 一批只读AST/文件/路径/Git diff核对退出0：5组源/项目副本一致，31个本地链接可达，旧helper名称和两参数签名保留，实际实现范围为13个Skill/规格文件，历史test_projection.py未变。
- 静态追踪场景：Issue多个FD共享映射、明确独立FD、不同目标阻断、未知create恢复、原子保存失败、旧映射迁移与块外人工内容。此为源码/指令证据，没有执行用例或远程实验。
- 执行了定向Get-Content/rg、受管FD与专属worktree/本FD提交，以及上述compile和静态检查。未运行测试、模型、GitHub/其它网络、安装、最终构建、格式化/lint/vet或验证脚本。

## 限制和下一步

pending步骤是Skill执行合同，不是新的后台事务服务；遵守情况未通过模型或远程运行验证。GitHub整body更新没有CAS保证，读取后仍有竞争窗口；明确停止条件而非声称无竞争。外部发行/全局Skill未更新。旧测试保留但未执行，不冒充已验证兼容。

下一步独立实现Reviewer，符合门槛后自动本地合并归档；本轮confirm仅授权实现，不授权实际发布。
