ALTER TABLE diting.audit_events
    ADD COLUMN IF NOT EXISTS host_id String AFTER host_name;

ALTER TABLE diting.audit_events
    MODIFY TTL event_date + INTERVAL 90 DAY;
