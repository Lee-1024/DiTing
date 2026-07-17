package collectorhealth

import (
	"context"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu    sync.Mutex
	items map[string]Heartbeat
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: map[string]Heartbeat{}}
}

func (r *MemoryRepository) List(_ context.Context, now time.Time) ([]Heartbeat, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]Heartbeat, 0, len(r.items))
	for _, item := range r.items {
		result = append(result, Enrich(item, now))
	}
	return result, nil
}

func (r *MemoryRepository) Upsert(_ context.Context, update HeartbeatUpdate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item := r.items[update.HostID]
	item.HostID = update.HostID
	item.HostName = update.HostName
	item.InputMode = inputMode(update.InputMode)
	if update.LastError != "" || update.ClearError {
		item.LastError = update.LastError
	}
	item.LastSeenAt = update.LastSeenAt
	if item.LastSeenAt.IsZero() {
		item.LastSeenAt = time.Now().UTC()
	}
	if update.LastEventTime != nil {
		item.LastEventTime = update.LastEventTime
	}
	if update.LastWriteAt != nil {
		item.LastWriteAt = update.LastWriteAt
	}
	item.EventsWritten += update.EventsWritten
	item.UpdatedAt = time.Now().UTC()
	r.items[update.HostID] = item
	return nil
}
