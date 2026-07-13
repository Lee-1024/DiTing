package riskstatus

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) ListByEventIDs(ctx context.Context, eventIDs []string) (map[string]Disposition, error) {
	result := map[string]Disposition{}
	if len(eventIDs) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(ctx, `
SELECT event_id, status, note, handled_by, handled_at, created_at, updated_at
FROM diting_risk_dispositions
WHERE event_id = ANY($1)
`, eventIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		disposition, err := scanDisposition(rows)
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

func (r *PostgresRepository) Upsert(ctx context.Context, disposition Disposition) (Disposition, error) {
	status, err := NormalizeStatus(disposition.Status)
	if err != nil {
		return Disposition{}, err
	}
	disposition.Status = status
	row := r.pool.QueryRow(ctx, `
INSERT INTO diting_risk_dispositions (event_id, status, note, handled_by, handled_at, created_at, updated_at)
VALUES ($1, $2::varchar, $3, $4, CASE WHEN $2::varchar = 'open' THEN NULL ELSE NOW() END, NOW(), NOW())
ON CONFLICT (event_id) DO UPDATE
SET status = EXCLUDED.status,
    note = EXCLUDED.note,
    handled_by = EXCLUDED.handled_by,
    handled_at = CASE WHEN EXCLUDED.status = 'open' THEN NULL ELSE NOW() END,
    updated_at = NOW()
RETURNING event_id, status, note, handled_by, handled_at, created_at, updated_at
`, disposition.EventID, disposition.Status, disposition.Note, disposition.HandledBy)
	return scanDisposition(row)
}

type dispositionScanner interface {
	Scan(dest ...any) error
}

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
