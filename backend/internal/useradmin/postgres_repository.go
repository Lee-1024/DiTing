package useradmin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"diting/backend/internal/auth"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := r.pool.Query(ctx, `
SELECT
	u.id::text,
	u.username,
	u.display_name,
	u.email,
	u.status,
	COALESCE(array_agg(role.name ORDER BY role.name) FILTER (WHERE role.name IS NOT NULL), ARRAY[]::varchar[]),
	u.created_at,
	u.updated_at
FROM diting_users u
LEFT JOIN diting_user_roles ur ON ur.user_id = u.id
LEFT JOIN diting_roles role ON role.id = ur.role_id
GROUP BY u.id
ORDER BY u.created_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *PostgresRepository) CreateUser(ctx context.Context, request CreateUserRequest) (User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer rollback(ctx, tx)

	row := tx.QueryRow(ctx, `
INSERT INTO diting_users (id, username, password_hash, display_name, email, status, created_at, updated_at)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, NOW(), NOW())
RETURNING id::text, username, display_name, email, status, created_at, updated_at
`, request.Username, auth.HashPassword(request.Password, randomSalt()), request.DisplayName, request.Email, normalizeStatus(request.Status))
	user, err := scanUserWithoutRoles(row)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrConflict
		}
		return User{}, err
	}
	if err := replaceUserRoles(ctx, tx, user.ID, request.Roles); err != nil {
		return User{}, err
	}
	user.Roles = normalizeRoles(request.Roles)
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return user, nil
}

func (r *PostgresRepository) UpdateUser(ctx context.Context, id string, request UpdateUserRequest) (User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer rollback(ctx, tx)

	current, err := getUserForUpdate(ctx, tx, id)
	if err != nil {
		return User{}, err
	}
	if removesLastAdmin(ctx, tx, current, request.Roles, request.Status) {
		return User{}, ErrLastAdmin
	}
	row := tx.QueryRow(ctx, `
UPDATE diting_users
SET display_name = $2,
    email = $3,
    status = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING id::text, username, display_name, email, status, created_at, updated_at
`, id, request.DisplayName, request.Email, normalizeStatus(request.Status))
	user, err := scanUserWithoutRoles(row)
	if err != nil {
		return User{}, mapUserNotFound(err)
	}
	if err := replaceUserRoles(ctx, tx, id, request.Roles); err != nil {
		return User{}, err
	}
	user.Roles = normalizeRoles(request.Roles)
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return user, nil
}

func (r *PostgresRepository) ResetPassword(ctx context.Context, id string, password string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE diting_users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, id, auth.HashPassword(password, randomSalt()))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteUser(ctx context.Context, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(ctx, tx)

	current, err := getUserForUpdate(ctx, tx, id)
	if err != nil {
		return err
	}
	if current.Status == "active" && hasRole(current.Roles, "admin") && activeAdminCount(ctx, tx) == 1 {
		return ErrLastAdmin
	}
	tag, err := tx.Exec(ctx, `DELETE FROM diting_users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

func (r *PostgresRepository) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id::text, name, description, created_at, updated_at
FROM diting_roles
ORDER BY name ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := []Role{}
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return roles, nil
}

type userScanner interface {
	Scan(dest ...any) error
}

func scanUser(scanner userScanner) (User, error) {
	var user User
	if err := scanner.Scan(&user.ID, &user.Username, &user.DisplayName, &user.Email, &user.Status, &user.Roles, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return User{}, err
	}
	return user, nil
}

func scanUserWithoutRoles(scanner userScanner) (User, error) {
	var user User
	if err := scanner.Scan(&user.ID, &user.Username, &user.DisplayName, &user.Email, &user.Status, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return User{}, mapUserNotFound(err)
	}
	return user, nil
}

func getUserForUpdate(ctx context.Context, tx pgx.Tx, id string) (User, error) {
	row := tx.QueryRow(ctx, `
SELECT id::text, username, display_name, email, status, created_at, updated_at
FROM diting_users
WHERE id = $1
FOR UPDATE
`, id)
	user, err := scanUserWithoutRoles(row)
	if err != nil {
		return User{}, mapUserNotFound(err)
	}
	roles, err := userRoles(ctx, tx, id)
	if err != nil {
		return User{}, err
	}
	user.Roles = roles
	return user, nil
}

func userRoles(ctx context.Context, tx pgx.Tx, userID string) ([]string, error) {
	rows, err := tx.Query(ctx, `
SELECT role.name
FROM diting_user_roles ur
JOIN diting_roles role ON role.id = ur.role_id
WHERE ur.user_id = $1
ORDER BY role.name
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := []string{}
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return roles, nil
}

func replaceUserRoles(ctx context.Context, tx pgx.Tx, userID string, roles []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM diting_user_roles WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for _, role := range normalizeRoles(roles) {
		tag, err := tx.Exec(ctx, `
INSERT INTO diting_user_roles (user_id, role_id)
SELECT $1::uuid, id FROM diting_roles WHERE name = $2
`, userID, role)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrRoleNotFound
		}
	}
	return nil
}

func removesLastAdmin(ctx context.Context, tx pgx.Tx, current User, nextRoles []string, nextStatus string) bool {
	if current.Status != "active" || !hasRole(current.Roles, "admin") {
		return false
	}
	if normalizeStatus(nextStatus) == "active" && hasRole(nextRoles, "admin") {
		return false
	}
	return activeAdminCount(ctx, tx) == 1
}

func activeAdminCount(ctx context.Context, tx pgx.Tx) int {
	var count int
	if err := tx.QueryRow(ctx, `
SELECT count(*)
FROM diting_users u
JOIN diting_user_roles ur ON ur.user_id = u.id
JOIN diting_roles role ON role.id = ur.role_id
WHERE u.status = 'active' AND role.name = 'admin'
`).Scan(&count); err != nil {
		return 0
	}
	return count
}

func mapUserNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func rollback(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func randomSalt() string {
	var data [16]byte
	if _, err := io.ReadFull(rand.Reader, data[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(data[:])
}

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "duplicate key value")
}
