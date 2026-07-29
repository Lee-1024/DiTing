# Production Pre-Release Initialization

This runbook prepares a clean pre-release environment and reseeds production-grade collection and audit baselines.

## Cleanup Scope

Run:

```bash
scripts/clear-test-data-linux.sh --config backend/configs/config.yaml --yes
```

The command clears collected runtime data and previous collection/audit initialization:

- ClickHouse `audit_events`
- PostgreSQL `diting_risk_dispositions`
- PostgreSQL `diting_collector_heartbeats`
- PostgreSQL `diting_host_assets`
- PostgreSQL `diting_system_configs` key `collector_filter`
- PostgreSQL `diting_audit_rules`

It then seeds the production pre-release collector filter and audit rules from `ProductionPreReleaseBaselineSQL`.

It preserves users, roles, operation logs, and enforcement policies.

## Collector Filter Baseline

The pre-release reset command rewrites `collector_filter` during the rehearsal reset. Migration `014_prerelease_collector_filter.sql` still provides a first-install fallback when no config exists.

The baseline is designed for high-volume hosts:

- Always keep `high` and `critical` events after audit-rule enrichment.
- Drop routine low-risk `root` `process_exec`, `file_access`, and `network_connect` events before audit-rule enrichment, so broad rules cannot promote root noise into stored high-severity events.
- Let explicit root high-risk signals pass into rule enrichment: reverse shell patterns, download-to-shell patterns, sensitive-file mutation, dangerous permission changes, and suspicious outbound ports.
- Keep normal-user commands by default.
- Drop high-frequency low-value reads from `/proc`, `/sys`, and simple `/dev` pseudo files.
- Drop low-risk monitoring probe noise from common agents.

This makes root collection defensive rather than exhaustive, while ordinary users remain closely audited.

## Audit Rule Baseline

The reset command clears prior audit rules and writes production-prefixed rules:

- `生产-反弹 Shell 命令`
- `生产-下载后执行`
- `生产-提权与账号变更`
- `生产-危险权限变更`
- `生产-高危端口外联`
- `生产-解释器直接外联`
- `生产-敏感文件读取`
- `生产-敏感文件写入`
- `生产-敏感文件删除或权限变更`
- `生产-Web 服务拉起 Shell`
- `生产-Shell 拉起下载工具外联`
- `生产-Shell 拉起解释器外联`

Review these after the first day:

- Container control commands can be noisy on Kubernetes admin hosts.
- Download-tool network connections can be medium volume on package mirrors or build hosts.

## Enforcement Posture

Use `audit` mode during pre-release. Do not deploy default `enforce` policies until rule hits have been reviewed.

Recommended initial policy candidates:

- Reverse shell blocking.
- Sensitive file write/delete blocking.
- Web service spawning shell blocking.
- High-risk outbound port blocking.

Promotion rule: move a policy from `audit` to `enforce` only after the exact rule hit has been reviewed on the target host class and there is a rollback owner.

## First-Day Checks

After one full collection day, compare:

- Total events per host.
- Root versus non-root event ratio.
- Top process names and command lines.
- Filtered event count from collector heartbeat `dropped_events`.
- High and critical event count.

If one host still produces hundreds of thousands of stored events, add a narrow filter rule for the top low-risk source instead of broadening user-level drops.
