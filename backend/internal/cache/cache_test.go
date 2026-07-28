package cache

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestKeyIsStableForQueryParameterOrder(t *testing.T) {
	left := Key("audit.operations", map[string]string{
		"page":       "1",
		"start_time": "2026-07-28T00:00:00Z",
		"end_time":   "2026-07-28T23:59:59Z",
	})
	right := Key("audit.operations", map[string]string{
		"end_time":   "2026-07-28T23:59:59Z",
		"start_time": "2026-07-28T00:00:00Z",
		"page":       "1",
	})

	if left != right {
		t.Fatalf("expected stable keys, got %q and %q", left, right)
	}
}

func TestMemoryCacheHonorsTTL(t *testing.T) {
	cache := NewMemory()
	ctx := context.Background()

	if err := cache.Set(ctx, "key", []byte("value"), time.Millisecond); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if value, ok, err := cache.Get(ctx, "key"); err != nil || !ok || string(value) != "value" {
		t.Fatalf("expected cached value, got value=%q ok=%v err=%v", value, ok, err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, ok, err := cache.Get(ctx, "key"); err != nil || ok {
		t.Fatalf("expected expired value to miss, ok=%v err=%v", ok, err)
	}
}

func TestMemoryCacheDeletePrefix(t *testing.T) {
	cache := NewMemory()
	ctx := context.Background()
	_ = cache.Set(ctx, "stats:one", []byte("1"), time.Minute)
	_ = cache.Set(ctx, "stats:two", []byte("2"), time.Minute)
	_ = cache.Set(ctx, "audit:one", []byte("3"), time.Minute)

	if err := cache.DeletePrefix(ctx, "stats:"); err != nil {
		t.Fatalf("DeletePrefix returned error: %v", err)
	}
	if _, ok, _ := cache.Get(ctx, "stats:one"); ok {
		t.Fatal("expected stats:one to be deleted")
	}
	if _, ok, _ := cache.Get(ctx, "audit:one"); !ok {
		t.Fatal("expected audit:one to remain")
	}
}

func TestSingleflightRunsLoaderOnceForConcurrentRequests(t *testing.T) {
	group := NewSingleflight()
	ctx := context.Background()
	var mu sync.Mutex
	calls := 0
	loader := func(context.Context) ([]byte, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		return []byte("loaded"), nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, err := group.Do(ctx, "same-key", loader)
			if err != nil {
				t.Errorf("Do returned error: %v", err)
			}
			if string(value) != "loaded" {
				t.Errorf("expected loaded value, got %q", value)
			}
		}()
	}
	wg.Wait()

	if calls != 1 {
		t.Fatalf("expected loader to run once, got %d", calls)
	}
}
