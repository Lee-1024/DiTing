ALTER TABLE diting_collector_heartbeats
    ADD COLUMN IF NOT EXISTS input_mode VARCHAR(32) NOT NULL DEFAULT 'file';

ALTER TABLE diting_collector_heartbeats
    ADD COLUMN IF NOT EXISTS last_error TEXT NOT NULL DEFAULT '';
