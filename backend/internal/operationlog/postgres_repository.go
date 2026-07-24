package operationlog

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository 创建并初始化 New Postgres Repository 实例。
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Create 创建新的 Create。
func (r *PostgresRepository) Create(ctx context.Context, entry Entry) error {
	_, err := r.pool.Exec(ctx, `
INSERT INTO diting_operation_logs (id, user_id, username, method, path, status, ip, user_agent, created_at)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, NOW())
`, entry.UserID, entry.Username, entry.Method, entry.Path, entry.Status, entry.IP, entry.UserAgent)
	return err
}

// List 查询并返回 List 列表。
func (r *PostgresRepository) List(ctx context.Context, query Query) ([]Entry, int, error) {
	where, args := buildWhere(query)
	limit := query.PageSize
	if limit <= 0 {
		limit = 10
	}
	offset := 0
	if query.Page > 1 {
		offset = (query.Page - 1) * limit
	}

	listArgs := append(append([]any{}, args...), limit, offset)
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
SELECT id::text, COALESCE(user_id::text, ''), username, method, path, status, ip, user_agent, created_at
FROM diting_operation_logs
WHERE %s
ORDER BY created_at DESC
LIMIT $%d OFFSET $%d
`, where, len(args)+1, len(args)+2), listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []Entry{}
	for rows.Next() {
		var item Entry
		if err := rows.Scan(&item.ID, &item.UserID, &item.Username, &item.Method, &item.Path, &item.Status, &item.IP, &item.UserAgent, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT count(*) FROM diting_operation_logs WHERE %s`, where), args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// buildWhere 构建 build Where 所需的数据或表达式。
func buildWhere(query Query) (string, []any) {
	conditions := []string{"created_at >= $1", "created_at <= $2"}
	args := []any{query.StartTime, query.EndTime}
	if query.Username != "" {
		args = append(args, query.Username)
		conditions = append(conditions, fmt.Sprintf("username = $%d", len(args)))
	}
	if query.Method != "" {
		args = append(args, strings.ToUpper(query.Method))
		conditions = append(conditions, fmt.Sprintf("method = $%d", len(args)))
	}
	if query.Keyword != "" {
		args = append(args, "%"+query.Keyword+"%")
		conditions = append(conditions, fmt.Sprintf("path ILIKE $%d", len(args)))
	}
	if query.Status > 0 {
		args = append(args, query.Status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	return strings.Join(conditions, " AND "), args
}
