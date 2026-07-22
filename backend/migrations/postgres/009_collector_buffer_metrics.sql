ALTER TABLE diting_collector_heartbeats
    ADD COLUMN IF NOT EXISTS buffered_events BIGINT NOT NULL DEFAULT 0;

ALTER TABLE diting_collector_heartbeats
    ADD COLUMN IF NOT EXISTS dropped_events BIGINT NOT NULL DEFAULT 0;
