package collectorhealth

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

func (r *PostgresRepository) List(ctx context.Context, now time.Time) ([]Heartbeat, error) {
	rows, err := r.pool.Query(ctx, `
SELECT host_id, host_name, last_seen_at, last_event_time, last_write_at, events_written, updated_at
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
		if err := rows.Scan(&item.HostID, &item.HostName, &item.LastSeenAt, &item.LastEventTime, &item.LastWriteAt, &item.EventsWritten, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, Enrich(item, now))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PostgresRepository) Upsert(ctx context.Context, update HeartbeatUpdate) error {
	if update.HostID == "" {
		return nil
	}
	if update.LastSeenAt.IsZero() {
		update.LastSeenAt = time.Now().UTC()
	}
	_, err := r.pool.Exec(ctx, `
INSERT INTO diting_collector_heartbeats (host_id, host_name, last_seen_at, last_event_time, last_write_at, events_written, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (host_id) DO UPDATE
SET host_name = EXCLUDED.host_name,
    last_seen_at = EXCLUDED.last_seen_at,
    last_event_time = COALESCE(EXCLUDED.last_event_time, diting_collector_heartbeats.last_event_time),
    last_write_at = COALESCE(EXCLUDED.last_write_at, diting_collector_heartbeats.last_write_at),
    events_written = diting_collector_heartbeats.events_written + EXCLUDED.events_written,
    updated_at = NOW()
`, update.HostID, update.HostName, update.LastSeenAt, update.LastEventTime, update.LastWriteAt, update.EventsWritten)
	return err
}
