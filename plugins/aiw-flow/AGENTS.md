# Development Rules

- Python 3.12+ preferred; current implementation stays compatible with Python 3.9+ for local validation.
- Use typed Python.
- Prefer standard library.
- Use asyncio for subprocess execution.
- Never use shell=True.
- Never concatenate untrusted input into commands.
- All state writes must be atomic.
- All session-changing operations require locks.
- Tests must not require a real Codex installation.
- Do not weaken tests to make them pass.
- Do not commit or push.

