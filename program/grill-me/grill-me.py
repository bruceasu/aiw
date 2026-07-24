#!/usr/bin/env python3

import os
import sys
import subprocess
import tempfile
from datetime import datetime


# AI 后端：
#   codex
#   copilot
AI_BACKEND = os.getenv("GRILL_BACKEND", "codex").strip().lower()

# CLI 程序名称或完整路径
CODEX_BIN = os.getenv("CODEX_BIN", "codex")
COPILOT_BIN = os.getenv("COPILOT_BIN", "copilot")

# 可选模型
CODEX_MODEL = os.getenv("CODEX_MODEL", "").strip()
COPILOT_MODEL = os.getenv("COPILOT_MODEL", "").strip()


def get_environment_context():
    """自动勘测本地环境。"""
    context = []

    try:
        files = subprocess.check_output(
            "find . -maxdepth 3 -not -path '*/.*' | head -n 30",
            shell=True,
            text=True,
            encoding="utf-8",
            errors="replace",
        )
        context.append(f"[Local Filesystem Structure]:\n{files}")
    except Exception:
        pass

    important_files = [
        "package.json",
        "go.mod",
        "Cargo.toml",
        "requirements.txt",
        "Makefile",
        "README.md",
    ]

    for file_name in important_files:
        if os.path.exists(file_name):
            try:
                with open(
                    file_name,
                    "r",
                    encoding="utf-8",
                    errors="replace",
                ) as file:
                    context.append(
                        f"[{file_name} Content]:\n{file.read(500)}"
                    )
            except Exception:
                pass

    try:
        branch = subprocess.check_output(
            "git rev-parse --abbrev-ref HEAD",
            shell=True,
            text=True,
            encoding="utf-8",
            errors="replace",
        ).strip()

        last_commit = subprocess.check_output(
            "git log -1 --oneline",
            shell=True,
            text=True,
            encoding="utf-8",
            errors="replace",
        ).strip()

        context.append(
            f"[Git Context]: Branch={branch}, "
            f"LastCommit={last_commit}"
        )
    except Exception:
        pass

    return "\n\n".join(context)


def messages_to_prompt(messages):
    """
    将 OpenAI messages 格式转换成 Codex/Copilot
    可以直接理解的完整对话提示词。

    因为每次调用 CLI 都是独立进程，所以每一轮都要把完整
    conversation history 重新传给 CLI。
    """
    sections = []

    for message in messages:
        role = message.get("role", "unknown")
        content = message.get("content", "")

        if role == "system":
            title = "SYSTEM INSTRUCTIONS"
        elif role == "user":
            title = "USER"
        elif role == "assistant":
            title = "ASSISTANT"
        else:
            title = role.upper()

        sections.append(
            f"===== {title} =====\n"
            f"{content}"
        )

    sections.append(
        "===== INSTRUCTION =====\n"
        "Continue the conversation as the assistant.\n"
        "Return only the next assistant response.\n"
        "Do not repeat the conversation transcript."
    )

    return "\n\n".join(sections)


def run_command(command, input_text=None):
    """执行外部 CLI，并返回标准输出。"""
    try:
        result = subprocess.run(
            command,
            input=input_text,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            encoding="utf-8",
            errors="replace",
            check=False,
        )
    except FileNotFoundError as e:
        raise RuntimeError(
            f"Command not found: {command[0]}"
        ) from e

    if result.returncode != 0:
        error_message = result.stderr.strip()

        if not error_message:
            error_message = result.stdout.strip()

        if not error_message:
            error_message = (
                f"Command exited with code {result.returncode}"
            )

        raise RuntimeError(error_message)

    output = result.stdout.strip()

    if not output:
        raise RuntimeError(
            f"{command[0]} returned an empty response."
        )

    return output


def call_codex(messages):
    """调用 Codex CLI。"""
    prompt = messages_to_prompt(messages)

    # 使用临时文件只保存 Codex 的最终回答，
    # 避免把 Codex 自身的运行日志当成回答。
    with tempfile.NamedTemporaryFile(
        mode="w",
        encoding="utf-8",
        suffix=".txt",
        prefix="grill_codex_",
        delete=False,
    ) as temp_file:
        output_file = temp_file.name

    command = [
        CODEX_BIN,
        "exec",
        "--sandbox",
        "read-only",
        "--skip-git-repo-check",
        "--output-last-message",
        output_file,
    ]

    if CODEX_MODEL:
        command.extend([
            "--model",
            CODEX_MODEL,
        ])

    # “-” 表示从标准输入读取 prompt。
    command.append("-")

    try:
        run_command(
            command,
            input_text=prompt,
        )

        if not os.path.exists(output_file):
            raise RuntimeError(
                "Codex did not create the output file."
            )

        with open(
            output_file,
            "r",
            encoding="utf-8",
            errors="replace",
        ) as file:
            reply = file.read().strip()

        if not reply:
            raise RuntimeError(
                "Codex returned an empty response."
            )

        return reply

    finally:
        try:
            os.remove(output_file)
        except OSError:
            pass


def call_copilot(messages):
    """调用 GitHub Copilot CLI。"""
    prompt = messages_to_prompt(messages)

    command = [
        COPILOT_BIN,
        "-p",
        prompt,
    ]

    if COPILOT_MODEL:
        command.extend([
            "--model",
            COPILOT_MODEL,
        ])

    return run_command(command)


def call_ai(messages):
    """根据配置调用 Codex 或 Copilot。"""
    if AI_BACKEND == "codex":
        return call_codex(messages)

    if AI_BACKEND == "copilot":
        return call_copilot(messages)

    raise RuntimeError(
        f"Unsupported GRILL_BACKEND: {AI_BACKEND}. "
        "Expected 'codex' or 'copilot'."
    )


def generate_handoff_document(
    conversation_history,
    focus_arg="",
):
    """生成脱敏的跨 Session 交接文档。"""
    handoff_system_prompt = (
        "You are an expert technical program manager.\n"
        "Write a detailed handoff document summarizing the current "
        "conversation so a fresh agent can continue the work.\n\n"
        "RULES:\n"
        "1. STORAGE LOCATION: Saved to OS temporary directory.\n"
        "2. SUGGESTED SKILLS: Include a 'Suggested Skills' section "
        "recommending specific tools such as file-read and git-diff.\n"
        "3. NO DUPLICATION: Reference existing artifacts by path "
        "instead of copying them.\n"
        "4. REDACTION: Redact API keys or passwords to [REDACTED].\n"
        "5. FOCUS: Tailor the document based on the user's focus "
        "argument."
    )

    messages = [
        {
            "role": "system",
            "content": handoff_system_prompt,
        },
        {
            "role": "user",
            "content": (
                f"Focus: {focus_arg}\n\n"
                f"History:\n{conversation_history}"
            ),
        },
    ]

    try:
        reply = call_ai(messages)

        temp_dir = tempfile.gettempdir()
        filepath = os.path.join(
            temp_dir,
            "agent_handoff_"
            f"{datetime.now().strftime('%Y%m%d_%H%M%S')}.md",
        )

        with open(
            filepath,
            "w",
            encoding="utf-8",
        ) as file:
            file.write(reply)

        print(f"\n💾 [Handoff Saved] 👉 {filepath}")

    except Exception as e:
        print(f"\n⚠️ Handoff failed: {e}")


def format_conversation_history(messages):
    """将 messages 转成 handoff 使用的文本。"""
    return "\n".join(
        f"{message['role']}: {message['content']}"
        for message in messages
    )


def main():
    if len(sys.argv) < 2:
        print(
            'Usage: grill "requirement" [next_focus]'
        )
        print()
        print("Environment:")
        print(
            "  GRILL_BACKEND=codex|copilot"
        )
        print(
            "  CODEX_BIN=codex"
        )
        print(
            "  COPILOT_BIN=copilot"
        )
        print(
            "  CODEX_MODEL=<optional model>"
        )
        print(
            "  COPILOT_MODEL=<optional model>"
        )
        sys.exit(1)

    if AI_BACKEND not in {"codex", "copilot"}:
        print(
            f"Unsupported GRILL_BACKEND: {AI_BACKEND}",
            file=sys.stderr,
        )
        print(
            "Use 'codex' or 'copilot'.",
            file=sys.stderr,
        )
        sys.exit(1)

    user_requirement = sys.argv[1]
    next_focus = (
        sys.argv[2]
        if len(sys.argv) > 2
        else ""
    )

    env_context = get_environment_context()

    # 原版 Grill-me 的核心提示词
    system_prompt = (
        "You are an elite software architect inside a local "
        "terminal environment.\n"
        "1. Interview the user relentlessly about every aspect "
        "of this until you reach a shared understanding.\n"
        "2. Walk down each branch of the decision tree, resolving "
        "dependencies between decisions one-by-one.\n"
        "3. Ask the questions ONE AT A TIME, waiting for feedback "
        "on each question before continuing. Multiple questions "
        "at once are bewildering.\n"
        "4. For each question, provide your recommended answer "
        "and rationale.\n"
        "5. If a fact can be found by exploring the environment, "
        "look it up rather than asking the user.\n"
        "When user says 'Grill Done', reply with "
        "'SUCCESS: Ready to execute.' and summarize the final spec."
    )

    messages = [
        {
            "role": "system",
            "content": system_prompt,
        },
        {
            "role": "user",
            "content": (
                f"Local Environment:\n"
                f"{env_context}\n\n"
                f"Requirement:\n"
                f"{user_requirement}"
            ),
        },
    ]

    print(
        f"\n🔥 [Grill-me Engine] "
        f"Backend: {AI_BACKEND}"
    )
    print("Session active...")

    while True:
        try:
            reply = call_ai(messages)

            print(f"\n🤖 AI:\n{reply}\n")

            messages.append({
                "role": "assistant",
                "content": reply,
            })

            if "SUCCESS: Ready to execute." in reply:
                generate_handoff_document(
                    format_conversation_history(messages),
                    next_focus,
                )
                break

            user_input = input(
                "⌨️  Your Answer: "
            )

            if not user_input.strip():
                continue

            messages.append({
                "role": "user",
                "content": user_input,
            })

        except KeyboardInterrupt:
            print()

            generate_handoff_document(
                format_conversation_history(messages),
                "Emergency recovery.",
            )
            break

        except Exception as e:
            print(
                f"\n⚠️ AI invocation failed: {e}",
                file=sys.stderr,
            )
            break


if __name__ == "__main__":
    main()