package postgres

import (
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
