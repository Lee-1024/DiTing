package collectorhealth

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
func (r *PostgresRepository) List(ctx context.Context, now time.Time) ([]Heartbeat, error) {
	rows, err := r.pool.Query(ctx, `
SELECT host_id, host_name, input_mode, last_error, last_seen_at, last_event_time, last_write_at, events_written, buffered_events, dropped_events, updated_at
FROM diting_collector_heartbeats
ORDER BY last_seen_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Heartbeat{}
	for rows.Next() {
		var item Heartbeat
		if err := rows.Scan(&item.HostID, &item.HostName, &item.InputMode, &item.LastError, &item.LastSeenAt, &item.LastEventTime, &item.LastWriteAt, &item.EventsWritten, &item.BufferedEvents, &item.DroppedEvents, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, Enrich(item, now))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// Upsert 处理 Upsert 相关逻辑。
func (r *PostgresRepository) Upsert(ctx context.Context, update HeartbeatUpdate) error {
	if update.HostID == "" {
		return nil
	}
	if update.LastSeenAt.IsZero() {
		update.LastSeenAt = time.Now().UTC()
	}
	_, err := r.pool.Exec(ctx, `
INSERT INTO diting_collector_heartbeats (host_id, host_name, input_mode, last_error, last_seen_at, last_event_time, last_write_at, events_written, buffered_events, dropped_events, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
ON CONFLICT (host_id) DO UPDATE
SET host_name = EXCLUDED.host_name,
    input_mode = EXCLUDED.input_mode,
    last_error = CASE
        WHEN EXCLUDED.last_error != '' OR $11 THEN EXCLUDED.last_error
        ELSE diting_collector_heartbeats.last_error
    END,
    last_seen_at = EXCLUDED.last_seen_at,
    last_event_time = COALESCE(EXCLUDED.last_event_time, diting_collector_heartbeats.last_event_time),
    last_write_at = COALESCE(EXCLUDED.last_write_at, diting_collector_heartbeats.last_write_at),
    events_written = diting_collector_heartbeats.events_written + EXCLUDED.events_written,
    buffered_events = EXCLUDED.buffered_events,
    dropped_events = EXCLUDED.dropped_events,
    updated_at = NOW()
`, update.HostID, update.HostName, inputMode(update.InputMode), update.LastError, update.LastSeenAt, update.LastEventTime, update.LastWriteAt, update.EventsWritten, update.BufferedEvents, update.DroppedEvents, update.ClearError)
	return err
}

// inputMode 处理 input Mode 相关逻辑。
func inputMode(value string) string {
	if value == "" {
		return "file"
	}
	return value
}
