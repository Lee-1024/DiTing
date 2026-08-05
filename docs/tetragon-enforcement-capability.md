# Tetragon Enforcement Capability Matrix

DiTing uses Tetragon as its runtime observability and enforcement layer. Policy templates must map to official Tetragon capabilities and must not promise behavior that depends on unverified command-line patterns.

## Runtime Prerequisites

- Tetragon must run privileged or with required BPF permissions.
- `/sys/fs/bpf` must be mounted and writable by Tetragon.
- BTF must be available through `/sys/kernel/btf/vmlinux` or a compatible fallback.
- Parent process matching requires Tetragon parent tracking. Deployments using `matchParentBinaries` with `followChildren: true` must start Tetragon with `--parents-map-enabled=true`.
- `Override` requires kernel support for BPF override. For `security_` hooks, Linux 5.7+ is required by Tetragon documentation.

## Template Capability Mapping

| DiTing Template | Strong Enforcement Hooks | Primary Use | Boundary |
|---|---|---|---|
| Dangerous command | `sys_execve` | Block direct execution of known dangerous binaries | Complex shell semantics are not guaranteed by argv matching |
| Sensitive file | `security_file_permission` | Block reading or writing sensitive files by process/user/ancestry context | Permission values use Linux MAY_READ/MAY_WRITE semantics |
| Delete protection | `security_path_unlink`, `security_path_rmdir` | Block deletion of files/directories by path | Preferred over `rm` command matching |
| Permission change | `sys_chmod`, `sys_fchmodat`, `sys_chown`, `sys_fchownat`, path/security hooks when available | Block chmod/chown on protected paths | Path resolution differs by syscall/hook |
| Sudo ancestry | `matchParentBinaries` with `followChildren: true` | Distinguish sudo-root process chains from direct root sessions | Requires parent tracking and verified binary paths |

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
| Parent map enabled | `docker exec tetragon /usr/bin/tetragon --help \| grep parents-map` and check container command | Tetragon was started with `--parents-map-enabled=true` |
| Direct root excluded | Log in as root, then run a protected root-excluded action | Allowed when no sudo ancestry selector matches |
| Sudo ancestry blocked | Ordinary user runs `sudo vim /etc/docker/daemon.json` against a protected file policy | The protected hook event matches `matchParentBinaries` and is blocked |
| Delete file blocked | `rm /home/ubuntu/test` against a delete protection policy | Event function is `security_path_unlink`, action includes `Override` and `Sigkill`, file remains |
| Delete directory blocked | `rmdir /home/ubuntu/testdir` against a delete protection policy | Event function is `security_path_rmdir`, directory remains |
| Alert surfaced | Trigger any enforce policy | Header notification shows user, command, and target path when present |
| Rule persisted | Trigger any enforce policy | Risk/rule data includes `diting-enforcement` and critical severity |

If sudo ancestry does not match, first verify Tetragon was restarted with `--parents-map-enabled=true`, then verify the sudo binary path (`/usr/bin/sudo` or `/bin/sudo`) before changing policy semantics.
