CREATE TABLE IF NOT EXISTS diting_collector_heartbeats (
    host_id VARCHAR(128) PRIMARY KEY,
    host_name VARCHAR(128) NOT NULL DEFAULT '',
    last_seen_at TIMESTAMPTZ NOT NULL,
    last_event_time TIMESTAMPTZ,
    last_write_at TIMESTAMPTZ,
    events_written BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_diting_collector_heartbeats_last_seen_at ON diting_collector_heartbeats(last_seen_at);
