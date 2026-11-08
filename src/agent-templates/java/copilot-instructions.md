# .github/copilot-instructions-for-java.md

Read `AGENTS.md` first, then `java/AGENTS.md`.

## Java Routing
- Spring markers:
  `spring-boot`, `@RestController`, `application.yml`, `src/main/java`
  - also load `.agents/prompts/domains/java-spring.md`
- bugfix, feature, review, debugging, test, docs, or risky work:
  also load the matching file in `.agents/prompts/task-modes/`

## Defaults
- inspect contracts, tests, and config first
- respect controller, service, and repository boundaries
- treat DTO, schema, and transaction changes as risk-sensitive
- use static review by default
- do not run Maven/Gradle test or build goals, tests, builds, verification
  scripts, external runtime network calls,
  or permission probes without resource-budget authorization
- use Maven or Gradle for project library dependency resolution under
  `.agents/prompts/core/resource-budget.md`
- when authorized, run one module-focused command and ask before widening
