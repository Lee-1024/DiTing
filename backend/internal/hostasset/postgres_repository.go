package hostasset

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
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
func (r *PostgresRepository) Create(ctx context.Context, asset HostAsset) (HostAsset, error) {
	asset = normalizeAsset(asset)
	row := r.pool.QueryRow(ctx, `
INSERT INTO diting_host_assets (id, host_id, host_name, node_name, display_name, host_ip, environment, owner, department, description, created_at, updated_at)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
RETURNING id::text, host_id, host_name, node_name, display_name, host_ip, environment, owner, department, description, created_at, updated_at
`, asset.HostID, asset.HostName, asset.NodeName, asset.DisplayName, asset.HostIP, asset.Environment, asset.Owner, asset.Department, asset.Description)
	return scanAsset(row)
}

// List 查询并返回 List 列表。
func (r *PostgresRepository) List(ctx context.Context) ([]HostAsset, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id::text, host_id, host_name, node_name, display_name, host_ip, environment, owner, department, description, created_at, updated_at
FROM diting_host_assets
ORDER BY updated_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assets := []HostAsset{}
	for rows.Next() {
		asset, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return assets, nil
}

// Get 查询并返回指定的 Get。
func (r *PostgresRepository) Get(ctx context.Context, id string) (HostAsset, error) {
	row := r.pool.QueryRow(ctx, `
SELECT id::text, host_id, host_name, node_name, display_name, host_ip, environment, owner, department, description, created_at, updated_at
FROM diting_host_assets
WHERE id = $1
`, id)
	asset, err := scanAsset(row)
	if err != nil {
		return HostAsset{}, mapNotFound(err)
	}
	return asset, nil
}

// Update 更新指定的 Update。
func (r *PostgresRepository) Update(ctx context.Context, id string, asset HostAsset) (HostAsset, error) {
	asset = normalizeAsset(asset)
	row := r.pool.QueryRow(ctx, `
UPDATE diting_host_assets
SET host_id = $2,
    host_name = $3,
    node_name = $4,
    display_name = $5,
    host_ip = $6,
    environment = $7,
    owner = $8,
    department = $9,
    description = $10,
    updated_at = NOW()
WHERE id = $1
RETURNING id::text, host_id, host_name, node_name, display_name, host_ip, environment, owner, department, description, created_at, updated_at
`, id, asset.HostID, asset.HostName, asset.NodeName, asset.DisplayName, asset.HostIP, asset.Environment, asset.Owner, asset.Department, asset.Description)
	updated, err := scanAsset(row)
	if err != nil {
		return HostAsset{}, mapNotFound(err)
	}
	return updated, nil
}

// Delete 删除指定的 Delete。
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	commandTag, err := r.pool.Exec(ctx, `DELETE FROM diting_host_assets WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type assetScanner interface {
	Scan(dest ...any) error
}

// scanAsset 从查询结果中扫描并组装 scan Asset。
func scanAsset(scanner assetScanner) (HostAsset, error) {
	var asset HostAsset
	var createdAt time.Time
	var updatedAt time.Time
	if err := scanner.Scan(
		&asset.ID,
		&asset.HostID,
		&asset.HostName,
		&asset.NodeName,
		&asset.DisplayName,
		&asset.HostIP,
		&asset.Environment,
		&asset.Owner,
		&asset.Department,
		&asset.Description,
		&createdAt,
		&updatedAt,
	); err != nil {
		return HostAsset{}, err
	}
	asset.CreatedAt = createdAt
	asset.UpdatedAt = updatedAt
	return asset, nil
}

// mapNotFound 映射 map Not Found 的错误或数据结构。
func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
