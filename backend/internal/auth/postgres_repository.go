package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserRepository 创建并初始化 New Postgres User Repository 实例。
func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

// FindByUsername 处理 Find By Username 相关逻辑。
func (r *PostgresUserRepository) FindByUsername(ctx context.Context, username string) (User, error) {
	row := r.pool.QueryRow(ctx, `
SELECT
	u.id::text,
	u.username,
	u.password_hash,
	u.display_name,
	u.email,
	u.status,
	COALESCE(array_agg(role.name) FILTER (WHERE role.name IS NOT NULL), ARRAY[]::varchar[])
FROM diting_users u
LEFT JOIN diting_user_roles ur ON ur.user_id = u.id
LEFT JOIN diting_roles role ON role.id = ur.role_id
WHERE u.username = $1
GROUP BY u.id
`, username)
	var user User
	if err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.DisplayName, &user.Email, &user.Status, &user.Roles); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrInvalidCredentials
		}
		return User{}, err
	}
	return user, nil
}

// UpdatePassword 更新指定的 Update Password。
func (r *PostgresUserRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE diting_users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash)
	return err
}
