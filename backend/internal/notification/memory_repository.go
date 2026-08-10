package notification

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu                 sync.Mutex
	seq                int
	items              map[string]Notification
	activeKeys         map[string]string
	enforcementSources map[string]string
	reads              map[string]map[string]bool
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		items:              map[string]Notification{},
		activeKeys:         map[string]string{},
		enforcementSources: map[string]string{},
		reads:              map[string]map[string]bool{},
	}
}

func (r *MemoryRepository) Upsert(_ context.Context, input Input) (Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if input.Type == TypeEnforcement {
		if id, ok := r.enforcementSources[input.SourceID]; ok {
			return r.items[id], nil
		}
	}
	if id, ok := r.activeKeys[input.DedupeKey]; ok {
		item := r.items[id]
		item.Title = input.Title
		item.Description = input.Description
		item.Severity = input.Severity
		item.Target = input.Target
		item.UpdatedAt = time.Now().UTC()
		r.items[id] = item
		return item, nil
	}
	now := time.Now().UTC()
	r.seq++
	item := Notification{
		ID: fmt.Sprintf("notification-%d", r.seq), Type: input.Type, DedupeKey: input.DedupeKey,
		SourceID: input.SourceID, Title: input.Title, Description: input.Description,
		Severity: input.Severity, Target: input.Target, Status: StatusOpen,
		CreatedAt: now, UpdatedAt: now,
	}
	r.items[item.ID] = item
	r.activeKeys[input.DedupeKey] = item.ID
	if input.Type == TypeEnforcement {
		r.enforcementSources[input.SourceID] = item.ID
	}
	return item, nil
}

func (r *MemoryRepository) Resolve(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.activeKeys[key]
	if !ok {
		return nil
	}
	item := r.items[id]
	now := time.Now().UTC()
	item.Status = StatusResolved
	item.ResolvedAt = &now
	item.UpdatedAt = now
	r.items[id] = item
	delete(r.activeKeys, key)
	return nil
}

func (r *MemoryRepository) List(_ context.Context, userID, view string, limit int) (ListResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	view = NormalizeView(view)
	all := make([]Notification, 0, len(r.items))
	counts := Counts{All: len(r.items)}
	for _, stored := range r.items {
		item := stored
		item.Read = r.reads[item.ID] != nil && r.reads[item.ID][userID]
		if !item.Read {
			counts.Unread++
		}
		if item.Type == TypeEnforcement && item.Status == StatusOpen {
			counts.Pending++
		}
		if view == "unread" && item.Read {
			continue
		}
		if view == "pending" && (item.Type != TypeEnforcement || item.Status != StatusOpen) {
			continue
		}
		all = append(all, item)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.After(all[j].CreatedAt) })
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if len(all) > limit {
		all = all[:limit]
	}
	return ListResult{Items: all, Counts: counts}, nil
}

func (r *MemoryRepository) MarkRead(_ context.Context, userID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return ErrNotFound
	}
	if r.reads[id] == nil {
		r.reads[id] = map[string]bool{}
	}
	r.reads[id][userID] = true
	return nil
}

func (r *MemoryRepository) MarkAllRead(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id := range r.items {
		if r.reads[id] == nil {
			r.reads[id] = map[string]bool{}
		}
		r.reads[id][userID] = true
	}
	return nil
}

func (r *MemoryRepository) Handle(_ context.Context, id, disposition, username string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok || item.Type != TypeEnforcement {
		return ErrNotFound
	}
	normalized, err := NormalizeDisposition(disposition)
	if err != nil {
		return err
	}
	if item.Status == StatusResolved {
		return nil
	}
	now := time.Now().UTC()
	item.Status = StatusResolved
	item.Disposition = normalized
	item.HandledBy = username
	item.HandledAt = &now
	item.ResolvedAt = &now
	item.UpdatedAt = now
	r.items[id] = item
	delete(r.activeKeys, item.DedupeKey)
	return nil
}
