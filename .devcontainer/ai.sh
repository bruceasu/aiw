#!/bin/bash
# 这个脚本是在DOCKER中运行的，默认使用balanced profile，可以通过参数指定profile
DEFAULT_PROFILE="balanced"

unixformat(){
  #check dos2unix exist
  if ! which dos2unix &>/dev/null
  then
      sed -i 's/\r//' $1
  else
      dos2unix $1 &>/dev/null
  fi
}

case $1 in
	fast|architect|balanced|heavy)
        PROFILE="${1:-$DEFAULT_PROFILE}"
        shift;;
    *)
        echo "Using default profile: $DEFAULT_PROFILE"
        PROFILE="$DEFAULT_PROFILE";;
esac

TASK=$(cat $1)
MD5SUM=$(echo -n "$TASK" | md5sum | awk '{print $1}')
DATE=$(date +%Y-%m-%d)
git checkout -b ai/task-$DATE-$MD5SUM

while true
do
  codex exec \
    --dangerously-auto-approve \
    --profile "$PROFILE" \
    "$@" "$TASK"

  scripts/verify.sh

  if [ $? -eq 0 ]; then
      git add .
      git commit -m "auto implementation"
      # send notification
      break
  fi
done

# 命令	功能	深度说明
# codex resume # 恢复 session
# codex resume --last  # 恢复最近 session

# 会话控制
# 命令	功能	深度说明
# /new	开启新会话	清空当前上下文，但保留在同一个 CLI 进程中
# /resume	恢复历史会话	打开一个选择器，显示最近的会话列表
# /fork	克隆当前会话	把当前对话复制到新线程，适合"我想试另一个方案但不想丢掉当前进度"
# /quit / /exit	退出 CLI	完全退出 Codex CLI
# /compact	压缩上下文	总结当前对话以节省 token，避免触发上下文长度限制


# 命令	功能	深度说明
# /model	切换模型和推理等级	交互式选择，支持运行中切换
# /personality	切换沟通风格	friendly（友好）/ pragmatic（务实）/ none（无风格）
# /plan	进入规划模式	让 Codex 先制定计划再执行，适合复杂任务


# 权限与状态
# 命令	功能	深度说明
# /permissions	调整授权模式	运行时切换 Auto/Read Only/Full Access
# /status	查看会话信息	显示模型、token 用量、账户信息
# /statusline	自定义状态栏	交互式调整底部状态栏显示的内容
# /debug-config	调试配置	打印完整的配置加载链路和策略诊断信息

# 授权模式深度解析
# 6.1 三种基本模式
# 权限项	Auto（默认）	Read Only	Full Access
# 读取文件	✅	✅	✅
# 编辑文件	✅	❌	✅
# 工作目录运行命令	✅	❌	✅
# 访问工作目录外文件	需确认	❌	✅
# 访问网络	需确认	❌	✅
# 6.2 精细化权限控制（Flags 参数）
# 除了三种预设模式，你还可以通过 Flags 参数组合出更精确的权限：
# # 模式一：自动编辑，但运行不可信命令时需批准
# codex --sandbox workspace-write --ask-for-approval untrusted
# # 模式二：只读，从不请求批准（纯聊天/分析场景）
# codex --sandbox read-only --ask-for-approval never
# # 模式三：完全自动（仅在隔离环境中使用！）
# codex --dangerously-bypass-approvals-and-sandbox
# # 别名：--yolo（OpenAI 官方真的用了这个名字）
# BashCopy
# 6.3 --full-auto 与 --yolo 的区别
# 这是很多人搞混的两个选项：
# 对比项	--full-auto	--yolo
# 全称	--full-auto	--dangerously-bypass-approvals-and-sandbox
# 沙箱	保留沙箱保护	完全关闭沙箱
# 审批	减少审批提示	完全关闭审批
# 网络访问	仍受沙箱控制	完全开放
# 适用场景	日常开发（低摩擦）	CI/CD 隔离环境
# 安全性	较安全	危险


# 文件与工具
# 命令	功能	深度说明
# /mention	引用文件或目录	把特定文件加入对话上下文
# /diff	查看 Git 变更	包括未追踪的文件
# /review	代码审查	支持对比分支、检查未提交更改、分析特定 commit
# /mcp	查看 MCP 工具	列出所有已配置的 MCP 工具
# /apps	浏览应用连接器	查看和插入 ChatGPT 连接器
# /ps	查看后台任务	检查后台终端的状态和输出

# 其他
# 命令	功能	深度说明
# /init	初始化 AGENTS.md	为项目生成指导文件

# 多场景别名方案
# 但一个别名显然不够。实际开发中，不同场景需要不同的配置：
# # 日常开发：高推理 + 网络搜索 + 自动模式
# alias cx='codex -m gpt-5.3-codex -c model_reasoning_effort="high" --search'
# # 代码审查：只读模式，防止误改
# alias cxr='codex -m gpt-5.3-codex --sandbox read-only --ask-for-approval never'
# # 快速问答：轻量模型，省成本
# alias cxq='codex -m o4-mini -c model_reasoning_effort="medium"'
# # 全自动模式：适合信任度高的项目（慎用）
# alias cxa='codex -m gpt-5.3-codex --full-auto --search'
# # CI/CD 脚本模式：非交互式
# alias cxci='codex exec'


# 使用 Profile 代替别名（更优雅的方案）
# Codex CLI 支持 --profile 参数，可以在配置文件中预定义多个配置组合：
# # ~/.codex/config.toml
# # 默认配置
# model = "gpt-5.3-codex"
# model_reasoning_effort = "high"
# web_search = "live"
# # 代码审查 profile
# [profiles.review]
# sandbox_mode = "read-only"
# approval_policy = "never"
# # 轻量模式 profile
# [profiles.quick]
# model = "o4-mini"
# model_reasoning_effort = "medium"
# web_search = "disabled"
# TOMLCopy
# 使用方式：
# codex --profile review   # 代码审查模式
# codex --profile quick    # 轻量快速模式
# codex                    # 使用默认配置
