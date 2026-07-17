ALTER TABLE diting_host_assets
    ADD COLUMN IF NOT EXISTS host_id VARCHAR(255) NOT NULL DEFAULT '';

ALTER TABLE diting_host_assets
    ADD COLUMN IF NOT EXISTS host_name VARCHAR(255) NOT NULL DEFAULT '';

ALTER TABLE diting_host_assets
    ADD COLUMN IF NOT EXISTS department VARCHAR(128) NOT NULL DEFAULT '';

UPDATE diting_host_assets
SET host_id = node_name
WHERE host_id = '';

UPDATE diting_host_assets
SET host_name = display_name
WHERE host_name = '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_diting_host_assets_host_id_unique ON diting_host_assets(host_id);
CREATE INDEX IF NOT EXISTS idx_diting_host_assets_host_name ON diting_host_assets(host_name);
