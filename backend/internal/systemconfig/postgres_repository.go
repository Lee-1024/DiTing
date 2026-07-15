package systemconfig

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) GetCollectorFilter(ctx context.Context) (CollectorFilterConfig, error) {
	var data []byte
	err := r.pool.QueryRow(ctx, `SELECT value FROM diting_system_configs WHERE key = $1`, CollectorFilterKey).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return DefaultCollectorFilterConfig(), nil
	}
	if err != nil {
		return CollectorFilterConfig{}, err
	}
	return unmarshalCollectorFilterConfig(data)
}

func (r *PostgresRepository) SaveCollectorFilter(ctx context.Context, config CollectorFilterConfig) error {
	data, err := marshalCollectorFilterConfig(normalizeCollectorFilterConfig(config))
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
INSERT INTO diting_system_configs (key, value, description, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    description = EXCLUDED.description,
    updated_at = NOW()
`, CollectorFilterKey, string(data), "Collector noise filter configuration")
	return err
}

func marshalCollectorFilterConfig(config CollectorFilterConfig) ([]byte, error) {
	return json.Marshal(normalizeCollectorFilterConfig(config))
}

func unmarshalCollectorFilterConfig(data []byte) (CollectorFilterConfig, error) {
	var config CollectorFilterConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return CollectorFilterConfig{}, err
	}
	return normalizeCollectorFilterConfig(config), nil
}
