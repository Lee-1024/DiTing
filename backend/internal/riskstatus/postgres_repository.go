package riskstatus

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository 创建并初始化 New Postgres Repository 实例。
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// List 查询并返回 List 列表。
func (r *PostgresRepository) List(ctx context.Context, status string, limit int) ([]Disposition, error) {
	if limit <= 0 || limit > 500 {
		limit = 500
	}
	status, err := NormalizeStatus(status)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
SELECT event_id, status, note, handled_by, handled_at, created_at, updated_at, scope, fingerprint
FROM diting_risk_dispositions
WHERE status = $1
ORDER BY updated_at DESC
LIMIT $2
`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Disposition{}
	for rows.Next() {
		disposition, err := scanDispositionWithScope(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, disposition)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// ListByEventIDs 查询并返回 List By Event IDs 列表。
func (r *PostgresRepository) ListByEventIDs(ctx context.Context, eventIDs []string) (map[string]Disposition, error) {
	result := map[string]Disposition{}
	if len(eventIDs) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(ctx, `
SELECT event_id, status, note, handled_by, handled_at, created_at, updated_at, scope, fingerprint
FROM diting_risk_dispositions
WHERE event_id = ANY($1)
`, eventIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		disposition, err := scanDispositionWithScope(rows)
		if err != nil {
			return nil, err
		}
		result[disposition.EventID] = disposition
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// ListByFingerprints 查询并返回 List By Fingerprints 列表。
func (r *PostgresRepository) ListByFingerprints(ctx context.Context, fingerprints []string) (map[string]Disposition, error) {
	result := map[string]Disposition{}
	if len(fingerprints) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(ctx, `
SELECT event_id, status, note, handled_by, handled_at, created_at, updated_at, scope, fingerprint
FROM diting_risk_dispositions
WHERE status = 'ignore_similar'
  AND scope = 'similar'
  AND fingerprint = ANY($1)
ORDER BY updated_at DESC
`, fingerprints)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		disposition, err := scanDispositionWithScope(rows)
		if err != nil {
			return nil, err
		}
		if _, exists := result[disposition.Fingerprint]; !exists {
			result[disposition.Fingerprint] = disposition
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// Upsert 处理 Upsert 相关逻辑。
func (r *PostgresRepository) Upsert(ctx context.Context, disposition Disposition) (Disposition, error) {
	status, err := NormalizeStatus(disposition.Status)
	if err != nil {
		return Disposition{}, err
	}
	disposition.Status = status
	if disposition.Status == StatusIgnoreSimilar {
		disposition.Scope = "similar"
	} else if disposition.Scope == "" {
		disposition.Scope = "event"
	}
	row := r.pool.QueryRow(ctx, `
INSERT INTO diting_risk_dispositions (event_id, status, note, handled_by, handled_at, created_at, updated_at, scope, fingerprint)
VALUES ($1, $2::varchar, $3, $4, CASE WHEN $2::varchar = 'open' THEN NULL ELSE NOW() END, NOW(), NOW(), $5, $6)
ON CONFLICT (event_id) DO UPDATE
SET status = EXCLUDED.status,
    note = EXCLUDED.note,
    handled_by = EXCLUDED.handled_by,
    handled_at = CASE WHEN EXCLUDED.status = 'open' THEN NULL ELSE NOW() END,
    scope = EXCLUDED.scope,
    fingerprint = EXCLUDED.fingerprint,
    updated_at = NOW()
RETURNING event_id, status, note, handled_by, handled_at, created_at, updated_at, scope, fingerprint
`, disposition.EventID, disposition.Status, disposition.Note, disposition.HandledBy, disposition.Scope, disposition.Fingerprint)
	return scanDispositionWithScope(row)
}

type dispositionScanner interface {
	Scan(dest ...any) error
}

// scanDisposition 从查询结果中扫描并组装 scan Disposition。
func scanDisposition(scanner dispositionScanner) (Disposition, error) {
	var disposition Disposition
	var handledAt *time.Time
	if err := scanner.Scan(
		&disposition.EventID,
		&disposition.Status,
		&disposition.Note,
		&disposition.HandledBy,
		&handledAt,
		&disposition.CreatedAt,
		&disposition.UpdatedAt,
	); err != nil {
		return Disposition{}, err
	}
	disposition.HandledAt = handledAt
	return disposition, nil
}

// scanDispositionWithScope 从查询结果中扫描并组装 scan Disposition With Scope。
func scanDispositionWithScope(scanner dispositionScanner) (Disposition, error) {
	var disposition Disposition
	var handledAt *time.Time
	if err := scanner.Scan(
		&disposition.EventID,
		&disposition.Status,
		&disposition.Note,
		&disposition.HandledBy,
		&handledAt,
		&disposition.CreatedAt,
		&disposition.UpdatedAt,
		&disposition.Scope,
		&disposition.Fingerprint,
	); err != nil {
		return Disposition{}, err
	}
	disposition.HandledAt = handledAt
	return disposition, nil
}
