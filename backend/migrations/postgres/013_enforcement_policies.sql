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
