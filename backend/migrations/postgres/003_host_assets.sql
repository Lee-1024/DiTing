CREATE TABLE IF NOT EXISTS diting_host_assets (
    id UUID PRIMARY KEY,
    node_name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    host_ip VARCHAR(128) NOT NULL DEFAULT '',
    environment VARCHAR(64) NOT NULL DEFAULT '',
    owner VARCHAR(128) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_diting_host_assets_display_name ON diting_host_assets(display_name);
