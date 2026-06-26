# .github/copilot-instructions-for-java.md

Read `AGENTS.md` first, then `java/AGENTS.md`.

## Java Routing
- Spring markers:
  `spring-boot`, `@RestController`, `application.yml`, `src/main/java`
  - also load `prompts/domains/java-spring.md`
- bugfix, feature, review, debugging, test, docs, or risky work:
  also load the matching file in `prompts/task-modes/`

## Defaults
- inspect contracts, tests, and config first
- respect controller, service, and repository boundaries
- treat DTO, schema, and transaction changes as risk-sensitive
- prefer the nearest `./scripts/verify.sh`
- otherwise use project-standard Maven or Gradle checks
