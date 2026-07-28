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
		"audit_command_stats_hourly",
		"audit_rule_hit_stats_hourly",
		"audit_host_behavior_hourly",
		"CREATE MATERIALIZED VIEW IF NOT EXISTS diting.mv_audit_overview_hourly",
		"AggregatingMergeTree",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected migration to contain %q", expected)
		}
	}
}
