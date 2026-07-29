# AGENTS.md

## Purpose
This is the default local rule file for Java subtrees.
Use it when the task is mostly Java or the current directory contains Java markers.

## Inspect First
- `pom.xml`, `build.gradle*`, and wrapper files
- controllers or handlers near the touched path
- services, repositories, DTOs, and config classes
- tests near the touched code

## Detect The Java Domain
- Spring markers:
  `spring-boot`, `@RestController`, `application.yml`, `src/main/java`
  - also load `../prompts/domains/java-spring.md`

## Keep Stable
- controller, service, repository, and config boundaries
- request and response contracts
- transaction boundaries
- dependency injection and build-tool conventions

## High-Risk Java Areas
- dependency upgrades
- auth, schema, or migration changes
- transaction changes
- public API or event contract changes

## Validation Options
Use static review by default. Do not automatically run `scripts/verify.sh`,
Maven, Gradle, tests, or builds.

When the shared resource budget authorizes runtime validation, choose one
smallest relevant command:
- `mvn -pl module -am test`
- `./gradlew :module:test`

Ask before repository-wide commands. Rerun only after a relevant change.

## Escalation
If a deeper subtree has its own `AGENTS.md` or `CODEX.md`, prefer that local file.
