# Prompt Authoring

## Treat Instructions As Code
- keep entry files short and stable
- put detailed rules into small reusable prompt files
- organize files by repo type, domain, and task mode

## Avoid Common Failures
- avoid duplicated rules
- avoid conflicting rules
- avoid very broad files with weak routing
- avoid stale examples and stale validation commands

## For Every Prompt File
- say when it should load
- say what should stay stable
- say how to validate
- keep one main concern per file

## Maintenance
- update routing files when you add a new prompt
- remove prompts that no longer match the repo
- keep examples and verification scripts aligned with the real template
