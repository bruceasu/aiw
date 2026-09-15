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
- use static review by default
- after implementation, run one compile-only check using a `compile*` script or
  the narrowest Maven or Gradle compile goal; do not retain a final artifact
- do not run tests, `build*` scripts, final-artifact builds, verification
  scripts, network calls, or permission probes without authorization
- when authorized, run one module-focused command and ask before widening
