package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"diting/backend/internal/config"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func DSN(cfg config.PostgresConfig) string {
	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	return fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Database, cfg.Username, cfg.Password, sslMode)
}

func Connect(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, DSN(cfg))
}

func ExecuteBootstrap(ctx context.Context, pool Execer) error {
	return ExecuteSQL(ctx, pool, bootstrapSQL+"\n"+defaultProcessChainRiskRulesSQL)
}

func MigrationFiles(dir string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func ExecuteMigrations(ctx context.Context, pool Execer, dir string) error {
	files, err := MigrationFiles(dir)
	if err != nil {
		return err
	}
	for _, path := range files {
		if err := ExecuteMigrationFile(ctx, pool, path); err != nil {
			return fmt.Errorf("execute postgres migration %s: %w", path, err)
		}
	}
	return nil
}

func ExecuteMigrationFile(ctx context.Context, pool Execer, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return ExecuteSQL(ctx, pool, string(data))
}

func ExecuteSQL(ctx context.Context, pool Execer, sql string) error {
	for _, statement := range splitStatements(sql) {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := pool.Exec(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func splitStatements(sql string) []string {
	parts := strings.Split(sql, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			statements = append(statements, trimmed)
		}
	}
	return statements
}

const bootstrapSQL = `
CREATE TABLE IF NOT EXISTS diting_users (
    id UUID PRIMARY KEY,
    username VARCHAR(64) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(128) NOT NULL,
    email VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS diting_roles (
    id UUID PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS diting_user_roles (
    user_id UUID NOT NULL REFERENCES diting_users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES diting_roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS diting_audit_rules (
    id UUID PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    event_type VARCHAR(64) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    severity VARCHAR(32) NOT NULL,
    risk_score INTEGER NOT NULL,
    match_expr JSONB NOT NULL,
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_by UUID REFERENCES diting_users(id),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_diting_audit_rules_event_type ON diting_audit_rules(event_type);
CREATE INDEX IF NOT EXISTS idx_diting_audit_rules_enabled ON diting_audit_rules(enabled);
CREATE UNIQUE INDEX IF NOT EXISTS idx_diting_audit_rules_name_unique ON diting_audit_rules(name);

CREATE TABLE IF NOT EXISTS diting_system_configs (
    key VARCHAR(128) PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS diting_operation_logs (
    id UUID PRIMARY KEY,
    user_id UUID,
    username VARCHAR(64) NOT NULL,
    method VARCHAR(16) NOT NULL,
    path VARCHAR(512) NOT NULL,
    status INTEGER NOT NULL,
    ip VARCHAR(128) NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_diting_operation_logs_created_at ON diting_operation_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_diting_operation_logs_username ON diting_operation_logs(username);

CREATE TABLE IF NOT EXISTS diting_host_assets (
    id UUID PRIMARY KEY,
    host_id VARCHAR(255) NOT NULL DEFAULT '',
    host_name VARCHAR(255) NOT NULL DEFAULT '',
    node_name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    host_ip VARCHAR(128) NOT NULL DEFAULT '',
    environment VARCHAR(64) NOT NULL DEFAULT '',
    owner VARCHAR(128) NOT NULL DEFAULT '',
    department VARCHAR(128) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE diting_host_assets
    ADD COLUMN IF NOT EXISTS host_id VARCHAR(255) NOT NULL DEFAULT '';

ALTER TABLE diting_host_assets
    ADD COLUMN IF NOT EXISTS host_name VARCHAR(255) NOT NULL DEFAULT '';

ALTER TABLE diting_host_assets
    ADD COLUMN IF NOT EXISTS department VARCHAR(128) NOT NULL DEFAULT '';

UPDATE diting_host_assets
SET host_id = node_name
WHERE host_id = '';

UPDATE diting_host_assets
SET host_name = display_name
WHERE host_name = '';

CREATE INDEX IF NOT EXISTS idx_diting_host_assets_display_name ON diting_host_assets(display_name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_diting_host_assets_host_id_unique ON diting_host_assets(host_id);
CREATE INDEX IF NOT EXISTS idx_diting_host_assets_host_name ON diting_host_assets(host_name);

CREATE TABLE IF NOT EXISTS diting_risk_dispositions (
    event_id VARCHAR(128) PRIMARY KEY,
    status VARCHAR(32) NOT NULL DEFAULT 'open',
    note TEXT NOT NULL DEFAULT '',
    handled_by VARCHAR(128) NOT NULL DEFAULT '',
    handled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_diting_risk_dispositions_status ON diting_risk_dispositions(status);
CREATE INDEX IF NOT EXISTS idx_diting_risk_dispositions_updated_at ON diting_risk_dispositions(updated_at);

CREATE TABLE IF NOT EXISTS diting_collector_heartbeats (
    host_id VARCHAR(128) PRIMARY KEY,
    host_name VARCHAR(128) NOT NULL DEFAULT '',
    input_mode VARCHAR(32) NOT NULL DEFAULT 'file',
    last_error TEXT NOT NULL DEFAULT '',
    last_seen_at TIMESTAMPTZ NOT NULL,
    last_event_time TIMESTAMPTZ,
    last_write_at TIMESTAMPTZ,
    events_written BIGINT NOT NULL DEFAULT 0,
    buffered_events BIGINT NOT NULL DEFAULT 0,
    dropped_events BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE diting_collector_heartbeats
    ADD COLUMN IF NOT EXISTS input_mode VARCHAR(32) NOT NULL DEFAULT 'file';

ALTER TABLE diting_collector_heartbeats
    ADD COLUMN IF NOT EXISTS last_error TEXT NOT NULL DEFAULT '';

ALTER TABLE diting_collector_heartbeats
    ADD COLUMN IF NOT EXISTS buffered_events BIGINT NOT NULL DEFAULT 0;

ALTER TABLE diting_collector_heartbeats
    ADD COLUMN IF NOT EXISTS dropped_events BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_diting_collector_heartbeats_last_seen_at ON diting_collector_heartbeats(last_seen_at);

INSERT INTO diting_roles (id, name, description, created_at, updated_at)
VALUES (gen_random_uuid(), 'admin', 'System administrator', NOW(), NOW())
ON CONFLICT (name) DO NOTHING;

INSERT INTO diting_users (id, username, password_hash, display_name, email, status, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    'admin',
    'sha256$diting-admin$fdb286ed57f54bc847d9b5bd1eadd595ac513cf95917765e06de8eebae081ee6',
    'Administrator',
    '',
    'active',
    NOW(),
    NOW()
)
ON CONFLICT (username) DO NOTHING;

INSERT INTO diting_user_roles (user_id, role_id)
SELECT u.id, r.id
FROM diting_users u, diting_roles r
WHERE u.username = 'admin' AND r.name = 'admin'
ON CONFLICT DO NOTHING;

INSERT INTO diting_audit_rules (id, name, description, event_type, enabled, severity, risk_score, match_expr, tags, created_at, updated_at)
VALUES
(
    '00000000-0000-0000-0000-000000000101',
    '反弹 Shell 命令',
    '检测 bash -i、nc -e、/dev/tcp 等常见反弹 Shell 行为。',
    'process_exec',
    true,
    'critical',
    95,
    '{"operator":"or","conditions":[{"field":"cmdline","op":"contains","value":"bash -i"},{"field":"cmdline","op":"contains","value":"nc -e"},{"field":"cmdline","op":"contains","value":"/dev/tcp/"},{"field":"cmdline","op":"contains","value":"python -c"},{"field":"cmdline","op":"contains","value":"perl -e"}]}'::jsonb,
    '["reverse-shell","critical-command"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000102',
    '下载并执行脚本',
    '检测 curl、wget 下载后通过 sh/bash 执行远程脚本的行为。',
    'process_exec',
    true,
    'critical',
    90,
    '{"operator":"or","conditions":[{"field":"cmdline","op":"contains","value":"curl "},{"field":"cmdline","op":"contains","value":"wget "},{"field":"cmdline","op":"contains","value":"| sh"},{"field":"cmdline","op":"contains","value":"| bash"},{"field":"cmdline","op":"contains","value":"curl -fsSL"},{"field":"cmdline","op":"contains","value":"wget -qO-"}]}'::jsonb,
    '["download-exec","critical-command"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000103',
    '权限切换命令',
    '检测 sudo、su、passwd 等权限切换或账号相关命令。',
    'process_exec',
    true,
    'high',
    75,
    '{"operator":"or","conditions":[{"field":"process_name","op":"in","values":["sudo","su","passwd"]},{"field":"cmdline","op":"contains","value":"sudo "},{"field":"cmdline","op":"contains","value":"su -"}]}'::jsonb,
    '["privilege","account"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000104',
    '敏感文件访问',
    '检测访问 /etc/shadow、/etc/passwd、SSH 私钥或 authorized_keys 等敏感文件的命令。',
    'process_exec',
    true,
    'high',
    80,
    '{"operator":"or","conditions":[{"field":"cmdline","op":"contains","value":"/etc/shadow"},{"field":"cmdline","op":"contains","value":"/etc/passwd"},{"field":"cmdline","op":"contains","value":"id_rsa"},{"field":"cmdline","op":"contains","value":"authorized_keys"}]}'::jsonb,
    '["sensitive-file","credential"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000105',
    '危险权限变更',
    '检测 chmod 777、递归放宽权限或修改 root 属主等高风险权限变更。',
    'process_exec',
    true,
    'high',
    70,
    '{"operator":"or","conditions":[{"field":"cmdline","op":"contains","value":"chmod 777"},{"field":"cmdline","op":"contains","value":"chmod -R 777"},{"field":"cmdline","op":"contains","value":"chown root"}]}'::jsonb,
    '["permission","hardening"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000106',
    '容器控制命令',
    '检测 docker、kubectl、ctr、crictl 等容器管理命令。',
    'process_exec',
    true,
    'high',
    65,
    '{"operator":"or","conditions":[{"field":"process_name","op":"in","values":["docker","kubectl","ctr","crictl"]},{"field":"cmdline","op":"contains","value":"docker "},{"field":"cmdline","op":"contains","value":"kubectl "}]}'::jsonb,
    '["container","admin-command"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000201',
    '高危端口网络连接',
    '检测连接常见反连、代理、远控或非常规服务端口的网络行为。',
    'network_connect',
    true,
    'high',
    80,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"network_connect"},{"field":"protocol","op":"eq","value":"tcp"},{"field":"dst_port","op":"in","values":["4444","5555","6666","7777","8888","9999","31337"]}]}'::jsonb,
    '["network","suspicious-port"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000202',
    '命令解释器发起网络连接',
    '检测 bash、sh、python、perl、php 等解释器进程直接发起网络连接。',
    'network_connect',
    true,
    'high',
    85,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"network_connect"},{"field":"process_name","op":"in","values":["bash","sh","dash","zsh","python","python3","perl","php","ruby","node"]}]}'::jsonb,
    '["network","interpreter"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000203',
    '下载工具发起网络连接',
    '检测 curl、wget、nc、ncat、telnet 等工具发起网络连接。',
    'network_connect',
    true,
    'medium',
    60,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"network_connect"},{"field":"process_name","op":"in","values":["curl","wget","nc","ncat","telnet","socat"]}]}'::jsonb,
    '["network","tool"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000301',
    '敏感文件探针访问',
    '检测 Tetragon 文件访问事件中读取或打开 /etc/passwd、/etc/shadow、sudoers、SSH 配置与密钥等敏感路径。',
    'file_access',
    true,
    'high',
    80,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"file_access"},{"field":"file_path","op":"in","values":["/etc/passwd","/etc/shadow","/etc/sudoers","/etc/group","/etc/gshadow","/etc/ssh/sshd_config"]}]}'::jsonb,
    '["file-access","sensitive-file"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000302',
    'SSH 敏感目录访问',
    '检测 Tetragon 文件访问事件中访问 SSH 配置、密钥或授权文件路径。',
    'file_access',
    true,
    'high',
    82,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"file_access"},{"field":"file_path","op":"regex","value":"(^/etc/ssh/|^/root/\\.ssh/|^/home/[^/]+/\\.ssh/)"}]}'::jsonb,
    '["file-access","ssh","credential"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000303',
    '敏感文件写入',
    '检测 Tetragon 文件事件中对账号、sudo、SSH、计划任务等敏感路径的写入、创建或截断行为。',
    'file_access',
    true,
    'critical',
    92,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"file_access"},{"field":"file_path","op":"regex","value":"(^/etc/(passwd|shadow|group|gshadow|sudoers|sudoers\\.d/|ssh/|crontab)|^/var/spool/cron/|^/root/\\.ssh/|^/home/[^/]+/\\.ssh/)"},{"field":"file_operation","op":"regex","value":"(?i)(write|truncate|create|creat|open.*wronly|open.*rdwr|security_file_permission|security_file_open)"}]}'::jsonb,
    '["file-access","sensitive-file","write"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000304',
    '敏感文件权限变更',
    '检测 Tetragon 文件事件中对敏感路径的 chmod、chown、扩展属性等权限或属主变更行为。',
    'file_access',
    true,
    'critical',
    90,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"file_access"},{"field":"file_path","op":"regex","value":"(^/etc/(passwd|shadow|group|gshadow|sudoers|sudoers\\.d/|ssh/|crontab)|^/var/spool/cron/|^/root/\\.ssh/|^/home/[^/]+/\\.ssh/)"},{"field":"file_operation","op":"regex","value":"(?i)(chmod|chown|fchmod|fchown|setxattr|removexattr|security_inode_setattr)"}]}'::jsonb,
    '["file-access","sensitive-file","permission"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000305',
    '敏感文件删除',
    '检测 Tetragon 文件事件中删除账号、sudo、SSH、计划任务等敏感路径的行为。',
    'file_access',
    true,
    'critical',
    94,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"file_access"},{"field":"file_path","op":"regex","value":"(^/etc/(passwd|shadow|group|gshadow|sudoers|sudoers\\.d/|ssh/|crontab)|^/var/spool/cron/|^/root/\\.ssh/|^/home/[^/]+/\\.ssh/)"},{"field":"file_operation","op":"regex","value":"(?i)(unlink|unlinkat|rmdir|security_inode_unlink|security_inode_rmdir)"}]}'::jsonb,
    '["file-access","sensitive-file","delete"]'::jsonb,
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;
`

const defaultProcessChainRiskRulesSQL = `
INSERT INTO diting_audit_rules (id, name, description, event_type, enabled, severity, risk_score, match_expr, tags, created_at, updated_at)
VALUES
(
    '00000000-0000-0000-0000-000000000401',
    'Shell 下载工具外联链路',
    '检测 bash/sh 等 Shell 父进程拉起 curl、wget、nc、ncat、socat、telnet 等工具发起网络连接。',
    'network_connect',
    true,
    'high',
    85,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"network_connect"},{"field":"parent_process_name","op":"in","values":["bash","sh","dash","zsh"]},{"field":"process_name","op":"in","values":["curl","wget","nc","ncat","socat","telnet"]}]}'::jsonb,
    '["process-chain","network","download-tool"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000402',
    'Web 服务拉起 Shell',
    '检测 nginx、apache、httpd 等 Web 服务父进程拉起 sh、bash、dash、zsh 的可疑链路。',
    'process_exec',
    true,
    'high',
    88,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"process_exec"},{"field":"parent_process_name","op":"in","values":["nginx","apache","apache2","httpd"]},{"field":"process_name","op":"in","values":["sh","bash","dash","zsh"]}]}'::jsonb,
    '["process-chain","webshell"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000403',
    'Shell 拉起解释器外联',
    '检测 bash/sh 等 Shell 父进程拉起 python、perl、php、ruby、node 等解释器发起网络连接。',
    'network_connect',
    true,
    'high',
    82,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"network_connect"},{"field":"parent_process_name","op":"in","values":["bash","sh","dash","zsh"]},{"field":"process_name","op":"in","values":["python","python3","perl","php","ruby","node"]}]}'::jsonb,
    '["process-chain","network","interpreter"]'::jsonb,
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;
`
