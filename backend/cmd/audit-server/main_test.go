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

	"diting/backend/internal/postgres"
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

func TestParseArgsSupportsClearTestDataMode(t *testing.T) {
	mode, _ := parseArgs([]string{"audit-server", "clear-test-data"})

	if mode != "clear-test-data" {
		t.Fatalf("expected clear-test-data mode, got %q", mode)
	}
}

func TestPostgresRuntimeDataCleanupStatements(t *testing.T) {
	statements := postgresRuntimeDataCleanupStatements()

	expected := []string{
		"DELETE FROM diting_ai_risk_analyses",
		"DELETE FROM diting_risk_dispositions",
		"DELETE FROM diting_collector_heartbeats",
		"DELETE FROM diting_host_assets",
		"DELETE FROM diting_audit_rules",
	}
	for _, statement := range expected {
		if !containsString(statements, statement) {
			t.Fatalf("expected cleanup statement %q in %#v", statement, statements)
		}
	}
	if containsString(statements, "DELETE FROM diting_system_configs WHERE key = 'collector_filter'") {
		t.Fatalf("collector filter config must be preserved during cleanup: %#v", statements)
	}
}

func TestClickHouseRuntimeDataTablesIncludeOperationAggregates(t *testing.T) {
	tables := clickHouseRuntimeDataTables("diting")

	expected := []string{
		"diting.audit_events",
		"diting.audit_operation_groups_hourly",
		"diting.audit_overview_hourly",
		"diting.audit_host_stats_hourly",
		"diting.audit_user_stats_hourly",
		"diting.audit_command_stats_hourly",
		"diting.audit_rule_hit_stats_hourly",
		"diting.audit_host_behavior_hourly",
	}
	for _, table := range expected {
		if !containsString(tables, table) {
			t.Fatalf("expected clickhouse cleanup table %q in %#v", table, tables)
		}
	}
}

func TestSplitLogHandlerWritesErrorsToStderr(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	logger := slog.New(newSplitLogHandler(&stdout, &stderr))

	logger.Info("info message")
	if stderr.Len() != 0 {
		t.Fatalf("expected info log to avoid stderr, got %s", stderr.String())
	}
	logger.Error("error message")
	if !strings.Contains(stdout.String(), "error message") {
		t.Fatalf("expected error log in stdout, got %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "error message") {
		t.Fatalf("expected error log in stderr, got %s", stderr.String())
	}
}

func TestProductionPreReleaseBaselineSeedsCollectorFilterAndAuditRules(t *testing.T) {
	for _, expected := range []string{
		"INSERT INTO diting_system_configs",
		"'collector_filter'",
		"pre-root-process-low-risk",
		"INSERT INTO diting_audit_rules",
		"生产-反弹 Shell 命令",
		"生产-敏感文件写入",
		"生产-Web 服务拉起 Shell",
		"ON CONFLICT (id) DO UPDATE",
	} {
		if !strings.Contains(postgres.ProductionPreReleaseBaselineSQL, expected) {
			t.Fatalf("expected production pre-release baseline SQL to include %q", expected)
		}
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestCollectorInputModeDefaultsAndNormalizes(t *testing.T) {
	if collectorInputMode("") != "file" {
		t.Fatal("expected empty collector input mode to default to file")
	}
	if collectorInputMode(" GRPC ") != "grpc" {
		t.Fatal("expected collector input mode to normalize grpc")
	}
}

func TestCollectorOutputModeDefaultsAndNormalizes(t *testing.T) {
	if collectorOutputMode("") != "clickhouse" {
		t.Fatal("expected empty collector output mode to default to clickhouse")
	}
	if collectorOutputMode(" API ") != "api" {
		t.Fatal("expected collector output mode to normalize api")
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
