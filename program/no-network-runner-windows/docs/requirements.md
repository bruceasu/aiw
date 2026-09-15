# Requirements

## Goal

Provide a Windows executable named `aiw-no-network-runner` that runs one
already-approved local command with technically enforced network denial.

The caller is AIW Workflow Core. AIW supplies the frozen argv from its
Verification Plan; this program must not invent, expand, or shell-parse a
command.

## Command Interface

```text
aiw-no-network-runner [--workdir <absolute-path>] [--timeout-seconds <n>] -- <executable> <arg>...
```

Rules:

- `--` is mandatory and appears exactly once.
- The target is launched directly. `cmd.exe`, PowerShell, shell operators,
  response files controlled outside the plan, and string command parsing are
  out of scope and must be rejected by the caller or runner.
- `--workdir` must be an existing absolute directory.
- `--timeout-seconds` must be a positive bounded integer. A timeout terminates
  the sandbox process tree and returns a distinct non-zero exit code.
- The runner returns the target's exit code when the target starts and exits.
- The runner writes diagnostics to stderr and may emit one JSON result record
  to stdout only after the target exits.

## Security Invariants

1. The target process has no outbound or inbound network capability.
2. Network denial is established before the target process is created.
3. Failure to create the sandbox means the target is not started.
4. The runner does not alter Windows Firewall, WFP policy, registry policy,
   system proxy settings, or any other global machine/user network setting.
5. The runner requires no elevation. If the selected Windows isolation API
   rejects the current host, it exits non-zero.
6. The target receives only a minimal environment. `PATH` is permitted only
   when needed to resolve an already-approved executable; secrets and ambient
   AI provider variables are not inherited.
7. The source worktree is read-only. A separate per-run temporary directory is
   writable for language caches and test artifacts.
8. The runner records the effective sandbox identity, allowed filesystem paths,
   target argv, start/end times, timeout state, and exit code. It never logs
   secret environment values.

## Filesystem Contract

The first supported profile is local Go unit tests. It needs:

- read-only: worktree, Go installation/runtime files;
- read/write: a new runner-owned temporary directory for `GOCACHE`, `GOTMPDIR`,
  and test temporary files;
- no implicit write permission to the worktree.

A test that requires an external service, writes source fixtures, downloads a
module, or accesses a package registry is unsupported by this profile. AIW
must report it as a Gate instead of weakening the runner.

## Non-goals

- Cross-platform support in this program.
- Container, VM, WSL, Docker, or Hyper-V management.
- Permanent firewall/WFP rules or administrator elevation.
- Dependency installation, package download, integration tests, and test
  selection.
- A claim that environment variables alone enforce network denial.

## Docker Profile Requirements

When the selected Adapter is Docker, the runner additionally MUST:

- use a preinstalled Docker daemon and a locally present, pinned image digest;
- invoke `docker run --pull never`; a missing image is a Gate, never an image
  pull during `supervise`;
- use `--network none`, `--read-only`, `--cap-drop ALL`, and
  `--security-opt no-new-privileges`;
- mount the worktree read-only, and mount only a runner-created temporary
  directory or tmpfs as writable;
- never mount the Docker socket, named pipes, host devices, the host root, or
  any user home/configuration directory;
- set a bounded PID and memory limit; and
- remove the container after execution.

Docker image preparation is a separate, user-authorized setup operation. It is
not part of a supervised test run.
