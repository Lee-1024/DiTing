package server

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"diting/backend/internal/cache"
)

func TestResponseCacheServesRepeatedGETFromCache(t *testing.T) {
	store := cache.NewMemory()
	calls := 0
	handler := responseCache(store, cache.NewSingleflight(), "test", time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"calls":` + strconv.Itoa(calls) + `}`))
	}))

	first := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/operations?page=1&keyword=id", nil)
	handler.ServeHTTP(first, req)

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, req)

	if calls != 1 {
		t.Fatalf("expected handler to run once, got %d", calls)
	}
	if first.Body.String() != second.Body.String() {
		t.Fatalf("expected cached response body, got first=%s second=%s", first.Body.String(), second.Body.String())
	}
	if second.Header().Get("X-DiTing-Cache") != "HIT" {
		t.Fatalf("expected cache hit header, got %q", second.Header().Get("X-DiTing-Cache"))
	}
}

func TestResponseCacheDoesNotCacheNonGET(t *testing.T) {
	store := cache.NewMemory()
	calls := 0
	handler := responseCache(store, cache.NewSingleflight(), "test", time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte("call"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/audit/operations", strings.NewReader("{}"))
	handler.ServeHTTP(httptest.NewRecorder(), req)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if calls != 2 {
		t.Fatalf("expected non-GET requests to bypass cache, got %d calls", calls)
	}
}

func TestResponseCacheDoesNotStoreErrors(t *testing.T) {
	store := cache.NewMemory()
	calls := 0
	handler := responseCache(store, cache.NewSingleflight(), "test", time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "failed", http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/overview", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if calls != 2 {
		t.Fatalf("expected error responses to bypass cache storage, got %d calls", calls)
	}
}

func TestCacheTTLUsesNamespaceOverride(t *testing.T) {
	options := routerOptions{
		responseCacheTTL: 15 * time.Second,
		responseCacheTTLs: map[string]time.Duration{
			"stats.hosts.behavior": 5 * time.Minute,
		},
	}
	if got := cacheTTL(options, "stats.hosts.behavior"); got != 5*time.Minute {
		t.Fatalf("expected namespace ttl override, got %v", got)
	}
	if got := cacheTTL(options, "stats.overview"); got != 15*time.Second {
		t.Fatalf("expected default ttl, got %v", got)
	}
}

func TestResponseCacheSuppressesConcurrentMisses(t *testing.T) {
	store := cache.NewMemory()
	var calls int32
	start := make(chan struct{})
	handler := responseCache(store, cache.NewSingleflight(), "test", time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		<-start
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/hosts?host_name=host-1", nil)
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler.ServeHTTP(httptest.NewRecorder(), req)
		}()
	}
	time.Sleep(50 * time.Millisecond)
	close(start)
	wg.Wait()

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected single backend call for concurrent cache miss, got %d", got)
	}
}
