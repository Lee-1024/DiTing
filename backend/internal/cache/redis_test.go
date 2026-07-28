package cache

import (
	"context"
	"testing"
	"time"
)

func TestRedisCacheReturnsConnectionErrorForUnavailableRedis(t *testing.T) {
	cache := NewRedis(RedisConfig{Addr: "127.0.0.1:9"})
	defer cache.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, ok, err := cache.Get(ctx, "missing")
	if err == nil {
		t.Fatal("expected connection error")
	}
	if ok {
		t.Fatal("expected unavailable redis to be a miss with error")
	}
}
