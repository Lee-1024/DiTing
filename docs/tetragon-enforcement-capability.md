# Tetragon Enforcement Capability Matrix

DiTing uses Tetragon as its runtime observability and enforcement layer. Policy templates must map to official Tetragon 1.7 capabilities and must not promise behavior that depends on unverified command-line patterns.

## Runtime Prerequisites

- Tetragon must run privileged or with the required BPF permissions.
- `/sys/fs/bpf` must be mounted and writable by Tetragon.
- BTF must be available through `/sys/kernel/btf/vmlinux` or a compatible fallback.
- Sudo child tracking uses Tetragon `matchBinaries` with `followChildren: true`. DiTing should not require `--parents-map-enabled=true` by default because that flag only serves `matchParentBinaries` and can fail on some kernels when the loaded BPF object lacks `PARENTS_MAP_ENABLED`.
- `Override` requires kernel support for BPF override. For `security_` hooks, Linux 5.7+ is required by Tetragon documentation.

## Template Capability Mapping

| DiTing Template | Strong Enforcement Hooks | Primary Use | Boundary |
|---|---|---|---|
| Dangerous command | `sys_execve` | Block direct execution of known dangerous binaries | Complex shell semantics are not guaranteed by argv matching |
| Sensitive file | `security_file_permission` | Block reading or writing sensitive files by process/user/sudo-child context | Permission is a bitmask; use `Mask` for `MAY_READ=4` and `MAY_WRITE=2` |
| Delete protection | `security_path_unlink`, `security_path_rmdir` | Block deletion of files/directories by path | Preferred over `rm` command matching |
| Permission change | `sys_chmod`, `sys_fchmodat`, `sys_chown`, `sys_fchownat`, path/security hooks when available | Block chmod/chown on protected paths | Path resolution differs by syscall/hook |
| Sudo child tracking | `matchBinaries` on `/usr/bin/sudo` and `/bin/sudo` with `followChildren: true` | Distinguish sudo-root process chains from direct root sessions | Requires policy to be loaded before the sudo child process starts |

## Tetragon 1.7 Runtime Parameters

Use this table instead of pasting the full `tetragon --help` output into deployment docs.

| Flag | Recommendation | Why |
|---|---|---|
| `--export-filename=/data/tetragon/logs/tetragon.log` | Required for file collector mode | DiTing tails this JSON export file when `collector.input_mode=file` |
| `--server-address=0.0.0.0:54321` | Required for gRPC collector mode | DiTing can consume Tetragon events through gRPC |
| `--tracing-policy-dir=/etc/tetragon/tetragon.tp.d` | Required for policy sync | Collector writes generated policies into this directory |
| `--enable-process-cred` | Required | Adds UID/EUID/GID/EGID credential fields used by DiTing identity and root-exclusion logic |
| `--enable-process-ns` | Recommended | Adds namespace context to process and kprobe events, useful for host/container separation |
| `--enable-ancestors=base,kprobe,tracepoint,lsm` | Recommended | Exports ancestor context on event records; `base` is required by other event types for correct reference counting |
| `--username-metadata=unix` | Optional | Lets Tetragon resolve host UIDs to usernames; DiTing can also resolve through `/etc/passwd`, so this is not required |
| `--export-file-max-size-mb=100` and `--export-file-max-backups=10` | Recommended for production | Default rotation is small for busy hosts; increase retention to avoid losing audit data between collector outages |
| `--export-file-perm=640` | Optional | Use when the collector runs outside the Tetragon container and needs group read access |
| `--event-queue-size`, `--rb-size`, `--rb-size-total` | Tune only after observing drops | Increase on busy nodes if health/debug pages show event loss or ring-buffer pressure |
| `--process-cache-size` | Tune only for high process churn | Increase if process metadata is missing under heavy fork/exec workloads |
| `--parents-map-enabled` | Do not use as DiTing default | Officially required only for `matchParentBinaries`; it caused `PARENTS_MAP_ENABLED` rodata failures on some 5.4 nodes |
| `--enable-process-environment-variables` | Avoid by default | Can leak secrets and significantly increase event size |
| `--enable-policy-filter*` | Do not enable by default | Kubernetes/cgroup policy filtering is outside DiTing's current host-policy model |
| `--debug`, `--verbose`, `--pprof-address`, `--gops-address` | Debug only | Useful for short troubleshooting windows, not normal production runtime |

Recommended Docker command for DiTing host enforcement:

```yaml
command:
  - /usr/bin/tetragon
  - --server-address
  - 0.0.0.0:54321
  - --export-filename
  - /data/tetragon/logs/tetragon.log
  - --tracing-policy-dir
  - /etc/tetragon/tetragon.tp.d
  - --enable-process-cred
  - --enable-process-ns
  - --enable-ancestors=base,kprobe,tracepoint,lsm
  - --export-file-max-size-mb=100
  - --export-file-max-backups=10
```

## Enforcement Actions

- `Sigkill` kills the current process and is useful as a stop signal.
- `Override` should be preferred where supported because it returns an error from the probed hook/syscall before the operation continues.
- DiTing strong-enforcement templates should emit `Override` when supported and `Sigkill` as a fallback action.

## Product Rules

- Do not implement strong enforcement by matching `sudo` command arguments.
- Do not label a template strong enforcement unless it targets the actual kernel behavior being protected.
- All enforce-mode policies must include `diting-enforcement` and `diting-blocked-command` tags for notification and audit correlation.
- Every generated template must have a documented manual verification command.

## Manual Verification Matrix

Run these checks on every target OS/kernel/Tetragon version before marking a host as enforcement-ready.

| Scenario | Command | Expected Result |
|---|---|---|
| Tetragon bpffs mount | `mountpoint -q /sys/fs/bpf` | bpffs is mounted and `/sys/fs/bpf/tetragon` can be created by Tetragon |
| Ancestor export enabled | Check container command | Tetragon was started with `--enable-ancestors=base,kprobe,tracepoint,lsm` |
| Direct root excluded | Log in as root, then run a protected root-excluded action | Allowed when no sudo-child selector matches |
| Sudo child blocked | Ordinary user runs `sudo vim /etc/docker/daemon.json` against a protected file policy | The protected hook event matches sudo `matchBinaries` with `followChildren: true` and is blocked |
| Sensitive file mask | Open protected file with `vim` | Event permission value matches `Mask` against `MAY_READ=4` or `MAY_WRITE=2`, including combined value `6` |
| Delete file blocked | `rm /home/ubuntu/test` against a delete protection policy | Event function is `security_path_unlink`, action includes `Override` and `Sigkill`, file remains |
| Delete directory blocked | `rmdir /home/ubuntu/testdir` against a delete protection policy | Event function is `security_path_rmdir`, directory remains |
| Alert surfaced | Trigger any enforce policy | Header notification shows user, command, and target path when present |
| Rule persisted | Trigger any enforce policy | Risk/rule data includes `diting-enforcement` and critical severity |

If sudo child tracking does not match, verify the policy was loaded before running sudo, then verify the sudo binary path (`/usr/bin/sudo` or `/bin/sudo`) before changing policy semantics.
