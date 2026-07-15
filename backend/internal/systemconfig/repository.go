package systemconfig

import (
	"context"
	"sync"
)

const CollectorFilterKey = "collector_filter"

type CollectorFilterConfig struct {
	Enabled               bool     `json:"enabled"`
	IgnoreProcessNames    []string `json:"ignoreProcessNames"`
	IgnoreCommandKeywords []string `json:"ignoreCommandKeywords"`
	IgnoreUsers           []string `json:"ignoreUsers"`
	KeepSeverities        []string `json:"keepSeverities"`
}

type Repository interface {
	GetCollectorFilter(ctx context.Context) (CollectorFilterConfig, error)
	SaveCollectorFilter(ctx context.Context, config CollectorFilterConfig) error
}

func DefaultCollectorFilterConfig() CollectorFilterConfig {
	return CollectorFilterConfig{
		Enabled:        false,
		KeepSeverities: []string{"high", "critical"},
	}
}

type MemoryRepository struct {
	mu              sync.Mutex
	collectorFilter CollectorFilterConfig
	hasFilter       bool
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{}
}

func (r *MemoryRepository) GetCollectorFilter(_ context.Context) (CollectorFilterConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.hasFilter {
		return DefaultCollectorFilterConfig(), nil
	}
	return normalizeCollectorFilterConfig(r.collectorFilter), nil
}

func (r *MemoryRepository) SaveCollectorFilter(_ context.Context, config CollectorFilterConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.collectorFilter = normalizeCollectorFilterConfig(config)
	r.hasFilter = true
	return nil
}

func normalizeCollectorFilterConfig(config CollectorFilterConfig) CollectorFilterConfig {
	if len(config.KeepSeverities) == 0 {
		config.KeepSeverities = []string{"high", "critical"}
	}
	if config.IgnoreProcessNames == nil {
		config.IgnoreProcessNames = []string{}
	}
	if config.IgnoreCommandKeywords == nil {
		config.IgnoreCommandKeywords = []string{}
	}
	if config.IgnoreUsers == nil {
		config.IgnoreUsers = []string{}
	}
	return config
}
