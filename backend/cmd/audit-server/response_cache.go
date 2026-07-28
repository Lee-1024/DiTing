package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"diting/backend/internal/cache"
	"diting/backend/internal/config"
)

func newRequiredResponseCache(ctx context.Context, cfg config.RedisConfig) (cache.Cache, time.Duration, error) {
	if !cfg.Enabled || strings.TrimSpace(cfg.Addr) == "" {
		return nil, 0, fmt.Errorf("redis is required for api response cache")
	}
	store := cache.NewRedis(cache.RedisConfig{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	if err := store.Ping(ctx); err != nil {
		_ = store.Close()
		return nil, 0, fmt.Errorf("connect redis response cache: %w", err)
	}
	return store, responseCacheTTL(cfg), nil
}

func responseCacheTTL(cfg config.RedisConfig) time.Duration {
	if cfg.ResponseCacheTTLSeconds <= 0 {
		return 15 * time.Second
	}
	return time.Duration(cfg.ResponseCacheTTLSeconds) * time.Second
}
