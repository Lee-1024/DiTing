package operationlog

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, entry Entry) error {
	_, err := r.pool.Exec(ctx, `
INSERT INTO diting_operation_logs (id, user_id, username, method, path, status, ip, user_agent, created_at)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, NOW())
`, entry.UserID, entry.Username, entry.Method, entry.Path, entry.Status, entry.IP, entry.UserAgent)
	return err
}
