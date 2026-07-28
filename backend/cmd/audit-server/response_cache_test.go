package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"diting/backend/internal/cache"
	"diting/backend/internal/config"
)

func TestNewRequiredResponseCacheRejectsMissingRedis(t *testing.T) {
	_, _, err := newRequiredResponseCache(context.Background(), config.RedisConfig{})
	if err == nil {
		t.Fatal("expected missing redis config to fail")
	}
	if !strings.Contains(err.Error(), "redis") {
		t.Fatalf("expected redis error, got %v", err)
	}
}

func TestNewRequiredResponseCacheRejectsUnavailableRedis(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _, err := newRequiredResponseCache(ctx, config.RedisConfig{
		Enabled:                 true,
		Addr:                    "127.0.0.1:9",
		ResponseCacheTTLSeconds: 15,
	})
	if err == nil {
		t.Fatal("expected unavailable redis to fail")
	}
}

func TestResponseCacheTTLDefault(t *testing.T) {
	if got := responseCacheTTL(config.RedisConfig{}); got != 15*time.Second {
		t.Fatalf("expected default ttl 15s, got %s", got)
	}
	if got := responseCacheTTL(config.RedisConfig{ResponseCacheTTLSeconds: 30}); got != 30*time.Second {
		t.Fatalf("expected ttl 30s, got %s", got)
	}
}

var _ cache.Cache
