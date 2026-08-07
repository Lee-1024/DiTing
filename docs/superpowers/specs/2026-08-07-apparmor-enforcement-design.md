# AppArmor Enforcement Design

## Goal

Replace DiTing's unverified Tetragon enforcement templates with an AppArmor-backed sensitive-file enforcement path for Ubuntu 20.04 hosts that cannot upgrade from Linux 5.4.

Tetragon remains the audit and observability source. AppArmor becomes the enforcement engine.

## Runtime Model

The existing `audit-server collector` process is started manually as root. The first release adds AppArmor management directly to that process. It does not add another daemon, socket, or systemd dependency.

At startup and before each policy synchronization, the collector checks:

- effective UID is root;
- `/sys/module/apparmor/parameters/enabled` reports `Y`;
- `apparmor_parser` is discoverable with `exec.LookPath`;
- the configured policy directory is writable.

Failure disables only enforcement synchronization. Tetragon event collection continues.

## Supported Policy

The first release supports only enabled `sensitive_file` policies in `enforce` mode. Other enforcement templates are reported as unsupported and are never represented as successfully deployed.

The API already persists the structured `definition` JSON. The collector consumes `filePaths` from that definition instead of parsing generated YAML.

AppArmor protects the configured paths from processes executed through `/usr/bin/sudo` or `/bin/sudo`. The sudo profile and its children inherit the restriction. A directly logged-in root process remains unconfined and is not affected.

## Profile Management

Generated profiles are stored below the configured collector enforcement directory. The collector:

1. validates and normalizes absolute protected paths;
2. writes a temporary profile;
3. runs `apparmor_parser -Q` without a shell;
4. replaces the loaded profile with `apparmor_parser -r`;
5. atomically promotes the temporary file;
6. restores and reloads the previous profile when activation fails.

The generated profile is deterministic so unchanged policy sets do not reload AppArmor.

## Deployment Semantics

Deployment status is reported per policy and host:

- `deployed`: supported policy is included in the profile and the profile loaded successfully;
- `failed`: definition or AppArmor activation failed;
- `disabled`: policy is disabled or removed;
- `unsupported`: reserved for capability and template incompatibility in the next schema revision; the first release reports these as `failed` with an explicit unsupported message to remain compatible with the current database constraint.

No restart of Tetragon or the host is performed.

## Alerts

AppArmor denial records are authoritative enforcement evidence. The first release prepares policy tags and deployment correlation; ingestion of kernel/audit AppArmor denial records into the existing notification pipeline is a separate implementation task in the same workstream and must be verified against the target host's active audit transport.

## Safety

- No `sh -c` is used for AppArmor operations.
- Only absolute paths are accepted.
- Newlines, NUL bytes, AppArmor variables, aliases, and unsupported glob syntax are rejected.
- The last valid profile remains loaded if synchronization fails.
- Emergency disable unloads the managed profile only after the desired policy set is empty.

