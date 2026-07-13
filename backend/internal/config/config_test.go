package config

import "testing"

func TestLoadReadsServerAndDatabaseConfig(t *testing.T) {
	cfg, err := Load("../../configs/config.example.yaml")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Fatalf("expected server port 8080, got %d", cfg.Server.Port)
	}
	if cfg.ClickHouse.Database != "diting" {
		t.Fatalf("expected ClickHouse database diting, got %q", cfg.ClickHouse.Database)
	}
	if cfg.Postgres.Database != "diting" {
		t.Fatalf("expected PostgreSQL database diting, got %q", cfg.Postgres.Database)
	}
}
