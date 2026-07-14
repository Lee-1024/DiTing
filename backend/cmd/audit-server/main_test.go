package main

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseArgsReadsModeAndConfigPath(t *testing.T) {
	mode, configPath := parseArgs([]string{"audit-server", "collector", "--config", "./configs/config.yaml"})

	if mode != "collector" {
		t.Fatalf("expected collector mode, got %q", mode)
	}
	if configPath != "./configs/config.yaml" {
		t.Fatalf("expected config path, got %q", configPath)
	}
}

func TestParseArgsSupportsMigrateClickHouseMode(t *testing.T) {
	mode, configPath := parseArgs([]string{"audit-server", "migrate-clickhouse", "--config", "./configs/config.yaml"})

	if mode != "migrate-clickhouse" {
		t.Fatalf("expected migrate-clickhouse mode, got %q", mode)
	}
	if configPath != "./configs/config.yaml" {
		t.Fatalf("expected config path, got %q", configPath)
	}
}

func TestParseArgsSupportsCollectorOnceMode(t *testing.T) {
	mode, _ := parseArgs([]string{"audit-server", "collector-once"})

	if mode != "collector-once" {
		t.Fatalf("expected collector-once mode, got %q", mode)
	}
}

func TestParseArgsSupportsMigratePostgresMode(t *testing.T) {
	mode, _ := parseArgs([]string{"audit-server", "migrate-postgres"})

	if mode != "migrate-postgres" {
		t.Fatalf("expected migrate-postgres mode, got %q", mode)
	}
}

func TestParseArgsDefaultsToAPIAndExampleConfig(t *testing.T) {
	mode, configPath := parseArgs([]string{"audit-server"})

	if mode != "api" {
		t.Fatalf("expected api mode, got %q", mode)
	}
	if configPath != "./configs/config.example.yaml" {
		t.Fatalf("expected example config path, got %q", configPath)
	}
}

func TestMigrationFilesReturnsSQLFilesInNameOrder(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"002_second.sql", "001_first.sql", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("-- test"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	files, err := migrationFiles(dir)
	if err != nil {
		t.Fatalf("migrationFiles returned error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 sql files, got %d", len(files))
	}
	if filepath.Base(files[0]) != "001_first.sql" || filepath.Base(files[1]) != "002_second.sql" {
		t.Fatalf("unexpected file order %#v", files)
	}
}

func TestNewLogHandlerFormatsTimeInCST(t *testing.T) {
	var output bytes.Buffer
	handler := newLogHandler(&output)
	record := slog.NewRecord(time.Date(2026, 7, 14, 2, 30, 0, 0, time.UTC), slog.LevelInfo, "time check", 0)

	if err := handler.Handle(context.Background(), record); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	logLine := output.String()
	if !strings.Contains(logLine, "time=\"2026-07-14 10:30:00.000 CST\"") {
		t.Fatalf("expected slog time field to be formatted in CST, got %q", logLine)
	}
}
