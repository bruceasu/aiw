# Design

## Module

`no-network-runner-windows` is one deep Module. Its Interface is deliberately
small: accept one direct argv plus bounded execution settings, then either
return the target result or fail before target creation.

The caller must not learn AppContainer profile creation, sandbox identity,
filesystem grants, process-tree termination, or Windows error translation.
That hidden complexity is the Module's depth and gives AIW one stable seam.

```text
AIW frozen Verification Plan
            |
            v
aiw-no-network-runner -- argv
            |
            v
Windows AppContainer Adapter
  - sandbox identity
  - no network capabilities
  - read-only / writable paths
  - child process + timeout cleanup
            |
            v
direct target process
```

## Recommended Adapter: Docker Linux Container

For the long-term profile, the Windows host launches a prebuilt Linux test
image through Docker Desktop. Docker's `none` network driver creates only a
loopback device, so the container has no connection to the host or external
containers. The runner uses a read-only root filesystem and a read-only bind
mount for the worktree; per-run cache and temporary output use tmpfs.

Illustrative generated argv (the runner, not AIW, creates it):

```text
docker run --rm --pull never --network none --read-only \
  --cap-drop ALL --security-opt no-new-privileges \
  --pids-limit 256 --memory 2g \
  --mount type=bind,src=<canonical-worktree>,dst=/workspace,readonly \
  --tmpfs /tmp:rw,noexec,nosuid,size=1g \
  -e GOCACHE=/tmp/go-cache -e GOTMPDIR=/tmp/go-tmp \
  -w /workspace <pinned-image@sha256:...> -- <approved argv>
```

The image must be prepared outside supervision and contain every compiler,
runtime, and dependency needed by the approved test. The runner checks Docker
availability and image presence before it starts a container. A missing daemon
or image returns `sandbox-unavailable` and AIW opens a Gate.

This Adapter is not allowed to mount `/var/run/docker.sock`, a Windows Docker
named pipe, host devices, or any writable host directory except the dedicated
runner temporary directory. Giving a target container Docker daemon access
would undermine the sandbox.

Relevant Docker references:

- [None network driver](https://docs.docker.com/engine/network/drivers/none/)
- [docker run options](https://docs.docker.com/reference/cli/docker/container/run/)
- [Read-only bind mounts](https://docs.docker.com/engine/storage/bind-mounts/)

## Alternative Adapter: Windows 11 AppContainer

The proposed Adapter uses the Windows 11 experimental
`Experimental_CreateProcessInSandbox` API with:

- `app_container = true`;
- no `internetClient`, `internetClientServer`, or
  `privateNetworkClientServer` capability;
- an identity unique to one execution;
- read-only grants for the worktree and language runtime;
- read/write grants only for the per-run temporary directory;
- an optional Job Object timeout/kill-on-close policy for the launched process
  tree.

Microsoft documents AppContainer as default-deny for network and filesystem
access unless permissions are explicitly granted. The API is Windows 11 only,
experimental, dynamically loaded from `processmodel.dll`, and requires a
FlatBuffer (`SBOX`) sandbox specification. See:

- [Create Process in Sandbox](https://learn.microsoft.com/en-us/windows/win32/secauthz/createprocessinsandbox)
- [Windows Filtering Platform overview](https://learn.microsoft.com/en-us/windows/win32/fwp/about-windows-filtering-platform)

## Docker Execution Algorithm

1. Validate direct argv and resolve the canonical worktree.
2. Verify the Docker daemon is reachable and the pinned image is already
   local. Do not pull.
3. Create a runner-owned temporary directory, construct the fixed Docker argv,
   and start the container with `--network none`.
4. Stream bounded output, enforce timeout, and force-remove the container on
   interruption or timeout.
5. Emit the redacted result; delete only the runner-owned temporary directory.

## Windows AppContainer Execution Algorithm

1. Parse and validate argv without a shell.
2. Resolve the target executable before sandboxing; reject missing targets.
3. Create a new temporary directory and generate a unique sandbox identity.
4. Build a sandbox specification with AppContainer enabled and no network
   capabilities.
5. Grant only the required filesystem paths.
6. Start the target with the sandbox API. If this fails, remove temporary state
   and return a `sandbox-unavailable` error; do not launch the target normally.
7. Collect bounded stdout/stderr, wait for exit or timeout, and terminate the
   process tree on timeout.
8. Emit a redacted structured result and clean the temporary directory.

## Internal Seams

Internal seams are test-only and are not part of the executable Interface:

- `SandboxLauncher`: creates a sandboxed process from a validated request.
- `DockerLauncher`: creates a fixed, no-network container from a validated
  request.
- `Clock`: supplies timeout timestamps.
- `ProcessWaiter`: waits, collects output, and terminates on timeout.
- `FilesystemGrantPlanner`: produces canonical allowed paths.

Production selects exactly one Adapter per profile: `DockerLauncher` for the
Docker profile or `AppContainerSandboxLauncher` for the Windows-native profile.
A fake launcher can assert ordering in tests: validation -> sandbox creation ->
target launch.

## Result Contract

The runner uses stable exit classifications in addition to the target exit
code:

| Condition | Result classification | Target started |
| --- | --- | --- |
| Invalid input | `invalid-request` | No |
| Unsupported Windows/API | `sandbox-unavailable` | No |
| Sandbox setup failure | `sandbox-unavailable` | No |
| Timeout | `timed-out` | Yes |
| Target exits | `target-exited` | Yes |

AIW maps `sandbox-unavailable` to its existing dependency/enforcer Gate. It
must never execute the original target as a fallback.

## Verification Plan

Implementation is not accepted until these local checks exist:

1. argv parsing rejects shell forms and missing `--`.
2. sandbox creation failure proves the target launcher was not invoked.
3. a local target can read the granted worktree and write only the temporary
   directory.
4. a target that attempts DNS resolution and a TCP connection fails inside the
   sandbox.
5. timeout kills the target and its child process tree.
6. no permanent Windows Firewall/WFP rules remain before or after execution.
7. the Docker Adapter refuses an absent image instead of pulling it, has no
   network interface other than loopback, and has no Docker socket mount.
