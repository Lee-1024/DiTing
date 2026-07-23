CREATE TABLE IF NOT EXISTS diting_enforcement_policies (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    template TEXT NOT NULL,
    mode TEXT NOT NULL DEFAULT 'audit',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    target_hosts TEXT[] NOT NULL DEFAULT '{}',
    definition JSONB NOT NULL DEFAULT '{}'::jsonb,
    yaml TEXT NOT NULL,
    deployment_status TEXT NOT NULL DEFAULT 'draft',
    deployment_message TEXT NOT NULL DEFAULT '',
    deployed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT diting_enforcement_policies_mode_check CHECK (mode IN ('audit', 'enforce', 'disabled')),
    CONSTRAINT diting_enforcement_policies_deployment_status_check CHECK (deployment_status IN ('draft', 'deployed', 'failed', 'disabled'))
);

CREATE INDEX IF NOT EXISTS idx_diting_enforcement_policies_updated_at
    ON diting_enforcement_policies (updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_diting_enforcement_policies_deployment_status
    ON diting_enforcement_policies (deployment_status);

ALTER TABLE diting_enforcement_policies
    ADD COLUMN IF NOT EXISTS definition JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE TABLE IF NOT EXISTS diting_enforcement_policy_deployments (
    id UUID PRIMARY KEY,
    policy_id UUID NOT NULL REFERENCES diting_enforcement_policies(id) ON DELETE CASCADE,
    host_id TEXT NOT NULL,
    host_name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft',
    message TEXT NOT NULL DEFAULT '',
    deployed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT diting_enforcement_policy_deployments_status_check CHECK (status IN ('draft', 'deployed', 'failed', 'disabled')),
    CONSTRAINT diting_enforcement_policy_deployments_unique_host UNIQUE (policy_id, host_id)
);

CREATE INDEX IF NOT EXISTS idx_diting_enforcement_policy_deployments_policy
    ON diting_enforcement_policy_deployments (policy_id);

CREATE INDEX IF NOT EXISTS idx_diting_enforcement_policy_deployments_status
    ON diting_enforcement_policy_deployments (status);
