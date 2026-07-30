CREATE TABLE IF NOT EXISTS diting_ai_risk_analyses (
    event_id VARCHAR(128) PRIMARY KEY,
    ai_severity VARCHAR(32) NOT NULL,
    verdict VARCHAR(32) NOT NULL,
    confidence INTEGER NOT NULL DEFAULT 0,
    reason TEXT NOT NULL DEFAULT '',
    evidence JSONB NOT NULL DEFAULT '[]'::jsonb,
    suggestion TEXT NOT NULL DEFAULT '',
    model VARCHAR(128) NOT NULL DEFAULT '',
    raw_response TEXT NOT NULL DEFAULT '',
    analyzed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_diting_ai_risk_analyses_verdict ON diting_ai_risk_analyses(verdict);
CREATE INDEX IF NOT EXISTS idx_diting_ai_risk_analyses_analyzed_at ON diting_ai_risk_analyses(analyzed_at);
