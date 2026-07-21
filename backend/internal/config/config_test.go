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
	if cfg.Collector.HostID != "server-001" {
		t.Fatalf("expected collector host id server-001, got %q", cfg.Collector.HostID)
	}
	if cfg.Collector.HostName != "diting-test-host" {
		t.Fatalf("expected collector host name diting-test-host, got %q", cfg.Collector.HostName)
	}
	if cfg.Collector.TetragonGRPCAddr != "127.0.0.1:54321" {
		t.Fatalf("expected collector grpc address 127.0.0.1:54321, got %q", cfg.Collector.TetragonGRPCAddr)
	}
	if cfg.Collector.ReconnectIntervalSeconds != 5 {
		t.Fatalf("expected reconnect interval 5, got %d", cfg.Collector.ReconnectIntervalSeconds)
	}
	if cfg.Collector.Token != "change-me-collector-token" {
		t.Fatalf("expected collector token from config, got %q", cfg.Collector.Token)
	}
	if cfg.Collector.OutputMode != "clickhouse" {
		t.Fatalf("expected collector output mode clickhouse, got %q", cfg.Collector.OutputMode)
	}
	if cfg.Collector.IngestURL != "http://127.0.0.1:8080/api/v1/ingest/events" {
		t.Fatalf("expected ingest url from config, got %q", cfg.Collector.IngestURL)
	}
}
