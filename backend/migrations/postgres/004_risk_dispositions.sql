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
