CREATE TABLE IF NOT EXISTS diting_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(32) NOT NULL,
    dedupe_key VARCHAR(255) NOT NULL,
    source_id VARCHAR(255) NOT NULL DEFAULT '',
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    severity VARCHAR(32) NOT NULL DEFAULT 'warning',
    target VARCHAR(1024) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'open',
    disposition VARCHAR(32) NOT NULL DEFAULT '',
    handled_by VARCHAR(128) NOT NULL DEFAULT '',
    handled_at TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_diting_notifications_enforcement_source
ON diting_notifications(source_id)
WHERE type = 'enforcement';

CREATE UNIQUE INDEX IF NOT EXISTS idx_diting_notifications_active_dedupe
ON diting_notifications(dedupe_key)
WHERE status = 'open';

CREATE INDEX IF NOT EXISTS idx_diting_notifications_created_at
ON diting_notifications(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_diting_notifications_pending
ON diting_notifications(status, type, created_at DESC);

CREATE TABLE IF NOT EXISTS diting_notification_reads (
    notification_id UUID NOT NULL REFERENCES diting_notifications(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES diting_users(id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (notification_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_diting_notification_reads_user
ON diting_notification_reads(user_id, read_at DESC);