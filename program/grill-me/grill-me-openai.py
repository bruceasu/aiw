#!/usr/bin/env python

import os
import sys
import json
import subprocess
import tempfile
from datetime import datetime
from openai import OpenAI

# 自动初始化 API 客户端
client = OpenAI(
    api_key=os.getenv("OPENAI_API_KEY"),
    base_url=os.getenv("OPENAI_API_BASE", "https://openai.com") 
)

def get_environment_context():
    """高流精髓：自动勘测本地环境，拒绝白痴提问"""
    context = []
    try:
        files = subprocess.check_output("find . -maxdepth 3 -not -path '*/.*' | head -n 30", shell=True, text=True)
        context.append(f"[Local Filesystem Structure]:\n{files}")
    except Exception: pass

    important_files = ['package.json', 'go.mod', 'Cargo.toml', 'requirements.txt', 'Makefile', 'README.md']
    for f in important_files:
        if os.path.exists(f):
            try:
                with open(f, 'r', encoding='utf-8') as file:
                    context.append(f"[{f} Content]:\n{file.read(500)}")
            except Exception: pass

    try:
        branch = subprocess.check_output("git rev-parse --abbrev-ref HEAD", shell=True, text=True).strip()
        last_commit = subprocess.check_output("git log -1 --oneline", shell=True, text=True).strip()
        context.append(f"[Git Context]: Branch={branch}, LastCommit={last_commit}")
    except Exception: pass

    return "\n\n".join(context)

def generate_handoff_document(conversation_history, focus_arg=""):
    """高流精髓：生成无污染、脱敏的跨 Session 交接文档"""
    handoff_system_prompt = (
        "You are an expert technical program manager.\n"
        "Write a detailed handoff document summarizing the current conversation so a fresh agent can continue the work.\n\n"
        "RULES:\n"
        "1. STORAGE LOCATION: Saved to OS temporary directory.\n"
        "2. SUGGESTED SKILLS: Include a 'Suggested Skills' section recommending specific tools (file-read, git-diff, etc.).\n"
        "3. NO DUPLICATION: Reference existing artifacts by path instead of copying.\n"
        "4. REDACTION: Redact API keys or passwords to [REDACTED].\n"
        "5. FOCUS: Tailor the document based on the user's focus argument."
    )
    try:
        response = client.chat.completions.create(
            model=os.getenv("CODEX_MODEL", "gpt-4o"),
            messages=[{"role": "system", "content": handoff_system_prompt}, {"role": "user", "content": f"Focus: {focus_arg}\n\nHistory:\n{conversation_history}"}]
        )
        temp_dir = tempfile.gettempdir()
        filepath = os.path.join(temp_dir, f"agent_handoff_{datetime.now().strftime('%Y%m%d_%H%M%S')}.md")
        with open(filepath, "w", encoding="utf-8") as f: f.write(response.choices.message.content)
        print(f"\n💾 [Handoff Saved] 👉 {filepath}")
    except Exception as e: print(f"\n⚠️ Handoff failed: {e}")

def main():
    if len(sys.argv) < 2:
        print("Usage: grill \"requirement\" [next_focus]")
        sys.exit(1)

    user_requirement = sys.argv[1]
    next_focus = sys.argv[2] if len(sys.argv) > 2 else ""
    env_context = get_environment_context()

    # 完美融入原版 Grill-me 的 3 大灵魂提示词
    system_prompt = (
        "You are an elite software architect inside a local terminal environment.\n"
        "1. Interview the user relentlessly about every aspect of this until you reach a shared understanding.\n"
        "2. Walk down each branch of the decision tree, resolving dependencies between decisions one-by-one.\n"
        "3. Ask the questions ONE AT A TIME, waiting for feedback on each question before continuing. Multiple questions at once are bewildering.\n"
        "4. For each question, provide your recommended answer and rationale.\n"
        "5. If a fact can be found by exploring the environment, look it up rather than asking the user.\n"
        "When user says 'Grill Done', reply with 'SUCCESS: Ready to execute.' and summarize the final spec."
    )

    messages = [
        {"role": "system", "content": system_prompt},
        {"role": "user", "content": f"Local Environment:\n{env_context}\n\nRequirement:\n{user_requirement}"}
    ]

    print("\n🔥 [Grill-me Engine] Session active...")
    while True:
        try:
            response = client.chat.completions.create(model=os.getenv("CODEX_MODEL", "gpt-4o"), messages=messages)
            reply = response.choices.message.content
            print(f"\n🤖 AI:\n{reply}\n")
            messages.append({"role": "assistant", "content": reply})
            
            if "SUCCESS: Ready to execute." in reply:
                generate_handoff_document("\n".join([f"{m['role']}: {m['content']}" for m in messages]), next_focus)
                break
                
            user_input = input("⌨️  Your Answer: ")
            if not user_input.strip(): continue
            messages.append({"role": "user", "content": user_input})
        except KeyboardInterrupt:
            generate_handoff_document("\n".join([f"{m['role']}: {m['content']}" for m in messages]), "Emergency recovery.")
            break

if name == "__main__": main()