package postgres

import (
	"context"
	"fmt"
	"os"
	"strings"

	"diting/backend/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func DSN(cfg config.PostgresConfig) string {
	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	return fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Database, cfg.Username, cfg.Password, sslMode)
}

func Connect(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, DSN(cfg))
}

func ExecuteMigrationFile(ctx context.Context, pool *pgxpool.Pool, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, statement := range splitStatements(string(data)) {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := pool.Exec(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func splitStatements(sql string) []string {
	parts := strings.Split(sql, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			statements = append(statements, trimmed)
		}
	}
	return statements
}
