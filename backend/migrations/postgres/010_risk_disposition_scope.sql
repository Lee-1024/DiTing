ALTER TABLE diting_risk_dispositions
    ADD COLUMN IF NOT EXISTS scope VARCHAR(32) NOT NULL DEFAULT 'event';

ALTER TABLE diting_risk_dispositions
    ADD COLUMN IF NOT EXISTS fingerprint TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_diting_risk_dispositions_fingerprint ON diting_risk_dispositions(fingerprint);
