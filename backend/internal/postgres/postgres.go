package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"diting/backend/internal/config"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

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

func MigrationFiles(dir string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func ExecuteMigrations(ctx context.Context, pool Execer, dir string) error {
	files, err := MigrationFiles(dir)
	if err != nil {
		return err
	}
	for _, path := range files {
		if err := ExecuteMigrationFile(ctx, pool, path); err != nil {
			return fmt.Errorf("execute postgres migration %s: %w", path, err)
		}
	}
	return nil
}

func ExecuteMigrationFile(ctx context.Context, pool Execer, path string) error {
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
