package clickhouse

import (
	"os"
	"strings"
	"testing"
)

func TestQueryAccelerationMigrationDefinesAggregateTables(t *testing.T) {
	data, err := os.ReadFile("../../migrations/clickhouse/003_query_acceleration.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(data)
	for _, expected := range []string{
		"audit_overview_hourly",
		"audit_host_stats_hourly",
		"audit_user_stats_hourly",
		"audit_host_user_stats_hourly",
		"audit_command_stats_hourly",
		"audit_rule_hit_stats_hourly",
		"audit_operation_groups_hourly",
		"audit_host_behavior_hourly",
		"CREATE MATERIALIZED VIEW IF NOT EXISTS diting.mv_audit_overview_hourly",
		"AggregatingMergeTree",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected migration to contain %q", expected)
		}
	}
}

func TestQueryAccelerationMigrationAvoidsHostAliasSubstitution(t *testing.T) {
	data, err := os.ReadFile("../../migrations/clickhouse/003_query_acceleration.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(data)
	if strings.Contains(sql, "if(host_id != '', host_id, if(node_name != '', node_name, host_name)) AS host_key,\n    anyLast(host_name) AS host_name") {
		t.Fatalf("host stats view must not aggregate host_name/node_name in the same select that computes host_key")
	}
	for _, expected := range []string{
		"host_name AS raw_host_name",
		"node_name AS raw_node_name",
		"anyLast(raw_host_name) AS host_name",
		"anyLast(raw_node_name) AS node_name",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected host stats view to contain %q", expected)
		}
	}
}

func TestQueryAccelerationMigrationDefinesOperationGroupsWithoutStartupBackfill(t *testing.T) {
	data, err := os.ReadFile("../../migrations/clickhouse/003_query_acceleration.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(data)
	for _, expected := range []string{
		"CREATE MATERIALIZED VIEW IF NOT EXISTS diting.mv_audit_operation_groups_hourly",
		"host_name AS raw_host_name",
		"node_name AS raw_node_name",
		"anyLast(raw_host_name) AS host_name",
		"anyLast(raw_node_name) AS node_name",
		"GROUP BY hour, event_second, audit_host_key",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected operation group migration to contain %q", expected)
		}
	}
	for _, forbidden := range []string{
		"TRUNCATE TABLE IF EXISTS diting.audit_operation_groups_hourly",
		"INSERT INTO diting.audit_operation_groups_hourly",
		"WHERE event_time >= now() - INTERVAL 31 DAY",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("auto migration must not run startup backfill statement %q", forbidden)
		}
	}
}

func TestManualQueryAccelerationResetScriptRecreatesAggregateObjects(t *testing.T) {
	data, err := os.ReadFile("../../migrations/clickhouse/manual_reset_query_acceleration.sql")
	if err != nil {
		t.Fatalf("read manual reset script: %v", err)
	}
	sql := string(data)
	for _, expected := range []string{
		"DROP VIEW IF EXISTS diting.mv_audit_operation_groups_hourly",
		"DROP TABLE IF EXISTS diting.audit_operation_groups_hourly",
		"CREATE TABLE IF NOT EXISTS diting.audit_operation_groups_hourly",
		"CREATE MATERIALIZED VIEW IF NOT EXISTS diting.mv_audit_operation_groups_hourly",
		"CREATE MATERIALIZED VIEW IF NOT EXISTS diting.mv_audit_host_stats_hourly",
		"CREATE MATERIALIZED VIEW IF NOT EXISTS diting.mv_audit_host_user_stats_hourly",
		"CREATE MATERIALIZED VIEW IF NOT EXISTS diting.mv_audit_host_behavior_rule_hourly",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected manual reset script to contain %q", expected)
		}
	}
}
