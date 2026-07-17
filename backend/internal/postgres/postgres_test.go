package postgres

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"diting/backend/internal/config"
)

func TestDSNBuildsPostgresConnectionString(t *testing.T) {
	dsn := DSN(config.PostgresConfig{
		Host: "10.54.56.54", Port: 31060, Database: "myappdb", Username: "admin", Password: "secure_password", SSLMode: "disable",
	})

	for _, part := range []string{
		"host=10.54.56.54",
		"port=31060",
		"dbname=myappdb",
		"user=admin",
		"password=secure_password",
		"sslmode=disable",
	} {
		if !strings.Contains(dsn, part) {
			t.Fatalf("expected DSN to contain %q, got %q", part, dsn)
		}
	}
}

func TestMigrationFilesReturnsSortedSQLFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"002_second.sql", "001_first.sql", "readme.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("-- noop"), 0o600); err != nil {
			t.Fatalf("write migration file: %v", err)
		}
	}

	files, err := MigrationFiles(dir)
	if err != nil {
		t.Fatalf("MigrationFiles returned error: %v", err)
	}

	expected := []string{filepath.Join(dir, "001_first.sql"), filepath.Join(dir, "002_second.sql")}
	if !reflect.DeepEqual(files, expected) {
		t.Fatalf("expected sorted SQL files %#v, got %#v", expected, files)
	}
}

func TestBootstrapAddsHostAssetColumnsBeforeIndexes(t *testing.T) {
	alterIndex := strings.Index(bootstrapSQL, "ADD COLUMN IF NOT EXISTS host_id")
	indexIndex := strings.Index(bootstrapSQL, "idx_diting_host_assets_host_id_unique")
	if alterIndex == -1 {
		t.Fatal("expected bootstrap SQL to add host_id column for existing host asset tables")
	}
	if indexIndex == -1 {
		t.Fatal("expected bootstrap SQL to create host_id index")
	}
	if alterIndex > indexIndex {
		t.Fatal("expected host_id column to be added before host_id index is created")
	}
}

func TestBootstrapAddsCollectorHeartbeatModeColumns(t *testing.T) {
	for _, column := range []string{"ADD COLUMN IF NOT EXISTS input_mode", "ADD COLUMN IF NOT EXISTS last_error"} {
		if !strings.Contains(bootstrapSQL, column) {
			t.Fatalf("expected bootstrap SQL to include %q", column)
		}
	}
}
