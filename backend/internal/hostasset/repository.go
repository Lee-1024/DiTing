package hostasset

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type HostAsset struct {
	ID          string    `json:"id"`
	NodeName    string    `json:"nodeName"`
	DisplayName string    `json:"displayName"`
	HostIP      string    `json:"hostIp"`
	Environment string    `json:"environment"`
	Owner       string    `json:"owner"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Repository interface {
	Create(ctx context.Context, asset HostAsset) (HostAsset, error)
	List(ctx context.Context) ([]HostAsset, error)
	Get(ctx context.Context, id string) (HostAsset, error)
	Update(ctx context.Context, id string, asset HostAsset) (HostAsset, error)
	Delete(ctx context.Context, id string) error
}

var ErrNotFound = errors.New("host asset not found")

type MemoryRepository struct {
	mu     sync.Mutex
	assets []HostAsset
	next   int
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{next: 1, assets: []HostAsset{}}
}

func (r *MemoryRepository) Create(_ context.Context, asset HostAsset) (HostAsset, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	asset.ID = fmt.Sprintf("host-%d", r.next)
	asset.CreatedAt = now
	asset.UpdatedAt = now
	r.next++
	r.assets = append(r.assets, asset)
	return asset, nil
}

func (r *MemoryRepository) List(_ context.Context) ([]HostAsset, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]HostAsset, len(r.assets))
	copy(result, r.assets)
	return result, nil
}

func (r *MemoryRepository) Get(_ context.Context, id string) (HostAsset, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, asset := range r.assets {
		if asset.ID == id {
			return asset, nil
		}
	}
	return HostAsset{}, ErrNotFound
}

func (r *MemoryRepository) Update(_ context.Context, id string, next HostAsset) (HostAsset, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index, asset := range r.assets {
		if asset.ID != id {
			continue
		}
		next.ID = id
		next.CreatedAt = asset.CreatedAt
		next.UpdatedAt = time.Now().UTC()
		r.assets[index] = next
		return next, nil
	}
	return HostAsset{}, ErrNotFound
}

func (r *MemoryRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index, asset := range r.assets {
		if asset.ID != id {
			continue
		}
		r.assets = append(r.assets[:index], r.assets[index+1:]...)
		return nil
	}
	return ErrNotFound
}
