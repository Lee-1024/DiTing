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
