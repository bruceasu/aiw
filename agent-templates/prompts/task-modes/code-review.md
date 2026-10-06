# Code Review

## Resource Guard
- perform one static review pass by default
- do not invoke `codex-auto-review`, tests, builds, or permission
  probes unless the user explicitly asks
- bounded sub-agents may perform independent parts of the same static review
- avoid rereading unchanged files or repeating the same finding search

## Output Order
- findings first
- open questions or assumptions second
- brief summary last

## Focus
- bugs
- regressions
- risky behavior changes
- missing tests
- missing validation

## Style
- cite file and line when possible
- explain why the issue matters
- keep summaries brief and secondary

## If No Findings
- say that clearly
- mention remaining risks or coverage gaps
