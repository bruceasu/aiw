# ANGETS.md

This is a legacy typo copy of `AGENTS.md`.
Use `AGENTS.md` in this directory first.

## Validation Options
Use static review by default. The following commands are options only when the
shared resource budget authorizes runtime validation:

- `go test ./path/to/package`
- `go vet ./path/to/package`
- `go build ./path/to/package`

Run one focused command and ask before widening scope.
