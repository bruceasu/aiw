# Java Spring

## Inspect First
- controllers or handlers
- services or domain components
- repositories
- DTOs and validation rules
- config and tests

## Keep Stable
- controller, service, and repository boundaries
- request and response contracts
- transaction boundaries
- dependency injection style
- serialization and validation behavior

## Validate
- prefer the nearest verification script
- otherwise use project-standard Maven or Gradle checks
- add unit, slice, or integration tests that match the changed boundary
