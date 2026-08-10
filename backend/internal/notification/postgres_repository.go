package notification

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

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Upsert(ctx context.Context, input Input) (Notification, error) {
	row := r.pool.QueryRow(ctx, `
INSERT INTO diting_notifications (type, dedupe_key, source_id, title, description, severity, target, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'open', NOW(), NOW())
ON CONFLICT DO NOTHING
RETURNING id::text, type, dedupe_key, source_id, title, description, severity, target, status,
          disposition, handled_by, handled_at, false, created_at, updated_at, resolved_at
`, input.Type, input.DedupeKey, input.SourceID, input.Title, input.Description, input.Severity, input.Target)
	item, err := scanNotification(row)
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, err
	}
	if input.Type == TypeEnforcement {
		return r.getBySource(ctx, input.Type, input.SourceID)
	}
	return r.getOpenByDedupeKey(ctx, input.DedupeKey)
}

func (r *PostgresRepository) Resolve(ctx context.Context, dedupeKey string) error {
	_, err := r.pool.Exec(ctx, `
UPDATE diting_notifications
SET status = 'resolved', resolved_at = NOW(), updated_at = NOW()
WHERE dedupe_key = $1 AND status = 'open'
`, dedupeKey)
	return err
}

func (r *PostgresRepository) List(ctx context.Context, userID, view string, limit int) (ListResult, error) {
	view = NormalizeView(view)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var counts Counts
	if err := r.pool.QueryRow(ctx, `
SELECT
    COUNT(*) FILTER (WHERE reads.notification_id IS NULL),
    COUNT(*) FILTER (WHERE notifications.type = 'enforcement' AND notifications.status = 'open'),
    COUNT(*)
FROM diting_notifications notifications
LEFT JOIN diting_notification_reads reads
  ON reads.notification_id = notifications.id AND reads.user_id = $1::uuid
`, userID).Scan(&counts.Unread, &counts.Pending, &counts.All); err != nil {
		return ListResult{}, err
	}
	rows, err := r.pool.Query(ctx, `
SELECT notifications.id::text, notifications.type, notifications.dedupe_key, notifications.source_id,
       notifications.title, notifications.description, notifications.severity, notifications.target,
       notifications.status, notifications.disposition, notifications.handled_by, notifications.handled_at,
       reads.notification_id IS NOT NULL, notifications.created_at, notifications.updated_at, notifications.resolved_at
FROM diting_notifications notifications
LEFT JOIN diting_notification_reads reads
  ON reads.notification_id = notifications.id AND reads.user_id = $1::uuid
WHERE ($2 = 'all')
   OR ($2 = 'unread' AND reads.notification_id IS NULL)
   OR ($2 = 'pending' AND notifications.type = 'enforcement' AND notifications.status = 'open')
ORDER BY notifications.created_at DESC
LIMIT $3
`, userID, view, limit)
	if err != nil {
		return ListResult{}, err
	}
	defer rows.Close()
	items := []Notification{}
	for rows.Next() {
		item, scanErr := scanNotification(rows)
		if scanErr != nil {
			return ListResult{}, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, err
	}
	return ListResult{Items: items, Counts: counts}, nil
}

func (r *PostgresRepository) MarkRead(ctx context.Context, userID, id string) error {
	command, err := r.pool.Exec(ctx, `
INSERT INTO diting_notification_reads (notification_id, user_id, read_at)
SELECT id, $1::uuid, NOW() FROM diting_notifications WHERE id = $2::uuid
ON CONFLICT (notification_id, user_id) DO UPDATE SET read_at = EXCLUDED.read_at
`, userID, id)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `
INSERT INTO diting_notification_reads (notification_id, user_id, read_at)
SELECT id, $1::uuid, NOW() FROM diting_notifications
ON CONFLICT (notification_id, user_id) DO UPDATE SET read_at = EXCLUDED.read_at
`, userID)
	return err
}

func (r *PostgresRepository) Handle(ctx context.Context, id, disposition, username string) error {
	normalized, err := NormalizeDisposition(disposition)
	if err != nil {
		return err
	}
	command, err := r.pool.Exec(ctx, `
UPDATE diting_notifications
SET status = 'resolved', disposition = $2, handled_by = $3, handled_at = NOW(),
    resolved_at = NOW(), updated_at = NOW()
WHERE id = $1::uuid AND type = 'enforcement' AND status = 'open'
`, id, normalized, username)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) getBySource(ctx context.Context, notificationType, sourceID string) (Notification, error) {
	return scanNotification(r.pool.QueryRow(ctx, `
SELECT id::text, type, dedupe_key, source_id, title, description, severity, target, status,
       disposition, handled_by, handled_at, false, created_at, updated_at, resolved_at
FROM diting_notifications
WHERE type = $1 AND source_id = $2
ORDER BY created_at DESC
LIMIT 1
`, notificationType, sourceID))
}

func (r *PostgresRepository) getOpenByDedupeKey(ctx context.Context, dedupeKey string) (Notification, error) {
	return scanNotification(r.pool.QueryRow(ctx, `
SELECT id::text, type, dedupe_key, source_id, title, description, severity, target, status,
       disposition, handled_by, handled_at, false, created_at, updated_at, resolved_at
FROM diting_notifications
WHERE dedupe_key = $1 AND status = 'open'
ORDER BY created_at DESC
LIMIT 1
`, dedupeKey))
}

type notificationScanner interface {
	Scan(dest ...any) error
}

func scanNotification(scanner notificationScanner) (Notification, error) {
	var item Notification
	var handledAt *time.Time
	var resolvedAt *time.Time
	if err := scanner.Scan(
		&item.ID, &item.Type, &item.DedupeKey, &item.SourceID, &item.Title, &item.Description,
		&item.Severity, &item.Target, &item.Status, &item.Disposition, &item.HandledBy,
		&handledAt, &item.Read, &item.CreatedAt, &item.UpdatedAt, &resolvedAt,
	); err != nil {
		return Notification{}, err
	}
	item.HandledAt = handledAt
	item.ResolvedAt = resolvedAt
	return item, nil
}
