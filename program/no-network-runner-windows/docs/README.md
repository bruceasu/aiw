# no-network-runner-windows

## Status

Design only. No executable, build configuration, or AIW integration is created
in this directory yet.

This program is the proposed Windows host Adapter for the external command
used by supervised local unit tests:

```text
aiw-no-network-runner -- <approved-command> <approved-args...>
```

It is not a convenience wrapper around `go test`. Its required property is
that the child process cannot access a network. If the host cannot establish
that property, the program must fail before starting the target command.

Read the documents in this order:

1. [requirements.md](requirements.md) — contract and security invariants.
2. [design.md](design.md) — proposed deep Module and its Docker and Windows
   Adapters.
3. [decision.md](decision.md) — feasibility decision and explicit hang points.
