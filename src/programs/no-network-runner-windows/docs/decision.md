# Feasibility Decision

## Answer

There is no simple, broadly compatible, trustworthy Windows implementation
that satisfies all requirements.

The simplest credible long-term path is a Docker Linux-container Adapter with
a prebuilt local test image. It provides one portable isolation model on the
Windows host, avoids global firewall policy changes, and can use Docker's
`--network none` network namespace plus read-only mounts.

The runner must still be a small native host program or carefully constrained
host launcher: it owns fixed Docker argv generation, pinned-image checks,
timeout cleanup, and structured results. It must not accept arbitrary Docker
options from AIW or an Agent.

Windows 11 AppContainer remains the native alternative when Docker Desktop is
not an acceptable dependency. It is not the preferred first implementation
because its sandbox process API is experimental and requires FlatBuffer
specification construction.

## Rejected Shortcuts

| Shortcut | Why it is rejected |
| --- | --- |
| Environment variable such as `NO_NETWORK=1` | Advisory only; the target can ignore it. |
| PowerShell or batch wrapper | Cannot technically prevent Winsock/network use. |
| Temporary Windows Firewall rule | Changes global policy, may require elevation, and cleanup failure can affect the user. |
| Direct WFP filter manipulation | WFP supports application filtering, but its configuration is system policy; the documented BFE security model grants filter configuration to administrators. |
| Disable adapter / proxy / DNS | Global side effect and not a complete network block. |
| Run bare `go test` | Violates the no-network requirement. |
| Docker with a floating image tag or automatic pull | Lets supervision obtain unreviewed code or access the registry. |
| Docker with socket/pipe mount or `--privileged` | Lets the target control the daemon or escape the intended restriction. |

## Hang Conditions

Leave this program and AIW focused-test execution gated when any condition is
true:

- The host is not Windows 11 or does not expose the sandbox API.
- The AppContainer experiment cannot start a typical local Go unit test using
  only the planned filesystem grants.
- The program cannot prove that DNS and TCP attempts fail while the test runs.
- The required Windows SDK/API contract is unavailable without adding an
  unapproved dependency.
- A supported test needs writes outside the runner temporary directory or
  needs network/dependency download.

## Implementation Decision Needed Later

Before code is written, choose one of these explicit scopes:

1. **Windows 11 experimental MVP:** native executable, fail closed on
   unsupported hosts.
2. **Docker MVP (recommended for the product):** require Docker Desktop and a
   local pinned test image; run with `--network none`, no pull, read-only
   worktree, tmpfs cache, and no daemon socket mount.
3. **Cross-version Windows without Docker:** requires a different approved
   isolation provider or administrator-managed infrastructure. Do not emulate
   it with firewall scripts.
4. **Gate-controlled degraded execution:** retain the Gate by default. After
   an operator explicitly waives it for an already-authorized Verification
   Plan, AIW may run only the frozen command without technical network
   isolation and must record that the boundary was waived. This is not an
   isolation provider and does not replace the runner.

No implementation decision is made by these documents.
