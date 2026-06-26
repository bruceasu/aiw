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

## Validation
Prefer the nearest `./scripts/verify.sh`.
Otherwise use project-standard Maven or Gradle commands.

## Escalation
If a deeper subtree has its own `AGENTS.md` or `CODEX.md`, prefer that local file.
