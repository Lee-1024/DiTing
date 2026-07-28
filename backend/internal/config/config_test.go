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
	if !cfg.Redis.Enabled {
		t.Fatal("expected redis to be enabled in example config")
	}
	if cfg.Redis.Addr != "127.0.0.1:6379" {
		t.Fatalf("expected redis addr 127.0.0.1:6379, got %q", cfg.Redis.Addr)
	}
	if cfg.Redis.DB != 0 {
		t.Fatalf("expected redis db 0, got %d", cfg.Redis.DB)
	}
	if cfg.Redis.ResponseCacheTTLSeconds != 15 {
		t.Fatalf("expected redis response cache ttl 15, got %d", cfg.Redis.ResponseCacheTTLSeconds)
	}
	if cfg.Redis.HostProfileCacheTTLSeconds != 300 {
		t.Fatalf("expected host profile cache ttl 300, got %d", cfg.Redis.HostProfileCacheTTLSeconds)
	}
}
